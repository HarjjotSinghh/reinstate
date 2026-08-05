package preflight

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

const testHead = "1111111111111111111111111111111111111111"

func syntheticCredentialRemote(user, password, destination, query string) string {
	value := "https://" + user + ":" + password + "@" + destination
	if query != "" {
		value += "?" + query
	}
	return value
}

type versionRunner struct {
	output string
	err    error
}

func (runner versionRunner) Version(context.Context, string, ...string) (string, error) {
	return runner.output, runner.err
}

type agentVersionRunner struct {
	output agentcheck.VersionOutput
	err    error
}

func (runner agentVersionRunner) Version(context.Context, string, ...string) (agentcheck.VersionOutput, error) {
	return runner.output, runner.err
}

func TestVerifyFirstObservationThenStableBaseline(t *testing.T) {
	t.Parallel()
	fixture := newFixture(t, syntheticCredentialRemote("user", "secret", "example.com/org/repo.git", "token=private#fragment"))
	input := Input{SessionRef: "claude:controlled", Agent: "claude", Workspace: fixture.workspace, SourceFresh: true}

	first, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if first.Decision != DecisionConfirmationRequired || first.BlockExitCode != 0 {
		t.Fatalf("first decision = %s exit=%d", first.Decision, first.BlockExitCode)
	}
	if got := findCheck(t, first, "baseline.unavailable"); got.Severity != SeverityWarning || got.Status != StatusUnknown {
		t.Fatalf("baseline check = %+v", got)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"user:secret", "token=private", "#fragment", "example.com/org/repo"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("report leaked %q: %s", secret, encoded)
		}
	}
	secondFirst, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, secondFirst) {
		t.Fatalf("equal observations produced different reports\nfirst=%+v\nsecond=%+v", first, secondFirst)
	}

	baseline, err := BaselineFromReport(first, time.Date(2026, 8, 5, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	input.Baseline = &baseline
	second, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if second.Decision != DecisionReady {
		t.Fatalf("second decision = %s checks=%+v", second.Decision, second.Checks)
	}
	for _, id := range []string{"git.repository", "git.branch", "git.head", "git.working_tree"} {
		if check := findCheck(t, second, id); check.Status != StatusMatch || check.Severity != SeverityInfo {
			t.Fatalf("%s = %+v", id, check)
		}
	}
}

func TestVerifyRepositoryReplacementAndStaleSourceBlock(t *testing.T) {
	t.Parallel()
	fixture := newFixture(t, "https://example.com/org/repo.git")
	input := Input{SessionRef: "codex:controlled", Agent: "codex", Workspace: fixture.workspace, SourceFresh: true}
	first, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := BaselineFromReport(first, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	input.Baseline = &baseline
	fixture.remote = "https://example.com/different/repository.git"
	replaced, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Decision != DecisionBlocked || replaced.BlockExitCode != exitcode.Safety {
		t.Fatalf("replacement decision=%s exit=%d checks=%+v", replaced.Decision, replaced.BlockExitCode, replaced.Checks)
	}
	if check := findCheck(t, replaced, "git.repository"); check.Status != StatusChanged || check.Severity != SeverityBlock {
		t.Fatalf("repository check = %+v", check)
	}

	fixture.remote = "https://example.com/org/repo.git"
	input.SourceFresh = false
	stale, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if stale.Decision != DecisionBlocked || stale.BlockExitCode != exitcode.Safety {
		t.Fatalf("stale decision=%s exit=%d", stale.Decision, stale.BlockExitCode)
	}
}

func TestVerifyGitUnavailableDoesNotManufactureDerivativeMismatches(t *testing.T) {
	t.Parallel()
	fixture := newFixture(t, "https://example.com/org/repo.git")
	input := Input{SessionRef: "claude:controlled", Agent: "claude", Workspace: fixture.workspace, SourceFresh: true}
	first, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := BaselineFromReport(first, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	input.Baseline = &baseline
	fixture.options.Workspace.Runner = workspace.GitRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		return nil, workspace.ErrGitUnavailable
	})

	report, err := Verify(context.Background(), input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionBlocked || report.BlockExitCode != exitcode.Compatibility {
		t.Fatalf("Git-unavailable decision = %s/%d; checks=%+v", report.Decision, report.BlockExitCode, report.Checks)
	}
	if check := findCheck(t, report, "git.available"); check.Status != StatusMissing || check.Severity != SeverityBlock || check.ExitCode != exitcode.Compatibility {
		t.Fatalf("git.available = %+v", check)
	}
	for _, id := range []string{"git.repository", "git.branch", "git.head", "git.working_tree"} {
		check := findCheck(t, report, id)
		if check.Status != StatusUnknown || check.Severity != SeverityInfo || check.ExitCode != 0 {
			t.Fatalf("derivative check %s = %+v", id, check)
		}
	}
}

func TestAuthorizeRequiresExactFreshWarningSet(t *testing.T) {
	t.Parallel()
	report := validPolicyReport([]Check{
		{ID: "baseline.unavailable", Status: StatusUnknown, Severity: SeverityWarning, Provenance: workspace.ProvenanceUnavailable, ExitCode: exitcode.Safety},
		{ID: "git.branch", Status: StatusChanged, Severity: SeverityWarning, Provenance: workspace.ProvenanceCurrentObservation, ExitCode: exitcode.Safety},
		{ID: "agent.version", Status: StatusMatch, Severity: SeverityInfo, Provenance: workspace.ProvenanceCurrentObservation},
	})
	for name, test := range map[string]struct {
		allowed []string
		code    int
	}{
		"none":          {allowed: []string{}, code: exitcode.Safety},
		"partial":       {allowed: []string{"git.branch"}, code: exitcode.Safety},
		"duplicate":     {allowed: []string{"git.branch", "git.branch"}, code: exitcode.Usage},
		"wildcard":      {allowed: []string{"*"}, code: exitcode.Usage},
		"unknown":       {allowed: []string{"stale.warning"}, code: exitcode.Usage},
		"informational": {allowed: []string{"agent.version"}, code: exitcode.Usage},
	} {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			authorization, err := Authorize(report, test.allowed)
			if err == nil || authorization.Allowed || authorization.ExitCode != test.code {
				t.Fatalf("Authorize(%v) = %+v, %v", test.allowed, authorization, err)
			}
		})
	}
	authorization, err := Authorize(report, []string{"git.branch", "baseline.unavailable"})
	if err != nil || !authorization.Allowed || authorization.ExitCode != 0 {
		t.Fatalf("exact authorization = %+v, %v", authorization, err)
	}

	blocked := validPolicyReport(append(append([]Check(nil), report.Checks...), Check{
		ID: "agent.executable", Status: StatusMissing, Severity: SeverityBlock,
		Provenance: workspace.ProvenanceCurrentObservation, ExitCode: exitcode.Compatibility,
	}))
	authorization, err = Authorize(blocked, []string{"git.branch", "baseline.unavailable"})
	if err == nil || authorization.Allowed || authorization.ExitCode != exitcode.Compatibility {
		t.Fatalf("blocked authorization = %+v, %v", authorization, err)
	}
}

