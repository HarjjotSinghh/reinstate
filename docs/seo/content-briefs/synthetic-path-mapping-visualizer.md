# Content brief: synthetic path-mapping visualizer

## Page

- Title: macOS ↔ Windows Coding-Agent Path-Mapping Visualizer
- URL: `/tools/path-mapping-visualizer`
- Type: interactive technical explainer
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending

## Audience and intent

- Audience: developers moving Claude Code or Codex work between macOS and Windows.
- Problem: absolute local paths and agent-native layout keys differ across devices.
- Primary query: `coding agent path mapping macOS Windows`
- Secondary questions: what is a canonical project ID; which fields change; does Reinstate rewrite transcript prose; is tool input stored?
- Intent: platform-specific and answer.
- Next action: inspect a fixed synthetic direction and then read the operational cross-platform guide.

## Product truth

- Capabilities: configured canonical project IDs, `${REPO:<id>}` normalization, destination expansion, Claude project-directory recomputation, and Codex `session_meta.cwd` rewriting.
- Limitations: only recognized structural paths change; unknown fields and free-form prose are not rewritten; physical RC7 acceptance gates remain open.
- Evidence: `internal/pathmap/token.go`, Claude and Codex adapter implementations and tests, and synthetic fixtures.
- Version: `v0.1.0-rc.7`.

## Outline

- H1: macOS ↔ Windows coding-agent path-mapping visualizer
- Direct answer: show how one fixed synthetic project becomes a portable token and expands through the destination mapping.
- H2/H3: interactive synthetic trace; recognized fields; what remains unchanged; operational limits; source evidence.

## Links

- Inbound: footer, compatibility page, macOS/Windows use case, research hub.
- Outbound internal: `/use-cases/macos-and-windows`, `/compatibility`, `/glossary`, `/docs/architecture`.
- External sources: immutable pathmap and adapter source links.

## Structured data

- Primary type: `WebPage`.
- Breadcrumb: Home → Path-mapping visualizer.
- Optional types: none; no HowTo because the control is an explainer rather than an operational procedure.

## Media

- Interactive diagram uses only fixed synthetic paths and buttons/selects.
- Initial HTML contains a complete macOS-to-Windows example when JavaScript is disabled.
- Selections are not persisted, transmitted, or included in analytics; the page disables site analytics.

## Acceptance criteria

- No free-text, file, clipboard-read, upload, or network input.
- No visualizer selection or path value is written to cookies, local storage, session storage, analytics, or the network.
- Only allow-listed synthetic direction and agent options can update the trace.
- Both agents explain exactly which structural locator changes.
- Unknown fields and transcript prose are visibly marked unchanged.
- The page and its route-specific social card pass build, SEO, and link checks.
