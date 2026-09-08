// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package readiness

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// errProbe is the canned failure a fake verifier returns. A probe failing is an
// ordinary event — a vendor binary that is momentarily unreadable, a workspace
// on a disconnected volume — not a test-only curiosity.
var errProbe = errors.New("workspace is unavailable")

// fakeVerifier is the only engine dependency the prober has, replaced wholesale.
// No real preflight runs in this file: no filesystem, no vendor binary, no
// clock, and therefore no reason for a frame or a decision to differ between
// machines.
//
// The call ledger is mutex-guarded because the concurrency test runs every
// command the prober hands back from its own goroutine, which is exactly how
// Bubble Tea runs them.
type fakeVerifier struct {
	mu      sync.Mutex
	calls   map[string]int
	respond func(sessionindex.Record) (preflight.Report, error)
}

func newFakeVerifier(respond func(sessionindex.Record) (preflight.Report, error)) *fakeVerifier {
	return &fakeVerifier{calls: make(map[string]int), respond: respond}
}

// ready is the common fake: every record verifies cleanly.
func readyVerifier() *fakeVerifier {
	return newFakeVerifier(func(sessionindex.Record) (preflight.Report, error) {
		return preflight.Report{Decision: preflight.DecisionReady}, nil
	})
}

func (f *fakeVerifier) verify(_ context.Context, record sessionindex.Record) (preflight.Report, error) {
	f.mu.Lock()
	f.calls[record.Key]++
	f.mu.Unlock()
	if f.respond == nil {
		return preflight.Report{}, nil
	}
	return f.respond(record)
}

// total returns how many probes ran across every record.
func (f *fakeVerifier) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	sum := 0
	for _, count := range f.calls {
		sum += count
	}
	return sum
}

// count returns how many probes ran for one record.
func (f *fakeVerifier) count(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[key]
}

// resumable is the ordinary record: the index believes it can resume, so
// readiness is a question only a probe can answer.
func resumable(key string) sessionindex.Record {
	return sessionindex.Record{
		Key:       key,
		ID:        key,
		Agent:     sessionindex.AgentClaude,
		CanResume: true,
		CanFork:   true,
	}
}

// resumableRecords builds a corpus of distinct resumable records.
func resumableRecords(count int) []sessionindex.Record {
	records := make([]sessionindex.Record, 0, count)
	for index := 0; index < count; index++ {
		records = append(records, resumable(fmt.Sprintf("claude:%04d", index)))
	}
	return records
}

