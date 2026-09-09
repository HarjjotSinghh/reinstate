package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

func TestResolveAgentRoots(t *testing.T) {
	cases := []struct {
		name   string
		lookup map[string]string // absent key -> getenv returns ""
		names  []string
		want   AgentRoots
	}{
		{
			name:   "all set",
			lookup: map[string]string{"CLAUDE_CONFIG_DIR": "/iso/claude", "CODEX_HOME": "/iso/codex"},
			names:  []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME"},
			want:   AgentRoots{"CLAUDE_CONFIG_DIR": "/iso/claude", "CODEX_HOME": "/iso/codex"},
		},
		{
			name:   "some unset",
			lookup: map[string]string{"CLAUDE_CONFIG_DIR": "/iso/claude"},
			names:  []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "XDG_DATA_HOME"},
			want:   AgentRoots{"CLAUDE_CONFIG_DIR": "/iso/claude"},
		},
		{
			// Set-to-blank and unset must resolve identically: every other
			// reader in the codebase treats os.Getenv's "" the same way
			// whether the variable was never set or was set to blank/
			// whitespace (adapter/claude, adapter/codex, adapter/opencode,
			// preflight all use `strings.TrimSpace(os.Getenv(...)) != ""`).
			name:   "set blank does not count as set",
			lookup: map[string]string{"CODEX_HOME": "   "},
			names:  []string{"CODEX_HOME"},
			want:   AgentRoots{},
		},
		{
			name:   "none set",
			lookup: map[string]string{},
			names:  []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME"},
			want:   AgentRoots{},
		},
		{
			name:   "no names",
			lookup: map[string]string{"CLAUDE_CONFIG_DIR": "/iso/claude"},
			names:  nil,
			want:   AgentRoots{},
		},
		{
			name:   "value is trimmed",
			lookup: map[string]string{"CLAUDE_CONFIG_DIR": "  /iso/claude  "},
			names:  []string{"CLAUDE_CONFIG_DIR"},
			want:   AgentRoots{"CLAUDE_CONFIG_DIR": "/iso/claude"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			getenv := func(name string) string { return c.lookup[name] }
			got := ResolveAgentRoots(c.names, getenv)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v, want %#v", got, c.want)
			}
		})
	}
}

