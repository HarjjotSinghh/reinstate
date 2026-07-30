# Content brief: move a coding-agent session from Mac to Windows

## Page

- Proposed title: Move a Coding-Agent Session from Mac to Windows
- URL: `/guides/move-a-coding-agent-session-from-mac-to-windows`
- Page type: platform-specific practical guide
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers moving an active Claude Code or Codex CLI task
  from a macOS checkout to a native Windows checkout.
- Primary problem: vendor-native state contains structural source paths that
  are invalid on the destination.
- Primary query: `move coding agent session from Mac to Windows`
- Secondary questions: how path remapping works; whether Git moves the session;
  whether WSL is the same device; how native resume is verified.
- Search intent: platform-specific how-to
- Expected next action: configure one canonical project ID and inspect a scoped
  destination pull dry-run.
- Existing-page overlap reviewed: the use-case page owns the evaluation
  question; this guide owns the executable transfer.

## Product truth

- Current capabilities used: local discovery, encrypted same-vendor push/pull,
  canonical project mapping, structural path tokens, restore backup, atomic
  write, and native vendor resume.
- Current limitations: native Windows and physical two-device acceptance remain
  open release gates; WSL2 is separate and WSL1 is unsupported.
- Version tested: source contract for Reinstate `v0.1.0`.
- Evidence: CLI/pathmap/adapter/sync tests, synthetic fixtures, compatibility
  data, and release notes.
- Claims that require verification: physical-device success, performance,
  unsupported path fields, and versions outside the compatibility matrix.
- Prohibited or roadmap claims: arbitrary prose rewriting, credential sync,
  silent Claude-to-Codex translation, or stable certification.

## Outline

- H1: How to move a coding-agent session from Mac to Windows
- Direct answer: map the same canonical project ID to each OS-native checkout,
  push one encrypted same-vendor session from macOS, dry-run the Windows
  restore, then resume it with the original vendor.
- Structure: prerequisites; key points; install; configure paths; discover;
  push; dry-run; pull; native resume; parameters; platform notes; failures;
  rollback; verification; FAQ; related resources.
- Original value: exposes the exact structural path boundary and treats native
  Windows and WSL2 as separate environments.

## Links and structured data

- Inbound opportunities: homepage, macOS/Windows use case, compatibility,
  integration pages, troubleshooting.
- Outbound internal links: getting started, configuration, adapters,
  compatibility, security, Claude Code integration, Codex integration.
- Primary type: `TechArticle`; additional type: visible-step `HowTo`.
- Breadcrumbs: Home → Guides → page.
- Parity note: every HowTo step name and description is visible at its anchor.

## Media and metadata

- Diagram: branded Mac → encrypted checkpoint → Windows path-mapping flow.
- Alt text: encrypted same-vendor session moving from a macOS checkout into a
  mapped Windows checkout.
- Title/description/social copy: route-specific and derived from the reviewed
  page metadata.
- Raw data: no physical-device success-rate or timing dataset is available.

## Acceptance criteria

- [x] One dominant platform-specific intent
- [x] Every command has purpose, output, parameters, platform context, failure,
  and safe recovery
- [x] Same-vendor and release-candidate boundaries remain visible
- [x] No real paths, credentials, or transcripts enter examples
- [x] Structured steps match visible content
- [x] Route-specific social card and local discoverability gates pass

