package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/daemon"
	"github.com/HarjjotSinghh/reinstate/internal/daemon/daemontest"
	syncengine "github.com/HarjjotSinghh/reinstate/internal/sync"
)

// fakeManager is a service manager that records what the CLI asked of it.
type fakeManager struct {
	mu        sync.Mutex
	calls     []string
	spec      daemon.Spec
	installed bool
	running   bool
}

func (m *fakeManager) Kind() string                            { return "fake" }
func (m *fakeManager) DefinitionPath(spec daemon.Spec) string  { return "/fake/" + spec.Label }
func (m *fakeManager) Render(spec daemon.Spec) ([]byte, error) { return []byte(spec.Label), nil }
func (m *fakeManager) record(call string, spec daemon.Spec) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, call)
	m.spec = spec
}
func (m *fakeManager) Install(_ context.Context, spec daemon.Spec) error {
	m.record("install", spec)
	m.installed, m.running = true, true
	return nil
}
func (m *fakeManager) Uninstall(_ context.Context, spec daemon.Spec) error {
	m.record("uninstall", spec)
	m.installed, m.running = false, false
	return nil
}
func (m *fakeManager) Start(_ context.Context, spec daemon.Spec) error {
	m.record("start", spec)
	m.running = true
	return nil
}
func (m *fakeManager) Stop(_ context.Context, spec daemon.Spec) error {
	m.record("stop", spec)
	m.running = false
	return nil
}
func (m *fakeManager) Status(_ context.Context, spec daemon.Spec) (daemon.State, error) {
	m.record("status", spec)
	detail := "stopped"
	if m.running {
		detail = "running"
	}
	return daemon.State{Installed: m.installed, Running: m.running, Definition: m.DefinitionPath(spec), Detail: detail}, nil
}

type recordingNotifier struct {
	mu    sync.Mutex
	shown []string
}

func (n *recordingNotifier) Notify(title, body string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.shown = append(n.shown, title+": "+body)
	return nil
}

func (n *recordingNotifier) all() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string{}, n.shown...)
}

// runningDaemon is `rein daemon run` executing in the background with a
// fake clock, a fake watcher, and an observer the test waits on.
type runningDaemon struct {
	t      *testing.T
	clock  *daemontest.FakeClock
	events chan daemon.Change
	seen   chan daemon.Event
	notify *recordingNotifier
	stdout *syncBuffer
	stderr *syncBuffer
	cancel context.CancelFunc
	done   chan int
}

func startDaemon(t *testing.T, d *pairDevice, manager daemon.Manager) *runningDaemon {
	t.Helper()
	r := &runningDaemon{
		t: t, clock: daemontest.NewFakeClock(), events: make(chan daemon.Change, 16), seen: make(chan daemon.Event, 4096),
		notify: &recordingNotifier{}, stdout: &syncBuffer{}, stderr: &syncBuffer{}, done: make(chan int, 1),
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	t.Setenv("REINSTATE_HOME", d.home)
	go func() {
		r.done <- d.execute(runOptions{stdout: r.stdout, stderr: r.stderr, ctx: ctx, daemon: daemonSeams{
			manager: manager, clock: r.clock, events: r.events, notifier: r.notify,
			observe: func(e daemon.Event) { r.seen <- e },
		}}, "daemon", "run")
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-r.done:
		case <-time.After(10 * time.Second):
		}
	})
	return r
}

// until collects loop events until n idles have passed.
func (r *runningDaemon) until(n int) []daemon.Event {
	r.t.Helper()
	var events []daemon.Event
	deadline := time.After(30 * time.Second)
	for n > 0 {
		select {
		case e := <-r.seen:
			events = append(events, e)
			if e.Kind == "idle" {
				n--
			}
		case code := <-r.done:
			r.t.Fatalf("daemon exited with %d: out=%q err=%q", code, r.stdout.String(), r.stderr.String())
		case <-deadline:
			r.t.Fatalf("daemon did not settle; events: %v", events)
		}
	}
	return events
}

func (r *runningDaemon) advance(d time.Duration) []daemon.Event {
	r.t.Helper()
	return r.until(r.clock.Advance(d))
}

func (r *runningDaemon) stop() int {
	r.t.Helper()
	r.cancel()
	select {
	case code := <-r.done:
		r.done <- code
		return code
	case <-time.After(10 * time.Second):
		r.t.Fatal("daemon did not stop")
		return -1
	}
}