// drain runs a command tree to completion and returns every message it made,
// the way the Bubble Tea runtime would.
func drain(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, next := range batch {
			msgs = append(msgs, drain(t, next)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// batchCommands unwraps the batch Probe returns into the individual probe
// commands. Calling the batch command itself performs no probing: it only names
// the commands the runtime should then run, each on its own goroutine.
func batchCommands(t *testing.T, cmd tea.Cmd) []tea.Cmd {
	t.Helper()
	if cmd == nil {
		t.Fatal("Probe returned no command; there was work to do")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Probe returned %T, want a tea.BatchMsg of probe commands", msg)
	}
	return batch
}

// probedKeys extracts the record keys reported by a set of ProbedMsg values.
func probedKeys(t *testing.T, msgs []tea.Msg) []string {
	t.Helper()
	var keys []string
	for _, msg := range msgs {
		probed, ok := msg.(ProbedMsg)
		if !ok {
			t.Fatalf("probe produced %T, want ProbedMsg", msg)
		}
		keys = append(keys, probed.Keys...)
	}
	return keys
}

// TestFromReportMapsEveryDecision pins the whole mapping, including the two
// answers that are easy to get wrong: an unrecognised decision and a failed
// probe both mean "not known", never "cannot resume".
func TestFromReportMapsEveryDecision(t *testing.T) {
	tests := []struct {
		name     string
		decision preflight.Decision
		want     ui.Readiness
	}{
		{name: "ready", decision: preflight.DecisionReady, want: ui.ReadinessReady},
		{name: "confirmation required", decision: preflight.DecisionConfirmationRequired, want: ui.ReadinessWarn},
		{name: "blocked", decision: preflight.DecisionBlocked, want: ui.ReadinessBlocked},
		{name: "unknown decision", decision: preflight.Decision("something new"), want: ui.ReadinessUnknown},
		{name: "empty decision", decision: "", want: ui.ReadinessUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := FromReport(preflight.Report{Decision: test.decision}, nil)
			if got != test.want {
				t.Fatalf("FromReport(%q) = %v, want %v", test.decision, got, test.want)
			}
		})
	}

	// A failed probe learned nothing about the environment. Reporting blocked
	// would be a claim the user has to disprove by hand, so every decision —
	// including one the report happens to carry — collapses to unknown.
	t.Run("an error outranks every decision", func(t *testing.T) {
		for _, decision := range []preflight.Decision{
			preflight.DecisionReady,
			preflight.DecisionConfirmationRequired,
			preflight.DecisionBlocked,
			preflight.Decision("something new"),
			"",
		} {
			got := FromReport(preflight.Report{Decision: decision}, errProbe)
			if got != ui.ReadinessUnknown {
				t.Fatalf("FromReport(%q, err) = %v, want %v; a failed probe must not claim %q",
					decision, got, ui.ReadinessUnknown, ui.ReadinessBlocked.Label())
			}
		}
	})
}

// TestFromReportDistinguishesUninspectableFromGenuineBlocks is the table-driven
// mapping test for row 13 of the CLI experience matrix, "readiness glyphs
// resolve for visible rows; a read-only agent shows blocked without a probe",
// and its v0.6.0-rc.7 regression.
//
// preflight.Verify shares one wall clock across several independent
// observers. When that clock runs out mid-observation the check in question
// comes back as a normal, no-error report — not a verify error — with
// Status unknown (or, for an infrastructure failure such as a cancelled
// capability scan, Status error with ExitCode exitcode.Runtime) and Severity
// block, and that alone makes Decision Blocked. Nothing about the session was
// actually established in that case, so it must read the same as a probe
// that failed outright: Unknown, not Blocked. A report whose Blocked decision
// rests on an actual finding (a missing workspace, a foreign repository) must
// still read Blocked — that is what keeps a genuinely unresumable session
// from disappearing into "checking" forever.
func TestFromReportDistinguishesUninspectableFromGenuineBlocks(t *testing.T) {
	tests := []struct {
		name   string
		report preflight.Report
		err    error
		want   ui.Readiness
	}{
		{
			name: "timeout-shaped blocked report: the agent-version probe never got an answer",
			report: preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "agent.executable", Status: preflight.StatusPresent, Severity: preflight.SeverityInfo},
				{ID: "agent.layout", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo},
				{ID: "agent.version", Status: preflight.StatusUnknown, Severity: preflight.SeverityBlock,
					ExitCode: exitcode.Compatibility, Message: "the native agent version probe failed"},
			}},
			want: ui.ReadinessUnknown,
		},
		{
			name: "timeout-shaped blocked report: capability discovery was cancelled by the deadline",
			report: preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "capability.probe.cancelled.deadbeef", Status: preflight.StatusError, Severity: preflight.SeverityBlock,
					ExitCode: exitcode.Runtime, Message: "capability discovery exceeded the bounded preflight deadline"},
			}},
			want: ui.ReadinessUnknown,
		},
		{
			name: "genuinely blocked report: the recorded workspace does not exist",
			report: preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "workspace.available", Status: preflight.StatusMissing, Severity: preflight.SeverityBlock,
					ExitCode: exitcode.Compatibility, Message: "the recorded workspace does not exist"},
			}},
			want: ui.ReadinessBlocked,
		},
		{
			name: "genuinely blocked report: the session records a foreign repository",
			report: preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "git.repository", Status: preflight.StatusChanged, Severity: preflight.SeverityBlock,
					ExitCode: exitcode.Safety, Message: "the session records a foreign repository_url"},
			}},
			want: ui.ReadinessBlocked,
		},
		{
			name: "a genuine finding outranks an uninspectable check alongside it",
			report: preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "agent.version", Status: preflight.StatusUnknown, Severity: preflight.SeverityBlock},
				{ID: "workspace.available", Status: preflight.StatusMissing, Severity: preflight.SeverityBlock},
			}},
			want: ui.ReadinessBlocked,
		},
		{
			name:   "ready report",
			report: preflight.Report{Decision: preflight.DecisionReady},
			want:   ui.ReadinessReady,
		},
		{
			name: "verify returned an error",
			report: preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "workspace.available", Status: preflight.StatusMissing, Severity: preflight.SeverityBlock},
			}},
			err:  errProbe,
			want: ui.ReadinessUnknown,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FromReport(test.report, test.err); got != test.want {
				t.Fatalf("FromReport(...) = %v, want %v", got, test.want)
			}
		})
	}
}

