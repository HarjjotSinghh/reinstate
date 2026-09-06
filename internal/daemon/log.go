package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// Log rotation: the current file is renamed to .1 (and .1 to .2, ...) once
// it passes MaxLogBytes; LogKeep rotated copies are kept.
const (
	MaxLogBytes = 1 << 20
	LogKeep     = 3
)

// RotatingLog is a size-bounded append-only log file. Every line is one
// write; rotation happens before the write that would cross the bound, so
// the current file never exceeds MaxLogBytes by more than one line.
type RotatingLog struct {
	mu   sync.Mutex
	path string
	max  int64
	keep int
	file *os.File
	size int64
}

// OpenLog opens (or creates) the log at path.
func OpenLog(path string) (*RotatingLog, error) {
	return openLog(path, MaxLogBytes, LogKeep)
}

func openLog(path string, max int64, keep int) (*RotatingLog, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	l := &RotatingLog{path: path, max: max, keep: keep}
	if err := l.open(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *RotatingLog) open() error {
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_ = fsx.ProtectOwnerOnly(l.path, false)
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	l.file, l.size = file, info.Size()
	return nil
}

// Write implements io.Writer.
func (l *RotatingLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return 0, os.ErrClosed
	}
	if l.size > 0 && l.size+int64(len(p)) > l.max {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := l.file.Write(p)
	l.size += int64(n)
	return n, err
}

func (l *RotatingLog) rotate() error {
	if err := l.file.Close(); err != nil {
		return err
	}
	l.file = nil
	for i := l.keep - 1; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", l.path, i)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		if err := os.Rename(from, fmt.Sprintf("%s.%d", l.path, i+1)); err != nil {
			return err
		}
	}
	if l.keep >= 1 {
		if err := os.Rename(l.path, l.path+".1"); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return l.open()
}

// Close closes the current file.
func (l *RotatingLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}
