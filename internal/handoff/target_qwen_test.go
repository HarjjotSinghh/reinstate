package handoff

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func qwenTestCapsule(workspace string) capsule.Capsule {
	return capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			ID:        "00112233445566778899aabbccddeeff",
			SchemaVer: capsule.SchemaVersion,
		},
		Workspace: capsule.Workspace{
			ProjectID: "demo",
			Root:      "${REPO:demo}",
			Path:      workspace,
		},
	}
}

func TestQwenTargetRegistered(t *testing.T) {
	t.Parallel()
	got, ok := Target(sessionindex.AgentQwen)
	if !ok {
		t.Fatal("qwen target is not registered")
	}
	caps := got.Capabilities()
	if caps.Agent != sessionindex.AgentQwen {
		t.Fatalf("capabilities = %+v", caps)
	}
	if !caps.SupportsPinnedID {
		t.Fatal("qwen --session-id pins the destination id; the capability must say so")
	}
	if !caps.SupportsInitialPrompt {
		t.Fatal("qwen --prompt-interactive seeds the bootstrap; the capability must say so")
	}
	if caps.MaxArgvBytes != DefaultMaxArgvBytes {
		t.Fatalf("argv budget = %d", caps.MaxArgvBytes)
	}
	if caps.AttachmentSupport {
		t.Fatal("the capsule never re-embeds attachments, so the destination must not claim support")
	}
}

