# SEO, AEO, and AI-search operations

This runbook turns Reinstate's discoverability implementation into a repeatable
production practice. It does not claim that any search-engine account,
production log source, analytics property, or indexing request has been
configured or verified.

## Responsibility boundary

| Activity | This branch can prepare or test | Requires an account or production access |
| -------- | ------------------------------- | ----------------------------------------- |
| Build, metadata, sitemap, robots, schema, and link checks | Yes | No |
| Public HTTP smoke tests with crawler user-agent strings | Yes | No |
| Google Search Console domain verification and URL Inspection | No | Yes |
| Bing Webmaster Tools verification and URL Inspection | No | Yes |
| IndexNow diff planning, validation, proof route, and mocked submission tests | Yes | No |
| IndexNow key provisioning and production submission | No | Yes |
| Plausible account/site configuration and production environment values | No | Yes |
| CDN/WAF policy inspection and verified crawler log analysis | No | Yes |
| AI-search query testing in provider interfaces | Template only | Yes; run manually |

Never mark an external action complete from repository state alone. Attach a
dated screenshot, export, response log, or ticket when the action is performed.

## Before touching an external console

1. Confirm the current release and product claims against `README.md`,
   `CHANGELOG.md`, `docs/compatibility.md`, and
   `website/src/data/product.ts`.
2. Run:

   ```sh
   npm --prefix website test
   npm --prefix website run build
   ```

3. Verify production returns `200` for:

   ```text
   https://reinstate.dev/
   https://reinstate.dev/robots.txt
   https://reinstate.dev/sitemap-index.xml
   ```

4. Confirm the sitemap contains canonical, indexable URLs only. Preview, API,
   error, redirect, draft, and `noindex` URLs must not appear.
5. Record the Git commit, deployed version, timestamp, operator, and production
   URL used for the checks.

## Google Search Console

### Domain verification

Domain-property verification is an account- and DNS-level action and is not
automatable in this branch.

1. Sign in to the project owner's Search Console account.
2. Create a domain property for `reinstate.dev`.
3. Add the exact DNS TXT value provided by Google through the authoritative DNS
   provider.
4. Wait for DNS propagation, then complete verification in Search Console.
5. Record the property owner, verification date, evidence location, and DNS
   change reference without copying private account details into this repo.

`PUBLIC_GOOGLE_SITE_VERIFICATION` supports an optional HTML meta token for a
URL-prefix property. It is not a substitute for DNS verification of a domain
property. The token is public by design; never place a private API credential
in a `PUBLIC_*` variable.

### Sitemap submission

After property verification:

1. Submit `https://reinstate.dev/sitemap-index.xml`.
2. Confirm Search Console can fetch it successfully.
3. Record the submitted URL, submission time, discovered URL count, and any
   parsing or fetch error.
4. Recheck after a meaningful route or canonical change. Do not repeatedly
   resubmit an unchanged sitemap.

### Launch URL inspection

Inspect these launch-critical canonicals:

```text
https://reinstate.dev/
https://reinstate.dev/docs
https://reinstate.dev/docs/getting-started
https://reinstate.dev/guides
https://reinstate.dev/guides/move-a-coding-agent-session-from-mac-to-windows
https://reinstate.dev/guides/sync-claude-code-sessions-across-devices
https://reinstate.dev/guides/sync-codex-sessions-across-devices
https://reinstate.dev/guides/use-cloudflare-r2-for-coding-agent-session-storage
https://reinstate.dev/guides/use-s3-for-coding-agent-session-storage
https://reinstate.dev/blog
https://reinstate.dev/blog/why-git-does-not-sync-coding-agent-sessions
https://reinstate.dev/integrations
https://reinstate.dev/integrations/claude-code
https://reinstate.dev/integrations/codex
https://reinstate.dev/compatibility
https://reinstate.dev/security
https://reinstate.dev/about/reinstate
https://reinstate.dev/open-source
https://reinstate.dev/roadmap
https://reinstate.dev/research
https://reinstate.dev/changelog
https://reinstate.dev/privacy
https://reinstate.dev/use-cases
https://reinstate.dev/use-cases/desktop-and-laptop
https://reinstate.dev/use-cases/encrypted-session-backup
https://reinstate.dev/use-cases/work-and-personal-computers
https://reinstate.dev/use-cases/macos-and-windows
https://reinstate.dev/compare
```

