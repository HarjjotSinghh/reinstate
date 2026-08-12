# Phase 4 structured-handoff report template

Copy this template into exactly one device-owned report. Keep it cumulative,
sanitized, and append-only: preserve failures and add targeted rechecks as new
evidence. Never commit transcript text, prompts, responses, capsule bodies,
credentials, configuration values, private paths, filenames, diffs, or raw
child-process output.

Allowed report paths for `v0.4.0-rc.1` are:

- `docs/testing/results/REPORT_DATE-macos-phase4-V040RC1.md`; and
- `docs/testing/results/REPORT_DATE-windows-phase4-V040RC1.md`.

Only the macOS coordinator report may add the final reconciliation block after
independently verifying both immutable device-report commits.

## Verdict

- **Device verdict:** `PASS | FAIL`
- **Milestone:** `SETUP | ARTIFACT_VERIFIED | MATRIX_COMPLETE | RECONCILED`
- **Required counts:** `0 PASS / 0 PARTIAL / 0 FAIL / 44 NOT TESTED`
- **Optional source-only counts:** `0 PASS / 0 FAIL / 8 NOT TESTED`
- **Release-blocking findings:** `0`

`PARTIAL` and `NOT TESTED` do not pass a required row. A missing required result
is `FAIL`. Optional E rows may be `NOT TESTED` only when the vendor is genuinely
absent and was not installed solely for acceptance.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `YYYY-MM-DDTHH:MM:SSZ` |
| Device | `macos-arm64` or `windows-amd64` |
| OS/version/build | `<sanitized>` |
| CPU architecture/native process | `<sanitized>` |
| Filesystem | `<name only; no private path>` |
| Tested tag | `v0.4.0-rc.1` |
| Tested full commit | `<40 lowercase hexadecimal characters>` |
| Installed binary SHA-256 | `<64 lowercase hexadecimal characters>` |
| Installed version JSON | `<version, full commit, date; no private path>` |
| Claude Code version/state | `<version and source/destination state>` |
| Codex CLI version/state | `<version and source/destination state>` |
| Gemini/OpenCode/Grok state | `<version or ABSENT for each>` |
| Git version | `<version>` |
| Go version/toolchain | `go1.25.12` |
| Report branch | `test/v0.4.0-rc.1-<platform>-report` |
| Device-report commit | `<full commit containing the terminated device block>` |
| Draft report PR | `<URL or NOT CREATED>` |

## 2. Signed artifact and installer chain

No downloaded executable may run until its tag, checksum, API digest, GitHub
attestation, archive membership, and expected platform identity pass.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Annotated tag signature verifies against `.github/allowed_signers` | `PASS/FAIL` |  |
| Tag peels to the recorded full commit on `origin/main` | `PASS/FAIL` |  |
| Published release is non-draft, prerelease, and tied to the tag | `PASS/FAIL` |  |
| Exact expected release asset set is present | `PASS/FAIL` |  |
| Checksums, GitHub API digests, and attestations match the tag and commit | `PASS/FAIL` |  |
| Archives have safe relative membership, correct platform identity, docs, and SBOMs | `PASS/FAIL` |  |
| Live bootstrap is byte-identical to the tested commit and pins only `v0.4.0-rc.1` | `PASS/FAIL` |  |
| Bootstrap installer digest matches the tagged canonical installer | `PASS/FAIL` |  |
| Fresh isolated install did not replace a user binary or persist PATH changes | `PASS/FAIL` |  |
| `rein` and `reinstate` have identical verified bytes and report `v0.4.0-rc.1` | `PASS/FAIL` |  |
| Installed binary reports the literal full tested commit | `PASS/FAIL` |  |

Any failure here stops product-behavior testing. A source build is supplemental
and never substitutes for the installed tagged artifact.

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean tagged worktree and `go mod tidy -diff` | `PASS/FAIL` |  |
| `make verify` with Go 1.25.12 | `PASS/FAIL` |  |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS/FAIL` |  |
| Required cross-build, fuzz-smoke, snapshot, artifact, and installer gates | `PASS/FAIL` |  |
| Phase 1, Phase 2, and Phase 3 regression | `PASS/FAIL` |  |
| Capsule/projection/CLI goldens are unchanged across repeated runs | `PASS/FAIL` |  |
| Phase 4 adversarial security tests and `make fixture-scan` | `PASS/FAIL` |  |
| `TestLongHistoryParseCapsuleProjectionUnderCeiling`: 400 events, ≤98,304 bytes, <2,000 ms | `PASS/FAIL` |  |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME`, install directory, process homes, and repositories | `PASS/FAIL` |  |
| Fresh controlled source/destination sessions; no older corpus or handoff reused | `PASS/FAIL` |  |
| Operator vendor trees, credentials, keychains, `.env`, and token stores were never read | `PASS/FAIL` |  |
| No backend, passphrase, storage credential, or capsule sync was used | `PASS/FAIL` |  |
| Vendor configuration changed only inside disposable test-owned homes | `PASS/FAIL` |  |
| Reports contain no transcript, prompt, response, secret, private path, filename, diff, capsule body, or raw child error | `PASS/FAIL` |  |
| Source fingerprints were unchanged; vendor-store writes occurred only through an explicitly launched vendor CLI | `PASS/FAIL` |  |

