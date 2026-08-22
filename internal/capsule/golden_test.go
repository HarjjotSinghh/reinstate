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

	sources := []struct {
		agent          string
		sessionID      string
		adapterVersion string
		fixture        string
		newReader      func() transcript.Reader
	}{
		{
			agent:          sessionindex.AgentClaude,
			sessionID:      "session-syn-001",
			adapterVersion: "2.1.220",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "claude", "unknown-records", "projects",
				"-Users-fixture-user-code-demo", "session-syn-001.jsonl"),
			newReader: func() transcript.Reader { return &transcript.ClaudeReader{} },
		},
		{
			agent:          sessionindex.AgentCodex,
			sessionID:      "00000000-0000-4000-8000-00000000a101",
			adapterVersion: "0.145.0",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "codex", "unknown-records",
				"rollout-2026-08-01T15-00-00-00000000-0000-4000-8000-00000000a101.jsonl"),
			newReader: func() transcript.Reader { return &transcript.CodexReader{} },
		},
		{
			agent:          sessionindex.AgentGemini,
			sessionID:      "gemini-handoff-shared",
			adapterVersion: "0.28.2",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "gemini", "jsonl",
				"session-shared.jsonl"),
			newReader: func() transcript.Reader { return &transcript.GeminiReader{} },
		},
		{
			agent:          sessionindex.AgentOpenCode,
			sessionID:      "ses_fixture001",
			adapterVersion: "1",
			fixture:        filepath.Join("..", "..", "testdata", "handoff", "opencode", "storage"),
			newReader: func() transcript.Reader {
				return &transcript.OpenCodeReader{DataRoot: filepath.Join("..", "..", "testdata", "handoff", "opencode", "storage")}
			},
		},
		{
			agent:          sessionindex.AgentGrok,
			sessionID:      "01987654-basic-0000-0000-000000000001",
			adapterVersion: "1",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "grok", "basic", "sessions",
				"%2FUsers%2Ffixture-user%2Fcode%2Fdemo", "01987654-basic-0000-0000-000000000001"),
			newReader: func() transcript.Reader { return transcript.NewGrokReader() },
		},
		{
			agent:          sessionindex.AgentKimi,
			sessionID:      "session_01987654-3210-7890-abcd-ef0123456789",
			adapterVersion: "1",
			fixture: filepath.Join("..", "..", "testdata", "handoff", "kimi", "basic", "sessions",
				"wd_fixture-user_a1b2c3d4e5f6", "session_01987654-3210-7890-abcd-ef0123456789"),
			newReader: func() transcript.Reader { return transcript.NewKimiReader() },
		},
	}

	for _, source := range sources {
		for _, destination := range []string{sessionindex.AgentClaude, sessionindex.AgentCodex} {
			if source.agent == destination {
				continue
			}
			name := source.agent + "-to-" + destination
			t.Run(name, func(t *testing.T) {
				reader := source.newReader()
				first := buildGoldenCapsule(t, source.agent, destination, source.sessionID, source.adapterVersion, source.fixture, reader)
				second := buildGoldenCapsule(t, source.agent, destination, source.sessionID, source.adapterVersion, source.fixture, reader)
				firstBytes := canonical(t, first)
				secondBytes := canonical(t, second)

				if first.Identity.ID != second.Identity.ID || !bytes.Equal(firstBytes, secondBytes) {
					t.Fatalf("identical parse boundaries produced different capsules: ids %q / %q", first.Identity.ID, second.Identity.ID)
				}
				assertPortableGolden(t, firstBytes)

				goldenPath := filepath.Join("..", "..", "testdata", "handoff", "golden", "capsule", name+".json")
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
					t.Fatalf("capsule golden drifted; regenerate with UPDATE_GOLDEN=1 go test ./internal/capsule -run TestCapsuleGoldensAreDeterministic/%s", name)
				}
			})
		}
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

	security := capsule.Security{SourceInstructionsAreUntrustedHistory: true}
	if forced, ok := reader.(interface{ ForcedSecurity() capsule.Security }); ok {
		security = forced.ForcedSecurity()
	}
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
		Security:     security,
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
