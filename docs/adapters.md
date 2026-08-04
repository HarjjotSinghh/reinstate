# Adapters

Reinstate uses per-agent **adapters** to discover, export, and restore
vendor-native session files without translating sessions across agents.
Later configuration adapters will separately normalize portable intent and
render each harness's native MCP/skills/plugins/settings format.

Phase 4 adds separate, directional **transcript-source** and
**handoff-target** capabilities. A session adapter that can sync bytes does not
automatically qualify to parse portable history or create a safe destination
session.

## Phase 1 (`v0.1.0`)

| Adapter | Native sessions | Cross-agent handoff | Universal configuration | Notes |
| ------- | --------------- | ------------------- | ----------------------- | ----- |
| Claude Code | In scope | Phase 4 source + target | Post–Phase 1 | Phase 1 is same-vendor only |
| OpenAI Codex CLI | In scope | Phase 4 source + target | Post–Phase 1 | Phase 1 is same-vendor only |

## Later

| Adapter | Session discovery | Cross-agent handoff |
| ------- | ----------------- | ------------------- |
| Gemini CLI | Planned after Phase 1 | Phase 4 after Claude ↔ Codex |
| OpenCode | Planned after Phase 1 | Phase 4 after Claude ↔ Codex |
| Cursor | Exploring | Exploring |
| Grok Build | Exploring | Exploring |
| Copilot CLI / Orca / others | Exploring | Requires public format/API and acceptance evidence |

Planned configuration targets include Claude Code, Codex, Gemini CLI, OpenCode,
and Grok. Each adapter will advertise support independently for MCP servers,
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings. An
adapter that can resume sessions does not automatically support configuration,
and unsupported or lossy mappings must be reported before apply.

See [universal-configuration.md](universal-configuration.md).

## Cross-agent capability states (Phase 4 target)

Support is reported per direction and per version for:

| Capability | Meaning |
| ---------- | ------- |
| Source parse | Safely reads visible messages, tool relationships, attachments, and unknown records |
| Structured handoff | Builds a task/workspace capsule without needing the source model |
| Destination launch | Creates and verifies a new destination-native session |
| Reconstructed history | Projects normalized visible history into target-native storage; experimental unless explicitly promoted |
| Capability verification | Reports missing tools, MCP, skills, instructions, sandbox, and runtime state |
| Fidelity report | Classifies every component as exact/normalized/summarized/referenced/omitted |

Claude → Codex and Codex → Claude are separate rows in the future compatibility
matrix. “Supported session adapter” must never be shortened to “supported
handoff.” See
[cross-agent-continuation.md](cross-agent-continuation.md).

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
