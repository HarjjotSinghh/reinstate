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

const kimiFixtureID = "session_01987654-3210-7890-abcd-ef0123456789"

func kimiFixture(t *testing.T, kind string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "handoff", "kimi", kind, "sessions",
		"wd_fixture-user_a1b2c3d4e5f6", kimiFixtureID)
}

func kimiScanRecord(root string) sessionindex.Record {
	return sessionindex.Record{
		ID:         kimiFixtureID,
		Agent:      sessionindex.AgentKimi,
		SourcePath: root,
		Workspace:  "/Users/fixture-user/code/demo",
		Project:    "demo",
	}
}

func kimiParse(t *testing.T, kind string) ([]capsule.Event, ParseReport, Boundary) {
	t.Helper()
	reader := NewKimiReader()
	record := kimiScanRecord(kimiFixture(t, kind))
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot(%s): %v", kind, err)
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse(%s): %v", kind, err)
	}
	return events, report, boundary
}

// kimiWantEvent pins one position of the parsed sequence. Empty fields are not
// asserted.
type kimiWantEvent struct {
	actor      capsule.Actor
	kind       capsule.Kind
	nativeType string
	nativeName string
	text       string // exact block text when set
	callID     string
	linkedCall string
	reason     string
}

func kimiAssertSequence(t *testing.T, events []capsule.Event, want []kimiWantEvent) {
	t.Helper()
	if len(events) != len(want) {
		for i, ev := range events {
			t.Logf("event[%02d] %s %s %s call=%q link=%q", i, ev.Actor, ev.Kind, ev.NativeType, ev.CallID, ev.LinkedCallID)
		}
		t.Fatalf("event count = %d, want %d", len(events), len(want))
	}
	for i, w := range want {
		ev := events[i]
		if ev.Actor != w.actor || ev.Kind != w.kind || ev.NativeType != w.nativeType {
			t.Fatalf("event[%d] = (%s, %s, %s), want (%s, %s, %s)",
				i, ev.Actor, ev.Kind, ev.NativeType, w.actor, w.kind, w.nativeType)
		}
		if w.nativeName != "" && ev.NativeName != w.nativeName {
			t.Fatalf("event[%d] NativeName = %q, want %q", i, ev.NativeName, w.nativeName)
		}
		if w.callID != "" && ev.CallID != w.callID {
			t.Fatalf("event[%d] CallID = %q, want %q", i, ev.CallID, w.callID)
		}
		if w.linkedCall != "" && ev.LinkedCallID != w.linkedCall {
			t.Fatalf("event[%d] LinkedCallID = %q, want %q", i, ev.LinkedCallID, w.linkedCall)
		}
		if w.reason != "" && ev.Reason != w.reason {
			t.Fatalf("event[%d] Reason = %q, want %q", i, ev.Reason, w.reason)
		}
		if w.text != "" {
			joined := ""
			for _, block := range ev.Blocks {
				joined += block.Text
			}
			if joined != w.text {
				t.Fatalf("event[%d] text = %q, want %q", i, joined, w.text)
			}
		}
		if ev.ID == "" || ev.ContentHash == "" {
			t.Fatalf("event[%d] missing id/hash: %#v", i, ev)
		}
		if ev.Portability != capsule.PortabilityExact && ev.Reason == "" {
			t.Fatalf("event[%d] non-exact event missing reason: %#v", i, ev)
		}
	}
}

