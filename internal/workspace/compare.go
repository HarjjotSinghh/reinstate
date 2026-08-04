package workspace

import "slices"

// Compare evaluates current observations only against trustworthy expected
// values. Missing historical baselines remain unknown and never become match.
func Compare(expected Expectation, actual Fingerprint) Comparison {
	checks := []Check{
		compareWorkspace(expected.Workspace, actual.Workspace),
		compareRepository(expected.RepositoryID, actual.Git),
		compareBranch(expected.Branch, actual.Git),
		compareHead(expected.Head, actual.Git),
		compareWorkingTree(expected.Dirty, expected.WorkingTreeDigest, actual.Git),
	}
	return Comparison{Decision: aggregateDecision(checks, nil), Checks: checks}
}

func compareWorkspace(expected *ExpectedString, actual WorkspaceFingerprint) Check {
	check := Check{
		ID: "workspace.available", Provenance: expectationProvenance(expected),
		Actual: actual.Path,
	}
	if expected != nil {
		check.Expected = expected.Value
	}
	switch {
	case !actual.Exists:
		check.Status, check.Severity = StatusMissing, SeverityBlock
		check.Message = "the recorded workspace is missing"
		check.Repair = "restore or remap the recorded workspace before continuing"
	case !actual.Directory:
		check.Status, check.Severity = StatusMissing, SeverityBlock
		check.Message = "the recorded workspace is not a directory"
		check.Repair = "restore or remap the recorded workspace before continuing"
	default:
		check.Status, check.Severity = StatusPresent, SeverityInfo
		check.Message = "the recorded workspace is available"
	}
	return check
}

func compareRepository(expected *ExpectedString, actual GitFingerprint) Check {
	check := Check{
		ID: "git.repository", Provenance: expectationProvenance(expected),
		Actual: actual.RepositoryID,
	}
	if expected == nil || !trustedProvenance(expected.Provenance) {
		check.Status, check.Severity = StatusUnknown, SeverityInfo
		check.Message = "the session has no trustworthy repository identity baseline"
		return check
	}
	check.Expected = expected.Value
	if !actual.Repository {
		check.Status, check.Severity = StatusMissing, SeverityBlock
		check.Message = "the recorded workspace is not a Git repository"
		check.Repair = "open the session in the recorded repository"
		return check
	}
	if actual.RepositoryID == "" {
		check.Status, check.Severity = StatusUnknown, SeverityWarning
		check.Message = "the current repository identity cannot be compared safely"
		check.Repair = "verify the repository manually or configure its canonical remote"
		return check
	}
	if expected.Value == actual.RepositoryID || slices.Contains(actual.repositoryIDs, expected.Value) {
		check.Status, check.Severity = StatusMatch, SeverityInfo
		check.Message = "the current repository matches the recorded repository"
		return check
	}
	check.Status, check.Severity = StatusChanged, SeverityBlock
	check.Message = "the current repository differs from the recorded repository"
	check.Repair = "switch to the recorded repository before continuing"
	return check
}

func compareBranch(expected *ExpectedString, actual GitFingerprint) Check {
	check := Check{ID: "git.branch", Provenance: expectationProvenance(expected), Actual: actual.Branch}
	if expected == nil || !trustedProvenance(expected.Provenance) {
		check.Status, check.Severity = StatusUnknown, SeverityInfo
		check.Message = "the session has no trustworthy branch baseline"
		return check
	}
	check.Expected = expected.Value
	if !actual.Repository {
		check.Status, check.Severity = StatusMissing, SeverityBlock
		check.Message = "the recorded Git branch cannot be verified outside a Git repository"
		check.Repair = "restore the recorded Git repository before continuing"
		return check
	}
	if actual.Unborn {
		check.Status, check.Severity = StatusUnknown, SeverityWarning
		check.Message = "the current branch cannot be compared safely"
		check.Repair = "restore the recorded branch or explicitly continue without it"
		return check
	}
	if actual.Detached {
		check.Status, check.Severity = StatusChanged, SeverityWarning
		check.Actual = "detached"
		check.Message = "the workspace has a detached HEAD instead of the recorded branch"
		check.Repair = "switch to the recorded branch or acknowledge warning git.branch"
		return check
	}
	if expected.Value == actual.Branch {
		check.Status, check.Severity = StatusMatch, SeverityInfo
		check.Message = "the current branch matches the branch recorded by the session"
		return check
	}
	check.Status, check.Severity = StatusChanged, SeverityWarning
	check.Message = "the current branch differs from the branch recorded by the session"
	check.Repair = "switch to the recorded branch or acknowledge warning git.branch"
	return check
}

