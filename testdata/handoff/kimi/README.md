# Kimi Code CLI handoff fixtures

Synthetic `wire.jsonl` trees for the T2 reader. No real transcripts.

| Tree | What it covers |
| ---- | -------------- |
| `basic/` | One user turn, two assistant messages, two tool calls, harness metadata |
| `unknown-records/` | `profile.bind` (system prompt must stay out of the capsule) and `kimi.future_event` (unknown type, referenced, no body) |
| `partial-final-record/` | Trailing truncated JSONL line; Snapshot must stop before it |

Session id is `session_01987654-3210-7890-abcd-ef0123456789`.
`state.json` `version` is `2`. First `wire.jsonl` record is `metadata` with `protocol_version` `1.5`.
