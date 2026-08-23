package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/lock"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	"github.com/HarjjotSinghh/reinstate/internal/sync"
	"github.com/HarjjotSinghh/reinstate/internal/tui/wizard"
)

func defaultRegistry() *adapter.Registry {
	userHome, _ := os.UserHomeDir()
	projects := map[string]string{}
	if reinstateHome, err := config.Home(); err == nil {
		if cfg, loadErr := config.LoadConfig(reinstateHome); loadErr == nil {
			for _, project := range cfg.Projects {
				projects[project.ID] = project.LocalRoot
			}
		}
	}
	r := adapter.NewRegistry()
	for _, descriptor := range agents.Capable(agents.CapabilitySync) {
		if descriptor.NewSyncAdapter == nil {
			continue
		}
		env := agents.Env{Home: userHome, LookupEnv: os.Getenv}
		if descriptor.Storage.RootEnv != "" {
			env.FixtureRoot = strings.TrimSpace(os.Getenv(descriptor.Storage.RootEnv))
		}
		instance, err := descriptor.NewSyncAdapter(env)
		if err != nil || instance == nil {
			continue
		}
		assignAdapterProjects(instance, projects)
		_ = r.Register(instance)
	}
	return r
}

func assignAdapterProjects(instance adapter.Adapter, projects map[string]string) {
	value := reflect.ValueOf(instance)
	if value.Kind() != reflect.Pointer {
		return
	}
	field := value.Elem().FieldByName("Projects")
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(projects))
	}
}

func newInitCmd() *cobra.Command {
	var (
		endpoint, bucket, region, prefix, configuredProfileID string
		projectMappings                                       []string
		nonInteractive                                        bool
		force                                                 bool
		link                                                  bool
		paste                                                 bool
		hosted                                                bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup (backend, encryption, path map)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if link {
				return printPairingCode(cmd)
			}
			home, err := config.Home()
			if err != nil {
				return NewExitError(ExitConfig, err.Error())
			}
			existingFiles, err := existingInitFiles(home)
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			if len(existingFiles) != 0 && !force {
				return NewExitError(
					ExitSafety,
					"reinstate home is already initialized; rerun init with --force to back up and replace existing config/state",
				)
			}
			if err := config.EnsureLayout(home); err != nil {
				return err
			}
			if hosted {
				if endpoint != "" || bucket != "" || prefix != "" || paste || configuredProfileID != "" {
					return NewExitError(ExitUsage, "--hop takes its endpoint, bucket, and profile from the signed-in account; do not combine it with --endpoint, --bucket, --prefix, --profile-id, or --paste")
				}
				return initHosted(cmd, home, existingFiles, projectMappings)
			}
			// Interactive setup collects the non-secret coordinates first, so a
			// mistake in one field never discards the others and a bad value is
			// reported while the reader is still looking at it.
			capability := resolveCapability(cmd, localCommandOptions{terminalCheck: defaultInitTerminalCheck}, false)
			if wizardApplies(capability, nonInteractive, paste, endpoint, bucket) {
				defaults := wizard.Result{
					Provider: "r2",
					Endpoint: endpoint,
					Bucket:   bucket,
					Region:   region,
					Prefix:   prefix,
				}
				if paste {
					payload, pasteErr := readPairingCode(cmd)
					if pasteErr != nil {
						return pasteErr
					}
					defaults = pairingDefaults(payload)
				}
				result, ok, wizardErr := runInitWizard(cmd, capability, defaults)
				if wizardErr != nil {
					return wizardErr
				}
				if !ok {
					PrintHuman(cmd.ErrOrStderr(), "setup cancelled; nothing was written")
					return nil
				}
				endpoint = result.Endpoint
				bucket = result.Bucket
				region = result.Region
				prefix = result.Prefix
				if result.JoinExisting {
					configuredProfileID = result.ProfileID
				}
			}
			profileID := configuredProfileID
			if profileID == "" {
				profileID = uuid.NewString()
			} else if _, err := uuid.Parse(profileID); err != nil {
				return NewExitError(ExitUsage, "--profile-id must be a UUID copied from the first device")
			}
			deviceID := uuid.NewString()
			reader := bufio.NewReader(cmd.InOrStdin())
			if endpoint == "" {
				endpoint = os.Getenv("REINSTATE_S3_ENDPOINT")
			}
			if bucket == "" {
				bucket = os.Getenv("REINSTATE_S3_BUCKET")
			}
			if region == "" {
				region = os.Getenv("REINSTATE_S3_REGION")
				if region == "" {
					region = "auto"
				}
			}
			if !nonInteractive && endpoint == "" {
				endpoint, err = promptLine(reader, cmd.ErrOrStderr(), "S3/R2 endpoint: ")
				if err != nil {
					return NewExitError(ExitUsage, err.Error())
				}
			}
			if !nonInteractive && bucket == "" {
				bucket, err = promptLine(reader, cmd.ErrOrStderr(), "Bucket: ")
				if err != nil {
					return NewExitError(ExitUsage, err.Error())
				}
			}
			if endpoint == "" || bucket == "" {
				return NewExitError(ExitUsage, "init requires endpoint and bucket via prompts, flags, or environment")
			}
			accessKey := os.Getenv("REINSTATE_S3_ACCESS_KEY_ID")
			secretKey := os.Getenv("REINSTATE_S3_SECRET_ACCESS_KEY")
			if (accessKey == "") != (secretKey == "") {
				return NewExitError(ExitAuthStorage, "both storage credential environment values are required")
			}
			storeInKeyring := false
			if accessKey == "" {
				if nonInteractive {
					return NewExitError(ExitAuthStorage, "non-interactive init requires the documented environment credential provider")
				}
				accessSecret, readErr := crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), "S3/R2 access key: ")
				if readErr != nil {
					return NewExitError(ExitAuthStorage, readErr.Error())
				}
				defer crypto.Zero(accessSecret)
				secretSecret, readErr := crypto.ReadHiddenSecret(cmd.InOrStdin(), cmd.ErrOrStderr(), "S3/R2 secret key: ")
				if readErr != nil {
					return NewExitError(ExitAuthStorage, readErr.Error())
				}
				defer crypto.Zero(secretSecret)
				accessKey = string(accessSecret)
				secretKey = string(secretSecret)
				storeInKeyring = true
			}
			credRef := fmt.Sprintf("reinstate/%s/s3", profileID)
			cfg := schema.DefaultConfig(profileID, deviceID)
			cfg.Storage = schema.StorageConfig{
				Type:          "s3",
				Endpoint:      endpoint,
				Region:        region,
				Bucket:        bucket,
				Prefix:        prefix,
				CredentialRef: credRef,
			}
			if prefix == "" {
				cfg.Storage.Prefix = "profiles/" + profileID
			}
			cfg.RemoteProfileRequired = configuredProfileID != ""
			for _, mapping := range projectMappings {
				project, parseErr := parseProjectMapping(mapping)
				if parseErr != nil {
					return NewExitError(ExitUsage, parseErr.Error())
				}
				cfg.Projects = append(cfg.Projects, project)
			}
			// A failed backend probe is fatal. Do not leave a config that looks
			// initialized but cannot reach storage.
			ctx := context.Background()
			if os.Getenv("REINSTATE_BACKEND") == "memory" {
				if configuredProfileID != "" {
					disk, err := memory.NewDisk(memoryBackendRoot(home))
					if err != nil {
						return NewExitError(ExitRuntime, err.Error())
					}
					manifestKey := strings.Trim(cfg.Storage.Prefix, "/") + "/manifest.age"
					if err := requireRemoteProfileManifest(ctx, disk, manifestKey); err != nil {
						return NewExitError(ExitAuthStorage, err.Error())
					}
				}
			} else {
				client, err := s3.New(ctx, s3.Config{
					Endpoint: endpoint, Region: region, Bucket: bucket, Prefix: cfg.Storage.Prefix,
					AccessKey: accessKey, SecretKey: secretKey,
				})
				if err != nil {
					return NewExitError(ExitAuthStorage, err.Error())
				}
				if configuredProfileID != "" {
					if err := requireRemoteProfileManifest(ctx, client, "manifest.age"); err != nil {
						return NewExitError(ExitAuthStorage, err.Error())
					}
				}
				probe := "probes/" + uuid.NewString()
				if _, err := client.Put(ctx, probe, strings.NewReader("ok"), 2, backend.PutOptions{IfNoneMatch: true}); err != nil {
					return NewExitError(ExitAuthStorage, "storage probe put failed: "+err.Error())
				} else if err := client.Delete(ctx, probe); err != nil {
					return NewExitError(ExitAuthStorage, "storage probe cleanup failed: "+err.Error())
				}
			}
			keyringStore := credentials.NewKeyringStore()
			if len(existingFiles) != 0 {
				backupPath, err := fsx.BackupFiles(
					home,
					filepath.Join(home, "backups"),
					"reinitialize",
					existingFiles...,
				)
				if err != nil {
					return NewExitError(ExitRuntime, "back up existing init state: "+err.Error())
				}
				backupRelative, err := filepath.Rel(home, backupPath)
				if err != nil {
					return NewExitError(ExitRuntime, "report init backup path: "+err.Error())
				}
				PrintHuman(
					cmd.OutOrStdout(),
					"backed up existing config/state to %s before reinitializing",
					filepath.ToSlash(backupRelative),
				)
			}
			if storeInKeyring {
				if err := keyringStore.Set(credRef, credentials.StorageCredentials{
					AccessKeyID: accessKey, SecretAccessKey: secretKey,
				}); err != nil {
					return NewExitError(ExitAuthStorage, err.Error())
				}
			}
			if err := config.SaveConfig(home, cfg); err != nil {
				if storeInKeyring {
					_ = keyringStore.Delete(credRef)
				}
				return err
			}
			if err := config.SaveState(home, schema.NewState()); err != nil {
				return err
			}
			PrintHuman(cmd.OutOrStdout(), "initialized reinstate home (config.toml + state.json). Passphrase is not stored; you will enter it on push/pull.")
			PrintHuman(cmd.OutOrStdout(), "profile_id=%s (use this exact ID on every device)", profileID)
			PrintHuman(cmd.OutOrStdout(), "home configured; credential_ref=%s", credRef)
			return nil
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "S3/R2 endpoint URL")
	cmd.Flags().StringVar(&bucket, "bucket", "", "bucket name")
	cmd.Flags().StringVar(&region, "region", "auto", "region")
	cmd.Flags().StringVar(&prefix, "prefix", "", "object key prefix")
	cmd.Flags().StringVar(&configuredProfileID, "profile-id", "", "existing profile UUID from the first device")
	cmd.Flags().StringArrayVar(&projectMappings, "project", nil, "portable project mapping ID=/absolute/local/path (repeatable)")
	cmd.Flags().BoolVar(&nonInteractive, "yes", false, "non-interactive mode; requires endpoint, bucket, and environment credential provider")
	cmd.Flags().BoolVar(&link, "link", false, "print this profile's pairing code for another device")
	cmd.Flags().BoolVar(&paste, "paste", false, "start setup from a pairing code printed by another device")
	cmd.Flags().BoolVar(&force, "force", false, "back up and replace an already-initialized home")
	cmd.Flags().BoolVar(&hosted, "hop", false, "use the Reinstate Hop locker of the signed-in account instead of your own bucket")
	return cmd
}

