# Phase 4 structured-handoff report — v0.4.0-rc.3 Windows amd64

Copy of the Phase 4 template with this dispatch's substitution table. Cumulative, sanitized, append-only. `stable_v0.4.0_authorized=false`. `current_stable=v0.3.0`.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `30 PASS / 0 PARTIAL / 14 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `0 PASS / 6 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `9`

`PARTIAL` and `NOT TESTED` do not pass a required row. Missing required TTY evidence is `FAIL`.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-13T06:16:38Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 native x64, `10.0.26200.0`; PowerShell `5.1.26100.8328` 64-bit; machine `HARJOTS-BEAST` |
| CPU architecture/native process | amd64; native 64-bit PowerShell; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.3` |
| Tested full commit | `34e85763380fd733b472e8560eb11e9ecf8d81cb` |
| Installed binary SHA-256 | `e7183dd85d8b1c254cec4ac4c04007c2b98091f4ba11de4ec1a97eb42527ae49` |
| Installed version JSON | `reinstate 0.4.0-rc.3` / `34e85763380fd733b472e8560eb11e9ecf8d81cb` / `2026-08-13T05:31:19Z` |
| Claude Code version/state | before `2.1.229` in range `2.1.219–2.1.229`; after last row `2.1.229` unchanged; source and destination |
| Codex CLI version/state | before `0.147.0` in range `0.133.0–0.147.0`; after last row `0.147.0` unchanged; source and destination |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` present; OpenCode `1.18.2` present, **not isolated**; Grok `0.2.101` present. Unchanged after last row. |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | host `go1.26.1`; gates used `GOTOOLCHAIN=go1.25.12` → `go1.25.12 windows/amd64` |
| Report branch | `test/v0.4.0-rc.3-windows-amd64-report` |
| Device-report commit | filled after this file is committed |
| Draft report PR | filled after open |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | annotated tag object; peels to TEST_COMMIT; published prerelease used |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | `git rev-parse v0.4.0-rc.3^{}` = `34e85763380fd733b472e8560eb11e9ecf8d81cb`; `merge-base --is-ancestor` vs `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub prerelease `v0.4.0-rc.3` |
| Exact expected release asset set is present | `PASS` | published 25-asset set for this tag (coordinator-verified; this run consumed the live Windows install path) |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | installed binary reports the peeled commit |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | `stage-release-assets.ps1` and `check-release-artifacts.ps1` exit 0 on tagged worktree |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.3` | `PASS` | `https://reinstate.dev/install.ps1` sha256 `8f8603cd73db0dc0f32e336894dba5e9d7f5cf1474394e995f02ed02485b5fcf`; `$Version = "v0.4.0-rc.3"`; no surviving rc.1/rc.2 pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | live digest re-proved on Mac fetch and again on the Windows host before install |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `INSTALL_DIR` in a new acceptance tree; `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.3` | `PASS` | both sha256 `e7183dd85d8b1c254cec4ac4c04007c2b98091f4ba11de4ec1a97eb42527ae49` |
| Installed binary reports the literal full tested commit | `PASS` | `34e85763380fd733b472e8560eb11e9ecf8d81cb` |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | exit 0, 851 ms, Go 1.25.12 |
| `make verify` with Go 1.25.12 | `FAIL` | GNU Make 4.4.1 (MSYS2); `internal/adapter/claude` `TestClaudeDefaultRootSupportedVersionFileDoesNotRequireExecutable`; `internal/cli` `TestCLISyntheticSyncPath`; `TestDiffMissingManifestUsesAuthStorageExit` |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | same three tests; no DATA RACE lines observed in fail list |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | `stage-release-assets.ps1`, `check-release-artifacts.ps1`, `test-install.ps1` exit 0; snapshot/binary-identity harness capture noisy, not used as the sole fail |
| Phase 1, Phase 2, and Phase 3 regression | `FAIL` | blocked by `make verify` / race package fails above |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | `TestHandoffExecutedOutputMatchesDryRunByteForByte` PASS |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | `go test ./internal/handoff ./internal/fixture` exit 0 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | `10.9662ms`; projection `46541` bytes |

