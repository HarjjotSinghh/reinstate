package sessionindex

import (
	"path/filepath"
	"testing"
)

func TestAgentRootMatchesIndexedCustomLayouts(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	tests := []struct {
		agent string
		path  string
		root  string
	}{
		{AgentClaude, filepath.Join(base, "custom-claude", "projects", "encoded-project", "session.jsonl"), filepath.Join(base, "custom-claude")},
		{AgentCodex, filepath.Join(base, "custom-codex", "sessions", "2026", "08", "05", "rollout.jsonl"), filepath.Join(base, "custom-codex")},
	}
	for _, test := range tests {
		test := test
		t.Run(test.agent, func(t *testing.T) {
			t.Parallel()
			if got := AgentRoot(Record{Agent: test.agent, SourcePath: test.path}); got != test.root {
				t.Fatalf("AgentRoot() = %q, want %q", got, test.root)
			}
		})
	}
}

func TestAgentRootRejectsUnrecognizedOrRelativePaths(t *testing.T) {
	t.Parallel()
	for _, record := range []Record{
		{Agent: AgentGemini, SourcePath: filepath.Join(t.TempDir(), "projects", "one.jsonl")},
		{Agent: AgentClaude, SourcePath: "projects/one.jsonl"},
		{Agent: AgentCodex, SourcePath: filepath.Join(t.TempDir(), "other", "one.jsonl")},
	} {
		if got := AgentRoot(record); got != "" {
			t.Fatalf("AgentRoot(%+v) = %q", record, got)
		}
	}
}