// TestProbeCachingAndRetryMatchesFromReport is the Prober-level half of the
// row 13 table: FromReport's mapping decides the glyph, but the prober's own
// job is what makes that mapping matter — an Unknown answer must be retried
// on the next Probe call rather than sticking forever, while a terminal
// answer (Ready, Warn, or a genuinely Blocked report) must never be re-probed.
func TestProbeCachingAndRetryMatchesFromReport(t *testing.T) {
	tests := []struct {
		name          string
		respond       func(sessionindex.Record) (preflight.Report, error)
		wantReadiness ui.Readiness
		wantRetried   bool
	}{
		{
			name: "timeout-shaped blocked report",
			respond: func(sessionindex.Record) (preflight.Report, error) {
				return preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
					{ID: "agent.version", Status: preflight.StatusUnknown, Severity: preflight.SeverityBlock},
				}}, nil
			},
			wantReadiness: ui.ReadinessUnknown,
			wantRetried:   true,
		},
		{
			name: "genuinely blocked report",
			respond: func(sessionindex.Record) (preflight.Report, error) {
				return preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
					{ID: "workspace.available", Status: preflight.StatusMissing, Severity: preflight.SeverityBlock},
				}}, nil
			},
			wantReadiness: ui.ReadinessBlocked,
			wantRetried:   false,
		},
		{
			name: "ready report",
			respond: func(sessionindex.Record) (preflight.Report, error) {
				return preflight.Report{Decision: preflight.DecisionReady}, nil
			},
			wantReadiness: ui.ReadinessReady,
			wantRetried:   false,
		},
		{
			name: "verify error",
			respond: func(sessionindex.Record) (preflight.Report, error) {
				return preflight.Report{}, errProbe
			},
			wantReadiness: ui.ReadinessUnknown,
			wantRetried:   true,
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := newFakeVerifier(test.respond)
			prober := New(verifier.verify)
			record := resumable(fmt.Sprintf("claude:case-%d", index))

			drain(t, prober.Probe(context.Background(), []sessionindex.Record{record}))
			if got := prober.Lookup(record); got != test.wantReadiness {
				t.Fatalf("Lookup after the first probe = %v, want %v", got, test.wantReadiness)
			}
			if got := verifier.count(record.Key); got != 1 {
				t.Fatalf("the first Probe ran verify %d times, want 1", got)
			}

			cmd := prober.Probe(context.Background(), []sessionindex.Record{record})
			if test.wantRetried {
				if cmd == nil {
					t.Fatal("a second Probe call declined to retry a record whose answer was Unknown")
				}
				drain(t, cmd)
				if got := verifier.count(record.Key); got != 2 {
					t.Fatalf("after the retry, verify ran %d times, want 2", got)
				}
				// A retry that has not resolved yet must not flicker the glyph.
				if got := prober.Lookup(record); got != ui.ReadinessUnknown {
					t.Fatalf("Lookup while a retry is settling = %v, want %v", got, ui.ReadinessUnknown)
				}
			} else {
				if cmd != nil {
					t.Fatal("a second Probe call re-verified a terminal (non-retryable) answer")
				}
				if got := verifier.count(record.Key); got != 1 {
					t.Fatalf("verify ran %d times for a terminal answer, want exactly 1", got)
				}
				if got := prober.Lookup(record); got != test.wantReadiness {
					t.Fatalf("Lookup after the no-op Probe = %v, want %v", got, test.wantReadiness)
				}
			}
		})
	}
}

// TestProbeStopsRetryingAtMaxProbeRetries bounds the retry loop: a record
// whose report keeps coming back uninspectable must not compete for
// MaxConcurrentProbes' slots forever. After maxProbeRetries attempts, Probe
// stops scheduling it — the row is left reading "checking"
// (ui.ReadinessUnknown, the same glyph it already had, so nothing flickers)
// rather than being promoted to anything else on no real evidence.
func TestProbeStopsRetryingAtMaxProbeRetries(t *testing.T) {
	verifier := newFakeVerifier(func(sessionindex.Record) (preflight.Report, error) {
		return preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
			{ID: "agent.version", Status: preflight.StatusUnknown, Severity: preflight.SeverityBlock},
		}}, nil
	})
	prober := New(verifier.verify)
	record := resumable("claude:durably-uninspectable")

	for attempt := 1; attempt <= maxProbeRetries; attempt++ {
		cmd := prober.Probe(context.Background(), []sessionindex.Record{record})
		if cmd == nil {
			t.Fatalf("attempt %d: Probe declined to (re)probe before the retry cap was reached", attempt)
		}
		drain(t, cmd)
		if got := verifier.count(record.Key); got != attempt {
			t.Fatalf("after attempt %d, verify ran %d times, want %d", attempt, got, attempt)
		}
		if got := prober.Lookup(record); got != ui.ReadinessUnknown {
			t.Fatalf("Lookup after attempt %d = %v, want %v (it must never flicker to anything else)", attempt, got, ui.ReadinessUnknown)
		}
	}

	// The cap is reached: a further Probe call declines to try again.
	if cmd := prober.Probe(context.Background(), []sessionindex.Record{record}); cmd != nil {
		t.Fatal("Probe scheduled another attempt after the retry cap was reached")
	}
	if got := verifier.count(record.Key); got != maxProbeRetries {
		t.Fatalf("verify ran %d times in total, want exactly %d (the cap)", got, maxProbeRetries)
	}
	if got := prober.Lookup(record); got != ui.ReadinessUnknown {
		t.Fatalf("Lookup once the cap is reached = %v, want %v", got, ui.ReadinessUnknown)
	}
}

