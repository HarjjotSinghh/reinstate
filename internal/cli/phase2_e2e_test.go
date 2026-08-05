package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// TestCLIPhase2SyntheticContinuityFlow drives the real CLI entrypoint through
// the complete configless Phase 2 workflow against synthetic vendor records.
// It intentionally provides no sync config, credentials, passphrase, or
// backend.
func TestCLIPhase2SyntheticContinuityFlow(t *testing.T) {
	home := t.TempDir()
	claudeRoot := filepath.Join(t.TempDir(), "claude")
	codexRoot := filepath.Join(t.TempDir(), "codex")
	if err := os.CopyFS(
		claudeRoot,
		os.DirFS(filepath.Join("..", "..", "testdata", "sessionindex", "claude", "macos")),
	); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(
		codexRoot,
		os.DirFS(filepath.Join("..", "..", "testdata", "adapters", "codex", "macos")),
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "phase2-must-not-open-a-backend")
	sources := []sessionindex.Source{
		sessionindex.NewClaudeSource(claudeRoot),
		sessionindex.NewCodexSource(codexRoot),
	}
	runner := &recordingLaunchRunner{}

	run := func(args ...string) (string, string, int) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		code := Execute(Options{
			Name:                "rein",
			Stdout:              &stdout,
			Stderr:              &stderr,
			Args:                args,
			SessionSources:      sources,
			SessionLaunchRunner: runner,
			PreflightVerifier:   readyPreflightVerifier{},
		})
		return stdout.String(), stderr.String(), code
	}
	requireOK := func(step string, args ...string) string {
		t.Helper()
		stdout, stderr, code := run(args...)
		if code != ExitOK {
			t.Fatalf("%s exit=%d stdout=%q stderr=%q", step, code, stdout, stderr)
		}
		return stdout
	}

	listed := requireOK("sessions", "sessions", "--json")
	if !strings.Contains(listed, `"claude:claude-syn-macos"`) ||
		!strings.Contains(listed, `"codex:rollout-syn-001"`) {
		t.Fatalf("sessions missing synthetic references: %s", listed)
	}
	searched := requireOK("search", "search", "Synthetic", "Claude", "--json")
	if !strings.Contains(searched, `"claude:claude-syn-macos"`) ||
		strings.Contains(searched, "Synthetic Claude macOS fixture request") {
		t.Fatalf("unsafe or incorrect search response: %s", searched)
	}
	inspected := requireOK("inspect", "inspect", "claude:claude-syn-macos", "--json")
	if !strings.Contains(inspected, `"prompt_preview": "Synthetic Claude macOS fixture request"`) {
		t.Fatalf("unsafe or incorrect inspect response: %s", inspected)
	}

	for name, args := range map[string][]string{
		"last":        {"last", "--dry-run", "--json"},
		"resume":      {"resume", "codex:rollout-syn-001", "--dry-run", "--json"},
		"Claude fork": {"fork", "claude:claude-syn-macos", "--dry-run", "--json"},
		"Codex fork":  {"fork", "codex:rollout-syn-001", "--dry-run", "--json"},
	} {
		raw := requireOK(name, args...)
		var plan sessionindex.LaunchPlan
		if err := json.Unmarshal([]byte(raw), &plan); err != nil {
			t.Fatalf("%s plan JSON: %v\n%s", name, err, raw)
		}
		if plan.Executable == "" ||
			len(plan.Args) == 0 ||
			plan.Dir != "/Users/fixture-user/code/demo" {
			t.Fatalf("%s incomplete plan: %+v", name, plan)
		}
	}
	if len(runner.plans) != 0 {
		t.Fatalf("dry-run flow launched a vendor: %+v", runner.plans)
	}

	requireOK("native delegate", "resume", "claude:claude-syn-macos")
	if len(runner.plans) != 1 ||
		runner.plans[0].SessionRef != "claude:claude-syn-macos" {
		t.Fatalf("native delegation plans: %+v", runner.plans)
	}

	for _, forbidden := range []string{"config.toml", "state.json", "device.json", "backups"} {
		if _, err := os.Stat(filepath.Join(home, forbidden)); !os.IsNotExist(err) {
			t.Fatalf("Phase 2 flow created sync state %s: %v", forbidden, err)
		}
	}
	if _, err := os.Stat(sessionindex.IndexPath(home)); err != nil {
		t.Fatalf("Phase 2 flow did not create derived index: %v", err)
	}
}
