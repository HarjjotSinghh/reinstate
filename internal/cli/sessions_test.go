package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

type readyPreflightVerifier struct{}

func (readyPreflightVerifier) Verify(_ context.Context, input preflight.Input) (preflight.Report, error) {
	return preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		SessionRef:    input.SessionRef,
		Decision:      preflight.DecisionReady,
		Checks: []preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Provenance: workspace.ProvenanceCurrentObservation, Message: "controlled source is fresh",
		}},
		Workspace: workspace.Fingerprint{Git: workspace.GitFingerprint{
			WorkingTree: workspace.WorkingTreeFingerprint{State: workspace.WorkingTreeUnavailable},
		}},
	}, nil
}

type staticSessionSource struct {
	name   string
	result sessionindex.ScanResult
	err    error
}

func (source staticSessionSource) Name() string { return source.name }

func (source staticSessionSource) Scan(context.Context) (sessionindex.ScanResult, error) {
	return source.result, source.err
}

type recordingLaunchRunner struct {
	plans []sessionindex.LaunchPlan
	err   error
}

func (runner *recordingLaunchRunner) Run(
	_ context.Context,
	plan sessionindex.LaunchPlan,
) error {
	runner.plans = append(runner.plans, plan)
	return runner.err
}

func runLocalCLI(
	t *testing.T,
	sources []sessionindex.Source,
	runner sessionindex.LaunchRunner,
	stdin string,
	terminal bool,
	args ...string,
) (string, string, int) {
	t.Helper()
	t.Setenv("REINSTATE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name:                "rein",
		Stdout:              &stdout,
		Stderr:              &stderr,
		Stdin:               strings.NewReader(stdin),
		Args:                args,
		SessionSources:      sources,
		SessionLaunchRunner: runner,
		PreflightVerifier:   readyPreflightVerifier{},
		TerminalChecker: func(io.Reader, io.Writer) bool {
			return terminal
		},
	})
	return stdout.String(), stderr.String(), code
}

func localTestSources(workspace string) []sessionindex.Source {
	newer := time.Date(2026, 7, 30, 8, 2, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	claudePath := filepath.Join(workspace, "claude-session.jsonl")
	codexPath := filepath.Join(workspace, "codex-session.jsonl")
	return []sessionindex.Source{
		staticSessionSource{
			name: sessionindex.AgentClaude,
			result: sessionindex.ScanResult{Records: []sessionindex.Record{{
				ID:            "claude-one",
				Agent:         sessionindex.AgentClaude,
				Title:         "New session",
				Project:       "phase-two",
				Workspace:     workspace,
				Branch:        "phase2/alpha",
				UpdatedAt:     newer,
				SizeBytes:     123,
				MessageCount:  4,
				PromptPreview: "  user \x1b[31mpreview\nSECRET-PREVIEW  ",
				Files:         []string{"src/alpha.go"},
				CanResume:     true,
				CanFork:       true,
				SourcePath:    claudePath,
				SourceModTime: newer.UnixNano(),
				SourceSize:    123,
				SearchText:    "CONTROLLED SEARCH-ONLY-CONTENT unicode-β alpha",
			}}},
		},
		staticSessionSource{
			name: sessionindex.AgentCodex,
			result: sessionindex.ScanResult{Records: []sessionindex.Record{{
				ID:            "codex-two",
				Agent:         sessionindex.AgentCodex,
				Title:         "Older session",
				Project:       "phase-two",
				Workspace:     workspace,
				Branch:        "phase2/beta",
				UpdatedAt:     older,
				SizeBytes:     456,
				MessageCount:  2,
				PromptPreview: "older controlled prompt",
				Files:         []string{"src/beta.go"},
				CanResume:     true,
				CanFork:       true,
				SourcePath:    codexPath,
				SourceModTime: older.UnixNano(),
				SourceSize:    456,
				SearchText:    "older needle beta",
			}}},
		},
	}
}

func TestSessionsAndSearchAreConfiglessAndDoNotPrintPromptText(t *testing.T) {
	workspace := t.TempDir()
	sources := localTestSources(workspace)

	stdout, stderr, code := runLocalCLI(
		t,
		sources,
		nil,
		"",
		false,
		"sessions",
		"--json",
	)
	if code != ExitOK {
		t.Fatalf("sessions exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, forbidden := range []string{
		"SECRET-PREVIEW",
		"SEARCH-ONLY-CONTENT",
		"claude-session.jsonl",
	} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("sessions exposed %q: %s", forbidden, stdout)
		}
	}
	var listed localSessionsOutput
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("decode sessions: %v\n%s", err, stdout)
	}
	if len(listed.Sessions) != 2 {
		t.Fatalf("sessions=%d want 2: %s", len(listed.Sessions), stdout)
	}
	if listed.Sessions[0].Key != "claude:claude-one" {
		t.Fatalf("first=%q want newest Claude", listed.Sessions[0].Key)
	}

	stdout, stderr, code = runLocalCLI(
		t,
		sources,
		nil,
		"",
		false,
		"search",
		"controlled",
		"unicode-β",
		"--agent",
		"claude",
		"--branch",
		"ALPHA",
		"--file",
		"alpha.go",
		"--json",
	)
	if code != ExitOK {
		t.Fatalf("search exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.Contains(stdout, "SEARCH-ONLY-CONTENT") ||
		strings.Contains(stdout, "SECRET-PREVIEW") {
		t.Fatalf("search exposed matched prompt text: %s", stdout)
	}
	var matched localSessionsOutput
	if err := json.Unmarshal([]byte(stdout), &matched); err != nil {
		t.Fatalf("decode search: %v\n%s", err, stdout)
	}
	if len(matched.Sessions) != 1 ||
		matched.Sessions[0].Key != "claude:claude-one" {
		t.Fatalf("unexpected matches: %+v", matched.Sessions)
	}
}

func TestLocalIndexCreatesOnlyPrivateDerivedCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name:           "rein",
		Stdout:         &stdout,
		Stderr:         &stderr,
		Args:           []string{"sessions", "--json"},
		SessionSources: []sessionindex.Source{},
	})
	if code != ExitOK {
		t.Fatalf("sessions exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, forbidden := range []string{"config.toml", "state.json", "device.json", "backups"} {
		if _, err := os.Stat(filepath.Join(home, forbidden)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("configless command created %s: %v", forbidden, err)
		}
	}
	indexPath := sessionindex.IndexPath(home)
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("derived index missing: %v", err)
	}
}

