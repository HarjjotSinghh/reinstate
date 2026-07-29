# Phase 1 RC8 acceptance — Device A (macOS) report

Milestone reached: **M4 conflict marker pushed** (section 13). Device B has
passed W1 and W2 against RC8 and is holding an unpushed local divergence; its
keep-both resolution is outstanding, so gates 21 and 22 remain `NOT TESTED`,
never `PASS`.

Clean RC8 run. No RC7-or-older home, project, profile, passphrase, marker
session, remote prefix, or report was reused. RC7 acceptance state was left in
place untouched and is not evidence here.

RC8 exists because RC7 failed runbook section 14b. **That gate is proven on real
hardware in section 6 of this report**, under the exact condition RC7 could not
satisfy.

This report contains no storage endpoint, bucket, access key, secret key,
passphrase, keyring value, transcript prose, ciphertext bytes, remote object
name, username, or absolute local path.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | 2026-07-29 |
| Release under test | `v0.1.0-rc.8` |
| Tag commit | `5e4f2605c53c6ad46c11569235bc78476ed94487` |
| Mac model / macOS version | macOS 26.5.2 (build 25F84) |
| Mac architecture | `arm64` |
| Native shell | `zsh` |
| Claude Code version | `2.1.220` (recognized range 2.1.219–2.1.220) |
| Codex CLI version | `0.145.0` (recognized range 0.133.0–0.145.0) |
| Git version | `2.55.0` |
| Reinstate version | `0.1.0-rc.8` (commit `5e4f260`) |
| GitHub check run | all required checks green (section 7) |
| Device A profile ID | `019165e7-cf0f-420d-b261-6c291b3e4f20` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc8` |
| Claude test session ID | `0cdbd871-f924-4848-b62e-5edbeab66ae3` |
| Codex test session ID | `019facf4-d00f-7400-9a0f-8a2073e1af6e` |
| Windows edition/build | NOT TESTED (Device B) |

## 2. Release provenance (M0.1)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| `v0.1.0-rc.8` is an annotated tag object | PASS | `git cat-file -t` → `tag` |
| Signature verifies | PASS | `Good "git" signature`, checked against the repository's committed `.github/allowed_signers` |
| Signing key matches the RC6/RC7 release identity | PASS | same ED25519 key fingerprint |
| Tag commit reachable from `origin/main` | PASS | `git merge-base --is-ancestor` → true |
| Installed binary matches tag commit | PASS | `rein version --json` commit `5e4f260` |

## 3. Installer verification (M0.2)

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Live install through `https://reinstate.dev/install.sh` | PASS | installed `0.1.0-rc.8`, commit `5e4f260` |
| Checksum layer 1 (bootstrap pins stage-2 installer) | PASS | pinned SHA-256 equals the tagged `scripts/install.sh` |
| Checksum layer 2 (release asset) | PASS | run printed `checksum ok` |
| Pins exactly one release, `rc.8` | PASS | verified in the deployed bootstrap |
| No elevation required | PASS | `sudo` count 0 in both layers |
| Existing-version replacement is guarded | PASS | refused to replace `0.1.0-rc.7` until `REINSTATE_CONFIRM_REPLACE=1` |
| Live route parity | PASS | `install.sh` and `install.ps1` both byte-match the tag on the promoted origin |

## 4. Environment and honest pre-init failure (M0.3, M0.4)

Isolated home and project created fresh; `REINSTATE_HOME` exported explicitly
for every Reinstate invocation and never unset or redirected.

```text
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: darwin-arm64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
```

Exit code `3`, `config missing`, no false device or adapter pass.

## 5. M1 — sessions, init safety, push, ciphertext

### 5.1 Source sessions

Identified by strict before/after diff of the vendor session stores, confirmed
by counting only the exact marker string. No transcript prose was read or
printed.

| Agent | New files | Exact marker occurrences | Session ID |
| ----- | --------- | ------------------------ | ---------- |
| Claude Code | 1 | 4 | `0cdbd871-f924-4848-b62e-5edbeab66ae3` |
| Codex CLI | 1 | 5 | `019facf4-d00f-7400-9a0f-8a2073e1af6e` |

