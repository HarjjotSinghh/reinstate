//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// processExists reports whether pid is a live process this user can see.
func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}