WinGet Links shims for goreleaser/syft are untrusted mount points; package-directory binaries were used. Ordinary Microsoft Defender realtime remained enabled. An exclusion was added only for the isolated acceptance tree after Defender quarantined `rein.exe` mid-matrix (harness deviation).

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new `v040rc3` acceptance tree; five isolation variables set before product commands |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | tagged synthetic fixtures copied into throwaway homes; no rc.1/rc.2 homes |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | only isolated `CLAUDE_CONFIG_DIR` / `CODEX_HOME` / `GEMINI_CLI_HOME` / `GROK_HOME` |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | `rein init` refused empty S3 endpoint; no push/pull of capsules |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | dummy MCP/skill files created then removed under the isolated Claude home only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | D10 source sha256 unchanged; no-launch/dry-run did not spawn vendors except the F8 SSH-PTY mis-test, which was superseded by `ssh -T` |

OpenCode has no home override: E3/E4 `NOT TESTED`; D7 does not claim an isolated OpenCode result.

## 5. Required matrix — 44 rows

Every cross-agent result is a **new destination session continuing the same task**.

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | missing required Windows Terminal evidence for real-TTY Claude → Codex launch; SSH is not that TTY; ConPTY was not invented |
| A2 | `FAIL` | missing required Windows Terminal evidence for real-TTY Codex → Claude launch |
| A3 | `FAIL` | logged-out repeat requires the A1 TTY launch |
| A4 | `PASS` | tagged partial-final-record fixture; installed `--no-launch --json` exit 0 |
| A5 | `FAIL` | destination restatement requires a real destination launch; not invented |
| A6 | `FAIL` | duplicate-effects check requires a real destination launch |
| A7 | `PASS` | `handoff list --json` showed stored lineage for the no-launch/F8 artifact; inspect later exit 0 |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | tagged unit test PASS; installed `--dry-run --json` and `--no-launch --json` both exit 0 on the same frozen source |
| B2 | `PASS` | tagged 200-turn fixture `--policy balanced --dry-run --json` exit 0 |
| B3 | `PASS` | `--policy checkpoint --dry-run --json` exit 0 |
| B4 | `PASS` | `--policy balanced --dry-run --json` exit 0 |
| B5 | `PASS` | `--policy full --dry-run --json` exit 0 |
| B6 | `FAIL` | sampled `fidelity.json` contained exact, normalized, referenced, omitted; `summarized` not present |
| B7 | `PASS` | attachments fixture `--dry-run --json` exit 0 |
| B8 | `PASS` | unknown-records fixture `--dry-run --json` exit 0 |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | plan reported `branch=fixture/windows` and `dirty=true` matching the disposable Git checkout |
| C2 | `PASS` | live Git dirty/HEAD won over fixture transcript claims |
| C3 | `FAIL` | missing-workspace rename harness failed; row not completed |
| C4 | `PASS` | JSON `code=compatibility`: working directory is a different repository than the source session |
| C5 | `PASS` | persisted capsule used `${REPO:demo}` and did not contain the fixture-user absolute path token |
| C6 | `FAIL` | missing MCP not shown as a distinct pre-launch report on the attempted fixture |
| C7 | `FAIL` | missing skill/instruction not shown as a distinct pre-launch report |
| C8 | `FAIL` | PATH shim for `2.1.230` did not fail-close; dry-run still produced a plan (harness did not bind Go `LookPath`) |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | injection fixture `--dry-run --json` exit 0; content not pasted |
| D2 | `PASS` | delimiter/fence fixture `--dry-run --json` exit 0 |
| D3 | `PASS` | sampled projection had no source system/developer instruction |
| D4 | `PASS` | secret fixture with `--show-redactions`; planted credential patterns absent from captured plan JSON |
| D5 | `PASS` | launch-free rows used isolated homes only |
| D6 | `PASS` | `handoffs/` created under isolated `REINSTATE_HOME`, outside the repository |
| D7 | `PASS` | no-launch/dry-run used isolated vendor homes; OpenCode not collected |
| D8 | `FAIL` | no S3/R2 config; live `push`/`pull` exclusion of `handoffs/` not exercised |
| D9 | `FAIL` | Grok `--no-redact` returned `session not found` (isolated Grok home layout); warning surface not collected |
| D10 | `PASS` | source session file sha256 identical before and after `--no-launch` |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | help and JSON use `structured handoff` / new destination session; `destination_session_mode=new` |
| F2 | `PASS` | `rein resume claude:<id> --with codex --dry-run --json` exit 0; mode `structured handoff` |
| F3 | `PASS` | `--json` without `--dry-run`/`--no-launch` emitted usage JSON requiring those flags (public usage / exit 2 class) |
| F4 | `PASS` | unknown `--allow-warning` ID: usage JSON `is not a current warning` |
| F5 | `PASS` | `--no-launch --json` exit 0; plan printed; no destination spawn |
| F6 | `PASS` | `handoff inspect <id> --json` exit 0 on stored lineage |
| F7 | `FAIL` | two-agent bare native ID collision was not constructed |
| F8 | `PASS` | `ssh -T` launch with exact warning IDs fail-closed: native agent resume/fork requires an interactive terminal; re-run from a real TTY or use `--dry-run`. No spawn. First SSH PTY attempt was superseded (not a non-TTY). |

