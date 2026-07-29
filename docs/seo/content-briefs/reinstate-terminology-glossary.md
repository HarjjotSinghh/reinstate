# Content brief: Reinstate terminology glossary

## Page

- Title: Reinstate Terminology: Session Sync and Continuity Glossary
- URL: `/glossary`
- Type: reference glossary
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending

## Audience and intent

- Audience: developers evaluating or operating Reinstate and contributors reading its architecture.
- Problem: words such as profile, snapshot, manifest, native resume, and portable handoff can be confused with vendor accounts, backups, or cross-agent translation.
- Primary query: `Reinstate terminology`
- Secondary questions: what is a Reinstate profile; what is a snapshot; what does native resume mean; what do compatibility states mean?
- Intent: answer and evaluation.
- Next action: follow the linked compatibility, format, path-mapping, or architecture source.

## Product truth

- Capabilities: Phase 1 profile-scoped encrypted same-vendor session sync, immutable snapshots, encrypted manifests, canonical project mapping, and fail-closed compatibility.
- Limitations: portable cross-agent handoffs are planned; a profile is not a hosted account; current format terminology is project-specific and pre-1.0.
- Evidence: `internal/schema`, `internal/sync`, `internal/pathmap`, `internal/adapter`, `docs/product-strategy.md`, and `ROADMAP.md`.
- Version: `v0.1.0-rc.8`.

## Outline

- H1: Reinstate terminology: session sync and continuity glossary
- Direct answer: define the vocabulary as current implementation terms and clearly label planned concepts.
- H2/H3: current sync terms; path and resume terms; compatibility states; current-versus-planned boundary; source links.

## Links

- Inbound: global footer, research hub, compatibility page, format specification, path visualizer.
- Outbound internal: `/about/reinstate`, `/compatibility`, `/research/encrypted-snapshot-format-v1`, `/tools/path-mapping-visualizer`, `/roadmap`.
- External sources: immutable Reinstate source links only where needed.

## Structured data

- Primary type: `WebPage`.
- Breadcrumb: Home → Glossary.
- Optional types: none; this is not FAQ or DefinedTermSet markup until a maintained schema contract is added.

## Media

- No screenshot required.
- Definition cards and explicit status labels provide the original visual structure.
- No private or user-derived data.

## Acceptance criteria

- Every required term is defined in visible initial HTML.
- `portable handoff` is explicitly planned rather than current.
- Compatibility states match adapter constants and the public compatibility page.
- The page has a unique branded social card, visible verification date, and primary-source links.
