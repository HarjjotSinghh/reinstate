package preflight

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

func FuzzAuthorizeWarningIDs(f *testing.F) {
	f.Add("baseline.unavailable", "git.branch")
	f.Add("*", "baseline.unavailable")
	f.Add("git.branch", "git.branch")
	f.Add("stale.warning", "")
	f.Add(" baseline.unavailable ", "\tgit.branch\n")
	f.Fuzz(func(t *testing.T, first, second string) {
		report := validPolicyReport([]Check{
			{ID: "baseline.unavailable", Status: StatusUnknown, Severity: SeverityWarning, Provenance: workspace.ProvenanceUnavailable, ExitCode: exitcode.Safety},
			{ID: "git.branch", Status: StatusChanged, Severity: SeverityWarning, Provenance: workspace.ProvenanceCurrentObservation, ExitCode: exitcode.Safety},
			{ID: "agent.version", Status: StatusMatch, Severity: SeverityInfo, Provenance: workspace.ProvenanceCurrentObservation},
		})
		authorization, err := Authorize(report, []string{first, second})
		first = strings.TrimSpace(first)
		second = strings.TrimSpace(second)
		exact := first == "baseline.unavailable" && second == "git.branch" ||
			first == "git.branch" && second == "baseline.unavailable"
		if exact {
			if err != nil || !authorization.Allowed {
				t.Fatalf("exact set refused: %+v %v", authorization, err)
			}
			return
		}
		if err == nil || authorization.Allowed || authorization.ExitCode != exitcode.Usage && authorization.ExitCode != exitcode.Safety {
			t.Fatalf("non-exact set authorized: %q %q => %+v %v", first, second, authorization, err)
		}
	})
}