// TestQwenTargetPlanStartsANewSession is the T4 contract in one assertion: the
// argv creates a new session at a pinned id and seeds it with a briefing. It
// must never contain --resume, which would continue an existing Qwen thread.
func TestQwenTargetPlanStartsANewSession(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	target := &QwenTarget{Root: t.TempDir(), ForceCompat: adapter.CompatibilitySupported}

	plan, fidelity, err := target.Plan(qwenTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Executable != "qwen" {
		t.Fatalf("executable = %q", plan.Executable)
	}
	if len(plan.Args) != 4 || plan.Args[0] != "--session-id" || plan.Args[2] != "--prompt-interactive" {
		t.Fatalf("args = %v", plan.Args)
	}
	if plan.Args[1] != plan.SessionID {
		t.Fatalf("argv id %q does not match plan id %q", plan.Args[1], plan.SessionID)
	}
	if plan.Args[3] != string(plan.Bootstrap) {
		t.Fatal("the seeded prompt is not the planned bootstrap")
	}
	for _, arg := range plan.Args {
		if arg == "--resume" || arg == "--continue" || arg == "--fork-session" {
			t.Fatalf("a handoff starts a new session; args = %v", plan.Args)
		}
	}
	if plan.Dir != workspace {
		t.Fatalf("cwd = %q, want the verified workspace", plan.Dir)
	}
	if len(plan.Files) != 0 {
		t.Fatalf("a destination plan must not write vendor-internal files: %v", plan.Files)
	}
	if fidelity.Mode != capsule.FidelityModeStructuredHandoff {
		t.Fatalf("fidelity mode = %q", fidelity.Mode)
	}
}

func TestQwenTargetPlanIsDeterministicAndVendorAcceptable(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	target := &QwenTarget{Root: t.TempDir()}
	c := qwenTestCapsule(workspace)

	first, _, err := target.Plan(c, PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := target.Plan(c, PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Args, second.Args) {
		t.Fatalf("dry-run and execute would launch different argv: %v vs %v", first.Args, second.Args)
	}
	// `qwen --session-id not-a-uuid` is a usage error, so anything the plan
	// emits must satisfy the vendor's own pattern.
	if !qwenSessionIDPattern.MatchString(first.SessionID) {
		t.Fatalf("session id %q is not a lowercase UUID the vendor accepts", first.SessionID)
	}
}

// TestQwenSessionIDDoesNotCollideWithClaudeForOneCapsule guards the salt. One
// capsule handed to two destinations must not derive one shared session id.
func TestQwenSessionIDDoesNotCollideWithClaudeForOneCapsule(t *testing.T) {
	t.Parallel()
	c := qwenTestCapsule(t.TempDir())
	qwenID, err := qwenSessionID(c)
	if err != nil {
		t.Fatal(err)
	}
	claudeID, err := claudeSessionID(c)
	if err != nil {
		t.Fatal(err)
	}
	if qwenID == claudeID {
		t.Fatalf("both destinations derived %q from one capsule", qwenID)
	}
}

func TestQwenTargetPlanRefusesAnIndexedCollision(t *testing.T) {
	t.Parallel()
	target := &QwenTarget{
		Root: t.TempDir(),
		SessionExists: func(context.Context, string) (bool, error) {
			return true, nil
		},
	}
	_, _, err := target.Plan(qwenTestCapsule(t.TempDir()), PolicyBalanced)
	if !errors.Is(err, ErrQwenSessionIDCollision) {
		t.Fatalf("Plan() error = %v, want ErrQwenSessionIDCollision", err)
	}
}

func TestQwenTargetPlanRefusesANonUUIDSessionID(t *testing.T) {
	t.Parallel()
	target := &QwenTarget{
		Root:         t.TempDir(),
		NewSessionID: func() (string, error) { return "not-a-uuid", nil },
	}
	_, _, err := target.Plan(qwenTestCapsule(t.TempDir()), PolicyBalanced)
	if err == nil || !strings.Contains(err.Error(), "not a lowercase UUID") {
		t.Fatalf("Plan() error = %v, want a plan-time UUID refusal", err)
	}
}

func TestQwenTargetPlanRequiresAWorkspace(t *testing.T) {
	t.Parallel()
	c := qwenTestCapsule("")
	if _, _, err := (&QwenTarget{}).Plan(c, PolicyBalanced); err == nil {
		t.Fatal("Plan() accepted a capsule with no verified workspace path")
	}
}

func TestQwenTargetPlanFailsClosedOnAnOversizedBootstrap(t *testing.T) {
	t.Parallel()
	target := &QwenTarget{
		Root:         t.TempDir(),
		MaxArgvBytes: 128,
		Bootstrap: func(capsule.Capsule, Policy) ([]byte, error) {
			return []byte(strings.Repeat("x", 4096)), nil
		},
	}
	_, _, err := target.Plan(qwenTestCapsule(t.TempDir()), PolicyBalanced)
	if !errors.Is(err, ErrArgvExceedsBudget) {
		t.Fatalf("Plan() error = %v, want ErrArgvExceedsBudget", err)
	}
}

func TestQwenTargetLaunchUsesThePlannedArgv(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	target := &QwenTarget{Root: t.TempDir()}
	plan, _, err := target.Plan(qwenTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingLaunchRunner{}
	if err := target.Launch(context.Background(), plan, runner); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if runner.last.Agent != sessionindex.AgentQwen {
		t.Fatalf("launched agent = %q", runner.last.Agent)
	}
	if runner.last.Operation != sessionindex.OperationHandoff {
		t.Fatalf("operation = %q, want handoff", runner.last.Operation)
	}
	if !reflect.DeepEqual(runner.last.Args, plan.Args) {
		t.Fatalf("launched argv = %v, planned %v", runner.last.Args, plan.Args)
	}
	if runner.last.Dir != workspace {
		t.Fatalf("launch cwd = %q", runner.last.Dir)
	}
}

func TestQwenTargetLaunchRefusesAnIncompletePlan(t *testing.T) {
	t.Parallel()
	target := &QwenTarget{Root: t.TempDir()}
	runner := &recordingLaunchRunner{}
	tests := []struct {
		name string
		plan DestinationPlan
	}{
		{"no executable", DestinationPlan{Args: []string{"--session-id", "x"}, Dir: t.TempDir()}},
		{"no args", DestinationPlan{Executable: "qwen", Dir: t.TempDir()}},
		{"no workspace", DestinationPlan{Executable: "qwen", Args: []string{"--session-id", "x"}}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := target.Launch(context.Background(), tt.plan, runner); err == nil {
				t.Fatal("Launch() accepted an incomplete plan")
			}
		})
	}
}

// TestQwenTargetVerifyResolvesThePinnedID exercises the whole point of a pinned
// id: the destination session is found, not guessed.
func TestQwenTargetVerifyResolvesThePinnedID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := t.TempDir()
	target := &QwenTarget{Root: root}
	plan, _, err := target.Plan(qwenTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}

	id, state, err := target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if state != VerifyUnresolved || id != "" {
		t.Fatalf("before launch: id=%q state=%q, want unresolved", id, state)
	}

	dir := filepath.Join(root, "projects", QwenProjectKey(workspace), "chats")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, plan.SessionID+".jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	id, state, err = target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if state != VerifyResolved || id != plan.SessionID {
		t.Fatalf("after launch: id=%q state=%q, want the pinned id resolved", id, state)
	}
}

// TestQwenTargetVerifyRecomputesTheProjectBucket is the multi-device rule: a
// source device's directory name is never reused, because the vendor lower-cases
// the path before sanitising it on Windows and only on Windows.
func TestQwenTargetVerifyRecomputesTheProjectBucket(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := t.TempDir()
	target := &QwenTarget{Root: root}
	plan, _, err := target.Plan(qwenTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// The session exists, but under a bucket derived from some other device's
	// path. Verify must not find it.
	dir := filepath.Join(root, "projects", "-Users-someone-else-code-demo", "chats")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, plan.SessionID+".jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id, state, err := target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if state != VerifyUnresolved || id != "" {
		t.Fatalf("id=%q state=%q, want unresolved for a foreign project bucket", id, state)
	}
}

