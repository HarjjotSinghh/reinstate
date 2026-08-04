package workspace

import (
	"strings"
	"testing"
)

func trustedString(value string) *ExpectedString {
	return &ExpectedString{Value: value, Provenance: ProvenanceVendorRecorded}
}

func baseFingerprint() Fingerprint {
	head := strings.Repeat("a", 40)
	return Fingerprint{
		SchemaVersion: SchemaVersion,
		Provenance:    ProvenanceCurrentObservation,
		Workspace:     WorkspaceFingerprint{Path: "/work/repo", Exists: true, Directory: true},
		Git: GitFingerprint{
			Available: true, Repository: true, Root: "/work/repo",
			RepositoryID: "remote-sha256:one", repositoryIDs: []string{"remote-sha256:one"},
			Branch: "main", Head: head,
			WorkingTree:          WorkingTreeFingerprint{State: WorkingTreeClean},
			ExpectedHeadRelation: relationFromCounts(0, 0),
		},
	}
}

func TestCompareUsesOnlyTrustworthyProvenance(t *testing.T) {
	t.Parallel()
	fingerprint := baseFingerprint()
	expected := Expectation{
		Workspace:    trustedString("/work/repo"),
		RepositoryID: trustedString(fingerprint.Git.RepositoryID),
		Branch:       trustedString("main"),
		Head:         trustedString(fingerprint.Git.Head),
	}
	comparison := Compare(expected, fingerprint)
	if comparison.Decision != DecisionReady {
		t.Fatalf("decision = %s: %+v", comparison.Decision, comparison.Checks)
	}
	if checkByID(t, comparison, "git.repository").Status != StatusMatch ||
		checkByID(t, comparison, "git.branch").Status != StatusMatch ||
		checkByID(t, comparison, "git.head").Status != StatusMatch {
		t.Fatalf("trusted values did not match: %+v", comparison.Checks)
	}
	if tree := checkByID(t, comparison, "git.working_tree"); tree.Status != StatusUnknown || tree.Severity != SeverityInfo {
		t.Fatalf("missing historical tree baseline was manufactured: %+v", tree)
	}
}

func TestCompareNeverTreatsCurrentObservationAsHistorical(t *testing.T) {
	t.Parallel()
	fingerprint := baseFingerprint()
	comparison := Compare(Expectation{
		Branch: &ExpectedString{Value: "main", Provenance: ProvenanceCurrentObservation},
	}, fingerprint)
	branch := checkByID(t, comparison, "git.branch")
	if branch.Status != StatusUnknown || branch.Provenance != ProvenanceCurrentObservation {
		t.Fatalf("live observation became historical match: %+v", branch)
	}
}

func TestCompareTrustsExplicitPrelaunchBaselineAndTreeDigest(t *testing.T) {
	t.Parallel()
	fingerprint := baseFingerprint()
	fingerprint.Git.WorkingTree.Digest = "sha256:" + strings.Repeat("c", 64)
	comparison := Compare(Expectation{
		Branch: &ExpectedString{Value: "main", Provenance: ProvenanceReinstatePrelaunchObserved},
		WorkingTreeDigest: &ExpectedString{
			Value:      fingerprint.Git.WorkingTree.Digest,
			Provenance: ProvenanceReinstatePrelaunchObserved,
		},
	}, fingerprint)
	if branch := checkByID(t, comparison, "git.branch"); branch.Status != StatusMatch {
		t.Fatalf("prelaunch branch = %+v", branch)
	}
	if tree := checkByID(t, comparison, "git.working_tree"); tree.Status != StatusMatch {
		t.Fatalf("prelaunch tree = %+v", tree)
	}
}

func TestCompareNonGitWithoutBaselineWarnsButGitExpectationBlocks(t *testing.T) {
	t.Parallel()
	fingerprint := Fingerprint{
		Workspace: WorkspaceFingerprint{Path: "/work", Exists: true, Directory: true},
		Git:       GitFingerprint{WorkingTree: WorkingTreeFingerprint{State: WorkingTreeUnavailable}},
	}
	without := Compare(Expectation{}, fingerprint)
	if without.Decision != DecisionConfirmationRequired || checkByID(t, without, "git.working_tree").Severity != SeverityWarning {
		t.Fatalf("non-Git without baseline = %+v", without)
	}
	with := Compare(Expectation{Branch: trustedString("main")}, fingerprint)
	if with.Decision != DecisionBlocked || checkByID(t, with, "git.branch").Severity != SeverityBlock {
		t.Fatalf("non-Git with baseline = %+v", with)
	}
}

func TestCompareRepositoryMismatchBlocksAndBranchDriftWarns(t *testing.T) {
	t.Parallel()
	fingerprint := baseFingerprint()
	fingerprint.Git.Branch = "feature/other"
	comparison := Compare(Expectation{
		RepositoryID: trustedString("remote-sha256:different"),
		Branch:       trustedString("main"),
	}, fingerprint)
	if comparison.Decision != DecisionBlocked {
		t.Fatalf("decision = %s", comparison.Decision)
	}
	if repository := checkByID(t, comparison, "git.repository"); repository.Status != StatusChanged || repository.Severity != SeverityBlock {
		t.Fatalf("repository check = %+v", repository)
	}
	if branch := checkByID(t, comparison, "git.branch"); branch.Status != StatusChanged || branch.Severity != SeverityWarning {
		t.Fatalf("branch check = %+v", branch)
	}
}

func TestCompareDirtyTreeRequiresConfirmationWithoutClaimingDrift(t *testing.T) {
	t.Parallel()
	fingerprint := baseFingerprint()
	fingerprint.Git.WorkingTree = WorkingTreeFingerprint{State: WorkingTreeModified, Untracked: 1}
	comparison := Compare(Expectation{}, fingerprint)
	tree := checkByID(t, comparison, "git.working_tree")
	if tree.Status != StatusUnknown || tree.Severity != SeverityWarning || comparison.Decision != DecisionConfirmationRequired {
		t.Fatalf("dirty tree check = %+v decision=%s", tree, comparison.Decision)
	}
}

func TestCompareHeadRelationMessages(t *testing.T) {
	t.Parallel()
	for _, relation := range []Relation{RelationAhead, RelationBehind, RelationDiverged, RelationUnknown} {
		relation := relation
		t.Run(string(relation), func(t *testing.T) {
			t.Parallel()
			fingerprint := baseFingerprint()
			fingerprint.Git.Head = strings.Repeat("b", 40)
			fingerprint.Git.ExpectedHeadRelation = CommitRelation{
				Relation: relation, Ahead: 1, Behind: 1,
				Knowable: relation != RelationUnknown, LocalOnly: true,
			}
			comparison := Compare(Expectation{Head: trustedString(strings.Repeat("a", 40))}, fingerprint)
			check := checkByID(t, comparison, "git.head")
			if check.Status != StatusChanged || check.Severity != SeverityWarning || comparison.Decision != DecisionConfirmationRequired {
				t.Fatalf("check = %+v decision=%s", check, comparison.Decision)
			}
			if relation != RelationUnknown && !strings.Contains(check.Message, string(relation)) {
				t.Fatalf("message %q lacks relation %q", check.Message, relation)
			}
		})
	}
}

func checkByID(t *testing.T, comparison Comparison, id string) Check {
	t.Helper()
	for _, check := range comparison.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("missing check %q", id)
	return Check{}
}