func TestInspectShowsOnlyBoundedSafePreview(t *testing.T) {
	workspace := t.TempDir()
	stdout, stderr, code := runLocalCLI(
		t,
		localTestSources(workspace),
		nil,
		"",
		false,
		"inspect",
		"claude:claude-one",
		"--json",
	)
	if code != ExitOK {
		t.Fatalf("inspect exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var inspected localInspectOutput
	if err := json.Unmarshal([]byte(stdout), &inspected); err != nil {
		t.Fatalf("decode inspect: %v\n%s", err, stdout)
	}
	if inspected.Session.PromptPreview != "user preview SECRET-PREVIEW" {
		t.Fatalf("unsafe preview=%q", inspected.Session.PromptPreview)
	}
	for _, forbidden := range []string{"\x1b", "SEARCH-ONLY-CONTENT", "claude-session.jsonl"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("inspect exposed %q: %s", forbidden, stdout)
		}
	}
}

func TestNativeDryRunLastAndRealLaunch(t *testing.T) {
	workspace := t.TempDir()
	sources := localTestSources(workspace)
	runner := &recordingLaunchRunner{}

	stdout, stderr, code := runLocalCLI(
		t,
		sources,
		runner,
		"",
		false,
		"resume",
		"claude:claude-one",
		"--dry-run",
		"--json",
	)
	if code != ExitOK {
		t.Fatalf("dry resume exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var plan sessionindex.LaunchPlan
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("decode launch plan: %v\n%s", err, stdout)
	}
	if plan.Executable != "claude" ||
		strings.Join(plan.Args, "\x00") != "--resume\x00claude-one" ||
		plan.Dir != workspace {
		t.Fatalf("unexpected launch plan: %+v", plan)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("dry-run launched: %+v", runner.plans)
	}

	stdout, stderr, code = runLocalCLI(
		t,
		sources,
		runner,
		"",
		false,
		"last",
		"--dry-run",
		"--json",
	)
	if code != ExitOK {
		t.Fatalf("last exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("decode last plan: %v\n%s", err, stdout)
	}
	if plan.SessionRef != "claude:claude-one" {
		t.Fatalf("last selected %q", plan.SessionRef)
	}

	stdout, stderr, code = runLocalCLI(
		t,
		sources,
		runner,
		"",
		false,
		"fork",
		"codex:codex-two",
	)
	if code != ExitOK {
		t.Fatalf("fork exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if len(runner.plans) != 1 {
		t.Fatalf("launches=%d want 1", len(runner.plans))
	}
	if got := runner.plans[0]; got.Executable != "codex" ||
		strings.Join(got.Args, "\x00") != "fork\x00codex-two" {
		t.Fatalf("unexpected real plan: %+v", got)
	}
}

func TestResumeWithProducesHandoffPlanAndNotice(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	runner := &recordingLaunchRunner{}

	directOut, directErr, code := runHandoffCLI(t, home, vendorHome, sources, runner,
		"handoff", "codex:source-session", "--to", "claude", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("direct handoff exit=%d stdout=%q stderr=%q", code, directOut, directErr)
	}
	aliasOut, aliasErr, code := runHandoffCLI(t, home, vendorHome, sources, runner,
		"resume", "codex:source-session", "--with", "claude", "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("resume --with exit=%d stdout=%q stderr=%q", code, aliasOut, aliasErr)
	}
	if !strings.Contains(aliasErr, "Structured handoff") || !strings.Contains(aliasErr, "not native resume") {
		t.Fatalf("resume --with notice missing: %q", aliasErr)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("dry-run launched: %+v", runner.plans)
	}

	var direct, alias handoffPlanOutput
	if err := json.Unmarshal([]byte(directOut), &direct); err != nil {
		t.Fatalf("decode direct handoff: %v\n%s", err, directOut)
	}
	if err := json.Unmarshal([]byte(aliasOut), &alias); err != nil {
		t.Fatalf("decode resume --with: %v\n%s", err, aliasOut)
	}
	normalizeHandoffPlanPaths(&direct)
	normalizeHandoffPlanPaths(&alias)
	if !reflect.DeepEqual(direct, alias) {
		t.Fatalf("resume --with plan differs\ndirect: %+v\nalias:  %+v", direct, alias)
	}
}

func TestResumeWithForkConflictIsUsageError(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, nil,
		"resume", "codex:source-session", "--with", "claude", "--fork")
	if code != ExitUsage {
		t.Fatalf("resume --with --fork exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestPickerHandoffIsExplicitAndRoutesToPipeline(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("HOME", vendorHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	runner := &recordingLaunchRunner{}
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &stdout, Stderr: &stderr,
		Stdin: strings.NewReader("h 1\nclaude\n"), SessionSources: sources,
		SessionLaunchRunner: runner, PreflightVerifier: readyPreflightVerifier{},
		AgentProcessChecker: func(context.Context, string, processcheck.Target) (bool, bool, error) {
			return false, true, nil
		},
		TerminalChecker: func(io.Reader, io.Writer) bool { return true },
	})
	if code != ExitOK {
		t.Fatalf("picker handoff exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if len(runner.plans) != 1 || runner.plans[0].Operation != sessionindex.OperationHandoff {
		t.Fatalf("picker handoff launches=%+v", runner.plans)
	}
	if !strings.Contains(stdout.String(), "h NUMBER (hand off to another agent)") ||
		!strings.Contains(stderr.String(), "not native resume") {
		t.Fatalf("picker handoff surface missing stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func normalizeHandoffPlanPaths(output *handoffPlanOutput) {
	if len(output.Destination.Args) > 0 {
		output.Destination.Args[len(output.Destination.Args)-1] = filepath.Base(output.Destination.Args[len(output.Destination.Args)-1])
	}
	for index, path := range output.PlannedFiles {
		output.PlannedFiles[index] = filepath.Base(path)
	}
}

func TestNativeLaunchRefusalsUseStableCodes(t *testing.T) {
	workspace := t.TempDir()
	sources := localTestSources(workspace)

	stdout, stderr, code := runLocalCLI(
		t,
		sources,
		nil,
		"",
		false,
		"resume",
		"claude:claude-one",
		"--json",
	)
	if code != ExitUsage || !strings.Contains(stderr, "--dry-run") {
		t.Fatalf("json launch exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	readOnly := staticSessionSource{
		name: sessionindex.AgentGemini,
		result: sessionindex.ScanResult{Records: []sessionindex.Record{{
			ID:             "read-only",
			Agent:          sessionindex.AgentGemini,
			Project:        "phase-two",
			Workspace:      workspace,
			UpdatedAt:      time.Now().UTC(),
			ReadOnlyReason: "read only by design",
			SourcePath:     filepath.Join(workspace, "gemini.json"),
			SourceModTime:  1,
			SourceSize:     1,
		}}},
	}
	stdout, stderr, code = runLocalCLI(
		t,
		[]sessionindex.Source{readOnly},
		nil,
		"",
		false,
		"fork",
		"gemini:read-only",
		"--dry-run",
	)
	if code != ExitCompatibility || !strings.Contains(stderr, "read only by design") {
		t.Fatalf("read-only exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestMissingAndAmbiguousReferencesNeverLaunch(t *testing.T) {
	workspace := t.TempDir()
	updated := time.Now().UTC()
	duplicate := func(agent string) sessionindex.Source {
		return staticSessionSource{
			name: agent,
			result: sessionindex.ScanResult{Records: []sessionindex.Record{{
				ID:            "duplicate",
				Agent:         agent,
				Workspace:     workspace,
				UpdatedAt:     updated,
				CanResume:     true,
				CanFork:       true,
				SourcePath:    filepath.Join(workspace, agent+".jsonl"),
				SourceModTime: 1,
				SourceSize:    1,
			}}},
		}
	}
	sources := []sessionindex.Source{
		duplicate(sessionindex.AgentClaude),
		duplicate(sessionindex.AgentCodex),
	}
	runner := &recordingLaunchRunner{}

	stdout, stderr, code := runLocalCLI(
		t,
		sources,
		runner,
		"",
		false,
		"resume",
		"duplicate",
		"--dry-run",
	)
	if code != ExitUsage ||
		!strings.Contains(stderr, "claude:duplicate") ||
		!strings.Contains(stderr, "codex:duplicate") {
		t.Fatalf("ambiguous exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("ambiguous reference launched: %+v", runner.plans)
	}

	stdout, stderr, code = runLocalCLI(
		t,
		sources,
		runner,
		"",
		false,
		"inspect",
		"missing",
	)
	if code != ExitUsage || !strings.Contains(stderr, "session not found") {
		t.Fatalf("missing exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if len(runner.plans) != 0 {
		t.Fatalf("missing reference launched: %+v", runner.plans)
	}
}

func TestInteractivePickerFiltersAndLaunchesExactSelection(t *testing.T) {
	workspace := t.TempDir()
	runner := &recordingLaunchRunner{}
	stdout, stderr, code := runLocalCLI(
		t,
		localTestSources(workspace),
		runner,
		"/needle\n1\n",
		true,
	)
	if code != ExitOK {
		t.Fatalf("picker exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if len(runner.plans) != 1 || runner.plans[0].SessionRef != "codex:codex-two" {
		t.Fatalf("picker launched %+v\nstdout=%s\nstderr=%s", runner.plans, stdout, stderr)
	}
	if strings.Contains(stdout, "controlled prompt") {
		t.Fatalf("picker exposed prompt preview: %s", stdout)
	}
}

func TestInteractivePickerCancelAndNonTTY(t *testing.T) {
	workspace := t.TempDir()
	sources := localTestSources(workspace)
	runner := &recordingLaunchRunner{}

	stdout, stderr, code := runLocalCLI(t, sources, runner, "q\n", true)
	if code != ExitOK || len(runner.plans) != 0 {
		t.Fatalf("cancel exit=%d launches=%d stdout=%q stderr=%q", code, len(runner.plans), stdout, stderr)
	}

	stdout, stderr, code = runLocalCLI(t, sources, runner, "", false)
	if code != ExitUsage ||
		!strings.Contains(stderr, "rein sessions --json") ||
		len(runner.plans) != 0 {
		t.Fatalf("non-TTY exit=%d launches=%d stdout=%q stderr=%q", code, len(runner.plans), stdout, stderr)
	}
}

func TestSearchWithoutQueryIsUsageError(t *testing.T) {
	stdout, stderr, code := runLocalCLI(
		t,
		[]sessionindex.Source{},
		nil,
		"",
		false,
		"search",
	)
	if code != ExitUsage {
		t.Fatalf("search exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestNativeChildFailurePropagates(t *testing.T) {
	workspace := t.TempDir()
	runner := &recordingLaunchRunner{err: errors.New("controlled child failure")}
	stdout, stderr, code := runLocalCLI(
		t,
		localTestSources(workspace),
		runner,
		"",
		false,
		"resume",
		"claude:claude-one",
	)
	if code != ExitRuntime || !strings.Contains(stderr, "controlled child failure") {
		t.Fatalf("child failure exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestRunContextPropagatesCancellation(t *testing.T) {
	t.Setenv("REINSTATE_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	code := RunContext(ctx, Options{
		Name:           "rein",
		Stdout:         &stdout,
		Stderr:         &stderr,
		Args:           []string{"sessions"},
		SessionSources: []sessionindex.Source{},
	})
	if code != ExitRuntime || !strings.Contains(stderr.String(), "canceled") {
		t.Fatalf(
			"canceled exit=%d stdout=%q stderr=%q",
			code,
			stdout.String(),
			stderr.String(),
		)
	}
}
