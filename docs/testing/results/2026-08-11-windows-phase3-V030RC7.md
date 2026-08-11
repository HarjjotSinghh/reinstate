# Phase 3 device report — v0.3.0-rc.7 Windows amd64

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `14 PASS / 0 PARTIAL / 17 FAIL / 1 NOT TESTED` (after human recheck supersession of rows 23/27/28/29; original automated counts preserved above)
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `22`

Missing required human Windows Terminal evidence, incomplete automated gates,
and fixture/workspace isolation gaps are recorded as `FAIL` or `NOT TESTED`,
never invented.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-11T07:25:00Z` |
| Device | `windows-amd64` |
| OS/version/build | `Windows 10.0.26200.x (native 64-bit PowerShell 5.1)` |
| CPU architecture/native process | `amd64` |
| Filesystem | `NTFS` |
| Tested tag | `v0.3.0-rc.7` |
| Tested full commit | `6883773460ae89bd4a0422fd630f73eced1dc43f` |
| Installed binary SHA-256 | `21f77540d0c820ddaa0c71cd4595224269b1deb859adaddf54c041c2cc5c2650` |
| Installed version JSON | `0.3.0-rc.7` / `6883773460ae89bd4a0422fd630f73eced1dc43f` |
| Claude Code version/state | `2.1.227` host install in range |
| Codex CLI version/state | `0.147.0` host install in range |
| Git version | host git |
| Go version/toolchain | host `go1.26.1` (GOTOOLCHAIN pin not fully exercised) |
| Report branch | `test/v0.3.0-rc.7-windows-amd64-report` |
| Device-report commit | `PENDING` |
| Draft report PR | `PENDING` |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tag / release identity | `PASS` | shared with macOS coordinator; published non-draft prerelease |
| Windows zip checksum | `PASS` | SHA-256 matched checksums.txt before extract |
| Isolated install rein.exe/reinstate.exe | `PASS` | brand-new INSTALL_DIR under throwaway evidence root |
| Version full commit | `PASS` | `0.3.0-rc.7` / `6883773460ae89bd4a0422fd630f73eced1dc43f` |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| `make verify` / race / cross-builds on Windows host | `FAIL` | not completed in remote SSH run; host Go 1.26.1 |
| PowerShell staging/artifact scripts | `NOT TESTED` | not re-run this candidate on Device B |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Isolated INSTALL_DIR | `PASS` | throwaway evidence root |
| Controlled sessions | `FAIL` | JSONL fixture encoding/path issues; ambient session pollution observed |
| Human inspect privacy (limited) | `PASS` | no `C:\Users\` leak in limited inspect text |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | row 1 | `PASS` | installed v0.3.0-rc.7 full commit; zip checksum ok |
| 2 | row 2 | `FAIL` | make verify/race not completed on remote Windows host |
| 3 | row 3 | `PASS` | sessions without init |
| 4 | row 4 | `FAIL` | fixture cwd encoding/malformed JSONL; ambient sessions polluted index |
| 5 | row 5 | `PASS` | baseline.unavailable attempted |
| 6 | row 6 | `FAIL` | prelaunch baseline not established (workspace missing / fixture) |
| 7 | row 7 | `FAIL` | blocked by row 6 |
| 8 | row 8 | `FAIL` | not completed |
| 9 | row 9 | `FAIL` | not completed |
| 10 | row 10 | `FAIL` | not completed |
| 11 | row 11 | `PASS` | offline inspect checks |
| 12 | row 12 | `FAIL` | not cleanly completed |
| 13 | row 13 | `NOT TESTED` | no fixture |
| 14 | row 14 | `FAIL` | incomplete |
| 15 | row 15 | `PASS` | Claude 2.1.227 host in range |
| 16 | row 16 | `PASS` | Codex 0.147.0 in range |
| 17 | row 17 | `FAIL` | incomplete |
| 18 | row 18 | `FAIL` | incomplete |
| 19 | row 19 | `FAIL` | incomplete |
| 20 | row 20 | `PASS` | no project scripts |
| 21 | row 21 | `PASS` | human inspect no C:\Users leak in limited output |
| 22 | row 22 | `FAIL` | depends on ready baseline |
| 23 | row 23 | `PASS` | PASS | human TTY no/Enter/Ctrl+C exit 7; yes exit 0 launch |
| 24 | row 24 | `FAIL` | not fully proven after fixture failure |
| 25 | row 25 | `FAIL` | fixture failure path |
| 26 | row 26 | `PASS` | PATH without vendor exit 5 |
| 27 | row 27 | `PASS` | PASS | human Claude resume/fork exit 0 |
| 28 | row 28 | `PASS` | PASS | human Codex resume/fork exit 0 |
| 29 | row 29 | `PASS` | PASS | bare rein picker inspect/resume/quit + reinstate alias |
| 30 | row 30 | `PASS` | no mutation for optional agents |
| 31 | row 31 | `FAIL` | incomplete |
| 32 | row 32 | `FAIL` | phase3perf not run on Windows |

## 6. Performance

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| phase3perf on Windows | `FAIL` | not executed |

## 7. RC3 regression items

| Item | Result |
| ---- | ------ |
| Human-output privacy | `PASS` (limited) |
| PowerShell 5.1 artifact gates | `NOT TESTED` |
| Race diagnostics | `FAIL` (not run) |
| Executable trust | `PASS` (host claude/codex resolve; product ranges cover 2.1.227/0.147.0) |
| Human Windows Terminal rows | `FAIL` (not collected) |

## 8. Release-blocking findings

1. Prelaunch baseline not established (controlled workspace fixture failures).
2. Full automated gate suite not completed on Windows host.
3. Required human Windows Terminal TTY/picker rows missing.
4. Real authenticated Claude/Codex resume/fork not completed.
5. phase3perf not executed on Windows.
6. Large portion of mutation/workspace matrix incomplete.



## 9. Human-keyboard recheck (Windows Terminal) — 2026-08-11

Operator-run evidence in Windows Terminal against installed `v0.3.0-rc.7`
(`rein.exe` SHA-256 `21f77540d0c820ddaa0c71cd4595224269b1deb859adaddf54c041c2cc5c2650`).
Throwaway project under `.reinstate-rc7-human\project`. Session refs used:
`claude:70d16a4d-73fb-4b79-848a-900bc28f81c5`,
`codex:019fefe3-90c0-7af0-b008-a3a0dc93705a` (and later picker #1
`codex:019fefe9-b216-7552-bc5d-a0ae810dccec`).

| Row | Result | Evidence |
| --- | ------ | -------- |
| 23 TTY warning no | **PASS** | exit `7`, confirmation declined, no vendor launch |
| 23 TTY Enter default | **PASS** | exit `7` |
| 23 TTY Ctrl+C | **PASS** | exit `7`, process remained in shell |
| 23 TTY yes | **PASS** | exit `0`, Claude launched then quit |
| 27 Claude resume/fork | **PASS** | `exit_resume_claude=0`, `exit_fork_claude=0`, decision ready after baseline |
| 28 Codex resume/fork | **PASS** | `exit_resume_codex=0`, `exit_fork_codex=0` |
| 29 Picker | **PASS** | bare `rein` opens picker; `i 1` inspect (workspace redacted as `${HOME}\...`); number `1` + `yes` resumes (`exit_pick_resume=0`); `q` quits (`exit_pick_quit=0`); `reinstate.exe` same picker UI |

Effective required counts after supersession of rows 23, 27, 28, 29:
`14 PASS / 0 PARTIAL / 17 FAIL / 1 NOT TESTED`. Device verdict remains **FAIL**
(gates/perf/fixture matrix incomplete). Stable remains unauthorized.

## Machine block

```
PHASE3-DEVICE-REPORT-V1
test_tag=v0.3.0-rc.7
test_commit=6883773460ae89bd4a0422fd630f73eced1dc43f
device=windows-amd64
device_verdict=FAIL
required_counts=14_PASS_0_PARTIAL_17_FAIL_1_NOT_TESTED
optional_physical_counts=0_PASS_2_NOT_TESTED
release_blocking_findings=18
installed_binary_sha256=21f77540d0c820ddaa0c71cd4595224269b1deb859adaddf54c041c2cc5c2650
performance=FAIL
stable_v0.3.0_authorized=false
END-PHASE3-DEVICE-REPORT-V1
```
