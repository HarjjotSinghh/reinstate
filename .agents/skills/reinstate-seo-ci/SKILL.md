---
name: reinstate-seo-ci
description: Add and maintain automated SEO quality gates for Reinstate's Astro website, covering builds, metadata, canonicals, sitemaps, links, structured data, content frontmatter, status codes, and performance regressions.
---

# Reinstate SEO CI

## Required checks

### Build

- production build succeeds
- generated routes match expectations

### Metadata

For every indexable HTML page:

- exactly one title
- exactly one description
- exactly one canonical
- exactly one H1
- valid robots directive
- Open Graph image and alt
- Twitter image and alt

### Canonicals

- absolute HTTPS
- correct host
- status 200
- no redirect
- no fragment
- unique

### Sitemap

- valid XML
- canonical URLs only
- no draft, preview, API, redirect, 404, or `noindex` URL

### Structured data

- parseable JSON
- safe serialization
- valid absolute URLs
- valid dates
- no prohibited fabricated fields
- visible-content match

### Content

- required frontmatter
- unique slug
- updated date
- no future publication unless scheduled
- related links resolve
- no placeholder copy
- current product claims

### Links

- internal links
- anchors
- assets
- canonical destinations

### Performance

- Lighthouse SEO score 1.0 as a smoke test
- accessibility regression warning
- LCP lab warning above 2500 ms
- CLS warning above 0.1
- field INP tracked separately

## Workflow

1. Inspect existing CI.
2. choose minimal maintained tools.
3. add scripts.
4. add a representative route list.
5. create actionable failure messages.
6. avoid flaky network dependence where possible.
7. run locally.
8. run in CI.
9. document how to update baselines.
