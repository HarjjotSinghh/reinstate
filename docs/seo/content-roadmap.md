# Discoverability content roadmap

This register turns the playbook's first 30 opportunities into an evidence-aware
backlog. `Implemented` means a canonical answer exists in this branch;
`Covered` means the intent is answered inside a broader canonical page;
`Planned` means publication would currently require new product evidence or a
dedicated editorial pass.

## P0 launch content

| # | Opportunity | Canonical/status |
| ---: | --- | --- |
| 1 | What is Reinstate? | Implemented: `/about/reinstate` |
| 2 | Sync Claude Code sessions across devices | Implemented: `/guides/sync-claude-code-sessions-across-devices` |
| 3 | Sync Codex sessions across devices | Implemented: `/guides/sync-codex-sessions-across-devices` |
| 4 | Move a coding-agent session from macOS to Windows | Implemented: `/guides/move-a-coding-agent-session-from-mac-to-windows` |
| 5 | Continue work between a work and personal computer | Implemented: `/use-cases/work-and-personal-computers` |
| 6 | Reinstate security model | Implemented: `/security` and `/docs/security-model` |
| 7 | Supported agents, OSs, and storage | Implemented: `/compatibility` and `/compatibility.json` |
| 8 | Reinstate installation guide | Implemented: `/docs/installation` |
| 9 | Upload and restore a session | Implemented: `/docs/sync-a-session` and `/docs/restore-a-session` |
| 10 | Troubleshooting guide | Implemented: `/docs/troubleshooting` |
| 11 | Reinstate FAQ | Implemented: `/docs/faq` |
| 12 | Reinstate architecture | Implemented: `/docs/architecture` |

## P1 demand capture

| # | Opportunity | Canonical/status |
| ---: | --- | --- |
| 13 | Where Claude Code stores sessions | Covered by `/integrations/claude-code` and `/docs/adapters`; dedicated research/reference page planned |
| 14 | Where Codex stores sessions | Covered by `/integrations/codex` and `/docs/adapters`; dedicated research/reference page planned |
| 15 | Why Git does not sync coding-agent conversations | Implemented: `/blog/why-git-does-not-sync-coding-agent-sessions` |
| 16 | Reinstate versus manual session copying | Implemented: `/compare/reinstate-vs-manual-session-copying` |
| 17 | Reinstate versus remote desktop | Implemented: `/compare/reinstate-vs-remote-desktop` |
| 18 | Use Cloudflare R2 for encrypted storage | Implemented: `/guides/use-cloudflare-r2-for-coding-agent-session-storage` |
| 19 | Use Amazon S3 for encrypted storage | Implemented: `/guides/use-s3-for-coding-agent-session-storage` |
| 20 | How path remapping works between macOS and Windows | Implemented with a fixed synthetic explainer at `/tools/path-mapping-visualizer`, supported by `/use-cases/macos-and-windows` and `/docs/architecture` |
| 21 | What Reinstate does not sync | Implemented: `/docs/limitations` |
| 22 | Verify a restored coding-agent session | Implemented inside both agent-specific guides |
| 23 | Handle agent format and supported-version changes | Implemented through `/compatibility/agent-version-history`, `/compatibility`, `/docs/adapters`, and troubleshooting |
| 24 | Build a Reinstate adapter | Partly covered by `/docs/adapters`; publish a full tutorial only with a supported extension contract |

## P2 authority and shareability

| # | Opportunity | Status and evidence gate |
| ---: | --- | --- |
| 25 | Anatomy of a coding-agent session | Partly implemented: `/research/encrypted-snapshot-format-v1` documents Reinstate's current encrypted wrapper; vendor-format research remains gated |
| 26 | Cross-device portability test methodology | Implemented at `/research` with the source acceptance runbook; completed RC6 results remain release-gated |
| 27 | Agent-session compatibility report | Implemented: `/compatibility`, `/compatibility.json`, and `/research`; periodic field reports require a completed evidence window |
| 28 | Threat-modeling encrypted session sync | Planned: formal security review and explicit scope |
| 29 | Lessons from implementing local-first encryption | Planned: maintainer-authored engineering evidence |
| 30 | Open session portability as infrastructure | Planned: roadmap/standards essay clearly separated from current native resume |

## Editorial rules

- Do not create a thin new URL when the current canonical already satisfies the
  intent.
- Do not publish P2 evidence pages just to satisfy a calendar.
- Do not update review dates without rerunning commands, checking sources, or
  performing the stated editorial review.
- Every new brief starts from
  [`content-briefs/template.md`](content-briefs/template.md) and maps its
  questions to [`question-inventory.md`](question-inventory.md).
- A published research asset needs methodology, exact environment, failures,
  limitations, raw synthetic data where appropriate, date, and commit SHA.

## Route publication gates

Three planned URL families remain deliberately absent rather than shipping thin
or unsupported pages:

- **Blog tag archives:** create `/blog/tags/[tag]` only after a tag has at least
  three substantive, indexable articles and the archive can provide a unique
  answer-first introduction. Until then, the blog hub is the canonical
  discovery page; do not generate empty or one-post tag pages.
- **Named competitor comparisons:** create
  `/compare/reinstate-vs-[real-competitor]` only after a dated editorial brief
  identifies real query/user demand and every comparison fact has a current
  first-party source. A comparison must distinguish product category and
  limitations, not turn a competitor name into a doorway page.
- **Binding terms:** publish `/terms` only after the maintainer or legal owner
  approves the offering, jurisdiction, effective date, and complete policy
  text under the existing legal-publication gate. Do not fabricate legal copy
  to fill the information architecture.

Record the approving brief, owner, review date, sources, and canonical decision
before introducing any of these routes.
