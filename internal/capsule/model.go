package capsule

// Schema is the continuity-capsule schema identifier.
const Schema = "reinstate.continuity-capsule/v1"

// SchemaVersion is the integer schema version embedded in Identity.
const SchemaVersion = 1

// FidelityModeStructuredHandoff is the only Mode value used in v0.4.0.
const FidelityModeStructuredHandoff = "structured_handoff"

// FidelityModeReconstructedConversation is reserved and unused in v0.4.0.
const FidelityModeReconstructedConversation = "reconstructed_conversation"

// Capsule is the top-level continuity-capsule v1 document.
//
// The hashed body has no wall-clock created_at; lineage timestamps live outside
// this structure. Absolute source paths are never serialized.
type Capsule struct {
	Schema       string         `json:"schema"`
	Identity     Identity       `json:"identity"`
	RawSource    RawSource      `json:"raw_source"`
	Task         Task           `json:"task"`
	Workspace    Workspace      `json:"workspace"`
	Conversation Conversation   `json:"conversation"`
	Capabilities CapabilityDiff `json:"capabilities"`
	Security     Security       `json:"security"`
	Fidelity     Fidelity       `json:"fidelity"`
	Projection   Projection     `json:"projection"`
}

// Identity is the content-derived capsule identity and lineage link.
type Identity struct {
	ID          string `json:"id"`
	LineageRoot string `json:"lineage_root"`
	Parent      Parent `json:"parent_session"`
	SchemaVer   int    `json:"schema_version"`
}

// Parent identifies the source session that produced this capsule.
type Parent struct {
	Agent          string `json:"agent"`
	SessionID      string `json:"id"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	AdapterVersion string `json:"adapter_version"`
}

// RawSource freezes the immutable vendor artifact boundary used to build the
// capsule. Path is private and never serialized.
type RawSource struct {
	Agent          string `json:"agent"`
	SessionID      string `json:"session_id"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	AdapterVersion string `json:"adapter_version"`
	ByteOffset     int64  `json:"byte_offset"`
	SizeBytes      int64  `json:"size_bytes"`
	Partial        bool   `json:"partial,omitempty"`
	Path           string `json:"-"`
}

// TextField is a single-valued task field with portability metadata.
type TextField struct {
	Text        string      `json:"text,omitempty"`
	Portability Portability `json:"portability"`
	Reason      string      `json:"reason,omitempty"`
	Label       string      `json:"label,omitempty"`
}

// ListField is a multi-valued task field with portability metadata.
type ListField struct {
	Items       []string    `json:"items,omitempty"`
	Portability Portability `json:"portability"`
	Reason      string      `json:"reason,omitempty"`
	Label       string      `json:"label,omitempty"`
}

// Task is the deterministic checkpoint derived for a structured handoff.
//
// Fields that cannot be derived without a model (constraints, decisions,
// rejected_approaches) are recorded as omitted with an explicit reason.
type Task struct {
	Goal                      TextField `json:"goal"`
	LatestUserIntent          TextField `json:"latest_user_intent"`
	RecentUserMessages        ListField `json:"recent_user_messages"`
	Constraints               ListField `json:"constraints"`
	Decisions                 ListField `json:"decisions"`
	RejectedApproaches        ListField `json:"rejected_approaches"`
	Completed                 ListField `json:"completed"`
	Pending                   ListField `json:"pending"`
	ChangedFiles              ListField `json:"changed_files"`
	FilesTouchedPerTranscript ListField `json:"files_touched_per_transcript"`
	Tests                     ListField `json:"tests"`
	NextAction                TextField `json:"next_action"`
	OpenQuestions             ListField `json:"open_questions,omitempty"`
}

// Workspace is live workspace truth with portable path tokens only.
// Path holds the private absolute root and is never serialized.
type Workspace struct {
	ProjectID         string   `json:"project_id"`
	Root              string   `json:"root"`
	Branch            string   `json:"branch,omitempty"`
	Head              string   `json:"head,omitempty"`
	Dirty             bool     `json:"dirty"`
	WorkingTreeDigest string   `json:"working_tree_digest,omitempty"`
	ChangedFiles      []string `json:"changed_files,omitempty"`
	// ChangedFilesOmitted counts changed paths the bound list could not carry.
	// Renderers must show it: a destination shown a short list without a count
	// is being told the working tree is cleaner than it is.
	ChangedFilesOmitted int      `json:"changed_files_omitted,omitempty"`
	Tests               []string `json:"tests,omitempty"`
	Path                string   `json:"-"`
}

// Conversation holds canonical events and an optional sidecar history ref.
type Conversation struct {
	Events         []Event `json:"events"`
	FullHistoryRef string  `json:"full_history_ref,omitempty"`
}

// MissingCapability is one destination gap reported in a capability diff.
type MissingCapability struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Impact string `json:"impact"`
}

// CapabilityDiff compares source and destination agent capabilities.
//
// Source and Destination are privacy-safe summary maps (no paths, secrets, or
// command lines). Missing lists destination gaps.
type CapabilityDiff struct {
	Source      map[string]any      `json:"source"`
	Destination map[string]any      `json:"destination"`
	Missing     []MissingCapability `json:"missing,omitempty"`
}

// Security records redaction and trust metadata for the capsule.
type Security struct {
	Redactions                            []Redaction `json:"redactions,omitempty"`
	SourceInstructionsAreUntrustedHistory bool        `json:"source_instructions_are_untrusted_history"`
	DestinationWarning                    string      `json:"destination_warning,omitempty"`
	RedactionForced                       bool        `json:"redaction_forced,omitempty"`
}

// Component is one fidelity classification entry.
type Component struct {
	Name        string      `json:"name"`
	Portability Portability `json:"portability"`
	Count       int         `json:"count"`
	Bytes       int64       `json:"bytes"`
	Reason      string      `json:"reason,omitempty"`
}

// Fidelity is the component-level portability report.
//
// Overall is the worst portability among included components. Mode is
// structured_handoff in v0.4.0; reconstructed_conversation is unused.
type Fidelity struct {
	Overall     Portability `json:"overall"`
	Mode        string      `json:"mode"`
	Components  []Component `json:"components"`
	Unsupported []string    `json:"unsupported,omitempty"`
}

// Projection summarizes the destination-facing briefing budget and hashes.
type Projection struct {
	Policy           string   `json:"policy"`
	EstimatedBytes   int64    `json:"estimated_bytes"`
	EstimatedTokens  int64    `json:"estimated_tokens"`
	IncludedEventIDs []string `json:"included_event_ids,omitempty"`
	SidecarRef       string   `json:"sidecar_ref,omitempty"`
	BootstrapSHA256  string   `json:"bootstrap_sha256,omitempty"`
	MarkdownSHA256   string   `json:"markdown_sha256,omitempty"`
}
