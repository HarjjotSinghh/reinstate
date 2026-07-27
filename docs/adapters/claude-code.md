# Claude Code adapter

- **Layout:** `~/.claude/projects/<project>/*.jsonl` (and config variants)
- **Transform:** schema-aware path fields only; prose preserved
- **Session scope:** top-level resumable sessions only; nested `subagents/`
  artifacts are not listed or synced as independent sessions
- **Exclusions:** `auth.json`, credentials, `.env`, caches
- **Fixtures:** `testdata/adapters/claude/`

## Universal configuration (roadmap)

A separate configuration adapter may eventually import and render Claude
Code's supported MCP, skills/instructions, hooks/loops, plugin, marketplace, and
safe setting declarations. Session support does not imply config support.
Writes must preserve unrelated Claude-managed state and continue to exclude
credentials and plugin caches. See
[universal-configuration.md](../universal-configuration.md).

Compatibility states: see [compatibility.md](../compatibility.md).

## Real-device validation (human gate)

1. Create a short Claude session on Windows with an absolute project path.
2. `rein push --agent claude --session <id>`
3. On macOS: `rein pull --agent claude --session <id>`
4. Resume the session in Claude Code; confirm context and remapped paths.
