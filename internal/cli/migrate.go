package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/lock"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	"github.com/HarjjotSinghh/reinstate/internal/sync"
)

// migrateStateFile records an in-progress migration out of the locker so an
// interrupted run resumes against the same destination without rewriting
// what it already verified. It holds coordinates and digests only: no
// passphrase, no credential, no key.
const migrateStateFile = "migrate-byo.json"

// migrateState is the on-disk shape of migrateStateFile.
type migrateState struct {
	SchemaVersion int                  `json:"schema_version"`
	ProfileID     string               `json:"profile_id"`
	Destination   schema.StorageConfig `json:"destination"`
	// Done maps verified snapshot ids to their plaintext digests.
	Done      map[string]string `json:"done"`
	StartedAt string            `json:"started_at"`
	UpdatedAt string            `json:"updated_at"`
	// CompletedAt is set once the manifest was written and verified; the
	// file then stays so a rerun reuses the profile id instead of making a
	// second copy under a new one. Switching the config removes it.
	CompletedAt string `json:"completed_at,omitempty"`
}

func newSyncMigrateCmd() *cobra.Command {
	var (
		to, endpoint, bucket, region, prefix string
		switchConfig, keepHop, forgetHop     bool
		asJSON                               bool
	)
	cmd := &cobra.Command{
		Use:   "migrate --to byo [--endpoint URL --bucket NAME]",
		Short: "Move everything from the Reinstate Hop locker to your own bucket",
		Long: "Leave the hosted tier with your history intact. Every snapshot and the manifest\n" +
			"are read from the locker, opened on this device, re-sealed under a new BYO\n" +
			"passphrase, written to the destination bucket, and verified by reading them back.\n" +
			"The root key never leaves this device and is never written to the destination.\n" +
			"The locker is only read, so this works while the account is read-only (lapsed);\n" +
			"it is never emptied or deleted here (that is account deletion).\n\n" +
			"An interrupted run resumes: rerun the same command and verified snapshots are\n" +
			"skipped. Afterwards you are offered to switch this device to the destination\n" +
			"and, optionally, to forget the Hop sign-in on this device.\n\n" +
			"Credentials: REINSTATE_S3_ACCESS_KEY_ID / REINSTATE_S3_SECRET_ACCESS_KEY or a\n" +
			"hidden prompt. Passphrase: REINSTATE_PASSPHRASE_FD or a hidden prompt (entered twice).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if to != "byo" {
				return NewExitError(ExitUsage, "--to must be byo (the only migration destination)")
			}
			if switchConfig && keepHop {
				return NewExitError(ExitUsage, "--switch and --keep-hop-config exclude each other")
			}
			return runSyncMigrate(cmd, migrateOptions{
				endpoint: endpoint, bucket: bucket, region: region, prefix: prefix,
				switchConfig: switchConfig, keepHop: keepHop, forgetHop: forgetHop, asJSON: asJSON,
			})
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "destination kind; byo is the only value")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "destination S3/R2 endpoint URL (or REINSTATE_S3_ENDPOINT, or a prompt)")
	cmd.Flags().StringVar(&bucket, "bucket", "", "destination bucket (or REINSTATE_S3_BUCKET, or a prompt)")
	cmd.Flags().StringVar(&region, "region", "", "destination region (default auto)")
	cmd.Flags().StringVar(&prefix, "prefix", "", "destination key prefix (default profiles/<new profile id>)")
	cmd.Flags().BoolVar(&switchConfig, "switch", false, "switch this device's config to the destination after verification without asking")
	cmd.Flags().BoolVar(&keepHop, "keep-hop-config", false, "leave this device's config on the locker after verification without asking")
	cmd.Flags().BoolVar(&forgetHop, "forget-hop", false, "after switching, forget this device's Hop sign-in (the locker and account are untouched)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

type migrateOptions struct {
	endpoint, bucket, region, prefix string
	switchConfig, keepHop, forgetHop bool
	asJSON                           bool
}

func runSyncMigrate(cmd *cobra.Command, o migrateOptions) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	home, err := config.Home()
	if err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return configLoadExitError(err)
	}
	if cfg.Storage.Type != schema.StorageHop {
		return NewExitError(ExitConfig, "this profile does not use the Reinstate Hop locker (storage.type is "+cfg.Storage.Type+"); there is nothing to migrate")
	}
	if cfg.Encryption.Type != schema.EncryptionRootKey {
		return NewExitError(ExitConfig, "the locker is read with the root-key model; encryption.type is "+cfg.Encryption.Type)
	}
	if o.asJSON && !o.switchConfig && !o.keepHop {
		return NewExitError(ExitUsage, "--json requires --switch or --keep-hop-config to decide what happens after verification")
	}
	mutationLock, err := lock.Acquire(home, "mutation")
	if err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	defer func() { _ = mutationLock.Release() }()

	// Resume state decides the destination when it exists; flags must agree.
	state, resumed, err := loadMigrateState(home)
	if err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	if resumed {
		for name, pair := range map[string][2]string{
			"endpoint": {o.endpoint, state.Destination.Endpoint},
			"bucket":   {o.bucket, state.Destination.Bucket},
			"prefix":   {o.prefix, state.Destination.Prefix},
		} {
			if pair[0] != "" && pair[0] != pair[1] {
				return NewExitError(ExitUsage, fmt.Sprintf("a migration to %s at %s is in progress (%d snapshots verified); --%s %q differs. Finish or delete %s first",
					state.Destination.Bucket, state.Destination.Endpoint, len(state.Done), name, pair[0], filepath.Join(home, migrateStateFile)))
			}
		}
		if !o.asJSON && state.CompletedAt != "" {
			PrintHuman(errOut, "A migration to %s at %s completed at %s; checking it is still intact.", state.Destination.Bucket, state.Destination.Endpoint, state.CompletedAt)
		} else if !o.asJSON {
			PrintHuman(errOut, "Resuming the migration to %s at %s (%d snapshots already verified).", state.Destination.Bucket, state.Destination.Endpoint, len(state.Done))
		}
	} else {
		dest, err := migrateDestination(cmd, reader, o)
		if err != nil {
			return err
		}
		state = &migrateState{
			SchemaVersion: 1, ProfileID: uuid.NewString(), Destination: dest,
			Done: map[string]string{}, StartedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if state.Destination.Prefix == "" {
			state.Destination.Prefix = "profiles/" + state.ProfileID
		}
		state.Destination.CredentialRef = fmt.Sprintf("reinstate/%s/s3", state.ProfileID)
	}

	// Destination credentials and passphrase. Neither is written to the
	// state file; a resumed run asks again.
	accessKey, secretKey, promptedCredentials, err := migrateCredentials(cmd)
	if err != nil {
		return err
	}
	passphrase, err := readNewPassphrase(cmd)
	if err != nil {
		return NewExitError(ExitUsage, err.Error())
	}
	defer crypto.Zero(passphrase)

	// Source: the locker, read with the root key this device holds. The
	// migration never writes to it, so a read-only (lapsed) account works.
	sourceBackend, hosted, err := hostedBackend(cmd)
	if err != nil {
		return err
	}
	sourceKeys, err := rootKeysFromConfig(ctx, cmd, cfg, home, sourceBackend, "")
	if err != nil {
		return hostedError(hosted, err)
	}
	destBackend, err := migrateDestinationBackend(home, state.Destination, accessKey, secretKey)
	if err != nil {
		return err
	}
	var codec sync.EnvelopeCodec
	codec, _ = ctx.Value(envelopeCodecContextKey{}).(sync.EnvelopeCodec)
	source := &sync.Engine{Backend: sourceBackend, Keys: sourceKeys, Prefix: "", Codec: codec}
	dest := &sync.Engine{Backend: destBackend, Keys: crypto.NewPassphraseProvider(string(passphrase)), Prefix: state.Destination.Prefix, Codec: codec}

	if err := saveMigrateState(home, state); err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	var stateErr error
	migration := &sync.Migration{
		Source: source, Destination: dest, Done: state.Done,
		Progress: func(p sync.MigrateProgress) {
			if p.SnapshotID == "" {
				if !o.asJSON {
					PrintHuman(errOut, "manifest written and verified (%d snapshots, %s)", p.Total, humanBytes(p.Bytes))
				}
				return
			}
			state.Done[p.SnapshotID] = p.Digest
			if err := saveMigrateState(home, state); err != nil && stateErr == nil {
				stateErr = err
			}
			if o.asJSON {
				return
			}
			what := "written"
			switch {
			case p.Skipped:
				what = "verified earlier"
			case p.Existing:
				what = "found and verified"
			}
			PrintHuman(errOut, "[%d/%d] %s %s (%s so far)", p.Completed, p.Total, p.SnapshotID, what, humanBytes(p.Bytes))
		},
	}
	report, err := migration.Run(ctx)
	if err != nil {
		done := report.Written + report.Verified + report.Skipped
		switch {
		case errors.Is(err, context.Canceled):
			return NewExitError(ExitRuntime, fmt.Sprintf("migration interrupted after %d of %d snapshots; rerun the same command to resume", done, report.Snapshots))
		case errors.Is(err, sync.ErrDestinationInUse):
			return NewExitError(ExitSafety, fmt.Sprintf("%v; the root key must never live in BYO storage and another profile's storage is never merged into. Choose an empty bucket or prefix", err))
		case errors.Is(err, sync.ErrMigrateIncomplete):
			return hostedError(hosted, NewExitError(ExitAuthStorage, fmt.Sprintf("%v; nothing was written. Retry later; if it persists, the locker listing is incomplete", err)))
		case errors.Is(err, sync.ErrResumeMismatch):
			return NewExitError(ExitSafety, fmt.Sprintf("%v. Use the passphrase the earlier run used; to start over instead, delete %s and the destination prefix", err, filepath.Join(home, migrateStateFile)))
		case errors.Is(err, sync.ErrMigrateVerify):
			return NewExitError(ExitSafety, fmt.Sprintf("migration stopped after %d of %d snapshots: %v. That object was left for you to inspect; a rerun hits it again, so remove or move it at the destination first", done, report.Snapshots, err))
		}
		return hostedError(hosted, NewExitError(ExitAuthStorage, fmt.Sprintf("migration stopped after %d of %d snapshots: %v; rerun the same command to resume", done, report.Snapshots, err)))
	}
	if stateErr != nil {
		return NewExitError(ExitRuntime, "record migration progress: "+stateErr.Error())
	}
	state.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveMigrateState(home, state); err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}

	// Offer the switch. A BYO profile needs the credential in the OS keyring
	// when it was typed here rather than supplied by the environment.
	switched := false
	forgot := false
	decide := o.switchConfig
	if !o.switchConfig && !o.keepHop {
		PrintHuman(errOut, "Migration verified: %d snapshots and %d sessions now live at %s (%s).", report.Snapshots, report.ManifestSessions, state.Destination.Bucket, state.Destination.Prefix)
		if decide, err = promptYesNo(reader, errOut, "Switch this device to the destination bucket now? The locker stays as it is. [y/N] "); err != nil {
			return NewExitError(ExitUsage, err.Error())
		}
	}
	if decide {
		if err := switchConfigToBYO(home, cfg, state, accessKey, secretKey, promptedCredentials); err != nil {
			return err
		}
		switched = true
		forget := o.forgetHop
		if !o.forgetHop && !o.asJSON {
			if forget, err = promptYesNo(reader, errOut, "Forget this device's Hop sign-in too? The locker and account stay until you delete the account. [y/N] "); err != nil {
				return NewExitError(ExitUsage, err.Error())
			}
		}
		if forget {
			if err := hopSeamsFrom(cmd).tokenStore().DeleteDeviceToken(); err != nil && !errors.Is(err, credentials.ErrNoDeviceToken) {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			forgot = true
		}
	}

	if o.asJSON {
		return WriteJSON(out, map[string]any{
			"migrated":   report,
			"profile_id": state.ProfileID,
			"destination": map[string]string{
				"endpoint": state.Destination.Endpoint, "bucket": state.Destination.Bucket,
				"region": state.Destination.Region, "prefix": state.Destination.Prefix,
			},
			"switched":   switched,
			"forgot_hop": forgot,
		})
	}
	PrintHuman(out, "Migrated %d snapshots (%s) and %d sessions to %s at %s under %s.", report.Snapshots, humanBytes(report.Bytes), report.ManifestSessions, state.Destination.Bucket, state.Destination.Endpoint, state.Destination.Prefix)
	PrintHuman(out, "Every object there is sealed to the new passphrase; the root key was not written.")
	PrintHuman(out, "profile_id=%s (use this exact ID with rein init --profile-id on every other device)", state.ProfileID)
	if switched {
		PrintHuman(out, "This device now syncs to the destination (storage.type=s3, encryption.type=age-scrypt); the previous config.toml and state.json were backed up under backups/ (copy them back to revert).")
		if !promptedCredentials {
			PrintHuman(out, "The access keys came from REINSTATE_S3_* and were not stored in the OS keyring; keep them exported, or run rein init again to store them.")
		}
	} else {
		PrintHuman(out, "This device still syncs to the locker; rerun with --switch to change that. Nothing in the locker was changed.")
	}
	if forgot {
		PrintHuman(out, "This device's Hop sign-in was forgotten. The locker and account remain; deleting them is rein account delete.")
	}
	return nil
}

