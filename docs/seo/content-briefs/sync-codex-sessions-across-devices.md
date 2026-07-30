# Content brief: sync Codex sessions across devices

## Page

- Proposed title: Sync Codex Sessions Across Devices
- URL: `/guides/sync-codex-sessions-across-devices`
- Page type: practical guide
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers continuing Codex CLI work on another configured
  computer.
- Primary problem: a Codex rollout refers to the source checkout path and is
  absent from the destination's native date-partitioned rollout tree.
- Primary query: `sync Codex sessions across devices`
- Secondary questions: where Codex stores rollouts; how `session_meta.cwd` is
  remapped; whether `history.jsonl` moves; how native resume is verified.
- Search intent: how-to
- Expected next action: inspect a session-specific pull dry-run.
- Existing-page overlap reviewed: the Codex integration owns support facts;
  this guide owns the command sequence.

## Product truth

- Current capabilities used: same-vendor rollout discovery, structural
  `session_meta.cwd` tokenization, local encryption, immutable snapshots,
  destination path expansion, backup/atomic restore, and native Codex resume.
- Current limitations: pre-1.0 stable release; native Windows, macOS amd64,
  WSL2, remote CI, and physical two-device acceptance remain open gates.
- Version tested: Reinstate `v0.1.0`; Codex ranges live on compatibility.
- Evidence: adapter/pathmap/crypto/sync implementation and tests, deterministic
  synthetic rollout fixtures, CLI contracts, and release notes.
- Claims that require verification: physical restore success, performance,
  agent versions beyond the declared range, and unsupported platforms.
- Prohibited or roadmap claims: Codex-to-Claude native resume, transcript
  rewriting, credentials/configuration sync, and stable certification.

## Outline

- H1: How to sync Codex sessions across devices
- Direct answer: identify a rollout, encrypt and push it, map the same project
  ID on the destination, inspect pull dry-run, restore, and use Codex's native
  resume command.
- H2/H3 structure: prerequisites; key points; install/setup; list; push; dry-run;
  pull; native resume; expected outputs; parameters; platforms; security;
  verification; rollback; common errors; FAQ; related resources.
- Original value: explains why only the structural working-directory field is
  rewritten while free-form transcript text is preserved.

## Links

- Inbound opportunities: homepage, integrations hub, Codex integration,
  compatibility, troubleshooting, macOS/Windows use case.
- Outbound internal links: getting started, Codex integration, adapters,
  architecture, security, compatibility, troubleshooting, changelog.
- External primary sources: Reinstate source/release and current Codex native
  resume documentation where needed.

## Structured data

- Primary type: `TechArticle`
- Breadcrumbs: Home → Guides → page
- Additional type: `HowTo` only for visible steps shared with the schema.
- Search-feature caveat: Schema.org validity is not a promise of a Google
  search enhancement.

## Media

- Diagram: structural cwd remapping from source project root to canonical token
  to destination project root.
- Alt text: Codex rollout preserving native layout while its structural working
  directory changes between two devices.
- Raw data: no public benchmark or physical-device success dataset exists yet.

## Acceptance criteria

- [x] Every command has purpose, parameters, expected output, platform note,
  failure mode, and safe undo
- [x] Linux/WSL2 wording does not imply stable Phase 1 certification
- [x] `history.jsonl` and credentials are not described as synced
- [x] Native resume remains Codex → Codex
- [x] Visible steps exactly match `HowTo` schema
- [x] Route-specific social card and all website gates pass

Review evidence: [published content acceptance register](../content-acceptance-register.md).
