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

func newAccountCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "account",
		Short: "Hosted-tier key model: root key, keyring, recovery code",
		Long: `Manage this device's enrolment in the hosted key model.

The first device generates the root key and the recovery code. Other devices
enrol from the recovery code (rein account recover) and read everything the
first device wrote. The root key and recovery code never leave a device; the
keyring in storage holds only wrapped copies.`,
	}
	root.AddCommand(newAccountInitCmd(), newAccountRecoverCmd(), newAccountStatusCmd())
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
				return NewExitError(ExitSafety, "this device is already enrolled; rein account status shows the keyring state")
			} else if !os.IsNotExist(err) {
				return NewExitError(ExitConfig, "read account state: "+err.Error())
			}
			ctx := context.Background()
			store, prefix, err := backendFromConfig(cfg, home)
			if err != nil {
				return err
			}
			keyringKey := keyring.ObjectKey(prefix)
			if _, _, err := keyring.Load(ctx, store, keyringKey); err == nil {
				return NewExitError(ExitSafety, "a keyring already exists for this profile; enrol this device with rein account recover instead")
			} else if !errors.Is(err, keyring.ErrNotFound) {
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
				return NewExitError(ExitAuthStorage, err.Error())
			}
			if err := saveAccountEnrolment(home, cfg, now, ring.CurrentGeneration, "init"); err != nil {
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
				return NewExitError(ExitSafety, "this device is already enrolled; rein account status shows the keyring state")
			} else if !os.IsNotExist(err) {
				return NewExitError(ExitConfig, "read account state: "+err.Error())
			}
			ctx := context.Background()
			store, prefix, err := backendFromConfig(cfg, home)
			if err != nil {
				return err
			}
			keyringKey := keyring.ObjectKey(prefix)
			ring, _, err := keyring.Load(ctx, store, keyringKey)
			if errors.Is(err, keyring.ErrNotFound) {
				return NewExitError(ExitAuthStorage, "no keyring found at the configured storage; run rein account init on the first device, or check the profile and prefix")
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
			rootKey, err := ring.UnwrapWithRecoveryCode(recoveryCode)
			if errors.Is(err, keyring.ErrRecoveryMismatch) {
				return NewExitError(ExitAuthStorage, "recovery code does not match this keyring; nothing was written")
			}
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			defer crypto.Zero(rootKey)

			now := time.Now().UTC()
			secrets := seams.secretStore()
			secretRef := deviceSecretRef(cfg.ProfileID, cfg.DeviceID)
			existing, err := loadDeviceKey(secrets, secretRef)
			if err != nil {
				return err
			}
			listed := ring.HasDevice(cfg.DeviceID)
			switch {
			case existing != nil && listed:
				// The keyring already holds a wrap for the key this device
				// keeps (for example the local record was lost, or an earlier
				// enrolment crashed before writing it). Re-attach without
				// touching the device key or the keyring.
				current, earlier, err := ring.UnwrapForDevice(cfg.DeviceID, existing)
				if err != nil {
					return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s but not the key held in the OS keyring (%v); nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device", cfg.DeviceID, err))
				}
				crypto.Zero(current)
				for _, key := range earlier {
					crypto.Zero(key)
				}
				if err := saveAccountEnrolment(home, cfg, now, ring.CurrentGeneration, "recover"); err != nil {
					return err
				}
				PrintHuman(cmd.OutOrStdout(), "device already enrolled; local enrolment record restored, keyring and device key unchanged")
				PrintHuman(cmd.OutOrStdout(), "profile_id=%s device_id=%s key_generation=%d devices=%d", cfg.ProfileID, cfg.DeviceID, ring.CurrentGeneration, ring.DeviceCount())
				PrintHuman(cmd.ErrOrStderr(), "%s", unrecoverableNotice)
				return nil
			case existing != nil:
				return NewExitError(ExitSafety, fmt.Sprintf("the OS keyring already holds a key for device %s that the keyring does not list; nothing was written. Remove that entry (%s) or choose a new device_id in config before enrolling", cfg.DeviceID, secretRef))
			case listed:
				return NewExitError(ExitSafety, fmt.Sprintf("the keyring lists device %s but this machine holds no key for it; nothing was written. Choose a new device_id in config, or revoke this device from another enrolled device", cfg.DeviceID))
			}

			deviceKey, err := age.GenerateX25519Identity()
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			if err := secrets.SetSecret(secretRef, []byte(deviceKey.String())); err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			updated, err := keyring.Update(ctx, store, keyringKey, func(k *keyring.Keyring) error {
				return k.Enrol(rootKey, cfg.DeviceID, deviceKey.Recipient(), now)
			})
			if err != nil {
				// Only the key this command just created is rolled back.
				_ = secrets.DeleteSecret(secretRef)
				return NewExitError(ExitAuthStorage, err.Error())
			}
			if err := saveAccountEnrolment(home, cfg, now, updated.CurrentGeneration, "recover"); err != nil {
				return err
			}
			PrintHuman(cmd.OutOrStdout(), "device enrolled from the recovery code; this device now reads everything written under key generation %d", updated.CurrentGeneration)
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
			if store, prefix, err := backendFromConfig(cfg, home); err != nil {
				r.Error = err.Error()
			} else if ring, _, err := keyring.Load(context.Background(), store, keyring.ObjectKey(prefix)); err == nil {
				r.KeyringPresent = true
				r.KeyGeneration = ring.CurrentGeneration
				r.EnrolledDevices = ring.DeviceCount()
				r.DeviceInKeyring = ring.HasDevice(cfg.DeviceID)
			} else if !errors.Is(err, keyring.ErrNotFound) {
				r.Error = err.Error()
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
				PrintHuman(out, "keyring: unavailable (%s)", r.Error)
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
func saveAccountEnrolment(home string, cfg *schema.Config, now time.Time, generation int, via string) error {
	cfg.Encryption.Type = schema.EncryptionRootKey
	if err := config.SaveConfig(home, cfg); err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	account := &schema.Account{
		SchemaVersion:         schema.AccountSchemaVersion,
		ProfileID:             cfg.ProfileID,
		DeviceID:              cfg.DeviceID,
		KeyGeneration:         generation,
		RecoveryCodeConfirmed: true,
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
	keys, err := crypto.NewRootKeyProvider(current, earlier...)
	crypto.Zero(current)
	for _, key := range earlier {
		crypto.Zero(key)
	}
	if err != nil {
		return nil, NewExitError(ExitRuntime, err.Error())
	}
	return keys, nil
}
