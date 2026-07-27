# Reinstate answer inventory

Last reviewed: **2026-07-27**  
Product version: **v0.1.0-rc.6**

This inventory gives every high-intent product question one canonical answer
location. Supporting pages may repeat a shorter contextual answer, but they
must link to the canonical page and preserve the same release limitations.

Status meanings:

- **Implemented** — a visible direct answer exists in the generated branch.
- **Evidence-gated** — the page answers the question but explicitly withholds
  a conclusion that needs physical, production, or external evidence.
- **Planned query test** — the page exists locally; citation/search testing
  waits for deployment and provider access.

## Product

| ID | Question | Canonical answer | Status |
| --- | --- | --- | --- |
| P01 | What is Reinstate? | `/about/reinstate` | Implemented |
| P02 | What problem does Reinstate solve? | `/about/reinstate` | Implemented |
| P03 | Is Reinstate open source? | `/open-source` | Implemented |
| P04 | Is Reinstate free? | `/docs/faq` | Implemented |
| P05 | Does Reinstate require an account? | `/docs/faq` | Implemented |
| P06 | Does Reinstate run its own storage service? | `/docs/faq` | Implemented |
| P07 | Which coding agents does Reinstate support? | `/compatibility` | Implemented |
| P08 | Which operating systems does Reinstate support? | `/compatibility` | Evidence-gated |

## Workflow

| ID | Question | Canonical answer | Status |
| --- | --- | --- | --- |
| W01 | How do I sync a Claude Code session across devices? | `/guides/sync-claude-code-sessions-across-devices` | Evidence-gated |
| W02 | How do I move a Codex session to another computer? | `/guides/sync-codex-sessions-across-devices` | Evidence-gated |
| W03 | How do I continue an AI coding session on a laptop? | `/use-cases/work-and-personal-computers` | Implemented |
| W04 | Can I move a session from macOS to Windows? | `/use-cases/macos-and-windows` | Evidence-gated |
| W05 | What happens when project paths differ? | `/docs/architecture` | Implemented |
| W06 | Can I restore without copying the whole repository? | `/docs/faq` | Implemented |
| W07 | Does Reinstate sync Git changes? | `/compare/reinstate-vs-git` | Implemented |
| W08 | Does Reinstate sync MCP servers or credentials? | `/docs/faq` | Implemented |

## Security

| ID | Question | Canonical answer | Status |
| --- | --- | --- | --- |
| S01 | Where is session data encrypted? | `/security` | Implemented |
| S02 | Who can read a Reinstate archive? | `/security` | Implemented |
| S03 | Where is encrypted data stored? | `/security` | Implemented |
| S04 | Does Reinstate upload API keys? | `/security` | Implemented |
| S05 | What happens if an S3 bucket is compromised? | `/docs/security-model` | Implemented |
| S06 | What metadata remains visible? | `/docs/security-model` | Implemented |
| S07 | How are encryption keys managed? | `/docs/security-model` | Implemented |
| S08 | What is outside the threat model? | `/docs/security-model` | Implemented |

## Technical

| ID | Question | Canonical answer | Status |
| --- | --- | --- | --- |
| T01 | Where does Claude Code store sessions? | `/docs/adapters` | Implemented |
| T02 | Where does Codex store sessions? | `/docs/adapters` | Implemented |
| T03 | What does a Reinstate adapter do? | `/docs/adapters` | Implemented |
| T04 | How does path remapping work? | `/docs/architecture` | Implemented |
| T05 | What happens after an agent changes its session format? | `/docs/faq` | Implemented |
| T06 | How is compatibility tested? | `/compatibility` | Evidence-gated |
| T07 | What files are included or excluded? | `/docs/adapters` | Implemented |

## Evaluation

| ID | Question | Canonical answer | Status |
| --- | --- | --- | --- |
| E01 | Is Reinstate a remote desktop? | `/compare/reinstate-vs-remote-desktop` | Implemented |
| E02 | Is Reinstate a cloud IDE? | `/docs/faq` | Implemented |
| E03 | Is Reinstate a backup tool? | `/docs/faq` | Implemented |
| E04 | How is Reinstate different from Git? | `/compare/reinstate-vs-git` | Implemented |
| E05 | How is Reinstate different from manually copying session files? | `/compare/reinstate-vs-manual-session-copying` | Implemented |
| E06 | When should I not use Reinstate? | `/about/reinstate` | Implemented |

## Verification routine

For every release or monthly audit:

1. Render each canonical route and confirm the question or an unambiguous
   equivalent is visible as a heading or direct-answer block.
2. Confirm the extracted answer remains correct without surrounding marketing.
3. Compare version, platform, agent, storage, encryption, and same-vendor
   claims with the central product source and release evidence.
4. Check internal links from the supporting integration, use-case, security,
   and comparison pages.
5. After deployment, run the fixed provider query set in
   `ai-search-query-baseline.md`; do not infer citation success from local HTML.
6. Record incorrect or stale answers as content defects, not ranking problems.

