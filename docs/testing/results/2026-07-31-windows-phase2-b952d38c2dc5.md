# Phase 2 acceptance — native-Windows Device B report

**Verdict:** `FAIL`

**Milestone:** `DEVICE_COMPLETE`

**Required counts:** `23 PASS / 1 PARTIAL / 6 FAIL / 0 NOT TESTED`

**Optional physical counts:** `2 PASS / 0 NOT TESTED`

This report covers only the exact disposable sessions and paths created for
this run. No evidence was reused from `5c60ec2` or another product commit. No
real transcript content, authentication material, secret, credential,
passphrase, preserved OpenCode database backup, or unrelated session row was
used as evidence. Device B is complete, but Phase 2 remains pending the Mac
coordinator's cross-device reconciliation.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-07-31T15:28:22Z` through `2026-07-31T16:44:00Z` |
| Device | native-Windows Device B |
| Tested Git commit | `b952d38c2dc57b0a96bc696860318ea7c2975800` |
| Signed tag, if any | None; development acceptance |
| Reinstate version JSON | `{"name":"reinstate","version":"v0.1.0-39-gb952d38","commit":"b952d38c2dc57b0a96bc696860318ea7c2975800","date":"2026-07-31T15:28:41Z"}` |
| OS/version/build | Microsoft Windows 11 Pro `10.0.26200`, build `26200` |
| Architecture | 64-bit OS, native `amd64` process |
| Native shell | Windows PowerShell Desktop `5.1.26100.8328` |
| Claude Code version/state | `2.1.220`; installed; controlled resume physically exercised |
| Codex CLI version/state | `codex-cli 0.146.0`; installed; controlled resume launch failed to complete |
| Gemini CLI version/state | `0.53.0`; installed; controlled read-only path passed |
| OpenCode version/state | `1.18.2`; installed; controlled read-only path and timestamp regression passed |
| Git version | `2.52.0.windows.1` |
| Go version | host `go1.26.1 windows/amd64`; gates used `GOTOOLCHAIN=go1.25.12` |
| Report branch | `test/phase2-b952d38c2dc5-windows-report` |
| Draft PR | Pending creation after the first report-only push |

Provenance: the tested commit was fetched without merge, is reachable from
`origin/fix/opencode-top-level-timestamps`, is not reachable from
`origin/main`, and has no tag. The tip of the source branch later contained
only the prompt-document commit; product behavior was tested strictly at the
earlier requested product commit.

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | PASS | clean worktree `HEAD` exactly matched the full requested SHA |
| Binary reports the tested commit | PASS | `rein version --json` returned the full SHA; source-build SHA-256 `69C103248A6D6B0E916A0DD5437C7FE88BAB2DD220210D2ADF25BA33B6693C38` |
| Product tree was clean before testing | PASS | `git status --short` was empty before the first build and physical gate |
| Report branch starts at the tested commit | PASS | branch was created directly from the requested product SHA |
| Report is the only committed change | PENDING W4 FINALIZATION | verified before staging; final commit/push proof is added after PR creation |
| No secret/transcript/private path was committed | PENDING W4 FINALIZATION | report privacy scan and one-file diff required before push |

## 3. Isolation and local-only proof

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | PASS | target did not exist before the first Phase 2 command |
| No `rein init` run | PASS | no init command was executed |
| No `config.toml` or sync state created | PASS | config, state, and backup paths stayed absent |
| No credential/passphrase/keyring request | PASS | none occurred |
| No backend/network dependency | PASS | every Reinstate local command ran without a sync backend |
| Only derived index state created | PASS | only `cache/session-index-v1.sqlite` appeared |
| Index and parent permissions are owner-only | PASS | effective Windows ACL principals were owner, SYSTEM, and Administrators; no other or broad-read principal existed |

No `R2.txt`, storage coordinate, profile, credential reference, passphrase, or
keyring entry was created or used. The preserved pre-rebuild OpenCode database
backup was neither touched nor inspected.

## 4. Controlled corpus

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | `claude:2a05fa82-d17a-45f1-94c9-b1698ec63f25` | alpha | true | full |
| Codex | `codex:019fb8d0-a4ef-7563-bf8b-873ba5a5cd01` | unicode-beta | true | full |
| Gemini | `gemini:6e646c8c-e738-4fca-ae1e-b37220c25247` | alpha | true | read-only |
| OpenCode | `opencode:ses_0471c26a5ffehy6apVm000Oopr` | unicode-beta | true | read-only |

Initial Claude, Codex, and Gemini commands returned exact controlled responses.
The successful OpenCode retry used the installed CLI's explicitly listed free
model `opencode/deepseek-v4-flash-free`, returned the exact controlled marker,
and left no controlled process running.

