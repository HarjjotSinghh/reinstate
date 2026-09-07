package pi

import (
	"context"
	"encoding/json"
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
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "pi", osName)
}

func recordByID(t *testing.T, records []sessionindex.Record, id string) sessionindex.Record {
	t.Helper()
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	t.Fatalf("no record with id %q among %d records", id, len(records))
	return sessionindex.Record{}
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
			if len(result.Records) != 2 {
				t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
			}
			record := recordByID(t, result.Records, tt.wantID)
			if record.Agent != sessionindex.AgentPi {
				t.Fatalf("agent = %q", record.Agent)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q", record.Workspace)
			}
			if record.CanResume || record.ReadOnlyReason != sessionindex.PiReadOnlyReason {
				t.Fatalf("resume=%t reason=%q", record.CanResume, record.ReadOnlyReason)
			}
			if record.PromptPreview == "" {
				t.Fatal("empty preview")
			}
		})
	}
}

// TestScanVersion3ExtractsMessageContent covers the real (version 3) shape,
// where each turn item is {"type":"message","message":{"role":...,
// "content":[{"type":"text","text":...}, ...]}}. Regression coverage for
// F-PI-CONTENT-EXTRACTION: prompt_preview and search text must come from
// the first user message.content text, never from the assistant's or from
// the unparsed message wrapper.
func TestScanVersion3ExtractsMessageContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantToken string
	}{
		{"macos", "01987700-1111-7890-abcd-ef0123456789", "fixture-v3-search-token"},
		{"windows", "01987711-2222-7abc-def0-123456789abc", "fixture-v3-search-token"},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			record := recordByID(t, result.Records, tt.wantID)
			if record.MessageCount != 2 {
				t.Fatalf("message_count = %d, want 2 (one user, one assistant)", record.MessageCount)
			}
			if !strings.Contains(record.PromptPreview, tt.wantToken) {
				t.Fatalf("prompt_preview = %q, want it to contain %q", record.PromptPreview, tt.wantToken)
			}
			if strings.Contains(record.PromptPreview, "Reading agentcheck") ||
				strings.Contains(record.PromptPreview, "Checking newlines") {
				t.Fatalf("prompt_preview leaked assistant text: %q", record.PromptPreview)
			}
			if !strings.Contains(record.SearchText, tt.wantToken) {
				t.Fatalf("search_text = %q, want it to contain %q", record.SearchText, tt.wantToken)
			}
			if strings.Contains(record.SearchText, "considering the retry budget") ||
				strings.Contains(record.SearchText, "checking newlines") {
				t.Fatalf("search_text leaked assistant/thinking text: %q", record.SearchText)
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

// writeSessionFile writes one JSONL session with the given lines (each
// already JSON-encoded) under a fresh temp hometree root and returns the
// path to the session file it wrote.
func writeSessionFile(t *testing.T, lines []string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions", "fixture-project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func line(t *testing.T, item map[string]any) string {
	t.Helper()
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// TestReadConversationMalformedContent covers content shapes that must
// yield no extracted text and never fail the read: an absent message, a
// message that isn't an object, content that isn't an array, and content
// array entries that aren't text-bearing objects.
func TestReadConversationMalformedContent(t *testing.T) {
	t.Parallel()
	header := map[string]any{
		"type":      "session",
		"version":   3,
		"id":        "01900000-0000-7000-8000-000000000000",
		"timestamp": "2026-08-20T09:00:00.000Z",
		"cwd":       "/Users/fixture-user/code/malformed",
	}

	tests := []struct {
		name         string
		item         map[string]any
		wantMessages int
	}{
		{
			// message isn't an object, and there is no legacy top-level
			// "type":"user"/"assistant" to fall back to (only the
			// ambiguous "message" wrapper type), so the role cannot be
			// resolved and the turn is not counted.
			name:         "message is a string, not an object",
			item:         map[string]any{"type": "message", "id": "m1", "message": "not-an-object"},
			wantMessages: 0,
		},
		{
			name:         "message has no content field",
			item:         map[string]any{"type": "message", "id": "m1", "message": map[string]any{"role": "user"}},
			wantMessages: 1,
		},
		{
			name:         "content is a number",
			item:         map[string]any{"type": "message", "id": "m1", "message": map[string]any{"role": "user", "content": 42}},
			wantMessages: 1,
		},
		{
			name: "content array holds only non-text blocks",
			item: map[string]any{
				"type": "message", "id": "m1",
				"message": map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{"type": "tool_use", "id": "tu1", "input": map[string]any{}},
						map[string]any{"type": "thinking", "thinking": "no text field"},
					},
				},
			},
			wantMessages: 1,
		},
		{
			name: "content array holds unrecognized element types",
			item: map[string]any{
				"type": "message", "id": "m1",
				"message": map[string]any{
					"role":    "user",
					"content": []any{42, true, nil, []any{"nested"}},
				},
			},
			wantMessages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := writeSessionFile(t, []string{line(t, header), line(t, tt.item)})
			out, err := readConversation(path)
			if err != nil {
				t.Fatalf("readConversation returned an error: %v", err)
			}
			if out.firstPrompt != "" {
				t.Fatalf("firstPrompt = %q, want empty", out.firstPrompt)
			}
			if out.prompts.String() != "" {
				t.Fatalf("prompts = %q, want empty", out.prompts.String())
			}
			if out.messages != tt.wantMessages {
				t.Fatalf("messages = %d, want %d", out.messages, tt.wantMessages)
			}
		})
	}
}

// TestTurnRoleAndText exercises the role/text resolution directly across
// the version 3 shape, the legacy version 1 shape, and malformed inputs.
func TestTurnRoleAndText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		item     map[string]any
		wantRole string
		wantText string
	}{
		{
			name: "version 3 user turn with a single text part",
			item: map[string]any{
				"type": "message",
				"message": map[string]any{
					"role":    "user",
					"content": []any{map[string]any{"type": "text", "text": "hello"}},
				},
			},
			wantRole: "user",
			wantText: "hello",
		},
		{
			name: "version 3 assistant turn mixing thinking, tool_use, and text",
			item: map[string]any{
				"type": "message",
				"message": map[string]any{
					"role": "assistant",
					"content": []any{
						map[string]any{"type": "thinking", "thinking": "hidden"},
						map[string]any{"type": "tool_use", "id": "tu1"},
						map[string]any{"type": "text", "text": "visible"},
					},
				},
			},
			wantRole: "assistant",
			wantText: "visible",
		},
		{
			name:     "legacy version 1 top-level shape",
			item:     map[string]any{"type": "user", "text": "legacy prompt"},
			wantRole: "user",
			wantText: "legacy prompt",
		},
		{
			name:     "legacy version 1 assistant shape",
			item:     map[string]any{"type": "assistant", "text": "legacy reply"},
			wantRole: "assistant",
			wantText: "legacy reply",
		},
		{
			name:     "unrelated event type",
			item:     map[string]any{"type": "model_change", "modelId": "x"},
			wantRole: "model_change",
			wantText: "",
		},
		{
			name:     "message present but not an object",
			item:     map[string]any{"type": "user", "message": "oops", "text": "fallback text"},
			wantRole: "user",
			wantText: "fallback text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			role, text := turnRoleAndText(tt.item)
			if role != tt.wantRole {
				t.Fatalf("role = %q, want %q", role, tt.wantRole)
			}
			if text != tt.wantText {
				t.Fatalf("text = %q, want %q", text, tt.wantText)
			}
		})
	}
}