## 5. Required matrix — 44 rows

Use the exact IDs and pass conditions in
[`phase-4-cross-agent-handoff-acceptance.md`](../phase-4-cross-agent-handoff-acceptance.md).

### Matrix A — flagship quota-switch

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| A1 | `NOT TESTED` |  |
| A2 | `NOT TESTED` |  |
| A3 | `NOT TESTED` |  |
| A4 | `NOT TESTED` |  |
| A5 | `NOT TESTED` |  |
| A6 | `NOT TESTED` |  |
| A7 | `NOT TESTED` |  |

### Matrix B — fidelity and policy

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| B1 | `NOT TESTED` |  |
| B2 | `NOT TESTED` |  |
| B3 | `NOT TESTED` |  |
| B4 | `NOT TESTED` |  |
| B5 | `NOT TESTED` |  |
| B6 | `NOT TESTED` |  |
| B7 | `NOT TESTED` |  |
| B8 | `NOT TESTED` |  |

### Matrix C — workspace and capability truth

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| C1 | `NOT TESTED` |  |
| C2 | `NOT TESTED` |  |
| C3 | `NOT TESTED` |  |
| C4 | `NOT TESTED` |  |
| C5 | `NOT TESTED` |  |
| C6 | `NOT TESTED` |  |
| C7 | `NOT TESTED` |  |
| C8 | `NOT TESTED` |  |

### Matrix D — security

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| D1 | `NOT TESTED` |  |
| D2 | `NOT TESTED` |  |
| D3 | `NOT TESTED` |  |
| D4 | `NOT TESTED` |  |
| D5 | `NOT TESTED` |  |
| D6 | `NOT TESTED` |  |
| D7 | `NOT TESTED` |  |
| D8 | `NOT TESTED` |  |
| D9 | `NOT TESTED` |  |
| D10 | `NOT TESTED` |  |

### Matrix F — CLI contract

| ID | Result | Sanitized evidence |
| -- | ------ | ------------------ |
| F1 | `NOT TESTED` |  |
| F2 | `NOT TESTED` |  |
| F3 | `NOT TESTED` |  |
| F4 | `NOT TESTED` |  |
| F5 | `NOT TESTED` |  |
| F6 | `NOT TESTED` |  |
| F7 | `NOT TESTED` |  |
| F8 | `NOT TESTED` |  |

### Matrix G — performance

| ID | Samples | Median | p95 | Maximum | Ceiling | Result | Sanitized evidence |
| -- | ------- | ------ | --- | ------- | ------- | ------ | ------------------ |
| G1 | 1 tagged-source unit run per device |  |  |  | `<2,000 ms`; projection `≤98,304 bytes` | `NOT TESTED` |  |
| G2 | 1 warmup + 5 installed `handoff --dry-run` runs |  |  |  | macOS max `8s`; Windows max `12s` | `NOT TESTED` |  |
| G3 | 1 warmup + 20 installed `handoff list --limit 100 --json` runs |  |  |  | macOS p95 `2s`; Windows p95 `4s` | `NOT TESTED` |  |

For G2 record the controlled source's turn count, source bytes, projection
bytes, truncation count, and selected policy. For G3 prove exactly 100 valid
test-owned lineage rows were returned. Any timeout, failed output validation,
source mutation, unbounded growth, or more than 25 percent comparable same-host
p95 regression is release-blocking even when the absolute ceiling passes.
The development observation of `9.9 ms` and `46,541 bytes` is context only; it
is not device evidence and must not be copied into the measured fields.

## 6. Optional source-only matrix — 8 rows

