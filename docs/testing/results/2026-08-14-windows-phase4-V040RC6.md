# Phase 4 structured-handoff report — v0.4.0-rc.6 Windows amd64

Copy of the Phase 4 template with this dispatch's substitutions. Cumulative and
sanitized. No transcript text, prompts, responses, capsule bodies, credentials,
private paths, filenames, diffs, or raw child-process output.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `3 PASS / 0 PARTIAL / 41 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `0 PASS / 6 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `4`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. OpenCode rows are `NOT TESTED` because that vendor has no home
override. Gemini and Grok were already installed and were not installed for
acceptance. `stable_v0.4.0_authorized=false`; `current_stable=v0.3.0`.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-14T18:45:00Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| CPU architecture/native process | AMD64; 64-bit Windows PowerShell 5.1.26100.8328 Desktop; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.6` |
| Tested full commit | `00366cb53f4246bde50c7f6fe96f7c1e1660ec88` |
| Installed binary SHA-256 | `8a42ccbecf91ee7f9d969de4169c1ec1b12fcf4c8bb7e8ac7e1bed72474ff5bd` |
| Installed version JSON | version `0.4.0-rc.6`; commit `00366cb53f4246bde50c7f6fe96f7c1e1660ec88`; date `2026-08-14T17:09:29Z` |
| Claude Code version/state | before first product command: host global `2.1.232` unused; test PATH pin `2.1.229` source+destination; after last row pin still `2.1.229` |
| Codex CLI version/state | `0.147.0` source+destination; unchanged after last row |
| Gemini/OpenCode/Grok state | Gemini `0.53.0`; OpenCode `1.18.2` (not isolated; omitted from process PATH); Grok `0.2.101`; all unchanged after last row |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12 windows/amd64` via `GOTOOLCHAIN=go1.25.12` (host default go1.26.1 unused) |
| Report branch | `test/v0.4.0-rc.6-windows-amd64-report` |
| Device-report commit | `this PR commit` |
| Draft report PR | filled after `gh pr create --draft` |

Host identity: native Windows 11 x64, computer name HARJOTS-BEAST, user admin.
Ordinary Microsoft Defender real-time protection was enabled. GNU Make 4.4.1
(MSYS2), MinGW-w64 gcc 16.1.0, goreleaser 2.17.0 (WinGet package directory, not
the 0-byte Links stub), syft 1.50.0.

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | `git verify-tag` Good git SSH signature, ED25519, allowed_signers identity |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | annotated tag `v0.4.0-rc.6`; peeled commit equals TEST_COMMIT; ancestor of github/main |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub API: draft false, prerelease true, tag `v0.4.0-rc.6`, target main |
| Exact expected release asset set is present | `PASS` | 25 uploaded assets including windows_amd64 zip/exe/sbom and checksums.txt |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | zip digest `241ae299d7a10be60271e0b1ea8ada05b3f2ff22aaa0373c42c18cfb03678973`; exe digest equals installed binary `8a42ccbecf91ee7f9d969de4169c1ec1b12fcf4c8bb7e8ac7e1bed72474ff5bd` |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | PowerShell `check-release-artifacts.ps1` exit 0 on snapshot dist |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.6` | `PASS` | live `install.ps1` sha256 `25bfb4c493c3495b40b121137213df6c3e822d1b74e72264b497da4a83d56aa8`; version pin `v0.4.0-rc.6`; no rc.1–rc.5 pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | live pin is the published installer with baked `v0.4.0-rc.6`; tagged `scripts/install.ps1` remains env-pinned (`02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe`) |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; User PATH not persisted; pre-existing user binary hash unchanged |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.6` | `PASS` | identical sha256; version JSON name reinstate, version 0.4.0-rc.6 |
| Installed binary reports the literal full tested commit | `PASS` | version `--json` commit field is the 40-character TEST_COMMIT |

