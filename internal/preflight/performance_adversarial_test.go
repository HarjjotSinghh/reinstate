package preflight

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

const (
	phase3SyntheticSamples = 20
	phase3Head             = "1111111111111111111111111111111111111111"
)

// Absolute device latency is certified with installed artifacts. This test
// guards the development ceiling and, more importantly, asserts the exact
// bounded subprocess count so a fast CI host cannot hide unbounded work.
func TestWarmVerifySyntheticLatencyAndProbeCount(t *testing.T) {
	fixture := newPhase3PerformanceFixture(t)
	first, err := Verify(context.Background(), fixture.input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := BaselineFromReport(first, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	fixture.input.Baseline = &baseline

	gitBefore := fixture.git.calls.Load()
	agentBefore := fixture.agent.calls.Load()
	runtimeBefore := fixture.runtime.calls.Load()
	durations := make([]time.Duration, 0, phase3SyntheticSamples)
	for range phase3SyntheticSamples {
		started := time.Now()
		report, verifyErr := Verify(context.Background(), fixture.input, fixture.options)
		durations = append(durations, time.Since(started))
		if verifyErr != nil {
			t.Fatal(verifyErr)
		}
		if report.Decision != DecisionReady {
			t.Fatalf("warm decision = %s, checks=%+v", report.Decision, report.Checks)
		}
	}

	if got, want := fixture.git.calls.Load()-gitBefore, int64(4*phase3SyntheticSamples); got != want {
		t.Fatalf("warm Git probe count = %d, want %d", got, want)
	}
	if got, want := fixture.agent.calls.Load()-agentBefore, int64(phase3SyntheticSamples); got != want {
		t.Fatalf("warm agent probe count = %d, want %d", got, want)
	}
	if got, want := fixture.runtime.calls.Load()-runtimeBefore, int64(2*phase3SyntheticSamples); got != want {
		t.Fatalf("warm runtime probe count = %d, want %d", got, want)
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := durations[(95*len(durations)-1)/100]
	if p95 > time.Second {
		t.Logf("synthetic warm p95 missed the Apple Silicon target: %s", p95)
	}
	// Absolute wall-clock certification belongs to the installed-artifact
	// device matrix. Exact probe counts above are the deterministic CI guard;
	// overloaded shared runners must not turn scheduler delay into a product
	// regression.
	t.Logf("synthetic warm samples=%d median=%s p95=%s max=%s", len(durations), durations[len(durations)/2], p95, durations[len(durations)-1])
}

func TestVerifyHonorsParentCancellationAndSharedDeadline(t *testing.T) {
	t.Run("parent cancellation", func(t *testing.T) {
		fixture := newPhase3PerformanceFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		started := time.Now()
		_, err := Verify(ctx, fixture.input, fixture.options)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Verify() error = %v, want context cancellation", err)
		}
		if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
			t.Fatalf("cancelled Verify() took %s", elapsed)
		}
	})

	t.Run("shared deadline", func(t *testing.T) {
		fixture := newPhase3PerformanceFixture(t)
		fixture.options.Timeout = 25 * time.Millisecond
		fixture.options.Agent.Runner = phase3BlockingAgentRunner{}
		started := time.Now()
		report, err := Verify(context.Background(), fixture.input, fixture.options)
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
			t.Fatalf("deadline-bounded Verify() took %s", elapsed)
		}
		if report.Decision != DecisionBlocked || report.BlockExitCode != exitcode.Runtime {
			t.Fatalf("deadline report = %s/%d, checks=%+v", report.Decision, report.BlockExitCode, report.Checks)
		}
		if check := phase3FindCheck(report, "agent.version"); check.Status != StatusError || check.Severity != SeverityBlock {
			t.Fatalf("agent deadline check = %+v", check)
		}
		if check := phase3FindCheckByActual(report, string(capability.DiagnosticCancelled)); check.Status != StatusError || check.Severity != SeverityBlock {
			t.Fatalf("capability deadline check = %+v", check)
		}
	})
}