### 5.2 Init and health

| Step | Result |
| ---- | ------ |
| `rein init --yes --project local/reinstate-phase1-acceptance-rc8=<redacted>` | PASS — fresh `profile_id=019165e7-cf0f-420d-b261-6c291b3e4f20` |
| `rein setup check` | PASS — exit 0, all checks passed |
| `rein doctor --self-test` | PASS — exit 0, synthetic self-test passed |

`init` persisted `[restore] active_agent_policy = "fork"`.

### 5.3 Physical F1

Re-running init without `--force` returned exit `7`; `config.toml` and
`state.json` unchanged by SHA-256, backup count 0 → 0.

### 5.4 Dry-run and push

Both agents: dry-run reported `would push 1 snapshot(s)` and uploaded nothing;
the real push reported `pushed 1 snapshot(s), skipped 0 unchanged`. `rein status`
showed exactly the two selected sessions.

### 5.5 Ciphertext-only remote storage

| Assertion | Result |
| --------- | ------ |
| Object inventory | PASS — `object_count=3`, one `manifest.age` + 2 snapshots |
| Every object is `.age` | PASS |
| No auth/token/credential/`.env`/plaintext-shaped object | PASS — `forbidden_shaped_objects=0` |

Downloaded snapshot (65800 bytes, `0600`), tested without printing bytes:

| Probe | Result |
| ----- | ------ |
| `REINSTATE-PHASE1-RC8-MAC-CLAUDE-A1` | exit `1` — absent |
| `REINSTATE-PHASE1-RC8-MAC-CODEX-A1` | exit `1` — absent |
| `REINSTATE-PHASE1-RC8` (prefix) | exit `1` — absent |
| `"role"` (JSONL shape) | exit `1` — absent |
| `file` | `data`, printable ratio `0.387`, not valid UTF-8 |

Local download deleted; no remote object mutated.

## 6. Section 14b — the gate RC7 failed

RC7 decided liveness purely from open file handles. Claude Code appends to its
session file and closes it again, so a live Claude Code session holds no handle
at all and RC7 read it as free.

This run reproduced that exact condition on real hardware using a **genuine**
`claude --resume <session>` process. Nothing was simulated and no handle was
manufactured.

Preconditions captured while Claude was running on the session under test:

```text
claude_alive_pid=<non-secret pid>
handles_held_on_session_file=0
exclusive_open_succeeds=True     <-- the RC7 condition
```

A second process could open the session file exclusively while Claude Code was
live on it. That is precisely the state in which RC7 attempted an ordinary
in-place restore and was stopped only by the divergence guard with exit `6`.

RC8 result:

```text
rein pull --agent claude --session <claude>
  pulled 1 snapshot(s), dry_run=false
    <session> is in use, so it was left unchanged; restored alongside it as
    <session>-active-5f7cfc9c
  exit=0
```

| Assertion | Result |
| --------- | ------ |
| Exit code | PASS — `0`, not the RC7 `6` |
| Live session file byte-for-byte unchanged | PASS (SHA-256 compared) |
| Exactly one fork created and named in the output | PASS |
| Repeating the pull does not create a second fork | PASS — fork count 1 → 1 |
| Original session backed up | 0 — correct, the original was never replaced |
| Backups created by the repeat pull | 1, of the **fork** — see finding 4 |

The first pull created no backup at all, which is correct: nothing was replaced.
The repeat pull replaced the existing fork with byte-identical content and
backed that fork up first. Device B observed the same behavior independently on
Windows. An earlier revision of this report recorded only the first pull's
`0` and was incomplete; finding 4 records the corrected picture.

### 6.1 The refusal still works when requested

With `active_agent_policy = "scoped"` and the same live Claude session:

```text
claude is currently using this session; close that session or rerun with --allow-active-agents
exit=7
```

