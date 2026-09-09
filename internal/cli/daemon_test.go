package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
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

func startDaemon(t *testing.T, d *pairDevice, manager daemon.Manager, clock *daemontest.FakeClock) *runningDaemon {
	t.Helper()
	r := &runningDaemon{
		t: t, clock: clock, events: make(chan daemon.Change, 16), seen: make(chan daemon.Event, 4096),
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
			logText, _ := os.ReadFile(daemon.LogPath(os.Getenv("REINSTATE_HOME")))
			r.t.Fatalf("daemon did not settle; events: %v\nstdout=%q\nstderr=%q\ndaemon.log=%q", events, r.stdout.String(), r.stderr.String(), logText)
		}
	}
	return events
}

func (r *runningDaemon) advance(d time.Duration) []daemon.Event {
	r.t.Helper()
	return r.until(r.clock.Advance(d))
}

// waitFor discards events until one of kind arrives. A change is observed
// only after the loop has armed its debounce timer, so waiting for it
// before advancing the clock cannot race the loop; waiting for an idle
// could, because an idle left over from the previous step satisfies it
// before the change is even taken.
func (r *runningDaemon) waitFor(kind string) daemon.Event {
	r.t.Helper()
	deadline := time.After(30 * time.Second)
	var seen []string
	for {
		select {
		case e := <-r.seen:
			if e.Kind == kind {
				return e
			}
			seen = append(seen, e.Kind)
		case code := <-r.done:
			r.t.Fatalf("daemon exited with %d before %q: out=%q err=%q", code, kind, r.stdout.String(), r.stderr.String())
		case <-deadline:
			r.t.Fatalf("no %q event; saw %v", kind, seen)
		}
	}
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
	// One clock for the whole journey: the loop stamps the status file with
	// it, and `rein daemon status` reads those stamps against it. Reading
	// them against the wall clock made this test fail a day after it was
	// written.
	clock := daemontest.NewFakeClock()
	// daemonStatus runs rein daemon status on A with the fake manager.
	daemonStatus := func() (string, int) {
		t.Helper()
		t.Setenv("REINSTATE_HOME", a.home)
		out, errb := &syncBuffer{}, &syncBuffer{}
		code := a.execute(runOptions{stdout: out, stderr: errb, daemon: daemonSeams{manager: manager, clock: clock}}, "daemon", "status")
		return out.String(), code
	}

	// A second daemon for the same home is refused while the first holds
	// the lock.
	d := startDaemon(t, a, manager, clock)
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
	d.waitFor("change")
	d.until(1) // the idle that follows the change, so the next advance sees the push
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
	for _, want := range []string{"login:    fake", "daemon:   running (pid", "push:     pushed 1 snapshot(s), skipped 0 unchanged, 10m ago", "pull:     pulled 0 snapshot(s), skipped 1 already synced, just now", "devices:  macbook (this device), desktop", `pending:  device "desktop" wants to join`, "watching: " + status.Roots[0]} {
		if !strings.Contains(out, want) {
			t.Fatalf("daemon status missing %q:\n%s", want, out)
		}
	}
	if line := daemonSummaryLine(a.home, d.clock.Now()); !strings.Contains(line, "daemon running") || !strings.Contains(line, `"desktop" wants to join`) || !strings.Contains(line, "2 device(s)") {
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
	if status.PID != 0 || status.Alive(d.clock.Now()) {
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
	d := startDaemon(t, a, &fakeManager{}, daemontest.NewFakeClock())
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
	d.waitFor("change")
	d.until(1) // the idle that follows the change, so the next advance sees the push
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

// ---------- #424: pinning the agent-root environment ----------

// blankAgentRootEnv clears every catalog RootEnv variable for the
// lifetime of the test, so a test's own env, uninvolved agent roots this
// host happens to have configured (this repo's own dev environment sets
// CLAUDE_CONFIG_DIR, CODEX_HOME, and XDG_DATA_HOME to isolated fixture
// paths) never leak into what #424's capture and comparison resolve. It
// never touches a real, unisolated agent home: every one of these
// variables is either already a synthetic fixture path on this host or is
// cleared outright, and nothing under this package writes through them.
func blankAgentRootEnv(t *testing.T) {
	t.Helper()
	for _, name := range agentRootEnvNames() {
		t.Setenv(name, "")
	}
}

// TestAgentRootEnvNames asserts the fixed, catalog-derived variable set
// install-time capture, startup comparison, and status all agree on: the
// three home variables reinstate#424 names explicitly, sorted, with no
// duplicates even though catalog descriptors can share a variable (OpenCode
// and any future agent reusing XDG_DATA_HOME).
func TestAgentRootEnvNames(t *testing.T) {
	names := agentRootEnvNames()
	for _, want := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "XDG_DATA_HOME"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("agentRootEnvNames() = %v, missing %q", names, want)
		}
	}
	sorted := append([]string{}, names...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(names, sorted) {
		t.Fatalf("agentRootEnvNames() = %v, not sorted", names)
	}
	seen := map[string]int{}
	for _, n := range names {
		seen[n]++
	}
	for name, n := range seen {
		if n > 1 {
			t.Fatalf("agentRootEnvNames() repeats %q", name)
		}
	}
}

// TestResolvedAgentRoots checks the getenv-driven capture that both
// `daemon install` and `daemon run` use, against a fake environment rather
// than the process's real one.
func TestResolvedAgentRoots(t *testing.T) {
	fake := map[string]string{"CLAUDE_CONFIG_DIR": `D:\iso\claude`, "CODEX_HOME": "", "XDG_DATA_HOME": "  "}
	got := resolvedAgentRoots(func(name string) string { return fake[name] })
	want := daemon.AgentRoots{"CLAUDE_CONFIG_DIR": `D:\iso\claude`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolvedAgentRoots() = %#v, want %#v", got, want)
	}
}

func TestFormatAgentRootDiffs(t *testing.T) {
	diffs := []daemon.AgentRootsDiff{
		{Name: "CLAUDE_CONFIG_DIR", Recorded: `D:\iso\claude`, Current: ""},
		{Name: "CODEX_HOME", Recorded: "", Current: `D:\login\codex`},
	}
	got := formatAgentRootDiffs(diffs)
	want := `CLAUDE_CONFIG_DIR recorded=D:\iso\claude current=unset; CODEX_HOME recorded=unset current=D:\login\codex`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if formatAgentRootDiffs(nil) != "" {
		t.Fatalf("empty diffs should format to \"\"")
	}
}

// TestApplyInstalledAgentRoots is the table-driven core of #424: no
// recorded baseline is a no-op; a baseline that matches the process's own
// environment is applied harmlessly; a baseline that disagrees refuses
// unless allowChange is set, and even then the process ends up pinned to
// the *recorded* roots, never the drifted ones it was launched with — the
// entire point of --allow-root-change is to permit the start, not to adopt
// the drift.
func TestApplyInstalledAgentRoots(t *testing.T) {
	// The variable under test must be one applyInstalledAgentRoots actually
	// resolves (agentRootEnvNames(), the catalog's RootEnv set) — a made-up
	// name would never appear on the "current" side of the comparison and
	// would make every case look like a mismatch.
	const testVar = "CLAUDE_CONFIG_DIR"

	cases := []struct {
		name          string
		recordSpec    *daemon.AgentRoots // nil: never call daemon install's WriteAgentRoots
		envValue      string             // "" leaves testVar unset
		allowChange   bool
		wantErr       bool
		wantCode      int
		wantErrSubstr string
		wantEnvAfter  string
	}{
		{
			name:         "no recorded baseline: skip the check entirely, env left untouched",
			recordSpec:   nil,
			envValue:     `D:\whatever\the\login\shell\has`,
			wantEnvAfter: `D:\whatever\the\login\shell\has`,
		},
		{
			name:         "recorded matches current: applied, no error",
			recordSpec:   &daemon.AgentRoots{testVar: "/iso/agent-home"},
			envValue:     "/iso/agent-home",
			wantEnvAfter: "/iso/agent-home",
		},
		{
			name:          "recorded set, current unset (the H7 mechanism): refused without the flag",
			recordSpec:    &daemon.AgentRoots{testVar: "/iso/agent-home"},
			envValue:      "",
			wantErr:       true,
			wantCode:      ExitSafety,
			wantErrSubstr: "recorded=/iso/agent-home current=unset",
		},
		{
			name:         "recorded set, current unset, override given: starts pinned to the recorded root, not the drifted (unset) one",
			recordSpec:   &daemon.AgentRoots{testVar: "/iso/agent-home"},
			envValue:     "",
			allowChange:  true,
			wantEnvAfter: "/iso/agent-home",
		},
		{
			name:          "recorded unset, current newly set: refused without the flag too",
			recordSpec:    &daemon.AgentRoots{},
			envValue:      "/surprising/new/root",
			wantErr:       true,
			wantCode:      ExitSafety,
			wantErrSubstr: "recorded=unset current=/surprising/new/root",
		},
		{
			// The gap this fix closes: --allow-root-change must not let a
			// variable that was unset at install (absent from the recorded
			// baseline) silently adopt whatever the process's own launch
			// environment now carries for it. It must end up force-unset,
			// pinned to unset just as firmly as a customized variable is
			// pinned to its recorded value.
			name:         "recorded unset, current newly set, override given: starts pinned to unset, never adopts the drifted value",
			recordSpec:   &daemon.AgentRoots{},
			envValue:     "/surprising/new/root",
			allowChange:  true,
			wantEnvAfter: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			blankAgentRootEnv(t)
			home := t.TempDir()
			if c.recordSpec != nil {
				if err := daemon.WriteAgentRoots(home, *c.recordSpec); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv(testVar, c.envValue)

			err := applyInstalledAgentRoots(home, c.allowChange)
			if c.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want a refusal")
				}
				ee, ok := err.(*ExitError)
				if !ok {
					t.Fatalf("err = %v (%T), want *ExitError", err, err)
				}
				if ee.Code != c.wantCode {
					t.Fatalf("code = %d, want %d", ee.Code, c.wantCode)
				}
				if !strings.Contains(ee.Message, c.wantErrSubstr) {
					t.Fatalf("message %q does not contain %q", ee.Message, c.wantErrSubstr)
				}
				if !strings.Contains(ee.Message, "reinstate#424") {
					t.Fatalf("message %q should reference reinstate#424", ee.Message)
				}
				if !strings.Contains(ee.Message, "--allow-root-change") {
					t.Fatalf("message %q should name the override flag", ee.Message)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got := os.Getenv(testVar); got != c.wantEnvAfter {
				t.Fatalf("env after = %q, want %q", got, c.wantEnvAfter)
			}
		})
	}
}

// TestDaemonInstallRecordsAgentRootsAndUninstallClearsThem is the
// persistence half of #424 through the real CLI commands: `daemon install`
// captures the agent-root environment the installing shell resolved and
// `daemon uninstall` drops the recorded baseline so a later install starts
// clean. It never touches a real agent home: CLAUDE_CONFIG_DIR and
// CODEX_HOME point at throwaway directories under t.TempDir() for the
// whole test.
func TestDaemonInstallRecordsAgentRootsAndUninstallClearsThem(t *testing.T) {
	blankAgentRootEnv(t)
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000i424root")
	t.Setenv(hopURLEnv, plane.srv.URL)
	isolatedClaude := filepath.Join(t.TempDir(), "isolated-claude-home")
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD"} {
		t.Setenv(env, "")
	}
	t.Setenv("CLAUDE_CONFIG_DIR", isolatedClaude)
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
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("%v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	if out, errb, code := a.run("account", "init"); code != ExitOK {
		t.Fatalf("account init: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := daemon.ReadAgentRoots(a.home); !errors.Is(err, daemon.ErrNoAgentRoots) {
		t.Fatalf("before install: err=%v, want ErrNoAgentRoots", err)
	}
	if out, errb, code := run("daemon", "install"); code != ExitOK {
		t.Fatalf("install: exit=%d out=%q err=%q", code, out, errb)
	} else if !strings.Contains(out, "pinned agent roots: CLAUDE_CONFIG_DIR") {
		t.Fatalf("install output did not announce the pinned roots: %q", out)
	}
	roots, err := daemon.ReadAgentRoots(a.home)
	if err != nil {
		t.Fatalf("after install: %v", err)
	}
	if want := (daemon.AgentRoots{"CLAUDE_CONFIG_DIR": isolatedClaude}); !reflect.DeepEqual(roots, want) {
		t.Fatalf("recorded roots = %#v, want %#v", roots, want)
	}
	if out, errb, code := run("daemon", "uninstall"); code != ExitOK {
		t.Fatalf("uninstall: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := daemon.ReadAgentRoots(a.home); !errors.Is(err, daemon.ErrNoAgentRoots) {
		t.Fatalf("after uninstall: err=%v, want ErrNoAgentRoots", err)
	}
}

// TestDaemonStatusReportsAgentRoots checks that `rein daemon status`
// (human and --json) surfaces the pinned roots and, once the environment
// drifts from what was recorded, a clear warning that the next `daemon
// run` will refuse.
func TestDaemonStatusReportsAgentRoots(t *testing.T) {
	blankAgentRootEnv(t)
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	manager := &fakeManager{}
	stdout, stderr := &syncBuffer{}, &syncBuffer{}
	opts := Options{Name: "rein", Stdout: stdout, Stderr: stderr, Daemon: daemonSeams{manager: manager}}

	status := func() (string, string) {
		t.Helper()
		stdout.b.Reset()
		stderr.b.Reset()
		opts.Args = []string{"daemon", "status"}
		if code := Execute(opts); code != ExitOK {
			t.Fatalf("status: exit=%d out=%q err=%q", code, stdout.String(), stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	statusJSON := func() map[string]any {
		t.Helper()
		stdout.b.Reset()
		stderr.b.Reset()
		opts.Args = []string{"daemon", "status", "--json"}
		if code := Execute(opts); code != ExitOK {
			t.Fatalf("status --json: exit=%d out=%q err=%q", code, stdout.String(), stderr.String())
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(stdout.String()), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}

	// No baseline recorded yet (daemon install was never run here).
	out, _ := status()
	if !strings.Contains(out, "roots:    no baseline recorded") {
		t.Fatalf("status without a baseline: %q", out)
	}
	payload := statusJSON()
	roots, _ := payload["agent_roots"].(map[string]any)
	if recorded, _ := roots["recorded"].(bool); recorded {
		t.Fatalf("agent_roots.recorded should be false before install: %#v", roots)
	}

	// Record a baseline as `daemon install` would, then match the current
	// environment: no drift warning.
	pinned := daemon.AgentRoots{"CLAUDE_CONFIG_DIR": `D:\iso\claude`}
	if err := daemon.WriteAgentRoots(home, pinned); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", `D:\iso\claude`)
	out, _ = status()
	if !strings.Contains(out, `root:     CLAUDE_CONFIG_DIR=D:\iso\claude`) {
		t.Fatalf("status did not print the pinned root: %q", out)
	}
	if strings.Contains(out, "environment changed since install") {
		t.Fatalf("status warned about drift when the environment matches: %q", out)
	}
	payload = statusJSON()
	roots, _ = payload["agent_roots"].(map[string]any)
	if recorded, _ := roots["recorded"].(bool); !recorded {
		t.Fatalf("agent_roots.recorded should be true: %#v", roots)
	}
	if match, _ := roots["matches_current_environment"].(bool); !match {
		t.Fatalf("agent_roots.matches_current_environment should be true: %#v", roots)
	}

	// Drift the environment away from what was recorded.
	t.Setenv("CLAUDE_CONFIG_DIR", `D:\login\claude`)
	out, _ = status()
	if !strings.Contains(out, "environment changed since install") || !strings.Contains(out, "--allow-root-change") {
		t.Fatalf("status did not warn about the drift: %q", out)
	}
	payload = statusJSON()
	roots, _ = payload["agent_roots"].(map[string]any)
	if match, _ := roots["matches_current_environment"].(bool); match {
		t.Fatalf("agent_roots.matches_current_environment should be false after drift: %#v", roots)
	}
	if _, ok := roots["diffs"]; !ok {
		t.Fatalf("agent_roots.diffs missing after drift: %#v", roots)
	}
}

// TestDaemonRunRefusesWhenAgentRootsDrifted is the end-to-end regression
// test for #424 / Hop row H7: a home whose recorded agent roots point at
// an isolated location, run under a process whose own environment (as a
// scheduled task's login environment would be) no longer has that
// override, refuses to start instead of silently falling back to the
// live/default agent roots — and --allow-root-change lets it start
// (still pinned to the recorded, isolated root, never the live one).
func TestDaemonRunRefusesWhenAgentRootsDrifted(t *testing.T) {
	blankAgentRootEnv(t)
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000i424refusal")
	t.Setenv(hopURLEnv, plane.srv.URL)
	isolatedClaude := filepath.Join(t.TempDir(), "isolated-claude-home")
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD"} {
		t.Setenv(env, "")
	}
	t.Setenv("CLAUDE_CONFIG_DIR", isolatedClaude)
	project := writeClaudeFixture(t)
	a := newPairDevice(t, plane, "macbook")
	t.Setenv("REINSTATE_HOME", a.home)
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("%v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	if err := daemon.WriteAgentRoots(a.home, daemon.AgentRoots{"CLAUDE_CONFIG_DIR": isolatedClaude}); err != nil {
		t.Fatal(err)
	}

	// The login environment the scheduled task would actually inherit: no
	// CLAUDE_CONFIG_DIR override at all, unlike the shell that installed.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	withSecretFD(t, "REINSTATE_PASSPHRASE_FD", "daemon-test-passphrase-not-real")

	out, errb, code := a.run("daemon", "run", "--home", a.home)
	if code != ExitSafety {
		t.Fatalf("daemon run with drifted roots: exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(errb, "CLAUDE_CONFIG_DIR") || !strings.Contains(errb, "reinstate#424") || !strings.Contains(errb, "--allow-root-change") {
		t.Fatalf("refusal message: %q", errb)
	}

	// --allow-root-change permits the start: run() blocks on the loop, so
	// call applyInstalledAgentRoots directly (the same function `daemon
	// run` calls first) to prove the override actually clears the refusal
	// and pins the recorded root rather than the drifted one, without
	// standing up the whole loop.
	withSecretFD(t, "REINSTATE_PASSPHRASE_FD", "daemon-test-passphrase-not-real")
	if err := applyInstalledAgentRoots(a.home, true); err != nil {
		t.Fatalf("applyInstalledAgentRoots with allowChange=true: %v", err)
	}
	if got := os.Getenv("CLAUDE_CONFIG_DIR"); got != isolatedClaude {
		t.Fatalf("CLAUDE_CONFIG_DIR after override = %q, want the pinned %q, not the drifted login value", got, isolatedClaude)
	}
}

// TestWindowsDaemonInstallRegistersRealTaskWithPinnedRoots is the one
// Windows-native proof for #424: it drives the exact pieces `rein daemon
// install` uses — resolvedAgentRoots, daemon.WriteAgentRoots, and a real
// schtasksManager over the real schtasks.exe — against a throwaway --home
// and a harmless registered command (hostname.exe, which exits immediately
// on any arguments rather than running anything), then reads both back:
// `schtasks /Query` for the registration itself, and the persisted
// agent-roots.json for the pinned environment. It never touches a live
// agent home (CLAUDE_CONFIG_DIR/CODEX_HOME point at synthetic fixture
// paths for the whole test) and always removes the task it registers.
// Skips cleanly off Windows, without schtasks, or without permission to
// register a task (schtasks /Create refuses a UAC-filtered admin token —
// docs/hop.md, "run rein daemon install from an elevated shell").
func TestWindowsDaemonInstallRegistersRealTaskWithPinnedRoots(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("schtasks is Windows-only")
	}
	if _, err := os.Stat(`C:\Windows\System32\schtasks.exe`); err != nil {
		t.Skip("schtasks.exe not found on this host")
	}
	blankAgentRootEnv(t)
	isolatedClaude := filepath.Join(t.TempDir(), "isolated-claude-home")
	isolatedCodex := filepath.Join(t.TempDir(), "isolated-codex-home")
	t.Setenv("CLAUDE_CONFIG_DIR", isolatedClaude)
	t.Setenv("CODEX_HOME", isolatedCodex)

	home := t.TempDir()
	manager, err := daemon.NewManager(runtime.GOOS, t.TempDir(), nil) // nil runner: the real ExecRunner
	if err != nil {
		t.Fatal(err)
	}
	spec := daemon.Spec{
		Label: "com.reinstate.daemon.test-i424-" + t.Name(),
		// A harmless, throwaway action: hostname.exe exits immediately (a
		// nonzero code, "unsupported option") on any extra arguments
		// rather than running anything or waiting on input, so the task
		// Install unconditionally runs once (schtasks /Run) can never
		// hang or touch a real agent home.
		Executable: `C:\Windows\System32\hostname.exe`,
		Home:       home,
		LogPath:    filepath.Join(daemon.Dir(home), "launch.log"),
	}
	// Matches daemonSpec(): an XML whose principal names no user can be
	// refused ("Access is denied") under a UAC-filtered admin token.
	if u, err := user.Current(); err == nil {
		spec.UserID = u.Username
	}
	ctx := context.Background()
	if err := manager.Install(ctx, spec); err != nil {
		t.Skipf("schtasks /Create refused (likely needs an elevated shell; docs/hop.md): %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Uninstall(context.Background(), spec); err != nil {
			t.Errorf("cleanup: uninstall %s: %v", spec.Label, err)
		}
		state, err := manager.Status(context.Background(), spec)
		if err == nil && state.Installed {
			t.Errorf("cleanup left %s registered", spec.Label)
		}
	})

	// This is the install step's other half: record the agent-root
	// environment the (synthetic, isolated) installing shell resolved,
	// next to the same throwaway --home schtasks was just told to serve.
	roots := resolvedAgentRoots(os.Getenv)
	if err := daemon.WriteAgentRoots(home, roots); err != nil {
		t.Fatal(err)
	}

	// Read back what was registered: the real Task Scheduler entry ...
	state, err := manager.Status(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Installed {
		t.Fatalf("status after install: %+v", state)
	}

	// ... and the roots pinned alongside it.
	got, err := daemon.ReadAgentRoots(home)
	if err != nil {
		t.Fatal(err)
	}
	want := daemon.AgentRoots{"CLAUDE_CONFIG_DIR": isolatedClaude, "CODEX_HOME": isolatedCodex}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recorded roots = %#v, want %#v", got, want)
	}
}
