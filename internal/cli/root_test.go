package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

func runCLI(t *testing.T, name string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Execute(Options{
		Name:   name,
		Stdout: &out,
		Stderr: &errb,
		Args:   args,
	})
	return out.String(), errb.String(), code
}

func TestNoArgsShowsHelpExit2(t *testing.T) {
	out, _, code := runCLI(t, "reinstate")
	if code != ExitUsage {
		t.Fatalf("exit=%d want %d out=%q", code, ExitUsage, out)
	}
	if !strings.Contains(out, "Usage") && !strings.Contains(out, "usage") && !strings.Contains(strings.ToLower(out), "reinstate") {
		// Cobra help may go to stdout
		if out == "" {
			t.Fatalf("expected help output, got empty")
		}
	}
}

func TestHelpExit0(t *testing.T) {
	out, _, code := runCLI(t, "reinstate", "--help")
	if code != ExitOK {
		t.Fatalf("exit=%d want 0 out=%q", code, out)
	}
	if !strings.Contains(out, "version") {
		t.Fatalf("help missing version: %q", out)
	}
}

func TestUnknownCommandExit2(t *testing.T) {
	_, errb, code := runCLI(t, "reinstate", "not-a-real-command")
	if code != ExitUsage {
		t.Fatalf("exit=%d want %d stderr=%q", code, ExitUsage, errb)
	}
}

func TestVersionJSON(t *testing.T) {
	out, _, code := runCLI(t, "reinstate", "version", "--json")
	if code != ExitOK {
		t.Fatalf("exit=%d out=%q", code, out)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if m["name"] != "reinstate" {
		t.Fatalf("name=%q", m["name"])
	}
	if m["version"] == "" {
		t.Fatal("empty version")
	}
}

func TestExitErrorMapping(t *testing.T) {
	if ExitCodeFrom(nil) != ExitOK {
		t.Fatal("nil")
	}
	if ExitCodeFrom(NewExitError(ExitConfig, "x")) != ExitConfig {
		t.Fatal("config")
	}
	if ExitCodeFrom(NewExitError(ExitConflict, "c")) != ExitConflict {
		t.Fatal("conflict")
	}
}

func TestReinAndReinstateSame(t *testing.T) {
	out1, _, c1 := runCLI(t, "rein", "version", "--json")
	out2, _, c2 := runCLI(t, "reinstate", "version", "--json")
	if c1 != ExitOK || c2 != ExitOK {
		t.Fatalf("codes %d %d", c1, c2)
	}
	var a, b map[string]string
	_ = json.Unmarshal([]byte(out1), &a)
	_ = json.Unmarshal([]byte(out2), &b)
	if a["version"] != b["version"] {
		t.Fatalf("version mismatch %q vs %q", a["version"], b["version"])
	}
}

func TestInitDoesNotExposeSecretFlags(t *testing.T) {
	out, _, code := runCLI(t, "reinstate", "init", "--help")
	if code != ExitOK {
		t.Fatalf("exit=%d out=%q", code, out)
	}
	for _, forbidden := range []string{"--access-key", "--secret-key", "--passphrase"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("init help exposes secret-bearing flag %q: %s", forbidden, out)
		}
	}
}

func TestInitWithProfileIDRejectsMissingRemoteManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")

	_, errb, code := runCLI(
		t,
		"reinstate",
		"init",
		"--profile-id", "33333333-3333-4333-8333-333333333333",
		"--endpoint", "https://example.r2.cloudflarestorage.com",
		"--bucket", "reinstate-test",
		"--yes",
	)
	if code != ExitAuthStorage || !strings.Contains(errb, "remote profile manifest not found") {
		t.Fatalf("init exit=%d want %d stderr=%q", code, ExitAuthStorage, errb)
	}
	if _, err := os.Stat(config.ConfigPath(home)); !os.IsNotExist(err) {
		t.Fatalf("failed additional-device init wrote config: %v", err)
	}
}

