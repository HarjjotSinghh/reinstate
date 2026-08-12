package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

type fakeCodexLaunchRunner struct {
	plans []sessionindex.LaunchPlan
	err   error
}

func (r *fakeCodexLaunchRunner) Run(_ context.Context, plan sessionindex.LaunchPlan) error {
	r.plans = append(r.plans, plan)
	return r.err
}

func TestCodexTargetRegistered(t *testing.T) {
	t.Parallel()

	got, ok := Target(sessionindex.AgentCodex)
	if !ok {
		t.Fatal("codex target not registered")
	}
	caps := got.Capabilities()
	if caps.SupportsPinnedID {
		t.Fatal("SupportsPinnedID = true, want false")
	}
	if !caps.SupportsInitialPrompt {
		t.Fatal("SupportsInitialPrompt = false, want true")
	}
	if caps.MaxArgvBytes != DefaultMaxArgvBytes {
		t.Fatalf("MaxArgvBytes = %d, want %d", caps.MaxArgvBytes, DefaultMaxArgvBytes)
	}
}

func TestCodexTargetPlanArgvAndDir(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := NewCodexTarget(nil)
	plan, _, err := target.Plan(testCodexCapsule(workspace, "ship handoff", "finish WP-17"), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Executable != "codex" || len(plan.Args) != 1 {
		t.Fatalf("argv = %q %v", plan.Executable, plan.Args)
	}
	if plan.Dir != workspace {
		t.Fatalf("Dir = %q, want %q", plan.Dir, workspace)
	}
	if plan.SessionID != "" {
		t.Fatalf("SessionID = %q, want empty", plan.SessionID)
	}
	if string(plan.Bootstrap) != plan.Args[0] {
		t.Fatal("Bootstrap does not match argv prompt")
	}
	if !strings.Contains(string(plan.Bootstrap), "projection.md") {
		t.Fatalf("bootstrap missing projection.md pointer: %q", plan.Bootstrap)
	}
}

func TestCodexTargetArgvFallbackAtCeiling(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	// Force the full bootstrap over budget so Plan must use the short form.
	// Short argv is ~146 bytes; full with long goal/intent is ~565.
	target := NewCodexTarget(&CodexTarget{MaxArgvBytes: 200})
	capBody := testCodexCapsule(workspace, strings.Repeat("G", 200), strings.Repeat("I", 200))
	plan, _, err := target.Plan(capBody, PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	wantShort := string(buildCodexBootstrap(capBody, true))
	if string(plan.Bootstrap) != wantShort || plan.Args[0] != wantShort {
		t.Fatalf("fallback bootstrap = %q, want %q", plan.Bootstrap, wantShort)
	}
	if !strings.Contains(wantShort, "projection.md") {
		t.Fatal("short bootstrap must reference projection.md only")
	}
	if err := ValidateDestinationArgv(plan, target.Capabilities().MaxArgvBytes); err != nil {
		t.Fatalf("short bootstrap still over budget: %v", err)
	}
}

func TestCodexTargetArgvFallbackStillOverBudget(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := NewCodexTarget(&CodexTarget{MaxArgvBytes: 8})
	_, _, err := target.Plan(testCodexCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if !errors.Is(err, ErrArgvExceedsBudget) {
		t.Fatalf("Plan error = %v, want %v", err, ErrArgvExceedsBudget)
	}
}

func TestCodexTargetLaunchUsesFakeRunner(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := NewCodexTarget(nil)
	plan, _, err := target.Plan(testCodexCapsule(workspace, "goal", "intent"), PolicyCheckpoint)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runner := &fakeCodexLaunchRunner{}
	if err := target.Launch(context.Background(), plan, runner); err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if len(runner.plans) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.plans))
	}
	got := runner.plans[0]
	if got.Executable != "codex" || got.Dir != workspace || len(got.Args) != 1 || got.Args[0] != string(plan.Bootstrap) {
		t.Fatalf("launch plan = %+v", got)
	}
	if got.Operation != codexLaunchOperation {
		t.Fatalf("Operation = %q, want %q", got.Operation, codexLaunchOperation)
	}
}

func TestCodexTargetVerifyResolvesSingleMatch(t *testing.T) {
	t.Parallel()

	codexRoot := t.TempDir()
	workspace := t.TempDir()
	target := NewCodexTarget(&CodexTarget{Root: codexRoot, ForceCompat: adapter.CompatibilitySupported})
	plan, _, err := target.Plan(testCodexCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	launchStart := time.Now().UTC().Add(-time.Minute)
	writeCodexRollout(t, codexRoot, "11111111-1111-4111-8111-111111111111", workspace, string(plan.Bootstrap), launchStart.Add(time.Second))

	id, state, err := target.Verify(context.Background(), plan, launchStart)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if state != VerifyResolved || id != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("Verify = %q/%q, want resolved/1111…", id, state)
	}
}