// migrateDestination collects the destination coordinates from flags,
// environment, or prompts.
func migrateDestination(cmd *cobra.Command, reader *bufio.Reader, o migrateOptions) (schema.StorageConfig, error) {
	dest := schema.StorageConfig{Type: schema.StorageS3, Endpoint: o.endpoint, Bucket: o.bucket, Region: o.region, Prefix: o.prefix}
	if dest.Endpoint == "" {
		dest.Endpoint = os.Getenv("REINSTATE_S3_ENDPOINT")
	}
	if dest.Bucket == "" {
		dest.Bucket = os.Getenv("REINSTATE_S3_BUCKET")
	}
	if dest.Region == "" {
		dest.Region = os.Getenv("REINSTATE_S3_REGION")
	}
	if dest.Region == "" {
		dest.Region = "auto"
	}
	var err error
	if dest.Endpoint == "" && !o.asJSON {
		if dest.Endpoint, err = promptLine(reader, cmd.ErrOrStderr(), "Destination S3/R2 endpoint: "); err != nil {
			return dest, NewExitError(ExitUsage, err.Error())
		}
	}
	if dest.Bucket == "" && !o.asJSON {
		if dest.Bucket, err = promptLine(reader, cmd.ErrOrStderr(), "Destination bucket: "); err != nil {
			return dest, NewExitError(ExitUsage, err.Error())
		}
	}
	if dest.Endpoint == "" || dest.Bucket == "" {
		return dest, NewExitError(ExitUsage, "the destination needs an endpoint and a bucket via flags, environment, or prompts")
	}
	dest.Prefix = strings.Trim(dest.Prefix, "/")
	return dest, nil
}