// TestQwenTargetVerifyFailsClosedWithoutARoot keeps unit tests off a real
// ~/.qwen: with no explicit root and no QWEN_HOME, Verify reports unresolved
// rather than resolving a home directory.
func TestQwenTargetVerifyFailsClosedWithoutARoot(t *testing.T) {
	t.Setenv("QWEN_HOME", "")
	target := &QwenTarget{}
	plan := DestinationPlan{SessionID: "11111111-2222-4333-8444-555555555555", Dir: t.TempDir()}
	id, state, err := target.Verify(context.Background(), plan, time.Now())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if state != VerifyUnresolved || id != "" {
		t.Fatalf("id=%q state=%q, want unresolved with no configured root", id, state)
	}
}

func TestQwenProjectKeyMatchesTheVendorRule(t *testing.T) {
	t.Parallel()
	// The key lowercases on Windows, because Windows paths are case-insensitive
	// and the vendor's own bucket is lowercased there. A workspace is always a
	// path on the running host, so a POSIX path never reaches this function on
	// Windows — but a table that hard-codes the POSIX answer asserts the host's
	// behaviour rather than the rule, and fails on Windows CI for a case that
	// cannot occur.
	//
	// The expectation therefore applies the same host rule the function
	// documents. What is pinned on every platform is the sanitisation itself:
	// every byte outside [A-Za-z0-9] becomes '-', and a trailing separator is
	// cleaned away first.
	expect := func(posix string) string {
		if runtime.GOOS == "windows" {
			return strings.ToLower(posix)
		}
		return posix
	}
	tests := []struct {
		name, path, want string
	}{
		{"posix path", "/Users/fixture-user/code/demo", expect("-Users-fixture-user-code-demo")},
		{"trailing slash is cleaned", "/Users/fixture-user/code/demo/", expect("-Users-fixture-user-code-demo")},
		{"dots and digits", "/tmp/a.b/c9", expect("-tmp-a-b-c9")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := QwenProjectKey(tt.path); got != tt.want {
				t.Fatalf("QwenProjectKey(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestQwenProjectKeyLowercasesOnlyOnWindows pins the host rule itself, so the
// table above cannot quietly agree with a function that stopped applying it.
func TestQwenProjectKeyLowercasesOnlyOnWindows(t *testing.T) {
	t.Parallel()
	got := QwenProjectKey("/Users/Fixture/Code")
	if runtime.GOOS == "windows" {
		if got != strings.ToLower(got) {
			t.Fatalf("QwenProjectKey = %q; Windows buckets are lowercased", got)
		}
		return
	}
	if got != "-Users-Fixture-Code" {
		t.Fatalf("QwenProjectKey = %q; case is preserved off Windows", got)
	}
}

func TestQwenTargetCompatibilityIsInjectable(t *testing.T) {
	t.Parallel()
	for _, want := range []adapter.Compatibility{
		adapter.CompatibilitySupported,
		adapter.CompatibilityUntested,
		adapter.CompatibilityNotInstalled,
	} {
		target := &QwenTarget{ForceCompat: want}
		got, err := target.Compatible(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("Compatible() = %q, want %q", got, want)
		}
	}
}

// TestQwenTargetMaterializeWritesNothingUnderTheQwenHome is the ADR 0003 line:
// a destination never writes into another vendor's private store.
func TestQwenTargetMaterializeWritesNothingUnderTheQwenHome(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := t.TempDir()
	target := &QwenTarget{Root: root}
	plan, _, err := target.Plan(qwenTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Materialize(context.Background(), plan); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("Materialize wrote into the Qwen home: %v", names)
	}
}

// TestQwenPlanDoesNotRefuseAMultiLineBriefing pins the fix for a defect that
// only native Windows could show.
//
// Windows CreateProcess truncates an argv element at an embedded CR/LF, which
// is real: rc.9 caught Codex receiving only the first line of its briefing.
// planDestination already handles it for every destination by falling back to
// the short file-backed projection.
//
// QwenTarget.Plan used to refuse outright instead, and planDestination returns
// on a Plan error before that fallback can run. A briefing is multi-line by
// construction, so every Qwen handoff failed on native Windows — `handoff:
// Qwen bootstrap is not safe to pass as argv on this platform` — while the same
// code passed on macOS, where the guard is inert.
func TestQwenPlanDoesNotRefuseAMultiLineBriefing(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	target := &QwenTarget{
		Root:        t.TempDir(),
		ForceCompat: adapter.CompatibilitySupported,
		Bootstrap: func(capsule.Capsule, Policy) ([]byte, error) {
			return []byte("Reinstate structured handoff - not native resume.\nRead the briefing at <path>.\n"), nil
		},
	}
	plan, _, err := target.Plan(qwenTestCapsule(workspace), PolicyBalanced)
	if err != nil {
		t.Fatalf("Plan refused a multi-line briefing: %v", err)
	}
	if plan.SessionID == "" {
		t.Fatal("Plan produced no pinned session id")
	}
	// Plan may still carry the newlines; the fallback that removes them lives in
	// planDestination. What must not happen is a refusal before it can run.
	if plan.Executable != qwenExecutable {
		t.Fatalf("executable = %q, want %q", plan.Executable, qwenExecutable)
	}
}