// TestRow13AnUninspectableBlockSelfCorrectsOnRetry is the named regression
// test for CLI experience row 13, "readiness glyphs resolve for visible rows;
// a read-only agent shows blocked without a probe", and its v0.6.0-rc.7
// regression: under a ScopeAll listing with many rows to probe at once,
// enough preflight checks missed their shared deadline that most launches
// showed at least one row as Blocked whose ground truth was Ready — and,
// because the wrong verdict was cached exactly like a real one, it never
// self-corrected for the life of the process.
//
// This simulates the shape that regression actually produced: a first probe
// that comes back in the exact could-not-evaluate-in-time shape
// preflight.Verify leaves behind (a normal report, Decision Blocked, with
// only an unresolved check behind it), followed by a second attempt that
// completes normally — the shape a host produces once whatever contention
// caused the first timeout has passed. The row must read "checking"
// throughout, never Blocked, and must land on the correct answer without
// anything else (a keystroke, a reload) asking again.
func TestRow13AnUninspectableBlockSelfCorrectsOnRetry(t *testing.T) {
	const key = "claude:row-13"
	attempt := 0
	verifier := newFakeVerifier(func(sessionindex.Record) (preflight.Report, error) {
		attempt++
		if attempt == 1 {
			return preflight.Report{Decision: preflight.DecisionBlocked, Checks: []preflight.Check{
				{ID: "agent.version", Status: preflight.StatusUnknown, Severity: preflight.SeverityBlock,
					Message: "the native agent version probe failed"},
			}}, nil
		}
		return preflight.Report{Decision: preflight.DecisionReady}, nil
	})
	prober := New(verifier.verify)
	record := resumable(key)

	drain(t, prober.Probe(context.Background(), []sessionindex.Record{record}))
	if got := prober.Lookup(record); got != ui.ReadinessUnknown {
		t.Fatalf("after the timeout-shaped report, Lookup = %v, want %v (never Blocked — this is the rc.7 regression)",
			got, ui.ReadinessUnknown)
	}

	cmd := prober.Probe(context.Background(), []sessionindex.Record{record})
	if cmd == nil {
		t.Fatal("Probe did not retry a row that was still reading Unknown")
	}
	drain(t, cmd)
	if got := prober.Lookup(record); got != ui.ReadinessReady {
		t.Fatalf("after the retry succeeded, Lookup = %v, want %v", got, ui.ReadinessReady)
	}
	if got := verifier.count(key); got != 2 {
		t.Fatalf("verify ran %d times, want exactly 2 (the timed-out attempt and the retry that succeeded)", got)
	}
}

