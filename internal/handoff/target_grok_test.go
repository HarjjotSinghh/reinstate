package handoff

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func grokTestCapsule(workspace string) capsule.Capsule {
	return capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			ID:        "aabbccddeeff00112233445566778899",
			SchemaVer: capsule.SchemaVersion,
		},
		Workspace: capsule.Workspace{
			ProjectID: "demo",
			Root:      "${REPO:demo}",
			Path:      workspace,
		},
	}
}

// grokFixtureRoot builds a synthetic Grok home. It never touches a real
// ~/.grok: every test in this file passes the returned root explicitly.
func grokFixtureRoot(t *testing.T, sessions map[string]map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for project, ids := range sessions {
		for id, cwd := range ids {
			dir := filepath.Join(root, "sessions", project, id)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if cwd == "" {
				continue
			}
			body := `{"info":{"id":"` + id + `","cwd":"` + cwd + `"},"chat_format_version":1}`
			if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

func TestGrokTargetRegistered(t *testing.T) {
	t.Parallel()

	got, ok := Target("grok")
	if !ok {
		t.Fatal("grok target not registered")
	}
	caps := got.Capabilities()
	// --session-id names the new conversation's UUID, so the destination
	// session id is known before launch rather than reconciled afterwards.
	if !caps.SupportsPinnedID || caps.Agent != "grok" || caps.MaxArgvBytes != DefaultMaxArgvBytes {
		t.Fatalf("capabilities = %+v", caps)
	}
	if caps.AttachmentSupport {
		t.Fatal("no vendor-published Grok attachment contract exists; it must not be claimed")
	}
}

// TestGrokTargetPlanArgvExact pins the exact destination argv. This starts a
// NEW Grok session; it is never a cross-agent resume.
func TestGrokTargetPlanArgvExact(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := &GrokTarget{Root: grokFixtureRoot(t, nil)}
	plan, _, err := target.Plan(grokTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable != "grok" {
		t.Fatalf("executable = %q", plan.Executable)
	}
	if len(plan.Args) != 3 || plan.Args[0] != "--session-id" {
		t.Fatalf("args = %v", plan.Args)
	}
	if plan.Args[1] != plan.SessionID {
		t.Fatalf("argv session id %q != plan session id %q", plan.Args[1], plan.SessionID)
	}
	if !sessionindex.IsGrokSessionID(plan.SessionID) {
		t.Fatalf("session id %q is not a UUID; grok --session-id requires one", plan.SessionID)
	}
	if plan.Dir != workspace {
		t.Fatalf("dir = %q, want %q", plan.Dir, workspace)
	}
	if len(plan.Files) != 0 {
		t.Fatalf("Grok destination planned vendor-internal files: %v", plan.Files)
	}
	if !strings.Contains(string(plan.Bootstrap), "not native resume") {
		t.Fatalf("bootstrap does not state that this is a new session: %q", plan.Bootstrap)
	}
}

// TestGrokTargetPlanIsDeterministic keeps dry-run and execute planning on the
// exact same argv, which is what makes the dry-run output trustworthy.
func TestGrokTargetPlanIsDeterministic(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := &GrokTarget{Root: grokFixtureRoot(t, nil)}
	first, _, err := target.Plan(grokTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := target.Plan(grokTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Args, second.Args) || first.SessionID != second.SessionID {
		t.Fatalf("plan is not deterministic:\n%v\n%v", first.Args, second.Args)
	}
}

// TestGrokSessionIDDiffersFromClaude keeps one capsule from deriving the same
// UUID in two vendors' stores.
func TestGrokSessionIDDiffersFromClaude(t *testing.T) {
	t.Parallel()

	c := grokTestCapsule(t.TempDir())
	grokID, err := grokSessionIDFor(c)
	if err != nil {
		t.Fatal(err)
	}
	claudeID, err := claudeSessionID(c)
	if err != nil {
		t.Fatal(err)
	}
	if grokID == claudeID {
		t.Fatalf("Grok and Claude derived the same session id %q from one capsule", grokID)
	}
}

// TestGrokTargetRefusesExistingSessionID is the vendor precondition:
// `grok --session-id <uuid>` requires that the UUID "must not already exist
// under the target session directory".
func TestGrokTargetRefusesExistingSessionID(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	// Derive the id the target will choose, then plant it.
	planned, err := grokSessionIDFor(grokTestCapsule(workspace))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		project string
		cwd     string
	}{
		{"same project directory", "%2Fsome%2Fplace", workspace},
		// The vendor scopes the precondition to the target session directory,
		// but a UUID collision anywhere in the store is refused: Reinstate
		// cannot make room in a store it never writes to.
		{"another project directory", "%2Funrelated%2Fproject", "/unrelated/project"},
		{"no summary.json", "%2Fopaque", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := grokFixtureRoot(t, map[string]map[string]string{
				test.project: {planned: test.cwd},
			})
			target := &GrokTarget{Root: root}
			if _, _, err := target.Plan(grokTestCapsule(workspace), PolicyBalanced); !errors.Is(err, ErrGrokSessionIDCollision) {
				t.Fatalf("Plan error = %v, want ErrGrokSessionIDCollision", err)
			}
			// Materialize re-checks immediately before launch, because Plan and
			// Launch are separated by capsule storage and operator confirmation.
			plan := DestinationPlan{
				Agent: "grok", Executable: "grok",
				Args:      []string{"--session-id", planned, "briefing"},
				Dir:       workspace,
				SessionID: planned,
			}
			if err := target.Materialize(context.Background(), plan); !errors.Is(err, ErrGrokSessionIDCollision) {
				t.Fatalf("Materialize error = %v, want ErrGrokSessionIDCollision", err)
			}
		})
	}
}

// TestGrokTargetMaterializeWritesNothing pins ADR 0003 for this destination:
// no planned files, and no directory-trust record invented for a vendor whose
// trust file shape has never been measured.
func TestGrokTargetMaterializeWritesNothing(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	root := grokFixtureRoot(t, nil)
	target := &GrokTarget{Root: root}
	plan, _, err := target.Plan(grokTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	before := treeSnapshot(t, root)
	if err := target.Materialize(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	after := treeSnapshot(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Materialize changed the Grok root:\nbefore %v\nafter  %v", before, after)
	}
}

func TestGrokTargetLaunchUsesPlannedArgv(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	target := &GrokTarget{Root: grokFixtureRoot(t, nil)}
	plan, _, err := target.Plan(grokTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingLaunchRunner{}
	if err := target.Launch(context.Background(), plan, runner); err != nil {
		t.Fatal(err)
	}
	if runner.last.Executable != "grok" || !reflect.DeepEqual(runner.last.Args, plan.Args) {
		t.Fatalf("launched %s %v, want grok %v", runner.last.Executable, runner.last.Args, plan.Args)
	}
	if runner.last.Operation != sessionindex.OperationHandoff {
		t.Fatalf("operation = %q, want handoff", runner.last.Operation)
	}
	if runner.last.Dir != workspace {
		t.Fatalf("cwd = %q, want %q", runner.last.Dir, workspace)
	}
	if err := target.Launch(context.Background(), plan, nil); err == nil {
		t.Fatal("Launch without a runner succeeded")
	}
}

func TestGrokTargetVerify(t *testing.T) {
	t.Parallel()

	const id = "01987654-3210-7890-abcd-ef0123456789"
	workspace := "/Users/fixture-user/code/demo"
	tests := []struct {
		name      string
		sessions  map[string]map[string]string
		sessionID string
		wantID    string
		wantState string
	}{
		{
			name:      "resolved with matching cwd",
			sessions:  map[string]map[string]string{"%2FUsers%2Ffixture-user%2Fcode%2Fdemo": {id: workspace}},
			sessionID: id, wantID: id, wantState: VerifyResolved,
		},
		{
			name:      "resolved when the vendor recorded no cwd",
			sessions:  map[string]map[string]string{"%2Fopaque": {id: ""}},
			sessionID: id, wantID: id, wantState: VerifyResolved,
		},
		{
			name:      "unresolved when the session never appeared",
			sessions:  nil,
			sessionID: id, wantState: VerifyUnresolved,
		},
		{
			name:      "unresolved when the recorded cwd is a different workspace",
			sessions:  map[string]map[string]string{"%2Felsewhere": {id: "/elsewhere"}},
			sessionID: id, wantState: VerifyUnresolved,
		},
		{
			name: "ambiguous when one uuid appears under two projects",
			sessions: map[string]map[string]string{
				"%2FUsers%2Ffixture-user%2Fcode%2Fdemo": {id: workspace},
				"%2Fopaque":                             {id: ""},
			},
			sessionID: id, wantState: VerifyAmbiguous,
		},
		{
			name:      "unresolved without a planned session id",
			sessions:  map[string]map[string]string{"%2Fopaque": {id: ""}},
			sessionID: "", wantState: VerifyUnresolved,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			target := &GrokTarget{Root: grokFixtureRoot(t, test.sessions)}
			plan := DestinationPlan{Agent: "grok", Dir: workspace, SessionID: test.sessionID}
			gotID, gotState, err := target.Verify(context.Background(), plan, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if gotID != test.wantID || gotState != test.wantState {
				t.Fatalf("Verify() = %q, %q, want %q, %q", gotID, gotState, test.wantID, test.wantState)
			}
		})
	}
}

// TestGrokTargetCompatibleNeverSpawnsWithExplicitRoot keeps the unit suite from
// executing the operator's real grok binary.
func TestGrokTargetCompatibleNeverSpawnsWithExplicitRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sessions map[string]map[string]string
		force    adapter.Compatibility
		want     adapter.Compatibility
	}{
		{
			name:     "prepared root with sessions",
			sessions: map[string]map[string]string{"%2Fdemo": {"01987654-3210-7890-abcd-ef0123456789": "/demo"}},
			want:     adapter.CompatibilitySupported,
		},
		{
			name:     "prepared root not used yet",
			sessions: nil,
			want:     adapter.CompatibilitySupported,
		},
		{
			name:  "forced untested",
			force: adapter.CompatibilityUntested,
			want:  adapter.CompatibilityUntested,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			target := &GrokTarget{Root: grokFixtureRoot(t, test.sessions), ForceCompat: test.force}
			got, err := target.Compatible(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("Compatible() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGrokTargetPlanRequiresWorkspace(t *testing.T) {
	t.Parallel()

	target := &GrokTarget{Root: grokFixtureRoot(t, nil)}
	if _, _, err := target.Plan(grokTestCapsule(""), PolicyBalanced); err == nil {
		t.Fatal("Plan succeeded without a verified workspace path")
	}
}

// TestGrokTargetRejectsNonUUIDSessionID guards the generator: `--session-id`
// requires a valid UUID, and a value of another shape must never be handed to
// the vendor.
func TestGrokTargetRejectsNonUUIDSessionID(t *testing.T) {
	t.Parallel()

	target := &GrokTarget{
		Root:         grokFixtureRoot(t, nil),
		NewSessionID: func() (string, error) { return "not-a-uuid", nil },
	}
	_, _, err := target.Plan(grokTestCapsule(t.TempDir()), PolicyBalanced)
	if err == nil || !strings.Contains(err.Error(), "not a UUID") {
		t.Fatalf("Plan error = %v, want a non-UUID refusal", err)
	}
}

func TestRefuseNoRedactDestination(t *testing.T) {
	t.Parallel()

	if err := refuseNoRedactDestination("grok"); err == nil {
		t.Fatal("--no-redact into Grok was allowed")
	}
	for _, agent := range []string{"claude", "codex", ""} {
		if err := refuseNoRedactDestination(agent); err != nil {
			t.Fatalf("refuseNoRedactDestination(%q) = %v, want nil", agent, err)
		}
	}
}

func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		paths = append(paths, rel+":"+info.Mode().String())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}
