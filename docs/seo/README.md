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
- [the fixed AI-search query baseline](ai-search-query-baseline.md); and
- [the monthly audit template](monthly-audit-template.md).

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
- RSS and `llms.txt` discovery files; and
- generated-site SEO validation in CI.

Search Console, Bing Webmaster Tools, production bot logs, external profile
updates, and referral analytics require production access or account-level
configuration and are tracked as operational follow-up work rather than
fabricated as code-complete.
