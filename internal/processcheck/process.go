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
		return false, nil
	}
	for _, p := range procs {
		if matchesAgentProcess(agent, p.Image, p.CommandLine) {
			return true, nil
		}
	}
	return false, nil
}

// Target describes the session a restore is about to replace.
type Target struct {
	// SessionID is the vendor session identifier being restored.
	SessionID string
	// Path is the on-disk session file that would be replaced.
	Path string
	// ProjectRoot is the local root of the mapped project the session belongs
	// to. It may be empty when no mapping is configured.
	ProjectRoot string
}

// SessionBusy reports whether a running instance of agent is using target.
//
// An open file handle is not sufficient evidence on its own. Claude Code
// appends to its session file and closes it again, so a live Claude Code
// session holds no handle at all and a handle-only check reports it as free.
// Treating "no handle" as "not in use" is how an in-place restore can land on
// a session someone is actively working in.
//
// Detection therefore deliberately biases toward busy. Under the default fork
// policy a false positive costs one extra session file, while a false negative
// costs a live session. Three independent signals are consulted, and any one of
// them is enough.
func SessionBusy(ctx context.Context, agent string, target Target) (busy bool, scoped bool, err error) {
	agent, err = normalizeAgent(agent)
	if err != nil {
		return false, false, err
	}
	if strings.TrimSpace(target.Path) == "" && strings.TrimSpace(target.SessionID) == "" {
		// Without a target there is nothing to scope to.
		active, activeErr := AgentActive(ctx, agent)
		return active, false, activeErr
	}

	procs, err := listProcesses(ctx)
	if err != nil {
		return false, false, nil
	}
	holders, _, err := sessionFileHolders(ctx, target.Path)
	if err != nil {
		holders = nil
	}
	cwds := agentWorkingDirectories(ctx, agent, procs)
	return decideSessionBusy(agent, target, procs, holders, cwds), true, nil
}

// decideSessionBusy resolves the liveness question from host facts.
//
// Signals, in order of strength:
//  1. a matching agent process holds the session file open;
//  2. a matching agent process names the exact session on its command line,
//     which covers `claude --resume <id>` and `codex resume <id>`; and
//  3. a matching agent process is working inside the session's project, which
//     covers a session chosen interactively where the id never reaches argv.
func decideSessionBusy(
	agent string, target Target, procs []Process, holders []int, cwds map[int]string,
) bool {
	holding := make(map[int]bool, len(holders))
	for _, pid := range holders {
		holding[pid] = true
	}
	sessionID := strings.TrimSpace(target.SessionID)
	projectRoot := cleanRoot(target.ProjectRoot)

	for _, p := range procs {
		if !matchesAgentProcess(agent, p.Image, p.CommandLine) {
			continue
		}
		if holding[p.PID] {
			return true
		}
		if sessionID != "" && strings.Contains(p.CommandLine, sessionID) {
			return true
		}
		if projectRoot != "" {
			if within(cleanRoot(cwds[p.PID]), projectRoot) {
				return true
			}
		}
	}
	return false
}

func cleanRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Clean(path), string(filepath.Separator))
}

// within reports whether candidate is root or lives beneath it.
func within(candidate, root string) bool {
	if candidate == "" || root == "" {
		return false
	}
	if candidate == root {
		return true
	}
	return strings.HasPrefix(candidate, root+string(filepath.Separator))
}

// Spec is the catalog ProcessSpec used to recognize a running agent.
type Spec struct {
	Images      []string
	NodeMarkers []string
	Identify    []Identity
}

// Identity is one vendor self-identification environment pair.
type Identity struct {
	Name  string
	Value string
}

// specs is installed by CLI init from the catalog. Nil means tests that do
// not import the CLI should use testFallbackSpecs.
var specs map[string]Spec

// SetSpecs replaces the process-matcher table.
func SetSpecs(next map[string]Spec) {
	converted := make(map[string]Spec, len(next))
	for name, spec := range next {
		converted[strings.ToLower(strings.TrimSpace(name))] = spec
	}
	specs = converted
}

func currentSpecs() map[string]Spec {
	if specs != nil {
		return specs
	}
	return testFallbackSpecs()
}

func normalizeAgent(agent string) (string, error) {
	agent = strings.ToLower(strings.TrimSpace(agent))
	if _, ok := currentSpecs()[agent]; ok {
		return agent, nil
	}
	return "", fmt.Errorf("unsupported agent %q", agent)
}

func matchesAgentProcess(agent, image, commandLine string) bool {
	spec, ok := currentSpecs()[strings.ToLower(strings.TrimSpace(agent))]
	if !ok {
		return false
	}
	name := strings.ToLower(filepath.Base(strings.TrimSpace(image)))
	name = strings.TrimSuffix(name, ".exe")
	normalized := strings.ToLower(strings.ReplaceAll(commandLine, "\\", "/"))

	for _, identity := range spec.Identify {
		if identity.Name == "" {
			continue
		}
		token := strings.ToLower(identity.Name) + "="
		if identity.Value != "" {
			token += strings.ToLower(identity.Value)
		}
		if strings.Contains(normalized, token) {
			return true
		}
	}

	for _, imageName := range spec.Images {
		imageName = strings.ToLower(strings.TrimSpace(imageName))
		if imageName == "" {
			continue
		}
		if name == imageName || nativeVariant(name, imageName) {
			return true
		}
	}

	if name != "node" && name != "nodejs" {
		return false
	}
	for _, marker := range spec.NodeMarkers {
		marker = strings.ToLower(strings.ReplaceAll(marker, "\\", "/"))
		if marker != "" && strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func testFallbackSpecs() map[string]Spec {
	return map[string]Spec{
		"claude": {
			Images:      []string{"claude"},
			NodeMarkers: []string{"/@anthropic-ai/claude-code/", "/claude-code/cli.js"},
		},
		"codex": {
			Images:      []string{"codex"},
			NodeMarkers: []string{"/@openai/codex/"},
		},
		"gemini":   {Images: []string{"gemini"}},
		"grok":     {Images: []string{"grok"}},
		"opencode": {Images: []string{"opencode"}},
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
