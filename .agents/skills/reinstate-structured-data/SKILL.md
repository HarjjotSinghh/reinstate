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
- Validate with Schema.org Validator and Google's Rich Results Test.
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

### Blog

Consider:

- `BlogPosting` or `Article`
- `BreadcrumbList`

### FAQ

Use `FAQPage` only when all questions and answers are visible.

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
