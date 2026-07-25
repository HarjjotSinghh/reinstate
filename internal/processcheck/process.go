// Package processcheck detects supported coding-agent processes before restore.
package processcheck

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// AgentActive reports whether a supported agent appears to be running.
func AgentActive(ctx context.Context, agent string) (bool, error) {
	agent = strings.ToLower(strings.TrimSpace(agent))
	switch agent {
	case "claude", "codex":
		return agentActive(ctx, agent)
	default:
		return false, fmt.Errorf("unsupported agent %q", agent)
	}
}

func matchesAgentProcess(agent, image, commandLine string) bool {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(image)))
	name = strings.TrimSuffix(name, ".exe")
	switch agent {
	case "claude":
		if name == "claude" || nativeVariant(name, "claude") {
			return true
		}
	case "codex":
		if name == "codex" || nativeVariant(name, "codex") {
			return true
		}
	}

	if name != "node" && name != "nodejs" {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(commandLine, "\\", "/"))
	switch agent {
	case "claude":
		return strings.Contains(normalized, "/@anthropic-ai/claude-code/") ||
			strings.Contains(normalized, "/claude-code/cli.js")
	case "codex":
		return strings.Contains(normalized, "/@openai/codex/")
	default:
		return false
	}
}

func nativeVariant(name, agent string) bool {
	for _, target := range []string{
		"aarch64-apple-darwin",
		"x86_64-apple-darwin",
		"aarch64-unknown-linux",
		"x86_64-unknown-linux",
		"aarch64-pc-windows",
		"x86_64-pc-windows",
	} {
		if strings.HasPrefix(name, agent+"-"+target) {
			return true
		}
	}
	return false
}
