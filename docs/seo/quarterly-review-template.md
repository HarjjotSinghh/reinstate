# Quarterly SEO, AEO, AI-search, and content-governance review

Create a dated copy of this template for each completed quarter. The quarterly
review combines comparable monthly evidence, audits the whole information
system, and decides what to keep, update, merge, redirect, remove, research, or
stop doing.

Do not invent traffic, rankings, citations, conversions, product outcomes,
release evidence, or competitor changes. A repository check cannot substitute
for Search Console, Bing, analytics, production logs, manual AI testing,
support evidence, or physical-device acceptance.

## Review identity

| Field | Value |
| --- | --- |
| Quarter and year |  |
| Review window, including time zone |  |
| Comparable prior quarter |  |
| Reviewer(s) |  |
| Approver |  |
| Production commit at quarter end |  |
| Reinstate release/tag at quarter end |  |
| Monthly audits included |  |
| Weekly reports included |  |
| Prior quarterly review |  |
| Evidence directory |  |

List launch periods, incomplete months, analytics migrations, consent changes,
provider reporting changes, major releases, incidents, or query-inventory
changes that make comparisons non-equivalent.

## Evidence-state rules

Use:

- `Available`;
- `Partial — <coverage or quality reason>`;
- `Unavailable — access not configured`;
- `Unavailable — provider does not expose`;
- `Unavailable — insufficient sample`;
- `Unavailable — no comparable prior window`; or
- `Not applicable — <reason>`.

Empty means not reviewed. Zero requires a complete named source that observed
zero in the stated window. Counts use `N/A — count` as denominator. Every
percentage, rate, share, coverage, pass rate, or growth calculation requires
the raw numerator and denominator. Preserve source exports and query/filter
definitions so another reviewer can reproduce the number.

## External-access disclosure

| Evidence | Access owner/operator | State | Export/review date | Window/coverage | Sanitized repository artifact or reason unavailable |
| --- | --- | --- | --- | --- | --- |
| Google Search Console queries/pages/indexing/CWV |  |  |  |  |  |
| Bing Webmaster Tools queries/indexing |  |  |  |  |  |
| IndexNow production plans/responses |  |  |  |  |  |
| Privacy-approved analytics and event definitions |  |  |  |  |  |
| CDN/WAF configuration |  |  |  |  |  |
| Verified production crawler logs |  |  |  |  |  |
| Backlink/referring-domain data |  |  |  |  |  |
| Manual fixed AI-query runs |  |  |  |  |  |
| GitHub traffic/referrals/releases |  |  |  |  |  |
| Support, install, and restore outcomes |  |  |  |  |  |
| Physical compatibility/release evidence |  |  |  |  |  |
| Approved competitor-result observations |  |  |  |  |  |

Repository-only work may prepare, validate, and document these flows. It does
not prove account ownership or production observations. Do not commit keys,
credentials, raw user/IP identifiers, session contents, source code, bucket
names, private queries, or unsanitized logs.

## Executive decision

### Evidence-backed quarter summary

Summarize the largest verified outcome, largest quality risk, product/release
truth change, strongest content opportunity, pruning decision, and material
unavailable evidence. Separate observations from causal hypotheses.

### Decision table

| Area | Decision | Primary evidence | Confidence and limitation | Owner | Due date | Acceptance criteria |
| --- | --- | --- | --- | --- | --- | --- |
| Information architecture |  |  |  |  |  |  |
| Content portfolio/pruning |  |  |  |  |  |  |
| Schema and technical SEO |  |  |  |  |  |  |
| Product positioning |  |  |  |  |  |  |
| AI-search/crawler access |  |  |  |  |  |  |
| Referral and conversion measurement |  |  |  |  |  |  |
| Product-quality guardrails |  |  |  |  |  |  |
| Primary research |  |  |  |  |  |  |

## 1. Complete KPI framework

Use the same filters and windows for current and prior values. When the
definition changes, start a new series and document the break instead of
splicing incompatible numbers.