Any failure here stops product-behavior testing. Chain passed; product matrix continued.

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | detached TEST_COMMIT; tidy -diff exit 0 |
| `make verify` with Go 1.25.12 | `FAIL` | exit 2. Two tests blocked ~63.6s on hanging `--version` grandchild pipes: `TestRunVersionCommandUnblocksGrandchildPipes`, `TestExecRunnerWaitDelayUnblocksGrandchildPipes`. Isolation env leak of `GROK_HOME` first also failed `TestSessionBusyAcceptsSourceOnlyAgents` (304s); superseded by a clean-env rerun that dropped the Grok row only |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | included in `make verify`; same two grandchild failures |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | real goreleaser package path (Links stub is 0-byte and cannot execute). `snapshot.ps1` 0; `stage-release-assets.ps1` 0; `check-release-artifacts.ps1` 0; `test-install.ps1` 0 |
| Phase 1, Phase 2, and Phase 3 regression | `FAIL` | folded into `make verify` FAIL above |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `FAIL` | G2 installed dry-run samples not collected; Plan route blocked |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | fixture-scan exit 0 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | 11.7589 ms; projection 46541 bytes |

First snapshot attempt used the WinGet `Links` goreleaser symlink (0 bytes) and
failed `NativeCommandFailed`. Superseded by invoking the package-directory
binary. Not a product FAIL.

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new rc.6 acceptance tree; five isolation variables set before product commands |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | tagged synthetic fixtures copied into throwaway homes; no rc.1–rc.5 homes reused |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | first `sessions --agent claude --json` exit 0, 1020 ms, no operator-home sessions |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | no `init`/push/pull |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | throwaway Claude/Codex/Gemini/Grok homes only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | no destination vendor launched; F8 refused before spawn |

OpenCode has no home override: E3/E4 `NOT TESTED`. D7 does not claim an isolated
OpenCode result. OpenCode executable omitted from the test PATH.

## 5. Required matrix — 44 rows

