package cli

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// readPairingCode reads a pairing code without echo on the approving
// device. Automation sets REINSTATE_PAIRING_CODE_FD; the code is never a
// flag or a plain variable.
func (s accountSeams) readPairingCode(cmd *cobra.Command, prompt string) ([]byte, error) {
	if s.pairingPrompt != nil {
		return s.pairingPrompt(prompt)
	}
	if secret, configured, err := crypto.ReadSecretFD(crypto.PairingCodeFDEnv); configured {
		return secret, err
	}
	return crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), prompt)
}

// newAccountJoinCmd is the joining device's half of device approval.
func newAccountJoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Enrol this device by approval from an already-enrolled device",
		Long: `Join the account from a new device with nothing typed here.

This device shows a short pairing code and waits. On any already-enrolled
device, run rein devices approve and enter the code; the root key arrives
wrapped so that neither the control plane nor anyone without the code can
read it, and this device can then read everything in the locker.

If no other device is available, rein account recover enrols this device
from the recovery code instead.`,
		Args: cobra.NoArgs,
		RunE: runAccountJoin,
	}
	return cmd
}

func runAccountJoin(cmd *cobra.Command, _ []string) error {
	seams := accountSeamsFrom(cmd)
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return err
	}
	home, cfg, err := loadAccountHome()
	if err != nil {
		return err
	}
	if _, err := config.LoadAccount(home); err == nil {
		return NewExitError(ExitSafety, alreadyEnrolledHere)
	} else if !os.IsNotExist(err) {
		return NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	if tok.DeviceID != "" && tok.DeviceID != cfg.DeviceID {
		return NewExitError(ExitConfig, fmt.Sprintf("this home's device_id (%s) is not the signed-in device (%s); run rein init --hop so the keyring and the control plane agree on this device's identity", cfg.DeviceID, tok.DeviceID))
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	store, prefix, err := backendFromConfig(cmd, cfg, home)
	if err != nil {
		return err
	}
	keyringKey := keyring.ObjectKey(prefix)
	ring, _, err := keyring.Load(ctx, store, keyringKey)
	if errors.Is(err, keyring.ErrNotFound) {
		return NewExitError(ExitAuthStorage, "no keyring exists for this profile yet; run rein account init on the first device, then rein account join here")
	}
	if exit := exitForKeyringRefusal(err); exit != nil {
		return exit
	}
	if err != nil {
		return NewExitError(ExitAuthStorage, err.Error())
	}
	if ring.ProfileID != cfg.ProfileID {
		return NewExitError(ExitConfig, fmt.Sprintf("keyring belongs to profile %s, this home is configured for %s", ring.ProfileID, cfg.ProfileID))
	}

	now := time.Now().UTC()
	secrets := seams.secretStore()
	secretRef := deviceSecretRef(cfg.ProfileID, cfg.DeviceID)
	deviceKey, err := loadDeviceKey(secrets, secretRef)
	if err != nil {
		return err
	}
	// A keyring that already lists this device with this machine's key is
	// never taken as proof of enrolment: the public key is published in
	// the pairing request, so a control plane that also holds the bucket
	// could forge a keyring wrapping its own root key for it. Joining
	// always goes through a fresh request and an approval typed on an
	// enrolled device; the approver re-seals for a listed key without
	// appending a second wrap.
	if deviceKey == nil {
		// The key is generated and stored before the request exists, so a
		// crash between the two leaves nothing an approval could target.
		deviceKey, err = age.GenerateX25519Identity()
		if err != nil {
			return NewExitError(ExitRuntime, err.Error())
		}
		if err := secrets.SetSecret(secretRef, []byte(deviceKey.String())); err != nil {
			return NewExitError(ExitAuthStorage, err.Error())
		}
	} else if listed := ring.DevicePublicKey(cfg.DeviceID); listed != "" && listed != deviceKey.Recipient().String() {
		return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s with a different key than this machine holds; nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device", cfg.DeviceID))
	}

	pairing, err := keyring.NewPairing()
	if err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	defer pairing.Zero()
	publicKey := deviceKey.Recipient().String()
	req, err := client.CreatePairing(ctx, tok.Token, publicKey,
		base64.StdEncoding.EncodeToString(pairing.Salt), pairing.Binding(publicKey))
	if err != nil {
		return hopExitError(err)
	}

	errOut := cmd.ErrOrStderr()
	PrintHuman(errOut, "")
	PrintHuman(errOut, "Pairing code for this device (never sent to the control plane):")
	PrintHuman(errOut, "")
	PrintHuman(errOut, "    %s", pairing.Code)
	PrintHuman(errOut, "")
	PrintHuman(errOut, "On an already-enrolled device, run:  rein devices approve")
	PrintHuman(errOut, "and enter the code exactly as shown. The request expires at %s (Ctrl-C to cancel).", req.ExpiresAt)

	sleep := hopSeamsFrom(cmd).sleep
	payload, err := client.WaitForPairing(ctx, tok.Token, req, sleep)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			_ = client.ExpirePairing(context.Background(), tok.Token, req.ID)
			return NewExitError(ExitRuntime, "pairing cancelled; the request was withdrawn")
		}
		return hopExitError(err)
	}
	rootKey, err := pairing.OpenRootKey(payload.Payload, req.ID, deviceKey, payload.KeyGeneration)
	if err != nil {
		return NewExitError(ExitSafety, fmt.Sprintf("the received root key could not be opened (%v); nothing was written. Run rein account join again", err))
	}
	defer crypto.Zero(rootKey)

	// Trust, then verify against storage: the keyring's wrap for this
	// device must decrypt to the same root key of the same generation, so
	// a control plane that altered either channel is caught before any
	// data is written under a wrong key.
	floor, err := confirmKeyGenerationFloor(cmd, home, cfg)
	if err != nil {
		return err
	}
	ring, err = verifyJoinedKeyring(ctx, store, keyringKey, cfg.DeviceID, deviceKey, rootKey, payload.KeyGeneration, floor)
	if err != nil {
		return err
	}
	if err := saveAccountEnrolmentConfirmed(home, cfg, now, ring, "join", false); err != nil {
		return err
	}
	// The record exists only now, so the floor confirmed above had nowhere
	// to be written until this point.
	if err := rememberKeyGenerationFloor(home, floor); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	approver := payload.ApprovedBy.Name
	if approver == "" {
		approver = "another device"
	} else {
		approver = fmt.Sprintf("%q", approver)
	}
	PrintHuman(out, "device approved by %s; this device can now read the locker", approver)
	PrintHuman(out, "profile_id=%s device_id=%s key_generation=%d", cfg.ProfileID, cfg.DeviceID, payload.KeyGeneration)
	return nil
}

