package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestHandoffCommandRegistersFullSurface(t *testing.T) {
	root := NewRoot(Options{Name: "rein", TerminalChecker: func(io.Reader, io.Writer) bool { return false }})
	cmd, _, err := root.Find([]string{"handoff"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"last", "from", "to", "policy", "dry-run", "json", "no-launch", "export",
		"allow-warning", "allow-active", "allow-untested", "show-redactions",
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

func TestHandoffPlanPrintsGrokDestinationWarning(t *testing.T) {
	const warning = "Grok conversations are uploaded by the destination CLI under its documented behavior."
	plan := handoff.PlanResult{
		HandoffID: "grok-warning",
		Capsule: capsule.Capsule{
			RawSource: capsule.RawSource{Agent: sessionindex.AgentGrok, SessionID: "synthetic"},
			Security:  capsule.Security{DestinationWarning: warning, RedactionForced: true},
		},
		Destination: handoff.DestinationPlan{Agent: sessionindex.AgentCodex, Executable: "codex", Args: []string{"brief"}},
	}

	var human bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&human)
	if err := writeHandoffPlan(cmd, plan, false, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(human.String(), "destination warning "+warning) {
		t.Fatalf("human output omitted Grok destination warning: %s", human.String())
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
	if output.Security.DestinationWarning != warning || !output.Security.RedactionForced {
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
		_, _, code := runHandoffCLI(t, home, t.TempDir(), sources, &recordingLaunchRunner{},
			"handoff", "codex:source-session", "--to", "claude", "--dry-run")
		if code != ExitCompatibility {
			t.Fatalf("exit=%d want %d", code, ExitCompatibility)
		}
	})

	t.Run("safety", func(t *testing.T) {
		t.Setenv("REINSTATE_HOME", home)
		t.Setenv("HOME", vendorHome)
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
		t.Setenv("HOME", vendorHome)
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
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("export mode=%#o want 0600", info.Mode().Perm())
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
	if err := os.WriteFile(filepath.Join(claudeRoot, "version"), []byte("2.1.227\n"), 0o600); err != nil {
		t.Fatal(err)
	}
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

func runHandoffCLI(t *testing.T, home, vendorHome string, sources []sessionindex.Source, runner sessionindex.LaunchRunner, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("HOME", vendorHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(""), Args: args,
		SessionSources: sources, SessionLaunchRunner: runner, PreflightVerifier: readyPreflightVerifier{},
		AgentProcessChecker: func(context.Context, string, processcheck.Target) (bool, bool, error) {
			return false, true, nil
		},
		TerminalChecker: func(io.Reader, io.Writer) bool { return false },
	})
	return stdout.String(), stderr.String(), code
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
