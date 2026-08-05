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

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

const retryInterval = 20 * time.Millisecond

// Lock is one held OS advisory lock.
type Lock struct {
	file *os.File
}

// Acquire waits until path can be exclusively locked or ctx ends.
func Acquire(ctx context.Context, path string) (*Lock, error) {
	return acquire(ctx, path, tryExclusive)
}

// AcquireShared holds a shared lock until Close. It is used to prevent a
// derived database from being replaced while any process has it open.
func AcquireShared(ctx context.Context, path string) (*Lock, error) {
	return acquire(ctx, path, tryShared)
}

func acquire(ctx context.Context, path string, attempt func(*os.File) (bool, error)) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := fsx.ProtectOwnerOnly(path, false); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("protect lock file: %w", err)
	}
	for {
		acquired, lockErr := attempt(file)
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
