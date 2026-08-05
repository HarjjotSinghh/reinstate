//go:build windows

package filelock

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsLockContentionErrorsAreRetryable(t *testing.T) {
	for _, retryable := range []error{windows.ERROR_LOCK_VIOLATION, windows.ERROR_IO_PENDING} {
		if !retryableLockError(retryable) {
			t.Fatalf("error %v was not retryable", retryable)
		}
	}
	if retryableLockError(windows.ERROR_ACCESS_DENIED) {
		t.Fatal("access denied was treated as lock contention")
	}
	if retryableLockError(errors.New("unrelated")) {
		t.Fatal("unrelated error was treated as lock contention")
	}
}
