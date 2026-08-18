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

// The inventory feeds the Phase 5 device reports, so "installed" must mean the
// executable is on PATH. A directory another tool planted under the home
// directory is not an installation and must not promote a row.
func TestDoctorAgentsInstalledTracksExecutableNotPlantedDirectory(t *testing.T) {
	home := isolateHome(t)
	for _, dir := range []string{".qwen", ".openhands", ".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(home, dir, "skills"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out, errb, code := runCLI(t, "rein", "doctor", "--agents")
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	if !strings.Contains(out, "key\ttier\tinstalled\troot\tsessions\tnotes") {
		t.Fatalf("missing header: %q", out)
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 4 || fields[0] == "key" {
			continue
		}
		if fields[2] != "no" || fields[3] != "no" {
			t.Fatalf("planted directory reported as installed/root: %q", line)
		}
	}
}

func TestDoctorAgentsJSONIsProbeArtifact(t *testing.T) {
	home := isolateHome(t)
	// A resolvable tree whose bucket embeds the account name, the shape Kimi
	// Code uses. Without it this assertion would have nothing to redact.
	bucket := filepath.Join(home, ".codex", "sessions", "wd_"+testAccountName+"_ab12cd34-17")
	if err := os.MkdirAll(bucket, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bucket, "rollout.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
	if realHome, _ := os.UserHomeDir(); realHome != "" && strings.Contains(out, realHome) {
		t.Fatalf("probe leaked home path %q", realHome)
	}
	if strings.Contains(out, "/Users/") || strings.Contains(out, `C:\Users\`) {
		t.Fatalf("probe leaked absolute user path: %s", out)
	}
	// Vendors embed the account name in bucket directory names, so an absolute
	// path is not the only way it reaches a committed artifact.
	if strings.Contains(strings.ToLower(out), testAccountName) {
		t.Fatalf("probe leaked the account name %q: %s", testAccountName, out)
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

// testAccountName stands in for the operating-system account name so leak
// assertions have a distinctive token to search for.
const testAccountName = "probetestuser"

func isolateHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), testAccountName)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("REINSTATE_HOME", filepath.Join(home, ".reinstate"))
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("GEMINI_CLI_HOME", "")
	t.Setenv("GROK_HOME", "")
	t.Setenv("PATH", filepath.Join(home, "bin"))
	return home
}
