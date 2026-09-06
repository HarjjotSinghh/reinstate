# Grok handoff fixtures

Synthetic. No real transcripts, credentials, or private paths. Boundary
authority is `updates.jsonl` (an append-only restore log), not
`chat_history.jsonl`, which is model-facing and may be rewritten by
`/compact` — see `internal/transcript/grok.go`'s `GrokReader` doc comment.

## Cases

| Directory | What it covers |
| --------- | -------------- |
| `basic/` | text, one session, no compaction |
| `compacted/` | a `/compact` checkpoint plus `compaction_requests/` holding the pre-compact turns |
| `partial-final-record/` | `updates.jsonl` truncated mid-record (two complete lines, one torn, no trailing newline); `chat_history.jsonl` is a complete, ordinary conversation, since the reader hashes only `updates.jsonl`'s frozen prefix |
| `partial-final-record-windows/` | same shape as `partial-final-record/`, but `summary.json`'s `cwd` and the session directory's percent-encoded workspace segment are Windows-shaped (`C:\Users\fixture-user\code\demo`), so the fixture resolves on a native Windows host |

Both `partial-final-record*` directories are driven, on both platforms,
through the real `handoff.Plan()` pipeline (the function
`rein handoff --no-launch --json` calls) by
`internal/handoff/partial_final_record_route_test.go`'s
`TestPlanPartialFinalRecordByteExactCrossCheck`, which cross-checks the
resulting capsule's `raw_source.byte_offset`/`raw_source.artifact_sha256`
against a boundary independently recomputed from `updates.jsonl`'s own
bytes — not just the reader-layer check in `internal/transcript`'s
`TestGrokReaderPartialFinalRecordExcluded`.
