# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Relicensed core from MIT to **Apache License 2.0** (research recommendation)
- README diagrams: ASCII boxes → Mermaid flowcharts
- Star history embed: dark/light `<picture>` + interactive fallback link
- Docs and CLI help prefer short alias **`rein`** (`reinstate` remains full name)
- **Product positioning:** continuity layer for coding-agent work (not multi-device-only);
  multi-device E2EE sync remains the entry wedge
- Roadmap expanded: Phase 2 local session index → verified resume → portable
  handoffs → automatic sync → thin Console/ACP client → team continuity

### Added

- Initial open-source repository structure
- Project documentation (README, architecture, adapters, security model)
- Community health files (CODE_OF_CONDUCT, CONTRIBUTING, SECURITY, SUPPORT)
- GitHub issue/PR templates and CI workflow scaffolding
- CLI package layout (`cmd/reinstate`, `internal/*`)
- Short CLI alias **`rein`** (symlink to `reinstate` via `make build` / install)
- Apache License 2.0
- [docs/product-strategy.md](docs/product-strategy.md) — strategy, ICP, non-goals (no full ADE/harness)

### Planned

See [ROADMAP.md](ROADMAP.md) for the authoritative phase list. Highlights:

- **Phase 1 public `v0.1.0`:** Claude + Codex encrypted sync (engine largely in place)
- **Phase 2:** local universal session switcher (`sessions` / `search` / `resume` / `last`)
- **Phase 3:** verified resume (workspace + capability fingerprint)
- **Phase 4:** portable cross-agent handoffs (explicit checkpoints)
- **Phase 5+:** auto multi-device habit, thin Console/ACP client, team continuity

## [0.0.0] - 2026-07-25

### Added

- Project bootstrap by **Harjot Singh Rana**
- Public roadmap and governance

---

[Unreleased]: https://github.com/HarjjotSinghh/reinstate/compare/v0.0.0...HEAD
[0.0.0]: https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.0.0
