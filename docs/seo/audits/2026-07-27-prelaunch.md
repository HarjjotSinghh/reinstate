# SEO, AEO, and AI-search prelaunch audit — 2026-07-27

## Audit identity

| Field | Value |
| --- | --- |
| Audited implementation commit | `59cecde121ce06a9ccbd7f5b93329a37082cdec1` |
| Branch | `feat/seo-aeo-aseo-foundation` |
| Worktree | `/Users/harjjotsinghh/Documents/Projects/reinstate-seo` |
| Product version | `v0.1.0-rc.6` |
| Local audit date | 2026-07-27 |
| Production deployment | Not deployed |
| Previous comparable audit | Initial landing-page baseline in the supplied playbook |

This report distinguishes generated-branch evidence from production, account,
real-device, field-performance, and third-party evidence. `Unavailable` does
not mean zero, and a local pass does not imply indexing, ranking, rich-result,
or AI-citation eligibility.

## Executive summary

The repository implementation is ready for deployment review. The generated
site has 49 indexable canonical URLs, 64 HTML pages, 64 unique route-specific
Open Graph cards, one permanent redirect, and a maximum logical crawl depth of
two. Automated metadata, structured-data visibility, links, fragments,
orphans, sitemap, media, freshness, performance, browser interaction, and
repository verification gates pass.

No Critical or High branch defect remains. Production is still serving the old
landing site, so deployment and every production-only observation remain open
launch gates. The only local quality warning is homepage lab LCP at 2,852 ms;
all other representative LCP measurements are 1,802–2,252 ms, all 16
Lighthouse routes have zero CLS, and accessibility, best practices, and SEO
score 100.

## Changes since the initial landing baseline

- Added sitemap generation, explicit crawler policy, canonical redirects, a
  real branded 404, verification hooks, and a bounded production smoke test.
- Centralized metadata, Open Graph, Twitter cards, crawler directives, dates,
  and safe JSON-LD in shared Astro layouts.
- Added visible/schema-parity enforcement for WebPage, TechArticle,
  BlogPosting, BreadcrumbList, FAQPage, HowTo, HowToStep, and Question data.
- Added complete docs, guides, blog, integrations, use cases, comparisons,
  security, compatibility, about, open-source, roadmap, research, glossary,
  release-history, and synthetic-tool surfaces.
- Added combined, blog-only, and changelog RSS feeds plus a factual optional
  `llms.txt`.
- Added four evidence-safe authority assets: terminology glossary, fixed
  synthetic path visualizer, encrypted snapshot v1 implementation
  specification, and tagged agent-version history.
- Added privacy-safe analytics contracts, Search Console and Bing verification
  hooks, IndexNow planning/submission controls, freshness ownership, uptime
  monitoring, and weekly/monthly/quarterly/release evidence templates.
- Installed and synchronized all nine supplied repository-local skills for
  Codex-compatible agents and Claude Code.

## Verification evidence

| Gate | Result |
| --- | --- |
| Fresh dependency install | 600 packages installed; npm audit reported 0 vulnerabilities |
| Website source tests | 27 Vitest files, 155 tests passed |
| Script contract suites | SEO 9/9; links 6/6; performance 5/5; IndexNow 9/9; preview 2/2; freshness 4/4; Lighthouse 4/4; production discovery 7/7 |
| Astro production build | Passed; all expected static routes, cards, feeds, and sitemap generated |
| Generated SEO | 49 indexable pages, 64 HTML pages, 64 social cards, 1 redirect, 49 sitemap URLs |
| Internal links | 2,856 links, 457 asset references, 254 fragments; all 49 indexable URLs reachable in at most 2 steps |
| Freshness | 55 reviewed records; 0 warnings; 0 errors |
| Static performance | 18 representative templates passed |
| Rendered browser | 16 routes passed; SEO/accessibility/best practices 100; performance 92–99; CLS 0 |
| Visualizer execution | 4 fixed states passed at 390×844; 0 interaction requests; 0 persistence changes |
| Responsive media | 2 source images passed generation parity |
| Open Graph | 64/64 unique PNGs; all 1200×630; 81,274–112,102 bytes |
| GitHub social preview | Current, 1280×640 PNG, 96,906 bytes |
| Repository verification | vet, golangci-lint, unit tests, race tests, govulncheck, and release-equivalent build passed |

## Findings by severity

### Critical

None in the audited branch.

### High

None in the audited branch.

The following are High launch blockers but not local implementation defects:

1. The audited commit is not deployed.
2. Native macOS, macOS amd64, Windows, WSL2, and complete two-device RC6
   acceptance evidence is still open.
3. Real-browser keyboard, 200% zoom, dark-theme, no-JavaScript, Windows, and
   assistive-technology acceptance is not recorded.

### Medium

1. Homepage lab LCP is 2,852 ms against the 2,500 ms target. Global CSS
   inlining reduced it to 2,551–2,554 ms but caused 23 HTML-budget failures and
   was correctly rejected. Field LCP and INP are unavailable.
2. Google Search Console, Bing Webmaster Tools, Plausible, verified CDN/WAF
   crawler logs, and manual AI-query results are unavailable until authorized
   production setup.

### Low

None left without a documented owner or evidence gate.

## Page and content findings

- Every indexable page has a unique title, description, canonical, H1,
  answer-first paragraph, route-specific card, visible breadcrumb path, and
  sitemap entry.
- Docs, guides, and the engineering article have validated intent, author,
  publication/update/review dates, related links, and structured metadata.
- All five guides expose prerequisites, steps, expected output, failure modes,
  verification, and rollback. Visible HowTo steps match JSON-LD.
