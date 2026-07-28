package fsx

import (
	"fmt"
	"os"
	"time"
)

// Fingerprint captures the observable state of a restore target so a concurrent
// writer can be detected before the target is replaced.
//
// Restores are atomic (temp file plus rename), so a live agent can never read a
// torn file. The remaining risk is narrower: an agent appends to the target
// while the replacement is being prepared, and the rename would then discard
// those writes. Comparing a fingerprint taken before the work against one taken
// immediately before the rename closes that window.
type Fingerprint struct {
	Exists  bool
	Size    int64
	ModTime time.Time
}

// FingerprintFile records the current state of path. A missing file is not an
// error; it yields a zero-value Fingerprint with Exists set to false.
func FingerprintFile(path string) (Fingerprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Fingerprint{}, nil
		}
		return Fingerprint{}, err
	}
	return Fingerprint{Exists: true, Size: info.Size(), ModTime: info.ModTime()}, nil
}

// Equal reports whether two fingerprints describe the same file state.
func (f Fingerprint) Equal(other Fingerprint) bool {
	if f.Exists != other.Exists {
		return false
	}
	if !f.Exists {
		return true
	}
	return f.Size == other.Size && f.ModTime.Equal(other.ModTime)
}

// VerifyUnchanged re-reads path and fails if it no longer matches before.
func VerifyUnchanged(path string, before Fingerprint) error {
	current, err := FingerprintFile(path)
	if err != nil {
		return err
	}
	if !current.Equal(before) {
		return fmt.Errorf(
			"session file changed on disk during restore; another agent is writing to it, so the restore was abandoned to avoid discarding those changes")
	}
	return nil
}