### SEO KPIs

| KPI | Current | Prior | Change | Numerator | Denominator | Definition/source | State and interpretation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Indexed canonical pages |  |  |  | Indexed eligible canonical URLs | Eligible canonical indexable URLs | Search Console/Bing coverage reconciled to sitemap |  |
| Non-brand impressions |  |  |  | Impressions matching the documented non-brand filter | N/A — count | Search Console query export; attach excluded brand variants |  |
| Non-brand clicks |  |  |  | Clicks matching the same non-brand filter | N/A — count | Search Console query export |  |
| Branded search growth |  |  |  | Current branded volume minus prior comparable branded volume | Prior comparable branded volume | Search Console; report impressions and clicks separately |  |
| Average click-through rate by page |  |  |  | Clicks for each canonical | Impressions for the same canonical | Search Console page export; do not average page percentages without weighting |  |
| Top 10 query coverage |  |  |  | Fixed monitored queries with a documented position from 1 through 10 | Fixed monitored queries with valid observations | Named rank source or documented Search Console method |  |
| Crawl errors |  |  |  | Failed crawl events or affected URLs by error class | Verified crawl events or eligible URLs; state which | Production logs and Search Console/Bing |  |
| Excluded-page reasons |  |  |  | URLs in each named exclusion reason | Total excluded discovered URLs | Search Console Page Indexing export |  |
| Core Web Vitals pass rate |  |  |  | URLs/groups graded Good with sufficient field data | URLs/groups with sufficient field data | Search Console CWV or CrUX; lab results are separate |  |
| Referring domains |  |  |  | Unique qualified linking root domains | N/A — count | Named link-data source with qualification rules |  |
| Qualified organic conversions |  |  |  | Organic sessions/users with an approved qualified event | Organic sessions/users eligible for that event; state unit | Privacy-approved analytics |  |

Branded-search growth is undefined when the prior denominator is zero or
unavailable. Report the raw current/prior volumes and mark growth
`Unavailable — insufficient sample` instead of dividing by zero.

### AEO KPIs

| KPI | Current | Prior | Change | Numerator | Denominator | Definition/source | State and interpretation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Featured snippet appearances |  |  |  | Fixed query observations where a Reinstate page owns the verified snippet | Eligible fixed queries tested | Manual or approved rank observation with locale/device/date |  |
| Question-query impressions |  |  |  | Impressions matching the documented question-query filter | N/A — count | Search Console query export |  |
| FAQ and troubleshooting entrances |  |  |  | Organic entrances to FAQ and troubleshooting canonicals | All organic entrances, when reporting share | Analytics; also report raw count |  |
| Snippet-oriented page CTR |  |  |  | Clicks to the declared snippet-oriented page set | Impressions for that same page set | Search Console |  |
| Direct-answer engagement |  |  |  | Sessions meeting the documented engagement rule on answer-first pages | Eligible sessions to those pages | Analytics; preserve the engagement definition |  |
| Copy-command rate from answer pages |  |  |  | Eligible `command_copy` events/sessions on answer pages | Eligible answer-page sessions; state event deduplication | Analytics event taxonomy |  |
| Support deflection |  |  |  | Support-intent sessions with a verified self-serve resolution | Support-intent sessions with an observed outcome | Explicit feedback/support instrumentation only |  |
| Cited answer accuracy in manual tests |  |  |  | Evaluated cited answers graded fully accurate | All evaluated cited answers | Fixed manual query rubric and evidence |  |

Do not infer support deflection from lower ticket volume, page views, time on
page, or command copies. If no outcome instrument exists, mark it
`Unavailable`.

### ASEO KPIs