// initHosted writes a config that syncs to the signed-in account's locker.
// The profile is the account (one locker, one profile), the device is the
// enrolled device, and no storage coordinate or credential is stored: the
// control plane supplies them per session. Provisioning the locker is the
// reachability probe.
func initHosted(cmd *cobra.Command, home string, existingFiles, projectMappings []string) error {
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return err
	}
	if tok.AccountID == "" || tok.DeviceID == "" {
		return NewExitError(ExitAuthStorage, "the stored device token predates locker support; run rein login again")
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	locker, err := client.ProvisionLocker(ctx, tok.Token)
	if err != nil {
		return hopExitError(err)
	}
	cfg := schema.DefaultConfig(tok.AccountID, tok.DeviceID)
	cfg.Storage = schema.StorageConfig{Type: schema.StorageHop}
	cfg.Hop.URL = tok.ControlPlaneURL
	for _, mapping := range projectMappings {
		project, parseErr := parseProjectMapping(mapping)
		if parseErr != nil {
			return NewExitError(ExitUsage, parseErr.Error())
		}
		cfg.Projects = append(cfg.Projects, project)
	}
	if len(existingFiles) != 0 {
		backupPath, err := fsx.BackupFiles(home, filepath.Join(home, "backups"), "reinitialize", existingFiles...)
		if err != nil {
			return NewExitError(ExitRuntime, "back up existing init state: "+err.Error())
		}
		backupRelative, err := filepath.Rel(home, backupPath)
		if err != nil {
			return NewExitError(ExitRuntime, "report init backup path: "+err.Error())
		}
		PrintHuman(cmd.OutOrStdout(), "backed up existing config/state to %s before reinitializing", filepath.ToSlash(backupRelative))
	}
	if err := config.SaveConfig(home, cfg); err != nil {
		return err
	}
	if err := config.SaveState(home, schema.NewState()); err != nil {
		return err
	}
	PrintHuman(cmd.OutOrStdout(), "initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=%s", schema.StorageHop)
	PrintHuman(cmd.OutOrStdout(), "locker %s at %s (location %s, plan %s)", locker.Bucket, locker.Endpoint, locker.LocationHint, locker.Plan)
	PrintHuman(cmd.OutOrStdout(), "profile_id=%s device_id=%s", cfg.ProfileID, cfg.DeviceID)
	PrintHuman(cmd.OutOrStdout(), "next: rein account init on this first device (or rein account join on another), then rein push")
	return nil
}

