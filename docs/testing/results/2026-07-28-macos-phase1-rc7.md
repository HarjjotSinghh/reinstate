# Phase 1 RC7 acceptance — Device A (macOS) report

Milestone reached: **M2 complete** (section 11). Device B has passed W1; W2 and W3 are outstanding, so
every Windows-dependent gate is `NOT TESTED`, never `PASS`.

This is a clean RC7 run. No RC6 home, project, profile, passphrase, marker
session, remote prefix, or report was reused. The RC6 acceptance state was left
in place untouched and is not evidence here.

This report contains no storage endpoint, bucket, access key, secret key,
passphrase, keyring value, transcript prose, ciphertext bytes, remote object
name, username, or absolute local path.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | 2026-07-28 |
| Release under test | `v0.1.0-rc.7` |
| Tag commit | `66211599dd7cfb74f1436d2221b983050e8b1bc2` |
| Mac model / macOS version | macOS 26.5.2 (build 25F84) |
| Mac architecture | `arm64` |
| Native shell | `zsh` |
| Claude Code version | `2.1.220` (recognized range 2.1.219–2.1.220) |
| Codex CLI version | `0.145.0` (recognized range 0.133.0–0.145.0) |
| Git version | `2.55.0` |
| Reinstate version | `0.1.0-rc.7` (commit `6621159`) |
| GitHub check run | all required checks green (section 7) |
| Device A profile ID | `2949d464-03f4-4de1-b326-2b3072bcb2a5` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc7` |
| Claude test session ID | `1cf4ab6d-3e36-424d-8f30-4f41858b7f20` |
| Codex test session ID | `019fa82a-2b87-71d2-947d-a8146d3049fd` |
| Windows edition/build | NOT TESTED (Device B) |

## 2. Release provenance (M0.1)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `v0.1.0-rc.7` is an annotated tag object | PASS | `git cat-file -t` → `tag` |
| Signature verifies | **PASS** | `Good "git" signature for the release identity`, checked against the repository's committed `.github/allowed_signers` |
| Signing key matches the RC6 release identity | PASS | same ED25519 key fingerprint as the key that signed `v0.1.0-rc.6` |
| Tag commit reachable from `origin/main` | PASS | `git merge-base --is-ancestor` → true |
| Installed binary matches tag commit | PASS | `rein version --json` commit `6621159` |

This is stronger than the RC6 run, where signature verification was recorded as
`NOT TESTED` because no allowed-signers file was configured. Verification here
used the trust anchor already committed to the repository; no local trust
configuration was created or modified to force a pass.

## 3. Installer verification (M0.2)

All runs were unelevated; `sudo` appears zero times in either script layer.

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `https://reinstate.dev/install.sh` reachable | PASS | HTTP `200`, 2398 bytes |
| Pins exactly one release, `rc.7` | PASS | the only version token in the bootstrap is `v0.1.0-rc.7` |
| No `latest` resolution | PASS | 0 occurrences in the bootstrap |
| Checksum layer 1 (bootstrap pins stage-2 installer) | PASS | pinned SHA-256 equals the SHA-256 of `scripts/install.sh` at the tag, byte for byte |
| Checksum layer 2 (release asset) | PASS | stage 2 fetches `checksums.txt` and verifies; the run printed `checksum ok` |
| Fresh install produces `0.1.0-rc.7` | PASS | pre-existing binaries moved aside; installer printed `Installed reinstate v0.1.0-rc.7` and `Installed rein alias` |
| No elevation required | PASS | no prompt; `sudo` count 0 in both layers |
| Binaries resolve under `~/.local/bin` | PASS | `rein` and `reinstate` both resolve there |
| Idempotent re-run | PASS | second run printed `Reinstate v0.1.0-rc.7 is already installed` |
| Installer is PATH-safe | PASS | installer added **zero** PATH lines — no `# Reinstate CLI` marker in any shell profile |

**Non-blocking environment finding (unchanged from RC6).** The operator's login
shell already resolves `~/.local/bin` twice from two pre-existing user-owned
profile lines that carry no Reinstate marker. The installer neither created nor
duplicated an entry.

## 4. Environment and honest pre-init failure (M0.3, M0.4)

