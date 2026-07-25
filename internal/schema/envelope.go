package schema

// EnvelopeSchemaVersion is the snapshot envelope schema.
const EnvelopeSchemaVersion = 1

// EnvelopeKind identifies Reinstate session snapshots.
const EnvelopeKind = "reinstate-session-snapshot"

// Envelope is plaintext metadata before age encryption.
type Envelope struct {
	SchemaVersion  int            `json:"schema_version"`
	Kind           string         `json:"kind"`
	SnapshotID     string         `json:"snapshot_id"`
	ParentRevision string         `json:"parent_revision"`
	Agent          string         `json:"agent"`
	AdapterSchema  int            `json:"adapter_schema"`
	SourceAgentVer string         `json:"source_agent_version"`
	SourcePlatform string         `json:"source_platform"`
	ProjectID      string         `json:"project_id"`
	SessionID      string         `json:"session_id"`
	CreatedAt      string         `json:"created_at"`
	Files          []EnvelopeFile `json:"files"`
}

// EnvelopeFile describes one payload file inside the snapshot.
type EnvelopeFile struct {
	Path   string `json:"path"`
	Mode   int    `json:"mode"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
