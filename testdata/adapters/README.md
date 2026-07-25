# Synthetic adapter fixtures

All content here is **synthetic**. No real user transcripts, tokens, or
credentials. The fixture scanner (`go test ./internal/fixture`) fails the
build if secret-like patterns appear.

- `claude/{macos,windows,wsl}/` — Claude Code JSONL session shapes
- `codex/{macos,windows,wsl}/` — Codex rollout JSONL shapes

Regenerate the deterministic corpus with `fixture.Generate` through its Go
test, then run `make fixture-scan`. Never copy files from a real agent home.