// verifyJoinedKeyring reloads the keyring and confirms it lists this device
// with a wrap that opens to exactly the root key received through the
// pairing, in the generation the approver named.
//
// A joining device has no local anchor of its own: it has never read this
// account's keyring, so it cannot say whose account key the object is signed
// under. Every generation must verify under whatever key the object
// publishes — that much is checked whether or not there is an anchor — and
// the thing that makes this account's keyring is the root key an enrolled
// device relayed through the approval, matched here against the keyring's
// own wrap for this device. Both channels must agree, so neither the relay
// nor the storage can substitute a keyring alone. The account key is pinned
// from this object once they do, and every later read is checked against it.
//
// floor is the account-wide key generation the control plane reports, and it
// applies here too: an approval relaying a genuine older generation, and a
// bucket holding the matching older keyring, agree with each other and would
// otherwise satisfy the paragraph above.
func verifyJoinedKeyring(ctx context.Context, store backend.Backend, key, deviceID string, deviceKey *age.X25519Identity, rootKey []byte, generation int, floor keyringFloor) (*keyring.Keyring, error) {
	ring, _, err := keyring.Load(ctx, store, key)
	if exit := exitForKeyringRefusal(err); exit != nil {
		return nil, exit
	}
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	if ring.CurrentGeneration != generation {
		return nil, NewExitError(ExitSafety, fmt.Sprintf("the keyring is at generation %d but the approval named %d; nothing was written. Run rein account join again", ring.CurrentGeneration, generation))
	}
	keys, err := trustKeyring(keyringAnchor{floor: floor}, ring, func() (map[int][]byte, error) {
		return ring.UnwrapGenerations(deviceID, deviceKey)
	})
	if exit := exitForKeyringRefusal(err); exit != nil {
		return nil, exit
	}
	if err != nil {
		return nil, NewExitError(ExitSafety, fmt.Sprintf("the keyring does not hold a working wrap for this device (%v); nothing was written. Run rein account join again", err))
	}
	defer keyring.ZeroGenerations(keys)
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		return nil, NewExitError(ExitSafety, err.Error())
	}
	if subtle.ConstantTimeCompare(keys[ring.CurrentGeneration], rootKey) != 1 || identity.Recipient().String() != ring.CurrentRecipient() {
		return nil, NewExitError(ExitSafety, "the root key received through the pairing does not match the keyring's current generation; nothing was written. The storage or relay may have been tampered with")
	}
	return ring, nil
}

