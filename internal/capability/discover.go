package capability

import (
	"context"
	"path/filepath"
	"strings"
)

// Discover returns a deterministic, privacy-safe inventory for Claude Code
// and Codex. It performs local filesystem reads only and never executes a
// command or follows a symlink.
func Discover(opts Options) Inventory {
	return DiscoverContext(context.Background(), opts)
}

// DiscoverContext is Discover with cooperative cancellation. Cancellation is
// transactional: it returns no partial items or diagnostics, only a fixed,
// path-free cancellation diagnostic.
func DiscoverContext(ctx context.Context, opts Options) Inventory {
	c := newCollectorContext(ctx)
	if c.cancelled() {
		return cancelledInventory()
	}
	opts = normalizeOptions(opts, c)
	scanClaude(c, opts)
	scanCodex(c, opts)
	return c.inventory()
}

// DiscoverAgentContext discovers declarations for exactly one supported agent.
// It does not inspect the other agent's roots, so unrelated configuration cannot
// consume the selected agent's preflight deadline or contribute diagnostics.
func DiscoverAgentContext(ctx context.Context, agent Agent, opts Options) Inventory {
	c := newCollectorContext(ctx)
	if c.cancelled() {
		return cancelledInventory()
	}
	if agent != AgentClaude && agent != AgentCodex {
		c.addDiagnostic(Diagnostic{Code: DiagnosticInvalidRoot})
		return c.inventory()
	}
	opts = normalizeAgentOptions(opts, c, agent)
	switch agent {
	case AgentClaude:
		scanClaude(c, opts)
	case AgentCodex:
		scanCodex(c, opts)
	}
	return c.inventory()
}

func normalizeOptions(opts Options, c *collector) Options {
	return normalizeAgentOptions(opts, c, "")
}

func normalizeAgentOptions(opts Options, c *collector, agent Agent) Options {
	if (agent == "" || agent == AgentClaude) && opts.ClaudeHome == "" && opts.UserHome != "" {
		opts.ClaudeHome = filepath.Join(opts.UserHome, ".claude")
	}
	if (agent == "" || agent == AgentCodex) && opts.CodexHome == "" && opts.UserHome != "" {
		opts.CodexHome = filepath.Join(opts.UserHome, ".codex")
	}
	if opts.WorkingDir == "" {
		opts.WorkingDir = opts.ProjectRoot
	}
	roots := []string{opts.UserHome, opts.ProjectRoot, opts.WorkingDir, opts.ManagedRoot}
	if agent == "" || agent == AgentClaude {
		roots = append(roots, opts.ClaudeHome)
	}
	if agent == "" || agent == AgentCodex {
		roots = append(roots, opts.CodexHome)
	}
	for _, root := range roots {
		if c.cancelled() {
			return opts
		}
		if root != "" && !filepath.IsAbs(root) {
			c.addDiagnostic(Diagnostic{Code: DiagnosticInvalidRoot})
		}
	}
	if opts.ProjectRoot != "" && opts.WorkingDir != "" && !withinRoot(opts.ProjectRoot, opts.WorkingDir) {
		c.addDiagnostic(Diagnostic{Scope: ScopeProject, Code: DiagnosticInvalidRoot})
		opts.ProjectRoot = ""
		opts.WorkingDir = ""
	}
	return opts
}

