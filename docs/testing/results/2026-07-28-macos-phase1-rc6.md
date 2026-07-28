# Phase 1 RC6 acceptance — Device A (macOS) report

Milestone reached: **M2 complete** (sections 10–11). M3 awaits a Windows `WINDOWS-RC6-W2-READY` report.

The row results in section 7 are **Device A evidence only**, captured at M1.
Device B has since reported its W1 leg; that report is validated in sections 10
and 11 but is deliberately not merged into section 7, because cross-device
reconciliation of all 21 rows happens at M4. Every row in section 7 that depends
on Windows therefore remains `NOT TESTED` here, never `PASS`.

This report supersedes `2026-07-27-macos-phase1-rc6.md` on this branch, which
recorded an earlier attempt at the same release that did not complete. That
attempt's isolated home, project, and profile were **not** reused; see finding 3.

This report contains no storage endpoint, bucket, access key, secret key,
passphrase, keyring value, transcript prose, ciphertext bytes, remote object
name, username, or absolute local path.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | 2026-07-28 |
| Release under test | `v0.1.0-rc.6` |
| Tag commit | `9019bd9cb4094eae648339dfecb2c6449c1b60d2` |
| Mac model / macOS version | macOS 26.5.2 (build 25F84) |
| Mac architecture | `arm64` |
| Native shell | `zsh` |
| Claude Code version | `2.1.220` (recognized range 2.1.219–2.1.220) |
| Codex CLI version | `0.145.0` (recognized range 0.133.0–0.145.0) |
| Git version | `2.55.0` |
| Reinstate version | `0.1.0-rc.6` (commit `9019bd9`) |
| GitHub check run | all required checks green (section 6) |
| Device A profile ID | `fd182697-957a-421f-8ee0-b45c18bf61a7` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc6` |
| Claude test session ID | `0eb4f696-c513-4bd8-8b80-8d9a8b964718` |
| Codex test session ID | `019fa608-ec57-7071-b6be-d8047004bbc9` |
| Windows edition/build | NOT TESTED (Device B) |

## 2. Release provenance (M0.1)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `v0.1.0-rc.6` is an annotated tag object | PASS | `git cat-file -t` → `tag` |
| Tag carries a signature block | PASS | one `BEGIN SSH SIGNATURE` block present |
| Tag commit reachable from `origin/main` | PASS | `git merge-base --is-ancestor` → true |
| Installed binary matches tag commit | PASS | `rein version --json` commit `9019bd9` |

Signature **verification** could not be completed: `git tag -v` exits non-zero
with `gpg.ssh.allowedSignersFile needs to be configured`. Per the prompt, trust
configuration was **not** changed to force a pass. The tag is annotated and
signed; cryptographic verification of the signer is `NOT TESTED`. Non-blocking
for RC6 acceptance, but it should be resolved before final sign-off.

## 3. Installer verification (M0.2)

All runs were unelevated; `sudo` appears zero times in either script layer.

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `https://reinstate.dev/install.sh` reachable | PASS | HTTP `200`, `content-length: 2398` |
| Pins exactly one release, `rc.6` | PASS | only version token in bootstrap is `v0.1.0-rc.6` |
| No `latest` resolution | PASS | 0 occurrences in bootstrap; the single occurrence in the stage-2 script is the refusal string `Refusing an unpinned latest release.` |
| Checksum layer 1 (bootstrap pins stage-2 installer) | PASS | `PINNED_INSTALLER_SHA256` equals the SHA-256 of `scripts/install.sh` at the tag, byte-for-byte |
| Checksum layer 2 (release asset) | PASS | stage 2 downloads `checksums.txt` and verifies the asset; run printed `checksum ok` |
| Fresh install produces `0.1.0-rc.6` | PASS | pre-existing binaries moved aside; installer printed `Installed reinstate v0.1.0-rc.6` and `Installed rein alias` |
| No elevation required | PASS | no prompt; `sudo` count 0 in both layers |
| Binaries resolve under `~/.local/bin` | PASS | `rein` and `reinstate` both resolve there |
| Idempotent re-run | PASS | second and third runs printed `Reinstate v0.1.0-rc.6 is already installed` |
| Installer is PATH-safe | PASS | installer added **zero** PATH lines across three runs — no `# Reinstate CLI` marker in `.zshrc`, `.zprofile`, `.bashrc`, or `.profile` |

