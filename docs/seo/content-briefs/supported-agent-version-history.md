# Content brief: supported agent-version history

## Page

- Title: Claude Code and Codex Version Support History
- URL: `/compatibility/agent-version-history`
- Type: compatibility change tracker
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending

## Audience and intent

- Audience: developers deciding whether to upgrade an agent or Reinstate release.
- Problem: the current compatibility matrix does not by itself show when tested ranges or fail-closed behavior changed.
- Primary query: `Reinstate supported Claude Code Codex versions`
- Secondary questions: when were inclusive ranges introduced; did RC4–Reinstate change the range; what happens outside it; does source compatibility equal physical certification?
- Intent: freshness and evaluation.
- Next action: run `rein setup check` and inspect the linked current compatibility evidence.

## Product truth

- Capabilities: Claude Code `2.1.219`–`2.1.220` and Codex CLI `0.133.0`–`0.146.0` are the current inclusive source-tested stable ranges; out-of-range or prerelease versions fail closed as untested.
- Limitations: the changelog does not publish exact historical ranges for RC1–RC2; current source tests do not close every physical platform gate.
- Evidence: `website/src/data/compatibility.json`, adapter constants/tests, `CHANGELOG.md`, and release history.
- Version: current release candidate `v0.2.0-rc.3`; stable release `v0.1.0`.

## Outline

- H1: Claude Code and Codex version support history
- Direct answer: state current ranges, the RC3 introduction, and the absence of later range changes.
- H2/H3: current matrix; compatibility state behavior; release-by-release tracker; evidence rules; upgrade checklist.

## Links

- Inbound: compatibility page, integrations hub, changelog, footer.
- Outbound internal: `/compatibility`, `/changelog`, `/integrations/claude-code`, `/integrations/codex`, `/docs/troubleshooting`.
- External sources: immutable adapter test evidence and tagged changelogs.

## Structured data

- Primary type: `TechArticle`.
- Breadcrumb: Home → Compatibility → Agent-version history.
- Optional types: none.

## Media

- Unified current-range table and release timeline.
- No trend graph; two current ranges and six evidence events are clearer as tables.
- Dates and status labels are visible.

## Acceptance criteria

- Current ranges are rendered from `compatibility.json`, not retyped in page markup.
- Timeline order and release dates derive from the repository release history.
- RC1–RC2 uncertainty is explicit.
- Every release row links to tagged evidence.
- No out-of-range vendor version is called supported.
- The page has a unique social card and passes freshness, SEO, and internal-link checks.
