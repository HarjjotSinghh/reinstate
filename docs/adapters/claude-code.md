# Claude Code adapter

- **Layout:** `~/.claude/projects/<project>/*.jsonl` (and config variants)
- **Transform:** schema-aware path fields only; prose preserved
- **Exclusions:** `auth.json`, credentials, `.env`, caches
- **Fixtures:** `testdata/adapters/claude/`

Compatibility states: see [compatibility.md](../compatibility.md).

## Real-device validation (human gate)

1. Create a short Claude session on Windows with an absolute project path.
2. `rein push --agent claude --session <id>`
3. On macOS: `rein pull --agent claude --session <id>`
4. Resume the session in Claude Code; confirm context and remapped paths.
