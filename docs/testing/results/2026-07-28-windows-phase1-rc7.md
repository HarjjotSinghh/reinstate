# Phase 1 RC7 acceptance — Device B (Windows) report

Milestone verdict: **`WINDOWS-RC7-W1-FAIL`**. Device B acceptance stopped at
the private `R2.txt` contract check, before W0 installer execution or any W1
init, status, pull, restore, resume, backup, or remote-storage operation.

This is an acceptance-input blocker, not evidence of a Reinstate product
failure. The supplied file was not modified, deleted, copied, committed, or
displayed. The validator did not output any field name or value, and this
report does not intentionally reproduce them.

## 1. Test record

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | `2026-07-28T11:51:35Z` |
| Release under test | `v0.1.0-rc.7` |
| Tag commit | `66211599dd7cfb74f1436d2221b983050e8b1bc2` |
| Native Windows environment | NOT TESTED — stopped before W0 environment capture |
| Claude Code version | NOT TESTED |
| Codex CLI version | NOT TESTED |
| Git version | NOT TESTED |
| Reinstate version | NOT TESTED |
| Device A profile ID | `2949d464-03f4-4de1-b326-2b3072bcb2a5` |
| Canonical project ID | `local/reinstate-phase1-acceptance-rc7` |
| Claude test session ID | `1cf4ab6d-3e36-424d-8f30-4f41858b7f20` |
| Codex test session ID | `019fa82a-2b87-71d2-947d-a8146d3049fd` |

## 2. Isolation and Mac handoff validation

| Check | Result | Evidence |
| ----- | ------ | -------- |
| RC7 report branch starts at peeled tag commit | PASS | branch HEAD equals `66211599dd7cfb74f1436d2221b983050e8b1bc2` |
| RC7 Windows Reinstate home absent before run | PASS | boolean `false` |
| RC7 Windows disposable project absent before run | PASS | boolean `false` |
| RC6 acceptance state reused or modified | PASS | no |
| Mac report read from its pushed RC7 report branch | PASS | commit `d2edc860e7f41c08eb84da98a5992173d86c9e55` handoff validated |
| Required `MAC-RC7-M1` lines missing | PASS | `0` |
| Secret-coordinate assignments in Mac report | PASS | `0` |
| Mac report remote session count | PASS | exactly `2` |
| Mac physical F1 and ciphertext checks | PASS | both recorded `PASS` |

The local tag is an annotated tag object at the expected peeled commit and that
commit is reachable from `origin/main`. Independent Windows signature
verification was **NOT TESTED** because the mandatory private-file contract
failed first.

## 3. Blocking private-file contract failure

The RC7 automation overlay requires an exact regular UTF-8 file containing one
non-empty entry for each of the five declared contract keys, no unknown keys,
an HTTPS endpoint, and a separate bucket. It also requires stopping on any
conflict and forbids altering the file contents.

The validator inspected only structure in memory and emitted booleans and
counts. It never emitted a key name or value.

### Exact failing operation

```text
operation=inline PowerShell RC7 R2 schema validator
input=$HOME\R2.txt
exit=1
sanitized_output=R2 validation failed without disclosing values
```

### Sanitized diagnostics

```text
regular_file=true
reparse_point=false
utf8_valid=true
utf8_bom=false
line_count=5
nonempty_line_count=5
known_required_key_count=0
unknown_key_count=5
duplicate_key_count=0
missing_equals_count=0
empty_value_count=0
endpoint_https=false
endpoint_bucket_separate=false
```

Because all five parsed key names were outside the exact RC7 contract,
endpoint/bucket validation could not proceed. ACL hardening was deliberately
not attempted after schema failure. No installer, binary, isolated home,
disposable project, profile, config, state, backup, agent session, or remote
object was created or mutated.

## 4. Mandatory sign-off checklist (all 23 rows)

The counts remain those of the validated Mac M1 report because Device B
produced no behavioral evidence. A Device A half stays `PARTIAL`; a
Windows-dependent gate stays `NOT TESTED`.

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| 1 | `install.sh` returns 200 and installs RC7 on Mac | PASS | Mac report §3 |
| 2 | `install.ps1` returns 200 and installs RC7 on Windows | NOT TESTED | stopped before W0 |
| 3 | Both installers are idempotent and PATH-safe | PARTIAL | Mac half only |
| 4 | Pre-init missing-config failure is accurate | PARTIAL | Mac half only |
| 5 | Post-init setup check and self-test pass on both devices | PARTIAL | Mac half only |
| 6 | Claude setup prompt completes on the Mac | PASS | Mac report §5.2 |
| 7 | Codex setup prompt completes on Windows | NOT TESTED | private-file contract blocked start |
| 8 | Only two selected test sessions reach the remote manifest | PASS | Mac report §5.4 |
| 9 | Remote manifest/snapshots are ciphertext-only | PASS | Mac report §5.5 |
| 10 | Wrong passphrase fails without mutation | PARTIAL | Mac evidence only; Windows execution outstanding |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | Device B W1 |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | Device B W1 |
| 13 | Unrelated running agents do not block a restore | PASS | Mac report §6.1 |
| 14 | A live session is forked, never overwritten | NOT TESTED | Device B §14b |
| 15 | `scoped` policy still refuses, naming that session | PARTIAL | Mac strict-policy evidence only; Windows scoped test outstanding |
| 16 | Existing Windows target is backed up before restore | NOT TESTED | Device B W2 |
| 17 | Claude Windows-to-Mac resume succeeds | NOT TESTED | later handoff |
| 18 | Codex Windows-to-Mac resume succeeds | NOT TESTED | later handoff |
| 19 | Existing Mac targets are backed up before restore | PASS | Mac report §6.1 |
| 20 | Unchanged pushes skip without new snapshots | PASS | Mac report |
| 21 | Divergence records a conflict without overwrite | NOT TESTED | W3 |
| 22 | `--keep-both` preserves both branches | NOT TESTED | W3 |
| 23 | All required GitHub checks are green | PASS | Mac report §7 |

**Counts: 8 PASS / 5 PARTIAL / 0 FAIL / 10 NOT TESTED.**

All 23 mandatory rows passed: **No.** Phase 1 remains open.

## 5. Findings

Release-blocking product findings: **none established on Device B**.

Acceptance-blocking finding:

1. The supplied private file has five syntactically non-empty `key=value`
   entries, but zero names match the exact RC7 five-key contract. Device B
   cannot safely construct the child-only storage/passphrase launcher from it.

Highest untested product risk remains tagged runbook §14b: the real-hardware
Windows Restart Manager path has still not been exercised with the exact target
session file held open.

## 6. Repository hygiene

- Branch `test/phase1-rc7-windows-report` was created from the peeled RC7 tag
  commit.
- `R2.txt` is locally excluded, untracked, and absent from the report branch.
- The only repository change is this sanitized report.
- No product code, secret, transcript, ciphertext, installer state, profile,
  remote object, tag, release, deployment, or product branch was changed.

## 7. Milestone block

```text
WINDOWS-RC7-W1-FAIL
release=v0.1.0-rc.7
tag_commit=66211599dd7cfb74f1436d2221b983050e8b1bc2
failure_gate=private_r2_contract
failure_exit=1
required_key_count=0
unknown_key_count=5
acceptance_state_mutated=false
remote_state_mutated=false
windows_report_path=docs/testing/results/2026-07-28-windows-phase1-rc7.md
END-WINDOWS-RC7-W1
```
