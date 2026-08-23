package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/daemon"
	"github.com/HarjjotSinghh/reinstate/internal/filelock"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// daemonSeams are the deterministic seams of `rein daemon`. Production
// leaves every field nil.
type daemonSeams struct {
	// manager overrides the OS service manager.
	manager daemon.Manager
	// runner overrides the command runner behind the real manager.
	runner daemon.Runner
	// notifier overrides OS notifications.
	notifier daemon.Notifier
	// clock overrides the loop's clock.
	clock daemon.Clock
	// events overrides the filesystem watcher.
	events <-chan daemon.Change
	// observe sees every loop event.
	observe func(daemon.Event)
	// executable overrides os.Executable for install.
	executable string
}

type daemonSeamsContextKey struct{}

// rootOptionsContextKey carries the Options the root command was built
// with, so a command can run another command in-process with the same
// seams (the resume hook runs pull this way).
type rootOptionsContextKey struct{}

func rootOptionsFrom(cmd *cobra.Command) (Options, bool) {
	if ctx := cmd.Context(); ctx != nil {
		if o, ok := ctx.Value(rootOptionsContextKey{}).(Options); ok {
			return o, true
		}
	}
	return Options{}, false
}

func daemonSeamsFrom(cmd *cobra.Command) daemonSeams {
	if ctx := cmd.Context(); ctx != nil {
		if s, ok := ctx.Value(daemonSeamsContextKey{}).(daemonSeams); ok {
			return s
		}
	}
	return daemonSeams{}
}

// daemonRunFlags are the loop settings `rein daemon run` accepts and
// `rein daemon install` bakes into the service definition.
type daemonRunFlags struct {
	pullEvery time.Duration
	debounce  time.Duration
	poll      bool
	home      string
	// env is install-only: extra KEY=VALUE pairs for the service.
	env []string
}

func (f *daemonRunFlags) bind(cmd *cobra.Command) {
	cmd.Flags().DurationVar(&f.pullEvery, "pull-every", daemon.DefaultPullEvery, "how often to pull remote sessions")
	cmd.Flags().DurationVar(&f.debounce, "debounce", daemon.DefaultDebounce, "quiet period after a session change before the push")
	cmd.Flags().BoolVar(&f.poll, "poll", false, "scan the session stores on a timer instead of watching them")
	cmd.Flags().StringVar(&f.home, "home", "", "Reinstate home to serve (the scheduled task on Windows passes it this way)")
}

// args renders the flags back for the service definition, omitting
// defaults so the definition stays readable.
func (f daemonRunFlags) args() []string {
	out := []string{"daemon", "run"}
	if f.pullEvery > 0 && f.pullEvery != daemon.DefaultPullEvery {
		out = append(out, "--pull-every", f.pullEvery.String())
	}
	if f.debounce > 0 && f.debounce != daemon.DefaultDebounce {
		out = append(out, "--debounce", f.debounce.String())
	}
	if f.poll {
		out = append(out, "--poll")
	}
	return out
}