Plan-based rows (`handoff --dry-run` / `--no-launch`) failed with
`handoff: session busy check: exit status 1` after ~305 s. On Windows the busy
check lists processes via `powershell Get-CimInstance Win32_Process`. Under
host load that command returned exit 1. F8 refuses before Plan, so it did not
hit this path. Missing dest-ack is FAIL.

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | dest-ack not collected; Plan blocked by busy-check; no Windows Terminal first-reply |
| A2 | `FAIL` | dest-ack not collected; same busy-check / console rule |
| A3 | `FAIL` | empty dest `--dry-run --json` exit 1, ~305 s, `session busy check: exit status 1`; dest was not UNTESTED solely for empty layout — Plan never completed |
| A4 | `FAIL` | partial-final fixture Plan blocked by the same busy-check |
| A5 | `FAIL` | dest restatement not collected |
| A6 | `FAIL` | completed-marker non-repeat not collected |
| A7 | `FAIL` | `handoff list --json` exit 0, mode `structured handoff`, `handoffs: []`; no stored lineage from a successful `--no-launch` |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `FAIL` | installed dry-run vs `--no-launch` byte compare not collected; Plan blocked |
| B2 | `FAIL` | long-history `--dry-run --json` not completed |
| B3 | `FAIL` | checkpoint policy Plan blocked |
| B4 | `FAIL` | balanced policy Plan blocked |
| B5 | `FAIL` | full policy Plan blocked |
| B6 | `FAIL` | compaction fixture Plan blocked; `summarized` class not observed |
| B7 | `FAIL` | attachments fixture Plan blocked |
| B8 | `FAIL` | unknown-records fixture Plan blocked |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `FAIL` | Plan blocked before workspace bind |
| C2 | `FAIL` | Git-vs-transcript contradiction not collected |
| C3 | `FAIL` | missing-workspace block not collected |
| C4 | `FAIL` | fixture-user/macos `demo` leaf from a differently named git repo: `--dry-run --json` exit 1 at ~305 s (`session busy check`), not exit 5 different-repository. RC6 leaf-match not reached |
| C5 | `FAIL` | foreign-OS macos os-roots `--dry-run --json` from a local `demo` checkout: same busy-check exit 1; `${REPO:}` remap not observed |
| C6 | `FAIL` | destination MCP-gap warning not collected |
| C7 | `FAIL` | destination skill/instruction gap not collected |
| C8 | `FAIL` | 2.1.230 fail-closed Plan not collected |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `FAIL` | prompt-injection fixture Plan blocked |
| D2 | `FAIL` | fence-breakout fixture Plan blocked |
| D3 | `FAIL` | source-instruction exclusion not scanned on a produced projection |
| D4 | `FAIL` | secret-leakage fixture Plan blocked |
| D5 | `PASS` | launch-free F3/F8/list/doctor only; no credential files read |
| D6 | `PASS` | handoffs store under isolated `REINSTATE_HOME` |
| D7 | `PASS` | five isolation variables set; Claude session list had no operator-home sessions; OpenCode skipped |
| D8 | `PASS` | no push/pull executed; capsule sync not used |
| D9 | `FAIL` | Grok `--no-redact` Plan blocked by busy-check |
| D10 | `FAIL` | source hash-before/after not collected on a completed Plan |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `FAIL` | `handoff list --json` prints `structured handoff`; Plan JSON surfaces not collected |
| F2 | `FAIL` | `resume --with` Plan blocked |
| F3 | `PASS` | `--json` without `--dry-run`/`--no-launch` exit 2 in ~1020–1036 ms |
| F4 | `FAIL` | warning-ack matrix not completed (unknown-ID case not independently recorded after busy-check failures) |
| F5 | `FAIL` | `--no-launch --json` exit 1, ~305 s, busy-check |
| F6 | `FAIL` | inspect/export of stored artifacts not collected (`handoffs: []`) |
| F7 | `FAIL` | ambiguous native ID not collected |
| F8 | `PASS` | non-TTY dest launch exit 7 in 1021–1032 ms; `--allow-warning` still exit 7. RC2/RC5/RC6 refuse-before-Plan: wall time is not spawn-scale (not multi-second LookPath/child start). Index/Plan not entered |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run | 11.76 ms | 11.76 ms | 11.76 ms | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | projection 46541 bytes |
| G2 | 1 warmup + 5 installed `--dry-run` |  |  |  | Windows max `12s` | `FAIL` | Plan blocked; no valid samples |
| G3 | 1 warmup + 20 `handoff list --limit 100 --json` |  |  |  | Windows p95 `4s` | `FAIL` | 100 `--no-launch` setup not completed |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` | `FAIL` | Gemini installed; Plan blocked by Claude-source busy-check |
| E2 | `yes` | `FAIL` | same |
| E3 | `yes` | `NOT TESTED` | OpenCode 1.18.2 present; no isolation override |
| E4 | `yes` | `NOT TESTED` | same OpenCode isolation skip |
| E5 | `yes` | `FAIL` | Grok installed; Plan blocked before fixture-user cwd check |
| E6 | `yes` | `FAIL` | same |
| E7 | `yes` | `FAIL` | destination `--to grok` not independently recorded (Plan/busy-check on Claude source) |
| E8 | `yes` | `FAIL` | Grok native resume refusal not independently recorded |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree at TEST_COMMIT |
| `make verify` is green on both mandatory platforms | `FAIL` | Windows `make verify` exit 2 |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 dest-ack missing; Plan blocked |
| Every fidelity class has real report evidence and byte-stable goldens | `FAIL` | B6 not collected |
| Injection, secret, and bounded-read security gates leak nothing | `FAIL` | D1–D4 Plan blocked; fixture-scan PASS |
| 200-turn projection is bounded and reported | `PASS` | G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `FAIL` | C5 not reached |
| Dry-run and executed structured-plan output are byte-identical | `FAIL` | B1 not collected |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | observed 0, 1, 2, 3, 7; F8 no spawn |

## 8. Findings and repository hygiene

### Release-blocking

1. Installed `rein handoff --dry-run/--no-launch` returns runtime exit `1` after ~305 s: `handoff: session busy check: exit status 1`. Windows `processcheck.listProcesses` runs `Get-CimInstance Win32_Process` via powershell. That listing failed under host load, so Plan never ran. Blocks C4 leaf-match, C5 remap, A3 empty dest, B/D/E Plan rows, G2, G3 setup, R2–R5, and dest-ack. `--allow-active` does not skip a busy-check **error**.
2. `make verify` FAIL: hanging `--version` grandchild pipes not unblocked within the unit budget (~63.6 s vs expected in-time kill). RC3/RC5 R4 process-tree not cleared in `go test`.
3. Dest-ack A1/A2/A5/A6 not collected (no Windows Terminal first-reply; Plan also blocked). Missing required evidence is FAIL.
4. RC6 C4 leaf-match remap not evidenced: wrong-repo cwd never reached exit `5` / different-repository.

### Non-blocking

- WinGet `Links\goreleaser.exe` is a 0-byte symlink that cannot execute; the package-directory binary works. Harness PATH, not product.
- Isolation variables must not be exported into `make verify` / `go test` (Grok busy unit test 304 s). Harness. Superseded by clean-env rerun.
- `cmd /c "exe" args` quoting dropped arguments (8 ms fake exits). Harness. Superseded by `Start-Process` and direct `cmd.exe /c`.

### Test-harness deviations and supersessions

- First `make verify` exported vendor isolation into unit tests; Grok busy test FAIL superseded by clean-env verify that retained only the two grandchild-pipe FAILs.
- First snapshot used the Links goreleaser stub; superseded by package-directory PATH, all four PowerShell gates exit 0.
- Fake `cmd /c` matrix (all exits 0 in 8 ms) discarded. Real Start-Process matrix produced the 305 s busy-check failures and F8/F3 PASS.
- OpenCode skipped (`NOT TESTED: no isolation override`).
- Autonomous SSH without a usable Windows Terminal PTY; dest-ack recorded FAIL rather than skipped.

R1–R6 and RC2/RC3/RC5/RC6:

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `PASS` | throwaway Claude home had `projects/` and no `version` file; doctor `agent.claude: SUPPORTED (layout-projects-jsonl-v1)` |
| R2 | `FAIL` | absolute-path fixture Plan blocked |
| R3 | `FAIL` | dirty-tree `changed_files` Plan blocked |
| R4 | `FAIL` | unit grandchild hang 63.6 s; installed hanging `--version` Plan not collected (busy-check first) |
| R5 | `FAIL` | slash-commands fixture Plan blocked |
| R6 | `PASS` | pinned `2.1.229` in range for this run; doctor Claude SUPPORTED on layout; global `2.1.232` was not on the product PATH |
| RC2 C4 | `FAIL` | see C4 |
| RC2 F8 | `PASS` | see F8 timing |
| RC2 E5/E6 | `FAIL` | Plan blocked |
| RC2 R4 classification | `FAIL` | installed hang row not collected |
| RC2 D9 | `FAIL` | Plan blocked |
| RC2 list isolation | `PASS` | Claude `sessions --json` had no operator-home sessions |
| RC2 B3 | `FAIL` | Plan blocked |
| RC2 B6 | `FAIL` | Plan blocked |
| RC2 C6/C7 | `FAIL` | Plan blocked |
| RC3 flagship A empty dest | `FAIL` | exit 1 busy-check, not a completed Plan |
| RC3 R4 hang | `FAIL` | unit 63.6 s; installed not collected |
| RC3 Windows make verify | `FAIL` | see gates |
| RC3 G3 list | `FAIL` | 100 `--no-launch` not completed |
| RC3 B6 summarized | `FAIL` | Plan blocked |
| RC5 R4 hang | `FAIL` | see R4 |
| RC5 F8 | `PASS` | refuse-before-Plan ~1022 ms |
| RC5 B6 | `FAIL` | Plan blocked |
| RC5 C5 | `FAIL` | see C5 |
| RC5 E5/E6 | `FAIL` | Plan blocked |
| RC6 C4 | `FAIL` | leaf-match not reached |
| RC6 F8 | `PASS` | refuse-before-index; `--allow-warning` still exit 7 |
| Dest-ack A1/A2/A5/A6 | `FAIL` | see Matrix A |

## 9. Required terminated device block

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.6
test_commit=00366cb53f4246bde50c7f6fe96f7c1e1660ec88
installed_binary_sha256=8a42ccbecf91ee7f9d969de4169c1ec1b12fcf4c8bb7e8ac7e1bed72474ff5bd
required_pass=3
required_partial=0
required_fail=41
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
release_blocking_findings=4
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
