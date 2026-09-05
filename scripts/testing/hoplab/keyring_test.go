package main

import (
	"errors"
	"os"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/credentials"

	keyring "github.com/zalando/go-keyring"
)

// These tests use go-keyring's own in-memory mock (the same one
// internal/credentials' tests use, keyring.MockInit) rather than the real
// OS credential store, so `go test` never writes to this machine's actual
// Windows Credential Manager.

func TestKeyringShowReportsNoTokenOnAFreshHome(t *testing.T) {
	keyring.MockInit()
	h := BuildDeviceHome(t.TempDir(), "device-a")
	if err := keyringShow(h); err != nil {
		t.Fatalf("keyringShow on a fresh home: %v", err)
	}
}

func TestKeyringShowUsesThePerHomeEntryFromInternalCredentials(t *testing.T) {
	keyring.MockInit()
	lab := t.TempDir()
	h := BuildDeviceHome(lab, "device-a")

	// Write a token directly at the entry internal/credentials.DeviceTokenEntry
	// derives for h's REINSTATE_HOME -- what a real `rein login` run with
	// that REINSTATE_HOME would have written -- and confirm keyringShow
	// finds it without duplicating the entry-name rule itself.
	if err := keyringEntryFor(h, func() error {
		return credentials.NewKeyringStore().SetDeviceToken(credentials.DeviceToken{
			Token: "tok-a", ControlPlaneURL: "http://127.0.0.1:8082", AccountID: "acct-1", DeviceID: "dev-a",
		})
	}); err != nil {
		t.Fatal(err)
	}
	if err := keyringShow(h); err != nil {
		t.Fatalf("keyringShow: %v", err)
	}

	// A different device's home must not see it: internal/credentials
	// derives a distinct entry per REINSTATE_HOME (commit 2521485f).
	other := BuildDeviceHome(lab, "device-b")
	var otherPresent bool
	if err := keyringEntryFor(other, func() error {
		_, err := credentials.NewKeyringStore().GetDeviceToken()
		otherPresent = err == nil
		if errors.Is(err, credentials.ErrNoDeviceToken) {
			return nil
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if otherPresent {
		t.Fatal("device-b's OS-keyring entry already holds a token; the two homes are not isolated")
	}
}

func TestKeyringClearDeviceIsIdempotent(t *testing.T) {
	keyring.MockInit()
	h := BuildDeviceHome(t.TempDir(), "device-a")

	if err := keyringClearDevice(h); err != nil {
		t.Fatalf("clear on an empty entry: %v", err)
	}

	if err := keyringEntryFor(h, func() error {
		return credentials.NewKeyringStore().SetDeviceToken(credentials.DeviceToken{
			Token: "tok-a", ControlPlaneURL: "http://127.0.0.1:8082",
		})
	}); err != nil {
		t.Fatal(err)
	}
	if err := keyringClearDevice(h); err != nil {
		t.Fatalf("clear: %v", err)
	}
	err := keyringEntryFor(h, func() error {
		_, err := credentials.NewKeyringStore().GetDeviceToken()
		return err
	})
	if !errors.Is(err, credentials.ErrNoDeviceToken) {
		t.Fatalf("GetDeviceToken after clear: err=%v, want ErrNoDeviceToken", err)
	}
}

func TestKeyringEntryForRestoresTheOuterEnvironment(t *testing.T) {
	t.Setenv("REINSTATE_HOME", `D:\outer\home`)
	h := BuildDeviceHome(t.TempDir(), "device-a")
	if err := keyringEntryFor(h, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("REINSTATE_HOME"); got != `D:\outer\home` {
		t.Fatalf("REINSTATE_HOME after keyringEntryFor = %q, want the outer value restored", got)
	}
}
