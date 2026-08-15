# Phase 4 structured-handoff report — Windows x64 (`v0.4.0-rc.11`)

Copy of the Phase 4 template with this dispatch's substitutions. Cumulative, sanitized, append-only. No transcript, prompt, response, capsule body, credential, or operator-tree path is committed.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `40 PASS / 0 PARTIAL / 4 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `6 PASS / 0 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `1`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result is `FAIL`. Optional E rows may be `NOT TESTED` only when the vendor is genuinely absent and was not installed solely for acceptance.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-15T12:35:34Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro `10.0.26200` (native, never WSL); PowerShell `5.1.26100.8328` |
| CPU architecture/native process | `AMD64` 64-bit native |
| Filesystem | `NTFS` |
| Tested tag | `v0.4.0-rc.11` |
| Tested full commit | `e05610bff7f4e8f36f7b4227a248dcccd4f7eb6b` |
| Installed binary SHA-256 | `36245aaf7c61c9852f6c4a112b15d82fb2cf7415c4483ca58308cde880d45f29` |
| Installed version JSON | `reinstate 0.4.0-rc.11 (e05610bff7f4e8f36f7b4227a248dcccd4f7eb6b 2026-08-15T11:58:51Z)` |
| Claude Code version/state | pin `2.1.229` on isolated PATH for in-range dest; host PATH `2.1.233` (above ceiling) used only for R6/C8 fail-closed; dest **not logged in** (`LOGIN_OK` absent) |
| Codex CLI version/state | `0.147.0` (in range); dest **not logged in** (`LOGIN_OK` absent) |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` present (isolated copy); OpenCode `1.18.2` present but no isolation override; Grok `0.2.101` present (isolated copy) |
| Git version | `git version 2.52.0.windows.1` |
| Go version/toolchain | `go1.25.13 windows/amd64` (`$env:GOTOOLCHAIN='go1.25.13'`) |
| Report branch | `test/v0.4.0-rc.11-windows-amd64-report` |
| Device-report commit | `f1ded2d235cf731a5d41f59404482d9f171813ab` (report-only tip follows) |
| Draft report PR | `NOT CREATED` |

Before first product command and after last row: Claude pin `2.1.229`, Codex `0.147.0`, Gemini `0.53.0`, OpenCode `1.18.2`, Grok `0.2.101`. **No mid-run version change.** `DISABLE_AUTOUPDATER=1` set. Host Claude `2.1.233` unused except C8/R6 fail-closed.

## 2. Signed artifact and installer chain

No downloaded executable ran until tag, checksum, API digest, attestation, archive membership, and platform identity passed. Live install into a new ISO `install` directory with `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` and `REINSTATE_SKIP_PATH_UPDATE=1`.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | SSH-signed annotated tag; `git verify-tag v0.4.0-rc.11` Good ED25519 |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | peel `e05610bff7f4e8f36f7b4227a248dcccd4f7eb6b`; ancestor of `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | draft=false, prerelease=true, tag `v0.4.0-rc.11` |
| Exact expected release asset set is present | `PASS` | 25 assets (checksums + 5 archives + 5 SBOMs + 5 raw binaries + 8 Linux packages + source) |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | checksums.txt matches API digests; installed SHA-256 matches `windows_amd64.exe`; zip attestation verifies tag+commit |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | zip: 6 relative entries, `rein.exe`+`reinstate.exe`+docs; no `..` or absolute names |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.11` | `PASS` | live `https://reinstate.dev/install.ps1` sha256 `fee9a83f0ae79b515288ddc5aa715145919fbef0251a64e54fde170d28ffaa2c` = `website/public/install.ps1`; `$Version = "v0.4.0-rc.11"` only; no rc.1–rc.10 pin |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | `PinnedInstallerSha256` = `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` = `scripts/install.ps1` at TEST_COMMIT |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`; `REINSTATE_SKIP_PATH_UPDATE=1` |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.11` | `PASS` | identical SHA-256 `36245aaf…d45f29` |
| Installed binary reports the literal full tested commit | `PASS` | full TEST_COMMIT in `version` |

## 3. Automated gates

GnuWin32 `make` 3.81 present; not used for GOTOOLCHAIN. PowerShell gates + `go test` / `-race` directly.

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | source clone HEAD = TEST_COMMIT; tidy exit 0 |
| `make verify` with Go 1.25.13 | `FAIL` | host `go test ./...` fails only `internal/doctest` `TestProductionDeploymentRejectsInvalidWebsiteTagDate` (POSIX `sh` on Windows; same host class as rc.9/rc.10). `internal/handoff` PASS. govulncheck green |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | same single doctest FAIL; `internal/handoff` race PASS |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `FAIL` | `scripts/snapshot.ps1` blocked by WinGet `goreleaser` shim (no associated app); not a tagged-binary identity defect |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | R1–R6 PASS on isolated homes |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | D10 source hash unchanged; G3 source hash unchanged |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | `go test ./internal/fixture` ok; D1–D10 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | G1 inner `10.9576 ms`, projection `46541` bytes |
| `TestHandoffExecutedOutputMatchesDryRunByteForByte` | `PASS` | `go test ./internal/cli -run …` exit 0 |
| `govulncheck` | `PASS` | `No vulnerabilities found` (v1.6.0) |
| `gofmt -l .` | `PASS` | empty |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | rc.11 ISO homes; all five vendor overrides; OpenCode not isolated |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | disposable `task-alpha` / uniquified fixtures; not rc.10 ISO |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | dest login **not** performed; credential values not read |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | D8 |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | isolated Claude pin `2.1.229`; Codex `config.toml` project trust in isolated home only (literal-quoted Windows keys) |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | D10; G3; no live dest launch |