| KPI | Current | Prior | Change | Numerator | Denominator | Definition/source | State and interpretation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| AI referral sessions |  |  |  | Sessions matching the versioned AI referrer/UTM rules | N/A — count | Privacy-approved analytics |  |
| AI referral conversion rate |  |  |  | AI-referral sessions with approved qualified action | AI-referral sessions eligible for the action | Analytics |  |
| Citation frequency in fixed test queries |  |  |  | Completed query-provider observations citing Reinstate | All completed in-scope query-provider observations | Fixed manual AI-query run |  |
| Cited URL distribution |  |  |  | Citations to each Reinstate canonical | All Reinstate citations in completed observations | Manual AI-query evidence |  |
| Factual accuracy |  |  |  | Mentioned/cited outputs graded fully accurate | All mentioned/cited outputs graded | Manual rubric |  |
| Brand inclusion without citation |  |  |  | Observations mentioning Reinstate without a citation | All observations mentioning Reinstate | Manual AI-query evidence |  |
| Third-party source mentions |  |  |  | Verified third-party pages or outputs mentioning Reinstate | N/A — count | Named search/link/manual evidence source |  |
| Crawler success rate |  |  |  | Successful ordinary responses to verified in-scope bot requests | Total verified in-scope bot requests | Production logs with current identity verification |  |
| Bot `403`, `429`, and `5xx` rate |  |  |  | Verified bot responses in each status class | Total verified in-scope bot requests | Production logs; retain each status count |  |
| Freshness of cited pages |  |  |  | Distinct cited canonicals reviewed after relevant release/fact change | Distinct cited Reinstate canonicals | Citation run plus page/release metadata |  |

AI referral analytics are partial because providers may not pass a distinct
referrer. Citation tests are samples, not population estimates. Record provider,
query, date, locale, signed-in state, product version, mention, citation, cited
URL, accuracy, competitor/result context, and corrective action.

### Product-quality guardrails

| Guardrail | Current | Prior | Change | Numerator | Denominator | Definition/source | State and decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Install success |  |  |  | Verified successful installs | Install attempts with an observed outcome | Immutable acceptance evidence or privacy-approved telemetry |  |
| Restore success |  |  |  | Verified same-vendor native resumes after restore | Restore attempts with an observed outcome | Immutable acceptance, support, or approved telemetry |  |
| Support issue rate |  |  |  | New in-scope support issues | Named active-user/install/restore population | Support system plus defined population |  |
| Compatibility regression rate |  |  |  | Previously passing matrix cells that failed retest | Previously passing cells retested this quarter | Immutable compatibility evidence |  |
| Documentation failure reports |  |  |  | Verified reports caused by incorrect/incomplete docs | N/A — count | Support/issue labels and reviewed reports |  |
| Outdated-page count |  |  |  | Pages failing release/compatibility freshness review | Substantive indexable pages reviewed | Content inventory and release diff |  |

Clicks, downloads, command copies, and page views do not prove install or
restore success. Visibility may grow while product guardrails regress; report
both without hiding the conflict.

## 2. Information architecture review

| Question | Evidence/method | Finding | Decision | Owner/due | Acceptance criteria |
| --- | --- | --- | --- | --- | --- |
| Can every indexable canonical be reached within the defined crawl-depth budget? | Built-site crawl |  |  |  |  |
| Do hubs expose the authoritative docs, guides, integrations, use cases, comparisons, and research? | Link graph and navigation review |  |  |  |  |
| Does one canonical own each important intent? | Query-to-page map |  |  |  |  |
| Are docs task references distinct from outcome guides and product pages? | Content inventory |  |  |  |  |
| Are orphaned, thin, duplicate, or dead-end pages present? | Crawl and analytics |  |  |  |  |
| Do breadcrumbs, canonicals, sitemap, and visible hierarchy agree? | Rendered HTML and schema |  |  |  |  |
| Are preview, temporary, draft, API, 404, and redirected routes excluded correctly? | Build, robots, sitemap, status crawl |  |  |  |  |

Record proposed URL changes with redirect, internal-link, sitemap, canonical,
structured-data, RSS, `llms.txt`, IndexNow, backlink, and external-profile
implications.

## 3. Content pruning and consolidation

