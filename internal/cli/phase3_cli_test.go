package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

type preflightVerifierFunc func(context.Context, preflight.Input) (preflight.Report, error)

func (function preflightVerifierFunc) Verify(ctx context.Context, input preflight.Input) (preflight.Report, error) {
	return function(ctx, input)
}

type controlledAgentVersionRunner struct{}

func (controlledAgentVersionRunner) Version(context.Context, string, ...string) (agentcheck.VersionOutput, error) {
	return agentcheck.VersionOutput{Stdout: "2.1.220 (Claude Code)"}, nil
}

type sequenceSessionSource struct {
	name    string
	results []sessionindex.ScanResult
	err     error
	calls   int
}

func (source *sequenceSessionSource) Name() string { return source.name }

func (source *sequenceSessionSource) Scan(context.Context) (sessionindex.ScanResult, error) {
	if source.err != nil {
		return sessionindex.ScanResult{}, source.err
	}
	if len(source.results) == 0 {
		return sessionindex.ScanResult{}, nil
	}
	index := source.calls
	if index >= len(source.results) {
		index = len(source.results) - 1
	}
	source.calls++
	return source.results[index], nil
}

func phase3Report(input preflight.Input, decision preflight.Decision, checks ...preflight.Check) preflight.Report {
	hasSourceCheck := false
	for _, check := range checks {
		if check.ID == "source.fresh" {
			hasSourceCheck = true
			break
		}
	}
	if !hasSourceCheck {
		checks = append([]preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Provenance: workspace.ProvenanceCurrentObservation, Message: "controlled source is fresh",
		}}, checks...)
	}
	report := preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		SessionRef:    input.SessionRef,
		Decision:      decision,
		Checks:        append([]preflight.Check(nil), checks...),
		Workspace: workspace.Fingerprint{Git: workspace.GitFingerprint{
			WorkingTree: workspace.WorkingTreeFingerprint{State: workspace.WorkingTreeUnavailable},
		}},
	}
	if decision == preflight.DecisionBlocked {
		report.BlockExitCode = ExitSafety
	}
	return report
}

func warningCheck(id string) preflight.Check {
	return preflight.Check{
		ID: id, Status: preflight.StatusUnknown, Severity: preflight.SeverityWarning,
		Provenance: workspace.ProvenanceUnavailable,
		Message:    "controlled environment warning",
		ExitCode:   ExitSafety,
	}
}

func phase3Record(workspacePath string) sessionindex.Record {
	return sessionindex.Record{
		ID: "phase3-one", Agent: sessionindex.AgentClaude, Title: "Phase 3 session",
		Project: "phase-three", Workspace: workspacePath, CanResume: true, CanFork: true,
		SourcePath: workspacePath + "/projects/phase3-one.jsonl", SourceModTime: 1, SourceSize: 1,
	}
}

func executePhase3CLI(
	t *testing.T,
	home string,
	sources []sessionindex.Source,
	verifier preflight.Verifier,
	runner sessionindex.LaunchRunner,
	stdin string,
	terminal bool,
	args ...string,
) (string, string, int) {
	t.Helper()
	t.Setenv("REINSTATE_HOME", home)
	var stdout, stderr bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(stdin), Args: args,
		SessionSources: sources, SessionLaunchRunner: runner, PreflightVerifier: verifier,
		TerminalChecker: func(io.Reader, io.Writer) bool { return terminal },
	})
	return stdout.String(), stderr.String(), code
}