func existingInitFiles(home string) ([]string, error) {
	var existing []string
	for _, name := range []string{"config.toml", "state.json"} {
		if _, err := os.Lstat(filepath.Join(home, name)); err == nil {
			existing = append(existing, name)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return existing, nil
}

func requireRemoteProfileManifest(ctx context.Context, store backend.Backend, key string) error {
	body, _, err := store.Get(ctx, key)
	if errors.Is(err, backend.ErrNotFound) {
		return fmt.Errorf("remote profile manifest not found at configured storage coordinates")
	}
	if err != nil {
		return fmt.Errorf("remote profile manifest probe failed: %w", err)
	}
	if _, err := io.Copy(io.Discard, body); err != nil {
		_ = body.Close()
		return fmt.Errorf("remote profile manifest probe failed while reading: %w", err)
	}
	if err := body.Close(); err != nil {
		return fmt.Errorf("remote profile manifest probe failed while closing: %w", err)
	}
	return nil
}

func parseProjectMapping(value string) (schema.ProjectConfig, error) {
	id, localRoot, ok := strings.Cut(value, "=")
	id = strings.TrimSpace(id)
	localRoot = strings.TrimSpace(localRoot)
	if !ok || id == "" || localRoot == "" {
		return schema.ProjectConfig{}, fmt.Errorf("--project must use ID=/absolute/local/path")
	}
	if strings.ContainsAny(id, "\r\n\t") {
		return schema.ProjectConfig{}, fmt.Errorf("project ID contains control characters")
	}
	if !filepath.IsAbs(localRoot) {
		return schema.ProjectConfig{}, fmt.Errorf("project local path must be absolute")
	}
	return schema.ProjectConfig{ID: id, LocalRoot: filepath.Clean(localRoot)}, nil
}

func promptLine(reader *bufio.Reader, output io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(output, prompt); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("value must not be empty")
	}
	return value, nil
}

func newListCmd() *cobra.Command {
	var asJSON bool
	var agent string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local agent sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := defaultRegistry()
			var all []adapter.Session
			for _, name := range reg.Names() {
				if agent != "all" && agent != "" && name != agent {
					continue
				}
				a, _ := reg.Get(name)
				ss, err := a.Discover(context.Background(), adapter.DiscoverOptions{})
				if err != nil {
					return err
				}
				all = append(all, ss...)
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), all)
			}
			for _, s := range all {
				PrintHuman(cmd.OutOrStdout(), "%s\t%s\t%s", s.Agent, s.ID, s.ProjectID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().StringVar(&agent, "agent", "all", "agent filter: "+agentFilterHelp(agents.TierSync, true))
	return cmd
}

func engineFromConfig(cmd *cobra.Command, passphrase string) (*sync.Engine, *schema.Config, string, error) {
	home, err := config.Home()
	if err != nil {
		return nil, nil, "", NewExitError(ExitConfig, err.Error())
	}
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return nil, nil, "", configLoadExitError(err)
	}
	requireRemoteManifest := cfg.RemoteProfileRequired
	if state, stateErr := config.LoadState(home); stateErr == nil {
		requireRemoteManifest = requireRemoteManifest ||
			state.LastManifestRev != "" ||
			len(state.Sessions) != 0
	}
	b, enginePrefix, hosted, err := openBackend(cmd, cfg, home)
	if err != nil {
		return nil, nil, "", err
	}
	var keys crypto.KeyProvider
	if cfg.Encryption.Type == schema.EncryptionRootKey {
		keys, err = rootKeysFromConfig(context.Background(), cmd, cfg, home, b, enginePrefix)
		if err != nil {
			return nil, nil, "", hostedError(hosted, err)
		}
	} else if hosted != nil {
		return nil, nil, "", NewExitError(ExitConfig, "the hosted tier uses the root-key model; run rein account init on the first device (encryption.type must be "+schema.EncryptionRootKey+", not "+cfg.Encryption.Type+")")
	} else {
		if passphrase == "" {
			passphrase = cachedPassphraseFrom(cmd)
		}
		if passphrase == "" {
			secret, err := crypto.ReadPassphrase(cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return nil, nil, "", NewExitError(ExitUsage, err.Error())
			}
			passphrase = string(secret)
			crypto.Zero(secret)
		}
		keys = crypto.NewPassphraseProvider(passphrase)
	}
	var envelopeCodec sync.EnvelopeCodec
	if commandContext := cmd.Context(); commandContext != nil {
		envelopeCodec, _ = commandContext.Value(envelopeCodecContextKey{}).(sync.EnvelopeCodec)
	}
	eng := &sync.Engine{
		Backend:               b,
		Keys:                  keys,
		Prefix:                enginePrefix,
		RequireRemoteManifest: requireRemoteManifest,
		Codec:                 envelopeCodec,
	}
	if hosted != nil {
		rememberHosted(cmd, hosted)
	}
	return eng, cfg, home, nil
}

// memoryBackendRoot is where the disk-backed "memory" backend keeps objects.
// REINSTATE_MEMORY_BACKEND_DIR lets two homes share one store, which is how
// the CLI journeys simulate two devices against one locker.
func memoryBackendRoot(home string) string {
	if dir := strings.TrimSpace(os.Getenv("REINSTATE_MEMORY_BACKEND_DIR")); dir != "" && filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(home, "cache", "memory-backend")
}

// backendFromConfig opens the configured storage: disk-backed "memory" for
// local e2e, the hosted locker for storage.type "hop", else S3. The
// returned prefix is the engine-side key prefix (empty when the client
// already scopes keys).
func backendFromConfig(cmd *cobra.Command, cfg *schema.Config, home string) (backend.Backend, string, error) {
	b, prefix, _, err := openBackend(cmd, cfg, home)
	return b, prefix, err
}