// TestProbeNeverExceedsMaxConcurrentProbes proves the concurrency cap itself,
// not just that concurrent access is race-free (TestProbeIsSafeUnderConcurrency
// covers that): every command Probe hands back for a page of 25 records is run
// from its own goroutine at once, exactly like Bubble Tea would, and the
// number actually executing inside verify at any instant must never rise
// above MaxConcurrentProbes even though 25 commands started together — and
// every one of the 25 must still resolve once the cap lets it through.
func TestProbeNeverExceedsMaxConcurrentProbes(t *testing.T) {
	const recordCount = 25

	var current int32
	var peak int32
	started := make(chan struct{}, recordCount)
	gate := make(chan struct{})

	verifier := newFakeVerifier(func(sessionindex.Record) (preflight.Report, error) {
		n := atomic.AddInt32(&current, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if n <= p {
				break
			}
			if atomic.CompareAndSwapInt32(&peak, p, n) {
				break
			}
		}
		started <- struct{}{}
		<-gate // held open until the test has confirmed the first wave's size
		atomic.AddInt32(&current, -1)
		return preflight.Report{Decision: preflight.DecisionReady}, nil
	})
	prober := New(verifier.verify)
	records := resumableRecords(recordCount)

	commands := batchCommands(t, prober.Probe(context.Background(), records))
	if len(commands) != recordCount {
		t.Fatalf("Probe scheduled %d commands, want one per record (%d)", len(commands), recordCount)
	}

	messages := make(chan tea.Msg, recordCount)
	var group sync.WaitGroup
	for _, cmd := range commands {
		group.Add(1)
		go func(cmd tea.Cmd) {
			defer group.Done()
			messages <- cmd()
		}(cmd)
	}

	// Exactly MaxConcurrentProbes commands can acquire a slot and reach
	// started; the rest block inside Probe's own semaphore. Waiting on the
	// channel is deterministic — no sleep-and-poll — for the count that must
	// arrive; a short grace period then checks that a 7th never sneaks in.
	for i := 0; i < MaxConcurrentProbes; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of MaxConcurrentProbes (%d) probes had started", i, MaxConcurrentProbes)
		}
	}
	select {
	case <-started:
		t.Fatal("more than MaxConcurrentProbes probes were running at once")
	case <-time.After(50 * time.Millisecond):
	}
	if got := atomic.LoadInt32(&current); got != MaxConcurrentProbes {
		t.Fatalf("concurrent probes in flight = %d, want exactly %d", got, MaxConcurrentProbes)
	}

	close(gate)
	group.Wait()
	close(messages)

	if got := atomic.LoadInt32(&peak); got != MaxConcurrentProbes {
		t.Fatalf("peak concurrent probes = %d, want exactly %d", got, MaxConcurrentProbes)
	}

	seen := make(map[string]struct{}, recordCount)
	for msg := range messages {
		for _, key := range probedKeys(t, []tea.Msg{msg}) {
			seen[key] = struct{}{}
		}
	}
	if len(seen) != recordCount {
		t.Fatalf("%d distinct records were reported resolved, want %d", len(seen), recordCount)
	}
	for _, record := range records {
		if got := prober.Lookup(record); got != ui.ReadinessReady {
			t.Fatalf("Lookup(%q) = %v, want %v", record.Key, got, ui.ReadinessReady)
		}
	}
}

// TestUnresumableRecordsAreNeverProbed is a cost invariant as much as a
// correctness one. A read-only record cannot resume whatever a report says, so
// running a workspace and vendor-version probe for one is pure waste.
func TestUnresumableRecordsAreNeverProbed(t *testing.T) {
	tests := []struct {
		name   string
		record sessionindex.Record
	}{
		{
			name:   "read-only reason",
			record: sessionindex.Record{Key: "grok:4c2e9055", CanResume: true, ReadOnlyReason: "agent has no native resume"},
		},
		{
			name:   "cannot resume",
			record: sessionindex.Record{Key: "claude:0917bd6f"},
		},
		{
			name:   "both",
			record: sessionindex.Record{Key: "gemini:a13f7702", ReadOnlyReason: "source file is missing"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := readyVerifier()
			prober := New(verifier.verify)

			if got := prober.Lookup(test.record); got != ui.ReadinessBlocked {
				t.Fatalf("Lookup = %v, want %v", got, ui.ReadinessBlocked)
			}
			if got := verifier.total(); got != 0 {
				t.Fatalf("verify ran %d times for an unresumable record, want 0", got)
			}

			// Probe must skip it too, or the surface would pay for the probe on
			// the next keystroke instead of this one.
			if cmd := prober.Probe(context.Background(), []sessionindex.Record{test.record}); cmd != nil {
				drain(t, cmd)
				t.Fatal("Probe scheduled work for an unresumable record")
			}
			if got := verifier.total(); got != 0 {
				t.Fatalf("verify ran %d times after Probe, want 0", got)
			}
		})
	}
}