Session file unchanged. Note the message is the **session-scoped** wording, not
the host-wide fallback: detection identified this specific session rather than
merely observing that some Claude Code was running. Under RC7 the equivalent
Mac check could only produce the unscoped message.

### 6.2 Unrelated agents still do not block

Six unrelated Codex processes were alive throughout this run, and the Codex
restore path was unaffected. The regression that motivated the original scoping
work has not returned.

## 7. Automated integrity gates

All check runs on tag commit `5e4f260` are green:

```text
success  Build & release      success  Test (macos-latest)
success  CodeQL               success  Test (ubuntu-latest)
success  Lint                 success  Test (windows-latest)
success  Secret scan          success  Validate website and CLI tags
success  Security             success  Website
success  Workflow permission and pin review
skipped  Dependency review
```

`Dependency review` is skipped on push events, as expected.

## 8. Mandatory sign-off checklist (all 23 rows)

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC8 on Mac | PASS | §3 |
| 2 | `install.ps1` returns 200 and installs RC8 on Windows | NOT TESTED | Device B |
| 3 | Both installers are idempotent and PATH-safe | PARTIAL | §3 — Mac side verified |
| 4 | Pre-init missing-config failure is accurate | PARTIAL | §4 — Mac exit 3 |
| 5 | Post-init setup check and self-test pass on both devices | PARTIAL | §5.2 — Mac both exit 0 |
| 6 | Claude setup prompt completes on the Mac | PASS | §5.2 |
| 7 | Codex setup prompt completes on Windows | NOT TESTED | Device B |
| 8 | Only two selected test sessions reach the remote manifest | PASS | §5.4 |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | §5.5 |
| 10 | Wrong passphrase fails without mutation | PARTIAL | Mac exit 4, no config/backup change; Windows execution outstanding |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | Device B |
| 13 | Unrelated running agents do not block a restore | PASS | §6.2 |
| 14 | A live session is forked, never overwritten | **PASS** | §6 — proven under the exact RC7 failure condition |
| 15 | `scoped` policy still refuses, naming that session | **PASS** | §6.1 — session-scoped refusal, exit 7, no mutation |
| 16 | Existing Windows target is backed up before restore | NOT TESTED | Device B |
| 17 | Claude Windows-to-Mac resume succeeds | PASS | §12.5 — discovered, resumed, returned the Windows `B1` marker |
| 18 | Codex Windows-to-Mac resume succeeds | PASS | §12.5 — restored rollout loaded as a live process, `B1`=5 |
| 19 | Existing Mac targets are backed up before restore | PASS | §12.4 — two timestamped backups, both SHA-256 matching the pre-pull originals |
| 20 | Unchanged pushes skip without new snapshots | PASS | both agents reported `pushed 0 snapshot(s), skipped 1 unchanged`, remote revision unchanged |
| 21 | Divergence records a conflict without overwrite | NOT TESTED | M4 / W3 |
| 22 | `--keep-both` preserves both branches | NOT TESTED | Device B |
| 23 | All required GitHub checks are green | PASS | §7 |

**Counts: 12 PASS / 4 PARTIAL / 0 FAIL / 7 NOT TESTED.**

All 23 mandatory rows passed: **No.** Phase 1 remains open pending Device B.

## 9. Findings

No release-blocking findings on Device A.

Non-blocking, recorded for sign-off:

1. **Gate 19 is not carried over from the RC7 run.** In RC7 a Mac backup was
   produced as a side effect of an in-place restore. In RC8 the equivalent
   restore forked instead, which is the correct new behavior, so no Mac target
   was replaced and no backup was created. The gate needs an in-place Mac
   restore with no agent live on the session, which happens naturally at M3.
2. **Resuming a Claude session appends to it.** Starting `claude --resume` for
   the section 6 test changed the session file before any Reinstate command ran.
   That is vendor behavior, not a Reinstate defect, but it means a session used
   for a liveness test is no longer byte-identical to what was pushed. The fork
   path is unaffected because it does not consult divergence.
