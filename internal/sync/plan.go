// Package sync implements push/pull orchestration and conflict handling.
package sync

import (
	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// PushItem is one session planned for upload.
type PushItem struct {
	Agent     string
	SessionID string
	ProjectID string
	LocalPath string
}

// PullItem is one session planned for download.
type PullItem struct {
	Agent      string
	SessionID  string
	SnapshotID string
	ProjectID  string
}

// Plan is a dry-run friendly summary.
type Plan struct {
	Push []PushItem `json:"push,omitempty"`
	Pull []PullItem `json:"pull,omitempty"`
}

// SessionKey builds a stable map key.
func SessionKey(agent, sessionID string) string {
	return agent + ":" + sessionID
}

// DetectConflict returns true when local and remote revisions diverge.
func DetectConflict(localRev, remoteRev, baseRev string) bool {
	if remoteRev == "" || localRev == "" {
		return false
	}
	if localRev == remoteRev {
		return false
	}
	// diverged if both moved past a shared base (or base unknown but both differ)
	if baseRev != "" && localRev != baseRev && remoteRev != baseRev {
		return true
	}
	return localRev != remoteRev
}

// ManifestSessionLookup finds a session in the remote manifest.
func ManifestSessionLookup(m *schema.Manifest, agent, sessionID string) (schema.ManifestSession, bool) {
	if m == nil {
		return schema.ManifestSession{}, false
	}
	s, ok := m.Sessions[SessionKey(agent, sessionID)]
	return s, ok
}

// FromAdapterSessions converts discovered sessions into push items.
func FromAdapterSessions(sessions []adapter.Session) []PushItem {
	out := make([]PushItem, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, PushItem{
			Agent:     s.Agent,
			SessionID: s.ID,
			ProjectID: s.ProjectID,
			LocalPath: s.Path,
		})
	}
	return out
}
