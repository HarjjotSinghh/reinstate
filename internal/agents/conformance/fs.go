package conformance

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	errWriteAttempt = errors.New("conformance: write, rename, truncate, or lock under a vendor tree")
	errOutsideRoot  = errors.New("conformance: open outside fixture root")
)

// isolationFS is a read-only view of one fixture root. Every open is recorded.
// Write, rename, truncate, lock, and any path outside root fail closed.
type isolationFS struct {
	root  string
	mu    sync.Mutex
	opens []string
}

func newIsolationFS(root string) (*isolationFS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("conformance: fixture root is not a directory: %s", abs)
	}
	return &isolationFS{root: abs}, nil
}

// Open implements fs.FS.
func (f *isolationFS) Open(name string) (fs.File, error) {
	return f.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile records a read-only open. Any write flag is rejected.
func (f *isolationFS) OpenFile(name string, flag int, _ os.FileMode) (*os.File, error) {
	if writeFlag(flag) {
		return nil, fmt.Errorf("%w: open %s", errWriteAttempt, name)
	}
	path, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	f.record(path)
	return os.OpenFile(path, os.O_RDONLY, 0)
}

// Stat is read-only metadata.
func (f *isolationFS) Stat(name string) (os.FileInfo, error) {
	path, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	f.record(path)
	return os.Stat(path)
}

// Rename is always a closed failure.
func (f *isolationFS) Rename(oldpath, newpath string) error {
	return fmt.Errorf("%w: rename %s -> %s", errWriteAttempt, oldpath, newpath)
}

// Truncate is always a closed failure.
func (f *isolationFS) Truncate(name string, _ int64) error {
	return fmt.Errorf("%w: truncate %s", errWriteAttempt, name)
}

// Lock is always a closed failure.
func (f *isolationFS) Lock(name string) error {
	return fmt.Errorf("%w: lock %s", errWriteAttempt, name)
}

// Remove is always a closed failure.
func (f *isolationFS) Remove(name string) error {
	return fmt.Errorf("%w: remove %s", errWriteAttempt, name)
}

// Opens returns a copy of recorded open paths.
func (f *isolationFS) Opens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.opens))
	copy(out, f.opens)
	return out
}

func (f *isolationFS) resolve(name string) (string, error) {
	if name == "" {
		return "", errOutsideRoot
	}
	var path string
	if filepath.IsAbs(name) {
		path = filepath.Clean(name)
	} else {
		path = filepath.Join(f.root, filepath.Clean(name))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !withinRoot(f.root, abs) {
		return "", fmt.Errorf("%w: %s", errOutsideRoot, abs)
	}
	return abs, nil
}

func (f *isolationFS) record(path string) {
	f.mu.Lock()
	f.opens = append(f.opens, path)
	f.mu.Unlock()
}

func writeFlag(flag int) bool {
	return flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0
}

func withinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if path == root {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(path, root+sep)
}

var _ fs.FS = (*isolationFS)(nil)
