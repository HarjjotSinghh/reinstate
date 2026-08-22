package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// fakeAgentBin installs deterministic `claude` and `codex` executables on PATH.
// Version probes must never read the contributor's installed agents.
func fakeAgentBin(t *testing.T, versions map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, line := range versions {
		if runtime.GOOS == "windows" {
			body := "@echo off\r\necho " + line + "\r\n"
			if err := os.WriteFile(filepath.Join(dir, name+".cmd"), []byte(body), 0o700); err != nil {
				t.Fatal(err)
			}
			continue
		}
		body := "#!/bin/sh\nprintf '%s\\n' '" + line + "'\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
}

// outsideRootPath is a stable synthetic path that belongs to no configured
// root. It is deliberately low-entropy: a temp-directory path would be masked
// by secret redaction, which is exactly how the rc.1 fixtures hid this defect.
func outsideRootPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Users\fixture-user\notes\todo.md`
	}
	return "/Users/fixture-user/notes/todo.md"
}

// realisticClaudeInstall plants a Claude Code source session shaped like a real
// installation: projects/<encoded>/<uuid>.jsonl, no <root>/version file (Claude
// Code never writes one), and absolute workspace paths inside tool inputs.
func realisticClaudeInstall(t *testing.T, vendorHome, workspace string) sessionindex.Record {
	t.Helper()
	encoded := strings.ReplaceAll(strings.Trim(filepath.ToSlash(workspace), "/"), "/", "-")
	encoded = strings.ReplaceAll(encoded, ":", "-")
	projects := filepath.Join(vendorHome, ".claude", "projects", "-"+encoded)
	if err := os.MkdirAll(projects, 0o700); err != nil {
		t.Fatal(err)
	}
	sessionID := "00000000-0000-4000-8000-0000000000ab"
	path := filepath.Join(projects, sessionID+".jsonl")
	calc := filepath.ToSlash(filepath.Join(workspace, "calc.go"))
	body := fmt.Sprintf(
		`{"type":"user","uuid":"u1","sessionId":%q,"cwd":%q,"timestamp":"2026-08-12T09:00:00Z","message":{"role":"user","content":"Fix the rounding bug in calc.go"}}`+"\n"+
			`{"type":"assistant","uuid":"a1","sessionId":%q,"timestamp":"2026-08-12T09:00:01Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"call_read","name":"Read","input":{"file_path":%q}}]}}`+"\n"+
			`{"type":"user","uuid":"u2","sessionId":%q,"timestamp":"2026-08-12T09:00:02Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_read","content":"package demo"}]}}`+"\n"+
			`{"type":"assistant","uuid":"a2","sessionId":%q,"timestamp":"2026-08-12T09:00:03Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"call_notes","name":"Read","input":{"file_path":%q}}]}}`+"\n"+
			`{"type":"user","uuid":"u3","sessionId":%q,"timestamp":"2026-08-12T09:00:04Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_notes","content":"- rounding"}]}}`+"\n"+
			`{"type":"assistant","uuid":"a3","sessionId":%q,"timestamp":"2026-08-12T09:00:05Z","message":{"role":"assistant","content":[{"type":"text","text":"Rounding fixed."}]}}`+"\n",
		sessionID, workspace, sessionID, calc, sessionID, sessionID, outsideRootPath(), sessionID, sessionID)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vendorHome, ".claude", "version")); err == nil {
		t.Fatal("fixture wrote a vendor version file; real Claude Code installs do not have one")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return sessionindex.Record{
		ID: sessionID, Agent: sessionindex.AgentClaude, Project: "github.com/example/demo",
		Workspace: workspace, UpdatedAt: info.ModTime(), SourcePath: path,
		SourceModTime: info.ModTime().UnixNano(), SourceSize: info.Size(), SizeBytes: info.Size(),
	}
}

