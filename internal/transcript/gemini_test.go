package transcript

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestGeminiReaderRewindDropsExclusiveTail(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "testdata", "handoff", "gemini", "rewind", "session-rewind.jsonl")
	events, report := parseGeminiFixture(t, path, "gemini-handoff-rewind")

	if len(events) != 1 {
		t.Fatalf("events = %d, want 1 after exclusive rewind (got %#v)", len(events), summarizeGeminiEvents(events))
	}
	if events[0].Actor != capsule.ActorUser || events[0].Kind != capsule.KindMessage {
		t.Fatalf("surviving event = %s/%s, want user/message", events[0].Actor, events[0].Kind)
	}
	if got := geminiEventText(events[0]); got != "First Gemini user turn" {
		t.Fatalf("surviving text = %q", got)
	}
	for _, ev := range events {
		body := geminiEventText(ev)
		if strings.Contains(body, "REWOUND_") || strings.Contains(body, "First Gemini model turn") {
			t.Fatalf("rewound content leaked into capsule: %q", body)
		}
	}
	if report.Events != 1 {
		t.Fatalf("report.Events = %d, want 1", report.Events)
	}

	// On-disk lines still contain the rewound turns (append-only marker).
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, keepOnDisk := range []string{"REWOUND_TURN_SHOULD_LEAVE_FILE", `"$rewindTo":"model-1"`} {
		if !strings.Contains(string(raw), keepOnDisk) {
			t.Fatalf("fixture missing on-disk evidence %q", keepOnDisk)
		}
	}
}

func TestGeminiReaderRewindUnknownIDIsNoopWithWarning(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "testdata", "handoff", "gemini", "rewind", "session-rewind-unknown.jsonl")
	events, report := parseGeminiFixture(t, path, "gemini-handoff-rewind-unknown")

	if len(events) != 2 {
		t.Fatalf("events = %d, want 2 (noop rewind)", len(events))
	}
	found := false
	for _, w := range report.Warnings {
		if w.Code == warningRewindTargetMissing {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing %s warning; got %#v", warningRewindTargetMissing, report.Warnings)
	}
}

func TestGeminiReaderLegacyAndJSONLEquivalent(t *testing.T) {
	t.Parallel()
	legacyPath := filepath.Join("..", "..", "testdata", "handoff", "gemini", "legacy-json", "session-shared.json")
	jsonlPath := filepath.Join("..", "..", "testdata", "handoff", "gemini", "jsonl", "session-shared.jsonl")

	legacyEvents, _ := parseGeminiFixture(t, legacyPath, "gemini-handoff-shared")
	jsonlEvents, _ := parseGeminiFixture(t, jsonlPath, "gemini-handoff-shared")

	if len(legacyEvents) != len(jsonlEvents) {
		t.Fatalf("event count legacy=%d jsonl=%d\nlegacy=%#v\njsonl=%#v",
			len(legacyEvents), len(jsonlEvents),
			summarizeGeminiEvents(legacyEvents), summarizeGeminiEvents(jsonlEvents))
	}
	for i := range legacyEvents {
		left, right := legacyEvents[i], jsonlEvents[i]
		if left.Actor != right.Actor || left.Kind != right.Kind || left.Portability != right.Portability {
			t.Fatalf("event[%d] actor/kind/portability mismatch: %#v vs %#v", i, summarizeGeminiEvent(left), summarizeGeminiEvent(right))
		}
		if left.NativeName != right.NativeName {
			t.Fatalf("event[%d] native name %q vs %q", i, left.NativeName, right.NativeName)
		}
		if geminiEventText(left) != geminiEventText(right) {
			t.Fatalf("event[%d] text mismatch:\n legacy=%q\n jsonl=%q", i, geminiEventText(left), geminiEventText(right))
		}
		if left.ContentHash != right.ContentHash {
			t.Fatalf("event[%d] content hash mismatch", i)
		}
		if left.Kind == capsule.KindToolCall && (left.CallID == "" || left.CallID != right.CallID) {
			t.Fatalf("event[%d] tool call id mismatch %q vs %q", i, left.CallID, right.CallID)
		}
		if left.Kind == capsule.KindToolResult && left.LinkedCallID != right.LinkedCallID {
			t.Fatalf("event[%d] linked call id mismatch", i)
		}
	}

	// Mapping expectations for the shared conversation.
	want := []struct {
		actor       capsule.Actor
		kind        capsule.Kind
		portability capsule.Portability
	}{
		{capsule.ActorUser, capsule.KindMessage, capsule.PortabilityExact},
		{capsule.ActorAssistant, capsule.KindMessage, capsule.PortabilityExact},
		{capsule.ActorAssistant, capsule.KindToolCall, capsule.PortabilityNormalized},
		{capsule.ActorTool, capsule.KindToolResult, capsule.PortabilityNormalized},
		{capsule.ActorUser, capsule.KindMessage, capsule.PortabilityExact},
		{capsule.ActorAssistant, capsule.KindMessage, capsule.PortabilityExact},
	}
	if len(jsonlEvents) != len(want) {
		t.Fatalf("shared events = %d, want %d", len(jsonlEvents), len(want))
	}
	for i, row := range want {
		if jsonlEvents[i].Actor != row.actor || jsonlEvents[i].Kind != row.kind || jsonlEvents[i].Portability != row.portability {
			t.Fatalf("shared[%d] = %#v, want %#v", i, summarizeGeminiEvent(jsonlEvents[i]), row)
		}
	}
}

