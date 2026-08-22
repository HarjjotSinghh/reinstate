package handoff

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

var pipelineTestSecret = "AKIA" + strings.Repeat("A", 16)

type pipelineReader struct {
	events []capsule.Event
	report transcript.ParseReport
	compat adapter.Compatibility
}

func (r *pipelineReader) Name() string { return sessionindex.AgentCodex }

func (r *pipelineReader) Probe(context.Context, sessionindex.Record) (adapter.Compatibility, error) {
	if r.compat == "" {
		return adapter.CompatibilitySupported, nil
	}
	return r.compat, nil
}

func (r *pipelineReader) Snapshot(_ context.Context, rec sessionindex.Record) (transcript.Boundary, error) {
	return transcript.SnapshotJSONL(rec.SourcePath, rec.Agent, rec.ID, transcript.MaxJSONLineBytes)
}

func (r *pipelineReader) Parse(context.Context, transcript.Boundary) ([]capsule.Event, transcript.ParseReport, error) {
	return append([]capsule.Event(nil), r.events...), r.report, nil
}

type pipelineVerifier struct {
	report preflight.Report
}

func (v pipelineVerifier) Verify(_ context.Context, input preflight.Input) (preflight.Report, error) {
	report := v.report
	report.SessionRef = input.SessionRef
	return report, nil
}

type pipelineTarget struct {
	materialized int
	launched     int
	verified     int
	verifyErr    error
	maxArgvBytes int
}

type pipelineClaudeTarget struct {
	*ClaudeTarget
}

func (t *pipelineClaudeTarget) Compatible(context.Context) (adapter.Compatibility, error) {
	return adapter.CompatibilitySupported, nil
}

func (t *pipelineTarget) Name() string { return sessionindex.AgentClaude }

func (t *pipelineTarget) Capabilities() TargetCapabilities {
	maxArgvBytes := t.maxArgvBytes
	if maxArgvBytes == 0 {
		maxArgvBytes = DefaultMaxArgvBytes
	}
	return TargetCapabilities{Agent: sessionindex.AgentClaude, SupportsPinnedID: true, SupportsInitialPrompt: true, MaxArgvBytes: maxArgvBytes}
}

func (t *pipelineTarget) Compatible(context.Context) (adapter.Compatibility, error) {
	return adapter.CompatibilitySupported, nil
}

func (t *pipelineTarget) Plan(c capsule.Capsule, _ Policy) (DestinationPlan, capsule.Fidelity, error) {
	return DestinationPlan{
		Agent: sessionindex.AgentClaude, Executable: "claude",
		Args: []string{"--session-id", "00000000-0000-4000-8000-000000000001", "stub"},
		Dir:  c.Workspace.Path, SessionID: "00000000-0000-4000-8000-000000000001",
		Bootstrap: []byte("stub"),
	}, c.Fidelity, nil
}

func (t *pipelineTarget) Materialize(context.Context, DestinationPlan) error {
	t.materialized++
	return nil
}

func (t *pipelineTarget) Launch(ctx context.Context, plan DestinationPlan, runner sessionindex.LaunchRunner) error {
	t.launched++
	return runner.Run(ctx, sessionindex.LaunchPlan{
		Agent: plan.Agent, Operation: sessionindex.OperationHandoff,
		Executable: plan.Executable, Args: plan.Args, Dir: plan.Dir,
	})
}

func (t *pipelineTarget) Verify(context.Context, DestinationPlan, time.Time) (string, string, error) {
	t.verified++
	if t.verifyErr != nil {
		return "", VerifyUnresolved, t.verifyErr
	}
	return "destination-session", VerifyResolved, nil
}

type pipelineRunner struct {
	calls int
	plan  sessionindex.LaunchPlan
	err   error
}

type concurrentClaudeRunner struct {
	calls  atomic.Int32
	exists *atomic.Bool
}

func (r *concurrentClaudeRunner) Run(context.Context, sessionindex.LaunchPlan) error {
	r.calls.Add(1)
	r.exists.Store(true)
	return nil
}

func (r *pipelineRunner) Run(_ context.Context, plan sessionindex.LaunchPlan) error {
	r.calls++
	r.plan = plan
	return r.err
}

func TestPlanRedactsBeforeCheckpointAndIsDeterministic(t *testing.T) {
	rec, reader, _, target, opts := pipelineFixture(t)

	first, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(first.TempDir) })
	second, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(second.TempDir) })

	if reader.report.Events != first.Parse.Events || first.Parse.Events != 2 {
		t.Fatalf("parse report = %+v", first.Parse)
	}
	if target.materialized != 0 || target.launched != 0 || target.verified != 0 {
		t.Fatalf("Plan caused target side effects: %+v", target)
	}
	if first.HandoffID != second.HandoffID || !reflect.DeepEqual(first.Capsule, second.Capsule) {
		t.Fatal("equal synthetic inputs produced different capsule plans")
	}
	if !bytes.Equal(first.Artifacts.ProjectionMD, second.Artifacts.ProjectionMD) ||
		!bytes.Equal(first.Artifacts.Bootstrap, second.Artifacts.Bootstrap) ||
		!bytes.Equal(first.Artifacts.SidecarEvents, second.Artifacts.SidecarEvents) {
		t.Fatal("equal synthetic inputs produced different artifacts")
	}

	canonical, err := capsule.CanonicalBytes(first.Capsule)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{
		"capsule": canonical, "projection": first.Artifacts.ProjectionMD,
		"bootstrap": first.Artifacts.Bootstrap, "sidecar": first.Artifacts.SidecarEvents,
	} {
		if bytes.Contains(body, []byte(pipelineTestSecret)) {
			t.Fatalf("%s leaked the synthetic secret", name)
		}
	}
	if strings.Contains(first.Capsule.Task.LatestUserIntent.Text, pipelineTestSecret) {
		t.Fatal("checkpoint was derived before transcript redaction")
	}
	if !strings.Contains(first.Capsule.Task.LatestUserIntent.Text, "[redacted:aws_key:") {
		t.Fatalf("latest intent lacks a redaction marker: %q", first.Capsule.Task.LatestUserIntent.Text)
	}
	if first.RedactionCounts[string(capsule.CategoryAWSKey)] == 0 {
		t.Fatalf("redaction counts = %v", first.RedactionCounts)
	}
	if first.Capsule.Task.Constraints.Portability != capsule.PortabilityOmitted ||
		first.Capsule.Task.Constraints.Reason != reasonRequiresOptionalSummarizer {
		t.Fatalf("constraints were invented: %+v", first.Capsule.Task.Constraints)
	}
	if first.Destination.Dir != rec.Workspace || first.Destination.Executable != "claude" {
		t.Fatalf("destination plan = %+v", first.Destination)
	}
	if first.EstimatedBytes <= 0 || first.EstimatedTokens <= 0 || len(first.PlannedFiles) < 4 {
		t.Fatalf("incomplete preview: bytes=%d tokens=%d files=%v", first.EstimatedBytes, first.EstimatedTokens, first.PlannedFiles)
	}
	if first.Capsule.Projection.BootstrapSHA256 != sha256Hex(first.Destination.Bootstrap) ||
		first.Capsule.Projection.MarkdownSHA256 != sha256Hex(first.Artifacts.ProjectionMD) {
		t.Fatalf("projection hashes do not describe exact artifacts: %+v", first.Capsule.Projection)
	}
	for _, path := range first.PlannedFiles {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Plan wrote permanent path %q: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(first.TempDir, first.HandoffID, capsuleFileName)); err != nil {
		t.Fatalf("preview capsule: %v", err)
	}
}