func scanClaude(c *collector, opts Options) {
	if filepath.IsAbs(opts.ClaudeHome) {
		claudeAnchor := opts.ClaudeHome
		if filepath.IsAbs(opts.UserHome) && withinRoot(opts.UserHome, opts.ClaudeHome) {
			// A default $HOME/.claude is not itself a trusted root. Anchor it at
			// the verified home so a symlink cannot redirect discovery.
			claudeAnchor = opts.UserHome
		}
		addInstructionFile(c, claudeAnchor, filepath.Join(opts.ClaudeHome, "CLAUDE.md"), AgentClaude, ScopeUser, SourceClaudeMemory, "CLAUDE.md", false)
		scanSkillRoot(c, claudeAnchor, filepath.Join(opts.ClaudeHome, "skills"), AgentClaude, ScopeUser, SourceClaudeSkill, false)
		scanLegacyCommands(c, claudeAnchor, filepath.Join(opts.ClaudeHome, "commands"), ScopeUser)

		// Claude's legacy default state is ~/.claude.json. With an explicit
		// CLAUDE_CONFIG_DIR, global state is isolated beneath that directory.
		statePath := filepath.Join(opts.ClaudeHome, ".claude.json")
		stateAnchor := claudeAnchor
		if filepath.IsAbs(opts.UserHome) && filepath.Clean(opts.ClaudeHome) == filepath.Join(filepath.Clean(opts.UserHome), ".claude") {
			statePath = filepath.Join(opts.UserHome, ".claude.json")
			stateAnchor = opts.UserHome
		}
		scanClaudeStateMCP(c, stateAnchor, statePath, opts)
	}

	for _, dir := range projectDirectories(opts, c, AgentClaude) {
		if c.cancelled() {
			return
		}
		addInstructionFile(c, opts.ProjectRoot, filepath.Join(dir, "CLAUDE.md"), AgentClaude, ScopeProject, SourceClaudeMemory, "CLAUDE.md", false)
		addInstructionFile(c, opts.ProjectRoot, filepath.Join(dir, ".claude", "CLAUDE.md"), AgentClaude, ScopeProject, SourceClaudeMemory, "CLAUDE.md", false)
		addInstructionFile(c, opts.ProjectRoot, filepath.Join(dir, "CLAUDE.local.md"), AgentClaude, ScopeLocal, SourceClaudeMemory, "CLAUDE.local.md", false)
		scanNamedFiles(c, opts.ProjectRoot, filepath.Join(dir, ".claude", "rules"), AgentClaude, KindInstruction, ScopeProject, SourceClaudeRule, ".md", true)
		scanSkillRoot(c, opts.ProjectRoot, filepath.Join(dir, ".claude", "skills"), AgentClaude, ScopeProject, SourceClaudeSkill, false)
		scanLegacyCommands(c, opts.ProjectRoot, filepath.Join(dir, ".claude", "commands"), ScopeProject)
	}
	if filepath.IsAbs(opts.ProjectRoot) {
		scanClaudeMCPFile(c, opts.ProjectRoot, filepath.Join(opts.ProjectRoot, ".mcp.json"), ScopeProject, SourceClaudeMCPJSON)
	}

	if managedDir := claudeManagedDir(opts); managedDir != "" {
		addInstructionFile(c, opts.ManagedRoot, filepath.Join(managedDir, "CLAUDE.md"), AgentClaude, ScopeManaged, SourceClaudeMemory, "CLAUDE.md", false)
		scanClaudeMCPFile(c, opts.ManagedRoot, filepath.Join(managedDir, "managed-mcp.json"), ScopeManaged, SourceClaudeManagedMCP)
	}
}

func scanCodex(c *collector, opts Options) {
	if c.cancelled() {
		return
	}
	if filepath.IsAbs(opts.CodexHome) {
		codexAnchor := opts.CodexHome
		if filepath.IsAbs(opts.UserHome) && withinRoot(opts.UserHome, opts.CodexHome) {
			// A default $HOME/.codex is not itself a trusted root. Anchor it at
			// the verified home so a symlink cannot redirect discovery.
			codexAnchor = opts.UserHome
		}
		addInstructionFile(c, codexAnchor, filepath.Join(opts.CodexHome, "AGENTS.override.md"), AgentCodex, ScopeUser, SourceCodexInstruction, "AGENTS.override.md", false)
		addInstructionFile(c, codexAnchor, filepath.Join(opts.CodexHome, "AGENTS.md"), AgentCodex, ScopeUser, SourceCodexInstruction, "AGENTS.md", false)
		scanCodexMCPFile(c, codexAnchor, filepath.Join(opts.CodexHome, "config.toml"), ScopeUser)
	}
	if filepath.IsAbs(opts.UserHome) {
		scanSkillRoot(c, opts.UserHome, filepath.Join(opts.UserHome, ".agents", "skills"), AgentCodex, ScopeUser, SourceCodexSkill, false)
	}

	for _, dir := range projectDirectories(opts, c, AgentCodex) {
		if c.cancelled() {
			return
		}
		addInstructionFile(c, opts.ProjectRoot, filepath.Join(dir, "AGENTS.override.md"), AgentCodex, ScopeProject, SourceCodexInstruction, "AGENTS.override.md", false)
		addInstructionFile(c, opts.ProjectRoot, filepath.Join(dir, "AGENTS.md"), AgentCodex, ScopeProject, SourceCodexInstruction, "AGENTS.md", false)
		scanSkillRoot(c, opts.ProjectRoot, filepath.Join(dir, ".agents", "skills"), AgentCodex, ScopeProject, SourceCodexSkill, false)
		scanCodexMCPFile(c, opts.ProjectRoot, filepath.Join(dir, ".codex", "config.toml"), ScopeProject)
	}

	// OpenAI documents /etc/codex as the administrator layer on Unix. This
	// package is intentionally limited to the requested macOS/Windows matrix;
	// it does not invent an undocumented native-Windows equivalent.
	if opts.GOOS == "darwin" && filepath.IsAbs(opts.ManagedRoot) {
		managed := filepath.Join(opts.ManagedRoot, "etc", "codex")
		scanSkillRoot(c, opts.ManagedRoot, filepath.Join(managed, "skills"), AgentCodex, ScopeManaged, SourceCodexSkill, false)
		scanCodexMCPFile(c, opts.ManagedRoot, filepath.Join(managed, "config.toml"), ScopeManaged)
	} else if opts.GOOS != "windows" && opts.GOOS != "darwin" && opts.GOOS != "" {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Code: DiagnosticUnsupportedOS})
	}
}

