# W5 — Website release truth

**Executor:** one Sonnet agent. **Verifier:** one Sonnet agent (diff every
public sentence about platforms, agents, and the hosted tier against ADR
0005 and the tree).
**Branch:** `v060/w5-website` from `release/v0.6.0-rc.1`.
**Reference:** the `v0.5.2-rc.1` release commit `8caa943a` shows every
website file a candidate touches. `website/` is an Astro site; `npm ci` is
already done (`node_modules` present); `vitest run` is the fast gate.

## T-501 — Release data

- `website/src/data/releases.ts`: prepend `v0.6.0-rc.1` (date placeholder
  `2026-09-DD`; W8 sets it) with a summary in the house style: what it
  carries, that it does not authorize stable, that stable remains `v0.5.1`,
  and that its acceptance is native Windows x64 with macOS deferred.
- `website/src/data/product.ts`: `currentRelease` stays `v0.5.1` for a
  candidate (check what `v0.5.2-rc.1` did and match it); `releaseStatus`
  text names the candidate and the Windows-first waiver; `requiresAccount`
  stays `false`; `supportedAgents` and `supportedStorage` updated only if the
  tree supports the change (OpenCode now syncs; Reinstate Hop is not a
  storage option users can pick yet).
- `website/src/data/released-tiers.json` and `agent-version-history.ts`:
  OpenCode T5, Kimi T2 as **merged-but-unreleased** until stable (#369 asks
  for a next-release marker; add the marker if the shape allows, otherwise
  leave a comment and note it).
- `website/src/data/compatibility.json`: only the fields W3 does not own.
- `website/public/install.*` are **not** touched (W9).

## T-502 — Tests follow the truth

`product-truth.test.ts`, `seo.test.ts`, `linkable-assets.test.ts`: update
expectations to the new truth. A test that asserted `v0.5.1` in three places
now asserts the candidate in the same three; a test that guarded "no extra T5
agents" now names OpenCode explicitly. Do not delete a guard.

## T-503 — Docs mirror and the hosted-tier sentences

- Add `website/src/content/docs/hop.md` mirroring `docs/hop.md` after W1's
  truth pass (coordinate: take W1's merged text, do not fork it), with the
  site's frontmatter conventions, and link it from the docs navigation the
  way the other mirrored pages are linked.
- `website/src/pages/contact.astro` ("no hosted account, no paid support
  tier") and `website/src/content/docs/storage.md` ("does not require a
  Reinstate-hosted account"): keep them true — Reinstate does not require an
  account; the hosted tier exists in the client and is not yet open. No
  pricing, no trial, no sign-up call to action.
- `website/src/pages/roadmap.astro` if it renders Phase 6: match W1's split.

## T-504 — Gates

`npm run build` and `npx vitest run` green. Of the `npm test` chain, run
what works offline (`test:seo`, `test:links` if it is local, `test:freshness`,
`test:agent-surface`, `check:media`, `check:github-social`) and list what
needs the network and was not run.

## Done when

Gates 1–3 (the Go doctest `TestWebsiteReleaseTruthStaysSynchronized` and
`TestReleaseAndSupportClaims` read these files), the verifier's report, and
the list of skipped network checks in the PR description.
