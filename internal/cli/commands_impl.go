package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	"github.com/HarjjotSinghh/reinstate/internal/sync"
)

func defaultRegistry() *adapter.Registry {
	r := adapter.NewRegistry()
	_ = r.Register(&claude.Adapter{})
	_ = r.Register(&codex.Adapter{})
	return r
}

func newInitCmd() *cobra.Command {
	var (
		endpoint, bucket, region, prefix string
		accessKey, secretKey             string
		nonInteractive                   bool
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
			profileID := uuid.NewString()
			deviceID := uuid.NewString()
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
			if accessKey == "" {
				accessKey = os.Getenv("REINSTATE_S3_ACCESS_KEY_ID")
			}
			if secretKey == "" {
				secretKey = os.Getenv("REINSTATE_S3_SECRET_ACCESS_KEY")
			}
			if !nonInteractive && (endpoint == "" || bucket == "") {
				return NewExitError(ExitUsage, "init requires --endpoint and --bucket (or env), and storage keys via env/keyring; passphrase is prompted by encrypt ops, never CLI args")
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
			if accessKey != "" && secretKey != "" {
				store := credentials.NewFileStore(filepath.Join(home, "credentials"))
				if err := store.Set(credRef, credentials.StorageCredentials{AccessKeyID: accessKey, SecretAccessKey: secretKey}); err != nil {
					return NewExitError(ExitAuthStorage, err.Error())
				}
			}
			if err := config.SaveConfig(home, cfg); err != nil {
				return err
			}
			if err := config.SaveState(home, schema.NewState()); err != nil {
				return err
			}
			// Optional probe when credentials present (real S3 backend only)
			if accessKey != "" && secretKey != "" && endpoint != "" && bucket != "" && os.Getenv("REINSTATE_BACKEND") != "memory" {
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
					PrintHuman(cmd.ErrOrStderr(), "warning: storage probe put failed: %v", err)
				} else if err := client.Delete(ctx, probe); err != nil {
					PrintHuman(cmd.ErrOrStderr(), "warning: storage probe delete failed: %v", err)
				}
			}
			PrintHuman(cmd.OutOrStdout(), "initialized reinstate home (config.toml + state.json). Passphrase is not stored; you will enter it on push/pull.")
			PrintHuman(cmd.OutOrStdout(), "home configured; credential_ref=%s", credRef)
			return nil
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "S3/R2 endpoint URL")
	cmd.Flags().StringVar(&bucket, "bucket", "", "bucket name")
	cmd.Flags().StringVar(&region, "region", "auto", "region")
	cmd.Flags().StringVar(&prefix, "prefix", "", "object key prefix")
	cmd.Flags().StringVar(&accessKey, "access-key", "", "access key id (prefer env REINSTATE_S3_ACCESS_KEY_ID)")
	cmd.Flags().StringVar(&secretKey, "secret-key", "", "secret access key (prefer env)")
	cmd.Flags().BoolVar(&nonInteractive, "yes", false, "allow missing optional fields when env provides them")
	return cmd
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

func engineFromConfig(passphrase string) (*sync.Engine, *schema.Config, string, error) {
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
	if os.Getenv("REINSTATE_BACKEND") == "memory" {
		disk, err := memory.NewDisk(filepath.Join(home, "cache", "memory-backend"))
		if err != nil {
			return nil, nil, "", NewExitError(ExitRuntime, err.Error())
		}
		b = disk
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
		passphrase = os.Getenv("REINSTATE_PASSPHRASE")
	}
	if passphrase == "" {
		return nil, nil, "", NewExitError(ExitUsage, "set REINSTATE_PASSPHRASE for non-interactive encrypt (TTY prompt not available in this build path)")
	}
	return &sync.Engine{Backend: b, Passphrase: passphrase}, cfg, home, nil
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Compare local vs remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, _, err := engineFromConfig("")
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
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show pending change metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			// metadata only — list local sessions vs remote keys
			reg := defaultRegistry()
			var local []string
			for _, a := range reg.All() {
				ss, _ := a.Discover(context.Background(), adapter.DiscoverOptions{})
				for _, s := range ss {
					local = append(local, sync.SessionKey(s.Agent, s.ID))
				}
			}
			eng, _, _, err := engineFromConfig("")
			remote := []string{}
			if err == nil {
				if man, merr := eng.FetchManifest(context.Background()); merr == nil {
					for k := range man.Sessions {
						remote = append(remote, k)
					}
				}
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
	cmd.Flags().String("agent", "", "agent filter")
	cmd.Flags().String("session", "", "session id")
	return cmd
}

func newPushCmd() *cobra.Command {
	var asJSON, dryRun, all bool
	var agent, session string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Encrypt and upload local sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, _, err := engineFromConfig("")
			if err != nil {
				return err
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
					items = append(items, sync.PushItem{Agent: s.Agent, SessionID: s.ID, ProjectID: s.ProjectID, LocalPath: s.Path})
				}
			}
			if !all && session == "" && len(items) > 1 {
				return NewExitError(ExitUsage, "specify --session or --all")
			}
			var uploaded []string
			for _, it := range items {
				id, err := eng.PushSession(context.Background(), it, dryRun)
				if err != nil {
					if strings.Contains(err.Error(), "credential") {
						return NewExitError(ExitSafety, err.Error())
					}
					return err
				}
				uploaded = append(uploaded, id)
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]any{"snapshots": uploaded, "dry_run": dryRun})
			}
			PrintHuman(cmd.OutOrStdout(), "pushed %d snapshot(s) dry_run=%v", len(uploaded), dryRun)
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

func newPullCmd() *cobra.Command {
	var asJSON, dryRun, all bool
	var agent, session string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download, decrypt, and restore sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, home, err := engineFromConfig("")
			if err != nil {
				return err
			}
			man, err := eng.FetchManifest(context.Background())
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
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
				dest := filepath.Join(home, "cache", "pull", s.SnapshotID)
				_, _, err := eng.PullSession(context.Background(), sync.PullItem{
					Agent: s.Agent, SessionID: s.SessionID, SnapshotID: s.SnapshotID, ProjectID: s.ProjectID,
				}, dest, dryRun)
				if err != nil {
					return err
				}
				pulled++
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]any{"pulled": pulled, "dry_run": dryRun})
			}
			PrintHuman(cmd.OutOrStdout(), "pulled %d snapshot(s) dry_run=%v", pulled, dryRun)
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

func newConflictsCmd() *cobra.Command {
	root := &cobra.Command{Use: "conflicts", Short: "List and resolve sync conflicts"}
	var asJSON bool
	list := &cobra.Command{
		Use:   "list",
		Short: "List conflicts",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := config.Home()
			if err != nil {
				return NewExitError(ExitConfig, err.Error())
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
			home, err := config.Home()
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
			home, err := config.Home()
			if err != nil {
				return err
			}
			var how sync.Resolution
			switch {
			case keepLocal:
				how = sync.KeepLocal
			case keepRemote:
				how = sync.KeepRemote
			case keepBoth:
				how = sync.KeepBoth
			default:
				return NewExitError(ExitUsage, "specify --keep-local|--keep-remote|--keep-both")
			}
			if err := sync.Resolve(home, args[0], how); err != nil {
				return err
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
