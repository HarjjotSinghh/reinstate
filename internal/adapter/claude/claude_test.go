package claude

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
)

func TestClaudeDiscoverExportRestore(t *testing.T) {
	// use synthetic fixture tree
	root := t.TempDir()
	proj := filepath.Join(root, "projects", "fixture-project")
	if err := os.MkdirAll(proj, 0o700); err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(proj, "session-001.jsonl")
	content := `{"type":"meta","cwd":"/Users/fixture-user/code/demo"}` + "\n" +
		`{"type":"user","message":{"content":"hello prose not a path"}}` + "\n"
	if err := os.WriteFile(session, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{
		Root: root,
		Home: "/Users/fixture-user",
		Projects: map[string]string{
			"fixture-project": "/Users/fixture-user/code/demo",
		},
	}
	inst, compat, err := a.Detect(context.Background())
	if err != nil || compat != adapter.CompatibilitySupported || inst.Root != root {
		t.Fatalf("%+v %s %v", inst, compat, err)
	}
	sessions, err := a.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(sessions) != 1 {
		t.Fatalf("%+v %v", sessions, err)
	}
	if sessions[0].RelativePath != "projects/fixture-project/session-001.jsonl" {
		t.Fatalf("relative path = %q", sessions[0].RelativePath)
	}
	plan, err := a.PlanExport(context.Background(), sessions[0], adapter.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := a.Export(context.Background(), plan, &buf); err != nil {
		t.Fatal(err)
	}
	// tar is binary; path rewrite is asserted after restore below.
	_ = buf
	// restore into new root
	outRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outRoot, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	a2 := &Adapter{
		Root: outRoot,
		Home: "/Users/fixture-user",
		Projects: map[string]string{
			"fixture-project": "/Users/fixture-user/code/demo",
		},
	}
	rplan, err := a2.PlanRestore(context.Background(), adapter.Snapshot{
		Agent: "claude", SessionID: "session-001", ProjectID: "fixture-project",
		RelativePath: sessions[0].RelativePath,
	}, adapter.RestoreOptions{BackupRoot: filepath.Join(outRoot, "reinstate-backups")})
	if err != nil || rplan.Refuse != "" {
		t.Fatalf("%+v %v", rplan, err)
	}
	if err := a2.Restore(context.Background(), rplan, bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(rplan.Session.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(restored, []byte("hello prose not a path")) {
		t.Fatalf("prose lost: %s", restored)
	}
	wantPath := filepath.Join(outRoot, "projects", "fixture-project", "session-001.jsonl")
	if rplan.Session.Path != wantPath {
		t.Fatalf("restore path = %q, want %q", rplan.Session.Path, wantPath)
	}
}

func TestCommittedPlatformFixturesDiscover(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "..", "testdata", "adapters", "claude")
	for _, platform := range []string{"macos", "windows", "wsl"} {
		t.Run(platform, func(t *testing.T) {
			root := filepath.Join(fixtureRoot, platform)
			adapterUnderTest := &Adapter{Root: root}
			sessions, err := adapterUnderTest.Discover(context.Background(), adapter.DiscoverOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(sessions) != 1 {
				t.Fatalf("got %d sessions", len(sessions))
			}
			if sessions[0].ID != "session-syn-001" ||
				sessions[0].RelativePath != "projects/fixture-project/session-syn-001.jsonl" {
				t.Fatalf("unexpected fixture session: %+v", sessions[0])
			}
		})
	}
}

func TestClaudeSupportedVersionRange(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{version: "2.1.218", want: false},
		{version: "2.1.219", want: true},
		{version: "2.1.220", want: true},
		{version: "2.1.221", want: false},
		{version: "2.2.0", want: false},
		{version: "2.1.220-beta.1", want: false},
		{version: "not-a-version", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := isSupportedVersion(tt.version); got != tt.want {
				t.Fatalf("isSupportedVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestClaudeDiscoverSkipsSubagentArtifacts(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "projects", "fixture-project")
	if err := os.MkdirAll(filepath.Join(project, "session-main", "subagents", "workflows", "wf-1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "projects", "subagents"), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(project, "session-main.jsonl"):                                            `{"type":"user"}` + "\n",
		filepath.Join(project, "session-main", "subagents", "agent-worker.jsonl"):               `{"type":"user"}` + "\n",
		filepath.Join(project, "session-main", "subagents", "workflows", "wf-1", "agent.jsonl"): `{"type":"user"}` + "\n",
		filepath.Join(root, "projects", "subagents", "project-session.jsonl"):                   `{"type":"user"}` + "\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := (&Adapter{Root: root}).Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("discovered %d sessions, want two top-level sessions: %+v", len(sessions), sessions)
	}
	got := map[string]bool{}
	for _, session := range sessions {
		got[session.ID] = true
	}
	for _, expected := range []string{"session-main", "project-session"} {
		if !got[expected] {
			t.Fatalf("missing top-level session %q: %+v", expected, sessions)
		}
	}
}

func TestClaudeRestoreBacksUpExistingSession(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects", "p"), 0o700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "projects", "p", "s.jsonl")
	if err := os.WriteFile(dest, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var artifact bytes.Buffer
	tw := tar.NewWriter(&artifact)
	body := []byte("{\"type\":\"user\",\"message\":{\"content\":\"new\"}}\n")
	if err := tw.WriteHeader(&tar.Header{Name: "projects/p/s.jsonl", Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	backupRoot := filepath.Join(root, "backups")
	a := &Adapter{Root: root}
	plan, err := a.PlanRestore(context.Background(), adapter.Snapshot{
		SessionID: "s", ProjectID: "p", RelativePath: "projects/p/s.jsonl",
	}, adapter.RestoreOptions{BackupRoot: backupRoot})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Restore(context.Background(), plan, bytes.NewReader(artifact.Bytes())); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(filepath.Join(backupRoot, "*", "projects", "p", "s.jsonl"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups=%v err=%v", backups, err)
	}
	got, err := os.ReadFile(backups[0])
	if err != nil || string(got) != "old\n" {
		t.Fatalf("backup=%q err=%v", got, err)
	}
}

func TestClaudeRejectsUnsafeSnapshotPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Root: root}
	_, err := a.PlanRestore(context.Background(), adapter.Snapshot{
		SessionID: "s", ProjectID: "p", RelativePath: "../../auth.json",
	}, adapter.RestoreOptions{})
	if err == nil {
		t.Fatal("expected unsafe snapshot path rejection")
	}
}

func TestUntestedRefusesRestore(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "projects"), 0o700)
	a := &Adapter{Root: root, ForceCompat: adapter.CompatibilityUntested}
	plan, err := a.PlanRestore(context.Background(), adapter.Snapshot{SessionID: "s", ProjectID: "p"}, adapter.RestoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Refuse == "" {
		t.Fatal("expected refuse")
	}
}

func TestClaudeDetectEmptyRootIsUntested(t *testing.T) {
	root := t.TempDir()
	a := &Adapter{Root: root}
	_, compat, err := a.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if compat != adapter.CompatibilityUntested {
		t.Fatalf("empty/unknown Claude layout reported %s, want UNTESTED", compat)
	}
}

func TestClaudeDefaultRootUnknownVersionIsUntested(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", "")
	root := filepath.Join(home, ".claude")
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "version"), []byte("9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, compatibility, err := (&Adapter{}).Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if compatibility != adapter.CompatibilityUntested {
		t.Fatalf("unknown version reported %s", compatibility)
	}
}

func TestClaudeDefaultRootSupportedVersionFileDoesNotRequireExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", "")
	root := filepath.Join(home, ".claude")
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "version"), []byte("2.1.220\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	install, compatibility, err := (&Adapter{}).Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if compatibility != adapter.CompatibilitySupported {
		t.Fatalf("supported version file reported %s", compatibility)
	}
	if install.Version != "2.1.220" {
		t.Fatalf("detected version %q", install.Version)
	}
}

func TestClaudeTransformLeavesPathLikeTranscriptContentUntouched(t *testing.T) {
	input := []byte(
		`{"type":"meta","cwd":"/Users/fixture-user/code/demo"}` + "\n" +
			`{"type":"assistant","message":{"content":"/Users/fixture-user/code/demo"}}` + "\n",
	)
	out := transformJSONL(input, func(string) string { return "${REPO:demo}" })
	if !bytes.Contains(out, []byte(`"cwd":"${REPO:demo}"`)) {
		t.Fatalf("known structural cwd was not transformed: %s", out)
	}
	if !bytes.Contains(out, []byte(`"content":"/Users/fixture-user/code/demo"`)) {
		t.Fatalf("path-like transcript prose was mutated: %s", out)
	}
}
