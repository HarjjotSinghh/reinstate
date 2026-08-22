package handoff

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"

	_ "modernc.org/sqlite"
)

type fakeOpenCodeLaunchRunner struct {
	plans []sessionindex.LaunchPlan
	err   error
}

func (r *fakeOpenCodeLaunchRunner) Run(_ context.Context, plan sessionindex.LaunchPlan) error {
	r.plans = append(r.plans, plan)
	return r.err
}

func testOpenCodeCapsule(workspace, goal, intent string) capsule.Capsule {
	return capsule.Capsule{
		Schema:    capsule.Schema,
		Identity:  capsule.Identity{ID: "cap-opencode-test", SchemaVer: capsule.SchemaVersion},
		Workspace: capsule.Workspace{Path: workspace, ProjectID: "proj", Root: "${REPO:proj}"},
		Task: capsule.Task{
			Goal:             capsule.TextField{Text: goal, Portability: capsule.PortabilityNormalized},
			LatestUserIntent: capsule.TextField{Text: intent, Portability: capsule.PortabilityExact},
		},
		Fidelity: capsule.Fidelity{Mode: capsule.FidelityModeStructuredHandoff},
	}
}

// openCodeStore builds a store with OpenCode's real table shapes: the message
// row carries the role and the text lives in its part rows.
func openCodeStore(t *testing.T, root string) *sql.DB {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, transcript.OpenCodeDatabaseName))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL,
			directory TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL,
			session_id TEXT NOT NULL, data TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	return db
}

