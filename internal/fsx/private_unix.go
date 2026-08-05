//go:build !windows

package fsx

import "os"

// ProtectOwnerOnly removes group/other access from one private path.
func ProtectOwnerOnly(path string, directory bool) error {
	mode := os.FileMode(0o600)
	if directory {
		mode = 0o700
	}
	return os.Chmod(path, mode)
}
