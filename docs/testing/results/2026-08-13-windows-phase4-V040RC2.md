# Phase 4 structured-handoff report — v0.4.0-rc.2 Windows amd64

Copy of the Phase 4 template with dispatch substitutions for `v0.4.0-rc.2`.
Private evidence stayed on the Windows host; this file contains no transcripts,
capsule bodies, credentials, or operator vendor paths.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `11 PASS / 0 PARTIAL / 33 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `0 PASS / 6 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `8`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. OpenCode rows are `NOT TESTED` because there is no isolation override.
Gemini and Grok were present and were not given complete source-only rows.

Current stable remains `v0.3.0`. This report does not authorize `v0.4.0`.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-13T03:08:26Z` (install) / `2026-08-13T03:29:31Z` (after versions) |
| Device | `windows-amd64` |
| OS/version/build | `Windows 11 Pro 10.0.26200` native 64-bit Windows PowerShell `5.1.26100.8328` |
| CPU architecture/native process | `amd64` / 64-bit process `True` |
| Filesystem | `NTFS` |
| Tested tag | `v0.4.0-rc.2` |
| Tested full commit | `d609ea5ba6527b22034f4e09a81fe7c47c3fb589` |
| Installed binary SHA-256 | `1cdca628757bdc3b777806aa613eb614bef62597f2208f0c075695342937ac7d` |
| Installed version JSON | `0.4.0-rc.2` / `d609ea5ba6527b22034f4e09a81fe7c47c3fb589` / `2026-08-13T00:01:18Z` |
| Claude Code version/state | before `2.1.229`; after `2.1.229`; doctor `SUPPORTED (2.1.229)`; `DISABLE_AUTOUPDATER=1` |
| Codex CLI version/state | before `0.147.0`; after `0.147.0`; doctor `SUPPORTED (0.147.0)` |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` present; OpenCode `1.18.2` present on host, removed from test PATH (`NOT TESTED: no isolation override`); Grok `0.2.101` present |
| Git version | `git 2.52.0.windows.1` |
| Go version/toolchain | host default `go1.26.1`; `GOTOOLCHAIN=go1.25.12` → `go1.25.12 windows/amd64` |
| Report branch | `test/v0.4.0-rc.2-windows-amd64-report` |
| Device-report commit | `56baa02f0aa6fbdf3a44b8ada99bb65fe504dc94` |
| Draft report PR | `PENDING` |

Antivirus: Microsoft Defender enabled, real-time protection enabled (G1–G3).

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | `git verify-tag v0.4.0-rc.2` exit 0, Good signature |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | peel `d609ea5ba6527b22034f4e09a81fe7c47c3fb589`; ancestor of `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | GitHub API: draft false, prerelease true, tag `v0.4.0-rc.2` |
| Exact expected release asset set is present | `PASS` | 25 assets |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | live bootstrap pinned installer SHA-256 `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe`; published zip identity script exit 0 |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | `snapshot.ps1`, `stage-release-assets.ps1`, `check-release-artifacts.ps1` exit 0 |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.2` | `PASS` | live SHA-256 `0c2acc6e0b201eaaa71e02a6dfe222d8887efbc1a4ebcfaf19bacaf6dae35a35` matches tagged `website/public/install.ps1`; pin `v0.4.0-rc.2`; no `v0.4.0-rc.1` pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | bootstrap contains tagged `scripts/install.ps1` digest |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `INSTALL_DIR` under throwaway root; `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; user PATH unchanged |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.2` | `PASS` | identical SHA-256; version `0.4.0-rc.2` |
| Installed binary reports the literal full tested commit | `PASS` | `d609ea5ba6527b22034f4e09a81fe7c47c3fb589` |

Host note: PowerShell `Invoke-WebRequest` could not resolve `reinstate.dev`; `curl.exe` fetched the live bootstrap. Not a product pin failure.

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `FAIL` | `make verify` via Git bash returned exit 0 in 526 ms with empty log (no-op); not accepted |
| `make verify` with Go 1.25.12 | `FAIL` | same; GnuWin32 make 3.81 present but unused |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | all packages PASS except `internal/doctest` `TestProductionDeploymentRejectsInvalidWebsiteTagDate` (Windows `sh` deploy-website script message mismatch). No DATA RACE lines |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | `snapshot.ps1` 52 s including Syft SBOMs; stage/artifacts/test-install/published-zip-identity exit 0 |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | `go test` packages including `internal/handoff`, `internal/workspace`, `internal/executabletrust` PASS on `go1.25.12` |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | `TestHandoffExecutedOutputMatchesDryRunByteForByte` and handoff package tests PASS |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | `go test ./internal/handoff` and `./internal/fixture` exit 0 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | `12.0914ms`; projection `46541` bytes |