func TestPolicyRejectsForgedOrInconsistentReports(t *testing.T) {
	t.Parallel()
	base := validPolicyReport([]Check{
		{ID: "baseline.unavailable", Status: StatusUnknown, Severity: SeverityWarning, Provenance: workspace.ProvenanceUnavailable, ExitCode: exitcode.Safety},
		{ID: "agent.version", Status: StatusMatch, Severity: SeverityInfo, Provenance: workspace.ProvenanceCurrentObservation},
	})

	mutations := map[string]func(Report) Report{
		"forged ready decision": func(report Report) Report {
			report.Decision = DecisionReady
			return report
		},
		"forged block exit": func(report Report) Report {
			report.BlockExitCode = exitcode.Runtime
			return report
		},
		"duplicate check ID": func(report Report) Report {
			report.Checks = append(report.Checks, report.Checks[0])
			return report
		},
		"invalid check provenance": func(report Report) Report {
			report.Checks[0].Provenance = workspace.Provenance("attacker_claim")
			return report
		},
	}
	for name, mutate := range mutations {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			report := base
			report.Checks = append([]Check(nil), base.Checks...)
			report = mutate(report)

			authorization, err := Authorize(report, []string{"baseline.unavailable"})
			if err == nil || authorization.Allowed || authorization.ExitCode != exitcode.Runtime {
				t.Fatalf("Authorize(forged report) = %+v, %v", authorization, err)
			}
			if _, err := BaselineFromReport(report, time.Now()); err == nil {
				t.Fatal("BaselineFromReport accepted a forged report")
			}
		})
	}
}

func validPolicyReport(checks []Check) Report {
	checks = normalizeChecks(checks)
	decision, code := aggregate(checks)
	return Report{SchemaVersion: SchemaVersion, SessionRef: "claude:controlled", Decision: decision, BlockExitCode: code, Checks: checks}
}