### Matrix G — performance

Antivirus realtime stayed enabled.

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run | 10.9662 ms | 10.9662 ms | 10.9662 ms | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | projection `46541` bytes; 400-event fixture |
| G2 | 1 warmup + 5 installed `--dry-run --json` | 3446 ms | 3566 ms | 3566 ms | Windows max `12s` | `PASS` | long-history frozen source; policy balanced; all samples exit 0 |
| G3 | 1 warmup + 20 `handoff list --limit 100 --json` | 73 ms | 95 ms | 95 ms | Windows p95 `4s` | `FAIL` | 100 `--no-launch` attempts into a dedicated home; subsequent `list --limit 100 --json` returned **0** lineage rows (stdout discarded during setup; likely unacknowledged warnings). Timings are empty-store and not accepted. |

## R1–R6 and RC2 regressions (not in the 44-count)

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `PASS` | throwaway Claude home had `projects/` and **no** `version` file; `doctor` `agent.claude: SUPPORTED (layout-projects-jsonl-v1)` |
| R2 | `PASS` | absolute-path fixture plan used `${REPO:…}` tokens; persisted capsule had no fixture-user absolute path |
| R3 | `PASS` | dirty disposable tree appeared in `changed_files` (not empty / none) |
| R4 | `FAIL` | timed-out out-of-range probe classified UNTESTED was not collected |
| R5 | `PASS` | slash-commands fixture `--dry-run --json` exit 0 |
| R6 | `FAIL` | host Claude `2.1.229` stayed in range and unchanged; fake `2.1.230` fail-closed exit 5 not proven |
| RC2 C4 | `PASS` | wrong-repo refused |
| RC2 F8 | `PASS` | non-TTY fail-closed |
| RC2 E5/E6 | `FAIL` | Grok source-only not collected |
| RC2 R4 | `FAIL` | see R4 |
| RC2 D9 | `FAIL` | see D9 |
| RC2 list isolation | `PASS` | after seeding one fixture, `rein list --json` showed only that session |
| RC2 B3 | `PASS` | checkpoint policy |
| RC2 B6 | `FAIL` | see B6 |
| RC2 C6/C7 | `FAIL` | see C6/C7 |

## 6. Optional source-only matrix — 8 rows

