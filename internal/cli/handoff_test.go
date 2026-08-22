package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func TestHandoffCommandRegistersFullSurface(t *testing.T) {
	root := NewRoot(Options{Name: "rein", TerminalChecker: func(io.Reader, io.Writer) bool { return false }})
	cmd, _, err := root.Find([]string{"handoff"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"last", "from", "to", "policy", "dry-run", "json", "no-launch", "export",
		"allow-warning", "allow-active", "allow-untested", "show-redactions", "no-redact",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("handoff flag --%s is not registered", name)
		}
	}
	for _, name := range []string{"list", "inspect", "export"} {
		if child, _, err := root.Find([]string{"handoff", name}); err != nil || child.Name() != name {
			t.Errorf("handoff %s is not registered: child=%v err=%v", name, child, err)
		}
	}
}

func TestHandoffNoLaunchSpawnsNothingAndMatchesDryRun(t *testing.T) {
	home, vendorHome, sources, transcriptPath := handoffCLIFixture(t)
	runner := &recordingLaunchRunner{}

	dryOut, dryErr, code := runHandoffCLI(t, home, vendorHome, sources, runner,
		"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("dry-run exit=%d stdout=%s stderr=%s", code, dryOut, dryErr)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("dry-run spawned %d processes", len(runner.plans))
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote outside temporary directories: %+v", entries)
	}

	before, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	noLaunchOut, noLaunchErr, code := runHandoffCLI(t, home, vendorHome, sources, runner,
		"handoff", "codex:source-session", "--to", "claude", "--no-launch", "--json")
	if code != ExitOK {
		t.Fatalf("no-launch exit=%d stdout=%s stderr=%s", code, noLaunchOut, noLaunchErr)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("no-launch spawned %d processes", len(runner.plans))
	}
	if dryOut != noLaunchOut {
		t.Fatalf("dry-run/no-launch output differs\ndry: %s\nno-launch: %s", dryOut, noLaunchOut)
	}
	after, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("handoff mutated source transcript")
	}

	var output handoffPlanOutput
	if err := json.Unmarshal([]byte(noLaunchOut), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, noLaunchOut)
	}
	if output.Mode != handoffMode || output.DestinationSessionMode != "new" ||
		output.Source.Agent != "codex" || output.Destination.Agent != "claude" {
		t.Fatalf("unexpected handoff identity: %+v", output)
	}
	if output.Destination.Executable != "claude" || len(output.Destination.Args) != 3 ||
		output.Destination.Args[0] != "--session-id" {
		t.Fatalf("destination command = %+v", output.Destination)
	}
	wantLineage := filepath.Join(home, "handoffs", "lineage.jsonl")
	if !containsString(output.PlannedFiles, wantLineage) {
		t.Fatalf("planned files omit lineage append %q: %#v", wantLineage, output.PlannedFiles)
	}
	store, err := handoff.OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	c, _, err := store.Get(output.HandoffID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Identity.ID != output.HandoffID {
		t.Fatalf("stored handoff=%q want %q", c.Identity.ID, output.HandoffID)
	}
	lineage, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 1 || lineage[0].Launched {
		t.Fatalf("no-launch lineage = %+v", lineage)
	}
}

func TestHandoffClaudeCollisionCheckRequiresFreshSource(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	sources[1] = staticSessionSource{name: sessionindex.AgentClaude, err: os.ErrPermission}
	runner := &recordingLaunchRunner{}
	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, runner,
		"handoff", "codex:source-session", "--to", "claude")
	if code != ExitRuntime {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("failed Claude refresh spawned %d processes", len(runner.plans))
	}
}