// TestKimiReaderParsesNativeWire pins the shape Kimi Code CLI 0.36.1 writes at
// wire protocol 1.5: the assistant's text, tool calls, and tool results all
// arrive as context.append_loop_event, never as a role "assistant"
// context.append_message.
func TestKimiReaderParsesNativeWire(t *testing.T) {
	t.Parallel()
	reader := NewKimiReader()
	record := kimiScanRecord(kimiFixture(t, "basic"))
	compat, err := reader.Probe(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("Probe = %q, want supported", compat)
	}

	events, report, boundary := kimiParse(t, "basic")
	if boundary.Partial {
		t.Fatal("basic fixture is complete; Partial should be false")
	}
	if report.Events != len(events) {
		t.Fatalf("report.Events = %d, want %d", report.Events, len(events))
	}
	if report.UnknownRecords != 0 {
		t.Fatalf("UnknownRecords = %d, want 0 on the known vocabulary", report.UnknownRecords)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want none", report.Warnings)
	}

	const loop = "context.append_loop_event/"
	kimiAssertSequence(t, events, []kimiWantEvent{
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "metadata"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "profile.bind", reason: reasonKimiSourcePrompt},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "permission.set_mode"},
		{actor: capsule.ActorUser, kind: capsule.KindMessage, nativeType: "turn.prompt", text: "Bound the retry budget in agentcheck"},
		// The role "user" context.append_message that repeats turn.prompt is
		// deduplicated, so position 4 is the injected reminder, not a message.
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "context.append_message", reason: reasonKimiHarnessInjection},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "plugin.session_start"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "llm.tools_snapshot"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: loop + "step.begin"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "llm.request"},
		{actor: capsule.ActorAssistant, kind: capsule.KindMessage, nativeType: loop + "content.part", text: "Reading the current budget."},
		{actor: capsule.ActorAssistant, kind: capsule.KindMetadata, nativeType: loop + "content.part", nativeName: "thinking", reason: reasonKimiVendorPart},
		{actor: capsule.ActorAssistant, kind: capsule.KindToolCall, nativeType: loop + "tool.call", nativeName: "Read", callID: "call_fixture_1", reason: reasonKimiToolNormalized},
		{actor: capsule.ActorTool, kind: capsule.KindToolResult, nativeType: loop + "tool.result", linkedCall: "call_fixture_1", reason: reasonKimiToolResult},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: loop + "step.end"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "usage.record"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: loop + "step.begin"},
		{actor: capsule.ActorAssistant, kind: capsule.KindMessage, nativeType: loop + "content.part", text: "The first attempt allows two seconds."},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: loop + "step.end"},
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "turn.ended"},
	})

	for _, ev := range events {
		blob := ev.NativeType + ev.NativeName + ev.Reason
		for _, block := range ev.Blocks {
			blob += block.Text + block.Ref
		}
		if strings.Contains(blob, "NEVER_COPY") {
			t.Fatalf("payload leaked into the capsule: %#v", ev)
		}
	}

	again, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != len(events) {
		t.Fatalf("second parse event count = %d, want %d", len(again), len(events))
	}
	for i := range events {
		if again[i].ID != events[i].ID || again[i].ContentHash != events[i].ContentHash {
			t.Fatalf("parse not deterministic at %d", i)
		}
	}
}

// TestKimiToolCallCarriesTokenizedArguments checks that the tool call read out
// of a loop event keeps its arguments and that workspace paths inside them are
// tokenized before they reach the capsule.
func TestKimiToolCallCarriesTokenizedArguments(t *testing.T) {
	t.Parallel()
	events, _, _ := kimiParse(t, "basic")

	var call *capsule.Event
	for i := range events {
		if events[i].Kind == capsule.KindToolCall {
			call = &events[i]
			break
		}
	}
	if call == nil {
		t.Fatal("no tool call event was produced")
	}
	if len(call.Blocks) != 1 || call.Blocks[0].Type != capsule.BlockTypeToolInput {
		t.Fatalf("tool call blocks = %#v, want one tool_input block", call.Blocks)
	}
	args := call.Blocks[0].Text
	if !strings.Contains(args, "internal/agentcheck/agent.go") {
		t.Fatalf("tool call arguments lost the path: %q", args)
	}
	if strings.Contains(args, "/Users/fixture-user/code/demo") {
		t.Fatalf("tool call arguments kept an untokenized workspace path: %q", args)
	}
	if call.Portability != capsule.PortabilityNormalized {
		t.Fatalf("tool call portability = %q, want normalized", call.Portability)
	}
}

// TestKimiToolResultCarriesOutput checks the tool result body survives and
// stays linked to the call that produced it.
func TestKimiToolResultCarriesOutput(t *testing.T) {
	t.Parallel()
	events, _, _ := kimiParse(t, "basic")

	var result *capsule.Event
	for i := range events {
		if events[i].Kind == capsule.KindToolResult {
			result = &events[i]
			break
		}
	}
	if result == nil {
		t.Fatal("no tool result event was produced")
	}
	if len(result.Blocks) != 1 || result.Blocks[0].Type != capsule.BlockTypeToolOutput {
		t.Fatalf("tool result blocks = %#v, want one tool_output block", result.Blocks)
	}
	if !strings.Contains(result.Blocks[0].Text, "retryBudget = 2 * time.Second") {
		t.Fatalf("tool result lost its output: %q", result.Blocks[0].Text)
	}
	if result.LinkedCallID != "call_fixture_1" {
		t.Fatalf("tool result LinkedCallID = %q, want call_fixture_1", result.LinkedCallID)
	}
}