The isolated home and project were created fresh. `REINSTATE_HOME` was exported
explicitly for every Reinstate invocation and never unset or redirected.

```text
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: darwin-arm64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
exit=3
```

Exit `3`, `config missing`, no false device or adapter pass.

## 5. M1 — sessions, init safety, push, ciphertext

### 5.1 Source sessions

Both sessions were created non-interactively from the disposable project and
closed cleanly, identified by a strict before/after diff of the vendor session
stores and confirmed by counting **only** the exact marker string. No transcript
prose was read, printed, or summarized.

| Agent | New files after invocation | Exact marker occurrences | Session ID |
| ----- | -------------------------- | ------------------------ | ---------- |
| Claude Code | 1 | 4 | `1cf4ab6d-3e36-424d-8f30-4f41858b7f20` |
| Codex CLI | 1 | 5 | `019fa82a-2b87-71d2-947d-a8146d3049fd` |

Codex persisted a brand-new rollout; no older acceptance session was reused.

### 5.2 Setup-prompt outcomes and first-device init

Storage coordinates were supplied only in each child process environment; the
passphrase only through an anonymous pipe named by `REINSTATE_PASSPHRASE_FD`. No
secret appeared in argv, the parent environment, or a temporary plaintext file.

| Step | Result | Evidence |
| ---- | ------ | -------- |
| `rein init --yes --project local/reinstate-phase1-acceptance-rc7=<redacted>` | PASS | exit 0; fresh `profile_id=2949d464-03f4-4de1-b326-2b3072bcb2a5` |
| `rein setup check` | PASS | exit 0, `all checks passed`, both adapters `SUPPORTED` |
| `rein doctor --self-test` | PASS | exit 0, `self_test: synthetic self-test passed` |

`init` persisted the new restore policy with its default:

```toml
[restore]
  active_agent_policy = "fork"
```

### 5.3 Physical F1 — default refusal to re-initialize

```text
reinstate home is already initialized; rerun init with --force to back up and replace existing config/state
exit=7
```

`config.toml` and `state.json` unchanged (SHA-256 compared); backup count 0 → 0.

### 5.4 Dry-run and push (selected IDs only, never `--all`)

| Command | Output | Result |
| ------- | ------ | ------ |
| `push --agent claude --session <claude> --dry-run` | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true` | PASS |
| `push --agent claude --session <claude>` | `pushed 1 snapshot(s), skipped 0 unchanged` | PASS |
| `push --agent codex --session <codex> --dry-run` | `would push 1 snapshot(s), would skip 0 unchanged, dry_run=true` | PASS |
| `push --agent codex --session <codex>` | `pushed 1 snapshot(s), skipped 0 unchanged` | PASS |

`rein status` reports exactly the two selected sessions:

```text
remote revision: 4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574 (2 sessions)
  claude:1cf4ab6d-... -> 75c7ae2c-643b-421f-bbc4-aedb068d7f96
  codex:019fa82a-...  -> 4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574
```

### 5.5 Ciphertext-only remote storage

Inspected through a purpose-built scoped SigV4 S3 client reading the storage
coordinates at runtime. No provider web UI was used. Remote object names are not
reproduced.

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| List succeeded against the fresh prefix only | PASS | HTTP `200` |
| Object inventory | PASS | `object_count=3` — one `manifest.age` + `snapshot_age_count=2` |
| Every object is `.age` | PASS | `all_objects_end_with_.age=true` |
| Snapshot names are opaque UUIDs | PASS | matched `snapshots/<uuid>.age` |
| No auth/token/credential/`.env`/plaintext-shaped object | PASS | `forbidden_shaped_objects=0` |

One snapshot (47352 bytes) was downloaded to an owner-only (`0600`) temporary
file and tested without printing any bytes:

| Probe | Result |
| ----- | ------ |
| `grep -aFq 'REINSTATE-PHASE1-RC7-MAC-CLAUDE-A1'` | exit `1` — **absent** |
| `grep -aFq 'REINSTATE-PHASE1-RC7-MAC-CODEX-A1'` | exit `1` — **absent** |
| `grep -aFq 'REINSTATE-PHASE1-RC7'` (any marker prefix) | exit `1` — **absent** |
| `grep -aFq '"role"'` (JSONL shape) | exit `1` — **absent** |
| `file` | `data` |
| age header present | true |
| Decodes as UTF-8 | false (printable ratio `0.390`) |

The local download was deleted; no remote object was deleted or mutated.

An earlier attempt at this probe wrote its download to the wrong directory, so
the four `grep` calls returned exit `2` (file not found) rather than exit `1`
(marker absent). That run proved nothing and was discarded; the destination was
corrected and the probe rerun in full to produce the table above.

## 6. RC7 restore-safety behavior (the change under test)

Device A is the ideal adversarial environment for this: **1 `claude` and 6
`codex` processes belonging to unrelated work were running throughout**. Under
RC6 that state alone forced `exit 7` on any restore, which is precisely what
blocked the RC6 Mac run at M3.

### 6.1 Unrelated running agents no longer block a restore

```text
rein pull --agent codex --session <codex> --dry-run
  would pull 1 snapshot(s), dry_run=true
  exit=0