func claudeManagedDir(opts Options) string {
	if !filepath.IsAbs(opts.ManagedRoot) {
		return ""
	}
	switch opts.GOOS {
	case "darwin":
		return filepath.Join(opts.ManagedRoot, "Library", "Application Support", "ClaudeCode")
	case "windows":
		return filepath.Join(opts.ManagedRoot, "Program Files", "ClaudeCode")
	default:
		return ""
	}
}

func projectDirectories(opts Options, c *collector, agent Agent) []string {
	if !filepath.IsAbs(opts.ProjectRoot) || !filepath.IsAbs(opts.WorkingDir) || !withinRoot(opts.ProjectRoot, opts.WorkingDir) {
		return nil
	}
	rel, err := filepath.Rel(opts.ProjectRoot, opts.WorkingDir)
	if err != nil {
		return nil
	}
	dirs := []string{filepath.Clean(opts.ProjectRoot)}
	if rel == "." {
		return dirs
	}
	current := filepath.Clean(opts.ProjectRoot)
	for depth, part := range strings.Split(rel, string(filepath.Separator)) {
		if c.cancelled() {
			return nil
		}
		if depth+1 >= maxDepth {
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindInstruction, Scope: ScopeProject, Code: DiagnosticLimitReached})
			break
		}
		current = filepath.Join(current, part)
		dirs = append(dirs, current)
	}
	return dirs
}

func addInstructionFile(c *collector, root, path string, agent Agent, scope Scope, source SourceKind, name string, lazy bool) {
	if c.cancelled() || !filepath.IsAbs(root) {
		return
	}
	status, _ := inspectPath(c.ctx, root, path, -1)
	if c.cancelled() {
		return
	}
	switch status {
	case pathRegular:
		c.add(Item{Agent: agent, Kind: KindInstruction, Name: name, Scope: scope, State: StateCandidate, SourceKind: source, Lazy: lazy})
	case pathSymlink:
		c.add(Item{Agent: agent, Kind: KindInstruction, Name: name, Scope: scope, State: StateUnverified, SourceKind: source, Lazy: lazy})
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindInstruction, Scope: scope, Code: DiagnosticSymlink})
	case pathUnsafe, pathFailed:
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindInstruction, Scope: scope, Code: diagnosticForStatus(status)})
	}
}

func scanSkillRoot(c *collector, anchor, root string, agent Agent, scope Scope, source SourceKind, lazy bool) {
	if c.cancelled() || !filepath.IsAbs(anchor) {
		return
	}
	status, _ := inspectPath(c.ctx, anchor, root, -1)
	if c.cancelled() {
		return
	}
	if status == pathMissing {
		return
	}
	if status != pathDirectory {
		if status == pathSymlink {
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: DiagnosticSymlink})
		} else {
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: diagnosticForStatus(status)})
		}
		return
	}
	scanSkillDirs(c, anchor, root, agent, scope, source, lazy, 0)
}