After first index refresh: planted fixtures only (this run). Isolation restart not required.

## 5. Required matrix — 44 rows

Use the exact IDs and pass conditions in `phase-4-cross-agent-handoff-acceptance.md`. Cross-agent work starts a **new destination session continuing the same task**.

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | TTY `GetConsoleMode=True` via `ssh -tt` + local PTY. Dry-run dest argv `n=1` one-line absolute `projection.md`, no CR/LF, mode `structured handoff`. Live Codex dest **not launched**: `LOGIN_OK` absent; isolated dest homes have no auth files. Missing required dest-ack is FAIL (HARNESS). Source Claude was not running. |
| A2 | `FAIL` | TTY true. Dry-run dest argv `n=3` with `projection.md` present. Live Claude dest **not launched** (`LOGIN_OK` absent). Missing dest-ack is FAIL (HARNESS). |
| A3 | `FAIL` | Live Claude→Codex from logged-out source not collected because dest-ack was blocked on dest login. Missing required live A3 is FAIL. |
| A4 | `PASS` | tagged `partial-final-record` fixture; latest complete user intent survives |
| A5 | `FAIL` | No live dest first-reply. Cannot prove five acknowledgement bullets (goal, latest request, changed files, test state, next action). Missing dest-ack is FAIL. |
| A6 | `PASS` | harmless MARKER SHA-256 unchanged (no live dest rewrite) |
| A7 | `PASS` | `rein handoff list --json` `len(handoffs)=4` (no `n` field); scored from stored lineage after `--no-launch` (lineage before launch / list dir recovery) |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | unit `./internal/cli` + installed dry-run and no-launch exit 0 |
| B2 | `PASS` | uniquified long-history; `estimated_bytes=46737` |
| B3 | `PASS` | checkpoint `--no-launch` exit 0 |
| B4 | `PASS` | balanced dry-run exit 0 |
| B5 | `PASS` | full dry-run exit 0 |
| B6 | `PASS` | uniquified compaction; `summarized` class present |
| B7 | `PASS` | uniquified attachments dry-run exit 0 |
| B8 | `PASS` | uniquified unknown-records dry-run exit 0 |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | dirty tree `dirty=True` |
| C2 | `PASS` | workspace-wins with C1 |
| C3 | `PASS` | missing leaf `missing-ws-rc11` (not `demo`) exit 5 |
| C4 | `PASS` | wrong-repo exit 5; subdir/nongit/foreign-leaf covered |
| C5 | `PASS` | repo token; no absolute leak |
| C6 | `PASS` | capability warnings present; dry-run exit 0 |
| C7 | `PASS` | skill with C6 |
| C8 | `PASS` | host `2.1.233` no pin fail-closed exit 5 `UNTESTED` |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | injection fixture |
| D2 | `PASS` | fence-breakout |
| D3 | `PASS` | no source-authority dump |
| D4 | `PASS` | show-redactions |
| D5 | `PASS` | launch-free isolated homes |
| D6 | `PASS` | handoffs under isolated `REINSTATE_HOME` |
| D7 | `PASS` | OpenCode skipped (no isolation override); throwaway vendor homes |
| D8 | `PASS` | no backend; no capsule sync |
| D9 | `PASS` | grok `--no-redact` refused exit 2 |
| D10 | `PASS` | source sha unchanged |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | `structured handoff` label; banned phrasing absent |
| F2 | `PASS` | `resume claude:<uuid> --with codex --dry-run --json` (no `--last`) exit 0 |
| F3 | `PASS` | `--json` without dry-run/no-launch exit 2 (`LASTEXITCODE`, not Start-Process) |
| F4 | `PASS` | unknown and wildcard warning IDs exit 2 |
| F5 | `PASS` | `--no-launch` exit 0 |
| F6 | `PASS` | inspect/export; id length 32 |
| F7 | `PASS` | overlapping native IDs exit 6 |
| F8 | `PASS` | non-TTY launch refuse exit 7 in 52 ms (not weakened) |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit | n/a | n/a | 10.9576 ms | `<2,000 ms`; projection `≤98,304` B | `PASS` | inner parse/project time; projection `46541` B |
| G2 | 1935,1923,1981,1946,1924 ms | 1935 ms | n/a | 1981 ms | 12 s | `PASS` | 1 warmup + 5 timed, all exit 0; uniquified long-history |
| G3 | n=100 `--no-launch` | n/a | 70 ms | 71 ms | 4 s | `PASS` | dedicated G3 home; 20 timed all exit 0; source hash unchanged |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | yes | `PASS` | Gemini rewind id `gemini:gemini-rewind-win` → Claude dry-run exit 0 |
| E2 | yes | `PASS` | same Gemini id → Codex dry-run exit 0 |
| E3 | yes (no isolation override) | `NOT TESTED` | OpenCode skip; never reported isolated |
| E4 | yes (no isolation override) | `NOT TESTED` | OpenCode skip |
| E5 | yes | `PASS` | Grok→Claude fixture |
| E6 | yes | `PASS` | Grok→Codex fixture |
| E7 | yes | `PASS` | dest grok `--json` usage exit 2 |
| E8 | yes | `PASS` | `resume grok:<id>` refused exit 5 |

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS` | tagged tree |
| `make verify` is green on both mandatory platforms | `FAIL` | Windows doctest host FAIL (see §3) |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `FAIL` | A1–A2 dest-ack not collected |
| Every fidelity class has real report evidence and byte-stable goldens | `PASS` | B6 + automated gates |
| Injection, secret, and bounded-read security gates leak nothing | `PASS` | D1–D10 |
| 200-turn projection is bounded and reported | `PASS` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS` | C5 |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS` | D7–D8 + F3–F4 |
| Destination argv never embeds CR/LF; Codex dest falls back to one-line absolute `projection.md` | `PASS` | RC10.argv / A1 dry-run `n=1` |

## 8. Findings and repository hygiene

### Release-blocking

- Required dest-ack A1/A2/A3/A5 **FAIL** (HARNESS): throwaway dest Claude/Codex were not logged in (`LOGIN_OK` missing). Live five-bullet first-reply therefore not collected. Missing required dest-ack is FAIL. Dry-run argv and lineage-before-launch still PASS.

### Non-blocking

- `internal/doctest` invalid-date test FAIL on native Windows (POSIX `sh`); same host class as rc.9/rc.10. `internal/handoff` unit+race PASS.
- GoReleaser snapshot blocked by WinGet shim; tagged release zip/exe identity already verified.
- OpenCode present but un-isolated: E3/E4 `NOT TESTED`.

### Test-harness deviations and supersessions

- Live dest-ack used `ssh -tt` so remote `GetConsoleMode` was true. Did **not** use `05-destack.ps1`.
- A1.dryrun / A2.dryrun / RC10.argv remain PASS and are not superseded.
- First-pass B2/B6/B7/B8/G2/E1/E2/R1 FAIL from un-uniquified fixture IDs / dest off-PATH; **superseded** after uniquify + Codex-kept R1 + Gemini sessionindex id. Final rows above are the collected results.
- G2 first-pass harness scored PASS on exit 2 (session not found); **superseded** by uniquified G2 all exit 0, max 1981 ms.

### RC / R (reported separately; not in 44-count)

| ID | Result | Notes |
| -- | ------ | ----- |
| R1 | `PASS` | off-PATH inspect `status=supported`, `layout_recognized=true`, `executable_present=false`; dry-run exit 0; dest launch fail-closed exit 7 |
| R2 | `PASS` | reader paths tokenized; C5 artifacts |
| R3 | `PASS` | dirty tree `changed_files` |
| R4 | `PASS` | hanging `--version` shim exit 5 in 10838 ms (Compatibility `UNTESTED`) |
| R5 | `PASS` | slash-prefixed prose fixture |
| R6 | `PASS` | ceiling through `2.1.229`; host `2.1.233` fail-closed |
| RC2 C4/F8/D9/B3/B6/C6/C7 | `PASS` | |
| RC3 empty dest / G3 / B6 | `PASS` | empty dest plan-supported; G3 n=100 |
| RC5 R4 process-tree / F8 refuse-before-Plan | `PASS` | F8 52 ms |
| RC6 C4 leaf-match / F8 refuse-before-index / C5 | `PASS` | |
| RC8 R1 layout scan / C3 not demo / F2 no `--last` / F3 JSON usage / F7 exit 6 / E8 `resume grok:<id>` | `PASS` | |
| RC9 inspect JSON `status=supported` | `PASS` | with R1 |
| RC10 dest argv one-line `projection.md` | `PASS` | dry-run `n=1` |
| RC11 five-bullet first-reply | `FAIL` | not collected (dest not logged in) |
| RC11 lineage-before-launch / list recovery | `PASS` | A7 `len(handoffs)=4` |
| RC11 dest-home workspace trust | `PASS` (setup) | isolated Codex `trust_level=trusted` literal-quoted keys; live dest-ack not exercised |

## 9. Required terminated device block

Required counts must sum to 44 and optional counts to 8. This block occurs exactly once and remains the last report content until reconciliation.

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.11
test_commit=e05610bff7f4e8f36f7b4227a248dcccd4f7eb6b
installed_binary_sha256=36245aaf7c61c9852f6c4a112b15d82fb2cf7415c4483ca58308cde880d45f29
required_pass=40
required_partial=0
required_fail=4
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