For each URL, record:

- live-test status and rendered-HTML availability;
- user-declared and Google-selected canonical;
- robots/indexing status;
- last crawl and crawl result, if available;
- discovered sitemap and referring-page information;
- enhancement or structured-data issues; and
- whether indexing was requested.

Request indexing only for launch-critical pages or meaningful corrections, not
for every deployment. Search Console actions and results require the verified
account; this repository cannot assert that they succeeded.

### Google generative-AI eligibility and reporting

Google's current guidance treats generative-AI visibility as an extension of
the core Search index, not a separate markup system. After verification, an
authorized owner should:

1. review the Search Console setting that controls inclusion in Search
   generative-AI features;
2. confirm the intended setting for `reinstate.dev`;
3. review the Generative AI performance report when it is available;
4. record its date range, query/page filters, impressions, clicks, and source;
   and
5. mark the report `Unavailable` when the property, feature, or sample is not
   available instead of inferring AI visibility from ordinary Search totals.

Do not treat `llms.txt`, special AI markup, or extra schema as a Google ranking
control. Reinstate publishes `llms.txt` as a concise optional resource for
systems that may choose to use it; Google states that Google Search ignores it.

## Bing Webmaster Tools and IndexNow

### Webmaster verification

1. Add `https://reinstate.dev` to the owner's Bing Webmaster Tools account or
   use an approved import from Search Console.
2. Complete the verification method selected in Bing.
3. If HTML meta verification is selected, set
   `PUBLIC_BING_SITE_VERIFICATION` to the provided public token in the
   production deployment environment and redeploy. The layout emits
   `msvalidate.01` only when the value is present.
4. Submit `https://reinstate.dev/sitemap-index.xml`.
5. Inspect the same launch URLs and record crawl, canonical, and indexing
   findings.

These are account-level actions. This branch only provides the optional
verification hook and cannot verify account ownership or submission results.

### IndexNow readiness

The repository is ready to plan and submit IndexNow changes, but no production
key has been provisioned and no submission has been made. Readiness does not
mean that a search engine has accepted or indexed any URL.

