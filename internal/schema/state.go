package schema

import (
	"fmt"
	"time"
)

// StateSchemaVersion is the current local state schema.
const StateSchemaVersion = 1

// State is local sync bookkeeping (JSON).
type State struct {
	SchemaVersion   int                     `json:"schema_version"`
	LastRemoteETag  string                  `json:"last_remote_etag"`
	LastManifestRev string                  `json:"last_manifest_revision"`
	Sessions        map[string]SessionState `json:"sessions"`
	UpdatedAt       string                  `json:"updated_at"`
	// VerifyReportedAt is when this device ran the post-first-push
	// verification and posted its report; empty until then.
	VerifyReportedAt string `json:"verify_reported_at,omitempty"`
}

// SessionState tracks one session's last known revision.
type SessionState struct {
	Agent          string `json:"agent"`
	SessionID      string `json:"session_id"`
	LocalRevision  string `json:"local_revision"`
	RemoteRevision string `json:"remote_revision"`
	UpdatedAt      string `json:"updated_at"`
}

// ValidateState checks schema version.
func ValidateState(s *State) error {
	if s == nil {
		return fmt.Errorf("nil state")
	}
	if s.SchemaVersion != StateSchemaVersion {
		return fmt.Errorf("unsupported state schema_version %d (want %d)", s.SchemaVersion, StateSchemaVersion)
	}
	if s.Sessions == nil {
		s.Sessions = map[string]SessionState{}
	}
	return nil
}

// NewState returns empty v1 state.
func NewState() *State {
	return &State{
		SchemaVersion: StateSchemaVersion,
		Sessions:      map[string]SessionState{},
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
}

// MigrateState upgrades older state documents when possible.
func MigrateState(schemaVersion int, raw map[string]any) (*State, error) {
	if schemaVersion == StateSchemaVersion {
		return nil, fmt.Errorf("use normal decode for current version")
	}
	if schemaVersion == 0 {
		return nil, fmt.Errorf("missing schema_version")
	}
	return nil, fmt.Errorf("no migration path from state schema_version %d", schemaVersion)
}