**Non-blocking environment finding.** The operator's login shell already
resolves `~/.local/bin` twice, from two pre-existing user-owned profile lines
(`.zshrc` and `.zprofile`) that predate this run and carry no Reinstate marker.
The installer neither created nor duplicated an entry, so the installer-side
PATH gate passes; the duplication is pre-existing local configuration.

## 4. Environment and honest pre-init failure (M0.3, M0.4)

Isolated home and project were created fresh; `REINSTATE_HOME` was exported
explicitly for every Reinstate invocation and never unset or redirected.

Pre-init `rein setup check`:

```text
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: darwin-arm64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
exit=3
```

| Check | Result |
| ----- | ------ |
| Exit code `3` | PASS |
| Reports `config missing` | PASS |
| Device detection does not falsely fail | PASS (`darwin-arm64`) |
| Adapters do not falsely pass config | PASS (config is the only failing check) |

## 5. M1 — sessions, init safety, push, ciphertext

### 5.1 Source sessions

Both sessions were created non-interactively from the disposable project and
closed cleanly. Each was identified by a strict before/after diff of the vendor
session stores, then confirmed by counting **only** the exact marker string.
No transcript prose was read, printed, or summarized.

| Agent | New files after invocation | Exact marker occurrences | Session ID |
| ----- | -------------------------- | ------------------------ | ---------- |
| Claude Code | 1 | 4 | `0eb4f696-c513-4bd8-8b80-8d9a8b964718` |
| Codex CLI | 1 | 5 | `019fa608-ec57-7071-b6be-d8047004bbc9` |

Codex persisted a brand-new rollout; no older acceptance session was reused.

### 5.2 Setup-prompt outcomes and first-device init

The tagged Claude Code setup prompt is **Prompt version 6** and requires the
agent to detect and preserve an already-exported `REINSTATE_HOME`. That was
satisfied: the isolated home was exported before any Reinstate command and every
command ran against it. Storage coordinates were supplied only in each child
process environment; the passphrase was supplied only through an anonymous pipe
named by `REINSTATE_PASSPHRASE_FD`. No secret ever appeared in argv, the parent
environment, or a temporary plaintext file.

| Step | Result | Evidence |
| ---- | ------ | -------- |
| `rein init --yes --project local/reinstate-phase1-acceptance-rc6=<redacted>` | PASS | exit 0; fresh `profile_id=fd182697-957a-421f-8ee0-b45c18bf61a7` |
| `rein setup check` | PASS | exit 0, `all checks passed`, both adapters `SUPPORTED` |
| `rein doctor --self-test` | PASS | exit 0, `self_test: synthetic self-test passed` |

### 5.3 Physical F1 — default refusal to re-initialize

Re-ran the identical init **without** `--force`:

```text
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
exit=7
```

| Assertion | Result |
| --------- | ------ |
| Exit code `7` | PASS |
| `config.toml` unchanged (SHA-256 compared) | PASS |
| `state.json` unchanged (SHA-256 compared) | PASS |
| No new backup created (0 → 0) | PASS |

### 5.4 Dry-run and push (selected IDs only, never `--all`)

| Command | Output | Result |
| ------- | ------ | ------ |
| `push --agent claude --session <claude> --dry-run` | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true` | PASS |
| `push --agent claude --session <claude>` | `pushed 1 snapshot(s), skipped 0 unchanged` | PASS |
| `push --agent codex --session <codex> --dry-run` | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true` | PASS |
| `push --agent codex --session <codex>` | `pushed 1 snapshot(s), skipped 0 unchanged` | PASS |

Neither dry-run uploaded anything, and each said `would push`, never `pushed`.

`rein status` with the correct passphrase reports exactly the two selected
sessions and nothing else:

```text
remote revision: f415705a-fe83-4664-a685-03370c07dddd (2 sessions)
  claude:0eb4f696-... -> b21abf18-8262-41cc-a979-5c0868d38e27
  codex:019fa608-...  -> f415705a-fe83-4664-a685-03370c07dddd
exit=0
```

### 5.5 Ciphertext-only remote storage