func scanSkillDirs(c *collector, anchor, dir string, agent Agent, scope Scope, source SourceKind, lazy bool, depth int) {
	if c.cancelled() || c.full(agent, KindSkill) {
		return
	}
	if depth >= maxDepth {
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: DiagnosticLimitReached})
		return
	}
	entries, truncated, status := readDirBounded(c.ctx, anchor, dir)
	if c.cancelled() {
		return
	}
	if status != pathDirectory {
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: diagnosticForStatus(status)})
		return
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: DiagnosticLimitReached})
	}
	for _, entry := range entries {
		if c.cancelled() || c.full(agent, KindSkill) {
			return
		}
		path := filepath.Join(dir, entry.Name())
		status, _ := inspectPath(c.ctx, anchor, path, -1)
		if c.cancelled() {
			return
		}
		if status == pathSymlink {
			c.add(Item{Agent: agent, Kind: KindSkill, Name: entry.Name(), Scope: scope, State: StateUnverified, SourceKind: source, Lazy: lazy})
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: DiagnosticSymlink})
			continue
		}
		if status != pathDirectory {
			continue
		}
		skillStatus, _ := inspectPath(c.ctx, anchor, filepath.Join(path, "SKILL.md"), -1)
		if c.cancelled() {
			return
		}
		switch skillStatus {
		case pathRegular:
			c.add(Item{Agent: agent, Kind: KindSkill, Name: entry.Name(), Scope: scope, State: StateCandidate, SourceKind: source, Lazy: lazy})
			continue
		case pathSymlink:
			c.add(Item{Agent: agent, Kind: KindSkill, Name: entry.Name(), Scope: scope, State: StateUnverified, SourceKind: source, Lazy: lazy})
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: KindSkill, Scope: scope, Code: DiagnosticSymlink})
			continue
		}
		scanSkillDirs(c, anchor, path, agent, scope, source, lazy, depth+1)
	}
}

func scanLegacyCommands(c *collector, anchor, root string, scope Scope) {
	scanNamedFiles(c, anchor, root, AgentClaude, KindSkill, scope, SourceClaudeLegacyCmd, ".md", false)
}

func scanNamedFiles(c *collector, anchor, root string, agent Agent, kind Kind, scope Scope, source SourceKind, extension string, lazy bool) {
	if c.cancelled() || !filepath.IsAbs(anchor) {
		return
	}
	status, _ := inspectPath(c.ctx, anchor, root, -1)
	if c.cancelled() {
		return
	}
	if status == pathMissing {
		return
	}
	if status != pathDirectory {
		if status == pathSymlink {
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: kind, Scope: scope, Code: DiagnosticSymlink})
		} else {
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: kind, Scope: scope, Code: diagnosticForStatus(status)})
		}
		return
	}
	scanNamedFilesAt(c, anchor, root, agent, kind, scope, source, extension, lazy, 0)
}

func scanNamedFilesAt(c *collector, anchor, dir string, agent Agent, kind Kind, scope Scope, source SourceKind, extension string, lazy bool, depth int) {
	if c.cancelled() || c.full(agent, kind) {
		return
	}
	if depth >= maxDepth {
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: kind, Scope: scope, Code: DiagnosticLimitReached})
		return
	}
	entries, truncated, status := readDirBounded(c.ctx, anchor, dir)
	if c.cancelled() {
		return
	}
	if status != pathDirectory {
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: kind, Scope: scope, Code: diagnosticForStatus(status)})
		return
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: agent, Kind: kind, Scope: scope, Code: DiagnosticLimitReached})
	}
	for _, entry := range entries {
		if c.cancelled() || c.full(agent, kind) {
			return
		}
		path := filepath.Join(dir, entry.Name())
		status, _ := inspectPath(c.ctx, anchor, path, -1)
		if c.cancelled() {
			return
		}
		if status == pathSymlink {
			if strings.EqualFold(filepath.Ext(entry.Name()), extension) {
				c.add(Item{Agent: agent, Kind: kind, Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Scope: scope, State: StateUnverified, SourceKind: source, Lazy: lazy})
			}
			c.addDiagnostic(Diagnostic{Agent: agent, Kind: kind, Scope: scope, Code: DiagnosticSymlink})
			continue
		}
		if status == pathDirectory {
			scanNamedFilesAt(c, anchor, path, agent, kind, scope, source, extension, lazy, depth+1)
			continue
		}
		if status == pathRegular && strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			c.add(Item{Agent: agent, Kind: kind, Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Scope: scope, State: StateCandidate, SourceKind: source, Lazy: lazy})
		}
	}
}
