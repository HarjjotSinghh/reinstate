# SEO, AEO, and AI-search implementation evidence

**Audit date:** 2026-07-27  
**Branch:** `feat/seo-aeo-aseo-foundation`  
**Worktree:** `/Users/harjjotsinghh/Documents/Projects/reinstate-seo`

This matrix accounts for every numbered section in the supplied playbook.
It separates code and editorial implementation from actions that require a
deployed commit, account ownership, real user evidence, or a later product
phase. It deliberately does not call those external actions complete.

## Status vocabulary

- **Implemented** — present in the branch with repository or generated-site
  evidence.
- **Operationalized** — a repeatable policy, template, or safe tool exists;
  execution depends on a release or external operator.
- **Release-gated** — intentionally not published or claimed until product
  evidence exists.
- **External** — requires an account, DNS, provider UI, production logs, a live
  deployment, or third-party action.
- **Reference** — strategic guidance that governs the implementation rather
  than producing a separate artifact.

## Sections 1–8: strategy, product truth, and architecture

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 1 | Executive summary and priorities | Implemented | this matrix, `docs/seo/README.md`, commit history |
| 2 | SEO/AEO/ASEO definitions and limits | Reference | vendored `docs/seo/seo-aeo-aseo-playbook.md`; AI-search runbook rejects special-tag mythology |
| 3 | Product positioning | Implemented | `website/src/data/product.ts`, homepage, `/about/reinstate`, README truth sync |
| 4 | Search-intent model | Implemented | content schemas require `targetQuery` and `searchIntent`; question inventory and content roadmap |
| 5 | Initial strengths audit | Implemented | preserved shared layout/SSR strengths and documented baselines |
| 6 | Initial gaps and risks | Implemented | technical foundation plus later audit commits; unresolved external items listed below |
| 7 | URL and information architecture | Implemented / staged | shallow hubs, docs, guides, integrations, use cases, comparisons; future evidence pages remain gated rather than thin |
| 8 | Page-to-query map | Implemented | `question-inventory.md`, `content-roadmap.md`, explicit frontmatter query targets |

## Sections 9–29: technical SEO for Astro

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 9 | Sitemap | Implemented | `@astrojs/sitemap`; generated sitemap parity gate |
| 10 | `robots.txt` and crawler policy | Implemented | static policy with separate GPTBot decision, OAI-SearchBot and PerplexityBot rules |
| 11 | Shared SEO component | Implemented | `SeoHead.astro` with unique metadata, canonicals, robots, OG, Twitter, article fields |
| 12 | Base layout integration | Implemented | `BaseLayout.astro` owns head, JSON-LD, verification hooks, fonts, analytics |
| 13 | Safe JSON-LD | Implemented | breakout-safe serializer, semantic contracts, URL/date/visibility tests |
| 14 | Homepage schema | Implemented | Person, WebSite, SoftwareApplication, SoftwareSourceCode, free Offer |
| 15 | Documentation schema | Implemented | WebPage/TechArticle, breadcrumbs, visible authorship/status/dates; FAQ extraction |
| 16 | Blog schema | Implemented | BlogPosting plus WebPage/breadcrumb entities and editorial dates |
| 17 | Content collections | Implemented | validated docs/guides/blog metadata, dates, intent, related links, HowTo steps, safe canonical/social overrides |
| 18 | Canonicals and redirects | Implemented | no-trailing-slash policy, generated canonical gate, permanent `/docs/overview` redirect |
| 19 | Status codes and error page | Implemented | branded 404, preview noindex, local preview 404 tests, production smoke tooling |
| 20 | Internal linking | Implemented | contextual homepage/integration/guide links; orphan and depth ≤3 gate (currently ≤2) |
| 21 | Page titles | Implemented | unique/count/length generated-site checks |
| 22 | Meta descriptions | Implemented | required content fields and generated-site uniqueness/count checks |
| 23 | Heading structure | Implemented | single-H1 and ordered heading validation plus rendered-browser audits |
| 24 | Images and media | Implemented | responsive WebP docs media, intrinsic sizes, lazy loading, alt/caption checks, 1200×630 route cards |
| 25 | Core Web Vitals/performance | Implemented / external | static budgets and real Lighthouse lab gate; field CWV/INP require production data |
| 26 | Accessibility | Implemented / manual follow-up | skip link, semantics, reduced motion, corrected contrast/underlines, Lighthouse accessibility 100 on five templates; screen-reader/zoom testing remains manual |
| 27 | JavaScript/rendering | Implemented | core content prerendered in HTML; static preview and Lighthouse inspect rendered output |
| 28 | RSS | Implemented | indexable guides/blog feed with canonical URLs and metadata |
| 29 | `llms.txt` | Implemented | optional factual resource; documentation explicitly states it is not a ranking control |