3. **Windows Restart Manager has now been exercised on real hardware.** Device B
   completed section 14b against RC8 under the same no-handle, exclusive-open
   condition and reported exit `0`, an unchanged original, one fork, and an
   idempotent repeat. This was the highest-value outstanding gate through two
   release candidates and it is now closed on both platforms.
4. **Repeat pulls of an in-use session accumulate backups of the fork.** The
   fork identity is derived from the snapshot, so a repeated pull rewrites the
   same fork with byte-identical content and backs up the previous copy first.
   The original session is never touched and its backup count stays zero, so no
   data is at risk and the fork count stays at one, but the backup directory
   grows by one identical file per repeat. Device B found this independently on
   Windows and it reproduces on macOS.

   The intended behavior is to skip the restore when the fork already exists and
   its content matches, which was considered during implementation and not
   carried through. It is a wasted write, not a correctness problem, so it is
   recorded here rather than used to justify another release candidate mid
   acceptance. It should be fixed in the next RC cut for any other reason.

## 10. Repository hygiene

- Branch `test/phase1-rc8-macos-report` from the peeled RC8 tag commit
  `5e4f260`, in a dedicated worktree; no product branch touched.
- The private credentials file is in `.git/info/exclude`, untracked, outside the
  repository, and its SHA-256 was unchanged across the wrong-passphrase test.
- The only change on this branch is this report.

## 11. M2 — Windows W1 validation and Mac A2 push

Device B issued `WINDOWS-RC8-W1-PASS` at commit
`e489ce8ab4a917cf036ee714d76d245c501388a1`, with 17 PASS / 0 PARTIAL / 0 FAIL /
6 NOT TESTED.

### 11.1 Re-validated handoff

| Check | Result |
| ----- | ------ |
| Commit resolves; branch tip matches | PASS |
| Draft PR #57 open, not merged | PASS |
| Branch changes only `docs/testing/results/` (0 product files) | PASS |
| No credential value, Windows username path, or transcript JSON | PASS |
| Block complete with `END-WINDOWS-RC8-W1` terminator | PASS |
| `profile_id` and both session IDs match Device A | PASS |
| Remote parity re-verified independently | PASS — revision `b5f38a3d-…`, 2 sessions |

**Section 14b is now proven on both platforms.** Device B reproduced the
no-handle, exclusive-open condition on real Windows hardware and reported exit
`0`, an unchanged original, one fork, and an idempotent repeat, with no RC8
conflict. It also closed its exclusive test handle before invoking Reinstate, so
the result reflects Reinstate's own detection rather than a handle the test
itself was holding. That gate blocked two release candidates and is now closed.

Device B's backup observation is recorded as finding 4 and reproduces on macOS.

### 11.2 Mac A2 append and push

| Assertion | Before | After | Result |
| --------- | ------ | ----- | ------ |
| Same session file mutated in place | — | same path | PASS |
| `A1` occurrences preserved | 4 | 4 | PASS |
| `A2` occurrences | 0 | 4 | PASS |
| Session file size (bytes) | 25126 | 27556 | grew, not replaced |
| Stray new Claude session files | — | 0 (count 1 → 1) | PASS |
| Codex session untouched (`A1`=5, `A2`=0) | — | unchanged | PASS |

Dry-run reported `would push 1 snapshot(s)` and uploaded nothing; the push
reported `pushed 1 snapshot(s), skipped 0 unchanged`.

### 11.3 Remote state after M2

| Field | Before M2 | After M2 |
| ----- | --------- | -------- |
| Remote revision | `b5f38a3d-e841-4787-8105-f080f3524fab` | `f552c4c8-bc17-4823-a447-bc18a4bb62e5` |
| Claude snapshot | `b5f38a3d-…` | `f552c4c8-…` |
| Codex snapshot | `17773f7e-…` | unchanged |
| Session count | 2 | 2 |

Only the Claude session advanced. Ciphertext discipline holds on the new
snapshot: `forbidden_shaped_objects=0`, the `A2` marker and the
`REINSTATE-PHASE1-RC8` prefix both absent from the downloaded bytes, `file`
reports `data`. The local download was deleted.