Gemini, OpenCode, and Grok are source-only. Destinations remain Claude Code and Codex only.

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` `0.53.0` | `FAIL` | Gemini source `--last` not collected in the isolated home |
| E2 | `yes` `0.53.0` | `FAIL` | same |
| E3 | `yes` `1.18.2` | `NOT TESTED` | OpenCode has no isolation override; skipped; never claimed isolated |
| E4 | `yes` `1.18.2` | `NOT TESTED` | same |
| E5 | `yes` `0.2.101` | `FAIL` | Grok isolated home: `session not found` |
| E6 | `yes` `0.2.101` | `FAIL` | same |
| E7 | `yes` | `FAIL` | source-only destination refusal not executed |
| E8 | `yes` | `FAIL` | Grok native resume/fork refusal not executed |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree at TEST_COMMIT |
| `make verify` is green on both mandatory platforms | `FAIL` | Windows `make verify` FAIL |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 |
| Every fidelity class has real report evidence and byte-stable goldens | `FAIL` | B6 |
| Injection, secret, and bounded-read security gates leak nothing | `FAIL` | D8–D9 incomplete |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS` | C5 |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 unit |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | F3–F4 + D7; D8 sync not run |

## 8. Findings and repository hygiene

### Release-blocking

1. Flagship A1–A3 and A5–A6: missing Windows Terminal real-TTY destination-launch evidence (human-owned; ConPTY not invented).
2. `make verify` and `go test -race` fail three unit tests on this host (`TestClaudeDefaultRootSupportedVersionFileDoesNotRequireExecutable`, `TestCLISyntheticSyncPath`, `TestDiffMissingManifestUsesAuthStorageExit`).
3. B6: `summarized` fidelity class not present in the sampled report.
4. C3, C6, C7: missing-workspace and missing MCP/skill rows not proven.
5. C8/R6: out-of-range Claude `2.1.230` fail-closed exit 5 not proven.
6. D8/D9: capsule sync exclusion and Grok warning/`--no-redact` not proven.
7. G3: dedicated 100-row lineage list returned 0 rows; timings rejected.
8. F7 and R4: ambiguous native ID and timed-out UNTESTED classification not collected.
9. Optional installed Gemini/Grok E rows failed or were not collected (candidate blockers). Human-owned WT picker / console warning-ack / in-prompt repo-swap remain FAIL.

### Non-blocking

- WinGet `Links` shims cannot be executed (untrusted mount point); package-directory binaries used.
- GnuWin32 Make 3.81 is on PATH; gates used MSYS2 Make 4.4.1.
- Defender quarantined the isolated `rein.exe` after an SSH-PTY launch attempt; exclusion of the acceptance tree only; realtime stayed on.
- `rein init` requires S3/R2 endpoint; handoff store worked without it.
- Start-Process/cmd `%ERRORLEVEL%` without delayed expansion reported 0; classifications used JSON `code` bodies.

### Test-harness deviations and supersessions

- F8: first SSH (PTY) launch superseded by `ssh -T` non-TTY fail-closed.
- F2: `--last` is not a `resume` flag; superseded by `resume claude:<id> --with codex --dry-run --json`.
- R1/LISTISO first-pass string match superseded by doctor/list JSON bodies.
- G3 empty-store timings superseded as FAIL.
- Report-only diff: this file. Merge-base with TEST_COMMIT is TEST_COMMIT. Product files unchanged.

## 9. Required terminated device block

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.3
test_commit=34e85763380fd733b472e8560eb11e9ecf8d81cb
installed_binary_sha256=e7183dd85d8b1c254cec4ac4c04007c2b98091f4ba11de4ec1a97eb42527ae49
required_pass=30
required_partial=0
required_fail=14
required_not_tested=0
optional_pass=0
optional_fail=6
optional_not_tested=2
artifact_chain=PASS
isolation_privacy=PASS
flagship_directions=FAIL
fidelity_policy=FAIL
workspace_capability=FAIL
security=FAIL
cli_contract=FAIL
performance=FAIL
phase1_phase2_phase3_regression=FAIL
release_blocking_findings=9
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