func TestCodexTargetVerifyAmbiguousTwoCandidates(t *testing.T) {
	t.Parallel()

	codexRoot := t.TempDir()
	workspace := t.TempDir()
	target := NewCodexTarget(&CodexTarget{Root: codexRoot})
	plan, _, err := target.Plan(testCodexCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	launchStart := time.Now().UTC().Add(-time.Minute)
	bootstrap := string(plan.Bootstrap)
	writeCodexRollout(t, codexRoot, "22222222-2222-4222-8222-222222222221", workspace, bootstrap, launchStart.Add(time.Second))
	writeCodexRollout(t, codexRoot, "22222222-2222-4222-8222-222222222222", workspace, bootstrap, launchStart.Add(2*time.Second))

	id, state, err := target.Verify(context.Background(), plan, launchStart)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if state != VerifyAmbiguous || id != "" {
		t.Fatalf("Verify = %q/%q, want ambiguous/empty", id, state)
	}
}

func TestCodexTargetVerifyIgnoresOlderRollout(t *testing.T) {
	t.Parallel()

	codexRoot := t.TempDir()
	workspace := t.TempDir()
	target := NewCodexTarget(&CodexTarget{Root: codexRoot})
	plan, _, err := target.Plan(testCodexCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	launchStart := time.Now().UTC()
	bootstrap := string(plan.Bootstrap)

	// Pre-existing older rollout in the same workspace with the same text must
	// never be selected, even when it is the only candidate.
	writeCodexRollout(t, codexRoot, "33333333-3333-4333-8333-333333333333", workspace, bootstrap, launchStart.Add(-2*time.Hour))

	id, state, err := target.Verify(context.Background(), plan, launchStart)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if state != VerifyUnresolved || id != "" {
		t.Fatalf("Verify = %q/%q, want unresolved (older rollout ignored)", id, state)
	}

	// A post-launch match resolves.
	writeCodexRollout(t, codexRoot, "44444444-4444-4444-8444-444444444444", workspace, bootstrap, launchStart.Add(time.Second))
	id, state, err = target.Verify(context.Background(), plan, launchStart)
	if err != nil {
		t.Fatalf("Verify after new rollout: %v", err)
	}
	if state != VerifyResolved || id != "44444444-4444-4444-8444-444444444444" {
		t.Fatalf("Verify = %q/%q, want resolved/4444…", id, state)
	}
}

func TestCodexTargetVerifyBootstrapHashMismatchUnresolved(t *testing.T) {
	t.Parallel()

	codexRoot := t.TempDir()
	workspace := t.TempDir()
	target := NewCodexTarget(&CodexTarget{Root: codexRoot})
	plan, _, err := target.Plan(testCodexCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	launchStart := time.Now().UTC().Add(-time.Minute)
	writeCodexRollout(t, codexRoot, "55555555-5555-4555-8555-555555555555", workspace, "different prompt", launchStart.Add(time.Second))

	id, state, err := target.Verify(context.Background(), plan, launchStart)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if state != VerifyUnresolved || id != "" {
		t.Fatalf("Verify = %q/%q, want unresolved", id, state)
	}
	// Sanity: hash helper matches sha256 of bootstrap bytes.
	sum := sha256.Sum256(plan.Bootstrap)
	if sha256Hex(plan.Bootstrap) != hex.EncodeToString(sum[:]) {
		t.Fatal("sha256Hex mismatch")
	}
}

func TestCodexTargetCompatibleUsesSyntheticRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := NewCodexTarget(&CodexTarget{Root: root})
	compat, err := target.Compatible(context.Background())
	if err != nil {
		t.Fatalf("Compatible: %v", err)
	}
	if compat != adapter.CompatibilitySupported {
		t.Fatalf("compat = %q, want SUPPORTED", compat)
	}
}

func testCodexCapsule(workspace, goal, intent string) capsule.Capsule {
	return capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			ID:        "cap-codex-test",
			SchemaVer: capsule.SchemaVersion,
		},
		Workspace: capsule.Workspace{Path: workspace, ProjectID: "proj", Root: "${REPO:proj}"},
		Task: capsule.Task{
			Goal:             capsule.TextField{Text: goal, Portability: capsule.PortabilityNormalized},
			LatestUserIntent: capsule.TextField{Text: intent, Portability: capsule.PortabilityExact},
		},
		Fidelity: capsule.Fidelity{Mode: capsule.FidelityModeStructuredHandoff},
	}
}

func writeCodexRollout(t *testing.T, root, id, cwd, userMsg string, mtime time.Time) {
	t.Helper()
	dir := filepath.Join(root, "sessions", "2026", "08", "12")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "rollout-2026-08-12T12-00-00-" + id + ".jsonl"
	path := filepath.Join(dir, name)
	body := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"` + id + `","session_id":"` + id + `","cwd":` + jsonQuote(cwd) + `}}`,
		`{"type":"event_msg","payload":{"type":"user_message","message":` + jsonQuote(userMsg) + `}}`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func jsonQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
