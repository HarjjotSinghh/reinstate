# Content brief: Reinstate CLI command reference

## Page

- Proposed title: Reinstate CLI Command Reference
- URL: `/docs/cli-reference`
- Page type: command reference
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers and coding agents that need current Reinstate syntax
  plus the safety contract around each command.
- Primary problem: terse `--help` output does not contain expected evidence,
  platform behavior, failure categories, or safe recovery.
- Primary query: `Reinstate CLI commands`
- Secondary questions: aliases; exit codes; flags; JSON output; passphrase
  input; conflicts; shell completion; whether undo exists.
- Search intent: navigational/reference
- Expected next action: select the exact command and inspect a dry-run or
  read-only preflight before any mutation.
- Existing-page overlap reviewed: getting started owns the happy path; the
  operational docs own deep workflows; this page owns exhaustive command
  lookup.

## Product truth

- Current capabilities used: only commands registered by the Reinstate Cobra command
  tree and verified against local `rein ... --help` output.
- Current limitations: no general undo, search, generic resume, handoff,
  universal configuration, MCP, skill, plugin, or marketplace command exists.
- Version tested: `v0.1.0`.
- Evidence: live local command help, `internal/cli`, CLI tests, e2e tests,
  configuration/storage contracts, and exit-code definitions.
- Claims that require verification: public installer/deployment state and
  physical multi-device success.
- Prohibited claims: planned syntax as current, secret CLI flags, cross-vendor
  native resume, or stable-release certification.

## Outline

- H1: Reinstate CLI command reference
- Direct answer: `rein` and `reinstate` are identical; the page documents every
  current command's purpose, syntax, expected result, parameters, platform
  notes, failures, and safe recovery.
- Structure: prerequisites; global rules; exit codes; inspect; configure;
  discover; transfer; conflicts; completion; evidence; failures; security.
- Original value: turns every terse help entry into a complete AEO command
  contract without inventing a command.

## Links, schema, and media

- Inbound opportunities: docs navigation, getting started, installation,
  configuration, troubleshooting.
- Outbound internal links: all operational docs, compatibility, limitations.
- Primary type: `TechArticle`; breadcrumbs: Home → Docs → page.
- Media: route-specific branded Open Graph card; no screenshot is required
  because source-checked text is more accessible and maintainable.
- Visible/schema parity: title, date, owner, breadcrumb, and article headline
  are rendered from the same page metadata.

## Acceptance criteria

- [x] Every shipped command and subcommand is represented
- [x] Purpose, expected result, parameters, platform differences, failures,
  and undo/recovery appear for each command
- [x] Current help output and implementation source were checked
- [x] Planned command surfaces remain clearly excluded
- [x] No secret-bearing example is present
- [x] Route-specific social card and local discoverability gates pass

