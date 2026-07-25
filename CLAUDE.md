@AGENTS.md

Claude Code-specific note: never inspect the developer's real `~/.claude`
tree while contributing. Use only `testdata/adapters/claude/` or temporary
synthetic fixtures. End-user setup behavior is defined by
`docs/prompts/claude-code-setup.md`.