func TestBlockExitPrecedenceIsDeterministic(t *testing.T) {
	t.Parallel()
	checks := []Check{
		{ID: "source.fresh", Severity: SeverityBlock, ExitCode: exitcode.Safety},
		{ID: "workspace.available", Severity: SeverityBlock, ExitCode: exitcode.Compatibility},
		{ID: "git.probe", Severity: SeverityBlock, ExitCode: exitcode.Runtime},
	}
	for _, permutation := range [][]Check{
		checks,
		{checks[2], checks[0], checks[1]},
		{checks[1], checks[2], checks[0]},
	} {
		decision, code := aggregate(permutation)
		if decision != DecisionBlocked || code != exitcode.Runtime {
			t.Fatalf("aggregate(%+v) = %s/%d", permutation, decision, code)
		}
	}
	decision, code := aggregate(checks[:2])
	if decision != DecisionBlocked || code != exitcode.Safety {
		t.Fatalf("safety/compatibility precedence = %s/%d", decision, code)
	}
}

func TestNormalizeChecksKeepsMostTrustworthyDuplicate(t *testing.T) {
	t.Parallel()
	compatibility := Check{
		ID: "workspace.available", Status: StatusMissing, Severity: SeverityBlock,
		Provenance: workspace.ProvenanceUnavailable, ExitCode: exitcode.Compatibility,
	}
	runtimeFailure := Check{
		ID: "workspace.available", Status: StatusError, Severity: SeverityBlock,
		Provenance: workspace.ProvenanceCurrentObservation, ExitCode: exitcode.Runtime,
	}
	for _, checks := range [][]Check{
		{compatibility, runtimeFailure},
		{runtimeFailure, compatibility},
	} {
		normalized := normalizeChecks(checks)
		if len(normalized) != 1 || !reflect.DeepEqual(normalized[0], runtimeFailure) {
			t.Fatalf("normalizeChecks(%+v) = %+v, want runtime diagnostic", checks, normalized)
		}
	}
}

func TestCapabilityComparisonKeepsScopeAndNamesPrivateInIDs(t *testing.T) {
	t.Parallel()
	input := Input{Agent: "claude", Baseline: &environment.PrelaunchBaseline{
		Capabilities: []environment.Capability{
			{Agent: "claude", Kind: "mcp", Name: "same-name", Scope: "user", State: "declared", Provenance: environment.PrelaunchObservedProvenance},
			{Agent: "claude", Kind: "mcp", Name: "same-name", Scope: "project", State: "declared", Provenance: environment.PrelaunchObservedProvenance},
		},
	}}
	inventory := capability.Inventory{Items: []capability.Item{
		{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "same-name", Scope: capability.ScopeUser, State: capability.StateDeclared},
	}}
	checks := capabilityChecks(input, inventory)
	var matches, missing int
	for _, check := range checks {
		if strings.Contains(check.ID, "same-name") {
			t.Fatalf("capability ID exposed logical name: %q", check.ID)
		}
		switch check.Status {
		case StatusMatch:
			matches++
		case StatusMissing:
			missing++
		}
	}
	if matches != 1 || missing != 1 {
		t.Fatalf("scope comparison = matches:%d missing:%d checks:%+v", matches, missing, checks)
	}
}

func TestCapabilityComparisonWarnsOnMCPTransportDrift(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		from capability.Transport
		to   capability.Transport
	}{
		{name: "stdio to http", from: capability.TransportStdio, to: capability.TransportHTTP},
		{name: "known to unknown", from: capability.TransportStdio, to: capability.TransportUnknown},
		{name: "unknown to known", from: capability.TransportUnknown, to: capability.TransportStdio},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			input := Input{Agent: "claude", Baseline: &environment.PrelaunchBaseline{
				Capabilities: []environment.Capability{{
					Agent: "claude", Kind: "mcp", Name: "controlled-server", Scope: "project",
					State: "declared", Transport: string(test.from), Provenance: environment.PrelaunchObservedProvenance,
				}},
			}}
			inventory := capability.Inventory{Items: []capability.Item{{
				Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "controlled-server",
				Scope: capability.ScopeProject, State: capability.StateDeclared, Transport: test.to,
			}}}

			checks := capabilityChecks(input, inventory)
			if len(checks) != 1 {
				t.Fatalf("capabilityChecks() returned %d checks: %+v", len(checks), checks)
			}
			check := checks[0]
			if check.Status != StatusChanged || check.Severity != SeverityWarning || check.ExitCode != exitcode.Safety ||
				check.Expected != string(test.from) || check.Actual != string(test.to) {
				t.Fatalf("transport drift check = %+v", check)
			}
			if strings.Contains(check.ID, "controlled-server") || strings.Contains(check.ID, string(test.from)) || strings.Contains(check.ID, string(test.to)) {
				t.Fatalf("transport drift check ID leaked private metadata: %q", check.ID)
			}
		})
	}
}

