# Codex partial-final-record (native Windows)

Same shape as `../partial-final-record/` — three complete `rollout` JSONL
records plus a fourth truncated mid-line, no trailing newline — but
`session_meta.payload.cwd` is a Windows path (`C:\Users\fixture-user\code\demo`).

Pins the truncation boundary offset and the SHA-256 of the bytes before it,
independently recomputed from the fixture's own bytes, at two layers:

- `internal/transcript`'s `TestCodexReaderPartialFinalRecord/windows`
  exercises `CodexReader.Snapshot`/`Parse` directly against this file.
- `internal/handoff`'s `TestPlanPartialFinalRecordByteExactCrossCheck`
  (`partial_final_record_route_test.go`) drives the real `handoff.Plan()`
  pipeline — the function `internal/cli/handoff.go` calls for
  `rein handoff --no-launch --json` — against this fixture from a throwaway
  git repository named `demo`, and cross-checks the resulting capsule's
  `raw_source.byte_offset`/`raw_source.artifact_sha256`.

See `../claude/partial-final-record-windows/README.md`'s correction note:
`codex:D4` never carried the same false OS-shape-blocked claim this
workstream first attached to `claude:D4` (the second pass already reached a
byte-exact cross-check for `codex` on macOS against a real session, per
`docs/testing/results/2026-09-06-windows-v060rc1-pretag.md` line 1851), but
the same mechanism applies here too: `internal/handoff/workspace.go`'s
`shouldRemapWorkspace` and `sameProjectLeaf` gate the `handoff` route's
workspace remap/refusal on the literal `/fixture-user/` substring plus a
matching git-root directory name, identically on every OS — never on
`isForeignOSPath`. This fixture's Windows shape is reader-boundary-math
coverage, not a different CLI-route capability; see
`TestPlanPartialFinalRecordRefusalIsOSShapeIndependent` in
`internal/handoff/partial_final_record_route_test.go`.