The fresh profile prefix was inspected through a purpose-built, scoped
SigV4 S3 client that reads the storage coordinates at runtime from the private
file. No provider web UI was used. Remote object names are not reproduced here.

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| List succeeded against the fresh prefix only | PASS | HTTP `200` |
| Object inventory | PASS | `object_count=3` — one `manifest.age` + `snapshot_age_count=2` |
| Every object is `.age` | PASS | `all_objects_end_with_.age=true` |
| Snapshot names are opaque UUIDs | PASS | matched `snapshots/<uuid>.age` |
| No auth/token/credential/`.env`/plaintext-shaped object | PASS | `forbidden_shaped_objects=0` |

One snapshot was downloaded to an owner-only (`0600`) temporary file and tested
without printing any bytes:

| Probe | Result |
| ----- | ------ |
| `grep -aFq 'REINSTATE-PHASE1-RC6-MAC-CLAUDE-A1'` | exit `1` — **absent** |
| `grep -aFq 'REINSTATE-PHASE1-RC6-MAC-CODEX-A1'` | exit `1` — **absent** |
| `grep -aFq 'REINSTATE-PHASE1-RC6'` (any marker prefix) | exit `1` — **absent** |
| `grep -aFq '"role"'` (JSONL shape) | exit `1` — **absent** |
| `file` | `data` (not readable JSON/JSONL) |
| age header present | true |
| Decodes as UTF-8 | false (printable ratio `0.389`) |

The local download was deleted; no remote object was deleted or mutated.

### 5.6 Supplementary Mac-side evidence

These two gates are formally assigned to Device B or to M3. They were exercised
on the Mac as corroboration only and are **not** counted as complete.

Wrong passphrase (generated in memory; the private file was not modified —
verified by SHA-256 before and after):

```text
decrypt: identity did not match any of the recipients: incorrect identity for recipient block: incorrect passphrase
exit=4
```

`config.toml`, `state.json`, and backup count were all unchanged.

Unchanged no-op push, both agents:

```text
pushed 0 snapshot(s), skipped 1 unchanged, dry_run=false
```

Remote revision stayed `f415705a-...` and `object_count` stayed `3`, proving no
new snapshot or revision was created.

## 6. Automated integrity gates (tagged runbook section 18)

All check runs on tag commit `9019bd9` are green:

```text
success  Build & release       success  Test (macos-latest)
success  CodeQL                success  Test (ubuntu-latest)
success  Lint                  success  Test (windows-latest)
success  Secret scan           success  Website
success  Security              success  Workflow permission and pin review
skipped  Dependency review
```

Section 18 sub-gates were mapped to the jobs that actually execute them:

| Section 18 requirement | Covered by | Result |
| ---------------------- | ---------- | ------ |
| Go tests on Ubuntu, macOS, Windows | `Test (ubuntu/macos/windows-latest)` | PASS |
| Native Windows bootstrap execution and PATH behavior | `TestWindowsPublicBootstrapContract` on windows-latest | PASS |
| POSIX bootstrap behavior and hash-mismatch refusal | `TestPOSIXPublicBootstrapContract`, `TestPOSIXPublicBootstrapRejectsInstallerHashMismatch` | PASS |
| Exact-tag and no-`latest` static contracts | `TestPublicBootstrapStaticContract` | PASS |
| Website `npm ci`, tests, production build | `Website` job | PASS |
| Byte-for-byte script inclusion in build output | `Verify public installer assets` (`cmp` both scripts) | PASS |
| Lint, race, docs, fixture secret scan, vulnerability | `Lint`, `Race` step, `internal/doctest`, `Secret scan`, `govulncheck` | PASS |

`Dependency review` is `skipped` because the run was a push event, not a pull
request. That is expected and is not a missing required check.

Local regression run at the tag commit: `go test ./...` exit 0 (20 packages ok,
0 failed) and `go vet ./...` exit 0.

## 7. Mandatory sign-off checklist (all 21 rows)

