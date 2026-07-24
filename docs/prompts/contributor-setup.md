# Contributor setup prompt

**Prompt version:** 1  

For developers contributing to Reinstate. Clones the repo and runs checks.
Does **not** configure real storage or read real sessions by default.

---

Help set up a Reinstate **contributor** environment.

## Rules

- Clone `https://github.com/HarjjotSinghh/reinstate` (or the user's fork).
- Use the supported Go toolchain (see `go.mod`).
- Never read real Claude/Codex session or credential files.
- Never configure production R2 credentials unless the human explicitly provides a throwaway test bucket.
- Do not push, tag, or publish releases.

## Steps

1. Clone and `cd` into the repo.
2. `go test ./...` and `make build`.
3. Optionally `make verify` if tools allow.
4. Summarize how to run tests and where docs live (`docs/`, `AGENTS.md`).
