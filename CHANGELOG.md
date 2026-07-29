# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-rc.8] - 2026-07-29

### Fixed

- Stop treating "no open file handle" as proof that a session is not in use.
  Claude Code appends to its session file and closes it again, so a live Claude
  Code session holds no handle and the handle-only check introduced in
  `v0.1.0-rc.7` reported it as free, letting a restore target a session someone
  was working in. Liveness now also matches an agent that names the exact
  session on its command line, or that is working inside the session's mapped
  project, and biases toward "in use" because the fork policy makes a false
  positive cheap and a false negative expensive. Unrelated agents in other
  projects still never block a restore.

### Changed

- Clarify the landing-page file-sync comparison around machine-specific project
  keys, make the failed-resume mismatch readable at normal scroll speed, and
  replace ambiguous identity ownership language with precise remapping and
  reconstruction behavior.

## [0.1.0-rc.7] - 2026-07-28

### Added

- Add a production SEO, answer-engine optimization, and AI-search foundation
  for the Astro website, including canonical metadata, crawler policy,
  structured data, sitemap, RSS, public product pages, and automated CI checks.
- Add unique 1200×630 social preview images for every indexable route, using
  Reinstate's logo, typography, palette, and axonometric illustration language.
- Add a reproducible 1280×640 GitHub repository preview plus an owner-operated,
  evidence-gated metadata, release-summary, and launch-post runbook.
- Add answer-first integration, compatibility, security, use-case, project,
  open-source, and changelog pages with explicit release-candidate boundaries.
- Add reviewed Claude Code and Codex session-sync guides, an engineering blog,
  a privacy notice, RSS distribution, and a machine-readable security contact.
- Add generated-site SEO, internal-link, anchor, social-image, and static
  performance regression gates to tests, CI, and production deployment checks.
- Add reviewed IndexNow sitemap-diff planning, server-only key proof, bounded
  batching and retries, soft-fail response logging, and non-submitting CI tests.

### Changed

- Expand documentation metadata with search intent, freshness, and topic fields
  while exposing visible review dates and breadcrumbs.
- Point the repository's website reference at the canonical `reinstate.dev`
  domain.
- Separate signed website-only deployment identity
  (`website-vYYYY.MM.DD.N`) from semantic CLI release tags while retaining
  explicit, byte-verified installer parity with the release derived from both
  public bootstraps.

### Fixed

- Scope the restore active-agent check to the exact session file being
  replaced instead of asking whether any Claude Code or Codex process is alive
  on the host. Running unrelated agents in other projects is the normal state
  of a working machine and no longer blocks `pull` or `conflicts resolve`, so
  nobody has to close background agents to restore a session. Detection uses
  open file handles (`lsof` on Unix, Restart Manager on Windows) and falls back
  to the previous host-wide answer only where handles cannot be enumerated,
  reporting that imprecision in the refusal message.
- Restore a session that genuinely is in use alongside the live one as a
  distinct vendor-safe session instead of refusing, so a restore never waits on
  a human closing an agent. The fork identity is derived from the snapshot, so
  repeating the pull is idempotent rather than accumulating copies.
- Detect a concurrent agent write to a restore target and abandon the restore
  instead of discarding those changes at the final rename.
- Allow the guarded immutable Vercel discoverability smoke to record and
  narrowly exempt the provider-injected preview `noindex` header while keeping
  the promoted production-origin check strict.
- Keep local CLI build metadata anchored to `v*` release tags so website-only
  deployment tags cannot become the reported Reinstate version.
- Parse both structured Vercel CLI 57 deployment results and legacy bare-URL
  output before immutable installer verification and production promotion.

## [0.1.0-rc.6] - 2026-07-27

### Changed

- Expand the post-Phase-1 roadmap from a generic MCP/skills sync bullet into
  universal agent configuration: one non-secret desired-state profile rendered
  across supported harnesses and encrypted across devices.
- Document planned MCP, skills/instructions, hooks/loops, plugins,
  marketplaces, safe settings, drift reconciliation, supply-chain controls,
  and authentication coordination while keeping credentials excluded.
- Pin the public installers, end-user setup prompts, compatibility evidence,
  and fresh-device acceptance runbook to `v0.1.0-rc.6`.
- Require setup agents to preserve and confirm an existing absolute
  `REINSTATE_HOME` instead of silently falling back to the default home.
- Add committed RC6 Mac Claude Code and native-Windows Codex acceptance prompts
  that keep evidence and report ownership isolated by device.
- Disable automatic Vercel Git deployments and require a signed-tag production
  workflow that verifies both immutable and promoted installer routes byte for
  byte.

### Fixed

- Validate an additional device's encrypted remote manifest with a readable
  object request instead of a metadata-only `HeadObject` probe, avoiding
  Cloudflare R2's generic `400 Bad Request` failure while still leaving no
  local configuration behind when the probe fails.
- Resolve Codex rollout working directories to configured canonical project
  IDs, exclude unmapped projects, normalize resolved roots during export, and
  reject duplicate mappings.
- Report `would pull` during `pull --dry-run` instead of claiming that sessions
  were restored.
- Return the stable redacted `config missing` error without exposing the
  absolute Reinstate home or `config.toml` path.
- Delimit the PowerShell replacement prompt's target-version variable so the
  requested version is visible before confirmation.

## [0.1.0-rc.5] - 2026-07-27

### Changed

- Pin the public installers, end-user setup prompts, compatibility evidence, and
  physical-device acceptance runbook to `v0.1.0-rc.5`.
- Keep local and CI verification release-equivalent while avoiding redundant
  documentation-contract, fixture-scan, and production-KDF work.
  High-level deterministic tests use real age envelopes at a reduced test-only
  scrypt cost; the ordinary full suite still covers the production default.
- Add `make quick` as an explicitly non-release fast development gate.

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
- Bound POSIX installer replacement prompts to 30 seconds by default, reject
  invalid timeout overrides, and fail closed immediately when the active shell
  cannot perform a timed TTY read, preventing unattended `/dev/tty` hangs and
  detecting timed-read support correctly across macOS Bash 3 and Linux Bash 5.

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
- **Phase 5+:** universal cross-harness configuration + auto multi-device habit,
  thin Console/ACP client, team continuity

---

[Unreleased]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.8...HEAD
[0.1.0-rc.8]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.7...v0.1.0-rc.8
[0.1.0-rc.7]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.6...v0.1.0-rc.7
[0.1.0-rc.6]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.5...v0.1.0-rc.6
[0.1.0-rc.5]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.4...v0.1.0-rc.5
[0.1.0-rc.4]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.3...v0.1.0-rc.4
[0.1.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.2...v0.1.0-rc.3
[0.1.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.1...v0.1.0-rc.2
[0.1.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.1.0-rc.1