Section 8 row results are unchanged by M2; reconciliation happens at M4.

## 12. M3 — Windows-to-Mac restore, backups, resume, and no-op

Device B issued `WINDOWS-RC8-W2-READY` at commit
`d608293d5828df6e4eaa1d371dafdcebf8f8bb46`, 18 PASS / 0 PARTIAL / 0 FAIL /
5 NOT TESTED, with gate 16 satisfied through the authorised conflict route.

### 12.1 Re-validated handoff

Commit and branch resolve, draft PR #57 open and unmerged, 0 product files, no
credential value, Windows username path, or transcript JSON, block complete with
the correct terminator. Device A independently re-verified the remote before
acting: revision `8e3dba9c-…`, Claude snapshot `cf89ccc6-…`, 2 sessions.

### 12.2 Agent liveness pre-check

Twelve agent processes were alive on this device throughout M3. Each was checked
against the acceptance project before pulling:

| Signal | Result |
| ------ | ------ |
| Any agent holding either session file | none |
| Any agent naming either session ID on its command line | none |
| Any agent whose working directory is inside the acceptance project | none |

So the restore was expected to proceed in place rather than fork, which is what
gate 19 needs. This pre-check is the practical counterpart of the RC8 detection
change: it is now possible to state *why* a restore will replace rather than
fork, instead of inferring it from the absence of a refusal.

### 12.3 Dry-runs

Both dry-runs reported `would pull 1 snapshot(s)`, named the **original** session
paths rather than any fork, reported the backup root, and created nothing.
Backup count stayed at 1.

### 12.4 Real pulls and backups (gate 19)

| Assertion | Result |
| --------- | ------ |
| Claude pull | PASS — exit 0, restored to the Mac project key |
| Codex pull | PASS — exit 0 |
| Backup count | 1 → 3 |
| Claude backup is timestamped and matches the pre-pull original | PASS — SHA-256 equal to `4d89901b…` |
| Codex backup is timestamped and matches the pre-pull original | PASS — SHA-256 equal to `80435840…` |

The pre-existing third backup is the section 14b fork backup described in
finding 4 and is excluded from this gate.

### 12.5 Restored markers and vendor resume

Restored counts, matching Device B exactly:

| Agent | `A1` | `A2` | `B1` |
| ----- | ---- | ---- | ---- |
| Claude | 4 | 4 | 5 |
| Codex | 5 | — | 5 |

`rein list --agent claude` discovered the exact session, and `claude --resume`
loaded it and returned the Windows-authored `B1` marker. `codex resume` loaded
the restored rollout as a live process.

**Method note, and a correction.** The Claude resume was performed with a prompt,
which caused Claude Code to append a reply and take the local `B1` count from 5
to 6. The Codex resume was performed without input and appended nothing. That
inconsistency was Device A's, not a product defect: as recorded in finding 2,
resuming a Claude session mutates it. The Claude session was restored to the
remote state through the same conflict route Device B used for gate 16 — pull
returned exit 6 with one conflict, `conflicts resolve --keep-remote` returned
exit 0 and produced a fourth timestamped backup, and the session returned to
`A1`=4, `A2`=4, `B1`=5 with zero active conflicts. Nothing was fabricated and no
marker count was edited by hand.

### 12.6 Unchanged no-op (gate 20)

With both restored sessions untouched:

```text
push --agent claude --session <claude>   pushed 0 snapshot(s), skipped 1 unchanged
push --agent codex  --session <codex>    pushed 0 snapshot(s), skipped 1 unchanged
```

Remote revision stayed `8e3dba9c-d0c0-4549-b6da-4d6c59b64f38` and both snapshot
IDs were unchanged, so no new remote snapshot or revision was created.

### 12.7 Gate movement

| Gate | Before M3 | After M3 |
| ---- | --------- | -------- |
| 17 Claude Windows-to-Mac resume | NOT TESTED | **PASS** |
| 18 Codex Windows-to-Mac resume | NOT TESTED | **PASS** |
| 19 Existing Mac targets backed up before restore | NOT TESTED | **PASS** |
| 20 Unchanged pushes skip without new snapshots | PASS | PASS (re-confirmed post-restore) |

