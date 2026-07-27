package codex

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
)

func TestCodexDiscoverExportRestore(t *testing.T) {
	root := t.TempDir()
	sessDir := filepath.Join(root, "sessions", "2026", "07", "25")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sessDir, "rollout-file-name.jsonl")
	content := `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"/Users/fixture-user/code/demo"}}` + "\n" +
		`{"type":"message","role":"user","content":"Synthetic fixture request"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{
		Root: root,
		Home: "/Users/fixture-user",
		Projects: map[string]string{
			"github.com/example/demo": "/Users/fixture-user/code/demo",
		},
	}
	sessions, err := a.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(sessions) != 1 {
		t.Fatalf("%+v %v", sessions, err)
	}
	if sessions[0].ID != "rollout-syn-001" || sessions[0].ProjectID != "github.com/example/demo" {
		t.Fatalf("nested session_meta was not parsed: %+v", sessions[0])
	}
	if sessions[0].RelativePath != "sessions/2026/07/25/rollout-file-name.jsonl" {
		t.Fatalf("relative path = %q", sessions[0].RelativePath)
	}
	var buf bytes.Buffer
	plan, _ := a.PlanExport(context.Background(), sessions[0], adapter.ExportOptions{})
	if err := a.Export(context.Background(), plan, &buf); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	a2 := &Adapter{Root: out, Home: "/Users/fixture-user", Projects: a.Projects}
	rplan, err := a2.PlanRestore(context.Background(), adapter.Snapshot{
		Agent: "codex", SessionID: "rollout-syn-001", ProjectID: "github.com/example/demo",
		RelativePath: sessions[0].RelativePath,
	}, adapter.RestoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a2.Restore(context.Background(), rplan, bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rplan.Session.Path); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(out, "sessions", "2026", "07", "25", "rollout-file-name.jsonl")
	if rplan.Session.Path != wantPath {
		t.Fatalf("restore path = %q, want %q", rplan.Session.Path, wantPath)
	}
}

func TestCodexDiscoverUsesCanonicalProjectMappingAndFiltersUnmapped(t *testing.T) {
	root := t.TempDir()
	sessDir := filepath.Join(root, "sessions", "2026", "07", "27")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	projectRoot := t.TempDir()
	for name, cwd := range map[string]string{
		"mapped.jsonl":   projectRoot,
		"unmapped.jsonl": filepath.Join(t.TempDir(), "other"),
	} {
		content := `{"type":"session_meta","payload":{"id":"` + name + `","cwd":"` + filepath.ToSlash(cwd) + `"}}` + "\n"
		if err := os.WriteFile(filepath.Join(sessDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a := &Adapter{
		Root: root,
		Projects: map[string]string{
			"local/reinstate-phase1-acceptance-rc6": projectRoot,
		},
	}
	sessions, err := a.Discover(context.Background(), adapter.DiscoverOptions{
		ProjectID: "local/reinstate-phase1-acceptance-rc6",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ProjectID != "local/reinstate-phase1-acceptance-rc6" {
		t.Fatalf("unexpected mapped sessions: %+v", sessions)
	}
}

func TestCodexDiscoverRejectsDuplicateResolvedProjectRoots(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	projectRoot := t.TempDir()
	a := &Adapter{
		Root: root,
		Projects: map[string]string{
			"project/one": projectRoot,
			"project/two": projectRoot,
		},
	}
	if _, err := a.Discover(context.Background(), adapter.DiscoverOptions{}); err == nil {
		t.Fatal("expected duplicate resolved project roots to be rejected")
	}
}

func TestCodexMapperNormalizesResolvedProjectRoot(t *testing.T) {
	physicalRoot := t.TempDir()
	parent := t.TempDir()
	symlinkRoot := filepath.Join(parent, "project-link")
	if err := os.Symlink(physicalRoot, symlinkRoot); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	resolvedPhysicalRoot, err := filepath.EvalSymlinks(physicalRoot)
	if err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Projects: map[string]string{"project/demo": symlinkRoot}}
	got := a.mapper().Normalize(filepath.Join(resolvedPhysicalRoot, "src", "main.go"))
	if got != "${REPO:project/demo}/src/main.go" {
		t.Fatalf("resolved root normalized as %q", got)
	}
}

func TestCommittedPlatformFixturesDiscover(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "..", "testdata", "adapters", "codex")
	expectedProject := map[string]string{
		"macos":   "/Users/fixture-user/code/demo",
		"windows": `C:\Users\fixture-user\code\demo`,
		"wsl":     "/home/fixture-user/code/demo",
	}
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
			if sessions[0].ID != "rollout-syn-001" || sessions[0].ProjectID != expectedProject[platform] {
				t.Fatalf("unexpected fixture session: %+v", sessions[0])
			}
		})
	}
}

func TestCodexSupportedVersionRange(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{version: "0.132.9", want: false},
		{version: "0.133.0", want: true},
		{version: "0.140.0", want: true},
		{version: "0.145.0", want: true},
		{version: "0.145.1", want: false},
		{version: "0.146.0", want: false},
		{version: "0.145.0-beta.1", want: false},
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

func TestCodexRestoreRejectsUnexpectedArchiveEntry(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	var artifact bytes.Buffer
	tw := tar.NewWriter(&artifact)
	body := []byte("{}\n")
	if err := tw.WriteHeader(&tar.Header{Name: "auth.json", Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Root: root}
	plan, err := a.PlanRestore(context.Background(), adapter.Snapshot{
		SessionID: "s", RelativePath: "sessions/2026/07/25/s.jsonl",
	}, adapter.RestoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Restore(context.Background(), plan, bytes.NewReader(artifact.Bytes())); err == nil {
		t.Fatal("expected unexpected archive entry rejection")
	}
	if _, err := os.Stat(plan.Session.Path); !os.IsNotExist(err) {
		t.Fatalf("restore mutated destination: %v", err)
	}
}

func TestCodexKeepBothRewritesStructuralSessionID(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "sessions", "2026", "07", "25")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(sourceDir, "source.jsonl")
	if err := os.WriteFile(sourcePath, []byte(
		`{"type":"session_meta","payload":{"id":"source-id","cwd":"/tmp/project"}}`+"\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	source := &Adapter{Root: root}
	sessions, err := source.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(sessions) != 1 {
		t.Fatalf("sessions=%v err=%v", sessions, err)
	}
	var artifact bytes.Buffer
	plan, err := source.PlanExport(context.Background(), sessions[0], adapter.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Export(context.Background(), plan, &artifact); err != nil {
		t.Fatal(err)
	}

	targetRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetRoot, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := &Adapter{Root: targetRoot}
	restorePlan, err := target.PlanRestore(context.Background(), adapter.Snapshot{
		SessionID: "source-id", RelativePath: "sessions/2026/07/25/source.jsonl",
	}, adapter.RestoreOptions{
		ForkSessionID:           "source-id-remote-deadbeef",
		DestinationRelativePath: "sessions/2026/07/25/source-id-remote-deadbeef.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Restore(context.Background(), restorePlan, bytes.NewReader(artifact.Bytes())); err != nil {
		t.Fatal(err)
	}
	restored, err := target.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(restored) != 1 {
		t.Fatalf("sessions=%v err=%v", restored, err)
	}
	if restored[0].ID != "source-id-remote-deadbeef" {
		t.Fatalf("fork id was not rewritten: %+v", restored[0])
	}
}

func TestCodexRejectsUnsafeSnapshotPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Root: root}
	_, err := a.PlanRestore(context.Background(), adapter.Snapshot{
		SessionID: "s", RelativePath: "sessions/../../auth.json",
	}, adapter.RestoreOptions{})
	if err == nil {
		t.Fatal("expected unsafe snapshot path rejection")
	}
}

func TestCodexDetectEmptyRootIsUntested(t *testing.T) {
	root := t.TempDir()
	a := &Adapter{Root: root}
	_, compat, err := a.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if compat != adapter.CompatibilityUntested {
		t.Fatalf("empty/unknown Codex layout reported %s, want UNTESTED", compat)
	}
}

func TestCodexDefaultRootWithoutVerifiedBinaryIsUntested(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", "")
	if err := os.MkdirAll(filepath.Join(home, ".codex", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, compatibility, err := (&Adapter{}).Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if compatibility != adapter.CompatibilityUntested {
		t.Fatalf("unverified Codex binary reported %s", compatibility)
	}
}

func TestCodexTransformLeavesPathLikeTranscriptContentUntouched(t *testing.T) {
	input := []byte(
		`{"type":"session_meta","payload":{"cwd":"/Users/fixture-user/code/demo"}}` + "\n" +
			`{"type":"message","content":"/Users/fixture-user/code/demo"}` + "\n",
	)
	out := transformJSONL(input, func(string) string { return "${REPO:demo}" })
	if !bytes.Contains(out, []byte(`"cwd":"${REPO:demo}"`)) {
		t.Fatalf("known structural cwd was not transformed: %s", out)
	}
	if !bytes.Contains(out, []byte(`"content":"/Users/fixture-user/code/demo"`)) {
		t.Fatalf("path-like transcript prose was mutated: %s", out)
	}
}