func TestPhase3InspectAndDryRunEnvironmentContracts(t *testing.T) {
	workspacePath := t.TempDir()
	source := staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{phase3Record(workspacePath)}}}
	blocked := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		return phase3Report(input, preflight.DecisionBlocked, preflight.Check{
			ID: "git.repository", Status: preflight.StatusChanged, Severity: preflight.SeverityBlock,
			Provenance: workspace.ProvenanceReinstatePrelaunchObserved, Message: "repository differs", ExitCode: ExitSafety,
		}), nil
	})

	stdout, stderr, code := executePhase3CLI(t, t.TempDir(), []sessionindex.Source{source}, blocked, nil, "", false, "inspect", "claude:phase3-one", "--json")
	if code != ExitOK || stderr != "" {
		t.Fatalf("blocked inspect exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var inspected localInspectOutput
	if err := json.Unmarshal([]byte(stdout), &inspected); err != nil {
		t.Fatal(err)
	}
	if inspected.Environment.Decision != preflight.DecisionBlocked || inspected.Environment.SessionRef != "claude:phase3-one" {
		t.Fatalf("inspect environment = %+v", inspected.Environment)
	}

	stdout, stderr, code = executePhase3CLI(t, t.TempDir(), []sessionindex.Source{source}, blocked, nil, "", false, "resume", "claude:phase3-one", "--dry-run", "--json")
	if code != ExitSafety || stdout != "" {
		t.Fatalf("blocked dry-run exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var envelope ErrorJSON
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
		t.Fatalf("blocked dry-run JSON envelope: %v\n%s", err, stderr)
	}
	if envelope.Code != "safety" || envelope.Details["environment"] == nil || envelope.Details["launch_plan"] == nil {
		t.Fatalf("blocked envelope = %+v", envelope)
	}

	warning := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		return phase3Report(input, preflight.DecisionConfirmationRequired, warningCheck("baseline.unavailable")), nil
	})
	runner := &recordingLaunchRunner{}
	home := t.TempDir()
	stdout, stderr, code = executePhase3CLI(t, home, []sessionindex.Source{source}, warning, runner, "", false, "resume", "claude:phase3-one", "--dry-run", "--json")
	if code != ExitOK || stderr != "" || len(runner.plans) != 0 {
		t.Fatalf("warning dry-run exit=%d launches=%d stdout=%q stderr=%q", code, len(runner.plans), stdout, stderr)
	}
	var dry map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &dry); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"agent", "session_ref", "operation", "executable", "args", "cwd", "environment"} {
		if _, ok := dry[key]; !ok {
			t.Fatalf("dry-run omitted %q: %s", key, stdout)
		}
	}
	assertNoPhase3Baseline(t, home, "claude:phase3-one")
}

func TestPhase3WarningAcknowledgementAndTTYRules(t *testing.T) {
	workspacePath := t.TempDir()
	source := staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{phase3Record(workspacePath)}}}
	verifier := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		return phase3Report(
			input,
			preflight.DecisionConfirmationRequired,
			warningCheck("baseline.unavailable"),
			preflight.Check{
				ID: "agent.version", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
				Provenance: workspace.ProvenanceCurrentObservation, Message: "controlled version matches",
			},
		), nil
	})

	for _, test := range []struct {
		name     string
		stdin    string
		terminal bool
		args     []string
		wantCode int
		launch   bool
	}{
		{name: "non tty missing", args: []string{"resume", "claude:phase3-one"}, wantCode: ExitSafety},
		{name: "exact acknowledgement", args: []string{"resume", "claude:phase3-one", "--allow-environment-warning", "baseline.unavailable"}, wantCode: ExitOK, launch: true},
		{name: "unknown", args: []string{"resume", "claude:phase3-one", "--allow-environment-warning", "stale.warning"}, wantCode: ExitUsage},
		{name: "wildcard", args: []string{"resume", "claude:phase3-one", "--allow-environment-warning", "*"}, wantCode: ExitUsage},
		{name: "informational", args: []string{"resume", "claude:phase3-one", "--allow-environment-warning", "agent.version"}, wantCode: ExitUsage},
		{name: "duplicate", args: []string{"resume", "claude:phase3-one", "--allow-environment-warning", "baseline.unavailable", "--allow-environment-warning", "baseline.unavailable"}, wantCode: ExitUsage},
		{name: "tty exact yes", stdin: "yes\n", terminal: true, args: []string{"resume", "claude:phase3-one"}, wantCode: ExitOK, launch: true},
		{name: "tty shorthand rejected then yes", stdin: "y\nyes\n", terminal: true, args: []string{"resume", "claude:phase3-one"}, wantCode: ExitOK, launch: true},
		{name: "tty no", stdin: "no\n", terminal: true, args: []string{"resume", "claude:phase3-one"}, wantCode: ExitSafety},
		{name: "tty default no", stdin: "\n", terminal: true, args: []string{"resume", "claude:phase3-one"}, wantCode: ExitSafety},
		{name: "tty eof", terminal: true, args: []string{"resume", "claude:phase3-one"}, wantCode: ExitSafety},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &recordingLaunchRunner{}
			stdout, stderr, code := executePhase3CLI(t, t.TempDir(), []sessionindex.Source{source}, verifier, runner, test.stdin, test.terminal, test.args...)
			if code != test.wantCode || (len(runner.plans) == 1) != test.launch {
				t.Fatalf("exit=%d launches=%d stdout=%q stderr=%q", code, len(runner.plans), stdout, stderr)
			}
			if test.name == "tty shorthand rejected then yes" && !strings.Contains(stdout, "Enter exactly yes or no") {
				t.Fatalf("exact confirmation guidance missing: %s", stdout)
			}
		})
	}

	root := NewRoot(Options{PreflightVerifier: readyPreflightVerifier{}})
	for _, name := range []string{"resume", "fork", "last"} {
		command, _, err := root.Find([]string{name})
		if err != nil || command.Flags().Lookup("allow-environment-warning") == nil {
			t.Fatalf("%s repeatable warning flag missing: %v", name, err)
		}
	}
}

