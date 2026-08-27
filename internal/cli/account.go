package cli

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// accountSeams are the deterministic-test overrides for the hosted key model.
// Production leaves both nil: the device key lives in the OS keyring and the
// recovery code is read from a hidden terminal prompt or the documented
// descriptor.
type accountSeams struct {
	secrets        credentials.SecretStore
	recoveryPrompt func(prompt string) ([]byte, error)
	pairingPrompt  func(prompt string) ([]byte, error)
}

type accountSeamsContextKey struct{}

func accountSeamsFrom(cmd *cobra.Command) accountSeams {
	if ctx := cmd.Context(); ctx != nil {
		if seams, ok := ctx.Value(accountSeamsContextKey{}).(accountSeams); ok {
			return seams
		}
	}
	return accountSeams{}
}

func (s accountSeams) secretStore() credentials.SecretStore {
	if s.secrets != nil {
		return s.secrets
	}
	return credentials.NewKeyringStore()
}

// readRecoveryCode reads a recovery code without echo. Automation sets
// REINSTATE_RECOVERY_CODE_FD; the code is never a flag or a plain variable.
func (s accountSeams) readRecoveryCode(cmd *cobra.Command, prompt string) ([]byte, error) {
	if s.recoveryPrompt != nil {
		return s.recoveryPrompt(prompt)
	}
	if secret, configured, err := crypto.ReadSecretFD(crypto.RecoveryCodeFDEnv); configured {
		return secret, err
	}
	return crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), prompt)
}

// deviceSecretRef names this device's key in the secret store.
func deviceSecretRef(profileID, deviceID string) string {
	return fmt.Sprintf("reinstate/%s/device/%s", profileID, deviceID)
}

// unrecoverableNotice is the plain statement every enrolment prints. It is
// the whole recovery policy: no escrow, no operator access, local copies
// untouched.
const unrecoverableNotice = `If you lose every enrolled device and this recovery code, nobody can
recover the locker: not you, and not the operator, who only ever holds
ciphertext. Local session copies on each machine are unaffected.`

// alreadyEnrolledHere is what init, join, and recover say when this home
// already carries an enrolment record. It names the way out, because there
// is one and it is not obvious: a machine that was revoked still has the
// record of the enrolment it lost, and nothing else removes it.
// rein init --force copies the record into a backup set and takes it off the
// home, after which join or recover can enrol this machine again.
const alreadyEnrolledHere = "this device is already enrolled; rein account status shows the keyring state. " +
	"If this device was revoked and you are enrolling it again, run rein init --hop --force (or rein init --force for BYO storage) first: " +
	"it backs up this home's config, state, and enrolment record, then removes the enrolment record so this command can run"

func newAccountCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "account",
		Short: "Hosted-tier key model: root key, keyring, recovery code",
		Long: `Manage this device's enrolment in the hosted key model.

The first device generates the root key and the recovery code. Other devices
join by device approval (rein account join, confirmed on an enrolled device
with rein devices approve) or, when no other device is available, from the
recovery code (rein account recover). The root key and recovery code never leave a device; the
keyring in storage holds only wrapped copies.`,
	}
	root.AddCommand(newAccountInitCmd(), newAccountJoinCmd(), newAccountRecoverCmd(), newAccountStatusCmd())
	return root
}

func newAccountInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate the root key and recovery code on this first device",
		Long: `Generate the account's root key on this device, write the keyring to the
configured storage, and show the recovery code exactly once.

The recovery code is shown once and must be re-entered before anything is
written. It is never stored on disk, in config, or in logs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			seams := accountSeamsFrom(cmd)
			home, cfg, err := loadAccountHome()
			if err != nil {
				return err
			}
			if _, err := config.LoadAccount(home); err == nil {
				return NewExitError(ExitSafety, alreadyEnrolledHere)
			} else if !os.IsNotExist(err) {
				return NewExitError(ExitConfig, "read account state: "+err.Error())
			}
			ctx := context.Background()
			store, prefix, err := backendFromConfig(cmd, cfg, home)
			if err != nil {
				return err
			}
			keyringKey := keyring.ObjectKey(prefix)
			if _, _, err := keyring.Load(ctx, store, keyringKey); err == nil {
				return NewExitError(ExitSafety, "a keyring already exists for this profile; enrol this device with rein account recover instead")
			} else if !errors.Is(err, keyring.ErrNotFound) {
				if exit := exitForKeyringRefusal(err); exit != nil {
					return exit
				}
				return NewExitError(ExitAuthStorage, err.Error())
			}

			rootKey, err := crypto.NewRootKey()
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			defer crypto.Zero(rootKey)
			recoveryCode, err := keyring.GenerateRecoveryCode()
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			deviceKey, err := age.GenerateX25519Identity()
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}

			// Shown once, on the prompt stream, never on stdout where it could
			// be captured by a redirect or --json consumer.
			out := cmd.ErrOrStderr()
			PrintHuman(out, "")
			PrintHuman(out, "Your recovery code (shown once, never stored anywhere):")
			PrintHuman(out, "")
			PrintHuman(out, "    %s", recoveryCode)
			PrintHuman(out, "")
			PrintHuman(out, "Write it down and keep it somewhere safe. It is the only copy of the")
			PrintHuman(out, "root key outside your enrolled devices.")
			PrintHuman(out, "%s", unrecoverableNotice)
			PrintHuman(out, "")
			typed, err := seams.readRecoveryCode(cmd, "Re-enter the recovery code to confirm you saved it: ")
			if err != nil {
				return NewExitError(ExitUsage, err.Error())
			}
			confirmed, err := keyring.NormalizeRecoveryCode(string(typed))
			crypto.Zero(typed)
			if err != nil || subtle.ConstantTimeCompare([]byte(confirmed), []byte(recoveryCode)) != 1 {
				return NewExitError(ExitSafety, "recovery code confirmation did not match; nothing was written. Run rein account init again and re-enter the new code exactly")
			}

			now := time.Now().UTC()
			ring, err := keyring.New(cfg.ProfileID, rootKey, recoveryCode, cfg.DeviceID, deviceKey, now)
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			secrets := seams.secretStore()
			secretRef := deviceSecretRef(cfg.ProfileID, cfg.DeviceID)
			if err := refuseExistingDeviceSecret(secrets, secretRef); err != nil {
				return err
			}
			if err := secrets.SetSecret(secretRef, []byte(deviceKey.String())); err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			if err := keyring.Create(ctx, store, keyringKey, ring); err != nil {
				// Only the key this command just created is rolled back.
				_ = secrets.DeleteSecret(secretRef)
				if exit := exitForKeyringRefusal(err); exit != nil {
					return exit
				}
				return NewExitError(ExitAuthStorage, err.Error())
			}
			if err := saveAccountEnrolment(home, cfg, now, ring, "init"); err != nil {
				return err
			}
			PrintHuman(cmd.OutOrStdout(), "account initialized: root key generated on this device, keyring written to storage")
			PrintHuman(cmd.OutOrStdout(), "profile_id=%s device_id=%s key_generation=%d devices=1", cfg.ProfileID, cfg.DeviceID, ring.CurrentGeneration)
			PrintHuman(cmd.OutOrStdout(), "recovery code confirmed on this device; encryption.type=%s", schema.EncryptionRootKey)
			return nil
		},
	}
	return cmd
}

func newAccountRecoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "Enrol this device from the recovery code",
		Long: `Enrol a fresh device when no other device is available to approve it.

