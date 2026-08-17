package kimi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func scan(t *testing.T, root string) sessionindex.ScanResult {
	t.Helper()
	source, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func fixture(t *testing.T, osName string) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "kimi", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName        string
		wantID        string
		wantWorkspace string
		wantProject   string
		wantFiles     []string
	}{
		{
			osName:        "macos",
			wantID:        "session_01987654-3210-7890-abcd-ef0123456789",
			wantWorkspace: "/Users/fixture-user/code/demo",
			wantProject:   "demo",
			wantFiles:     []string{"internal/agentcheck/agent.go"},
		},
		{
			osName:        "windows",
			wantID:        "session_01912345-6789-7abc-def0-123456789abc",
			wantWorkspace: `C:\Users\fixture-user\code\demo`,
			wantProject:   "demo",
			// Separators are kept as the agent recorded them. The shared
			// NormalizeFiles does not rewrite them for any agent, and
			// internal/pathmap owns cross-OS rewriting.
			wantFiles: []string{`internal\pathmap\rewrite.go`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentKimi || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Key != sessionindex.CompositeReference(sessionindex.AgentKimi, tt.wantID) {
				t.Fatalf("key = %q", record.Key)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q, want %q", record.Workspace, tt.wantWorkspace)
			}
			if record.Project != tt.wantProject {
				t.Fatalf("project = %q, want %q", record.Project, tt.wantProject)
			}
			if !reflect.DeepEqual(record.Files, tt.wantFiles) {
				t.Fatalf("files = %v, want %v", record.Files, tt.wantFiles)
			}
			if record.PromptPreview == "" || record.Title == "" {
				t.Fatalf("preview = %q title = %q", record.PromptPreview, record.Title)
			}
			if record.UpdatedAt.IsZero() {
				t.Fatal("updated_at is zero; state.json carries it")
			}
			if record.CanResume || record.CanFork {
				t.Fatalf("capabilities resume=%t fork=%t, want false", record.CanResume, record.CanFork)
			}
			if record.ReadOnlyReason != sessionindex.KimiReadOnlyReason {
				t.Fatalf("read_only_reason = %q", record.ReadOnlyReason)
			}
		})
	}
}

// The size and mtime must follow the append-only log, because state.json is
// rewritten for metadata-only changes such as a rename.
func TestAuthorityIsTheWireLog(t *testing.T) {
	t.Parallel()
	root := fixture(t, "macos")
	record := scan(t, root).Records[0]
	info, err := os.Stat(filepath.Join(record.SourcePath, "agents", "main", "wire.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if record.SourceSize != info.Size() || record.SizeBytes != info.Size() {
		t.Fatalf("size = %d / %d, want %d", record.SourceSize, record.SizeBytes, info.Size())
	}
}

func TestScanIsDeterministic(t *testing.T) {
	t.Parallel()
	root := fixture(t, "macos")
	if !reflect.DeepEqual(scan(t, root).Records, scan(t, root).Records) {
		t.Fatal("two scans of one fixture disagreed")
	}
}

// Subagents carry their own state.json. Without the "agents" exclusion the
// glob reports them as top-level sessions, the same bug Claude Code's reader
// had to avoid.
func TestSubagentIsNotASession(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	session := writeSession(t, root, "session_01987654-3210-7890-abcd-ef0123456789", 2)
	sub := filepath.Join(session, "agents", "agent-0")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(sub, "state.json"), map[string]any{
		"id": "session_subagent", "cwd": "/Users/fixture-user/code/demo", "version": 2,
	})
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want only the parent session", len(result.Records))
	}
	if strings.Contains(result.Records[0].ID, "subagent") {
		t.Fatalf("subagent was indexed as a session: %s", result.Records[0].ID)
	}
}