rein pull --agent codex --session <codex>
  pulled 1 snapshot(s), dry_run=false
  exit=0
```

| Assertion | Result |
| --------- | ------ |
| Real pull succeeded with 6 unrelated Codex processes alive | PASS (exit 0) |
| Session content preserved | PASS (marker count 5 → 5) |
| Existing target backed up before restore | PASS (backups 0 → 1) |
| Dry-run said `would pull`, never `pulled` | PASS |

### 6.2 The policy is real, not merely disabled detection

With `restore.active_agent_policy = "strict"` and the same processes running:

```text
codex appears to be running and this host cannot tell which session it is using; close it or rerun with --allow-active-agents
exit=7
```

Session file unchanged, no new backup. The message correctly reports the
unscoped basis, because `strict` deliberately discards the target path and asks
the host-wide question. The policy was restored to `fork` afterwards.

Taken together these show the guard still exists and still refuses when asked
to, while the default no longer punishes a developer for having other agents
open.

### 6.3 Not yet proven on this device

The forked-restore path (gate 14) needs an agent genuinely holding the **target**
session file. That condition belongs to Device B section 14b and was not
manufactured here. `scoped`-with-held-file (gate 15) is likewise Device B.

## 7. Automated integrity gates (tagged runbook section 18)

All check runs on tag commit `6621159` are green:

```text
success  Build & release      success  Test (macos-latest)
success  CodeQL               success  Test (ubuntu-latest)
success  Dependabot           success  Test (windows-latest)
success  Lint                 success  Validate website and CLI tags
success  Secret scan          success  Website
success  Security             success  Workflow permission and pin review
skipped  Dependency review
```

`Dependency review` is skipped on push events, as expected.

Section 18 sub-gates map to the jobs that execute them: Go tests on all three
platforms; `TestWindowsPublicBootstrapContract` and its hash-mismatch companion
on windows-latest; `TestPOSIXPublicBootstrapContract` and its hash-mismatch
companion; `TestPublicBootstrapStaticContract` for exact-tag/no-`latest`;
the `Website` job for `npm ci`, tests and production build; `Verify public
installer assets` for byte-for-byte script inclusion; and `Lint`, the race step,
`internal/doctest`, `Secret scan`, and `govulncheck`.

Live production routes were additionally verified byte-for-byte against the tag:
`https://reinstate.dev/install.sh` and `install.ps1` both match `v0.1.0-rc.7`.

## 8. Mandatory sign-off checklist (all 23 rows)

