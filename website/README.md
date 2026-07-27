# reinstate.dev website

Marketing site, docs, and waitlist for [Reinstate](https://github.com/HarjjotSinghh/reinstate).

## Stack

- Astro 7 + Tailwind CSS v4 + MDX
- Docs from `src/content/docs` (synced from repo `docs/`)
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

## Performance budgets

Run the dependency-free static-build budget check after changing layouts,
fonts, scripts, styles, or media:

```bash
npm run build
npm run check:performance
```

The check covers the homepage, getting-started docs, Claude Code integration,
privacy page, and the guide and blog hubs when present. It reports raw and
deterministic gzip sizes for HTML, CSS, executable JavaScript, route media, and
the initial static transfer. It also limits render-blocking style/script counts,
font candidates declared by route CSS, font preloads, external references, and
local asset requests. External render-blocking stylesheets or scripts fail
because their size cannot be verified from the build.

Budgets intentionally leave reviewable headroom above the current output while
keeping each page type bounded:

| Route | HTML raw / gzip | CSS raw / gzip | Static transfer raw / gzip |
| --- | ---: | ---: | ---: |
| `/` | 200 / 35 KiB | 140 / 28 KiB | 460 / 180 KiB |
| `/docs/getting-started` | 64 / 14 KiB | 80 / 18 KiB | 220 / 90 KiB |
| `/integrations/claude-code` | 48 / 12 KiB | 90 / 20 KiB | 230 / 95 KiB |
| `/privacy` | 48 / 12 KiB | 80 / 18 KiB | 210 / 85 KiB |
| `/guides`, `/blog` | 72 / 16 KiB | 100 / 22 KiB | 250 / 100 KiB |

Every route also has a 16 KiB raw / 6 KiB gzip executable-JavaScript
budget, at most one render-blocking script, at most 16 declared font files
totalling 240 KiB raw / 245 KiB gzip, and no more than two font preloads. The
homepage has a larger transfer allowance for its code-native illustrations;
editorial routes have tighter HTML and CSS limits. Route media is capped
separately and included in static transfer. Declared fonts are reported
separately because the browser selects files by family and Unicode range
instead of downloading every `@font-face` candidate.

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

Automatic Vercel Git deployments are disabled. Production must come from the
signed release tag at the exact `origin/main` commit.

```bash
# One-time project/environment setup:
cd website
npx vercel link --project reinstate-web --scope harjjot
npx vercel env add WAITLIST_GIST_ID production   # or Turso vars
npx vercel env add GITHUB_TOKEN production
cd ..

# After the signed tag and GitHub prerelease exist:
./scripts/deploy-website-production.sh vX.Y.Z
```

Live project: **https://reinstate-web.vercel.app** (Vercel project `harjjot/reinstate-web`).

Root directory for the Vercel project must be `website/`.

The deployment script refuses dirty, non-`main`, unpushed, unsigned, or
tag-mismatched source. It builds and tests locally, deploys without moving the
production alias, byte-verifies both installers at the immutable deployment
URL, promotes that deployment, and verifies the live routes again. Do not run
`vercel --prod` directly.

## Design

See the repository-root `PRODUCT.md` and `website/AGENTS.md`.
