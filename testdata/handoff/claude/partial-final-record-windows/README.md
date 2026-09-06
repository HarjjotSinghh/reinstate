# Claude partial-final-record (native Windows)

Same shape as `../partial-final-record/` — two complete JSONL records plus a
third record truncated mid-line, no closing brace, no trailing newline — but
`cwd` and the `projects/` directory name (`C--Users-fixture-user-code-demo`)
are Windows-shaped so the fixture's recorded workspace resolves on a native
Windows host. Pins the truncation boundary offset and the SHA-256 of the
bytes before it for the Windows leg of the `claude:D4` cross-check that the
macOS-shaped fixture could not reach on Windows (`handoff` refused: working
directory is a different repository than the source session).
