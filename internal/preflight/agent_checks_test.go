package preflight

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
)

func TestAgentChecksTimedOutProbeIsUntestedCompatibility(t *testing.T) {
	t.Parallel()
	checks := agentChecks(agentcheck.Result{
		Status:            agentcheck.StatusError,
		Message:           "version probe timed out",
		LayoutRecognized:  true,
		ExecutablePresent: true,
		Version:           "",
	}, false)
	version := findAgentCheck(t, checks, "agent.version")
	if version.Status != StatusUnknown || version.Severity != SeverityBlock || version.ExitCode != exitcode.Compatibility {
		t.Fatalf("timed-out probe check = %+v, want unknown/block/compatibility", version)
	}
	if !strings.Contains(version.Message, "timed out") {
		t.Fatalf("timed-out probe message = %q", version.Message)
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