// openBackend is backendFromConfig that also returns the hosted credential
// source when the locker is in use, so callers can report the control
// plane's refusal instead of the storage layer's wrapped error.
func openBackend(cmd *cobra.Command, cfg *schema.Config, home string) (backend.Backend, string, *hop.Source, error) {
	if os.Getenv("REINSTATE_BACKEND") == "memory" {
		disk, err := memory.NewDisk(memoryBackendRoot(home))
		if err != nil {
			return nil, "", nil, NewExitError(ExitRuntime, err.Error())
		}
		return disk, cfg.Storage.Prefix, nil, nil
	}
	if cfg.Storage.Type == schema.StorageHop {
		client, source, err := hostedBackend(cmd)
		if err != nil {
			return nil, "", nil, err
		}
		return client, "", source, nil
	}
	creds, err := credentials.Resolve(home, cfg.Storage.CredentialRef)
	if err != nil {
		return nil, "", nil, NewExitError(ExitAuthStorage, err.Error())
	}
	client, err := s3.New(context.Background(), s3.Config{
		Endpoint: cfg.Storage.Endpoint, Region: cfg.Storage.Region,
		Bucket: cfg.Storage.Bucket, Prefix: cfg.Storage.Prefix,
		AccessKey: creds.AccessKeyID, SecretKey: creds.SecretAccessKey,
	})
	if err != nil {
		return nil, "", nil, NewExitError(ExitAuthStorage, err.Error())
	}
	return client, "", nil, nil
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Compare local vs remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, _, err := engineFromConfig(cmd, "")
			if err != nil {
				return err
			}
			man, err := eng.FetchManifest(context.Background())
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			type row struct {
				Key      string `json:"key"`
				Snapshot string `json:"snapshot"`
				Updated  string `json:"updated_at"`
			}
			var rows []row
			for k, s := range man.Sessions {
				rows = append(rows, row{Key: k, Snapshot: s.SnapshotID, Updated: s.UpdatedAt})
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]any{"remote_sessions": rows, "revision": man.Revision})
			}
			PrintHuman(cmd.OutOrStdout(), "remote revision: %s (%d sessions)", man.Revision, len(rows))
			for _, r := range rows {
				PrintHuman(cmd.OutOrStdout(), "  %s -> %s", r.Key, r.Snapshot)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func newDiffCmd() *cobra.Command {
	var asJSON bool
	var agentName, sessionID string
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show pending change metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			// metadata only — list local sessions vs remote keys
			reg := defaultRegistry()
			var local []string
			for _, a := range reg.All() {
				if agentName != "" && a.Name() != agentName {
					continue
				}
				ss, err := a.Discover(context.Background(), adapter.DiscoverOptions{})
				if err != nil {
					return err
				}
				for _, s := range ss {
					if sessionID != "" && s.ID != sessionID {
						continue
					}
					local = append(local, sync.SessionKey(s.Agent, s.ID))
				}
			}
			eng, _, _, err := engineFromConfig(cmd, "")
			if err != nil {
				return err
			}
			remote := []string{}
			man, err := eng.FetchManifest(context.Background())
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			for key, session := range man.Sessions {
				if agentName != "" && session.Agent != agentName {
					continue
				}
				if sessionID != "" && session.SessionID != sessionID {
					continue
				}
				remote = append(remote, key)
			}
			out := map[string]any{"local": local, "remote": remote}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), out)
			}
			PrintHuman(cmd.OutOrStdout(), "local: %d sessions; remote: %d sessions", len(local), len(remote))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent filter")
	cmd.Flags().StringVar(&sessionID, "session", "", "session id")
	return cmd
}

func newPushCmd() *cobra.Command {
	var asJSON, dryRun, all bool
	var agent, session string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Encrypt and upload local sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, home, err := engineFromConfig(cmd, "")
			if err != nil {
				return err
			}
			mutationLock, err := lock.Acquire(home, "mutation")
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			defer func() { _ = mutationLock.Release() }()
			state, err := config.LoadState(home)
			if err != nil {
				return NewExitError(ExitConfig, err.Error())
			}
			reg := defaultRegistry()
			var items []sync.PushItem
			for _, name := range reg.Names() {
				if agent != "" && name != agent {
					continue
				}
				a, _ := reg.Get(name)
				ss, err := a.Discover(context.Background(), adapter.DiscoverOptions{})
				if err != nil {
					return err
				}
				for _, s := range ss {
					if session != "" && s.ID != session {
						continue
					}
					items = append(items, sync.PushItem{
						Agent: s.Agent, SessionID: s.ID, ProjectID: s.ProjectID,
						LocalPath: s.Path, RelativePath: s.RelativePath,
						BaseKnown: true,
					})
					if prior, ok := state.Sessions[sync.SessionKey(s.Agent, s.ID)]; ok {
						items[len(items)-1].BaseRevision = prior.RemoteRevision
					}
				}
			}
			if !all && session == "" && len(items) > 1 {
				return NewExitError(ExitUsage, "specify --session or --all")
			}
			if len(items) == 0 {
				return NewExitError(ExitUsage, "no matching local sessions found")
			}
			remoteManifest, err := eng.FetchManifest(context.Background())
			if err != nil {
				return hostedError(hostedFrom(cmd), NewExitError(ExitAuthStorage, err.Error()))
			}
			var uploaded []string
			var skipped int
			var conflicted []string
			for _, it := range items {
				a, ok := reg.Get(it.Agent)
				if !ok {
					return NewExitError(ExitCompatibility, "adapter unavailable for "+it.Agent)
				}
				sessionMeta := adapter.Session{
					ID:           it.SessionID,
					Agent:        it.Agent,
					ProjectID:    it.ProjectID,
					Path:         it.LocalPath,
					RelativePath: it.RelativePath,
				}
				localHash, err := sessionRevision(a, sessionMeta)
				if err != nil {
					return err
				}
				key := sync.SessionKey(it.Agent, it.SessionID)
				prior, priorKnown := state.Sessions[key]
				remote, remoteKnown := remoteManifest.Sessions[key]
				if priorKnown && remoteKnown &&
					prior.LocalRevision == localHash &&
					prior.RemoteRevision == remote.SnapshotID {
					skipped++
					continue
				}
				plan, err := a.PlanExport(context.Background(), sessionMeta, adapter.ExportOptions{DryRun: dryRun})
				if err != nil {
					return err
				}
				exportDir := filepath.Join(home, "cache", "exports")
				if err := os.MkdirAll(exportDir, 0o700); err != nil {
					return err
				}
				exportFile, err := os.CreateTemp(exportDir, ".reinstate-export-*.tar")
				if err != nil {
					return err
				}
				exportPath := exportFile.Name()
				if err := a.Export(context.Background(), plan, exportFile); err != nil {
					_ = exportFile.Close()
					_ = os.Remove(exportPath)
					return err
				}
				if err := exportFile.Close(); err != nil {
					_ = os.Remove(exportPath)
					return err
				}
				defer func() { _ = os.Remove(exportPath) }()
				exportItem := it
				exportItem.LocalPath = exportPath
				id, err := eng.PushSession(context.Background(), exportItem, dryRun)
				if err != nil {
					if errors.Is(err, sync.ErrConflict) {
						remoteSnapshot := ""
						if manifest, fetchErr := eng.FetchManifest(context.Background()); fetchErr == nil {
							if remote, ok := manifest.Sessions[sync.SessionKey(it.Agent, it.SessionID)]; ok {
								remoteSnapshot = remote.SnapshotID
							}
						}
						// LocalRevision is the current local hash, the same
						// key pull --all records for this divergence, so push
						// and pull share one record instead of two.
						_ = sync.SaveConflict(home, sync.Conflict{
							Agent: it.Agent, SessionID: it.SessionID, ProjectID: it.ProjectID,
							LocalRevision: localHash, RemoteRevision: remoteSnapshot,
							RemoteSnapshot: remoteSnapshot,
						})
						if session != "" {
							return NewExitError(ExitConflict, err.Error())
						}
						// push --all keeps going so one diverged session
						// does not hold every other session's changes back
						// (the daemon runs this push after every change).
						conflicted = append(conflicted, key)
						continue
					}
					if hosted := hostedFrom(cmd); hosted != nil && hosted.LastError() != nil {
						return hostedError(hosted, err)
					}
					if strings.Contains(err.Error(), "credential") {
						return NewExitError(ExitSafety, err.Error())
					}
					return err
				}
				uploaded = append(uploaded, id)
				if !dryRun {
					state.Sessions[key] = schema.SessionState{
						Agent: it.Agent, SessionID: it.SessionID,
						LocalRevision: localHash, RemoteRevision: id,
						UpdatedAt: time.Now().UTC().Format(time.RFC3339),
					}
				}
			}
			if !dryRun {
				state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				if len(uploaded) != 0 {
					state.LastManifestRev = uploaded[len(uploaded)-1]
				} else {
					state.LastManifestRev = remoteManifest.Revision
				}
				if err := config.SaveState(home, state); err != nil {
					return err
				}
				if hosted := hostedFrom(cmd); hosted != nil && len(uploaded) != 0 {
					// The first_push product event comes from this report;
					// a failed report never fails a push that completed.
					if err := hosted.ReportFirstPush(context.Background()); err != nil {
						PrintHuman(cmd.ErrOrStderr(), "note: could not report the push to the control plane: %v", err)
					}
				}
			}
			sort.Strings(conflicted)
			conflictErr := func() error {
				if len(conflicted) == 0 {
					return nil
				}
				return NewExitError(ExitConflict, fmt.Sprintf("%d session(s) diverged from the locker; conflict recorded for %s (pushed %d other snapshot(s))",
					len(conflicted), strings.Join(conflicted, ", "), len(uploaded)))
			}
			if asJSON {
				if err := WriteJSON(cmd.OutOrStdout(), map[string]any{
					"snapshots": uploaded, "skipped": skipped, "dry_run": dryRun, "conflicts": conflicted,
				}); err != nil {
					return err
				}
				return conflictErr()
			}
			if dryRun {
				PrintHuman(cmd.OutOrStdout(), "would push %d snapshot(s), would skip %d unchanged, dry_run=true", len(uploaded), skipped)
				return conflictErr()
			}
			PrintHuman(cmd.OutOrStdout(), "pushed %d snapshot(s), skipped %d unchanged, dry_run=%v", len(uploaded), skipped, dryRun)
			return conflictErr()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan only")
	cmd.Flags().StringVar(&agent, "agent", "", "agent filter")
	cmd.Flags().StringVar(&session, "session", "", "session id")
	cmd.Flags().BoolVar(&all, "all", false, "all sessions")
	return cmd
}