`PARTIAL` means the Device A half is evidenced and the Device B half is
outstanding. Nothing Windows-dependent is marked `PASS`.

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC6 on Mac | PASS | §3 — HTTP 200, both checksum layers, `0.1.0-rc.6` installed unelevated |
| 2 | `install.ps1` returns 200 and installs RC6 on Windows | NOT TESTED | Device B |
| 3 | Both installers are idempotent and PATH-safe | PARTIAL | §3 — Mac idempotent, installer added 0 PATH entries; Windows outstanding |
| 4 | Pre-init missing-config failure is accurate | PARTIAL | §4 — Mac exit 3 `config missing`; Windows outstanding |
| 5 | Post-init setup check and self-test pass on both devices | PARTIAL | §5.2 — Mac both exit 0; Windows outstanding |
| 6 | Claude setup prompt completes on the Mac | PASS | §5.2 — Prompt version 6 outcomes, isolated home preserved |
| 7 | Codex setup prompt completes on Windows | NOT TESTED | Device B |
| 8 | Only two selected test sessions reach the remote manifest | PASS | §5.4 — `2 sessions`, `--all` never used |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | §5.5 — 4 absence probes, `file`=`data`, non-UTF-8 |
| 10 | Wrong passphrase fails without mutation | PARTIAL | §5.6 — Mac exit 4, no mutation; mandated Windows execution outstanding |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| 13 | Active-agent overwrite is refused | NOT TESTED | Device B (W2) |
| 14 | Existing Windows target is backed up before restore | NOT TESTED | Device B (W2) |
| 15 | Claude Windows-to-Mac resume succeeds | NOT TESTED | Mac M3, needs Windows B1 push |
| 16 | Codex Windows-to-Mac resume succeeds | NOT TESTED | Mac M3, needs Windows B1 push |
| 17 | Existing Mac targets are backed up before restore | NOT TESTED | Mac M3 |
| 18 | Unchanged pushes skip without new snapshots | PARTIAL | §5.6 — Mac no-op verified, revision and object count unchanged; post-restore variant is M3 |
| 19 | Divergence records a conflict without overwrite | NOT TESTED | M4 / W3 |
| 20 | `--keep-both` preserves both branches | NOT TESTED | Device B (W3) |
| 21 | All required GitHub checks are green | PASS | §6 |

**Counts: 5 PASS / 5 PARTIAL / 0 FAIL / 11 NOT TESTED.**

All 21 mandatory rows passed: **No.** Phase 1 remains open pending Device B.

## 8. Findings

No release-blocking findings on Device A.

Non-blocking, recorded for sign-off:

1. **Tag signature not cryptographically verified.** `git tag -v v0.1.0-rc.6`
   fails because `gpg.ssh.allowedSignersFile` is unset. Trust configuration was
   deliberately not modified to force a pass. Configure an allowed-signers file
   before final sign-off.
2. **Pre-existing duplicate `~/.local/bin` PATH entry** from two operator-owned
   profile lines. Not installer-caused; the installer added zero entries.
3. **Prior aborted RC6 state was present and was archived, not deleted.** Both
   isolated paths already existed, and the pre-existing home already contained a
   `profile_id`. Reusing it would have invalidated the "fresh RC6 profile"
   requirement, and deleting it is forbidden, so both were renamed aside with an
   `.aborted-<UTC timestamp>` suffix and left intact. This run used a genuinely
   fresh home, project, profile, and remote prefix.
4. **The private credentials file did not match the documented schema and was
   normalized.** It used the key names `endpoint`, `bucket`, `access_key`,
   `access_secret`, `passphrase`, and `passphrase_rc6` instead of the five
   contract keys, and carried **two** passphrase fields. The underlying values
   were structurally valid (HTTPS endpoint with no path suffix, bucket separate
   from endpoint). The file was rewritten into the exact contract schema with
   values copied verbatim, the RC6-scoped passphrase field selected, owner-only
   mode `0600` applied, and the original preserved as a `0600` backup. **This
   directly affects the Windows handoff** — see the handoff note below.

## 9. Repository hygiene

- Branch `test/phase1-rc6-macos-report`, created from the peeled RC6 tag commit
  `9019bd9`, in a dedicated worktree so no product branch was touched.
- The private credentials file is listed in `.git/info/exclude`, is untracked,
  and is absent from the staged diff. It was also relocated out of the
  repository working tree entirely.
- The only change on this branch is this report. No product code, workflow,
  script, or documentation file was modified.