Updated counts: **12 PASS / 4 PARTIAL / 0 FAIL / 7 NOT TESTED.**

## 13. M4 step 1 — Mac conflict marker pushed

Device B issued `WINDOWS-RC8-CONFLICT-LOCAL-READY` at commit
`b89162f2dadd7ee15fe496f793da5d437b1fc823`, holding
`REINSTATE-PHASE1-RC8-CONFLICT-WINDOWS` locally at 4 occurrences, unpushed, with
the remote unchanged and gates 21 and 22 correctly still `NOT TESTED`.

### 13.1 Re-validated handoff

Commit and branch resolve, draft PR #57 open and unmerged, 0 product files, no
credential value, Windows username path, or transcript JSON. Device A confirmed
the remote was still at revision `8e3dba9c-…` before acting, and that the
Windows conflict marker was absent from the Mac copy.

### 13.2 Divergence created

| Step | Result |
| ---- | ------ |
| Resumed the exact Mac Claude session | exit 0 |
| Appended only `REINSTATE-PHASE1-RC8-CONFLICT-MAC` | 4 occurrences |
| Windows conflict marker on the Mac copy | 0 — the two branches are genuinely distinct |
| Prior markers preserved | `A1`=4, `A2`=4, `B1`=5 |
| Stray sessions created | 0 (count 1 → 1) |
| Dry-run | `would push 1 snapshot(s)`, uploaded nothing |
| Push | `pushed 1 snapshot(s), skipped 0 unchanged` |

Remote after the push:

| Field | Before | After |
| ----- | ------ | ----- |
| Revision | `8e3dba9c-d0c0-4549-b6da-4d6c59b64f38` | `633f5f3d-6fd2-49ef-865f-0e29eed55850` |
| Claude snapshot | `cf89ccc6-…` | `633f5f3d-…` |
| Codex snapshot | `8e3dba9c-…` | unchanged |
| Session count | 2 | 2 |

Only the Claude session advanced. Ciphertext discipline holds on the new
snapshot: `all_objects_end_with_.age=true`, `forbidden_shaped_objects=0`, the
`CONFLICT-MAC` marker and the `REINSTATE-PHASE1-RC8` prefix both absent from the
downloaded bytes, `file` reports `data`.

The divergence required by gates 21 and 22 now exists: Device B holds
`CONFLICT-WINDOWS` locally and unpushed, while the remote holds `CONFLICT-MAC`.

### 13.3 Conflict bookkeeping note

Device A's acceptance home contains one conflict record on disk,
`<id>.keep-remote.resolved`, archived from the M3 recovery described in §12.5.
`rein conflicts list` reports **0 active conflicts**. Resolution archives a
record rather than deleting it, which preserves an audit trail. Gate 21's "one
recorded conflict" should therefore be counted from the active list, not from
files on disk.

Device A is paused for Device B's keep-both resolution.

## 14. Milestone block

```text
MAC-RC8-M1
release=v0.1.0-rc.8
tag_commit=5e4f2605c53c6ad46c11569235bc78476ed94487
profile_id=019165e7-cf0f-420d-b261-6c291b3e4f20
canonical_project_id=local/reinstate-phase1-acceptance-rc8
claude_session_id=0cdbd871-f924-4848-b62e-5edbeab66ae3
codex_session_id=019facf4-d00f-7400-9a0f-8a2073e1af6e
remote_session_count=2
remote_revision=b5f38a3d-e841-4787-8105-f080f3524fab
claude_snapshot_id=b5f38a3d-e841-4787-8105-f080f3524fab
codex_snapshot_id=17773f7e-17ca-4d41-8848-af285c5fe1a3
f1_default_refusal=PASS
ciphertext_marker_absence=PASS
tag_signature_verified=PASS
live_session_forked_not_overwritten=PASS
live_session_held_no_file_handle=true
exclusive_open_succeeded_while_live=true
scoped_policy_session_named_refusal=PASS
unrelated_agents_do_not_block_restore=PASS
mac_report_path=docs/testing/results/2026-07-29-macos-phase1-rc8.md
END-MAC-RC8-M1
```