| ID | Vendor present | Result | Sanitized evidence |
| -- | -------------- | ------ | ------------------ |
| E1 | `yes/no` | `NOT TESTED` |  |
| E2 | `yes/no` | `NOT TESTED` |  |
| E3 | `yes/no` | `NOT TESTED` |  |
| E4 | `yes/no` | `NOT TESTED` |  |
| E5 | `yes/no` | `NOT TESTED` |  |
| E6 | `yes/no` | `NOT TESTED` |  |
| E7 | `yes/no` | `NOT TESTED` |  |
| E8 | `yes/no` | `NOT TESTED` |  |

If any installed optional source row fails, record `FAIL` and treat it as a
candidate blocker. Absence alone is the only permitted `NOT TESTED` reason.

## 7. Architecture §14 closeout

| Definition-of-done assertion | Result | Evidence |
| ---------------------------- | ------ | -------- |
| All 27 packets and their tests are present in the tagged commit | `PASS/FAIL` |  |
| `make verify` is green on both mandatory platforms | `PASS/FAIL` |  |
| Claude → Codex and Codex → Claude work with source closed and no source API call | `PASS/FAIL` | A1–A2 |
| Every fidelity class has real report evidence and byte-stable goldens | `PASS/FAIL` | B6 + automated gates |
| Injection, secret, and bounded-read security gates leak nothing | `PASS/FAIL` | D1–D10 |
| 200-turn projection is bounded and reported | `PASS/FAIL` | B2 + G1 |
| Windows ↔ macOS uses canonical project IDs without source absolute paths | `PASS/FAIL` | C5 |
| Dry-run and executed structured-plan output are byte-identical | `PASS/FAIL` | B1 |
| Required Phase 4 docs, product-truth register, runbook, and dispatch are present | `PASS/FAIL` | tagged tree |
| Exit codes remain 0/1/2/3/5/6/7; no vendor-internal writes or capsule sync | `PASS/FAIL` | D7–D8 + F3–F4 |

## 8. Findings and repository hygiene

### Release-blocking

- None recorded yet.

### Non-blocking

- None recorded yet.

### Test-harness deviations and supersessions

- None recorded yet.

Record the report-only diff, privacy scan result, merge-base with `TEST_COMMIT`,
branch tip, and every superseded row. Never amend or force-push evidence already
shared with the coordinator.

## 9. Required terminated device block

Required counts must sum to 44 and optional counts to 8. This block occurs
exactly once and remains the last report content until reconciliation.

```text
PHASE4-DEVICE-REPORT-V1
device=<macos-arm64|windows-amd64>
test_tag=v0.4.0-rc.1
test_commit=<40-character commit>
installed_binary_sha256=<sha256>
required_pass=<count>
required_partial=<count>
required_fail=<count>
required_not_tested=<count>
optional_pass=<count>
optional_fail=<count>
optional_not_tested=<count>
artifact_chain=PASS|FAIL
isolation_privacy=PASS|FAIL
flagship_directions=PASS|FAIL
fidelity_policy=PASS|FAIL
workspace_capability=PASS|FAIL
security=PASS|FAIL
cli_contract=PASS|FAIL
performance=PASS|FAIL
phase1_phase2_phase3_regression=PASS|FAIL
release_blocking_findings=<count>
product_files_changed=0
secrets_transcripts_or_capsules_committed=false
END-PHASE4-DEVICE-REPORT-V1
```

## 10. Final reconciliation — macOS coordinator only

Append this only after independently verifying both report commits, ancestry,
report-only diffs, arithmetic, privacy scans, original failures, and
supersessions. Passing rc.1 certifies only that candidate; it never authorizes
stable `v0.4.0`.

```text
PHASE4-RC1-FINAL-RECONCILIATION-V1
test_tag=v0.4.0-rc.1
test_commit=<40-character commit>
macos_device_report_commit=<40-character commit>
windows_device_report_commit=<40-character commit>
macos_required_pass=<count>
macos_required_partial=<count>
macos_required_fail=<count>
macos_required_not_tested=<count>
windows_required_pass=<count>
windows_required_partial=<count>
windows_required_fail=<count>
windows_required_not_tested=<count>
macos_release_blocking_findings=<count>
windows_release_blocking_findings=<count>
both_reports_branch_from_test_commit=true|false
both_report_only_diffs=true|false
both_privacy_scans_pass=true|false
phase4_rc1_tagged_artifact_acceptance=PASS|FAIL
stable_v0.4.0_authorized=false
current_stable=v0.3.0
END-PHASE4-RC1-FINAL-RECONCILIATION-V1
```
