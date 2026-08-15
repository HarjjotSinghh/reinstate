# Phase 4 structured-handoff report — Windows x64 (`v0.4.0-rc.10`)

Copy of the Phase 4 report template with dispatch substitutions for
`v0.4.0-rc.10`. Cumulative, sanitized, append-only. No transcripts, prompts,
responses, capsule bodies, credentials, configuration values, private paths,
filenames, diffs, or raw child-process output.

## Verdict

- **Device verdict:** `FAIL`
- **Milestone:** `MATRIX_COMPLETE`
- **Required counts:** `40 PASS / 0 PARTIAL / 4 FAIL / 0 NOT TESTED`
- **Optional source-only counts:** `6 PASS / 0 FAIL / 2 NOT TESTED`
- **Release-blocking findings:** `1`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. Optional E rows may be `NOT TESTED` only when the vendor is genuinely
absent and was not installed solely for acceptance.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-08-15T09:10:00Z` |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 x64 |
| CPU architecture/native process | amd64 native (not WSL) |
| Filesystem | NTFS |
| Tested tag | `v0.4.0-rc.10` |
| Tested full commit | `78a00a1bbf6953171d05a5c6680f76f4680ef464` |
| Installed binary SHA-256 | `b2a88ac1e512e5152ad063b6626af3c68c8957b277544b05ca86348685afc65d` |
| Installed version JSON | `0.4.0-rc.10` / `78a00a1bbf6953171d05a5c6680f76f4680ef464` / `2026-08-15T07:39:29Z` |
| Claude Code version/state | pin `2.1.229` on isolated PATH for in-range dest; host PATH `2.1.232` (above ceiling) used only for R6/C8 fail-closed |
| Codex CLI version/state | `0.147.0` (in range); dest logged in via isolated home (credentials not read) |
| Gemini/OpenCode/Grok state | Gemini `0.53.0` present; OpenCode `1.18.2` present but no isolation override; Grok `0.2.101` present |
| Git version | present |
| Go version/toolchain | `go1.25.13` via `GOTOOLCHAIN` |
| Report branch | `test/v0.4.0-rc.10-windows-amd64-report` |
| Device-report commit | `4db8590433a82278d4c9e4917cd2f2835bab5d32` |
| Draft report PR | https://github.com/HarjjotSinghh/reinstate/pull/233 |

## 2. Signed artifact and installer chain

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS` | SSH-signed annotated tag peels to TEST_COMMIT |
| Tag peels to the recorded full commit on `origin/main` | `PASS` | ancestor of `origin/main` |
| Published release is non-draft, prerelease, and tied to the tag | `PASS` | draft=false, prerelease=true |
| Exact expected release asset set is present | `PASS` | 25 assets |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS` | installed SHA-256 matches `windows_amd64.exe` |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS` | windows_amd64 identity |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.10` | `PASS` | live `install.ps1` `$Version = "v0.4.0-rc.10"` |
| Bootstrap installer digest matches the tagged canonical installer | `PASS` | `02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe` |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS` | `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.10` | `PASS` | identical SHA-256 |
| Installed binary reports the literal full tested commit | `PASS` | full TEST_COMMIT |

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS` | source clone at TEST_COMMIT |
| `make verify` with Go 1.25.13 | `FAIL` | host `go test ./...` fails only `TestProductionDeploymentRejectsInvalidWebsiteTagDate` (`internal/doctest`, POSIX `sh` on Windows; same host issue as rc.9). Other packages including `internal/handoff` PASS |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `FAIL` | same single doctest FAIL; `internal/handoff` race PASS |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `FAIL` | snapshot blocked by WinGet `goreleaser` shim (no associated app / untrusted mount); not a tagged-binary identity defect |
| Phase 1, Phase 2, and Phase 3 regression | `PASS` | R1–R6 PASS on isolated homes |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS` | D10 source hash unchanged; G3 source hash unchanged |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS` | D1–D10 |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS` | G1 inner `10.9 ms`, projection `46541` bytes |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS` | rc.10 iso homes; all five vendor overrides; not rc.9 demo cwd |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS` | disposable `task-alpha` for flagship dest-ack |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS` | dest login proven by size/mtime/LOGIN_OK only; credential values not read |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS` | D8 |
| Vendor configuration changed only inside disposable test-owned homes | `PASS` | isolated Claude pin restore of `2.1.229` exe; Codex project-trust TOML in isolated home only |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS` | this file |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS` | D10; dest sessions created only by live dest launch |

