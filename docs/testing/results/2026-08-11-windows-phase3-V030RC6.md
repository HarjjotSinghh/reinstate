# Reinstate v0.3.0-rc.6 Windows amd64 Phase 3 verified-resume report

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `6 PASS / 0 PARTIAL / 20 FAIL / 6 NOT TESTED`
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `6`

Native Windows x64 evidence collected over same-LAN SSH (`winps` / key auth)
into a process-scoped isolated install. Missing required human Windows Terminal
rows and incomplete installed product matrices are FAIL, never simulated.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-11T05:25:00Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 native x64 (HARJOTS-BEAST) |
| CPU architecture/native process | `amd64` native PowerShell 5.1 (not WSL) |
| Filesystem | NTFS |
| Tested tag | `v0.3.0-rc.6` |
| Tested full commit | `2ac3c49b68f40b1b7825e7ff737978de6091759b` |
| Installed binary SHA-256 | `a7a76aeed8479cc4531257bcef64297174a129b99664d90eb0228866fc8ddfcb` |
| Installed version JSON | `version=0.3.0-rc.6; commit=2ac3c49b68f40b1b7825e7ff737978de6091759b` |
| Claude Code version/state | `2.1.227`; SUPPORTED |
| Codex CLI version/state | `0.147.0`; SUPPORTED (`npm` install; extensionless resolution) |
| Report branch | `test/v0.3.0-rc.6-windows-amd64-report` |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tag/signature/commit chain | `PASS` | Shared with macOS coordinator evidence; tag `v0.3.0-rc.6` |
| 25-asset checksums | `PASS` | Full set downloaded/verified on coordinator; Windows zip present |
| Live `install.ps1` pin | `PASS` | live bootstrap pins `v0.3.0-rc.6`; matches commit |
| Isolated install | `PASS` | `INSTALL_DIR` under process-temp acceptance bin; `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` |
| Alias identity + full commit | `PASS` | rein.exe ≡ reinstate.exe SHA-256; version JSON full commit |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Host Go/Git present | `PASS` | go1.26.1 / Git 2.52 (host); product binary is tagged RC6 |
| `make verify` / race / PS artifact gates | `FAIL` | full Windows-native verify+race+snapshot/stage/check-release matrix not completed in this remote run |
| Phase 1/2 style regression on Windows source tree | `NOT TESTED` | not completed as full `go test ./...` on this host in-session |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Isolated `REINSTATE_HOME` | `PASS` | temp acceptance homes only |
| No rein init/storage secrets | `PASS` | configless doctor/sessions |
| Keyring over SSH | `PASS` (warn) | keyring unavailable warning expected without interactive logon session |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | Artifact/installer provenance | `PASS` | §2 |
| 2 | Full verification/race/cross-builds/regression | `FAIL` | incomplete Windows native gate battery this run |
| 3 | Fresh configless home | `PASS` | doctor without init |
| 4 | Fresh controlled Claude and Codex sessions | `FAIL` | throwaway homes prepared; full controlled vendor session creation matrix incomplete |
| 5 | First inspect baseline.unavailable | `NOT TESTED` | blocked by incomplete controlled sessions |
| 6 | Successful verified launch baseline | `FAIL` | not demonstrated |
| 7 | Unchanged baseline match | `FAIL` | depends on 6 |
| 8 | Repo replacement exit 7 | `NOT TESTED` | |
| 9 | Missing workspace block | `NOT TESTED` | |
| 10 | Branch/detached/unborn | `NOT TESTED` | |
| 11 | HEAD relations | `NOT TESTED` | |
| 12 | Dirty-tree privacy | `FAIL` | not completed on Windows |
| 13 | Credential remotes | `NOT TESTED` | |
| 14 | Worktree/symlink/Unicode/case | `FAIL` | incomplete (symlink privilege historically host-limited) |
| 15 | Claude executable/version/layout | `PASS` | doctor `SUPPORTED (2.1.227)` |
| 16 | Codex executable/version/layout | `PASS` | doctor `SUPPORTED (0.147.0)` |
| 17 | Instruction matrix | `FAIL` | incomplete |
| 18 | Skill matrix | `FAIL` | incomplete |
| 19 | MCP matrix | `FAIL` | incomplete |
| 20 | Runtime matrix | `FAIL` | incomplete |
| 21 | Inspect human/JSON | `FAIL` | not closed with controlled session corpus |
| 22 | Native dry-run | `FAIL` | not closed with controlled session corpus |
| 23 | TTY warning Windows Terminal | `FAIL` | required human Windows Terminal evidence absent |
| 24 | Non-TTY warning IDs | `FAIL` | incomplete |
| 25 | Invalid warning IDs | `FAIL` | incomplete |
| 26 | Hard blockers | `PASS` | agent version ranges now accept installed hosts; prior RC5 exit-5 cascade cleared for compatibility |
| 27 | Real Claude resume/fork | `FAIL` | incomplete |
| 28 | Real Codex resume/fork | `FAIL` | incomplete |
| 29 | Picker both aliases | `FAIL` | human picker matrix absent |
| 30 | Gemini/OpenCode read-only | `PASS` | optional vendors not installed; not claimed supported |
| 31 | Adversarial installed matrix | `FAIL` | incomplete |
| 32 | Latency ceilings | `FAIL` | Windows `phase3perf` not completed this run |

## 6. RC3 regression items

| Item | Result |
| ---- | ------ |
| 1. Human-output privacy | `FAIL` (not re-proven this run) |
| 2. PowerShell 5.1 artifact gates | `FAIL` (not re-run this session) |
| 3. Race diagnostics | `FAIL` (not completed) |
| 4. Executable trust PATHEXT | `PASS` (Codex/Claude resolve; versions SUPPORTED) |
| 5. Human Windows Terminal rows | `FAIL` (missing) |

## 7. Findings

### Release-blocking

1. Full Windows automated gate battery (verify/race/snapshot/stage) not completed remotely this session.
2. Controlled session + inspect/dry-run/resume matrices incomplete.
3. Real same-vendor resume/fork not demonstrated.
4. Human Windows Terminal picker/warning rows missing.
5. Windows phase3perf not executed.
6. Several workspace/capability mutation rows incomplete.

### Non-blocking

- RC6 agent version widen confirmed: Claude 2.1.227 and Codex 0.147.0 SUPPORTED on native Windows (primary RC5 failure mode cleared).
- SSH remote shell control from macOS works for non-TTY automation.

## 8. Required terminated device block

```text
PHASE3-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.3.0-rc.6
test_commit=2ac3c49b68f40b1b7825e7ff737978de6091759b
installed_binary_sha256=a7a76aeed8479cc4531257bcef64297174a129b99664d90eb0228866fc8ddfcb
required_pass=6
required_partial=0
required_fail=20
required_not_tested=6
optional_physical_pass=0
optional_physical_not_tested=2
baseline_provenance=FAIL
workspace_git=FAIL
agent_compatibility=PASS
capability_privacy=FAIL
resume_fork=FAIL
picker=FAIL
performance=FAIL
phase1_phase2_regression=FAIL
release_blocking_findings=6
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
```