func TestVerifyAdversarialInputsStayBoundedAndPrivate(t *testing.T) {
	fixture := newPhase3PerformanceFixture(t)
	const secret = "private-test-marker"

	fixture.git.status = []byte("# branch.oid " + phase3Head + "\x00# branch.head main\x00? \x1b[31m" + secret + "\u202e\xff\x00")
	fixture.git.remote = []byte("remote.origin.url\nhttps://user:" + secret + "@example.invalid/private/" + secret + ".git?token=" + secret + "#" + secret + "\x00")

	if err := os.Remove(filepath.Join(fixture.workspace, ".nvmrc")); err != nil {
		t.Fatal(err)
	}
	oversizedRuntime := append([]byte(`{"engines":{"node":"`+secret+`"},"padding":"`), bytes.Repeat([]byte("x"), (1<<20)+64)...)
	if err := os.WriteFile(filepath.Join(fixture.workspace, "package.json"), oversizedRuntime, 0o600); err != nil {
		t.Fatal(err)
	}
	oversizedGo := append([]byte("module example.invalid/"+secret+"\ngo 1.25.0\n"), bytes.Repeat([]byte("x"), (1<<20)+64)...)
	if err := os.WriteFile(filepath.Join(fixture.workspace, "go.mod"), oversizedGo, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(fixture.agentRoot, ".claude.json"), []byte(`{"mcpServers":{"safe":{"command":"`+secret), 0o600); err != nil {
		t.Fatal(err)
	}
	oversizedMCP := append([]byte(`{"mcpServers":{"safe":{"command":"`+secret+`"}},"padding":"`), bytes.Repeat([]byte("x"), (1<<20)+64)...)
	if err := os.WriteFile(filepath.Join(fixture.workspace, ".mcp.json"), oversizedMCP, 0o600); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 320; index++ {
		dir := filepath.Join(fixture.agentRoot, "skills", fmt.Sprintf("bounded-skill-%03d", index))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(secret+"\x1b[2J\u202e"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	report, err := Verify(context.Background(), fixture.input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Capabilities.Items) > 256 {
		t.Fatalf("capability inventory is unbounded: %d items", len(report.Capabilities.Items))
	}
	if len(report.Checks) > 512 {
		t.Fatalf("preflight check set is unexpectedly large: %d", len(report.Checks))
	}
	if len(report.Runtimes) != 2 || report.Runtimes[0].Status != runtimecheck.StatusUnknown || report.Runtimes[1].Status != runtimecheck.StatusUnknown {
		t.Fatalf("oversized runtime declarations = %+v", report.Runtimes)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > 512<<10 {
		t.Fatalf("encoded report is unexpectedly large: %d bytes", len(encoded))
	}
	for _, forbidden := range []string{secret, "user:", "token=", "\x1b[", "\u202e", fixture.workspace, fixture.agentRoot, fixture.userHome} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("environment report leaked %q", forbidden)
		}
	}
	if report.Decision != DecisionConfirmationRequired {
		t.Fatalf("adversarial bounded inputs decision = %s, checks=%+v", report.Decision, report.Checks)
	}
	if !phase3HasCapabilityDiagnostic(report, capability.DiagnosticOversized) || !phase3HasCapabilityDiagnostic(report, capability.DiagnosticMalformed) || !phase3HasCapabilityDiagnostic(report, capability.DiagnosticLimitReached) {
		t.Fatalf("missing bounded capability diagnostics: %+v", report.Capabilities.Diagnostics)
	}
}

func TestMalformedWorkspaceOutputBecomesFixedRuntimeBlock(t *testing.T) {
	fixture := newPhase3PerformanceFixture(t)
	const secret = "RAW-WORKSPACE-FAILURE-SENTINEL"
	fixture.git.status = []byte("# branch.head " + secret + "\x00# branch.head other\x00\x1b[31m")
	report, err := Verify(context.Background(), fixture.input, fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionBlocked || report.BlockExitCode != exitcode.Runtime {
		t.Fatalf("malformed workspace report = %s/%d", report.Decision, report.BlockExitCode)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "\x1b") {
		t.Fatalf("malformed workspace output leaked into report: %s", encoded)
	}
	if check := phase3FindCheck(report, "git.status"); check.Status != StatusError || check.ExitCode != exitcode.Runtime {
		t.Fatalf("malformed workspace check = %+v", check)
	}
}

func FuzzValidateReportBoundary(f *testing.F) {
	f.Add("baseline.unavailable", "safe message")
	f.Add("../unsafe", "\x1b[31msecret")
	f.Add(strings.Repeat("a", 300), "oversized")
	f.Fuzz(func(t *testing.T, id, message string) {
		check := Check{
			ID: id, Status: StatusUnknown, Severity: SeverityWarning,
			Provenance: workspace.ProvenanceUnavailable, Message: message,
			ExitCode: exitcode.Safety,
		}
		checks := normalizeChecks([]Check{check})
		decision, code := aggregate(checks)
		report := Report{
			SchemaVersion: SchemaVersion,
			SessionRef:    "claude:controlled",
			Decision:      decision,
			BlockExitCode: code,
			Checks:        checks,
		}
		err := validateReport(report)
		if err == nil && !validCheckID(id) {
			t.Fatalf("invalid check ID accepted: %q", id)
		}
	})
}

func BenchmarkWarmVerifySynthetic(b *testing.B) {
	fixture := newPhase3PerformanceFixture(b)
	first, err := Verify(context.Background(), fixture.input, fixture.options)
	if err != nil {
		b.Fatal(err)
	}
	baseline, err := BaselineFromReport(first, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		b.Fatal(err)
	}
	fixture.input.Baseline = &baseline
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Verify(context.Background(), fixture.input, fixture.options); err != nil {
			b.Fatal(err)
		}
	}
}

type phase3PerformanceFixture struct {
	workspace string
	agentRoot string
	userHome  string
	input     Input
	options   Options
	git       *phase3CountingGitRunner
	agent     *phase3CountingAgentRunner
	runtime   *phase3CountingRuntimeRunner
}

func newPhase3PerformanceFixture(tb testing.TB) *phase3PerformanceFixture {
	tb.Helper()
	value := &phase3PerformanceFixture{
		workspace: tb.TempDir(),
		agentRoot: tb.TempDir(),
		userHome:  tb.TempDir(),
	}
	if err := os.Mkdir(filepath.Join(value.agentRoot, "projects"), 0o700); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(value.workspace, ".nvmrc"), []byte("20.11.1\n"), 0o600); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(value.workspace, "go.mod"), []byte("module example.invalid/test\n\ngo 1.25.12\n"), 0o600); err != nil {
		tb.Fatal(err)
	}
	value.git = &phase3CountingGitRunner{
		root:   value.workspace,
		status: []byte("# branch.oid " + phase3Head + "\x00# branch.head main\x00"),
		remote: []byte("remote.origin.url\nhttps://example.invalid/team/repo.git\x00"),
	}
	value.agent = &phase3CountingAgentRunner{}
	value.runtime = &phase3CountingRuntimeRunner{}
	if err := os.WriteFile(filepath.Join(value.agentRoot, "claude"), []byte("controlled agent"), 0o700); err != nil {
		tb.Fatal(err)
	}
	value.input = Input{
		SessionRef:  "claude:phase3-performance",
		Agent:       "claude",
		Workspace:   value.workspace,
		AgentRoot:   value.agentRoot,
		SourceFresh: true,
	}
	value.options = Options{
		Timeout:   2 * time.Second,
		Workspace: workspace.ProbeOptions{Runner: value.git},
		Agent: agentcheck.Options{
			Root: value.agentRoot,
			LookPath: func(string) (string, error) {
				return filepath.Join(value.agentRoot, "claude"), nil
			},
			Runner: value.agent,
		},
		Capability: capability.Options{GOOS: "darwin", UserHome: value.userHome},
		Runtime:    runtimecheck.Options{Runner: value.runtime},
	}
	return value
}