func TestHandoffCLIErrorMapsWrappedLaunchCauses(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		cause    error
		wantCode int
	}{
		{name: "non interactive", code: ExitRuntime, cause: sessionindex.ErrNonInteractiveLaunch, wantCode: ExitSafety},
		{name: "boundary changed", code: ExitRuntime, cause: sessionindex.ErrLaunchBoundaryChanged, wantCode: ExitSafety},
		{name: "unsupported action", code: ExitRuntime, cause: sessionindex.ErrNativeActionUnsupported, wantCode: ExitCompatibility},
		{name: "executable absent", code: ExitRuntime, cause: sessionindex.ErrExecutableNotFound, wantCode: ExitCompatibility},
		{name: "workspace absent", code: ExitRuntime, cause: sessionindex.ErrWorkspaceUnavailable, wantCode: ExitCompatibility},
		{name: "explicit code preserved", code: ExitCompatibility, cause: sessionindex.ErrNonInteractiveLaunch, wantCode: ExitCompatibility},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := handoffCLIError(&handoff.PipelineError{Code: test.code, Err: fmt.Errorf("wrapped: %w", test.cause)})
			exitError, ok := err.(*ExitError)
			if !ok || exitError.Code != test.wantCode {
				t.Fatalf("error=%T %+v want exit %d", err, err, test.wantCode)
			}
		})
	}
	direct := handoffCLIError(fmt.Errorf("direct: %w", sessionindex.ErrExecutableNotFound))
	if exitError, ok := direct.(*ExitError); !ok || exitError.Code != ExitCompatibility {
		t.Fatalf("direct mapped error=%T %+v", direct, direct)
	}
}

func TestHandoffHumanOutputAlwaysStatesMode(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{},
		"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--show-redactions")
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if !strings.HasPrefix(line, "Structured handoff — a new claude session, not native resume:") {
			t.Fatalf("human line lacks mode contract: %q", line)
		}
	}
	if !strings.Contains(stdout, ": command \"claude\"") {
		t.Fatalf("exact command missing: %s", stdout)
	}
}

func TestHandoffNoRedactIsAKnownFlag(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{},
		"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--no-redact")
	if strings.Contains(stderr, "unknown flag") {
		t.Fatalf("--no-redact still unknown: stderr=%s", stderr)
	}
	if code != ExitOK {
		t.Fatalf("codex --no-redact exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestHandoffGrokNoRedactIsRefused(t *testing.T) {
	home, vendorHome, sources, sessionDir := grokHandoffCLIFixture(t)
	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{},
		"handoff", "grok:01987654-basic-0000-0000-000000000001", "--to", "claude", "--dry-run", "--no-redact")
	if strings.Contains(stderr, "unknown flag") {
		t.Fatalf("--no-redact still unknown: stderr=%s", stderr)
	}
	if code != ExitUsage {
		t.Fatalf("grok --no-redact exit=%d stdout=%s stderr=%s session=%s", code, stdout, stderr, sessionDir)
	}
	if !strings.Contains(strings.ToLower(stderr), "no-redact") && !strings.Contains(stderr, "Grok") {
		t.Fatalf("grok --no-redact stderr=%s", stderr)
	}
}

func TestHandoffNonTTYAllowWarningRefusesBeforeSpawn(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "")
	hanging := t.TempDir()
	writeHangingHandoffShim(t, hanging, "claude")
	t.Setenv("PATH", hanging+string(os.PathListSeparator)+os.Getenv("PATH"))
	start := time.Now()
	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "codex:source-session", "--to", "claude", "--allow-warning", "baseline.unavailable")
	elapsed := time.Since(start)
	if code != ExitSafety {
		t.Fatalf("non-TTY launch exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "interactive terminal") {
		t.Fatalf("non-TTY stderr=%s", stderr)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("non-TTY refuse took %s; must happen before LookPath/version delay", elapsed)
	}
}