// newDevicesCmd lists enrolled devices and approves pairing requests.
func newDevicesCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "devices",
		Short: "List this account's devices, approve new ones, revoke lost ones",
		Args:  cobra.NoArgs,
		RunE:  runDevicesList,
	}
	root.AddCommand(newDevicesApproveCmd(), newDevicesRevokeCmd())
	root.PersistentFlags().Bool("json", false, "emit machine-readable JSON")
	return root
}

func runDevicesList(cmd *cobra.Command, _ []string) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	devices, err := client.Devices(ctx, tok.Token)
	if err != nil {
		return hopExitError(err)
	}
	pending, err := client.PendingPairings(ctx, tok.Token)
	if err != nil {
		return hopExitError(err)
	}
	// The keyring is the key-model truth; the control plane only knows
	// enrolment. Best effort: a device with no configured storage still
	// gets the control-plane view.
	inKeyring := map[string]bool{}
	keyringSeen := false
	keyringRefused := ""
	generation := 0
	if home, cfg, err := loadAccountHome(); err == nil {
		if store, prefix, err := backendFromConfig(cmd, cfg, home); err == nil {
			// Listing holds no key of any kind and needs none: the
			// generation signatures verify against the account key alone.
			// A keyring signed by another key, rolled back, or replaced
			// under this device must not be reported as the account's
			// key-model truth, so a refusal here is printed as a refusal.
			ring, _, loadErr := keyring.Load(ctx, store, keyring.ObjectKey(prefix))
			switch {
			case loadErr != nil:
				if exitForKeyringRefusal(loadErr) != nil {
					keyringRefused = loadErr.Error()
				}
			default:
				anchor, anchorErr := loadKeyringAnchor(cmd, home, cfg)
				if anchorErr == nil {
					anchorErr = anchor.check(ring)
				}
				if anchorErr != nil {
					keyringRefused = anchorErr.Error()
				} else {
					keyringSeen = true
					generation = ring.CurrentGeneration
					for _, d := range devices {
						inKeyring[d.ID] = ring.HasDevice(d.ID)
					}
				}
			}
		}
	}
	if asJSON {
		type deviceRow struct {
			hop.Device
			InKeyring *bool `json:"in_keyring,omitempty"`
		}
		rows := make([]deviceRow, 0, len(devices))
		for _, d := range devices {
			row := deviceRow{Device: d}
			if keyringSeen {
				v := inKeyring[d.ID]
				row.InKeyring = &v
			}
			rows = append(rows, row)
		}
		report := map[string]any{"devices": rows, "pending_pairings": pending}
		if keyringSeen {
			report["key_generation"] = generation
		}
		if keyringRefused != "" {
			report["keyring_error"] = keyringRefused
		}
		return WriteJSON(cmd.OutOrStdout(), report)
	}
	out := cmd.OutOrStdout()
	if keyringRefused != "" {
		PrintHuman(out, "keyring: this device refuses it (%s)", keyringRefused)
	}
	for _, d := range devices {
		line := fmt.Sprintf("%s  %s (%s), enrolled %s, last seen %s", d.ID, d.Name, d.Platform, d.CreatedAt, d.LastSeenAt)
		switch {
		case d.Revoked():
			line += ", revoked " + d.RevokedAt
		case keyringSeen && inKeyring[d.ID]:
			line += fmt.Sprintf(", holds a root-key wrap (key generation %d)", generation)
		case keyringSeen:
			line += ", no root-key wrap yet"
		}
		PrintHuman(out, "%s", line)
	}
	if len(pending) == 0 {
		PrintHuman(out, "no pending pairing requests")
		return nil
	}
	for _, p := range pending {
		PrintHuman(out, "pending approval: %s (%s) asked to join at %s (request %s, expires %s)", p.Device.Name, p.Device.Platform, p.CreatedAt, p.ID, p.ExpiresAt)
	}
	PrintHuman(out, "run rein devices approve and enter the code shown on the new device")
	return nil
}