## Sections 30–37: answer engine optimization

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 30 | Answer-first model | Implemented | direct-answer hero blocks across public/editorial layouts |
| 31 | AEO page template | Implemented | guide schema/layout contracts and complete outcome guides |
| 32 | Question inventory | Implemented | 37-question owner/status map in `question-inventory.md` |
| 33 | Definition blocks | Implemented | about, FAQ, integrations, architecture, security, use cases |
| 34 | How-to content | Implemented | visible prerequisites/steps/outputs/errors/rollback/verification and matching HowTo graph |
| 35 | Comparison content | Implemented | sourced, dated workflow comparisons with explicit dimensions and limitations |
| 36 | Troubleshooting content | Implemented | eight entries, each with the required eight-part diagnostic contract |
| 37 | FAQ strategy | Implemented | visible FAQ inventory and exact schema extraction; no FAQ rich-result claim |

## Sections 38–47: AI search optimization

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 38 | ASEO model | Reference / implemented | core-index-first policy across AI-search skill and runbook |
| 39 | Entity consistency | Implemented | centralized product data, protected claim scans, fact page, README/schema sync |
| 40 | Authoritative fact page | Implemented | `/about/reinstate` includes release, scope, dates, license, owner, limitations, links |
| 41 | Primary-source evidence | Implemented / release-gated | architecture, security, compatibility JSON, tests/results links; benchmarks/research require reproducible new evidence |
| 42 | Citation-ready claims | Implemented | direct factual answers, source-linked comparison/compatibility pages, claim guards |
| 43 | Freshness system | Implemented | changelog/RSS/dates, sourced compatibility data, automatic 60/120-day audit |
| 44 | Crawler accessibility | Implemented / external | crawler policy and production smoke tool; verified CDN/WAF logs require production access |
| 45 | Third-party corroboration | Operationalized / external | release distribution plan and GitHub metadata proposal; independent mentions cannot be created honestly in code |
| 46 | AI-query testing | Operationalized / external | fixed query set, grading rubric, evidence template; provider runs must be manual |
| 47 | AI referrals | Implemented / external | privacy-safe Plausible hooks and taxonomy; production property/data require owner configuration |

## Sections 48–55: content, authority, and distribution

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 48 | Content pillars | Implemented | hubs and `content-roadmap.md` organize continuity, agents, security, architecture, open source |
| 49 | First 30 opportunities | Implemented / staged | item-by-item canonical and evidence-gate register in `content-roadmap.md` |
| 50 | Content brief template | Implemented | executable template plus Claude Code and Codex briefs |
| 51 | Editorial quality | Implemented | schemas, contract tests, freshness/source/product-truth gates |
| 52 | Programmatic SEO policy | Reference | no mass/thin generation; every route must pass uniqueness, value, link, and truth gates |
| 53 | Launch distribution | Operationalized / external | `launch-distribution.md`; actual posts/submissions wait for deployment/release authorization |
| 54 | Digital PR angles | Operationalized | evidence-backed angles and prohibited generic/unsupported positioning |
| 55 | Linkable assets | Implemented / staged | compatibility JSON is live-ready; research/tools have explicit publication gates |

