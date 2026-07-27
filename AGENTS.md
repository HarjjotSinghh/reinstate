# AGENTS.md — guidance for AI coding agents working on Reinstate

## Project

**Reinstate** is an open-source **continuity layer for coding-agent work**:
search, resume, and hand off sessions across agents, projects, environments,
and devices — with E2E encryption and BYO storage for multi-device sync.

**CLI:** prefer short alias **`rein`**. Full name **`reinstate`** is the same binary.

- **Author:** Harjot Singh Rana ([@HarjjotSinghh](https://github.com/HarjjotSinghh))
- **License:** Apache-2.0
- **Toolchain:** Go 1.25.12+ (1.24 is end-of-life and not a release target)
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
testdata/            Deterministic synthetic fixtures
references/          Product research (not runtime code)
```

## Commands

```bash
make build    # ./bin/reinstate + ./bin/rein (symlink)
make test
make vet
make verify
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

Phase 1 remains deliberately narrow: same-vendor Claude Code and Codex session
sync. Portable handoffs, MCP, skills, config, and broader indexing come later.
The later configuration direction is **universal agent configuration**:
declare MCP servers, skills, hooks/loops, plugins, marketplaces, instructions,
and safe settings once; adapters render them into supported harnesses and
encrypted sync distributes only non-secret desired state across devices. Never
turn that into raw config-tree mirroring, credential sync, a Reinstate-owned
plugin runtime, or a Reinstate-owned marketplace. See
`docs/universal-configuration.md`.

## Style

- Conventional Commits
- Go: standard formatting (`gofmt`), table-driven tests
- No network in unit tests without fakes
