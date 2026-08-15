# Phase 4 Windows x64 device report — v0.4.0-rc.9

Copy of the Phase 4 structured-handoff template with this dispatch’s substitutions. Cumulative, sanitized, append-only. No transcript text, prompts, responses, capsule bodies, credentials, private paths, filenames, diffs, or raw child-process output.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `38 PASS / 0 PARTIAL / 6 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `6 PASS / 0 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `1`

Required arithmetic: 38+0+6+0=44. Optional: 6+0+2=8.

Device FAIL is dest-ack A1/A2/A3/A5/A6/A7: real TTY collected (`IsTerminal` true); throwaway dest Claude/Codex were **not logged in** in isolated homes before the first A row. F8 was not weakened. This is HARNESS, not product.

**PRODUCT RC9 R1 PASS:** `rein inspect --json` `environment.agent.status=supported` with `layout_recognized=true`, `executable_present=false`; dry-run exit `0`; dest launch fail-closed exit `7`.

`stable_v0.4.0_authorized=false`. `current_stable=v0.3.0`. This candidate is not authorized as stable `v0.4.0`.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-15T03:02:37Z` start; dest-ack TTY `2026-08-15T03:12:23Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11, `10.0.26200.8328` |
| CPU architecture/native process | x64; 64-bit Windows PowerShell 5.1.26100.8328; never WSL |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.9` |
| Tested full commit | `792cfb9ba492345f3cc7fb48632f38553e955d30` |
| Installed binary SHA-256 | `6898b70a5d973716459b9853ee3e794922c3cad481d255a95f1fa28e715c2171` |
| Installed version JSON | `version=0.4.0-rc.9`, `commit=792cfb9ba492345f3cc7fb48632f38553e955d30`, `date=2026-08-15T02:46:31Z`, `name=reinstate` |
| Claude Code version/state | Host before: `2.1.232` (above ceiling, unused). PATH pin `2.1.229` for all product rows. After pin: `2.1.229`. Dest Claude in isolated home: not logged in. |
| Codex CLI version/state | `0.147.0` before and after (in 0.133.0–0.147.0). Dest Codex in isolated home: not logged in. |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` present. OpenCode `1.18.2` present, **not isolated**. Grok `0.2.101` present. |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.25.13 windows/amd64`; `$env:GOTOOLCHAIN='go1.25.13'` |
| Report branch | `test/v0.4.0-rc.9-windows-amd64-report` |
| Device-report commit | `PENDING` |
| Draft report PR | `NOT CREATED` |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | SSH allowed_signers good signature |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | peeled `792cfb9ba492345f3cc7fb48632f38553e955d30`; ancestor of `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | draft=false, prerelease=true |
| Exact expected release asset set is present | `PASS` | 25 assets |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | Harness row `art.checksums` first FAIL (checksums.txt not saved in evidence dir). Independent verify: zip `74082bd9c15bbbd8dec8cde0feb0febce51b886a8dd3dc7f85a1fb1a7d21c3c0` matches checksums.txt and GitHub API `sha256:74082bd9…`; exe `6898b70a5d973716459b9853ee3e794922c3cad481d255a95f1fa28e715c2171` matches checksums.txt. Attestation PASS. Supersedes harness FAIL. |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | zip entries=6, unsafe=0 |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.9` | `PASS` | live `install.ps1` sha256 `ca8556cc77f9439dbadc15fd35505eda887322b6187fecf54714ba4daf348787`; pins=`v0.4.0-rc.9` only; no rc.1–rc.8 pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | canonical `scripts/install.ps1` sha256 `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | user PATH unchanged; user binary unchanged |
| `rein` and `reinstate` have identical verified bytes and report `0.4.0-rc.9` | `PASS` | both sha256 `6898b70a5d973716459b9853ee3e794922c3cad481d255a95f1fa28e715c2171`; version `0.4.0-rc.9`. Harness `art.binary_identity` first FAIL; independent hashes+version JSON supersede. |
| Installed binary reports the literal full tested commit | `PASS` | version JSON commit literal match |

