package lock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireIsExclusiveAndReleaseUnlocks(t *testing.T) {
	home := t.TempDir()
	first, err := Acquire(home, "mutation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(home, "mutation"); err == nil {
		t.Fatal("expected active lock refusal")
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire(home, "mutation")
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestAcquireReclaimsExpiredLock(t *testing.T) {
	home := t.TempDir()
	lockDir := filepath.Join(home, "locks")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(lockDir, "mutation.lock")
	if err := os.WriteFile(path, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-staleAfter - time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	mutex, err := Acquire(home, "mutation")
	if err != nil {
		t.Fatal(err)
	}
	if err := mutex.Release(); err != nil {
		t.Fatal(err)
	}
}
