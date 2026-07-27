# Website discoverability

This directory contains Reinstate's implementation guidance for:

- search engine optimization (SEO);
- answer engine optimization (AEO); and
- AI search engine optimization (ASEO).

The [playbook](seo-aeo-aseo-playbook.md) is the strategic and technical
reference. Repository-local agent workflows live in
[`/.agents/skills`](../../.agents/skills/README.md).

Operational follow-up uses:

- [the production operations runbook](operations.md);
- [the release and launch distribution runbook](launch-distribution.md);
- [the evidence-aware content roadmap](content-roadmap.md);
- [the section-by-section implementation evidence matrix](implementation-status.md);
- [the fixed AI-search query baseline](ai-search-query-baseline.md); and
- [the weekly report template](weekly-report-template.md);
- [the monthly audit template](monthly-audit-template.md);
- [the quarterly review template](quarterly-review-template.md); and
- [the immutable release evidence template](release-evidence-template.md).

Production observations:

- [2026-07-27 pre-deployment discovery baseline](baselines/2026-07-27-pre-deployment-production-discovery.md)
- [2026-07-27 local rendered-browser baseline](baselines/2026-07-27-local-lighthouse.md)

## Source-of-truth order

Discoverability work must not turn roadmap direction into a current product
claim. Resolve facts in this order:

1. released code, tests, and release metadata;
2. current compatibility, security, and user documentation;
3. `website/src/data/product.ts`;
4. page copy and structured data;
5. this playbook.

## Implementation scope

The first implementation pass covers:

- canonical product facts;
- sitemap and crawler policy;
- shared metadata and structured data;
- index controls for preview and error routes;
- enriched documentation metadata and breadcrumbs;
- high-intent integration, compatibility, security, and fact pages;
- answer-first guides and an evidence-backed engineering blog;
- route-specific branded Open Graph images;
- an explicit website privacy notice and security contact file;
- RSS and `llms.txt` discovery files; and
- generated-site SEO, internal-link, and static-performance validation in CI;
  and
- reviewed IndexNow sitemap-diff planning, key-proof handling, rate-safe
  soft-fail submission, and non-submitting CI validation.

Search Console, Bing Webmaster Tools, IndexNow key provisioning and production
submission, production bot logs, external profile updates, and referral
analytics require production access or account-level configuration and are
tracked as operational follow-up work rather than fabricated as code-complete.