func newDevicesApproveCmd() *cobra.Command {
	var requestID string
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Approve a new device by entering the code it shows",
		Long: `Approve a pending pairing request from this already-enrolled device.

Reads the code shown on the joining device (hidden prompt, or
REINSTATE_PAIRING_CODE_FD for automation), checks it against the request,
appends a root-key wrap for the new device to the keyring, and relays the
root key sealed so that only the code holder can open it. A wrong code
approves nothing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevicesApprove(cmd, requestID)
		},
	}
	cmd.Flags().StringVar(&requestID, "request", "", "approve this pairing request id (required when several are pending)")
	return cmd
}

func runDevicesApprove(cmd *cobra.Command, requestID string) error {
	seams := accountSeamsFrom(cmd)
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	pending, err := client.PendingPairings(ctx, tok.Token)
	if err != nil {
		return hopExitError(err)
	}
	var req *hop.PairingRequest
	switch {
	case requestID != "":
		for i := range pending {
			if pending[i].ID == requestID {
				req = &pending[i]
			}
		}
		if req == nil {
			return NewExitError(ExitUsage, fmt.Sprintf("no pending pairing request %s; rein devices lists the open ones", requestID))
		}
	case len(pending) == 0:
		return NewExitError(ExitUsage, "no pending pairing requests; run rein account join on the new device first")
	case len(pending) == 1:
		req = &pending[0]
	default:
		msg := "several pairing requests are pending; rerun with --request <id>:"
		for _, p := range pending {
			msg += fmt.Sprintf("\n  %s  %s (%s), opened %s", p.ID, p.Device.Name, p.Device.Platform, p.CreatedAt)
		}
		return NewExitError(ExitUsage, msg)
	}
	errOut := cmd.ErrOrStderr()
	PrintHuman(errOut, "Device %q (%s) asked to join this account (request opened %s).", req.Device.Name, req.Device.Platform, req.CreatedAt)
	PrintHuman(errOut, "Approve only if that machine is yours and is showing a pairing code right now.")
	typed, err := seams.readPairingCode(cmd, "Pairing code shown on the new device: ")
	if err != nil {
		return NewExitError(ExitUsage, err.Error())
	}
	generation, err := approvePairingRequest(ctx, cmd, client, tok.Token, *req, string(typed))
	crypto.Zero(typed)
	if err != nil {
		return err
	}
	PrintHuman(cmd.OutOrStdout(), "approved device %q (%s); it now reads everything under key generation %d", req.Device.Name, req.Device.Platform, generation)
	return nil
}

// approvePairingRequest is the whole approving-device flow given a typed
// code: verify the code against the request, append the new device's wrap
// to the keyring (compare-and-swap), and relay the sealed root key. It is
// deliberately free of prompting so the daemon (ticket #10, prompt hook
// from #9) can call it with a code collected elsewhere.
func approvePairingRequest(ctx context.Context, cmd *cobra.Command, client *hop.Client, token string, req hop.PairingRequest, typedCode string) (generation int, err error) {
	salt, err := base64.StdEncoding.DecodeString(req.Salt)
	if err != nil {
		return 0, NewExitError(ExitSafety, "the pairing request carries a malformed salt; nothing was approved")
	}
	pairing, err := keyring.PairingFromCode(typedCode, salt)
	if err != nil {
		return 0, NewExitError(ExitUsage, err.Error())
	}
	defer pairing.Zero()
	if !pairing.VerifyBinding(req.PublicKey, req.Binding) {
		return 0, NewExitError(ExitSafety, "the code does not match this pairing request (wrong code, or the request was altered in transit); nothing was approved")
	}
	recipient, err := age.ParseX25519Recipient(req.PublicKey)
	if err != nil {
		return 0, NewExitError(ExitSafety, "the pairing request carries a malformed device key; nothing was approved")
	}
	// The request was listed while pending, but the hidden prompt can sit
	// open for longer than the request lives. Refuse before any write
	// rather than append a wrap the control plane will then refuse to
	// relay; the control plane's own clock is re-checked at relay time.
	if err := pairingStillOpen(req, time.Now()); err != nil {
		return 0, err
	}

	home, cfg, err := loadAccountHome()
	if err != nil {
		return 0, err
	}
	if cfg.Encryption.Type != schema.EncryptionRootKey {
		return 0, NewExitError(ExitConfig, "this device is not enrolled in the hosted key model; only an enrolled device can approve another")
	}
	store, prefix, err := backendFromConfig(cmd, cfg, home)
	if err != nil {
		return 0, err
	}
	deviceKey, err := loadEnrolledDeviceKey(accountSeamsFrom(cmd), cfg, home)
	if err != nil {
		return 0, err
	}

	// Compare-and-swap: the root keys are unwrapped from the keyring the
	// closure is handed, so a generation rollover landing in between is
	// seen on the retry (the new device is enrolled into the generation
	// that is current then, and into every earlier one this device can
	// read, so it reads the whole locker). A device this machine can no
	// longer open (it was revoked meanwhile) approves nothing.
	anchor, err := loadKeyringAnchor(cmd, home, cfg)
	if err != nil {
		return 0, err
	}
	var appended []keyring.AppendedWrap
	var rootKey []byte
	updated, err := keyring.Update(ctx, store, keyring.ObjectKey(prefix), func(k *keyring.Keyring) error {
		appended = nil
		crypto.Zero(rootKey)
		keys, err := trustKeyring(anchor, k, func() (map[int][]byte, error) {
			return k.UnwrapGenerations(cfg.DeviceID, deviceKey)
		})
		if err != nil {
			return err
		}
		defer keyring.ZeroGenerations(keys)
		rootKey = append([]byte(nil), keys[k.CurrentGeneration]...)
		if listed := k.DevicePublicKey(req.Device.ID); listed != "" {
			if listed == req.PublicKey {
				// A previous approval wrote the wrap but the relay call
				// failed; sealing again for the same key is safe.
				return nil
			}
			return fmt.Errorf("the keyring already lists device %s with a different key; revoke it before approving a new enrolment", req.Device.ID)
		}
		appended, err = k.EnrolAll(keys, req.Device.ID, recipient, time.Now().UTC())
		return err
	})
	defer crypto.Zero(rootKey)
	if exit := exitForKeyringRefusal(err); exit != nil {
		return 0, exit
	}
	if errors.Is(err, keyring.ErrDeviceNotEnrolled) {
		return 0, NewExitError(ExitAuthStorage, fmt.Sprintf("this device cannot open the current key generation (%v); only an enrolled device can approve another", err))
	}
	if err != nil {
		return 0, NewExitError(ExitAuthStorage, err.Error())
	}
	if err := observeKeyring(home, anchor.floor, updated); err != nil {
		return 0, err
	}
	payload, err := pairing.SealRootKey(rootKey, req.ID, recipient, updated.CurrentGeneration)
	if err != nil {
		return 0, NewExitError(ExitRuntime, err.Error())
	}
	if err := client.ApprovePairing(ctx, token, req.ID, payload, updated.CurrentGeneration); err != nil {
		// The control plane refused a request that was pending when the
		// code was entered (it expired, or another device decided it). No
		// wrap this call appended may outlive the request — in any
		// generation, or the refused device could still unwrap
		// pre-revocation history: the joining device would otherwise find
		// itself enrolled on its next join with no approval event behind
		// it. Exactly the wraps this call wrote are removed, matched by
		// generation and ciphertext, so wraps that were already there when
		// it ran are left alone.
		if len(appended) > 0 && (errors.Is(err, hop.ErrPairingExpired) || errors.Is(err, hop.ErrPairingDecided)) {
			if rbErr := rollBackPairingWrap(ctx, store, prefix, req, appended); rbErr != nil {
				return 0, NewExitError(ExitAuthStorage, fmt.Sprintf("%v; the wraps appended for device %s could not be removed (%v): run rein devices approve again once the device retries, or revoke it", err, req.Device.ID, rbErr))
			}
			return 0, NewExitError(ExitAuthStorage, fmt.Sprintf("%v; every wrap appended for device %s was removed again, nothing was approved", err, req.Device.ID))
		}
		return 0, hopExitError(err)
	}
	return updated.CurrentGeneration, nil
}

// pairingStillOpen refuses a request whose expiry has passed on this
// device's clock. A missing or malformed expiry is not trusted either: the
// relay is the only path and it always stamps one.
func pairingStillOpen(req hop.PairingRequest, now time.Time) error {
	expires, err := time.Parse(time.RFC3339Nano, req.ExpiresAt)
	if err != nil {
		return NewExitError(ExitSafety, "the pairing request carries a malformed expiry; nothing was approved")
	}
	if !now.Before(expires) {
		return NewExitError(ExitUsage, fmt.Sprintf("pairing request %s expired at %s; nothing was approved. Run rein account join again on the new device and enter the fresh code", req.ID, req.ExpiresAt))
	}
	return nil
}

// rollBackPairingWrap removes the wraps approvePairingRequest appended for
// req when the relay then refused it, under the same compare-and-swap as
// the enrolment. The approval enrolled the device into every generation
// this device could read (EnrolAll), so the rollback covers all of them —
// taking back only the current generation's wrap would leave the refused
// device able to unwrap pre-revocation history. It takes back only the
// wraps this approval actually wrote: appended names them by generation and
// by ciphertext, so a generation that already held a wrap for this device
// when the approval ran keeps it. Removing nothing is not an error: a
// concurrent revocation or rollover may already have taken them.
func rollBackPairingWrap(ctx context.Context, store backend.Backend, prefix string, req hop.PairingRequest, appended []keyring.AppendedWrap) error {
	_, err := keyring.Update(ctx, store, keyring.ObjectKey(prefix), func(k *keyring.Keyring) error {
		k.UnenrolAppended(req.Device.ID, appended)
		return nil
	})
	return err
}

func newDevicesRevokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <device-id|name>",
		Short: "Revoke a device: start a new key generation without it",
		Long: `Revoke a device that is lost, retired, or no longer trusted.

Run from any other enrolled device. It reads the recovery code (hidden
prompt, or REINSTATE_RECOVERY_CODE_FD for automation), starts a new key
generation in the keyring (a fresh root key wrapped for every remaining
device and under the recovery code, and signed by the account key that code
derives; earlier generations stay so everything already in the locker
remains readable), tells the control plane, which refuses the revoked
device's token from then on, and raises the account's key generation floor
there so the other devices refuse the earlier keyring even before they have
read the new one.

The revoked device keeps whatever it already pulled and cannot read what a
device pushes once that device has the new key generation. It cannot mint
new locker credentials either — but a credential it minted before the
revocation keeps working against the bucket until it expires, up to an
hour, so within that window it can still write bytes it can no longer read,
and it can put the old keyring back. A device that has already read the new
generation refuses that; a device that has not is covered by the control
plane's floor, and a control plane that does not carry one leaves that
device with nothing to check against until it next reads the keyring — the
command says so on stderr when it meets one. Revoking the same device twice
is harmless.

A device cannot revoke itself; use another enrolled device.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDevicesRevoke(cmd, args[0])
		},
	}
	return cmd
}