Pruning is an evidence-based content decision, not deletion of low-traffic new
pages by reflex. Review product relevance, unique intent, accuracy, links,
citations, conversions, maintenance cost, and time to mature.

| Canonical | Purpose/intent | Published/meaningfully updated | Product truth/current release | Organic evidence | AI citation/referral evidence | Backlinks | Qualified actions | Overlap/unique value | Decision | Destination/redirect | Owner/due | Acceptance criteria |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  | Keep / Update / Merge / Redirect / Remove / Noindex |  |  |  |

Decision requirements:

- **Keep:** name the unique intent/value and next review date.
- **Update:** identify the stale claim, query gap, evidence, or answer defect.
- **Merge:** select one surviving canonical, map every useful section and
  backlink, and avoid two near-duplicate pages.
- **Redirect:** require a genuinely equivalent destination and update internal
  links, canonical, sitemap, schema, RSS, and IndexNow deletion/addition plan.
- **Remove:** require no suitable replacement, return an intentional status,
  remove internal references, and record lost-link risk.
- **Noindex:** state why the page must remain public but should not be indexed;
  do not use it to hide a quality problem indefinitely.

Before merging or removing, attach Search Console, Bing, analytics, referral,
backlink, and AI-citation evidence—or mark each source unavailable. Record a
rollback path and validate the production result.

## 4. Schema and technical-system review

| Surface | Current schema/behavior | Visible-content match | Validation evidence | Debt/risk | Decision | Owner/due | Acceptance criteria |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Homepage entities |  |  |  |  |  |  |  |
| Product/evaluation pages |  |  |  |  |  |  |  |
| Docs |  |  |  |  |  |  |  |
| Guides/HowTo |  |  |  |  |  |  |  |
| Blog/Article |  |  |  |  |  |  |  |
| FAQ |  |  |  |  |  |  |  |
| Breadcrumbs |  |  |  |  |  |  |  |
| Open Graph/Twitter cards |  |  |  |  |  |  |  |
| Sitemap/robots/canonical policy |  |  |  |  |  |  |  |

Confirm valid JSON, absolute production URLs, correct dates, route-specific
images and alt text, one canonical, visible-schema agreement, and no fake
ratings, reviews, pricing, customer counts, awards, benchmarks, or unsupported
platform claims.

## 5. Product positioning and ecosystem review

| Claim/dimension | Current canonical wording | Released evidence | External-profile consistency | Search/AI result observation | Conflict or opportunity | Decision/owner/due/acceptance |
| --- | --- | --- | --- | --- | --- | --- |
| Product definition/category |  |  |  |  |  |  |
| Primary audience/use case |  |  |  |  |  |  |
| Same-vendor native resume |  |  |  |  |  |  |
| Encryption and user-owned storage |  |  |  |  |  |  |
| Supported agents/platforms |  |  |  |  |  |  |
| Open-source/license/account requirement |  |  |  |  |  |  |
| Current versus planned capabilities |  |  |  |  |  |  |
| Non-affiliation and comparison fairness |  |  |  |  |  |  |

Competitor/result observations require query, locale, date, signed-in state,
URL, and archived evidence where permitted. Do not speculate about competitor
internals, traffic, users, security, or roadmap.

## 6. Crawler, search, AI, and referral review

- [ ] Search-bot access and GPTBot training policy were reviewed separately.
- [ ] Verified production logs were used for crawler rates; spoofable
      user-agent-only observations were excluded.
- [ ] `403`, `429`, `5xx`, redirect loops, WAF challenges, and excessive bot
      traffic were reviewed by bot family.
- [ ] Search Console and Bing indexing, exclusions, canonicals, security, and
      manual actions were reviewed by authorized operators.
- [ ] IndexNow submissions were limited to reviewed changed URLs and did not
      include unchanged deploys.
- [ ] The fixed AI-query inventory, provider set, locale, and rubric were
      versioned before comparison.
