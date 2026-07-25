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

```bash
cd website
vercel link   # once
vercel env add WAITLIST_GIST_ID production   # or Turso vars
vercel env add GITHUB_TOKEN production
vercel --prod
```

Live project: **https://reinstate-web.vercel.app** (Vercel project `harjjot/reinstate-web`).

Root directory for the Vercel project must be `website/`.

## Design

See the repository-root `PRODUCT.md` and `website/AGENTS.md`.
