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
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
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
		return NewExitError(ExitSafety, "this device is already enrolled; rein account status shows the keyring state")
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
	if deviceKey != nil && ring.HasDevice(cfg.DeviceID) {
		if _, _, err := ring.UnwrapForDevice(cfg.DeviceID, deviceKey); err != nil {
			return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s but not the key held in the OS keyring (%v); nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device", cfg.DeviceID, err))
		}
		if err := saveAccountEnrolmentConfirmed(home, cfg, now, ring.CurrentGeneration, "join", false); err != nil {
			return err
		}
		PrintHuman(cmd.OutOrStdout(), "device already enrolled; local enrolment record restored, keyring and device key unchanged")
		return nil
	}
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
	if err := verifyJoinedKeyring(ctx, store, keyringKey, cfg.DeviceID, deviceKey, rootKey, payload.KeyGeneration); err != nil {
		return err
	}
	if err := saveAccountEnrolmentConfirmed(home, cfg, now, payload.KeyGeneration, "join", false); err != nil {
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
func verifyJoinedKeyring(ctx context.Context, store backend.Backend, key, deviceID string, deviceKey *age.X25519Identity, rootKey []byte, generation int) error {
	ring, _, err := keyring.Load(ctx, store, key)
	if err != nil {
		return NewExitError(ExitAuthStorage, err.Error())
	}
	if ring.CurrentGeneration != generation {
		return NewExitError(ExitSafety, fmt.Sprintf("the keyring is at generation %d but the approval named %d; nothing was written. Run rein account join again", ring.CurrentGeneration, generation))
	}
	fromRing, _, err := ring.UnwrapForDevice(deviceID, deviceKey)
	if err != nil {
		return NewExitError(ExitSafety, fmt.Sprintf("the keyring does not hold a working wrap for this device (%v); nothing was written. Run rein account join again", err))
	}
	defer crypto.Zero(fromRing)
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		return NewExitError(ExitSafety, err.Error())
	}
	if subtle.ConstantTimeCompare(fromRing, rootKey) != 1 || identity.Recipient().String() != ring.CurrentRecipient() {
		return NewExitError(ExitSafety, "the root key received through the pairing does not match the keyring's current generation; nothing was written. The storage or relay may have been tampered with")
	}
	return nil
}

