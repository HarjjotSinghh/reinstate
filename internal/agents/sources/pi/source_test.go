package pi

import (
	"context"
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
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "pi", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName        string
		wantID        string
		wantWorkspace string
		wantProject   string
	}{
		{
			osName:        "macos",
			wantID:        "01987654-3210-7890-abcd-ef0123456789",
			wantWorkspace: "/Users/fixture-user/code/demo",
			wantProject:   "demo",
		},
		{
			osName:        "windows",
			wantID:        "01912345-6789-7abc-def0-123456789abc",
			wantWorkspace: `C:\Users\fixture-user\code\demo`,
			wantProject:   "demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentPi || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Key != sessionindex.CompositeReference(sessionindex.AgentPi, tt.wantID) {
				t.Fatalf("key = %q", record.Key)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q, want %q", record.Workspace, tt.wantWorkspace)
			}
			if record.Project != tt.wantProject {
				t.Fatalf("project = %q, want %q", record.Project, tt.wantProject)
			}
			if record.Title != tt.wantID {
				t.Fatalf("title = %q, want id", record.Title)
			}
			if record.PromptPreview != tt.wantProject {
				t.Fatalf("preview = %q, want project", record.PromptPreview)
			}
			if record.MessageCount != 0 {
				t.Fatalf("message_count = %d, want 0 (header-only T1)", record.MessageCount)
			}
			if record.UpdatedAt.IsZero() {
				t.Fatal("updated_at is zero")
			}
			if record.CanResume || record.CanFork {
				t.Fatalf("capabilities resume=%t fork=%t, want false", record.CanResume, record.CanFork)
			}
			if record.ReadOnlyReason != sessionindex.PiReadOnlyReason {
				t.Fatalf("read_only_reason = %q", record.ReadOnlyReason)
			}
		})
	}
}

func TestScanIsDeterministic(t *testing.T) {
	t.Parallel()
	root := fixture(t, "macos")
	if !reflect.DeepEqual(scan(t, root).Records, scan(t, root).Records) {
		t.Fatal("two scans of one fixture disagreed")
	}
}

func TestSkillsTreeIsNotASession(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSession(t, root, "ok", headerLine("ok", "/Users/fixture-user/code/demo"))
	skill := filepath.Join(root, "skills", "personal")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "leaked.jsonl"), []byte(headerLine("skill-session", "/tmp")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want only the real session", len(result.Records))
	}
	if result.Records[0].ID != "ok" {
		t.Fatalf("indexed %q", result.Records[0].ID)
	}
}

func TestLaterLinesAreNotMessages(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSession(t, root, "tree",
		headerLine("tree", "/Users/fixture-user/code/demo"),
		`{"id":"entry-1","parentId":null}`,
		`{"id":"entry-2","parentId":"entry-1","type":"message"}`,
	)
	record := scan(t, root).Records[0]
	if record.MessageCount != 0 {
		t.Fatalf("message_count = %d, T1 must not invent a transcript", record.MessageCount)
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
			name: "empty file",
			prep: func(t *testing.T, root string) {
				writeSession(t, root, "empty")
			},
			wantRecords: 0,
		},
		{
			name: "missing type fails closed",
			prep: func(t *testing.T, root string) {
				writeSession(t, root, "notype", `{"id":"ok","cwd":"/Users/fixture-user/code/demo"}`)
			},
			wantRecords: 0,
		},
		{
			name: "unknown type fails closed",
			prep: func(t *testing.T, root string) {
				writeSession(t, root, "other", `{"type":"message","id":"ok"}`)
			},
			wantRecords: 0,
		},
		{
			name: "truncated final record still yields the session",
			prep: func(t *testing.T, root string) {
				writeSession(t, root, "truncated",
					headerLine("truncated", "/Users/fixture-user/code/demo"),
					`{"id":"par`,
				)
			},
			wantRecords: 1,
		},
		{
			name: "invalid utf-8 does not reach a record",
			prep: func(t *testing.T, root string) {
				writeSession(t, root, "utf8",
					headerLine("utf8", "/Users/fixture-user/code/demo"),
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
				if strings.Contains(record.ID, "par") && record.ID != "truncated" {
					t.Fatal("a partial trailing record was indexed")
				}
			}
		})
	}
}

func writeSession(t *testing.T, root, name string, lines ...string) {
	t.Helper()
	dir := filepath.Join(root, "sessions", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := ""
	if len(lines) > 0 {
		body = strings.Join(lines, "\n") + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name+".jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func headerLine(id, cwd string) string {
	return `{"type":"session","id":"` + id + `","cwd":"` + cwd + `","timestamp":"2026-08-17T12:00:00.000Z","version":3}`
}
