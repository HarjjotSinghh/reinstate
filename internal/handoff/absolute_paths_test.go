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

type pathFixture struct {
	agent   string
	reader  func() transcript.Reader
	fixture string
}

func absolutePathFixtures() []pathFixture {
	return []pathFixture{
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
}

func slashCommandFixtures() []pathFixture {
	return []pathFixture{
		{
			agent:  sessionindex.AgentClaude,
			reader: func() transcript.Reader { return &transcript.ClaudeReader{} },
			fixture: filepath.Join("claude", "slash-commands", "projects",
				"-Users-fixture-user-code-demo", "session-syn-002.jsonl"),
		},
		{
			agent:  sessionindex.AgentCodex,
			reader: func() transcript.Reader { return &transcript.CodexReader{} },
			fixture: filepath.Join("codex", "slash-commands",
				"rollout-2026-08-01T17-00-00-00000000-0000-4000-8000-00000000ac01.jsonl"),
		},
	}
}

// buildFixtureCapsule runs the real reader, checkpoint derivation, and policy
// projection over a synthetic fixture and assembles the capsule the pipeline
// would assemble. It never reads a vendor tree outside testdata.
func buildFixtureCapsule(t *testing.T, f pathFixture) (capsule.Capsule, capsule.Task) {
	t.Helper()
	// Only the string is used for ${HOME} tokenization; no home tree is read.
	t.Setenv("HOME", "/Users/fixture-user")
	t.Setenv("USERPROFILE", "/Users/fixture-user")

	rec := sessionindex.Record{
		ID: "path-fixture", Agent: f.agent,
		Project:    "github.com/example/demo",
		Workspace:  "/Users/fixture-user/code/demo",
		SourcePath: filepath.Join(repoRoot(t), "testdata", "handoff", f.fixture),
	}
	reader := f.reader()
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
	return c, task
}

// TestAbsoluteToolPathsSurviveCapsuleCanonicalization reproduces the second
// rc.1 blocker end to end at the capsule boundary: a source session whose tool
// input carries an absolute path used to reach capsule canonicalization
// verbatim through task.files_touched_per_transcript and fail with
// `capsule: absolute filesystem path is not allowed: "<workspace>/calc.go"`.
func TestAbsoluteToolPathsSurviveCapsuleCanonicalization(t *testing.T) {
	for _, test := range absolutePathFixtures() {
		t.Run(test.agent, func(t *testing.T) {
			c, task := buildFixtureCapsule(t, test)
			for _, item := range task.FilesTouchedPerTranscript.Items {
				if strings.HasPrefix(item, "/") || strings.Contains(item, "/Users/fixture-user") {
					t.Fatalf("files_touched_per_transcript kept an absolute path: %q", item)
				}
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

// TestSlashCommandsAndProsePathsCompleteTheHandoff is the rc.1 prose blocker:
// canonicalization judged every string in the capsule as a possible path, so a
// session whose first message was `/init do the thing` — or any message naming
// an absolute path — aborted the handoff. Message bodies are prose and the
// capsule must carry them exactly as the user wrote them.
func TestSlashCommandsAndProsePathsCompleteTheHandoff(t *testing.T) {
	for _, test := range slashCommandFixtures() {
		t.Run(test.agent, func(t *testing.T) {
			c, task := buildFixtureCapsule(t, test)

			raw, err := capsule.CanonicalBytes(c)
			if err != nil {
				t.Fatalf("capsule canonicalization rejected prose: %v", err)
			}
			if err := capsule.Validate(c); err != nil {
				t.Fatalf("capsule validate: %v", err)
			}

			// The prose must survive verbatim, not merely survive.
			for _, want := range []string{
				"/init do the thing",
				"/compact",
				"not in /etc/fixture-hosts like last time",
				"/Users/fixture-user/code/demo/calc.go is the file I meant",
			} {
				if !strings.Contains(strings.Join(task.RecentUserMessages.Items, "\n"), want) {
					t.Fatalf("task.recent_user_messages lost prose %q: %v", want, task.RecentUserMessages.Items)
				}
				if !strings.Contains(string(raw), want) {
					t.Fatalf("canonical bytes lost prose %q", want)
				}
			}

			// Prose is not a path field: nothing prose-shaped may be lifted into
			// the capsule's file references.
			for _, item := range task.FilesTouchedPerTranscript.Items {
				if strings.HasPrefix(item, "/") {
					t.Fatalf("files_touched_per_transcript kept an absolute path: %q", item)
				}
			}
		})
	}
}

// TestUntokenizedPathFieldsStillAbortTheHandoff is the invariant the prose fix
// must not weaken. Each mutation is what a reader that forgot to tokenize would
// emit, and each must still stop the handoff loudly.
func TestUntokenizedPathFieldsStillAbortTheHandoff(t *testing.T) {
	absolute := "/Users/fixture-user/code/demo/calc.go"

	mutations := []struct {
		name string
		want string
		edit func(*capsule.Capsule)
	}{
		{
			name: "workspace.root",
			want: "workspace.root",
			edit: func(c *capsule.Capsule) { c.Workspace.Root = "/Users/fixture-user/code/demo" },
		},
		{
			name: "workspace.changed_files",
			want: "workspace.changed_files",
			edit: func(c *capsule.Capsule) { c.Workspace.ChangedFiles = []string{absolute} },
		},
		{
			name: "task.changed_files",
			want: "task.changed_files",
			edit: func(c *capsule.Capsule) { c.Task.ChangedFiles.Items = []string{absolute} },
		},
		{
			name: "task.files_touched_per_transcript",
			want: "task.files_touched_per_transcript",
			edit: func(c *capsule.Capsule) { c.Task.FilesTouchedPerTranscript.Items = []string{absolute} },
		},
		{
			name: "block.path",
			want: ".path",
			edit: func(c *capsule.Capsule) { c.Conversation.Events[0].Blocks[0].Path = absolute },
		},
		{
			name: "block.ref",
			want: ".ref",
			edit: func(c *capsule.Capsule) { c.Conversation.Events[0].Blocks[0].Ref = absolute },
		},
		{
			// The rc.1 defect #188 fix: a tool argument that names a file must
			// carry a token, whatever else the arguments contain.
			name: "tool_input.file_path",
			want: ".file_path",
			edit: func(c *capsule.Capsule) {
				replaceFirstToolInput(c, `{"file_path":"`+absolute+`","old_string":"/init stays prose"}`)
			},
		},
		{
			name: "tool_input.paths",
			want: ".paths[0]",
			edit: func(c *capsule.Capsule) {
				replaceFirstToolInput(c, `{"command":"go test ./...","paths":["`+absolute+`"]}`)
			},
		},
		{
			name: "conversation.full_history_ref",
			want: "conversation.full_history_ref",
			edit: func(c *capsule.Capsule) { c.Conversation.FullHistoryRef = "/tmp/sidecar/events.jsonl" },
		},
		{
			name: "projection.sidecar_ref",
			want: "projection.sidecar_ref",
			edit: func(c *capsule.Capsule) { c.Projection.SidecarRef = "/tmp/sidecar/events.jsonl" },
		},
	}

	for _, test := range absolutePathFixtures() {
		t.Run(test.agent, func(t *testing.T) {
			for _, mutation := range mutations {
				t.Run(mutation.name, func(t *testing.T) {
					c, _ := buildFixtureCapsule(t, test)
					mutation.edit(&c)
					_, err := capsule.CanonicalBytes(c)
					if err == nil {
						t.Fatalf("capsule accepted an absolute path in %s", mutation.name)
					}
					if !strings.Contains(err.Error(), mutation.want) {
						t.Fatalf("error %q does not name the offending field %q", err, mutation.want)
					}
					if err := capsule.Validate(c); err == nil {
						t.Fatalf("Validate accepted an absolute path in %s", mutation.name)
					}
				})
			}
		})
	}
}

// replaceFirstToolInput rewrites the first tool-call argument payload in the
// capsule, standing in for a reader that skipped path tokenization.
func replaceFirstToolInput(c *capsule.Capsule, text string) {
	for i := range c.Conversation.Events {
		for j := range c.Conversation.Events[i].Blocks {
			if c.Conversation.Events[i].Blocks[j].Type == capsule.BlockTypeToolInput {
				c.Conversation.Events[i].Blocks[j].Text = text
				return
			}
		}
	}
	panic("fixture has no tool_input block")
}
