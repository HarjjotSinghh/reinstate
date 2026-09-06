# W9 — Stable promotion (founder-gated)

**Executor:** one Sonnet agent for the commits and checks; the founder for
the signatures and publication; the coordinator reviews.

Preconditions: W7b's tagged-artifact Windows report is `PASS` on every
required row under the contract's disposition rules
([`docs/testing/v0.6.0-windows-acceptance.md`](../../../testing/v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict))
and committed to `main`; the founder has recorded the stable promotion
decision (a sentence in `RELEASING.md`'s `v0.6.0` stable evidence section is
enough).

## T-901 — Stable release commit

`chore(release): v0.6.0` on `release/v0.6.0` from `main`:

- `CHANGELOG.md`: `## [0.6.0] - <date>` above the rc section, in the shape of
  `[0.5.0]` over `[0.5.0-rc.6]`: what authorizes it (the Windows report), the
  waiver, the deferred issue, and the dispositions carried.
- `scripts/install.sh`, `scripts/install.ps1`, `website/public/install.*`
  pinned to `v0.6.0`; canonical installer digests recomputed per
  `RELEASING.md` step 1; `internal/doctest/bootstrap_install_contract_test.go`
  updated.
- `CITATION.cff`, `website/src/data/product.ts` (`currentRelease`,
  `stableRelease`, `releaseStatus` naming Windows-only acceptance and the
  deferred macOS issue), `releases.ts`, `released-tiers.json` (OpenCode T5,
  Kimi T2 released), `compatibility.json`.
- `RELEASING.md`: `v0.6.0` stable evidence under the waiver.
- `ROADMAP.md`: Phase 6C ✅ with the date.

Then the same proofs as T-802, the PR, CI, merge.

## T-902 — Founder steps (exact commands in `clarifications.md`)

Sign and push `v0.6.0`; verify the draft's assets, checksums, attestations,
and both installers against the draft; publish (not prerelease); let
**Publish package managers** run; create and push the signed
`website-vYYYY.MM.DD.N` tag at `origin/main`; run
`scripts/deploy-website-production.sh` after the validation workflow passes.

## T-903 — Close out

Fast-forward `hop/main` to `main`; delete the merged `hop/*` ticket branches
and their worktrees under `D:\Projects\reinstate-worktrees`; close hosted
#16 with the report link; update `reinstate-hosted/STATUS.md` "Where the
code lives"; remove the pruned worktree entries (`git worktree prune`).
