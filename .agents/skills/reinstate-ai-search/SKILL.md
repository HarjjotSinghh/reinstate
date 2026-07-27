---
name: reinstate-ai-search
description: Audit and improve Reinstate for ChatGPT Search, Google generative search, Perplexity, Bing Copilot, and other retrieval-driven AI systems through crawler access, entity clarity, citation-ready facts, freshness, and external corroboration.
---

# Reinstate AI Search

## Principles

- Normal SEO is the foundation.
- Google does not require special AI markup or AI text files.
- `llms.txt` is optional navigation, not a ranking guarantee.
- `OAI-SearchBot` controls ChatGPT Search discovery separately from GPTBot training policy.
- `PerplexityBot` is the documented Perplexity search crawler.
- Crawler access must also work through CDN and WAF layers.
- Citation likelihood comes from relevance, evidence, clarity, and authority, not keyword stuffing.

## Workflow

1. Load `reinstate-product-truth`.
2. verify crawler policy.
3. test homepage and key pages with relevant user agents.
4. inspect server logs for 403, 429, and 5xx.
5. audit entity consistency across:
   - website
   - README
   - GitHub About
   - releases
   - package metadata
   - social profiles
6. audit citation units:
   - definitions
   - facts
   - procedures
   - tables
   - benchmarks
7. verify visible dates and versions.
8. identify missing primary-source evidence.
9. identify third-party corroboration opportunities.
10. run the fixed AI-query test set manually.
11. record mentions, citations, URLs, accuracy, date, and corrective action.

## Never

- guarantee AI citations
- add invisible model-targeted text
- generate fan-out pages at scale
- invent external validation
- confuse search crawling and training
- trust spoofable user-agent strings for security