// plantOpenCodeSession writes one session whose first human turn is text.
func plantOpenCodeSession(t *testing.T, db *sql.DB, id, directory, text string, created time.Time) {
	t.Helper()
	stamp := created.UnixMilli()
	if _, err := db.Exec(`INSERT INTO session VALUES (?,?,?,?,?,?)`,
		id, "p1", "planted", directory, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	messageID := "msg_" + id
	// An assistant row is written first with an earlier id so that ordering by
	// time and role, not by insertion, is what finds the human turn.
	if _, err := db.Exec(`INSERT INTO message VALUES (?,?,?,?)`,
		"msg_a_"+id, id, stamp+1, `{"role":"assistant"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO message VALUES (?,?,?,?)`,
		messageID, id, stamp, `{"role":"user"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO part VALUES (?,?,?,?)`,
		"prt_"+id, messageID, id, `{"type":"text","text":`+jsonQuote(text)+`}`); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeTargetRegistered(t *testing.T) {
	t.Parallel()

	got, ok := Target(sessionindex.AgentOpenCode)
	if !ok {
		t.Fatal("opencode target not registered")
	}
	caps := got.Capabilities()
	// Measured: `opencode --session <unknown-id>` refuses with "Session not
	// found" and creates nothing, so the ID cannot be pinned before launch.
	if caps.SupportsPinnedID {
		t.Fatal("SupportsPinnedID = true, want false; OpenCode assigns the session id")
	}
	if !caps.SupportsInitialPrompt {
		t.Fatal("SupportsInitialPrompt = false, want true")
	}
	if caps.MaxArgvBytes != DefaultMaxArgvBytes {
		t.Fatalf("MaxArgvBytes = %d, want %d", caps.MaxArgvBytes, DefaultMaxArgvBytes)
	}
}

func TestOpenCodeTargetPlanArgvAndDir(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	plan, _, err := NewOpenCodeTarget(nil).
		Plan(testOpenCodeCapsule(workspace, "ship the handoff", "finish the target"), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Executable != "opencode" {
		t.Fatalf("Executable = %q", plan.Executable)
	}
	// The prompt is the value of a flag, not a bare positional. OpenCode's
	// default command reads a bare positional as a project path, so a
	// Codex-shaped `opencode "<bootstrap>"` would try to start in a directory
	// named after the whole briefing.
	if len(plan.Args) != 2 || plan.Args[0] != OpenCodeNewSessionFlag {
		t.Fatalf("argv = %v, want [%s <bootstrap>]", plan.Args, OpenCodeNewSessionFlag)
	}
	if plan.Args[1] != string(plan.Bootstrap) {
		t.Fatal("Bootstrap does not match the argv prompt")
	}
	if plan.Dir != workspace {
		t.Fatalf("Dir = %q, want %q", plan.Dir, workspace)
	}
	if plan.SessionID != "" {
		t.Fatalf("SessionID = %q, want empty; the vendor assigns it", plan.SessionID)
	}
	if !strings.Contains(string(plan.Bootstrap), "projection.md") {
		t.Fatalf("bootstrap missing projection.md pointer: %q", plan.Bootstrap)
	}
	if !strings.Contains(string(plan.Bootstrap), "not native resume") {
		t.Fatalf("bootstrap must say a handoff is not a native resume: %q", plan.Bootstrap)
	}
}

func TestOpenCodeTargetRefusesPlanWithoutWorkspace(t *testing.T) {
	t.Parallel()

	if _, _, err := NewOpenCodeTarget(nil).
		Plan(testOpenCodeCapsule("", "goal", "intent"), PolicyBalanced); err == nil {
		t.Fatal("a capsule with no verified workspace planned a launch")
	}
}

func TestOpenCodeTargetArgvFallbackAtCeiling(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	body := testOpenCodeCapsule(workspace, strings.Repeat("G", 200), strings.Repeat("I", 200))
	target := NewOpenCodeTarget(&OpenCodeTarget{MaxArgvBytes: 450})
	plan, _, err := target.Plan(body, PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := string(buildOpenCodeBootstrap(body, true))
	if string(plan.Bootstrap) != want || plan.Args[1] != want {
		t.Fatalf("fallback bootstrap = %q, want %q", plan.Bootstrap, want)
	}
}

// TestOpenCodeTargetMaterializeWritesNothing is the ADR 0003 boundary for this
// vendor, and the reason the target exists in this shape at all: OpenCode's
// sessions live in one embedded database the vendor owns, so materialization
// must create no file of any kind.
func TestOpenCodeTargetMaterializeWritesNothing(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	root := t.TempDir()
	openCodeStore(t, root)
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(filepath.Join(root, transcript.OpenCodeDatabaseName+suffix))
	}
	before, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	target := NewOpenCodeTarget(&OpenCodeTarget{Root: root})
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Materialize(context.Background(), plan); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	// Verify reads the store; both together must leave the root untouched.
	if _, _, err := target.Verify(context.Background(), plan, time.Now()); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	after, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		var names []string
		for _, entry := range after {
			names = append(names, entry.Name())
		}
		t.Fatalf("the destination wrote under the agent root: %v", names)
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("the destination wrote into the workspace: %d entries", len(entries))
	}
}

func TestOpenCodeTargetMaterializeRefusesPlannedFiles(t *testing.T) {
	t.Parallel()

	target := NewOpenCodeTarget(nil)
	plan := DestinationPlan{
		Agent: sessionindex.AgentOpenCode, Executable: "opencode",
		Args:  []string{OpenCodeNewSessionFlag, "x"},
		Dir:   t.TempDir(),
		Files: []PlannedFile{{Path: filepath.Join(t.TempDir(), "planted.json")}},
	}
	if err := target.Materialize(context.Background(), plan); err == nil {
		t.Fatal("a plan carrying vendor files was materialized")
	}
}

func TestOpenCodeTargetLaunchUsesPlannedArgv(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := NewOpenCodeTarget(nil)
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeOpenCodeLaunchRunner{}
	if err := target.Launch(context.Background(), plan, runner); err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if len(runner.plans) != 1 {
		t.Fatalf("launched %d times", len(runner.plans))
	}
	got := runner.plans[0]
	if got.Agent != sessionindex.AgentOpenCode || got.Operation != openCodeLaunchOperation {
		t.Fatalf("plan = %+v", got)
	}
	if got.Dir != workspace || got.Args[0] != OpenCodeNewSessionFlag {
		t.Fatalf("plan = %+v", got)
	}
	if got.SessionRef != "" {
		t.Fatalf("SessionRef = %q; a handoff starts a new session and has no source ref", got.SessionRef)
	}
	if err := target.Launch(context.Background(), plan, nil); err == nil {
		t.Fatal("Launch without a runner was allowed")
	}
}

// TestOpenCodeTargetVerifyReconcilesTheNewSession covers the reconciliation
// table. The destination session id is not knowable at launch, so a handoff
// resolves it afterwards or says honestly that it could not.
func TestOpenCodeTargetVerifyReconcilesTheNewSession(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	root := t.TempDir()
	db := openCodeStore(t, root)
	target := NewOpenCodeTarget(&OpenCodeTarget{Root: root})
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := string(plan.Bootstrap)
	launch := time.Now().Add(-time.Minute)

	tests := []struct {
		name  string
		plant func()
		wantI string
		want  string
	}{
		{
			name:  "no candidate",
			plant: func() {},
			want:  VerifyUnresolved,
		},
		{
			name: "session in another workspace",
			plant: func() {
				plantOpenCodeSession(t, db, "ses_elsewhere", t.TempDir(), bootstrap, time.Now())
			},
			want: VerifyUnresolved,
		},
		{
			name: "session that predates the launch",
			plant: func() {
				plantOpenCodeSession(t, db, "ses_older", workspace, bootstrap, launch.Add(-time.Hour))
			},
			want: VerifyUnresolved,
		},
		{
			name: "session whose first turn is a different prompt",
			plant: func() {
				plantOpenCodeSession(t, db, "ses_other_prompt", workspace, "unrelated work", time.Now())
			},
			want: VerifyUnresolved,
		},
		{
			name: "the launched session",
			plant: func() {
				plantOpenCodeSession(t, db, "ses_launched", workspace, bootstrap, time.Now())
			},
			wantI: "ses_launched",
			want:  VerifyResolved,
		},
		{
			name: "two indistinguishable candidates",
			plant: func() {
				plantOpenCodeSession(t, db, "ses_twin", workspace, bootstrap, time.Now())
			},
			want: VerifyAmbiguous,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.plant()
			id, state, err := target.Verify(context.Background(), plan, launch)
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if state != test.want || id != test.wantI {
				t.Fatalf("Verify = %q/%q, want %q/%q", id, state, test.wantI, test.want)
			}
		})
	}
}

// TestOpenCodeTargetVerifyRefusesIncompletePlan keeps reconciliation from
// guessing when it has nothing to match on.
func TestOpenCodeTargetVerifyRefusesIncompletePlan(t *testing.T) {
	t.Parallel()

	target := NewOpenCodeTarget(&OpenCodeTarget{Root: t.TempDir()})
	if _, _, err := target.Verify(context.Background(), DestinationPlan{Bootstrap: []byte("x")}, time.Now()); err == nil {
		t.Fatal("Verify accepted a plan with no workspace")
	}
	if _, _, err := target.Verify(context.Background(), DestinationPlan{Dir: t.TempDir()}, time.Now()); err == nil {
		t.Fatal("Verify accepted a plan with no bootstrap")
	}
}

// TestOpenCodeTargetVerifyOpensTheStoreReadOnly is the property the whole
// embedded-store design rests on: reconciliation must not write under the
// vendor's root.
//
// It asserts the outcome rather than the mechanism. An earlier version checked
// that a DSN string carried mode=ro, immutable=1 and query_only(1), which
// stopped being how the store is opened: immutable=1 also hides the
// write-ahead log, so a session the vendor had only just created was invisible
// to the very reconciliation that caused it. The store is now read through a
// private copy, and what has to remain true is that nothing appears beside the
// original.
func TestOpenCodeTargetVerifyOpensTheStoreReadOnly(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	root := t.TempDir()
	db := openCodeStore(t, root)
	target := NewOpenCodeTarget(&OpenCodeTarget{Root: root})
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	plantOpenCodeSession(t, db, "ses_launched", workspace, string(plan.Bootstrap), time.Now())
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(filepath.Join(root, transcript.OpenCodeDatabaseName+suffix))
	}

	if _, state, err := target.Verify(context.Background(), plan, time.Now().Add(-time.Minute)); err != nil || state != VerifyResolved {
		t.Fatalf("Verify = %q, %v", state, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != transcript.OpenCodeDatabaseName {
			t.Fatalf("Verify left %q beside the vendor store", entry.Name())
		}
	}
}

// TestOpenCodeTargetVerifyWithoutAStoreIsUnresolved keeps an absent store an
// honest "could not resolve" rather than an error the operator cannot act on.
func TestOpenCodeTargetVerifyWithoutAStoreIsUnresolved(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := NewOpenCodeTarget(&OpenCodeTarget{Root: t.TempDir()})
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	id, state, err := target.Verify(context.Background(), plan, time.Now())
	if err != nil || state != VerifyUnresolved || id != "" {
		t.Fatalf("Verify = %q/%q/%v, want unresolved", id, state, err)
	}
}

func TestOpenCodeTargetCompatibleHonoursOverride(t *testing.T) {
	t.Parallel()

	target := NewOpenCodeTarget(&OpenCodeTarget{ForceCompat: adapter.CompatibilitySupported})
	got, err := target.Compatible(context.Background())
	if err != nil || got != adapter.CompatibilitySupported {
		t.Fatalf("Compatible = %q, %v", got, err)
	}
	// An install whose version cannot be judged is uncertainty, not health.
	untested := NewOpenCodeTarget(&OpenCodeTarget{
		Inspect: func(context.Context) adapter.Compatibility { return adapter.CompatibilityUntested },
	})
	if got, err := untested.Compatible(context.Background()); err != nil || got != adapter.CompatibilityUntested {
		t.Fatalf("Compatible = %q, %v", got, err)
	}
}

// TestPipelineKeepsTheDestinationsOwnArgvShape is the regression for a
// hard-coded agent switch in the pipeline.
//
// The pipeline re-renders the briefing after the target has planned, and used
// to substitute it by replacing the whole argv with one bare element for every
// destination that was not Claude. OpenCode reads a bare positional as a
// project path, so that dropped the flag and planned a launch into a directory
// named after the entire briefing. The briefing must land where the target put
// its own bootstrap, whatever shape that is.
func TestPipelineKeepsTheDestinationsOwnArgvShape(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	rendered := []byte("rendered destination briefing")
	body := testOpenCodeCapsule(workspace, "goal", "intent")

	tests := []struct {
		name   string
		target HandoffTarget
		want   []string
	}{
		{
			name:   "opencode takes its prompt as an option value",
			target: NewOpenCodeTarget(nil),
			want:   []string{OpenCodeNewSessionFlag, string(rendered)},
		},
		{
			name:   "codex takes its prompt as a bare positional",
			target: NewCodexTarget(nil),
			want:   []string{string(rendered)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, _, err := planDestination(
				context.Background(), test.target, body, PolicyBalanced,
				Options{ReinstateHome: t.TempDir()}, rendered)
			if err != nil {
				t.Fatalf("planDestination: %v", err)
			}
			if strings.Join(plan.Args, "\x00") != strings.Join(test.want, "\x00") {
				t.Fatalf("argv = %q, want %q", plan.Args, test.want)
			}
			if string(plan.Bootstrap) != string(rendered) {
				t.Fatalf("Bootstrap = %q, want the rendered briefing", plan.Bootstrap)
			}
		})
	}
}

// TestOpenCodeArgvCarriesItsOwnBootstrap is what makes the pipeline's
// substitution safe for this destination.
//
// planDestination replaces the planned briefing with the rendered one by
// finding the argv element equal to plan.Bootstrap. OpenCode reads a bare
// positional as a project path, so a substitution that dropped the flag would
// plan a launch into a directory named after the entire briefing. That cannot
// happen while the planned argv carries the bootstrap as the flag's value,
// which is what this pins.
func TestOpenCodeArgvCarriesItsOwnBootstrap(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	target := NewOpenCodeTarget(&OpenCodeTarget{Root: t.TempDir()})
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Args) != 2 || plan.Args[0] != OpenCodeNewSessionFlag {
		t.Fatalf("argv = %q, want the new-session flag and its value", plan.Args)
	}
	if plan.Args[1] != string(plan.Bootstrap) {
		t.Fatal("the flag's value is not plan.Bootstrap; the pipeline substitution would not find it")
	}

	rewritten := rewriteBootstrapArgs(plan, []byte("rendered briefing"))
	if len(rewritten.Args) != 2 || rewritten.Args[0] != OpenCodeNewSessionFlag {
		t.Fatalf("substitution dropped the flag: %q", rewritten.Args)
	}
	if rewritten.Args[1] != "rendered briefing" {
		t.Fatalf("substitution did not replace the briefing: %q", rewritten.Args)
	}
}
