package cline

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