func TestInitWithProfileIDRecordsRemoteManifestRequirement(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	profileID := "33333333-3333-4333-8333-333333333333"
	prefix := "profiles/" + profileID

	store, err := memory.NewDisk(filepath.Join(home, "cache", "memory-backend"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(
		context.Background(),
		prefix+"/manifest.age",
		strings.NewReader("synthetic encrypted manifest"),
		0,
		backend.PutOptions{},
	); err != nil {
		t.Fatal(err)
	}

	out, errb, code := runCLI(
		t,
		"reinstate",
		"init",
		"--profile-id", profileID,
		"--endpoint", "https://example.r2.cloudflarestorage.com",
		"--bucket", "reinstate-test",
		"--yes",
	)
	if code != ExitOK {
		t.Fatalf("init exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	rawConfig, err := os.ReadFile(config.ConfigPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawConfig), "remote_profile_required = true") {
		t.Fatalf("additional-device config does not require the remote profile:\n%s", rawConfig)
	}
}

func TestInitRefusesToOverwriteInitializedHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")

	existing := schema.DefaultConfig(
		"11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222",
	)
	existing.Storage.Endpoint = "https://original.example.invalid"
	existing.Storage.Bucket = "original-bucket"
	existing.Storage.Prefix = "profiles/11111111-1111-4111-8111-111111111111"
	existing.Storage.CredentialRef = "reinstate/11111111-1111-4111-8111-111111111111/s3"
	if err := config.SaveConfig(home, existing); err != nil {
		t.Fatal(err)
	}
	state := schema.NewState()
	state.LastManifestRev = "original-revision"
	if err := config.SaveState(home, state); err != nil {
		t.Fatal(err)
	}
	configBefore, err := os.ReadFile(config.ConfigPath(home))
	if err != nil {
		t.Fatal(err)
	}
	stateBefore, err := os.ReadFile(config.StatePath(home))
	if err != nil {
		t.Fatal(err)
	}

	_, errb, code := runCLI(
		t,
		"reinstate",
		"init",
		"--endpoint", "https://replacement.example.invalid",
		"--bucket", "replacement-bucket",
		"--yes",
	)
	if code != ExitSafety || !strings.Contains(errb, "already initialized") {
		t.Fatalf("init exit=%d want %d stderr=%q", code, ExitSafety, errb)
	}
	configAfter, err := os.ReadFile(config.ConfigPath(home))
	if err != nil {
		t.Fatal(err)
	}
	stateAfter, err := os.ReadFile(config.StatePath(home))
	if err != nil {
		t.Fatal(err)
	}
	if string(configAfter) != string(configBefore) || string(stateAfter) != string(stateBefore) {
		t.Fatal("refused init changed existing config or state")
	}
	backups, err := filepath.Glob(filepath.Join(home, "backups", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("refused init created backups: %v", backups)
	}
}

func TestInitForceBacksUpExistingConfigAndState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")

	existing := schema.DefaultConfig(
		"11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222",
	)
	existing.Storage.Endpoint = "https://original.example.invalid"
	existing.Storage.Bucket = "original-bucket"
	existing.Storage.Prefix = "profiles/11111111-1111-4111-8111-111111111111"
	existing.Storage.CredentialRef = "reinstate/11111111-1111-4111-8111-111111111111/s3"
	if err := config.SaveConfig(home, existing); err != nil {
		t.Fatal(err)
	}
	state := schema.NewState()
	state.LastManifestRev = "original-revision"
	if err := config.SaveState(home, state); err != nil {
		t.Fatal(err)
	}
	configBefore, err := os.ReadFile(config.ConfigPath(home))
	if err != nil {
		t.Fatal(err)
	}
	stateBefore, err := os.ReadFile(config.StatePath(home))
	if err != nil {
		t.Fatal(err)
	}

	out, errb, code := runCLI(
		t,
		"reinstate",
		"init",
		"--endpoint", "https://replacement.example.invalid",
		"--bucket", "replacement-bucket",
		"--yes",
		"--force",
	)
	if code != ExitOK {
		t.Fatalf("init --force exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	if !strings.Contains(out, "backups/") {
		t.Fatalf("init --force did not report the backup set location: %q", out)
	}
	backupSets, err := os.ReadDir(filepath.Join(home, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backupSets) != 1 || !backupSets[0].IsDir() {
		t.Fatalf("backup sets = %v, want one directory", backupSets)
	}
	backupRoot := filepath.Join(home, "backups", backupSets[0].Name())
	configBackup, err := os.ReadFile(filepath.Join(backupRoot, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	stateBackup, err := os.ReadFile(filepath.Join(backupRoot, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(configBackup) != string(configBefore) || string(stateBackup) != string(stateBefore) {
		t.Fatal("forced init backup does not preserve previous config and state")
	}
	newConfig, err := config.LoadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if newConfig.ProfileID == existing.ProfileID ||
		newConfig.Storage.Endpoint != "https://replacement.example.invalid" {
		t.Fatalf("forced init did not replace config: %+v", newConfig)
	}
}

func TestStatusAllowsMissingManifestForNewFirstDevice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	setTestPassphraseFD(t)

	out, errb, code := runCLI(
		t,
		"reinstate",
		"init",
		"--endpoint", "https://example.r2.cloudflarestorage.com",
		"--bucket", "reinstate-test",
		"--yes",
	)
	if code != ExitOK {
		t.Fatalf("init exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	out, errb, code = runCLI(t, "reinstate", "status")
	if code != ExitOK || !strings.Contains(out, "(0 sessions)") {
		t.Fatalf("status exit=%d want %d stdout=%q stderr=%q", code, ExitOK, out, errb)
	}
}

func TestStatusRejectsMissingManifestForJoinedProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	writeTestConfigState(t, home, true)
	setTestPassphraseFD(t)

	_, errb, code := runCLI(t, "reinstate", "status")
	if code != ExitAuthStorage || !strings.Contains(errb, "remote profile manifest not found") {
		t.Fatalf("status exit=%d want %d stderr=%q", code, ExitAuthStorage, errb)
	}
}

func TestStatusRejectsMissingManifestForEstablishedFirstDevice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	writeTestConfigState(t, home, false)
	state, err := config.LoadState(home)
	if err != nil {
		t.Fatal(err)
	}
	state.LastManifestRev = "synthetic-prior-manifest-revision"
	if err := config.SaveState(home, state); err != nil {
		t.Fatal(err)
	}
	setTestPassphraseFD(t)

	_, errb, code := runCLI(t, "reinstate", "status")
	if code != ExitAuthStorage || !strings.Contains(errb, "remote profile manifest not found") {
		t.Fatalf("status exit=%d want %d stderr=%q", code, ExitAuthStorage, errb)
	}
}

func TestDiffMissingManifestUsesAuthStorageExit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)
	writeTestConfigState(t, home, true)
	setTestPassphraseFD(t)

	_, errb, code := runCLI(t, "reinstate", "diff")
	if code != ExitAuthStorage || !strings.Contains(errb, "remote profile manifest not found") {
		t.Fatalf("diff exit=%d want %d stderr=%q", code, ExitAuthStorage, errb)
	}
}

func writeTestConfigState(t *testing.T, home string, remoteProfileRequired bool) {
	t.Helper()
	cfg := schema.DefaultConfig(
		"33333333-3333-4333-8333-333333333333",
		"44444444-4444-4444-8444-444444444444",
	)
	cfg.RemoteProfileRequired = remoteProfileRequired
	cfg.Storage.Endpoint = "https://example.r2.cloudflarestorage.com"
	cfg.Storage.Bucket = "reinstate-test"
	cfg.Storage.Prefix = "profiles/" + cfg.ProfileID
	cfg.Storage.CredentialRef = "reinstate/" + cfg.ProfileID + "/s3"
	if err := config.SaveConfig(home, cfg); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveState(home, schema.NewState()); err != nil {
		t.Fatal(err)
	}
}

func setTestPassphraseFD(t *testing.T) {
	t.Helper()
	passphraseFile, err := os.CreateTemp(t.TempDir(), "passphrase-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = passphraseFile.Close() })
	if _, err := passphraseFile.WriteString("synthetic-test-passphrase-not-secret\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := passphraseFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.Itoa(int(passphraseFile.Fd())))
}

func TestConflictReadCommandsRequireConfig(t *testing.T) {
	t.Setenv("REINSTATE_HOME", t.TempDir())
	for _, args := range [][]string{
		{"conflicts", "list"},
		{"conflicts", "show", "missing"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			_, errb, code := runCLI(t, "reinstate", args...)
			if code != ExitConfig {
				t.Fatalf("%v exit=%d want %d stderr=%q", args, code, ExitConfig, errb)
			}
		})
	}
}

func TestStatusMissingConfigDoesNotLeakAbsolutePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	_, errb, code := runCLI(t, "reinstate", "status")
	if code != ExitConfig {
		t.Fatalf("exit=%d want %d stderr=%q", code, ExitConfig, errb)
	}
	if !strings.Contains(errb, "config missing") {
		t.Fatalf("missing stable config error: %q", errb)
	}
	if strings.Contains(errb, home) || strings.Contains(errb, "config.toml") {
		t.Fatalf("missing-config error leaked local path: %q", errb)
	}
}

func TestSetupCheckJSONPreservesFailureExit(t *testing.T) {
	t.Setenv("REINSTATE_HOME", t.TempDir())
	out, errb, code := runCLI(t, "reinstate", "setup", "check", "--json")
	if code != ExitConfig {
		t.Fatalf("exit=%d want %d stdout=%q stderr=%q", code, ExitConfig, out, errb)
	}
	if !strings.Contains(out, `"summary"`) || !strings.Contains(out, `"status": "fail"`) {
		t.Fatalf("missing JSON failure report: %q", out)
	}
}

func TestPlanSessionRestore(t *testing.T) {
	probeErr := errors.New("probe failed")
	const sessionPath = "/home/dev/.claude/projects/demo/abc.jsonl"

	forkTests := []struct {
		name string
		busy bool
		want restoreDisposition
	}{
		{name: "fork policy restores in place when idle", busy: false, want: restoreInPlace},
		{name: "fork policy forks rather than refusing when busy", busy: true, want: restoreAsFork},
	}
	for _, tc := range forkTests {
		t.Run(tc.name, func(t *testing.T) {
			checker := func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) {
				return tc.busy, true, nil
			}
			got, err := planSessionRestore(
				context.Background(), checker, "claude",
				processcheck.Target{Path: sessionPath}, schema.ActiveAgentFork)
			if err != nil {
				t.Fatalf("fork policy must never refuse: %v", err)
			}
			if got != tc.want {
				t.Fatalf("disposition=%v want %v", got, tc.want)
			}
		})
	}

	// The default (empty) policy must behave as fork, never as a refusal.
	t.Run("empty policy defaults to forking", func(t *testing.T) {
		checker := func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return true, true, nil }
		got, err := planSessionRestore(
			context.Background(), checker, "claude", processcheck.Target{Path: sessionPath}, "")
		if err != nil || got != restoreAsFork {
			t.Fatalf("disposition=%v err=%v, want fork and no error", got, err)
		}
	})

	// Conflict resolution keeps the refusal, because --keep-both is the
	// explicit way to fork there.
	t.Run("requireSessionRestorable refuses under the fork policy", func(t *testing.T) {
		checker := func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return true, true, nil }
		err := requireSessionRestorable(
			context.Background(), checker, "claude",
			processcheck.Target{Path: sessionPath}, schema.ActiveAgentFork)
		if err == nil || !strings.Contains(err.Error(), "is currently using this session") {
			t.Fatalf("expected a refusal, got %v", err)
		}
	})

	tests := []struct {
		name        string
		policy      string
		busy        bool
		scoped      bool
		probeErr    error
		wantErr     bool
		wantMessage string
		wantPath    string
	}{
		{
			name:     "scoped policy allows an idle target",
			policy:   schema.ActiveAgentScoped,
			busy:     false,
			scoped:   true,
			wantPath: sessionPath,
		},
		{
			name:        "scoped policy refuses a session the agent is holding",
			policy:      schema.ActiveAgentScoped,
			busy:        true,
			scoped:      true,
			wantErr:     true,
			wantMessage: "is currently using this session",
			wantPath:    sessionPath,
		},
		{
			name:        "unscoped fallback explains the imprecision",
			policy:      schema.ActiveAgentScoped,
			busy:        true,
			scoped:      false,
			wantErr:     true,
			wantMessage: "cannot tell which session it is using",
			wantPath:    sessionPath,
		},
		{
			name:     "off policy skips the check entirely",
			policy:   schema.ActiveAgentOff,
			busy:     true,
			scoped:   true,
			wantPath: "",
		},
		{
			name:        "strict policy asks the host-wide question",
			policy:      schema.ActiveAgentStrict,
			busy:        true,
			scoped:      false,
			wantErr:     true,
			wantMessage: "cannot tell which session it is using",
			// Strict must discard the target so detection stays host-wide.
			wantPath: "",
		},
		{
			name:        "unknown policy is rejected",
			policy:      "sometimes",
			wantErr:     true,
			wantMessage: "unsupported restore.active_agent_policy",
		},
		{
			name:        "probe failure is surfaced, not swallowed",
			policy:      schema.ActiveAgentScoped,
			probeErr:    probeErr,
			wantErr:     true,
			wantMessage: "cannot verify",
			wantPath:    sessionPath,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			checked := false
			checker := func(_ context.Context, _ string, tgt processcheck.Target) (bool, bool, error) {
				checked = true
				gotPath = tgt.Path
				return tc.busy, tc.scoped, tc.probeErr
			}
			_, err := planSessionRestore(
				context.Background(), checker, "claude",
				processcheck.Target{Path: sessionPath}, tc.policy)

			if tc.wantErr && err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantMessage != "" && !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("error %q does not contain %q", err, tc.wantMessage)
			}
			if tc.probeErr != nil && !errors.Is(err, tc.probeErr) {
				t.Fatalf("probe failure was not preserved: %v", err)
			}
			if tc.policy == schema.ActiveAgentOff && checked {
				t.Fatalf("off policy must not consult the checker")
			}
			if checked && gotPath != tc.wantPath {
				t.Fatalf("checker received path %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}
