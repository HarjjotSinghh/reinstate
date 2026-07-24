package schema

// ManifestSchemaVersion is the remote manifest schema.
const ManifestSchemaVersion = 1

// Manifest is the decrypted remote session index.
type Manifest struct {
	SchemaVersion int                        `json:"schema_version"`
	Revision      string                     `json:"revision"`
	Sessions      map[string]ManifestSession `json:"sessions"`
	UpdatedAt     string                     `json:"updated_at"`
}

// ManifestSession points at an immutable encrypted snapshot.
type ManifestSession struct {
	Agent      string `json:"agent"`
	SessionID  string `json:"session_id"`
	SnapshotID string `json:"snapshot_id"`
	ProjectID  string `json:"project_id"`
	UpdatedAt  string `json:"updated_at"`
}

// NewManifest returns an empty v1 manifest.
func NewManifest(revision string) *Manifest {
	return &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		Revision:      revision,
		Sessions:      map[string]ManifestSession{},
	}
}
