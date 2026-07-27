---
name: reinstate-structured-data
description: Select, implement, validate, and test truthful JSON-LD for Reinstate pages. Use for homepage, documentation, guides, articles, FAQs, software facts, source-code facts, and breadcrumbs.
---

# Reinstate Structured Data

## Principles

- JSON-LD must match visible page content.
- Use the most specific accurate type.
- Do not invent reviews, ratings, prices, users, or capabilities.
- Use absolute canonical URLs.
- Escape serialized JSON safely.
- Prefer an `@graph` when several related entities appear.
- Validate Schema.org vocabulary and values with Schema.org Validator.
- Use Google's Rich Results Test only for types Google currently documents as
  supported search features.
- Rich-result eligibility is not guaranteed.

## Page mapping

### Homepage

Consider:

- `Organization` or accurate project publisher
- `WebSite`
- `SoftwareApplication`
- `SoftwareSourceCode`

### Documentation

Consider:

- `TechArticle`
- `HowTo` only for a genuine visible sequence
- `BreadcrumbList`

`HowTo` can remain valid Schema.org semantics even when Google exposes no
corresponding rich-result feature. Never describe vocabulary validity as
Google rich-result eligibility.

### Blog

Consider:

- `BlogPosting` or `Article`
- `BreadcrumbList`

### FAQ

Use `FAQPage` only when all questions and answers are visible.

Google removed its FAQ rich-result feature in 2026. `FAQPage` may still express
the visible page entity for other consumers, but do not promise a Google FAQ
enhancement or treat Rich Results Test absence as a schema.org failure.

### Comparison

Use `WebPage`, `Article`, or `TechArticle` as appropriate. Do not mark a comparison as a product review unless it is genuinely a review and all required facts exist.

## Current verified software facts

Load `reinstate-product-truth` before generating schema.

## Validation workflow

1. Identify the main page entity.
2. List visible facts.
3. map facts to schema properties.
4. remove properties lacking visible support.
5. serialize safely.
6. validate syntax.
7. validate URLs and dates.
8. compare generated JSON-LD with the rendered page.
9. add regression tests.
10. report fields intentionally omitted.

## Prohibited fields without proof

- `aggregateRating`
- `review`
- unsupported `operatingSystem`
- unsupported agents
- fake organization details
- fake founding dates
- fake awards
- fake download counts
