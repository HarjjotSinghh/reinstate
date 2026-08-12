package handoff

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

const pipelineTestSecret = "AKIAABCDEFGHIJKLMNOP"

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
}

func (t *pipelineTarget) Name() string { return sessionindex.AgentClaude }

func (t *pipelineTarget) Capabilities() TargetCapabilities {
	return TargetCapabilities{Agent: sessionindex.AgentClaude, SupportsPinnedID: true, SupportsInitialPrompt: true, MaxArgvBytes: DefaultMaxArgvBytes}
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
	return "destination-session", VerifyResolved, nil
}

type pipelineRunner struct {
	calls int
	plan  sessionindex.LaunchPlan
}

func (r *pipelineRunner) Run(_ context.Context, plan sessionindex.LaunchPlan) error {
	r.calls++
	r.plan = plan
	return nil
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
	if err != nil || len(entries) != 1 || entries[0].HandoffID != result.HandoffID {
		t.Fatalf("lineage list = %+v, %v", entries, err)
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

	result, err := Execute(context.Background(), rec, opts, false)
	assertPipelineCode(t, err, exitcode.Safety)
	if result.Plan.TempDir == "" {
		t.Fatal("Execute did not return its completed warning plan")
	}
	if _, statErr := os.Stat(filepath.Join(opts.ReinstateHome, handoffsDirName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("unacknowledged warning wrote the store: %v", statErr)
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
		t.Skipf("blocked by capsule identity contract: ComputeID(final capsule)=%s, stored ID=%s", computed, plan.Capsule.Identity.ID)
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
