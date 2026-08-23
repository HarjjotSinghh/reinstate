# Synthetic adapter fixtures

All content here is **synthetic**. No real user transcripts, tokens, or
credentials. The fixture scanner (`go test ./internal/fixture`) fails the
build if secret-like patterns appear.

- `claude/{macos,windows,wsl}/` — Claude Code JSONL session shapes
- `codex/{macos,windows,wsl}/` — Codex rollout JSONL shapes
- `opencode/{macos,windows}/store.sql` — OpenCode embedded-store seeds

OpenCode keeps every session in one SQLite database rather than a file per
session, so its fixture is a deterministic SQL seed instead of a session file:
committing a binary `.db` would be neither reviewable nor byte-stable across
SQLite builds. Tests hydrate an `opencode.db` from the seed. The `macos` and
`windows` seeds differ only in path shape, which is what the cross-OS remapping
test exercises; each carries a `credential` row so the exclusion contract has
something to prove is never exported.

Regenerate the deterministic corpus with `fixture.Generate` through its Go
test, then run `make fixture-scan`. Never copy files from a real agent home.