// migrateCredentials resolves the destination's access keys from the
// environment or hidden prompts. prompted reports that they were typed and
// therefore need storing when the config switches.
func migrateCredentials(cmd *cobra.Command) (accessKey, secretKey string, prompted bool, err error) {
	accessKey = os.Getenv("REINSTATE_S3_ACCESS_KEY_ID")
	secretKey = os.Getenv("REINSTATE_S3_SECRET_ACCESS_KEY")
	if (accessKey == "") != (secretKey == "") {
		return "", "", false, NewExitError(ExitAuthStorage, "both REINSTATE_S3_ACCESS_KEY_ID and REINSTATE_S3_SECRET_ACCESS_KEY are required")
	}
	if os.Getenv("REINSTATE_BACKEND") == "memory" && accessKey == "" {
		return "", "", false, nil
	}
	if accessKey != "" {
		return accessKey, secretKey, false, nil
	}
	access, err := crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), "Destination S3/R2 access key: ")
	if err != nil {
		return "", "", false, NewExitError(ExitAuthStorage, err.Error())
	}
	defer crypto.Zero(access)
	secret, err := crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), "Destination S3/R2 secret key: ")
	if err != nil {
		return "", "", false, NewExitError(ExitAuthStorage, err.Error())
	}
	defer crypto.Zero(secret)
	return string(access), string(secret), true, nil
}

