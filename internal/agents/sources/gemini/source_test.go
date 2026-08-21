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
	"strings"
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

// writeGeminiChat lays out tmp/<dir>/chats/session-1.jsonl carrying only a
// projectHash, which is all current Gemini records on a chat.
func writeGeminiChat(t *testing.T, root, dir, hash string) {
	t.Helper()
	chatDir := filepath.Join(root, "tmp", dir, "chats")
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
	body, err := json.Marshal(chat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chatDir, "session-1.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanOne(t *testing.T, root string) sessionindex.Record {
	t.Helper()
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
	return result.Records[0]
}

// TestProjectRootMarkerResolvesWorkspace covers the layout current Gemini
// writes: the session directory is named for the project rather than its
// digest, and the absolute path is recorded beside the chats in
// .project_root. Without reading that marker the join depends entirely on
// projects.json, so a pruned or stale index leaves a bare digest as the
// project name.
func TestProjectRootMarkerResolvesWorkspace(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(root, "code", "demo")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(workspace))
	hash := hex.EncodeToString(sum[:])

	// No projects.json at all: the marker alone must carry the join.
	if err := os.MkdirAll(filepath.Join(root, "tmp", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tmp", "demo", ".project_root"),
		[]byte(workspace+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeGeminiChat(t, root, "demo", hash)

	got := scanOne(t, root)
	if got.Workspace != workspace {
		t.Fatalf("Workspace = %q, want %q", got.Workspace, workspace)
	}
	if got.Project != "demo" {
		t.Fatalf("Project = %q, want %q (a bare hash means the marker was not read)", got.Project, "demo")
	}
}

// TestProjectHashResolvesFromDifferentlyCasedSpelling covers what Gemini does
// on a case-insensitive filesystem: it records a lower-cased path in
// projects.json and .project_root, but hashes the path in its real on-disk
// case. Joining on the recorded spelling alone therefore never matches, which
// is why every Windows Gemini record carried an empty workspace and a bare
// digest for a project.
func TestProjectHashResolvesFromDifferentlyCasedSpelling(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(root, "Code", "Demo")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	lowered := filepath.Join(root, "code", "demo")
	if _, err := os.Stat(lowered); err != nil {
		t.Skip("filesystem is case-sensitive; Gemini's case drift cannot occur here")
	}

	// Gemini hashes the real case...
	sum := sha256.Sum256([]byte(workspace))
	hash := hex.EncodeToString(sum[:])
	// ...but writes the lower-cased spelling down.
	if err := os.MkdirAll(filepath.Join(root, "tmp", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tmp", "demo", ".project_root"),
		[]byte(lowered), 0o644); err != nil {
		t.Fatal(err)
	}
	writeGeminiChat(t, root, "demo", hash)

	got := scanOne(t, root)
	if got.Workspace == "" {
		t.Fatal("Workspace is empty: the recorded spelling was not case-corrected before hashing")
	}
	if !strings.EqualFold(got.Workspace, lowered) {
		t.Fatalf("Workspace = %q, want a spelling of %q", got.Workspace, lowered)
	}
	if !strings.EqualFold(got.Project, "demo") {
		t.Fatalf("Project = %q, want %q", got.Project, "demo")
	}
}