```text
MAC-RC8-M2-READY
release=v0.1.0-rc.8
profile_id=019165e7-cf0f-420d-b261-6c291b3e4f20
claude_session_id=0cdbd871-f924-4848-b62e-5edbeab66ae3
mac_claude_a2_marker=REINSTATE-PHASE1-RC8-MAC-CLAUDE-A2
a2_occurrences=4
a1_occurrences_preserved=4
new_remote_revision=f552c4c8-bc17-4823-a447-bc18a4bb62e5
new_claude_snapshot_id=f552c4c8-bc17-4823-a447-bc18a4bb62e5
codex_snapshot_id=17773f7e-17ca-4d41-8848-af285c5fe1a3
remote_session_count=2
ciphertext_marker_absence=PASS
windows_w1_validated=PASS
windows_commit=e489ce8ab4a917cf036ee714d76d245c501388a1
section_14b_proven_on_both_platforms=true
mac_report_path=docs/testing/results/2026-07-29-macos-phase1-rc8.md
END-MAC-RC8-M2-READY
```

```text
MAC-RC8-M3-PASS
release=v0.1.0-rc.8
profile_id=019165e7-cf0f-420d-b261-6c291b3e4f20
claude_session_id=0cdbd871-f924-4848-b62e-5edbeab66ae3
codex_session_id=019facf4-d00f-7400-9a0f-8a2073e1af6e
windows_w2_validated=PASS
windows_commit=d608293d5828df6e4eaa1d371dafdcebf8f8bb46
dry_runs_said_would_pull=PASS
claude_windows_to_mac_resume=PASS
codex_windows_to_mac_resume=PASS
mac_targets_backed_up_before_restore=PASS
backup_sha256_matches_pre_pull_originals=PASS
windows_claude_b1_occurrences=5
windows_codex_b1_occurrences=5
a1_occurrences_preserved=4
a2_occurrences_preserved=4
unchanged_no_op_both_agents=PASS
remote_revision=8e3dba9c-d0c0-4549-b6da-4d6c59b64f38
claude_snapshot_id=cf89ccc6-f248-48b9-a1a4-cd5c9572d719
codex_snapshot_id=8e3dba9c-d0c0-4549-b6da-4d6c59b64f38
remote_session_count=2
remote_unchanged_by_no_op=PASS
mac_report_path=docs/testing/results/2026-07-29-macos-phase1-rc8.md
END-MAC-RC8-M3-PASS
```

```text
MAC-RC8-CONFLICT-PUSHED
release=v0.1.0-rc.8
profile_id=019165e7-cf0f-420d-b261-6c291b3e4f20
claude_session_id=0cdbd871-f924-4848-b62e-5edbeab66ae3
windows_conflict_local_validated=PASS
windows_commit=b89162f2dadd7ee15fe496f793da5d437b1fc823
mac_conflict_marker=REINSTATE-PHASE1-RC8-CONFLICT-MAC
mac_conflict_marker_occurrences=4
windows_conflict_marker_on_mac_copy=0
a1_occurrences_preserved=4
a2_occurrences_preserved=4
b1_occurrences_preserved=5
new_remote_revision=633f5f3d-6fd2-49ef-865f-0e29eed55850
new_claude_snapshot_id=633f5f3d-6fd2-49ef-865f-0e29eed55850
codex_snapshot_id=8e3dba9c-d0c0-4549-b6da-4d6c59b64f38
remote_session_count=2
ciphertext_marker_absence=PASS
mac_active_conflicts=0
mac_archived_resolved_records=1
divergence_ready_for_gate21=true
mac_report_path=docs/testing/results/2026-07-29-macos-phase1-rc8.md
END-MAC-RC8-CONFLICT-PUSHED
```