// newDevicesCmd lists enrolled devices and approves pairing requests.
func newDevicesCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "devices",
		Short: "List this account's devices and approve new ones",
		Args:  cobra.NoArgs,
		RunE:  runDevicesList,
	}
	root.AddCommand(newDevicesApproveCmd())
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
	if home, cfg, err := loadAccountHome(); err == nil {
		if store, prefix, err := backendFromConfig(cmd, cfg, home); err == nil {
			if ring, _, err := keyring.Load(ctx, store, keyring.ObjectKey(prefix)); err == nil {
				keyringSeen = true
				for _, d := range devices {
					inKeyring[d.ID] = ring.HasDevice(d.ID)
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
		return WriteJSON(cmd.OutOrStdout(), map[string]any{"devices": rows, "pending_pairings": pending})
	}
	out := cmd.OutOrStdout()
	for _, d := range devices {
		line := fmt.Sprintf("%s  %s (%s), enrolled %s, last seen %s", d.ID, d.Name, d.Platform, d.CreatedAt, d.LastSeenAt)
		if keyringSeen {
			if inKeyring[d.ID] {
				line += ", holds a root-key wrap"
			} else {
				line += ", no root-key wrap yet"
			}
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
	rootKey, err := unwrapDeviceRootKey(ctx, cmd, cfg, home, store, prefix)
	if err != nil {
		return 0, err
	}
	defer crypto.Zero(rootKey)

	// Compare-and-swap: on interference the keyring is reloaded and the
	// enrolment re-checked; a generation rollover under our feet makes
	// Enrol refuse (this root key no longer belongs to the current
	// generation), so a just-revoked state can never be extended.
	appended := false
	updated, err := keyring.Update(ctx, store, keyring.ObjectKey(prefix), func(k *keyring.Keyring) error {
		appended = false
		if listed := k.DevicePublicKey(req.Device.ID); listed != "" {
			if listed == req.PublicKey {
				// A previous approval wrote the wrap but the relay call
				// failed; sealing again for the same key is safe.
				return nil
			}
			return fmt.Errorf("the keyring already lists device %s with a different key; revoke it before approving a new enrolment", req.Device.ID)
		}
		if err := k.Enrol(rootKey, req.Device.ID, recipient, time.Now().UTC()); err != nil {
			return err
		}
		appended = true
		return nil
	})
	if err != nil {
		return 0, NewExitError(ExitAuthStorage, err.Error())
	}
	payload, err := pairing.SealRootKey(rootKey, req.ID, recipient, updated.CurrentGeneration)
	if err != nil {
		return 0, NewExitError(ExitRuntime, err.Error())
	}
	if err := client.ApprovePairing(ctx, token, req.ID, payload, updated.CurrentGeneration); err != nil {
		// The control plane refused a request that was pending when the
		// code was entered (it expired, or another device decided it). A
		// wrap this call appended must not outlive the request: the
		// joining device would otherwise find itself enrolled on its next
		// join with no approval event behind it. Only the wrap made for
		// this request's key is removed; a competing approval's wrap for
		// the same device id stays.
		if appended && (errors.Is(err, hop.ErrPairingExpired) || errors.Is(err, hop.ErrPairingDecided)) {
			if rbErr := rollBackPairingWrap(ctx, store, prefix, req); rbErr != nil {
				return 0, NewExitError(ExitAuthStorage, fmt.Sprintf("%v; the wrap appended for device %s could not be removed (%v): run rein devices approve again once the device retries, or revoke it", err, req.Device.ID, rbErr))
			}
			return 0, NewExitError(ExitAuthStorage, fmt.Sprintf("%v; the wrap appended for device %s was removed again, nothing was approved", err, req.Device.ID))
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

// rollBackPairingWrap removes the wrap approvePairingRequest appended for
// req when the relay then refused it, under the same compare-and-swap as
// the enrolment. Removing nothing is not an error: a concurrent revocation
// or rollover may already have taken it.
func rollBackPairingWrap(ctx context.Context, store backend.Backend, prefix string, req hop.PairingRequest) error {
	_, err := keyring.Update(ctx, store, keyring.ObjectKey(prefix), func(k *keyring.Keyring) error {
		k.Unenrol(req.Device.ID, req.PublicKey)
		return nil
	})
	return err
}

// unwrapDeviceRootKey resolves this device's key and unwraps the current
// root key from the keyring, refusing when the device is not enrolled.
func unwrapDeviceRootKey(ctx context.Context, cmd *cobra.Command, cfg *schema.Config, home string, store backend.Backend, prefix string) ([]byte, error) {
	seams := accountSeamsFrom(cmd)
	notEnrolled := "this device is not enrolled in the hosted key model; run rein account recover with the recovery code, or rein account init on a first device"
	if _, err := config.LoadAccount(home); err != nil {
		if os.IsNotExist(err) {
			return nil, NewExitError(ExitConfig, notEnrolled)
		}
		return nil, NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	secret, err := seams.secretStore().GetSecret(deviceSecretRef(cfg.ProfileID, cfg.DeviceID))
	if errors.Is(err, credentials.ErrSecretNotFound) {
		return nil, NewExitError(ExitConfig, "device key missing from the OS keyring; "+notEnrolled)
	}
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	deviceKey, err := age.ParseX25519Identity(strings.TrimSpace(string(secret)))
	crypto.Zero(secret)
	if err != nil {
		return nil, NewExitError(ExitConfig, "device key in the OS keyring is malformed; "+notEnrolled)
	}
	ring, _, err := keyring.Load(ctx, store, keyring.ObjectKey(prefix))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, NewExitError(ExitAuthStorage, "keyring missing at the configured storage; the profile is configured for the root-key model but no keyring exists")
	}
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	current, earlier, err := ring.UnwrapForDevice(cfg.DeviceID, deviceKey)
	if errors.Is(err, keyring.ErrDeviceNotEnrolled) {
		return nil, NewExitError(ExitAuthStorage, fmt.Sprintf("this device is not enrolled in key generation %d; %s", ring.CurrentGeneration, notEnrolled))
	}
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	for _, key := range earlier {
		crypto.Zero(key)
	}
	return current, nil
}
