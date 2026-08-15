# Phase 4 structured-handoff report — Windows amd64 / v0.4.0-rc.8

Copy of the Phase 4 template with this dispatch's substitution table applied.
No rc.1–rc.7 corpus, home, capsule, lineage, or report row was reused.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `38 PASS / 0 PARTIAL / 6 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `6 PASS / 0 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `2`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. Optional E rows may be `NOT TESTED` only when the vendor is genuinely
absent and was not installed solely for acceptance.

`stable_v0.4.0_authorized=false`. `current_stable=v0.3.0`. This report does not
authorize stable `v0.4.0`.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-15T00:28:08Z` start; matrix complete `2026-08-15T00:49:41Z`; dest-ack `2026-08-15T00:55:19Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| CPU architecture/native process | AMD64; 64-bit Windows PowerShell 5.1.26100.8328 Desktop; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.8` |
| Tested full commit | `083bc0817c434ed923b0c2e39a72f8f6deca11f5` |
| Installed binary SHA-256 | `e6cd58f7cfbcc8573752970430222d26fe5a7079e3ab141b378f4e8b10b8d313` |
| Installed version JSON | name `reinstate`; version `0.4.0-rc.8`; commit `083bc0817c434ed923b0c2e39a72f8f6deca11f5`; date `2026-08-15T00:15:41Z` |
| Claude Code version/state | before first product command: host global `2.1.232` unused; test PATH pin `2.1.229` source+destination; after last row pin still `2.1.229` |
| Codex CLI version/state | `0.147.0` source+destination; unchanged after last row |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` (present); OpenCode `1.18.2` (no isolation override; omitted from process PATH); Grok `0.2.101`; pin/Codex/Grok unchanged after last row |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.13 windows/amd64` via `GOTOOLCHAIN=go1.25.13` (host default go1.26.1 unused) |
| Report branch | `test/v0.4.0-rc.8-windows-amd64-report` |
| Device-report commit | `7793c1c4f02375ca0c47657f8666cc3dcc6c52f3` |
| Draft report PR | `NOT CREATED` |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | `git verify-tag` Good "git" signature for the allowed SSH ED25519 key |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | peeled `083bc0817c434ed923b0c2e39a72f8f6deca11f5`; ancestor of `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | draft=false; prerelease=true; 25 assets |
| Exact expected release asset set is present | `PASS` | checksums + 5 archives + 5 SBOMs + 5 raw binaries + 8 Linux packages + source |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | windows zip sha256 `0a1d36a4…` equals API digest; windows exe sha256 `e6cd58f7…` equals checksums.txt and the isolated install; `gh attestation verify` exit 0 |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | zip 6 relative entries; 0 unsafe; exe present |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.8` | `PASS` | live `install.ps1` sha256 `251923c95ca9e63c240e6d4a349db67ef87ef60cd87a9a59717a1452e24405a8`; pins only `v0.4.0-rc.8` |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | tagged `scripts/install.ps1` `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; User PATH unchanged; pre-existing user binary hash unchanged |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.8` | `PASS` | identical sha256 `e6cd58f7…`; version JSON `0.4.0-rc.8` |
| Installed binary reports the literal full tested commit | `PASS` | `version --json` commit `083bc0817c434ed923b0c2e39a72f8f6deca11f5` |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | detached TEST_COMMIT; tidy -diff exit 0 (33303 ms) |
| `make verify` with Go 1.25.13 | `FAIL` | GnuWin32 make 3.81 cannot apply Unix `GOTOOLCHAIN=` prefixes. Superseded by direct `go test` + PowerShell gates. Not scored as a product defect |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | overall exit 1; sole failing package `internal/doctest` `TestProductionDeploymentRejectsInvalidWebsiteTagDate` (Windows `sh` host). Product packages including `internal/handoff` (8.345 s) passed. Not scored as a CLI product defect |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | real goreleaser package path (WinGet Links stub is 0-byte). `snapshot.ps1` 0 (33442 ms); `stage-release-assets.ps1` 0; `check-release-artifacts.ps1` 0; `test-install.ps1` 0. `govulncheck` exit 0 on go1.25.13 |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | product packages in `go test ./...` ok (`internal/cli`, `sessionindex`, `preflight`, `adapter`, `handoff`) |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | handoff unit + B1 unit exit 0 |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | fixture-scan exit 0; D1/D2/D4 installed dry-run Plan |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | G1 unit exit 0 in 1611 ms |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new rc.8 isolation root; five vendor overrides set before first product command |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | tagged fixtures + new git repos; no rc.1–rc.7 homes |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | PATH pin + throwaway homes only; OpenCode omitted from PATH |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | no `rein init`; push/pull unused |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | dummy MCP/skill only under throwaway Claude home |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | launch-free rows used fixtures; dest-ack launch did not produce a logged-in dest session |

## 5. Required matrix — 44 rows

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | Real TTY collected (`GetConsoleMode=true`, filetype 2). Throwaway dest Claude not logged in; print-mode probe failed immediately. Dest-ack missing |
| A2 | `FAIL` | Real TTY collected. Throwaway dest Codex not logged in; no Codex session. Dest-ack missing |
| A3 | `FAIL` | Logged-out dest-ack not collected |
| A4 | `PASS` | Tagged `partial-final-record`; `--last --from claude --to codex --dry-run --json` 1355 ms; mode `structured handoff` |
| A5 | `FAIL` | No destination first-reply restatement collected |
| A6 | `FAIL` | MARKER non-repeat not observed; dest not launched |
| A7 | `FAIL` | `handoff list` exit 0 after G3, but destination IDs unresolved without launch |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | Installed `--dry-run` and `--no-launch` JSON both exit 0; unit `TestHandoffExecutedOutputMatchesDryRunByteForByte` exit 0 |
| B2 | `PASS` | long-history balanced dry-run 1399 ms Plan |
| B3 | `PASS` | checkpoint `--no-launch --json` exit 0; `projection_events` present |
| B4 | `PASS` | unknown-records balanced dry-run 1369 ms |
| B5 | `PASS` | attachments full dry-run 1372 ms |
| B6 | `PASS` | compaction fixture `--no-launch`; `fidelity.json` includes `summarized` |
| B7 | `PASS` | subagents dry-run exit 0 |
| B8 | `PASS` | macOS os-roots dry-run exit 0 |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | Dirty demo tree; dry-run Plan exit 0 |
| C2 | `PASS` | Workspace-wins covered by unit tests plus dirty-tree dry-run |
| C3 | `PASS` | Same-OS missing workspace leaf `missing-ws-rc8` (not `demo`); JSON `compatibility` / environment preflight blocked; product exit 5 in 716 ms |
| C4 | `PASS` | Other-repo cwd: JSON `compatibility` / different repository (576 ms). Same-repo subdir Plan 1367 ms. Non-git cwd Plan 1387 ms |
| C5 | `PASS` | macOS os-roots on Windows from leaf-matching `demo`; Plan 1388 ms; no busy-check skip |
| C6 | `PASS` | Dummy source MCP produced `handoff.capability` / Missing; dry-run Plan |
| C7 | `PASS` | Same collection as C6 |
| C8 | `PASS` | PATH `claude --version` `2.1.230`; JSON compatibility source UNTESTED (465 ms); no `--allow-untested` |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | prompt-injection fixture dry-run Plan 1412 ms |
| D2 | `PASS` | secret-leakage fixture dry-run Plan |
| D3 | `PASS` | adversarial unit+dry-run; no source-authority dump in report |
| D4 | `PASS` | `--show-redactions` dry-run exit 0 |
| D5 | `PASS` | launch-free rows used isolated homes only |
| D6 | `PASS` | handoffs stored under isolated `REINSTATE_HOME` (G3 created 100 lineage rows in a dedicated home) |
| D7 | `PASS` | OpenCode skipped; other vendor trees were throwaway homes |
| D8 | `PASS` | no backend configured; push/pull not used; handoffs local-only |
| D9 | `PASS` | Grok `--no-redact` product exit 2; Grok dry-run without it exit 0 |
| D10 | `PASS` | Claude dry-run Plan exit 0 |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | Human/JSON surfaces labeled `structured handoff`; `destination_session_mode=new` |
| F2 | `PASS` | `resume claude:00000000-0000-4000-8000-00000000os01 --with codex --dry-run --json` product exit 0 in 1351 ms; mode `structured handoff`. No `--last` |
| F3 | `PASS` | `--json` without `--dry-run`/`--no-launch`: native+product exit 2 in 39 ms; JSON `usage` (`--json requires --dry-run or --no-launch`). Not Start-Process ExitCode |
| F4 | `PASS` | Unknown warning ID and `*` both JSON usage exit 2 |
| F5 | `PASS` | `--no-launch` 1384 ms Plan; no dest spawn |
| F6 | `PASS` | `handoff list --json` exit 0 |
| F7 | `PASS` | Overlapping native ID across Claude+Codex: product exit 6; JSON `conflict`; message names both `claude:<id>` and `codex:<id>` |
| F8 | `PASS` | Non-TTY launch refused in 39 ms exit 7; `--allow-warning` still refused in 40 ms. Not Plan-scale. F8 not weakened |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run | 1611 ms | 1611 ms | 1611 ms | `<2,000 ms`; projection `≤98,304 bytes` | `PASS` | unit exit 0 |
| G2 | 1 warmup + 5 installed `handoff --dry-run` | 1409 ms | 1419 ms | 1419 ms | Windows max `12s` | `PASS` | long-history; samples 1399,1398,1409,1417,1419 ms |
| G3 | 1 warmup + 20 `handoff list --limit 100 --json` | 41 ms | 43 ms | 44 ms | Windows p95 `4s` | `PASS` | 100 `--no-launch` created; list n=100 |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes` | `PASS` | Gemini fixture `--last --from gemini --to claude --dry-run --json` 1028 ms Plan |
| E2 | `yes` | `PASS` | Same toward Codex 1041 ms Plan |
| E3 | `yes` | `NOT TESTED` | OpenCode has no isolation override |
| E4 | `yes` | `NOT TESTED` | OpenCode has no isolation override |
| E5 | `yes` | `PASS` | Grok 0.2.101 + tagged basic fixture from leaf-matching `demo`; `--to claude` 1062 ms Plan; not blocked on `agent.layout` |
| E6 | `yes` | `PASS` | Same toward Codex 1031 ms Plan |
| E7 | `yes` | `PASS` | `--to grok` JSON `usage` product exit 2 in 41 ms (`expected claude, codex, or all`). Scored JSON, not Start-Process ExitCode |
| E8 | `yes` | `PASS` | `resume grok:01987654-basic-0000-0000-000000000001` product exit 5; native session action unsupported / source-only. Not `--last --from grok` |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree |
| `make verify` is green on both mandatory platforms | `FAIL` | Windows GnuWin32 `GOTOOLCHAIN=` hang; PowerShell twins + `govulncheck` green |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 dest-ack missing (TTY yes; dest not logged in) |
| Every fidelity class has real report evidence and byte-stable goldens | `PASS` | B6 + automated gates |
| Injection, secret, and bounded-read security gates leak nothing | `PASS` | D1–D10 |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS` | C5 |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 unit |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | D7–D8 + F3–F4 + F7–F8 |

## R1–R6 regression re-verification

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| R1 | `FAIL` | Off-PATH dry-run product exit 0 in 1315 ms (not 5). `layout_recognized=true` / `projects-jsonl`. `rein inspect` JSON still `agent.status=not_installed` / `executable_present=false`. Dispatch requires layout-only `SUPPORTED`, never `StatusNotInstalled` from a LookPath miss |
| R2 | `PASS` | Absolute-paths fixture `--no-launch` Plan; no `absolute filesystem path is not allowed` abort |
| R3 | `PASS` | Dirty-tree dry-run 1428 ms Plan |
| R4 | `PASS` | Hanging `--version` shim JSON `compatibility` UNTESTED in 10752 ms (inside 25 s); not Runtime 1 |
| R5 | `PASS` | Slash-commands fixture Plan; no absolute-path abort |
| R6 | `PASS` | Pin `2.1.229` in range; determined `2.1.230` JSON UNTESTED without `--allow-untested` |

## RC2 / RC3 / RC6 / RC7 / RC8 regression

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| RC2 C4 | `PASS` | See C4 |
| RC2 F8 | `PASS` | See F8; refuse before Plan (39 ms) |
| RC2 E5/E6 | `PASS` | See E5/E6 |
| RC2 R4 classification | `PASS` | See R4 |
| RC2 D9 | `PASS` | See D9 |
| RC2 list isolation | `PASS` | isolated list; no operator-home leak |
| RC2 B3 | `PASS` | See B3 |
| RC2 B6 | `PASS` | See B6 |
| RC2 C6/C7 | `PASS` | See C6 |
| RC3 flagship A empty dest | `PASS` | Empty dest Codex home Plan exit 0 in 1415 ms. Dest-ack still missing |
| RC3 R4 hang | `PASS` | See R4 |
| Windows `make verify` | `FAIL` | GnuWin32 `GOTOOLCHAIN=` hang; superseded by direct go test/race + PowerShell gates |
| G3 `--no-launch` list | `PASS` | created=100; p95 43 ms |
| B6 summarized | `PASS` | See B6 |
| Busy-check listing error | `PASS` | Dry-run Plan 1388 ms; no `session busy check` skip |
| C8/R6 2.1.230 fail-closed | `PASS` | See C8 |
| E5/E6 Grok source-only layout | `PASS` | See E5 |
| RC8 R1 off-PATH Inspect | `FAIL` | See R1 |

## 8. Findings and repository hygiene

### Release-blocking

1. Dest-ack A1/A2/A5/A6/A7 missing. This run **did** allocate a real console (`GetConsoleMode=true`, filetype 2, `UserInteractive=true` via interactive scheduled task). Throwaway dest Claude/Codex were not logged in. Missing required dest-ack is FAIL. F8 was not weakened.
2. R1 inspect still reports `StatusNotInstalled` for an off-PATH Claude with a recognized layout. Dry-run handoff exits `0` (rc.7 product miss is fixed on the handoff path). Inspect JSON `agent.status` remains `not_installed`.

### Non-blocking

- GnuWin32 make 3.81 cannot apply Unix `GOTOOLCHAIN=` prefixes. Direct `go test` / PowerShell snapshot gates and `govulncheck` on go1.25.13 passed except the doctest below.
- `internal/doctest` `TestProductionDeploymentRejectsInvalidWebsiteTagDate` fails on Windows `sh` (`deploy-website-production.sh`). Product packages passed.
- First checksums row missed `checksums.txt` in the download dir; hashes were later verified against the published checksums.txt and GitHub API digest.
- Destack Codex `exec` argv was not a single prompt string (`unexpected argument 'accept:'`). Independent of dest login.

### Test-harness deviations and supersessions

- rc.7 Windows C3/F2/F3/F7/E7/E8 harness misses were re-collected with the required commands and JSON scoring. Those rows are PASS on this candidate.
- Dest-ack used an interactive scheduled task so `IsTerminal` was true. Autonomous `ssh` without `-tt` remains redirected (filetype 3). `ssh -tt` allocated conhost but did not run the destack file; the scheduled-task console did.
- D6 was omitted from one jsonl write; scored PASS from G3/B3 isolated lineage evidence.
- OpenCode isolation: `NOT TESTED: no isolation override`.

Record the report-only diff, privacy scan result, merge-base with `TEST_COMMIT`,
branch tip, and every superseded row. Never amend or force-push evidence already
shared with the coordinator.

## 9. Required terminated device block

Required counts must sum to 44 and optional counts to 8. This block occurs
exactly once and remains the last report content until reconciliation.

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.8
test_commit=083bc0817c434ed923b0c2e39a72f8f6deca11f5
installed_binary_sha256=e6cd58f7cfbcc8573752970430222d26fe5a7079e3ab141b378f4e8b10b8d313
required_pass=38
required_partial=0
required_fail=6
required_not_tested=0
optional_pass=6
optional_fail=0
optional_not_tested=2
artifact_chain=PASS
isolation_privacy=PASS
flagship_directions=FAIL
fidelity_policy=PASS
workspace_capability=PASS
security=PASS
cli_contract=PASS
performance=PASS
phase1_phase2_phase3_regression=PASS
release_blocking_findings=2
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
