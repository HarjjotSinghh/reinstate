# Monthly SEO, AEO, and AI-search audit

Use this template for a production monthly review. Leave unavailable evidence
marked `Unavailable`; never convert missing access into a zero or a fabricated
pass.

This template follows the repository-local product-truth, technical-SEO,
structured-data, answer-optimization, AI-search, SEO-CI, and site-audit skills
under `.agents/skills/`.

## Audit identity

| Field | Value |
| ----- | ----- |
| Audit month |  |
| Review period |  |
| Auditor(s) |  |
| Production deployment commit |  |
| Reinstate release/version |  |
| Previous audit |  |
| Report/evidence directory |  |

## External-access disclosure

These inputs require account-level or production access and are not automated
or proven complete by the repository branch:

| Evidence | Access owner | Available? | Date exported/reviewed | Location or reason unavailable |
| -------- | ------------ | ---------- | ---------------------- | ------------------------------ |
| Google Search Console |  | No / Yes |  |  |
| Bing Webmaster Tools |  | No / Yes |  |  |
| Plausible analytics |  | No / Yes |  |  |
| CDN/WAF configuration |  | No / Yes |  |  |
| Production request logs |  | No / Yes |  |  |
| Core Web Vitals field data |  | No / Yes |  |  |
| Manual AI-query results |  | No / Yes |  |  |

## Executive summary

### Outcome

One evidence-backed paragraph covering material changes, the largest risk, the
largest opportunity, and whether prior Critical or High findings were closed.

### Findings by priority

| Priority | Count | Summary |
| -------- | ----- | ------- |
| Critical — blocks crawling, indexing, rendering, security, or truthful representation |  |  |
| High — strongly affects discovery, intent satisfaction, conversion, or citation accuracy |  |  |
| Medium — improves quality, structure, performance, or maintainability |  |  |
| Low — limited-impact polish |  |  |

## Changes since the prior audit

| Change | URL/component | Release or commit | Expected effect | Evidence observed |
| ------ | ------------- | ----------------- | --------------- | ----------------- |
|  |  |  |  |  |

Do not attribute a traffic or citation change to a deployment without enough
evidence to separate it from seasonality, provider changes, or low sample size.

## 1. Product-truth audit

- [ ] Current release, release status, supported agents, operating systems,
      storage, encryption, license, and account requirement were rechecked.
- [ ] Native resume is described as same-vendor only.
- [ ] Roadmap features are visibly labeled as planned.
- [ ] No ratings, reviews, customer counts, benchmarks, awards, or platform
      support were added without evidence.
- [ ] Website, docs, README, compatibility, changelog, schema, RSS, `llms.txt`,
      and external profiles agree.

| Claim | Classification: verified/planned/ambiguous/unsupported | Evidence | Conflicting URL/file | Required correction |
| ----- | ----------------------------------------------------- | -------- | -------------------- | ------------------- |
|  |  |  |  |  |

## 2. Technical SEO audit

### Build and discovery

- [ ] `npm --prefix website test` passes.
- [ ] `npm --prefix website run build` passes.
- [ ] Production crawl covers every canonical HTML page.
- [ ] Sitemap parses and matches canonical indexable pages.
- [ ] Sitemap excludes preview, API, draft, redirect, 404, and `noindex` URLs.
- [ ] `robots.txt` exposes the sitemap and preserves the intended crawler
      policy.
- [ ] Missing URLs return a real `404`, not a soft 404.
- [ ] Redirects are intentional and permanent only where appropriate.

### Page contract

- [ ] Exactly one unique title and description.
- [ ] Exactly one absolute HTTPS canonical on `reinstate.dev`.
- [ ] Exactly one robots directive.
- [ ] Exactly one H1 with semantic heading order.
- [ ] Important content is present in initial HTML.
- [ ] Open Graph and Twitter titles, descriptions, images, and image alt text.
- [ ] Internal links and assets resolve.
- [ ] Indexable canonical returns `200` without a redirect.
- [ ] Preview and temporary pages remain `noindex` and outside the sitemap.

### Structured data

- [ ] JSON-LD parses after safe serialization.
- [ ] Schema type matches the page's visible purpose.
- [ ] URLs are absolute and dates are valid.
- [ ] Breadcrumb schema matches visible breadcrumbs.
- [ ] Article, software, source, and FAQ facts match visible content.
- [ ] No fabricated ratings, reviews, prices, users, awards, or unsupported
      capabilities.
- [ ] Representative pages were validated using current schema tools.

| URL | Finding | Priority | Evidence | Owner | Acceptance criterion |
| --- | ------- | -------- | -------- | ----- | -------------------- |
|  |  |  |  |  |  |

## 3. Content and AEO audit

- [ ] Each high-intent page answers its main question immediately.
- [ ] Extracted answer paragraphs remain accurate without surrounding context.
- [ ] Procedures include prerequisites, ordered steps, expected outcome,
      limitations, security notes, and troubleshooting.
- [ ] Comparison tables use explicit dimensions and qualify current versus
      planned behavior.
- [ ] Question headings match real user intent without duplicating the same FAQ
      across pages.
- [ ] Every substantive page has a meaningful visible update date.
- [ ] Commands were checked against the current CLI and release.
- [ ] Thin, overlapping, orphaned, or stale pages were identified.
- [ ] Related links point to the most authoritative canonical source.