func TestWriteReadAgentRoots(t *testing.T) {
	home := t.TempDir()
	if _, err := ReadAgentRoots(home); !errors.Is(err, ErrNoAgentRoots) {
		t.Fatalf("unwritten home: err=%v, want ErrNoAgentRoots", err)
	}
	roots := AgentRoots{"CLAUDE_CONFIG_DIR": `D:\iso\claude`, "CODEX_HOME": `D:\iso\codex`}
	if err := WriteAgentRoots(home, roots); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAgentRoots(home)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, roots) {
		t.Fatalf("got %#v, want %#v", got, roots)
	}
	// The file is owner-only, like status.json.
	info, err := os.Stat(AgentRootsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
	// An empty map still round-trips as recorded (distinct from never
	// having installed at all).
	if err := WriteAgentRoots(home, AgentRoots{}); err != nil {
		t.Fatal(err)
	}
	got, err = ReadAgentRoots(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
	if err := RemoveAgentRoots(home); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAgentRoots(home); !errors.Is(err, ErrNoAgentRoots) {
		t.Fatalf("after remove: err=%v, want ErrNoAgentRoots", err)
	}
	// Removing an already-absent file is not an error.
	if err := RemoveAgentRoots(home); err != nil {
		t.Fatalf("remove of already-removed: %v", err)
	}
}

func TestReadAgentRootsCorrupt(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(Dir(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(AgentRootsPath(home), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAgentRoots(home); err == nil || errors.Is(err, ErrNoAgentRoots) {
		t.Fatalf("corrupt file: err=%v, want a decode error", err)
	}
}

func TestCompareAgentRoots(t *testing.T) {
	cases := []struct {
		name             string
		recorded, actual AgentRoots
		wantNames        []string
	}{
		{
			name:     "identical",
			recorded: AgentRoots{"CLAUDE_CONFIG_DIR": "/a"},
			actual:   AgentRoots{"CLAUDE_CONFIG_DIR": "/a"},
		},
		{
			name:     "both empty",
			recorded: AgentRoots{},
			actual:   AgentRoots{},
		},
		{
			name:      "value changed",
			recorded:  AgentRoots{"CLAUDE_CONFIG_DIR": "/a"},
			actual:    AgentRoots{"CLAUDE_CONFIG_DIR": "/b"},
			wantNames: []string{"CLAUDE_CONFIG_DIR"},
		},
		{
			name:      "recorded set, now unset (the #424 scenario: install captured an isolated home, the login environment has none)",
			recorded:  AgentRoots{"CLAUDE_CONFIG_DIR": "/iso"},
			actual:    AgentRoots{},
			wantNames: []string{"CLAUDE_CONFIG_DIR"},
		},
		{
			name:      "recorded unset, now set",
			recorded:  AgentRoots{},
			actual:    AgentRoots{"CODEX_HOME": "/new"},
			wantNames: []string{"CODEX_HOME"},
		},
		{
			name:      "multiple variables, only some differ",
			recorded:  AgentRoots{"CLAUDE_CONFIG_DIR": "/a", "CODEX_HOME": "/x"},
			actual:    AgentRoots{"CLAUDE_CONFIG_DIR": "/a", "CODEX_HOME": "/y"},
			wantNames: []string{"CODEX_HOME"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diffs := CompareAgentRoots(c.recorded, c.actual)
			got := []string{}
			for _, d := range diffs {
				got = append(got, d.Name)
			}
			sort.Strings(got)
			want := append([]string{}, c.wantNames...)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("diff names = %v, want %v (diffs=%#v)", got, want, diffs)
			}
		})
	}
}

func TestCompareAgentRootsDiffFields(t *testing.T) {
	diffs := CompareAgentRoots(AgentRoots{"CLAUDE_CONFIG_DIR": "/iso"}, AgentRoots{})
	if len(diffs) != 1 {
		t.Fatalf("diffs = %#v", diffs)
	}
	d := diffs[0]
	if d.Name != "CLAUDE_CONFIG_DIR" || d.Recorded != "/iso" || d.Current != "" {
		t.Fatalf("diff = %#v", d)
	}
}

func TestApplyAgentRoots(t *testing.T) {
	roots := AgentRoots{"CLAUDE_CONFIG_DIR": "/pinned/claude", "CODEX_HOME": "/pinned/codex"}
	names := []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME"}
	applied := map[string]string{}
	setenv := func(name, value string) error {
		applied[name] = value
		return nil
	}
	unsetenv := func(name string) error {
		t.Fatalf("unsetenv(%q) called, but every name in names is present in roots", name)
		return nil
	}
	if err := ApplyAgentRoots(roots, names, setenv, unsetenv); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(applied, map[string]string(roots)) {
		t.Fatalf("applied = %#v, want %#v", applied, roots)
	}
}

// TestApplyAgentRootsUnsetsNamesAbsentFromRoots is the regression this fix
// exists for: a catalog variable that was unset at install time (absent
// from roots) must be force-unset, not left as whatever the calling
// process's own environment currently carries for it. Without this, a
// variable nobody customized at install could pick up drift from the
// daemon's own launch environment under --allow-root-change, reopening the
// #424 mechanism from the opposite direction (unset -> set instead of
// set -> unset).
func TestApplyAgentRootsUnsetsNamesAbsentFromRoots(t *testing.T) {
	roots := AgentRoots{"CLAUDE_CONFIG_DIR": "/pinned/claude"}
	// CODEX_HOME was never customized at install (absent from roots) but
	// the calling process currently has it set to something roots never
	// saw — exactly the drift ApplyAgentRoots must erase, not adopt.
	names := []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME"}
	applied := map[string]string{}
	unset := map[string]bool{}
	setenv := func(name, value string) error {
		applied[name] = value
		return nil
	}
	unsetenv := func(name string) error {
		unset[name] = true
		return nil
	}
	if err := ApplyAgentRoots(roots, names, setenv, unsetenv); err != nil {
		t.Fatal(err)
	}
	if applied["CLAUDE_CONFIG_DIR"] != "/pinned/claude" {
		t.Fatalf("applied = %#v, want CLAUDE_CONFIG_DIR pinned", applied)
	}
	if !unset["CODEX_HOME"] {
		t.Fatalf("unset = %#v, want CODEX_HOME force-unset (it was absent from roots)", unset)
	}
	if _, ok := applied["CODEX_HOME"]; ok {
		t.Fatalf("applied = %#v, CODEX_HOME must never be setenv'd — it was absent from roots", applied)
	}
}

func TestApplyAgentRootsPropagatesError(t *testing.T) {
	roots := AgentRoots{"CLAUDE_CONFIG_DIR": "/a"}
	names := []string{"CLAUDE_CONFIG_DIR"}
	wantErr := errors.New("setenv refused")
	err := ApplyAgentRoots(roots, names, func(string, string) error { return wantErr }, func(string) error { return nil })
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wrapping %v", err, wantErr)
	}
}

// TestApplyAgentRootsPropagatesUnsetError checks the unset side of the same
// error contract: a failing unsetenv (a name absent from roots) must also
// surface, not be swallowed.
func TestApplyAgentRootsPropagatesUnsetError(t *testing.T) {
	names := []string{"CODEX_HOME"}
	wantErr := errors.New("unsetenv refused")
	err := ApplyAgentRoots(AgentRoots{}, names, func(string, string) error { return nil }, func(string) error { return wantErr })
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wrapping %v", err, wantErr)
	}
}

// TestApplyAgentRootsRealSetenv exercises the intended call shape
// (os.Setenv/os.Unsetenv) end to end, restoring the environment afterward:
// one recorded variable is pinned to its value, one unrecorded-but-currently
// -set variable is force-unset.
func TestApplyAgentRootsRealSetenv(t *testing.T) {
	pinned := "REINSTATE_TEST_AGENT_ROOT_VAR"
	drifted := "REINSTATE_TEST_AGENT_ROOT_DRIFT_VAR"
	t.Setenv(pinned, "")
	t.Setenv(drifted, "/drifted/from/launch/env")
	roots := AgentRoots{pinned: "/pinned"}
	names := []string{pinned, drifted}
	if err := ApplyAgentRoots(roots, names, os.Setenv, os.Unsetenv); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(pinned); got != "/pinned" {
		t.Fatalf("got %q", got)
	}
	if got, ok := os.LookupEnv(drifted); ok {
		t.Fatalf("drifted var still set to %q, want unset (it was absent from roots)", got)
	}
}

func TestAgentRootsPath(t *testing.T) {
	home := filepath.Join("D:", "iso", "home")
	got := AgentRootsPath(home)
	want := filepath.Join(home, "daemon", "agent-roots.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
