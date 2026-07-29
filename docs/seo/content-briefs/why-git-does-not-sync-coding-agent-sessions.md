# Content brief: why Git does not sync coding-agent sessions

## Page

- Proposed title: Why Git Does Not Sync Coding-Agent Sessions
- URL: `/blog/why-git-does-not-sync-coding-agent-sessions`
- Page type: technical explainer
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0-rc.8`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers who expect a Git clone or pull to reproduce an
  in-progress Claude Code or Codex task.
- Primary problem: source history and vendor-native agent session state have
  different ownership, identity, storage, and portability contracts.
- Primary query: `why Git does not sync coding agent sessions`
- Secondary questions: what a session contains; whether Reinstate replaces Git;
  how paths and encryption differ; when manual copying fails.
- Search intent: problem/education
- Expected next action: keep Git as source truth and evaluate a scoped,
  same-vendor session dry-run separately.
- Existing-page overlap reviewed: comparison pages own explicit purchasing
  dimensions; this article owns the underlying technical model.

## Product truth

- Current capabilities used: vendor adapters, structural path mapping, local
  encryption, immutable sync snapshots, and same-vendor restore.
- Current limitations: Reinstate does not transfer the repository, commits,
  branches, dependencies, credentials, or cross-vendor transcripts.
- Version tested: implementation contract for Reinstate `v0.1.0-rc.8`.
- Evidence: `internal/adapter`, `internal/pathmap`, `internal/crypto`,
  `internal/sync`, deterministic fixtures, and documented Git boundary.
- Claims that require verification: third-party vendor internals beyond
  committed compatibility evidence and any measured productivity effect.
- Prohibited claims: Git replacement, universal agent support, transparent
  cross-agent resume, or quantified time savings.

## Outline

- H1: Why Git does not sync coding-agent sessions
- Direct answer: Git moves source snapshots and history, while a coding-agent
  session is vendor-native task state stored outside the repository; both are
  required for continuity but solve different problems.
- Structure: definitions; Git's boundary; session contents; path identity;
  encryption/storage; Reinstate's role; limitations; decision checklist; FAQ.
- Original value: grounds the distinction in Reinstate's actual adapter,
  path-map, crypto, and sync boundaries rather than a generic analogy.

## Links, schema, and media

- Inbound opportunities: homepage problem section, FAQ, comparison hub,
  getting started.
- Outbound internal links: Git comparison, architecture, adapters, security,
  Claude Code integration, Codex integration, limitations.
- Primary type: `BlogPosting`; breadcrumbs: Home → Blog → page.
- Diagram: Git source lane beside encrypted vendor-session lane.
- Alt text: source commits and vendor-native coding-agent session state moving
  through separate continuity paths.
- Raw data: no productivity or restore-rate metric is available.

## Acceptance criteria

- [x] Git remains source truth and Reinstate remains a continuity layer
- [x] Definitions distinguish repository state from vendor session state
- [x] Consequential claims link to implementation evidence
- [x] No fabricated vendor or performance claim
- [x] Blog schema and visible category/date/author remain aligned
- [x] Route-specific social card and local discoverability gates pass