func compareHead(expected *ExpectedString, actual GitFingerprint) Check {
	check := Check{ID: "git.head", Provenance: expectationProvenance(expected), Actual: actual.Head}
	if expected == nil || !trustedProvenance(expected.Provenance) {
		check.Status, check.Severity = StatusUnknown, SeverityInfo
		check.Message = "the session has no trustworthy Git HEAD baseline"
		return check
	}
	check.Expected = expected.Value
	if !actual.Repository {
		check.Status, check.Severity = StatusMissing, SeverityBlock
		check.Message = "the recorded Git HEAD cannot be verified outside a Git repository"
		check.Repair = "restore the recorded Git repository before continuing"
		return check
	}
	if actual.Head == "" {
		check.Status, check.Severity = StatusUnknown, SeverityWarning
		check.Message = "the current Git HEAD cannot be compared safely"
		check.Repair = "restore the recorded commit or acknowledge warning git.head"
		return check
	}
	if expected.Value == actual.Head {
		check.Status, check.Severity = StatusMatch, SeverityInfo
		check.Message = "the current Git HEAD matches the recorded commit"
		return check
	}
	check.Status, check.Severity = StatusChanged, SeverityWarning
	check.Message = headChangeMessage(actual.ExpectedHeadRelation)
	check.Repair = "review the local commit relation or acknowledge warning git.head"
	return check
}

func headChangeMessage(relation CommitRelation) string {
	if !relation.Knowable {
		return "the current Git HEAD differs and its local relation to the recorded commit is unknown"
	}
	switch relation.Relation {
	case RelationAhead:
		return "the current Git HEAD is ahead of the commit recorded by the session"
	case RelationBehind:
		return "the current Git HEAD is behind the commit recorded by the session"
	case RelationDiverged:
		return "the current Git HEAD has diverged from the commit recorded by the session"
	default:
		return "the current Git HEAD differs from the commit recorded by the session"
	}
}

func compareWorkingTree(expected *ExpectedBool, expectedDigest *ExpectedString, git GitFingerprint) Check {
	actual := git.WorkingTree
	check := Check{
		ID: "git.working_tree", Provenance: workingTreeProvenance(expected, expectedDigest),
		Actual: actual.State,
	}
	if actual.State == WorkingTreeUnavailable {
		if expected != nil && trustedProvenance(expected.Provenance) ||
			expectedDigest != nil && trustedProvenance(expectedDigest.Provenance) {
			check.Status, check.Severity = StatusMissing, SeverityBlock
			check.Message = "the recorded working-tree baseline cannot be verified outside a Git repository"
			check.Repair = "restore the recorded Git repository before continuing"
		} else {
			check.Status, check.Severity = StatusUnknown, SeverityWarning
			check.Message = "the current working tree is unavailable and no historical tree baseline exists"
			check.Repair = "review warning git.working_tree before continuing"
		}
		return check
	}
	if expectedDigest != nil && trustedProvenance(expectedDigest.Provenance) {
		check.Expected = expectedDigest.Value
		check.Actual = actual.Digest
		if expectedDigest.Value == actual.Digest {
			check.Status, check.Severity = StatusMatch, SeverityInfo
			check.Message = "the current working tree matches the recorded digest"
		} else {
			check.Status, check.Severity = StatusChanged, SeverityWarning
			check.Message = "the current working tree differs from the recorded digest"
			check.Repair = "review local modifications or acknowledge warning git.working_tree"
		}
		return check
	}
	dirty := actual.State == WorkingTreeModified
	if expected == nil || !trustedProvenance(expected.Provenance) {
		check.Status = StatusUnknown
		if dirty {
			check.Severity = SeverityWarning
			check.Message = "the current working tree is modified; no historical tree baseline exists"
			check.Repair = "review local modifications or acknowledge warning git.working_tree"
		} else {
			check.Severity = SeverityInfo
			check.Message = "the current working tree is clean; no historical tree baseline exists"
		}
		return check
	}
	check.Expected = expected.Value
	if expected.Value == dirty {
		check.Status, check.Severity = StatusMatch, SeverityInfo
		check.Message = "the current working-tree state matches the recorded baseline"
		return check
	}
	check.Status, check.Severity = StatusChanged, SeverityWarning
	check.Message = "the current working-tree state differs from the recorded baseline"
	check.Repair = "review local modifications or acknowledge warning git.working_tree"
	return check
}

func trustedProvenance(provenance Provenance) bool {
	return provenance == ProvenanceVendorRecorded ||
		provenance == ProvenanceReinstateCheckpoint ||
		provenance == ProvenanceReinstatePrelaunchObserved
}

func expectationProvenance(expected *ExpectedString) Provenance {
	if expected == nil {
		return ProvenanceUnavailable
	}
	return expected.Provenance
}

func expectationBoolProvenance(expected *ExpectedBool) Provenance {
	if expected == nil {
		return ProvenanceUnavailable
	}
	return expected.Provenance
}

func workingTreeProvenance(expected *ExpectedBool, digest *ExpectedString) Provenance {
	if digest != nil {
		return digest.Provenance
	}
	return expectationBoolProvenance(expected)
}

func aggregateDecision(checks []Check, diagnostics []Diagnostic) Decision {
	decision := DecisionReady
	for _, check := range checks {
		switch check.Severity {
		case SeverityBlock:
			return DecisionBlocked
		case SeverityWarning:
			decision = DecisionConfirmationRequired
		}
	}
	for _, diagnostic := range diagnostics {
		switch diagnostic.Severity {
		case SeverityBlock:
			return DecisionBlocked
		case SeverityWarning:
			decision = DecisionConfirmationRequired
		}
	}
	return decision
}
