package codex

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
)

func TestCodexDiscoverExportRestore(t *testing.T) {
	root := t.TempDir()
	sessDir := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sessDir, "rollout-syn-001.jsonl")
	content := `{"type":"session_meta","id":"rollout-syn-001","cwd":"/Users/fixture-user/code/demo"}` + "\n" +
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
	var buf bytes.Buffer
	plan, _ := a.PlanExport(context.Background(), sessions[0], adapter.ExportOptions{})
	if err := a.Export(context.Background(), plan, &buf); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	a2 := &Adapter{Root: out, Home: "/Users/fixture-user", Projects: a.Projects}
	rplan, err := a2.PlanRestore(context.Background(), adapter.Snapshot{
		Agent: "codex", SessionID: "rollout-syn-001", ProjectID: "github.com/example/demo",
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
}
