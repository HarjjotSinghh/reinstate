# Kimi Code CLI index fixtures

Synthetic session trees for the T1 index source. No real session, prompt, or
path from any device is copied here.

The record shapes are modelled on Kimi Code CLI `0.36.1` (wire protocol `1.5`)
and were corrected on 2026-08-22 after the vendor's own op registry was checked
in the shipped bundle. The shape these fixtures previously encoded — a role
`"assistant"` `context.append_message` carrying a `toolCalls[]` array, with
ISO-8601 string timestamps — is not what the CLI writes.

What it actually writes:

| Record | Carries |
| ------ | ------- |
| `turn.prompt` | the operator's text |
| `context.append_message` | **only role `"user"`** — the prompt repeated, and harness injections |
| `context.append_loop_event` | the whole assistant side: `step.begin`, `content.part`, `tool.call` (arguments under `args`), `tool.result`, `step.end` |

Timestamps are **epoch-millis integers** everywhere: `created_at` and `time` in
`wire.jsonl`, `createdAt` and `updatedAt` in `state.json`.

The role `"assistant"` `context.append_message` is real, but only for a session
migrated from the legacy `kimi-cli` store, which writes protocol `1.0` and no
`turn.prompt` at all. Both committed trees here are native `1.5`, because that
is what a current install produces; the migrated shape is covered by
`TestLegacyMigratedShapeStillIndexes` and by
`testdata/handoff/kimi/legacy-migrated`.

`NEVER_COPY_THIS_SYSTEM_PROMPT` and `NEVER_COPY_THIS_THINKING` are deliberate.
A system prompt and a thinking part are not conversation, and
`TestModelInternalsNeverReachTheIndex` asserts neither reaches an index record.

See `docs/session-storage/kimi.md`.