// readNewPassphrase reads the destination passphrase: once from
// REINSTATE_PASSPHRASE_FD when automation opted in, otherwise twice from the
// terminal so a typo does not seal the migrated history to a passphrase
// nobody knows.
func readNewPassphrase(cmd *cobra.Command) ([]byte, error) {
	if secret, configured, err := crypto.ReadSecretFD(crypto.PassphraseFDEnv); configured {
		return secret, err
	}
	first, err := crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), "New BYO passphrase: ")
	if err != nil {
		return nil, err
	}
	second, err := crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), "Re-enter the passphrase: ")
	if err != nil {
		crypto.Zero(first)
		return nil, err
	}
	defer crypto.Zero(second)
	if string(first) != string(second) {
		crypto.Zero(first)
		return nil, fmt.Errorf("passphrases do not match; nothing was written")
	}
	return first, nil
}

// migrateDestinationBackend opens the destination: the disk-backed memory
// store under REINSTATE_BACKEND=memory (how journeys stand in a bucket),
// else an S3 client with the supplied keys.
func migrateDestinationBackend(home string, dest schema.StorageConfig, accessKey, secretKey string) (backend.Backend, error) {
	if os.Getenv("REINSTATE_BACKEND") == "memory" {
		disk, err := memory.NewDisk(memoryBackendRoot(home))
		if err != nil {
			return nil, NewExitError(ExitRuntime, err.Error())
		}
		return disk, nil
	}
	client, err := s3.New(context.Background(), s3.Config{
		Endpoint: dest.Endpoint, Region: dest.Region, Bucket: dest.Bucket,
		AccessKey: accessKey, SecretKey: secretKey,
	})
	if err != nil {
		return nil, NewExitError(ExitAuthStorage, err.Error())
	}
	return client, nil
}

