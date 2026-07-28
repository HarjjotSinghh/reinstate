package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// TestCLISyntheticSyncPath drives the real CLI entrypoint (Execute) for
// init → list → push --dry-run → push → status → pull --dry-run → pull
// against the disk-backed memory backend with synthetic session fixtures only.
func TestCLISyntheticSyncPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	_ = os.Unsetenv("REINSTATE_PASSPHRASE")

	// plant a synthetic Claude session under a fake agent root via home layout
	// Adapters look at real ~/.claude; for isolation we use fixture paths through
	// list/push by planting under temp and overriding via a dedicated agent root
	// is not supported on default adapters — instead create sessions under
	// REINSTATE_HOME is not enough. Use push of a discovered path by writing
	// into a custom tree and exercising list with no sessions is ok for init/status,
	// but we need sessions for push. Plant under $HOME/.claude for the test process.
	userHome := t.TempDir()
	// UserHomeDir uses HOME on Unix and USERPROFILE on Windows.
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)
	sourceProject := filepath.Join(userHome, "Projects", "reinstate-phase1-mac")
	targetProject := filepath.Join(userHome, "Projects", "reinstate-phase1-windows")
	if err := os.MkdirAll(sourceProject, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetProject, 0o700); err != nil {
		t.Fatal(err)
	}
	claudeProjectsRoot := filepath.Join(userHome, ".claude", "projects")
	sourceClaudeRoot := filepath.Join(claudeProjectsRoot, claudeProjectDirectoryForTest(sourceProject))
	if err := os.MkdirAll(sourceClaudeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userHome, ".claude", "version"), []byte("2.1.219\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(sourceClaudeRoot, "session-e2e.jsonl")
	meta, err := json.Marshal(map[string]any{"type": "meta", "cwd": sourceProject})
	if err != nil {
		t.Fatal(err)
	}
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"synthetic e2e"}}`+"\n")...)
	if err := os.WriteFile(sessionPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	testCodec := &fastAgeEnvelopeCodec{}
	runWithChecker := func(processChecker AgentProcessChecker, args ...string) (stdout, stderr string, code int) {
		passphraseFile, err := os.CreateTemp(t.TempDir(), "passphrase-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = passphraseFile.Close() }()
		if _, err := passphraseFile.WriteString("e2e-test-passphrase-not-real\n"); err != nil {
			t.Fatal(err)
		}
		if _, err := passphraseFile.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.Itoa(int(passphraseFile.Fd())))
		var out, errb bytes.Buffer
		code = Execute(Options{
			Name: "reinstate", Stdout: &out, Stderr: &errb, Args: args,
			AgentProcessChecker: processChecker,
			EnvelopeCodec:       testCodec,
		})
		return out.String(), errb.String(), code
	}
	inactiveChecker := func(_ context.Context, _, _ string) (bool, bool, error) { return false, true, nil }
	run := func(args ...string) (stdout, stderr string, code int) {
		return runWithChecker(inactiveChecker, args...)
	}
	// runWithCheckerAndPolicy pins restore.active_agent_policy for one run so
	// each policy can be exercised against the same synthetic profile.
	runWithCheckerAndPolicy := func(
		processChecker AgentProcessChecker, policy string, args ...string,
	) (stdout, stderr string, code int) {
		cfg, err := config.LoadConfig(home)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Restore.ActiveAgentPolicy = policy
		if err := config.SaveConfig(home, cfg); err != nil {
			t.Fatal(err)
		}
		return runWithChecker(processChecker, args...)
	}

	// init
	out, errb, code := run(
		"init",
		"--endpoint", "https://example.r2.cloudflarestorage.com",
		"--bucket", "reinstate-test",
		"--project", "local/reinstate="+sourceProject,
		"--yes",
	)
	if code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(out, "credential_ref=") {
		t.Fatalf("init missing credential_ref: %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, "config.toml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials")); !os.IsNotExist(err) {
		t.Fatalf("init wrote plaintext credential directory: %v", err)
	}

	// list
	out, errb, code = run("list", "--agent", "claude", "--json")
	if code != ExitOK {
		t.Fatalf("list exit=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "session-e2e") {
		t.Fatalf("list missing session: %q", out)
	}

	// Human dry-run output must describe a plan, not claim an upload happened.
	out, errb, code = run("push", "--agent", "claude", "--session", "session-e2e", "--dry-run")
	if code != ExitOK {
		t.Fatalf("push dry-run exit=%d err=%q out=%q", code, errb, out)
	}
	if !strings.Contains(out, "would push 1 snapshot(s)") || strings.Contains(out, "pushed 1 snapshot(s)") {
		t.Fatalf("push dry-run described a completed upload: %q", out)
	}

	// JSON dry-run remains machine-readable and non-mutating.
	out, errb, code = run("push", "--agent", "claude", "--session", "session-e2e", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("push dry-run exit=%d err=%q out=%q", code, errb, out)
	}

	// real push
	out, errb, code = run("push", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitOK {
		t.Fatalf("push exit=%d err=%q out=%q", code, errb, out)
	}
	var pushRes map[string]any
	if err := json.Unmarshal([]byte(out), &pushRes); err != nil {
		t.Fatalf("push json: %v %q", err, out)
	}
	snapshots, ok := pushRes["snapshots"].([]any)
	if !ok || len(snapshots) != 1 {
		t.Fatalf("push snapshots = %#v, want one", pushRes["snapshots"])
	}
	pushedSnapshot, ok := snapshots[0].(string)
	if !ok || pushedSnapshot == "" {
		t.Fatalf("push snapshot = %#v, want non-empty ID", snapshots[0])
	}
	stateAfterPush, err := config.LoadState(home)
	if err != nil {
		t.Fatal(err)
	}
	if stateAfterPush.LastManifestRev != pushedSnapshot {
		t.Fatalf(
			"last_manifest_revision = %q, want latest snapshot/manifest revision %q",
			stateAfterPush.LastManifestRev,
			pushedSnapshot,
		)
	}
	out, errb, code = run("push", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitOK {
		t.Fatalf("second push exit=%d err=%q out=%q", code, errb, out)
	}
	var secondPush map[string]any
	if err := json.Unmarshal([]byte(out), &secondPush); err != nil {
		t.Fatalf("second push json: %v %q", err, out)
	}
	if secondPush["skipped"] != float64(1) {
		t.Fatalf("unchanged session was not skipped: %v", secondPush)
	}

	// Simulate the destination device by keeping the canonical project ID while
	// changing its local root from the source Mac path to a Windows-side path.
	cfg, err := config.LoadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 1 || cfg.Projects[0].ID != "local/reinstate" {
		t.Fatalf("unexpected project mappings: %+v", cfg.Projects)
	}
	cfg.Projects[0].LocalRoot = targetProject
	if err := config.SaveConfig(home, cfg); err != nil {
		t.Fatal(err)
	}
	targetSessionPath := filepath.Join(
		claudeProjectsRoot,
		claudeProjectDirectoryForTest(targetProject),
		"session-e2e.jsonl",
	)

	// status
	out, errb, code = run("status", "--json")
	if code != ExitOK {
		t.Fatalf("status exit=%d err=%q out=%q", code, errb, out)
	}
	if !strings.Contains(out, "session-e2e") && !strings.Contains(out, "claude:") {
		t.Fatalf("status missing session: %q", out)
	}

	// A mutating pull must refuse to replace an existing session while the
	// selected vendor process is active.
	if err := os.MkdirAll(filepath.Dir(targetSessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetSessionPath, []byte(`{"type":"user","message":{"content":"existing destination"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The scoped checker reports the agent is holding this exact session file.
	busyChecker := func(_ context.Context, _, _ string) (bool, bool, error) { return true, true, nil }
	out, errb, code = runWithCheckerAndPolicy(busyChecker, schema.ActiveAgentScoped,
		"pull", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitSafety || !strings.Contains(errb, "is currently using this session") {
		t.Fatalf("active-agent pull exit=%d err=%q out=%q", code, errb, out)
	}

	// Under the default fork policy a busy session is never blocked and never
	// overwritten: the live file is left byte-for-byte intact and the remote
	// copy lands beside it as a distinct session.
	liveBefore, err := os.ReadFile(targetSessionPath)
	if err != nil {
		t.Fatal(err)
	}
	out, errb, code = runWithCheckerAndPolicy(busyChecker, schema.ActiveAgentFork,
		"pull", "--agent", "claude", "--session", "session-e2e")
	if code != ExitOK {
		t.Fatalf("fork policy pull exit=%d err=%q out=%q", code, errb, out)
	}
	if !strings.Contains(out, "is in use, so it was left unchanged") {
		t.Fatalf("fork was not reported to the operator: %q", out)
	}
	liveAfter, err := os.ReadFile(targetSessionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(liveAfter) != string(liveBefore) {
		t.Fatal("fork policy modified the live session file")
	}
	forks, err := filepath.Glob(filepath.Join(filepath.Dir(targetSessionPath), "session-e2e-active-*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(forks) != 1 {
		t.Fatalf("expected exactly one forked session, got %v", forks)
	}
	// Re-pulling the same remote state must be idempotent, not pile up forks.
	out, errb, code = runWithCheckerAndPolicy(busyChecker, schema.ActiveAgentFork,
		"pull", "--agent", "claude", "--session", "session-e2e")
	if code != ExitOK {
		t.Fatalf("repeat fork pull exit=%d err=%q out=%q", code, errb, out)
	}
	forksAgain, err := filepath.Glob(filepath.Join(filepath.Dir(targetSessionPath), "session-e2e-active-*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(forksAgain) != 1 {
		t.Fatalf("repeated pull created duplicate forks: %v", forksAgain)
	}
	for _, fork := range forksAgain {
		if err := os.Remove(fork); err != nil {
			t.Fatal(err)
		}
	}

	// --allow-active-agents clears the liveness refusal. Every other safety
	// check still applies, so this local file (deliberately diverged above)
	// is reported as a conflict rather than silently overwritten.
	out, errb, code = runWithChecker(busyChecker,
		"pull", "--agent", "claude", "--session", "session-e2e", "--json", "--allow-active-agents")
	if code == ExitSafety || strings.Contains(errb, "is currently using this session") {
		t.Fatalf("--allow-active-agents did not clear the refusal: exit=%d err=%q out=%q", code, errb, out)
	}
	if code != ExitConflict {
		t.Fatalf("diverged target should record a conflict: exit=%d err=%q out=%q", code, errb, out)
	}
	// Clear the conflict recorded above so later steps start from clean state.
	if err := os.RemoveAll(filepath.Join(home, "conflicts")); err != nil {
		t.Fatal(err)
	}

	// Simulate the second device: remove the source session before pull. A
	// successful pull must restore into Claude's vendor tree, not merely cache
	// decrypted bytes under REINSTATE_HOME.
	if err := os.Remove(sessionPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(targetSessionPath); err != nil {
		t.Fatal(err)
	}

	// Human pull dry-run must validate and plan without claiming a restore.
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--dry-run")
	if code != ExitOK {
		t.Fatalf("pull human dry-run exit=%d err=%q out=%q", code, errb, out)
	}
	if !strings.Contains(out, "would pull 1 snapshot(s)") || strings.Contains(out, "pulled 1 snapshot(s)") {
		t.Fatalf("pull dry-run described a completed restore: %q", out)
	}

	// JSON pull dry-run remains machine-readable and non-mutating.
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("pull dry-run exit=%d err=%q out=%q", code, errb, out)
	}
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run recreated source Claude session path: %v", err)
	}
	if _, err := os.Stat(targetSessionPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run mutated destination Claude session path: %v", err)
	}

	// real pull must restore the session where Claude discovery can find it.
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitOK {
		t.Fatalf("pull exit=%d err=%q out=%q", code, errb, out)
	}
	restored, err := os.ReadFile(targetSessionPath)
	if err != nil {
		t.Fatalf("Claude session was not restored under the destination project: %v", err)
	}
	if !bytes.Contains(restored, []byte("synthetic e2e")) {
		t.Fatalf("restored Claude session lost content: %q", restored)
	}
	var restoredMeta map[string]any
	firstRecord := bytes.SplitN(restored, []byte("\n"), 2)[0]
	if err := json.Unmarshal(firstRecord, &restoredMeta); err != nil {
		t.Fatalf("restored Claude metadata is invalid: %v", err)
	}
	restoredCWD, _ := restoredMeta["cwd"].(string)
	if filepath.Clean(restoredCWD) != filepath.Clean(targetProject) ||
		filepath.Clean(restoredCWD) == filepath.Clean(sourceProject) {
		t.Fatalf("restored Claude cwd = %q, want destination %q", restoredCWD, targetProject)
	}
	out, errb, code = run("list", "--agent", "claude", "--json")
	if code != ExitOK || !strings.Contains(out, "session-e2e") {
		t.Fatalf("Claude cannot discover restored session: exit=%d err=%q out=%q", code, errb, out)
	}
	if testCodec.encryptions.Load() == 0 {
		t.Fatal("CLI synthetic sync path did not exercise the injected age envelope codec")
	}
}

func claudeProjectDirectoryForTest(projectPath string) string {
	if resolved, err := filepath.EvalSymlinks(projectPath); err == nil {
		projectPath = resolved
	}
	var directory strings.Builder
	for _, unit := range utf16.Encode([]rune(projectPath)) {
		if unit >= 'a' && unit <= 'z' || unit >= 'A' && unit <= 'Z' || unit >= '0' && unit <= '9' {
			directory.WriteByte(byte(unit))
		} else {
			directory.WriteByte('-')
		}
	}
	return directory.String()
}

func TestVerifyRestoredSessionRequiresPlannedDestination(t *testing.T) {
	root := t.TempDir()
	wrongDirectory := filepath.Join(root, "projects", "source-project")
	if err := os.MkdirAll(wrongDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wrongDirectory, "session-001.jsonl"), []byte(`{"type":"user"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plannedPath := filepath.Join(root, "projects", "destination-project", "session-001.jsonl")
	plan := adapter.RestorePlan{Session: adapter.Session{
		ID:           "session-001",
		Agent:        "claude",
		Path:         plannedPath,
		RelativePath: "projects/destination-project/session-001.jsonl",
	}}
	selectedAdapter := &claude.Adapter{Root: root}

	if _, err := verifyRestoredSession(context.Background(), selectedAdapter, plan); err == nil {
		t.Fatal("expected a matching ID at the wrong path to fail restore verification")
	}

	if err := os.MkdirAll(filepath.Dir(plannedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plannedPath, []byte(`{"type":"user"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := verifyRestoredSession(context.Background(), selectedAdapter, plan)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Path != plannedPath {
		t.Fatalf("verified path = %q, want %q", restored.Path, plannedPath)
	}
}
