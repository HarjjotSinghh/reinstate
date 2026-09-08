package preflight

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
)

// TestAgentChecksTimedOutProbeIsUninspectableRuntime is the CLI experience
// row 13 regression pin for agentChecks itself (the v0.6.0-rc.8 follow-on:
// see readiness.uninspectable's doc comment). A version probe that ran out
// of its own time budget (agentcheck.Result.TimedOut) established nothing
// about the executable, favorable or not, so the resulting check must carry
// exitcode.Runtime — the "could not produce trustworthy evidence" code — not
// exitcode.Compatibility, which readiness.uninspectable (and any other
// caller distinguishing could-not-evaluate from a genuine finding) would
// otherwise be unable to tell apart from the sibling case in
// TestAgentChecksDeterministicallyFailedProbeIsBlockCompatibility below.
func TestAgentChecksTimedOutProbeIsUninspectableRuntime(t *testing.T) {
	t.Parallel()
	for _, readOnly := range []bool{false, true} {
		checks := agentChecks(agentcheck.Result{
			Status:            agentcheck.StatusError,
			Message:           "version probe timed out",
			LayoutRecognized:  true,
			ExecutablePresent: true,
			Version:           "",
			TimedOut:          true,
		}, readOnly)
		version := findAgentCheck(t, checks, "agent.version")
		if version.Status != StatusUnknown || version.Severity != SeverityBlock || version.ExitCode != exitcode.Runtime {
			t.Fatalf("readOnly=%t timed-out probe check = %+v, want unknown/block/runtime", readOnly, version)
		}
		if !strings.Contains(version.Message, "timed out") {
			t.Fatalf("timed-out probe message = %q", version.Message)
		}
	}
}

// TestAgentChecksDeterministicallyFailedProbeIsBlockCompatibility is the
// other half of the row 13 follow-on pin: a version probe that failed for a
// reason unrelated to the clock — the executable's identity could not be
// captured (corrupted, deleted, or otherwise not launchable), it exited with
// a real error, or it was swapped out from under the probe — reproduces
// identically on every retry. This is a genuine, actionable finding and must
// resolve to a real Blocked, not to readiness's "still checking".
//
// Status stays Unknown here, same as the timed-out case above — Status
// StatusError is reserved system-wide for "the check's own machinery failed"
// and validCheckExit enforces that it always pairs with ExitCode Runtime, so
// a genuine agent-compatibility finding cannot use it. ExitCode
// (Compatibility, not Runtime) is the only, and sufficient, signal
// readiness.uninspectable needs to tell this apart from the timed-out case:
// before the exit-code split this test locks in, both cases carried
// ExitCode exitcode.Compatibility too, which is what let a permanently
// broken agent install read as merely inconclusive.
func TestAgentChecksDeterministicallyFailedProbeIsBlockCompatibility(t *testing.T) {
	t.Parallel()
	for _, readOnly := range []bool{false, true} {
		checks := agentChecks(agentcheck.Result{
			Status:            agentcheck.StatusError,
			Message:           "native agent executable identity is unavailable",
			LayoutRecognized:  true,
			ExecutablePresent: true,
			Version:           "",
			TimedOut:          false,
		}, readOnly)
		version := findAgentCheck(t, checks, "agent.version")
		if version.Status != StatusUnknown || version.Severity != SeverityBlock || version.ExitCode != exitcode.Compatibility {
			t.Fatalf("readOnly=%t deterministically-failed probe check = %+v, want unknown/block/compatibility", readOnly, version)
		}
		if !strings.Contains(version.Message, "identity is unavailable") {
			t.Fatalf("deterministically-failed probe message = %q", version.Message)
		}
	}
}

func TestAgentChecksReadOnlyDeterminedUntestedIsCompatibilityBlock(t *testing.T) {
	t.Parallel()
	checks := agentChecks(agentcheck.Result{
		Status:            agentcheck.StatusUntested,
		Message:           "native agent version is outside the verified range",
		LayoutRecognized:  true,
		ExecutablePresent: true,
		Version:           "2.1.239",
	}, true)
	version := findAgentCheck(t, checks, "agent.version")
	if version.Status != StatusUnknown || version.Severity != SeverityBlock || version.ExitCode != exitcode.Compatibility {
		t.Fatalf("determined untested read-only check = %+v, want unknown/block/compatibility", version)
	}
}

func TestAgentChecksReadOnlySourceOnlyAgentIsInformational(t *testing.T) {
	t.Parallel()
	checks := agentChecks(agentcheck.Result{
		Status:  agentcheck.StatusUntested,
		Message: "agent does not support native verified resume",
		Agent:   "grok",
	}, true)
	for _, id := range []string{"agent.executable", "agent.layout", "agent.version"} {
		check := findAgentCheck(t, checks, id)
		if check.Severity != SeverityInfo {
			t.Fatalf("%s = %+v, want informational for source-only read-only handoff", id, check)
		}
	}
}

func TestAgentChecksReadOnlyMissingExecutableIsInformational(t *testing.T) {
	t.Parallel()
	checks := agentChecks(agentcheck.Result{
		Status:            agentcheck.StatusNotInstalled,
		LayoutRecognized:  true,
		ExecutablePresent: false,
	}, true)
	present := findAgentCheck(t, checks, "agent.executable")
	if present.Status != StatusMissing || present.Severity != SeverityInfo {
		t.Fatalf("read-only missing executable = %+v", present)
	}
	version := findAgentCheck(t, checks, "agent.version")
	if version.Severity != SeverityInfo {
		t.Fatalf("read-only missing version = %+v", version)
	}
}

func TestAgentChecksLayoutOnlySupportedDoesNotClaimVerifiedRange(t *testing.T) {
	t.Parallel()
	checks := agentChecks(agentcheck.Result{
		Status:            agentcheck.StatusSupported,
		LayoutRecognized:  true,
		ExecutablePresent: false,
	}, true)
	version := findAgentCheck(t, checks, "agent.version")
	if version.Status != StatusUnknown || version.Severity != SeverityInfo {
		t.Fatalf("layout-only supported version = %+v", version)
	}
	if strings.Contains(version.Message, "verified range") {
		t.Fatalf("layout-only supported claimed a verified range: %q", version.Message)
	}
	present := findAgentCheck(t, checks, "agent.executable")
	if present.Severity != SeverityInfo {
		t.Fatalf("layout-only supported executable = %+v", present)
	}
}

func TestAgentChecksDestinationMissingExecutableStillBlocks(t *testing.T) {
	t.Parallel()
	checks := agentChecks(agentcheck.Result{
		Status:            agentcheck.StatusSupported,
		LayoutRecognized:  true,
		ExecutablePresent: false,
	}, false)
	present := findAgentCheck(t, checks, "agent.executable")
	if present.Status != StatusMissing || present.Severity != SeverityBlock || present.ExitCode != exitcode.Compatibility {
		t.Fatalf("destination missing executable = %+v", present)
	}
}

func findAgentCheck(t *testing.T, checks []Check, id string) Check {
	t.Helper()
	for _, check := range checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("missing check %s in %+v", id, checks)
	return Check{}
}
