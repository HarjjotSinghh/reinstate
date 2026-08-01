# Phase 2 acceptance — native-Windows Device B report

**Verdict:** `PASS`

**Milestone:** `DEVICE_COMPLETE`

**Required counts:** `30 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`

**Optional physical counts:** `2 PASS / 0 NOT TESTED`

This report covers only the exact disposable sessions and paths created for
this run. No evidence was reused from `5c60ec2` or another product commit. No
real transcript content, authentication material, secret, credential,
passphrase, preserved OpenCode database backup, or unrelated session row was
used as evidence. Device B is complete, but Phase 2 remains pending the Mac
coordinator's cross-device reconciliation.

The original `23 PASS / 1 PARTIAL / 6 FAIL` evidence is retained below. Dated
targeted rechecks on the unchanged product commit supersede rows 2, 6, 21-23,
26, and 32, producing the current all-green Windows matrix.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | original run `2026-07-31T15:28:22Z` through `2026-07-31T16:44:00Z`; targeted rechecks through `2026-08-01T06:59:22Z` |
| Device | native-Windows Device B |
| Tested Git commit | `b952d38c2dc57b0a96bc696860318ea7c2975800` |
| Signed tag, if any | None; development acceptance |
| Reinstate version JSON | `{"name":"reinstate","version":"v0.1.0-39-gb952d38","commit":"b952d38c2dc57b0a96bc696860318ea7c2975800","date":"2026-07-31T15:28:41Z"}` |
| OS/version/build | Microsoft Windows 11 Pro `10.0.26200`, build `26200` |
| Architecture | 64-bit OS, native `amd64` process |
| Native shell | Windows PowerShell Desktop `5.1.26100.8328` |
| Claude Code version/state | `2.1.220`; installed; controlled resume and fork physically passed |
| Codex CLI version/state | `codex-cli 0.146.0`; installed; controlled resume and fork physically passed |
| Gemini CLI version/state | `0.53.0`; installed; controlled read-only path passed |
| OpenCode version/state | `1.18.2`; installed; controlled read-only path and timestamp regression passed |
| Git version | `2.52.0.windows.1` |
| Go version | host `go1.26.1 windows/amd64`; gates used `GOTOOLCHAIN=go1.25.12` |
| Report branch | `test/phase2-b952d38c2dc5-windows-report` |
| Draft PR | [#71](https://github.com/HarjjotSinghh/reinstate/pull/71), draft and unmerged |

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
| Report is the only committed change | PASS | staged, commit, merge-base-to-tip, and PR file lists contain this report only |
| No secret/transcript/private path was committed | PASS | staged and final report privacy scans found no private path, username, token/key pattern, or transcript content |

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

### Original run — 2026-07-31 (preserved)

The following table and errors are the original W1 evidence. They remain here
unchanged in substance; the later targeted recheck supersedes their row-2 and
row-32 verdicts.

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

### Targeted recheck — 2026-08-01 (current verdict)

The recheck used a fresh process-scoped Windows PowerShell environment and the
already-installed MSYS2 toolchain; nothing was installed or added to the
machine-wide or user `PATH`:

```powershell
$env:Path = 'C:\msys64\usr\bin;C:\msys64\mingw64\bin;' + $env:Path
$env:GOTOOLCHAIN = 'go1.25.12'
$env:CGO_ENABLED = '1'
make verify
go test -race ./... -count=1
```

- Exact product worktree `HEAD` was
  `b952d38c2dc57b0a96bc696860318ea7c2975800` and clean.
- `make verify` exited `0` with `verify ok`. Its sanitized 68-line log SHA-256
  is `5579A496A1EFCA75B7D112C46C994397D4153E697E981D42937C58D9E0B6DE16`.
- `go test -race ./... -count=1` exited `0` across every package. Its
  sanitized 25-line log SHA-256 is
  `7F16C2194407F1E967131F4B09FCD06A5E465AB01369D79E843549B664A54BBD`.
- MSYS2 emitted a non-fatal `tee: /dev/stderr: No such file or directory`
  compatibility warning during `make verify`; the command still completed all
  gates and exited `0`.
- The earlier rejected recheck against report commit `45fd25e` is preserved
  outside the repository but is not product evidence because its build stamp
  was not the requested product SHA.

Current result: rows 2 and 32 are `PASS`. No installation is required on this
device; the minimal safe remediation was the process-scoped `PATH` plus the
explicit Go/CGO variables above.

## 6. Configless index and refresh

### Original run — 2026-07-31 (preserved)

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

### Delayed targeted alias recheck — 2026-08-01 (current verdict)

The active Codex session was correctly treated as a moving vendor source. A
disposable PowerShell checker outside the repository waited for its exact
watched PID to exit, then ran the aliases sequentially, never concurrently:

```powershell
rein sessions --json
reinstate sessions --json
```

The checker stored no raw session JSON or transcript content. It retained only
sanitized comparison counts, ordered warning metadata, and hashes.

- Attempt 1 is preserved but rejected as parity evidence: a new Codex process
  began during the 60-second quiescence interval after the watched process
  exited.
- Attempt 2 completed before the next Codex process started. Both commands
  exited `0`, both outputs parsed, and each contained 100 sessions and seven
  warnings.
- Complete normalized JSON, ordered sessions, ordered warnings, and ordering
  were all exactly equal. Each corresponding pair of SHA-256 hashes matched.
- The sanitized attempt-2 result SHA-256 is
  `4C2D87DF04CA828CE4E41A40A6C601D5CF64A595150E860A3B3FE226B4AD6BA9`.
- The first alias completed before the second began; no concurrent alias
  process existed, no source/product mutation occurred, and no raw JSON was
  stored.

Current result: row 6 is `PASS`. The original `PARTIAL` was a moving-source
harness limitation, not a reproducible Reinstate parity defect.

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

### Original physical attempt — 2026-07-31 (preserved)

Physical results from the original run:

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

### Human-keyboard targeted recheck — 2026-08-01 (current verdict)

All rechecks ran in fresh real Windows Terminal windows with human keyboard
input. Automated byte injection, ConPTY byte drivers, `AppActivate`, and
`SendKeys` were not used. Clean child environments removed only the inherited
nesting-marker names `CODEX_THREAD_ID` and `CLAUDECODE`; their values and all
authentication material were never printed or inspected. Before every launch,
the exact source fingerprint and clean controlled process state were checked,
and Reinstate dry-run JSON re-proved executable, argv, CWD, and native ID.

- **Codex resume — row 21 `PASS`.** Fresh source
  `codex:019fb9d2-c35c-7142-8e7f-a6dd97b58109` launched as
  `codex resume 019fb9d2-c35c-7142-8e7f-a6dd97b58109` through the exact
  `PowerShell -> rein -> codex` chain. One exact controlled user challenge and
  one exact assistant response were recorded. Reinstate exited `0`; refreshed
  discovery retained exactly one source identity; the independent resume
  dry-run remained exact; zero controlled processes remained.
- **Claude vendor-native fork — row 22 `PASS`.** Fresh source
  `claude:94957a40-15b9-4de1-975b-bc99e6478ba4` launched as
  `claude --resume 94957a40-15b9-4de1-975b-bc99e6478ba4 --fork-session`
  through `PowerShell -> rein -> cmd -> claude`. The source SHA-256 remained
  `C623F76B89762A8728F7A2AB9ED34420CFD5317269F43B2F1BF2C83DEBFAF940`.
  Fresh fork `claude:8ab38622-946c-412a-a103-a291b40bd2a0` recorded the exact
  challenge/response, was separately rediscovered and inspected, and then
  completed a second physical resume with a fresh exact response. Both vendor
  runs exited `0`; zero controlled processes remained.
- **Codex vendor-native fork — row 23 `PASS`.** Fresh source
  `codex:019fbb8a-4e6e-71f2-8138-b9e3f77a1985` launched as
  `codex fork 019fbb8a-4e6e-71f2-8138-b9e3f77a1985` through the exact
  `PowerShell -> rein -> codex` chain. The source remained byte-identical.
  Fresh rollout `codex:019fbb9f-8b44-76c3-a130-50e3f92dc3a7` contained both
  source and fork `session_meta` events, but refresh correctly pinned identity
  to the fork filename: source and fork each appeared exactly once as distinct
  resumable/forkable records. The fork was inspected and physically resumed
  independently with an exact fresh challenge/response. Both runs exited `0`;
  zero controlled processes remained.
- Every controlled process was matched by exact native ID and ancestry before
  observation. No process was terminated during these successful rechecks;
  documented vendor exit commands produced normal exit `0` throughout.

The correctly equipped physical recheck did not reproduce a Reinstate defect.
Rows 21-23 now pass on the unchanged product commit.

## 9. Interactive switcher

### Original attempt — 2026-07-31 (preserved)

The real Windows Terminal path used a fresh static title with application-title
override suppression. Bare `rein` launched a real TTY picker, but the exact
static-title input driver could not prove that filter/inspect/invalid/cancel
line input landed, and `q` did not terminate the picker. The exact disposable
`rein.exe` and PowerShell chain was terminated. No picker mutation was
attempted afterward, and bare `reinstate` was not falsely called tested.

Original result: row 26 `FAIL`.

Non-TTY behavior passed independently for both binary names: piped empty input
returned promptly with exit `2` and included the `rein sessions --json` hint.

### Human-keyboard targeted recheck — 2026-08-01 (current verdict)

The earlier targeted triage evidence for picker filter, inspect, invalid input,
`q`, EOF/interrupt, both bare `rein` and bare `reinstate`, read-only routing,
and the non-TTY refusal was retained. The missing capable-vendor routes were
then exercised in real red-tabbed Windows Terminal windows with human keyboard
input only:

- **Picker resume:** the exact query
  `REINSTATE-B952-PHASEC-CODEX-FORK-INDEPENDENT-RESUME-20260801T045341Z`
  returned one row, exactly
  `codex:019fbb9f-8b44-76c3-a130-50e3f92dc3a7`. Before selection no vendor
  child existed and the source fingerprint matched. Human input `1` launched
  exactly one `codex resume` child under the picker `rein.exe`; an exact fresh
  challenge/response was recorded. Reinstate exited `0`, the project remained
  clean, and zero controlled processes remained.
- **Picker fork:** the exact query
  `REINSTATE-B952-PHASEC-CLAUDE-FORK-INDEPENDENT-RESUME-20260801T041020Z`
  returned one row, exactly
  `claude:8ab38622-946c-412a-a103-a291b40bd2a0`. Before selection no vendor
  child existed and the source fingerprint matched. Human input `f 1` launched
  exactly `claude --resume 8ab38622-946c-412a-a103-a291b40bd2a0
  --fork-session` through `rein -> cmd -> claude`. The source remained
  byte-identical; distinct fork
  `claude:0cb79679-ae10-432c-8295-df0ecc8315a3` recorded the exact fresh
  challenge/response, was rediscovered and inspected, and completed an
  independent physical resume with another exact response. Both runs exited
  `0`; zero controlled processes remained.

Current result: row 26 `PASS` for filter, inspect, resume, fork, invalid input,
`q`, EOF/interrupt, both aliases, read-only routing, and non-TTY refusal.

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
| 2 | Full local verification and required cross-builds | — | PASS — section 5 targeted recheck |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | — | PASS — section 3 |
| 4 | `rein sessions` discovers exact Claude sessions | — | PASS — sections 4, 6 |
| 5 | `rein sessions` discovers exact Codex sessions | — | PASS — sections 4, 6 |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | — | PASS — section 6 delayed recheck |
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
| 21 | Codex native resume | — | PASS — section 8 targeted recheck |
| 22 | Claude vendor-native fork, source preserved | — | PASS — section 8 targeted recheck |
| 23 | Codex vendor-native fork, source preserved | — | PASS — section 8 targeted recheck |
| 24 | Missing/ambiguous reference and missing executor fail safely | — | PASS — sections 5, 8 |
| 25 | JSON/native-child separation and child failure propagation | — | PASS — sections 5, 8 |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | — | PASS — section 9 targeted recheck |
| 27 | Non-TTY prompt failure is immediate and actionable | — | PASS — section 9 |
| 28 | Gemini read-only physical path, when installed | — | PASS — section 10 |
| 29 | OpenCode read-only physical path, when installed | — | PASS — section 10 |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | — | PASS — sections 8, 10 |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | — | PASS — section 5 |
| 32 | Phase 1 automated regression remains green | — | PASS — section 5 targeted recheck |

## 12. Findings

### Original release-blocking findings — 2026-07-31 (preserved)

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

### Targeted resolution — 2026-08-01

All five original Windows blockers were resolved on the unchanged product
commit: the existing MSYS2 toolchain made verification and race green; the
quiescent sequential alias comparison was identical; Codex resume and both
vendor-native forks passed with independent resume proof; and the real
human-keyboard picker passed capable resume/fork in addition to its retained
interaction evidence.

Current Windows release-blocking findings: **none**. This is a Device B result,
not final Phase 2 release sign-off; Mac coordinator reconciliation remains
outside this report.

### Non-blocking

1. OpenCode retains millisecond vendor epochs while the Reinstate index exposes
   whole-second precision. The indexed second still comes exactly from the
   top-level `updated` epoch.
2. OpenCode physical discovery is workspace-scoped on this installation; row
   29 passed from the controlled OpenCode workspace.
3. Claude's enabled Serena integration created untracked `.serena/.gitignore`
   and `.serena/project.yml` files in one disposable controlled project. They
   were preserved as vendor-side evidence and were not committed; no Reinstate
   product or report-worktree file was affected.

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
- The first delayed alias attempt was contaminated by a newly started Codex
  process during its quiescence interval. It was preserved but rejected; the
  second attempt completed fully before the next process started.
- Byte-only ConPTY and `AppActivate`/`SendKeys` evidence was explicitly retired.
  Every passing targeted native/picker route used a human keyboard in real
  Windows Terminal.
- A compressed PowerShell PID predicate briefly misreported a still-live
  picker as gone. A corrected read-only check immediately proved the same
  picker/PID chain remained live; no input or product action occurred during
  the false alarm, and the physical test then continued normally.

## 13. Repository hygiene

- report-only branch: `test/phase2-b952d38c2dc5-windows-report`
- tested base commit: `b952d38c2dc57b0a96bc696860318ea7c2975800`
- changed files: this report only
- private/local artifacts excluded: isolated home, SQLite index, build
  artifacts, cross-builds, raw logs, PTY helpers, disposable projects, and
  controlled vendor files
- product code unchanged: true
- secrets/transcripts committed: `false`

Final W4 verification proved one report file in the staged,
merge-base-to-tip, and PR file lists; a clean privacy scan; draft/unmerged PR
[#71](https://github.com/HarjjotSinghh/reinstate/pull/71); and exact
local/remote branch parity after the final report update.

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
required_pass=30
required_partial=0
required_fail=0
required_not_tested=0
optional_physical_pass=2
optional_physical_not_tested=0
configless_local_only=PASS
preview_privacy=PASS
claude_resume_fork=PASS
codex_resume_fork=PASS
picker=PASS
phase1_regression=PASS
release_blocking_findings=0
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```
