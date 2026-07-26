# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Refuse to overwrite an initialized Reinstate home unless `rein init --force`
  is explicitly selected, and back up the previous `config.toml` and
  `state.json` together before replacement.
- Require `rein init --profile-id` to find the existing encrypted remote
  manifest before writing local configuration, catching endpoint, bucket, and
  prefix mistakes during setup.
- Return an error when a joined or established profile's `status`, `diff`,
  `pull`, or `push` cannot find `manifest.age`, instead of reporting a healthy
  empty profile. A new first-device profile may still report an empty remote
  and create its manifest on the first push.

## [0.1.0-rc.4] - 2026-07-26

### Fixed

- Map Claude Code sessions to the configured canonical project ID and derive
  restore destinations from each device's `local_root`, including Claude's
  exact Windows/macOS directory-key rules for spaces, Unicode, and long paths.
- Verify restored sessions at the exact planned vendor path instead of
  accepting a matching session ID elsewhere in the agent tree.
- Fail closed with a repush instruction when a legacy Claude snapshot lacks a
  canonical project mapping, avoiding false-success cross-device restores.
- Exclude unmapped Claude projects when canonical mappings are configured and
  require a destination mapping for canonical snapshots, including empty-map
  configurations.
- Normalize Claude transcript paths through resolved project roots while
  denormalizing them through the destination device's configured root.
- Report `would push` during `push --dry-run` instead of claiming that a
  snapshot was uploaded.

### Changed

- Pin the public installers and end-user setup prompts to `v0.1.0-rc.4`.
- Harden the two-device acceptance runbook with a fresh-profile requirement,
  exact-ID Codex resume, Claude sibling-session disambiguation, hidden-prompt
  passphrase guards, and byte-level ciphertext checks.
- Add coordinated Mac Claude Code and native-Windows Codex verification prompts
  that produce separate sanitized acceptance reports.

## [0.1.0-rc.3] - 2026-07-26

### Fixed

- Accept the tested Claude Code `2.1.219`–`2.1.220` and Codex CLI
  `0.133.0`–`0.145.0` ranges instead of requiring one exact vendor version.
- Make `setup check` fail with compatibility exit code `5` when an installed
  adapter is untested and therefore blocked from push/pull.
- Require a valid Reinstate config before `conflicts list` or `conflicts show`
  can report an empty result.
- Exclude Claude Code `subagents/` artifacts from the top-level resumable
  session list.
- Override the website's transitive `path-to-regexp` dependency to patched
  version `6.3.0`.

## [0.1.0-rc.2] - 2026-07-25

### Fixed

- Release CI restores the remote annotated tag object after checkout before
  verifying its SSH signature.

## [0.1.0-rc.1] - 2026-07-25

### Added

- Phase 0 / Phase 1 authority plan and product contracts
- ADR documenting Phase 0 foundation and Phase 1 Claude/Codex sessions scope
- Compatibility matrix and support states
- Cobra CLI with stable exit codes (`rein` / `reinstate`)
- Versioned config (TOML) and state (JSON) with atomic writes
- Device detection (macOS/Windows/Linux/WSL2; WSL1 refused)
- Redacted `doctor` / `setup check` and synthetic self-test
- Synthetic fixtures + secret scanner
- Hardened CI (fmt/vet/test/race/docs/fixtures/lint/security) and GoReleaser config
- Checksum-verifying installers (`scripts/install.sh`, `scripts/install.ps1`)
- Versioned AI-agent setup prompts under `docs/prompts/`
- S3-compatible backend client + memory test backend
- age scrypt envelopes with tamper/wrong-passphrase tests
- Path mapping, project identity, manifests, push/pull/conflicts
- Claude Code and Codex adapters (fixture-backed)
- Native OS-keyring credential storage and hidden TTY/file-descriptor passphrase input
- Streamed portable artifacts with authenticated metadata/hash validation
- Timestamped restore backups, mutation locks, profile isolation, and executable conflict resolution
- Active Claude Code/Codex process refusal before mutating session restores
- Cross-platform installer contract tests, release SBOMs, and artifact attestations
- Safe installer replacement checks, native Windows `rein.exe`, and release verification scripts
- Deterministic six-environment adapter fixtures and structured contributor/compatibility workflows
- Short CLI alias **`rein`** (same binary as `reinstate`)
- [Product strategy](docs/product-strategy.md) defining the continuity-layer
  positioning, first ICP, product layers, and non-goals

### Changed

- Relicensed core from MIT to **Apache License 2.0**
- README diagrams: ASCII boxes → Mermaid flowcharts
- Roadmap and support docs aligned to Phase 0 foundation + Phase 1 sessions
- Copy-paste setup prompts now continue through init, dry-run, sync, and restore verification
- Codex restores preserve date-partitioned rollout paths; both adapters fail closed on unverified versions
- Star history embed: dark/light `<picture>` + interactive fallback link
- Docs and CLI help prefer short alias **`rein`** (`reinstate` remains full name)
- **Product positioning:** continuity layer for coding-agent work (not multi-device-only);
  multi-device E2EE sync remains the entry wedge
- Roadmap expanded: Phase 2 local session index → verified resume → portable
  handoffs → automatic sync → thin Console/ACP client → team continuity

### Removed

- Invented `v0.0.0` changelog release history (no tag/release existed)
- Secret-bearing init flags, plaintext credential files, and ordinary environment passphrases

### Planned

See [ROADMAP.md](ROADMAP.md) for the authoritative phase list. Highlights:

- **Phase 1 public `v0.1.0`:** Claude + Codex encrypted sync (engine largely in place)
- **Phase 2:** local universal session switcher (`sessions` / `search` / `resume` / `last`)
- **Phase 3:** verified resume (workspace + capability fingerprint)
- **Phase 4:** portable cross-agent handoffs (explicit checkpoints)
- **Phase 5+:** auto multi-device habit, thin Console/ACP client, team continuity

---

[Unreleased]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.4...HEAD
[0.1.0-rc.4]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.3...v0.1.0-rc.4
[0.1.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.2...v0.1.0-rc.3
[0.1.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.1...v0.1.0-rc.2
[0.1.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.1.0-rc.1