type phase3CountingGitRunner struct {
	calls  atomic.Int64
	root   string
	status []byte
	remote []byte
}

func (runner *phase3CountingGitRunner) Run(ctx context.Context, _ string, args ...string) ([]byte, error) {
	runner.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch {
	case phase3EqualArgs(args, "rev-parse", "--path-format=absolute", "--show-toplevel"):
		return []byte(runner.root + "\n"), nil
	case len(args) != 0 && args[len(args)-1] == "--untracked-files=normal":
		return append([]byte(nil), runner.status...), nil
	case phase3EqualArgs(args, "rev-parse", "--is-shallow-repository"):
		return []byte("false\n"), nil
	case phase3EqualArgs(args, "config", "--local", "--no-includes", "--null", "--get-regexp", `^remote\..*\.url$`):
		return append([]byte(nil), runner.remote...), nil
	default:
		return nil, errors.New("unexpected synthetic Git probe")
	}
}

type phase3CountingAgentRunner struct {
	calls atomic.Int64
}

func (runner *phase3CountingAgentRunner) Version(ctx context.Context, _ string, _ ...string) (agentcheck.VersionOutput, error) {
	runner.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return agentcheck.VersionOutput{}, err
	}
	return agentcheck.VersionOutput{Stdout: "2.1.220 (Claude Code)"}, nil
}

type phase3BlockingAgentRunner struct{}

func (phase3BlockingAgentRunner) Version(ctx context.Context, _ string, _ ...string) (agentcheck.VersionOutput, error) {
	<-ctx.Done()
	return agentcheck.VersionOutput{}, ctx.Err()
}

type phase3CountingRuntimeRunner struct {
	calls atomic.Int64
}

func (runner *phase3CountingRuntimeRunner) Version(ctx context.Context, name string, _ ...string) (string, error) {
	runner.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	switch name {
	case "node":
		return "v20.11.1\n", nil
	case "go":
		return "go version go1.25.12 darwin/arm64\n", nil
	default:
		return "", runtimecheck.ErrExecutableNotFound
	}
}

func phase3EqualArgs(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func phase3FindCheck(report Report, id string) Check {
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	return Check{}
}

func phase3FindCheckByActual(report Report, actual string) Check {
	for _, check := range report.Checks {
		if check.Actual == actual {
			return check
		}
	}
	return Check{}
}

func phase3HasCapabilityDiagnostic(report Report, code capability.DiagnosticCode) bool {
	for _, diagnostic := range report.Capabilities.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