func TestPhase3DryRunWarningIDValidationUsesUsageExit(t *testing.T) {
	workspacePath := t.TempDir()
	source := staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{phase3Record(workspacePath)}}}
	verifier := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		return phase3Report(input, preflight.DecisionConfirmationRequired, warningCheck("baseline.unavailable")), nil
	})
	for _, test := range []struct {
		name     string
		allowed  []string
		wantCode int
	}{
		{name: "exact current warning", allowed: []string{"baseline.unavailable"}, wantCode: ExitOK},
		{name: "unknown", allowed: []string{"stale.warning"}, wantCode: ExitUsage},
		{name: "wildcard", allowed: []string{"*"}, wantCode: ExitUsage},
		{name: "duplicate", allowed: []string{"baseline.unavailable", "baseline.unavailable"}, wantCode: ExitUsage},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"resume", "claude:phase3-one", "--dry-run", "--json"}
			for _, warning := range test.allowed {
				args = append(args, "--allow-environment-warning", warning)
			}
			_, stderr, code := executePhase3CLI(t, t.TempDir(), []sessionindex.Source{source}, verifier, nil, "", false, args...)
			if code != test.wantCode {
				t.Fatalf("exit=%d want=%d stderr=%q", code, test.wantCode, stderr)
			}
			if test.wantCode == ExitUsage && !strings.Contains(stderr, `"code": "usage"`) {
				t.Fatalf("dry-run warning validation did not retain usage semantics: %s", stderr)
			}
		})
	}
}

func TestPhase3WarningPromptCancellationDeclinesWithSafetyExit(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	command := &cobra.Command{}
	command.SetContext(ctx)
	var output bytes.Buffer
	command.SetOut(&output)
	started := make(chan struct{})
	blocked := make(chan struct{})
	reader := func() (string, bool, error) {
		close(started)
		<-blocked
		return "yes", true, nil
	}
	result := make(chan error, 1)
	go func() {
		_, err := authorizeEnvironment(
			command,
			localCommandOptions{terminalCheck: func(io.Reader, io.Writer) bool { return true }},
			phase3Report(
				preflight.Input{SessionRef: "claude:controlled", Agent: sessionindex.AgentClaude},
				preflight.DecisionConfirmationRequired,
				warningCheck("baseline.unavailable"),
			),
			sessionindex.LaunchPlan{Agent: sessionindex.AgentClaude, SessionRef: "claude:controlled", Operation: sessionindex.OperationResume},
			nil,
			reader,
		)
		result <- err
	}()
	<-started
	cancel()
	select {
	case err := <-result:
		var exitErr *ExitError
		if !errors.As(err, &exitErr) || exitErr.Code != ExitSafety {
			t.Fatalf("cancellation error = %v, want safety exit %d", err, ExitSafety)
		}
	case <-time.After(time.Second):
		t.Fatal("warning confirmation did not react to cancellation")
	}
	close(blocked)
}

