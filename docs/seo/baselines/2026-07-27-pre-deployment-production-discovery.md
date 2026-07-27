# Pre-deployment production discovery baseline — 2026-07-27

This is a read-only public HTTP baseline for `https://reinstate.dev` before the
SEO/AEO/AI-search branch is deployed. Missing branch-added routes and discovery
files are expected at this point; they are observations, not regressions in the
undeployed branch.

## Observation context

| Field | Value |
| ----- | ----- |
| Observation time | `2026-07-27T14:30:51Z` |
| Repository commit at observation | `5d1e89dd1b67e4fc5940e22378aab733dfbbfb1a` |
| Target | Production `https://reinstate.dev` |
| Method | Unauthenticated public `curl` GET requests |
| Redirect handling | Redirects were not followed; `redirect_url` records any direct redirect target |
| Branch deployment state | This SEO branch was **not deployed** |

The requests did not use Search Console, Bing Webmaster Tools, analytics,
production logs, CDN/WAF administration, or any private account. A supplied
user-agent string is not proof of crawler identity.

## Route results

| URL | Status | Content type | Redirect outcome |
| --- | ---: | ------------ | ---------------- |
| `https://reinstate.dev/` | `200` | `text/html; charset=utf-8` | None; effective URL unchanged |
| `https://reinstate.dev/docs` | `200` | `text/html; charset=utf-8` | None; effective URL unchanged |
| `https://reinstate.dev/robots.txt` | `404` | `text/html` | None; effective URL unchanged |
| `https://reinstate.dev/sitemap-index.xml` | `404` | `text/html` | None; effective URL unchanged |
| `https://reinstate.dev/llms.txt` | `404` | `text/html` | None; effective URL unchanged |
| `https://reinstate.dev/security` | `404` | `text/html` | None; effective URL unchanged |

The homepage and existing docs hub were available. `robots.txt`,
`sitemap-index.xml`, `llms.txt`, and `/security` are part of the SEO branch's
new or changed production surface, so their pre-deployment `404` responses are
expected. This baseline does not claim that they will work until a deployment
and post-deployment rerun prove it.

## User-agent response results

Each request targeted the production homepage and changed only the public
`User-Agent` header.

| Supplied user-agent string | Status | Content type | Redirect outcome |
| -------------------------- | -----: | ------------ | ---------------- |
| `OAI-SearchBot` | `200` | `text/html; charset=utf-8` | None; effective URL unchanged |
| `PerplexityBot` | `200` | `text/html; charset=utf-8` | None; effective URL unchanged |
| `GPTBot` | `200` | `text/html; charset=utf-8` | None; effective URL unchanged |

These results show only that the public endpoint returned ordinary HTML to
requests carrying those strings from this test network. They do not verify the
request source as OpenAI or Perplexity, prove access from documented crawler IP
ranges, reveal a production WAF rule, or override a crawler's robots policy.

Production did not serve `robots.txt` during this observation. Therefore this
baseline cannot demonstrate the branch's intended distinction between
`OAI-SearchBot` search discovery and the separate GPTBot training policy.

## Reproduction commands

```sh
for url in \
  https://reinstate.dev/ \
  https://reinstate.dev/docs \
  https://reinstate.dev/robots.txt \
  https://reinstate.dev/sitemap-index.xml \
  https://reinstate.dev/llms.txt \
  https://reinstate.dev/security
do
  curl --max-time 20 --connect-timeout 10 \
    -sS -o /dev/null \
    -w "$url\t%{http_code}\t%{content_type}\t%{redirect_url}\t%{url_effective}\n" \
    "$url"
done

for agent in OAI-SearchBot PerplexityBot GPTBot
do
  curl --max-time 20 --connect-timeout 10 \
    -sS -A "$agent" -o /dev/null \
    -w "$agent\t%{http_code}\t%{content_type}\t%{redirect_url}\t%{url_effective}\n" \
    https://reinstate.dev/
done
```

## Required post-deployment rerun

After the SEO branch is deployed:

1. Repeat the route table and require successful discovery-file responses with
   their intended content types.
2. Confirm `/security` and every other newly deployed canonical route returns
   `200` without a redirect.
3. Repeat the public user-agent smoke tests.
4. Follow the
   [production crawler and WAF checks](../operations.md#production-crawler-and-waf-checks),
   including the separately authorized production-log review.
5. Follow the
   [Search Console launch URL inspection steps](../operations.md#launch-url-inspection)
   only after an authorized operator has verified the property.
6. Create a new dated post-deployment baseline instead of overwriting this
   historical observation.

Account verification, sitemap submission, URL Inspection, analytics review,
verified bot-log analysis, and WAF administration remain external operational
actions. None was performed or inferred for this baseline.

## Final pre-deployment automated smoke

The branch's bounded, read-only production checker was run at
`2026-07-27T16:40:25Z` from commit
`4bdca3327a3b1268f19e7febbaf71576af07488d`. It made 45 unauthenticated
`GET`/`HEAD` checks: 10 passed and 35 failed, producing 65 findings. Production
still exposed zero sitemap URLs and zero route-specific Open Graph images
because this branch remained undeployed. The failures were therefore consistent
with the earlier observation rather than evidence about the generated branch
build.

The canonical-host check passed: `https://www.reinstate.dev/` returned a direct
permanent `308` to `https://reinstate.dev/`. A separate deep-link observation
also returned `308` from `https://www.reinstate.dev/docs/getting-started` to the
same path on the apex host.

The redacted local evidence artifact was written with mode `0600` under the
ignored `website/artifacts/discovery/` directory. Its SHA-256 digest is
`28c6894371a2798c0099f580fa8016cfce9d43fc6a6e0950426ae7fee6159bb8`.
The artifact itself is intentionally not committed; a deployment operator
should create a new immutable post-deployment record instead of treating this
pre-deployment failure as launch evidence.
