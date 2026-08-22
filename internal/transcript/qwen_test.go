package transcript

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func qwenFixture(t *testing.T, class, sessionID string) string {
	t.Helper()
	return filepath.Join(
		"..", "..", "testdata", "handoff", "qwen", class,
		"projects", "-Users-fixture-user-code-demo", "chats", sessionID+".jsonl",
	)
}

func qwenIndexRecord(t *testing.T, path, sessionID string) sessionindex.Record {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return sessionindex.Record{
		ID:            sessionID,
		Agent:         sessionindex.AgentQwen,
		Workspace:     "/Users/fixture-user/code/demo",
		Project:       "demo",
		SourcePath:    path,
		SourceModTime: info.ModTime().UnixNano(),
		SourceSize:    info.Size(),
	}
}

// noQwenInstall keeps every reader test off the contributor's machine: the
// shared compatibility contract only consults a version when one is readable.
func noQwenInstall() VersionResolver {
	return func(context.Context, sessionindex.Record) (string, agentcheck.VersionEvidence) {
		return "", agentcheck.VersionUnavailable
	}
}

func parseQwenFixture(t *testing.T, class, sessionID string) ([]capsule.Event, ParseReport, Boundary) {
	t.Helper()
	path := qwenFixture(t, class, sessionID)
	reader := &QwenReader{ResolveVersion: noQwenInstall()}
	record := qwenIndexRecord(t, path, sessionID)
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return events, report, boundary
}

