// Package workspace observes and compares the local workspace used for a
// native coding-agent continuation.
package workspace

import (
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/fileidentity"
)

const SchemaVersion = 1

const DefaultProbeTimeout = 2 * time.Second

// MaxChangedPaths bounds WorkingTreeFingerprint.Changed.
//
// The list travels into a continuity capsule, where it is counted twice against
// capsule.MaxFileReferences (once as workspace truth, once as the derived task
// checkpoint) and rendered into an 8 KiB destination bootstrap. A cap this size
// leaves the transcript's own file references room inside those budgets while
// still naming far more files than a briefing can usefully show. Paths past the
// cap are counted in ChangedOmitted rather than dropped in silence.
const MaxChangedPaths = 64

type Provenance string

const (
	ProvenanceVendorRecorded             Provenance = "vendor_recorded"
	ProvenanceReinstateCheckpoint        Provenance = "reinstate_checkpoint"
	ProvenanceReinstatePrelaunchObserved Provenance = "reinstate_prelaunch_observed"
	ProvenanceCurrentObservation         Provenance = "current_observation"
	ProvenanceUnavailable                Provenance = "unavailable"
)

type Status string

const (
	StatusMatch   Status = "match"
	StatusPresent Status = "present"
	StatusChanged Status = "changed"
	StatusMissing Status = "missing"
	StatusUnknown Status = "unknown"
	StatusError   Status = "error"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityBlock   Severity = "block"
)

type Decision string

const (
	DecisionReady                Decision = "ready"
	DecisionConfirmationRequired Decision = "confirmation_required"
	DecisionBlocked              Decision = "blocked"
)

type ExpectedString struct {
	Value      string     `json:"value"`
	Provenance Provenance `json:"provenance"`
}

type ExpectedBool struct {
	Value      bool       `json:"value"`
	Provenance Provenance `json:"provenance"`
}

// Expectation contains only facts with explicit historical provenance. A nil
// field is intentionally unknown; callers must not populate it from a live
// observation made during indexing or inspection.
type Expectation struct {
	Workspace         *ExpectedString `json:"workspace,omitempty"`
	RepositoryID      *ExpectedString `json:"repository_id,omitempty"`
	Branch            *ExpectedString `json:"branch,omitempty"`
	Head              *ExpectedString `json:"head,omitempty"`
	Dirty             *ExpectedBool   `json:"dirty,omitempty"`
	WorkingTreeDigest *ExpectedString `json:"working_tree_digest,omitempty"`
}

type WorkspaceFingerprint struct {
	Path      string `json:"-"`
	Exists    bool   `json:"exists"`
	Directory bool   `json:"directory"`
	// Identity privately binds the observed directory object to the final
	// launch. It is never serialized in human or JSON reports.
	Identity fileidentity.Identity `json:"-"`
}

type WorkingTreeState string

const (
	WorkingTreeClean       WorkingTreeState = "clean"
	WorkingTreeModified    WorkingTreeState = "modified"
	WorkingTreeUnavailable WorkingTreeState = "unavailable"
)

type WorkingTreeFingerprint struct {
	State           WorkingTreeState `json:"state"`
	Uncertain       bool             `json:"uncertain,omitempty"`
	Digest          string           `json:"digest,omitempty"`
	Staged          int              `json:"staged"`
	Unstaged        int              `json:"unstaged"`
	Untracked       int              `json:"untracked"`
	Conflicted      int              `json:"conflicted"`
	Submodule       int              `json:"submodule"`
	CountsTruncated bool             `json:"counts_truncated,omitempty"`

	// Changed holds the repository-relative paths behind the counts above,
	// sorted, de-duplicated, and capped at MaxChangedPaths. It is never
	// serialized: a probe report is printed by verify and doctor, and a path
	// list would leak working-tree contents into those reports and into any
	// log that captures them. Only a handoff consumes it, and only after
	// internal/pathmap has rewritten every entry into a portable token.
	Changed []string `json:"-"`
	// ChangedOmitted counts distinct changed paths that exceeded the cap.
	// A consumer that renders Changed must surface this count; a silently
	// short list would tell the destination the tree is cleaner than it is.
	ChangedOmitted int `json:"-"`
}

type Relation string

const (
	RelationEqual    Relation = "equal"
	RelationAhead    Relation = "ahead"
	RelationBehind   Relation = "behind"
	RelationDiverged Relation = "diverged"
	RelationUnknown  Relation = "unknown"
)

type CommitRelation struct {
	Relation  Relation `json:"relation"`
	Ahead     int      `json:"ahead,omitempty"`
	Behind    int      `json:"behind,omitempty"`
	Knowable  bool     `json:"knowable"`
	LocalOnly bool     `json:"local_only"`
}

type GitFingerprint struct {
	Available            bool                   `json:"available"`
	Repository           bool                   `json:"repository"`
	Root                 string                 `json:"-"`
	RepositoryID         string                 `json:"repository_id,omitempty"`
	RepositoryIDSource   string                 `json:"repository_id_source,omitempty"`
	Branch               string                 `json:"branch,omitempty"`
	Detached             bool                   `json:"detached"`
	Unborn               bool                   `json:"unborn"`
	Head                 string                 `json:"head,omitempty"`
	Shallow              bool                   `json:"shallow"`
	WorkingTree          WorkingTreeFingerprint `json:"working_tree"`
	Upstream             string                 `json:"upstream,omitempty"`
	UpstreamRelation     CommitRelation         `json:"upstream_relation"`
	ExpectedHeadRelation CommitRelation         `json:"expected_head_relation"`

	// repositoryIDs supports matching any configured remote without exposing
	// extra repository metadata in JSON.
	repositoryIDs []string
	rootPath      string
}

type Fingerprint struct {
	SchemaVersion int                  `json:"schema_version"`
	Provenance    Provenance           `json:"provenance"`
	Workspace     WorkspaceFingerprint `json:"workspace"`
	Git           GitFingerprint       `json:"git"`
}

type Diagnostic struct {
	ID       string   `json:"id"`
	Status   Status   `json:"status"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

type ProbeResult struct {
	Fingerprint Fingerprint  `json:"fingerprint"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type Check struct {
	ID         string     `json:"id"`
	Status     Status     `json:"status"`
	Severity   Severity   `json:"severity"`
	Expected   any        `json:"expected,omitempty"`
	Actual     any        `json:"actual,omitempty"`
	Provenance Provenance `json:"provenance"`
	Message    string     `json:"message"`
	Repair     string     `json:"repair,omitempty"`
}

type Comparison struct {
	Decision Decision `json:"decision"`
	Checks   []Check  `json:"checks"`
}

type Verification struct {
	Fingerprint Fingerprint  `json:"fingerprint"`
	Comparison  Comparison   `json:"comparison"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type ProbeOptions struct {
	Runner  GitRunner
	Timeout time.Duration
}
