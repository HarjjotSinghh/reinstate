# Phase 2 Local Index Acceptance Report Template

Copy this file to:

```text
docs/testing/results/YYYY-MM-DD-macos-phase2-TEST_ID.md
docs/testing/results/YYYY-MM-DD-windows-phase2-TEST_ID.md
```

Replace every placeholder. Delete instructional comments before committing.
The report is cumulative and sanitized. It must not contain a username,
absolute home path, transcript prose, assistant reasoning/messages, tool
output, environment dumps, auth material, storage credentials, passphrases, or
unrelated session metadata.

# Phase 2 acceptance — DEVICE report

**Verdict:** `PASS | PARTIAL | FAIL | NOT TESTED`
**Milestone:** `DEVICE_COMPLETE | FINAL_RECONCILIATION`
**Required counts:** `N PASS / N PARTIAL / N FAIL / N NOT TESTED`
**Optional physical counts:** `N PASS / N NOT TESTED`

This report covers only the exact disposable sessions and paths created for
this run. No real transcript content or secret was used as evidence.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | |
| Device | `macOS Device A | native-Windows Device B` |
| Tested Git commit | |
| Signed tag, if any | |
| Reinstate version JSON | |
| OS/version/build | |
| Architecture | |
| Native shell | |
| Claude Code version/state | |
| Codex CLI version/state | |
| Gemini CLI version/state | |
| OpenCode version/state | |
| Git version | |
| Go version | |
| Report branch | |
| Draft PR | |

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | | |
| Binary reports the tested commit | | |
| Product tree was clean before testing | | |
| Report branch starts at the tested commit | | |
| Report is the only committed change | | |
| No secret/transcript/private path was committed | | |

## 3. Isolation and local-only proof

Record booleans and redacted relative paths only.

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | | |
| No `rein init` run | | |
| No `config.toml` or sync state created | | |
| No credential/passphrase/keyring request | | |
| No backend/network dependency | | |
| Only derived index state created | | |
| Index and parent permissions are owner-only | | |

## 4. Controlled corpus

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | | | | full |
| Codex | | | | full |
| Gemini | | | | read-only / not installed |
| OpenCode | | | | read-only / not installed |

Do not include full prompts, full responses, transcript excerpts, absolute
paths, or unrelated session rows.

## 5. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Focused session-index/CLI/adapter tests | | |
| Full Go test suite | | |
| Race suite | | |
| Vet / `make verify` | | |
| Required cross-builds | | |
| Phase 1 regression | | |

For each failure, include the exact command, exit code, and sanitized error.

## 6. Configless index and refresh

Record `rein sessions`, alias parity, deterministic ordering, stable composite
identity, no-change idempotency, append refresh, new-session refresh, corrupt
index rebuild, and owner-only permissions.

## 7. Search and inspect

Record results for prompt, agent, project, branch, file, AND terms, limit,
case-insensitive matching, Unicode, and zero match. State explicitly whether
`sessions`, `search`, and `inspect` obeyed the preview policy.

## 8. Last, resume, and fork

Record dry-run executable/argv/cwd fields before real launches. Record exact
same-vendor resume/fork results, source-preservation booleans, missing and
ambiguous reference behavior, JSON/native-child separation, and child exit
propagation.

## 9. Interactive switcher

Record TTY/console type and results for filter, inspect, resume, fork, invalid
input, cancel/EOF/interrupt, and non-TTY invocation. Do not include a
screenshot or full picker output if it exposes unrelated session metadata.

## 10. Read-only adapters

Record Gemini/OpenCode discovery/search/inspect when installed, or
`NOT_INSTALLED` and `NOT TESTED` for the optional physical row. Fixture-backed
automated evidence is still required. Record exit `5` refusal and zero launch
or mutation for resume/fork.

## 11. Mandatory matrix

Copy the 32-row table from
`docs/testing/phase-2-local-index-acceptance.md`, fill the current device
column, and cite this report's section for every result. Do not mark peer-device
work as local evidence.

## 12. Findings

### Release-blocking

List findings or write `None`.

### Non-blocking

List findings or write `None`.

### Test-harness deviations

Record retries, unavailable vendors, and deviations. A deviation cannot be
silently converted to `PASS`.

## 13. Repository hygiene

- report-only branch:
- tested base commit:
- changed files:
- private/local artifacts excluded:
- product code unchanged:
- secrets/transcripts committed: `false`

## 14. Device milestone block

Use one block. Values are machine-parseable single-line fields.

```text
PHASE2-DEVICE-REPORT-V1
device=macos|windows
test_commit=<full-sha>
reinstate_version=<version>
report_path=<repo-relative-path>
report_branch=<report-only-branch>
claude_ref=<composite-reference>
codex_ref=<composite-reference>
gemini_state=PASS|NOT_INSTALLED|FAIL
opencode_state=PASS|NOT_INSTALLED|FAIL
required_pass=<integer>
required_partial=<integer>
required_fail=<integer>
required_not_tested=<integer>
optional_physical_pass=<integer>
optional_physical_not_tested=<integer>
configless_local_only=PASS|FAIL
preview_privacy=PASS|FAIL
claude_resume_fork=PASS|FAIL
codex_resume_fork=PASS|FAIL
picker=PASS|FAIL
phase1_regression=PASS|FAIL
release_blocking_findings=<integer>
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```

## 15. Final reconciliation block

Only the Mac coordinator adds this after validating the complete Windows
report, both branch tips, and both report-only diffs.

```text
PHASE2-FINAL-RECONCILIATION-V1
test_commit=<full-sha>
mac_report_commit=<full-sha>
windows_report_commit=<full-sha>
mandatory_rows=32
required_dual_device_rows_passed=<integer>
optional_rows_28_29_macos=PASS|NOT_TESTED|FAIL
optional_rows_28_29_windows=PASS|NOT_TESTED|FAIL
automated_gates=PASS|FAIL
physical_macos=PASS|FAIL
physical_windows=PASS|FAIL
release_blocking_findings=<integer>
phase2_status=PASS|FAIL
END-PHASE2-FINAL-RECONCILIATION-V1
```
