# Local rendered-browser baseline — 2026-07-27

This is a reproducible lab smoke test for commit `a343a76`, not field Core Web
Vitals and not a production ranking claim.

## Environment

- Lighthouse `13.4.1`
- Chrome Headless Shell `151.0.7922.47`
- local prerendered `dist/client` served by the repository's static preview
  server
- Lighthouse mobile defaults and simulated throttling
- analytics disabled
- five representative page templates

Full JSON reports are generated under `website/artifacts/lighthouse/` and are
uploaded by CI. That directory is intentionally ignored locally because the
reports are large and environment-dependent.

## Passing run

| Route | Performance | Accessibility | Best practices | SEO | LCP | CLS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `/` | 92 | 100 | 100 | 100 | 2,852 ms | 0.000 |
| `/docs/getting-started` | 99 | 100 | 100 | 100 | 1,967 ms | 0.000 |
| `/integrations/claude-code` | 99 | 100 | 100 | 100 | 1,670 ms | 0.000 |
| `/guides/sync-claude-code-sessions-across-devices` | 98 | 100 | 100 | 100 | 2,121 ms | 0.000 |
| `/compatibility` | 99 | 100 | 100 | 100 | 1,969 ms | 0.000 |

Scores can vary between lab runs. The command fails for SEO below 100,
accessibility or best practices below 95, performance below 80, or a failed
critical semantic audit. LCP above 2.5 seconds and CLS above 0.1 are reported as
warnings because lab timing is noisy; they remain investigation targets.

The homepage LCP warning reproduced around 2.85 seconds after the passing run.
The identified LCP element was visible text, not a late-loaded image. Two
above-the-fold font files are now preloaded, but this local result remains above
the target. Production field LCP and INP remain unavailable until the deployed
site has sufficient real-user evidence.

## Accessibility correction made during the run

The first run correctly failed light-theme muted text and green links for
insufficient contrast. The shared tokens were darkened, persistent underlines
were restored for inline prose links, and the complete five-route rerun reached
100 accessibility on every representative route.
