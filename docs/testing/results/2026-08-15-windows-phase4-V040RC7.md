# Phase 4 structured-handoff report — Windows amd64 / v0.4.0-rc.7

Copy of the Phase 4 template with this dispatch's substitution table applied.
No rc.1–rc.6 corpus, home, capsule, lineage, or report row was reused.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `34 PASS / 0 PARTIAL / 10 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `4 PASS / 2 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `4`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. Optional E rows may be `NOT TESTED` only when the vendor is genuinely
absent and was not installed solely for acceptance.

`stable_v0.4.0_authorized=false`. `current_stable=v0.3.0`. This report does not
authorize stable `v0.4.0`.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-14T22:49:21Z` start; matrix complete `2026-08-14T23:23:53Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| CPU architecture/native process | AMD64; 64-bit Windows PowerShell 5.1.26100.8328 Desktop; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.7` |
| Tested full commit | `a82a0ebe79a75f433bcfd28a26d6fd23976f3f71` |
| Installed binary SHA-256 | `c4049f511d5a9b3edc8ea3cff990d4cbf741dd74435771e285ae3c9a06a53eea` |
| Installed version JSON | name `reinstate`; version `0.4.0-rc.7`; commit `a82a0ebe79a75f433bcfd28a26d6fd23976f3f71`; date `2026-08-14T22:21:10Z` |
| Claude Code version/state | before first product command: host global `2.1.232` unused; test PATH pin `2.1.229` source+destination; after last row pin still `2.1.229` |
| Codex CLI version/state | `0.147.0` source+destination; unchanged after last row |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` (present); OpenCode `1.18.2` (no isolation override; omitted from process PATH); Grok `0.2.101`; pin/Codex/Grok unchanged after last row |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.12 windows/amd64` via `GOTOOLCHAIN=go1.25.12` (host default go1.26.1 unused) |
| Report branch | `test/v0.4.0-rc.7-windows-amd64-report` |
| Device-report commit | filled after commit |
| Draft report PR | filled after `gh pr create --draft` |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | `git verify-tag` Good "git" signature for the allowed SSH ED25519 key |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | peeled `a82a0ebe79a75f433bcfd28a26d6fd23976f3f71`; ancestor of `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | draft=false; prerelease=true; 25 assets |
| Exact expected release asset set is present | `PASS` | checksums + 5 archives + 5 SBOMs + 5 raw binaries + 8 Linux packages + source |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | windows zip `81d256ea1c2ae60002713b9974f4c0fee18e42467e449ef6ad10147909baed25` equals published checksums.txt and API digest; exe `c4049f511d5a9b3edc8ea3cff990d4cbf741dd74435771e285ae3c9a06a53eea`; `gh attestation verify` exit 0 |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | zip 6 relative entries; 0 unsafe; exe present |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.7` | `PASS` | live `install.ps1` sha256 `014fbf7e77f23628b8d83514429c5dd5dbed3bc7515b25f8a34b69bb1e49e32d`; pins only `v0.4.0-rc.7` |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | tagged `scripts/install.ps1` `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; User PATH unchanged; pre-existing user binary hash unchanged |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.7` | `PASS` | identical sha256 `c4049f51…`; version JSON `0.4.0-rc.7` |
| Installed binary reports the literal full tested commit | `PASS` | `version --json` commit `a82a0ebe79a75f433bcfd28a26d6fd23976f3f71` |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | detached TEST_COMMIT; tidy -diff exit 0 (91 ms) |
| `make verify` with Go 1.25.12 | `FAIL` | GnuWin32 make 3.81 hung: `GOTOOLCHAIN` is not a cmd builtin. Superseded by direct `go test` + PowerShell gates. Not scored as a product defect |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS` | exit 0; 52212 ms |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | real goreleaser package path (WinGet Links stub is 0-byte). `snapshot.ps1` 0; `stage-release-assets.ps1` 0; `check-release-artifacts.ps1` 0; `test-install.ps1` 0 |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | `go test ./... -count=1` 34567 ms exit 0; `internal/handoff` 6.077 s; r4 grandchild unit 3680 ms exit 0 (not ~60 s) |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | handoff unit + B1 unit exit 0 |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | fixture-scan exit 0; D1/D2/D4 installed dry-run Plan |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | G1 unit exit 0 |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new `reinstate-v040rc7-iso-20260815` tree; no rc.1–rc.6 home reused |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | tagged fixtures copied into throwaway homes; live dest-ack not collected |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | five isolation variables set before first product command |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | no `rein init` / push / pull |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | throwaway Claude/Codex/Gemini/Grok homes only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | D10 source hash unchanged; dest-ack launch did not succeed |

