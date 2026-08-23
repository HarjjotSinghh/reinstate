package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var goldenSpec = Spec{
	Label:      "com.reinstate.daemon",
	Executable: "/usr/local/bin/rein",
	Args:       []string{"daemon", "run", "--pull-every", "5m"},
	Home:       "/Users/fixture-user/.reinstate",
	LogPath:    "/Users/fixture-user/.reinstate/daemon/service.log",
	Path:       "/usr/local/bin:/usr/bin:/bin",
}

var windowsSpec = Spec{
	Label:      "com.reinstate.daemon",
	Executable: `C:\Program Files\Reinstate\rein.exe`,
	Args:       []string{"daemon", "run", "--pull-every", "5m"},
	Home:       `C:\Users\fixture-user\.reinstate`,
	LogPath:    `C:\Users\fixture-user\.reinstate\daemon\service.log`,
	Path:       `C:\Windows\System32`,
	UserID:     `HARJOTS-PC\fixture-user`,
}

func TestRenderGolden(t *testing.T) {
	cases := []struct {
		goos, golden string
		spec         Spec
	}{
		{"darwin", "launchd.plist", goldenSpec},
		{"linux", "systemd.service", goldenSpec},
		{"windows", "schtasks.xml", windowsSpec},
	}
	for _, c := range cases {
		t.Run(c.goos, func(t *testing.T) {
			m, err := NewManager(c.goos, "/Users/fixture-user", nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := m.Render(c.spec)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("..", "..", "testdata", "daemon", c.golden)
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("%s differs from golden %s:\n%s", c.goos, path, got)
			}
		})
	}
}

func TestRenderEscapesAndQuotes(t *testing.T) {
	spec := goldenSpec
	spec.Home = `/Users/fixture-user/al "ice"/<home>&`
	m, _ := NewManager("darwin", "/Users/fixture-user", nil)
	plist, _ := m.Render(spec)
	if !strings.Contains(string(plist), `/Users/fixture-user/al &#34;ice&#34;/&lt;home&gt;&amp;`) {
		t.Fatalf("plist did not escape the home:\n%s", plist)
	}
	m, _ = NewManager("linux", "/Users/fixture-user", nil)
	unit, _ := m.Render(spec)
	if !strings.Contains(string(unit), `Environment=REINSTATE_HOME="/Users/fixture-user/al \"ice\"/<home>&"`) {
		t.Fatalf("unit did not quote the home:\n%s", unit)
	}
	m, _ = NewManager("windows", "", nil)
	spec.Home = `C:\Users\fixture-user\Al Ice\.reinstate`
	task, _ := m.Render(spec)
	if !strings.Contains(string(task), `<Arguments>daemon run --pull-every 5m --home &#34;C:\Users\fixture-user\Al Ice\.reinstate&#34;</Arguments>`) {
		t.Fatalf("task did not quote the home:\n%s", task)
	}
}

func TestLabelFor(t *testing.T) {
	def := "/Users/fixture-user/.reinstate"
	if got := LabelFor(def, def); got != DefaultLabel {
		t.Fatalf("default home: %s", got)
	}
	if got := LabelFor("", def); got != DefaultLabel {
		t.Fatalf("empty home: %s", got)
	}
	other := LabelFor("/tmp/other-home", def)
	if !strings.HasPrefix(other, DefaultLabel+".") || len(other) != len(DefaultLabel)+9 {
		t.Fatalf("custom home label: %s", other)
	}
	if other != LabelFor("/tmp/other-home/", def) {
		t.Fatal("label must not depend on a trailing slash")
	}
}

// recordingRunner records commands and answers from a script keyed by the
// command's first two words.
type recordingRunner struct {
	calls   []string
	answers map[string]string
	fail    map[string]bool
}

func (r *recordingRunner) run(_ context.Context, name string, args ...string) (string, error) {
	call := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	key := name
	if len(args) > 0 {
		key += " " + args[0]
	}
	if len(args) > 1 && name == "systemctl" {
		key += " " + args[1]
	}
	if r.fail[key] {
		return "", &execError{key}
	}
	return r.answers[key], nil
}

type execError struct{ what string }

func (e *execError) Error() string { return e.what + ": exit status 1: No such process" }

