package qwen

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "qwen", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantWorkspace, wantProject, wantPreview, wantFile string
	}{
		{
			"macos", "01987654-3210-7890-abcd-ef0123456789",
			"/Users/fixture-user/code/demo", "demo",
			"List the retry budget",
			"/Users/fixture-user/code/demo/internal/agentcheck/budget.go",
		},
		{
			"windows", "01912345-6789-7abc-def0-123456789abc",
			`C:\Users\fixture-user\code\demo`, "demo",
			"Map the Windows dest argv",
			`C:\Users\fixture-user\code\demo\internal\pathmap\map.go`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentQwen || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Workspace != tt.wantWorkspace || record.Project != tt.wantProject {
				t.Fatalf("workspace/project = %q / %q", record.Workspace, record.Project)
			}
			if record.CanResume || record.CanFork {
				t.Fatal("T1 must refuse resume and fork")
			}
			if record.ReadOnlyReason != sessionindex.QwenReadOnlyReason {
				t.Fatalf("read_only_reason = %q", record.ReadOnlyReason)
			}
			// A non-empty preview is not enough. Qwen carries its message body
			// in Gemini's parts[] shape, and reading it as Claude's content
			// blocks silently produced an empty preview and a title that was
			// only the session uuid.
			if record.PromptPreview != tt.wantPreview {
				t.Fatalf("preview = %q, want %q", record.PromptPreview, tt.wantPreview)
			}
			if record.Title != tt.wantPreview {
				t.Fatalf("title = %q, want the first user prompt %q", record.Title, tt.wantPreview)
			}
			if record.MessageCount < 2 {
				t.Fatalf("messages = %d, want the user and assistant turns", record.MessageCount)
			}
			if !contains(record.Files, tt.wantFile) {
				t.Fatalf("files = %v, want the functionCall argument %q", record.Files, tt.wantFile)
			}
			if !strings.Contains(record.SearchText, tt.wantPreview) {
				t.Fatalf("search text does not contain the prompt: %q", record.SearchText)
			}
		})
	}
}

// TestHarnessUserRecordsAreNotThePrompt guards the vendor's own provenance
// field. Qwen writes cron prompts and notifications as type:"user" with
// provenance:"system"; letting one become a session title would show the
// operator harness text they never typed.
func TestHarnessUserRecordsAreNotThePrompt(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "projects", "-Users-fixture-user-code-demo", "chats")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	lines := `{"cwd":"/Users/fixture-user/code/demo","message":{"role":"user","parts":[{"text":"CRON_NOTIFICATION_TEXT"}]},"parentUuid":null,"provenance":"system","sessionId":"01987654-cron-4000-8000-000000000005","subtype":"cron","timestamp":"2026-08-20T13:00:00.000Z","type":"user","uuid":"dddd0001-1111-4111-8111-dddddddddddd","version":"0.21.13"}
{"cwd":"/Users/fixture-user/code/demo","message":{"role":"user","parts":[{"text":"REAL_OPERATOR_PROMPT"}]},"parentUuid":"dddd0001-1111-4111-8111-dddddddddddd","provenance":"real_user","sessionId":"01987654-cron-4000-8000-000000000005","timestamp":"2026-08-20T13:00:01.000Z","type":"user","uuid":"dddd0002-2222-4222-8222-dddddddddddd","version":"0.21.13"}
`
	path := filepath.Join(dir, "01987654-cron-4000-8000-000000000005.jsonl")
	if err := os.WriteFile(path, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d", len(result.Records))
	}
	record := result.Records[0]
	if record.PromptPreview != "REAL_OPERATOR_PROMPT" {
		t.Fatalf("preview = %q, want the operator's prompt", record.PromptPreview)
	}
	if strings.Contains(record.SearchText, "CRON_NOTIFICATION_TEXT") {
		t.Fatal("harness-provenance text reached the search index as a user prompt")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
	dir := filepath.Join(root, "projects", "demo", "chats")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.jsonl"), []byte("{\"nope\":true}\n"), 0o644); err != nil {
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
