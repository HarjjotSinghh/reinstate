# Weekly SEO, AEO, AI-search, and product-quality report

Copy this template into a dated, repository-tracked report. Do not edit the
template to record a real week. A completed report must distinguish
repository-verifiable evidence from account-level, production, or manual
evidence.

Never turn missing access into `0`, “no change,” or “pass.” Never report a rate
without its numerator and denominator. Never infer install or restore success
from a click.

## Reporting identity

| Field | Value |
| --- | --- |
| Week, in ISO format |  |
| Observation window, including time zone |  |
| Comparable prior window |  |
| Report author |  |
| Reviewer/approver |  |
| Production deployment commit(s) |  |
| Public Reinstate release/tag |  |
| Previous weekly report |  |
| Monthly audit containing this week |  |
| Evidence directory |  |

List partial weeks, launch days, incidents, analytics changes, bot-policy
changes, and other comparison breaks before interpreting a delta.

## Evidence and `Unavailable` semantics

Use one of these exact states:

- `Available` — the source was accessed, the observation window is complete,
  and the evidence location is recorded.
- `Partial — <reason>` — some data exists, but coverage, sampling, retention,
  verification, or the time window is incomplete.
- `Unavailable — access not configured` — the required account, export, log,
  or permission has not been connected.
- `Unavailable — provider does not expose` — the provider has no usable
  measurement for this definition.
- `Unavailable — insufficient sample` — the source exists but cannot support
  the requested rate or comparison.
- `Unavailable — no comparable prior window` — current evidence exists, but a
  defensible change cannot be calculated.
- `Not applicable — <reason>` — the metric genuinely does not apply. This is
  not a substitute for missing access.

An empty cell means **not reviewed**, not zero. A numeric zero is valid only
when a named source was queried for the complete window and its evidence shows
zero. For every count, put `N/A — count` in the denominator column. For every
rate, share, or pass percentage, record both raw operands.

## External-access disclosure

Repository tests prove only repository and generated-build behavior. Record
who supplied every external observation.

| Evidence source | Required access | Access owner/operator | State | Export/review time | Coverage/window | Repository evidence path or reason unavailable |
| --- | --- | --- | --- | --- | --- | --- |
| Google Search Console | Verified domain property |  |  |  |  |  |
| Bing Webmaster Tools | Verified site |  |  |  |  |  |
| IndexNow production submission log | Provisioned public proof and private key |  |  |  |  |  |
| Privacy-approved site analytics | Analytics account |  |  |  |  |  |
| CDN/WAF configuration | Production edge account |  |  |  |  |  |
| Production request logs | Log store |  |  |  |  |  |
| Core Web Vitals field data | Search Console or CrUX |  |  |  |  |  |
| Backlink/referring-domain source | Approved link-data source |  |  |  |  |  |
| Fixed AI-query test | Manual provider access |  |  |  |  |  |
| GitHub traffic/referrals | Repository admin |  |  |  |  |  |
| Install outcome evidence | Release/acceptance or privacy-approved telemetry |  |  |  |  |  |
| Restore outcome evidence | Acceptance/support or privacy-approved telemetry |  |  |  |  |  |
| Support and documentation reports | Issue/support system |  |  |  |  |  |

Do not commit API keys, verification secrets, raw IP addresses, credentials,
private search queries, user identifiers, session contents, source code,
bucket names, or unsanitized logs. Link a sanitized export or record only the
aggregate and the authorized operator.

## Executive summary

### Evidence-backed outcome

Write one paragraph covering:

- what shipped;
- the largest verified discovery or quality change;
- the largest unresolved risk;
- whether a release or compatibility fact changed;
- which measurements remain unavailable; and
- whether any Critical or High action missed its due date.

### Priority summary

| Priority | Open at start | Added | Closed with evidence | Open at end | Summary |
| --- | ---: | ---: | ---: | ---: | --- |
| Critical — crawling, indexing, security, outage, or materially false claim |  |  |  |  |  |
| High — strong discovery, answer, conversion, citation, or product-success impact |  |  |  |  |  |
| Medium |  |  |  |  |  |
| Low |  |  |  |  |  |

## 1. Deployments, releases, and canonical changes

| Date/time | Production commit or release | Changed canonical URLs | Added/updated/removed/recanonicalized | Sitemap/robots/schema implication | IndexNow plan/result | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |

- [ ] Every production deployment in the week is listed.
- [ ] New, meaningfully updated, deleted, and recanonicalized URLs are
      distinguished from unchanged deploys.
- [ ] IndexNow was not sent for unchanged URLs.
- [ ] Search Console/Bing inspections, if performed, identify the authorized
      operator and result; repository code does not claim account access.
- [ ] Release and compatibility changes link immutable evidence, not only
      mutable website copy.

## 2. Weekly technical-discovery checks

### Build, sitemap, links, and indexing

