# Phase 1 RC8 acceptance — Device B (native Windows) report

Milestone reached: **W1 PASS**. This run also executes tagged runbook section
14b on real Windows hardware under the exact no-handle/exclusive-open condition
that RC7 failed.

This is clean RC8 evidence. No RC7-or-older home, project, profile, passphrase,
marker session, remote prefix, conflict, or report was reused. The historical
RC7 conflict was not opened, resolved, copied, or migrated into RC8 state.

This report contains no storage endpoint, bucket coordinate, access key, secret
key, passphrase, keyring value, auth file, transcript prose, ciphertext bytes,
remote object name, username, or absolute local path.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date (UTC) | 2026-07-29 |
| Release under test | `v0.1.0-rc.8` |
| Tag commit | `5e4f2605c53c6ad46c11569235bc78476ed94487` |
| Windows edition/build | Windows 11 Pro, build `26200` |
| Architecture | native Windows `amd64`; 64-bit OS |
| PowerShell | Windows PowerShell `5.1.26100.8328` |
| Claude Code version | `2.1.220` (`SUPPORTED`) |
| Codex CLI version | `0.145.0` (`SUPPORTED`) |
| Git version | `2.52.0.windows.1` |
| Reinstate version | `0.1.0-rc.8` (commit `5e4f260`) |
| Device A profile ID | `019165e7-cf0f-420d-b261-6c291b3e4f20` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc8` |
| Claude test session ID | `0cdbd871-f924-4848-b62e-5edbeab66ae3` |
| Codex test session ID | `019facf4-d00f-7400-9a0f-8a2073e1af6e` |
| Remote revision | `b5f38a3d-e841-4787-8105-f080f3524fab` |

## 2. Required Mac handoff validation

Validated the complete report at
`docs/testing/results/2026-07-29-macos-phase1-rc8.md` from commit
`88731789400c82de0cf5029873115e7a2aad0439` on
`test/phase1-rc8-macos-report`.

| Check | Result |
| ----- | ------ |
| `MAC-RC8-M1` release/tag/profile/canonical project match RC8 | PASS |
| Both supplied session IDs are fresh RC8 IDs | PASS |
| Remote session count is exactly 2 | PASS |
| Mac physical F1 default refusal | PASS |
| Mac ciphertext-marker absence | PASS |
| Tag signature verified on Mac | PASS |
| Mac section 14b live/no-handle/exclusive-open condition | PASS |
| No credential, passphrase, endpoint, access-key, or secret-key value detected | PASS |

The private contract's bucket string collides with ordinary product prose, so a
raw substring test is not meaningful for that one field. The Mac report does
not present a bucket assignment, coordinate, URL, or storage object name.

## 3. W0 — release, installer, environment, pre-init

### 3.1 Provenance

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Annotated tag object | PASS | `git cat-file -t` → `tag` |
| Signature | PASS | good SSH signature with the tag-matching committed allowed-signers file |
| Peeled commit | PASS | `5e4f2605c53c6ad46c11569235bc78476ed94487` |
| Reachable from `origin/main` | PASS | `git merge-base --is-ancestor` → true |

No global trust configuration was changed.

### 3.2 Live Windows installer

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Live route | PASS | HEAD `200`; live bootstrap byte-matched the tagged bootstrap |
| Exact release pin | PASS | exactly `v0.1.0-rc.8`; no older RC pin or latest resolver |
| Canonical stage-2 URL | PASS | exact tagged `scripts/install.ps1` |
| Bootstrap checksum layer | PASS | pinned hash matched both tag and downloaded canonical installer |
| Release-asset checksum layer | PASS | installer printed `checksum ok` |
| Release asset base | PASS | exact RC8 GitHub release path |
| Elevation | PASS | no elevation construct or elevated prompt |
| Install result | PASS | both aliases report RC8 commit `5e4f260` |
| Idempotency | PASS | rerun reported RC8 already installed |
| User PATH normalization | PASS | exactly 1 normalized install entry |

The machine began with RC7 installed. An unconfirmed non-interactive replacement
attempt exited `1`; the documented explicit replacement confirmation was then
applied, after which RC8 installed successfully. RC7 was never used as RC8
evidence.

### 3.3 Fresh isolation and honest pre-init result

Both exact RC8 isolation paths were absent before creation. The private
`R2.txt` was validated without displaying values: regular file, not a reparse
point, exact five-key schema, endpoint and bucket separate, and no allow ACL
for another principal.

`REINSTATE_HOME` was set to the exact RC8 home for every acceptance command.
The disposable project was created with only `.git` and `README.md`.

```text
summary: 1 check(s) failed
- [fail] config: config missing
- [ok] device: windows-amd64
- [ok] agent.claude: SUPPORTED (2.1.220)
- [ok] agent.codex: SUPPORTED (0.145.0)
- [ok] keyring: OS keyring provider reachable
exit=3
```

## 4. W1 — F3, init, F1, wrong passphrase, F2

Storage values were parsed only inside a local launcher outside the repository
and both vendor session trees. Each Reinstate child received storage values
only in its child environment. Passphrases were written through an inheritable
anonymous pipe and exposed only through `REINSTATE_PASSPHRASE_FD`.

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| F3 bad coordinates | PASS | exit `4`; `remote profile manifest not found`; config/state absent; backups `0→0` |
| Correct additional-device init | PASS | exit `0`; exact Mac profile and canonical mapping persisted |
| Post-init setup check | PASS | exit `0`; all checks passed; both adapters `SUPPORTED` |
| `doctor --self-test` | PASS | exit `0`; synthetic self-test passed |
| F1 default refusal | PASS | exit `7`; config/state SHA-256 unchanged; backups `0→0` |
| Wrong passphrase | PASS | exit `4`; decryption refusal; config/state/backups unchanged; target counts stayed `0/0` |
| Correct status | PASS | exactly 2 selected sessions; revision `b5f38a3d-e841-4787-8105-f080f3524fab` |
| F2 missing manifest | PASS | copied RC8 home; exactly one prefix field changed; exit `4`; real home hashes unchanged |
| Real remote after F2 | PASS | still exactly 2 sessions at the same revision and snapshot IDs |

The F2 copied probe home remains isolated and is not evidence for any later
positive gate.

## 5. Exact-ID restore, discovery, and same-vendor resume

No Claude or Codex vendor process named either exact target ID before the
restores.

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Codex dry-run | PASS | exit `0`, `would pull 1`; target `0→0`; backups `0→0` |
| Claude dry-run | PASS | exit `0`, `would pull 1`; target `0→0`; backups `0→0` |
| Codex real pull | PASS | exit `0`; exact target `0→1`; backups remained `0` |
| Claude real pull | PASS | exit `0`; mapped Windows-project target `0→1`; backups remained `0` |
| Restored Codex A1 marker | PASS | exact occurrence count `5`, matching Mac source evidence |
| Restored Claude A1 marker | PASS | exact occurrence count `4`, matching Mac source evidence |
| Codex exact-ID resume | PASS | exact challenge response; literal `codex resume <ID>` TUI process count `1`, then `0` after close |
| Claude exact-ID resume | PASS | one exact assistant challenge response from `claude --resume <ID>` |
| Claude mapped-project discovery | PASS | exact ID found under the Windows project key, never the Mac source slug |

Only exact marker counts and exact challenge-response equality were inspected.
No surrounding transcript prose was read into evidence or reported.

## 6. Tagged runbook section 14b on real Windows

This is the release-critical condition RC7 failed. A genuine
`claude --resume <ID>` TUI was alive on the restored session. Claude held no
open handle on the file, and a second process could still open it exclusively.
The exclusive test handle was closed before Reinstate ran.

```text
claude_alive_on_exact_session=True
restart_manager_supported=True
handles_held_on_session_file=0
exclusive_open_succeeded_while_live=True
active_policy_default_fork=True
```

RC8 result:

```text
rein pull --agent claude --session <ID>
pulled 1 snapshot(s), dry_run=false
  <ID> is in use, so it was left unchanged; restored alongside it as
  <ID>-active-b5f38a3d
