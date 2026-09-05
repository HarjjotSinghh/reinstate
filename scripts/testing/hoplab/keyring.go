package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	keyring "github.com/zalando/go-keyring"
)

// These two constants must track internal/credentials/keyring.go's
// keyringService and devicetoken.go's DeviceTokenRef exactly: hoplab reads
// and writes the very entry `rein login` and `rein whoami` use, through the
// same OS keyring library the product already depends on
// (github.com/zalando/go-keyring, already a direct module dependency --
// this file adds no new one). W4 does not own internal/credentials, so this
// mirrors the two names as literals rather than importing the package; a
// credentials_test.go style guard there would catch drift if either name
// ever changes, hoplab_test.go pins the literals used here.
const (
	hopKeyringService  = "reinstate"
	hopDeviceTokenName = "hop/device-token"
)

// keyringSnapshotPath is where `hoplab keyring save` writes one device's
// captured token, and where `load`/`clear` read it back from.
func keyringSnapshotPath(labRoot, device string) string {
	return filepath.Join(labRoot, device, "keyring-device-token.json")
}

// keyringSave copies the OS keyring's current Hop device token into a file
// under the named device's directory, so `rein login` can run again for a
// different device without losing the first one: the real, single-slot OS
// keyring is a hoplab limitation the type doc on DeviceHome explains, and
// this pair of commands (save/load) is the workaround for a sequential,
// real-binary two-device walk.
func keyringSave(labRoot, device string) error {
	raw, err := keyring.Get(hopKeyringService, hopDeviceTokenName)
	if errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("no device token is in the OS keyring right now; sign this device in first (`rein login`)")
	}
	if err != nil {
		return fmt.Errorf("read the OS keyring: %w", err)
	}
	path := keyringSnapshotPath(labRoot, device)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	// The captured value is the product's own device-token JSON
	// (credentials.DeviceToken, opaque here); indent for a readable diff,
	// this file never leaves the lab root and the secret scanner covers it
	// like any other testdata/results artifact.
	var pretty json.RawMessage = json.RawMessage(raw)
	buf, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		// Not JSON for some reason -- still capture it verbatim rather than
		// fail the snapshot.
		return os.WriteFile(path, []byte(raw), 0o600)
	}
	return os.WriteFile(path, buf, 0o600)
}

// keyringLoad restores a device's captured token into the OS keyring,
// making it the account `rein` sees until the next login or load.
func keyringLoad(labRoot, device string) error {
	path := keyringSnapshotPath(labRoot, device)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no saved token for %s; run `hoplab keyring save --device %s` after signing that device in", device, device)
		}
		return err
	}
	// Un-indent back to the single-line form the product itself writes,
	// though keyring.Set stores whatever string it is given either way.
	var compact map[string]any
	value := string(raw)
	if json.Unmarshal(raw, &compact) == nil {
		if b, err := json.Marshal(compact); err == nil {
			value = string(b)
		}
	}
	return keyring.Set(hopKeyringService, hopDeviceTokenName, value)
}

// keyringClear removes whatever device token is currently in the OS
// keyring (not any saved snapshot), the way `rein login` again from a
// clean slate needs.
func keyringClear() error {
	err := keyring.Delete(hopKeyringService, hopDeviceTokenName)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
