# Codex adapter

- **Layout:** `~/.codex/sessions/*.jsonl` rollouts
- **Transform:** `cwd` / path fields via pathmap tokens
- **Exclusions:** auth and credential files
- **Fixtures:** `testdata/adapters/codex/`

## Universal configuration (roadmap)

A separate configuration adapter may eventually import and render Codex's
supported MCP, skills/instructions, hooks/loops, plugin, marketplace, and safe
setting declarations. Session support does not imply config support. Writes
must preserve unrelated Codex-managed state and continue to exclude
`auth.json` and other credentials. See
[universal-configuration.md](../universal-configuration.md).

## Real-device validation (human gate)

Same dual-OS push/pull/resume procedure as Claude, using Codex CLI.
