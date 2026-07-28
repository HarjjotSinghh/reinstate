# reinstate.dev website

Marketing site, docs, and waitlist for [Reinstate](https://github.com/HarjjotSinghh/reinstate).

## Stack

- Astro 7 + Tailwind CSS v4 + MDX
- Validated docs, guides, and blog collections under `src/content`
- Waitlist API → Turso / libSQL (`@libsql/client`)
- Deploy: Vercel (`@astrojs/vercel`)

## Local development

```bash
cd website
cp .env.example .env
# Local file DB (no cloud account required):
echo "TURSO_DATABASE_URL=file:$(pwd)/data/waitlist.db" > .env
mkdir -p data

npm install
npm run dev
```

- Site: http://localhost:4321
- Waitlist: `POST /api/waitlist` with `{ "email": "you@example.com" }`

```bash
npm test
npm run build
npm run preview
```

`npm run preview` serves the prerendered files in `dist/client` at
`http://127.0.0.1:4321`. It is intended for Lighthouse and release QA and does
not execute server routes such as the waitlist API, RSS feed, or IndexNow
key-proof endpoint. Use `npm run dev` for ordinary development and the Vercel
local runtime when server-route parity is required.

## SEO validation

Run the production-build audit locally after changing routes, metadata,
structured data, crawler policy, or social cards:

```bash
npm run build
npm run check:seo
```

The dependency-free audit checks every generated indexable page, JSON-LD,
canonical and title uniqueness, route-specific 1200×630 PNG social cards,
robots crawler rules, and sitemap coverage. Failures include the generated file
and a suggested fix. Its focused fixture tests are also included in `npm test`.

## IndexNow readiness

IndexNow planning is dry-run-only unless an operator supplies both a previously
reviewed plan and `--submit`. The CI gate compares the generated sitemap with
itself, exercises the no-change path, and never reads a key or makes a network
request:

```bash
npm run build
npm run check:indexnow
```

For release diffing, deletion and recanonicalization inputs, key provisioning,
and post-deployment soft-fail submission, follow
[`docs/seo/operations.md`](../docs/seo/operations.md#indexnow-release-procedure).
Never pass a key on the command line or place it in a `PUBLIC_*` variable.

## Link validation

Run the generated-site link and fragment crawler after changing routes,
navigation, Markdown links, redirects, or public assets:

```bash
npm run build
npm run check:links
```

The dependency-free crawler resolves root-relative, document-relative, and
same-origin absolute links; verifies generated pages and public assets; checks
fragment targets; understands configured redirects and Vercel runtime routes;
and reports the source file, route, reference, and suggested fix. External URLs
are left for the production audit because availability and redirects can change
independently of this repository.

## Performance budgets

Run the dependency-free static-build budget check after changing layouts,
fonts, scripts, styles, or media:

```bash
npm run build
npm run check:performance
```

The check covers 14 required representatives across the homepage,
documentation index and articles, troubleshooting/FAQ, integration, privacy,
guide and blog indexes and articles, comparison, use case, compatibility, and
not-found templates. It reports raw and deterministic gzip sizes for HTML, CSS,
executable JavaScript, route media, and the initial static transfer. It also
limits render-blocking style/script counts, font candidates declared by route
CSS, font preloads, external references, and local asset requests. External
render-blocking stylesheets or scripts fail because their size cannot be
verified from the build.

Budgets intentionally leave reviewable headroom above the current output while
keeping each page type bounded:

| Route | HTML raw / gzip | CSS raw / gzip | Static transfer raw / gzip |
| --- | ---: | ---: | ---: |
| `/` | 200 / 35 KiB | 140 / 28 KiB | 460 / 180 KiB |
| `/docs/getting-started` | 64 / 14 KiB | 80 / 18 KiB | 220 / 90 KiB |
| `/docs/troubleshooting` | 96 / 22 KiB | 80 / 18 KiB | 260 / 105 KiB |
| `/integrations/claude-code` | 48 / 12 KiB | 90 / 20 KiB | 230 / 95 KiB |
| `/privacy` | 48 / 12 KiB | 80 / 18 KiB | 210 / 85 KiB |
| indexes, editorial, comparison, use-case, compatibility | 72 / 16 KiB | 100 / 22 KiB | 250 / 100 KiB |
| `/404` | 48 / 12 KiB | 80 / 18 KiB | 220 / 90 KiB |

Every route also has a 16 KiB raw / 6 KiB gzip executable-JavaScript
budget, at most one render-blocking script, at most 16 declared font files
totalling 240 KiB raw / 245 KiB gzip, and no more than two font preloads. The
homepage has a larger transfer allowance for its code-native illustrations;
editorial routes have tighter HTML and CSS limits. The evidence-bearing
comparison may use five blocking style bundles; all other templates allow at
most four. Route media is capped separately and included in static transfer.
Declared fonts are reported separately because the browser selects files by
family and Unicode range instead of downloading every `@font-face` candidate.

This is a deterministic regression gate over `dist/client`, not a browser
measurement. It does not produce Lighthouse scores or claim to measure field
Core Web Vitals, runtime rendering, cache behavior, CDN behavior, device CPU,
or network latency. For release QA, run several Lighthouse mobile navigations
against `npm run preview` and the production URL, compare the median lab LCP,
CLS, and TBT, then review PageSpeed Insights/CrUX and Search Console field data
when traffic is sufficient. Any future Lighthouse CI job should use a pinned
Chrome/Lighthouse version, a real preview server, multiple runs, and separately
calibrated thresholds; it should complement rather than replace this static
gate.

## Waitlist storage

**Local / verification:** libSQL file

```bash
mkdir -p data
echo "TURSO_DATABASE_URL=file:$(pwd)/data/waitlist.db" >> .env
```

**Turso cloud (preferred when logged in):**

```bash
/opt/homebrew/bin/turso auth login
/opt/homebrew/bin/turso db create reinstate-waitlist
/opt/homebrew/bin/turso db show reinstate-waitlist --url
/opt/homebrew/bin/turso db tokens create reinstate-waitlist
```

**Production fallback (already wired):** private GitHub Gist + `GITHUB_TOKEN`

- `WAITLIST_GIST_ID`
- `GITHUB_TOKEN`

Optional Resend notify (never required for signup success):

- `RESEND_API_KEY`
- `WAITLIST_NOTIFY_TO`
- `WAITLIST_FROM_EMAIL`

## Deploy

Automatic Vercel Git deployments are disabled. Production must come from an
approved signed website deployment tag at the exact `origin/main` commit:

- `website-vYYYY.MM.DD.N` identifies the exact website source to deploy. It is
  not a Reinstate version or GitHub Release.
- `vX.Y.Z[-prerelease]` separately identifies the CLI release pinned by both
  public bootstraps. The guarded script derives this tag from those files and
  refuses to deploy if they disagree.

```bash
# One-time project/environment setup:
cd website
vercel_cli() { npm exec --yes --package=vercel@57.0.0 -- vercel "$@"; }
vercel_cli link --project reinstate-web --scope harjjot
vercel_cli env add WAITLIST_GIST_ID production   # or Turso vars
vercel_cli env add GITHUB_TOKEN production
vercel_cli env add INDEXNOW_KEY production       # optional; prompts securely
cd ..

# After the signed tag-validation workflow passes for the exact origin/main commit:
./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N

# Current example: ship reviewed website changes without changing RC6:
./scripts/deploy-website-production.sh website-v2026.07.28.1
```

Canonical live site: **https://reinstate.dev** (Vercel project
`harjjot/reinstate-web`).

Root directory for the Vercel project must be `website/`.

The deployment script refuses dirty, non-`main`, unpushed, unsigned, or
tag-mismatched source. It verifies the signed website deployment tag, derives
the CLI installer tag independently from both public bootstraps, refuses a
version mismatch, verifies the corresponding signed published CLI release, and
requires the committed installers to match that release byte for byte. It
builds and tests locally, deploys without moving the production alias, verifies
the immutable deployment and its installers, promotes only after those checks
pass, and verifies the live origin again. Do not run `vercel --prod` directly.
Push the website tag and wait for the **Validate signed website deployment
tag** workflow to pass before running the command.

## Design

See the repository-root `PRODUCT.md` and `website/AGENTS.md`.