- Troubleshooting contains eleven complete diagnostic contracts.
- Comparisons use explicit sourced dimensions and avoid review/rating schema.
- Current and planned capabilities are separated. Native resume remains
  same-vendor; cross-agent continuity is an explicit later handoff.
- Product-truth review corrected two subtle overclaims: RC5 performed a
  metadata-only remote-manifest probe, not decryption; and the lower-level
  `${WORK:<alias>}` primitive is not wired into RC6 config or adapters.

## Technical SEO and structured-data findings

- `robots.txt` separates search discovery from GPTBot training policy and
  allows OAI-SearchBot and PerplexityBot. Production WAF behavior is not yet
  proven.
- Sitemap, canonical, redirect, noindex, 404, image, duplicate, orphan, depth,
  and schema gates are enforced against generated output.
- JSON-LD is breakout-safe and uses canonical absolute URLs. Dates and
  citation-relevant names must exist in visible HTML.
- The three feeds are independently generated, linked once in the shared head,
  and checked in the production discovery contract.
- Meta keywords are prohibited. `llms.txt` is documented as optional
  navigation, never a Google ranking control.

## Open Graph and media findings

All 64 generated HTML routes—including indexable content, the branded 404, and
noindex design previews—have distinct 1200×630 PNG cards. Each includes the
same bare dual-frame prompt mark and “Reinstate” lockup used in the header and
footer, Questrial display headings, Geist supporting text, route title,
description, content type, canonical route, current RC, Apache-2.0 label, and
the landing-page isometric continuity stack.

Visual inspection covered the homepage, getting-started documentation,
glossary, path visualizer, snapshot specification, agent-version history, and
1280×640 GitHub preview. A missing-glyph defect in the visualizer card was
found and fixed before this audit.

## AEO and AI-search findings

- The 41-question inventory assigns each high-intent question one canonical
  visible answer.
- Answer blocks, definition lists, comparison tables, troubleshooting
  contracts, and source links are in initial HTML without client rendering.
- Entity facts are centralized and protected against unsupported agents,
  platforms, ratings, testimonials, stable-release, and cross-agent claims.
- OAI-SearchBot, PerplexityBot, official bot-IP verification guidance, AI
  referral attribution, and a fixed manual query baseline are operationalized.
- No special AI tag, `llms.txt` guarantee, crawler-access guarantee, citation
  promise, or fabricated third-party authority claim is made.

## Production observation

The read-only smoke test ran 51 bounded unauthenticated checks against
`https://reinstate.dev`: 10 passed and 41 failed, with 75 findings, zero
sitemap URLs, and zero branch route cards observed. This is expected because
the branch is not deployed. Evidence:
`website/artifacts/production-discovery/2026-07-27-production.json`, SHA-256
`7bdebaddfe385c5b72dde9f9d5c9db07a593d6a2cc4c9f62932f7b5ea5c97fab`.

## Missing measurements

| Evidence | State |
| --- | --- |
| Production deployment commit/URL | Unavailable — not deployed |
| Google property, sitemap, URL Inspection, AI-feature setting | Unavailable — account/DNS owner required |
| Bing property, sitemap, URL inspection | Unavailable — account owner required |
| IndexNow key proof and production submission | Unavailable — deployed proof and secret required |
| Plausible property, retention/privacy decision, production events | Unavailable — analytics owner required |
| Verified OAI-SearchBot/PerplexityBot requests and WAF outcomes | Unavailable — CDN/WAF logs required |
| Field LCP, CLS, and INP | Unavailable — deployed traffic and sufficient sample required |
| Manual AI query/citation baseline | Unavailable — provider runs not performed |
| Real browser, screen reader, native Windows/macOS acceptance | Unavailable — release operators/devices required |
| Rankings, impressions, clicks, CTR, referrals, conversions | Unavailable — never infer from local implementation |

## Exact next tasks

1. Merge and deploy an immutable reviewed commit; record its SHA and deployment
   URL.
2. Run `npm --prefix website run check:production-discovery` immediately after
   deployment and again after production promotion. Require all canonical,
   sitemap, card, feed, and crawler-string checks to pass.
3. Complete the manual browser matrix and RC6 physical platform/two-device
   acceptance before a stable or complete Phase 1 claim.
4. Configure and verify Search Console and Bing, then submit the canonical
   sitemap. Record URL Inspection evidence for launch-critical pages.
5. Provision IndexNow only after the public key proof is live; review the
   generated delta digest before submission.
6. Enable Plausible only after owner/privacy review; validate controlled events
   without collecting session content, paths, commands, passphrases, or keys.
7. Review WAF rules and logs using current provider-published IP ranges plus
   user-agent verification.
8. Run the fixed AI-query baseline manually and record citations, cited URLs,
   factual errors, and “not cited” outcomes without interpreting absence as
   zero visibility.
9. Revisit homepage performance after field data exists. Do not raise
   thresholds or reintroduce the rejected global-inline configuration.

## Revalidation plan

For every material website or release change:

```sh
npm --prefix website ci
npm --prefix website test
npm --prefix website run build
npm --prefix website run check:seo
npm --prefix website run check:links
npm --prefix website run check:performance
npm --prefix website run check:freshness
npm --prefix website run check:media
npm --prefix website run check:github-social
npm --prefix website run check:path-visualizer
npm --prefix website run check:lighthouse
make verify
```

After deployment, add the production discovery check, console inspections,
manual browser matrix, verified crawler-log review, and evidence-backed
reporting. Do not mark an external row complete from source code or a local
browser run.
