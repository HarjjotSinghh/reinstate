package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// The account-wide key generation floor, and why the per-device one is not
// enough on its own.
//
// A device records the generation it last unwrapped and refuses a keyring
// below it. That protects a device which has already seen the rollover. It
// does nothing for the device that has not: a revoked machine, inside the
// credential window its docs describe, can put the *genuine* pre-rollover
// keyring back, and a device that has run nothing since accepts it — every
// signature in it verifies, because the account itself wrote it — and keeps
// sealing what it pushes to the root key the revoked device still holds.
//
// The control plane is the party that can see the rollover such a device
// missed. It authenticates every device and refuses a revoked one, and it
// carries the account's key generation as a number that only ever goes up.
// Every route that acts on the keyring asks for it and refuses a keyring
// below it.
//
// Two things this does not do, both stated in docs/hop.md and
// docs/security-model.md rather than left for a reader to discover:
//
//   - Against an operator holding the control plane **and** the bucket it
//     adds nothing at all: that party serves whatever floor suits it. The
//     recovery-code signature on every generation and the local anchor are
//     what address that adversary, as far as they do.
//   - It covers Hop profiles. A profile on your own bucket has no control
//     plane to ask, so on those the per-device floor and the local anchor
//     remain the whole of it; `rein devices revoke` needs a signed-in
//     account, so revocation is a Hop feature either way.

// keyringFloorSource names where a floor came from, so a refusal can say
// which party is reporting the account has moved on.
type keyringFloorSource string

const (
	// floorFromLocalRecord is the generation this device last unwrapped.
	floorFromLocalRecord keyringFloorSource = "this device has already seen"
	// floorFromControlPlane is a floor the control plane answered with just
	// now.
	floorFromControlPlane keyringFloorSource = "the control plane reports for this account"
	// floorFromLastConfirmed is the last floor the control plane confirmed
	// to this device, used when the control plane no longer serves one.
	floorFromLastConfirmed keyringFloorSource = "the control plane last confirmed to this device"
	// floorFromNoControlPlane is the decision recorded for a profile that
	// has no control plane to ask: BYO storage. The floor is 0, so it
	// refuses nothing, and the per-device anchor does the whole job.
	floorFromNoControlPlane keyringFloorSource = "not carried for this profile"
)

// keyringFloor is one route's decision about the account-wide floor.
//
// The zero value is deliberately unusable: `decided` is false, and
// keyringAnchor.check refuses an anchor carrying it. A route that reads the
// keyring without deciding therefore fails closed with a message naming the
// omission, rather than silently reverting to the per-device floor. See
// TestEveryKeyringAnchorDecidesTheFloor, which will not let a new anchor
// into internal/cli without one.
type keyringFloor struct {
	decided     bool
	generation  int
	source      keyringFloorSource
	confirmedAt string
}

// noKeyGenerationFloor is the decision for a route that genuinely has no
// control plane to ask. reason is recorded for the error message a later
// refusal would print.
func noKeyGenerationFloor(source keyringFloorSource) keyringFloor {
	return keyringFloor{decided: true, source: source}
}

// keyringFloorUndecidedError reports an anchor built without deciding about
// the account-wide floor. It should be unreachable — every constructor in
// this package decides — and is kept as a refusal rather than a skip so
// that a route added later which builds an anchor without asking fails
// closed instead of quietly reverting to the per-device floor.
//
// It covers routes that go through keyringAnchor, which is every route that
// reads the keyring today, and TestEveryKeyringAnchorDecidesTheFloor fails
// on any new anchor built without a floor. A route that unwrapped a keyring
// without building an anchor at all would evade both; trustKeyring is the
// only way this package unwraps, and it takes an anchor.
type keyringFloorUndecidedError struct{}

func (e *keyringFloorUndecidedError) Error() string {
	return "this command did not establish the account's key generation floor, so it cannot tell a current keyring from one a revoked device restored. Nothing was written; this is a bug in Reinstate — please report it with the command you ran"
}