func TestLookupIsUnknownUntilAProbeCompletes(t *testing.T) {
	verifier := newFakeVerifier(func(sessionindex.Record) (preflight.Report, error) {
		return preflight.Report{Decision: preflight.DecisionConfirmationRequired}, nil
	})
	prober := New(verifier.verify)
	record := resumable("claude:5f1c0b2a")

	if got := prober.Lookup(record); got != ui.ReadinessUnknown {
		t.Fatalf("Lookup before probing = %v, want %v", got, ui.ReadinessUnknown)
	}

	cmd := prober.Probe(context.Background(), []sessionindex.Record{record})
	if cmd == nil {
		t.Fatal("Probe returned no command for an unprobed record")
	}
	// The command has not run yet, so the answer is still not known.
	if got := prober.Lookup(record); got != ui.ReadinessUnknown {
		t.Fatalf("Lookup with a probe in flight = %v, want %v", got, ui.ReadinessUnknown)
	}

	if keys := probedKeys(t, drain(t, cmd)); len(keys) != 1 || keys[0] != record.Key {
		t.Fatalf("probe reported keys %v, want [%q]", keys, record.Key)
	}
	if got := prober.Lookup(record); got != ui.ReadinessWarn {
		t.Fatalf("Lookup after probing = %v, want %v", got, ui.ReadinessWarn)
	}
	if got := verifier.count(record.Key); got != 1 {
		t.Fatalf("verify ran %d times, want 1", got)
	}
}

// TestProbeIsANoOpWhenThereIsNothingToDo covers every reason the prober has to
// decline work. A surface calls Probe on every keystroke, so returning a command
// that re-verifies settled rows would make typing quadratically expensive.
func TestProbeIsANoOpWhenThereIsNothingToDo(t *testing.T) {
	records := resumableRecords(3)

	t.Run("disabled prober", func(t *testing.T) {
		prober := New(nil)
		if cmd := prober.Probe(context.Background(), records); cmd != nil {
			t.Fatal("a prober with no verifier scheduled work")
		}
	})

	t.Run("nil prober", func(t *testing.T) {
		var prober *Prober
		if cmd := prober.Probe(context.Background(), records); cmd != nil {
			t.Fatal("a nil prober scheduled work")
		}
	})

	t.Run("no records", func(t *testing.T) {
		verifier := readyVerifier()
		prober := New(verifier.verify)
		if cmd := prober.Probe(context.Background(), nil); cmd != nil {
			t.Fatal("Probe over no records scheduled work")
		}
		if cmd := prober.Probe(context.Background(), []sessionindex.Record{}); cmd != nil {
			t.Fatal("Probe over an empty slice scheduled work")
		}
	})

	t.Run("every record already cached", func(t *testing.T) {
		verifier := readyVerifier()
		prober := New(verifier.verify)
		drain(t, prober.Probe(context.Background(), records))
		if got := verifier.total(); got != len(records) {
			t.Fatalf("verify ran %d times, want %d", got, len(records))
		}

		if cmd := prober.Probe(context.Background(), records); cmd != nil {
			t.Fatal("Probe re-scheduled work for records that already have an answer")
		}
		if got := verifier.total(); got != len(records) {
			t.Fatalf("verify ran %d times after the second Probe, want %d", got, len(records))
		}
	})

	t.Run("every record already in flight", func(t *testing.T) {
		verifier := readyVerifier()
		prober := New(verifier.verify)

		first := prober.Probe(context.Background(), records)
		if first == nil {
			t.Fatal("the first Probe scheduled no work")
		}
		// Deliberately do not run it: the records are claimed, not answered.
		if second := prober.Probe(context.Background(), records); second != nil {
			t.Fatal("Probe scheduled a duplicate probe for records already in flight")
		}

		drain(t, first)
		for _, record := range records {
			if got := verifier.count(record.Key); got != 1 {
				t.Fatalf("verify ran %d times for %q, want exactly 1", got, record.Key)
			}
		}
	})

	t.Run("a mixed page probes only what is missing", func(t *testing.T) {
		verifier := readyVerifier()
		prober := New(verifier.verify)
		drain(t, prober.Probe(context.Background(), records[:1]))

		fresh := resumable("codex:9b7d4118")
		page := append(append([]sessionindex.Record{}, records...), fresh)
		keys := probedKeys(t, drain(t, prober.Probe(context.Background(), page)))
		if len(keys) != len(records) {
			t.Fatalf("probed %v, want the %d records without an answer", keys, len(records))
		}
		if got := verifier.count(records[0].Key); got != 1 {
			t.Fatalf("the already-answered record was probed %d times, want 1", got)
		}
		if got := verifier.count(fresh.Key); got != 1 {
			t.Fatalf("the new record was probed %d times, want 1", got)
		}
	})
}

