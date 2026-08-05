package sessionindex

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
)

func TestCodexSourceIndexesRolloutShapesWithoutAssistantText(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "sessions", "2026", "07", "rollout-fixture.jsonl")
	writeSyntheticJSONL(t, path, []string{
		`{"timestamp":"2026-07-30T09:00:00Z","type":"session_meta","payload":{"id":"codex-456","cwd":"/Users/alice/work/reinstate","git":{"branch":"phase-two","repository_url":"` + syntheticCredentialRemote("fixture-user", "fixture-secret", "github.com/example/reinstate.git", "token=fixture-secret") + `","commit_hash":"0123456789abcdef0123456789abcdef01234567"}}}`,
		`{"timestamp":"2026-07-30T09:01:00Z","type":"event_msg","payload":{"type":"user_message","message":"Find local sessions"}}`,
		`{"timestamp":"2026-07-30T09:02:00Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"ENVIRONMENT_DUMP_SECRET"}]}}`,
		`{"timestamp":"2026-07-30T09:03:00Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ASSISTANT_REASONING_SECRET"}]}}`,
		`{"timestamp":"2026-07-30T09:04:00Z","type":"response_item","payload":{"type":"function_call","name":"read_file","arguments":"{\"file_path\":\"internal/sessionindex/store.go\",\"content\":\"{\\\"path\\\":\\\"NOT_A_STRUCTURED_FIELD\\\"}\"}"}}`,
	}, `{"type":"event_msg","payload":{"type":"user_message"`)

	result, err := NewCodexSource(root).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.Key != "codex:codex-456" {
		t.Fatalf("key = %q", record.Key)
	}
	if record.Project != "reinstate" || record.Workspace != "/Users/alice/work/reinstate" {
		t.Fatalf("project/workspace = %q / %q", record.Project, record.Workspace)
	}
	if record.Branch != "phase-two" {
		t.Fatalf("branch = %q", record.Branch)
	}
	if record.RecordedEnvironment.RepositoryID.Value != environment.NormalizeRepositoryID("https://github.com/example/reinstate.git") ||
		record.RecordedEnvironment.RepositoryID.Provenance != "codex.session_meta.git.repository_url" ||
		record.RecordedEnvironment.Branch.Value != "phase-two" ||
		record.RecordedEnvironment.GitHead.Value != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("recorded environment = %+v", record.RecordedEnvironment)
	}
	if record.PromptPreview != "Find local sessions" {
		t.Fatalf("preview = %q", record.PromptPreview)
	}
	if record.Title != "codex-456" {
		t.Fatalf("title = %q, want ID fallback rather than prompt text", record.Title)
	}
	if record.MessageCount != 3 {
		t.Fatalf("message_count = %d, want 3", record.MessageCount)
	}
	if !record.UpdatedAt.Equal(time.Date(2026, 7, 30, 9, 4, 0, 0, time.UTC)) {
		t.Fatalf("updated_at = %s", record.UpdatedAt)
	}
	if len(record.Files) != 1 || record.Files[0] != "internal/sessionindex/store.go" {
		t.Fatalf("files = %#v", record.Files)
	}
	for _, forbidden := range []string{"ASSISTANT_REASONING_SECRET", "ENVIRONMENT_DUMP_SECRET", "NOT_A_STRUCTURED_FIELD"} {
		if strings.Contains(record.SearchText, forbidden) || containsString(record.Files, forbidden) {
			t.Fatalf("indexed forbidden value %q", forbidden)
		}
	}
	if !record.CanResume || !record.CanFork {
		t.Fatal("Codex session should support native resume and fork")
	}
	if !hasWarningCode(result.Warnings, "incomplete_trailing_record") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func containsString(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
