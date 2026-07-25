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
	claudeRoot := filepath.Join(userHome, ".claude", "projects", "fixture-project")
	if err := os.MkdirAll(claudeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userHome, ".claude", "version"), []byte("2.1.219\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(claudeRoot, "session-e2e.jsonl")
	if err := os.WriteFile(sessionPath, []byte(`{"type":"user","message":{"content":"synthetic e2e"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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
		})
		return out.String(), errb.String(), code
	}
	inactiveChecker := func(_ context.Context, _ string) (bool, error) { return false, nil }
	run := func(args ...string) (stdout, stderr string, code int) {
		return runWithChecker(inactiveChecker, args...)
	}

	// init
	out, errb, code := run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "reinstate-test", "--yes")
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

	// dry-run push
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
	out, errb, code = runWithChecker(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}, "pull", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitSafety || !strings.Contains(errb, "appears to be running") {
		t.Fatalf("active-agent pull exit=%d err=%q out=%q", code, errb, out)
	}

	// Simulate the second device: remove the source session before pull. A
	// successful pull must restore into Claude's vendor tree, not merely cache
	// decrypted bytes under REINSTATE_HOME.
	if err := os.Remove(sessionPath); err != nil {
		t.Fatal(err)
	}

	// pull dry-run must validate and plan without restoring vendor state.
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("pull dry-run exit=%d err=%q out=%q", code, errb, out)
	}
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run mutated Claude session path: %v", err)
	}

	// real pull must restore the session where Claude discovery can find it.
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitOK {
		t.Fatalf("pull exit=%d err=%q out=%q", code, errb, out)
	}
	restored, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("Claude session was not restored: %v", err)
	}
	if !bytes.Contains(restored, []byte("synthetic e2e")) {
		t.Fatalf("restored Claude session lost content: %q", restored)
	}
	out, errb, code = run("list", "--agent", "claude", "--json")
	if code != ExitOK || !strings.Contains(out, "session-e2e") {
		t.Fatalf("Claude cannot discover restored session: exit=%d err=%q out=%q", code, errb, out)
	}
}
