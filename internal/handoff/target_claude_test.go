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
	"unicode/utf16"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

type recordingLaunchRunner struct {
	last sessionindex.LaunchPlan
	err  error
}

func (r *recordingLaunchRunner) Run(_ context.Context, plan sessionindex.LaunchPlan) error {
	r.last = plan
	return r.err
}

func claudeTestCapsule(workspace string) capsule.Capsule {
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

func TestClaudeTargetRegistered(t *testing.T) {
	t.Parallel()

	got, ok := Target("claude")
	if !ok {
		t.Fatal("claude target not registered")
	}
	caps := got.Capabilities()
	if !caps.SupportsPinnedID || caps.Agent != "claude" || caps.MaxArgvBytes != DefaultMaxArgvBytes {
		t.Fatalf("capabilities = %+v", caps)
	}
}

func TestClaudeTargetPlanArgvExact(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	const pinned = "11111111-2222-4333-8444-555555555555"
	target := &ClaudeTarget{
		NewSessionID: func() (string, error) { return pinned, nil },
		SessionExists: func(context.Context, string) (bool, error) {
			return false, nil
		},
		Bootstrap: func(capsule.Capsule, Policy) ([]byte, error) {
			return []byte("bootstrap-body"), nil
		},
	}

	plan, _, err := target.Plan(claudeTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Executable != "claude" || plan.Dir != workspace || plan.SessionID != pinned {
		t.Fatalf("plan identity = %+v", plan)
	}
	wantArgs := []string{"--session-id", pinned, "bootstrap-body"}
	if len(plan.Args) != len(wantArgs) {
		t.Fatalf("args = %#v, want %#v", plan.Args, wantArgs)
	}
	for i := range wantArgs {
		if plan.Args[i] != wantArgs[i] {
			t.Fatalf("args = %#v, want %#v", plan.Args, wantArgs)
		}
	}
	if len(plan.Files) != 0 {
		t.Fatalf("Files = %#v, want empty (no vendor-internal writes)", plan.Files)
	}
}

func TestClaudeTargetPlanDeterministicAndRefusesCollision(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	c := claudeTestCapsule(workspace)
	target := &ClaudeTarget{
		SessionExists: func(context.Context, string) (bool, error) { return false, nil },
		Bootstrap: func(capsule.Capsule, Policy) ([]byte, error) {
			return []byte("ok"), nil
		},
	}

	first, _, err := target.Plan(c, PolicyCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := target.Plan(c, PolicyCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionID != second.SessionID || !reflect.DeepEqual(first.Args, second.Args) {
		t.Fatalf("separate plans differ: first=%+v second=%+v", first, second)
	}

	colliding := &ClaudeTarget{
		SessionExists: func(context.Context, string) (bool, error) {
			return true, nil
		},
		Bootstrap: func(capsule.Capsule, Policy) ([]byte, error) {
			return []byte("ok"), nil
		},
	}
	_, _, err = colliding.Plan(c, PolicyCheckpoint)
	if !errors.Is(err, ErrClaudeSessionIDCollision) {
		t.Fatalf("Plan error = %v, want %v", err, ErrClaudeSessionIDCollision)
	}
}

func TestClaudeTargetVerifyProjectKeyPath(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	config := t.TempDir()
	id := "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	key := claudeProjectKeyForTest(workspace)
	sessionPath := filepath.Join(config, "projects", key, id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}

	target := &ClaudeTarget{ConfigDir: config}
	plan := DestinationPlan{SessionID: id, Dir: workspace}

	_, state, err := target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify missing file: %v", err)
	}
	if state != VerifyUnresolved {
		t.Fatalf("state = %q, want %q", state, VerifyUnresolved)
	}

	if err := os.WriteFile(sessionPath, []byte(`{"type":"user"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gotID, state, err := target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify present: %v", err)
	}
	if state != VerifyResolved || gotID != id {
		t.Fatalf("Verify = (%q, %q), want (%q, %q)", gotID, state, id, VerifyResolved)
	}

	// Source-device key must not satisfy verification for this device's workspace.
	wrongKey := filepath.Join(config, "projects", "-Users-other-host-code-demo", id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(wrongKey), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrongKey, []byte(`{"type":"user"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(sessionPath)
	_, state, err = target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify wrong key: %v", err)
	}
	if state != VerifyUnresolved {
		t.Fatalf("wrong project key unexpectedly resolved: %q", state)
	}
}

func TestClaudeTargetLaunchUsesFakeRunnerArgv(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := &recordingLaunchRunner{}
	target := &ClaudeTarget{}
	plan := DestinationPlan{
		Agent:      "claude",
		Executable: "claude",
		Args:       []string{"--session-id", "ffff0000-0000-4000-8000-000000000001", "boot"},
		Dir:        workspace,
		SessionID:  "ffff0000-0000-4000-8000-000000000001",
		Bootstrap:  []byte("boot"),
	}
	if err := target.Launch(context.Background(), plan, runner); err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if runner.last.Executable != "claude" || runner.last.Dir != workspace {
		t.Fatalf("launch plan = %+v", runner.last)
	}
	if len(runner.last.Args) != 3 || runner.last.Args[0] != "--session-id" {
		t.Fatalf("launch args = %#v", runner.last.Args)
	}
}

func TestClaudeTargetLaunchNonTTYRefuses(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "")

	workspace := t.TempDir()
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	stdin, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close() }()
	defer func() { _ = w.Close() }()

	target := &ClaudeTarget{}
	plan := DestinationPlan{
		Executable: "claude",
		Args:       []string{"--session-id", "00000000-0000-4000-8000-000000000099", "boot"},
		Dir:        workspace,
		SessionID:  "00000000-0000-4000-8000-000000000099",
	}
	err = target.Launch(context.Background(), plan, sessionindex.ExecLaunchRunner{
		Stdin:      stdin,
		Executable: executable,
	})
	if !errors.Is(err, sessionindex.ErrNonInteractiveLaunch) {
		t.Fatalf("Launch error = %v, want %v", err, sessionindex.ErrNonInteractiveLaunch)
	}
}

func TestClaudeTargetMaterializeNoVendorWrites(t *testing.T) {
	t.Parallel()

	config := t.TempDir()
	target := &ClaudeTarget{ConfigDir: config}
	plan := DestinationPlan{
		Executable: "claude",
		Args:       []string{"--session-id", "00000000-0000-4000-8000-000000000010", "x"},
		Dir:        t.TempDir(),
		SessionID:  "00000000-0000-4000-8000-000000000010",
	}
	if err := target.Materialize(context.Background(), plan); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if entries, err := os.ReadDir(config); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Fatalf("ConfigDir mutated: %v", entries)
	}
}

func TestClaudeSessionIDIsDeterministicUUIDv4(t *testing.T) {
	t.Parallel()

	c := claudeTestCapsule(t.TempDir())
	id, err := claudeSessionID(c)
	if err != nil {
		t.Fatal(err)
	}
	again, err := claudeSessionID(c)
	if err != nil || again != id {
		t.Fatalf("deterministic ID = %q, %v; want %q", again, err, id)
	}
	parts := strings.Split(id, "-")
	if len(parts) != 5 || len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		t.Fatalf("id shape = %q", id)
	}
	if parts[2][0] != '4' {
		t.Fatalf("version nibble = %q, want 4", parts[2])
	}
	switch parts[3][0] {
	case '8', '9', 'a', 'b':
	default:
		t.Fatalf("variant nibble = %q", parts[3])
	}
}

func claudeProjectKeyForTest(projectPath string) string {
	if resolved, err := filepath.EvalSymlinks(projectPath); err == nil {
		projectPath = resolved
	}
	var directory strings.Builder
	for _, unit := range utf16.Encode([]rune(filepath.Clean(projectPath))) {
		if unit >= 'a' && unit <= 'z' || unit >= 'A' && unit <= 'Z' || unit >= '0' && unit <= '9' {
			directory.WriteByte(byte(unit))
		} else {
			directory.WriteByte('-')
		}
	}
	return directory.String()
}