func TestGeminiReaderExcludesSubagent(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "testdata", "handoff", "gemini", "jsonl", "session-subagent.jsonl")
	record := sessionindex.Record{
		Agent:      sessionindex.AgentGemini,
		ID:         "gemini-handoff-subagent",
		SourcePath: mustAbs(t, path),
	}
	var reader GeminiReader
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if !errors.Is(err, errGeminiSubagentSession) {
		t.Fatalf("Parse err = %v, want errGeminiSubagentSession", err)
	}
	if len(events) != 0 {
		t.Fatalf("subagent events = %#v, want none", summarizeGeminiEvents(events))
	}
	found := false
	for _, w := range report.Warnings {
		if w.Code == warningGeminiSubagent {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing subagent warning; got %#v", report.Warnings)
	}
}

func TestGeminiReaderParseDeterministic(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "testdata", "handoff", "gemini", "jsonl", "session-shared.jsonl")
	first, _ := parseGeminiFixture(t, path, "gemini-handoff-shared")
	second, _ := parseGeminiFixture(t, path, "gemini-handoff-shared")
	if len(first) != len(second) {
		t.Fatalf("lengths %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID || first[i].ContentHash != second[i].ContentHash {
			t.Fatalf("event[%d] not deterministic: id %q/%q hash %q/%q",
				i, first[i].ID, second[i].ID, first[i].ContentHash, second[i].ContentHash)
		}
	}
}

func TestGeminiReaderRegistered(t *testing.T) {
	t.Parallel()
	r, ok := Get(sessionindex.AgentGemini)
	if !ok {
		t.Fatal("gemini reader not registered")
	}
	if r.Name() != sessionindex.AgentGemini {
		t.Fatalf("Name() = %q", r.Name())
	}
	compat, err := r.Probe(context.Background(), sessionindex.Record{
		Agent:      sessionindex.AgentGemini,
		SourcePath: "session.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("Probe = %q, want supported", compat)
	}
}

func TestGeminiReaderNeverTouchesRealHome(t *testing.T) {
	t.Parallel()
	// Guardrail: reader only opens SourcePath from the record/boundary.
	var reader GeminiReader
	_, err := reader.Snapshot(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentGemini,
		ID:    "nope",
	})
	if err == nil {
		t.Fatal("expected error for empty SourcePath")
	}
	if strings.Contains(err.Error(), filepath.Join(".gemini")) {
		t.Fatalf("error must not reference real .gemini home: %v", err)
	}
}

func parseGeminiFixture(t *testing.T, relPath, sessionID string) ([]capsule.Event, ParseReport) {
	t.Helper()
	path := mustAbs(t, relPath)
	record := sessionindex.Record{
		Agent:      sessionindex.AgentGemini,
		ID:         sessionID,
		SourcePath: path,
	}
	var reader GeminiReader
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot(%s): %v", path, err)
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse(%s): %v", path, err)
	}
	return events, report
}

func mustAbs(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture %s: %v", abs, err)
	}
	return abs
}

func geminiEventText(ev capsule.Event) string {
	parts := make([]string, 0, len(ev.Blocks))
	for _, b := range ev.Blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func summarizeGeminiEvents(events []capsule.Event) []map[string]string {
	out := make([]map[string]string, 0, len(events))
	for _, ev := range events {
		out = append(out, summarizeGeminiEvent(ev))
	}
	return out
}

func summarizeGeminiEvent(ev capsule.Event) map[string]string {
	return map[string]string{
		"actor":       string(ev.Actor),
		"kind":        string(ev.Kind),
		"portability": string(ev.Portability),
		"native":      ev.NativeType,
		"name":        ev.NativeName,
		"text":        geminiEventText(ev),
	}
}