- Nothing was merged, tagged, released, or deployed.

## 10. M2 gate — Windows W1 handoff validation

A Windows report was received and validated. **Milestone M2 was not executed**,
because the Mac M2 trigger is a `WINDOWS-RC6-W1-PASS` report and the supplied
block is `WINDOWS-RC6-W1-PARTIAL`. Nothing was relabelled to force the gate.

### 10.1 Handoff integrity — validated

| Check | Result |
| ----- | ------ |
| Branch `test/phase1-rc6-windows-report` exists on origin | PASS |
| Commit `7cfd1ba218adfd41d9c91616e020e6dad2ecc1f8` resolves | PASS |
| Draft PR #33 open against `main`, not merged | PASS |
| Branch changes only `docs/testing/results/` (0 product files) | PASS |
| Report discloses no credential value from the private file | PASS |
| No Windows username path, ciphertext, or transcript JSON in report | PASS |
| `profile_id` matches Device A | PASS (`fd182697-…`) |
| `claude_session_id` / `codex_session_id` match Device A | PASS |
| Independently re-verified remote parity | PASS |

Device A independently confirmed the remote after the Windows leg: revision
`f415705a-…` unchanged, the same two sessions and snapshot IDs, `object_count=3`,
`forbidden_shaped_objects=0`. Windows completed W1 without mutating remote state.

### 10.2 Why the gate did not open

Three gates are `PARTIAL` (7, 11, 12). Each was downgraded for using the
**agent** method instead of the **human** method — but the automation overlay
requires the agent method. The committed prompts document states:

> where the tagged runbook assigns a command, hidden input, marker check,
> storage inspection, or evidence capture to a human, the device-owning agent
> performs it instead … the change of operator from human to agent does not
> [stop the run].

| Gate | Stated reason for PARTIAL | Overlay position |
| ---- | ------------------------- | ---------------- |
| 7 | Hidden-prompt method replaced by the automated provider | The overlay **mandates** that replacement: passphrase input is accepted "only through `REINSTATE_PASSPHRASE_FD`". Downgraded for complying. |
| 11 | No visual A1 confirmation for Claude | Prompt 2 step 14 assigns the agent: "Verify only the two exact A1 markers"; safety rules authorize "exact acceptance-marker counts". |
| 12 | No visual A1 confirmation for Codex | Same as gate 11. |

The Windows report's blocker states "a human was explicitly unavailable, so
hidden-prompt execution and visual transcript-marker confirmation could not
occur." Under the overlay neither is required: an exact-occurrence count on the
**restored file on disk** is an authorized, human-free operation. Device A used
exactly that method for its own marker evidence (§5.1).

This is a documented **method** deviation, not a product defect. No Reinstate
product blocker appeared on either device through W1.

### 10.3 Sequencing

M2 mutates the Mac Claude session (appends `A2`) and advances the remote Claude
snapshot. Gates 11 and 12 are the only evidence that the encrypted round trip
preserved **content**, and they are the baseline W2's `A2` verification builds
on. Closing them before the remote advances is the correct order, so Device A
holds rather than pushing `A2` onto an unverified baseline.

Device A remains paused. Nothing in this section changes the Device A row
results in §7; the counts there are unchanged.

## 11. M2 — corrected Windows W1-PASS and Mac A2 push

Device B re-validated its own gates 7, 11, and 12 under the automation overlay
and re-issued `WINDOWS-RC6-W1-PASS` at commit
`deca8217c3af680ed8fdc176e76c3c9dd5cec4e8`. The M2 precondition is now met and
M2 was executed.

### 11.1 Re-validated Windows handoff

| Check | Result |
| ----- | ------ |
| Commit `deca8217…` resolves; branch tip matches | PASS |
| Draft PR #33 open, not merged | PASS |
| Branch changes only `docs/testing/results/` (0 product files) | PASS |
| No credential value, Windows username path, ciphertext, or transcript JSON | PASS |
| Block complete with `END-WINDOWS-RC6-W1` terminator | PASS |
| `release`, `claude_discovery_and_resume`, `codex_resume`, `windows_report_path` present | PASS |
| Windows counts | 13 PASS / 0 PARTIAL / 0 FAIL / 8 NOT TESTED |

