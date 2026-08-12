package transcript

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestOpenCodeStorageOrdersMessagesDeterministically(t *testing.T) {
	t.Parallel()

	root := handoffOpenCodeStorageRoot(t)
	reader := &OpenCodeReader{DataRoot: root}
	record := sessionindex.Record{
		ID:    "ses_fixture001",
		Agent: sessionindex.AgentOpenCode,
	}

	compat, err := reader.Probe(context.Background(), record)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("Probe = %q, want SUPPORTED", compat)
	}

	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if isOpenCodeMetadataBoundary(boundary) {
		t.Fatal("expected storage boundary, got metadata sentinel")
	}

	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if report.Events != 2 {
		t.Fatalf("Events = %d, want 2", report.Events)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	// Lexicographic message IDs: msg_fixtureasst001 before msg_fixtureuser001.
	if events[0].Source.RecordKey != "msg_fixtureasst001" {
		t.Fatalf("first record = %q, want msg_fixtureasst001", events[0].Source.RecordKey)
	}
	if events[1].Source.RecordKey != "msg_fixtureuser001" {
		t.Fatalf("second record = %q, want msg_fixtureuser001", events[1].Source.RecordKey)
	}
	if events[0].Actor != capsule.ActorAssistant || events[1].Actor != capsule.ActorUser {
		t.Fatalf("actors = %q/%q", events[0].Actor, events[1].Actor)
	}
	if got := events[0].Blocks[0].Text; got != "Synthetic OpenCode assistant reply" {
		t.Fatalf("assistant text = %q", got)
	}
	if got := events[1].Blocks[0].Text; got != "Synthetic OpenCode user prompt for local indexing" {
		t.Fatalf("user text = %q", got)
	}

	events2, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse again: %v", err)
	}
	if events[0].ID != events2[0].ID || events[0].ContentHash != events2[0].ContentHash {
		t.Fatal("repeat Parse produced different event identity")
	}
}

func TestOpenCodeUnknownSchemaFallsBackToMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataRoot := filepath.Join(dir, "opencode")
	storage := filepath.Join(dataRoot, "storage")
	sessionID := "ses_unknownver"
	mustWrite(t, filepath.Join(storage, "session", "proj_x", sessionID+".json"), `{
  "id": "`+sessionID+`",
  "projectID": "proj_x",
  "version": "999",
  "time": {"created": 1, "updated": 2}
}`)
	mustWrite(t, filepath.Join(storage, "message", sessionID, "msg_a.json"), `{
  "id": "msg_a",
  "sessionID": "`+sessionID+`",
  "role": "user",
  "time": {"created": 1},
  "agent": "build",
  "model": {"providerID": "anthropic", "modelID": "claude"}
}`)

	listJSON, err := os.ReadFile(handoffOpenCodeMetadataList(t))
	if err != nil {
		t.Fatalf("read metadata fixture: %v", err)
	}
	runner := sessionindex.CommandRunnerFunc(func(_ context.Context, executable string, args ...string) ([]byte, error) {
		if executable != "opencode" {
			t.Fatalf("executable = %q", executable)
		}
		want := []string{"session", "list", "--format", "json"}
		if len(args) != len(want) {
			t.Fatalf("args = %#v", args)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args = %#v", args)
			}
		}
		return listJSON, nil
	})

	reader := &OpenCodeReader{DataRoot: dataRoot, Runner: runner}
	record := sessionindex.Record{ID: sessionID, Agent: sessionindex.AgentOpenCode}

	compat, err := reader.Probe(context.Background(), record)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("Probe = %q, want SUPPORTED (metadata fallback)", compat)
	}

	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !isOpenCodeMetadataBoundary(boundary) {
		t.Fatalf("path = %q, want metadata sentinel", boundary.Path())
	}

	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0 on fallback", len(events))
	}
	if len(report.Warnings) == 0 || report.Warnings[0].Code != reasonSourceBodiesUnavailable {
		t.Fatalf("warnings = %#v, want source_bodies_unavailable", report.Warnings)
	}
}