func TestPhase3WarningPromptCancellationWinsOverConcurrentYes(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	confirmed, err := confirmEnvironmentWarnings(ctx, io.Discard, func() (string, bool, error) {
		cancel()
		return "yes", true, nil
	})
	if err != nil || confirmed {
		t.Fatalf("confirmation/error = %t / %v, want declined without runtime error", confirmed, err)
	}
}

func TestTerminalLineReaderTreatsControlCAsClosedInput(t *testing.T) {
	t.Parallel()
	stream := struct {
		io.Reader
		io.Writer
	}{Reader: strings.NewReader("\x03"), Writer: io.Discard}
	terminal := term.NewTerminal(&stream, "")
	reader := terminalLineReader(terminal.ReadLine)
	line, ok, err := reader()
	if err != nil || ok || line != "" {
		t.Fatalf("line/ok/error = %q / %t / %v, want closed input", line, ok, err)
	}
}

func TestPhase3BaselinePersistsOnlyAfterSuccessfulChild(t *testing.T) {
	workspacePath := t.TempDir()
	source := staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{phase3Record(workspacePath)}}}
	var inputs []preflight.Input
	verifier := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		inputs = append(inputs, input)
		return phase3Report(input, preflight.DecisionReady), nil
	})

	home := t.TempDir()
	runner := &recordingLaunchRunner{}
	stdout, stderr, code := executePhase3CLI(t, home, []sessionindex.Source{source}, verifier, runner, "", false, "fork", "claude:phase3-one")
	if code != ExitOK || len(runner.plans) != 1 || len(inputs) != 2 {
		t.Fatalf("successful launch exit=%d plans=%d verifies=%d stdout=%q stderr=%q", code, len(runner.plans), len(inputs), stdout, stderr)
	}
	store, err := sessionindex.OpenDefault(home)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := store.GetPrelaunchBaseline(context.Background(), "claude:phase3-one")
	_ = store.Close()
	if err != nil || baseline.Provenance != environment.PrelaunchObservedProvenance || baseline.ObservedAt.IsZero() {
		t.Fatalf("baseline = %+v, %v", baseline, err)
	}
	stdout, stderr, code = executePhase3CLI(t, home, []sessionindex.Source{source}, verifier, nil, "", false, "inspect", "claude:phase3-one", "--json")
	if code != ExitOK || len(inputs) != 3 || inputs[2].Baseline == nil {
		t.Fatalf("persisted baseline was not supplied to inspect: exit=%d inputs=%d stdout=%q stderr=%q", code, len(inputs), stdout, stderr)
	}

	failureHome := t.TempDir()
	failingRunner := &recordingLaunchRunner{err: errors.New("controlled child failure")}
	_, _, code = executePhase3CLI(t, failureHome, []sessionindex.Source{source}, verifier, failingRunner, "", false, "resume", "claude:phase3-one")
	if code != ExitRuntime {
		t.Fatalf("failed child exit=%d", code)
	}
	assertNoPhase3Baseline(t, failureHome, "claude:phase3-one")
}

func TestPhase3StaleSourceAndChangedReverificationNeverLaunch(t *testing.T) {
	workspacePath := t.TempDir()
	record := phase3Record(workspacePath)
	source := &sequenceSessionSource{name: sessionindex.AgentClaude, results: []sessionindex.ScanResult{{Records: []sessionindex.Record{record}}}}
	home := t.TempDir()
	ready := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		if !input.SourceFresh {
			return phase3Report(input, preflight.DecisionBlocked, preflight.Check{
				ID: "source.fresh", Status: preflight.StatusChanged, Severity: preflight.SeverityBlock,
				Provenance: workspace.ProvenanceUnavailable, Message: "source stale", ExitCode: ExitSafety,
			}), nil
		}
		return phase3Report(input, preflight.DecisionReady), nil
	})
	_, _, code := executePhase3CLI(t, home, []sessionindex.Source{source}, ready, nil, "", false, "sessions", "--json")
	if code != ExitOK {
		t.Fatalf("seed sessions exit=%d", code)
	}
	source.err = errors.New("controlled stale source")
	runner := &recordingLaunchRunner{}
	stdout, stderr, code := executePhase3CLI(t, home, []sessionindex.Source{source}, ready, runner, "", false, "resume", "claude:phase3-one")
	if code != ExitSafety || len(runner.plans) != 0 {
		t.Fatalf("stale launch exit=%d plans=%d stdout=%q stderr=%q", code, len(runner.plans), stdout, stderr)
	}

	call := 0
	changedVerifier := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		call++
		report := phase3Report(input, preflight.DecisionReady)
		if call == 2 {
			report.Checks = append(report.Checks, preflight.Check{
				ID: "controlled.changed", Status: preflight.StatusPresent, Severity: preflight.SeverityInfo,
				Provenance: workspace.ProvenanceCurrentObservation, Message: "controlled second observation",
			})
		}
		return report, nil
	})
	runner = &recordingLaunchRunner{}
	stdout, stderr, code = executePhase3CLI(t, t.TempDir(), []sessionindex.Source{staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}}}, changedVerifier, runner, "", false, "resume", "claude:phase3-one")
	if code != ExitSafety || len(runner.plans) != 0 || call != 2 {
		t.Fatalf("changed reverify exit=%d plans=%d verifies=%d stdout=%q stderr=%q", code, len(runner.plans), call, stdout, stderr)
	}
}

