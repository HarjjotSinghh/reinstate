// Package filelock provides a small cross-process advisory lock used for
// private derived state. Lock files contain no data and are never removed, so
// a process crash cannot strand a false ownership marker.
package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

const retryInterval = 20 * time.Millisecond

// Lock is one held OS advisory lock.
type Lock struct {
	file *os.File
}

// Acquire waits until path can be exclusively locked or ctx ends.
func Acquire(ctx context.Context, path string) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("protect lock file: %w", err)
	}
	for {
		acquired, lockErr := tryExclusive(file)
		if lockErr != nil {
			_ = file.Close()
			return nil, fmt.Errorf("acquire file lock: %w", lockErr)
		}
		if acquired {
			return &Lock{file: file}, nil
		}
		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			_ = file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// Close releases the advisory lock and file descriptor.
func (lock *Lock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	err := unlock(lock.file)
	closeErr := lock.file.Close()
	lock.file = nil
	return errors.Join(err, closeErr)
}