func runDevicesRevoke(cmd *cobra.Command, target string) error {
	seams := accountSeamsFrom(cmd)
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	devices, err := client.Devices(ctx, tok.Token)
	if err != nil {
		return hopExitError(err)
	}
	victim, err := resolveDevice(devices, target)
	if err != nil {
		return err
	}
	home, cfg, err := loadAccountHome()
	if err != nil {
		return err
	}
	if victim.ID == cfg.DeviceID || victim.ID == tok.DeviceID {
		return NewExitError(ExitUsage, fmt.Sprintf("device %s (%s) is this device; revoke it from another enrolled device instead", victim.ID, victim.Name))
	}
	if cfg.Encryption.Type != schema.EncryptionRootKey {
		return NewExitError(ExitConfig, "this device is not enrolled in the hosted key model; only an enrolled device can revoke another")
	}
	store, prefix, err := backendFromConfig(cmd, cfg, home)
	if err != nil {
		return err
	}
	deviceKey, err := loadEnrolledDeviceKey(seams, cfg, home)
	if err != nil {
		return err
	}
	errOut := cmd.ErrOrStderr()
	PrintHuman(errOut, "Revoking %q (%s, %s). The account's key generation moves on without it;", victim.Name, victim.Platform, victim.ID)
	PrintHuman(errOut, "the recovery code is needed so the new generation stays recoverable.")
	typed, err := seams.readRecoveryCode(cmd, "Recovery code: ")
	if err != nil {
		return NewExitError(ExitUsage, err.Error())
	}
	recoveryCode, err := keyring.NormalizeRecoveryCode(string(typed))
	crypto.Zero(typed)
	if err != nil {
		return NewExitError(ExitUsage, err.Error())
	}

	// Compare-and-swap: the closure re-unwraps the current root key from
	// whatever keyring it is handed, so an approval or another revocation
	// landing in between is folded in (the new device gets a wrap in the
	// new generation; a device already revoked is reported, not revoked
	// into a third generation).
	now := time.Now().UTC()
	anchor, err := loadKeyringAnchor(cmd, home, cfg)
	if err != nil {
		return err
	}
	var generation int
	updated, err := keyring.Update(ctx, store, keyring.ObjectKey(prefix), func(k *keyring.Keyring) error {
		keys, err := trustKeyring(anchor, k, func() (map[int][]byte, error) {
			return k.UnwrapGenerations(cfg.DeviceID, deviceKey)
		})
		if err != nil {
			return err
		}
		defer keyring.ZeroGenerations(keys)
		next, err := k.Rollover(keys[k.CurrentGeneration], recoveryCode, []string{victim.ID}, cfg.DeviceID, now)
		if err != nil {
			return err
		}
		crypto.Zero(next)
		generation = k.CurrentGeneration
		return nil
	})
	already := false
	if exit := exitForKeyringRefusal(err); exit != nil {
		return exit
	}
	if errors.Is(err, keyring.ErrRecoveryMismatch) || errors.Is(err, keyring.ErrRecoveryWrapMalformed) {
		// Same two answers as `rein account recover`: a code to re-type, or
		// a keyring to restore. Nothing was revoked either way.
		return exitForRecoveryWrap(err, "revoked")
	}
	switch {
	case errors.Is(err, keyring.ErrSelfRevoke):
		return NewExitError(ExitUsage, "a device cannot revoke itself; revoke it from another enrolled device")
	case errors.Is(err, keyring.ErrDeviceNotEnrolled) && !errors.Is(err, keyring.ErrAlreadyRevoked):
		return NewExitError(ExitAuthStorage, fmt.Sprintf("this device cannot open the current key generation (%v); only an enrolled device can revoke another", err))
	case errors.Is(err, keyring.ErrAlreadyRevoked):
		// The victim already has no wrap in the current generation
		// (revoked earlier, possibly by a device racing this one, or it
		// never finished enrolling). The control plane is still told.
		already = true
		ring, _, loadErr := keyring.Load(ctx, store, keyring.ObjectKey(prefix))
		if exit := exitForKeyringRefusal(loadErr); exit != nil {
			return exit
		}
		if loadErr != nil {
			return NewExitError(ExitAuthStorage, loadErr.Error())
		}
		updated, generation = ring, ring.CurrentGeneration
	case err != nil:
		return NewExitError(ExitAuthStorage, err.Error())
	}
	// Pin the generation this device just created (or observed) before the
	// control plane is told: a rolled-back keyring is then refused here
	// even if the revocation call below fails and is retried.
	if err := observeKeyring(home, anchor.floor, updated); err != nil {
		return err
	}

	revocation, err := client.RevokeDevice(ctx, tok.Token, victim.ID)
	if err != nil {
		if already {
			return hopExitError(err)
		}
		return NewExitError(ExitAuthStorage, fmt.Sprintf("the keyring moved to key generation %d without %s, but the control plane could not be told (%v); its token still works until it is. Run rein devices revoke %s again", generation, victim.ID, err, victim.ID))
	}
	// Raise the account-wide floor last, once the keyring holds the new
	// generation and the token is refused. This is what reaches the devices
	// that have not read the keyring yet: without it, a device that has run
	// nothing since the previous generation would accept the pre-rollover
	// keyring the revoked device can still put back. Revoking is idempotent
	// on both halves, so a failure here is reported and re-run rather than
	// left to a later command to notice.
	carried, err := raiseKeyGenerationFloor(cmd, home, cfg, generation)
	if errors.Is(err, hop.ErrKeyGenerationRange) {
		// Retrying will not help: the number is past what the control
		// plane will store, and the keyring has already moved. Say so
		// rather than sending the operator round the loop again.
		return NewExitError(ExitAuthStorage, fmt.Sprintf("device %s is revoked and the keyring is at key generation %d, but the control plane will not hold a floor that high (%v). The revocation stands; the account-wide floor is left where it was, so a device that has not yet read key generation %d has only its own record to go on. Move the account to a fresh locker rather than revoking again", victim.ID, generation, err, generation))
	}
	if err != nil {
		return NewExitError(ExitAuthStorage, fmt.Sprintf("device %s is revoked and the keyring is at key generation %d, but the account's key generation floor could not be raised (%v); a device that has not yet read key generation %d would still accept the earlier keyring. Run rein devices revoke %s again", victim.ID, generation, err, generation, victim.ID))
	}
	out := cmd.OutOrStdout()
	switch {
	case already && !revocation.Revoked:
		PrintHuman(out, "device %q (%s) was already revoked; key generation %d, %d enrolled device(s)", victim.Name, victim.ID, generation, updated.DeviceCount())
	case already:
		PrintHuman(out, "device %q (%s) had no wrap in key generation %d; its token is now refused by the control plane", victim.Name, victim.ID, generation)
	default:
		PrintHuman(out, "revoked device %q (%s); key generation %d started with %d enrolled device(s), and the control plane refuses its token", victim.Name, victim.ID, generation, updated.DeviceCount())
		PrintHuman(out, "earlier key generations stay readable on every remaining device; what any device pushes once it has key generation %d is not readable by the revoked device", generation)
		PrintHuman(out, "it can mint no new locker credentials, but one it minted before now keeps working against the bucket until it expires (up to an hour), so it can still write objects during that window — objects it cannot read")
	}
	if carried {
		PrintHuman(out, "the control plane now reports key generation %d for this account, so a device that has not read the keyring since the rollover refuses the earlier copy too", generation)
	} else {
		PrintHuman(cmd.ErrOrStderr(), "note: this control plane does not carry an account key generation floor, so a device that has not read key generation %d yet has nothing to check a restored earlier keyring against — run rein account status on your other devices to bring each one up to date", generation)
	}
	return nil
}

