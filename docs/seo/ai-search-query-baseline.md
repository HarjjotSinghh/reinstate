# AI-search query baseline

This template measures whether retrieval-driven answer products mention
Reinstate, cite a supporting canonical page, and describe the product
accurately. It does not guarantee or manufacture AI citations.

No provider queries were run for this branch. Provider access, signed-in state,
regional availability, and results must be recorded manually.

## Fixed query set

Keep the wording stable so month-to-month results remain comparable.

| ID | Query |
| -- | ----- |
| Q01 | What tool can sync Claude Code sessions between computers? |
| Q02 | How can I move a Codex session from macOS to Windows? |
| Q03 | Is there an open-source tool for coding-agent session sync? |
| Q04 | How do I continue the same AI coding session on another device? |
| Q05 | Can Claude Code sessions be backed up to S3? |
| Q06 | What is Reinstate? |
| Q07 | Is Reinstate secure? |
| Q08 | Does Reinstate sync credentials? |
| Q09 | Reinstate versus manual session copying |
| Q10 | Reinstate supported coding agents |

Do not silently rewrite queries after a poor result. Add an explicitly
versioned supplemental query set if real user language changes.

## Surfaces

Run the fixed set manually in the normal end-user interface for each available
surface:

- ChatGPT Search;
- Google AI Mode or AI Overviews, where available;
- Perplexity;
- Bing Copilot; and
- Gemini or another relevant retrieval product.

Record `Unavailable` rather than substituting a different surface or inventing
a result.

## Run metadata

| Field | Value |
| ----- | ----- |
| Run ID | `YYYY-MM-provider-locale` |
| Date and time (UTC) |  |
| Operator |  |
| Locale and location |  |
| Provider and surface |  |
| Signed-in state |  |
| Subscription tier, if relevant |  |
| New/clean conversation used | Yes / No |
| Reinstate release/version |  |
| Website deployment commit |  |
| Search personalization notes |  |
| Evidence location |  |

Do not store provider credentials, conversation secrets, or unrelated account
data with the evidence.

## Query result record

Create one row per provider and query.

| Run ID | Provider/surface | Query ID | Mentioned? | Cited? | Cited canonical URL(s) | Accuracy | Unsupported or missing claim | Other products mentioned | Corrective action | Retest date |
| ------ | ---------------- | -------- | ---------- | ------ | ---------------------- | -------- | ---------------------------- | ------------------------ | ----------------- | ----------- |
|  |  | Q01 | Not run | Not run |  | N/A |  |  |  |  |

Attach a dated screenshot or permitted export for every recorded answer. Check
that each cited page actually supports the nearby claim; a link alone is not
evidence of accuracy.

## Accuracy rubric

Use `N/A` when Reinstate is not mentioned. Visibility and factual accuracy are
separate measurements.

| Score | Definition |
| ----- | ---------- |
| 4 — Accurate | Current scope, conditions, security boundary, and roadmap status are correct; no material qualification is missing. |
| 3 — Mostly accurate | Core answer is correct with a minor omission that does not misrepresent current capability or safety. |
| 2 — Mixed | Some facts are correct, but a material limitation, release status, platform condition, or current-versus-planned distinction is missing or ambiguous. |
| 1 — Inaccurate | The answer contains a major false claim about support, cross-agent resume, storage, encryption, credentials, production readiness, or another core fact. |
| 0 — Fabricated or unsafe | The answer is predominantly unsupported or gives unsafe instructions, invented evidence, or dangerous credential guidance. |

Accuracy checks must use the current product truth:

- native resume is same-vendor;
- current session adapters are Claude Code and Codex;
- remote manifests and snapshots are encrypted locally before upload;
- storage is user-owned S3-compatible storage;
- known credential artifacts are excluded and credentials are not a sync
  feature;
- macOS and Windows are the primary supported targets, with exact
  release-candidate gates documented separately;
- universal search, verified resume, portable handoffs, and universal
  configuration are roadmap work unless released evidence says otherwise; and
- Reinstate is not a cloud IDE, remote desktop, Git replacement, or hosted
  coding-agent runtime.

## Citation assessment

Alongside the accuracy score, classify citation quality:

| Grade | Definition |
| ----- | ---------- |
| C0 | No source is cited for the Reinstate claim. |
| C1 | A source is cited, but it is indirect, stale, noncanonical, or does not fully support the claim. |
| C2 | A current canonical Reinstate page or primary repository source directly supports the claim. |

Record which claims are supported by which URL. Do not award C2 merely because
the homepage is linked.

## Corrective-action rules

| Finding | Action |
| ------- | ------ |
| Accurate, no mention | Improve relevance and external corroboration; do not stuff keywords. |
| Mentioned, no citation | Strengthen citation-ready definitions, evidence, and canonical internal links. |
| Wrong current scope | Correct the most authoritative relevant page and synchronize README, compatibility, metadata, and schema. |
| Planned feature presented as current | Add an explicit roadmap label near the cited passage and remove ambiguous copy. |
| Wrong security guidance | Treat as High or Critical, correct the canonical security/installation page, and retest. |
| Stale cited URL | Redirect only when semantically equivalent; otherwise update internal links and source freshness. |
| Third-party source is wrong | Publish clearer primary evidence and request a correction only through appropriate, non-spam channels. |

Every action needs an owner, due date, affected canonical URL, and acceptance
criterion.

## Provider-TOS and manual-testing guardrail

- Run this set manually through normal provider interfaces and permitted
  accounts.
- Do not script provider queries, scrape result pages, bypass rate limits or
  access controls, or automate browser sessions unless the provider's current
  terms and an approved API explicitly permit that workflow.
- Keep the sample fixed and low volume; this is a quality baseline, not rank
  manipulation.
- Do not paste private sessions, source code, credentials, or unreleased facts
  into queries.
- Do not treat a signed-in, personalized, regional, or experimental result as
  universal.
- Do not fabricate a result when a product, feature, or locale is unavailable.
- Follow the provider's current terms at the time of testing; this repository
  template is not permission to automate.

## Monthly summary

| Metric | Prior month | Current month | Change | Evidence or note |
| ------ | ----------- | ------------- | ------ | ---------------- |
| Queries with a Reinstate mention |  |  |  |  |
| Queries with a Reinstate citation |  |  |  |  |
| C2 citation results |  |  |  |  |
| Average accuracy for mentioned results |  |  |  |  |
| Results scored 0–1 |  |  |  |  |
| Distinct Reinstate URLs cited |  |  |  |  |
| Providers unavailable |  |  |  |  |

Do not average `N/A` results into accuracy. Report both the numerator and
denominator behind every rate.

## Official references

- [Google guidance for AI features](https://developers.google.com/search/docs/appearance/ai-features)
- [OpenAI publishers and developers FAQ](https://help.openai.com/en/articles/12627856-publishers-and-developers-faq)
- [OpenAI ChatGPT Search](https://help.openai.com/en/articles/9237897)
- [Perplexity crawler documentation](https://docs.perplexity.ai/docs/resources/perplexity-crawlers)