## 5. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| OpenCode regression test first | PASS | `go test ./internal/sessionindex -run OpenCode -count=1`, exit `0` |
| Focused session-index/CLI/adapter tests | PASS | all requested focused packages passed, exit `0` |
| Formatting | PARTIAL | `gofmt -l .` returned zero files; exact `make fmt-check` could not run because `make` is absent |
| Full Go test suite | FAIL | `go test ./... -count=1`, exit `1`, three failures listed below |
| Race suite | FAIL | default command exit `2` because CGO was disabled; retry with `CGO_ENABLED=1` exit `1` because `gcc` is absent |
| Vet | PARTIAL | `go vet ./...` passed; exact `make vet` exit `1` because `make` is absent |
| Pinned lint | PASS | `golangci-lint v2.11.4`, zero issues |
| Pinned vulnerability scan | PASS | `govulncheck v1.6.0`, zero called vulnerabilities |
| Complete merge gate | FAIL | exact `make verify`, exit `1`, because `make` is absent |
| Required cross-builds | PASS | Windows amd64, Darwin arm64, Darwin amd64, and Linux amd64 with `CGO_ENABLED=0` all exited `0` |
| Phase 1 regression | FAIL | the full suite was not green |

Exact automated failures:

- `make fmt-check` — exit `1`: `make` is not recognized.
- `go test ./... -count=1` — exit `1`:
  `TestProductionDeploymentRejectsInvalidWebsiteTagDate`,
  `TestVerifyAvoidsRedundantDoctestRuns`, and
  `TestQuickGateStaysFocusedAndNonRelease` failed. The latter two explicitly
  reported `exec: "make": executable file not found in %PATH%`.
- `make vet` — exit `1`: `make` is not recognized. Direct `go vet ./...`
  passed.
- `go test -race ./... -count=1` — exit `2`: race requires CGO. With
  `CGO_ENABLED=1`, exit `1`: C compiler `gcc` was not found.
- `make verify` — exit `1`: `make` is not recognized.

## 6. Configless index and refresh

- Fresh-home `rein sessions`, `rein sessions --json`,
  `reinstate sessions --json`, and `rein list --help` all exited `0`.
- Exact controlled Claude and Codex references appeared in the unfiltered
  default result. Gemini also appeared after its newly created session.
- Repeated refreshes kept one stable identity for all four controlled records.
- A no-change refresh produced byte-equivalent controlled records for Claude,
  Codex, Gemini, and OpenCode.
- OpenCode became the newest controlled record after its creation.
- A vendor-native Claude append produced a new controlled marker; refresh found
  it and then remained idempotent.
- Reinstate indexing, search, inspect, dry-runs, and refusals did not change the
  controlled vendor sources. Only vendor launches changed their own sources.
- `rein` and `reinstate` sequential JSON differed in one unrelated, actively
  writing Codex row (`updated_at` and `size_bytes`); every controlled record and
  all warnings were equivalent. Row 6 is therefore `PARTIAL`, not `PASS`.
- A deliberately concurrent alias check made one process fail with
  `SQLITE_BUSY`; it is a harness deviation and not used as alias evidence.

## 7. Search and inspect

- Controlled prompt-fragment search found the exact Claude and Codex records.
- Agent, project, branch, and post-append file filters selected the intended
  controlled records.
- Multiple AND terms, case-insensitive matching, limit, Unicode beta, and a
  deliberate shell-metacharacter zero-match all behaved literally.
- The metacharacter zero-match returned an honest empty `sessions` value and
  exit `0`.
- `sessions` and `search` exposed metadata only; neither schema contained a
  transcript-passage or full-prompt field.
- Claude and Codex inspect returned accurate identity, project/workspace,
  branch, capability, and message metadata. Each controlled preview was exactly
  160 Unicode code points and came from the controlled user prompt.
- Controlled sources remained byte-identical across index/search/inspect.

## 8. Last, resume, and fork

All requested dry-runs exited `0`, started no agent, and changed no controlled
source. Exact plans were:

| Gate | Executable | Args | CWD |
| ---- | ---------- | ---- | --- |
| Claude resume | `claude` | `--resume`, exact native ID | alpha project |
| Codex resume | `codex` | `resume`, exact native ID | unicode-beta project |
| Claude fork | `claude` | `--resume`, exact native ID, `--fork-session` | alpha project |
| Codex fork | `codex` | `fork`, exact native ID | unicode-beta project |

`last --project` selected the correct controlled Claude and Codex records.
Global and agent-filter dry-runs also exited `0`.

Physical results:

- Claude resume: `PASS`. The exact controlled source gained one controlled user
  marker event and one assistant event whose only text exactly matched the
  challenge marker. The exact vendor process was seen. The interactive vendor
  ignored synthetic exit keys, so the verified disposable process chain was
  terminated after evidence capture.
- Codex resume: `FAIL`. The exact `codex.exe` child launched and held the
  controlled rollout, but produced no challenge marker after the bounded run
  plus an extra minute. No normal child exit was captured; the exact verified
  disposable chain was terminated and left no process.
- Claude fork: `FAIL`. The exact same-vendor fork command launched and preserved
  the source byte-for-byte, but no distinct controlled fork marker/session
  appeared after the bounded run plus an extra minute. No normal child exit was
  captured; the exact chain was terminated.
- Codex fork: `FAIL`. The real fork was not run after the blocking Codex resume
  failure. Its exact dry-run plan passed; a required unrun physical gate is not
  relabeled `PASS`.
- Missing reference failed safely with exit `2` and no launch.
- JSON without dry-run was refused for resume, fork, and last with exit `2`.
- Gemini and OpenCode resume/fork each refused with exit `5`, no controlled
  process, and no core vendor metadata mutation.
- Ambiguity, missing executable/workspace, cancellation, and child-failure
  propagation passed the focused deterministic test suite; destructive
  physical manufacture was not used.

## 9. Interactive switcher

The real Windows Terminal path used a fresh static title with application-title
override suppression. Bare `rein` launched a real TTY picker, but the exact
static-title input driver could not prove that filter/inspect/invalid/cancel
line input landed, and `q` did not terminate the picker. The exact disposable
`rein.exe` and PowerShell chain was terminated. No picker mutation was
attempted afterward, and bare `reinstate` was not falsely called tested.

Result: row 26 `FAIL`.

Non-TTY behavior passed independently for both binary names: piped empty input
returned promptly with exit `2` and included the `rein sessions --json` hint.

## 10. Read-only adapters

### Gemini

- Controlled filter, literal ID search, and inspect all exited `0` and selected
  `gemini:6e646c8c-e738-4fca-ae1e-b37220c25247`.
- Inspect advertised neither resume nor fork.
- Resume and fork each refused with exit `5`.
- No controlled Gemini process remained.

### OpenCode row 29 regression

- Controlled session:
  `opencode:ses_0471c26a5ffehy6apVm000Oopr`.
- Vendor top-level fields were exactly `id`, `title`, `updated`, `created`,
  `projectId`, and `directory`.
- Vendor top-level `updated` was `1785513374388` milliseconds since epoch.
- Reinstate `updated_at` was `2026-07-31T15:56:14Z`, non-zero and not year 1.
- The index stores whole seconds: `1785513374000` milliseconds. It exactly
  matched the vendor top-level `updated` epoch at whole-second precision, so
  provenance is the top-level field rather than a year-1 fallback.
- The controlled record appeared once in unfiltered default
  `rein sessions --json` without `--agent opencode` when run from its controlled
  OpenCode workspace.
- Explicit `--agent opencode`, literal native-ID search, and inspect each found
  the exact controlled reference.
- Inspect advertised read-only capability; resume and fork each refused with
  exit `5`.
- Core vendor fields `id`, `title`, `updated`, `created`, and `projectId`
  remained unchanged; no controlled OpenCode process remained.
- OpenCode's supported session list is workspace-scoped on this installation.
  The controlled record was absent when Reinstate ran from the report worktree
  and present from the controlled OpenCode workspace. This limitation is
  recorded; the required unfiltered default listing proof was produced in the
  correct controlled workspace.

The first OpenCode default-model attempt timed out after 300 seconds and left
two controlled child processes. Both were revalidated by unique controlled
marker/ID and exact parent chain, then terminated. Its transient session was not
used. The successful fresh retry used an explicitly listed free model and is
the only OpenCode evidence used for row 29.

