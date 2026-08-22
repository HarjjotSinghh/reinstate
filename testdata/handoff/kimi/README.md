# Kimi Code CLI handoff fixtures

Synthetic `wire.jsonl` trees for the T2 reader. No real transcript, prompt, or
path from any device is copied here.

The record shapes are modelled on Kimi Code CLI `0.36.1` (wire protocol `1.5`),
re-verified on 2026-08-22 by driving the real binary against a throwaway
`KIMI_CODE_HOME` and a loopback-only model endpoint. The important detail these
fixtures encode: **the assistant's text, tool calls, and tool results arrive as
`context.append_loop_event`**, not as a role `"assistant"`
`context.append_message`. See `docs/session-storage/kimi.md`.

| Tree | What it covers |
| ---- | -------------- |
| `basic/` | The native shape: `turn.prompt`, the user `context.append_message` that repeats it, a harness injection, `step.begin` / `content.part` / `tool.call` / `tool.result` / `step.end`, a thinking part, and harness metadata |
| `legacy-migrated/` | Wire protocol `1.0` as written by `kimi migrate`: every message is a `context.append_message`, including the assistant's, and there is no `turn.prompt` |
| `unknown-records/` | `profile.bind` (the system prompt must stay out of the capsule), an unknown top-level type, and an unknown loop-event type |
| `context-rewrite/` | A `context.clear` between two turns; Parse must report `kimi_context_rewritten` |
| `partial-final-record/` | Trailing truncated JSONL line; Snapshot must stop before it |

Session id is `session_01987654-3210-7890-abcd-ef0123456789`.
`state.json` `version` is `2`, with epoch-millis integer timestamps, matching
what the vendor writes. The first `wire.jsonl` record is `metadata`; its
`protocol_version` is `1.5` except in `legacy-migrated/`, which is `1.0`.
