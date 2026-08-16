# W7 — Release and acceptance cards

Maintainer and coordinator. Executors cannot complete these: device reports are
physical work on real machines with real agents.

Process: [../../../../RELEASING.md](../../../../RELEASING.md), unchanged.
Matrix: [../../../testing/phase-5-universal-agent-coverage-acceptance.md](../../../testing/phase-5-universal-agent-coverage-acceptance.md).

---

## T-070 — Release commit

**Owns:** `CHANGELOG.md`, version pins, `CITATION.cff`

1. Convert the `[Unreleased]` entries into a `v0.5.0` section, grouped by tier
   change rather than by file.
2. Confirm the tier census in the shipped catalog matches
   `docs/compatibility.md` and the website data.
3. Verify the release gate from
   [../agent-roster.md](../agent-roster.md): catalog refactor landed, probe
   shipped, at least six new agents at T1, at least three at T2.
4. If the counts are not met, **shrink the release scope**. Do not inflate a
   tier to reach a number.
5. Run `make verify`, the full race suite, and the website suite.

---

## T-071 — Candidate tag and dispatch

1. Signed tag `v0.5.0-rc.N`.
2. Draft release, verify assets, checksums, and attestations, then publish as a
   prerelease.
3. Update the live installers to pin the exact candidate tag.
4. **Dispatch only after the live installers pin the tag.** Testing an artifact
   the installers do not serve tests the wrong thing.
5. Generate and record the acceptance row count with
   `rein doctor --agents --acceptance-matrix`. Both device reports cite it.

---

## T-072 — macOS device report

**Platform:** Apple Silicon macOS, native arm64.

1. Install from the candidate tag and verify the full artifact chain before
   running anything.
2. Complete every matrix in the acceptance contract that applies to the
   installed agents.
3. Record `ABSENT` honestly for agents you do not have. Absence reduces scope;
   it does not fail a row.
4. Commit the report on its own branch and add the terminated device block.
5. This report is immutable once the terminated block is committed.

---

## T-073 — Windows device report

**Platform:** native Windows x64, not WSL.

Same as T-072. If WSL2 is also tested, it is a third device with its own
report, because native Windows and WSL2 are different devices with different
agent trees.

---

## T-074 — Reconciliation and stable promotion

**macOS coordinator report only**, after independently verifying both immutable
device-report commits.

1. Confirm both reports cite the same tag and full commit.
2. **Apply every tier reduction from either device.** A tier verified on one
   platform and not the other ships at the lower tier. It does not ship higher
   with a footnote.
3. Confirm zero release-blocking findings.
4. Complete the reconciliation block and record whether stable `v0.5.0` is
   authorized.
5. On authorization: signed `v0.5.0` tag, GitHub release, then the signed
   website tag and deploy.
6. Update `ROADMAP.md` to mark Phase 5 closed, with the candidate and the two
   report paths cited, in the style Phases 3 and 4 use.

**If reconciliation fails**, the candidate is dead. Fix, cut a new candidate,
and run both matrices again. A candidate is never partially promoted, and a
failed candidate is never re-tested after a fix without a new tag.