## 5. Required matrix — 44 rows

Product JSON `code`/`message` is authoritative when Start-Process `ExitCode` stayed 0.

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | Claude→Codex dest-ack not collected. `ssh -t` did not allocate a TTY from this agent; throwaway Claude print-mode probe failed (not logged in). Missing required dest-ack |
| A2 | `FAIL` | Codex→Claude dest-ack not collected. Dry-run Plan from Codex fixture exit 0 in 2283 ms, mode `structured handoff`, `destination_session_mode=new`. Dest acknowledgement missing |
| A3 | `FAIL` | Logged-out dest-ack not collected |
| A4 | `PASS` | Tagged `partial-final-record`; installed `--last --from claude --to codex --dry-run --json` 2152 ms; mode `structured handoff`; `destination_session_mode=new` |
| A5 | `FAIL` | No destination first-reply restatement collected |
| A6 | `FAIL` | MARKER non-repeat not observed; dest not launched |
| A7 | `FAIL` | Lineage list works (G3 n=100) but destination IDs unresolved without launch |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | Unit `TestHandoffExecutedOutputMatchesDryRunByteForByte` exit 0; installed dry-run 2142 ms and `--no-launch` 2131 ms both Plan `structured handoff` |
| B2 | `PASS` | Long-history fixture dry-run 2184 ms exit 0 |
| B3 | `PASS` | `--policy checkpoint --no-launch --json` exit 0; `projection_events` present |
| B4 | `PASS` | `--policy balanced` 2174 ms Plan |
| B5 | `PASS` | `--policy full` 2163 ms Plan |
| B6 | `PASS` | Compaction fixture; `fidelity.json` includes `summarized` |
| B7 | `PASS` | Attachments fixture 2165 ms Plan |
| B8 | `PASS` | Unknown-records fixture 2154 ms Plan |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | Dirty demo tree; dry-run Plan exit 0 |
| C2 | `PASS` | Workspace-wins covered by unit tests plus dirty-tree dry-run |
| C3 | `FAIL` | Windows os-roots from leaf-matching `demo` remapped to Plan (exit 0) instead of a missing-workspace block. Row not isolated from C5 leaf-match |
| C4 | `PASS` | Other-repo cwd: JSON `compatibility` / different repository (1373 ms). Same-repo subdir Plan 2140 ms. Non-git cwd Plan 2031 ms |
| C5 | `PASS` | macOS os-roots on Windows from leaf-matching `demo`; Plan 2179 ms; no 305 s busy-check skip |
| C6 | `PASS` | Dummy source MCP produced capability warning path; dry-run Plan |
| C7 | `PASS` | Same collection as C6 |
| C8 | `PASS` | PATH `claude --version` `2.1.230`; JSON `compatibility` source UNTESTED (1226 ms); no `--allow-untested` |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | Prompt-injection fixture dry-run Plan |
| D2 | `PASS` | Fence-breakout fixture dry-run Plan |
| D3 | `PASS` | Adversarial unit + dry-run; no source-authority dump in this report |
| D4 | `PASS` | Secret-leakage fixture `--show-redactions` Plan |
| D5 | `PASS` | Launch-free rows used throwaway homes only |
| D6 | `PASS` | `handoffs/` under isolated `REINSTATE_HOME` |
| D7 | `PASS` | OpenCode skipped; other vendor trees were throwaway homes |
| D8 | `PASS` | No backend; push/pull not used |
| D9 | `PASS` | Grok `--no-redact` refused (`usage`: redaction forced). Grok dry-run Plan 1806 ms |
| D10 | `PASS` | Source fixture hash unchanged across dry-run |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | Human/JSON surfaces labeled `structured handoff`; no banned cross-agent wording in collected Plan output |
| F2 | `FAIL` | `resume --last --with` completed in 36 ms; not proven byte-parity with `handoff` |
| F3 | `FAIL` | `--json` without `--dry-run`/`--no-launch` recorded 41 ms ExitCode 0; usage exit 2 not proven |
| F4 | `PASS` | Unknown warning ID and `*` both JSON `usage` (~2148–2173 ms) |
| F5 | `PASS` | `--no-launch` 2129 ms Plan; no dest spawn |
| F6 | `PASS` | `handoff list --json` exit 0 |
| F7 | `FAIL` | Bare `session-syn-001` was `session not found`, not exit 6 with two qualified options |
| F8 | `PASS` | Non-TTY launch refused in 37 ms: interactive terminal required; `--allow-warning` still refused. Not Plan-scale |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run |  |  |  | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | unit exit 0 |
| G2 | 1 warmup + 5 installed `handoff --dry-run` | 2176 ms | 2274 ms | 2278 ms | Windows max `12s` | `PASS` | long-history; mode `structured handoff`; busy_check=false |
| G3 | 1 warmup + 20 `handoff list --limit 100 --json` |  | 42 ms |  | Windows p95 `4s` | `PASS` | 100 `--no-launch` created; list n=100 |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` | `PASS` | Gemini fixture `--last --from gemini --to claude --dry-run --json` 1837 ms Plan |
| E2 | `yes` | `PASS` | Same toward Codex 1820 ms Plan |
| E3 | `yes` | `NOT TESTED` | OpenCode has no isolation override |
| E4 | `yes` | `NOT TESTED` | OpenCode has no isolation override |
| E5 | `yes` | `PASS` | Grok 0.2.101 + tagged basic fixture from leaf-matching `demo`; `--to claude` 1809 ms Plan; not blocked on `agent.layout` |
| E6 | `yes` | `PASS` | Same toward Codex 1812 ms Plan |
| E7 | `yes` | `FAIL` | `--to grok` usage refusal not cleanly recorded |
| E8 | `yes` | `FAIL` | `resume --last --from grok` rejected as unknown flag; native refuse not collected |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree |
| `make verify` is green on both mandatory platforms | `FAIL` | GnuWin32 make hung; direct go test/race + PS gates PASS. macOS owned separately |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 dest-ack missing; dry-run Plans PASS |
| Every fidelity class has real report evidence and byte-stable goldens | `PASS` | B6 + automated gates |
| Injection, secret, and bounded-read security gates leak nothing | `PASS` | D1–D10 |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS` | C5 |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 unit |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `FAIL` | F3/F7 not proven; D7–D8 PASS |

