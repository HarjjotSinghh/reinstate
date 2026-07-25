# AGENTS.md — guidance for AI coding agents working on Reinstate

## Project

**Reinstate** is an open-source **continuity layer for coding-agent work**:
search, resume, and hand off sessions across agents, projects, environments,
and devices — with E2E encryption and BYO storage for multi-device sync.

**CLI:** prefer short alias **`rein`**. Full name **`reinstate`** is the same binary.

- **Author:** Harjot Singh Rana ([@HarjjotSinghh](https://github.com/HarjjotSinghh))
- **License:** Apache-2.0
- **Language:** Go 1.22+ (see `go.mod` for pin)
- **Module:** `github.com/HarjjotSinghh/reinstate`

## Non-negotiables

1. **Never commit secrets**, real session transcripts, API keys, or passphrases.
2. **Security defaults stay safe** — encryption on; credentials excluded.
3. **Native resume is same-vendor** — do not invent silent cross-agent transcript
   translation. Cross-agent work uses **explicit portable handoffs** only.
4. **Path remapping is first-class** — Windows ↔ macOS is the flagship multi-device case.
5. **Continuity, not a harness** — do not grow into a full ADE (editor/terminal/multi-agent scheduler). Agents execute; Reinstate finds, verifies, hands off, syncs.
6. Prefer small, reviewable PRs over large rewrites.

## Product direction

Authoritative roadmap and strategy:

- `ROADMAP.md` — phases 0–7
- `docs/product-strategy.md` — positioning, ICP, non-goals
- `PRODUCT.md` — brand and marketing constraints

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
make build    # ./bin/reinstate + ./bin/rein (symlink)
make test
make vet
./bin/rein version
```

## Docs to update when you change UX

- `README.md` (user-facing)
- `docs/getting-started.md`, `docs/adapters.md` as needed
- `CHANGELOG.md` under `[Unreleased]`

## Product context

Read `references/` and `docs/product-strategy.md`. Positioning:

> Continuity layer for coding-agent work — local session index and verified
> resume for everyone; encrypted multi-device sync as the wedge; not Dropbox for
> a single agent folder; not another ADE.

## Style

- Conventional Commits
- Go: standard formatting (`gofmt`), table-driven tests
- No network in unit tests without fakes