// TestProbeIsSafeUnderConcurrency runs every command the prober hands back from
// its own goroutine, which is precisely what Bubble Tea does with a batch. The
// prober guards its cache with a mutex; this is the test that proves it, and it
// is only meaningful under `go test -race`.
func TestProbeIsSafeUnderConcurrency(t *testing.T) {
	const (
		recordCount = 50
		readers     = 8
	)
	verifier := readyVerifier()
	prober := New(verifier.verify)
	records := resumableRecords(recordCount)

	commands := batchCommands(t, prober.Probe(context.Background(), records))
	if len(commands) != recordCount {
		t.Fatalf("Probe scheduled %d commands, want one per record (%d)", len(commands), recordCount)
	}

	// Every goroutine waits on the same gate so the probes overlap rather than
	// happening to serialise themselves by starting late.
	gate := make(chan struct{})
	messages := make(chan tea.Msg, recordCount)
	var group sync.WaitGroup

	for _, cmd := range commands {
		group.Add(1)
		go func(cmd tea.Cmd) {
			defer group.Done()
			<-gate
			messages <- cmd()
		}(cmd)
	}
	// Readers model the surface redrawing while probes land.
	for reader := 0; reader < readers; reader++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-gate
			for _, record := range records {
				prober.Lookup(record)
				prober.Report(record.Key)
				prober.Enabled()
			}
		}()
	}

	close(gate)
	group.Wait()
	close(messages)

	seen := make(map[string]int, recordCount)
	for msg := range messages {
		for _, key := range probedKeys(t, []tea.Msg{msg}) {
			seen[key]++
		}
	}
	if len(seen) != recordCount {
		t.Fatalf("%d distinct records were reported, want %d", len(seen), recordCount)
	}
	for _, record := range records {
		if got := seen[record.Key]; got != 1 {
			t.Fatalf("record %q was reported %d times, want exactly 1", record.Key, got)
		}
		if got := verifier.count(record.Key); got != 1 {
			t.Fatalf("verify ran %d times for %q, want exactly 1", got, record.Key)
		}
		if got := prober.Lookup(record); got != ui.ReadinessReady {
			t.Fatalf("Lookup(%q) = %v, want %v", record.Key, got, ui.ReadinessReady)
		}
	}
	if got := verifier.total(); got != recordCount {
		t.Fatalf("verify ran %d times in total, want %d", got, recordCount)
	}

	prober.mu.Lock()
	cached, pending := len(prober.cache), len(prober.pending)
	prober.mu.Unlock()
	if cached != recordCount {
		t.Fatalf("cache holds %d results, want %d", cached, recordCount)
	}
	if pending != 0 {
		t.Fatalf("%d probes are still marked in flight after every command finished", pending)
	}

	// A settled prober asks for nothing more.
	if cmd := prober.Probe(context.Background(), records); cmd != nil {
		t.Fatal("Probe scheduled work after every record was answered")
	}
}

// TestProbeToleratesANilContext locks down the guard in Probe. A nil context
// reaches the index as a nil argument and panics inside database/sql; probing is
// best-effort background work and must never take the surface down with it.
func TestProbeToleratesANilContext(t *testing.T) {
	verifier := newFakeVerifier(func(sessionindex.Record) (preflight.Report, error) {
		return preflight.Report{Decision: preflight.DecisionReady}, nil
	})
	prober := New(verifier.verify)
	record := resumable("claude:5f1c0b2a")

	//nolint:staticcheck // passing nil is exactly the case under test.
	cmd := prober.Probe(nil, []sessionindex.Record{record})
	if cmd == nil {
		t.Fatal("Probe with a nil context declined to probe")
	}
	if keys := probedKeys(t, drain(t, cmd)); len(keys) != 1 || keys[0] != record.Key {
		t.Fatalf("probe reported keys %v, want [%q]", keys, record.Key)
	}
	if got := prober.Lookup(record); got != ui.ReadinessReady {
		t.Fatalf("Lookup = %v, want %v", got, ui.ReadinessReady)
	}
	if got := verifier.count(record.Key); got != 1 {
		t.Fatalf("verify ran %d times, want 1", got)
	}
}

// TestNilContextReachesTheVerifierUsable proves the substitute context is a real
// one, not the nil that was handed in.
func TestNilContextReachesTheVerifierUsable(t *testing.T) {
	var seen context.Context
	prober := New(func(ctx context.Context, _ sessionindex.Record) (preflight.Report, error) {
		seen = ctx
		return preflight.Report{Decision: preflight.DecisionReady}, nil
	})
	//nolint:staticcheck // passing nil is exactly the case under test.
	drain(t, prober.Probe(nil, []sessionindex.Record{resumable("claude:5f1c0b2a")}))

	if seen == nil {
		t.Fatal("the verifier was handed a nil context")
	}
	if err := seen.Err(); err != nil {
		t.Fatalf("the substitute context is already done: %v", err)
	}
}