| Metric/check | Current | Prior | Numerator | Denominator | Source and exact evidence | State | Interpretation/action |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Deployment-related crawl errors |  |  | Affected crawl events or URLs | Verified crawl events or eligible canonical URLs; state which | Production logs, Search Console, or Bing |  |  |
| Broken internal links |  |  | Broken internal targets | Crawled internal targets | Built-site link check |  |  |
| SEO CI failures |  |  | Failed SEO checks | Executed SEO checks | CI run |  |  |
| Indexing change |  |  | Newly indexed minus newly de-indexed canonicals | Eligible canonical indexable URLs | Search Console/Bing export |  |  |
| Indexed canonical pages |  |  | Indexed eligible canonicals | Eligible canonical indexable URLs | Search Console coverage plus sitemap |  |  |
| Sitemap URL change |  |  | Added + updated + removed canonical URLs | Prior sitemap canonical URLs | Reviewed sitemap diff |  |  |
| Sitemap fetch/parsing failures |  |  | Failed fetches or parses | Observed sitemap fetches | Search Console/Bing |  |  |
| Excluded-page reasons |  |  | URLs per named exclusion reason | Total excluded discovered URLs | Search Console Page Indexing |  |  |
| Missing/soft-404/canonical mismatch |  |  | Affected canonical URLs | Canonicals tested | Production crawl and inspection |  |  |

Attach the current and prior URL sets when reporting an indexing count. A
sitemap URL is not “indexed” merely because it builds or was submitted.

### Production crawler and WAF observations

A user-agent string alone does not verify crawler identity. Use the provider's
current official verification method where one exists. Put unverified
user-agent observations in a separate row and exclude them from verified bot
rates.

| Bot/family | Search, user-fetch, or training purpose | Identity verification method/date | Successful ordinary responses | `403` | `429` | `5xx` | Redirect loop/challenge | Total verified requests | Success rate | Error rate | Evidence/state |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- | --- |
| Googlebot | Search |  |  |  |  |  |  |  |  |  |  |
| Bingbot | Search |  |  |  |  |  |  |  |  |  |  |
| OAI-SearchBot | Search discovery |  |  |  |  |  |  |  |  |  |  |
| ChatGPT-User | User-initiated fetch |  |  |  |  |  |  |  |  |  |  |
| PerplexityBot | Search discovery |  |  |  |  |  |  |  |  |  |  |
| GPTBot | Training policy tracked separately |  |  |  |  |  |  |  |  |  |  |
| Other/unverified UA | Do not include in verified rates | Unverified |  |  |  |  |  |  | Unavailable | Unavailable |  |

Formulas:

- crawler success rate = successful ordinary verified responses ÷ total
  verified in-scope requests;
- bot `403`/`429`/`5xx` rate = statuses in that class ÷ total verified in-scope
  requests; and
- the combined error rate must retain the individual status counts.

Document exclusions such as intentional `robots.txt` denial, preview routes,
health checks, obvious spoofing, or duplicate retries. A public user-agent
smoke test proves response behavior for that request; it does not prove bot
identity or CDN log coverage.

## 3. Weekly search and landing-page pulse

| Metric | Current | Prior comparable | Numerator | Denominator | Source/filter | State | Interpretation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Top organic landing pages |  |  | Organic entrances per page | N/A — ranked count | Analytics and Search Console |  |  |
| Non-brand impressions |  |  | Impressions for documented non-brand query filter | N/A — count | Search Console query export |  |  |
| Non-brand clicks |  |  | Clicks for the same non-brand filter | N/A — count | Search Console query export |  |  |
| Branded impressions/clicks |  |  | Impressions and clicks for documented brand variants | N/A — count | Search Console query export |  |  |
| CTR by priority page |  |  | Page clicks | Page impressions | Search Console page export |  |  |
| Question-query impressions/clicks |  |  | Impressions and clicks for documented question filter | N/A — count | Search Console query export |  |  |
| FAQ/troubleshooting entrances |  |  | Organic entrances to FAQ/troubleshooting | All organic entrances | Analytics |  |  |
| Qualified organic actions |  |  | Organic sessions with an approved qualified event | Organic sessions | Privacy-approved analytics |  |  |
| New referring domains |  |  | Newly observed qualified root domains | N/A — count | Named link-data source |  |  |

Record the exact brand variants, question filters, qualified events, and page
set. Do not compare windows with different filters without labeling the break.
Average position is diagnostic, not a guaranteed rank for every user.

## 4. Weekly AI-search and referral pulse

Manual query tests must use the fixed query inventory, stable provider set,
documented locale, date, signed-in state, and product version. Do not automate
queries against provider terms.

