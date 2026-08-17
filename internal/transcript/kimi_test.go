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

func TestKimiReaderParsesBasicFixture(t *testing.T) {
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
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if boundary.Partial {
		t.Fatal("basic fixture is complete; Partial should be false")
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	if report.Events != len(events) || len(events) == 0 {
		t.Fatalf("events = %d report=%d", len(events), report.Events)
	}
	if report.UnknownRecords != 0 {
		t.Fatalf("UnknownRecords = %d, want 0 on the known vocabulary", report.UnknownRecords)
	}

	var foundUser, foundAssistant, foundTool bool
	for _, ev := range events {
		if ev.ID == "" || ev.ContentHash == "" {
			t.Fatalf("event missing id/hash: %#v", ev)
		}
		if ev.Portability != capsule.PortabilityExact && ev.Reason == "" {
			t.Fatalf("non-exact event missing reason: %#v", ev)
		}
		if ev.Kind == capsule.KindMessage && ev.Actor == capsule.ActorUser {
			for _, block := range ev.Blocks {
				if strings.Contains(block.Text, "Bound the retry budget") {
					foundUser = true
				}
			}
		}
		if ev.Kind == capsule.KindMessage && ev.Actor == capsule.ActorAssistant {
			foundAssistant = true
		}
		if ev.Kind == capsule.KindToolCall {
			foundTool = true
			if ev.Portability != capsule.PortabilityNormalized {
				t.Fatalf("tool call portability = %q", ev.Portability)
			}
		}
	}
	if !foundUser || !foundAssistant || !foundTool {
		t.Fatalf("missing conversation events user=%t assistant=%t tool=%t", foundUser, foundAssistant, foundTool)
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

func TestKimiUnknownRecordIsReferencedWithoutBody(t *testing.T) {
	t.Parallel()
	reader := NewKimiReader()
	record := kimiScanRecord(kimiFixture(t, "unknown-records"))
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	if report.UnknownRecords < 1 {
		t.Fatalf("UnknownRecords = %d, want >= 1", report.UnknownRecords)
	}
	var foundUnknown, foundPrompt bool
	for _, ev := range events {
		blob := ev.NativeType + ev.Reason
		for _, block := range ev.Blocks {
			blob += block.Text + block.Ref
		}
		if strings.Contains(blob, "NEVER_COPY") {
			t.Fatalf("payload leaked into the capsule: %#v", ev)
		}
		if ev.Kind == capsule.KindUnknown && ev.Portability == capsule.PortabilityReferenced && ev.Reason == "unrecognized_record_type" {
			foundUnknown = true
		}
		if ev.Reason == reasonKimiSourcePrompt {
			foundPrompt = true
			if ev.Portability != capsule.PortabilityReferenced {
				t.Fatalf("profile.bind portability = %q", ev.Portability)
			}
		}
	}
	if !foundUnknown {
		t.Fatal("missing referenced unknown record")
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