func eventOf(t *testing.T, events []daemon.Event, kind string) daemon.Event {
	t.Helper()
	for _, e := range events {
		if e.Kind == kind {
			return e
		}
	}
	kinds := make([]string, 0, len(events))
	for _, e := range events {
		kinds = append(kinds, e.Kind)
	}
	t.Fatalf("no %q among %v", kind, kinds)
	return daemon.Event{}
}

// TestDaemonJourneyHop runs the daemon for a Hop device against the fake
// control plane and the fake locker: the start-up pull and push, a push
// after a session changes, the pull schedule, a pending device approval
// surfaced by notification, status file, stderr line, and rein daemon
// status, and a clean stop. Everything the daemon does goes through the
// same push and pull the shell commands run.
func TestDaemonJourneyHop(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000000daemon")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	manager := &fakeManager{}

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	// daemonStatus runs rein daemon status on A with the fake manager.
	daemonStatus := func() (string, int) {
		t.Helper()
		t.Setenv("REINSTATE_HOME", a.home)
		out, errb := &syncBuffer{}, &syncBuffer{}
		code := a.execute(runOptions{stdout: out, stderr: errb, daemon: daemonSeams{manager: manager}}, "daemon", "status")
		return out.String(), code
	}

	// A second daemon for the same home is refused while the first holds
	// the lock.
	d := startDaemon(t, a, manager)
	start := d.until(1)
	eventOf(t, start, "start")
	if e := eventOf(t, start, "pull"); e.Err != nil {
		t.Fatalf("start-up pull: %v", e.Err)
	}
	eventOf(t, start, "approvals")
	if out, errb, code := a.run("daemon", "run"); code != ExitRuntime || !strings.Contains(errb, "already running") {
		t.Fatalf("second daemon: exit=%d out=%q err=%q", code, out, errb)
	}
	t.Setenv("REINSTATE_HOME", a.home)

	// The start-up push lands the fixture session in the locker.
	if e := eventOf(t, d.advance(3*time.Second), "push"); e.Err != nil {
		t.Fatalf("start-up push: %v", e.Err)
	}
	status, err := daemon.ReadStatus(a.home)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Push.OK || status.Push.Summary != "pushed 1 snapshot(s), skipped 0 unchanged" || status.Backend != "hop" || status.Watch != "fake" {
		t.Fatalf("status after first push: %+v", status)
	}
	if len(status.Devices) != 1 || status.Devices[0].Name != "macbook" || !status.Devices[0].This {
		t.Fatalf("devices in status: %+v", status.Devices)
	}
	if len(status.Roots) != 1 || status.Roots[0] != filepath.Join(userHome, ".claude", "projects") {
		t.Fatalf("watch roots: %v", status.Roots)
	}

	// A change to the session file: pushed after the debounce as a new
	// snapshot; a scheduled pull then finds everything already synced.
	sessionPath := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project), "session-locker.jsonl")
	if err := appendLine(sessionPath, `{"type":"assistant","message":{"content":"daemon saw this"}}`); err != nil {
		t.Fatal(err)
	}
	d.events <- daemon.Change{Path: sessionPath}
	d.until(1)
	if e := eventOf(t, d.advance(3*time.Second), "push"); e.Err != nil {
		t.Fatalf("push after change: %v", e.Err)
	}
	if e := eventOf(t, d.advance(5*time.Minute), "pull"); e.Err != nil {
		t.Fatalf("scheduled pull: %v", e.Err)
	}
	status, _ = daemon.ReadStatus(a.home)
	if status.Pull.Summary != "pulled 0 snapshot(s), skipped 1 already synced" {
		t.Fatalf("pull summary: %q", status.Pull.Summary)
	}
	out, errb, code := a.run("status")
	if code != ExitOK || strings.Contains(errb, "wants to join") {
		t.Fatalf("rein status with nothing pending: exit=%d out=%q err=%q", code, out, errb)
	}

	// A resume pulls first only when the daemon's last pull is stale:
	// just after the scheduled pull nothing is fetched; a minute later
	// the locker is read again (nothing newer, so nothing is said).
	resumeOpts := a.options(runOptions{stdout: &syncBuffer{}, stderr: &syncBuffer{}})
	requests := len(plane.s3.RequestLog())
	if note := resumePull(context.Background(), resumeOpts, a.home, d.clock.Now()); note != "" {
		t.Fatalf("resume pull right after the daemon pulled: %q", note)
	}
	if n := len(plane.s3.RequestLog()); n != requests {
		t.Fatalf("resume must not pull when the daemon just did: %d new request(s)", n-requests)
	}
	if note := resumePull(context.Background(), resumeOpts, a.home, d.clock.Now().Add(time.Minute)); note != "" {
		t.Fatalf("resume pull with nothing newer: %q", note)
	}
	if n := len(plane.s3.RequestLog()); n == requests {
		t.Fatal("resume should pull when the daemon's last pull is stale")
	}

	// Device B asks to join. The daemon's next poll notifies, records the
	// request in the status file, and the next shell command says so.
	b := newPairDevice(t, plane, "desktop")
	if out, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init: exit=%d out=%q err=%q", code, out, errb)
	}
	join := b.startJoin()
	t.Setenv("REINSTATE_HOME", a.home)
	polled := d.advance(5 * time.Minute)
	eventOf(t, polled, "approvals")
	eventOf(t, polled, "notify")
	if shown := d.notify.all(); len(shown) != 1 || shown[0] != "Reinstate: device wants to join: desktop wants to join your account. Run: rein devices approve" {
		t.Fatalf("notifications: %q", shown)
	}
	status, _ = daemon.ReadStatus(a.home)
	if len(status.Pending) != 1 || status.Pending[0].DeviceName != "desktop" || status.Pending[0].RequestID != "pair-1" || status.Pending[0].ExpiresAt.IsZero() {
		t.Fatalf("pending in status: %+v", status.Pending)
	}
	out, errb, code = a.run("status")
	if code != ExitOK || !strings.Contains(errb, `device "desktop" wants to join your account; run rein devices approve`) {
		t.Fatalf("rein status should announce the request on stderr: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.run("status", "--json"); code != ExitOK || strings.Contains(errb, "wants to join") {
		t.Fatalf("--json must keep stderr clean: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, errb, _ := a.run("devices"); strings.Contains(errb, "wants to join your account") {
		t.Fatalf("rein devices must not repeat the announcement: %q", errb)
	}
	out, code = daemonStatus()
	if code != ExitOK {
		t.Fatalf("daemon status: exit=%d out=%q", code, out)
	}
	for _, want := range []string{"login:    fake", "daemon:   running (pid", "push:     pushed 1 snapshot(s), skipped 0 unchanged, just now", "pull:     pulled 0 snapshot(s), skipped 1 already synced, just now", "devices:  macbook (this device), desktop", `pending:  device "desktop" wants to join`, "watching: " + status.Roots[0]} {
		if !strings.Contains(out, want) {
			t.Fatalf("daemon status missing %q:\n%s", want, out)
		}
	}
	if line := daemonSummaryLine(a.home, time.Now()); !strings.Contains(line, "daemon running") || !strings.Contains(line, `"desktop" wants to join`) || !strings.Contains(line, "2 device(s)") {
		t.Fatalf("switcher line: %q", line)
	}

	// Approval stays interactive: A types the code, B finishes, and the
	// daemon's next poll clears the request without a second notification.
	if out, errb, code := a.approve(join.code, false); code != ExitOK {
		t.Fatalf("approve: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := join.finish(t); code != ExitOK {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	t.Setenv("REINSTATE_HOME", a.home)
	eventOf(t, d.advance(time.Minute), "approvals")
	status, _ = daemon.ReadStatus(a.home)
	if len(status.Pending) != 0 || len(status.Devices) != 2 {
		t.Fatalf("status after approval: pending=%+v devices=%+v", status.Pending, status.Devices)
	}
	if shown := d.notify.all(); len(shown) != 1 {
		t.Fatalf("notified again: %q", shown)
	}
	if _, errb, _ := a.run("status"); strings.Contains(errb, "wants to join") {
		t.Fatalf("announcement should stop once approved: %q", errb)
	}

	// Stop: the status file says so, and the log has the story.
	if code := d.stop(); code != ExitOK {
		t.Fatalf("daemon exit=%d err=%q", code, d.stderr.String())
	}
	status, _ = daemon.ReadStatus(a.home)
	if status.PID != 0 || status.Alive(time.Now()) {
		t.Fatalf("stopped daemon still alive: %+v", status)
	}
	logText, err := os.ReadFile(daemon.LogPath(a.home))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"daemon started pid=", "push: pushed 1 snapshot(s)", "change: " + sessionPath, `approval pending: device "desktop"`, "daemon stopping"} {
		if !strings.Contains(string(logText), want) {
			t.Fatalf("log missing %q:\n%s", want, logText)
		}
	}
	if strings.Contains(string(logText), join.code) {
		t.Fatal("the pairing code must never reach the daemon log")
	}
	out, code = daemonStatus()
	if code != ExitOK || !strings.Contains(out, "daemon:   stopped (last heartbeat just now)") {
		t.Fatalf("daemon status after stop: exit=%d out=%q", code, out)
	}
}

func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// TestDaemonInstallLifecycle covers install, status, stop, start, and
// uninstall against a recording service manager, and the refusals that
// keep a daemon from being registered where it could not run.
func TestDaemonInstallLifecycle(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000install")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	manager := &fakeManager{}
	a := newPairDevice(t, plane, "macbook")
	run := func(args ...string) (string, string, int) {
		t.Helper()
		t.Setenv("REINSTATE_HOME", a.home)
		out, errb := &syncBuffer{}, &syncBuffer{}
		code := a.execute(runOptions{stdout: out, stderr: errb, daemon: daemonSeams{manager: manager, executable: "/opt/rein/bin/rein"}}, args...)
		return out.String(), errb.String(), code
	}
	if out, errb, code := run("daemon", "install"); code != ExitConfig || !strings.Contains(errb, "config") {
		t.Fatalf("install before init: exit=%d out=%q err=%q", code, out, errb)
	}
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("%v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	if out, errb, code := run("daemon", "install"); code != ExitConfig || !strings.Contains(errb, "rein account init") {
		t.Fatalf("install before account init: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.run("account", "init"); code != ExitOK {
		t.Fatalf("account init: exit=%d out=%q err=%q", code, out, errb)
	}
	out, errb, code := run("daemon", "status")
	if code != ExitOK || !strings.Contains(out, "not installed") || !strings.Contains(out, "never ran") {
		t.Fatalf("status before install: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := run("daemon", "install", "--env", "REINSTATE_S3_SECRET_ACCESS_KEY=abc"); code != ExitUsage || !strings.Contains(errb, "looks like a credential") || manager.installed {
		t.Fatalf("install must refuse to bake a credential into the definition: exit=%d out=%q err=%q calls=%v", code, out, errb, manager.calls)
	}
	out, errb, code = run("daemon", "install", "--pull-every", "10m", "--poll", "--env", "REINSTATE_BACKEND=memory")
	if code != ExitOK || !strings.Contains(out, "installed fake com.reinstate.daemon.") {
		t.Fatalf("install: exit=%d out=%q err=%q", code, out, errb)
	}
	spec := manager.spec
	if spec.Executable != "/opt/rein/bin/rein" || strings.Join(spec.Args, " ") != "daemon run --pull-every 10m0s --poll" || spec.Home != a.home || spec.Env["REINSTATE_BACKEND"] != "memory" {
		t.Fatalf("spec: %+v", spec)
	}
	if spec.Label == daemon.DefaultLabel || !strings.HasPrefix(spec.Label, daemon.DefaultLabel+".") {
		t.Fatalf("a non-default home must get its own label: %q", spec.Label)
	}
	if !strings.HasPrefix(spec.Path, filepath.Dir("/opt/rein/bin/rein")) || spec.LogPath != filepath.Join(a.home, "daemon", "launch.log") {
		t.Fatalf("spec path/log: %+v", spec)
	}
	out, _, code = run("daemon", "status", "--json")
	if code != ExitOK {
		t.Fatalf("status --json: exit=%d", code)
	}
	var payload struct {
		Service struct {
			Kind      string `json:"kind"`
			Installed bool   `json:"installed"`
			Running   bool   `json:"running"`
		} `json:"service"`
		Alive bool `json:"alive"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Service.Kind != "fake" || !payload.Service.Installed || !payload.Service.Running || payload.Alive {
		t.Fatalf("status payload: %+v", payload)
	}
	if out, _, code := run("daemon", "stop"); code != ExitOK || !strings.Contains(out, "stopped") {
		t.Fatalf("stop: exit=%d out=%q", code, out)
	}
	if out, _, code := run("daemon", "start"); code != ExitOK || !strings.Contains(out, "started") {
		t.Fatalf("start: exit=%d out=%q", code, out)
	}
	if out, _, code := run("daemon", "uninstall"); code != ExitOK || !strings.Contains(out, "uninstalled fake") {
		t.Fatalf("uninstall: exit=%d out=%q", code, out)
	}
	want := []string{"status", "install", "status", "stop", "start", "uninstall"}
	if strings.Join(manager.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("manager calls: %v", manager.calls)
	}
}

func TestDaemonRunFlagsRoundTrip(t *testing.T) {
	cases := []struct {
		flags daemonRunFlags
		want  string
	}{
		{daemonRunFlags{}, "daemon run"},
		{daemonRunFlags{pullEvery: daemon.DefaultPullEvery, debounce: daemon.DefaultDebounce}, "daemon run"},
		{daemonRunFlags{pullEvery: 10 * time.Minute, debounce: 5 * time.Second, poll: true}, "daemon run --pull-every 10m0s --debounce 5s --poll"},
	}
	for _, c := range cases {
		if got := strings.Join(c.flags.args(), " "); got != c.want {
			t.Errorf("%+v -> %q, want %q", c.flags, got, c.want)
		}
	}
}

func TestSessionRootFor(t *testing.T) {
	if got := sessionRootFor("claude", "/h/.claude"); got != filepath.Join("/h/.claude", "projects") {
		t.Fatal(got)
	}
	if got := sessionRootFor("codex", "/h/.codex"); got != filepath.Join("/h/.codex", "sessions") {
		t.Fatal(got)
	}
	if got := sessionRootFor("opencode", "/h/opencode"); got != "/h/opencode" {
		t.Fatal(got)
	}
}

// TestPullAllContinuesPastConflictedSessions: the daemon runs pull --all on
// a schedule, so one diverged session must neither hold back the other
// sessions' newer snapshots nor grow the conflicts directory on every tick.
func TestPullAllContinuesPastConflictedSessions(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000pullconflict")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	root := filepath.Join(os.Getenv("HOME"), ".claude", "projects", claudeProjectDirectoryForTest(project))
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	second := append(append([]byte{}, meta...), []byte("\n"+`{"type":"user","message":{"content":"second session"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, "session-second.jsonl"), second, 0o600); err != nil {
		t.Fatal(err)
	}

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	// Another device moved session-locker's head (this device's state no
	// longer names the remote snapshot) and this device edited its local
	// copy since its last sync. That is a divergence pull cannot resolve on
	// its own, while session-second stays in step.
	state, err := config.LoadState(a.home)
	if err != nil {
		t.Fatal(err)
	}
	lockerKey := syncengine.SessionKey("claude", "session-locker")
	entry := state.Sessions[lockerKey]
	entry.RemoteRevision = "snap-moved-elsewhere"
	state.Sessions[lockerKey] = entry
	if err := config.SaveState(a.home, state); err != nil {
		t.Fatal(err)
	}
	if err := appendLine(filepath.Join(root, "session-locker.jsonl"), `{"type":"assistant","message":{"content":"edited here too"}}`); err != nil {
		t.Fatal(err)
	}

	conflictFiles := func() int {
		entries, _ := filepath.Glob(filepath.Join(a.home, "conflicts", "c-*.json"))
		return len(entries)
	}
	for i := 1; i <= 3; i++ {
		out, errb, code := a.run("pull", "--all", "--json")
		if code != ExitConflict {
			t.Fatalf("pull %d: exit=%d out=%q err=%q", i, code, out, errb)
		}
		var result struct {
			Pulled    int      `json:"pulled"`
			Skipped   int      `json:"skipped"`
			Conflicts []string `json:"conflicts"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("pull %d output %q: %v", i, out, err)
		}
		if result.Skipped != 1 || len(result.Conflicts) != 1 || result.Conflicts[0] != lockerKey {
			t.Fatalf("pull %d should skip the synced session and report the diverged one: %+v", i, result)
		}
		if !strings.Contains(errb, "1 session(s) diverged locally") || !strings.Contains(errb, lockerKey) {
			t.Fatalf("pull %d stderr: %q", i, errb)
		}
		if n := conflictFiles(); n != 1 {
			t.Fatalf("after pull %d: %d conflict record(s), want exactly 1", i, n)
		}
	}
	// push --all meets the same divergence, does not add a record either,
	// and still pushes the other session's change.
	if err := appendLine(filepath.Join(root, "session-second.jsonl"), `{"type":"assistant","message":{"content":"second moved on"}}`); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		out, errb, code := a.run("push", "--all", "--json")
		if code != ExitConflict {
			t.Fatalf("push %d: exit=%d out=%q err=%q", i, code, out, errb)
		}
		var result struct {
			Snapshots []string `json:"snapshots"`
			Skipped   int      `json:"skipped"`
			Conflicts []string `json:"conflicts"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("push %d output %q: %v", i, out, err)
		}
		wantPushed := 0
		if i == 1 {
			wantPushed = 1
		}
		if len(result.Snapshots) != wantPushed || result.Skipped != 1-wantPushed || len(result.Conflicts) != 1 || result.Conflicts[0] != lockerKey {
			t.Fatalf("push %d should push the other session past the diverged one: %+v", i, result)
		}
		if n := conflictFiles(); n != 1 {
			t.Fatalf("after push %d: %d conflict record(s), want exactly 1", i, n)
		}
	}
}

// TestDaemonRunKeepsThePassphraseForItsLifetime: a BYO home still on the
// passphrase model hands the daemon its passphrase through a descriptor,
// which is read once; every later push and pull must reuse it rather than
// read the spent descriptor and fail with an empty secret.
func TestDaemonRunKeepsThePassphraseForItsLifetime(t *testing.T) {
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	for _, env := range []string{"REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	plane := newFakeControlPlane(t)
	a := newPairDevice(t, plane, "macbook")
	if out, errb, code := a.run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "reinstate-test", "--project", "local/locker="+project, "--yes"); code != ExitOK {
		t.Fatalf("init: exit=%d out=%q err=%q", code, out, errb)
	}

	// Without the descriptor the daemon refuses up front instead of failing
	// on its first pull.
	if out, errb, code := a.run("daemon", "run"); code != ExitConfig || !strings.Contains(errb, "cannot prompt for a passphrase") {
		t.Fatalf("daemon run without a passphrase source: exit=%d out=%q err=%q", code, out, errb)
	}

	withSecretFD(t, "REINSTATE_PASSPHRASE_FD", "daemon-test-passphrase-not-real")
	d := startDaemon(t, a, &fakeManager{})
	if e := eventOf(t, d.until(1), "pull"); e.Err != nil {
		t.Fatalf("start-up pull: %v", e.Err)
	}
	if e := eventOf(t, d.advance(3*time.Second), "push"); e.Err != nil {
		t.Fatalf("start-up push (second use of the passphrase): %v", e.Err)
	}
	if e := eventOf(t, d.advance(time.Minute), "pull"); e.Err != nil {
		t.Fatalf("scheduled pull (third use of the passphrase): %v", e.Err)
	}
	sessionPath := filepath.Join(os.Getenv("HOME"), ".claude", "projects", claudeProjectDirectoryForTest(project), "session-locker.jsonl")
	if err := appendLine(sessionPath, `{"type":"assistant","message":{"content":"still encrypted with the same passphrase"}}`); err != nil {
		t.Fatal(err)
	}
	d.events <- daemon.Change{Path: sessionPath}
	d.until(1)
	if e := eventOf(t, d.advance(3*time.Second), "push"); e.Err != nil {
		t.Fatalf("push after change: %v", e.Err)
	}
	status, err := daemon.ReadStatus(a.home)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Push.OK || !status.Pull.OK || status.Push.Summary != "pushed 1 snapshot(s), skipped 0 unchanged" {
		t.Fatalf("status: push=%+v pull=%+v", status.Push, status.Pull)
	}
	// The shell, given a fresh descriptor, reads what the daemon pushed.
	if code := d.stop(); code != ExitOK {
		t.Fatalf("daemon exit=%d err=%q", code, d.stderr.String())
	}
	withSecretFD(t, "REINSTATE_PASSPHRASE_FD", "daemon-test-passphrase-not-real")
	if out, errb, code := a.run("pull", "--all", "--json"); code != ExitOK || !strings.Contains(out, `"skipped": 1`) {
		t.Fatalf("pull after the daemon: exit=%d out=%q err=%q", code, out, errb)
	}
}
