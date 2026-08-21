package preflight

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// timedVersionRunner answers only after delay, and reports how much of its own
// budget it was actually given.
type timedVersionRunner struct {
	delay  time.Duration
	output agentcheck.VersionOutput
	budget chan time.Duration
}

func (runner *timedVersionRunner) Version(
	ctx context.Context, _ string, _ ...string,
) (agentcheck.VersionOutput, error) {
	if deadline, ok := ctx.Deadline(); ok {
		select {
		case runner.budget <- time.Until(deadline):
		default:
		}
	}
	select {
	case <-time.After(runner.delay):
		return runner.output, nil
	case <-ctx.Done():
		return agentcheck.VersionOutput{}, ctx.Err()
	}
}

// TestVersionProbeGetsTheWholeWindow is the regression this file exists for.
//
// Every observer shares one wall clock — the verifier's context — but the
// workspace probe runs first and shells out to Git. Run in sequence, the
// version probe inherited whatever the workspace probe left, so on a loaded
// host it was routinely given a fraction of its stated budget, timed out, and
// reported an installed agent as unmeasurable. "No version" is
// indistinguishable from "no agent" to a caller gating on it, so the user got a
// refusal they could do nothing about, from a probe that was never really run.
func TestVersionProbeGetsTheWholeWindow(t *testing.T) {
	t.Parallel()

	const window = 400 * time.Millisecond
	// Long enough that the leftover after it would have starved the version
	// probe, short enough to leave the window genuinely usable.
	const workspaceCost = window * 3 / 4

	fixture := newFixture(t, "https://example.com/org/repo.git")
	slowGit := fixture.options.Workspace.Runner
	fixture.options.Workspace.Runner = workspace.GitRunnerFunc(
		func(ctx context.Context, dir string, args ...string) ([]byte, error) {
			if reflect.DeepEqual(args, []string{"rev-parse", "--path-format=absolute", "--show-toplevel"}) {
				select {
				case <-time.After(workspaceCost):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return slowGit.Run(ctx, dir, args...)
		})

	runner := &timedVersionRunner{
		delay:  window / 2,
		output: agentcheck.VersionOutput{Stdout: "2.1.220 (Claude Code)"},
		budget: make(chan time.Duration, 1),
	}
	fixture.options.Timeout = window
	fixture.options.Agent.Runner = runner

	report, err := Verify(context.Background(), Input{
		SessionRef: "claude:controlled", Agent: "claude", Workspace: fixture.workspace,
		SourceFresh: true,
	}, fixture.options)
	if err != nil {
		t.Fatal(err)
	}

	// The probe must have been handed materially more than what the workspace
	// probe left behind, which is the whole point.
	var granted time.Duration
	select {
	case granted = <-runner.budget:
	default:
		t.Fatal("the version probe never ran")
	}
	if leftover := window - workspaceCost; granted <= leftover {
		t.Fatalf("version probe was granted %s; the workspace probe left only %s, "+
			"so it is still inheriting the remainder instead of the window", granted, leftover)
	}

	if report.Agent.Version != "2.1.220" {
		t.Fatalf("agent version = %q, want the measured version; status=%s message=%q",
			report.Agent.Version, report.Agent.Status, report.Agent.Message)
	}
	if check := findCheck(t, report, "agent.version"); check.Severity == SeverityBlock {
		t.Fatalf("a measurable agent was reported as unmeasurable: %+v", check)
	}
}

// TestVersionProbeStillHonoursTheWindow guards the other direction: overlapping
// the probe must not let it outlive the shared deadline.
func TestVersionProbeStillHonoursTheWindow(t *testing.T) {
	t.Parallel()

	const window = 100 * time.Millisecond
	fixture := newFixture(t, "https://example.com/org/repo.git")
	fixture.options.Timeout = window
	fixture.options.Agent.Runner = &timedVersionRunner{
		delay:  time.Hour,
		budget: make(chan time.Duration, 1),
	}

	started := time.Now()
	report, err := Verify(context.Background(), Input{
		SessionRef: "claude:controlled", Agent: "claude", Workspace: fixture.workspace,
		SourceFresh: true,
	}, fixture.options)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 20*window {
		t.Fatalf("Verify() took %s with a %s window; the probe escaped the shared deadline",
			elapsed, window)
	}
	if err == nil && report.Decision != DecisionBlocked {
		t.Fatalf("an unmeasurable agent produced decision %q, want blocked", report.Decision)
	}
}
