package preflight

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
)

func TestRuntimeBaselineIncludesDeclarationSourceAndNewRuntimes(t *testing.T) {
	t.Parallel()
	baseline := &environment.PrelaunchBaseline{Runtimes: []environment.Runtime{{
		Name: "node", Version: "22.12.0", SourceKind: "nvmrc",
		Provenance: environment.PrelaunchObservedProvenance,
	}}}

	changedSource := runtimeChecks(Input{Baseline: baseline}, []runtimecheck.Result{{
		Name: "node", Actual: "22.12.0", SourceKind: "package_json", Status: runtimecheck.StatusMatch,
	}})
	if check := checkWithID(t, changedSource, "runtime.node.baseline"); check.Status != StatusChanged || check.Severity != SeverityWarning {
		t.Fatalf("same version from changed declaration source = %+v", check)
	}

	newRuntime := runtimeChecks(Input{Baseline: baseline}, []runtimecheck.Result{
		{Name: "node", Actual: "22.12.0", SourceKind: "nvmrc", Status: runtimecheck.StatusMatch},
		{Name: "go", Actual: "1.25.12", SourceKind: "go_mod", Status: runtimecheck.StatusMatch},
	})
	if check := checkWithID(t, newRuntime, "runtime.go.baseline"); check.Status != StatusChanged || check.Severity != SeverityWarning {
		t.Fatalf("new runtime declaration = %+v", check)
	}
	if check := checkWithID(t, newRuntime, "runtime.node.baseline"); check.Status != StatusMatch || check.Severity != SeverityInfo {
		t.Fatalf("unchanged runtime declaration = %+v", check)
	}
}

func checkWithID(t *testing.T, checks []Check, id string) Check {
	t.Helper()
	for _, check := range checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("missing check %q in %+v", id, checks)
	return Check{}
}