// switchConfigToBYO backs up the hosted config and writes the BYO profile
// that points at the migrated destination. Local state keeps its snapshot
// ids (they were preserved), so no session is re-pushed or re-pulled.
func switchConfigToBYO(home string, old *schema.Config, state *migrateState, accessKey, secretKey string, storeCredentials bool) error {
	existing, err := existingInitFiles(home)
	if err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	if len(existing) != 0 {
		if _, err := fsx.BackupFiles(home, filepath.Join(home, "backups"), "migrate-byo", existing...); err != nil {
			return NewExitError(ExitRuntime, "back up hosted config: "+err.Error())
		}
	}
	cfg := schema.DefaultConfig(state.ProfileID, uuid.NewString())
	cfg.Storage = state.Destination
	cfg.Encryption.Type = schema.EncryptionPassphrase
	cfg.RemoteProfileRequired = true
	cfg.Agents = old.Agents
	cfg.Projects = old.Projects
	cfg.Restore = old.Restore
	if storeCredentials {
		if err := credentials.NewKeyringStore().Set(cfg.Storage.CredentialRef, credentials.StorageCredentials{AccessKeyID: accessKey, SecretAccessKey: secretKey}); err != nil {
			return NewExitError(ExitAuthStorage, err.Error())
		}
	}
	if err := config.SaveConfig(home, cfg); err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	st, err := config.LoadState(home)
	if err != nil {
		st = schema.NewState()
	}
	st.LastRemoteETag = ""
	if err := config.SaveState(home, st); err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	if err := os.Remove(filepath.Join(home, migrateStateFile)); err != nil && !os.IsNotExist(err) {
		return NewExitError(ExitRuntime, err.Error())
	}
	return nil
}

func loadMigrateState(home string) (*migrateState, bool, error) {
	raw, err := os.ReadFile(filepath.Join(home, migrateStateFile))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var st migrateState
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, false, fmt.Errorf("%s is unreadable: %w (delete it to start over)", migrateStateFile, err)
	}
	if st.SchemaVersion != 1 || st.ProfileID == "" || st.Destination.Bucket == "" {
		return nil, false, fmt.Errorf("%s is incomplete (delete it to start over)", migrateStateFile)
	}
	if st.Done == nil {
		st.Done = map[string]string{}
	}
	return &st, true, nil
}

func saveMigrateState(home string, st *migrateState) error {
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(home, migrateStateFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// promptYesNo asks a yes/no question; an empty answer or end of input is no.
func promptYesNo(reader *bufio.Reader, output io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprint(output, prompt); err != nil {
		return false, err
	}
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