func newPullCmd(processChecker AgentProcessChecker) *cobra.Command {
	var asJSON, dryRun, all, allowActiveAgents bool
	var agent, session string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download, decrypt, and restore sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, cfg, home, err := engineFromConfig(cmd, "")
			if err != nil {
				return err
			}
			mutationLock, err := lock.Acquire(home, "mutation")
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			defer func() { _ = mutationLock.Release() }()
			state, err := config.LoadState(home)
			if err != nil {
				return NewExitError(ExitConfig, err.Error())
			}
			man, err := eng.FetchManifest(context.Background())
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			reg := defaultRegistry()
			localSessions := map[string]adapter.Session{}
			for _, selectedAdapter := range reg.All() {
				sessions, discoverErr := selectedAdapter.Discover(context.Background(), adapter.DiscoverOptions{})
				if discoverErr != nil {
					return discoverErr
				}
				for _, localSession := range sessions {
					localSessions[sync.SessionKey(localSession.Agent, localSession.ID)] = localSession
				}
			}
			// Sessions an agent is actively using, which must be restored
			// alongside the live file instead of replacing it.
			forkSessions := map[string]bool{}
			projectRoots := configuredProjectRoots(cfg)
			if !dryRun && (all || session != "") {
				policy := cfg.Restore.ActiveAgentPolicy
				if allowActiveAgents {
					policy = schema.ActiveAgentOff
				}
				for _, remoteSession := range man.Sessions {
					if agent != "" && remoteSession.Agent != agent {
						continue
					}
					if session != "" && remoteSession.SessionID != session {
						continue
					}
					key := sync.SessionKey(remoteSession.Agent, remoteSession.SessionID)
					localSession, exists := localSessions[key]
					if !exists {
						// Nothing to overwrite, so no agent can be disturbed.
						continue
					}
					disposition, err := planSessionRestore(
						cmd.Context(), processChecker, remoteSession.Agent,
						processcheck.Target{
							SessionID:   remoteSession.SessionID,
							Path:        localSession.Path,
							ProjectRoot: projectRoots[localSession.ProjectID],
						}, policy)
					if err != nil {
						return NewExitError(ExitSafety, err.Error())
					}
					if disposition == restoreAsFork {
						forkSessions[key] = true
					}
				}
			}
			type pullPlan struct {
				Agent        string   `json:"agent"`
				SessionID    string   `json:"session_id"`
				SnapshotID   string   `json:"snapshot_id"`
				Destinations []string `json:"destinations"`
				BackupRoot   string   `json:"backup_root"`
				// ForkedSessionID is set when the live session was left alone
				// and the remote copy landed beside it under a new identity.
				ForkedSessionID string `json:"forked_session_id,omitempty"`
			}
			var plans []pullPlan
			var pulled, skipped int
			var conflicted []string
			for _, s := range man.Sessions {
				if agent != "" && s.Agent != agent {
					continue
				}
				if session != "" && s.SessionID != session {
					continue
				}
				if !all && session == "" {
					return NewExitError(ExitUsage, "specify --session or --all")
				}
				key := sync.SessionKey(s.Agent, s.SessionID)
				// A forked restore never replaces the local file, so divergence
				// protection does not apply to it.
				if localSession, exists := localSessions[key]; exists && !forkSessions[key] {
					localAdapter, ok := reg.Get(s.Agent)
					if !ok {
						return NewExitError(ExitCompatibility, "adapter unavailable for "+s.Agent)
					}
					localHash, hashErr := sessionRevision(localAdapter, localSession)
					if hashErr != nil {
						return hashErr
					}
					prior, known := state.Sessions[key]
					if session == "" && known && prior.RemoteRevision == s.SnapshotID && prior.LocalRevision != "" {
						// pull --all asks for what is newer remotely. This
						// snapshot is the one this device last synced, so
						// there is nothing newer to restore, and a local
						// edit since then belongs to the next push, not to
						// a conflict. Restoring anyway would rewrite and
						// back up an identical file on every pull, which
						// the daemon runs every few minutes. An explicit
						// --session still restores (and still records a
						// conflict when the local copy diverged).
						skipped++
						continue
					}
					if !known || prior.LocalRevision == "" || localHash != prior.LocalRevision {
						conflict := sync.Conflict{
							Agent: s.Agent, SessionID: s.SessionID, ProjectID: s.ProjectID,
							LocalRevision: localHash, RemoteRevision: s.SnapshotID,
							RemoteSnapshot: s.SnapshotID,
						}
						if !dryRun {
							if err := sync.SaveConflict(home, conflict); err != nil {
								return err
							}
						}
						if session != "" {
							return NewExitError(ExitConflict, "local session diverged; conflict recorded")
						}
						// pull --all keeps going: one diverged session must
						// not hold back every other session's newer
						// snapshot (the daemon runs this pull on a
						// schedule). The conflicts are reported together.
						conflicted = append(conflicted, key)
						continue
					}
				}
				dest := filepath.Join(home, "cache", "pull", s.SnapshotID)
				env, artifactPath, err := eng.PullArtifact(context.Background(), sync.PullItem{
					Agent: s.Agent, SessionID: s.SessionID, SnapshotID: s.SnapshotID, ProjectID: s.ProjectID,
				}, dest, dryRun)
				if err != nil {
					return err
				}
				a, ok := reg.Get(s.Agent)
				if !ok {
					return NewExitError(ExitCompatibility, "adapter unavailable for "+s.Agent)
				}
				restoreOptions := adapter.RestoreOptions{
					DryRun: dryRun, BackupRoot: filepath.Join(home, "backups"),
				}
				forkedID := ""
				if forkSessions[key] {
					// Derive the fork from the snapshot so re-pulling the same
					// remote state lands on the same file instead of piling up
					// a new copy on every attempt.
					forkedID = forkSessionID("active", s.SessionID, s.SnapshotID)
					restoreOptions.ForkSessionID = forkedID
					restoreOptions.DestinationRelativePath = forkRelativePath(env.Files[0].Path, forkedID)
				}
				restorePlan, err := a.PlanRestore(context.Background(), adapter.Snapshot{
					ID:           env.SnapshotID,
					Agent:        env.Agent,
					SessionID:    env.SessionID,
					ProjectID:    env.ProjectID,
					RelativePath: env.Files[0].Path,
				}, restoreOptions)
				if err != nil {
					return err
				}
				if restorePlan.Refuse != "" {
					return NewExitError(ExitCompatibility, restorePlan.Refuse)
				}
				plans = append(plans, pullPlan{
					Agent: s.Agent, SessionID: s.SessionID, SnapshotID: s.SnapshotID,
					Destinations: restorePlan.Files, BackupRoot: restorePlan.BackupRoot,
					ForkedSessionID: forkedID,
				})
				// A fork's identity is derived from the snapshot, so an existing
				// fork already holds exactly the bytes this restore would write.
				// Rewriting it would back up an identical copy on every repeat
				// and grow the backup directory for no benefit.
				if !dryRun && forkedID != "" && allPathsExist(restorePlan.Files) {
					if err := os.Remove(artifactPath); err != nil && !os.IsNotExist(err) {
						return err
					}
					PrintHuman(cmd.OutOrStdout(),
						"    %s is in use and %s already holds this snapshot; left unchanged",
						s.SessionID, forkedID)
					pulled++
					continue
				}
				if !dryRun {
					artifact, err := os.Open(artifactPath)
					if err != nil {
						return err
					}
					restoreErr := a.Restore(context.Background(), restorePlan, artifact)
					closeErr := artifact.Close()
					if restoreErr != nil {
						return restoreErr
					}
					if closeErr != nil {
						return closeErr
					}
					if err := os.Remove(artifactPath); err != nil && !os.IsNotExist(err) {
						return err
					}
					restored, err := verifyRestoredSession(context.Background(), a, restorePlan)
					if err != nil {
						return fmt.Errorf("verify restored session: %w", err)
					}
					localHash, err := sessionRevision(a, restored)
					if err != nil {
						return err
					}
					// A fork leaves the original session untouched, so it must
					// not be recorded as synchronized with this snapshot.
					if forkedID == "" {
						state.Sessions[key] = schema.SessionState{
							Agent: s.Agent, SessionID: s.SessionID,
							LocalRevision: localHash, RemoteRevision: s.SnapshotID,
							UpdatedAt: time.Now().UTC().Format(time.RFC3339),
						}
					}
				}
				pulled++
			}
			if pulled == 0 && session != "" {
				return NewExitError(ExitUsage, "remote session not found")
			}
			if !dryRun {
				state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				state.LastManifestRev = man.Revision
				if err := config.SaveState(home, state); err != nil {
					return err
				}
			}
			sort.Strings(conflicted)
			conflictErr := func() error {
				if len(conflicted) == 0 {
					return nil
				}
				return NewExitError(ExitConflict, fmt.Sprintf("%d session(s) diverged locally; conflict recorded for %s (pulled %d other snapshot(s))",
					len(conflicted), strings.Join(conflicted, ", "), pulled))
			}
			if asJSON {
				if err := WriteJSON(cmd.OutOrStdout(), map[string]any{
					"pulled": pulled, "skipped": skipped, "dry_run": dryRun, "plans": plans, "conflicts": conflicted,
				}); err != nil {
					return err
				}
				return conflictErr()
			}
			if dryRun {
				PrintHuman(cmd.OutOrStdout(), "would pull %d snapshot(s), would skip %d already synced, dry_run=true", pulled, skipped)
			} else {
				PrintHuman(cmd.OutOrStdout(), "pulled %d snapshot(s), skipped %d already synced, dry_run=false", pulled, skipped)
			}
			for _, plan := range plans {
				PrintHuman(cmd.OutOrStdout(), "  %s:%s -> %s (backups: %s)",
					plan.Agent, plan.SessionID, strings.Join(plan.Destinations, ", "), plan.BackupRoot)
				if plan.ForkedSessionID != "" {
					PrintHuman(cmd.OutOrStdout(),
						"    %s is in use, so it was left unchanged; restored alongside it as %s",
						plan.SessionID, plan.ForkedSessionID)
				}
			}
			return conflictErr()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan only")
	cmd.Flags().StringVar(&agent, "agent", "", "agent filter")
	cmd.Flags().StringVar(&session, "session", "", "session id")
	cmd.Flags().BoolVar(&all, "all", false, "all sessions")
	cmd.Flags().BoolVar(&allowActiveAgents, "allow-active-agents", false,
		"restore even if an agent is using the target session (still atomic and still backed up)")
	return cmd
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hash, _, err := crypto.SHA256Reader(file)
	return hash, err
}