func qwenTexts(events []capsule.Event) string {
	var parts []string
	for _, event := range events {
		for _, block := range event.Blocks {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func TestQwenReaderName(t *testing.T) {
	t.Parallel()
	if got := NewQwenReader().Name(); got != sessionindex.AgentQwen {
		t.Fatalf("Name() = %q, want %q", got, sessionindex.AgentQwen)
	}
	if _, ok := Get(sessionindex.AgentQwen); !ok {
		t.Fatal("qwen reader is not in the registry")
	}
}

func TestQwenProbeLayout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		record sessionindex.Record
		want   Compatibility
	}{
		{
			name: "chats jsonl",
			record: sessionindex.Record{
				Agent:      sessionindex.AgentQwen,
				SourcePath: "/root/projects/-Users-fixture-user-code-demo/chats/a.jsonl",
			},
			want: CompatibilitySupported,
		},
		{
			name: "archived chats jsonl",
			record: sessionindex.Record{
				Agent:      sessionindex.AgentQwen,
				SourcePath: "/root/projects/-Users-fixture-user-code-demo/chats/archive/a.jsonl",
			},
			want: CompatibilitySupported,
		},
		{
			name: "subagent transcript",
			record: sessionindex.Record{
				Agent:      sessionindex.AgentQwen,
				SourcePath: "/root/projects/-Users-fixture-user-code-demo/subagents/a/b.jsonl",
			},
			want: CompatibilityUnsupported,
		},
		{
			name: "runtime sidecar",
			record: sessionindex.Record{
				Agent:      sessionindex.AgentQwen,
				SourcePath: "/root/projects/-Users-fixture-user-code-demo/chats/a.runtime.json",
			},
			want: CompatibilityUnsupported,
		},
		{
			name: "outside the projects tree",
			record: sessionindex.Record{
				Agent:      sessionindex.AgentQwen,
				SourcePath: "/root/tmp/deadbeef/chats/a.jsonl",
			},
			want: CompatibilityUnsupported,
		},
		{
			name:   "empty path",
			record: sessionindex.Record{Agent: sessionindex.AgentQwen},
			want:   CompatibilityUnsupported,
		},
		{
			name: "another agent",
			record: sessionindex.Record{
				Agent:      sessionindex.AgentClaude,
				SourcePath: "/root/projects/demo/chats/a.jsonl",
			},
			want: CompatibilityUnsupported,
		},
	}
	reader := &QwenReader{ResolveVersion: noQwenInstall()}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := reader.Probe(context.Background(), tt.record)
			if err != nil {
				t.Fatalf("Probe() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Probe() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestQwenReaderReadsGeminiShapedMessageBody is the regression for the defect
// that shipped at T1: Qwen's record keys match Claude Code's, so reading the
// body as Claude's content-block array yields no text at all.
func TestQwenReaderReadsGeminiShapedMessageBody(t *testing.T) {
	t.Parallel()
	events, report, _ := parseQwenFixture(t, "basic", "01987654-basic-4000-8000-000000000001")
	if report.Events != len(events) || len(events) == 0 {
		t.Fatalf("events = %d, report = %d", len(events), report.Events)
	}
	text := qwenTexts(events)
	for _, want := range []string{
		"Basic Qwen handoff user prompt",
		"Reading the budget file.",
		"const retryBudget = 3",
		"The budget constant and the test disagree.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("parsed text is missing %q", want)
		}
	}

	var user, assistant, toolCall, toolResult int
	for _, event := range events {
		switch event.Kind {
		case capsule.KindMessage:
			if event.Actor == capsule.ActorUser {
				user++
			}
			if event.Actor == capsule.ActorAssistant {
				assistant++
			}
		case capsule.KindToolCall:
			toolCall++
		case capsule.KindToolResult:
			toolResult++
		}
	}
	if user != 1 || assistant != 2 || toolCall != 2 || toolResult != 2 {
		t.Fatalf("user=%d assistant=%d toolCall=%d toolResult=%d", user, assistant, toolCall, toolResult)
	}
}

func TestQwenReaderLinksToolCallsAndFlagsErrors(t *testing.T) {
	t.Parallel()
	events, _, _ := parseQwenFixture(t, "basic", "01987654-basic-4000-8000-000000000001")
	linked := map[string]string{}
	errors := 0
	for _, event := range events {
		if event.Kind == capsule.KindToolResult {
			linked[event.NativeName] = event.LinkedCallID
			for _, block := range event.Blocks {
				if block.IsError {
					errors++
				}
			}
		}
	}
	if linked["read_file"] != "call-basic-1" || linked["run_shell_command"] != "call-basic-2" {
		t.Fatalf("linked call ids = %v", linked)
	}
	if errors != 1 {
		t.Fatalf("error blocks = %d, want 1 (the failing run_shell_command)", errors)
	}
}

// TestQwenReaderTokenizesToolPaths asserts the reader-boundary path contract:
// no absolute vendor path reaches a capsule block.
func TestQwenReaderTokenizesToolPaths(t *testing.T) {
	t.Parallel()
	events, _, _ := parseQwenFixture(t, "basic", "01987654-basic-4000-8000-000000000001")
	for _, event := range events {
		for _, block := range event.Blocks {
			if strings.Contains(block.Text, "/Users/fixture-user/code/demo") {
				t.Fatalf("event %s leaked an absolute workspace path: %q", event.ID, block.Text)
			}
		}
	}
	if !strings.Contains(qwenTexts(events), "${REPO:demo}") {
		t.Fatal("tool input paths were not rewritten into a portable token")
	}
}

// TestQwenReaderExcludesRewoundBranch is the case Qwen does differently from
// every other agent in the catalog. /rewind leaves the discarded turns on disk
// on a dead branch of the uuid tree; a line-by-line reader would replay them.
func TestQwenReaderExcludesRewoundBranch(t *testing.T) {
	t.Parallel()
	events, report, _ := parseQwenFixture(t, "rewound", "01987654-rwnd-4000-8000-000000000002")
	text := qwenTexts(events)
	for _, want := range []string{"LIVE_FIRST_QUESTION", "LIVE_FIRST_ANSWER", "LIVE_REPLACEMENT_QUESTION", "LIVE_REPLACEMENT_ANSWER"} {
		if !strings.Contains(text, want) {
			t.Fatalf("live conversation is missing %q", want)
		}
	}
	for _, never := range []string{"DEAD_BRANCH_QUESTION", "DEAD_BRANCH_ANSWER"} {
		if strings.Contains(text, never) {
			t.Fatalf("rewound record %q was replayed into the capsule", never)
		}
	}

	var checkpoints int
	for _, event := range events {
		if event.Kind == capsule.KindCheckpoint && event.Reason == reasonQwenRewindMarker {
			checkpoints++
		}
	}
	if checkpoints != 1 {
		t.Fatalf("rewind checkpoints = %d, want 1", checkpoints)
	}
	if !hasWarning(report, warningQwenRewound) {
		t.Fatalf("warnings = %v, want %s", report.Warnings, warningQwenRewound)
	}
}

func TestQwenReaderStopsAtLastCompleteRecord(t *testing.T) {
	t.Parallel()
	events, _, boundary := parseQwenFixture(t, "partial-final-record", "01987654-part-4000-8000-000000000003")
	if !boundary.Partial {
		t.Fatal("boundary.Partial = false, want true for a torn trailing record")
	}
	if boundary.ByteOffset >= boundary.SizeBytes {
		t.Fatalf("byte offset %d must exclude the torn tail of %d bytes", boundary.ByteOffset, boundary.SizeBytes)
	}
	text := qwenTexts(events)
	if !strings.Contains(text, "COMPLETE_USER_RECORD") || !strings.Contains(text, "COMPLETE_ASSISTANT_RECORD") {
		t.Fatalf("complete records were dropped: %q", text)
	}
	if strings.Contains(text, "TORN_TRAILING_RECORD") {
		t.Fatal("a record past the frozen boundary was parsed")
	}
}

func TestQwenReaderClassifiesUnknownRecords(t *testing.T) {
	t.Parallel()
	events, report, _ := parseQwenFixture(t, "unknown-records", "01987654-unkn-4000-8000-000000000004")
	if report.UnknownRecords < 3 {
		t.Fatalf("unknown records = %d, want at least 3 (type, subtype, part)", report.UnknownRecords)
	}
	for _, event := range events {
		if event.Kind != capsule.KindUnknown {
			continue
		}
		if event.Portability != capsule.PortabilityReferenced && event.Portability != capsule.PortabilityOmitted {
			t.Fatalf("unknown event %s has portability %q", event.ID, event.Portability)
		}
		if event.Reason == "" {
			t.Fatalf("unknown event %s has no machine-readable reason", event.ID)
		}
		for _, block := range event.Blocks {
			if block.Type == capsule.BlockTypeText {
				t.Fatalf("unknown event %s guessed a text body", event.ID)
			}
		}
	}
	if !strings.Contains(qwenTexts(events), "KNOWN_ASSISTANT_RECORD") {
		t.Fatal("a known record after an unknown one was dropped")
	}
}

func TestQwenReaderIsDeterministic(t *testing.T) {
	t.Parallel()
	first, firstReport, _ := parseQwenFixture(t, "basic", "01987654-basic-4000-8000-000000000001")
	second, secondReport, _ := parseQwenFixture(t, "basic", "01987654-basic-4000-8000-000000000001")
	if len(first) != len(second) || firstReport.Events != secondReport.Events {
		t.Fatalf("event counts differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID || first[i].ContentHash != second[i].ContentHash {
			t.Fatalf("event %d differs between runs", i)
		}
	}
}

func TestQwenReaderParsesIndexedFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName    string
		slug      string
		sessionID string
		wantText  string
	}{
		{"macos", "-Users-fixture-user-code-demo", "01987654-3210-7890-abcd-ef0123456789", "List the retry budget"},
		{"windows", "c--users-fixture-user-code-demo", "01912345-6789-7abc-def0-123456789abc", "Map the Windows dest argv"},
	}
	reader := &QwenReader{ResolveVersion: noQwenInstall()}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.osName, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(
				"..", "..", "testdata", "sessionindex", "qwen", tt.osName,
				"projects", tt.slug, "chats", tt.sessionID+".jsonl",
			)
			record := qwenIndexRecord(t, path, tt.sessionID)
			boundary, err := reader.Snapshot(context.Background(), record)
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			events, _, err := reader.Parse(context.Background(), boundary)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !strings.Contains(qwenTexts(events), tt.wantText) {
				t.Fatalf("parsed text is missing %q", tt.wantText)
			}
		})
	}
}

func hasWarning(report ParseReport, code string) bool {
	for _, warning := range report.Warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}
