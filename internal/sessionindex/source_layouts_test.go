package sessionindex

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestClaudeAndCodexPortableWorkspaceShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		agent     string
		workspace string
		project   string
	}{
		{name: "Claude macOS", agent: AgentClaude, workspace: "/Users/alice/work/demo", project: "demo"},
		{name: "Claude Windows", agent: AgentClaude, workspace: `C:\Users\alice\work\demo`, project: "demo"},
		{name: "Claude WSL", agent: AgentClaude, workspace: "/mnt/c/Users/alice/work/demo", project: "demo"},
		{name: "Codex macOS", agent: AgentCodex, workspace: "/Users/alice/work/demo", project: "demo"},
		{name: "Codex Windows", agent: AgentCodex, workspace: `C:\Users\alice\work\demo`, project: "demo"},
		{name: "Codex WSL", agent: AgentCodex, workspace: "/mnt/c/Users/alice/work/demo", project: "demo"},
	}

	for index, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			id := fmt.Sprintf("portable-%d", index)
			var source Source
			switch test.agent {
			case AgentClaude:
				path := filepath.Join(root, "projects", "portable-project", id+".jsonl")
				writeSyntheticJSONL(t, path, []string{
					fmt.Sprintf(
						`{"type":"user","sessionId":%q,"cwd":%q,"message":{"role":"user","content":"PORTABLE_PROMPT"}}`,
						id,
						test.workspace,
					),
				}, "")
				source = NewClaudeSource(root)
			case AgentCodex:
				path := filepath.Join(root, "sessions", "2026", "07", id+".jsonl")
				writeSyntheticJSONL(t, path, []string{
					fmt.Sprintf(
						`{"type":"session_meta","payload":{"id":%q,"cwd":%q}}`,
						id,
						test.workspace,
					),
					`{"type":"event_msg","payload":{"type":"user_message","message":"PORTABLE_PROMPT"}}`,
				}, "")
				source = NewCodexSource(root)
			default:
				t.Fatalf("unknown test agent %q", test.agent)
			}

			result, err := source.Scan(context.Background())
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			if record.Workspace != test.workspace || record.Project != test.project {
				t.Fatalf(
					"workspace/project = %q / %q, want %q / %q",
					record.Workspace,
					record.Project,
					test.workspace,
					test.project,
				)
			}
			if record.Title != id || record.PromptPreview != "PORTABLE_PROMPT" {
				t.Fatalf("title/preview = %q / %q", record.Title, record.PromptPreview)
			}
		})
	}
}