func newConflictsCmd(processChecker AgentProcessChecker) *cobra.Command {
	root := &cobra.Command{Use: "conflicts", Short: "List and resolve sync conflicts"}
	var asJSON bool
	list := &cobra.Command{
		Use:   "list",
		Short: "List conflicts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := configuredHome()
			if err != nil {
				return err
			}
			cs, err := sync.ListConflicts(home)
			if err != nil {
				return err
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), cs)
			}
			for _, c := range cs {
				PrintHuman(cmd.OutOrStdout(), "%s %s %s local=%s remote=%s", c.ID, c.Agent, c.SessionID, c.LocalRevision, c.RemoteRevision)
			}
			return nil
		},
	}
	list.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	show := &cobra.Command{
		Use:  "show <id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := configuredHome()
			if err != nil {
				return err
			}
			cs, err := sync.ListConflicts(home)
			if err != nil {
				return err
			}
			for _, c := range cs {
				if c.ID == args[0] {
					return WriteJSON(cmd.OutOrStdout(), c)
				}
			}
			return NewExitError(ExitUsage, "conflict not found")
		},
	}
	var keepLocal, keepRemote, keepBoth, allowActiveAgents bool
	resolve := &cobra.Command{
		Use:  "resolve <id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, cfg, home, err := engineFromConfig(cmd, "")
			if err != nil {
				return err
			}
			var how sync.Resolution
			selectedStrategies := 0
			for _, selected := range []bool{keepLocal, keepRemote, keepBoth} {
				if selected {
					selectedStrategies++
				}
			}
			if selectedStrategies != 1 {
				return NewExitError(ExitUsage, "specify exactly one of --keep-local|--keep-remote|--keep-both")
			}
			switch {
			case keepLocal:
				how = sync.KeepLocal
			case keepRemote:
				how = sync.KeepRemote
			case keepBoth:
				how = sync.KeepBoth
			}
			if how == sync.KeepRemote {
				conflict, err := sync.GetConflict(home, args[0])
				if err != nil {
					return NewExitError(ExitUsage, err.Error())
				}
				policy := cfg.Restore.ActiveAgentPolicy
				if allowActiveAgents {
					policy = schema.ActiveAgentOff
				}
				if err := requireSessionRestorable(
					cmd.Context(), processChecker, conflict.Agent,
					processcheck.Target{
						SessionID:   conflict.SessionID,
						Path:        localSessionPath(conflict.Agent, conflict.SessionID),
						ProjectRoot: configuredProjectRoots(cfg)[conflict.ProjectID],
					}, policy,
				); err != nil {
					return NewExitError(ExitSafety, err.Error())
				}
			}
			mutationLock, err := lock.Acquire(home, "mutation")
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			defer func() { _ = mutationLock.Release() }()
			registry := defaultRegistry()
			if err := sync.Resolve(home, args[0], how, func(conflict sync.Conflict, strategy sync.Resolution) error {
				switch strategy {
				case sync.KeepLocal:
					return resolveKeepLocal(context.Background(), eng, home, registry, conflict)
				case sync.KeepRemote:
					return resolveKeepRemote(context.Background(), eng, home, registry, conflict, false)
				case sync.KeepBoth:
					return resolveKeepRemote(context.Background(), eng, home, registry, conflict, true)
				default:
					return fmt.Errorf("invalid resolution")
				}
			}); err != nil {
				return NewExitError(ExitConflict, err.Error())
			}
			PrintHuman(cmd.OutOrStdout(), "resolved %s via %s", args[0], how)
			return nil
		},
	}
	resolve.Flags().BoolVar(&keepLocal, "keep-local", false, "keep local")
	resolve.Flags().BoolVar(&keepRemote, "keep-remote", false, "keep remote")
	resolve.Flags().BoolVar(&keepBoth, "keep-both", false, "keep both")
	resolve.Flags().BoolVar(&allowActiveAgents, "allow-active-agents", false,
		"resolve even if an agent is using the target session (still atomic and still backed up)")
	root.AddCommand(list, show, resolve)
	return root
}