| Metric | Current | Prior comparable | Numerator | Denominator | Source/filter | State | Interpretation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Fixed-query brand inclusion |  |  | Completed query-provider observations mentioning Reinstate | All completed query-provider observations | Manual fixed-query run |  |  |
| Citation frequency |  |  | Completed observations citing Reinstate | All completed query-provider observations | Manual fixed-query run |  |  |
| Brand inclusion without citation |  |  | Mentions with no citation | All completed observations mentioning Reinstate | Manual fixed-query run |  |  |
| Factual accuracy |  |  | Mentioned/cited observations graded fully accurate | All mentioned/cited observations graded | Manual fixed-query run |  |  |
| AI referral sessions |  |  | Sessions matching documented AI referrer/UTM rules | N/A — count | Privacy-approved analytics |  |  |
| AI referral conversion rate |  |  | AI-referral sessions with approved qualified action | AI-referral sessions | Privacy-approved analytics |  |  |
| AI referral docs depth |  |  | AI-referral sessions reaching defined docs-depth threshold | AI-referral sessions | Privacy-approved analytics |  |  |
| AI referral install/GitHub/command-copy actions |  |  | Named actions from AI-referral sessions | AI-referral sessions | Privacy-approved analytics |  |  |
| Cited-page freshness |  |  | Distinct cited canonicals reviewed after the relevant release/change | Distinct cited Reinstate canonicals | Query run plus page metadata |  |  |

For every manual observation, retain provider, full query wording, result
status, mention, citation, cited URL, accuracy grade, date, locale, signed-in
state, product version, and corrective action. AI referrers are incomplete
evidence because providers may suppress or rewrite referrer data.

## 5. Weekly product-quality guardrails

Visibility is not a proxy for successful use.

| Guardrail | Current | Prior | Numerator | Denominator | Source | State | Release blocker or action |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Install success |  |  | Verified successful installs | Install attempts with an observed outcome | Acceptance evidence or privacy-approved telemetry |  |  |
| Restore success |  |  | Verified same-vendor native resumes after restore | Restore attempts with an observed outcome | Acceptance evidence, support, or privacy-approved telemetry |  |  |
| Support issue rate |  |  | New in-scope support issues | Defined active-user, install, or restore population | Issue/support source plus named population |  |  |
| Compatibility regression rate |  |  | Previously passing matrix cells that now fail | Previously passing cells retested | Immutable compatibility evidence |  |  |
| Documentation failure reports |  |  | Verified reports caused by incorrect/incomplete docs | N/A — count | Issue/support labels |  |  |
| Outdated-page count |  |  | Pages failing release/compatibility freshness review | Substantive indexable pages reviewed | Content inventory and release diff |  |  |

If outcome telemetry does not exist, mark install success, restore success, and
support rate `Unavailable`; do not substitute `install_command_copy`,
`download_click`, page views, or anecdotal messages.

## 6. Content shipped and freshness

| Canonical | New/material update/correction | Intent and direct answer | Release or evidence source | Internal links/schema/sitemap effect | Published commit/date | Owner | Follow-up measurement |
| --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |

- [ ] Every shipped page uses current product truth and same-vendor language.
- [ ] Commands were checked against released code.
- [ ] Updated dates reflect substantive review, not fake freshness.
- [ ] Thin, overlapping, stale, or orphaned pages discovered this week were
      added to the quarterly pruning inventory.
- [ ] New external claims have primary evidence.
- [ ] A changed release/compatibility fact links an immutable evidence record.

## 7. Finding and action register

| ID | Priority | Evidence-backed finding | Exact task | Owner | Due date | Acceptance criteria | Validation/evidence required | Status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  | Open |

Every Critical or High item requires a named owner, due date, and testable
acceptance criterion. “Investigate,” “improve SEO,” and “monitor” are not
acceptance criteria. Split external-account action, engineering change,
editorial correction, and product verification when different owners or
evidence are required.

## 8. Sign-off

- [ ] Repository build, SEO, link, performance, freshness, and relevant content
      checks were recorded.
- [ ] Production-only claims identify the authorized operator and evidence.
- [ ] Rates show numerator and denominator.
- [ ] `Unavailable`, `Partial`, zero, and not-applicable values follow this
      template's semantics.
- [ ] Search, AI referral, and product-success claims use comparable windows.
- [ ] Crawler identity was verified before inclusion in verified rates.
- [ ] No sensitive external export was committed.
- [ ] Every changed release or compatibility claim points to immutable
      evidence.

| Decision field | Value |
| --- | --- |
| Weekly report status | Complete / Complete with gaps / Blocked |
| Critical findings open |  |
| High findings open |  |
| Measurements unavailable |  |
| Next weekly report due |  |
| Next monthly audit due |  |
| Approver |  |

## Repository references

- [Monthly audit template](monthly-audit-template.md)
- [Quarterly review template](quarterly-review-template.md)
- [Release evidence template](release-evidence-template.md)
- [Operations and external-access runbook](operations.md)
- [Fixed AI-search query baseline](ai-search-query-baseline.md)
- [SEO, AEO, and ASEO playbook](seo-aeo-aseo-playbook.md)
