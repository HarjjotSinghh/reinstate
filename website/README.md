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