func TestFinalEnvironmentGuardRejectsWorkspaceAndRepositoryDrift(t *testing.T) {
	t.Run("workspace path becomes a file", func(t *testing.T) {
		fixture := newFinalGuardFixture(t)
		guard := finalEnvironmentGuard(nil, fixture.options, fixture.index, fixture.record, fixture.plan, fixture.report, preflight.WarningIDs(fixture.report))
		if err := guard(context.Background(), fixture.plan); err != nil {
			t.Fatalf("unchanged final guard = %v", err)
		}

		preserved := fixture.workspace + "-preserved"
		if err := os.Rename(fixture.workspace, preserved); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture.workspace, []byte("not a workspace"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := guard(context.Background(), fixture.plan)
		if err == nil || ExitCodeFrom(err) != ExitSafety || !strings.Contains(err.Error(), "execution boundary") {
			t.Fatalf("path-type drift guard = %v (exit %d)", err, ExitCodeFrom(err))
		}
	})

	t.Run("repository identity changes", func(t *testing.T) {
		fixture := newFinalGuardFixture(t)
		guard := finalEnvironmentGuard(nil, fixture.options, fixture.index, fixture.record, fixture.plan, fixture.report, preflight.WarningIDs(fixture.report))
		if err := guard(context.Background(), fixture.plan); err != nil {
			t.Fatalf("unchanged final guard = %v", err)
		}

		*fixture.remote = "https://example.com/other/repository.git"
		err := guard(context.Background(), fixture.plan)
		if err == nil || ExitCodeFrom(err) != ExitSafety || !strings.Contains(err.Error(), "execution boundary") {
			t.Fatalf("repository drift guard = %v (exit %d)", err, ExitCodeFrom(err))
		}
	})
}

type finalGuardFixture struct {
	workspace string
	remote    *string
	options   localCommandOptions
	index     *sessionindex.Index
	record    sessionindex.Record
	plan      sessionindex.LaunchPlan
	report    preflight.Report
}

func newFinalGuardFixture(t *testing.T) finalGuardFixture {
	t.Helper()
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspace")
	agentRoot := filepath.Join(root, "agent-home")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(agentRoot, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "controlled-claude"), []byte("controlled agent"), 0o700); err != nil {
		t.Fatal(err)
	}

	remote := "https://example.com/org/repo.git"
	record := phase3Record(workspacePath)
	record.SourcePath = filepath.Join(agentRoot, "projects", "phase3-one.jsonl")
	record.RecordedEnvironment.RepositoryID = environment.RecordedField{
		Value: remote, Provenance: "controlled.vendor.repository",
	}
	source := staticSessionSource{name: sessionindex.AgentClaude, result: sessionindex.ScanResult{Records: []sessionindex.Record{record}}}
	store, err := sessionindex.OpenDefault(filepath.Join(root, "reinstate-home"))
	if err != nil {
		t.Fatal(err)
	}
	index, err := sessionindex.NewIndex(store, source)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = index.Close() })

	gitRunner := workspace.GitRunnerFunc(func(_ context.Context, _ string, args ...string) ([]byte, error) {
		switch {
		case strings.Join(args, "\x00") == "rev-parse\x00--path-format=absolute\x00--show-toplevel":
			return []byte(workspacePath + "\n"), nil
		case len(args) > 0 && args[len(args)-1] == "--untracked-files=normal":
			return []byte("# branch.oid 1111111111111111111111111111111111111111\x00# branch.head main\x00"), nil
		case strings.Join(args, "\x00") == "rev-parse\x00--is-shallow-repository":
			return []byte("false\n"), nil
		case strings.Join(args, "\x00") == "config\x00--local\x00--no-includes\x00--null\x00--get-regexp\x00^remote\\..*\\.url$":
			return []byte("remote.origin.url\n" + remote + "\x00"), nil
		default:
			return nil, errors.New("unexpected controlled Git probe")
		}
	})
	options := localCommandOptions{verifier: preflight.Service{Options: preflight.Options{
		Workspace: workspace.ProbeOptions{Runner: gitRunner},
		Agent: agentcheck.Options{
			Root: agentRoot,
			LookPath: func(string) (string, error) {
				return filepath.Join(root, "controlled-claude"), nil
			},
			Runner: controlledAgentVersionRunner{},
		},
	}}}
	resolved, _, fresh, err := index.RefreshAndResolve(context.Background(), "claude:phase3-one")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := sessionindex.PlanLaunch(resolved, sessionindex.OperationResume)
	if err != nil {
		t.Fatal(err)
	}
	report, err := verifyLocalRecord(context.Background(), options, index, resolved, fresh)
	if err != nil {
		t.Fatal(err)
	}
	if authorization, err := preflight.Authorize(report, preflight.WarningIDs(report)); err != nil || !authorization.Allowed {
		t.Fatalf("fixture report is not authorizable: %+v, %v; report=%+v", authorization, err, report)
	}
	return finalGuardFixture{
		workspace: workspacePath, remote: &remote, options: options, index: index,
		record: resolved, plan: plan, report: report,
	}
}