// confirmKeyGenerationFloor establishes the account-wide floor for one
// command, on whatever profile it is running against.
//
// On a Hop profile it asks the control plane. That call adds no dependency
// this command did not already have: reaching a Hop locker means minting
// credentials from the same control plane in the same command, so a device
// that cannot reach it gets no locker credentials either. There is no state
// in which a device can sync but not ask, and so nothing here trades
// availability for the check.
//
// Whatever the control plane answers, the floor used is the higher of that
// and the last floor this device had confirmed, which account.json records
// and which only ever rises. That covers the two ways a live answer can be
// missing or lower than the truth — a deployment that stops serving the
// route (404, an older control plane) and one that answers with a number
// below what it has already told this device — without either of them
// dropping the account back to generation 0. It is not a defence against a
// control plane that is hostile from the start: such a party can serve 0 to
// a device that has never confirmed anything, which is the limit stated in
// docs/security-model.md.
//
// On a profile using your own bucket there is no control plane, and the
// decision recorded says so.
func confirmKeyGenerationFloor(cmd *cobra.Command, home string, cfg *schema.Config) (keyringFloor, error) {
	if cfg == nil || cfg.Storage.Type != schema.StorageHop {
		return noKeyGenerationFloor(floorFromNoControlPlane), nil
	}
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return keyringFloor{}, err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	confirmed, err := lastConfirmedKeyGenerationFloor(home)
	if err != nil {
		return keyringFloor{}, err
	}
	got, err := client.KeyGenerationFloor(ctx, tok.Token)
	if errors.Is(err, hop.ErrNoKeyGenerationFloor) {
		return confirmed, nil
	}
	if err != nil {
		return keyringFloor{}, hopExitError(err)
	}
	floor := keyringFloor{
		decided:     true,
		generation:  got.Generation,
		source:      floorFromControlPlane,
		confirmedAt: got.RaisedAt,
	}
	if err := rememberKeyGenerationFloor(home, floor); err != nil {
		return keyringFloor{}, err
	}
	if confirmed.generation > floor.generation {
		// The control plane is answering below what it has already told
		// this device. Keep the higher number and say where it came from.
		return confirmed, nil
	}
	return floor, nil
}

// lastConfirmedKeyGenerationFloor reads the floor this device last had
// confirmed. A device with no enrolment record, or one that has never
// reached a control plane carrying a floor, gets 0 — which refuses nothing,
// and is the honest answer: nothing has ever told this device the account
// moved on.
func lastConfirmedKeyGenerationFloor(home string) (keyringFloor, error) {
	account, err := config.LoadAccount(home)
	if os.IsNotExist(err) {
		return noKeyGenerationFloor(floorFromLastConfirmed), nil
	}
	if err != nil {
		return keyringFloor{}, NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	return keyringFloor{
		decided:     true,
		generation:  account.ControlPlaneKeyGeneration,
		source:      floorFromLastConfirmed,
		confirmedAt: account.ControlPlaneConfirmedAt,
	}, nil
}

// rememberKeyGenerationFloor records a confirmed floor in the enrolment
// record, moving it up only. A device with no record yet (it is joining, or
// recovering) has nowhere to put it and keeps the live answer for this
// command only; the enrolment written at the end of those commands records
// the floor then.
func rememberKeyGenerationFloor(home string, floor keyringFloor) error {
	if !floor.decided || floor.generation <= 0 {
		return nil
	}
	account, err := config.LoadAccount(home)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	if account.ControlPlaneKeyGeneration >= floor.generation {
		return nil
	}
	account.ControlPlaneKeyGeneration = floor.generation
	account.ControlPlaneConfirmedAt = time.Now().UTC().Format(time.RFC3339)
	if err := config.SaveAccount(home, account); err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	return nil
}

// raiseKeyGenerationFloor tells the control plane the account has reached
// generation, after a rollover has landed in the keyring. It is the only
// call that moves the account-wide floor, and the control plane keeps the
// higher of the two numbers, so a device whose view is behind cannot lower
// it and neither can the device being revoked.
//
// It reports three outcomes, because they need three different messages:
// raised, not carried by this control plane (an older deployment — the
// revocation stands, but a device that has not read the keyring is not
// covered by anything), and failed.
func raiseKeyGenerationFloor(cmd *cobra.Command, home string, cfg *schema.Config, generation int) (carried bool, err error) {
	if cfg == nil || cfg.Storage.Type != schema.StorageHop || generation < 1 {
		return false, nil
	}
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return false, err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	got, err := client.RaiseKeyGenerationFloor(ctx, tok.Token, generation)
	if errors.Is(err, hop.ErrNoKeyGenerationFloor) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("raise the account's key generation floor to %d: %w", generation, err)
	}
	floor := keyringFloor{decided: true, generation: got.Generation, source: floorFromControlPlane, confirmedAt: got.RaisedAt}
	if err := rememberKeyGenerationFloor(home, floor); err != nil {
		return true, err
	}
	return true, nil
}