WinGet `Links` shims for goreleaser/syft are untrusted mount points; package-directory binaries were used.

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new throwaway root; five vendor env vars set before product commands |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `FAIL` | flagship live Claude/Codex CLI sessions were not created; Windows os-roots tagged fixture used for dry-run |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `FAIL` | first `rein list` (legacy adapter Discover) indexed operator Claude JSONL paths. Home destroyed and restarted. Subsequent `rein sessions` (sessionindex) honored `CLAUDE_CONFIG_DIR`. OpenCode stripped from PATH |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | doctor config missing; push exit 3 |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | after restart |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `FAIL` | no vendor CLI session writes; destination not launched |

OpenCode isolation disposition: **NOT TESTED / skipped**. No override exists. Host binary was removed from the test `PATH` so `opencode session list` was not invoked. Rows E3–E4 are `NOT TESTED: no isolation override`. Never reported as isolated.

## 5. Required matrix — 44 rows

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | no Windows Terminal TTY Claude → Codex destination launch; SSH stdin redirected |
| A2 | `FAIL` | no Codex → Claude TTY destination launch |
| A3 | `FAIL` | logged-out source repeat not run |
| A4 | `FAIL` | partial-final-record fixture not completed against a live Windows workspace |
| A5 | `FAIL` | no destination-agent first reply (new destination session continuing the same task) |
| A6 | `FAIL` | no destination launch to prove marker non-repetition |
| A7 | `FAIL` | dry-run produced a handoff id; durable `--no-launch` required warning acks (exit 7) and was not stored in the working follow-up |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | `TestHandoffExecutedOutputMatchesDryRunByteForByte` exit 0 |
| B2 | `PASS` | tagged 200-turn unit: 400 events, 46541 bytes, 12.09 ms |
| B3 | `FAIL` | checkpoint policy not collected on the working Windows fixture |
| B4 | `PASS` | installed `--policy balanced --dry-run --json` exit 0; `destination_session_mode=new`; mode `structured handoff` |
| B5 | `FAIL` | full policy not collected on the working Windows fixture |
| B6 | `FAIL` | inspect of a durable store not completed; dry-run fidelity overall `exact` with two exact message components only |
| B7 | `FAIL` | attachments fixture not completed on a live Windows workspace |
| B8 | `FAIL` | unknown-records fixture not completed on a live Windows workspace |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | Windows os-roots fixture + disposable git repo: dry-run `dirty: true`, branch/head from live Git, changed_files three `${REPO:demo}/…` tokens matching staged/unstaged/untracked |
| C2 | `FAIL` | transcript vs Git contradiction not constructed |
| C3 | `FAIL` | missing-workspace row not separately recorded (macOS-path fixtures did block with preflight exit 5) |
| C4 | `FAIL` | wrong-project refusal not collected |
| C5 | `FAIL` | Windows fixture remapped through `${REPO:demo}`; operator-home absolute-path scan of failed long-history runs is not sufficient C5 evidence |
| C6 | `FAIL` | missing MCP not collected |
| C7 | `FAIL` | missing skill/instruction not collected |
| C8 | `PASS` | throwaway `claude.cmd` echoing `2.1.230` → handoff exit 5 |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `FAIL` | adversarial prompt-injection fixture not completed on a live Windows workspace |
| D2 | `FAIL` | fence-breakout fixture not completed on a live Windows workspace |
| D3 | `FAIL` | source-instruction exclusion not inspected in a stored projection |
| D4 | `FAIL` | secret-leakage fixture not completed on a live Windows workspace |
| D5 | `PASS` | launch-free dry-run used isolated homes; doctor keyring unavailable in this SSH logon |
| D6 | `FAIL` | durable `handoffs/` DACL not confirmed after `--no-launch` exit 7 |
| D7 | `PASS` | OpenCode skipped; launch-free rows did not spawn vendors |
| D8 | `PASS` | `rein push --json` exit 3 with no backend; no capsule sync |
| D9 | `FAIL` | Grok `--no-redact` exit 2; Grok `--dry-run` exit 1 (`unsupported agent "grok"` busy check). Warning-on-success not shown |
| D10 | `PASS` | tagged fixtures used read-only for launch-free rows |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | JSON `mode=structured handoff`, `destination_session_mode=new`; bootstrap text `structured handoff, not native resume` |
| F2 | `FAIL` | `resume --with` not collected on the working fixture |
| F3 | `FAIL` | earlier exit 2 was `session not found`, not a valid-session `--json` gate |
| F4 | `FAIL` | unacknowledged warnings on `--no-launch` exit 7 (`baseline.unavailable`, `git.working_tree`, `handoff.capability.attachment.support`); unknown/duplicate/wildcard/stale IDs not all collected on a valid session |
| F5 | `FAIL` | `--no-launch` with required warning acks not completed (exit 7 without acks) |
| F6 | `FAIL` | inspect/export of a durable store not completed |
| F7 | `FAIL` | ambiguous native ID not constructed |
| F8 | `FAIL` | non-TTY fail-closed not proven on a valid session (SSH stdin redirected; missing required TTY/non-TTY pair) |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run | 12.09 ms | 12.09 ms | 12.09 ms | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | 400 events; 46541 bytes; Defender on |
| G2 | incomplete |  |  |  | Windows max `12s` | `FAIL` | one working balanced dry-run observed under the ceiling; required warmup+5 timed samples not retained |
| G3 | incomplete |  |  |  | Windows p95 `4s` | `FAIL` | 100 valid `--no-launch` lineage rows not created (warning ack / env-inherit failures) |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` | `FAIL` | Gemini 0.53.0 present; no isolated Gemini source handoff |
| E2 | `yes` | `FAIL` | same |
| E3 | `yes` | `NOT TESTED` | OpenCode: no isolation override |
| E4 | `yes` | `NOT TESTED` | OpenCode: no isolation override |
| E5 | `yes` | `FAIL` | Grok present; dry-run exit 1 busy-check |
| E6 | `yes` | `FAIL` | same |
| E7 | `yes` | `FAIL` | source-only destination refusal not collected |
| E8 | `yes` | `FAIL` | Grok native continuation refusal not collected |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree at `TEST_COMMIT` |
| `make verify` is green on both mandatory platforms | `FAIL` | Windows `make verify` not actually executed; doctest fail |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 |
| Every fidelity class has real report evidence and byte-stable goldens | `FAIL` | B6 incomplete; goldens PASS in unit tests |
| Injection, secret, and bounded-read security gates leak nothing | `FAIL` | D1–D4 incomplete; unit adversarial PASS |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `FAIL` | C5 incomplete; Windows dry-run used `${REPO:demo}` |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 unit |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | observed 0, 2, 3, 5, 7; no 4/other |

## 8. Findings and repository hygiene

### Release-blocking

- Missing required Windows Terminal TTY evidence for A1–A2 (and A5–A6 destination restatement). Dispatch: missing required TTY evidence is FAIL.
- Required matrix not 44/44 PASS (11/44).
- `make verify` did not run; `internal/doctest` `TestProductionDeploymentRejectsInvalidWebsiteTagDate` fails on this Windows host (`sh` + deploy script output mismatch).
- G2/G3 required sampling plan not completed.
- `rein list` (legacy adapter) ignores `CLAUDE_CONFIG_DIR` and can index the operator Claude tree. Handoff/`rein sessions` honor the override. First-index contract failed until `list` was abandoned and the home destroyed.
- Live Claude/Codex destination launches were not performed; flagship quota-switch is unproven on this device.
- Optional Gemini/Grok source rows were not completed despite vendors being installed.
- R2, R4, R5 not PASS (R4 timeout-under-load not run).

### Non-blocking

- PowerShell 5.1 `Invoke-WebRequest` DNS to `reinstate.dev` failed; `curl.exe` succeeded.
- WinGet `Links` shims for goreleaser/syft cannot execute (untrusted mount point); package-dir binaries work.
- SSH session: stdout redirected, stdin redirected; Defender real-time on.
- Keyring unavailable in this SSH logon session.

### Test-harness deviations and supersessions

- Isolation restart after `rein list` operator-tree access. `rein sessions` used thereafter.
- OpenCode removed from test PATH rather than uninstalled.
- MacOS-path tagged fixtures preflight-block on Windows (missing workspace). Windows os-roots fixture plus a disposable repo at the fixture cwd unblocked dry-run.
- Report-only diff: this file. Merge-base `d609ea5ba6527b22034f4e09a81fe7c47c3fb589`.

## RC1 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `PASS` | throwaway Claude home has no `version` file; doctor `SUPPORTED (2.1.229)`; Windows-fixture dry-run exit 0 did not report source `UNTESTED` |
| R2 | `FAIL` | absolute-path fixture not completed against a live Windows workspace |
| R3 | `PASS` | dirty tree: capsule/plan `changed_files` listed three `${REPO:demo}/…` tokens; not `(none)` |
| R4 | `FAIL` | timed-out out-of-range probe under load not collected |
| R5 | `FAIL` | slash-prefixed prose fixture not completed on a live Windows workspace |
| R6 | `PASS` | host Claude `2.1.229` in range and `SUPPORTED`; fake `2.1.230` fail-closed exit 5. Versions unchanged after last row |

## 9. Required terminated device block

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.2
test_commit=d609ea5ba6527b22034f4e09a81fe7c47c3fb589
installed_binary_sha256=1cdca628757bdc3b777806aa613eb614bef62597f2208f0c075695342937ac7d
required_pass=11
required_partial=0
required_fail=33
required_not_tested=0
optional_pass=0
optional_fail=6
optional_not_tested=2
artifact_chain=PASS
isolation_privacy=FAIL
flagship_directions=FAIL
fidelity_policy=FAIL
workspace_capability=FAIL
security=FAIL
cli_contract=FAIL
performance=FAIL
phase1_phase2_phase3_regression=FAIL
release_blocking_findings=8
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
