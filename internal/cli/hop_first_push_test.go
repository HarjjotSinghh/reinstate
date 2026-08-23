package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// firstPushBudget is the product promise for the hosted journey: from
// `rein login` to the first successful push in under two minutes. The
// in-process journey has no network, so it is well inside the budget; the
// assertion exists so a regression that makes the path wait (a poll loop,
// a retry storm, a second provisioning round) fails loudly here.
const firstPushBudget = 2 * time.Minute

// hopDevice is one device in the hosted journey: its own Reinstate home,
// device-token store and secret store, talking to the shared fake control
// plane and locker. A wiped device is modelled by a fresh hopDevice.
type hopDevice struct {
	t         *testing.T
	plane     *fakeControlPlane
	name      string
	home      string
	tokens    *credentials.MemoryDeviceTokenStore
	secrets   credentials.SecretStore
	shownCode string
	// typedCode answers the hidden "Recovery code:" prompt of account recover.
	typedCode string
	// verifier replaces the verified-resume environment inspection; nil
	// uses the always-ready fake, the staging acceptance sets the real one.
	verifier preflight.Verifier
	// loginSleep replaces the wait between login polls; nil never waits
	// (the fake control plane approves on the first poll), the staging
	// acceptance waits for a real approval.
	loginSleep func(context.Context, time.Duration) error
	// ctx bounds one command (a real login waiting for approval); nil
	// means no deadline.
	ctx context.Context
}

func newHopDevice(t *testing.T, plane *fakeControlPlane, name string) *hopDevice {
	return &hopDevice{t: t, plane: plane, name: name, home: t.TempDir(), tokens: &credentials.MemoryDeviceTokenStore{}, secrets: credentials.NewMemorySecrets()}
}

func (d *hopDevice) run(args ...string) (stdout, stderr string, code int) {
	d.t.Helper()
	d.t.Setenv("REINSTATE_HOME", d.home)
	var out, errb bytes.Buffer
	verifier := d.verifier
	if verifier == nil {
		// The environment verifier is the one seam replaced: the in-process
		// journey runs where no vendor binary is installed, and a missing
		// executable is a block by design. The staging acceptance uses the
		// real verifier (see hop_first_push_acceptance_test.go).
		verifier = readyPreflightVerifier{}
	}
	code = Execute(Options{
		Name: "rein", Stdout: &out, Stderr: &errb, Args: args, Context: d.ctx,
		AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
		DeviceTokenStore:    d.tokens,
		DeviceSecrets:       d.secrets,
		PreflightVerifier:   verifier,
		OpenBrowser: func(u string) error {
			resp, err := http.Get(u)
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		},
		LoginPollSleep: d.loginSleepFunc(),
		DeviceName:     d.name,
		RecoveryCodePrompt: func(prompt string) ([]byte, error) {
			if strings.HasPrefix(prompt, "Recovery code") && d.typedCode != "" {
				return []byte(d.typedCode), nil
			}
			if !strings.Contains(prompt, "Re-enter") {
				return nil, errors.New("unexpected prompt " + prompt)
			}
			d.shownCode = recoveryCodePattern.FindString(errb.String())
			if d.shownCode == "" {
				return nil, errors.New("recovery code was not shown before the confirmation prompt")
			}
			return []byte(d.shownCode), nil
		},
	})
	return out.String(), errb.String(), code
}

func (d *hopDevice) loginSleepFunc() func(context.Context, time.Duration) error {
	if d.loginSleep != nil {
		return d.loginSleep
	}
	return func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
}

func (d *hopDevice) mustRun(step string, args ...string) string {
	d.t.Helper()
	out, errb, code := d.run(args...)
	if code != ExitOK {
		d.t.Fatalf("%s: exit=%d out=%q err=%q", step, code, out, errb)
	}
	return out
}

// journeyHome is the user's home directory for the journey: the three
// agents' session stores live under it and are wiped together.
type journeyHome struct {
	root    string
	project string
	claude  string // session file
	codex   string // rollout file
	db      string // OpenCode store
}