## R1–R6 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `PASS` | Throwaway Claude home had `projects/` and no `version` file; in-range dry-run 2151 ms Plan, not UNTESTED |
| R2 | `PASS` | Absolute-paths fixture `--no-launch` Plan; no `absolute filesystem path is not allowed` abort |
| R3 | `PASS` | Dirty-tree dry-run 2160 ms Plan |
| R4 | `PASS` | Hanging `--version` shim JSON `compatibility` UNTESTED in 11496 ms (inside 25 s); not Runtime 1; unit grandchild 3680 ms |
| R5 | `PASS` | Slash-commands fixture Plan; no absolute-path abort |
| R6 | `PASS` | Pin `2.1.229` in range; determined `2.1.230` JSON UNTESTED without `--allow-untested` |

## RC2 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C4 | `PASS` | See C4 |
| F8 | `PASS` | See F8 37 ms |
| E5/E6 | `PASS` | See E5/E6 |
| R4 classification | `PASS` | See R4 |
| D9 | `PASS` | See D9 |
| List isolation | `PASS` | Isolated `rein list` / `sessions` had no operator-home sessions |
| B3 | `PASS` | See B3 |
| B6 | `PASS` | See B6 |
| C6/C7 | `PASS` | See C6/C7 |

## RC3 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| Flagship A empty dest | `FAIL` | Dest Plan is supported (C5/A4 dry-run); dest-ack still missing |
| R4 hang | `PASS` | See R4 |
| Windows `make verify` | `FAIL` | GnuWin32 `GOTOOLCHAIN=` hang; superseded by direct go test/race |
| G3 `--no-launch` list | `PASS` | created=100; p95 42 ms |
| B6 summarized | `PASS` | See B6 |