exit=0
```

| Assertion | Result |
| --------- | ------ |
| Exit code is `0`, not RC7's `6` | PASS |
| Pull treats the session as in use despite zero handles | PASS |
| Original live file byte-for-byte unchanged | PASS |
| Exactly one named active fork created | PASS |
| Repeat pull creates no second fork | PASS (`1→1`) |
| RC8 conflict list before/after | PASS (`null→null`) |
| Original-session backups created | PASS (`0`) |
| Repeat-pull side effect | OBSERVED — one backup of the generated active fork per diagnostic repeat |
| Exact RC8 fork removed afterward | PASS (`1→0`) |
| Exact live Claude processes after close | PASS (`0`) |

An exit `6` or recorded RC8 conflict would have failed this gate. Neither
occurred. The historical RC7 conflict was outside the RC8 home and remained
untouched.

Two retained backup files exist from the two diagnostic repeat runs used to
validate the harness. Both filenames match the exact RC8 generated active fork;
none matches the original session ID. This does not violate section 14b's
mandatory one-fork/original-unchanged/no-conflict result, but it is a real
cross-device discrepancy from the Mac report's observed backup count of zero.

## 7. Mandatory sign-off checklist (all 23 rows)

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC8 on Mac | PASS | Mac report §3 |
| 2 | `install.ps1` returns 200 and installs RC8 on Windows | PASS | §3.2 |
| 3 | Both installers are idempotent and PATH-safe | PASS | Mac report §3; §3.2 |
| 4 | Pre-init missing-config failure is accurate | PASS | Mac report §4; §3.3 |
| 5 | Post-init setup check and self-test pass on both devices | PASS | Mac report §5.2; §4 |
| 6 | Claude setup prompt completes on the Mac | PASS | Mac report §5.2 |
| 7 | Codex setup prompt completes on Windows | PASS | §§3–5 |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Mac report §5.4; §4 |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Mac report §5.5 |
| 10 | Wrong passphrase fails without mutation | PASS | Mac report §8 row 10; §4 |
| 11 | Claude Mac-to-Windows resume succeeds | PASS | §5 |
| 12 | Codex Mac-to-Windows resume succeeds | PASS | §5 |
| 13 | Unrelated running agents do not block a restore | PASS | Mac report §6.2; Windows exact pulls succeeded |
| 14 | A live session is forked, never overwritten | PASS | Mac report §6; Windows §6 |
| 15 | `scoped` policy still refuses, naming that session | PASS | Mac report §6.1 |
| 16 | Existing Windows target is backed up before restore | NOT TESTED | W2 / section 14d |
| 17 | Claude Windows-to-Mac resume succeeds | NOT TESTED | later M3 |
| 18 | Codex Windows-to-Mac resume succeeds | NOT TESTED | later M3 |
| 19 | Existing Mac targets are backed up before restore | NOT TESTED | later M3 |
| 20 | Unchanged pushes skip without new snapshots | PASS | Mac report §8 row 20 |
| 21 | Divergence records a conflict without overwrite | NOT TESTED | later W3 |
| 22 | `--keep-both` preserves both branches | NOT TESTED | later W3 |
| 23 | All required GitHub checks are green | PASS | Mac report §7 |

**Counts: 17 PASS / 0 PARTIAL / 0 FAIL / 6 NOT TESTED.**

All 23 mandatory rows passed: **No.** Phase 1 remains open for W2, M3, and W3.

## 8. Findings and retry trace

No mandatory W1 or section 14b failure was observed.

Sign-off-relevant, currently classified non-blocking: a repeat live-session
pull reused the same fork name but backed up that existing fork before writing
it again. The final fork cardinality remained one and the original live file
was never touched, so the tagged mandatory gate passes. The side effect is not
fully idempotent and differs from the Mac report's `backups=0` observation.

Non-blocking operator-harness notes:

1. The W0 helper captured process stdout but not PowerShell information-stream
   messages, so its derived checksum booleans were false while the command's
   raw output visibly reported both checksum successes. Exact hashes and final
   versions independently matched.
2. Initial section 14b launcher attempts stopped before any pull because a
   nested TUI did not stay live, then because one CIM process was unwrapped as
   a scalar with no usable `.Count`. A later capture attempt ran the correct
   product behavior but returned harness exit `1` because console output
   bypassed its pipeline. The launcher was fixed and the final full section 14b
   run exited `0` with every assertion above true.
3. Resuming either vendor session appends vendor metadata/challenge records.
   Section 14b hashes were therefore taken only after the genuine live resume
   settled and immediately around both Reinstate pulls.

No failed harness attempt reused RC7 state, executed an RC8 pull before a
genuine exact live process existed, created a conflict, or left an active fork.

## 9. Repository hygiene

- Branch `test/phase1-rc8-windows-report` starts at peeled RC8 tag commit
  `5e4f2605c53c6ad46c11569235bc78476ed94487`.
- `R2.txt` is covered by `.git/info/exclude`, remains outside the repository,
  and is absent from the staged diff.
- The only repository change is this report.
- No product code, private contract data, transcript, or secret was committed.

## 10. Milestone block

```text
WINDOWS-RC8-W1-PASS
release=v0.1.0-rc.8
tag_commit=5e4f2605c53c6ad46c11569235bc78476ed94487
profile_id=019165e7-cf0f-420d-b261-6c291b3e4f20
canonical_project_id=local/reinstate-phase1-acceptance-rc8
claude_session_id=0cdbd871-f924-4848-b62e-5edbeab66ae3
codex_session_id=019facf4-d00f-7400-9a0f-8a2073e1af6e
f3_bad_coordinates_refused=PASS
f1_default_refusal=PASS
f2_missing_manifest_refused=PASS
wrong_passphrase_refused=PASS
remote_session_count=2
remote_revision=b5f38a3d-e841-4787-8105-f080f3524fab
claude_discovery_and_resume=PASS
codex_resume=PASS
section14b_live_session_forked_not_overwritten=PASS
live_session_held_no_file_handle=true
exclusive_open_succeeded_while_live=true
section14b_repeat_single_fork=PASS
section14b_repeat_backup_side_effect=OBSERVED
rc8_conflict_recorded=false
windows_report_path=docs/testing/results/2026-07-29-windows-phase1-rc8.md
END-WINDOWS-RC8-W1
```