`PARTIAL` means the Device A half is evidenced and the Device B half is
outstanding. Nothing Windows-dependent is marked `PASS`.

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC7 on Mac | PASS | §3 |
| 2 | `install.ps1` returns 200 and installs RC7 on Windows | NOT TESTED | Device B |
| 3 | Both installers are idempotent and PATH-safe | PARTIAL | §3 — Mac idempotent, 0 installer-added PATH entries |
| 4 | Pre-init missing-config failure is accurate | PARTIAL | §4 — Mac exit 3 |
| 5 | Post-init setup check and self-test pass on both devices | PARTIAL | §5.2 — Mac both exit 0 |
| 6 | Claude setup prompt completes on the Mac | PASS | §5.2 |
| 7 | Codex setup prompt completes on Windows | NOT TESTED | Device B |
| 8 | Only two selected test sessions reach the remote manifest | PASS | §5.4 |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | §5.5 |
| 10 | Wrong passphrase fails without mutation | PARTIAL | Mac exit 4, decryption refusal, no config/backup change; mandated Windows execution outstanding |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| 13 | Unrelated running agents do not block a restore | **PASS** | §6.1 — exit 0 with 6 unrelated Codex processes alive |
| 14 | A live session is forked, never overwritten | NOT TESTED | §6.3 — Device B 14b |
| 15 | `scoped` policy still refuses, naming that session | PARTIAL | §6.2 — `strict` refusal proven on Mac; `scoped`-with-held-file is Device B |
| 16 | Existing Windows target is backed up before restore | NOT TESTED | Device B |
| 17 | Claude Windows-to-Mac resume succeeds | NOT TESTED | Mac M3 |
| 18 | Codex Windows-to-Mac resume succeeds | NOT TESTED | Mac M3 |
| 19 | Existing Mac targets are backed up before restore | PASS | §6.1 — backups 0 → 1 before the restore |
| 20 | Unchanged pushes skip without new snapshots | PASS | `pushed 0 snapshot(s), skipped 1 unchanged` for both agents |
| 21 | Divergence records a conflict without overwrite | NOT TESTED | M4 / W3 |
| 22 | `--keep-both` preserves both branches | NOT TESTED | Device B |
| 23 | All required GitHub checks are green | PASS | §7 |

**Counts: 8 PASS / 5 PARTIAL / 0 FAIL / 10 NOT TESTED.**

All 23 mandatory rows passed: **No.** Phase 1 remains open pending Device B.

## 9. Findings

No release-blocking findings on Device A.

Non-blocking, recorded for sign-off:

1. **Pre-existing duplicate `~/.local/bin` PATH entry** from two operator-owned
   profile lines. Not installer-caused.
2. **The Windows Restart Manager path is still unexercised on real hardware.**
   It is compile-verified and passes `Test (windows-latest)`, but no test yet
   holds a file open on Windows and asserts that Restart Manager reports it.
   Device B section 14b is the gate that would prove it, and it is the highest
   remaining risk in this release.

## 10. Repository hygiene

- Branch `test/phase1-rc7-macos-report`, created from the peeled RC7 tag commit
  `6621159`, in a dedicated worktree; no product branch was touched.
- The private credentials file is listed in `.git/info/exclude`, is untracked,
  is absent from the staged diff, and lives outside the repository entirely. Its
  SHA-256 was compared before and after the wrong-passphrase test and was
  unchanged.
- The only change on this branch is this report.
- Nothing was merged, tagged, released, or deployed by this run.

## 11. M2 — Windows W1 validation and Mac A2 push

Device B issued `WINDOWS-RC7-W1-PASS` at commit
`82c324ce706ef9dc1a736e3c4bfa78851b75ba19`, after an earlier
`WINDOWS-RC7-W1-FAIL` that stopped before W0 on a private-file schema
precondition. That first attempt established no product defect, mutated nothing,
and is not evidence here.

### 11.1 Re-validated handoff

| Check | Result |
| ----- | ------ |
| Commit `82c324ce…` resolves; branch tip matches | PASS |
| Draft PR #53 open, not merged | PASS |
| Branch changes only `docs/testing/results/` (0 product files) | PASS |
| No credential value from the private file | PASS |
| No Windows username or absolute path, no transcript JSON, no ciphertext | PASS |
| Block complete with `END-WINDOWS-RC7-W1` terminator | PASS |
| `profile_id`, `claude_session_id`, `codex_session_id` match Device A | PASS |
| Windows counts | 16 PASS / 1 PARTIAL / 0 FAIL / 6 NOT TESTED |

Device A independently re-verified the remote before touching anything: revision
`4ffdb7ea-…` unchanged, the same two sessions and snapshot IDs. Windows completed
W1 without mutating remote state.

