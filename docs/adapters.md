# Adapters

Reinstate uses per-agent **adapters** to discover, export, and restore
vendor-native session files without translating sessions across agents.
Later configuration adapters will separately normalize portable intent and
render each harness's native MCP/skills/plugins/settings format.

## Phase 1 (`v0.1.0`)

| Adapter | Sessions | Universal configuration | Notes |
| ------- | -------- | ----------------------- | ----- |
| Claude Code | In scope | Post–Phase 1 | Same-vendor resume only |
| OpenAI Codex CLI | In scope | Post–Phase 1 | Same-vendor resume only |

## Later

| Adapter | Status |
| ------- | ------ |
| Gemini CLI | Planned after Phase 1 |
| OpenCode | Planned after Phase 1 |
| Cursor | Exploring |
| Grok Build | Exploring |

Planned configuration targets include Claude Code, Codex, Gemini CLI, OpenCode,
and Grok. Each adapter will advertise support independently for MCP servers,
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings. An
adapter that can resume sessions does not automatically support configuration,
and unsupported or lossy mappings must be reported before apply.

See [universal-configuration.md](universal-configuration.md).

## Compatibility states

See [compatibility.md](compatibility.md). Adapters report `SUPPORTED`, `UNTESTED`,
`UNSUPPORTED`, or `NOT_INSTALLED`.

## Claude project identity

Claude Code stores a project beneath a directory key derived from that
device's absolute project path. Reinstate records the configured canonical
project ID in snapshots and recomputes Claude's directory key from the
destination device's `local_root`. Snapshot archive paths remain source
metadata; they are never reused as cross-device restore destinations.

RC5 verifies the exact planned destination after restore. A matching session
ID elsewhere in `~/.claude/projects` is not accepted as success.

## Exclusions

Adapters hard-exclude auth, credentials, tokens, caches, logs, and regenerable
dependencies. Future configuration profiles may carry secret **references** but
never secret values. Fixtures are synthetic and scanned for secrets.
