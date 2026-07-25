# Adapters

Reinstate uses per-agent **adapters** to discover, export, and restore vendor-native
session files without translating across agents.

## Phase 1 (`v0.1.0`)

| Adapter | Sessions | Config (MCP/skills) | Notes |
| ------- | -------- | ------------------- | ----- |
| Claude Code | In scope | Post–Phase 1 | Same-vendor resume only |
| OpenAI Codex CLI | In scope | Post–Phase 1 | Same-vendor resume only |

## Later

| Adapter | Status |
| ------- | ------ |
| Gemini CLI | Planned after Phase 1 |
| OpenCode | Planned after Phase 1 |
| Cursor | Exploring |
| Grok Build | Exploring |

## Compatibility states

See [compatibility.md](compatibility.md). Adapters report `SUPPORTED`, `UNTESTED`,
`UNSUPPORTED`, or `NOT_INSTALLED`.

## Exclusions

Adapters hard-exclude auth, credentials, tokens, caches, logs, and regenerable
dependencies. Fixtures are synthetic and scanned for secrets.
