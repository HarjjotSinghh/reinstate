package sessionindex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGeminiSourceAppliesMetadataAndRewindReadOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "tmp", "project-hash", "chats", "session-current.jsonl")
	writeSyntheticJSONL(t, path, []string{
		`{"sessionId":"gemini-789","projectHash":"project-hash","startTime":"2026-07-30T08:00:00Z","lastUpdated":"2026-07-30T08:00:00Z","directories":["C:\\work\\gemini-demo"],"kind":"main"}`,
		`{"id":"user-1","timestamp":"2026-07-30T08:01:00Z","type":"user","content":[{"text":"Index Gemini chats"}]}`,
		`{"id":"model-1","timestamp":"2026-07-30T08:02:00Z","type":"gemini","content":"MODEL_PRIVATE_TEXT","thoughts":"MODEL_PRIVATE_THOUGHT","toolCalls":[{"name":"read_file","args":{"path":"README.md"},"result":"TOOL_OUTPUT_SECRET"}]}`,
		`{"id":"user-rewound","timestamp":"2026-07-30T08:03:00Z","type":"user","content":"REWOUND_PROMPT"}`,
		`{"$rewindTo":"model-1"}`,
		`{"$set":{"summary":"Gemini local indexing","lastUpdated":"2026-07-30T08:04:00Z"}}`,
	}, "")
	writeSyntheticJSONL(t,
		filepath.Join(root, "tmp", "project-hash", "chats", "session-child.jsonl"),
		[]string{
			`{"sessionId":"child","projectHash":"project-hash","kind":"subagent"}`,
			`{"id":"child-user","type":"user","content":"SUBAGENT_PROMPT"}`,
		},
		"",
	)

	result, err := NewGeminiSource(root).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.Key != "gemini:gemini-789" {
		t.Fatalf("key = %q", record.Key)
	}
	if record.Title != "Gemini local indexing" || record.PromptPreview != "Index Gemini chats" {
		t.Fatalf("title/preview = %q / %q", record.Title, record.PromptPreview)
	}
	if record.Project != "gemini-demo" || record.Workspace != `C:\work\gemini-demo` {
		t.Fatalf("project/workspace = %q / %q", record.Project, record.Workspace)
	}
	if record.MessageCount != 2 {
		t.Fatalf("message_count = %d, want 2", record.MessageCount)
	}
	if !record.UpdatedAt.Equal(time.Date(2026, 7, 30, 8, 4, 0, 0, time.UTC)) {
		t.Fatalf("updated_at = %s", record.UpdatedAt)
	}
	if record.CanResume || record.CanFork || record.ReadOnlyReason == "" {
		t.Fatalf("capabilities = resume:%t fork:%t reason:%q", record.CanResume, record.CanFork, record.ReadOnlyReason)
	}
	if len(record.Files) != 1 || record.Files[0] != "README.md" {
		t.Fatalf("files = %#v", record.Files)
	}
	for _, forbidden := range []string{
		"MODEL_PRIVATE_TEXT",
		"MODEL_PRIVATE_THOUGHT",
		"TOOL_OUTPUT_SECRET",
		"REWOUND_PROMPT",
		"SUBAGENT_PROMPT",
	} {
		if strings.Contains(record.SearchText, forbidden) {
			t.Fatalf("search text contains forbidden value %q", forbidden)
		}
	}
}

func TestGeminiSourceSupportsLegacyConversationJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "tmp", "legacy-project", "chats", "session-legacy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	const conversation = `{
		"sessionId":"legacy-id",
		"projectHash":"legacy-project",
		"lastUpdated":"2026-07-29T07:00:00Z",
		"summary":"Legacy chat",
		"messages":[
			{"id":"one","type":"user","content":"legacy user prompt"},
			{"id":"two","type":"gemini","content":"legacy assistant private"}
		]
	}`
	if err := os.WriteFile(path, []byte(conversation), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := NewGeminiSource(root).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.ID != "legacy-id" || record.Project != "legacy-project" {
		t.Fatalf("identity/project = %q / %q", record.ID, record.Project)
	}
	if !strings.Contains(record.SearchText, "legacy user prompt") ||
		strings.Contains(record.SearchText, "legacy assistant private") {
		t.Fatalf("search text = %q", record.SearchText)
	}
}
