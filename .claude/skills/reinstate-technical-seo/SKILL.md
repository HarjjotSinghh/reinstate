---
name: reinstate-technical-seo
description: Implement or audit technical SEO in Reinstate's Astro website, including crawlability, metadata, canonicals, sitemaps, robots, rendering, status codes, internal links, performance, and index controls.
---

# Reinstate Technical SEO

## Scope

Use for work under `website/`.

## Required checks

### Discovery

- `site` is `https://reinstate.dev`
- XML sitemap exists and contains canonical indexable pages
- `robots.txt` exists at the root
- `OAI-SearchBot` and `PerplexityBot` policy is explicit
- GPTBot training policy is handled separately
- no preview, API, or draft URLs are in the sitemap

### Indexation

- one canonical per page
- canonical is absolute and returns 200
- one robots directive per page
- drafts and previews use `noindex`
- missing pages return 404
- redirects are permanent only when appropriate
- no soft 404s

### Metadata

- unique title
- unique description
- Open Graph site name, title, description, URL, image, and image alt
- Twitter card, title, description, image, and image alt
- no meta keywords
- no unsupported product claims

### HTML

- important text is in initial HTML
- one H1
- semantic headings
- real anchor links
- accessible landmarks
- no critical content hidden behind interaction

### Performance

- LCP target 2500 ms or less
- INP target 200 ms or less in field data
- CLS target 0.1 or less
- minimize hydrated islands
- responsive images
- dimensions on media
- no unnecessary font families or third-party scripts

## Workflow

1. Read `astro.config.mjs`, `package.json`, layouts, content schemas, and page routes.
2. Build the site.
3. inspect generated output.
4. crawl production or preview.
5. produce a prioritized issue list.
6. implement the highest-impact safe fixes.
7. add tests.
8. rebuild.
9. report results and remaining risks.

## Do not

- canonicalize unrelated pages to the homepage
- block indexable pages in robots
- add JavaScript-only metadata
- add unsupported crawler names from stale lists
- modify visual design without need
- claim a ranking guarantee
