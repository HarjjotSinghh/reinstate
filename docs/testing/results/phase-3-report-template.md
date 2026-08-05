# Phase 3 verified-resume report template

Copy this template into exactly one device-owned report. Keep it cumulative,
sanitized, and append-only: preserve failures, add targeted rechecks as new
evidence, and never replace a failed row without recording why the new result
supersedes it.

Allowed report paths for `v0.3.0-rc.1` are:

- `docs/testing/results/REPORT_DATE-macos-phase3-V030RC1.md`; and
- `docs/testing/results/REPORT_DATE-windows-phase3-V030RC1.md`.

Only the macOS coordinator report may add the final reconciliation block after
it has independently verified both immutable device-report commits.

Apple Silicon macOS and native Windows x64 are the supported mandatory
platforms. Intel macOS and WSL2 are unsupported/unverified optional evidence;
they are outside this two-platform report, and their absence or failure does
not block RC1 or stable `v0.3.0`.

## Verdict

- **Device verdict:** `PASS | FAIL`
- **Milestone:** `SETUP | ARTIFACT_VERIFIED | MATRIX_COMPLETE | RECONCILED`
- **Required counts:** `0 PASS / 0 PARTIAL / 0 FAIL / 32 NOT TESTED`
- **Optional physical counts:** `0 PASS / 0 NOT TESTED`
- **Release-blocking findings:** `0`

`PARTIAL` and `NOT TESTED` are not passing results for a required row. A report
with either value in the required matrix has a `FAIL` device verdict.

## 1. Immutable test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `YYYY-MM-DDTHH:MM:SSZ` |
| Device | `macos-arm64` or `windows-amd64` |
| OS/version/build | `<sanitized>` |
| CPU architecture/native process | `<sanitized>` |
| Filesystem | `<name only; no private mount/path>` |
| Tested tag | `v0.3.0-rc.1` |
| Tested full commit | `<40 lowercase hexadecimal characters>` |
| Installed binary SHA-256 | `<64 lowercase hexadecimal characters>` |
| Installed version JSON | `<version, full commit, date; no private path>` |
| Claude Code version/state | `<version and capability state>` |
| Codex CLI version/state | `<version and capability state>` |
| Git version | `<version>` |
| Go version/toolchain | `go1.25.12` |
| Normal-corpus size | `<session/capability counts>` |
| Large-corpus size | `<session/capability counts>` |
| Report branch | `test/v0.3.0-rc.1-<macos-arm64|windows-amd64>-report` |
| Device-report commit | `<full commit containing the terminated device block>` |
| Draft report PR | `<URL or NOT CREATED>` |

## 2. Signed artifact and installer chain

No downloaded executable may run until its tag, checksum, API digest, GitHub
attestation, archive membership, and expected platform identity have passed.

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tag is annotated and SSH signature verifies against `.github/allowed_signers` | `PASS/FAIL` |  |
| Tag peels to the recorded full commit on `origin/main` | `PASS/FAIL` |  |
| Published release is non-draft, prerelease, and tied to the tag | `PASS/FAIL` |  |
| Exact 25-asset release set is present: `checksums.txt` plus 24 checksummed assets | `PASS/FAIL` |  |
| Every checksum and GitHub API asset digest matches | `PASS/FAIL` |  |
| Every release asset, including `checksums.txt`, has valid GitHub attestation provenance for the exact tag and commit | `PASS/FAIL` |  |
| Five platform archives have safe relative membership and the expected binary/documents | `PASS/FAIL` |  |
| Five archive SBOMs and the source archive are present and inspected | `PASS/FAIL` |  |
| Raw binaries and eight Linux native packages are checksummed and attested | `PASS/FAIL` |  |
| Live public bootstrap is byte-identical to the tested commit and pins only `v0.3.0-rc.1` | `PASS/FAIL` |  |
| Bootstrap-pinned canonical installer digest matches the tagged installer | `PASS/FAIL` |  |
| Install used a brand-new isolated `INSTALL_DIR` and did not replace a user binary | `PASS/FAIL` |  |
| Both aliases resolve to identical verified bytes and report version `0.3.0-rc.1` | `PASS/FAIL` |  |
| Installed binary reports the literal full 40-character tested commit | `PASS/FAIL` |  |

