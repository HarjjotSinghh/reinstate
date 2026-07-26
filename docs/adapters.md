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

## Claude project identity

Claude Code stores a project beneath a directory key derived from that
device's absolute project path. Reinstate records the configured canonical
project ID in snapshots and recomputes Claude's directory key from the
destination device's `local_root`. Snapshot archive paths remain source
metadata; they are never reused as cross-device restore destinations.

RC4 verifies the exact planned destination after restore. A matching session
ID elsewhere in `~/.claude/projects` is not accepted as success.

## Exclusions

Adapters hard-exclude auth, credentials, tokens, caches, logs, and regenerable
dependencies. Fixtures are synthetic and scanned for secrets.
