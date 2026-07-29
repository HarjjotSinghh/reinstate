# Content brief: sync Claude Code sessions across devices

## Page

- Proposed title: Sync Claude Code Sessions Across Devices
- URL: `/guides/sync-claude-code-sessions-across-devices`
- Page type: practical guide
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0-rc.8`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers moving Claude Code work between a work and
  personal computer or a desktop and laptop.
- Primary problem: the destination Claude Code installation cannot see or
  resume a vendor-native session stored on another device.
- Primary query: `sync Claude Code sessions across devices`
- Secondary questions: where Claude stores sessions; how macOS/Windows paths
  change; whether credentials move; how to verify native resume; how to undo.
- Search intent: how-to
- Expected next action: complete a dry-run and inspect the planned restore.
- Existing-page overlap reviewed: the integration page owns support/evaluation;
  this guide owns the executable workflow.

## Product truth

- Current capabilities used: same-vendor discovery, encrypted push/pull,
  canonical project mapping, destination-key restore verification, backups,
  conflict checks, and Claude's native resume command.
- Current limitations: release candidate; stable native Windows, macOS amd64,
  WSL2, remote CI, and physical two-device acceptance remain open gates.
- Version tested: source contract for Reinstate `v0.1.0-rc.8`; Claude stable
  range is maintained on the compatibility page.
- Evidence: CLI help/contracts, adapter/pathmap/crypto/sync tests, synthetic
  fixtures, compatibility data, and release notes.
- Claims that require verification: any physical-device success result, timing,
  throughput, restore success rate, or support outside current ranges.
- Prohibited or roadmap claims: Claude-to-Codex native resume, credential sync,
  universal configuration, and stable certification.

## Outline

- H1: How to sync Claude Code sessions across devices
- Direct answer: push a selected Claude session as locally encrypted state,
  pull it through the destination project mapping, verify the dry-run, then
  resume it with Claude Code.
- H2/H3 structure: prerequisites; key points; install/setup; discover; dry-run;
  push; pull; native resume; expected output; parameters; platform notes;
  security; verification; rollback; common errors; FAQ; related resources.
- Original value: explains Claude's destination project-directory key and why
  canonical mapping is required rather than copying a source path literally.

## Links

- Inbound opportunities: homepage, integrations hub, Claude integration,
  compatibility, troubleshooting, macOS/Windows use case.
- Outbound internal links: getting started, Claude integration, adapters,
  architecture, security, compatibility, troubleshooting, changelog.
- External primary sources: Reinstate source/release and Claude's current native
  resume documentation where needed.

## Structured data

- Primary type: `TechArticle`
- Breadcrumbs: Home → Guides → page
- Additional type: `HowTo` only for visible steps shared with the schema.
- Search-feature caveat: Schema.org validity does not guarantee a Google rich
  result; Google feature support must be checked separately.

## Media

- Diagram: reuse the branded continuity model only if it adds clarity beyond
  the written path mapping.
- Alt text: Claude Code session encrypted on one device and restored into the
  mapped project path on another.
- Raw data: no benchmark or physical-device success dataset is available yet.

## Acceptance criteria

- [x] Every command has purpose, parameters, expected output, platform note,
  failure mode, and safe undo
- [x] Linux/WSL2 wording does not imply stable Phase 1 certification
- [x] No secret, bucket name, project path, or real transcript enters examples
- [x] Native resume remains Claude Code → Claude Code
- [x] Visible steps exactly match `HowTo` schema
- [x] Route-specific social card and all website gates pass

Review evidence: [published content acceptance register](../content-acceptance-register.md).
