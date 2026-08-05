package preflight

import (
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

func TestRuntimeBaselineIncludesDeclarationSourceAndNewRuntimes(t *testing.T) {
	t.Parallel()
	baseline := &environment.PrelaunchBaseline{Runtimes: []environment.Runtime{{
		Name: "node", Declared: "22.12.0", Version: "22.12.0", SourceKind: "nvmrc",
		Provenance: environment.PrelaunchObservedProvenance,
	}}}

	changedSource := runtimeChecks(Input{Baseline: baseline}, []runtimecheck.Result{{
		Name: "node", Declared: "22.12.0", Actual: "22.12.0", SourceKind: "package_json", Status: runtimecheck.StatusMatch,
	}})
	if check := checkWithID(t, changedSource, "runtime.node.baseline"); check.Status != StatusChanged || check.Severity != SeverityWarning {
		t.Fatalf("same version from changed declaration source = %+v", check)
	}

	newRuntime := runtimeChecks(Input{Baseline: baseline}, []runtimecheck.Result{
		{Name: "node", Declared: "22.12.0", Actual: "22.12.0", SourceKind: "nvmrc", Status: runtimecheck.StatusMatch},
		{Name: "go", Declared: "1.25.12", Actual: "1.25.12", SourceKind: "go_mod", Status: runtimecheck.StatusMatch},
	})
	if check := checkWithID(t, newRuntime, "runtime.go.baseline"); check.Status != StatusChanged || check.Severity != SeverityWarning {
		t.Fatalf("new runtime declaration = %+v", check)
	}
	if check := checkWithID(t, newRuntime, "runtime.node.baseline"); check.Status != StatusMatch || check.Severity != SeverityInfo {
		t.Fatalf("unchanged runtime declaration = %+v", check)
	}
}

func TestRuntimeBaselineDetectsDeclarationOnlyDrift(t *testing.T) {
	t.Parallel()
	baseline := &environment.PrelaunchBaseline{Runtimes: []environment.Runtime{{
		Name: "node", Declared: ">=22 <23", Version: "22.12.0", SourceKind: "package_json",
		Provenance: environment.PrelaunchObservedProvenance,
	}}}

	checks := runtimeChecks(Input{Baseline: baseline}, []runtimecheck.Result{{
		Name: "node", Declared: ">=22", Actual: "22.12.0", SourceKind: "package_json", Status: runtimecheck.StatusMatch,
	}})
	check := checkWithID(t, checks, "runtime.node.baseline")
	if check.Status != StatusChanged || check.Severity != SeverityWarning || check.ExitCode != exitcode.Safety {
		t.Fatalf("declaration-only drift = %+v", check)
	}
}

func TestBaselineFromReportPersistsRuntimeDeclaration(t *testing.T) {
	t.Parallel()
	report := validPolicyReport([]Check{{
		ID: "runtime.node.declaration", Status: StatusMatch, Severity: SeverityInfo,
		Provenance: workspace.ProvenanceCurrentObservation,
	}})
	report.Workspace.Git.WorkingTree.State = workspace.WorkingTreeUnavailable
	report.Runtimes = []runtimecheck.Result{{
		Name: "node", Declared: ">=22 <23", Actual: "22.12.0", SourceKind: "package_json", Status: runtimecheck.StatusMatch,
	}}

	baseline, err := BaselineFromReport(report, time.Date(2026, 8, 5, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.Runtimes) != 1 || baseline.Runtimes[0].Declared != ">=22 <23" {
		t.Fatalf("runtime declaration was not persisted: %+v", baseline.Runtimes)
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
