//go:build !windows

package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtectOwnerOnlyTightensExistingUnixPermissions(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "state")
	if err := os.WriteFile(file, []byte("controlled"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o666); err != nil {
		t.Fatal(err)
	}

	if err := ProtectOwnerOnly(dir, true); err != nil {
		t.Fatal(err)
	}
	if err := ProtectOwnerOnly(file, false); err != nil {
		t.Fatal(err)
	}

	assertUnixPermissions(t, dir, 0o700)
	assertUnixPermissions(t, file, 0o600)
}

func TestEnsureOwnerOnlyDirTightensExistingUnixPermissions(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "existing")
	if err := os.Mkdir(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}

	if err := EnsureOwnerOnlyDir(dir); err != nil {
		t.Fatal(err)
	}
	assertUnixPermissions(t, dir, 0o700)
}

func assertUnixPermissions(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("permissions for %s = %04o, want %04o", filepath.Base(path), got, want)
	}
}
