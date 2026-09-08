package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LabState is what one `hoplab start` run records under its lab root, so a
// later `hoplab stop`, `hoplab approve`, or `hoplab env` invocation (a
// separate process, possibly in another terminal) can find the running
// hopd and fakelocker without the caller re-typing every address.
type LabState struct {
	Root        string `json:"root"`
	HopdPID     int    `json:"hopd_pid"`
	HopdAddr    string `json:"hopd_addr"`
	HopdBaseURL string `json:"hopd_base_url"`
	HopdLog     string `json:"hopd_log"`
	HopdDB      string `json:"hopd_db"`
	LockerPID   int    `json:"locker_pid"`
	LockerAddr  string `json:"locker_addr"`
	// PairingRecoveryCode is the code `hoplab pair init` captured from the
	// first device's `rein account init`, so `hoplab pair join` can enrol
	// every later device without the caller copying it by hand. It is a
	// synthetic lab account's recovery code -- deliberately excluded from
	// this struct's own JSON (json:"-"): hoplab-state.json is a shared,
	// world-readable (0644) file meant to be read and copied around freely
	// (pids, log paths, addresses), so the code itself must never round-trip
	// through it in the clear. save()/loadState() instead shuttle it through
	// a dedicated, mode-0600 sibling file (recoveryCodePath, below) that
	// never leaves -root and is never committed -- the same treatment
	// keyring-device-token.json (keyring.go) already gets from the OS
	// keyring next to it.
	PairingRecoveryCode string `json:"-"`
}

func statePath(root string) string { return filepath.Join(root, "hoplab-state.json") }

// recoveryCodePath is the mode-restricted sibling file that actually holds
// a `hoplab pair init` recovery code, kept out of hoplab-state.json's own
// JSON so a shared, world-readable state file never carries account-recovery
// secret material. See PairingRecoveryCode's doc comment.
func recoveryCodePath(root string) string {
	return filepath.Join(root, "hoplab-recovery-code.secret")
}

// saveRecoveryCode writes code to root's mode-0600 sibling file, or removes
// that file when code is empty (a LabState with no pairing recovery code
// yet -- the common case for every action except `pair init`).
func saveRecoveryCode(root, code string) error {
	if strings.TrimSpace(code) == "" {
		if err := os.Remove(recoveryCodePath(root)); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.WriteFile(recoveryCodePath(root), []byte(code+"\n"), 0o600)
}

// loadRecoveryCode reads back whatever saveRecoveryCode last wrote, trimmed
// of the trailing newline. Its own os.IsNotExist error surfaces unchanged so
// callers can tell "no code saved yet" apart from a real read failure, the
// same distinction loadState already makes for a missing hoplab-state.json.
func loadRecoveryCode(root string) (string, error) {
	b, err := os.ReadFile(recoveryCodePath(root))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (s LabState) save() error {
	if err := saveRecoveryCode(s.Root, s.PairingRecoveryCode); err != nil {
		return fmt.Errorf("write %s: %w", recoveryCodePath(s.Root), err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(statePath(s.Root), b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", statePath(s.Root), err)
	}
	return nil
}

func loadState(root string) (LabState, error) {
	var s LabState
	b, err := os.ReadFile(statePath(root))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	code, err := loadRecoveryCode(root)
	if err != nil && !os.IsNotExist(err) {
		return s, err
	}
	s.PairingRecoveryCode = code
	return s, nil
}

func removeState(root string) error {
	err := os.Remove(statePath(root))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if rcErr := os.Remove(recoveryCodePath(root)); rcErr != nil && !os.IsNotExist(rcErr) {
		return rcErr
	}
	return nil
}