**Round-trip content integrity is corroborated from both ends.** The restored
Windows target reported `A1_marker_count_before=4` and
`A1_marker_count_after=4`, matching the Device A source count in section 5.1
exactly. The encrypted push → remote → pull path preserved session content
across operating systems with no drift.

Device B recorded a non-destructive conflict during its own retry, resolved with
`--keep-remote` and one preserved backup, and explicitly excluded it from the
later `--keep-both` and section 14 gates. That scoping is correct: a retry
artifact is not divergence evidence, and it is not counted here.

### 11.2 Mac A2 append and push

The exact Claude session was resumed non-interactively and only the `A2` marker
was added. No restored file was hand-moved.

| Assertion | Before | After | Result |
| --------- | ------ | ----- | ------ |
| Same session file mutated in place | — | same path | PASS |
| `A1` occurrences preserved | 4 | 4 | PASS |
| `A2` occurrences | 0 | 4 | PASS |
| Session file size (bytes) | 10992 | 28029 | grew, not replaced |
| Stray new Claude session files | — | 0 (count 1 → 1) | PASS |
| Codex session untouched (`A1`=5, `A2`=0) | — | unchanged | PASS |

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
| Remote revision | `4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574` | `3e789e6e-b00b-492e-97a5-f0836d115dab` |
| Claude snapshot | `75c7ae2c-643b-421f-bbc4-aedb068d7f96` | `3e789e6e-b00b-492e-97a5-f0836d115dab` |
| Codex snapshot | `4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574` | unchanged |
| Session count | 2 | 2 |
| Object count | 3 | 4 (`snapshot_age_count=3`) |

Only the Claude session advanced. Ciphertext discipline holds on the new
snapshot: `.age`-only, `forbidden_shaped_objects=0`, the `A2` marker and the
`REINSTATE-PHASE1-RC7` prefix both **absent** from the downloaded bytes, and
`file` reports `data`. The local download was deleted.

Section 8 row results are unchanged by M2; the Device A counts still stand at
8 PASS / 5 PARTIAL / 0 FAIL / 10 NOT TESTED pending cross-device reconciliation
at M4.

Device A is paused for the report transfer to Windows.

## 12. Milestone block

```text
MAC-RC7-M1
release=v0.1.0-rc.7
tag_commit=66211599dd7cfb74f1436d2221b983050e8b1bc2
profile_id=2949d464-03f4-4de1-b326-2b3072bcb2a5
canonical_project_id=local/reinstate-phase1-acceptance-rc7
claude_session_id=1cf4ab6d-3e36-424d-8f30-4f41858b7f20
codex_session_id=019fa82a-2b87-71d2-947d-a8146d3049fd
remote_session_count=2
remote_revision=4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574
claude_snapshot_id=75c7ae2c-643b-421f-bbc4-aedb068d7f96
codex_snapshot_id=4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574
f1_default_refusal=PASS
ciphertext_marker_absence=PASS
tag_signature_verified=PASS
unrelated_agents_do_not_block_restore=PASS
strict_policy_still_refuses=PASS
mac_report_path=docs/testing/results/2026-07-28-macos-phase1-rc7.md
END-MAC-RC7-M1
```

```text
MAC-RC7-M2-READY
release=v0.1.0-rc.7
profile_id=2949d464-03f4-4de1-b326-2b3072bcb2a5
claude_session_id=1cf4ab6d-3e36-424d-8f30-4f41858b7f20
mac_claude_a2_marker=REINSTATE-PHASE1-RC7-MAC-CLAUDE-A2
a2_occurrences=4
a1_occurrences_preserved=4
new_remote_revision=3e789e6e-b00b-492e-97a5-f0836d115dab
new_claude_snapshot_id=3e789e6e-b00b-492e-97a5-f0836d115dab
codex_snapshot_id=4ffdb7ea-685f-4cd1-8984-0c9d7c1e6574
remote_session_count=2
remote_object_count=4
ciphertext_marker_absence=PASS
windows_w1_validated=PASS
windows_commit=82c324ce706ef9dc1a736e3c4bfa78851b75ba19
mac_report_path=docs/testing/results/2026-07-28-macos-phase1-rc7.md
END-MAC-RC7-M2-READY
```
