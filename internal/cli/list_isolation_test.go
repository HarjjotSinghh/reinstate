package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListHonorsClaudeConfigDir(t *testing.T) {
	decoyHome := t.TempDir()
	isolated := t.TempDir()
	writeClaudeSession(t, decoyHome, "decoy-session")
	writeClaudeSession(t, isolated, "isolated-session")

	t.Setenv("HOME", decoyHome)
	t.Setenv("USERPROFILE", decoyHome)
	t.Setenv("CLAUDE_CONFIG_DIR", isolated)
	t.Setenv("CODEX_HOME", t.TempDir())
	t.Setenv("REINSTATE_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &stdout, Stderr: &stderr,
		Args:            []string{"list", "--agent", "claude"},
		TerminalChecker: func(io.Reader, io.Writer) bool { return false },
	})
	if code != ExitOK {
		t.Fatalf("list exit=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "isolated-session") {
		t.Fatalf("list missed isolated session: %s", out)
	}
	if strings.Contains(out, "decoy-session") {
		t.Fatalf("list leaked decoy home session: %s", out)
	}
}

func writeClaudeSession(t *testing.T, root, id string) {
	t.Helper()
	dir := filepath.Join(root, "projects", "fixture-project")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"user","uuid":"` + id + `","sessionId":"` + id + `","message":{"role":"user","content":"isolated"}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
