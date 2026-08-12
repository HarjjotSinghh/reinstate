package transcript

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestClaudeReaderRegistered(t *testing.T) {
	t.Parallel()
	r, ok := Get("claude")
	if !ok || r == nil || r.Name() != "claude" {
		t.Fatalf("claude reader not registered: ok=%v r=%v", ok, r)
	}
}

func TestClaudeMappingRows(t *testing.T) {
	t.Parallel()
	r := &ClaudeReader{}

	t.Run("user_message_exact", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "compaction")
		ev := mustFind(t, events, capsule.ActorUser, capsule.KindMessage)
		if ev.Portability != capsule.PortabilityExact || ev.Reason != "" {
			t.Fatalf("user message = %+v", ev)
		}
		if len(ev.Blocks) == 0 || !strings.Contains(ev.Blocks[0].Text, "Synthetic compaction request") {
			t.Fatalf("user text missing: %+v", ev.Blocks)
		}
	})

	t.Run("is_meta_harness_metadata", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "unknown-records")
		ev := mustFindReason(t, events, reasonHarnessMeta)
		if ev.Actor != capsule.ActorHarness || ev.Kind != capsule.KindMetadata || ev.Portability != capsule.PortabilityReferenced {
			t.Fatalf("isMeta mapping = %+v", ev)
		}
	})

	t.Run("assistant_text_exact", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "compaction")
		ev := mustFind(t, events, capsule.ActorAssistant, capsule.KindMessage)
		if ev.Portability != capsule.PortabilityExact {
			t.Fatalf("assistant portability = %q", ev.Portability)
		}
	})

	t.Run("tool_use_normalized", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "parallel-tools")
		var calls []capsule.Event
		for _, ev := range events {
			if ev.Kind == capsule.KindToolCall {
				calls = append(calls, ev)
			}
		}
		if len(calls) != 2 {
			t.Fatalf("tool_call count = %d, want 2", len(calls))
		}
		ids := map[string]string{}
		for _, c := range calls {
			if c.Portability != capsule.PortabilityNormalized || c.Reason != reasonToolUseNormalized {
				t.Fatalf("tool_use = %+v", c)
			}
			if c.CallID == "" || c.NativeName == "" {
				t.Fatalf("tool_use missing id/name: %+v", c)
			}
			ids[c.CallID] = c.NativeName
		}
		if ids["call_a"] != "Read" || ids["call_b"] != "Read" {
			t.Fatalf("call ids = %#v", ids)
		}
	})

	t.Run("tool_result_links_and_is_error", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "parallel-tools")
		linked := map[string]capsule.Event{}
		for _, ev := range events {
			if ev.Kind == capsule.KindToolResult {
				linked[ev.LinkedCallID] = ev
			}
		}
		if linked["call_a"].LinkedCallID != "call_a" || linked["call_b"].LinkedCallID != "call_b" {
			t.Fatalf("parallel link map = %#v", linked)
		}
		errEv := linked["call_missing"]
		if len(errEv.Blocks) == 0 || !errEv.Blocks[0].IsError {
			t.Fatalf("is_error not preserved: %+v", errEv)
		}
		if errEv.Blocks[0].Meta["is_error"] != "true" {
			t.Fatalf("is_error meta missing: %+v", errEv.Blocks[0].Meta)
		}
	})

	t.Run("summary_summarized", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "compaction")
		ev := mustFind(t, events, capsule.ActorHarness, capsule.KindSummary)
		if ev.Portability != capsule.PortabilitySummarized || ev.Reason != reasonVendorCompaction {
			t.Fatalf("summary = %+v", ev)
		}
	})

	t.Run("system_instruction_referenced", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "unknown-records")
		ev := mustFindReason(t, events, reasonSourceInstruction)
		if ev.Actor != capsule.ActorHarness || ev.Kind != capsule.KindMetadata || ev.Portability != capsule.PortabilityReferenced {
			t.Fatalf("system mapping = %+v", ev)
		}
	})

	t.Run("unknown_record_referenced", func(t *testing.T) {
		t.Parallel()
		events, report := parseFixtureReport(t, r, "unknown-records")
		if report.UnknownRecords < 1 {
			t.Fatalf("UnknownRecords = %d, want >= 1", report.UnknownRecords)
		}
		ev := mustFind(t, events, capsule.ActorUnknown, capsule.KindUnknown)
		if ev.Portability != capsule.PortabilityReferenced || ev.Reason != "unrecognized_record_type" {
			t.Fatalf("unknown = %+v", ev)
		}
	})

	t.Run("attachment_path_referenced", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "attachments")
		var found bool
		for _, ev := range events {
			if ev.Kind == capsule.KindAttachment && ev.Portability == capsule.PortabilityReferenced {
				found = true
				if ev.Reason != reasonAttachmentReferenced {
					t.Fatalf("path attachment reason = %q", ev.Reason)
				}
				if len(ev.Blocks) == 0 || ev.Blocks[0].SHA256 == "" || ev.Blocks[0].Path != "" {
					t.Fatalf("path attachment blocks = %+v", ev.Blocks)
				}
			}
		}
		if !found {
			t.Fatal("missing referenced path attachment")
		}
	})

	t.Run("attachment_inline_omitted", func(t *testing.T) {
		t.Parallel()
		events := parseFixture(t, r, "attachments")
		var found bool
		for _, ev := range events {
			if ev.Kind == capsule.KindAttachment && ev.Portability == capsule.PortabilityOmitted {
				found = true
				if ev.Reason != reasonAttachmentUnavailable {
					t.Fatalf("inline attachment reason = %q", ev.Reason)
				}
				if len(ev.Blocks) == 0 || ev.Blocks[0].Meta["source"] != "inline" {
					t.Fatalf("inline attachment = %+v", ev.Blocks)
				}
			}
		}
		if !found {
			t.Fatal("missing omitted inline attachment")
		}
	})
}