**Round-trip content integrity is now independently corroborated.** The restored
Windows marker counts (Claude `4/4`, Codex `5/5`) match the Device A source
counts recorded in §5.1 exactly. The encrypted push → remote → pull path
preserved session content across operating systems with no drift. Windows also
reported zero `A2` and zero `B1` occurrences, confirming it verified an
uncontaminated A1 baseline.

### 11.2 Mac A2 append and push

The exact Claude session was resumed non-interactively and only the `A2` marker
was added. Nothing else was modified, and no restored file was hand-moved.

| Assertion | Before | After | Result |
| --------- | ------ | ----- | ------ |
| Same session file mutated in place | — | same path | PASS |
| `A1` occurrences preserved | 4 | 4 | PASS |
| `A2` occurrences | 0 | 4 | PASS |
| Session file size (bytes) | 10992 | 13501 | grew, not replaced |
| Stray new Claude session files created | — | 0 | PASS |
| Codex session untouched (`A1`=5, `A2`=0) | — | unchanged | PASS |

Push of that single ID:

```text
push --agent claude --session <claude> --dry-run
  would push 1 snapshot(s), would skip 0 unchanged, dry_run=true
push --agent claude --session <claude>
  pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false
```

The dry-run said `would push`, never `pushed`, and uploaded nothing.

### 11.3 Remote state after M2

| Field | Before M2 | After M2 |
| ----- | --------- | -------- |
| Remote revision | `f415705a-fe83-4664-a685-03370c07dddd` | `192a98c8-61da-4c14-9d15-525c141815b3` |
| Claude snapshot | `b21abf18-8262-41cc-a979-5c0868d38e27` | `192a98c8-61da-4c14-9d15-525c141815b3` |
| Codex snapshot | `f415705a-fe83-4664-a685-03370c07dddd` | unchanged |
| Session count | 2 | 2 |
| Object count | 3 | 4 (`snapshot_age_count=3`) |

Only the Claude session advanced. Ciphertext discipline still holds on the new
snapshot: `all_objects_end_with_.age=true`, `forbidden_shaped_objects=0`, the
`A2` marker and the `REINSTATE-PHASE1-RC6` prefix are both **absent** from the
downloaded bytes, and `file` reports `data`. The local download was deleted.

Device A is paused for the report transfer to Windows.

## 12. Milestone block

```text
MAC-RC6-M1
release=v0.1.0-rc.6
tag_commit=9019bd9cb4094eae648339dfecb2c6449c1b60d2
profile_id=fd182697-957a-421f-8ee0-b45c18bf61a7
canonical_project_id=local/reinstate-phase1-acceptance-rc6
claude_session_id=0eb4f696-c513-4bd8-8b80-8d9a8b964718
codex_session_id=019fa608-ec57-7071-b6be-d8047004bbc9
remote_session_count=2
f1_default_refusal=PASS
ciphertext_marker_absence=PASS
mac_report_path=docs/testing/results/2026-07-28-macos-phase1-rc6.md
END-MAC-RC6-M1
```

The earlier `MAC-RC6-M2-BLOCKED` status (section 10) is superseded: Device B
re-issued a valid `WINDOWS-RC6-W1-PASS`, so M2 ran.

```text
MAC-RC6-M2-READY
release=v0.1.0-rc.6
profile_id=fd182697-957a-421f-8ee0-b45c18bf61a7
claude_session_id=0eb4f696-c513-4bd8-8b80-8d9a8b964718
mac_claude_a2_marker=REINSTATE-PHASE1-RC6-MAC-CLAUDE-A2
a2_occurrences=4
a1_occurrences_preserved=4
new_remote_revision=192a98c8-61da-4c14-9d15-525c141815b3
new_claude_snapshot_id=192a98c8-61da-4c14-9d15-525c141815b3
codex_snapshot_id=f415705a-fe83-4664-a685-03370c07dddd
remote_session_count=2
remote_object_count=4
ciphertext_marker_absence=PASS
windows_w1_validated=PASS
windows_commit=deca8217c3af680ed8fdc176e76c3c9dd5cec4e8
mac_report_path=docs/testing/results/2026-07-28-macos-phase1-rc6.md
END-MAC-RC6-M2-READY
```
