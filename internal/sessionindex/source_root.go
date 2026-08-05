package sessionindex

import (
	"path/filepath"
	"strings"
)

// AgentRoot returns the private native-agent state root that contains the
// selected source file. It is derived from the already indexed source path so
// custom CLAUDE_CONFIG_DIR and CODEX_HOME layouts use the same root during
// compatibility and capability checks. The path is never serialized.
func AgentRoot(record Record) string {
	marker := ""
	switch strings.ToLower(record.Agent) {
	case AgentClaude:
		marker = "projects"
	case AgentCodex:
		marker = "sessions"
	default:
		return ""
	}
	path := filepath.Clean(record.SourcePath)
	if path == "." || !filepath.IsAbs(path) {
		return ""
	}
	directory := filepath.Dir(path)
	for depth := 0; depth < 12; depth++ {
		if strings.EqualFold(filepath.Base(directory), marker) {
			root := filepath.Dir(directory)
			if root != directory && filepath.IsAbs(root) {
				return root
			}
			return ""
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return ""
}