func writeHangingHandoffShim(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = os.Getenv("SYSTEMROOT")
		}
		if root == "" {
			t.Skip("SystemRoot is unset")
		}
		ping := filepath.Join(root, "System32", "ping.exe")
		path := filepath.Join(dir, name+".cmd")
		body := "@echo off\r\n\"" + ping + "\" -n 30 127.0.0.1 >nul\r\n"
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
		return
	}
	sleepBinary := "/bin/sleep"
	if _, err := os.Stat(sleepBinary); err != nil {
		sleepBinary = "/usr/bin/sleep"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+sleepBinary+" 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func grokHandoffCLIFixture(t *testing.T) (string, string, []sessionindex.Source, string) {
	t.Helper()
	home := t.TempDir()
	vendorHome := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vendorHome, ".claude", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	fakeAgentBin(t, map[string]string{"claude": "2.1.227 (Claude Code)"})
	sessionDir := filepath.Join(workspace, "grok-session")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	summary := `{
  "info": {"id": "01987654-basic-0000-0000-000000000001", "cwd": "` + filepath.ToSlash(workspace) + `"},
  "session_summary": "Synthetic Grok CLI no-redact session",
  "created_at": "2026-08-12T02:00:00Z",
  "updated_at": "2026-08-12T02:05:00Z",
  "num_messages": 1
}`
	if err := os.WriteFile(filepath.Join(sessionDir, "summary.json"), []byte(summary), 0o600); err != nil {
		t.Fatal(err)
	}
	history := `{"type":"user","content":[{"type":"text","text":"Grok CLI no-redact prompt"}],"prompt_index":0}` + "\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "chat_history.jsonl"), []byte(history), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatal(err)
	}
	record := sessionindex.Record{
		ID: "01987654-basic-0000-0000-000000000001", Agent: sessionindex.AgentGrok,
		Project: "controlled-project", Workspace: workspace, UpdatedAt: info.ModTime(),
		SourcePath: sessionDir, SourceModTime: info.ModTime().UnixNano(),
		SourceSize: info.Size(), SizeBytes: info.Size(),
	}
	sources := []sessionindex.Source{
		staticSessionSource{name: sessionindex.AgentGrok, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}},
		staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{}},
		staticSessionSource{name: sessionindex.AgentCodex, result: sessionindex.ScanResult{}},
	}
	return home, vendorHome, sources, sessionDir
}

func TestHandoffRefusesWrongGitRepository(t *testing.T) {
	home, vendorHome, sources, transcriptPath := handoffCLIFixture(t)
	workspace := filepath.Dir(transcriptPath)
	initCLIGitRepo(t, workspace)
	other := t.TempDir()
	initCLIGitRepo(t, other)
	t.Setenv("REINSTATE_HOME", home)
	setVendorHome(t, vendorHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	t.Chdir(other)
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(""),
		Args:           []string{"handoff", "codex:source-session", "--to", "claude", "--dry-run"},
		SessionSources: sources, SessionLaunchRunner: &recordingLaunchRunner{},
		PreflightVerifier: readyPreflightVerifier{},
		AgentProcessChecker: func(context.Context, string, processcheck.Target) (bool, bool, error) {
			return false, true, nil
		},
		TerminalChecker: func(io.Reader, io.Writer) bool { return false },
	})
	if code != ExitCompatibility {
		t.Fatalf("wrong-repo exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "different repository") {
		t.Fatalf("wrong-repo stderr=%s", stderr.String())
	}
}

func initCLIGitRepo(t *testing.T, dir string) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	command := exec.Command(git, "init", "--quiet", dir)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v: %s", dir, err, output)
	}
}

func TestHandoffPlanPrintsGrokDestinationWarning(t *testing.T) {
	plan := handoff.PlanResult{
		HandoffID: "grok-warning",
		Capsule: capsule.Capsule{
			RawSource: capsule.RawSource{Agent: sessionindex.AgentGrok, SessionID: "synthetic"},
			Security:  capsule.Security{DestinationWarning: transcript.DestinationWarningGrok, RedactionForced: true},
		},
		Destination: handoff.DestinationPlan{Agent: sessionindex.AgentCodex, Executable: "codex", Args: []string{"brief"}},
	}

	var human bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&human)
	if err := writeHandoffPlan(cmd, plan, false, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"destination warning Grok Build", "repository-content upload", "Git history", ".env", "xAI cloud storage", "forced capsule redaction"} {
		if !strings.Contains(human.String(), want) {
			t.Fatalf("human output omitted %q from Grok warning: %s", want, human.String())
		}
	}
	if strings.Contains(human.String(), "destination warning "+transcript.DestinationWarningGrok) {
		t.Fatalf("human output exposed machine warning ID instead of explicit prose: %s", human.String())
	}

	var machine bytes.Buffer
	cmd.SetOut(&machine)
	if err := writeHandoffPlan(cmd, plan, true, false); err != nil {
		t.Fatal(err)
	}
	var output handoffPlanOutput
	if err := json.Unmarshal(machine.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Security.DestinationWarning != transcript.DestinationWarningGrok || !output.Security.RedactionForced {
		t.Fatalf("machine security output=%+v", output.Security)
	}
}

