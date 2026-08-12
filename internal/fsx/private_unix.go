//go:build !windows

package fsx

import (
	"fmt"
	"os"
)

// ProtectOwnerOnly removes group/other access from one private path.
func ProtectOwnerOnly(path string, directory bool) error {
	return os.Chmod(path, ownerOnlyMode(directory))
}

// OwnerOnly reports whether path currently carries the owner-only protection
// ProtectOwnerOnly installs, plus a human-readable description of what was
// found. On Unix that protection is the 0600/0700 permission bits.
func OwnerOnly(path string, directory bool) (bool, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, "", err
	}
	want := ownerOnlyMode(directory)
	got := info.Mode().Perm()
	return got == want, fmt.Sprintf("mode %04o (want %04o)", got, want), nil
}

func ownerOnlyMode(directory bool) os.FileMode {
	if directory {
		return 0o700
	}
	return OwnerOnlyFilePerm
}
