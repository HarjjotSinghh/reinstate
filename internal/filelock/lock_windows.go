//go:build windows

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func tryExclusive(file *os.File) (bool, error) {
	return try(file, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY)
}

func tryShared(file *os.File) (bool, error) {
	return try(file, windows.LOCKFILE_FAIL_IMMEDIATELY)
}

func try(file *os.File, flags uint32) (bool, error) {
	overlapped := new(windows.Overlapped)
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		flags,
		0,
		1,
		0,
		overlapped,
	)
	if err == nil {
		return true, nil
	}
	if retryableLockError(err) {
		return false, nil
	}
	return false, err
}

func retryableLockError(err error) bool {
	// FAIL_IMMEDIATELY normally reports LOCK_VIOLATION. Treat a platform or
	// filesystem surfacing IO_PENDING as the same transient contention signal
	// so the context-bounded outer acquisition loop retries it.
	return errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING)
}

func unlock(file *os.File) error {
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, new(windows.Overlapped))
}
