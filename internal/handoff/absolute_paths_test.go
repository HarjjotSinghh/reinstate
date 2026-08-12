package handoff

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

// TestAbsoluteToolPathsSurviveCapsuleCanonicalization reproduces the second
// rc.1 blocker end to end at the capsule boundary: a source session whose tool
// input carries an absolute path used to reach capsule canonicalization
// verbatim through task.files_touched_per_transcript and fail with
// `capsule: absolute filesystem path is not allowed: "<workspace>/calc.go"`.
func TestAbsoluteToolPathsSurviveCapsuleCanonicalization(t *testing.T) {
	// Only the string is used for ${HOME} tokenization; no home tree is read.
	t.Setenv("HOME", "/Users/fixture-user")
	t.Setenv("USERPROFILE", "/Users/fixture-user")

	tests := []struct {
		agent   string
		reader  func() transcript.Reader
		fixture string
	}{
		{
			agent:  sessionindex.AgentClaude,
			reader: func() transcript.Reader { return &transcript.ClaudeReader{} },
			fixture: filepath.Join("claude", "absolute-paths", "projects",
				"-Users-fixture-user-code-demo", "session-syn-001.jsonl"),
		},
		{
			agent:  sessionindex.AgentCodex,
			reader: func() transcript.Reader { return &transcript.CodexReader{} },
			fixture: filepath.Join("codex", "absolute-paths",
				"rollout-2026-08-01T16-00-00-00000000-0000-4000-8000-00000000ab01.jsonl"),
		},
	}

	for _, test := range tests {
		t.Run(test.agent, func(t *testing.T) {
			rec := sessionindex.Record{
				ID: "absolute-paths", Agent: test.agent,
				Project:    "github.com/example/demo",
				Workspace:  "/Users/fixture-user/code/demo",
				SourcePath: filepath.Join(repoRoot(t), "testdata", "handoff", test.fixture),
			}
			reader := test.reader()
			boundary, err := reader.Snapshot(context.Background(), rec)
			if err != nil {
				t.Fatal(err)
			}
			events, _, err := reader.Parse(context.Background(), boundary)
			if err != nil {
				t.Fatal(err)
			}
			events = transcript.LinkToolResults(events)

			task := DeriveCheckpoint(CheckpointInput{Events: events})
			for _, item := range task.FilesTouchedPerTranscript.Items {
				if strings.HasPrefix(item, "/") || strings.Contains(item, "/Users/fixture-user") {
					t.Fatalf("files_touched_per_transcript kept an absolute path: %q", item)
				}
			}

			included, _, fidelity := Apply(PolicyBalanced, events)
			c := capsule.Capsule{
				Schema: capsule.Schema,
				Identity: capsule.Identity{SchemaVer: capsule.SchemaVersion, Parent: capsule.Parent{
					Agent: rec.Agent, SessionID: rec.ID, ArtifactSHA256: boundary.SHA256,
					AdapterVersion: "unknown",
				}},
				RawSource: capsule.RawSource{
					Agent: rec.Agent, SessionID: rec.ID, ArtifactSHA256: boundary.SHA256,
					AdapterVersion: "unknown", ByteOffset: boundary.ByteOffset, SizeBytes: boundary.SizeBytes,
				},
				Task: task,
				Workspace: capsule.Workspace{
					ProjectID: "github.com/example/demo",
					Root:      "${REPO:github.com/example/demo}",
					Path:      rec.Workspace,
				},
				Conversation: capsule.Conversation{Events: included},
				Security:     capsule.Security{SourceInstructionsAreUntrustedHistory: true},
				Fidelity:     fidelity,
				Projection:   capsule.Projection{Policy: string(PolicyBalanced)},
			}
			if _, err := capsule.CanonicalBytes(c); err != nil {
				t.Fatalf("capsule canonicalization rejected a reader-emitted value: %v", err)
			}
			if err := capsule.Validate(c); err != nil {
				t.Fatalf("capsule validate: %v", err)
			}
		})
	}
}
