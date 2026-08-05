package filelock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireSerializesIndependentHandles(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "controlled.lock")
	first, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := Acquire(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("contended acquire error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSharedLocksCoexistAndBlockExclusive(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "shared.lock")
	first, err := AcquireShared(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := AcquireShared(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := Acquire(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("exclusive acquire error = %v", err)
	}
}

func TestAcquireChecksContextAfterRetryableAttempt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cancelled-retry.lock")
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	_, err := acquire(ctx, path, func(*os.File) (bool, error) {
		attempts++
		cancel()
		return false, nil
	})
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("acquire error/attempts = %v / %d", err, attempts)
	}
}
