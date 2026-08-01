package sessionindex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeSourceIndexesOnlySafeUserContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sessionPath := filepath.Join(root, "projects", "-Users-alice-work-demo", "fixture.jsonl")
	writeSyntheticJSONL(t, sessionPath, []string{
		`{"type":"user","sessionId":"claude-123","cwd":"C:\\Users\\alice\\work\\demo","gitBranch":"feature/index","timestamp":"2026-07-30T10:00:00Z","message":{"role":"user","content":[{"type":"text","text":"\u001b[31mImplement   literal search\u001b[0m"},{"type":"tool_result","content":"TOOL_RESULT_SECRET"},{"type":"tool_use","input":{"file_path":"internal/search.go"}}]}}`,
		`{"type":"assistant","timestamp":"2026-07-30T10:01:00Z","message":{"role":"assistant","content":[{"type":"text","text":"ASSISTANT_SECRET"}]}}`,
		`{"type":"user","isMeta":true,"message":{"role":"user","content":"META_SECRET"}}`,
		`not-json`,
	}, `{"type":"user","message":`)
	writeSyntheticJSONL(t,
		filepath.Join(root, "projects", "-Users-alice-work-demo", "fixture", "subagents", "child.jsonl"),
		[]string{`{"type":"user","sessionId":"child","message":{"role":"user","content":"child prompt"}}`},
		"",
	)

	result, err := NewClaudeSource(root).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.Key != "claude:claude-123" || record.ID != "claude-123" {
		t.Fatalf("identity = %q / %q", record.Key, record.ID)
	}
	if record.Workspace != `C:\Users\alice\work\demo` || record.Project != "demo" {
		t.Fatalf("workspace/project = %q / %q", record.Workspace, record.Project)
	}
	if record.Branch != "feature/index" {
		t.Fatalf("branch = %q", record.Branch)
	}
	if record.PromptPreview != "Implement literal search" {
		t.Fatalf("preview = %q", record.PromptPreview)
	}
	if record.Title != "claude-123" {
		t.Fatalf("title = %q, want ID fallback rather than prompt text", record.Title)
	}
	if !record.CanResume || !record.CanFork {
		t.Fatal("Claude session should support native resume and fork")
	}
	if !record.UpdatedAt.Equal(time.Date(2026, 7, 30, 10, 1, 0, 0, time.UTC)) {
		t.Fatalf("updated_at = %s", record.UpdatedAt)
	}
	if len(record.Files) != 1 || record.Files[0] != "internal/search.go" {
		t.Fatalf("files = %#v", record.Files)
	}
	for _, forbidden := range []string{"ASSISTANT_SECRET", "TOOL_RESULT_SECRET", "META_SECRET", "child prompt"} {
		if strings.Contains(record.SearchText, forbidden) {
			t.Fatalf("private search text contains forbidden value %q", forbidden)
		}
	}
	if !hasWarningCode(result.Warnings, "malformed_record") ||
		!hasWarningCode(result.Warnings, "incomplete_trailing_record") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestClaudeSourceAbsentRootIsEmpty(t *testing.T) {
	t.Parallel()
	result, err := NewClaudeSource(filepath.Join(t.TempDir(), "missing")).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Records) != 0 {
		t.Fatalf("records = %d, want 0", len(result.Records))
	}
}

func writeSyntheticJSONL(t *testing.T, path string, complete []string, partial string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	content := strings.Join(complete, "\n")
	if len(complete) > 0 {
		content += "\n"
	}
	content += partial
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hasWarningCode(warnings []Warning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}
