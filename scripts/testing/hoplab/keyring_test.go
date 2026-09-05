package main

import (
	"strings"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

// These tests use go-keyring's own in-memory mock (the same one
// internal/credentials' tests use, keyring.MockInit) rather than the real
// OS credential store, so `go test` never writes to this machine's actual
// Windows Credential Manager.

func TestKeyringSaveRequiresASignedInDevice(t *testing.T) {
	keyring.MockInit()
	lab := t.TempDir()
	err := keyringSave(lab, "device-a")
	if err == nil {
		t.Fatal("keyringSave: want an error when the OS keyring holds no device token")
	}
	if !strings.Contains(err.Error(), "rein login") {
		t.Fatalf("error = %v, want it to say what to do", err)
	}
}

func TestKeyringLoadRequiresASavedSnapshot(t *testing.T) {
	keyring.MockInit()
	lab := t.TempDir()
	err := keyringLoad(lab, "device-a")
	if err == nil {
		t.Fatal("keyringLoad: want an error when there is nothing saved for this device")
	}
	if !strings.Contains(err.Error(), "hoplab keyring save") {
		t.Fatalf("error = %v, want it to say what to do", err)
	}
}

func TestKeyringSaveLoadSwapsTwoDevices(t *testing.T) {
	keyring.MockInit()
	lab := t.TempDir()

	// Device A signs in.
	tokenA := `{"token":"tok-a","control_plane_url":"http://127.0.0.1:8082","account_id":"acct-1","device_id":"dev-a"}`
	if err := keyring.Set(hopKeyringService, hopDeviceTokenName, tokenA); err != nil {
		t.Fatal(err)
	}
	if err := keyringSave(lab, "device-a"); err != nil {
		t.Fatalf("keyringSave device-a: %v", err)
	}

	// Device B signs in, replacing the OS keyring's one slot.
	tokenB := `{"token":"tok-b","control_plane_url":"http://127.0.0.1:8082","account_id":"acct-1","device_id":"dev-b"}`
	if err := keyring.Set(hopKeyringService, hopDeviceTokenName, tokenB); err != nil {
		t.Fatal(err)
	}
	if err := keyringSave(lab, "device-b"); err != nil {
		t.Fatalf("keyringSave device-b: %v", err)
	}

	// The keyring currently holds device B; loading device A must restore
	// its token, not device B's.
	if err := keyringLoad(lab, "device-a"); err != nil {
		t.Fatalf("keyringLoad device-a: %v", err)
	}
	got, err := keyring.Get(hopKeyringService, hopDeviceTokenName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "dev-a") || strings.Contains(got, "dev-b") {
		t.Fatalf("after loading device-a, OS keyring holds %q, want device-a's token", got)
	}

	// And back to device B.
	if err := keyringLoad(lab, "device-b"); err != nil {
		t.Fatalf("keyringLoad device-b: %v", err)
	}
	got, err = keyring.Get(hopKeyringService, hopDeviceTokenName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "dev-b") {
		t.Fatalf("after loading device-b, OS keyring holds %q, want device-b's token", got)
	}
}

func TestKeyringClearIsIdempotent(t *testing.T) {
	keyring.MockInit()
	if err := keyringClear(); err != nil {
		t.Fatalf("keyringClear on an empty keyring: %v", err)
	}
	if err := keyring.Set(hopKeyringService, hopDeviceTokenName, "x"); err != nil {
		t.Fatal(err)
	}
	if err := keyringClear(); err != nil {
		t.Fatalf("keyringClear: %v", err)
	}
	if _, err := keyring.Get(hopKeyringService, hopDeviceTokenName); err != keyring.ErrNotFound {
		t.Fatalf("Get after clear: err=%v, want ErrNotFound", err)
	}
}

func TestKeyringSnapshotPathIsPerDevice(t *testing.T) {
	pa := keyringSnapshotPath(`D:\lab`, "device-a")
	pb := keyringSnapshotPath(`D:\lab`, "device-b")
	if pa == pb {
		t.Fatalf("keyringSnapshotPath gave the same path for both devices: %q", pa)
	}
}
