package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/probe"
)

func TestDoctorAgentsHumanListsCatalog(t *testing.T) {
	isolateHome(t)
	out, errb, code := runCLI(t, "rein", "doctor", "--agents")
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	if !strings.Contains(out, "key\ttier\tinstalled") {
		t.Fatalf("missing header: %q", out)
	}
	for _, key := range agents.Keys() {
		if !strings.Contains(out, key+"\t") {
			t.Fatalf("missing catalog agent %q in %q", key, out)
		}
	}
	if strings.Contains(strings.ToLower(out), "all agents") ||
		strings.Contains(strings.ToLower(out), "universal resume") {
		t.Fatalf("overclaim in inventory: %q", out)
	}
}

func TestDoctorAgentsJSONIsProbeArtifact(t *testing.T) {
	isolateHome(t)
	out, errb, code := runCLI(t, "rein", "doctor", "--agents", "--json")
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	var art probe.Artifact
	if err := json.Unmarshal([]byte(out), &art); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if err := probe.Validate(art); err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.Contains(out, home) {
		t.Fatalf("probe leaked home path %q", home)
	}
	if strings.Contains(out, "/Users/") || strings.Contains(out, `C:\Users\`) {
		t.Fatalf("probe leaked absolute user path: %s", out)
	}
}

func TestDoctorAcceptanceMatrixRowCount(t *testing.T) {
	out, errb, code := runCLI(t, "rein", "doctor", "--agents", "--acceptance-matrix", "--json")
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	var matrix acceptanceMatrix
	if err := json.Unmarshal([]byte(out), &matrix); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	wantCore := matrixARows + matrixBRows + matrixGRows + matrixHRows
	if matrix.CoreRowCount != wantCore || matrix.Core["A"] != matrixARows {
		t.Fatalf("core = %+v", matrix.Core)
	}
	want := wantCore
	for _, agent := range matrix.Agents {
		want += agent.RowCount
	}
	if matrix.RowCount != want {
		t.Fatalf("row_count=%d want %d", matrix.RowCount, want)
	}
	// Contract: T2 => C+D; T5 => C+D+E. Current catalog is five T2+ agents.
	if matrix.RowCount != 100 {
		t.Fatalf("row_count=%d, expected 100 for current catalog (33 core + 17+17+11+11+11)", matrix.RowCount)
	}
}

func TestAgentStorageProbeWrappersInvokeSameFlags(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	sh, err := os.ReadFile(filepath.Join(root, "scripts", "testing", "agent-storage-probe.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ps1, err := os.ReadFile(filepath.Join(root, "scripts", "testing", "agent-storage-probe.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"sh": string(sh), "ps1": string(ps1)} {
		if !strings.Contains(body, "doctor") || !strings.Contains(body, "--agents") || !strings.Contains(body, "--json") {
			t.Fatalf("%s wrapper does not invoke doctor --agents --json", name)
		}
	}
}

func isolateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("REINSTATE_HOME", filepath.Join(home, ".reinstate"))
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("GEMINI_CLI_HOME", "")
	t.Setenv("GROK_HOME", "")
	t.Setenv("PATH", filepath.Join(home, "bin"))
}