func TestProductionHandoffRunnerPinsIdentitiesAndFinalGuard(t *testing.T) {
	binDir := t.TempDir()
	executableName := "claude"
	if runtime.GOOS == "windows" {
		executableName += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	executable := filepath.Join(binDir, executableName)
	if err := os.WriteFile(executable, []byte("controlled executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	workspace := t.TempDir()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	runner, err := hardenedHandoffRunner(cmd, sessionindex.Record{}, handoff.Options{}, handoff.PlanResult{
		Destination: handoff.DestinationPlan{Agent: "claude", Executable: "claude", Dir: workspace},
	})
	if err != nil {
		t.Fatal(err)
	}
	typed, ok := runner.(sessionindex.ExecLaunchRunner)
	if !ok {
		t.Fatalf("runner type=%T", runner)
	}
	if !filepath.IsAbs(typed.Executable) || typed.ExecutableIdentity.IsZero() ||
		typed.WorkspaceIdentity.IsZero() || typed.BeforeExec == nil {
		t.Fatalf("runner guards incomplete: %+v", typed)
	}
}

func TestHandoffUsageAndAmbiguityExitCodes(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	cases := []struct {
		name string
		args []string
		want int
	}{
		{name: "missing to", args: []string{"handoff", "codex:source-session", "--dry-run"}, want: ExitUsage},
		{name: "json launch", args: []string{"handoff", "codex:source-session", "--to", "claude", "--json"}, want: ExitUsage},
		{name: "last plus session", args: []string{"handoff", "codex:source-session", "--last", "--to", "claude", "--dry-run"}, want: ExitUsage},
		{name: "dry run export", args: []string{"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--export", "projection.md"}, want: ExitUsage},
		{name: "unknown destination", args: []string{"handoff", "codex:source-session", "--to", "gemini", "--dry-run"}, want: ExitUsage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, got := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{}, tc.args...)
			if got != tc.want {
				t.Fatalf("exit=%d want %d", got, tc.want)
			}
		})
	}

	duplicate := append([]sessionindex.Source{}, sources...)
	duplicate = append(duplicate, staticSessionSource{name: sessionindex.AgentGemini, result: sessionindex.ScanResult{Records: []sessionindex.Record{{
		ID: "source-session", Agent: sessionindex.AgentGemini, Workspace: t.TempDir(), UpdatedAt: time.Now().UTC(), SourcePath: filepath.Join(t.TempDir(), "session.json"),
	}}}})
	_, _, code := runHandoffCLI(t, t.TempDir(), vendorHome, duplicate, &recordingLaunchRunner{},
		"handoff", "source-session", "--to", "claude", "--dry-run")
	if code != ExitConflict {
		t.Fatalf("ambiguous reference exit=%d want %d", code, ExitConflict)
	}
}

