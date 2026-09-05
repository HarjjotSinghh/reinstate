//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

// hoplab pair is a Windows lab tool (like the rest of this package's
// process-spawning pieces); this stub keeps `go build ./...` and
// `go vet ./...` clean on every other GOOS, matching
// scripts/testing/conptydriver's conpty_other.go convention.

type liveSecretFD struct{}

func newLiveSecretFD() (*liveSecretFD, error) {
	return nil, fmt.Errorf("hoplab pair needs REINSTATE_RECOVERY_CODE_FD handle-passing, implemented for windows only")
}

func (f *liveSecretFD) ChildValue() string { return "" }
func (f *liveSecretFD) ApplyTo(*exec.Cmd)  {}
func (f *liveSecretFD) CloseReadInParent() {}
func (f *liveSecretFD) Feed(string) error  { return fmt.Errorf("not supported on this OS") }
func (f *liveSecretFD) Close()             {}

type fixedSecretFD struct{}

func newFixedSecretFD(string) (*fixedSecretFD, error) {
	return nil, fmt.Errorf("hoplab pair needs REINSTATE_RECOVERY_CODE_FD handle-passing, implemented for windows only")
}

func (f *fixedSecretFD) ChildValue() string { return "" }
func (f *fixedSecretFD) ApplyTo(*exec.Cmd)  {}
func (f *fixedSecretFD) Close()             {}
