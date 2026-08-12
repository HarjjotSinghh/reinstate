// Package fsx provides filesystem helpers (atomic write, permissions, backup).
package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic writes data to path using a temp sibling + rename.
// On failure the previous file (if any) remains intact.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".reinstate-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil && runtime.GOOS != "windows" {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	// Best-effort directory sync where supported.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// WriteFileAtomicFail injects a failure after writing temp (for tests).
func WriteFileAtomicFail(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".reinstate-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	_ = tmp.Close()
	return fmt.Errorf("injected atomic write failure")
}

// WritePrivateFile atomically writes one owner-only file. It is the single
// write path for private artifacts: 0600 where the filesystem honours Unix
// permission bits, and the protected DACL on Windows, where os.Chmod only
// toggles the read-only attribute.
func WritePrivateFile(path string, data []byte) error {
	if err := WriteFileAtomic(path, data, OwnerOnlyFilePerm); err != nil {
		return err
	}
	return ProtectOwnerOnly(path, false)
}

// EnsureOwnerOnlyDir creates dir with 0700 where supported.
func EnsureOwnerOnlyDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return ProtectOwnerOnly(path, true)
}

// OwnerOnlyFilePerm is 0600.
const OwnerOnlyFilePerm os.FileMode = 0o600