Before provisioning or changing production behavior, recheck the
[official IndexNow protocol documentation](https://www.indexnow.org/documentation)
and [current operational FAQ](https://www.indexnow.org/faq) for provider limits
or response-policy changes.

The implementation has these safety boundaries:

- plan generation is the default and never reads `INDEXNOW_KEY` or submits;
- submissions require a separate, reviewed plan file plus the explicit
  `--submit` flag;
- only canonical HTTPS URLs on `reinstate.dev` are accepted;
- the key is accepted only from the server/operator environment, never from a
  command-line option, plan file, `PUBLIC_*` variable, or structured log;
- a dynamic `/{key}.txt` server route returns the proof only for the exact
  configured key and otherwise returns `404`;
- the submitter verifies that public proof before posting any URL;
- batches contain 100 URLs by default and never more than 1,000;
- network errors, `429`, and `5xx` responses retry at most three times by
  default with bounded exponential delay; `Retry-After` is honored;
- `400`, `403`, and `422` responses are treated as permanent for that run;
- `200` and `202` are logged as accepted by the API, not as indexed; and
- provider or network failures are logged as soft failures and do not make the
  submit command fail the deployment. Operators must still review the final
  JSON result and open a follow-up when `ok` is `false`.

CI runs fixture tests with mocked HTTP responses and a no-change dry run against
the local sitemap. It has no key and cannot submit to IndexNow.

### IndexNow release procedure

Assign an operator before enabling submissions. That operator owns key rotation,
plan review, response evidence, and follow-up. Store the key in the approved
secret manager and in Vercel's server-only production environment:

```sh
cd website
npx vercel env add INDEXNOW_KEY production
```

Generate a key of 8–128 letters, numbers, or dashes inside the approved secret
manager. Do not paste it into Git, `.env.example`, an issue, a shell history
entry, or a `PUBLIC_*` variable. IndexNow ownership proof is public by protocol,
but the application deliberately does not expose a predictable key listing or
log the proof URL.

Before deployment, build the proposed release and compare its generated sitemap
with the currently deployed sitemap:

```sh
cd website
npm run build
INDEXNOW_RUN_DIR="$(mktemp -d)"
cp indexnow.changes.example.json "$INDEXNOW_RUN_DIR/changes.json"
npm run indexnow -- \
  --current dist/client/sitemap-index.xml \
  --previous https://reinstate.dev/sitemap-index.xml \
  --changes "$INDEXNOW_RUN_DIR/changes.json" \
  --output "$INDEXNOW_RUN_DIR/plan.json"
```

Edit the temporary changes file before generating the plan. Its fields mean:

```json
{
  "updated": ["/changelog"],
  "deleted": ["/removed-page"],
  "recanonicalized": [
    {
      "from": "/old-canonical",
      "to": "/new-canonical"
    }
  ]
}
```

`updated` URLs must exist in the new sitemap. `deleted` and
`recanonicalized.from` URLs must not exist there.
`recanonicalized.to` must exist there. Added URLs, removed URLs, and changed
`lastmod` values are collected automatically. Because not every generated
sitemap entry has a meaningful `lastmod`, declare materially revised existing
pages under `updated`.

For the first rollout only, if production has never exposed a sitemap, replace
the `--previous` value with `indexnow.previous-empty.xml` after verifying that
there is no previous canonical inventory to preserve. Never use the empty
baseline for an ordinary release: it would make every current URL appear new
and would miss removals.

Review the entire secret-free plan before deployment:

- `site` is exactly `https://reinstate.dev`;
- every `urlList` item should be notified and its `reasons` are accurate;
- removed or recanonicalized sources really return the intended `404`, `410`,
  or redirect after deployment;
- destinations are canonical, indexable, and present in the new sitemap;
- `planDigest`, commit, release, reviewer, and plan path are recorded; and
- the planned set does not contain unchanged URLs.

Deploy the reviewed commit. Confirm the new sitemap, deletions, redirects, and
destinations are live before submission. Load `INDEXNOW_KEY` into the operator
process from the approved secret manager without echoing it, then run:

```sh
cd website
npm run indexnow -- \
  --plan "$INDEXNOW_RUN_DIR/plan.json" \
  --submit
```

Plans expire after 48 hours by default. Regenerate and review an expired plan
instead of bypassing the age check. The submitter preflights the production key
proof and then posts to the global IndexNow endpoint. It logs timestamps, plan
digest, batch number, URL count, attempt, response status, and final summary,
but not the key, proof URL, request body, or provider response body.

The submit process exits successfully for a provider/network soft failure so it
can run after deployment without rolling the release back. Completion therefore
requires inspecting the final record:

- `ok: true` means all batches received HTTP `200` or `202`;
- `ok: false, softFailed: true` requires an operator-owned follow-up;
- a configuration, plan-integrity, stale-plan, or unsafe-URL error exits
  nonzero and must be corrected; and
- acceptance is not a guarantee of crawling, indexing, ranking, or AI citation.

Keep the response log with the launch evidence. Retry only after diagnosing the
status; never regenerate a broad plan merely to resend unchanged URLs. Rotate a
compromised key in the secret manager and Vercel, redeploy the proof route, and
verify the replacement before the next submission.

Treat the playbook's
[Bing and IndexNow guidance](seo-aeo-aseo-playbook.md#58-bing-and-indexnow) as
the implementation policy.

## Optional Plausible analytics

Analytics is opt-in. When both variables are blank, the site loads no Plausible
script:

```dotenv
PUBLIC_PLAUSIBLE_DOMAIN=
PUBLIC_PLAUSIBLE_SCRIPT_URL=
```

To enable it, an authorized operator must:

1. Create or select the Plausible site for the production domain.
2. Set `PUBLIC_PLAUSIBLE_DOMAIN` to the domain configured in that account.
3. Set `PUBLIC_PLAUSIBLE_SCRIPT_URL` to the script URL supplied by that
   Plausible deployment; do not guess or copy a stale URL.
4. Configure the values in the production deployment environment and redeploy.
5. Confirm the script loads successfully, respects the chosen privacy policy,
   and records a production page view.
6. Confirm test, preview, and local traffic are excluded or clearly segmented.
7. Record the account owner, enablement date, data-retention decision, and
   privacy-notice review.

Both values are public configuration, not secrets. Plausible API keys or
account credentials must never use `PUBLIC_*`.

### Analytics privacy boundary

Do not send:

- coding-agent session content, prompts, or tool output;
- source code, file contents, repository-private identifiers, or branch names;
- bucket names, endpoints, credentials, passphrases, tokens, or email
  addresses;
- waitlist form values or user-provided free text; or
- full URLs or custom properties containing sensitive query parameters.

The existing click helper sends only a controlled event name plus page
`location` and an optional declared `target`. No click event is emitted unless
an element has a reviewed `data-analytics-event` or its URL matches one of the
fixed RSS, release-asset, issue-form, or contribution-link rules. Integration,
storage-guide, and security-document events are emitted from exact route
matches. Review every proposed event or rule so its name and target cannot
contain user data.

### Implemented event taxonomy

| Event | Exact implementation |
|---|---|
| `install_command_copy` | Successful attempt to copy the homepage install command |
| `github_click` | A declaratively marked primary repository link |
| `docs_getting_started` | A declaratively marked getting-started entry link |
| `integration_view` | Exact Claude Code or Codex integration route load |
| `storage_guide_view` | Exact S3 or Cloudflare R2 storage-guide route load |
| `waitlist_submit` | Server-confirmed successful waitlist submission |
| `download_click` | Link whose path contains `/releases/download/` |
| `changelog_subscribe` | Link to the canonical `/rss.xml` feed |
| `issue_report_click` | Link whose path contains `/issues/new` |
| `contribute_click` | Link to the repository `CONTRIBUTING.md` |
| `security_doc_view` | Exact security overview or security-model route load |
| `command_copy` | A future command control explicitly marked with this event |

The generic `command_copy` event exists for non-install command controls. Do
not add it to the install control as well, because that would double count one
action. Page-view events intentionally use exact routes; hub pages and unrelated
URLs do not silently enter the funnel. Tests in
`website/src/lib/analytics.test.ts` pin the event inventory and all automatic
route/link classifications.

### AI-referral classification and reporting

The browser classifier emits only the controlled `ai_referral_channel`
vocabulary below. It never sends a raw referrer, query, URL, or UTM value as a
custom property.

| Channel | Exact referrer hosts | Exact accepted UTM tokens |
| ------- | -------------------- | ------------------------- |
| `chatgpt` | `chatgpt.com`, `chat.openai.com`, `openai.com`, including subdomains | `chatgpt`, `openai`, `oai-search` |
| `perplexity` | `perplexity.ai`, including subdomains | `perplexity` |
| `microsoft-copilot` | `copilot.microsoft.com`, including subdomains | `microsoft-copilot`, `ms-copilot`, `copilot` |
| `google-gemini` | `gemini.google.com`, `bard.google.com`, including subdomains | `google-gemini`, `gemini` |
| `google-ai-features` | none inferred from a generic Google referrer | `google-ai`, `ai-overview`, `ai-overviews`, `ai-mode` |

Matching is case-normalized but exact. Arbitrary substrings and generic Google
search referrals are deliberately not guessed.

`ai_referral_channel` is attached to approved custom actions only. Therefore:

- use Plausible's native referrer and UTM filters for total AI-referred
  sessions, including visits with no custom action;
- use the controlled property for action-assisted AI referral analysis;
- define an engaged session using the analytics provider's documented
  engagement measure and record that definition with the report;
- report repeat visitors only when the configured privacy model and provider
  support a defensible repeat measure; otherwise mark it `Unavailable`; and
- define assisted conversion as an approved action occurring in an
  AI-referred visit, never as causal credit for the conversion.

Every report must include the numerator, denominator, filter, date window, and
analytics configuration version. Production Plausible configuration remains an
external launch gate.

## Production crawler and WAF checks

`robots.txt` currently distinguishes search discovery from model training:
`OAI-SearchBot` and `PerplexityBot` are allowed, while GPTBot has a separate
policy. A robots rule does not prove that the CDN or WAF permits a crawler.

### Read-only post-deployment smoke test

Run the automated smoke test from at least one external network after deploying
an immutable candidate and again after promoting production:

```sh
npm --prefix website run check:production-discovery
```

The default target is exactly `https://reinstate.dev`. To check an immutable
public HTTPS deployment before promotion, acknowledge the different origin
explicitly:

```sh
npm --prefix website run check:production-discovery -- \
  --base-url https://DEPLOYMENT.example \
  --allow-non-production
```

The request origin may differ for an immutable deployment, but canonical,
Open Graph, robots-sitemap, and sitemap URLs must still point to
`https://reinstate.dev`. Non-production targets require the explicit flag and
must be public HTTPS origins. Credentials, URL paths, queries, fragments,
localhost, and private-network targets are refused. The checker sends no
cookies, authorization, form data, or request bodies.

The checker:

- uses only idempotent `GET` and `HEAD` requests;
- verifies every sitemap canonical with `GET` and every launch-critical
  canonical with `HEAD`;
- requires one matching production canonical and an indexable robots policy;
- follows the sitemap index with a maximum of 10 documents and 100 URLs;
- rejects duplicate, non-production, preview, API, and error sitemap entries;
- verifies `robots.txt`, its production sitemap directive, wildcard exclusions,
  and explicit `OAI-SearchBot` and `PerplexityBot` groups;
- verifies both crawler user agents receive canonical HTML for `/` and `/docs`
  plus valid robots and sitemap responses rather than a challenge page;
- verifies every unique Open Graph asset with `HEAD` and a bounded range `GET`,
  including `image/png` and the PNG's actual 1200×630 dimensions;
- requires a fixed nonexistent route to return `404` for both `GET` and `HEAD`;
- limits concurrency to four, each attempt to 10 seconds, response bodies to
  fixed sizes, and transient network/`429`/`5xx` retries to two attempts by
  default; and
- exits nonzero on a finding without changing the deployment, submitting a URL,
  requesting indexing, or updating an external account.

CLI tuning cannot exceed eight concurrent requests, three attempts, or a
30-second per-attempt timeout. Keep the defaults unless a recorded incident
justifies a temporary change. Retries honor `Retry-After` up to a bounded delay.

Unit tests use injected mock responses and make no external requests. Do not add
the production command to ordinary CI: it is an explicit post-deployment
observation whose target, release, operator, and time must be recorded.

### Evidence artifact handling

Every run writes timestamped JSON under the ignored
`website/artifacts/discovery/` directory and prints the exact path. Use
`--output` to select another local destination:

```sh
npm --prefix website run check:production-discovery -- \
  --output artifacts/discovery/release-candidate.json
```

The artifact contains the target origin, bounded configuration, timestamps,
durations, request methods, paths, simulated user-agent names, attempt statuses,
content types, fixed finding codes, and coverage totals. It deliberately omits
response bodies, cookies, authorization, arbitrary response headers, IP
addresses, and provider error text. Review the JSON before sharing it, compute
and record its SHA-256 digest, and attach it to the approved release evidence
store or ticket. Do not commit generated evidence to Git. Apply the project's
evidence-retention and access policy rather than inventing a new retention
period in this runbook.

Record the release tag and commit, immutable or production target, operator,
UTC run time, result, artifact location, and digest. A passing result proves
only what that public endpoint returned during the run. A failed result blocks
discoverability sign-off until it is explained and rerun; it does not roll back
or otherwise mutate the release.

User-agent strings are spoofable. These requests test the public response policy
only; they do not prove that a request came from OpenAI or Perplexity.

An operator with production log and WAF access must separately review:

- timestamp, path, raw user-agent, verified bot family, status, bytes, latency,
  cache result, and referrer when available;
- challenge, block, or rate-limit actions;
- repeated `403`, `429`, and `5xx` responses;
- crawler loops or excessive requests;
- sitemap fetches and canonical-page fetches, including an explicit finding
  when no sitemap request appears in the review window; and
- attempts to crawl `/api/` or `/preview/`.

Production log access, bot-IP verification, and WAF changes are not automatable
or completed in this branch.

## Rendered-browser quality gate

After a production build, run:

```sh
npm --prefix website run check:lighthouse
```

The command starts the repository's static preview server, opens five
representative routes in headless Chrome, and writes full JSON evidence under
`website/artifacts/lighthouse/`. CI uploads those reports even when the gate
fails. The gate covers rendered SEO, accessibility, best practices,
performance, LCP, CLS, named controls, labels, headings, language, image text
alternatives, canonicals, and descriptions.

Chrome must be installed or `CHROME_PATH` must point to a compatible executable.
CI's Ubuntu runner supplies Chrome. A local machine without Chrome can install
Chrome Headless Shell into an ignored cache with the Puppeteer browsers CLI and
set `CHROME_PATH` for the run; do not commit the downloaded browser.

Treat the result as a lab regression signal. It does not measure field INP,
prove 75th-percentile Core Web Vitals, or replace keyboard, zoom, screen-reader,
and real-device testing. Record production PageSpeed/Search Console field data
only when a sufficient sample exists.

For Perplexity WAF rules, verify the user-agent against the current IP ranges
published in Perplexity's crawler documentation. Do not hard-code a copied IP
list in this repository or allow traffic solely because it presents a
spoofable user-agent string.

## Launch-day record

| Check | Evidence | Owner | Time (UTC) | Result | Follow-up |
| ----- | -------- | ----- | ---------- | ------ | --------- |
| Production build/version |  |  |  | Not run |  |
| Post-deploy discoverability smoke |  |  |  | Not run |  |
| Robots and sitemap fetch |  |  |  | Not run |  |
| Google property/sitemap/generative-AI setting |  |  |  | Not run |  |
| Bing property/sitemap |  |  |  | Not run |  |
| IndexNow plan/submission |  |  |  | Not run |  |
| Launch URL Inspection |  |  |  | Not run |  |
| OAI-SearchBot smoke test |  |  |  | Not run |  |
| PerplexityBot smoke test |  |  |  | Not run |  |
| Production WAF/log review |  |  |  | Not run |  |
| Analytics/privacy review |  |  |  | Not run |  |
| AI-query baseline |  |  |  | Not run |  |

## Reporting cadence

### Weekly during launch

Record:

- deployments and changed canonical URLs;
- sitemap fetch or parsing changes;
- new indexing exclusions and crawl errors;
- broken links and SEO CI failures;
- top organic landing pages and qualified actions, when analytics exists;
- crawler `403`, `429`, and `5xx` observations, when logs exist;
- content published or materially revised; and
- owner and due date for every Critical or High finding.

Use `Unavailable — access not configured` instead of zero when a data source is
missing.

### Monthly

Complete [the monthly audit template](monthly-audit-template.md), attach Search
Console and Bing exports when available, and run the
[fixed AI-search query baseline](ai-search-query-baseline.md) manually.

Compare:

- branded and non-branded queries, impressions, clicks, CTR, and pages;
- indexed canonicals and exclusion reasons;
- landing-page engagement and qualified actions;
- question-query and troubleshooting performance;
- AI mentions, citations, cited URLs, and factual accuracy;
- crawler success and failure rates from verified logs;
- page freshness against releases and compatibility changes; and
- Core Web Vitals field data when traffic is sufficient.

Never infer unavailable numbers or claim ranking, citation, or conversion
improvements without comparable evidence.

## Official references

- [Google Search Essentials](https://developers.google.com/search/docs/essentials)
- [Google guidance for generative-AI search](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)
- [Astro sitemap integration](https://docs.astro.build/en/guides/integrations-guide/sitemap/)
- [OpenAI publishers and developers FAQ](https://help.openai.com/en/articles/12627856-publishers-and-developers-faq)
- [OpenAI ChatGPT Search](https://help.openai.com/en/articles/9237897)
- [Perplexity crawler documentation](https://docs.perplexity.ai/docs/resources/perplexity-crawlers)
