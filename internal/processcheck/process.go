// Package processcheck detects supported coding-agent processes before restore.
package processcheck

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Process describes one running process well enough to classify it.
type Process struct {
	PID         int
	Image       string
	CommandLine string
}

// AgentActive reports whether a supported agent appears to be running anywhere
// on the host.
//
// This is deliberately coarse: it answers "is any Claude Code running?", not
// "is this session in use?". Prefer SessionBusy, which scopes the question to
// the exact file a restore is about to replace. A developer with unrelated
// agents running in other projects is a normal state, not a reason to refuse.
func AgentActive(ctx context.Context, agent string) (bool, error) {
	agent, err := normalizeAgent(agent)
	if err != nil {
		return false, err
	}
	procs, err := listProcesses(ctx)
	if err != nil {
		return false, err
	}
	for _, p := range procs {
		if matchesAgentProcess(agent, p.Image, p.CommandLine) {
			return true, nil
		}
	}
	return false, nil
}

// SessionBusy reports whether a running instance of agent is holding the
// session file at path open.
//
// It answers the question a restore actually needs to ask. When the host cannot
// enumerate open file handles, it degrades to the coarse host-wide answer and
// reports scoped=false so callers can describe the result honestly.
func SessionBusy(ctx context.Context, agent, path string) (busy bool, scoped bool, err error) {
	agent, err = normalizeAgent(agent)
	if err != nil {
		return false, false, err
	}
	if strings.TrimSpace(path) == "" {
		// Without a target we cannot scope; fall back to the coarse check.
		active, activeErr := AgentActive(ctx, agent)
		return active, false, activeErr
	}

	pids, supported, err := sessionFileHolders(ctx, path)
	if err != nil {
		return false, false, err
	}
	if supported && len(pids) == 0 {
		// The handle table is authoritative and nobody holds the file.
		return false, true, nil
	}
	procs, err := listProcesses(ctx)
	if err != nil {
		return false, false, err
	}
	busy, scoped = decideSessionBusy(agent, procs, pids, supported)
	return busy, scoped, nil
}

// decideSessionBusy resolves the liveness question from host facts.
//
// When handle enumeration is supported, only a process that both holds the file
// and looks like the agent counts. Otherwise the answer degrades to the coarse
// host-wide match and is reported as unscoped.
func decideSessionBusy(agent string, procs []Process, holders []int, supported bool) (busy bool, scoped bool) {
	if !supported {
		for _, p := range procs {
			if matchesAgentProcess(agent, p.Image, p.CommandLine) {
				return true, false
			}
		}
		return false, false
	}
	if len(holders) == 0 {
		return false, true
	}
	holding := make(map[int]bool, len(holders))
	for _, pid := range holders {
		holding[pid] = true
	}
	for _, p := range procs {
		if holding[p.PID] && matchesAgentProcess(agent, p.Image, p.CommandLine) {
			return true, true
		}
	}
	return false, true
}

func normalizeAgent(agent string) (string, error) {
	agent = strings.ToLower(strings.TrimSpace(agent))
	switch agent {
	case "claude", "codex":
		return agent, nil
	default:
		return "", fmt.Errorf("unsupported agent %q", agent)
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