// plantJourneyHome points HOME at root and plants one session per agent:
// Claude Code (projects/<slug>/*.jsonl), Codex (.codex/sessions) and
// OpenCode (the embedded SQLite store under XDG_DATA_HOME).
func plantJourneyHome(t *testing.T, root string) journeyHome {
	t.Helper()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("CODEX_HOME", filepath.Join(root, ".codex"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdgdata"))
	h := journeyHome{root: root, project: filepath.Join(root, "Projects", "first-push")}
	if err := os.MkdirAll(h.project, 0o700); err != nil {
		t.Fatal(err)
	}

	claudeRoot := filepath.Join(root, ".claude", "projects", claudeProjectDirectoryForTest(h.project))
	if err := os.MkdirAll(claudeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "version"), []byte("2.1.219\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": h.project})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"synthetic first push claude"}}`+"\n")...)
	h.claude = filepath.Join(claudeRoot, "session-first-push.jsonl")
	if err := os.WriteFile(h.claude, content, 0o600); err != nil {
		t.Fatal(err)
	}

	// Codex syncs only sessions whose cwd is a configured project, so the
	// rollout (shaped like testdata/adapters/codex) points at the project.
	codexSessions := filepath.Join(root, ".codex", "sessions")
	if err := os.MkdirAll(codexSessions, 0o700); err != nil {
		t.Fatal(err)
	}
	rolloutMeta, _ := json.Marshal(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "rollout-first-push", "cwd": h.project}})
	rollout := append(rolloutMeta, '\n')
	rollout = append(rollout, []byte(`{"type":"message","role":"user","content":"synthetic first push codex"}`+"\n")...)
	h.codex = filepath.Join(codexSessions, "rollout-first-push.jsonl")
	if err := os.WriteFile(h.codex, rollout, 0o600); err != nil {
		t.Fatal(err)
	}

	storeDir := filepath.Join(root, "xdgdata", "opencode")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	h.db = filepath.Join(storeDir, "opencode.db")
	hydrateOpenCodeStore(t, h.db, filepath.Join("..", "..", "testdata", "adapters", "opencode", "macos", "store.sql"))
	return h
}

// wipe removes everything under the user's home, the way a reinstalled
// machine or a new laptop has nothing.
func (h journeyHome) wipe(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(h.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(h.root, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{h.claude, h.codex, h.db} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s survived the wipe: %v", p, err)
		}
	}
	if err := os.MkdirAll(h.project, 0o700); err != nil {
		t.Fatal(err)
	}
}

// reinstallAgents is the person installing and running Claude Code and
// Codex once on the wiped device so their layout roots exist. Reinstate
// restores into a vendor's layout and never invents it.
func (h journeyHome) reinstallAgents(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(h.root, ".claude", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.root, ".claude", "version"), []byte("2.1.219\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(h.root, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
}

// reinstallOpenCode gives OpenCode its store back: the vendor schema with no
// sessions, which is what OpenCode leaves after its first start.
func (h journeyHome) reinstallOpenCode(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(h.db), 0o700); err != nil {
		t.Fatal(err)
	}
	hydrateOpenCodeSchema(t, h.db, filepath.Join("..", "..", "testdata", "adapters", "opencode", "macos", "store.sql"))
}

// hydrateOpenCodeSchema creates the vendor schema (and its migration marker)
// without any of the seed's session rows.
func hydrateOpenCodeSchema(t *testing.T, dbPath, sqlPath string) {
	t.Helper()
	body, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, stmt := range strings.Split(string(body), ";") {
		var lines []string
		for _, line := range strings.Split(stmt, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				lines = append(lines, line)
			}
		}
		s := strings.TrimSpace(strings.Join(lines, "\n"))
		if s == "" || !(strings.HasPrefix(s, "CREATE") || strings.HasPrefix(s, "INSERT INTO migration")) {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("hydrate schema: %v", err)
		}
	}
}

// TestHopFirstPushJourney is the end-to-end hosted journey on one device,
// driven through the real CLI entrypoint against the in-process fake control
// plane and fake locker:
//
//	rein login → rein init --hop → rein account init → rein push --all
//	→ (the device is wiped) → rein login → rein init --hop
//	→ rein account recover → rein pull --all → verified resume per agent.
//
// It asserts the two things the product promises: the first_push event
// reaches the control plane exactly once, and sign-in to the first
// successful push takes under two minutes.
func TestHopFirstPushJourney(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000000test")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR"} {
		t.Setenv(env, "")
	}
	t.Setenv("TZ", "Asia/Kolkata")
	home := plantJourneyHome(t, t.TempDir())

	// --- day one: sign in, enrol, push ---
	laptop := newHopDevice(t, plane, "laptop")
	start := time.Now()
	laptop.mustRun("login", "login")
	laptop.mustRun("init --hop", "init", "--hop", "--project", "local/first-push="+home.project)
	laptop.mustRun("account init", "account", "init")
	recoveryCode := laptop.shownCode
	if recoveryCode == "" {
		t.Fatal("account init showed no recovery code")
	}

	pushOut := laptop.mustRun("push --all", "push", "--all", "--json")
	elapsed := time.Since(start)
	var pushed struct {
		Snapshots []string `json:"snapshots"`
	}
	if err := json.Unmarshal([]byte(pushOut), &pushed); err != nil {
		t.Fatalf("push output %q: %v", pushOut, err)
	}
	if len(pushed.Snapshots) != 3 {
		t.Fatalf("first push uploaded %d snapshots, want one per agent: %q", len(pushed.Snapshots), pushOut)
	}
	t.Logf("sign-in to first successful push: %s (budget %s)", elapsed.Round(time.Millisecond), firstPushBudget)
	if elapsed > firstPushBudget {
		t.Fatalf("sign-in to first push took %s, over the %s budget", elapsed, firstPushBudget)
	}
	if plane.firstPushes != 1 {
		t.Fatalf("first_push reached the control plane %d times after the first push, want 1", plane.firstPushes)
	}
	if plane.provisions != 1 {
		t.Fatalf("locker provisioned %d times, want 1", plane.provisions)
	}

	// The manifest names all three agents, and nothing in the locker is
	// readable without the root key.
	out := laptop.mustRun("status", "status", "--json")
	for _, ref := range []string{"claude:session-first-push", "codex:rollout-first-push", "opencode:ses_fixture001"} {
		if !strings.Contains(out, ref) {
			t.Fatalf("remote status missing %s: %q", ref, out)
		}
	}
	objects, _ := plane.s3.Store.List(context.Background(), "")
	for _, o := range objects {
		rc, _, err := plane.s3.Store.Get(context.Background(), o.Key)
		if err != nil {
			t.Fatal(err)
		}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(rc)
		rc.Close()
		for _, plaintext := range []string{"synthetic first push claude", "synthetic first push codex", "Synthetic OpenCode fixture request", recoveryCode} {
			if strings.Contains(body.String(), plaintext) {
				t.Fatalf("plaintext %q in the locker at %s", plaintext, o.Key)
			}
		}
	}

	// A second push has nothing to send and must not report again.
	if out := laptop.mustRun("second push", "push", "--all"); !strings.Contains(out, "skipped 3 unchanged") {
		t.Fatalf("second push output %q", out)
	}
	if plane.firstPushes != 1 {
		t.Fatalf("first_push reported %d times after a no-op push", plane.firstPushes)
	}

	// --- the device is wiped: sessions, Reinstate home, device key, token ---
	home.wipe(t)
	fresh := newHopDevice(t, plane, "laptop")

	// Recovery is the only way back in: a fresh device with the old
	// account's token is not enrolled in the keyring.
	fresh.mustRun("login again", "login")
	fresh.mustRun("init --hop again", "init", "--hop", "--project", "local/first-push="+home.project)
	if out, errb, code := fresh.run("pull", "--all"); code == ExitOK {
		t.Fatalf("pull before recover succeeded: out=%q err=%q", out, errb)
	} else if !strings.Contains(errb, "rein account recover") && !strings.Contains(errb, "rein account join") {
		t.Fatalf("pull before recover did not point at recovery: exit=%d err=%q", code, errb)
	}
	// Typed the way a person would: lower case, spaces instead of dashes.
	fresh.typedCode = strings.ToLower(strings.ReplaceAll(recoveryCode, "-", " "))
	out = fresh.mustRun("account recover", "account", "recover")
	fresh.typedCode = ""
	if !strings.Contains(out, "devices=2") {
		t.Fatalf("recover output %q", out)
	}
	status := fresh.mustRun("account status", "account", "status", "--json")
	var st map[string]any
	if err := json.Unmarshal([]byte(status), &st); err != nil || st["device_in_keyring"] != true || st["enrolled_via"] != "recover" || st["key_generation"] != float64(1) {
		t.Fatalf("account status after recover: %v %q", err, status)
	}

	// Before any agent is installed again there is no layout to restore
	// into; the refusal names the agent instead of a bare compatibility code.
	if out, errb, code := fresh.run("pull", "--all"); code != ExitCompatibility || !strings.Contains(errb, "claude session session-first-push: compatibility NOT_INSTALLED refuses restore; install and run claude once on this device") {
		t.Fatalf("pull before the agents exist: exit=%d out=%q err=%q", code, out, errb)
	}

	// With Claude Code and Codex back but not OpenCode, the pull restores
	// the first two and is refused on the third; the two that landed are
	// remembered, so the next pull does not mistake them for a conflict.
	home.reinstallAgents(t)
	if out, errb, code := fresh.run("pull", "--all"); code != ExitCompatibility || !strings.Contains(errb, "opencode session ses_fixture001: compatibility NOT_INSTALLED refuses restore") {
		t.Fatalf("pull without opencode: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := os.Stat(home.claude); err != nil {
		t.Fatalf("claude session not restored by the partial pull: %v", err)
	}
	home.reinstallOpenCode(t)
	pullOut := fresh.mustRun("pull --all", "pull", "--all", "--json")
	var pulled struct {
		Pulled  int `json:"pulled"`
		Skipped int `json:"skipped"`
	}
	// Claude Code and Codex landed in the partial pull and were remembered,
	// so this pull skips them as already synced and restores only OpenCode.
	if err := json.Unmarshal([]byte(pullOut), &pulled); err != nil || pulled.Pulled != 1 || pulled.Skipped != 2 {
		t.Fatalf("pull output %q: %v", pullOut, err)
	}
	if out := fresh.mustRun("conflicts list", "conflicts", "list"); strings.Contains(out, "session-first-push") || strings.Contains(out, "rollout-first-push") {
		t.Fatalf("the partial pull left a conflict behind: %q", out)
	}
	restored, err := os.ReadFile(home.claude)
	if err != nil || !bytes.Contains(restored, []byte("synthetic first push claude")) {
		t.Fatalf("claude session not restored: %v %q", err, restored)
	}
	if restored, err := os.ReadFile(home.codex); err != nil || !bytes.Contains(restored, []byte("synthetic first push codex")) {
		t.Fatalf("codex session not restored: %v", err)
	}
	if n := openCodeMessageCount(t, home.db, "ses_fixture001"); n != 2 {
		t.Fatalf("opencode session restored with %d messages, want 2", n)
	}

	// Verified resume of every agent's session on the recovered device: the
	// launch plan is complete and the verification report is not blocked.
	for ref, dir := range map[string]string{
		"claude:session-first-push": home.project,
		"codex:rollout-first-push":  home.project,
		"opencode:ses_fixture001":   "/Users/fixture-user/code/demo",
	} {
		raw := fresh.mustRun("resume "+ref, "resume", ref, "--dry-run", "--json")
		var plan struct {
			sessionindex.LaunchPlan
			Environment preflight.Report `json:"environment"`
		}
		if err := json.Unmarshal([]byte(raw), &plan); err != nil {
			t.Fatalf("resume %s plan: %v\n%s", ref, err, raw)
		}
		if plan.Executable == "" || len(plan.Args) == 0 || plan.Dir != dir {
			t.Fatalf("resume %s incomplete plan: %+v", ref, plan.LaunchPlan)
		}
		if plan.Environment.Decision != preflight.DecisionReady || plan.Environment.SessionRef != ref {
			t.Fatalf("resume %s verification report: %+v", ref, plan.Environment)
		}
	}

	// The recovered device pushes nothing new, and the account still counts
	// exactly one first push.
	if out := fresh.mustRun("push after recover", "push", "--all"); !strings.Contains(out, "skipped 3 unchanged") {
		t.Fatalf("push after recover %q", out)
	}
	if plane.firstPushes != 1 {
		t.Fatalf("first_push reached the control plane %d times over the whole journey, want exactly 1", plane.firstPushes)
	}
	if out := fresh.mustRun("hop status", "hop", "status"); !strings.Contains(out, "first push: 2026-08-23T12:05:00Z") || !strings.Contains(out, "Devices:  2 of 5") {
		t.Fatalf("hop status %q", out)
	}
}