## RC5 / RC6 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R4 hang process-tree | `PASS` | See R4 |
| F8 refuse-before-Plan | `PASS` | 37 ms; `--allow-warning` still refused |
| B6 summarized | `PASS` | See B6 |
| C5 foreign-OS remap | `PASS` | See C5 |
| E5/E6 fixture-user cwd | `PASS` | See E5/E6 |
| C4 leaf-match remap | `PASS` | See C4 |
| Dest-ack | `FAIL` | See A1/A2/A5/A6 |

## RC7 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| Busy-check listing error | `PASS` | Dry-run Plan 2179 ms; no `session busy check` skip; not ~305 s exit 1 |
| C8/R6 2.1.230 fail-closed | `PASS` | Determined `2.1.230` JSON UNTESTED / compatibility; no `--allow-untested` |
| R4 hung `--version` Compatibility | `PASS` | UNTESTED in 11496 ms; unit not ~60 s |
| E5/E6 Grok source-only layout | `PASS` | Leaf-matching `demo` Plan; not blocked on `agent.layout` |
| Dest-ack | `FAIL` | Required A1/A2/A5/A6 dest-ack missing |

## 8. Findings and repository hygiene

### Release-blocking

1. Dest-ack A1/A2/A5/A6/A7 missing. Autonomous `ssh -t` did not allocate a TTY; throwaway Claude/Codex were not logged in. Missing required evidence is FAIL.
2. C3 missing-workspace block was not isolated from leaf-match remap onto local `demo`.
3. F3 `--json` without `--dry-run`/`--no-launch` did not prove usage exit 2.
4. F7 ambiguous native ID was `session not found`, not exit 6 with two qualified options.

### Non-blocking

- GnuWin32 make 3.81 cannot apply Unix `GOTOOLCHAIN=` prefixes and hung. Direct `go test` / race / PowerShell snapshot gates passed. Known go1.25.12 stdlib govulncheck items are not a product FAIL for this candidate.
- `internal/doctest` `TestProductionDeploymentRejectsInvalidWebsiteTagDate` appeared in one `go test` transcript; overall `go test ./...` and race were recorded exit 0.
- E7/E8 optional rows failed on harness flags (`--last` on `resume`), not a Grok destination (destinations remain Claude/Codex only).
- Start-Process `ExitCode` often stayed 0 while JSON `code` on stderr carried compatibility/usage. Rows were scored from that JSON.

### Test-harness deviations and supersessions

- First matrix pass dropped `Start-Process` arguments (picker-only ~37 ms). Discarded. Superseded by ArgumentList string-join + renamed `-HandoffArgs` (not `$Args`) rerun at 23:18–23:23Z with 2.1–2.3 s Plans.
- `make verify` via GnuWin32 superseded by `go test ./...`, `-race`, and `scripts/*.ps1`.
- Checksums.txt first download via unauthenticated `gh` 401; superseded by published checksums.txt matching zip/exe digests.
- OpenCode: `NOT TESTED: no isolation override`.
- Dest-ack attempted via `ssh -t`; host printed `Pseudo-terminal will not be allocated because stdin is not a terminal`.

Report-only diff vs `TEST_COMMIT`: this file. Merge-base is `a82a0ebe79a75f433bcfd28a26d6fd23976f3f71`. Privacy scan: no secrets, transcripts, or capsule bodies.

## 9. Required terminated device block

Required counts must sum to 44 and optional counts to 8. This block occurs
exactly once and remains the last report content until reconciliation.

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.7
test_commit=a82a0ebe79a75f433bcfd28a26d6fd23976f3f71
installed_binary_sha256=c4049f511d5a9b3edc8ea3cff990d4cbf741dd74435771e285ae3c9a06a53eea
required_pass=34
required_partial=0
required_fail=10
required_not_tested=0
optional_pass=4
optional_fail=2
optional_not_tested=2
artifact_chain=PASS
isolation_privacy=PASS
flagship_directions=FAIL
fidelity_policy=PASS
workspace_capability=FAIL
security=PASS
cli_contract=FAIL
performance=PASS
phase1_phase2_phase3_regression=PASS
release_blocking_findings=4
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
