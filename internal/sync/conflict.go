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

// SaveConflict writes a conflict record under home/conflicts.
func SaveConflict(home string, c Conflict) error {
	if c.ID == "" {
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

// Resolve removes the conflict record after applying strategy bookkeeping.
// Actual file restore is performed by the caller using pull/backup helpers.
func Resolve(home string, id string, how Resolution) error {
	if how != KeepLocal && how != KeepRemote && how != KeepBoth {
		return fmt.Errorf("invalid resolution")
	}
	path := filepath.Join(home, "conflicts", id+".json")
	if _, err := os.Stat(path); err != nil {
		return err
	}
	// keep an audit trail
	audit := filepath.Join(home, "conflicts", id+"."+string(how)+".resolved")
	_ = fsx.WriteFileAtomic(audit, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
	return os.Remove(path)
}
