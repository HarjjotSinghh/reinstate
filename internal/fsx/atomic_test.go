package fsx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := WriteFileAtomic(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"ok":true}` {
		t.Fatalf("got %q", b)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm()&0o077 != 0 {
			t.Fatalf("permissions too open: %v", fi.Mode())
		}
	}
}

// TestWritePrivateFileIsOwnerOnlyOnEveryPlatform runs unskipped on Windows,
// where the protection is the DACL rather than 0600.
func TestWritePrivateFileIsOwnerOnlyOnEveryPlatform(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capsule.json")
	if err := WritePrivateFile(path, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("content = %q", body)
	}
	private, detail, err := OwnerOnly(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if !private {
		t.Fatalf("WritePrivateFile left %s readable by others: %s", path, detail)
	}
}

func TestOwnerOnlyRejectsSharedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.json")
	if err := os.WriteFile(path, []byte("body"), 0o666); err != nil {
		t.Fatal(err)
	}
	// Defeat any inherited umask so the Unix case is deterministic. On Windows
	// this only clears the read-only attribute and leaves the inherited DACL.
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	private, detail, err := OwnerOnly(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if private {
		t.Fatalf("unprotected file reported as owner-only: %s", detail)
	}
}

func TestWriteFileAtomicPreservesOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := WriteFileAtomic(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicFail(path, []byte("new"), 0o600); err == nil {
		t.Fatal("expected failure")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "old" {
		t.Fatalf("previous content lost: %q", b)
	}
}