func newDaemonCmd(opts Options) *cobra.Command {
	root := &cobra.Command{
		Use:   "daemon",
		Short: "Keep sessions synced in the background and surface device approvals",
		Long: `The daemon is a resident per-device process. It watches every detected
agent's session store and pushes after a session changes, pulls on a
schedule so the local index stays fresh, and polls the control plane for
devices waiting to join the account. It works the same on BYO storage and
on Reinstate Hop, and sends nothing that push and pull do not already send.

rein daemon install registers it with launchd (macOS), Task Scheduler
(Windows), or systemd --user (Linux) so it starts at login. rein daemon run
is the foreground loop the service runs; use it under your own supervisor
or to watch it work.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	root.AddCommand(
		newDaemonRunCmd(opts),
		newDaemonInstallCmd(),
		newDaemonUninstallCmd(),
		newDaemonStartCmd(),
		newDaemonStopCmd(),
		newDaemonStatusCmd(),
	)
	return root
}

// ---------- run ----------

func newDaemonRunCmd(opts Options) *cobra.Command {
	var flags daemonRunFlags
	var verbose bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the daemon loop in the foreground",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDaemon(cmd, opts, flags, verbose)
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVar(&verbose, "verbose", false, "also write the log to stderr")
	return cmd
}

func runDaemon(cmd *cobra.Command, opts Options, flags daemonRunFlags, verbose bool) error {
	if flags.home != "" {
		if !filepath.IsAbs(flags.home) {
			return NewExitError(ExitUsage, "--home must be an absolute path")
		}
		if err := os.Setenv("REINSTATE_HOME", flags.home); err != nil {
			return NewExitError(ExitRuntime, err.Error())
		}
	}
	home, cfg, err := loadAccountHome()
	if err != nil {
		return err
	}
	if cfg.Encryption.Type != schema.EncryptionRootKey && os.Getenv(crypto.PassphraseFDEnv) == "" {
		return NewExitError(ExitConfig, "the daemon cannot prompt for a passphrase; run rein account init so this device holds a root key (works on BYO storage and on Hop), or run it under a supervisor that sets "+crypto.PassphraseFDEnv)
	}
	seams := daemonSeamsFrom(cmd)

	// Single instance per home.
	if err := os.MkdirAll(daemon.Dir(home), 0o700); err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	lockCtx, cancelLock := context.WithTimeout(context.Background(), 500*time.Millisecond)
	instance, err := filelock.Acquire(lockCtx, daemon.LockPath(home))
	cancelLock()
	if err != nil {
		msg := "another rein daemon is already running for " + home
		if s, readErr := daemon.ReadStatus(home); readErr == nil && s.PID != 0 {
			msg += fmt.Sprintf(" (pid %d)", s.PID)
		}
		return NewExitError(ExitRuntime, msg)
	}
	defer func() { _ = instance.Close() }()

	logFile, err := daemon.OpenLog(daemon.LogPath(home))
	if err != nil {
		return NewExitError(ExitRuntime, "open daemon log: "+err.Error())
	}
	defer func() { _ = logFile.Close() }()
	var logOut io.Writer = logFile
	if verbose {
		logOut = io.MultiWriter(logFile, cmd.ErrOrStderr())
	}
	logger := log.New(logOut, "", log.LstdFlags|log.LUTC)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Detecting agents runs their version probes, which can take a few
	// seconds; a status written first lets rein daemon status say
	// "running" meanwhile instead of "stopped".
	backend := "byo"
	if cfg.Storage.Type == schema.StorageHop {
		backend = "hop"
	}
	if err := (daemon.Status{Version: daemon.StatusVersion, PID: os.Getpid(), StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Watch: "starting", Backend: backend, Pending: []daemon.PendingApproval{}}).Write(home); err != nil {
		logger.Printf("status: %v", err)
	}
	roots := watchRoots(ctx)
	events := seams.events
	watchMode := "fake"
	if events == nil {
		var watcher *daemon.Watcher
		if flags.poll {
			watcher = daemon.WatchPolling(ctx, roots, logger)
		} else {
			watcher = daemon.Watch(ctx, roots, logger)
		}
		defer watcher.Stop()
		events, watchMode = watcher.Events, watcher.Mode
	}

	var account daemon.Account
	if backend == "hop" {
		account = &hostedAccount{opts: opts}
	}
	notifier := seams.notifier
	if notifier == nil {
		notifier = daemon.OSNotifier{GOOS: runtime.GOOS, Run: seams.runner}
	}
	PrintHuman(cmd.ErrOrStderr(), "rein daemon running for %s (%s); log: %s; Ctrl-C stops it", home, backend, daemon.LogPath(home))
	return daemon.Run(ctx, daemon.Options{
		Home:     home,
		Syncer:   &inProcessSyncer{opts: opts},
		Account:  account,
		Notifier: notifier,
		Clock:    seams.clock,
		Events:   events,
		Logger:   logger,
		Observe:  seams.observe,
		Debounce: flags.debounce, PullEvery: flags.pullEvery,
		Watch: watchMode, Roots: roots, Backend: backend,
	})
}

// watchRoots lists the session directories of every detected agent the
// sync adapters cover. An agent that is not installed contributes nothing.
func watchRoots(ctx context.Context) []string {
	var roots []string
	reg := defaultRegistry()
	for _, name := range reg.Names() {
		a, ok := reg.Get(name)
		if !ok {
			continue
		}
		install, compat, err := a.Detect(ctx)
		if err != nil || compat == adapter.CompatibilityNotInstalled || install.Root == "" {
			continue
		}
		roots = append(roots, sessionRootFor(name, install.Root))
	}
	sort.Strings(roots)
	return roots
}

// sessionRootFor narrows an agent's root to the directory that holds its
// sessions, so the watcher is not woken by settings, caches, and logs.
func sessionRootFor(agent, root string) string {
	switch agent {
	case "claude":
		return filepath.Join(root, "projects")
	case "codex":
		return filepath.Join(root, "sessions")
	}
	return root
}

// inProcessSyncer runs the CLI's own push and pull in this process, with
// the same seams the daemon command was built with, so the daemon and the
// shell commands behave identically against the locker and the vendor
// stores. Output is captured; the JSON result becomes the summary.
type inProcessSyncer struct {
	opts Options
}

func (s *inProcessSyncer) Push(ctx context.Context) (string, error) {
	out, err := s.execute(ctx, "push", "--all", "--json")
	if err != nil {
		return "", err
	}
	var result struct {
		Snapshots []string `json:"snapshots"`
		Skipped   int      `json:"skipped"`
	}
	if jsonErr := json.Unmarshal(out, &result); jsonErr != nil {
		return strings.TrimSpace(string(out)), nil
	}
	return fmt.Sprintf("pushed %d snapshot(s), skipped %d unchanged", len(result.Snapshots), result.Skipped), nil
}

func (s *inProcessSyncer) Pull(ctx context.Context) (string, error) {
	pulled, skipped, err := s.pullAll(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pulled %d snapshot(s), skipped %d already synced", pulled, skipped), nil
}

// pullAll runs pull --all and reports how many snapshots it restored and
// how many were already synced.
func (s *inProcessSyncer) pullAll(ctx context.Context) (pulled, skipped int, err error) {
	out, err := s.execute(ctx, "pull", "--all", "--json")
	if err != nil {
		return 0, 0, err
	}
	var result struct {
		Pulled  int `json:"pulled"`
		Skipped int `json:"skipped"`
	}
	if jsonErr := json.Unmarshal(out, &result); jsonErr != nil {
		return 0, 0, fmt.Errorf("pull: unexpected output %q", strings.TrimSpace(string(out)))
	}
	return result.Pulled, result.Skipped, nil
}

// resumePullFresh is how recent the daemon's last successful pull must be
// for a resume to trust the local copy without pulling first.
const resumePullFresh = 15 * time.Second

// pullBeforeResume pulls the latest snapshots before a resume or fork when
// the daemon is running on this device and has not pulled within
// resumePullFresh, so a session edited on another device moments ago is
// what gets resumed. Without a running daemon nothing happens: the shell
// commands stay explicit. A failed pull never blocks the resume; it is
// reported on stderr and the local copy is used.
func pullBeforeResume(cmd *cobra.Command) {
	opts, ok := rootOptionsFrom(cmd)
	if !ok {
		return
	}
	home, err := config.Home()
	if err != nil {
		return
	}
	if note := resumePull(cmd.Context(), opts, home, time.Now()); note != "" {
		PrintHuman(cmd.ErrOrStderr(), "%s", note)
	}
}

// resumePull is pullBeforeResume's decision and outcome as one line for
// stderr; empty when nothing needs saying.
func resumePull(ctx context.Context, opts Options, home string, now time.Time) string {
	status, err := daemon.ReadStatus(home)
	if err != nil || !status.Alive(now) {
		return ""
	}
	if !status.Pull.LastOK.IsZero() && now.Sub(status.Pull.LastOK) < resumePullFresh {
		return ""
	}
	pulled, _, err := (&inProcessSyncer{opts: opts}).pullAll(ctx)
	switch {
	case errors.Is(err, daemon.ErrLocked):
		// The daemon is pulling right now; its result is as fresh as ours.
		return ""
	case errors.Is(err, daemon.ErrConflict):
		return "pull before resume: " + err.Error()
	case err != nil:
		return "pull before resume failed: " + err.Error() + "; resuming the local copy"
	case pulled > 0:
		return fmt.Sprintf("pulled %d newer snapshot(s) before resuming", pulled)
	}
	return ""
}

// execute runs one CLI command in-process and maps its exit code.
func (s *inProcessSyncer) execute(ctx context.Context, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	opts := s.opts
	opts.Context = ctx
	opts.Stdout, opts.Stderr, opts.Stdin = &stdout, &stderr, strings.NewReader("")
	opts.Args = args
	code := Execute(opts)
	switch code {
	case ExitOK:
		return stdout.Bytes(), nil
	case ExitConflict:
		return nil, fmt.Errorf("%s: %w", args[0], daemon.ErrConflict)
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = fmt.Sprintf("exit %d", code)
	}
	if strings.Contains(msg, "lock held") {
		return nil, fmt.Errorf("%s: %w", args[0], daemon.ErrLocked)
	}
	if code == ExitUsage && strings.Contains(msg, "no matching local sessions") {
		// Nothing to push is not a failure.
		return []byte(`{"snapshots":[],"skipped":0}`), nil
	}
	return nil, fmt.Errorf("%s: %s (exit %d)", args[0], msg, code)
}

// hostedAccount asks the control plane for pending pairing requests and
// the device list, with the same token store the shell commands use.
type hostedAccount struct {
	opts Options
}

func (a *hostedAccount) Pending(ctx context.Context) ([]daemon.PendingApproval, []daemon.Device, error) {
	seams := hopCommandOptions{tokens: a.opts.DeviceTokenStore}
	tok, err := seams.tokenStore().GetDeviceToken()
	if err != nil {
		return nil, nil, err
	}
	client := hop.New(tok.ControlPlaneURL)
	requests, err := client.PendingPairings(ctx, tok.Token)
	if err != nil {
		return nil, nil, err
	}
	devices, err := client.Devices(ctx, tok.Token)
	if err != nil {
		return nil, nil, err
	}
	pending := make([]daemon.PendingApproval, 0, len(requests))
	for _, r := range requests {
		expires, _ := time.Parse(time.RFC3339Nano, r.ExpiresAt)
		pending = append(pending, daemon.PendingApproval{
			RequestID: r.ID, DeviceID: r.Device.ID, DeviceName: r.Device.Name,
			Platform: r.Device.Platform, ExpiresAt: expires,
		})
	}
	list := make([]daemon.Device, 0, len(devices))
	for _, d := range devices {
		list = append(list, daemon.Device{ID: d.ID, Name: d.Name, Platform: d.Platform, This: d.ID == tok.DeviceID})
	}
	return pending, list, nil
}

// ---------- install / uninstall / start / stop / status ----------

// daemonSpec resolves the service manager and the definition for this home.
func daemonSpec(cmd *cobra.Command, flags daemonRunFlags) (daemon.Manager, daemon.Spec, string, error) {
	home, err := config.Home()
	if err != nil {
		return nil, daemon.Spec{}, "", NewExitError(ExitConfig, err.Error())
	}
	seams := daemonSeamsFrom(cmd)
	userHome, _ := os.UserHomeDir()
	manager := seams.manager
	if manager == nil {
		manager, err = daemon.NewManager(runtime.GOOS, userHome, seams.runner)
		if err != nil {
			return nil, daemon.Spec{}, "", NewExitError(ExitCompatibility, err.Error())
		}
	}
	exe := seams.executable
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return nil, daemon.Spec{}, "", NewExitError(ExitRuntime, err.Error())
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
	}
	defaultHome := filepath.Join(userHome, ".reinstate")
	spec := daemon.Spec{
		Label:      daemon.LabelFor(home, defaultHome),
		Executable: exe,
		Args:       flags.args(),
		LogPath:    filepath.Join(daemon.Dir(home), "service.log"),
		Path:       servicePath(exe),
	}
	if runtime.GOOS == "windows" {
		// Task Scheduler refuses a /XML create whose principal names no
		// user; the task runs as the installing user.
		if u, err := user.Current(); err == nil {
			spec.UserID = u.Username
		}
	}
	if filepath.Clean(home) != filepath.Clean(defaultHome) {
		spec.Home = home
	}
	for _, pair := range flags.env {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, daemon.Spec{}, "", NewExitError(ExitUsage, "--env takes KEY=VALUE, got "+pair)
		}
		if isCredentialEnvKey(key) {
			return nil, daemon.Spec{}, "", NewExitError(ExitUsage, "--env "+strings.TrimSpace(key)+" looks like a credential; the service definition is a plain file, keep secrets in the OS keyring or the backend's own credential store")
		}
		if spec.Env == nil {
			spec.Env = map[string]string{}
		}
		spec.Env[strings.TrimSpace(key)] = value
	}
	return manager, spec, home, nil
}

// isCredentialEnvKey reports whether an environment variable name looks
// like it carries a secret, so it never gets baked into a service
// definition on disk.
func isCredentialEnvKey(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, marker := range []string{"SECRET", "PASSPHRASE", "PASSWORD", "TOKEN", "CREDENTIAL", "PRIVATE_KEY", "API_KEY"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

// servicePath is the PATH the service runs with: the directory of the
// binary first, then the shell's PATH, then the usual system directories
// so vendor CLIs installed by a package manager are found.
func servicePath(exe string) string {
	parts := []string{filepath.Dir(exe)}
	if env := os.Getenv("PATH"); env != "" {
		parts = append(parts, env)
	}
	if runtime.GOOS != "windows" {
		parts = append(parts, "/usr/local/bin", "/usr/bin", "/bin", "/opt/homebrew/bin")
	}
	return strings.Join(parts, string(os.PathListSeparator))
}

func newDaemonInstallCmd() *cobra.Command {
	var flags daemonRunFlags
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register the daemon to start at login, and start it now",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, cfg, err := loadAccountHome(); err != nil {
				return err
			} else if cfg.Encryption.Type != schema.EncryptionRootKey {
				return NewExitError(ExitConfig, "the daemon needs the root-key model so it can run without a passphrase prompt; run rein account init first (it works on BYO storage as well as on Hop)")
			}
			manager, spec, home, err := daemonSpec(cmd, flags)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(daemon.Dir(home), 0o700); err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			if err := manager.Install(cmd.Context(), spec); err != nil {
				return NewExitError(ExitRuntime, fmt.Sprintf("install with %s: %v", manager.Kind(), err))
			}
			PrintHuman(cmd.OutOrStdout(), "installed %s %s (%s)", manager.Kind(), spec.Label, manager.DefinitionPath(spec))
			PrintHuman(cmd.OutOrStdout(), "the daemon starts at login and is starting now; rein daemon status shows it, log: %s", daemon.LogPath(home))
			return nil
		},
	}
	flags.bind(cmd)
	cmd.Flags().StringArrayVar(&flags.env, "env", nil, "extra KEY=VALUE for the service environment (not on Windows)")
	_ = cmd.Flags().MarkHidden("home")
	_ = cmd.Flags().MarkHidden("env")
	return cmd
}

func newDaemonUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Stop the daemon and remove its login registration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, spec, home, err := daemonSpec(cmd, daemonRunFlags{})
			if err != nil {
				return err
			}
			if err := manager.Uninstall(cmd.Context(), spec); err != nil {
				return NewExitError(ExitRuntime, fmt.Sprintf("uninstall with %s: %v", manager.Kind(), err))
			}
			_ = os.Remove(daemon.StatusPath(home))
			PrintHuman(cmd.OutOrStdout(), "uninstalled %s %s; the log under %s is kept", manager.Kind(), spec.Label, daemon.Dir(home))
			return nil
		},
	}
}

func newDaemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the installed daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, spec, _, err := daemonSpec(cmd, daemonRunFlags{})
			if err != nil {
				return err
			}
			if err := manager.Start(cmd.Context(), spec); err != nil {
				return NewExitError(ExitRuntime, fmt.Sprintf("start with %s: %v", manager.Kind(), err))
			}
			PrintHuman(cmd.OutOrStdout(), "started %s", spec.Label)
			return nil
		},
	}
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon (it starts again at the next login)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, spec, _, err := daemonSpec(cmd, daemonRunFlags{})
			if err != nil {
				return err
			}
			if err := manager.Stop(cmd.Context(), spec); err != nil {
				return NewExitError(ExitRuntime, fmt.Sprintf("stop with %s: %v", manager.Kind(), err))
			}
			PrintHuman(cmd.OutOrStdout(), "stopped %s", spec.Label)
			return nil
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether the daemon is installed and running, and what it last did",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, spec, home, err := daemonSpec(cmd, daemonRunFlags{})
			if err != nil {
				return err
			}
			state, err := manager.Status(cmd.Context(), spec)
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			status, statusErr := daemon.ReadStatus(home)
			now := time.Now()
			alive := statusErr == nil && status.Alive(now)
			if asJSON {
				payload := map[string]any{
					"service": map[string]any{
						"kind": manager.Kind(), "label": spec.Label, "definition": state.Definition,
						"installed": state.Installed, "running": state.Running, "detail": state.Detail,
					},
					"alive":    alive,
					"log_path": daemon.LogPath(home),
				}
				if statusErr == nil {
					payload["status"] = status
				}
				return WriteJSON(cmd.OutOrStdout(), payload)
			}
			out := cmd.OutOrStdout()
			registration := "not installed"
			if state.Installed {
				registration = "installed"
				if state.Detail != "" {
					registration += " (" + state.Detail + ")"
				}
			}
			PrintHuman(out, "service:  %s %s, %s", manager.Kind(), spec.Label, registration)
			switch {
			case errors.Is(statusErr, daemon.ErrNoStatus):
				PrintHuman(out, "daemon:   never ran for %s", home)
			case statusErr != nil:
				PrintHuman(out, "daemon:   status unreadable: %v", statusErr)
			case alive:
				PrintHuman(out, "daemon:   running (pid %d) since %s, %s watch, %s", status.PID, status.StartedAt.Local().Format(time.RFC3339), status.Watch, status.Backend)
			default:
				PrintHuman(out, "daemon:   stopped (last heartbeat %s)", ago(now, status.UpdatedAt))
			}
			if statusErr == nil {
				PrintHuman(out, "push:     %s", describeOutcome(now, status.Push))
				PrintHuman(out, "pull:     %s", describeOutcome(now, status.Pull))
				for _, root := range status.Roots {
					PrintHuman(out, "watching: %s", root)
				}
				if status.Backend == "hop" {
					if len(status.Devices) > 0 {
						names := make([]string, 0, len(status.Devices))
						for _, d := range status.Devices {
							name := d.Name
							if d.This {
								name += " (this device)"
							}
							names = append(names, name)
						}
						PrintHuman(out, "devices:  %s", strings.Join(names, ", "))
					}
					if status.ApprovalsError != "" {
						PrintHuman(out, "approvals: control plane unreachable: %s", status.ApprovalsError)
					}
					for _, p := range status.Pending {
						PrintHuman(out, "pending:  device %q wants to join (expires %s); run rein devices approve", p.DeviceName, p.ExpiresAt.Local().Format(time.Kitchen))
					}
					if len(status.Pending) == 0 && status.ApprovalsError == "" {
						PrintHuman(out, "pending:  no device is waiting for approval")
					}
				}
			}
			PrintHuman(out, "log:      %s", daemon.LogPath(home))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func describeOutcome(now time.Time, o daemon.Outcome) string {
	if o.At.IsZero() {
		return "not yet"
	}
	switch {
	case o.OK:
		return fmt.Sprintf("%s, %s", o.Summary, ago(now, o.At))
	case o.Conflict:
		return fmt.Sprintf("conflict recorded %s; rein conflicts list", ago(now, o.At))
	default:
		text := fmt.Sprintf("failed %s: %s", ago(now, o.At), o.Error)
		if !o.LastOK.IsZero() {
			text += fmt.Sprintf(" (last success %s)", ago(now, o.LastOK))
		}
		return text
	}
}

// ago renders a time relative to now, coarse enough to read at a glance.
func ago(now, at time.Time) string {
	if at.IsZero() {
		return "never"
	}
	d := now.Sub(at)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
}

// daemonSummaryLine is the one-line daemon state the switcher shows and
// rein daemon status repeats. Empty when no daemon has ever run here.
func daemonSummaryLine(home string, now time.Time) string {
	status, err := daemon.ReadStatus(home)
	if err != nil {
		return ""
	}
	var parts []string
	if status.Alive(now) {
		parts = append(parts, "daemon running")
	} else {
		parts = append(parts, "daemon stopped")
	}
	if !status.Push.At.IsZero() {
		label := "pushed " + ago(now, status.Push.At)
		if !status.Push.OK {
			label = "push failed " + ago(now, status.Push.At)
		}
		parts = append(parts, label)
	}
	if !status.Pull.At.IsZero() {
		label := "pulled " + ago(now, status.Pull.At)
		if !status.Pull.OK {
			label = "pull failed " + ago(now, status.Pull.At)
		}
		parts = append(parts, label)
	}
	if status.Backend == "hop" && len(status.Devices) > 0 {
		parts = append(parts, fmt.Sprintf("%d device(s)", len(status.Devices)))
	}
	if n := len(status.Pending); n == 1 {
		parts = append(parts, fmt.Sprintf("%q wants to join: rein devices approve", status.Pending[0].DeviceName))
	} else if n > 1 {
		parts = append(parts, fmt.Sprintf("%d devices want to join: rein devices approve", n))
	}
	return strings.Join(parts, " · ")
}

// announcePendingApprovals prints one stderr line per device waiting for
// approval, as recorded by the daemon, before any other command runs.
// Approval itself stays interactive: it needs the code typed.
func announcePendingApprovals(cmd *cobra.Command) {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "daemon", "devices", "completion", "help", "version":
			return
		}
	}
	if commandJSONMode(cmd, cmd.Root()) {
		return
	}
	home, err := config.Home()
	if err != nil {
		return
	}
	status, err := daemon.ReadStatus(home)
	if err != nil || len(status.Pending) == 0 {
		return
	}
	now := time.Now()
	for _, p := range status.Pending {
		if !p.ExpiresAt.IsZero() && p.ExpiresAt.Before(now) {
			continue
		}
		name := p.DeviceName
		if name == "" {
			name = p.DeviceID
		}
		PrintHuman(cmd.ErrOrStderr(), "device %q wants to join your account; run rein devices approve", name)
	}
}
