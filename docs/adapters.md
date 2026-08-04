# Adapters

Reinstate separates per-agent capabilities:

- **local read adapters** discover bounded metadata and user-prompt search text;
- **native executors** resume/fork through the same vendor;
- **sync adapters** export and restore vendor-native session files;
- Phase 4 **transcript-source** and **handoff-target** adapters parse portable
  history and create linked destination sessions; and
- later **configuration adapters** normalize portable intent and render each
  harness's native MCP/skills/plugins/settings format.

Phase 4 adds separate, directional **transcript-source** and
**handoff-target** capabilities. A session adapter that can sync bytes does not
automatically qualify to parse portable history or create a safe destination
session.

## Phase 1 (`v0.1.0`)

One capability never implies another. Read-only agents do not receive dummy
resume, fork, export, or restore implementations.

## Capability matrix

| Adapter | Phase 2 local index | Native resume/fork | `v0.1.0` encrypted sync | Universal config |
| ------- | ------------------- | ------------------ | ------------------------- | ---------------- |
| Claude Code | Included in `v0.2.0` | Included in `v0.2.0` | Supported | Later |
| OpenAI Codex CLI | Included in `v0.2.0` | Included in `v0.2.0` | Supported | Later |
| Gemini CLI | Read-only in `v0.2.0` | No | No | Later |
| OpenCode | Read-only in `v0.2.0` | No | No | Later |
| Cursor | Exploring | No | No | Exploring |
| Grok Build | Exploring | No | No | Planned |

Phase 2 automated gates and the complete tagged-artifact matrix passed on Apple
Silicon macOS and native Windows x64. Stable `v0.2.0` support is limited to
those verified platforms; Intel macOS and Linux/WSL2 remain preview/unverified.

## Phase 2 local read contract

All local records use:

```text
<agent>:<native-session-id>
```

Local discovery is config-independent and does not inherit Phase 1 project
mappings that would hide unmapped sessions. Sources expose only identity,
timestamps, workspace/project, recorded branch, title/name, bounded
user-authored prompt text, known file fields, counts, source fingerprint, and
capabilities. Assistant messages/reasoning, tool output, environment dumps,
credentials, and auth stores are excluded from the index.

| Adapter | Read source |
| ------- | ----------- |
| Claude Code | Stream project JSONL; exclude subagent artifacts; ignore incomplete trailing record |
| Codex CLI | Stream date-partitioned rollout JSONL and structural session metadata |
| Gemini CLI | Defensively read recognizable project chat JSON under the vendor data root |
| OpenCode | Use the documented local JSON session-list surface through a bounded runner |

Local-index capabilities and Phase 1 sync compatibility are separate
contracts. The local record advertises whether native resume/fork is available;
the existing sync adapter continues to enforce its verified version range
before export or restore.

## Native execution

| Agent | Resume | Fork |
| ----- | ------ | ---- |
| Claude Code | `claude --resume ID` | `claude --resume ID --fork-session` |
| Codex | `codex resume ID` | `codex fork ID` |

Plans store executable, argv, and recorded cwd separately. They never construct
a shell command string. Gemini/OpenCode resume or fork fails with compatibility
exit `5`.

## Future session and handoff adapters

| Adapter | Session discovery | Cross-agent handoff |
| ------- | ----------------- | ------------------- |
| Gemini CLI | Read-only in `v0.2.0` | Phase 4 after Claude ↔ Codex |
| OpenCode | Read-only in `v0.2.0` | Phase 4 after Claude ↔ Codex |
| Cursor | Exploring | Exploring |
| Grok Build | Exploring | Exploring |
| Copilot CLI / Orca / others | Exploring | Requires public format/API and acceptance evidence |

## Future configuration adapters

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

See [compatibility.md](compatibility.md). Phase 1 sync adapters report
`SUPPORTED`, `UNTESTED`, `UNSUPPORTED`, or `NOT_INSTALLED`; Phase 2 local
records report read/native capabilities independently.

## Claude project identity

Claude Code stores a project beneath a directory key derived from that
device's absolute project path. Reinstate records the configured canonical
project ID in snapshots and recomputes Claude's directory key from the
destination device's `local_root`. Snapshot archive paths remain source
metadata; they are never reused as cross-device restore destinations.

Reinstate verifies the exact planned destination after restore. A matching session
ID elsewhere in `~/.claude/projects` is not accepted as success.

The Phase 2 local reader intentionally sees all local top-level Claude sessions,
including projects that are not mapped for encrypted sync.

## Codex project identity

Codex stores the source working directory in each rollout's structural
`session_meta.cwd`. When project mappings are configured, Reinstate resolves that
directory to the configured canonical project ID during discovery and excludes
rollouts outside those mapped roots. Export normalizes the resolved source root
to a `${REPO:<id>}` token, and restore expands it through the destination
device's `local_root`. This keeps Windows and macOS paths out of portable
session identity while preserving Codex's native date-partitioned rollout
layout.

The Phase 2 local reader similarly indexes local Codex rollouts without
requiring a canonical sync mapping.

## Exclusions

Sync adapters hard-exclude auth, credentials, tokens, caches, logs, and
regenerable dependencies. The Phase 2 index additionally excludes assistant
messages/reasoning, tool output, environment dumps, and auth stores. Future
configuration profiles may carry secret **references** but never secret
values. Fixtures are synthetic and scanned for secrets.
