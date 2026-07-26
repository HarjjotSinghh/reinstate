package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/codex"
	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/lock"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	"github.com/HarjjotSinghh/reinstate/internal/sync"
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
	_ = r.Register(&claude.Adapter{Home: userHome, Projects: projects})
	_ = r.Register(&codex.Adapter{Home: userHome, Projects: projects})
	return r
}

func newInitCmd() *cobra.Command {
	var (
		endpoint, bucket, region, prefix, configuredProfileID string
		projectMappings                                       []string
		nonInteractive                                        bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup (backend, encryption, path map)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := config.Home()
			if err != nil {
				return NewExitError(ExitConfig, err.Error())
			}
			if err := config.EnsureLayout(home); err != nil {
				return err
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
			for _, mapping := range projectMappings {
				project, parseErr := parseProjectMapping(mapping)
				if parseErr != nil {
					return NewExitError(ExitUsage, parseErr.Error())
				}
				cfg.Projects = append(cfg.Projects, project)
			}
			// A failed backend probe is fatal. Do not leave a config that looks
			// initialized but cannot reach storage.
			if os.Getenv("REINSTATE_BACKEND") != "memory" {
				ctx := context.Background()
				client, err := s3.New(ctx, s3.Config{
					Endpoint: endpoint, Region: region, Bucket: bucket, Prefix: cfg.Storage.Prefix,
					AccessKey: accessKey, SecretKey: secretKey,
				})
				if err != nil {
					return NewExitError(ExitAuthStorage, err.Error())
				}
				probe := "probes/" + uuid.NewString()
				if _, err := client.Put(ctx, probe, strings.NewReader("ok"), 2, backend.PutOptions{IfNoneMatch: true}); err != nil {
					return NewExitError(ExitAuthStorage, "storage probe put failed: "+err.Error())
				} else if err := client.Delete(ctx, probe); err != nil {
					return NewExitError(ExitAuthStorage, "storage probe cleanup failed: "+err.Error())
				}
			}
			keyringStore := credentials.NewKeyringStore()
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
	return cmd
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
	cmd.Flags().StringVar(&agent, "agent", "all", "agent filter: claude|codex|all")
	return cmd
}

func engineFromConfig(cmd *cobra.Command, passphrase string) (*sync.Engine, *schema.Config, string, error) {
	home, err := config.Home()
	if err != nil {
		return nil, nil, "", NewExitError(ExitConfig, err.Error())
	}
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return nil, nil, "", NewExitError(ExitConfig, err.Error())
	}
	// Backend selection: disk-backed "memory" for local e2e, else S3.
	var b backend.Backend
	enginePrefix := ""
	if os.Getenv("REINSTATE_BACKEND") == "memory" {
		disk, err := memory.NewDisk(filepath.Join(home, "cache", "memory-backend"))
		if err != nil {
			return nil, nil, "", NewExitError(ExitRuntime, err.Error())
		}
		b = disk
		enginePrefix = cfg.Storage.Prefix
	} else {
		creds, err := credentials.Resolve(home, cfg.Storage.CredentialRef)
		if err != nil {
			return nil, nil, "", NewExitError(ExitAuthStorage, err.Error())
		}
		client, err := s3.New(context.Background(), s3.Config{
			Endpoint: cfg.Storage.Endpoint, Region: cfg.Storage.Region,
			Bucket: cfg.Storage.Bucket, Prefix: cfg.Storage.Prefix,
			AccessKey: creds.AccessKeyID, SecretKey: creds.SecretAccessKey,
		})
		if err != nil {
			return nil, nil, "", NewExitError(ExitAuthStorage, err.Error())
		}
		b = client
	}
	if passphrase == "" {
		secret, err := crypto.ReadPassphrase(cmd.InOrStdin(), cmd.ErrOrStderr())
		if err != nil {
			return nil, nil, "", NewExitError(ExitUsage, err.Error())
		}
		passphrase = string(secret)
		crypto.Zero(secret)
	}
	return &sync.Engine{Backend: b, Passphrase: passphrase, Prefix: enginePrefix}, cfg, home, nil
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
				return err
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
				return NewExitError(ExitAuthStorage, err.Error())
			}
			var uploaded []string
			var skipped int
			for _, it := range items {
				localHash, err := hashFile(it.LocalPath)
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
						_ = sync.SaveConflict(home, sync.Conflict{
							Agent: it.Agent, SessionID: it.SessionID, ProjectID: it.ProjectID,
							LocalRevision: it.BaseRevision, RemoteRevision: remoteSnapshot,
							RemoteSnapshot: remoteSnapshot,
						})
						return NewExitError(ExitConflict, err.Error())
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
				if err := config.SaveState(home, state); err != nil {
					return err
				}
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]any{
					"snapshots": uploaded, "skipped": skipped, "dry_run": dryRun,
				})
			}
			if dryRun {
				PrintHuman(cmd.OutOrStdout(), "would push %d snapshot(s), would skip %d unchanged, dry_run=true", len(uploaded), skipped)
				return nil
			}
			PrintHuman(cmd.OutOrStdout(), "pushed %d snapshot(s), skipped %d unchanged, dry_run=%v", len(uploaded), skipped, dryRun)
			return nil
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
	var asJSON, dryRun, all bool
	var agent, session string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download, decrypt, and restore sessions",
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
			if !dryRun && (all || session != "") {
				checkedAgents := map[string]bool{}
				for _, remoteSession := range man.Sessions {
					if agent != "" && remoteSession.Agent != agent {
						continue
					}
					if session != "" && remoteSession.SessionID != session {
						continue
					}
					key := sync.SessionKey(remoteSession.Agent, remoteSession.SessionID)
					if _, exists := localSessions[key]; !exists || checkedAgents[remoteSession.Agent] {
						continue
					}
					if err := requireAgentInactive(cmd.Context(), processChecker, remoteSession.Agent); err != nil {
						return NewExitError(ExitSafety, err.Error())
					}
					checkedAgents[remoteSession.Agent] = true
				}
			}
			type pullPlan struct {
				Agent        string   `json:"agent"`
				SessionID    string   `json:"session_id"`
				SnapshotID   string   `json:"snapshot_id"`
				Destinations []string `json:"destinations"`
				BackupRoot   string   `json:"backup_root"`
			}
			var plans []pullPlan
			var pulled int
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
				if localSession, exists := localSessions[key]; exists {
					localHash, hashErr := hashFile(localSession.Path)
					if hashErr != nil {
						return hashErr
					}
					prior, known := state.Sessions[key]
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
						return NewExitError(ExitConflict, "local session diverged; conflict recorded")
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
				restorePlan, err := a.PlanRestore(context.Background(), adapter.Snapshot{
					ID:           env.SnapshotID,
					Agent:        env.Agent,
					SessionID:    env.SessionID,
					ProjectID:    env.ProjectID,
					RelativePath: env.Files[0].Path,
				}, adapter.RestoreOptions{
					DryRun: dryRun, BackupRoot: filepath.Join(home, "backups"),
				})
				if err != nil {
					return err
				}
				if restorePlan.Refuse != "" {
					return NewExitError(ExitCompatibility, restorePlan.Refuse)
				}
				plans = append(plans, pullPlan{
					Agent: s.Agent, SessionID: s.SessionID, SnapshotID: s.SnapshotID,
					Destinations: restorePlan.Files, BackupRoot: restorePlan.BackupRoot,
				})
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
					localHash, err := hashFile(restored.Path)
					if err != nil {
						return err
					}
					state.Sessions[key] = schema.SessionState{
						Agent: s.Agent, SessionID: s.SessionID,
						LocalRevision: localHash, RemoteRevision: s.SnapshotID,
						UpdatedAt: time.Now().UTC().Format(time.RFC3339),
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
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]any{
					"pulled": pulled, "dry_run": dryRun, "plans": plans,
				})
			}
			PrintHuman(cmd.OutOrStdout(), "pulled %d snapshot(s) dry_run=%v", pulled, dryRun)
			for _, plan := range plans {
				PrintHuman(cmd.OutOrStdout(), "  %s:%s -> %s (backups: %s)",
					plan.Agent, plan.SessionID, strings.Join(plan.Destinations, ", "), plan.BackupRoot)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan only")
	cmd.Flags().StringVar(&agent, "agent", "", "agent filter")
	cmd.Flags().StringVar(&session, "session", "", "session id")
	cmd.Flags().BoolVar(&all, "all", false, "all sessions")
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
	var keepLocal, keepRemote, keepBoth bool
	resolve := &cobra.Command{
		Use:  "resolve <id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, home, err := engineFromConfig(cmd, "")
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
				if err := requireAgentInactive(cmd.Context(), processChecker, conflict.Agent); err != nil {
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
	root.AddCommand(list, show, resolve)
	return root
}

func configuredHome() (string, error) {
	home, err := config.Home()
	if err != nil {
		return "", NewExitError(ExitConfig, err.Error())
	}
	if _, err := config.LoadConfig(home); err != nil {
		return "", NewExitError(ExitConfig, err.Error())
	}
	return home, nil
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
	localHash, err := hashFile(selected.Path)
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
		targetID := remote.SessionID + "-remote-" + shortID(remote.SnapshotID)
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
	localHash, err := hashFile(restored.Path)
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

func requireAgentInactive(ctx context.Context, checker AgentProcessChecker, agent string) error {
	checkContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	active, err := checker(checkContext, agent)
	if err != nil {
		return fmt.Errorf("cannot verify that %s is inactive: %w", agent, err)
	}
	if active {
		return fmt.Errorf("%s appears to be running; close it before restoring sessions", agent)
	}
	return nil
}

func forkRelativePath(source, sessionID string) string {
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(source)))
	if dir == "." {
		return sessionID + ".jsonl"
	}
	return dir + "/" + sessionID + ".jsonl"
}

func shortID(value string) string {
	if len(value) > 8 {
		return value[:8]
	}
	return value
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
