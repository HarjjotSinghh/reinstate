//go:build !windows

package main

import (
	"os"
	"syscall"
)

// processAliveNative reports whether pid names a currently running
// process. On POSIX, os.FindProcess always succeeds; signal 0 probes
// existence without affecting the process. hoplab itself is a Windows lab
// tool (like its other process-spawning pieces), but this keeps `go
// build`/`go vet` clean on every other GOOS.
func processAliveNative(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
