# Per-agent session storage pages

Each page records where one coding agent keeps its local conversation state,
on every operating system Reinstate targets, and how confident we are about
each row.

Read [../session-storage-map.md](../session-storage-map.md) first. It owns the
confidence vocabulary, the cross-OS summary, and the rules every reader must
follow. These pages are the per-agent detail.

Nothing on these pages is a support claim. A documented layout does not mean
Reinstate reads it. Support states live in
[../compatibility.md](../compatibility.md), and the capability ladder lives in
[../agent-support-tiers.md](../agent-support-tiers.md).

## Shipped readers

| Agent | Page | Tier |
| ----- | ---- | ---- |
| Claude Code | [claude.md](claude.md) | T5 |
| Codex CLI | [codex.md](codex.md) | T5 |
| Gemini CLI | [gemini.md](gemini.md) | T2 |
| OpenCode | [opencode.md](opencode.md) | T2 |
| Grok Build | [grok.md](grok.md) | T2 |
| Kimi Code CLI | [kimi.md](kimi.md) | T2 |

## Phase 5 candidates

| Agent | Page | Current | Target |
| ----- | ---- | ------- | ------ |
| Pi | [pi.md](pi.md) | T0 | T3 |
| Qwen Code | [qwen.md](qwen.md) | T0 | T2 |
| Cursor CLI | [cursor.md](cursor.md) | T0 | T1 |
| GitHub Copilot CLI | [copilot.md](copilot.md) | T0 | T1 |
| Aider | [aider.md](aider.md) | T0 | T1 |
| Cline | [cline.md](cline.md) | T0 | T1 |
| Roo Code | [roo.md](roo.md) | T0 | T1 |
| Amp | [amp.md](amp.md) | T0 | T1 or T0 |
| OpenHands | [openhands.md](openhands.md) | T0 | T0 |
| ZCode | [zcode.md](zcode.md) | T0 | T0 |
| MiniMax | [minimax.md](minimax.md) | T0 | T0 |
| Antigravity CLI | [antigravity.md](antigravity.md) | T0 | T0 |

Every row on every candidate page starts `Unverified`. Promotion requires a
redacted device probe; see [../testing/agent-storage-probe.md](../testing/agent-storage-probe.md).

## Committed device evidence

A macOS probe on 2026-08-16 produced the first `AGENT-PROBE-V1` artifacts for
this phase, under
[../testing/results/agent-probes/](../testing/results/agent-probes/).

| Agent | Artifact | What it settled |
| ----- | -------- | --------------- |
| Kimi Code CLI | `2026-08-16-macos-kimi.json` | Root is `~/.kimi-code`; `session_index.jsonl` exists; bucket is `wd_<project>_<12-hex>` |
| Kimi Code CLI | `2026-08-17-windows-kimi.json` | Promoted Kimi to T1. Five sessions across three projects; identical `state.json` shape to macOS; index enumerated exactly the sessions on disk |
| GitHub Copilot CLI | `2026-08-17-windows-copilot.json` | SQLite appears: `session-store.db` plus a per-session `session.db`, absent from the macOS artifact |
| Qwen Code | `2026-08-16-macos-qwen.json` | Conversations are under `projects/<slug>/chats/`, not `tmp/`; marker corrected |
| Qwen Code | `2026-08-17-windows-qwen.json` | Two real JSONL sessions; first-line keys `cwd, message, parentUuid, provenance, sessionId, timestamp, type, uuid, version` |
| Qwen Code | `2026-08-17-macos-qwen.json` | Real JSONL conversation; keys match Windows; `<uuid-v4>-runtime.json` sidecars |
| Cursor CLI | `2026-08-17-windows-cursor.json` | `~/.cursor/chats/<32-hex>/<uuid-v4>/store.db` plus `meta.json`; editor `projects/` excluded |
| Cursor CLI | `2026-08-17-macos-cursor.json` | Same chats shape as Windows |
| Pi | `2026-08-17-macos-pi.json` | `~/.pi/agent/sessions/<slug>/<slug>-<uuid-v4>.jsonl`; first-line keys `cwd, id, timestamp, type, version` |
| Pi | `2026-08-17-windows-pi.json` | Same shape and keys as macOS |
| GitHub Copilot CLI | `2026-08-17-windows-copilot-cache-clear.json` | Old session ID absent from fresh tree after rename-aside |
| GitHub Copilot CLI | `2026-08-16-macos-copilot.json` | Substantial local `session-state/<uuid>/events.jsonl`; cache-versus-authoritative still open |

**No tier moved.** One platform is not dual-platform evidence, and every
promotion still needs a native Windows artifact.