## 5. Required matrix — 44 rows

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `FAIL` | TTY `GetConsoleMode` true via `ssh -tt` + local PTY. Dry-run dest argv `n=1` one-line absolute `projection.md`, no CR/LF. Live Codex dest started; first-reply assistant-only text was 222 characters and did not restate the five acknowledgement bullets. Live exit 1 after dest idle (`outcome unresolved`). Source Claude was not running. |
| A2 | `FAIL` | TTY true. Dry-run dest argv `n=3` with `projection.md` present. Live Claude dest `2.1.229` started; dest session had user/attachment records only (0 assistant characters) before idle kill; live exit 1. |
| A3 | `FAIL` | Not separately collected live. A1 already ran without source vendor API env vars in the test shell; dest-ack still lacked the five-bullet restatement. Missing required live A3 is FAIL. |
| A4 | `PASS` | tagged `partial-final-record` fixture |
| A5 | `FAIL` | Codex dest assistant-only first-reply did not restate goal, constraints, changed files, missing/uncertain, and next action as five bullets. Claude dest produced no assistant first-reply. |
| A6 | `PASS` | harmless MARKER SHA-256 unchanged across dest-ack launches |
| A7 | `PASS` | `rein handoff list --json` shows launched handoffs; Codex destination state `unresolved` (honest) |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `PASS` | unit + installed dry-run and no-launch exit 0 |
| B2 | `PASS` | estimated_bytes=46737 |
| B3 | `PASS` | checkpoint |
| B4 | `PASS` | balanced |
| B5 | `PASS` | full |
| B6 | `PASS` | all fidelity classes present |
| B7 | `PASS` | attachments referenced |
| B8 | `PASS` | unknown records hashed |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `PASS` | dirty tree |
| C2 | `PASS` | workspace-wins |
| C3 | `PASS` | missing leaf exit 5 |
| C4 | `PASS` | wrong-repo exit 5; subdir/nongit/foreign-leaf covered |
| C5 | `PASS` | repo token; no absolute leak |
| C6 | `PASS` | capability warnings exact |
| C7 | `PASS` | skill with C6 |
| C8 | `PASS` | host 2.1.232 no pin fail-closed exit 5 |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `PASS` | injection fixture |
| D2 | `PASS` | fence-breakout |
| D3 | `PASS` | no source-authority dump |
| D4 | `PASS` | show-redactions |
| D5 | `PASS` | launch-free isolated homes |
| D6 | `PASS` | handoffs under isolated REINSTATE_HOME |
| D7 | `PASS` | OpenCode skipped; throwaway vendor homes |
| D8 | `PASS` | no backend; no capsule sync |
| D9 | `PASS` | grok `--no-redact` refused exit 2 |
| D10 | `PASS` | source sha unchanged |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `PASS` | structured-handoff label; banned resume phrasing absent |
| F2 | `PASS` | resume + `--with` no `--last` |
| F3 | `PASS` | `--json` without dry-run/no-launch exit 2 |
| F4 | `PASS` | unknown/wild warning IDs exit 2 |
| F5 | `PASS` | inspect |
| F6 | `PASS` | id length 32 |
| F7 | `PASS` | overlapping native IDs exit 6 |
| F8 | `PASS` | non-TTY launch refuse exit 7 |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | inner unit | n/a | n/a | 10.9 ms | <2000 ms / ≤98304 B | `PASS` | report inner parse/project time, not wrapper `go test` |
| G2 | 1962,1965,1958,1993,1968 ms | 1965 ms | n/a | 1993 ms | 12 s | `PASS` | 1 warmup + 5 timed, all exit 0 |
| G3 | n=100 `--no-launch` | n/a | 66 ms | 68 ms | 4 s | `PASS` | dedicated G3 home; source hash unchanged |

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | yes | `PASS` | Gemini rewind id `gemini:gemini-rewind-win` |
| E2 | yes | `PASS` | same Gemini id |
| E3 | yes (no isolation override) | `NOT TESTED` | OpenCode skip |
| E4 | yes (no isolation override) | `NOT TESTED` | OpenCode skip |
| E5 | yes | `PASS` | Grok→Claude fixture |
| E6 | yes | `PASS` | Grok source |
| E7 | yes | `PASS` | dest grok usage JSON exit 2 |
| E8 | yes | `PASS` | `resume grok:<id>` refused non-zero |

## 7. Architecture §14 closeout

| Assertion | Result | Pointer |
| --------- | ------ | ------- |
| Dry-run and executed structured-plan output are byte-identical | `PASS` | B1 |
| Destination argv never embeds CR/LF; Codex dest falls back to one-line absolute `projection.md` | `PASS` | RC10.argv / A1 dry-run `n=1` |
| Native resume remains same-vendor only | `PASS` | F1 / F2 |
| Cross-agent work is explicit structured handoff | `PASS` | F1; dest-ack content still FAIL (A5) |
| Isolation and BYO storage defaults | `PASS` | D5–D8 |
| No silent transcript translation | `PASS` | product contract |

## 8. Findings and repository hygiene

### Release-blocking

1. Flagship dest-ack first-reply did not restate the five acknowledgement bullets (A1/A2/A3/A5). Product dest argv on Windows is one-line `projection.md` (rc.10 fix held). Failure is dest-loop restatement after live launch, not argv truncation.

### Non-blocking

- Host `go test` / race: single POSIX `sh` doctest on Windows (same as rc.9).
- Snapshot gate: WinGet `goreleaser` shim would not run.
- Isolated Claude pin `claude.exe` had been renamed aside; restored to `2.1.229` before dest-ack. Host PATH Claude remains `2.1.232` (R6/C8 fail-closed PASS).
- Codex dest TUI initially blocked on directory trust; isolated `config.toml` project trust unblocked launch. Dest then produced a short ack, not five bullets.
- Claude dest launched and wrote a new session with attachments only; no assistant restatement before idle kill.

### Test-harness deviations and supersessions

- Live dest-ack used `ssh -tt` with a local PTY so remote `GetConsoleMode` was true. Do not use `05-destack.ps1` (stdout redirect / F8).
- A1.dryrun / A2.dryrun / RC10.argv remain PASS and are not superseded.
- Live A1/A2 recapture on this TEST_COMMIT supersedes uncollected dest-ack; results are FAIL as above.
- Report-only diff is this file. Merge-base with TEST_COMMIT is TEST_COMMIT until the report commit. No product files changed.

### RC / R (reported separately; not in 44-count)

R1–R6 PASS. RC3 empty dest PASS. RC10.argv PASS.

## 9. Required terminated device block

Required counts must sum to 44 and optional counts to 8. This block occurs
exactly once and remains the last report content until reconciliation.

```text
PHASE4-DEVICE-REPORT-V1
device=windows-amd64
test_tag=v0.4.0-rc.10
test_commit=78a00a1bbf6953171d05a5c6680f76f4680ef464
installed_binary_sha256=b2a88ac1e512e5152ad063b6626af3c68c8957b277544b05ca86348685afc65d
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
