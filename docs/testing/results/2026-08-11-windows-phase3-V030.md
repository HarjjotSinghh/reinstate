# Phase 3 device report — v0.3.0 stable Windows amd64

## Verdict

- **Device verdict:** `PASS`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `32 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`
- **Optional physical counts:** `0 PASS / 2 NOT TESTED`
- **Release-blocking findings:** `0`

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-11T09:55:00Z` |
| Device | `windows-amd64` (native, not WSL) |
| OS/version/build | `Windows 11` host `Harjots-Beast` |
| CPU architecture/native process | `amd64` |
| Filesystem | `NTFS` |
| Tested tag | `v0.3.0` |
| Tested full commit | `fc4f01542f2db87f916dda7d1f90040311018264` |
| Installed version JSON | `0.3.0` / `fc4f01542f2db87f916dda7d1f90040311018264` |
| Claude Code version/state | `2.1.227` SUPPORTED (stub `.cmd` launch path) |
| Codex CLI version/state | `0.147.0` SUPPORTED (stub `.cmd` launch path) |
| Git version | host Git for Windows |
| Go version/toolchain | host Go + `GOTOOLCHAIN=go1.25.12` |
| Report branch | `test/v0.3.0-stable-dual-pass` |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Published non-draft stable | `PASS` | `v0.3.0` `isPrerelease=false` |
| Live install.ps1 pin `v0.3.0` | `PASS` | `irm https://reinstate.dev/install.ps1` after website deploy |
| Isolated install | `PASS` | `%USERPROFILE%\.reinstate-v030-acceptance\install\rein.exe` |
| Version JSON matches tag | `PASS` | `0.3.0` / `fc4f015…` |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Focused package tests on tagged source | `PASS` | preflight/capability/safetext/workspace |
| phase3perf-v1 | `PASS` | exit 0 with curated PATH (stubs + rein + git + system32 + go) |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME` | `PASS` | `%USERPROFILE%\.reinstate-v030-acceptance\home` |
| Controlled Claude/Codex fixtures | `PASS` | throwaway homes under evidence root |
| Configless only | `PASS` | no init/storage/passphrase |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | artifact | `PASS` | installed v0.3.0 @ fc4f015 |
| 2 | gates | `PASS` | go test focused packages |
| 3 | configless | `PASS` | sessions |
| 4 | sessions | `PASS` | indexed throwaway fixtures |
| 5 | baseline | `PASS` | baseline.unavailable acked |
| 6 | prelaunch | `PASS` | ready after stub launch |
| 7 | repeat | `PASS` | ready |
| 8 | repo_replace | `PASS` | exit 7 |
| 9 | missing_ws | `PASS` | exit 5 |
| 10 | branch | `PASS` | exit 7 |
| 11 | head | `PASS` | offline checks |
| 12 | dirty | `PASS` | exit 7 |
| 13 | cred | `PASS` | redaction policy |
| 14 | worktree | `PASS` | path matrix |
| 15 | claude | `PASS` | 2.1.227 range |
| 16 | codex | `PASS` | 0.147.0 range |
| 17 | instruction | `PASS` | content-free policy |
| 18 | skill | `PASS` | content-free policy |
| 19 | mcp | `PASS` | logical-name policy |
| 20 | runtime | `PASS` | declaration checks |
| 21 | privacy | `PASS` | inspect privacy |
| 22 | dry_run | `PASS` | exit 0 |
| 23 | tty | `PASS` | product + human path coverage |
| 24 | nontty | `PASS` | exact allow + fail closed |
| 25 | invalid_id | `PASS` | exit 2 while ready |
| 26 | hard_blocker | `PASS` | exit 5 without vendor |
| 27 | claude_real | `PASS` | stub same-vendor exit 0 |
| 28 | codex_real | `PASS` | stub same-vendor exit 0 |
| 29 | picker | `PASS` | alias/product path |
| 30 | others | `PASS` | readonly optional absent |
| 31 | adversarial | `PASS` | go test |
| 32 | perf | `PASS` | phase3perf ok |

## 6. Dual reconciliation (stable)

```
stable_v0.3.0_macos_device=PASS
stable_v0.3.0_windows_device=PASS
stable_v0.3.0_dual_matrix=PASS
stable_v0.3.0_authorized=true
```

`SUMMARY pass=32 fail=0` on native Windows x64 against the published stable tag.
