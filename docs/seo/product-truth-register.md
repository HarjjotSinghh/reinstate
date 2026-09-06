# Reinstate product-truth register

Last reviewed: 2026-08-27
Canonical website source: `website/src/data/product.ts`
Stable release: `v0.5.1`, dated 2026-08-21.
Reviewed candidate: `v0.5.2-rc.1`, dated 2026-08-23; interactive-CLI
tagged-artifact acceptance is still pending for it.

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
| Environment preflight | Every same-vendor native continuation runs an environment preflight before the agent starts: it reports the environment it can actually observe, compares only facts with trustworthy recorded provenance, and refuses a silent bad continuation instead of guessing. Shipped in `v0.3.0`. | `internal/preflight`, executable-trust/workspace-identity/version checks on both adapter launch paths, `docs/verified-resume.md`, Phase 3 CLI tests |
| Cross-agent behavior | `v0.4.0` provides explicit structured handoff into a new Claude Code or Codex session; it does not translate or transfer a native session | handoff contract, ADR 0003, CLI/doctest contracts |
| Handoff source scope | Claude Code, Codex CLI, Gemini CLI, OpenCode, Grok Build, and Kimi Code CLI can be sources; Gemini, OpenCode, Grok, and Kimi are source-only | directional compatibility matrix, reader tests |
| Current OS targets | Apple Silicon macOS and native Windows x64 are mandatory RC/stable targets; Intel macOS and Linux/WSL2 are optional and unsupported/unverified | release runbook, compatibility data, limitations |
| Encryption | Supported session snapshots and manifests are encrypted locally before upload using the current age envelope implementation | `internal/crypto`, sync engine, threat model |
| Storage | User-owned Amazon S3, Cloudflare R2, or compatible S3 storage | backend/config implementation and storage docs |
| Credentials | Auth and credential files are excluded; storage secrets remain in private input/keyring channels and are not synchronized | exclusion policy, keyring implementation, security docs |
| Paths | Recognized structural project roots are tokenized and expanded through a canonical project ID; arbitrary prose is not rewritten | `internal/pathmap`, adapter tests, configuration docs |
| License | Apache-2.0 | `LICENSE`, `product.ts` |
| Account requirement | The CLI does not require a Reinstate account | released architecture and `product.ts` |
| Current release | Stable is `v0.5.1`, dated 2026-08-21. `v0.5.2-rc.1`, dated 2026-08-23, is a candidate awaiting interactive-CLI tagged-artifact acceptance. | `website/src/data/product.ts`, `CHANGELOG.md` section `[0.5.1]`, release history |
| Maintainer | Harjot Singh Rana | repository metadata and `product.ts` |

## Conflicting claims and resolution

| Location or surface | Conflict | Classification | Resolution |
| --- | --- | --- | --- |
| Live GitHub About description | Previously “Sync and resume coding-agent work across every device”, which was broader than verified OS/acceptance scope | Resolved 2026-08-27 | The live About now reads “Find, resume, and hand off Claude Code and Codex sessions across machines — encrypted, your own bucket.” Verified against the GitHub API. It names both supported agents, keeps hand-off distinct from resume, and makes no OS claim, so it is within scope. |
| Live GitHub topics | Previously incomplete | Resolved 2026-08-27 | Eleven topics are now applied and verified against the GitHub API: `agents`, `claude-code`, `codex`, `context`, `state`, `cli`, `developer-tools`, `encryption`, `go`, `golang`, `session-management`. |
| Live GitHub social preview image | The generic social image may still not match the canonical entity/brand packet | Unverified | The reproducible 1280×640 image is prepared. Whether it has been applied was not checked — the GitHub API does not expose it in the fields queried on 2026-08-27. Confirm in repository settings before treating this as done. |
| Candidate surfaces | Phase 4 structured handoff is stable in `v0.4.0` | Verified | Docs must not describe handoff as pending candidate work. |
| Roadmap surfaces | Universal configuration and team continuity can be mistaken for current features | Planned | Current pages separate stable `v0.5.1` from later roadmap work. |
| OS language | Availability of a Linux binary can be mistaken for certified Phase 1 Linux agent resume | Ambiguous without qualification | Published install and guide copy says plain Linux is not a certified Phase 1 agent-resume target. |
| `rein doctor --self-test` | “Synthetic storage test” could be read as a probe of configured remote storage | Ambiguous | CLI reference now states that the self-test uses in-memory sync and local files; real storage evidence comes from `init`, `status`, or scoped sync operations. |
| Structured data | A schema type or claim could exceed visible current content | Unsupported if unmatched | Generated-build CI now requires visible parity for page/article names, FAQ questions, HowTo steps, breadcrumbs, and dates. |

## Unsupported claims removed or prohibited

- support for agents other than Claude Code and Codex as current Phase 1
  adapters;
- stable `v0.5.2` support without a separate reviewed promotion and fresh
  Apple Silicon macOS/native Windows tagged-artifact evidence;
- stable support for Intel macOS, WSL2, or plain Linux without completed
  acceptance evidence;
- Claude Code ↔ Codex native transcript translation, native cross-agent
  resume, the same native session, lossless transfer, or full-context claims;
- Gemini CLI, OpenCode, or Grok Build as `v0.4.0` handoff destinations;
- credential, auth-token, or raw vendor-config-tree synchronization;
- MCP, skill, plugin, marketplace, or agent-runtime commands in Reinstate;
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

1. Intel macOS and Linux/WSL2 remain optional unverified environments
   ([#97](https://github.com/HarjjotSinghh/reinstate/issues/97),
   [#98](https://github.com/HarjjotSinghh/reinstate/issues/98)).
2. Homebrew, Scoop, and npm may still list `0.3.0` until package-publish
   promotes this tag; GitHub Releases and `reinstate.dev` installers are the pin.
3. Search Console, Bing, IndexNow production ownership, Plausible, WAF/log,
   field Core Web Vitals, and manual AI-query evidence require owner/account or
   deployed-production access.
4. The branch is not the current production deployment, so the live site
   cannot yet validate its new routes, metadata, feeds, schemas, or cards.
5. GitHub About/topics/social preview remain externally stale until the
   repository owner applies the prepared packet.
6. Named competitor pages, benchmarks, public format research, user reports,
   and directory submissions remain evidence-gated and must not be fabricated.