func TestPlanCheckpointSidecarOmitsVerbatimBodies(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	opts.Policy = PolicyCheckpoint
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if len(plan.Capsule.Conversation.Events) != 0 {
		t.Fatalf("checkpoint included %d events", len(plan.Capsule.Conversation.Events))
	}
	if len(plan.Artifacts.SidecarEvents) == 0 {
		t.Fatal("checkpoint sidecar refs were dropped")
	}
	for _, forbidden := range []string{"work remains", "continue with key", pipelineTestSecret, `"blocks"`} {
		if bytes.Contains(plan.Artifacts.SidecarEvents, []byte(forbidden)) {
			t.Fatalf("checkpoint sidecar retained verbatim %q: %s", forbidden, plan.Artifacts.SidecarEvents)
		}
	}
	if !bytes.Contains(plan.Artifacts.SidecarEvents, []byte(`"event_id"`)) &&
		!bytes.Contains(plan.Artifacts.SidecarEvents, []byte(`"EventID"`)) {
		t.Fatalf("checkpoint sidecar missing refs: %s", plan.Artifacts.SidecarEvents)
	}
}

func TestPlanReportsMissingDestinationMCPAndSkill(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	user := t.TempDir()
	codexHome := filepath.Join(user, ".codex")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(`
[mcp_servers.browser]
command = "mcp-browser"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(user, ".agents", "skills", "review")
	if err := os.MkdirAll(skill, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# review\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts.Capability = capability.Options{
		GOOS:        runtime.GOOS,
		UserHome:    user,
		ClaudeHome:  filepath.Join(user, ".claude"),
		CodexHome:   codexHome,
		ProjectRoot: rec.Workspace,
		WorkingDir:  rec.Workspace,
	}

	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })

	foundMCP, foundSkill := false, false
	for _, missing := range plan.Capsule.Capabilities.Missing {
		if missing.Kind == KindMCP && missing.Name == "browser" {
			foundMCP = true
		}
		if missing.Kind == KindSkill && missing.Name == "review" {
			foundSkill = true
		}
	}
	if !foundMCP || !foundSkill {
		t.Fatalf("Missing = %+v, want mcp browser and skill review", plan.Capsule.Capabilities.Missing)
	}
	ids := strings.Join(plan.WarningIDs, " ")
	if !strings.Contains(ids, "handoff.capability.mcp.browser") ||
		!strings.Contains(ids, "handoff.capability.skill.review") {
		t.Fatalf("WarningIDs = %v", plan.WarningIDs)
	}

	opts.LaunchRunner = &pipelineRunner{}
	_, err = Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Safety)
}

func TestPlanActiveSourceSafety(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	opts.SessionBusy = func(context.Context, string, processcheck.Target) (bool, bool, error) {
		return true, true, nil
	}

	_, err := Plan(context.Background(), rec, opts)
	assertPipelineCode(t, err, exitcode.Safety)

	opts.AllowActive = true
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if !plan.SourceMayHaveAdvanced || !plan.Capsule.RawSource.Partial {
		t.Fatalf("active boundary not marked: %+v", plan.Capsule.RawSource)
	}
}

func TestPlanListingErrorDoesNotBlock(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	opts.SessionBusy = func(context.Context, string, processcheck.Target) (bool, bool, error) {
		return false, false, errors.New("exit status 1")
	}
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatalf("listing error blocked Plan: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
}

func TestPlanRefreshesAndResolvesSourceFirst(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	resolved := rec
	resolved.ID = "resolved-session"
	resolved.Key = "codex:resolved-session"
	calls := 0
	opts.ResolveSource = func(_ context.Context, got sessionindex.Record) (sessionindex.Record, bool, error) {
		calls++
		if got.Reference() != rec.Reference() {
			t.Fatalf("resolver input = %q, want %q", got.Reference(), rec.Reference())
		}
		return resolved, true, nil
	}
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if calls != 1 || plan.Capsule.RawSource.SessionID != resolved.ID {
		t.Fatalf("resolver calls=%d raw source=%+v", calls, plan.Capsule.RawSource)
	}

	opts.ResolveSource = func(context.Context, sessionindex.Record) (sessionindex.Record, bool, error) {
		return resolved, false, nil
	}
	_, err = Plan(context.Background(), rec, opts)
	assertPipelineCode(t, err, exitcode.Compatibility)

	resolverErrors := []struct {
		name string
		err  error
		code int
	}{
		{"ambiguous", sessionindex.ErrAmbiguous, exitcode.Conflict},
		{"not found", sessionindex.ErrNotFound, exitcode.Usage},
		{"runtime", errors.New("scan failed"), exitcode.Runtime},
	}
	for _, tt := range resolverErrors {
		t.Run(tt.name, func(t *testing.T) {
			copy := opts
			copy.ResolveSource = func(context.Context, sessionindex.Record) (sessionindex.Record, bool, error) {
				return sessionindex.Record{}, false, tt.err
			}
			_, err := Plan(context.Background(), rec, copy)
			assertPipelineCode(t, err, tt.code)
		})
	}
}

func TestPlanRejectsInvalidOptionsBeforeSideEffects(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	tests := []struct {
		name   string
		mutate func(*Options)
	}{
		{"same agent", func(o *Options) { o.ToAgent = rec.Agent }},
		{"bad policy", func(o *Options) { o.Policy = "everything" }},
		{"wildcard warning", func(o *Options) { o.AllowWarnings = []string{"handoff.*"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy := opts
			mkdirCalls := 0
			copy.MkdirTemp = func(string, string) (string, error) {
				mkdirCalls++
				return t.TempDir(), nil
			}
			tt.mutate(&copy)
			_, err := Plan(context.Background(), rec, copy)
			assertPipelineCode(t, err, exitcode.Usage)
			if mkdirCalls != 0 {
				t.Fatalf("invalid options created %d preview directories", mkdirCalls)
			}
		})
	}
}

func TestPlanRequiresResolverAndBusyCheck(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	opts.ResolveSource = nil
	_, err := Plan(context.Background(), rec, opts)
	assertPipelineCode(t, err, exitcode.Runtime)

	_, _, _, _, opts = pipelineFixture(t)
	opts.SessionBusy = nil
	_, err = Plan(context.Background(), rec, opts)
	assertPipelineCode(t, err, exitcode.Runtime)
}

func TestPlanDerivesNewestAncestorLineageRoot(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	writePipelineLineage(t, opts.ReinstateHome,
		LineageEntry{
			HandoffID: "handoff-old", LineageRoot: "ancestor-old",
			Destination: LineageEndpoint{Agent: rec.Agent, SessionID: rec.ID, State: VerifyResolved},
			Launched:    true,
		},
		LineageEntry{
			HandoffID: "handoff-no-launch", LineageRoot: "ignored-no-launch",
			Destination: LineageEndpoint{Agent: rec.Agent, SessionID: rec.ID, State: VerifyResolved},
			Launched:    false,
		},
		LineageEntry{
			HandoffID: "handoff-unresolved", LineageRoot: "ignored-unresolved",
			Destination: LineageEndpoint{Agent: rec.Agent, SessionID: rec.ID, State: VerifyUnresolved},
			Launched:    true,
		},
	)
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if plan.LineageRoot != "ancestor-old" || plan.Capsule.Identity.LineageRoot != "ancestor-old" {
		t.Fatalf("lineage root = %q / %q", plan.LineageRoot, plan.Capsule.Identity.LineageRoot)
	}
	computed, err := capsule.ComputeID(plan.Capsule)
	if err != nil || computed != plan.HandoffID {
		t.Fatalf("descendant content ID = %q, %v; want %q", computed, err, plan.HandoffID)
	}
}

func TestPlanWithoutExistingLineageStartsSelfRooted(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if plan.LineageRoot != plan.HandoffID || plan.Capsule.Identity.LineageRoot != plan.HandoffID {
		t.Fatalf("new lineage root = %q / %q, handoff=%q", plan.LineageRoot, plan.Capsule.Identity.LineageRoot, plan.HandoffID)
	}
	if _, err := os.Stat(filepath.Join(opts.ReinstateHome, handoffsDirName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Plan created handoff store during read-only lineage lookup: %v", err)
	}
}

func TestPlanLineageLookupIgnoresMalformedAndPartialLines(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	writePipelineLineage(t, opts.ReinstateHome, LineageEntry{
		HandoffID: "handoff-valid", LineageRoot: "ancestor-valid",
		Destination: LineageEndpoint{Agent: rec.Agent, SessionID: rec.ID, State: VerifyResolved},
		Launched:    true,
	})
	path := filepath.Join(opts.ReinstateHome, handoffsDirName, lineageFileName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("not-json\n{\"handoff_id\":\"partial"); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if plan.LineageRoot != "ancestor-valid" {
		t.Fatalf("lineage root = %q, want valid complete entry", plan.LineageRoot)
	}
}

func TestExecutePersistsLaunchesVerifiesAndRecordsLineage(t *testing.T) {
	rec, _, _, target, opts := pipelineFixture(t)
	runner := &pipelineRunner{}
	opts.LaunchRunner = runner
	opts.Now = func() time.Time { return time.Date(2026, 8, 12, 6, 0, 0, 0, time.UTC) }

	result, err := Execute(context.Background(), rec, opts, true)
	if err != nil {
		t.Fatal(err)
	}
	if target.materialized != 1 || target.launched != 1 || target.verified != 1 || runner.calls != 1 {
		t.Fatalf("execute calls: target=%+v runner=%d", target, runner.calls)
	}
	if !result.Launched || result.DestinationSessionID != "destination-session" || result.DestinationState != VerifyResolved {
		t.Fatalf("execute result = %+v", result)
	}
	if result.Lineage.HandoffID != result.HandoffID || result.Lineage.Destination.SessionID != "destination-session" ||
		result.Lineage.CreatedAt != opts.Now() {
		t.Fatalf("lineage = %+v", result.Lineage)
	}
	if _, err := os.Stat(result.Plan.TempDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Execute left preview temp dir: %v", err)
	}
	store, err := OpenStore(opts.ReinstateHome)
	if err != nil {
		t.Fatal(err)
	}
	stored, artifacts, err := store.Get(result.HandoffID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Identity.ID != result.HandoffID || !bytes.Equal(artifacts.ProjectionMD, result.Plan.Artifacts.ProjectionMD) {
		t.Fatal("stored handoff differs from the executed plan")
	}
	entries, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if !lineageHasOutcome(entries, result.HandoffID, true, VerifyResolved) {
		t.Fatalf("lineage list = %+v", entries)
	}
}

func TestExecuteRecordsLineageWhenPostLaunchVerifyFails(t *testing.T) {
	rec, _, _, target, opts := pipelineFixture(t)
	runner := &pipelineRunner{}
	opts.LaunchRunner = runner
	target.verifyErr = errors.New("reconciliation unavailable")

	result, err := Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Runtime)
	if !result.Launched || result.DestinationState != VerifyUnresolved {
		t.Fatalf("result = %+v", result)
	}
	store, openErr := OpenStore(opts.ReinstateHome)
	if openErr != nil {
		t.Fatal(openErr)
	}
	entries, listErr := store.List(10)
	if listErr != nil || !lineageHasOutcome(entries, result.HandoffID, true, VerifyUnresolved) {
		t.Fatalf("lineage after verify failure = %+v, %v", entries, listErr)
	}
}

func TestExecuteRecordsLineageWhenLaunchReturnsError(t *testing.T) {
	rec, _, _, target, opts := pipelineFixture(t)
	runner := &pipelineRunner{err: errors.New("child exit status 1")}
	opts.LaunchRunner = runner

	result, err := Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Runtime)
	if result.Launched || result.DestinationState != VerifyUnresolved {
		t.Fatalf("launch-error result = %+v", result)
	}
	if target.verified != 0 {
		t.Fatalf("pre-spawn failure called Verify %d times", target.verified)
	}
	store, openErr := OpenStore(opts.ReinstateHome)
	if openErr != nil {
		t.Fatal(openErr)
	}
	entries, listErr := store.List(10)
	if listErr != nil || !lineageHasOutcome(entries, result.HandoffID, false, VerifyUnresolved) {
		t.Fatalf("lineage after launch error = %+v, %v", entries, listErr)
	}
}

func TestExecuteReconcilesAndRecordsTypedPostSpawnError(t *testing.T) {
	rec, _, _, target, opts := pipelineFixture(t)
	exitErr := errors.New("child exit status 17")
	runner := &pipelineRunner{err: fmt.Errorf("%w: %w", sessionindex.ErrChildStarted, exitErr)}
	opts.LaunchRunner = runner

	result, err := Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Runtime)
	if !errors.Is(err, exitErr) {
		t.Fatalf("returned error = %v, want original child exit", err)
	}
	if !result.Launched || result.DestinationState != VerifyResolved || target.verified != 1 {
		t.Fatalf("post-spawn result=%+v verify calls=%d", result, target.verified)
	}
	store, openErr := OpenStore(opts.ReinstateHome)
	if openErr != nil {
		t.Fatal(openErr)
	}
	entries, listErr := store.List(10)
	if listErr != nil || !lineageHasOutcome(entries, result.HandoffID, true, VerifyResolved) {
		t.Fatalf("post-spawn lineage = %+v, %v", entries, listErr)
	}
}

func TestExecuteRequiresWarningsBeforeWritingStore(t *testing.T) {
	rec, _, verifier, _, opts := pipelineFixture(t)
	verifier.report.Decision = preflight.DecisionConfirmationRequired
	verifier.report.Checks[0] = preflight.Check{
		ID: "git.branch", Status: preflight.StatusChanged, Severity: preflight.SeverityWarning,
		Provenance: workspace.ProvenanceCurrentObservation, Message: "branch changed", ExitCode: exitcode.Safety,
	}
	opts.Verifier = verifier
	opts.LaunchRunner = &pipelineRunner{}

	result, err := Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Safety)
	if result.Plan.TempDir == "" {
		t.Fatal("Execute did not return its completed warning plan")
	}
	if _, statErr := os.Stat(filepath.Join(opts.ReinstateHome, handoffsDirName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("unacknowledged warning wrote the store: %v", statErr)
	}
}

func TestExecuteNoLaunchPersistsUnackedWarnings(t *testing.T) {
	rec, _, verifier, _, opts := pipelineFixture(t)
	verifier.report.Decision = preflight.DecisionConfirmationRequired
	verifier.report.Checks[0] = preflight.Check{
		ID: "git.branch", Status: preflight.StatusChanged, Severity: preflight.SeverityWarning,
		Provenance: workspace.ProvenanceCurrentObservation, Message: "branch changed", ExitCode: exitcode.Safety,
	}
	opts.Verifier = verifier

	result, err := Execute(context.Background(), rec, opts, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Launched || result.HandoffID == "" {
		t.Fatalf("no-launch result = %+v", result)
	}
	store, err := OpenStore(opts.ReinstateHome)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := store.List(10)
	if err != nil || len(entries) != 1 || entries[0].HandoffID != result.HandoffID || entries[0].Launched {
		t.Fatalf("lineage after unacked --no-launch = %+v, %v", entries, err)
	}
}

func TestExecuteRequiresRunnerBeforeWritingStore(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	result, err := Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Runtime)
	if result.Plan.TempDir == "" {
		t.Fatal("Execute did not return its completed plan")
	}
	if _, statErr := os.Stat(filepath.Join(opts.ReinstateHome, handoffsDirName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("missing runner wrote the store: %v", statErr)
	}
}

func TestPlanCapsuleContentIDContract(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	computed, err := capsule.ComputeID(plan.Capsule)
	if err != nil {
		t.Fatal(err)
	}
	if computed != plan.Capsule.Identity.ID {
		t.Fatalf("ComputeID(final capsule)=%s, stored ID=%s", computed, plan.Capsule.Identity.ID)
	}
}

func TestArgvUnsafeForLaunchWindowsNewlines(t *testing.T) {
	if argvUnsafeForLaunch("linux", []byte("a\nb")) {
		t.Fatal("non-windows argv may contain newlines")
	}
	if !argvUnsafeForLaunch("windows", []byte("a\nb")) {
		t.Fatal("windows argv must not contain LF")
	}
	if !argvUnsafeForLaunch("windows", []byte("a\rb")) {
		t.Fatal("windows argv must not contain CR")
	}
	if argvUnsafeForLaunch("windows", []byte("one line")) {
		t.Fatal("windows one-line argv is safe")
	}
}

func TestPlanDestinationNewlineBootstrapUsesShortArgv(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows CreateProcess truncates argv at CR/LF")
	}
	target := &pipelineTarget{maxArgvBytes: 64 << 10}
	home := filepath.Join(t.TempDir(), "reinstate-home")
	c := capsule.Capsule{
		Identity:  capsule.Identity{ID: "capsule-id"},
		Workspace: capsule.Workspace{Path: t.TempDir()},
	}
	rendered := []byte("structured handoff, not native resume\n\n## Goal\nRC9 dest-ack")
	plan, _, err := planDestination(
		context.Background(), target, c, PolicyBalanced,
		Options{ReinstateHome: home}, rendered,
	)
	if err != nil {
		t.Fatal(err)
	}
	if argvUnsafeForLaunch("windows", plan.Bootstrap) {
		t.Fatalf("bootstrap still has newlines: %q", plan.Bootstrap)
	}
	want := filepath.Join(home, handoffsDirName, c.Identity.ID, projectionFile)
	if !filepath.IsAbs(want) || !strings.Contains(string(plan.Bootstrap), want) {
		t.Fatalf("fallback bootstrap = %q, want absolute projection path %q", plan.Bootstrap, want)
	}
	if !strings.Contains(string(plan.Bootstrap), firstReplyAckOneLine()) {
		t.Fatalf("fallback bootstrap missing five-bullet ack: %q", plan.Bootstrap)
	}
	if len(plan.Args) == 0 || plan.Args[len(plan.Args)-1] != string(plan.Bootstrap) {
		t.Fatalf("argv = %#v, bootstrap = %q", plan.Args, plan.Bootstrap)
	}
	for _, arg := range plan.Args {
		if strings.ContainsAny(arg, "\r\n") {
			t.Fatalf("argv still has newline: %#v", plan.Args)
		}
	}
}

func TestPlanDestinationArgvFallbackUsesAbsoluteProjectionPath(t *testing.T) {
	target := &pipelineTarget{maxArgvBytes: 1024}
	home := filepath.Join(t.TempDir(), "reinstate-home")
	c := capsule.Capsule{
		Identity:  capsule.Identity{ID: "capsule-id"},
		Workspace: capsule.Workspace{Path: t.TempDir()},
	}
	plan, _, err := planDestination(
		context.Background(), target, c, PolicyBalanced,
		Options{ReinstateHome: home}, bytes.Repeat([]byte("x"), 2048),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, handoffsDirName, c.Identity.ID, projectionFile)
	if !filepath.IsAbs(want) || !strings.Contains(string(plan.Bootstrap), want) {
		t.Fatalf("fallback bootstrap = %q, want absolute projection path %q", plan.Bootstrap, want)
	}
	if !strings.Contains(string(plan.Bootstrap), firstReplyAckOneLine()) {
		t.Fatalf("byte-budget fallback missing five-bullet ack: %q", plan.Bootstrap)
	}
}

func TestClaudePipelineDryRunAndExecuteHaveByteIdenticalArgv(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	checks := 0
	sessionExists := func(context.Context, string) (bool, error) {
		checks++
		return false, nil
	}
	target := &ClaudeTarget{
		SessionExists: sessionExists,
		Bootstrap: func(c capsule.Capsule, _ Policy) ([]byte, error) {
			return RenderBootstrap(c, permanentHandoffDir(opts.ReinstateHome, c.Identity.ID))
		},
	}
	opts.Target = &pipelineClaudeTarget{ClaudeTarget: target}
	opts.SessionExists = sessionExists
	first, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(first.TempDir) })
	executed, err := Execute(context.Background(), rec, opts, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Destination.Args, executed.Plan.Destination.Args) ||
		!bytes.Equal(first.Destination.Bootstrap, executed.Plan.Destination.Bootstrap) {
		t.Fatalf("dry-run/execute plans differ: dry=%+v execute=%+v", first.Destination, executed.Plan.Destination)
	}
	if checks != 2 {
		t.Fatalf("dry-run + no-launch collision checks = %d, want planning checks only", checks)
	}
}

func TestExecuteClaudeFinalCollisionCheckPreventsLaunch(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	var checks atomic.Int32
	sessionExists := func(context.Context, string) (bool, error) {
		return checks.Add(1) > 1, nil
	}
	target := &ClaudeTarget{
		ConfigDir:     t.TempDir(),
		SessionExists: sessionExists,
		Bootstrap: func(c capsule.Capsule, _ Policy) ([]byte, error) {
			return RenderBootstrap(c, permanentHandoffDir(opts.ReinstateHome, c.Identity.ID))
		},
	}
	runner := &pipelineRunner{}
	opts.Target = &pipelineClaudeTarget{ClaudeTarget: target}
	opts.SessionExists = sessionExists
	opts.LaunchRunner = runner

	result, err := Execute(context.Background(), rec, opts, true)
	assertPipelineCode(t, err, exitcode.Safety)
	if !errors.Is(err, ErrClaudeSessionIDCollision) || runner.calls != 0 || checks.Load() != 2 {
		t.Fatalf("collision result: err=%v runner=%d checks=%d", err, runner.calls, checks.Load())
	}
	artifactDir := filepath.Join(opts.ReinstateHome, handoffsDirName, result.Plan.Capsule.Identity.ID)
	if _, statErr := os.Stat(artifactDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("collision persisted capsule artifacts: %v", statErr)
	}
}

func TestConcurrentClaudeExecutionsCannotBothLaunch(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	var checks atomic.Int32
	var exists atomic.Bool
	initialReady := make(chan struct{})
	sessionExists := func(context.Context, string) (bool, error) {
		n := checks.Add(1)
		if n <= 2 {
			if n == 2 {
				close(initialReady)
			}
			<-initialReady
			return false, nil
		}
		return exists.Load(), nil
	}
	target := &ClaudeTarget{
		ConfigDir:     t.TempDir(),
		SessionExists: sessionExists,
		Bootstrap: func(c capsule.Capsule, _ Policy) ([]byte, error) {
			return RenderBootstrap(c, permanentHandoffDir(opts.ReinstateHome, c.Identity.ID))
		},
	}
	runner := &concurrentClaudeRunner{exists: &exists}
	opts.Target = &pipelineClaudeTarget{ClaudeTarget: target}
	opts.SessionExists = sessionExists
	opts.LaunchRunner = runner

	type outcome struct {
		result ExecuteResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	for range 2 {
		go func() {
			result, err := Execute(context.Background(), rec, opts, true)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	var succeeded, collided int
	for range 2 {
		out := <-outcomes
		switch {
		case out.err == nil && out.result.Launched:
			succeeded++
		case errors.Is(out.err, ErrClaudeSessionIDCollision):
			collided++
		default:
			t.Fatalf("unexpected concurrent outcome: result=%+v err=%v", out.result, out.err)
		}
	}
	if succeeded != 1 || collided != 1 || runner.calls.Load() != 1 {
		t.Fatalf("concurrent outcomes: success=%d collision=%d launches=%d", succeeded, collided, runner.calls.Load())
	}
}

func TestResolveTargetClaudeRequiresCollisionCheck(t *testing.T) {
	_, err := resolveTarget(Options{}, sessionindex.AgentClaude)
	assertPipelineCode(t, err, exitcode.Runtime)

	target, err := resolveTarget(Options{
		SessionExists: func(context.Context, string) (bool, error) { return false, nil },
	}, sessionindex.AgentClaude)
	if err != nil || target == nil {
		t.Fatalf("resolveTarget with collision check = %T, %v", target, err)
	}
}

func TestPlanRefusesDifferentGitRepository(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	initTestGitRepo(t, rec.Workspace)
	other := t.TempDir()
	initTestGitRepo(t, other)
	opts.WorkingDir = other

	_, err := Plan(context.Background(), rec, opts)
	assertPipelineCode(t, err, exitcode.Compatibility)
	if !errors.Is(err, ErrCompatibility) || !strings.Contains(err.Error(), "different repository") {
		t.Fatalf("wrong-repo error = %v", err)
	}

	opts.WorkingDir = filepath.Join(rec.Workspace, "src")
	if err := os.MkdirAll(opts.WorkingDir, 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatalf("same-repo subdirectory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })

	opts.WorkingDir = t.TempDir()
	plan, err = Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatalf("cwd outside any git repo: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
}

func TestPlanFidelityReportIncludesOmittedTaskFields(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	opts.WorkingDir = rec.Workspace
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	seen := map[capsule.Portability]bool{}
	for _, component := range plan.Capsule.Fidelity.Components {
		seen[component.Portability] = true
	}
	if !seen[capsule.PortabilityExact] || !seen[capsule.PortabilityOmitted] {
		t.Fatalf("fidelity components = %+v, want exact and omitted", plan.Capsule.Fidelity.Components)
	}
}

func TestPlanFidelityReportKeepsSummarizedClass(t *testing.T) {
	rec, reader, _, _, opts := pipelineFixture(t)
	reader.events = append(reader.events, capsule.Event{
		ID: "summary-1", Order: 2, Actor: capsule.ActorHarness, Kind: capsule.KindSummary,
		Blocks:      []capsule.Block{{Type: capsule.BlockTypeText, Text: "vendor summary of prior turns"}},
		Portability: capsule.PortabilitySummarized, Reason: "vendor_compaction_summary",
		ContentHash: "summary-hash",
		Source:      capsule.SourcePointer{Agent: rec.Agent, SessionID: rec.ID, RecordKey: "summary-1", Index: 2},
	})
	opts.WorkingDir = rec.Workspace
	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	var found bool
	for _, component := range plan.Capsule.Fidelity.Components {
		if component.Portability == capsule.PortabilitySummarized {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fidelity components = %+v, want summarized", plan.Capsule.Fidelity.Components)
	}
}

func TestPlanRemapsForeignOSWorkspaceOntoLocalGitRoot(t *testing.T) {
	rec, _, verifier, _, opts := pipelineFixture(t)
	local := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(local, 0o700); err != nil {
		t.Fatal(err)
	}
	initTestGitRepo(t, local)
	foreign := `C:\Users\fixture-user\code\demo`
	if runtime.GOOS == "windows" {
		foreign = `/Users/fixture-user/code/demo`
	}
	rec.Workspace = foreign
	opts.WorkingDir = local
	capture := &capturingVerifier{report: verifier.report}
	opts.Verifier = capture

	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if capture.workspace != local {
		t.Fatalf("Verify workspace = %q, want remapped git root %q", capture.workspace, local)
	}
	if !strings.HasPrefix(plan.Capsule.Workspace.Root, "${REPO:") {
		t.Fatalf("capsule root = %q, want ${REPO:…}", plan.Capsule.Workspace.Root)
	}
	if strings.Contains(plan.Capsule.Workspace.Root, `C:`) || strings.Contains(plan.Capsule.Workspace.Root, "/Users/fixture-user") {
		t.Fatalf("capsule leaked source-device path: %q", plan.Capsule.Workspace.Root)
	}
}

func TestPlanRemapsFixtureUserWorkspaceOntoLocalGitRoot(t *testing.T) {
	rec, _, verifier, _, opts := pipelineFixture(t)
	local := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(local, 0o700); err != nil {
		t.Fatal(err)
	}
	initTestGitRepo(t, local)
	rec.Workspace = `/Users/fixture-user/code/demo`
	opts.WorkingDir = local
	capture := &capturingVerifier{report: verifier.report}
	opts.Verifier = capture

	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	if capture.workspace != local {
		t.Fatalf("Verify workspace = %q, want remapped git root %q", capture.workspace, local)
	}
	if strings.Contains(plan.Capsule.Workspace.Root, "/Users/fixture-user") {
		t.Fatalf("capsule leaked fixture path: %q", plan.Capsule.Workspace.Root)
	}
}

func TestPlanRefusesFixtureRemapOntoDifferentGitRepository(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	other := filepath.Join(t.TempDir(), "other")
	if err := os.MkdirAll(other, 0o700); err != nil {
		t.Fatal(err)
	}
	initTestGitRepo(t, other)
	rec.Workspace = `/Users/fixture-user/code/demo`
	opts.WorkingDir = other

	_, err := Plan(context.Background(), rec, opts)
	assertPipelineCode(t, err, exitcode.Compatibility)
	if !errors.Is(err, ErrCompatibility) || !strings.Contains(err.Error(), "different repository") {
		t.Fatalf("wrong-repo after fixture remap = %v", err)
	}
}

func TestRemapForeignWorkspace(t *testing.T) {
	t.Parallel()
	local := t.TempDir()
	if got := remapForeignWorkspace(`C:\Users\fixture-user\code\demo`, ""); got != `C:\Users\fixture-user\code\demo` {
		t.Fatalf("empty cwd remapped to %q", got)
	}
	if runtime.GOOS == "windows" {
		if !shouldRemapWorkspace(`/Users/fixture-user/code/demo`) {
			t.Fatal("posix fixture path should remap on Windows")
		}
		if shouldRemapWorkspace(`C:\Users\harjot\code\demo`) {
			t.Fatal("same-OS Windows path should not remap")
		}
		return
	}
	if !shouldRemapWorkspace(`C:\Users\fixture-user\code\demo`) {
		t.Fatal("Windows os-roots path should remap on Unix")
	}
	if shouldRemapWorkspace(local) {
		t.Fatal("same-OS local path should not remap")
	}
	if !shouldRemapWorkspace(`/Users/fixture-user/code/demo`) {
		t.Fatal("synthetic fixture-user path should remap")
	}
	demo := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(demo, 0o700); err != nil {
		t.Fatal(err)
	}
	got := remapForeignWorkspace(`/Users/fixture-user/code/demo`, demo)
	absDemo, err := filepath.Abs(demo)
	if err != nil {
		t.Fatal(err)
	}
	if got != absDemo && got != demo {
		t.Fatalf("matching leaf remapped to %q, want %q", got, absDemo)
	}
	other := filepath.Join(t.TempDir(), "other")
	if err := os.MkdirAll(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := remapForeignWorkspace(`/Users/fixture-user/code/demo`, other); got != `/Users/fixture-user/code/demo` {
		t.Fatalf("different leaf remapped to %q", got)
	}
}

type capturingVerifier struct {
	report    preflight.Report
	workspace string
}

func (v *capturingVerifier) Verify(_ context.Context, input preflight.Input) (preflight.Report, error) {
	v.workspace = input.Workspace
	report := v.report
	report.SessionRef = input.SessionRef
	report.Workspace.Workspace.Path = input.Workspace
	report.Workspace.Git.Root = input.Workspace
	return report, nil
}

func initTestGitRepo(t *testing.T, dir string) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	command := exec.Command(git, "init", "--quiet", dir)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=reinstate", "GIT_AUTHOR_EMAIL=reinstate@example.invalid")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v: %s", dir, err, output)
	}
}

func pipelineFixture(t *testing.T) (sessionindex.Record, *pipelineReader, *pipelineVerifier, *pipelineTarget, Options) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source.jsonl")
	if err := os.WriteFile(source, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := capsule.SourcePointer{Agent: sessionindex.AgentCodex, SessionID: "source-session", RecordKey: "event-1", Index: 0}
	events := []capsule.Event{
		{
			ID: capsule.EventID(src), Order: 0, Actor: capsule.ActorUser, Kind: capsule.KindMessage,
			Blocks: []capsule.Block{{Type: capsule.BlockTypeText, Text: "continue with key " + pipelineTestSecret,
				Meta: map[string]string{"authorization": "Bearer " + pipelineTestSecret}}},
			Portability: capsule.PortabilityExact, ContentHash: "event-hash-1", Source: src,
		},
		{
			ID:    capsule.EventID(capsule.SourcePointer{Agent: src.Agent, SessionID: src.SessionID, RecordKey: "event-2", Index: 1}),
			Order: 1, Actor: capsule.ActorAssistant, Kind: capsule.KindMessage,
			Blocks:      []capsule.Block{{Type: capsule.BlockTypeText, Text: "work remains"}},
			Portability: capsule.PortabilityExact, ContentHash: "event-hash-2",
			Source: capsule.SourcePointer{Agent: src.Agent, SessionID: src.SessionID, RecordKey: "event-2", Index: 1},
		},
	}
	reader := &pipelineReader{events: events, report: transcript.ParseReport{Events: len(events), ByKind: map[capsule.Kind]int{capsule.KindMessage: len(events)}}}
	report := preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		Decision:      preflight.DecisionReady,
		Checks: []preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Provenance: workspace.ProvenanceCurrentObservation, Message: "source is fresh",
		}},
		Workspace: workspace.Fingerprint{
			SchemaVersion: workspace.SchemaVersion,
			Provenance:    workspace.ProvenanceCurrentObservation,
			Workspace:     workspace.WorkspaceFingerprint{Path: root, Exists: true, Directory: true},
			Git: workspace.GitFingerprint{
				Available: true, Repository: true, Root: root, RepositoryID: "github.com/example/repo",
				Branch: "feature", Head: "abc123",
				WorkingTree: workspace.WorkingTreeFingerprint{State: workspace.WorkingTreeModified, Digest: "tree-digest"},
			},
		},
		Agent: agentcheck.Result{Agent: sessionindex.AgentCodex, Version: "0.145.0", Status: agentcheck.StatusSupported},
	}
	verifier := &pipelineVerifier{report: report}
	target := &pipelineTarget{}
	rec := sessionindex.Record{
		Key: "codex:source-session", ID: "source-session", Agent: sessionindex.AgentCodex,
		Project: "github.com/example/repo", Workspace: root, SourcePath: source, SourceSize: 3,
	}
	opts := Options{
		ToAgent: sessionindex.AgentClaude, Policy: PolicyBalanced, Reader: reader,
		Verifier: verifier, Target: target, ReinstateHome: filepath.Join(t.TempDir(), "reinstate-home"),
		ResolveSource: func(_ context.Context, input sessionindex.Record) (sessionindex.Record, bool, error) {
			return input, true, nil
		},
		SessionBusy: func(context.Context, string, processcheck.Target) (bool, bool, error) {
			return false, true, nil
		},
	}
	return rec, reader, verifier, target, opts
}

func assertPipelineCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want exit code %d", want)
	}
	var pipelineErr *PipelineError
	if !errors.As(err, &pipelineErr) || pipelineErr.Code != want {
		t.Fatalf("error = %T %v, want PipelineError code %d", err, err, want)
	}
}

func writePipelineLineage(t *testing.T, home string, entries ...LineageEntry) {
	t.Helper()
	dir := filepath.Join(home, handoffsDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		body.Write(line)
		body.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, lineageFileName), body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func lineageHasOutcome(entries []LineageEntry, id string, launched bool, state string) bool {
	for _, e := range entries {
		if e.HandoffID == id && e.Launched == launched && e.Destination.State == state {
			return true
		}
	}
	return false
}

type listDuringLaunchRunner struct {
	home       string
	n          int
	hasLineage bool
}

func (r *listDuringLaunchRunner) Run(context.Context, sessionindex.LaunchPlan) error {
	store, err := OpenStore(r.home)
	if err != nil {
		return err
	}
	entries, err := store.List(10)
	if err != nil {
		return err
	}
	r.n = len(entries)
	_, statErr := os.Stat(filepath.Join(store.Root(), lineageFileName))
	r.hasLineage = statErr == nil
	return nil
}

func TestExecuteRecordsLineageBeforeLaunchReturns(t *testing.T) {
	rec, _, _, _, opts := pipelineFixture(t)
	runner := &listDuringLaunchRunner{home: opts.ReinstateHome}
	opts.LaunchRunner = runner
	if _, err := Execute(context.Background(), rec, opts, true); err != nil {
		t.Fatal(err)
	}
	if runner.n < 1 {
		t.Fatalf("lineage visible during launch = %d, want >= 1", runner.n)
	}
	if !runner.hasLineage {
		t.Fatal("lineage.jsonl must exist before dest Launch returns")
	}
}

// TestRewriteBootstrapArgsKeepsDestinationFlags is the regression for a launch
// defect a physical journey caught: the pipeline re-renders the briefing after
// the target has planned, and it used to rebuild argv from a per-agent switch
// whose default was a single positional element. That is Codex's shape. For any
// destination that passes its prompt behind a flag it discarded both the flag
// and the pinned session id, so the launch created a session Verify could never
// resolve.
func TestRewriteBootstrapArgsKeepsDestinationFlags(t *testing.T) {
	t.Parallel()
	rendered := []byte("RENDERED_BRIEFING")
	tests := []struct {
		name string
		plan DestinationPlan
		want []string
	}{
		{
			name: "qwen keeps the pinned id and the prompt flag",
			plan: DestinationPlan{
				Agent:     "qwen",
				SessionID: "c94d7e0a-596e-481a-b4d4-f5518222b968",
				Args: []string{
					"--session-id", "c94d7e0a-596e-481a-b4d4-f5518222b968",
					"--prompt-interactive", "PLANNED_BOOTSTRAP",
				},
				Bootstrap: []byte("PLANNED_BOOTSTRAP"),
			},
			want: []string{
				"--session-id", "c94d7e0a-596e-481a-b4d4-f5518222b968",
				"--prompt-interactive", "RENDERED_BRIEFING",
			},
		},
		{
			name: "claude keeps the pinned id",
			plan: DestinationPlan{
				Agent:     "claude",
				SessionID: "11111111-2222-4333-8444-555555555555",
				Args:      []string{"--session-id", "11111111-2222-4333-8444-555555555555", "PLANNED_BOOTSTRAP"},
				Bootstrap: []byte("PLANNED_BOOTSTRAP"),
			},
			want: []string{"--session-id", "11111111-2222-4333-8444-555555555555", "RENDERED_BRIEFING"},
		},
		{
			name: "codex stays positional",
			plan: DestinationPlan{
				Agent:     "codex",
				Args:      []string{"PLANNED_BOOTSTRAP"},
				Bootstrap: []byte("PLANNED_BOOTSTRAP"),
			},
			want: []string{"RENDERED_BRIEFING"},
		},
		{
			name: "qwen falls back to the documented shape when no element carried the bootstrap",
			plan: DestinationPlan{
				Agent:     "qwen",
				SessionID: "c94d7e0a-596e-481a-b4d4-f5518222b968",
				Args:      []string{"--session-id", "c94d7e0a-596e-481a-b4d4-f5518222b968"},
			},
			want: []string{
				"--session-id", "c94d7e0a-596e-481a-b4d4-f5518222b968",
				"--prompt-interactive", "RENDERED_BRIEFING",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := rewriteBootstrapArgs(tt.plan, rendered)
			if !reflect.DeepEqual(got.Args, tt.want) {
				t.Fatalf("args = %v, want %v", got.Args, tt.want)
			}
		})
	}
}