func TestClaudeParallelToolLinks(t *testing.T) {
	t.Parallel()
	events := parseFixture(t, &ClaudeReader{}, "parallel-tools")
	calls := map[string]bool{}
	results := map[string]bool{}
	for _, ev := range events {
		switch ev.Kind {
		case capsule.KindToolCall:
			calls[ev.CallID] = true
		case capsule.KindToolResult:
			results[ev.LinkedCallID] = true
		}
	}
	for _, id := range []string{"call_a", "call_b"} {
		if !calls[id] || !results[id] {
			t.Fatalf("parallel link incomplete for %s: calls=%v results=%v", id, calls, results)
		}
	}
}

func TestClaudeSubagentsContentNeverAppears(t *testing.T) {
	t.Parallel()
	events := parseFixture(t, &ClaudeReader{}, "subagents")
	for _, ev := range events {
		for _, b := range ev.Blocks {
			if strings.Contains(b.Text, "SUBAGENT_ONLY_MARKER") {
				t.Fatalf("subagent content leaked into main parse: %+v", ev)
			}
		}
	}
	// Probe must refuse the subagents/ layout path.
	subPath := filepath.Join(repoRoot(t), "testdata", "handoff", "claude", "subagents", "projects", "-Users-fixture-user-code-demo", "subagents", "agent-syn-secret.jsonl")
	compat, err := (&ClaudeReader{}).Probe(context.Background(), sessionindex.Record{
		Agent:      "claude",
		ID:         "subagent-syn-001",
		SourcePath: subPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("subagent probe = %q, want UNSUPPORTED", compat)
	}
}

func TestClaudePartialFinalRecordExcluded(t *testing.T) {
	t.Parallel()
	r := &ClaudeReader{}
	rec := fixtureRecord(t, "partial-final-record")
	b, err := r.Snapshot(context.Background(), rec)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !b.Partial {
		t.Fatal("Partial = false, want true")
	}
	events, _, err := r.Parse(context.Background(), b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, ev := range events {
		for _, block := range ev.Blocks {
			if strings.Contains(block.Text, "TRUNCATED_PARTIAL_ONLY") {
				t.Fatalf("partial record surfaced: %+v", ev)
			}
		}
	}
}

func TestClaudeLongHistoryUnderPerfCeiling(t *testing.T) {
	t.Parallel()
	r := &ClaudeReader{}
	rec := fixtureRecord(t, "long-history")
	b, err := r.Snapshot(context.Background(), rec)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	start := time.Now()
	events, report, err := r.Parse(context.Background(), b)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if report.Events != 400 {
		t.Fatalf("Events = %d, want 400 (200 turns)", report.Events)
	}
	if len(events) != 400 {
		t.Fatalf("len(events) = %d, want 400", len(events))
	}
	const ceiling = 2 * time.Second
	if elapsed > ceiling {
		t.Fatalf("200-turn parse took %s, ceiling %s", elapsed, ceiling)
	}
}

func TestClaudeParseDeterministicIDsAndHashes(t *testing.T) {
	t.Parallel()
	r := &ClaudeReader{}
	rec := fixtureRecord(t, "parallel-tools")
	b, err := r.Snapshot(context.Background(), rec)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	a, _, err := r.Parse(context.Background(), b)
	if err != nil {
		t.Fatalf("Parse a: %v", err)
	}
	c, _, err := r.Parse(context.Background(), b)
	if err != nil {
		t.Fatalf("Parse b: %v", err)
	}
	if len(a) != len(c) {
		t.Fatalf("event count mismatch %d vs %d", len(a), len(c))
	}
	for i := range a {
		if a[i].ID != c[i].ID {
			t.Fatalf("event %d ID mismatch %q vs %q", i, a[i].ID, c[i].ID)
		}
		if a[i].ContentHash != c[i].ContentHash {
			t.Fatalf("event %d ContentHash mismatch %q vs %q", i, a[i].ContentHash, c[i].ContentHash)
		}
	}
}

func TestClaudeProbeVersionGate(t *testing.T) {
	t.Parallel()
	r := &ClaudeReader{}
	ctx := context.Background()

	supported := fixtureRecord(t, "compaction")
	compat, err := r.Probe(ctx, supported)
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("supported fixture probe = %q", compat)
	}

	dir := t.TempDir()
	projects := filepath.Join(dir, "projects", "fixture-project")
	if err := os.MkdirAll(projects, 0o700); err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(projects, "session.jsonl")
	if err := os.WriteFile(session, []byte(`{"type":"user","message":{"content":"x"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "version"), []byte("2.1.228\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	compat, err = r.Probe(ctx, sessionindex.Record{Agent: "claude", ID: "session", SourcePath: session})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUntested {
		t.Fatalf("outside-range probe = %q, want UNTESTED", compat)
	}

	bad := filepath.Join(dir, "not-projects", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(bad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	compat, err = r.Probe(ctx, sessionindex.Record{Agent: "claude", ID: "session", SourcePath: bad})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("bad layout probe = %q, want UNSUPPORTED", compat)
	}
}

func parseFixture(t *testing.T, r *ClaudeReader, name string) []capsule.Event {
	t.Helper()
	events, _ := parseFixtureReport(t, r, name)
	return events
}

func parseFixtureReport(t *testing.T, r *ClaudeReader, name string) ([]capsule.Event, ParseReport) {
	t.Helper()
	rec := fixtureRecord(t, name)
	b, err := r.Snapshot(context.Background(), rec)
	if err != nil {
		t.Fatalf("Snapshot(%s): %v", name, err)
	}
	events, report, err := r.Parse(context.Background(), b)
	if err != nil {
		t.Fatalf("Parse(%s): %v", name, err)
	}
	return events, report
}

func fixtureRecord(t *testing.T, name string) sessionindex.Record {
	t.Helper()
	path := filepath.Join(
		repoRoot(t), "testdata", "handoff", "claude", name,
		"projects", "-Users-fixture-user-code-demo", "session-syn-001.jsonl",
	)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return sessionindex.Record{
		Key:           "claude:00000000-0000-4000-8000-000000000001",
		ID:            "00000000-0000-4000-8000-000000000001",
		Agent:         "claude",
		SourcePath:    path,
		SourceModTime: info.ModTime().UnixNano(),
		SourceSize:    info.Size(),
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from test working directory")
		}
		dir = parent
	}
}

func mustFind(t *testing.T, events []capsule.Event, actor capsule.Actor, kind capsule.Kind) capsule.Event {
	t.Helper()
	for _, ev := range events {
		if ev.Actor == actor && ev.Kind == kind {
			return ev
		}
	}
	t.Fatalf("no event with actor=%s kind=%s in %d events", actor, kind, len(events))
	return capsule.Event{}
}

func mustFindReason(t *testing.T, events []capsule.Event, reason string) capsule.Event {
	t.Helper()
	for _, ev := range events {
		if ev.Reason == reason {
			return ev
		}
	}
	t.Fatalf("no event with reason=%s", reason)
	return capsule.Event{}
}
