package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestScanPreservesRewindTo(t *testing.T) {
	t.Parallel()
	root := abs(t, testdata(t, "sessionindex", "gemini", "macos"))
	want, err := sessionindex.NewGeminiSource(root).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	src, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	got, err := src.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog source drifted from sessionindex\n got %#v\nwant %#v", got, want)
	}
}

func testdata(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	elems := append([]string{filepath.Dir(file), "..", "..", "..", "..", "testdata"}, parts...)
	return filepath.Join(elems...)
}

func abs(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// TestProjectHashResolvesToWorkspace covers a chat that records only
// projectHash, which is what Gemini writes when the session has no directory
// field. The hash is sha256 of the absolute project path and projects.json
// lists those paths, so the workspace is recoverable. Without this the record
// surfaced a bare 64-character digest as its project name and carried no
// workspace at all, so Matrix C1 could not see distinct projects and C2 had
// nothing to compare.
func TestProjectHashResolvesToWorkspace(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(root, "code", "demo")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(workspace))
	hash := hex.EncodeToString(sum[:])

	projects := map[string]any{"projects": map[string]any{workspace: "demo"}}
	body, err := json.Marshal(projects)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}

	chatDir := filepath.Join(root, "tmp", hash, "chats")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chat := map[string]any{
		"sessionId":   "01987654-3210-7890-abcd-ef0123456789",
		"projectHash": hash,
		"startTime":   "2026-08-21T00:00:00.000Z",
		"lastUpdated": "2026-08-21T00:01:00.000Z",
		"messages":    []any{},
	}
	body, err = json.Marshal(chat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chatDir, "session-1.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	got := result.Records[0]
	if got.Workspace != workspace {
		t.Fatalf("Workspace = %q, want %q", got.Workspace, workspace)
	}
	if got.Project != "demo" {
		t.Fatalf("Project = %q, want %q (a bare hash means the join did not happen)", got.Project, "demo")
	}
}
