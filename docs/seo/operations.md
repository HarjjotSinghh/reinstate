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
https://reinstate.dev/guides/sync-claude-code-sessions-across-devices
https://reinstate.dev/guides/sync-codex-sessions-across-devices
https://reinstate.dev/blog
https://reinstate.dev/blog/why-git-does-not-sync-coding-agent-sessions
https://reinstate.dev/integrations
https://reinstate.dev/integrations/claude-code
https://reinstate.dev/integrations/codex
https://reinstate.dev/compatibility
https://reinstate.dev/security
https://reinstate.dev/about/reinstate
https://reinstate.dev/open-source
https://reinstate.dev/changelog
https://reinstate.dev/privacy
https://reinstate.dev/use-cases
https://reinstate.dev/use-cases/work-and-personal-computers
https://reinstate.dev/use-cases/macos-and-windows
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

IndexNow is not implemented or provisioned by this branch. Introduce it only
after the content and canonical system is stable.

Readiness criteria:

- [ ] An owner is assigned for the IndexNow account/key.
- [ ] The key is stored as a deployment secret, never in Git or `PUBLIC_*`.
- [ ] The key-location file or endpoint follows the provider's current
      requirements.
- [ ] Submissions use canonical production URLs only.
- [ ] Added, meaningfully updated, deleted, or recanonicalized URLs are
      submitted; unchanged URLs are not pinged on every deployment.
- [ ] Submission responses are logged without exposing the key.
- [ ] Failures are observable but do not block the production deployment.
- [ ] A rate limit and retry policy is documented before automation.

Treat the playbook's [Bing and IndexNow guidance](seo-aeo-aseo-playbook.md#58-bing-and-indexnow)
as the implementation policy.

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
an element has `data-analytics-event`. Review every proposed event attribute so
its name and target cannot contain user data.

## Production crawler and WAF checks

`robots.txt` currently distinguishes search discovery from model training:
`OAI-SearchBot` and `PerplexityBot` are allowed, while GPTBot has a separate
policy. A robots rule does not prove that the CDN or WAF permits a crawler.

Run public smoke tests from at least one external network:

```sh
for agent in OAI-SearchBot PerplexityBot; do
  curl -sS -A "$agent" -D - -o /dev/null https://reinstate.dev/
  curl -sS -A "$agent" -D - -o /dev/null https://reinstate.dev/docs
  curl -sS -A "$agent" -D - -o /dev/null https://reinstate.dev/robots.txt
  curl -sS -A "$agent" -D - -o /dev/null https://reinstate.dev/sitemap-index.xml
done
```

Expected:

- canonical pages return `200` with HTML;
- robots and sitemap return `200` with the correct content type;
- no challenge page, login wall, geo-block, `403`, `429`, or `5xx`;
- no crawler-specific redirect loop; and
- response time is recorded for comparison, not treated as field performance
  data.

User-agent strings are spoofable. These curls test the public response policy
only; they do not prove that a request came from OpenAI or Perplexity.

An operator with production log and WAF access must separately review:

- timestamp, path, verified bot family, status, bytes, latency, and cache result;
- challenge, block, or rate-limit actions;
- repeated `403`, `429`, and `5xx` responses;
- crawler loops or excessive requests;
- sitemap fetches and canonical-page fetches; and
- attempts to crawl `/api/` or `/preview/`.

Production log access, bot-IP verification, and WAF changes are not automatable
or completed in this branch.

## Launch-day record

| Check | Evidence | Owner | Time (UTC) | Result | Follow-up |
| ----- | -------- | ----- | ---------- | ------ | --------- |
| Production build/version |  |  |  | Not run |  |
| Robots and sitemap fetch |  |  |  | Not run |  |
| Google property/sitemap |  |  |  | Not run |  |
| Bing property/sitemap |  |  |  | Not run |  |
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
- [Google guidance for AI features](https://developers.google.com/search/docs/appearance/ai-features)
- [Astro sitemap integration](https://docs.astro.build/en/guides/integrations-guide/sitemap/)
- [OpenAI publishers and developers FAQ](https://help.openai.com/en/articles/12627856-publishers-and-developers-faq)
- [OpenAI ChatGPT Search](https://help.openai.com/en/articles/9237897)
- [Perplexity crawler documentation](https://docs.perplexity.ai/docs/resources/perplexity-crawlers)
