# AGENTS.md — guidance for AI coding agents working on Reinstate

## Project

**Reinstate** is an open-source CLI that syncs AI coding agent sessions and
configs across devices with E2E encryption and BYO storage.

- **Author:** Harjot Singh Rana ([@HarjjotSinghh](https://github.com/HarjjotSinghh))
- **License:** Apache-2.0
- **Language:** Go 1.22+
- **Module:** `github.com/HarjjotSinghh/reinstate`

## Non-negotiables

1. **Never commit secrets**, real session transcripts, API keys, or passphrases.
2. **Security defaults stay safe** — encryption on; credentials excluded.
3. **Same-vendor resume only** — do not invent cross-agent session translation.
4. **Path remapping is first-class** — Windows ↔ macOS is the default scenario.
5. Prefer small, reviewable PRs over large rewrites.

## Layout

```
cmd/reinstate/       CLI
internal/adapter/    Agent adapters
internal/crypto/     Encryption
internal/pathmap/    Portable path rewriting
internal/sync/       Push/pull/manifest
internal/version/    Build version
docs/                Human documentation
testdata/            Golden fixtures
references/          Product research (not runtime code)
```

## Commands

```bash
make build    # ./bin/reinstate
make test
make vet
```

## Docs to update when you change UX

- `README.md` (user-facing)
- `docs/getting-started.md`, `docs/adapters.md` as needed
- `CHANGELOG.md` under `[Unreleased]`

## Product context

Read `references/` for research background. Positioning:

> Universal, vendor-neutral, encrypted sync of sessions + MCP/skills/config —
> not Dropbox for a single agent folder.

## Style

- Conventional Commits
- Go: standard formatting (`gofmt`), table-driven tests
- No network in unit tests without fakes