Any failure in this section is a mandatory stop before product-behavior rows.
A source build is supplemental automation evidence and never substitutes for
the installed tagged artifact.

## 3. Automated gates

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Clean worktree and `go mod tidy -diff` | `PASS/FAIL` |  |
| `make verify` with pinned Go toolchain and Makefile-owned CGO settings | `PASS/FAIL` |  |
| Complete `CGO_ENABLED=1 go test -race ./... -count=1` | `PASS/FAIL` |  |
| Required four CGO-disabled cross-builds | `PASS/FAIL` |  |
| Five required fuzz-smoke surfaces | `PASS/FAIL` |  |
| Clean GoReleaser snapshot, staged assets, archive/SBOM inspection, and installer smoke | `PASS/FAIL` |  |
| Phase 1 and Phase 2 regression | `PASS/FAIL` |  |

## 4. Isolation and privacy

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Fresh isolated `REINSTATE_HOME` and disposable repositories | `PASS/FAIL` |  |
| Fresh controlled Claude and Codex sessions; no older corpus reused | `PASS/FAIL` |  |
| No `rein init`, credential, passphrase, keyring write, or backend dependency | `PASS/FAIL` |  |
| Only private derived index/baseline state was created with native owner-only protection | `PASS/FAIL` |  |
| No transcript, prompt/response, secret, private absolute path, config value, filename, diff, or raw child error was recorded | `PASS/FAIL` |  |
| Product/vendor configuration was unchanged | `PASS/FAIL` |  |

## 5. Required 32-row matrix

| # | Gate | Result | Sanitized evidence |
| - | ---- | ------ | ------------------ |
| 1 | Exact tag, full commit, installed hash, signature, checksum, attestation, archive, SBOM, and installer provenance | `NOT TESTED` |  |
| 2 | Full verification, race, cross-builds, and Phase 1/2 regression | `NOT TESTED` |  |
| 3 | Fresh configless home with no secret/storage dependency | `NOT TESTED` |  |
| 4 | Fresh controlled Claude and Codex sessions | `NOT TESTED` |  |
| 5 | First inspect is `baseline.unavailable` | `NOT TESTED` |  |
| 6 | Successful verified launch records only a prelaunch-observed baseline | `NOT TESTED` |  |
| 7 | Repeat unchanged report matches repository, branch, HEAD, and tree digest | `NOT TESTED` |  |
| 8 | Different repository at the same path blocks with exit `7` | `NOT TESTED` |  |
| 9 | Missing/non-directory workspace blocks with exit `5` | `NOT TESTED` |  |
| 10 | Branch, detached HEAD, and unborn states remain distinct | `NOT TESTED` |  |
| 11 | Equal/ahead/behind/diverged/unavailable HEAD relations are truthful and offline | `NOT TESTED` |  |
| 12 | Dirty-tree states warn without filenames or diffs | `NOT TESTED` |  |
| 13 | Credential-bearing remotes normalize without leakage | `NOT TESTED` |  |
| 14 | Worktree, symlink, Unicode, case, and native paths are safe | `NOT TESTED` |  |
| 15 | Claude executable/version/layout is fail-closed | `NOT TESTED` |  |
| 16 | Codex executable/version/layout is fail-closed | `NOT TESTED` |  |
| 17 | Instruction presence/change is bounded and content-free | `NOT TESTED` |  |
| 18 | Skill presence/change is bounded, content-free, and does not follow escaping links | `NOT TESTED` |  |
| 19 | MCP reporting is logical-name/transport-only and value-free | `NOT TESTED` |  |
| 20 | Recognized Node/Go declarations and installed versions are safe and truthful | `NOT TESTED` |  |
| 21 | Inspect human/JSON output agrees and never prompts/launches | `NOT TESTED` |  |
| 22 | Native dry-run preserves plan, adds report, and never mutates | `NOT TESTED` |  |
| 23 | TTY warning no/EOF/Ctrl-C refuses; yes launches once | `NOT TESTED` |  |
| 24 | Non-TTY launch requires every exact current warning ID | `NOT TESTED` |  |
| 25 | Unknown/stale/duplicate/wildcard/info/blocker IDs cannot bypass | `NOT TESTED` |  |
| 26 | Hard blockers never prompt and exit precedence is stable | `NOT TESTED` |  |
| 27 | Real same-vendor Claude resume and fork preserve the source | `NOT TESTED` |  |
| 28 | Real same-vendor Codex resume and fork preserve the source | `NOT TESTED` |  |
| 29 | Picker paths and both aliases apply identical policy | `NOT TESTED` |  |
| 30 | Gemini/OpenCode stay read-only with exit `5` | `NOT TESTED` |  |
| 31 | Hostile, timeout, cancellation, stale, race, concurrency, and privacy gates pass | `NOT TESTED` |  |
| 32 | Normal/large, cold/warm latency stays inside the RC1 ceilings | `NOT TESTED` |  |