func TestCorruption(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		prep        func(t *testing.T, root string)
		wantRecords int
	}{
		{
			name:        "absent root",
			prep:        func(*testing.T, string) {},
			wantRecords: 0,
		},
		{
			name: "empty marker directory",
			prep: func(t *testing.T, root string) {
				if err := os.MkdirAll(filepath.Join(root, "sessions"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantRecords: 0,
		},
		{
			name: "empty state file",
			prep: func(t *testing.T, root string) {
				dir := sessionDir(t, root, "session_empty")
				if err := os.WriteFile(filepath.Join(dir, "state.json"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantRecords: 0,
		},
		{
			name: "unsupported schema version fails closed",
			prep: func(t *testing.T, root string) {
				writeSession(t, root, "session_future", 99)
			},
			wantRecords: 0,
		},
		{
			name: "missing schema version fails closed",
			prep: func(t *testing.T, root string) {
				dir := sessionDir(t, root, "session_noversion")
				writeJSON(t, filepath.Join(dir, "state.json"), map[string]any{
					"id": "session_noversion", "cwd": "/Users/fixture-user/code/demo",
				})
			},
			wantRecords: 0,
		},
		{
			name: "unknown wire protocol fails closed",
			prep: func(t *testing.T, root string) {
				dir := writeSession(t, root, "session_wireproto", 2)
				writeWire(t, dir, `{"type":"metadata","protocol_version":"9.0"}`)
			},
			wantRecords: 0,
		},
		{
			name: "truncated final record still yields the session",
			prep: func(t *testing.T, root string) {
				dir := writeSession(t, root, "session_truncated", 2)
				writeWire(t, dir,
					`{"type":"metadata","protocol_version":"1.5"}`,
					`{"type":"turn.prompt","origin":{"kind":"user"},"input":[{"type":"text","text":"complete"}]}`,
					`{"type":"turn.prom`,
				)
			},
			wantRecords: 1,
		},
		{
			name: "invalid utf-8 does not reach a record",
			prep: func(t *testing.T, root string) {
				dir := writeSession(t, root, "session_utf8", 2)
				writeWire(t, dir,
					`{"type":"metadata","protocol_version":"1.5"}`,
					"{\"type\":\"turn.prompt\",\"origin\":{\"kind\":\"user\"},\"input\":[{\"type\":\"text\",\"text\":\"ok\"}]}",
					"\xff\xfe",
				)
			},
			wantRecords: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.prep(t, root)
			if tt.name == "absent root" {
				root = filepath.Join(root, "missing")
			}
			result := scan(t, root)
			if len(result.Records) != tt.wantRecords {
				t.Fatalf("records = %d, want %d", len(result.Records), tt.wantRecords)
			}
			for _, record := range result.Records {
				if !utf8.ValidString(record.ID) || !utf8.ValidString(record.Title) ||
					!utf8.ValidString(record.PromptPreview) || !utf8.ValidString(record.SearchText) {
					t.Fatal("invalid UTF-8 leaked into a record")
				}
				if strings.TrimSpace(record.ReadOnlyReason) == "" {
					t.Fatal("record below T3 has no ReadOnlyReason")
				}
				if strings.Contains(record.PromptPreview, "turn.prom") {
					t.Fatal("a partial trailing record was indexed")
				}
			}
		})
	}
}

func sessionDir(t *testing.T, root, id string) string {
	t.Helper()
	dir := filepath.Join(root, "sessions", "wd_fixture-user_a1b2c3d4e5f6", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeSession(t *testing.T, root, id string, version int) string {
	t.Helper()
	dir := sessionDir(t, root, id)
	writeJSON(t, filepath.Join(dir, "state.json"), map[string]any{
		"id":        id,
		"title":     "fixture",
		"cwd":       "/Users/fixture-user/code/demo",
		"updatedAt": "2026-08-16T09:00:00.000Z",
		"version":   version,
	})
	return dir
}

func writeWire(t *testing.T, sessionDir string, lines ...string) {
	t.Helper()
	dir := filepath.Join(sessionDir, "agents", "main")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "wire.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, value map[string]any) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