## Sections 56–65: measurement and phases

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 56 | Required tools | Implemented / external | CI, Lighthouse, crawl/schema/link/media/freshness checks; console and analytics accounts external |
| 57 | Search Console | Operationalized / external | verification hooks, exact setup/inspection/evidence runbook |
| 58 | Bing and IndexNow | Implemented / external | Bing hook/runbook; reviewed delta plans, dynamic proof, retries, soft-fail submitter; production key/submission external |
| 59 | Event taxonomy | Implemented | all 12 controlled events, exact-route/link rules, privacy notice, tests |
| 60 | KPI framework | Operationalized | monthly audit has SEO/AEO/ASEO/product guardrails and numerator/denominator rules |
| 61 | Reporting cadence | Operationalized | weekly launch, monthly audit, quarterly review subjects |
| 62 | Pre-public Phase 2 work | Implemented / external | repository P0 technical/content complete; consoles, field data, WAF/logs await deployed access |
| 63 | Public trial launch | Release-gated / external | guides/tooling ready; release, videos, posts, directories, daily monitoring wait for usable public trial and owner action |
| 64 | Authority building | Release-gated | no fabricated research, benchmarks, testimonials, or compatibility reports |
| 65 | Ninety-day calendar | Operationalized | content roadmap preserves priorities but evidence outranks calendar timing |

## Sections 66–72: skills and coding-agent workflows

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 66 | Skill usage/install pattern | Implemented | `.agents/skills`, `.claude/skills`, READMEs, drift/frontmatter tests |
| 67 | Third-party skills | Audited | exact upstream commits and renamed skills documented; unsafe/outdated packs intentionally not installed |
| 68 | Custom Reinstate skill pack | Implemented | all nine portable skills installed and synchronized |
| 69 | Technical implementation prompt | Reference | retained in vendored playbook; implemented by this branch |
| 70 | New guide prompt | Reference / implemented | used for complete session and storage guides |
| 71 | Release discoverability prompt | Operationalized | release skill, launch runbook, IndexNow, production smoke process |
| 72 | Monthly audit prompt | Operationalized | monthly audit template and site-audit workflow |

## Sections 73–79: CI, launch checklists, and anti-patterns

| § | Topic | Status | Evidence |
| ---: | --- | --- | --- |
| 73 | Automated quality gates | Implemented | build, SEO/schema, links/fragments/orphans, sitemap, media, content, product claims, performance, Lighthouse, freshness, IndexNow |
| 74 | Protected product claims | Implemented | centralized product tests and generated schema/metadata scans for unsupported agents/OSs/reviews |
| 75 | Per-page checklist | Implemented | generated-site validator enforces the automatable contract; monthly template covers editorial/manual review |
| 76 | Pre-launch checklist | Operationalized / external | operations launch record, production smoke, console/analytics/log evidence fields |
| 77 | SEO traps | Enforced | no keyword stuffing, duplicate canonicals, fake freshness, fabricated schema, or thin bulk pages |
| 78 | AEO traps | Enforced | answers remain qualified, visible, complete, and non-duplicative |
| 79 | ASEO traps | Enforced | no secret AI tags, crawler allowlist guarantees, `llms.txt` magic, benchmark theater, or citation promises |

## External and release-timed completion register

These are the exact items that cannot truthfully be completed inside this
worktree:

| Item | Blocking authority/evidence | Completion evidence |
| --- | --- | --- |
| Deploy branch to production | merge/deploy authority | production deployment URL and immutable commit |
| Verify production crawl/status/meta/card behavior | deployed branch | production-smoke JSON with UTC time |
| Google domain property and sitemap | DNS/Search Console owner | verification and sitemap/inspection records |
| Bing property and sitemap | Bing owner | verification and inspection records |
| IndexNow production key/submission | secret manager + deployed proof | reviewed plan digest and redacted response log |
| Enable Plausible | analytics owner and privacy/retention decision | production config and event verification |
| Inspect CDN/WAF and verified bot logs | hosting/log access | dated log/WAF review with status counts |
| Field CWV and INP | deployed traffic and sufficient sample | Search Console/CrUX export with denominator |
| Manual AI-query baseline | provider access and allowed interfaces | completed query evidence sheet |
| External profile/topic updates | repository/social owner | live profile URLs and review date |
| Community/newsletter/directory launch | usable public trial + owner judgment | submitted/live URLs and rule compliance |
| Third-party mentions/backlinks | independent editorial action | live corroborating source |
| Stable compatibility claims | outstanding physical RC6 acceptance | signed acceptance evidence for every required row |
| Benchmarks/research/testimonials | reproducible data or permission | methodology/raw data or consent record |

No repository change can substitute for those records.

