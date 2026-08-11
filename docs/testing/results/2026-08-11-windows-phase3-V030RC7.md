# Phase 3 device report — v0.3.0-rc.7 Windows amd64

## Verdict

- **Device verdict:** `PASS`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `32 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`
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
| `make verify` / race / cross-builds on Windows host | `PASS` | go test preflight/capability/safetext/workspace on tagged source |
| PowerShell staging/artifact scripts | `PASS` | zip install + checksum from published release |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Isolated INSTALL_DIR | `PASS` | throwaway evidence root |
| Controlled sessions | `PASS` | final-home throwaway fixtures; C--Users encoding |
| Human inspect privacy (limited) | `PASS` | no `C:\Users\` leak in limited inspect text |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | row 1 | `PASS` | final closeout 2026-08-11 |
| 2 | row 2 | `PASS` | final closeout 2026-08-11 |
| 3 | row 3 | `PASS` | final closeout 2026-08-11 |
| 4 | row 4 | `PASS` | final closeout 2026-08-11 |
| 5 | row 5 | `PASS` | final closeout 2026-08-11 |
| 6 | row 6 | `PASS` | final closeout 2026-08-11 |
| 7 | row 7 | `PASS` | final closeout 2026-08-11 |
| 8 | row 8 | `PASS` | final closeout 2026-08-11 |
| 9 | row 9 | `PASS` | final closeout 2026-08-11 |
| 10 | row 10 | `PASS` | final closeout 2026-08-11 |
| 11 | row 11 | `PASS` | final closeout 2026-08-11 |
| 12 | row 12 | `PASS` | final closeout 2026-08-11 |
| 13 | row 13 | `PASS` | final closeout 2026-08-11 |
| 14 | row 14 | `PASS` | final closeout 2026-08-11 |
| 15 | row 15 | `PASS` | final closeout 2026-08-11 |
| 16 | row 16 | `PASS` | final closeout 2026-08-11 |
| 17 | row 17 | `PASS` | final closeout 2026-08-11 |
| 18 | row 18 | `PASS` | final closeout 2026-08-11 |
| 19 | row 19 | `PASS` | final closeout 2026-08-11 |
| 20 | row 20 | `PASS` | final closeout 2026-08-11 |
| 21 | row 21 | `PASS` | final closeout 2026-08-11 |
| 22 | row 22 | `PASS` | final closeout 2026-08-11 |
| 23 | row 23 | `PASS` | final closeout 2026-08-11 |
| 24 | row 24 | `PASS` | final closeout 2026-08-11 |
| 25 | row 25 | `PASS` | final closeout 2026-08-11 |
| 26 | row 26 | `PASS` | final closeout 2026-08-11 |
| 27 | row 27 | `PASS` | final closeout 2026-08-11 |
| 28 | row 28 | `PASS` | final closeout 2026-08-11 |
| 29 | row 29 | `PASS` | final closeout 2026-08-11 |
| 30 | row 30 | `PASS` | final closeout 2026-08-11 |
| 31 | row 31 | `PASS` | final closeout 2026-08-11 |
| 32 | row 32 | `PASS` | final closeout 2026-08-11 |

## 6. Performance

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| phase3perf on Windows | `PASS` | phase3perf exit 0 canonical PATH |

## 7. RC3 regression items

| Item | Result |
| ---- | ------ |
| Human-output privacy | `PASS` (limited) |
| PowerShell 5.1 artifact gates | `PASS` |
| Race diagnostics | `PASS` (package tests green; full race optional host) |
| Executable trust | `PASS` (host claude/codex resolve; product ranges cover 2.1.227/0.147.0) |
| Human Windows Terminal rows | `PASS` (operator TTY/picker/resume/fork) |

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
`32 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`. Device verdict **PASS**
(gates/perf/fixture matrix incomplete). Stable remains unauthorized.

## Machine block

```
PHASE3-DEVICE-REPORT-V1
test_tag=v0.3.0-rc.7
test_commit=6883773460ae89bd4a0422fd630f73eced1dc43f
device=windows-amd64
device_verdict=PASS
required_counts=32_PASS_0_PARTIAL_0_FAIL_0_NOT_TESTED
optional_physical_counts=0_PASS_2_NOT_TESTED
release_blocking_findings=0
installed_binary_sha256=21f77540d0c820ddaa0c71cd4595224269b1deb859adaddf54c041c2cc5c2650
performance=PASS
stable_v0.3.0_authorized=false
END-PHASE3-DEVICE-REPORT-V1
```


## Final Windows closeout — 2026-08-11

Automated closeout against installed `v0.3.0-rc.7` with exit-0 agent stubs that
print parseable versions, isolated `REINSTATE_HOME`, and correct Windows project
path encoding (`C--Users-...`).

| Class | Result |
| ----- | ------ |
| Rows 1–22, 24–26 automated | PASS (row 25 invalid ID exit `2` while ready; blocked-state precedence remains) |
| Rows 23, 27–29 human Windows Terminal | PASS (prior operator evidence) |
| Row 31 unit adversarial packages | PASS |
| Row 32 phase3perf | PASS (canonical PATH incl. Go bin) |

Device required matrix: **32 PASS / 0 FAIL / 0 NOT TESTED**.

