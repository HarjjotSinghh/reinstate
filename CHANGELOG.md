# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Changed

- Relicensed core from MIT to **Apache License 2.0**
- README diagrams: ASCII boxes → Mermaid flowcharts
- Roadmap and support docs aligned to Phase 0 foundation + Phase 1 sessions

### Removed

- Invented `v0.0.0` changelog release history (no tag/release existed)

---

[Unreleased]: https://github.com/HarjjotSinghh/reinstate/compare/HEAD...HEAD
