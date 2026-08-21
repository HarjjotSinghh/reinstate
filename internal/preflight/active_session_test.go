package preflight

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
)

func activeInput() Input {
	return Input{
		SessionRef:  "claude:session-under-test",
		Agent:       "claude",
		SessionID:   "session-under-test",
		SessionPath: "/does/not/matter/session.jsonl",
		ProjectRoot: "/does/not/matter",
	}
}

func TestActiveSessionCheck(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name         string
		busy, scoped bool
		err          error
		delay        time.Duration
		readOnly     bool
		unscoped     bool
		omitProbe    bool
		wantPresent  bool
		wantSeverity Severity
		wantStatus   Status
	}{
		{
			name: "busy and scoped warns", busy: true, scoped: true,
			wantPresent: true, wantSeverity: SeverityWarning, wantStatus: StatusPresent,
		},
		{
			// The host knows the agent runs but not which session. Resume is
			// still the operator's call, so this warns rather than blocking.
			name: "busy but unscoped warns", busy: true, scoped: false,
			wantPresent: true, wantSeverity: SeverityWarning, wantStatus: StatusPresent,
		},
		{
			name: "free records the observation", busy: false,
			wantPresent: true, wantSeverity: SeverityInfo, wantStatus: StatusMatch,
		},
		{
			// A host without process listing must not be locked out of resume,
			// and must not be told its session is free either.
			name: "probe error is reported not escalated", err: errors.New("no ps on this host"),
			wantPresent: true, wantSeverity: SeverityInfo, wantStatus: StatusUnknown,
		},
		{
			name: "probe deadline is reported not escalated", busy: true, scoped: true,
			delay:       75 * time.Millisecond,
			wantPresent: true, wantSeverity: SeverityInfo, wantStatus: StatusUnknown,
		},
		{
			// A structured handoff only reads the source, and the handoff
			// pipeline enforces its own --allow-active against the same signal.
			name: "read-only handoff is not double-guarded", busy: true, scoped: true,
			readOnly: true, wantPresent: false,
		},
		{
			// Without a session to scope to, the probe could only answer the
			// host-wide question, which is not evidence about this session.
			name: "unscoped input omits the check", busy: true, scoped: true,
			unscoped: true, wantPresent: false,
		},
		{
			// Absent boundary means the question was never asked. Reporting a
			// session as free on evidence never gathered is the unsafe claim.
			name: "no boundary omits the check", omitProbe: true, wantPresent: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var calls int
			options := Options{SessionBusyTimeout: 40 * time.Millisecond}
			if !testCase.omitProbe {
				options.SessionBusy = func(
					ctx context.Context, agent string, target processcheck.Target,
				) (bool, bool, error) {
					calls++
					if agent != "claude" {
						t.Errorf("probe received agent %q, want claude", agent)
					}
					if testCase.delay > 0 {
						select {
						case <-time.After(testCase.delay):
						case <-ctx.Done():
							return false, false, ctx.Err()
						}
					}
					return testCase.busy, testCase.scoped, testCase.err
				}
			}

			input := activeInput()
			input.ReadOnly = testCase.readOnly
			if testCase.unscoped {
				input.SessionID, input.SessionPath = "", ""
			}

			await, release := startActiveSessionProbe(context.Background(), input, options)
			defer release()
			check, present := await()

			if present != testCase.wantPresent {
				t.Fatalf("check present = %t, want %t", present, testCase.wantPresent)
			}
			if !testCase.wantPresent {
				if calls != 0 {
					t.Fatalf("probe ran %d times when the check is omitted", calls)
				}
				return
			}
			if check.ID != "agent.active" {
				t.Fatalf("check ID = %q, want agent.active", check.ID)
			}
			if check.Severity != testCase.wantSeverity {
				t.Fatalf("severity = %q, want %q (message %q)",
					check.Severity, testCase.wantSeverity, check.Message)
			}
			if check.Status != testCase.wantStatus {
				t.Fatalf("status = %q, want %q", check.Status, testCase.wantStatus)
			}
			if check.Message == "" {
				t.Fatal("check carries no message")
			}
			if testCase.wantSeverity == SeverityWarning && check.Repair == "" {
				t.Fatal("a warning the operator must acknowledge carries no repair")
			}
		})
	}
}

// TestActiveSessionWarningIsAcknowledgeable is the E5 contract, driven through
// the real verifier rather than a hand-built report: a detected active session
// must reach the report as a warning, must refuse a launch that does not clear
// it, and must be clearable through the acknowledgement path resume already
// has, without a second differently-spelled flag.
func TestActiveSessionWarningIsAcknowledgeable(t *testing.T) {
	t.Parallel()
	fixture := newFixture(t, "https://example.com/org/repo.git")
	fixture.options.SessionBusy = func(
		context.Context, string, processcheck.Target,
	) (bool, bool, error) {
		return true, true, nil
	}

	input := Input{
		SessionRef: "claude:controlled", Agent: "claude", Workspace: fixture.workspace,
		SourceFresh: true,
		SessionID:   "controlled", SessionPath: fixture.workspace,
	}
	report, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}

	active := findCheck(t, report, "agent.active")
	if active.Severity != SeverityWarning {
		t.Fatalf("agent.active = %+v, want a warning", active)
	}
	if report.Decision != DecisionConfirmationRequired {
		t.Fatalf("decision = %q, want %q", report.Decision, DecisionConfirmationRequired)
	}

	ids := WarningIDs(report)
	if !slices.Contains(ids, "agent.active") {
		t.Fatalf("warning IDs = %v, want agent.active among them", ids)
	}

	// Acknowledging every warning except this one must still refuse, or the
	// guard could be cleared without ever naming the active session.
	others := slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return id == "agent.active" })
	partial, err := Authorize(report, others)
	if err == nil && partial.Allowed {
		t.Fatal("launch authorized without acknowledging the active session")
	}
	if !slices.Contains(partial.Warnings, "agent.active") {
		t.Fatalf("refusal did not name the outstanding warning: %v", partial.Warnings)
	}

	acknowledged, err := Authorize(report, ids)
	if err != nil {
		t.Fatalf("acknowledging the warnings failed: %v", err)
	}
	if !acknowledged.Allowed {
		t.Fatal("acknowledging agent.active did not authorize the launch")
	}
}

// TestNoActiveSessionNeedsNoAcknowledgement guards against the check becoming
// noise: a free session must not add a warning to every resume.
func TestNoActiveSessionNeedsNoAcknowledgement(t *testing.T) {
	t.Parallel()
	fixture := newFixture(t, "https://example.com/org/repo.git")
	fixture.options.SessionBusy = func(
		context.Context, string, processcheck.Target,
	) (bool, bool, error) {
		return false, true, nil
	}

	report, err := Verify(context.Background(), Input{
		SessionRef: "claude:controlled", Agent: "claude", Workspace: fixture.workspace,
		SourceFresh: true,
		SessionID:   "controlled", SessionPath: fixture.workspace,
	}, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if active := findCheck(t, report, "agent.active"); active.Severity != SeverityInfo {
		t.Fatalf("a free session produced %+v, want an info check", active)
	}
	if slices.Contains(WarningIDs(report), "agent.active") {
		t.Fatal("a free session required an acknowledgement")
	}
}
