package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	// synthetic lab account's recovery code, the same kind of ephemeral,
	// lab-root-only secret keyring-device-token.json (keyring.go) already
	// stores next to it -- never committed, never outside -root.
	PairingRecoveryCode string `json:"pairing_recovery_code,omitempty"`
}

func statePath(root string) string { return filepath.Join(root, "hoplab-state.json") }

func (s LabState) save() error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(s.Root), b, 0o644)
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
	return s, nil
}

func removeState(root string) error {
	err := os.Remove(statePath(root))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
