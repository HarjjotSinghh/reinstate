# Synthetic adapter fixtures

All content here is **synthetic**. No real user transcripts, tokens, or
credentials. The fixture scanner (`go test ./internal/fixture`) fails the
build if secret-like patterns appear.

- `claude/{macos,windows}/` — Claude Code JSONL session shapes
- `codex/{macos,windows}/` — Codex rollout JSONL shapes