Reads the recovery code (hidden prompt, or REINSTATE_RECOVERY_CODE_FD for
automation), unwraps the root key from the keyring, generates this device's
key, and appends a wrap for it. From then on this device reads everything
written under the current key generation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			seams := accountSeamsFrom(cmd)
			home, cfg, err := loadAccountHome()
			if err != nil {
				return err
			}
			if _, err := config.LoadAccount(home); err == nil {
				return NewExitError(ExitSafety, alreadyEnrolledHere)
			} else if !os.IsNotExist(err) {
				return NewExitError(ExitConfig, "read account state: "+err.Error())
			}
			if cfg.Storage.Type == schema.StorageHop {
				// The keyring's device id and the control plane's must
				// agree, or a revoked id could be enrolled again under a
				// token that belongs to a different device record.
				tok, _, err := hostedSession(cmd)
				if err != nil {
					return err
				}
				if tok.DeviceID == "" {
					return NewExitError(ExitConfig, "the stored device token names no device; run rein login again before enrolling")
				}
				if tok.DeviceID != cfg.DeviceID {
					return NewExitError(ExitConfig, fmt.Sprintf("this home's device_id (%s) is not the signed-in device (%s); run rein init --hop --force so the keyring and the control plane agree on this device's identity", cfg.DeviceID, tok.DeviceID))
				}
			}
			ctx := context.Background()
			store, prefix, err := backendFromConfig(cmd, cfg, home)
			if err != nil {
				return err
			}
			keyringKey := keyring.ObjectKey(prefix)
			ring, _, err := keyring.Load(ctx, store, keyringKey)
			if errors.Is(err, keyring.ErrNotFound) {
				return NewExitError(ExitAuthStorage, "no keyring found at the configured storage; run rein account init on the first device, or check the profile and prefix")
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
			typed, err := seams.readRecoveryCode(cmd, "Recovery code: ")
			if err != nil {
				return NewExitError(ExitUsage, err.Error())
			}
			recoveryCode, err := keyring.NormalizeRecoveryCode(string(typed))
			crypto.Zero(typed)
			if err != nil {
				return NewExitError(ExitUsage, err.Error())
			}
			// The recovery code is this device's anchor: it has no local
			// record to check against, because recover refuses to run
			// where one exists. The code derives the account signing key,
			// so the public half can be pinned here from the code alone
			// and the keyring required to be signed under it — the same
			// anchor an enrolled device carries in account.json, arrived
			// at without ever having seen this account's keyring before.
			//
			// The recovery wrap is checked first so that a mistyped code
			// is reported as a mistyped code rather than as a tampered
			// keyring; past that point the code is known to be this
			// account's, and a key mismatch can only be tampering.
			recoveryKeys, err := ring.UnwrapGenerationsWithRecoveryCode(recoveryCode)
			if exit := exitForRecoveryWrap(err, "written"); exit != nil {
				return exit
			}
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			keyring.ZeroGenerations(recoveryKeys)
			accountKey, err := keyring.DeriveAccountKey(cfg.ProfileID, recoveryCode)
			if err != nil {
				return NewExitError(ExitUsage, err.Error())
			}
			defer accountKey.Zero()
			// The floor still applies, and matters more here than almost
			// anywhere: a device enrolling from the code has no local
			// record at all, so the control plane's number is the only
			// thing that can tell it the account has moved past the
			// keyring it is being handed.
			floor, err := confirmKeyGenerationFloor(cmd, home, cfg)
			if err != nil {
				return err
			}
			codeAnchor := keyringAnchor{accountKey: accountKey.Public(), floor: floor}
			if err := codeAnchor.check(ring); err != nil {
				if exit := exitForKeyringRefusal(err); exit != nil {
					return exit
				}
				return NewExitError(ExitAuthStorage, err.Error())
			}

			now := time.Now().UTC()
			secrets := seams.secretStore()
			secretRef := deviceSecretRef(cfg.ProfileID, cfg.DeviceID)
			existing, err := loadDeviceKey(secrets, secretRef)
			if err != nil {
				return err
			}
			switch ring.DeviceMembership(cfg.DeviceID, existing) {
			case keyring.Enrolled:
				// The keyring already holds a wrap for the key this device
				// keeps (for example the local record was lost, or an earlier
				// enrolment crashed before writing it). Re-attach without
				// touching the device key or the keyring.
				current, earlier, err := ring.UnwrapForDevice(cfg.DeviceID, existing)
				if err != nil {
					return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s but its wrap does not open with the key held in the OS keyring (%v); nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device", cfg.DeviceID, err))
				}
				crypto.Zero(current)
				for _, key := range earlier {
					crypto.Zero(key)
				}
				if err := saveAccountEnrolment(home, cfg, now, ring, "recover"); err != nil {
					return err
				}
				if err := rememberKeyGenerationFloor(home, codeAnchor.floor); err != nil {
					return err
				}
				PrintHuman(cmd.OutOrStdout(), "device already enrolled; local enrolment record restored, keyring and device key unchanged")
				PrintHuman(cmd.OutOrStdout(), "profile_id=%s device_id=%s key_generation=%d devices=%d", cfg.ProfileID, cfg.DeviceID, ring.CurrentGeneration, ring.DeviceCount())
				PrintHuman(cmd.ErrOrStderr(), "%s", unrecoverableNotice)
				return nil
			case keyring.KeyMismatch:
				return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s but not the key held in the OS keyring; nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device (rein devices revoke %s)", cfg.DeviceID, cfg.DeviceID))
			case keyring.KeyGone:
				return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s but this machine holds no key for it; nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device (rein devices revoke %s)", cfg.DeviceID, cfg.DeviceID))
			case keyring.NotListed:
				if existing != nil && !ring.RevokedDevice(cfg.DeviceID) {
					return NewExitError(ExitSafety, fmt.Sprintf("the OS keyring already holds a key for device %s that the keyring does not list; nothing was written. Remove that entry (%s) or choose a new device_id in config before enrolling", cfg.DeviceID, secretRef))
				}
			}

			// A device that was revoked and is being enrolled again gets a
			// fresh key: the old one belongs to generations it already read,
			// and the old wraps in those generations are left as they are.
			deviceKey, err := age.GenerateX25519Identity()
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			if err := secrets.SetSecret(secretRef, []byte(deviceKey.String())); err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			// The code opens every generation (each was wrapped under it
			// when it started), so this device is enrolled into all of them
			// and reads the whole locker, not only what is written from now
			// on. Unwrapping happens inside the compare-and-swap so a
			// rollover landing in between is seen, not raced.
			updated, err := keyring.Update(ctx, store, keyringKey, func(k *keyring.Keyring) error {
				// Verified again on every attempt: a rollover — or a
				// forged generation — landing between the load above and
				// this write is seen here, not raced past.
				keys, err := trustKeyring(codeAnchor, k, func() (map[int][]byte, error) {
					return k.UnwrapGenerationsWithRecoveryCode(recoveryCode)
				})
				if err != nil {
					return err
				}
				defer keyring.ZeroGenerations(keys)
				_, err = k.EnrolAll(keys, cfg.DeviceID, deviceKey.Recipient(), now)
				return err
			})
			if err != nil {
				// Only the key this command just created is rolled back
				// (to the previous key when the device was re-enrolling).
				if existing != nil {
					_ = secrets.SetSecret(secretRef, []byte(existing.String()))
				} else {
					_ = secrets.DeleteSecret(secretRef)
				}
				if exit := exitForKeyringRefusal(err); exit != nil {
					return exit
				}
				if exit := exitForRecoveryWrap(err, "written"); exit != nil {
					return exit
				}
				return NewExitError(ExitAuthStorage, err.Error())
			}
			if err := saveAccountEnrolment(home, cfg, now, updated, "recover"); err != nil {
				return err
			}
			// The record exists only now, so the floor confirmed above had
			// nowhere to be written until this point.
			if err := rememberKeyGenerationFloor(home, codeAnchor.floor); err != nil {
				return err
			}
			PrintHuman(cmd.OutOrStdout(), "device enrolled from the recovery code; this device now reads everything written under key generation %d and the %d earlier one(s)", updated.CurrentGeneration, len(updated.GenerationNumbers())-1)
			PrintHuman(cmd.OutOrStdout(), "profile_id=%s device_id=%s key_generation=%d devices=%d", cfg.ProfileID, cfg.DeviceID, updated.CurrentGeneration, updated.DeviceCount())
			PrintHuman(cmd.ErrOrStderr(), "%s", unrecoverableNotice)
			return nil
		},
	}
	return cmd
}

func newAccountStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show keyring state, key generation, and enrolled devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			seams := accountSeamsFrom(cmd)
			home, cfg, err := loadAccountHome()
			if err != nil {
				return err
			}
			type report struct {
				ProfileID             string `json:"profile_id"`
				DeviceID              string `json:"device_id"`
				EncryptionType        string `json:"encryption_type"`
				EnrolledOnThisDevice  bool   `json:"enrolled_on_this_device"`
				EnrolledVia           string `json:"enrolled_via,omitempty"`
				RecoveryCodeConfirmed bool   `json:"recovery_code_confirmed"`
				DeviceKeyPresent      bool   `json:"device_key_present"`
				KeyringPresent        bool   `json:"keyring_present"`
				KeyringRefused        bool   `json:"keyring_refused"`
				KeyGeneration         int    `json:"key_generation,omitempty"`
				EnrolledDevices       int    `json:"enrolled_devices,omitempty"`
				DeviceInKeyring       bool   `json:"device_in_keyring"`
				AccountPath           string `json:"account_path"`
				Error                 string `json:"error,omitempty"`
			}
			r := report{
				ProfileID:      cfg.ProfileID,
				DeviceID:       cfg.DeviceID,
				EncryptionType: cfg.Encryption.Type,
				AccountPath:    filepath.ToSlash(config.AccountPath(home)),
			}
			if account, err := config.LoadAccount(home); err == nil {
				r.EnrolledOnThisDevice = true
				r.EnrolledVia = account.EnrolledVia
				r.RecoveryCodeConfirmed = account.RecoveryCodeConfirmed
				r.KeyGeneration = account.KeyGeneration
			}
			if _, err := seams.secretStore().GetSecret(deviceSecretRef(cfg.ProfileID, cfg.DeviceID)); err == nil {
				r.DeviceKeyPresent = true
			}
			// Status is a diagnostic and reports what it finds rather than
			// failing on it, so an unreadable enrolment record becomes the
			// reported reason the keyring cannot be judged here.
			anchor, anchorErr := loadKeyringAnchor(cmd, home, cfg)
			if store, prefix, err := backendFromConfig(cmd, cfg, home); err != nil {
				r.Error = err.Error()
			} else if ring, _, err := keyring.Load(context.Background(), store, keyring.ObjectKey(prefix)); err == nil {
				r.KeyringPresent = true
				r.KeyGeneration = ring.CurrentGeneration
				r.EnrolledDevices = ring.DeviceCount()
				r.DeviceInKeyring = ring.HasDevice(cfg.DeviceID)
				// Status holds no key of any kind, and needs none: the
				// generation signatures verify against the account key
				// alone, and the anchor reads nothing but public fields.
				// Together they catch a keyring signed by another key,
				// rolled back, or replaced under this device.
				switch {
				case anchorErr != nil:
					r.Error = anchorErr.Error()
				default:
					if err := anchor.check(ring); err != nil {
						r.Error = err.Error()
					}
				}
			} else if !errors.Is(err, keyring.ErrNotFound) {
				r.Error = err.Error()
				// An object that is there but does not verify is refused,
				// not unavailable: saying "unavailable" would read as a
				// storage problem when it is the opposite.
				r.KeyringRefused = exitForKeyringRefusal(err) != nil
			}
			if r.Error != "" && r.KeyringPresent {
				r.KeyringRefused = true
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), r)
			}
			out := cmd.OutOrStdout()
			PrintHuman(out, "profile_id=%s device_id=%s", r.ProfileID, r.DeviceID)
			PrintHuman(out, "encryption: %s", r.EncryptionType)
			if r.EnrolledOnThisDevice {
				PrintHuman(out, "this device: enrolled via %s; recovery code confirmed here: %s; device key in OS keyring: %s",
					r.EnrolledVia, yesNo(r.RecoveryCodeConfirmed), yesNo(r.DeviceKeyPresent))
			} else {
				PrintHuman(out, "this device: not enrolled (run rein account init on a first device, or rein account recover with the recovery code)")
			}
			if r.KeyringPresent {
				PrintHuman(out, "keyring: present; key generation %d; %d enrolled device(s); this device listed: %s",
					r.KeyGeneration, r.EnrolledDevices, yesNo(r.DeviceInKeyring))
			} else if r.Error == "" {
				PrintHuman(out, "keyring: not found at the configured storage")
			}
			if r.Error != "" {
				if r.KeyringRefused {
					PrintHuman(out, "keyring: this device refuses it (%s)", r.Error)
				} else {
					PrintHuman(out, "keyring: unavailable (%s)", r.Error)
				}
			}
			PrintHuman(out, "local record: %s", r.AccountPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func loadAccountHome() (string, *schema.Config, error) {
	home, err := config.Home()
	if err != nil {
		return "", nil, NewExitError(ExitConfig, err.Error())
	}
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return "", nil, configLoadExitError(err)
	}
	return home, cfg, nil
}