// resolveDevice picks one device by id or, when unique, by name.
func resolveDevice(devices []hop.Device, target string) (hop.Device, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return hop.Device{}, NewExitError(ExitUsage, "a device id or name is required; rein devices lists them")
	}
	for _, d := range devices {
		if d.ID == target {
			return d, nil
		}
	}
	var byName []hop.Device
	for _, d := range devices {
		if strings.EqualFold(d.Name, target) {
			byName = append(byName, d)
		}
	}
	switch len(byName) {
	case 1:
		return byName[0], nil
	case 0:
		return hop.Device{}, NewExitError(ExitUsage, fmt.Sprintf("no device %q on this account; rein devices lists them", target))
	}
	msg := fmt.Sprintf("%d devices are named %q; rerun with the id:", len(byName), target)
	for _, d := range byName {
		state := "enrolled " + d.CreatedAt
		if d.Revoked() {
			state = "revoked " + d.RevokedAt
		}
		msg += fmt.Sprintf("\n  %s  %s (%s), %s", d.ID, d.Name, d.Platform, state)
	}
	return hop.Device{}, NewExitError(ExitUsage, msg)
}

// loadEnrolledDeviceKey returns this device's key for the hosted key model,
// refusing when the device was never enrolled here.
func loadEnrolledDeviceKey(seams accountSeams, cfg *schema.Config, home string) (*age.X25519Identity, error) {
	notEnrolled := "this device is not enrolled in the hosted key model; run rein account recover with the recovery code, or rein account init on a first device"
	if _, err := config.LoadAccount(home); err != nil {
		if os.IsNotExist(err) {
			return nil, NewExitError(ExitConfig, notEnrolled)
		}
		return nil, NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	deviceKey, err := loadDeviceKey(seams.secretStore(), deviceSecretRef(cfg.ProfileID, cfg.DeviceID))
	if err != nil {
		return nil, err
	}
	if deviceKey == nil {
		return nil, NewExitError(ExitConfig, "device key missing from the OS keyring; "+notEnrolled)
	}
	return deviceKey, nil
}