func configuredHome() (string, error) {
	home, err := config.Home()
	if err != nil {
		return "", NewExitError(ExitConfig, err.Error())
	}
	if _, err := config.LoadConfig(home); err != nil {
		return "", configLoadExitError(err)
	}
	return home, nil
}

func configLoadExitError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return NewExitError(ExitConfig, "config missing")
	}
	return NewExitError(ExitConfig, err.Error())
}

func resolveKeepLocal(
	ctx context.Context,
	engine *sync.Engine,
	home string,
	registry *adapter.Registry,
	conflict sync.Conflict,
) error {
	selected, err := discoverSession(ctx, registry, conflict.Agent, conflict.SessionID)
	if err != nil {
		return err
	}
	selectedAdapter, ok := registry.Get(conflict.Agent)
	if !ok {
		return fmt.Errorf("adapter unavailable for %s", conflict.Agent)
	}
	plan, err := selectedAdapter.PlanExport(ctx, selected, adapter.ExportOptions{})
	if err != nil {
		return err
	}
	exportDir := filepath.Join(home, "cache", "exports")
	if err := os.MkdirAll(exportDir, 0o700); err != nil {
		return err
	}
	exportFile, err := os.CreateTemp(exportDir, ".reinstate-resolution-*")
	if err != nil {
		return err
	}
	exportPath := exportFile.Name()
	defer func() {
		_ = exportFile.Close()
		_ = os.Remove(exportPath)
	}()
	if err := selectedAdapter.Export(ctx, plan, exportFile); err != nil {
		return err
	}
	if err := exportFile.Close(); err != nil {
		return err
	}
	manifest, err := engine.FetchManifest(ctx)
	if err != nil {
		return err
	}
	remote, _ := sync.ManifestSessionLookup(manifest, conflict.Agent, conflict.SessionID)
	snapshotID, err := engine.PushSession(ctx, sync.PushItem{
		Agent: conflict.Agent, SessionID: conflict.SessionID, ProjectID: selected.ProjectID,
		LocalPath: exportPath, RelativePath: selected.RelativePath,
		BaseKnown: true, BaseRevision: remote.SnapshotID,
	}, false)
	if err != nil {
		return err
	}
	localHash, err := sessionRevision(selectedAdapter, selected)
	if err != nil {
		return err
	}
	return updateSessionState(home, selected.Agent, selected.ID, localHash, snapshotID)
}

func resolveKeepRemote(
	ctx context.Context,
	engine *sync.Engine,
	home string,
	registry *adapter.Registry,
	conflict sync.Conflict,
	keepBoth bool,
) error {
	manifest, err := engine.FetchManifest(ctx)
	if err != nil {
		return err
	}
	remote, ok := sync.ManifestSessionLookup(manifest, conflict.Agent, conflict.SessionID)
	if !ok {
		return fmt.Errorf("remote session no longer exists")
	}
	dest := filepath.Join(home, "cache", "pull", remote.SnapshotID)
	env, artifactPath, err := engine.PullArtifact(ctx, sync.PullItem{
		Agent: remote.Agent, SessionID: remote.SessionID,
		SnapshotID: remote.SnapshotID, ProjectID: remote.ProjectID,
	}, dest, false)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(artifactPath) }()
	selectedAdapter, ok := registry.Get(remote.Agent)
	if !ok {
		return fmt.Errorf("adapter unavailable for %s", remote.Agent)
	}
	options := adapter.RestoreOptions{BackupRoot: filepath.Join(home, "backups")}
	if keepBoth {
		targetID := forkSessionID("remote", remote.SessionID, remote.SnapshotID)
		options.ForkSessionID = targetID
		options.DestinationRelativePath = forkRelativePath(env.Files[0].Path, targetID)
	}
	restorePlan, err := selectedAdapter.PlanRestore(ctx, adapter.Snapshot{
		ID: env.SnapshotID, Agent: env.Agent, SessionID: env.SessionID,
		ProjectID: env.ProjectID, RelativePath: env.Files[0].Path,
	}, options)
	if err != nil {
		return err
	}
	if restorePlan.Refuse != "" {
		return fmt.Errorf("%s", restorePlan.Refuse)
	}
	artifact, err := os.Open(artifactPath)
	if err != nil {
		return err
	}
	restoreErr := selectedAdapter.Restore(ctx, restorePlan, artifact)
	closeErr := artifact.Close()
	if restoreErr != nil {
		return restoreErr
	}
	if closeErr != nil {
		return closeErr
	}
	restored, err := verifyRestoredSession(ctx, selectedAdapter, restorePlan)
	if err != nil {
		return fmt.Errorf("verify resolved session: %w", err)
	}
	localHash, err := sessionRevision(selectedAdapter, restored)
	if err != nil {
		return err
	}
	return updateSessionState(home, restored.Agent, restored.ID, localHash, remote.SnapshotID)
}