// saveAccountEnrolment switches the profile to the root-key model and records
// the local enrolment. Config is saved first so a crash between the two
// leaves a config whose key model matches the keyring already in storage.
func saveAccountEnrolment(home string, cfg *schema.Config, now time.Time, ring *keyring.Keyring, via string) error {
	return saveAccountEnrolmentConfirmed(home, cfg, now, ring, via, true)
}

// saveAccountEnrolmentConfirmed is saveAccountEnrolment with an explicit
// recovery-code flag: a device enrolled by approval (join) never saw the
// recovery code, and its record must say so. Three values are recorded and
// together they are the anchor every later read of the keyring is checked
// against: the account signing key every generation is signed under, the
// keyring's current generation, and that generation's root-key recipient.
func saveAccountEnrolmentConfirmed(home string, cfg *schema.Config, now time.Time, ring *keyring.Keyring, via string, recoveryConfirmed bool) error {
	cfg.Encryption.Type = schema.EncryptionRootKey
	if err := config.SaveConfig(home, cfg); err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	account := &schema.Account{
		SchemaVersion:         schema.AccountSchemaVersion,
		ProfileID:             cfg.ProfileID,
		DeviceID:              cfg.DeviceID,
		KeyGeneration:         ring.CurrentGeneration,
		KeyRecipient:          ring.CurrentRecipient(),
		AccountKey:            ring.AccountPublicKey(),
		RecoveryCodeConfirmed: recoveryConfirmed,
		EnrolledVia:           via,
		EnrolledAt:            now.Format(time.RFC3339),
	}
	if err := config.SaveAccount(home, account); err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	return nil
}

