package copilot

import (
	"context"
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
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "copilot", osName)
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
			if record.Agent != sessionindex.AgentCopilot || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q", record.Workspace)
			}
			if record.CanResume || record.ReadOnlyReason != sessionindex.CopilotReadOnlyReason {
				t.Fatalf("resume=%t reason=%q", record.CanResume, record.ReadOnlyReason)
			}
			if record.PromptPreview == "" || record.MessageCount < 1 {
				t.Fatalf("preview=%q messages=%d", record.PromptPreview, record.MessageCount)
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

func TestUnknownLayoutIsSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "session-state", "bad-session")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{\"nope\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 0 {
		t.Fatalf("records = %d, want 0", len(result.Records))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("want a session_read_failed warning")
	}
}

func TestSQLiteSidecarIsNotIndexed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "session-state", "only-db")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.db"), []byte("not-a-session"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "session-store.db"), []byte("not-a-session"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 0 {
		t.Fatalf("records = %+v", result.Records)
	}
}
