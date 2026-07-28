---
name: reinstate-site-audit
description: Run a prioritized monthly or pre-launch audit of Reinstate's production website and Astro repository across SEO, AEO, AI search, performance, content quality, freshness, links, schemas, and product-truth consistency.
---

# Reinstate Site Audit

## Audit sources

- production crawl
- repository
- generated build
- sitemap
- robots
- Search Console export
- Bing export
- analytics
- server logs
- Lighthouse or field data
- fixed AI-query test results
- release and compatibility data

Do not fabricate unavailable evidence.

## Priority model

### Critical

Blocks crawling, indexing, rendering, security, or truthful product representation.

### High

Strongly affects discovery, intent satisfaction, conversion, or citation accuracy.

### Medium

Improves quality, structure, performance, or maintainability.

### Low

Polish with limited measurable effect.

## Workflow

1. establish current product truth.
2. crawl canonical pages.
3. compare crawl with sitemap.
4. validate status codes and canonicals.
5. validate metadata.
6. validate structured data.
7. audit internal links and orphans.
8. audit content overlap and thin pages.
9. audit answer structure.
10. audit AI crawler access.
11. audit entity consistency.
12. audit freshness.
13. audit performance and accessibility.
14. compare analytics and search data.
15. produce tasks with owners and acceptance criteria.

## Output

- executive summary
- changes since prior audit
- critical failures
- high-impact opportunities
- page-level findings
- technical findings
- content findings
- AEO findings
- AI-search findings
- measurements missing
- exact engineering tasks
- exact editorial tasks
- validation plan
