package preflight

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

func TestWorkspaceExpectationFallsBackPerFieldToVendorRecording(t *testing.T) {
	t.Parallel()
	baselineBranch := "baseline-branch"
	recordedRepository := "remote-sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	recordedHead := "1111111111111111111111111111111111111111"

	got := workspaceExpectation(Input{
		Workspace: "/controlled/workspace",
		Baseline:  &environment.PrelaunchBaseline{Branch: baselineBranch},
		Recorded: environment.RecordedEnvironment{
			RepositoryID: environment.RecordedField{Value: recordedRepository},
			Branch:       environment.RecordedField{Value: "recorded-branch"},
			GitHead:      environment.RecordedField{Value: recordedHead},
		},
	})

	if got.RepositoryID == nil || got.RepositoryID.Value != recordedRepository || got.RepositoryID.Provenance != workspace.ProvenanceVendorRecorded {
		t.Fatalf("repository fallback = %+v", got.RepositoryID)
	}
	if got.Branch == nil || got.Branch.Value != baselineBranch || got.Branch.Provenance != workspace.ProvenanceReinstatePrelaunchObserved {
		t.Fatalf("baseline branch = %+v", got.Branch)
	}
	if got.Head == nil || got.Head.Value != recordedHead || got.Head.Provenance != workspace.ProvenanceVendorRecorded {
		t.Fatalf("head fallback = %+v", got.Head)
	}
}