## 11. Mandatory matrix

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | — | PASS — sections 1-2 |
| 2 | Full local verification and required cross-builds | — | FAIL — section 5 |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | — | PASS — section 3 |
| 4 | `rein sessions` discovers exact Claude sessions | — | PASS — sections 4, 6 |
| 5 | `rein sessions` discovers exact Codex sessions | — | PASS — sections 4, 6 |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | — | PARTIAL — section 6 |
| 7 | Derived index path, rebuild, idempotency, and private permissions | — | PASS — sections 3, 6 |
| 8 | Prompt-fragment literal search | — | PASS — section 7 |
| 9 | Agent filter | — | PASS — sections 7, 10 |
| 10 | Project filter | — | PASS — section 7 |
| 11 | Branch filter | — | PASS — section 7 |
| 12 | File filter | — | PASS — section 7 |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | — | PASS — section 7 |
| 14 | `sessions` and `search` do not dump transcript passages | — | PASS — section 7 |
| 15 | `inspect` metadata/160-code-point user preview policy | — | PASS — section 7 |
| 16 | Append/new-session refresh and no-change idempotency | — | PASS — section 6 |
| 17 | `last` selects the correct resumable session and filters | — | PASS — section 8 |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | — | PASS — section 8 |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | — | PASS — section 8 |
| 20 | Claude native resume | — | PASS — section 8 |
| 21 | Codex native resume | — | FAIL — section 8 |
| 22 | Claude vendor-native fork, source preserved | — | FAIL — section 8 |
| 23 | Codex vendor-native fork, source preserved | — | FAIL — section 8 |
| 24 | Missing/ambiguous reference and missing executor fail safely | — | PASS — sections 5, 8 |
| 25 | JSON/native-child separation and child failure propagation | — | PASS — sections 5, 8 |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | — | FAIL — section 9 |
| 27 | Non-TTY prompt failure is immediate and actionable | — | PASS — section 9 |
| 28 | Gemini read-only physical path, when installed | — | PASS — section 10 |
| 29 | OpenCode read-only physical path, when installed | — | PASS — section 10 |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | — | PASS — sections 8, 10 |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | — | PASS — section 5 |
| 32 | Phase 1 automated regression remains green | — | FAIL — section 5 |

## 12. Findings

### Release-blocking

1. Required full verification and merge-gate evidence is red: `make` and `gcc`
   are absent, the full suite has three failures, race cannot build, and
   `make verify` cannot run.
2. Full alias JSON parity is only partial because one unrelated active Codex
   source changed between sequential scans.
3. Codex native resume did not produce a challenge response.
4. Claude native fork did not produce a distinct controlled fork; Codex native
   fork was not run after its blocking resume failure.
5. The real TTY picker interaction matrix could not be proven and the picker
   did not accept the synthetic `q` path.

### Non-blocking

1. OpenCode retains millisecond vendor epochs while the Reinstate index exposes
   whole-second precision. The indexed second still comes exactly from the
   top-level `updated` epoch.
2. OpenCode physical discovery is workspace-scoped on this installation; row
   29 passed from the controlled OpenCode workspace.

### Test-harness deviations

- Two long-running lint/vulnerability wrappers initially had too-short outer
  timeouts; their read-only gates were rerun successfully.
- Several compact PowerShell summary wrappers had parser/return-object errors.
  Product commands not reached by those wrappers were rerun directly; no
  wrapper-only result was used as evidence.
- The first OpenCode command timed out and its exact controlled children were
  terminated before the successful fresh retry.
- Computer-use initialization was unavailable because its native pipe did not
  exist. Windows Terminal was used for native PTY attempts.
- One abandoned PID-targeted Windows Terminal attempt could have directed a
  harmless controlled marker to another terminal tab. No secret or private
  data was involved; PID targeting was immediately retired in favor of exact
  static titles.
- Interactive vendors ignored synthetic exit keys. Exact controlled process
  trees were revalidated before termination; no unrelated process was stopped.

## 13. Repository hygiene

- report-only branch: `test/phase2-b952d38c2dc5-windows-report`
- tested base commit: `b952d38c2dc57b0a96bc696860318ea7c2975800`
- changed files: this report only
- private/local artifacts excluded: isolated home, SQLite index, build
  artifacts, cross-builds, raw logs, PTY helpers, disposable projects, and
  controlled vendor files
- product code unchanged: true
- secrets/transcripts committed: `false`

Final W4 verification must prove one report file in the staged and
merge-base-to-tip diff, a clean privacy scan, a draft/unmerged PR, and exact
local/remote branch parity.

## 14. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=windows
test_commit=b952d38c2dc57b0a96bc696860318ea7c2975800
reinstate_version=v0.1.0-39-gb952d38
report_path=docs/testing/results/2026-07-31-windows-phase2-b952d38c2dc5.md
report_branch=test/phase2-b952d38c2dc5-windows-report
claude_ref=claude:2a05fa82-d17a-45f1-94c9-b1698ec63f25
codex_ref=codex:019fb8d0-a4ef-7563-bf8b-873ba5a5cd01
gemini_state=PASS
opencode_state=PASS
required_pass=23
required_partial=1
required_fail=6
required_not_tested=0
optional_physical_pass=2
optional_physical_not_tested=0
configless_local_only=PASS
preview_privacy=PASS
claude_resume_fork=FAIL
codex_resume_fork=FAIL
picker=FAIL
phase1_regression=FAIL
release_blocking_findings=5
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```
