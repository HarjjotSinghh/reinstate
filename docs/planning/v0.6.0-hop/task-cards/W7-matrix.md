# W7 — Pre-tag Windows matrix (and W7b, the tagged-artifact run)

**Executors:** three Sonnet agents (core + T0/T1 agents; T2–T5 agents; the
22 CLI rows), each in its own worktree and lab root. **Verifier:** one
Sonnet agent (re-run a random 10% of PASS rows plus every row that was
`FAIL` then `PASS`; check the counts).
**Branch:** `v060/w7-matrix` (results doc only).
**Contract:** [Windows acceptance](../../../testing/v0.6.0-windows-acceptance.md)
sections A–C and E. **Template:** `docs/testing/results/phase-5-report-template.md`.

## T-701 — The artifact

Pre-tag: on the release-candidate commit (after W8's release commit exists,
or on the branch tip if the coordinator says so), run `scripts/snapshot.ps1`,
`scripts/stage-release-assets.ps1`, `scripts/check-release-artifacts.ps1`,
`scripts/check-release-binary-identity.ps1`, `scripts/test-install.ps1`.
Install the staged `windows_amd64` archive into a fresh directory; `rein` and
`reinstate` are byte-identical; `rein version --json` names the commit.

W7b (after the founder publishes the tag): install from the live bootstrap
after proving it pins the exact tag, per the dispatch doc W8 writes.

## T-702 — Phase 5 generated matrix, Windows column

`rein doctor --agents --acceptance-matrix --json` on the installed binary
gives the required rows (178 at planning time: core A 10, B 9, G 8, H 6;
per-agent C/D/E/F). Run every row per
`docs/testing/phase-5-universal-agent-coverage-acceptance.md`, with the
`v0.5.0-rc.6` dispatch's lessons (B4 needs an OpenCode Git object store; the
upgrade path needs an index built by `v0.5.1` first; H4 refresh timing).
Agents that are genuinely absent from the host make their rows `NOT TESTED`
only where the contract allows; do not install an agent solely for a row.

## T-703 — CLI experience, 22 rows

Through W4's `conptydriver` against `scripts/tuisandbox` (root outside any
Git checkout). Rows 2 and 22 are the frozen-output guard: compare against a
`v0.5.1` binary installed from the GitHub release into another fresh
directory, byte for byte. Row 21 (ASCII glyphs on legacy conhost) needs a
console with no `WT_SESSION` and no UTF-8 code page; the driver can set that.

## T-704 — The report

`docs/testing/results/2026-09-DD-windows-v060rc1-pretag.md` (W7b:
`…-windows-v060rc1.md`): the Phase 5 template's sections, then a "CLI
experience" table (22 rows), then a pointer to W6's Hop report, then the
deferred-macOS table copied from the contract with every row `DEFERRED`.
`PARTIAL` and `NOT TESTED` do not pass a required row. The header records the
generated row count and the exact binary identity.

## Done when

Every required Windows row `PASS`; Gates 3 and 4; the verifier's sample
re-run agrees; the coordinator has read every `FAIL`-then-`PASS` row's
explanation.
