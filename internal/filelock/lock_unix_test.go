//go:build !windows

package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestFlockRetriesInterruptedAcquireAndUnlock(t *testing.T) {
	file, err := os.OpenFile(filepath.Join(t.TempDir(), "interrupted.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	acquireCalls := 0
	flock := func(_, operation int) error {
		if operation != unix.LOCK_EX|unix.LOCK_NB {
			t.Fatalf("operation = %d", operation)
		}
		acquireCalls++
		if acquireCalls == 1 {
			return unix.EINTR
		}
		return nil
	}
	acquired, err := tryFlock(file, unix.LOCK_EX|unix.LOCK_NB, flock)
	if err != nil || acquired || acquireCalls != 1 {
		t.Fatalf("interrupted acquired/error/calls = %t / %v / %d", acquired, err, acquireCalls)
	}
	acquired, err = tryFlock(file, unix.LOCK_EX|unix.LOCK_NB, flock)
	if err != nil || !acquired || acquireCalls != 2 {
		t.Fatalf("acquired/error/calls = %t / %v / %d", acquired, err, acquireCalls)
	}

	unlockCalls := 0
	err = unlockFlock(file, func(_, operation int) error {
		if operation != unix.LOCK_UN {
			t.Fatalf("operation = %d", operation)
		}
		unlockCalls++
		if unlockCalls == 1 {
			return unix.EINTR
		}
		return nil
	})
	if err != nil || unlockCalls != 2 {
		t.Fatalf("unlock error/calls = %v / %d", err, unlockCalls)
	}
}

func TestFlockInterruptedRetryPreservesContentionAndErrors(t *testing.T) {
	file, err := os.OpenFile(filepath.Join(t.TempDir(), "contention.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	for _, test := range []struct {
		name     string
		finalErr error
		wantErr  bool
	}{
		{name: "contention", finalErr: unix.EWOULDBLOCK},
		{name: "fatal", finalErr: unix.EBADF, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			flock := func(int, int) error {
				calls++
				if calls == 1 {
					return unix.EINTR
				}
				return test.finalErr
			}
			acquired, gotErr := tryFlock(file, unix.LOCK_EX|unix.LOCK_NB, flock)
			if gotErr != nil || acquired || calls != 1 {
				t.Fatalf("interrupted acquired/error/calls = %t / %v / %d", acquired, gotErr, calls)
			}
			acquired, gotErr = tryFlock(file, unix.LOCK_EX|unix.LOCK_NB, flock)
			if acquired || calls != 2 || (gotErr != nil) != test.wantErr {
				t.Fatalf("acquired/error/calls = %t / %v / %d", acquired, gotErr, calls)
			}
			if test.wantErr && !errors.Is(gotErr, test.finalErr) {
				t.Fatalf("error = %v, want %v", gotErr, test.finalErr)
			}
		})
	}
}