func TestHandoffPipelineExitCodes(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)

	t.Run("config", func(t *testing.T) {
		t.Setenv("REINSTATE_HOME", "relative")
		var stderr bytes.Buffer
		code := Execute(Options{Name: "rein", Stderr: &stderr, Args: []string{"handoff", "list"}})
		if code != ExitConfig {
			t.Fatalf("exit=%d stderr=%s", code, stderr.String())
		}
	})

	t.Run("compatibility", func(t *testing.T) {
		// Real detection, on purpose: this subtest asserts what detection
		// concludes. It is deterministic because an empty vendor home resolves
		// to no Claude root at all, which short-circuits to NotInstalled
		// without ever spawning a version probe.
		_, _, code := runHandoffCLIProbingDestination(t, home, t.TempDir(), sources, &recordingLaunchRunner{},
			"handoff", "codex:source-session", "--to", "claude", "--dry-run")
		if code != ExitCompatibility {
			t.Fatalf("exit=%d want %d", code, ExitCompatibility)
		}
	})

	t.Run("safety", func(t *testing.T) {
		t.Setenv("REINSTATE_HOME", home)
		setVendorHome(t, vendorHome)
		var stdout, stderr bytes.Buffer
		code := Execute(Options{
			Name: "rein", Stdout: &stdout, Stderr: &stderr,
			Args:           []string{"handoff", "codex:source-session", "--to", "claude", "--dry-run"},
			SessionSources: sources, SessionLaunchRunner: &recordingLaunchRunner{}, PreflightVerifier: readyPreflightVerifier{},
			AgentProcessChecker: func(context.Context, string, processcheck.Target) (bool, bool, error) {
				return true, true, nil
			},
		})
		if code != ExitSafety {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
	})

	t.Run("runtime", func(t *testing.T) {
		t.Setenv("REINSTATE_HOME", home)
		setVendorHome(t, vendorHome)
		var stderr bytes.Buffer
		code := Execute(Options{
			Name: "rein", Stderr: &stderr,
			Args:           []string{"handoff", "codex:source-session", "--to", "claude", "--dry-run"},
			SessionSources: sources, SessionLaunchRunner: &recordingLaunchRunner{}, PreflightVerifier: failingHandoffVerifier{},
			AgentProcessChecker: func(context.Context, string, processcheck.Target) (bool, bool, error) {
				return false, true, nil
			},
		})
		if code != ExitRuntime {
			t.Fatalf("exit=%d stderr=%s", code, stderr.String())
		}
	})
}

type failingHandoffVerifier struct{}

func (failingHandoffVerifier) Verify(context.Context, preflight.Input) (preflight.Report, error) {
	return preflight.Report{}, os.ErrPermission
}

func TestHandoffListInspectAcknowledgeAndExport(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	createdJSON, stderr, code := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{},
		"handoff", "codex:source-session", "--to", "claude", "--no-launch", "--json")
	if code != ExitOK {
		t.Fatalf("create exit=%d stdout=%s stderr=%s", code, createdJSON, stderr)
	}
	var created handoffPlanOutput
	if err := json.Unmarshal([]byte(createdJSON), &created); err != nil {
		t.Fatal(err)
	}

	listedJSON, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "list", "--json", "--limit", "1")
	if code != ExitOK {
		t.Fatalf("list exit=%d stdout=%s stderr=%s", code, listedJSON, stderr)
	}
	var listed handoffListOutput
	if err := json.Unmarshal([]byte(listedJSON), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Handoffs) != 1 || listed.Handoffs[0].HandoffID != created.HandoffID {
		t.Fatalf("listed=%+v", listed)
	}

	inspectedJSON, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "inspect", created.HandoffID, "--acknowledged", "--json")
	if code != ExitOK {
		t.Fatalf("inspect exit=%d stdout=%s stderr=%s", code, inspectedJSON, stderr)
	}
	var inspected handoffInspectOutput
	if err := json.Unmarshal([]byte(inspectedJSON), &inspected); err != nil {
		t.Fatal(err)
	}
	if inspected.Acknowledgement.Confirmed == nil || !*inspected.Acknowledgement.Confirmed {
		t.Fatalf("acknowledgement=%+v", inspected.Acknowledgement)
	}
	if inspected.Artifacts.ProjectionBytes == 0 || inspected.Artifacts.ProjectionSHA256 == "" {
		t.Fatalf("artifact metadata=%+v", inspected.Artifacts)
	}

	exported, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "export", created.HandoffID, "--format", "markdown")
	if code != ExitOK {
		t.Fatalf("export exit=%d stdout=%s stderr=%s", code, exported, stderr)
	}
	if !strings.HasPrefix(exported, "# Structured handoff — not native resume\n") || strings.HasSuffix(exported, "\n\n") {
		t.Fatalf("unexpected markdown export: %q", exported)
	}

	out := filepath.Join(t.TempDir(), "capsule.json")
	_, stderr, code = runHandoffCLI(t, home, vendorHome, sources, nil,
		"handoff", "export", created.HandoffID, "--format", "json", "--out", out)
	if code != ExitOK {
		t.Fatalf("file export exit=%d stderr=%s", code, stderr)
	}
	private, detail, err := fsx.OwnerOnly(out, false)
	if err != nil {
		t.Fatal(err)
	}
	if !private {
		t.Fatalf("exported capsule is not owner-only: %s", detail)
	}
}

