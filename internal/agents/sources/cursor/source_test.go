package cursor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "cursor", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantWorkspace string
	}{
		{"macos", "01987654-3210-7890-abcd-ef0123456789", "/Users/fixture-user/code/demo"},
		{"windows", "01912345-6789-7abc-def0-123456789abc", `C:\Users\fixture-user\code\demo`},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentCursor || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q", record.Workspace)
			}
			if record.CanResume || record.ReadOnlyReason != sessionindex.CursorReadOnlyReason {
				t.Fatalf("resume=%t reason=%q", record.CanResume, record.ReadOnlyReason)
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

func TestEditorTreeIsNotIndexed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMeta(t, filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json"), true)
	writeMeta(t, filepath.Join(root, "projects", "demo", "agent-transcripts", "sess-2", "meta.json"), true)
	result := scan(t, root)
	if len(result.Records) != 1 || result.Records[0].ID != "sess-1" {
		t.Fatalf("records = %+v", result.Records)
	}
}

func TestEmptyConversationIsSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMeta(t, filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "empty", "meta.json"), false)
	result := scan(t, root)
	if len(result.Records) != 0 {
		t.Fatalf("records = %+v", result.Records)
	}
}

func writeMeta(t *testing.T, path string, hasConversation bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(
		`{"createdAtMs":1,"cwd":"/tmp/demo","hasConversation":%t,"schemaVersion":1,"updatedAtMs":2}`,
		hasConversation,
	)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
