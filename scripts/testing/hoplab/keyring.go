package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/HarjjotSinghh/reinstate/internal/credentials"
)

// keyring.go is an optional, read-only-by-default diagnostic, not a step
// any pairing flow needs any more. Every real `rein` process hoplab
// launches (pair.go's reinEnviron) gets REINSTATE_HOME set to the acting
// device's own home, and internal/credentials.DeviceTokenEntry derives a
// distinct OS-keyring entry from that home (commit 2521485f: "give each
// Reinstate home its own device-token entry") -- so device-a and device-b
// already hold separate tokens with no save/load swap needed, the way an
// earlier version of this package required (see git history for
// keyringSave/keyringLoad, and docs/testing/windows-acceptance-host.md's
// Hop lab section for what the swap workaround was covering for). This
// file imports internal/credentials rather than duplicating its
// service/entry-name rule, per the same instruction that removed the
// duplicated literals.
//
// `hoplab keyring show`/`clear` exist only to answer "what did rein login
// actually write, and where" when something looks wrong -- never a
// password manager. show never prints the token itself, only where it
// lives and whether it is there.

func keyringUsage(w *os.File) {
	fmt.Fprint(w, `usage: hoplab keyring <show|clear> -root <dir> -device <name>

Optional diagnostic only -- no pairing flow needs this any more (every
device's REINSTATE_HOME already gets its own OS-keyring entry; see
internal/credentials.DeviceTokenEntry).

  show   report the OS-keyring entry -device's REINSTATE_HOME maps to, and
         whether a device token currently sits there (never prints the
         token itself).
  clear  remove whatever device token sits at that entry, for a clean-slate
         'rein login' as this device.
`)
}

// keyringEntryFor runs fn with REINSTATE_HOME set to h's Reinstate home --
// exactly what a real `rein` subprocess for h would see (pair.go's
// reinEnviron) -- so credentials.DeviceTokenEntry() resolves the same
// entry name `rein login`/`rein whoami` would use for this device. The
// process-wide env var is restored afterward; callers must not run this
// concurrently with anything else that reads or sets REINSTATE_HOME in
// this same process (fine for a single hoplab command invocation, which
// is the only place this runs).
func keyringEntryFor(h DeviceHome, fn func() error) error {
	const homeEnv = "REINSTATE_HOME"
	old, had := os.LookupEnv(homeEnv)
	if err := os.Setenv(homeEnv, h.ReinstateHome); err != nil {
		return err
	}
	defer func() {
		if had {
			_ = os.Setenv(homeEnv, old)
		} else {
			_ = os.Unsetenv(homeEnv)
		}
	}()
	return fn()
}

// keyringShow reports the OS-keyring entry h's REINSTATE_HOME maps to, and
// the non-secret fields of whatever device token is stored there (never
// the bearer token itself).
func keyringShow(h DeviceHome) error {
	var entry string
	var tok credentials.DeviceToken
	var present bool
	err := keyringEntryFor(h, func() error {
		entry = credentials.DeviceTokenEntry()
		t, err := credentials.NewKeyringStore().GetDeviceToken()
		switch {
		case err == nil:
			tok, present = t, true
			return nil
		case errors.Is(err, credentials.ErrNoDeviceToken):
			return nil
		default:
			return err
		}
	})
	if err != nil {
		return fmt.Errorf("read the OS keyring for %s: %w", h.Name, err)
	}
	fmt.Fprintf(os.Stderr, "hoplab: %s -> REINSTATE_HOME=%s -> OS-keyring entry %q\n", h.Name, h.ReinstateHome, entry)
	if !present {
		fmt.Fprintf(os.Stderr, "hoplab: no device token there; %s has not run `rein login` (with this REINSTATE_HOME), or it was cleared\n", h.Name)
		return nil
	}
	fmt.Fprintf(os.Stderr, "hoplab: device token present: control_plane_url=%s account_id=%s device_id=%s\n", tok.ControlPlaneURL, tok.AccountID, tok.DeviceID)
	return nil
}

// keyringClearDevice removes h's device token from the OS keyring, for a
// clean-slate `rein login` as this device. A missing entry is not an
// error.
func keyringClearDevice(h DeviceHome) error {
	err := keyringEntryFor(h, func() error {
		return credentials.NewKeyringStore().DeleteDeviceToken()
	})
	if err != nil {
		return fmt.Errorf("clear the OS keyring entry for %s: %w", h.Name, err)
	}
	return nil
}