func handoffCLIFixture(t *testing.T) (string, string, []sessionindex.Source, string) {
	t.Helper()
	home := t.TempDir()
	vendorHome := t.TempDir()
	workspace := t.TempDir()
	claudeRoot := filepath.Join(vendorHome, ".claude")
	if err := os.MkdirAll(filepath.Join(claudeRoot, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	// A real Claude Code installation has no <root>/version file, so the
	// destination adapter learns its version the way it does on a real host:
	// by asking the executable. The fake keeps that hermetic.
	fakeAgentBin(t, map[string]string{"claude": "2.1.227 (Claude Code)"})
	transcriptPath := filepath.Join(workspace, "rollout-source-session.jsonl")
	body := []byte("{\"timestamp\":\"2026-08-12T09:00:00Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"user_message\",\"message\":\"Continue the controlled task\"}}\n")
	if err := os.WriteFile(transcriptPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	record := sessionindex.Record{
		ID: "source-session", Agent: sessionindex.AgentCodex, Project: "controlled-project",
		Workspace: workspace, UpdatedAt: info.ModTime(), SourcePath: transcriptPath,
		SourceModTime: info.ModTime().UnixNano(), SourceSize: info.Size(), SizeBytes: info.Size(),
	}
	sources := []sessionindex.Source{
		staticSessionSource{name: sessionindex.AgentCodex, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}},
		staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{}},
	}
	return home, vendorHome, sources, transcriptPath
}

// setVendorHome points agent detection at a synthetic vendor home. UserHomeDir
// reads HOME on Unix and USERPROFILE on Windows, so both must be set or the
// destination adapter probes the real profile and reports NOT_INSTALLED.
func setVendorHome(t *testing.T, vendorHome string) {
	t.Helper()
	t.Setenv("HOME", vendorHome)
	t.Setenv("USERPROFILE", vendorHome)
}

func runHandoffCLI(t *testing.T, home, vendorHome string, sources []sessionindex.Source, runner sessionindex.LaunchRunner, args ...string) (string, string, int) {
	t.Helper()
	// Pin the destination's compatibility answer. Detection otherwise runs
	// `<agent> --version` as a child process under a two-second bound, and
	// under a saturated `go test ./...` that child does not reliably start in
	// time — measured, a two-line shell script killed at 2.0009s. Detection
	// then correctly reports UNTESTED, and tests that assert nothing about
	// versions fail for a reason they never meant to measure.
	//
	// Tests whose subject *is* compatibility opt out with
	// runHandoffCLIProbingDestination and keep the real probe.
	return runHandoffCLIWith(t, handoffCLIRun{
		home: home, vendorHome: vendorHome, sources: sources, runner: runner,
		destinationCompat: adapter.CompatibilitySupported,
	}, args...)
}

// runHandoffCLIProbingDestination keeps the real destination probe, for tests
// that assert what detection concludes rather than what the handoff plans.
func runHandoffCLIProbingDestination(t *testing.T, home, vendorHome string, sources []sessionindex.Source, runner sessionindex.LaunchRunner, args ...string) (string, string, int) {
	t.Helper()
	return runHandoffCLIWith(t, handoffCLIRun{
		home: home, vendorHome: vendorHome, sources: sources, runner: runner,
	}, args...)
}

type handoffCLIRun struct {
	home              string
	vendorHome        string
	claudeRoot        string
	codexRoot         string
	sources           []sessionindex.Source
	runner            sessionindex.LaunchRunner
	destinationCompat adapter.Compatibility
}

func runHandoffCLIWith(t *testing.T, run handoffCLIRun, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("REINSTATE_HOME", run.home)
	setVendorHome(t, run.vendorHome)
	t.Setenv("CLAUDE_CONFIG_DIR", run.claudeRoot)
	t.Setenv("CODEX_HOME", run.codexRoot)
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(""), Args: args,
		SessionSources: run.sources, SessionLaunchRunner: run.runner,
		PreflightVerifier:        readyPreflightVerifier{},
		HandoffDestinationCompat: run.destinationCompat,
		AgentProcessChecker: func(context.Context, string, processcheck.Target) (bool, bool, error) {
			return false, true, nil
		},
		TerminalChecker: func(io.Reader, io.Writer) bool { return false },
	})
	return stdout.String(), stderr.String(), code
}

