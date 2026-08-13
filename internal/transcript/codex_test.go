package transcript

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestCodexReaderDualRepresentationDedup(t *testing.T) {
	t.Parallel()
	events, report := parseCodexFixture(t, "long-history")

	var users, assistants []string
	for _, ev := range events {
		if ev.Kind != capsule.KindMessage {
			continue
		}
		text := eventText(ev)
		switch ev.Actor {
		case capsule.ActorUser:
			users = append(users, text)
		case capsule.ActorAssistant:
			assistants = append(assistants, text)
		}
	}
	if len(users) != 200 {
		t.Fatalf("user messages = %d, want 200 (one per turn); got %#v", len(users), users[:min(5, len(users))])
	}
	if len(assistants) != 200 {
		t.Fatalf("assistant messages = %d, want 200; got %#v", len(assistants), assistants[:min(5, len(assistants))])
	}
	for _, forbidden := range []string{
		"ENVIRONMENT_DUMP_SHOULD_DROP",
		"DUPLICATE_ASSIST_SHOULD_DROP",
	} {
		for _, ev := range events {
			if strings.Contains(eventText(ev), forbidden) {
				t.Fatalf("duplicate response_item leaked %q into events", forbidden)
			}
		}
	}
	if report.Events != len(events) {
		t.Fatalf("report.Events = %d, len(events) = %d", report.Events, len(events))
	}
}

func TestCodexReaderForkKeepsFilenameIdentity(t *testing.T) {
	t.Parallel()
	reader := &CodexReader{}
	ctx := context.Background()

	sourcePath := fixturePath(t, "forks", "rollout-2026-08-01T11-00-00-00000000-0000-4000-8000-00000000bb01.jsonl")
	forkPath := fixturePath(t, "forks", "rollout-2026-08-01T11-05-00-00000000-0000-4000-8000-00000000cc01.jsonl")

	const (
		sourceID = "00000000-0000-4000-8000-00000000bb01"
		forkID   = "00000000-0000-4000-8000-00000000cc01"
	)

	sourceBoundary, err := reader.Snapshot(ctx, sessionindex.Record{
		Agent: sessionindex.AgentCodex, ID: "ignored-in-file-id", SourcePath: sourcePath,
	})
	if err != nil {
		t.Fatalf("source Snapshot: %v", err)
	}
	forkBoundary, err := reader.Snapshot(ctx, sessionindex.Record{
		Agent: sessionindex.AgentCodex, ID: sourceID, SourcePath: forkPath, // in-file id is source
	})
	if err != nil {
		t.Fatalf("fork Snapshot: %v", err)
	}
	if sourceBoundary.SessionID != sourceID {
		t.Fatalf("source SessionID = %q, want %q", sourceBoundary.SessionID, sourceID)
	}
	if forkBoundary.SessionID != forkID {
		t.Fatalf("fork SessionID = %q, want filename UUID %q (not in-file %q)", forkBoundary.SessionID, forkID, sourceID)
	}

	sourceEvents, _, err := reader.Parse(ctx, sourceBoundary)
	if err != nil {
		t.Fatalf("source Parse: %v", err)
	}
	forkEvents, _, err := reader.Parse(ctx, forkBoundary)
	if err != nil {
		t.Fatalf("fork Parse: %v", err)
	}

	sourceBlob := eventsBlob(sourceEvents)
	if strings.Contains(sourceBlob, "FORK-A-ONLY") {
		t.Fatal("source parse absorbed fork-only content")
	}
	forkBlob := eventsBlob(forkEvents)
	if !strings.Contains(forkBlob, "FORK-A-ONLY-PROMPT") {
		t.Fatal("fork parse missing fork-only prompt")
	}
	for _, ev := range forkEvents {
		if ev.Source.SessionID != forkID {
			t.Fatalf("fork event Source.SessionID = %q, want %q", ev.Source.SessionID, forkID)
		}
	}
}

func TestCodexReaderReasoningItemsOmitted(t *testing.T) {
	t.Parallel()
	events, _ := parseCodexFixture(t, "reasoning-items")

	var omitted int
	for _, ev := range events {
		if ev.Portability == capsule.PortabilityOmitted {
			omitted++
			if ev.Reason != "vendor_opaque_state" {
				t.Fatalf("omitted reason = %q, want vendor_opaque_state", ev.Reason)
			}
			if len(ev.Blocks) != 0 {
				t.Fatalf("omitted reasoning retained blocks: %#v", ev.Blocks)
			}
			if strings.Contains(eventText(ev), "HIDDEN_SUMMARY") ||
				strings.Contains(eventText(ev), "encrypted") {
				t.Fatal("reasoning payload body leaked into event text")
			}
		}
	}
	if omitted != 2 {
		t.Fatalf("omitted reasoning events = %d, want 2", omitted)
	}
	var visibleAssist int
	for _, ev := range events {
		if ev.Actor == capsule.ActorAssistant && ev.Kind == capsule.KindMessage &&
			ev.Portability == capsule.PortabilityExact &&
			strings.Contains(eventText(ev), "VISIBLE_ASSISTANT_ONLY") {
			visibleAssist++
		}
	}
	if visibleAssist != 1 {
		t.Fatalf("visible assistant messages = %d, want 1 (deduped)", visibleAssist)
	}
}

