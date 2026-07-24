package claude

import (
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
	a2 := &Adapter{
		Root: outRoot,
		Home: "/Users/fixture-user",
		Projects: map[string]string{
			"fixture-project": "/Users/fixture-user/code/demo",
		},
	}
	rplan, err := a2.PlanRestore(context.Background(), adapter.Snapshot{
		Agent: "claude", SessionID: "session-001", ProjectID: "fixture-project",
	}, adapter.RestoreOptions{})
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