// runHandoffCLIWithVendorRoots names the vendor roots explicitly.
//
// Left empty, adapter detection falls back to running `<agent> --version` as a
// child process, and a test that needs that probe to succeed is really asking
// to win a scheduling race: under a saturated `go test ./...` even a two-line
// shell script does not always start within the probe's two-second bound, and
// the run is then correctly reported as UNTESTED. Naming a root that carries
// the expected layout is how a real installation is recognised without a
// subprocess, so a test whose subject is the handoff itself should say so
// rather than depend on ambient machine load.
//
// Tests that deliberately exercise the version probe — the out-of-range ones —
// must keep passing empty roots, and are unaffected by a timeout because a
// probe that cannot measure is UNTESTED, which is what they already expect.
func runHandoffCLIWithVendorRoots(
	t *testing.T, home, vendorHome, claudeRoot, codexRoot string,
	sources []sessionindex.Source, runner sessionindex.LaunchRunner, args ...string,
) (string, string, int) {
	t.Helper()
	return runHandoffCLIWith(t, handoffCLIRun{
		home: home, vendorHome: vendorHome, claudeRoot: claudeRoot, codexRoot: codexRoot,
		sources: sources, runner: runner,
	}, args...)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// TestPinnedDestinationConsultsNoVendorBinary proves the seam removes the child
// process rather than merely making it faster.
//
// The flake it exists for is a spawn race: adapter detection runs
// `<agent> --version` under a hard two-second bound, and under a saturated
// `go test ./...` even a two-line shell script does not reliably start in time
// — measured, killed at 2.0009s. Detection then correctly reports UNTESTED and
// a test asserting nothing about versions fails.
//
// A timing assertion could not settle that; it would only be quiet on a quiet
// machine. This removes the binary from PATH entirely. If detection still
// consulted it, the handoff would refuse with a compatibility exit, so a
// successful plan is proof that nothing was spawned.
func TestPinnedDestinationConsultsNoVendorBinary(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	// An empty PATH: no claude, no codex, nothing to probe.
	t.Setenv("PATH", t.TempDir())

	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{},
		"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("pinned destination still depended on a vendor binary: exit=%d stdout=%s stderr=%s",
			code, stdout, stderr)
	}
	if !strings.Contains(stdout, "\"destination\"") {
		t.Fatalf("plan did not reach the destination stage: %s", stdout)
	}
}

// TestProbingDestinationStillDependsOnDetection is the negative control. Without
// the pin, the same call with no vendor binary on PATH must refuse — otherwise
// the test above would pass for a reason unrelated to the seam.
func TestProbingDestinationStillDependsOnDetection(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	t.Setenv("PATH", t.TempDir())

	_, _, code := runHandoffCLIProbingDestination(t, home, vendorHome, sources, &recordingLaunchRunner{},
		"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--json")
	if code == ExitOK {
		t.Fatal("detection succeeded with no vendor binary on PATH; the pinned test proves nothing")
	}
}
