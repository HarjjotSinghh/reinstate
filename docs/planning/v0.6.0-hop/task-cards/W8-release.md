# W8 — Release commit, dispatch document, candidate PR

**Executor:** one Sonnet agent. **Verifier:** the coordinator, plus the
Go doctest release guards.
**Branch:** work directly on `release/v0.6.0-rc.1` after W1–W7 are merged.
**Reference:** `git show 8caa943a` (the `v0.5.2-rc.1` release commit) is the
template for exactly which files move together.

## T-801 — The release commit

One commit, `release: cut v0.6.0-rc.1`, containing:

- `CHANGELOG.md`: `## [0.6.0-rc.1] - <today>`; the link block.
- `CITATION.cff`: the `v0.5.2-rc.1` commit moved version and date to the
  candidate; the coordinator already set `0.6.0-rc.1`, so only the date moves.
- `website/public/install.sh`, `website/public/install.ps1`, and
  `internal/doctest/bootstrap_install_contract_test.go`: the `v0.5.2-rc.1`
  commit pinned the public bootstraps to the candidate itself (with the
  canonical installer digests recomputed per `RELEASING.md` step 1); do the
  same for `v0.6.0-rc.1`. This is the one place W8 edits `website/public/`.
- `README.md`, `docs/getting-started.md`, `docs/cli-reference.md`,
  `docs/prompts/*.md`: the version sentences `v0.5.2-rc.1` touched.
- `website/src/data/*` dates from W5's placeholders.
- `RELEASING.md`: the candidate gate's date and counts finalized; the
  `#DEFERRED-MACOS` placeholder replaced by the issue from T-804.
- `docs/testing/v0.6.0-rc.1-agent-verification-prompts.md`: the Windows
  dispatch in the shape of `v0.5.0-rc.6-agent-verification-prompts.md` —
  fixed release identity, evidence boundary, what this candidate specifically
  re-tests (the widened ranges, the Hop journeys, rows 2 and 22), the required
  counts, the report paths — plus a "Deferred: Apple Silicon macOS" section
  that reproduces the contract's deferred table and says the same tag is
  re-certified there when hardware returns.

## T-802 — Prove the exact commit

```bash
GOTOOLCHAIN=go1.25.13 go mod tidy -diff
make verify            # or the PowerShell twins per windows-acceptance-host.md
scripts/snapshot.ps1; scripts/stage-release-assets.ps1 dist; scripts/check-release-artifacts.ps1 dist; scripts/test-install.ps1 dist
git diff --exit-code -- go.mod go.sum
git status --porcelain   # empty
```

## T-803 — Push and open the PR

`git push -u origin release/v0.6.0-rc.1`; `gh pr create --base main --draft`
with a body that lists what the candidate carries, links the pre-tag Windows
report, states the waiver, and ends with the required generated-with footer.
Wait for CI; fix only CI-environment failures on the branch (a product
failure goes back to its workstream). Re-point #366 at this candidate's
Windows run with a comment; comment on #367 and #368 with W4's ConPTY
outcome. Do not merge the PR; the coordinator does.

## T-804 — The deferred-macOS issue

Open one public issue, "v0.6.0: Apple Silicon macOS acceptance deferred
(ADR 0005)", listing every deferred row from the contract's section E, the
tag it must be run against, and the instruction that a failure ships as
`v0.6.1`. Reference it from `RELEASING.md` and the dispatch doc.

## Done when

The PR is open and green; the release commit passes every doctest release
guard; the coordinator has verified the artifact identity locally.