## 6. Performance evidence

Use three cold full-refresh samples and twenty warm samples for both the normal
and synthetic large corpus. Cold samples must begin with the derived index
absent; warm samples reuse a completed refresh without changing source files.
Record every sample privately and commit only aggregates.

| Corpus/mode | Samples | Median | p95 | Maximum | RC1 ceiling | Result |
| ----------- | ------- | ------ | --- | ------- | ----------- | ------ |
| Normal warm inspect/dry-run | 20 |  |  |  | macOS p95 `2s`; Windows p95 `4s` | `NOT TESTED` |
| Normal cold full refresh | 3 |  |  |  | macOS max `8s`; Windows max `12s` | `NOT TESTED` |
| Large warm inspect/dry-run | 20 |  |  |  | macOS p95 `4s`; Windows p95 `8s` | `NOT TESTED` |
| Large cold full refresh | 3 |  |  |  | macOS max `12s`; Windows max `18s` | `NOT TESTED` |

Also record deterministic command/file counts, timeout count, and any comparable
same-host Phase 3 p95. Any timeout, 20–30 second regression, unbounded growth,
or greater than 25 percent comparable same-host p95 regression is blocking even
when an aggregate happens to fit an absolute ceiling.

## 7. Findings and repository hygiene

### Release-blocking

- None recorded yet.

### Non-blocking

- None recorded yet.

### Test-harness deviations

- None recorded yet.

Record the exact report-only diff, privacy scan result, merge-base with
`TEST_COMMIT`, branch tip, and whether any recheck superseded an earlier row.
Never amend or force-push evidence already handed to the coordinator.

## 8. Required terminated device block

Required counts must sum to 32. The block occurs exactly once and must be the
last content in the device report until an authorized coordinator adds the
separate final reconciliation section.

```text
PHASE3-DEVICE-REPORT-V1
device=<macos-arm64|windows-amd64>
test_tag=v0.3.0-rc.1
test_commit=<40-character commit>
installed_binary_sha256=<sha256>
required_pass=<count>
required_partial=<count>
required_fail=<count>
required_not_tested=<count>
optional_physical_pass=<count>
optional_physical_not_tested=<count>
baseline_provenance=PASS|FAIL
workspace_git=PASS|FAIL
agent_compatibility=PASS|FAIL
capability_privacy=PASS|FAIL
resume_fork=PASS|FAIL
picker=PASS|FAIL
performance=PASS|FAIL
phase1_phase2_regression=PASS|FAIL
release_blocking_findings=<count>
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE3-DEVICE-REPORT-V1
```

## 9. Final reconciliation — macOS coordinator only

Append this only after independently validating both exact device-report
commits, branch ancestry, report-only diffs, count arithmetic, privacy scans,
and all supersessions. Two passing RC1 reports certify tagged-artifact
acceptance only; they never authorize stable `v0.3.0`. A separate stable
promotion decision and fresh tagged-artifact validation on both supported
platforms remain required.

```text
PHASE3-RC1-FINAL-RECONCILIATION-V1
test_tag=v0.3.0-rc.1
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
rc1_tagged_artifact_acceptance=PASS|FAIL
stable_v0.3.0_authorized=false
END-PHASE3-RC1-FINAL-RECONCILIATION-V1
```
