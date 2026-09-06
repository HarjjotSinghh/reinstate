# Codex partial-final-record (native Windows)

Same shape as `../partial-final-record/` — three complete `rollout` JSONL
records plus a fourth truncated mid-line, no trailing newline — but
`session_meta.payload.cwd` is a Windows path (`C:\Users\fixture-user\code\demo`)
so the fixture's recorded workspace resolves on a native Windows host. Pins
the truncation boundary offset and the SHA-256 of the bytes before it for the
Windows leg of the `codex:D4` cross-check.