Install used live `https://reinstate.dev/install.ps1` into a new directory with `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`.

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | exit 0, 420 ms |
| `make verify` with Go 1.25.13 | `FAIL` | GnuWin32 make skipped (cannot apply Unix `GOTOOLCHAIN=` prefixes). Direct `go test ./...` exit 1: `TestProductionDeploymentRejectsInvalidWebsiteTagDate` (`sh` production-deploy script; unexpected invalid-date failure). POSIX-only doctest on Windows. PowerShell snapshot/stage/artifacts/install PASS. |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | same doctest only; other packages ok |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS` | snapshot 19198 ms exit 0; stage/check-artifacts/test-install exit 0 |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | agentcheck/preflight/handoff/cli/workspace unit packages PASS |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | `TestHandoffExecutedOutputMatchesDryRunByteForByte` exit 0 |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | handoff unit + fixture Scan exit 0 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | G1: projection 46541 bytes; unit 9.9 ms; wrapper 1568 ms |
| `govulncheck` | `PASS` | exit 0, 5217 ms, Go 1.25.13 |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | new rc.9 iso tree; not rc.1–rc.8 homes |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | fixtures + this-run lineage only |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | five overrides set; operator trees not listed/copied |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | D8; push/pull unused |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | dummy MCP/skill in throwaway Claude home only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | dest-ack did not launch dest (not logged in) |
| First index list isolation | `PASS` | `RC2.list_isolation` PASS; no operator-home leak |
| OpenCode | `NOT TESTED` | no isolation override; E3/E4 recorded as such; never reported isolated |

## 5. Required matrix — 44 rows

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | TTY PASS (`GetConsoleMode=true`, `UserInteractive=true`). Dest Claude/Codex **not logged in** in isolated homes before first A row (`dest.login` FAIL). Missing dest-ack. |
| A2 | `FAIL` | same dest-login gate. A2 dry-run Plan PASS separately (`A2.dryrun` exit 0). |
| A3 | `FAIL` | dest not logged in; logged-out dest-ack not a substitute for logged-in dest-ack |
| A4 | `PASS` | interrupted final-turn fixture; dry-run exit 0 |
| A5 | `FAIL` | no dest first-reply; dest not logged in |
| A6 | `FAIL` | MARKER non-repeat not observed; dest not launched |
| A7 | `FAIL` | dest IDs unresolved without launch |

Empty dest Plan: `RC3.flagshipA` PASS (exit 0). F8 not weakened (non-TTY refuse exit 7, 40 ms).

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | dry-run and `--no-launch` exit 0; unit goldens PASS |
| B2 | `PASS` | long-history balanced dry-run exit 0 |
| B3 | `PASS` | checkpoint `--no-launch`; `projection_events` present |
| B4 | `PASS` | unknown-records exit 0 |
| B5 | `PASS` | attachments full dry-run exit 0 |
| B6 | `PASS` | compaction; fidelity class `summarized` |
| B7 | `PASS` | subagents exit 0 |
| B8 | `PASS` | os-roots dry-run exit 0 |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | dirty tree dry-run exit 0 |
| C2 | `PASS` | workspace-wins covered with C1 |
| C3 | `PASS` | missing workspace leaf `missing-ws-rc9` (not demo); exit 5 |
| C4 | `PASS` | wrong-repo leaf-match exit 5; subdir exit 0; nongit exit 0 |
| C5 | `PASS` | foreign-OS remap dry-run exit 0; no busy-check hang |
| C6 | `PASS` | dummy MCP; exit 0 or 7 |
| C7 | `PASS` | dummy skill with C6 |
| C8 | `PASS` | determined 2.1.230 fail-closed exit 5 `untested` |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | injection fixture dry-run bounded |
| D2 | `PASS` | secret-leakage fixture bounded |
| D3 | `PASS` | no source-authority dump in report |
| D4 | `PASS` | `--show-redactions` categories/counts only |
| D5 | `PASS` | launch-free rows isolated homes only |
| D6 | `PASS` | handoffs under isolated `REINSTATE_HOME` |
| D7 | `PASS` | OpenCode skipped; other trees throwaway |
| D8 | `PASS` | no backend; no capsule sync |
| D9 | `PASS` | Grok `--no-redact` exit 2 |
| D10 | `PASS` | dry-run exit 0 |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | label `structured handoff`; no “same session”/“lossless” |
| F2 | `PASS` | `resume claude:<uuid> --with codex --dry-run --json` (no `--last`); exit 0 |
| F3 | `PASS` | `--json` without dry-run/no-launch; **product JSON exit 2** (not Start-Process ExitCode) |
| F4 | `PASS` | unknown and wildcard `--allow-warning` exit 2 |
| F5 | `PASS` | `--no-launch` exit 0 |
| F6 | `PASS` | `handoff list --json --limit 5` exit 0 |
| F7 | `PASS` | overlapping native IDs **exit 6** (conflict), not session-not-found |
| F8 | `PASS` | non-TTY refuse exit 7 in 40 ms; `--allow-warning handoff.interactive` still 7 |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged unit | 9.9 ms inner / 1568 ms wrapper | — | 1568 ms | `<2,000 ms`; `≤98,304` bytes | `PASS` | 200-turn; projection 46541 bytes |
| G2 | 1 warmup + 5 | 1361 ms | — | 1404 ms | Windows max `12s` | `PASS` | samples 1360,1356,1361,1396,1404 ms |
| G3 | 1 warmup + 20 | — | 41 ms | — | Windows p95 `4s` | `PASS` | 100 lineage rows created; list exit 0; samples 38–49 ms |

Ordinary antivirus enabled. No timeout, invalid output, or source mutation.

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | yes | `PASS` | Gemini → Claude dry-run exit 0 |
| E2 | yes | `PASS` | Gemini → Codex dry-run exit 0 |
| E3 | yes | `NOT TESTED` | OpenCode has no isolation override |
| E4 | yes | `NOT TESTED` | OpenCode has no isolation override |
| E5 | yes | `PASS` | Grok → Claude dry-run exit 0 |
| E6 | yes | `PASS` | Grok → Codex dry-run exit 0 |
| E7 | yes | `PASS` | Claude → Grok dest usage **product JSON exit 2** |
| E8 | yes | `PASS` | `resume grok:<id>` (not `--last --from grok`); non-zero refuse |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree `792cfb9ba492345f3cc7fb48632f38553e955d30` |
| `make verify` is green on both mandatory platforms | `FAIL` | Windows: GnuWin32 skipped; one POSIX doctest fail; govulncheck green |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 dest-ack missing (dest not logged in) |
| Every fidelity class has real report evidence and byte-stable goldens | `PASS` | B6 + gates |
| Injection, secret, and bounded-read security gates leak nothing | `PASS` | D1–D10 |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS` | C5 + R2 |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 unit + installed dry-run/`--no-launch` |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | F3=2, F7=6, F8=7, C3/C4/C8=5; D7–D8 |