// TestHandoffFromRealisticClaudeInstall is the rc.1 acceptance path: a Claude
// Code source session on a host with no vendor version file and absolute paths
// in its tool inputs must plan a Codex handoff successfully.
func TestHandoffFromRealisticClaudeInstall(t *testing.T) {
	home := t.TempDir()
	vendorHome := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vendorHome, ".codex", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	fakeAgentBin(t, map[string]string{
		"claude": "2.1.228 (Claude Code)",
		"codex":  "codex-cli 0.147.0",
	})
	record := realisticClaudeInstall(t, vendorHome, workspace)
	sources := []sessionindex.Source{
		staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}},
		staticSessionSource{name: sessionindex.AgentCodex, result: sessionindex.ScanResult{}},
	}

	// Both roots are named, so detection recognises the planted layouts instead
	// of racing a child process for a version string this test does not assert.
	stdout, stderr, code := runHandoffCLIWithVendorRoots(t, home, vendorHome,
		filepath.Join(vendorHome, ".claude"), filepath.Join(vendorHome, ".codex"),
		sources, nil,
		"handoff", "claude:"+record.ID, "--to", "codex", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}

	var plan handoffPlanOutput
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("plan json: %v (stdout=%s)", err, stdout)
	}
	if plan.Source.Agent != sessionindex.AgentClaude || plan.Destination.Agent != sessionindex.AgentCodex {
		t.Fatalf("unexpected endpoints: %+v -> %+v", plan.Source, plan.Destination)
	}
	if plan.HandoffID == "" || plan.Parse.Events == 0 {
		t.Fatalf("empty plan: id=%q parse=%+v", plan.HandoffID, plan.Parse)
	}
	if !strings.HasPrefix(plan.Workspace.Root, "${REPO:") {
		t.Fatalf("workspace root is not portable: %q", plan.Workspace.Root)
	}
	if strings.Contains(stdout, outsideRootPath()) {
		t.Fatalf("plan leaked an outside-root absolute path: %s", stdout)
	}
}

// TestHandoffFromClaudeInstallWithoutExecutable proves a source whose agent is
// not installed at all — closed, uninstalled, or simply off PATH — is still a
// usable handoff source. This is the core continuity promise.
func TestHandoffFromClaudeInstallWithoutExecutable(t *testing.T) {
	home := t.TempDir()
	vendorHome := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vendorHome, ".codex", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Only Codex is installed; the Claude source executable is unavailable.
	fakeAgentBin(t, map[string]string{"codex": "codex-cli 0.147.0"})
	record := realisticClaudeInstall(t, vendorHome, workspace)
	sources := []sessionindex.Source{
		staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}},
		staticSessionSource{name: sessionindex.AgentCodex, result: sessionindex.ScanResult{}},
	}

	// Both roots are named, so detection recognises the planted layouts instead
	// of racing a child process for a version string this test does not assert.
	stdout, stderr, code := runHandoffCLIWithVendorRoots(t, home, vendorHome,
		filepath.Join(vendorHome, ".claude"), filepath.Join(vendorHome, ".codex"),
		sources, nil,
		"handoff", "claude:"+record.ID, "--to", "codex", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("uninstalled source agent blocked the handoff: exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

// TestHandoffFromClaudeInstallOutsideVerifiedRange keeps the fail-closed half
// of the contract: a version that IS determinable and IS outside the verified
// range still refuses without --allow-untested.
func TestHandoffFromClaudeInstallOutsideVerifiedRange(t *testing.T) {
	home := t.TempDir()
	vendorHome := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vendorHome, ".codex", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	fakeAgentBin(t, map[string]string{
		"claude": "9.9.9 (Claude Code)",
		"codex":  "codex-cli 0.147.0",
	})
	record := realisticClaudeInstall(t, vendorHome, workspace)
	sources := []sessionindex.Source{
		staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}},
		staticSessionSource{name: sessionindex.AgentCodex, result: sessionindex.ScanResult{}},
	}

	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "claude:"+record.ID, "--to", "codex", "--dry-run", "--json")
	if code != ExitCompatibility {
		t.Fatalf("out-of-range source exit=%d, want %d (stdout=%s stderr=%s)", code, ExitCompatibility, stdout, stderr)
	}
	if !strings.Contains(stderr, "UNTESTED") {
		t.Fatalf("stderr did not name the compatibility state: %s", stderr)
	}

	stdout, stderr, code = runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "claude:"+record.ID, "--to", "codex", "--dry-run", "--json", "--allow-untested")
	if code != ExitOK {
		t.Fatalf("--allow-untested exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestHandoffFromClaudeInstallJustOutsideVerifiedRange(t *testing.T) {
	home := t.TempDir()
	vendorHome := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vendorHome, ".codex", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	fakeAgentBin(t, map[string]string{
		"claude": "2.1.239 (Claude Code)",
		"codex":  "codex-cli 0.147.0",
	})
	record := realisticClaudeInstall(t, vendorHome, workspace)
	sources := []sessionindex.Source{
		staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}},
		staticSessionSource{name: sessionindex.AgentCodex, result: sessionindex.ScanResult{}},
	}

	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "claude:"+record.ID, "--to", "codex", "--dry-run", "--json")
	if code != ExitCompatibility {
		t.Fatalf("2.1.239 source exit=%d, want %d (stdout=%s stderr=%s)", code, ExitCompatibility, stdout, stderr)
	}
	if !strings.Contains(stderr, "UNTESTED") {
		t.Fatalf("stderr did not name the compatibility state: %s", stderr)
	}
}