// TestKimiReaderParsesMigratedLegacyWire covers the other on-disk shape: a
// session migrated from the old kimi-cli store carries every message as a
// context.append_message at wire protocol 1.0 and has no turn.prompt at all.
func TestKimiReaderParsesMigratedLegacyWire(t *testing.T) {
	t.Parallel()
	events, report, _ := kimiParse(t, "legacy-migrated")
	if report.UnknownRecords != 0 {
		t.Fatalf("UnknownRecords = %d, want 0", report.UnknownRecords)
	}
	kimiAssertSequence(t, events, []kimiWantEvent{
		{actor: capsule.ActorHarness, kind: capsule.KindMetadata, nativeType: "metadata"},
		{actor: capsule.ActorUser, kind: capsule.KindMessage, nativeType: "context.append_message", text: "Bound the retry budget in agentcheck"},
		{actor: capsule.ActorAssistant, kind: capsule.KindMessage, nativeType: "context.append_message", text: "Reading the current budget."},
		{actor: capsule.ActorAssistant, kind: capsule.KindToolCall, nativeType: "context.append_message", nativeName: "read_file", callID: "call_legacy_1"},
		{actor: capsule.ActorTool, kind: capsule.KindToolResult, nativeType: "context.append_message", linkedCall: "call_legacy_1"},
		{actor: capsule.ActorAssistant, kind: capsule.KindMessage, nativeType: "context.append_message", text: "The first attempt allows two seconds."},
	})
}

// TestKimiContextRewriteIsReported checks that a wire log which discards
// history in place is surfaced rather than silently replayed in file order.
func TestKimiContextRewriteIsReported(t *testing.T) {
	t.Parallel()
	_, report, _ := kimiParse(t, "context-rewrite")

	var found bool
	for _, w := range report.Warnings {
		if w.Code == codeKimiContextRewritten {
			found = true
			if !strings.Contains(w.Message, "context.clear") {
				t.Fatalf("warning does not name the record: %q", w.Message)
			}
			if w.Agent != sessionindex.AgentKimi {
				t.Fatalf("warning agent = %q", w.Agent)
			}
		}
	}
	if !found {
		t.Fatalf("no %s warning; warnings = %v", codeKimiContextRewritten, report.Warnings)
	}

	// A fixture without a rewrite must not raise it.
	if _, clean, _ := kimiParse(t, "basic"); len(clean.Warnings) != 0 {
		t.Fatalf("basic fixture raised warnings: %v", clean.Warnings)
	}
}

func TestKimiUnknownRecordIsReferencedWithoutBody(t *testing.T) {
	t.Parallel()
	events, report, _ := kimiParse(t, "unknown-records")
	// One unknown top-level type and one unknown loop-event type.
	if report.UnknownRecords != 2 {
		t.Fatalf("UnknownRecords = %d, want 2", report.UnknownRecords)
	}
	var unknownTop, unknownLoop, foundPrompt bool
	for _, ev := range events {
		blob := ev.NativeType + ev.NativeName + ev.Reason
		for _, block := range ev.Blocks {
			blob += block.Text + block.Ref
		}
		if strings.Contains(blob, "NEVER_COPY") {
			t.Fatalf("payload leaked into the capsule: %#v", ev)
		}
		if ev.Kind == capsule.KindUnknown && ev.Portability == capsule.PortabilityReferenced &&
			ev.Reason == "unrecognized_record_type" {
			switch ev.NativeType {
			case "kimi.future_event":
				unknownTop = true
			case "context.append_loop_event/future.loop_event":
				unknownLoop = true
			}
		}
		if ev.Reason == reasonKimiSourcePrompt {
			foundPrompt = true
			if ev.Portability != capsule.PortabilityReferenced {
				t.Fatalf("profile.bind portability = %q", ev.Portability)
			}
		}
	}
	if !unknownTop {
		t.Fatal("missing referenced unknown top-level record")
	}
	if !unknownLoop {
		t.Fatal("missing referenced unknown loop event")
	}
	if !foundPrompt {
		t.Fatal("missing referenced profile.bind")
	}
}