## 8. Findings and repository hygiene

### Release-blocking

1. **Dest-ack A1/A2/A3/A5/A6/A7 FAIL (harness).** Real TTY collected. Isolated dest Claude/Codex not logged in (no process/User/Machine API keys; `claude auth status` / `codex login status` did not prove login). Operator vendor trees were not read. Missing dest-ack is FAIL. Do not weaken F8.

### Non-blocking

1. `go test ./...` / `-race`: `TestProductionDeploymentRejectsInvalidWebsiteTagDate` unexpected invalid-date failure via POSIX `sh` deploy script on Windows. Not a Windows tagged-binary identity defect.
2. Host Claude `2.1.232` is above the 2.1.229 ceiling; unused. PATH pin `2.1.229` before first product command; after still `2.1.229`.
3. GnuWin32 `make` skipped; PowerShell gates used.

### Test-harness deviations and supersessions

- `art.checksums` harness FAIL superseded by independent checksums.txt + GitHub API digest match (PASS).
- `art.binary_identity` harness FAIL superseded by identical `rein.exe`/`reinstate.exe` sha256 and version JSON (PASS).
- `gate.make_verify` scored from direct `go test`/`-race` after GnuWin32 skip; remains FAIL due to POSIX doctest.
- D6 recorded after matrix (handoffs dir present); same isolated home.
- A1–A7: PENDING_TTY then non-TTY FAIL then TTY+nologin FAIL. Latest: TTY true, dest login false.
- No product files edited. No existing `docs/testing/results/` file modified. No secrets/transcripts committed.

R1–R6: R1 PASS, R2 PASS, R3 PASS, R4 PASS, R5 PASS, R6 PASS.

RC2: C4, F8, E5/E6, R4 class, D9, list isolation, B3, B6, C6/C7 PASS.

RC3: flagship empty dest PASS; R4 hang PASS; G3 PASS; B6 PASS; Windows make/go test as above.

RC5: R4 process-tree unit PASS (20 ms); F8 refuse-before-Plan PASS.

RC6: C4 leaf-match PASS; F8 refuse-before-index PASS (40 ms).

RC7: busy-check PASS; C8/R6 PASS; E5/E6 PASS.

RC8: R1 layout scan + dry-run 0 PASS.

RC9: inspect JSON `status=supported` PASS.

## 9. Required terminated device block

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.9
test_commit=792cfb9ba492345f3cc7fb48632f38553e955d30
installed_binary_sha256=6898b70a5d973716459b9853ee3e794922c3cad481d255a95f1fa28e715c2171
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
release_blocking_findings=1
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```