func TestOpenCodeMissingMessageTreeFallsBack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataRoot := filepath.Join(dir, "opencode")
	// Data root exists with storage/ but no storage/message/ (SQLite-only).
	if err := os.MkdirAll(filepath.Join(dataRoot, "storage", "session"), 0o700); err != nil {
		t.Fatal(err)
	}
	listJSON := []byte(`[{"id":"ses_sqlite_only","title":"sqlite only","directory":"/Users/fixture-user/code/demo"}]`)
	runner := sessionindex.CommandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		return listJSON, nil
	})
	reader := &OpenCodeReader{DataRoot: dataRoot, Runner: runner}
	record := sessionindex.Record{ID: "ses_sqlite_only", Agent: sessionindex.AgentOpenCode}

	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !isOpenCodeMetadataBoundary(boundary) {
		t.Fatal("expected metadata fallback when message tree absent")
	}
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil || len(events) != 0 {
		t.Fatalf("Parse events=%d err=%v", len(events), err)
	}
	if len(report.Warnings) == 0 || report.Warnings[0].Code != reasonSourceBodiesUnavailable {
		t.Fatalf("warnings = %#v", report.Warnings)
	}
}

func TestOpenCodeMetadataFallbackCapsuleValidates(t *testing.T) {
	t.Parallel()

	reader := NewOpenCodeReader(nil)
	c := reader.MetadataFallbackCapsule("ses_fixture001")
	if err := capsule.Validate(c); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if c.Fidelity.Overall != capsule.PortabilityOmitted {
		t.Fatalf("Overall = %q, want omitted", c.Fidelity.Overall)
	}
	found := false
	for _, comp := range c.Fidelity.Components {
		if comp.Name == openCodeConversationComponent {
			found = true
			if comp.Portability != capsule.PortabilityOmitted {
				t.Fatalf("conversation portability = %q", comp.Portability)
			}
			if comp.Reason != reasonSourceBodiesUnavailable {
				t.Fatalf("reason = %q, want %q", comp.Reason, reasonSourceBodiesUnavailable)
			}
		}
	}
	if !found {
		t.Fatal("missing conversation fidelity component")
	}
	if len(c.Conversation.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(c.Conversation.Events))
	}
}

func TestResolveOpenCodeDataRootWindowsXDG(t *testing.T) {
	t.Parallel()

	got, err := ResolveOpenCodeDataRoot(
		func(string) string { return "" },
		func() (string, error) { return `C:\Users\fixture-user`, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(`C:\Users\fixture-user`, ".local", "share", "opencode")
	if got != want {
		t.Fatalf("got %q, want %q (XDG, not LOCALAPPDATA)", got, want)
	}

	got, err = ResolveOpenCodeDataRoot(
		func(key string) string {
			if key == "XDG_DATA_HOME" {
				return `D:\xdg-data`
			}
			return ""
		},
		func() (string, error) { return `C:\Users\fixture-user`, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join(`D:\xdg-data`, "opencode")
	if got != want {
		t.Fatalf("XDG override got %q, want %q", got, want)
	}
}

func TestOpenCodeReaderRegistered(t *testing.T) {
	t.Parallel()
	r, ok := Get(sessionindex.AgentOpenCode)
	if !ok || r == nil || r.Name() != sessionindex.AgentOpenCode {
		t.Fatalf("Get(opencode) = (%v, %v)", r, ok)
	}
}

func TestOpenCodeUnrecognizedRoleFailsClosedToMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataRoot := filepath.Join(dir, "opencode")
	storage := filepath.Join(dataRoot, "storage")
	sessionID := "ses_badrole"
	mustWrite(t, filepath.Join(storage, "session", "proj_x", sessionID+".json"), `{
  "id": "`+sessionID+`",
  "version": "1",
  "time": {"created": 1, "updated": 2}
}`)
	mustWrite(t, filepath.Join(storage, "message", sessionID, "msg_x.json"), `{
  "id": "msg_x",
  "sessionID": "`+sessionID+`",
  "role": "narrator",
  "time": {"created": 1}
}`)
	runner := sessionindex.CommandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`[{"id":"` + sessionID + `"}]`), nil
	})
	reader := &OpenCodeReader{DataRoot: dataRoot, Runner: runner}
	boundary, err := reader.Snapshot(context.Background(), sessionindex.Record{ID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	if !isOpenCodeMetadataBoundary(boundary) {
		t.Fatal("unrecognized role must not stay on storage tier")
	}
}

func handoffOpenCodeStorageRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "handoff", "opencode", "storage")
	if _, err := os.Stat(filepath.Join(root, "storage", "message", "ses_fixture001")); err != nil {
		t.Fatalf("storage fixture missing: %v", err)
	}
	return root
}

func handoffOpenCodeMetadataList(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "handoff", "opencode", "metadata-only", "session-list.json")
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(body)) {
		t.Fatalf("invalid json written to %s", path)
	}
}
