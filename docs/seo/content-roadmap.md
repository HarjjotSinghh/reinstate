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
| 4 | Move a coding-agent session from macOS to Windows | Implementation in progress: `/guides/move-a-coding-agent-session-from-mac-to-windows` |
| 5 | Continue work between a work and personal computer | Implemented: `/use-cases/work-and-personal-computers` |
| 6 | Reinstate security model | Implemented: `/security` and `/docs/security-model` |
| 7 | Supported agents, OSs, and storage | Implemented: `/compatibility` and `/compatibility.json` |
| 8 | Reinstate installation guide | Covered by `/docs/getting-started`; dedicated `/docs/installation` in progress |
| 9 | Upload and restore a session | Implemented in agent guides; dedicated operational docs in progress |
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
| 20 | How path remapping works between macOS and Windows | Covered by `/use-cases/macos-and-windows` and `/docs/architecture`; dedicated deep dive planned |
| 21 | What Reinstate does not sync | Dedicated `/docs/limitations` in progress; also covered by FAQ/security |
| 22 | Verify a restored coding-agent session | Implemented inside both agent-specific guides |
| 23 | Handle agent format changes | Covered by `/compatibility`, `/docs/adapters`, and troubleshooting |
| 24 | Build a Reinstate adapter | Partly covered by `/docs/adapters`; publish a full tutorial only with a supported extension contract |

## P2 authority and shareability

| # | Opportunity | Status and evidence gate |
| ---: | --- | --- |
| 25 | Anatomy of a coding-agent session | Planned: sanitized format research and stable terminology |
| 26 | Cross-device portability test methodology | Source acceptance checklist exists; public research treatment planned |
| 27 | Agent-session compatibility report | Machine-readable current matrix implemented; periodic report needs a completed evidence window |
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

