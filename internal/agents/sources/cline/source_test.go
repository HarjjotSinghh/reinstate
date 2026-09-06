package cline

import (
	"context"
	"encoding/json"
	"fmt"
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
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "cline", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantWorkspace string
		wantMessageCount              int
	}{
		{"macos", "1787122263475_fixture", "/Users/fixture-user/code/demo", 3},
		{"windows", "1787123369440_fixture", `C:\Users\fixture-user\code\demo`, 4},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentCline || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q", record.Workspace)
			}
			if record.CanResume || record.ReadOnlyReason != sessionindex.ClineReadOnlyReason {
				t.Fatalf("resume=%t reason=%q", record.CanResume, record.ReadOnlyReason)
			}
			if record.PromptPreview == "" {
				t.Fatal("empty preview")
			}
			// The committed *.messages.json sidecar pins this count; it comes
			// from the vendor's own record of the task, not a placeholder.
			if record.MessageCount != tt.wantMessageCount {
				t.Fatalf("message_count = %d, want %d", record.MessageCount, tt.wantMessageCount)
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

func TestMessagesSidecarIsNotIndexed(t *testing.T) {
	t.Parallel()
	root := fixture(t, "macos")
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1 (messages sidecar skipped)", len(result.Records))
	}
}

func TestCountMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		body   string // ignored when noFile is true
		noFile bool
		want   int
	}{
		{name: "three messages", body: `{"sessionId":"s","messages":[{"role":"user"},{"role":"assistant"},{"role":"user"}],"version":1}`, want: 3},
		{name: "empty array", body: `{"sessionId":"s","messages":[],"version":1}`, want: 0},
		{name: "missing sidecar", noFile: true, want: 0},
		{name: "malformed json", body: `{"sessionId":"s","messages":[`, want: 0},
		{name: "missing messages key", body: `{"sessionId":"s","version":1}`, want: 0},
		{name: "messages key not an array", body: `{"sessionId":"s","messages":"nope"}`, want: 0},
		{name: "messages key appears after other keys", body: `{"sessionId":"s","version":1,"messages":[{"role":"user"},{"role":"user"}]}`, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "x.messages.json")
			if !tt.noFile {
				if err := os.WriteFile(path, []byte(tt.body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := countMessages(path); got != tt.want {
				t.Fatalf("countMessages() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountMessagesOversizedSidecarDegradesToZero(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "x.messages.json")
	// One message repeated past maxMessagesSidecarBytes. The exact count is
	// never trusted from a partial scan of an oversized file; it degrades to
	// 0 rather than reporting a short count as if it were complete.
	var body []byte
	body = append(body, []byte(`{"sessionId":"s","messages":[`)...)
	one := []byte(`{"role":"user","text":"padding-padding-padding-padding"},`)
	for int64(len(body)) <= maxMessagesSidecarBytes {
		body = append(body, one...)
	}
	body = append(body, []byte(`{"role":"user"}]}`)...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := countMessages(path); got != 0 {
		t.Fatalf("countMessages() = %d, want 0 for an oversized sidecar", got)
	}
}

func TestMessagesSidecarPath(t *testing.T) {
	t.Parallel()
	got := messagesSidecarPath(filepath.Join("sessions", "abc", "abc.json"))
	want := filepath.Join("sessions", "abc", "abc.messages.json")
	if got != want {
		t.Fatalf("messagesSidecarPath() = %q, want %q", got, want)
	}
}

func TestUnknownLayoutIsSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "sessions", "bad")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{\"nope\":true}\n"), 0o644); err != nil {
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

// writeClineSession seeds a synthetic <slug>.json / <slug>.messages.json
// pair under root/sessions/<slug>/, no real content, matching the fixture
// layout under testdata/sessionindex/cline.
func writeClineSession(t *testing.T, root, slug, prompt string, messages []sidecarMessage) {
	t.Helper()
	dir := filepath.Join(root, "sessions", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(
		`{"cwd":"/tmp/demo","session_id":%q,"started_at":"2026-08-19T06:51:03Z","status":"completed","prompt":%q}`,
		slug, prompt,
	)
	if err := os.WriteFile(filepath.Join(dir, slug+".json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(struct {
		SessionID string           `json:"sessionId"`
		Messages  []sidecarMessage `json:"messages"`
		Version   int              `json:"version"`
	}{SessionID: slug, Messages: messages, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, slug+".messages.json"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSearchIndexesUserMessageBody covers Phase 5 Matrix C3 for Cline: search
// must find text from the message body, not only id/title/project/workspace.
// It also proves assistant-only text is excluded, matching the Claude
// reader's policy.
func TestSearchIndexesUserMessageBody(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const userToken = "fixture-search-token-cline"
	const assistantOnlyToken = "assistant-only-reply-marker-cline"
	writeClineSession(t, root, "slug-search", "", []sidecarMessage{
		{Role: "user", Text: "Investigate " + userToken + " in the retry loop"},
		{Role: "assistant", Text: assistantOnlyToken},
	})
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if !strings.Contains(record.SearchText, userToken) {
		t.Fatalf("search text does not contain the user message body: %q", record.SearchText)
	}
	if strings.Contains(record.SearchText, assistantOnlyToken) {
		t.Fatalf("search text leaked assistant-only content: %q", record.SearchText)
	}
	// meta.json's own prompt was empty, so the preview must fall back to the
	// sidecar's first user message rather than the bare session id.
	if !strings.Contains(record.PromptPreview, userToken) {
		t.Fatalf("prompt preview did not fall back to the first user message: %q", record.PromptPreview)
	}
}

// TestSearchTextBoundHolds proves a message far larger than
// sessionindex.MaxSearchTextBytes is truncated in the final SearchText
// rather than reported whole.
func TestSearchTextBoundHolds(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const marker = "fixture-search-token-cline-bound"
	oversized := marker + " " + strings.Repeat("padding ", 100000) // far over the 256 KiB search-text bound
	writeClineSession(t, root, "slug-bound", "", []sidecarMessage{
		{Role: "user", Text: oversized},
	})
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if !strings.Contains(record.SearchText, marker) {
		t.Fatalf("search text lost the leading marker: %q", record.SearchText[:min(200, len(record.SearchText))])
	}
	if len(record.SearchText) > sessionindex.MaxSearchTextBytes {
		t.Fatalf("search text = %d bytes, want at most %d", len(record.SearchText), sessionindex.MaxSearchTextBytes)
	}
}

// TestSearchTextRedactsControlSequences proves message-body text goes
// through the same SafeText sanitization the Claude reader applies, so a
// terminal escape sequence embedded in a message cannot survive into the
// stored index.
func TestSearchTextRedactsControlSequences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const marker = "fixture-search-token-cline-escape"
	writeClineSession(t, root, "slug-escape", "", []sidecarMessage{
		{Role: "user", Text: "\x1b[31m" + marker + "\x1b[0m\n more text"},
	})
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if strings.Contains(record.SearchText, "\x1b") {
		t.Fatalf("search text carries a raw terminal escape: %q", record.SearchText)
	}
	if !strings.Contains(record.SearchText, marker) {
		t.Fatalf("search text lost the sanitized marker: %q", record.SearchText)
	}
}

// TestMalformedSidecarStillYieldsRecord proves a *.messages.json this reader
// cannot parse degrades to an empty search text rather than failing the
// whole session record.
func TestMalformedSidecarStillYieldsRecord(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "sessions", "slug-malformed")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"cwd":"/tmp/demo","session_id":"slug-malformed","started_at":"2026-08-19T06:51:03Z","status":"completed","prompt":"a prompt"}`
	if err := os.WriteFile(filepath.Join(dir, "slug-malformed.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "slug-malformed.messages.json"), []byte(`not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if record.MessageCount != 0 {
		t.Fatalf("message_count = %d, want 0 for a malformed sidecar", record.MessageCount)
	}
}