func TestCapabilityDiagnosticAcknowledgementsAreScopeSpecific(t *testing.T) {
	t.Parallel()
	user := capability.Diagnostic{
		Agent: capability.AgentClaude, Kind: capability.KindMCP,
		Scope: capability.ScopeUser, Code: capability.DiagnosticReadFailed,
	}
	project := user
	project.Scope = capability.ScopeProject

	checks := capabilityChecks(Input{Agent: "claude"}, capability.Inventory{Diagnostics: []capability.Diagnostic{user, project}})
	if len(checks) != 2 || checks[0].ID == checks[1].ID {
		t.Fatalf("same-code scoped diagnostics did not get distinct IDs: %+v", checks)
	}
	report := validPolicyReport(checks)
	if authorization, err := Authorize(report, []string{checks[0].ID}); err == nil || authorization.Allowed || authorization.ExitCode != exitcode.Safety {
		t.Fatalf("partial scoped acknowledgement = %+v, %v", authorization, err)
	}

	priorReport := validPolicyReport(capabilityChecks(Input{Agent: "claude"}, capability.Inventory{Diagnostics: []capability.Diagnostic{user}}))
	priorID := priorReport.Checks[0].ID
	if authorization, err := Authorize(priorReport, []string{priorID}); err != nil || !authorization.Allowed {
		t.Fatalf("current scoped acknowledgement = %+v, %v", authorization, err)
	}
	currentReport := validPolicyReport(capabilityChecks(Input{Agent: "claude"}, capability.Inventory{Diagnostics: []capability.Diagnostic{project}}))
	if authorization, err := Authorize(currentReport, []string{priorID}); err == nil || authorization.Allowed || authorization.ExitCode != exitcode.Usage {
		t.Fatalf("stale cross-scope acknowledgement = %+v, %v", authorization, err)
	}
}

func TestRuntimeInfrastructureStatusMapsToRuntimeBlock(t *testing.T) {
	t.Parallel()
	checks := runtimeChecks(Input{}, []runtimecheck.Result{{
		Name: "node", SourceKind: "nvmrc", Status: runtimecheck.StatusError,
		Message: "runtime version probe failed",
	}})
	if len(checks) != 1 || checks[0].Status != StatusError || checks[0].Severity != SeverityBlock || checks[0].ExitCode != exitcode.Runtime {
		t.Fatalf("runtime error check = %+v", checks)
	}
}

type fixture struct {
	workspace string
	remote    string
	options   Options
}

func newFixture(t *testing.T, remote string) *fixture {
	t.Helper()
	workspacePath := t.TempDir()
	agentRoot := t.TempDir()
	agent := "claude"
	version := "2.1.220 (Claude Code)"
	marker := "projects"
	if strings.Contains(t.Name(), "RepositoryReplacement") {
		agent, version, marker = "codex", "codex-cli 0.146.0", "sessions"
	}
	if err := os.Mkdir(filepath.Join(agentRoot, marker), 0o700); err != nil {
		t.Fatal(err)
	}
	value := &fixture{workspace: workspacePath, remote: remote}
	value.options = Options{
		Workspace: workspace.ProbeOptions{Runner: workspace.GitRunnerFunc(func(_ context.Context, _ string, args ...string) ([]byte, error) {
			switch {
			case reflect.DeepEqual(args, []string{"rev-parse", "--path-format=absolute", "--show-toplevel"}):
				return []byte(workspacePath + "\n"), nil
			case len(args) > 0 && args[len(args)-1] == "--untracked-files=normal":
				return []byte("# branch.oid " + testHead + "\x00# branch.head main\x00"), nil
			case reflect.DeepEqual(args, []string{"rev-parse", "--is-shallow-repository"}):
				return []byte("false\n"), nil
			case reflect.DeepEqual(args, []string{"config", "--null", "--get-regexp", `^remote\..*\.url$`}):
				return []byte("remote.origin.url\n" + value.remote + "\x00"), nil
			default:
				return nil, errors.New("unexpected git probe")
			}
		})},
		Agent: agentcheck.Options{
			Root:     agentRoot,
			LookPath: func(string) (string, error) { return filepath.Join(agentRoot, agent), nil },
			Runner:   agentVersionRunner{output: agentcheck.VersionOutput{Stdout: version}},
		},
		Capability: capability.Options{GOOS: "darwin", UserHome: t.TempDir(), ProjectRoot: workspacePath, WorkingDir: workspacePath},
		Runtime:    runtimecheck.Options{Runner: versionRunner{}},
	}
	return value
}

func findCheck(t *testing.T, report Report, id string) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("check %q absent: %+v", id, report.Checks)
	return Check{}
}