func TestCodexReaderParallelToolLinking(t *testing.T) {
	t.Parallel()
	events, _ := parseCodexFixture(t, "parallel-tools")

	calls := map[string]string{}
	results := map[string]string{}
	for _, ev := range events {
		switch ev.Kind {
		case capsule.KindToolCall:
			calls[ev.CallID] = eventText(ev)
		case capsule.KindToolResult:
			results[ev.LinkedCallID] = eventText(ev)
		}
	}
	if len(calls) != 2 || len(results) != 2 {
		t.Fatalf("calls=%d results=%d, want 2 each; calls=%v results=%v", len(calls), len(results), calls, results)
	}
	if results["call_alpha"] != "ALPHA_CONTENTS" {
		t.Fatalf("alpha result = %q", results["call_alpha"])
	}
	if results["call_beta"] != "BETA_CONTENTS" {
		t.Fatalf("beta result = %q", results["call_beta"])
	}
	// Interleaved completion order must not break call_id linking.
	if _, ok := results["call_alpha"]; !ok {
		t.Fatal("missing linked result for call_alpha")
	}
}

func TestCodexReaderPartialFinalRecord(t *testing.T) {
	t.Parallel()
	reader := &CodexReader{}
	path := fixturePath(t, "partial-final-record", "rollout-2026-08-01T14-00-00-00000000-0000-4000-8000-00000000ff01.jsonl")
	boundary, err := reader.Snapshot(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentCodex, SourcePath: path,
	})
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !boundary.Partial {
		t.Fatal("Partial = false, want true for trailing incomplete line")
	}
	events, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, ev := range events {
		if strings.Contains(eventText(ev), "PARTIAL_LINE_SHOULD_NOT_PARSE") {
			t.Fatal("partial trailing record surfaced in events")
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if boundary.ByteOffset >= info.Size() {
		t.Fatalf("ByteOffset %d should exclude trailing partial (size %d)", boundary.ByteOffset, info.Size())
	}
}

func TestCodexReaderUnknownRecords(t *testing.T) {
	t.Parallel()
	events, report := parseCodexFixture(t, "unknown-records")
	if report.UnknownRecords < 2 {
		t.Fatalf("UnknownRecords = %d, want >= 2", report.UnknownRecords)
	}
	var referencedUnknown int
	for _, ev := range events {
		if ev.Portability == capsule.PortabilityReferenced &&
			ev.Actor == capsule.ActorUnknown &&
			ev.Kind == capsule.KindUnknown {
			referencedUnknown++
			if ev.Reason != "unrecognized_record_type" {
				t.Fatalf("unknown reason = %q", ev.Reason)
			}
		}
	}
	if referencedUnknown < 2 {
		t.Fatalf("referenced unknown events = %d, want >= 2", referencedUnknown)
	}
}

func TestCodexSessionIDFromFilename(t *testing.T) {
	t.Parallel()
	got := codexSessionIDFromFilename("rollout-2026-01-02T03-04-05-00000000-0000-4000-8000-00000000a001.jsonl")
	want := "00000000-0000-4000-8000-00000000a001"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if codexSessionIDFromFilename("rollout-syn-001.jsonl") != "" {
		t.Fatal("non-uuid filename unexpectedly produced a session id")
	}
}

func TestCodexReaderProbe(t *testing.T) {
	t.Parallel()
	reader := &CodexReader{}
	path := fixturePath(t, "long-history", "rollout-2026-08-01T10-00-00-00000000-0000-4000-8000-00000000aa01.jsonl")
	compat, err := reader.Probe(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentCodex, SourcePath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("compat = %q, want supported", compat)
	}
	compat, err = reader.Probe(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentClaude, SourcePath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("claude agent compat = %q, want unsupported", compat)
	}
}

func TestCodexReaderCompactionIsSummarized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-compaction.jsonl")
	body := strings.Join([]string{
		`{"timestamp":"2026-08-01T12:00:00Z","type":"session_meta","payload":{"id":"00000000-0000-4000-8000-00000000cc01","cwd":"/Users/fixture-user/code/demo"}}`,
		`{"timestamp":"2026-08-01T12:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"continue the task"}}`,
		`{"timestamp":"2026-08-01T12:00:02Z","type":"event_msg","payload":{"type":"context_compacted","message":"Prior turns compacted to a vendor summary"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := &CodexReader{}
	boundary, err := reader.Snapshot(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentCodex, SourcePath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	events, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	var summaries int
	for _, ev := range events {
		if ev.Kind == capsule.KindSummary {
			summaries++
			if ev.Portability != capsule.PortabilitySummarized {
				t.Fatalf("compaction portability = %q, want summarized", ev.Portability)
			}
			if !strings.Contains(eventText(ev), "vendor summary") {
				t.Fatalf("compaction text = %q", eventText(ev))
			}
		}
	}
	if summaries != 1 {
		t.Fatalf("summarized compaction events = %d, want 1", summaries)
	}
}

func parseCodexFixture(t *testing.T, caseName string) ([]capsule.Event, ParseReport) {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "handoff", "codex", caseName)
	matches, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no fixtures in %s", dir)
	}
	// Prefer a single primary rollout; for forks tests call Snapshot directly.
	path := matches[0]
	reader := &CodexReader{}
	boundary, err := reader.Snapshot(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentCodex, SourcePath: path,
	})
	if err != nil {
		t.Fatalf("Snapshot(%s): %v", path, err)
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse(%s): %v", path, err)
	}
	return events, report
}

func fixturePath(t *testing.T, caseName, file string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "handoff", "codex", caseName, file)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture %s: %v", path, err)
	}
	return path
}

func eventText(ev capsule.Event) string {
	var parts []string
	for _, b := range ev.Blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func eventsBlob(events []capsule.Event) string {
	var b strings.Builder
	for _, ev := range events {
		b.WriteString(eventText(ev))
		b.WriteByte('\n')
	}
	return b.String()
}
