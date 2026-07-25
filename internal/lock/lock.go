// Package lock provides local mutation locking for push/pull.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const staleAfter = 12 * time.Hour

// Mutex is a simple exclusive file lock for Reinstate home.
type Mutex struct {
	path string
	file *os.File
}

// Acquire creates an exclusive lock file under home/locks.
func Acquire(home, name string) (*Mutex, error) {
	dir := filepath.Join(home, "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+".lock")
	f, err := createLock(path)
	if os.IsExist(err) {
		info, statErr := os.Stat(path)
		if statErr == nil && time.Since(info.ModTime()) > staleAfter {
			if removeErr := os.Remove(path); removeErr == nil {
				f, err = createLock(path)
			}
		}
	}
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("lock held: %s", name)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(f, "%d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	return &Mutex{path: path, file: f}, nil
}

func createLock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

// Release removes the lock.
func (m *Mutex) Release() error {
	if m == nil {
		return nil
	}
	if m.file != nil {
		_ = m.file.Close()
	}
	return os.Remove(m.path)
}