func TestLaunchdLifecycleCommands(t *testing.T) {
	userHome := t.TempDir()
	runner := &recordingRunner{answers: map[string]string{"launchctl print": "\tstate = running\n\tpid = 4242\n\tendpoints = {\n\t\tstate = active\n\t}\n"}}
	m, _ := NewManager("darwin", userHome, runner.run)
	spec := goldenSpec
	ctx := context.Background()
	if err := m.Install(ctx, spec); err != nil {
		t.Fatal(err)
	}
	plist := filepath.Join(userHome, "Library", "LaunchAgents", spec.Label+".plist")
	if _, err := os.Stat(plist); err != nil {
		t.Fatalf("plist not written: %v", err)
	}
	state, err := m.Status(ctx, spec)
	if err != nil || !state.Installed || !state.Running || state.Detail != "running" {
		t.Fatalf("status: %+v err=%v", state, err)
	}
	if err := m.Stop(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if err := m.Uninstall(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plist); !os.IsNotExist(err) {
		t.Fatal("plist should be removed")
	}
	want := []string{
		"launchctl bootout gui/",
		"launchctl bootstrap gui/",
		"launchctl print gui/",
		"launchctl bootout gui/",
		"launchctl print gui/",
		"launchctl kickstart -k gui/",
		"launchctl bootout gui/",
	}
	if len(runner.calls) != len(want) {
		t.Fatalf("calls: %q", runner.calls)
	}
	for i, prefix := range want {
		if !strings.HasPrefix(runner.calls[i], prefix) {
			t.Fatalf("call %d = %q, want prefix %q", i, runner.calls[i], prefix)
		}
	}
	// Start on a machine where the plist is gone is refused with guidance.
	if err := m.Start(ctx, spec); err == nil || !strings.Contains(err.Error(), "rein daemon install") {
		t.Fatalf("start without plist: %v", err)
	}
}

func TestSystemdLifecycleCommands(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	runner := &recordingRunner{answers: map[string]string{"systemctl --user is-active": "active\n"}}
	m, _ := NewManager("linux", userHome, runner.run)
	spec := goldenSpec
	ctx := context.Background()
	if err := m.Install(ctx, spec); err != nil {
		t.Fatal(err)
	}
	unit := filepath.Join(userHome, ".config", "systemd", "user", spec.Label+".service")
	if _, err := os.Stat(unit); err != nil {
		t.Fatalf("unit not written: %v", err)
	}
	state, _ := m.Status(ctx, spec)
	if !state.Installed || !state.Running {
		t.Fatalf("status: %+v", state)
	}
	if err := m.Uninstall(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(unit); !os.IsNotExist(err) {
		t.Fatal("unit should be removed")
	}
	joined := strings.Join(runner.calls, "\n")
	for _, want := range []string{"systemctl --user daemon-reload", "systemctl --user enable --now com.reinstate.daemon.service", "systemctl --user disable --now com.reinstate.daemon.service"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
}

func TestSchtasksLifecycleCommands(t *testing.T) {
	runner := &recordingRunner{answers: map[string]string{"schtasks /Query": "HostName: PC\r\nTaskName: \\com.reinstate.daemon\r\nStatus: Running\r\n"}}
	m, _ := NewManager("windows", "", runner.run)
	spec := windowsSpec
	ctx := context.Background()
	if err := m.Install(ctx, spec); err != nil {
		t.Fatal(err)
	}
	state, _ := m.Status(ctx, spec)
	if !state.Installed || !state.Running || state.Detail != "Running" {
		t.Fatalf("status: %+v", state)
	}
	if err := m.Uninstall(ctx, spec); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(runner.calls, "\n")
	for _, want := range []string{"schtasks /Create /TN com.reinstate.daemon /XML ", "schtasks /Run /TN com.reinstate.daemon", "schtasks /End /TN com.reinstate.daemon", "schtasks /Delete /TN com.reinstate.daemon /F"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	runner.fail = map[string]bool{"schtasks /Query": true}
	if state, _ := m.Status(ctx, spec); state.Installed {
		t.Fatalf("unregistered task reported installed: %+v", state)
	}
}

func TestUTF16LE(t *testing.T) {
	got := utf16LE([]byte("<A>"))
	want := []byte{0xFF, 0xFE, '<', 0, 'A', 0, '>', 0}
	if string(got) != string(want) {
		t.Fatalf("got % x", got)
	}
}

func TestUnsupportedOS(t *testing.T) {
	if _, err := NewManager("plan9", "", nil); err != ErrUnsupportedOS {
		t.Fatalf("err=%v", err)
	}
}
