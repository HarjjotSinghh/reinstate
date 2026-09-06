package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// Conflict is metadata-only (no transcript plaintext).
type Conflict struct {
	ID             string `json:"id"`
	Agent          string `json:"agent"`
	SessionID      string `json:"session_id"`
	ProjectID      string `json:"project_id"`
	LocalRevision  string `json:"local_revision"`
	RemoteRevision string `json:"remote_revision"`
	RemoteSnapshot string `json:"remote_snapshot"`
	CreatedAt      string `json:"created_at"`
}

// Resolution strategy.
type Resolution string

const (
	KeepLocal  Resolution = "keep-local"
	KeepRemote Resolution = "keep-remote"
	KeepBoth   Resolution = "keep-both"
)

// SaveConflict writes a conflict record under home/conflicts. It is
// idempotent: when an unresolved record already describes the same
// divergence (agent, session, local revision, remote snapshot) nothing is
// written, so a daemon that pushes and pulls every few seconds against a
// conflict it cannot resolve does not grow the directory without bound.
func SaveConflict(home string, c Conflict) error {
	if c.ID == "" {
		existing, err := ListConflicts(home)
		if err != nil {
			return err
		}
		for _, e := range existing {
			if e.SameDivergence(c) {
				return nil
			}
		}
		c.ID = fmt.Sprintf("c-%d", time.Now().UnixNano())
	}
	if c.CreatedAt == "" {
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	dir := filepath.Join(home, "conflicts")
	if err := fsx.EnsureOwnerOnlyDir(dir); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return fsx.WriteFileAtomic(filepath.Join(dir, c.ID+".json"), append(b, '\n'), 0o600)
}

// SameDivergence reports whether two records describe the same local
// revision diverging from the same remote snapshot of the same session.
func (c Conflict) SameDivergence(o Conflict) bool {
	return c.Agent == o.Agent && c.SessionID == o.SessionID &&
		c.LocalRevision == o.LocalRevision && c.RemoteSnapshot == o.RemoteSnapshot
}

// ListConflicts returns conflict metadata records.
func ListConflicts(home string) ([]Conflict, error) {
	dir := filepath.Join(home, "conflicts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Conflict
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var c Conflict
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// GetConflict loads one conflict record by ID.
func GetConflict(home, id string) (Conflict, error) {
	path := filepath.Join(home, "conflicts", id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Conflict{}, err
	}
	var conflict Conflict
	if err := json.Unmarshal(raw, &conflict); err != nil {
		return Conflict{}, err
	}
	if conflict.ID != id {
		return Conflict{}, fmt.Errorf("conflict identity mismatch")
	}
	return conflict, nil
}

// Resolve runs the concrete resolution before recording the audit result and
// removing the conflict. Any failure preserves the original conflict record.
func Resolve(home string, id string, how Resolution, apply func(Conflict, Resolution) error) error {
	if how != KeepLocal && how != KeepRemote && how != KeepBoth {
		return fmt.Errorf("invalid resolution")
	}
	path := filepath.Join(home, "conflicts", id+".json")
	conflict, err := GetConflict(home, id)
	if err != nil {
		return err
	}
	if apply == nil {
		return fmt.Errorf("resolution executor required")
	}
	if err := apply(conflict, how); err != nil {
		return err
	}
	audit := filepath.Join(home, "conflicts", id+"."+string(how)+".resolved")
	if err := fsx.WriteFileAtomic(audit, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600); err != nil {
		return err
	}
	return os.Remove(path)
}
