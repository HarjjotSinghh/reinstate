package capsule_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func TestCapsuleGoldensAreDeterministic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		sourceAgent    string
		destination    string
		sessionID      string
		adapterVersion string
		fixture        string
		reader         transcript.Reader
	}{
		{
			name:           "claude-to-codex",
			sourceAgent:    sessionindex.AgentClaude,
			destination:    sessionindex.AgentCodex,
			sessionID:      "session-syn-001",
			adapterVersion: "2.1.220",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "claude", "unknown-records", "projects",
				"-Users-fixture-user-code-demo", "session-syn-001.jsonl"),
			reader: &transcript.ClaudeReader{},
		},
		{
			name:           "codex-to-claude",
			sourceAgent:    sessionindex.AgentCodex,
			destination:    sessionindex.AgentClaude,
			sessionID:      "00000000-0000-4000-8000-00000000a101",
			adapterVersion: "0.145.0",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "codex", "unknown-records",
				"rollout-2026-08-01T15-00-00-00000000-0000-4000-8000-00000000a101.jsonl"),
			reader: &transcript.CodexReader{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			first := buildGoldenCapsule(t, tc.sourceAgent, tc.destination, tc.sessionID, tc.adapterVersion, tc.fixture, tc.reader)
			second := buildGoldenCapsule(t, tc.sourceAgent, tc.destination, tc.sessionID, tc.adapterVersion, tc.fixture, tc.reader)
			firstBytes := canonical(t, first)
			secondBytes := canonical(t, second)

			if first.Identity.ID != second.Identity.ID || !bytes.Equal(firstBytes, secondBytes) {
				t.Fatalf("identical parse boundaries produced different capsules: ids %q / %q", first.Identity.ID, second.Identity.ID)
			}
			assertPortableGolden(t, firstBytes)

			goldenPath := filepath.Join("..", "..", "testdata", "handoff", "golden", "capsule", tc.name+".json")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(goldenPath, firstBytes, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(firstBytes, want) {
				t.Fatalf("capsule golden drifted; regenerate with UPDATE_GOLDEN=1 go test ./internal/capsule -run TestCapsuleGoldensAreDeterministic/%s", tc.name)
			}
		})
	}
}

func buildGoldenCapsule(t *testing.T, sourceAgent, destination, sessionID, adapterVersion, fixture string, reader transcript.Reader) capsule.Capsule {
	t.Helper()

	rec := sessionindex.Record{ID: sessionID, Agent: sourceAgent, SourcePath: fixture}
	boundary, err := reader.Snapshot(context.Background(), rec)
	if err != nil {
		t.Fatal(err)
	}
	events, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatal(err)
	}
	events = transcript.LinkToolResults(events)
	included, sidecar, fidelity := handoff.Apply(handoff.PolicyBalanced, events)
	task := handoff.DeriveCheckpoint(handoff.CheckpointInput{Events: events})

	c := capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			SchemaVer: capsule.SchemaVersion,
			Parent: capsule.Parent{Agent: sourceAgent, SessionID: sessionID,
				ArtifactSHA256: boundary.SHA256, AdapterVersion: adapterVersion},
		},
		RawSource: capsule.RawSource{Agent: sourceAgent, SessionID: sessionID,
			ArtifactSHA256: boundary.SHA256, AdapterVersion: adapterVersion,
			ByteOffset: boundary.ByteOffset, SizeBytes: boundary.SizeBytes, Partial: boundary.Partial, Path: boundary.Path()},
		Task: task,
		Workspace: capsule.Workspace{ProjectID: "github.com/example/demo", Root: "${REPO:github.com/example/demo}",
			Branch: "fixture/golden", Head: "0123456789abcdef", Dirty: false},
		Conversation: capsule.Conversation{Events: included},
		Capabilities: handoff.DiffCapabilities(capability.Inventory{}, capability.Inventory{}, sourceAgent, destination),
		Security:     capsule.Security{SourceInstructionsAreUntrustedHistory: true},
		Fidelity:     fidelity,
		Projection:   capsule.Projection{Policy: string(handoff.PolicyBalanced)},
	}
	if len(sidecar) > 0 {
		c.Conversation.FullHistoryRef = "sidecar/events.jsonl"
		c.Projection.SidecarRef = "sidecar/events.jsonl"
	}
	for _, event := range included {
		c.Projection.IncludedEventIDs = append(c.Projection.IncludedEventIDs, event.ID)
	}
	id, err := capsule.ComputeID(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Identity.ID = id
	c.Identity.LineageRoot = id
	if err := capsule.Validate(c); err != nil {
		t.Fatal(err)
	}
	return c
}

func canonical(t *testing.T, c capsule.Capsule) []byte {
	t.Helper()
	b, err := capsule.CanonicalBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertPortableGolden(t *testing.T, body []byte) {
	t.Helper()
	if bytes.Contains(body, []byte{'\r'}) {
		t.Fatal("golden contains CR instead of LF-normalized bytes")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{repoRoot, filepath.ToSlash(repoRoot), "/Users/", "/home/", `C:\\Users\\`, "C:/Users/"} {
		if forbidden != "" && strings.Contains(string(body), forbidden) {
			t.Fatalf("golden contains absolute host path marker %q", forbidden)
		}
	}
}
