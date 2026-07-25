package fixture

import (
	"os"
	"path/filepath"
)

// SyntheticFiles is the deterministic adapter fixture corpus.
var SyntheticFiles = map[string]string{
	"claude/macos/projects/fixture-project/session-syn-001.jsonl": `{"type":"meta","sessionId":"session-syn-001","cwd":"/Users/fixture-user/code/demo"}
{"type":"user","message":{"content":"Synthetic Claude fixture request"}}
`,
	"claude/windows/projects/fixture-project/session-syn-001.jsonl": `{"type":"meta","sessionId":"session-syn-001","cwd":"C:\\Users\\fixture-user\\code\\demo"}
{"type":"user","message":{"content":"Synthetic Claude Windows fixture request"}}
`,
	"claude/wsl/projects/fixture-project/session-syn-001.jsonl": `{"type":"meta","sessionId":"session-syn-001","cwd":"/home/fixture-user/code/demo"}
{"type":"user","message":{"content":"Synthetic Claude WSL2 fixture request"}}
`,
	"codex/macos/sessions/rollout-syn-001.jsonl": `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"/Users/fixture-user/code/demo"}}
{"type":"message","role":"user","content":"Synthetic Codex fixture request"}
`,
	"codex/windows/sessions/rollout-syn-001.jsonl": `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"C:\\Users\\fixture-user\\code\\demo"}}
{"type":"message","role":"user","content":"Synthetic Codex Windows fixture request"}
`,
	"codex/wsl/sessions/rollout-syn-001.jsonl": `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"/home/fixture-user/code/demo"}}
{"type":"message","role":"user","content":"Synthetic Codex WSL2 fixture request"}
`,
}

// Generate writes the deterministic synthetic corpus below root.
func Generate(root string) error {
	for relative, content := range SyntheticFiles {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return err
		}
	}
	return nil
}
