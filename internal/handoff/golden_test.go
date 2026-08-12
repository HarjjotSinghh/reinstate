package handoff

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func TestLongHistoryParseCapsuleProjectionUnderCeiling(t *testing.T) {
	fixture := filepath.Join(repoRoot(t), "testdata", "handoff", "claude", "long-history", "projects",
		"-Users-fixture-user-code-demo", "session-syn-001.jsonl")
	info, err := os.Stat(fixture)
	if err != nil {
		t.Fatal(err)
	}
	record := sessionindex.Record{
		ID: "session-syn-001", Agent: sessionindex.AgentClaude, SourcePath: fixture,
		SourceModTime: info.ModTime().UnixNano(), SourceSize: info.Size(),
	}
	reader := &transcript.ClaudeReader{}
	boundary, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	events, report, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	events = transcript.LinkToolResults(events)
	included, sidecar, fidelity := Apply(PolicyBalanced, events)
	task := DeriveCheckpoint(CheckpointInput{Events: events})
	c := capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{SchemaVer: capsule.SchemaVersion, Parent: capsule.Parent{
			Agent: record.Agent, SessionID: record.ID, ArtifactSHA256: boundary.SHA256, AdapterVersion: "2.1.220",
		}},
		RawSource: capsule.RawSource{
			Agent: record.Agent, SessionID: record.ID, ArtifactSHA256: boundary.SHA256, AdapterVersion: "2.1.220",
			ByteOffset: boundary.ByteOffset, SizeBytes: boundary.SizeBytes,
		},
		Task: task,
		Workspace: capsule.Workspace{
			ProjectID: "github.com/example/demo", Root: "${REPO:github.com/example/demo}",
		},
		Conversation: capsule.Conversation{Events: included},
		Security:     capsule.Security{SourceInstructionsAreUntrustedHistory: true},
		Fidelity:     fidelity,
		Projection:   capsule.Projection{Policy: string(PolicyBalanced)},
	}
	for _, event := range included {
		c.Projection.IncludedEventIDs = append(c.Projection.IncludedEventIDs, event.ID)
	}
	if len(sidecar) > 0 {
		c.Conversation.FullHistoryRef = "sidecar/events.jsonl"
		c.Projection.SidecarRef = "sidecar/events.jsonl"
	}
	c.Identity.ID, err = capsule.ComputeID(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Identity.LineageRoot = c.Identity.ID
	if _, err := capsule.CanonicalBytes(c); err != nil {
		t.Fatal(err)
	}
	projection, err := RenderProjection(c)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	if report.Events != 400 {
		t.Fatalf("events=%d want 400 (200 turns)", report.Events)
	}
	const projectionCeiling = 96 << 10
	if len(projection) > projectionCeiling {
		t.Fatalf("projection=%d bytes ceiling=%d", len(projection), projectionCeiling)
	}
	const wallClockCeiling = 2 * time.Second
	if elapsed > wallClockCeiling {
		t.Fatalf("200-turn parse+capsule+projection took %s, ceiling %s", elapsed, wallClockCeiling)
	}
	t.Logf("200-turn parse+capsule+projection: %s; projection=%d bytes; ceiling=%s", elapsed, len(projection), wallClockCeiling)
}