// TestReportIsWithheldUntilASuccessfulProbe keeps a failed probe from being
// mistaken for a clean report. A caller that opened a checklist on a zero report
// would show a user no warnings at all and call that an answer.
func TestReportIsWithheldUntilASuccessfulProbe(t *testing.T) {
	const (
		goodKey = "claude:5f1c0b2a"
		badKey  = "codex:9b7d4118"
	)
	want := preflight.Report{SchemaVersion: preflight.SchemaVersion, SessionRef: goodKey, Decision: preflight.DecisionReady}
	verifier := newFakeVerifier(func(record sessionindex.Record) (preflight.Report, error) {
		if record.Key == badKey {
			return preflight.Report{Decision: preflight.DecisionReady}, errProbe
		}
		return want, nil
	})
	prober := New(verifier.verify)

	t.Run("unknown key", func(t *testing.T) {
		if _, ok := prober.Report("gemini:never-probed"); ok {
			t.Fatal("Report answered for a record that was never probed")
		}
	})

	drain(t, prober.Probe(context.Background(), []sessionindex.Record{resumable(goodKey), resumable(badKey)}))

	t.Run("successful probe", func(t *testing.T) {
		got, ok := prober.Report(goodKey)
		if !ok {
			t.Fatal("Report withheld a successfully computed report")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Report = %+v, want %+v", got, want)
		}
		if readiness := prober.Lookup(resumable(goodKey)); readiness != ui.ReadinessReady {
			t.Fatalf("Lookup = %v, want %v", readiness, ui.ReadinessReady)
		}
	})

	t.Run("failed probe", func(t *testing.T) {
		got, ok := prober.Report(badKey)
		if ok {
			t.Fatalf("Report answered %+v for a probe that failed", got)
		}
		if !reflect.DeepEqual(got, preflight.Report{}) {
			t.Fatalf("Report returned %+v alongside ok=false, want the zero report", got)
		}
		// The row still renders, it just renders as not known.
		if readiness := prober.Lookup(resumable(badKey)); readiness != ui.ReadinessUnknown {
			t.Fatalf("Lookup after a failed probe = %v, want %v", readiness, ui.ReadinessUnknown)
		}
	})

	t.Run("nil prober", func(t *testing.T) {
		var nilProber *Prober
		if got, ok := nilProber.Report(goodKey); ok || !reflect.DeepEqual(got, preflight.Report{}) {
			t.Fatalf("nil Report = %+v, %v; want the zero report and false", got, ok)
		}
	})
}

// TestEnabledReportsWhetherReadinessIsComputedAtAll matters because a surface
// hides the whole status column when it is false, rather than drawing a column
// of placeholders that will never resolve.
func TestEnabledReportsWhetherReadinessIsComputedAtAll(t *testing.T) {
	tests := []struct {
		name   string
		prober *Prober
		want   bool
	}{
		{name: "nil prober", prober: nil, want: false},
		{name: "no verifier", prober: New(nil), want: false},
		{name: "configured", prober: New(readyVerifier().verify), want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.prober.Enabled(); got != test.want {
				t.Fatalf("Enabled = %v, want %v", got, test.want)
			}
		})
	}

	// A nil prober is a legitimate state, not a programming error: the surface
	// holds one whether or not a verifier was configured, and asks it on every
	// row it draws.
	t.Run("a nil prober answers rather than panicking", func(t *testing.T) {
		var prober *Prober
		if got := prober.Lookup(resumable("claude:5f1c0b2a")); got != ui.ReadinessUnknown {
			t.Fatalf("nil Lookup = %v, want %v", got, ui.ReadinessUnknown)
		}
		// Even for a record that is plainly unresumable: nothing is known,
		// because nothing is computing anything.
		blocked := sessionindex.Record{Key: "grok:4c2e9055", ReadOnlyReason: "agent has no native resume"}
		if got := prober.Lookup(blocked); got != ui.ReadinessUnknown {
			t.Fatalf("nil Lookup of a read-only record = %v, want %v", got, ui.ReadinessUnknown)
		}
	})

	// A disabled prober is asked the same questions and must answer the same way.
	t.Run("a disabled prober still answers Lookup", func(t *testing.T) {
		prober := New(nil)
		if got := prober.Lookup(resumable("claude:5f1c0b2a")); got != ui.ReadinessUnknown {
			t.Fatalf("Lookup = %v, want %v", got, ui.ReadinessUnknown)
		}
	})
}