- [ ] AI referrer/UTM channel rules and qualified events were versioned.
- [ ] Cited pages were checked against current release and compatibility facts.
- [ ] External third-party mentions and backlinks were checked for factual
      accuracy and quality.

| Finding | Channel/provider/bot | Numerator | Denominator | Evidence/state | Risk | Task | Owner/due | Acceptance criteria |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |

## 7. Backlink quality review

| Domain/page | New/lost/existing | Target canonical | Editorial relevance | Link placement/anchor | Factual accuracy | Risk/quality decision | Outreach/correction | Owner/due/acceptance |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |

Report referring-domain counts separately from quality. Do not buy links,
request deceptive anchors, or treat automated scraper pages as authority.
Record lost-link causes before changing a canonical.

## 8. Technical debt register

| ID | Component | Debt and user/discovery impact | Evidence | Risk if deferred | Proposed task | Owner | Due | Acceptance criteria | Status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  | Open |

Include crawl/build tooling, performance, accessibility, content schemas,
analytics privacy, log retention, bot verification, release evidence,
compatibility freshness, and editorial maintenance. Do not disguise a product
or security defect as SEO debt.

## 9. Primary research plan

| Research asset/question | User value and linkable claim | Methodology | Required raw evidence | Privacy/security review | Reproducibility plan | Owner | Due | Publication acceptance criteria |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |

Potential work includes session-format maps, compatibility reports,
cross-device restoration benchmarks, threat models, path-mapping tools, and
adapter references. Do not publish a benchmark or ecosystem report before the
method, fixtures, environment, version matrix, limitations, and downloadable
or auditable evidence are ready. Accuracy outranks calendar cadence.

## 10. Next-quarter content and experiment plan

| Initiative | Query/user problem | Baseline | Hypothesis | Change | Primary KPI | Product guardrail | Decision date | Owner | Due | Acceptance criteria |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |  |  |

Experiments must have a fixed baseline, one primary outcome, a product-quality
guardrail, minimum evidence condition, and stop/keep decision. Avoid running
ranking experiments that weaken product truth, accessibility, privacy,
security, or page usefulness.

## 11. Consolidated action register

| ID | Priority | Finding/decision | Exact engineering/editorial/external task | Owner | Due date | Acceptance criteria | Validation and evidence | Dependency/access owner | Status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  | Open |

Every Critical or High item needs a named owner, due date, testable acceptance
criteria, and validation evidence. External actions need the authorized access
owner. A task blocked on access remains open and uses `Unavailable`; it does
not become a pass.

## 12. Validation and sign-off

- [ ] All three monthly audits and every weekly report in the quarter are
      accounted for or each gap is disclosed.
- [ ] Every KPI is populated, explicitly unavailable, or not applicable with a
      reason.
- [ ] Every rate has raw operands and a reproducible source/filter.
- [ ] Search, analytics, crawler, backlink, AI, support, and release evidence
      identify access owner and review date.
- [ ] Release and compatibility claims link immutable evidence records.
- [ ] Pruning decisions include redirect/indexing/link/evidence effects.
- [ ] Product guardrails were considered before declaring visibility success.
- [ ] Repository checks and production checks are not conflated.
- [ ] No secret or sensitive external export is committed.

| Sign-off field | Value |
| --- | --- |
| Review status | Complete / Complete with gaps / Blocked |
| Critical findings open |  |
| High findings open |  |
| Material unavailable evidence |  |
| Pages kept/updated/merged/redirected/removed/noindexed |  |
| Research approved |  |
| Next weekly report |  |
| Next monthly audit |  |
| Next quarterly review |  |
| Approver |  |

## Repository references

- [Weekly report template](weekly-report-template.md)
- [Monthly audit template](monthly-audit-template.md)
- [Release evidence template](release-evidence-template.md)
- [Operations and external-access runbook](operations.md)
- [Fixed AI-search query baseline](ai-search-query-baseline.md)
- [Content roadmap](content-roadmap.md)
- [SEO, AEO, and ASEO playbook](seo-aeo-aseo-playbook.md)