func verifyRestoredSession(ctx context.Context, selectedAdapter adapter.Adapter, plan adapter.RestorePlan) (adapter.Session, error) {
	discovered, err := selectedAdapter.Discover(ctx, adapter.DiscoverOptions{})
	if err != nil {
		return adapter.Session{}, err
	}
	expectedPath := filepath.Clean(plan.Session.Path)
	expectedRelative := filepath.ToSlash(plan.Session.RelativePath)
	for _, session := range discovered {
		if session.ID != plan.Session.ID ||
			filepath.Clean(session.Path) != expectedPath ||
			(expectedRelative != "" && filepath.ToSlash(session.RelativePath) != expectedRelative) {
			continue
		}
		return session, nil
	}
	return adapter.Session{}, fmt.Errorf(
		"adapter could not discover restored %s session %s at planned destination %s",
		plan.Session.Agent,
		plan.Session.ID,
		plan.Session.Path,
	)
}

func discoverSession(ctx context.Context, registry *adapter.Registry, agentName, sessionID string) (adapter.Session, error) {
	selectedAdapter, ok := registry.Get(agentName)
	if !ok {
		return adapter.Session{}, fmt.Errorf("adapter unavailable for %s", agentName)
	}
	sessions, err := selectedAdapter.Discover(ctx, adapter.DiscoverOptions{})
	if err != nil {
		return adapter.Session{}, err
	}
	for _, session := range sessions {
		if session.ID == sessionID {
			return session, nil
		}
	}
	return adapter.Session{}, fmt.Errorf("%s session %s not found", agentName, sessionID)
}

// localSessionPath resolves the on-disk path of an existing local session.
// An empty result means the session is not present locally, so a restore has
// nothing to overwrite.
func localSessionPath(agent, sessionID string) string {
	for _, selectedAdapter := range defaultRegistry().All() {
		sessions, err := selectedAdapter.Discover(context.Background(), adapter.DiscoverOptions{})
		if err != nil {
			continue
		}
		for _, session := range sessions {
			if session.Agent == agent && session.ID == sessionID {
				return session.Path
			}
		}
	}
	return ""
}

// configuredProjectRoots maps canonical project IDs to their local roots so a
// liveness check can tell whether an agent is working in the same project.
func configuredProjectRoots(cfg *schema.Config) map[string]string {
	roots := map[string]string{}
	if cfg == nil {
		return roots
	}
	for _, project := range cfg.Projects {
		roots[project.ID] = project.LocalRoot
	}
	return roots
}

// restoreDisposition says how one restore target must be handled.
type restoreDisposition int

const (
	// restoreInPlace replaces the existing session file.
	restoreInPlace restoreDisposition = iota
	// restoreAsFork lands the remote copy beside a session that is in use.
	restoreAsFork
)

// planSessionRestore applies the active-agent policy to a single target.
//
// The liveness question is scoped to this session, so unrelated agents working
// in other projects never affect a restore. Under the default policy even a
// session that is genuinely in use does not block: the remote copy is restored
// alongside the live session instead.
func planSessionRestore(
	ctx context.Context, checker AgentProcessChecker, agent string,
	target processcheck.Target, policy string,
) (restoreDisposition, error) {
	switch policy {
	case schema.ActiveAgentOff:
		return restoreInPlace, nil
	case schema.ActiveAgentStrict:
		// Strict deliberately discards the target so the check stays host-wide.
		target = processcheck.Target{}
	case schema.ActiveAgentScoped, schema.ActiveAgentFork, "":
	default:
		return restoreInPlace, fmt.Errorf("unsupported restore.active_agent_policy %q", policy)
	}

	checkContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	busy, scoped, err := checker(checkContext, agent, target)
	if err != nil {
		return restoreInPlace, fmt.Errorf(
			"cannot verify whether %s is using this session: %w", agent, err)
	}
	if !busy {
		return restoreInPlace, nil
	}
	if policy == schema.ActiveAgentFork || policy == "" {
		return restoreAsFork, nil
	}
	if scoped {
		return restoreInPlace, fmt.Errorf(
			"%s is currently using this session; close that session or rerun with --allow-active-agents",
			agent)
	}
	return restoreInPlace, fmt.Errorf(
		"%s appears to be running and this host cannot tell which session it is using; "+
			"close it or rerun with --allow-active-agents", agent)
}

// requireSessionRestorable enforces the policy where forking is not an option.
//
// Conflict resolution already exposes forking explicitly through --keep-both,
// so the fork policy is treated as scoped here rather than silently changing
// what --keep-remote means.
func requireSessionRestorable(
	ctx context.Context, checker AgentProcessChecker, agent string,
	target processcheck.Target, policy string,
) error {
	if policy == schema.ActiveAgentFork || policy == "" {
		policy = schema.ActiveAgentScoped
	}
	_, err := planSessionRestore(ctx, checker, agent, target, policy)
	return err
}

func forkRelativePath(source, sessionID string) string {
	slashSource := filepath.ToSlash(source)
	// Keep the source's own extension so an embedded-store agent whose sessions
	// are addressed as ".json" does not have a fork mislabelled ".jsonl".
	ext := path.Ext(slashSource)
	if ext == "" {
		ext = ".jsonl"
	}
	dir := path.Dir(slashSource)
	if dir == "." {
		return sessionID + ext
	}
	return dir + "/" + sessionID + ext
}

// sessionRevision returns a stable per-session content revision. Embedded-store
// adapters implement adapter.SessionRevisioner because their sessions do not
// each own a file to hash; a file-per-session adapter falls back to hashing the
// session's file.
func sessionRevision(a adapter.Adapter, s adapter.Session) (string, error) {
	if revisioner, ok := a.(adapter.SessionRevisioner); ok {
		return revisioner.SessionRevision(context.Background(), s)
	}
	return hashFile(s.Path)
}

// allPathsExist reports whether every path is already present on disk.
func allPathsExist(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

// forkNamespace scopes the deterministic fork identifiers below. It is a fixed
// random UUID and carries no meaning beyond keeping fork names in their own
// space.
var forkNamespace = uuid.MustParse("6f9619ff-8b86-d011-b42d-00c04fc964ff")

// forkSessionID derives a fork identity from the session and snapshot it came
// from.
//
// The result is a real UUID. Vendors treat session identifiers as UUIDs, and a
// decorated form such as "<uuid>-remote-<short>" is accepted by Claude Code's
// interactive resume but rejected by `claude --print --resume`, which leaves a
// fork a human can open and automation cannot. Deriving the value keeps repeated
// restores of the same snapshot idempotent, which a random UUID would not.
func forkSessionID(kind, sessionID, snapshotID string) string {
	return uuid.NewSHA1(forkNamespace, []byte(kind+"\x00"+sessionID+"\x00"+snapshotID)).String()
}

func updateSessionState(home, agentName, sessionID, localRevision, remoteRevision string) error {
	state, err := config.LoadState(home)
	if err != nil {
		return err
	}
	state.Sessions[sync.SessionKey(agentName, sessionID)] = schema.SessionState{
		Agent: agentName, SessionID: sessionID,
		LocalRevision: localRevision, RemoteRevision: remoteRevision,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return config.SaveState(home, state)
}