func TestPhase3PickerReusesScannerAndReresolvesInspect(t *testing.T) {
	workspacePath := t.TempDir()
	initial := phase3Record(workspacePath)
	initial.Title = "initial picker title"
	refreshed := initial
	refreshed.Title = "refreshed picker title"
	refreshed.SourceModTime = 2
	refreshed.SourceSize = 2
	verifier := preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
		return phase3Report(input, preflight.DecisionConfirmationRequired, warningCheck("baseline.unavailable")), nil
	})

	launchSource := &sequenceSessionSource{name: sessionindex.AgentClaude, results: []sessionindex.ScanResult{
		{Records: []sessionindex.Record{initial}},
		{Records: []sessionindex.Record{refreshed}},
		{Records: []sessionindex.Record{refreshed}},
	}}
	runner := &recordingLaunchRunner{}
	stdout, stderr, code := executePhase3CLI(t, t.TempDir(), []sessionindex.Source{launchSource}, verifier, runner, "1\nyes\n", true)
	if code != ExitOK || len(runner.plans) != 1 {
		t.Fatalf("picker launch exit=%d plans=%d stdout=%q stderr=%q", code, len(runner.plans), stdout, stderr)
	}
	if launchSource.calls < 3 {
		t.Fatalf("picker source scans=%d want initial and two bound refreshes", launchSource.calls)
	}

	inspectSource := &sequenceSessionSource{name: sessionindex.AgentClaude, results: []sessionindex.ScanResult{
		{Records: []sessionindex.Record{initial}},
		{Records: []sessionindex.Record{refreshed}},
	}}
	stdout, stderr, code = executePhase3CLI(t, t.TempDir(), []sessionindex.Source{inspectSource}, readyPreflightVerifier{}, nil, "i 1\nq\n", true)
	if code != ExitOK || !strings.Contains(stdout, "Title: refreshed picker title") {
		t.Fatalf("picker inspect exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func assertNoPhase3Baseline(t *testing.T, home, reference string) {
	t.Helper()
	store, err := sessionindex.OpenDefault(home)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if _, err := store.GetPrelaunchBaseline(context.Background(), reference); !errors.Is(err, sessionindex.ErrPrelaunchBaselineNotFound) {
		t.Fatalf("unexpected baseline for %s: %v", reference, err)
	}
}
