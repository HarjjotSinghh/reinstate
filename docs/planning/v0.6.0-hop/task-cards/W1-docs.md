# W1 — Changelog, release docs, roadmap, product truth

**Executor:** one Sonnet agent. **Verifier:** one Sonnet agent (adversarial:
hunt for a claim the diff makes that the tree does not support).
**Branch:** `v060/w1-docs` from `release/v0.6.0-rc.1`.
**Owns:** see [file-ownership.md](../file-ownership.md). Do not touch
`docs/compatibility.md` until W3 has merged; the coordinator will say when.

Read first: [ADR 0005](../../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md),
`CHANGELOG.md` `[Unreleased]` (about 1,045 lines) and `[0.5.2-rc.1]`,
`RELEASING.md`, `ROADMAP.md`, `docs/hop.md`, the `v0.5.2-rc.1` release commit
(`git show --stat 8caa943a`) for the shape of a release's doc changes.

## T-101 — One `[0.6.0-rc.1]` section

The `[Unreleased]` section is three merges' worth of headings: `Security`,
`Fixed`, `Changed`, `Added`, `Fixed`, `Changed`, `Added`, `Fixed`. Reshape it
into `## [Unreleased]` (empty) followed by `## [0.6.0-rc.1] - YYYY-MM-DD`
(leave the literal `YYYY-MM-DD`; W8 sets it) with:

1. A **Highlights** paragraph in the voice of the `[0.5.2-rc.1]` one: the Hop
   client (one sentence naming the commands), OpenCode T5 and Kimi T2, and
   that this candidate also carries everything `v0.5.2-rc.1` introduced.
2. A **Not yet certified** paragraph: native Windows x64 acceptance is what
   this candidate exists to enable; macOS acceptance is deferred under ADR
   0005; stable remains `v0.5.1`; the hosted control plane is not yet open.
3. `### Security`, `### Added`, `### Changed`, `### Fixed` — one of each.
   Move bullets; do not rewrite, shorten, or drop any. Keep each bullet's
   nested sub-bullets with it. Where two bullets describe the same change
   from two merges, keep both and let the verifier flag it; do not merge
   prose.
4. The link-reference block at the bottom gains `[0.6.0-rc.1]` and
   `[Unreleased]` compares from it.

Then find the changelog guard added by #364 (`grep -rln CHANGELOG
internal/doctest`) and run it. Count bullets before and after
(`grep -c '^- ' CHANGELOG.md`); the count must not drop.

## T-102 — `RELEASING.md`

- Under **Supported platform boundary**, add **v0.6.0 Windows-first waiver**:
  what ADR 0005 D2 says, in the register of the `v0.2.0` reconciliation
  paragraph above it. Name the issue placeholder `#DEFERRED-MACOS` (W8
  replaces it with the real number).
- Add **v0.5.2-rc.1 candidate evidence**: published 2026-08-23; never
  certified on either platform; its content ships in `v0.6.0`; no stable
  `v0.5.2` (D4).
- Add **v0.6.0-rc.1 candidate gate**: what the candidate carries, which
  contract governs (`docs/testing/v0.6.0-windows-acceptance.md`), required
  counts (Phase 5 generated matrix at this catalog — take the number from
  `rein doctor --agents --acceptance-matrix` on the branch, 178 at planning
  time — plus 22 CLI rows plus the Hop parity rows), the pre-tag snapshot run
  as non-authorizing evidence, and "Publication means ready for tagged-artifact
  acceptance. It does **not** authorize stable `v0.6.0`. Current stable remains
  `v0.5.1`."
- In **Preconditions**, add one line: for `v0.6.0` the stable acceptance row
  is native Windows x64 under the waiver, with the macOS rows recorded as
  deferred rather than passed.
- Step 2 (tag): add a sentence that the signing key lives only on the
  maintainer's machines and an agent never tags.

## T-103 — `ROADMAP.md`

- Phase 6 becomes two entries. **Phase 6C — Cloud continuity (Hop) ✅** in
  `v0.6.0`: device registry and revocation ✅, key rotation (key generations)
  ✅, hardened push/pull habit (daemon) ✅, machine migration UX (recovery
  code, `sync migrate`) ✅, additional backends 📋, append-aware delta 📋. Add
  the sentence that `v0.6.0` acceptance is native Windows x64 with macOS
  deferred under ADR 0005, and that the hosted service opens separately.
  **Phase 6A/6B 📋** keep their tables, retargeted to `v0.7.0`.
- Phase 5's closing note stays as history; add one forward pointer after it:
  "OpenCode reached T5 and Kimi T2 in `v0.6.0`."
- "Last updated" → 2026-09-05. Stable release policy: add the `v0.6.0` line.

## T-104 — Product truth in the docs

- `docs/hop.md`: the intro's "The daemon follows" and the "What this does not
  do yet — Run a daemon" line are stale; the daemon shipped. Replace the
  intro's scope sentence with the full command list. Add, near the top, a
  short **Status** paragraph per D1: the hosted control plane is not yet
  open; the client ships so the protocol and the journeys are public and
  testable; point at "Choosing the control plane" for staging and self-hosted
  URLs. Nothing about price or trial.
- `README.md`: one Hop paragraph with the same status sentence; the platform
  sentence must not claim macOS for `v0.6.0`.
- `docs/cli-reference.md`: every Hop command and subcommand that
  `internal/cli/root.go` registers has an entry; `rein daemon`'s subcommands
  are listed; `--help` text and the reference agree (the doc gate reads
  `--help`).
- `docs/security-model.md`, `docs/getting-started.md`, `docs/faq.md`,
  `docs/features.md`, `docs/troubleshooting.md`: grep for "both platforms",
  "macOS and Windows", "verified on", "hosted", "sign in"; fix any sentence
  that would be false for `v0.6.0` under D1/D2.

## T-105 — `docs/compatibility.md` wording (after W3 merges)

Platform table: Windows 11 native row says `v0.6.0` verified; macOS arm64 row
says "verified through `v0.5.1`; `v0.6.0` acceptance deferred (ADR 0005)".
Range paragraphs carry "widened on native Windows evidence; macOS pending" for
the numbers W3 landed.

## Done when

Gate 1 and Gate 2 pass; the verifier finds no unsupported claim; the bullet
count did not drop; `git diff --stat` touches only owned files.