func TestKimiSnapshotExcludesPartialTrailingRecord(t *testing.T) {
	t.Parallel()
	reader := NewKimiReader()
	record := kimiScanRecord(kimiFixture(t, "partial-final-record"))
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if !boundary.Partial {
		t.Fatal("truncated fixture must set Partial")
	}
	if boundary.ByteOffset >= boundary.SizeBytes {
		t.Fatalf("ByteOffset %d should stop before SizeBytes %d", boundary.ByteOffset, boundary.SizeBytes)
	}
	events, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		for _, block := range ev.Blocks {
			if strings.Contains(block.Text, "turn.prom") && ev.Kind == capsule.KindMessage {
				t.Fatalf("partial trailing record was parsed: %#v", ev)
			}
		}
	}
	// The complete prefix is still the full native turn.
	complete, _, _ := kimiParse(t, "basic")
	if len(events) != len(complete) {
		t.Fatalf("complete prefix produced %d events, want %d", len(events), len(complete))
	}
}

func TestKimiProbeFailsClosedOnUnknownSchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	session := filepath.Join(dir, "session_future")
	if err := os.MkdirAll(filepath.Join(session, "agents", "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session, "state.json"), []byte(`{"id":"x","version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session, "agents", "main", "wire.jsonl"), []byte(`{"type":"metadata","protocol_version":"1.5"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compat, err := NewKimiReader().Probe(context.Background(), sessionindex.Record{
		ID: "x", Agent: sessionindex.AgentKimi, SourcePath: session,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("Probe = %q, want unsupported", compat)
	}
}

func TestKimiProbeFailsClosedOnUnknownWireProtocol(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	session := filepath.Join(dir, "session_proto")
	if err := os.MkdirAll(filepath.Join(session, "agents", "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session, "state.json"), []byte(`{"id":"x","version":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session, "agents", "main", "wire.jsonl"), []byte(`{"type":"metadata","protocol_version":"9.0"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compat, err := NewKimiReader().Probe(context.Background(), sessionindex.Record{
		ID: "x", Agent: sessionindex.AgentKimi, SourcePath: session,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("Probe = %q, want unsupported", compat)
	}
}

func TestKimiReaderAcceptsSessionindexFixtures(t *testing.T) {
	t.Parallel()
	reader := NewKimiReader()
	for _, platform := range []string{"macos", "windows"} {
		root := filepath.Join("..", "..", "testdata", "sessionindex", "kimi", platform, "sessions",
			"wd_fixture-user_a1b2c3d4e5f6", kimiFixtureID)
		if platform == "windows" {
			root = filepath.Join("..", "..", "testdata", "sessionindex", "kimi", platform, "sessions",
				"wd_fixture-user_0f1e2d3c4b5a", "session_01912345-6789-7abc-def0-123456789abc")
		}
		record := sessionindex.Record{
			ID:         filepath.Base(root),
			Agent:      sessionindex.AgentKimi,
			SourcePath: root,
		}
		compat, err := reader.Probe(context.Background(), record)
		if err != nil {
			t.Fatalf("%s Probe: %v", platform, err)
		}
		if compat != CompatibilitySupported {
			t.Fatalf("%s Probe = %q, want supported", platform, compat)
		}
		boundary, err := reader.Snapshot(context.Background(), record)
		if err != nil {
			t.Fatalf("%s Snapshot: %v", platform, err)
		}
		events, _, err := reader.Parse(context.Background(), boundary)
		if err != nil {
			t.Fatalf("%s Parse: %v", platform, err)
		}
		if len(events) == 0 {
			t.Fatalf("%s produced no events", platform)
		}
	}
}

func TestKimiReaderRegistered(t *testing.T) {
	t.Parallel()
	reader, ok := Get(sessionindex.AgentKimi)
	if !ok || reader == nil || reader.Name() != sessionindex.AgentKimi {
		t.Fatalf("Get(kimi) = %#v, %v", reader, ok)
	}
}