// loadDeviceKey returns the device key stored under ref, nil when none is
// stored, and an exit error when the store fails or the entry is malformed.
func loadDeviceKey(secrets credentials.SecretStore, ref string) (*age.X25519Identity, error) {
	secret, err := secrets.GetSecret(ref)
	if errors.Is(err, credentials.ErrSecretNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	defer crypto.Zero(secret)
	identity, err := age.ParseX25519Identity(strings.TrimSpace(string(secret)))
	if err != nil {
		return nil, NewExitError(ExitSafety, fmt.Sprintf("the OS keyring holds a malformed device key under %s; nothing was written. Remove that entry before enrolling", ref))
	}
	return identity, nil
}

// refuseExistingDeviceSecret keeps enrolment from overwriting a device key
// that already exists: an existing key may be the only way to read wraps in
// a keyring somewhere, so it is never replaced silently.
func refuseExistingDeviceSecret(secrets credentials.SecretStore, ref string) error {
	_, err := secrets.GetSecret(ref)
	if errors.Is(err, credentials.ErrSecretNotFound) {
		return nil
	}
	if err != nil {
		return NewExitError(ExitAuthStorage, err.Error())
	}
	return NewExitError(ExitSafety, fmt.Sprintf("the OS keyring already holds a device key under %s; nothing was written. Remove that entry or choose a new device_id in config before initializing", ref))
}

// rootKeysFromConfig resolves the hosted-tier key provider for push and pull:
// this device's key from the secret store, the keyring from storage, and the
// root key unwrapped in memory for the duration of the command.
func rootKeysFromConfig(ctx context.Context, cmd *cobra.Command, cfg *schema.Config, home string, store backend.Backend, prefix string) (crypto.KeyProvider, error) {
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
	if exit := exitForKeyringRefusal(err); exit != nil {
		return nil, exit
	}
	if err != nil {
		// The storage failure is kept reachable behind the exit error, not
		// flattened into its message: `rein sync verify` has to be able to
		// tell an endpoint that gave no answer from a locker that refused
		// this device, and it cannot ask that of a string.
		return nil, ExitErrorFrom(ExitAuthStorage, err)
	}
	anchor, err := loadKeyringAnchor(cmd, home, cfg)
	if err != nil {
		return nil, err
	}
	generations, err := trustKeyring(anchor, ring, func() (map[int][]byte, error) {
		return ring.UnwrapGenerations(cfg.DeviceID, deviceKey)
	})
	if exit := exitForKeyringRefusal(err); exit != nil {
		return nil, exit
	}
	if errors.Is(err, keyring.ErrDeviceNotEnrolled) {
		return nil, NewExitError(ExitAuthStorage, fmt.Sprintf("this device is not enrolled in key generation %d; %s", ring.CurrentGeneration, notEnrolled))
	}
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	defer keyring.ZeroGenerations(generations)
	// Pinned only now, once the keyring has been authenticated end to end.
	if err := observeKeyring(home, anchor.floor, ring); err != nil {
		return nil, err
	}
	current := generations[ring.CurrentGeneration]
	earlier := make([][]byte, 0, len(generations))
	for n, key := range generations {
		if n != ring.CurrentGeneration {
			earlier = append(earlier, key)
		}
	}
	keys, err := crypto.NewRootKeyProvider(current, earlier...)
	if err != nil {
		return nil, NewExitError(ExitRuntime, err.Error())
	}
	return keys, nil
}

// keyringAnchor is what a command knows about its account's keyring before
// it opens one: what this device recorded locally, and what the control
// plane says the account has reached.
//
// It exists because verifying the keyring's signatures proves an internal
// fact, not an absolute one: every generation in this object was signed by
// the account key this object publishes. A party with write access to the
// bucket satisfies that by replacing the whole object with one it signed
// itself. The anchor refuses that: the account key must be the one this
// device pinned at enrolment, the generation it last unwrapped must still be
// there, and that generation must still name the same root key.
//
// The local half is per device by construction, and that is the gap the
// floor fills: a device which has not yet read a rollover has nothing
// locally that a rollback would contradict. See keyring_floor.go.
type keyringAnchor struct {
	generation int
	recipient  string
	accountKey string
	// floor is this command's decision about the account-wide floor. Its
	// zero value is undecided, and check refuses it: every anchor built in
	// this package must say what it did about the floor, including the two
	// that deliberately have none to ask for.
	floor keyringFloor
}

// loadKeyringAnchor builds the anchor for one command: the local enrolment
// record, plus the account-wide key generation floor confirmed with the
// control plane (see confirmKeyGenerationFloor, which decides what to do on
// a profile that has none).
//
// A device with no enrolment record has no local anchor, which is correct:
// it has never seen this account's keyring, and the path it is on supplies
// its own anchor — the recovery code, or the root key relayed by an
// approval. It still gets the floor. A record that exists but does not
// validate is an error, never an absent anchor: schema.ValidateAccount
// requires every anchor field, so deleting one refuses the command instead
// of reaching the unanchored path.
func loadKeyringAnchor(cmd *cobra.Command, home string, cfg *schema.Config) (keyringAnchor, error) {
	floor, err := confirmKeyGenerationFloor(cmd, home, cfg)
	if err != nil {
		return keyringAnchor{}, err
	}
	account, err := config.LoadAccount(home)
	if os.IsNotExist(err) {
		return keyringAnchor{floor: floor}, nil
	}
	if err != nil {
		return keyringAnchor{}, NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	return anchorFromAccount(account, floor), nil
}

// anchorFromAccount is the one place an enrolment record becomes an anchor.
func anchorFromAccount(account *schema.Account, floor keyringFloor) keyringAnchor {
	return keyringAnchor{
		generation: account.KeyGeneration,
		recipient:  account.KeyRecipient,
		accountKey: account.AccountKey,
		floor:      floor,
	}
}

// keyringRolledBackError reports a keyring whose current generation is below
// a floor this command holds — either the generation this device already
// unwrapped, or the one the control plane reports for the account.
type keyringRolledBackError struct {
	saw, floor  int
	source      keyringFloorSource
	confirmedAt string
}

func (e *keyringRolledBackError) Error() string {
	source := e.source
	if source == "" {
		source = floorFromLocalRecord
	}
	when := ""
	if e.confirmedAt != "" {
		when = fmt.Sprintf(" (as of %s)", e.confirmedAt)
	}
	return fmt.Sprintf("keyring current_generation %d is below the %d %s%s; the keyring was rolled back (a revoked device may have restored an older copy inside its credential window). Nothing was written; run rein devices revoke again from a device that saw generation %d", e.saw, e.floor, source, when, e.floor)
}

// keyringRewrittenError reports a keyring whose history no longer matches
// what this device recorded: the generation it last unwrapped is gone, or
// now names a different root key. Appending never does either.
type keyringRewrittenError struct {
	generation  int
	want, found string
}

func (e *keyringRewrittenError) Error() string {
	what := fmt.Sprintf("key generation %d now names root key %s, not the %s this device recorded", e.generation, e.found, e.want)
	if e.found == "" {
		what = fmt.Sprintf("key generation %d, which this device has already unwrapped, is no longer in the keyring", e.generation)
	}
	return what + ". The keyring was replaced rather than appended to, which no legitimate change does; someone else can write this locker. Nothing was written: restore the account's keyring, or re-enrol this device from the recovery code once you know who has write access"
}

// checkKeyGenerationFloor fails closed when observed is below floor: a
// device that has seen a rollover, or been told about one, is never talked
// back into the generation a revoked device still holds.
func checkKeyGenerationFloor(observed, floor int, source keyringFloorSource, confirmedAt string) error {
	if observed < floor {
		return &keyringRolledBackError{saw: observed, floor: floor, source: source, confirmedAt: confirmedAt}
	}
	return nil
}

// keyringAnchorBrokenError reports an enrolment record that names a
// generation but not the values needed to check one. It should be
// unreachable — schema.ValidateAccount requires every anchor field — and is
// kept as a refusal rather than a skip so that a record which somehow
// reaches this point cannot buy an attacker the unanchored path.
type keyringAnchorBrokenError struct{ missing string }

func (e *keyringAnchorBrokenError) Error() string {
	return fmt.Sprintf("this device's enrolment record names a key generation but no %s, so it cannot tell this account's keyring from a replacement. Nothing was written: %s", e.missing, schema.AccountRemedy)
}

// check refuses a keyring this device must not act on. It reads nothing but
// public fields and holds no keys, so every command can afford it, including
// `rein account status` and `rein devices`.
//
// In order: the signatures, which are checked whether or not this device has
// an anchor of its own — a keyring holding one generation that does not
// verify is refused whole, never partly adopted; then the account-wide
// floor, which applies to every anchor including the two that carry no local
// record, because it is exactly the device with no local record that the
// per-device floor cannot help; then, with a local record, the account key
// those signatures are under must also be the one this device pinned, the
// current generation must not be below the one it already unwrapped, and
// that generation must still name the same root key.
func (a keyringAnchor) check(k *keyring.Keyring) error {
	if err := k.VerifyGenerations(a.accountKey); err != nil {
		return err
	}
	if !a.floor.decided {
		return &keyringFloorUndecidedError{}
	}
	if err := checkKeyGenerationFloor(k.CurrentGeneration, a.floor.generation, a.floor.source, a.floor.confirmedAt); err != nil {
		return err
	}
	if a.generation == 0 {
		return nil
	}
	if a.recipient == "" || a.accountKey == "" {
		missing := "root-key recipient"
		if a.accountKey == "" {
			missing = "account signing key"
		}
		return &keyringAnchorBrokenError{missing: missing}
	}
	if err := checkKeyGenerationFloor(k.CurrentGeneration, a.generation, floorFromLocalRecord, ""); err != nil {
		return err
	}
	if found := k.GenerationRecipient(a.generation); found != a.recipient {
		return &keyringRewrittenError{generation: a.generation, want: a.recipient, found: found}
	}
	return nil
}

// trustKeyring is the one place a loaded keyring becomes something this
// device will act on. The anchor check goes first — it verifies every
// generation's signature and pins the account key, and it needs no keys at
// all — and only then are the root keys unwrapped. Nothing is written under
// a generation this device could not authenticate.
//
// The returned keys must be zeroed by the caller.
func trustKeyring(anchor keyringAnchor, k *keyring.Keyring, unwrap func() (map[int][]byte, error)) (map[int][]byte, error) {
	if err := anchor.check(k); err != nil {
		return nil, err
	}
	return unwrap()
}

// observeKeyring records the keyring this device has just accepted: the
// current generation becomes the floor no later read may fall below, and its
// root-key recipient becomes the anchor a later read checks history against.
// It runs only after trustKeyring, so a device can never pin a generation it
// was unable to authenticate and lock itself out of its own account.
//
// floor is the decision the calling command already made, carried through
// rather than re-fetched: this re-check exists to catch the record moving
// under a long-running command, not to ask the control plane twice.
func observeKeyring(home string, floor keyringFloor, k *keyring.Keyring) error {
	account, err := config.LoadAccount(home)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return NewExitError(ExitConfig, "read account state: "+err.Error())
	}
	if err := anchorFromAccount(account, floor).check(k); err != nil {
		return NewExitError(ExitSafety, err.Error())
	}
	// The account key is pinned, so check above has already established it
	// is unchanged; only the generation and its recipient ever move.
	recipient := k.CurrentRecipient()
	if account.KeyGeneration == k.CurrentGeneration && account.KeyRecipient == recipient {
		return nil
	}
	account.KeyGeneration = k.CurrentGeneration
	account.KeyRecipient = recipient
	if err := config.SaveAccount(home, account); err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	return nil
}

// exitForRecoveryWrap separates the two things a recovery unwrap can mean,
// because they send a person to two different places.
//
// A wrap that did not open under the typed code is a code to check: the
// keyring parsed, so every generation's signature verified, and since
// keyring format 5 that signature covers the recovery wrap's parameters and
// ciphertext — so what is in front of the code is the wrap the account
// wrote. The remaining explanations are the code, and the keyring belonging
// to a different account.
//
// A wrap this build cannot even attempt — an unknown derivation or format, a
// salt or ciphertext that is not base64, a ciphertext shorter than its nonce
// — says nothing about the code, and saying "wrong code" there sends the
// person to check the one thing that is not wrong. It exits `7` with the
// same standing as any other tampering refusal.
//
// Both paths still exist because the signature is not a guarantee that
// every reader reached it: a caller that unwraps from an object it did not
// route through Parse would meet these shapes directly.
//
// undone is what this command did not do ("written", "revoked"), so the
// message names the caller's own outcome rather than a generic one.
func exitForRecoveryWrap(err error, undone string) error {
	switch {
	case errors.Is(err, keyring.ErrRecoveryWrapMalformed):
		return NewExitError(ExitSafety, fmt.Sprintf("%v. This is damage to the keyring, not a wrong recovery code: the object in storage does not hold a wrap this version can open at all. Nothing was %s; restore the account's keyring, or find out who else can write this locker", err, undone))
	case errors.Is(err, keyring.ErrRecoveryMismatch):
		return NewExitError(ExitAuthStorage, fmt.Sprintf("recovery code does not match this keyring; nothing was %s. The keyring's own signature is sound, so what did not match is the code as typed — or this keyring belongs to another account. Re-enter the code exactly as it was written down (case and dashes do not matter)", undone))
	}
	return nil
}

// exitForKeyringRefusal maps every keyring this device refuses to act on to
// a safety exit, leaving other errors untouched: one rolled back (against
// this device's own record or against the control plane's floor), one
// rewritten under it, one holding a generation that does not verify, one
// signed by another account's key, one whose local anchor is unusable, one
// reached by a command that never established the floor, and one that has
// grown past the size a read accepts. It is used where the refusal surfaces
// out of a compare-and-swap closure, and it is the reason every one of these
// exits `7` rather than the generic storage `4`.
func exitForKeyringRefusal(err error) error {
	var rolled *keyringRolledBackError
	var rewritten *keyringRewrittenError
	var broken *keyringAnchorBrokenError
	var undecided *keyringFloorUndecidedError
	switch {
	case errors.As(err, &rolled):
		return NewExitError(ExitSafety, rolled.Error())
	case errors.As(err, &rewritten):
		return NewExitError(ExitSafety, rewritten.Error())
	case errors.As(err, &broken):
		return NewExitError(ExitSafety, broken.Error())
	case errors.As(err, &undecided):
		return NewExitError(ExitSafety, undecided.Error())
	case errors.Is(err, keyring.ErrAccountKeyMismatch):
		return NewExitError(ExitSafety, err.Error()+". Nothing was written; the keyring in storage was replaced by one signed with a key this account never used")
	case errors.Is(err, keyring.ErrUnauthenticatedGeneration):
		return NewExitError(ExitSafety, err.Error()+". Nothing was written; a party with write access to the locker may have written a key generation of its own")
	case errors.Is(err, keyring.ErrTooLarge):
		return NewExitError(ExitSafety, err.Error())
	}
	return nil
}
