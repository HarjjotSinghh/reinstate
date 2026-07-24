package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	t.Setenv("REINSTATE_PASSPHRASE", "e2e-test-passphrase-not-real")
	// clear storage env so credential file path is used after init
	_ = os.Unsetenv("REINSTATE_S3_ACCESS_KEY_ID")
	_ = os.Unsetenv("REINSTATE_S3_SECRET_ACCESS_KEY")

	// plant a synthetic Claude session under a fake agent root via home layout
	// Adapters look at real ~/.claude; for isolation we use fixture paths through
	// list/push by planting under temp and overriding via a dedicated agent root
	// is not supported on default adapters — instead create sessions under
	// REINSTATE_HOME is not enough. Use push of a discovered path by writing
	// into a custom tree and exercising list with no sessions is ok for init/status,
	// but we need sessions for push. Plant under $HOME/.claude for the test process.
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	// On macOS UserHomeDir uses HOME
	claudeRoot := filepath.Join(userHome, ".claude", "projects", "fixture-project")
	if err := os.MkdirAll(claudeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(claudeRoot, "session-e2e.jsonl")
	if err := os.WriteFile(sessionPath, []byte(`{"type":"user","message":{"content":"synthetic e2e"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) (stdout, stderr string, code int) {
		var out, errb bytes.Buffer
		code = Execute(Options{Name: "reinstate", Stdout: &out, Stderr: &errb, Args: args})
		return out.String(), errb.String(), code
	}

	// init
	out, errb, code := run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "reinstate-test", "--yes",
		"--access-key", "AKIA_TEST", "--secret-key", "SECRET_TEST")
	if code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(out, "credential_ref=") {
		t.Fatalf("init missing credential_ref: %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, "config.toml")); err != nil {
		t.Fatal(err)
	}
	// credentials persisted for credential_ref
	ents, _ := os.ReadDir(filepath.Join(home, "credentials"))
	if len(ents) == 0 {
		t.Fatal("expected credential file from init")
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

	// status
	out, errb, code = run("status", "--json")
	if code != ExitOK {
		t.Fatalf("status exit=%d err=%q out=%q", code, errb, out)
	}
	if !strings.Contains(out, "session-e2e") && !strings.Contains(out, "claude:") {
		t.Fatalf("status missing session: %q", out)
	}

	// pull dry-run and pull
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("pull dry-run exit=%d err=%q out=%q", code, errb, out)
	}
	out, errb, code = run("pull", "--agent", "claude", "--session", "session-e2e", "--json")
	if code != ExitOK {
		t.Fatalf("pull exit=%d err=%q out=%q", code, errb, out)
	}
}
