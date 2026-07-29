# Reinstate product-truth register

Last reviewed: 2026-07-27  
Canonical website source: `website/src/data/product.ts`  
Reviewed release: `v0.1.0-rc.8`

This is the required output of the repository-local
`reinstate-product-truth` workflow. It separates released facts from roadmap
direction and records conflicts without converting missing evidence into a
marketing claim.

## Verified product facts

| Fact | Current wording | Primary evidence |
| --- | --- | --- |
| Product | Reinstate | Go module, CLI help, `product.ts` |
| Category | Open-source coding-agent session sync / continuity layer | released CLI scope, product strategy, `product.ts` |
| Primary outcome | Move a supported coding-agent session to another configured device and resume it with the same vendor | CLI e2e tests, adapter tests, getting started |
| Audience | Developers continuing coding-agent work across work/personal computers, desktop/laptop, projects, or environments | product strategy and published use cases |
| Current agents | Claude Code and Codex CLI | adapter registry, compatibility data, setup checks |
| Native-resume boundary | Claude Code → Claude Code and Codex → Codex only | adapter implementation, docs, protected claim tests |
| Cross-agent behavior | Explicit portable handoff is later roadmap work; no silent transcript translation | roadmap, universal-configuration docs, AGENTS.md |
| Current OS targets | macOS and native Windows; WSL2 is a separately documented open gate; plain Linux is not a certified Phase 1 resume target | device detection, compatibility data, limitations |
| Encryption | Supported session snapshots and manifests are encrypted locally before upload using the current age envelope implementation | `internal/crypto`, sync engine, threat model |
| Storage | User-owned Amazon S3, Cloudflare R2, or compatible S3 storage | backend/config implementation and storage docs |
| Credentials | Auth and credential files are excluded; storage secrets remain in private input/keyring channels and are not synchronized | exclusion policy, keyring implementation, security docs |
| Paths | Recognized structural project roots are tokenized and expanded through a canonical project ID; arbitrary prose is not rewritten | `internal/pathmap`, adapter tests, configuration docs |
| License | Apache-2.0 | `LICENSE`, `product.ts` |
| Account requirement | The CLI does not require a Reinstate account | released architecture and `product.ts` |
| Current release | `v0.1.0-rc.8`, pre-1.0 release candidate; no stable release yet | changelog, release history, compatibility data |
| Maintainer | Harjot Singh Rana | repository metadata and `product.ts` |

## Conflicting claims and resolution

| Location or surface | Conflict | Classification | Resolution |
| --- | --- | --- | --- |
| Live GitHub About description | “Sync and resume coding-agent work across every device” is broader than verified OS/acceptance scope | Unsupported as written | Owner-operated update remains required; exact replacement is in `launch-distribution.md`. |
| Live GitHub topics/social preview | Topics are incomplete and the generic social image does not match the canonical entity/brand packet | Stale external entity metadata | Reviewed topics and reproducible 1280×640 image are prepared; application remains an external repository-settings action. |
| Roadmap surfaces | Search, verified resume, portable handoffs, universal configuration, and team continuity can be mistaken for current features | Planned | Every current page labels these as planned and the product-claim scan rejects planned CLI syntax in current command examples. |
| OS language | Availability of a Linux binary can be mistaken for certified Phase 1 Linux agent resume | Ambiguous without qualification | Published install and guide copy says plain Linux is not a certified Phase 1 agent-resume target. |
| `rein doctor --self-test` | “Synthetic storage test” could be read as a probe of configured remote storage | Ambiguous | CLI reference now states that the self-test uses in-memory sync and local files; real storage evidence comes from `init`, `status`, or scoped sync operations. |
| Structured data | A schema type or claim could exceed visible current content | Unsupported if unmatched | Generated-build CI now requires visible parity for page/article names, FAQ questions, HowTo steps, breadcrumbs, and dates. |

## Unsupported claims removed or prohibited

- support for agents other than Claude Code and Codex as current Phase 1
  adapters;
- stable support for native Windows, macOS amd64, WSL2, plain Linux, or a
  physical two-device matrix without completed acceptance evidence;
- Claude Code ↔ Codex native transcript translation;
- credential, auth-token, or raw vendor-config-tree synchronization;
- generic `rein resume`, `rein search`, handoff, MCP, skill, plugin, or
  marketplace commands in RC8;
- customer counts, ratings, reviews, awards, market share, benchmarks,
  performance rates, restore-success rates, and productivity savings;
- formal security-audit or absolute-security guarantees;
- third-party endorsements or competitor capabilities without primary
  evidence; and
- fake freshness, fake schema reviews, or aggregate ratings.

These prohibitions are enforced across metadata/schema and current command
examples by `check-seo.mjs`, protected Vitest contracts, compatibility data,
and the central release-drift guard.

## Files and systems synchronized

- `website/src/data/product.ts`, `releases.ts`, and `compatibility.json`;
- shared layouts, schema builders, route-specific Open Graph generation, and
  analytics taxonomy;
- docs, guides, integration/use-case/comparison pages, FAQ, troubleshooting,
  research, roadmap, changelog, RSS, `llms.txt`, and compatibility JSON;
- README, product strategy, limitations, security, and launch/distribution
  packets; and
- portable Codex/Claude skills under `.agents/skills` and `.claude/skills`.

Static editorial evidence deliberately includes an explicit release label.
`product-truth.test.ts` fails when a current content-frontmatter release or RC
label drifts from `product.ts`; version-history and research sources are
separately allowed to preserve historical releases.

## Evidence used

- released Go command tree and `--help` output;
- unit, race, e2e, adapter, path-map, crypto, sync, installer, and
  compatibility tests;
- deterministic synthetic fixtures only—never real transcripts;
- `CHANGELOG.md`, release history, current compatibility matrix, roadmap,
  `PRODUCT.md`, `docs/product-strategy.md`, and AGENTS.md;
- generated HTML, JSON-LD, sitemap, robots, RSS, `llms.txt`, social-card, link,
  freshness, performance, and Lighthouse checks; and
- read-only production/GitHub observations recorded in the SEO operations
  artifacts.

## Unresolved questions and evidence gates

1. Native Windows RC8, macOS amd64, WSL2, and physical two-device acceptance
   need completed evidence before stable-support language.
2. No stable release exists; `v0.1.0-rc.8` remains the truthful public status.
3. Search Console, Bing, IndexNow production ownership, Plausible, WAF/log,
   field Core Web Vitals, and manual AI-query evidence require owner/account or
   deployed-production access.
4. The branch is not the current production deployment, so the live site
   cannot yet validate its new routes, metadata, feeds, schemas, or cards.
5. GitHub About/topics/social preview remain externally stale until the
   repository owner applies the prepared packet.
6. Named competitor pages, benchmarks, public format research, user reports,
   and directory submissions remain evidence-gated and must not be fabricated.