| Query/intent | Best current page | Answer accurate alone? | Gap or overlap | Editorial action | Acceptance criterion |
| ------------ | ----------------- | ---------------------- | -------------- | ---------------- | -------------------- |
|  |  |  |  |  |  |

## 4. AI-search audit

- [ ] `OAI-SearchBot` and `PerplexityBot` policy is explicit.
- [ ] Public user-agent smoke tests return ordinary successful responses.
- [ ] Verified production logs, when available, were reviewed for `403`, `429`,
      `5xx`, loops, and WAF challenges.
- [ ] Search crawling and GPTBot training policy remain separate decisions.
- [ ] Entity facts agree across website, README, GitHub, releases, and social
      profiles.
- [ ] Definitions, factual claims, procedures, and tables are citation-ready
      and supported by primary evidence.
- [ ] Dates, release versions, and compatibility claims are fresh.
- [ ] The fixed query set was tested manually without violating provider terms.
- [ ] Mentions, citations, cited URLs, accuracy, and corrective actions were
      recorded.

AI-query run: [attach or link a completed baseline](ai-search-query-baseline.md).

| Finding | Provider/query | Accuracy/citation grade | Affected canonical | Corrective action | Owner/due date |
| ------- | -------------- | ----------------------- | ------------------ | ----------------- | -------------- |
|  |  |  |  |  |  |

## 5. Search and analytics measurements

### SEO

| Metric | Prior | Current | Change | Denominator/source | Interpretation |
| ------ | ----- | ------- | ------ | ------------------ | -------------- |
| Indexed canonical pages |  |  |  |  |  |
| Non-brand impressions |  |  |  |  |  |
| Non-brand clicks |  |  |  |  |  |
| Branded impressions/clicks |  |  |  |  |  |
| CTR by priority page |  |  |  |  |  |
| Crawl/indexing errors |  |  |  |  |  |
| Qualified organic actions |  |  |  |  |  |

### AEO and AI search

| Metric | Prior | Current | Change | Denominator/source | Interpretation |
| ------ | ----- | ------- | ------ | ------------------ | -------------- |
| Question-query impressions/clicks |  |  |  |  |  |
| FAQ/troubleshooting entrances |  |  |  |  |  |
| Google Generative AI report impressions/clicks |  |  |  |  |  |
| Queries with AI mention |  |  |  |  |  |
| Queries with AI citation |  |  |  |  |  |
| C2 citation results |  |  |  |  |  |
| Accuracy score for mentioned results |  |  |  |  |  |
| AI referral sessions/actions |  |  |  |  |  |
| Verified crawler failure rate |  |  |  |  |  |

Use `Unavailable` for a missing source. Do not report a percentage without its
numerator and denominator.

## 6. Performance and accessibility

| URL/template | LCP lab | CLS lab | Lighthouse SEO | Accessibility | Field CWV/INP | Finding |
| ------------ | ------- | ------- | -------------- | ------------- | ------------- | ------- |
|  |  |  |  |  |  |  |

Targets:

- LCP: `2.5 s` or less;
- CLS: `0.1` or less;
- INP: `200 ms` or less in field data; and
- Lighthouse SEO `1.0` as a smoke test, not a business outcome.

Lab data does not substitute for field data. Mark field measurements
unavailable until the sample is sufficient.

## 7. Freshness, links, and corroboration

- [ ] Release and compatibility pages reflect the current release.
- [ ] Updated dates represent meaningful review, not fake freshness.
- [ ] Deprecated commands and routes have a correction or migration path.
- [ ] External repository metadata and profiles use current product language.
- [ ] New backlinks and third-party mentions were checked for accuracy.
- [ ] Broken, redirected, and stale external links were reviewed.
- [ ] Original research or benchmarks include reproducible methodology and raw
      evidence before publication.

## 8. Action register

| ID | Priority | Finding | Exact task | Owner | Due date | Acceptance criteria | Validation | Status |
| -- | -------- | ------- | ---------- | ----- | -------- | ------------------- | ---------- | ------ |
|  |  |  |  |  |  |  |  | Open |

Every Critical or High item needs an owner, due date, and testable acceptance
criterion. Split engineering and editorial work when they require different
reviewers.

## 9. Validation and sign-off

- [ ] All automated checks rerun after fixes.
- [ ] Changed production URLs recrawled.
- [ ] Search-console or Bing actions recorded by an authorized operator.
- [ ] WAF/log findings validated by someone with production access.
- [ ] Product claims reviewed against released evidence.
- [ ] Security-sensitive copy reviewed.
- [ ] AI-search corrections manually retested where appropriate.
- [ ] Remaining unavailable measurements listed below.

### Measurements still unavailable

| Measurement | Blocking access or condition | Owner | Next step |
| ----------- | ---------------------------- | ----- | --------- |
|  |  |  |  |

### Final decision

| Field | Value |
| ----- | ----- |
| Audit status | Complete / Complete with gaps / Blocked |
| Critical findings open |  |
| High findings open |  |
| Next weekly check |  |
| Next monthly audit |  |
| Approver |  |

## Official references

- [Google Search Essentials](https://developers.google.com/search/docs/essentials)
- [Google robots meta specifications](https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag)
- [Google structured-data introduction](https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data)
- [Google guidance for generative-AI search](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)
- [Schema.org](https://schema.org/)
- [OpenAI publishers and developers FAQ](https://help.openai.com/en/articles/12627856-publishers-and-developers-faq)
- [Perplexity crawler documentation](https://docs.perplexity.ai/docs/resources/perplexity-crawlers)
