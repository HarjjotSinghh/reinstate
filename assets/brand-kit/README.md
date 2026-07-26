# Reinstate brand kit

Generated from the selected **Pure Tight** logo and the DESIGN.md illustration
language (isometric flats, paper/ink, chartreuse accent, four stream layers).

## Regenerate

```bash
python3 scripts/generate_brand_kit.py
```

Requires `rsvg-convert` (librsvg).

## Logo (final)

See `logo/` (copied from `assets/logo/final/`) and website favicons:

| File | Use |
|------|-----|
| `website/public/favicon-light.svg` | Light theme tab icon |
| `website/public/favicon-dark.svg` | Dark theme tab icon |
| `website/public/favicon.svg` | Default fallback (dark) |
| `website/public/favicon.ico` | Legacy |

The site swaps `favicon-light` / `favicon-dark` when the user toggles theme
(`Header.astro` + boot script in `BaseLayout.astro`).

## Social assets

### Profiles

| Asset | Size | Path |
|-------|------|------|
| Avatar / PFP | 800² | `png/pfp-800.png` |
| Avatar light | 800² | `png/pfp-light-800.png` |
| Discord / Slack | 512² | `png/discord-512.png`, `png/slack-512.png` |

### Headers & covers

| Platform | Size | Path |
|----------|------|------|
| X / Twitter header | 1500×500 | `png/x-header-1500x500.png` |
| LinkedIn personal | 1584×396 | `png/linkedin-banner-1584x396.png` |
| LinkedIn company | 1128×191 | `png/linkedin-company-1128x191.png` |
| YouTube banner | 2560×1440 | `png/youtube-banner-2560x1440.png` |
| GitHub social | 1280×640 | `png/github-social-1280x640.png` |

### Open Graph

| Asset | Size | Path |
|-------|------|------|
| OG (light) | 1200×630 | `png/og-1200x630.png` → also `website/public/brand/og.png` |
| OG (dark) | 1200×630 | `png/og-dark-1200x630.png` |

### Post templates

| Platform | Size | Path |
|----------|------|------|
| X post | 1200×675 | `png/x-post-1200x675.png` |
| LinkedIn post | 1200×627 | `png/linkedin-post-1200x627.png` |
| Instagram square | 1080² | `png/ig-square-1080.png` |
| Poster / story-ish | 1080×1350 | `png/poster-1080x1350.png` |

Matching SVGs live in `svg/` for further edit.

## Website copies

Key files are mirrored to `website/public/brand/` for the marketing site
(`og.png`, headers, avatar, post templates, `banner.svg`).

## Brand rules (short)

1. Sharp, never soft — no blur/glow/glass.
2. One loud accent: chartreuse `#b8ff3c`.
3. Paper `#e4e7dd` / ink `#131f1a`.
4. Streams only as sessions / MCP / skills / settings colours.
5. Logo geometry: Pure Tight (dual frames + `>_`).

Full system: repo root `DESIGN.md`, `PRODUCT.md`.
