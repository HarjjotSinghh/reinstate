package sessionindex

import (
	"path/filepath"
	"testing"
	"time"
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

func TestAgentRootSurvivesCoalescedCustomAgentHome(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "custom-codex")
	old := Record{Agent: AgentCodex, ID: "same", Key: "codex:same", SourcePath: filepath.Join(root, "sessions", "2026", "old.jsonl"), SourceModTime: 1}
	newer := old
	newer.SourcePath = filepath.Join(root, "sessions", "2026", "new.jsonl")
	newer.SourceModTime = 2
	newer.UpdatedAt = time.Now().UTC()
	records, _ := CoalesceRecords([]Record{old, newer})
	if got := AgentRoot(records[0]); got != root {
		t.Fatalf("AgentRoot(coalesced) = %q, want %q", got, root)
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
