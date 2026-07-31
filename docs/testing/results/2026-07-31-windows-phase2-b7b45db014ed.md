# Phase 2 acceptance — native-Windows Device B report

**Verdict:** `FAIL`

**Milestone:** `DEVICE_COMPLETE`

**Required counts:** `23 PASS / 3 PARTIAL / 4 FAIL / 0 NOT TESTED`

**Optional physical counts:** `0 PASS / 2 NOT TESTED`

This report is a fresh native-Windows run of all 32 rows against the exact
requested commit. No result was carried forward. It covers only controlled
markers, composite references, relative path labels, counts, booleans, exit
codes, and sanitized errors. It contains no transcript prose, assistant
reasoning/messages, tool output, credentials, secret values, or absolute
private paths.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC execution window | 2026-07-31 |
| Device | native-Windows Device B |
| Tested Git commit | `b7b45db014edf030d820e503ee23b579c5032e69` |
| Signed tag, if any | None; development acceptance |
| Reachable from `origin/main` | No; `git merge-base --is-ancestor` exit `1` |
| Reinstate version JSON | version `v0.1.0-36-gb7b45db`, commit `b7b45db014edf030d820e503ee23b579c5032e69` |
| OS/version/build | Windows 10 Pro, 25H2, build `26200.8328` |
| Architecture | `X64`; 64-bit process |
| Native shell | Windows PowerShell Desktop `5.1.26100.8328`; never WSL |
| Claude Code version/state | `2.1.220`; `SUPPORTED`; authenticated non-interactive call passed |
| Codex CLI version/state | `codex-cli 0.146.0`; `UNTESTED`; `layout/version untested; writes blocked`, code `5` |
| Gemini CLI version/state | `0.38.0`; installed, authentication blocked controlled-session creation |
| OpenCode version/state | `1.18.2`; installed, local schema error blocked JSON discovery |
| Git version | `2.52.0.windows.1` |
| Go version | `go1.26.1 windows/amd64`; Make gates pinned `GOTOOLCHAIN=go1.25.12` |
| Make / C toolchain | MSYS2 GNU Make `4.4.1`; MSYS2 GCC `16.1.0`; `CGO_ENABLED=1` |
| Report branch | `test/phase2-b7b45db014ed-windows-report` |
| Draft PR | PENDING |

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | PASS | Clean HEAD equals the full requested SHA |
| Binary reports the tested commit | PASS | Both Windows `.exe` aliases were locally built with full-SHA ldflags; `rein version --json` matched |
| Product tree was clean before testing | PASS | Tracked change count `0` |
| Report branch starts at the tested commit | PASS | Merge base equals the requested SHA |
| Report is the only committed change | PENDING | Proved before commit in section 13; final staged and committed diff proof follows publication |
| No secret/transcript/private path was committed | PENDING | Final staged and committed diff proof follows publication |

The checkout contained a stale ignored `bin/rein.exe` from commit `e07a59b`.
It was rejected as evidence and overwritten by an exact full-SHA local build.
The repository Makefile's Windows build emits extensionless binaries, so
explicit `.exe` builds were required for the handoff's Windows commands.

## 3. Isolation and local-only proof

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | PARTIAL | The prescribed home was absent initially and remained cache-only, but one later picker harness incorrectly used the user profile; see finding R3 |
| No `rein init` run | PASS | No init, sync, storage, or conflict command was executed |
| No `config.toml` or sync state created | PASS | Prescribed home: absent config/state/backups before and after |
| No credential/passphrase/keyring request | PASS | No Phase 2 command requested any |
| No backend/network dependency | PASS | Local commands ran without a backend or storage coordinates |
| Only derived index state created in prescribed home | PASS | Tree: `cache/`, `cache/session-index-v1.sqlite` |
| Index and parent permissions are owner-only | PASS | Current user owns home/cache/DB; broad allow ACE count `0` |
| Accidental out-of-scope derived state | PARTIAL | A reserved-variable harness error created `%USERPROFILE%/cache/session-index-v1.sqlite`; no config/state/backups; intentionally not cleaned |

The accidental profile cache is a harness violation, not evidence of a
credential/backend request. It remains present because cleanup was explicitly
forbidden.

## 4. Controlled corpus

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | `claude:e9ced5ec-2e9f-4eea-85a7-1a2de2ecb62f` | `project_alpha` | Yes | full |
| Codex | `codex:019fb626-adca-7020-832d-60f71e433727` | `project_unicode` | Yes | full |
| Gemini | Not created | `project_alpha` | No — vendor authentication blocked | read-only, physical gate not tested |
| OpenCode | Not created | `project_unicode` | No — vendor schema error blocked | read-only, physical gate not tested |

Additional refresh-only Codex reference:
`codex:019fb632-9099-7f80-aeef-81bbe65c9b9c`.

Controlled creation evidence:

- Claude: exit `0`, one new session file, exact controlled response, one
  structured file reference, native ID matched before/after metadata.
- Codex: exit `0`, one new rollout, exact controlled response, native ID
  matched before/after metadata.
- Claude append: exit `0`, source grew, message count changed from `5` to `7`,
  exact controlled append marker present.
- Later Codex session: exit `0`, one new rollout, exact controlled response;
  it became newest among the three controlled records.
- The disposable alpha Git repo ended with one untracked `.serena/` directory.
  Its contents were not inspected or removed. The unicode repo remained clean.

## 5. Part A environment setup

### Successful checks

- Native Windows PowerShell, `X64`, exact clean commit: PASS.
- Winget-installed GnuWin32 Make `3.81`: installed, but unsuitable for this
  POSIX-recipe Makefile.
- Winget-installed WinLibs GCC `16.1.0`: installed and usable.
- Winget-installed MSYS2 plus Make `4.4.1` and GCC `16.1.0`: PASS.
- `go env CGO_ENABLED`: `1`.
- `go test -race ./internal/adapter`: exit `0`.
- Claude authenticated controlled call: exit `0`, exact retry marker.
- Console input driver: scoped `WriteConsoleInput` probe exited `0`.
- Raw Codex bytes: `codex-cli 0.146.0`.
- Codex probe: `agents={"claude":"SUPPORTED","codex":"UNTESTED"}`;
  `agent.codex={"name":"agent.codex","status":"fail","message":"layout/version untested; writes blocked","code":5}`.
- Probe home was not created.

### Part A failures and retries

| Command / operation | Exit | Result and retry |
| ------------------- | ---- | ---------------- |
| Initial file-info `foreach (...) { ... } \| Format-Table` helper | `1` | PowerShell parser: empty pipe element; retried with a collected array, exit `0` |
| Initial tool-detection `foreach (...) { ... } \| ConvertTo-Json` helper | `1` | Same parser error; retried with a collected array, exit `0` |
| `choco install make mingw -y` | `1` | Non-elevated shell; package-lock/access denied; `0` packages installed; continued via winget |
| First `make --version` verification wrapper | wrapper `0`; make did not start | `make` not on refreshed PATH; no make-process exit code; PATH corrected and verification passed |
| `make build` with GnuWin32 Make | `2` | `cmd.exe` could not parse POSIX `VAR=value` recipes |
| `make build` with `$env:SHELL` override | `2` | GnuWin32 Make still used `cmd.exe` |
| `make SHELL=C:/Progra~1/Git/bin/bash.exe build` | `2` | process creation failures |
| `make build` with `MAKESHELL` override | `2` | GnuWin32 Make still used `cmd.exe` |
| `pacman -S --noconfirm --needed make mingw-w64-x86_64-gcc` | `0` | Transient mirror `404`/connection-reset messages; all packages installed |
| First Claude authentication marker wrapper | `1` | Claude process exit `0` and JSON parsed, but exact result mismatch; retry exit `0` with exact marker |
| `.\bin\rein.exe setup check --json` | `5` | Expected config-missing plus Codex `UNTESTED`; stdout/stderr were separated and parsed |
| Computer Use initialization | no subprocess exit | Native pipe unavailable, OS error `2`; fell back to scoped native console control |
| First A7 console probe | `124` | UI activation succeeded but input timed out; exact Ctrl+C cleanup |
| Second A7 console probe | `124` | Expected title was unavailable; scoped console-buffer input then succeeded |
| First AttachConsole cleanup assertion | `1` | Ctrl+C was generated but process had not yet exited; direct console input completed probe with exit `0` |

## 6. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| `make fmt-check` | PASS with deviation | Exit `0`; MSYS emitted `tee: /dev/stderr: No such file or directory`; independent `gofmt -l .` returned `0` files |
| Focused session-index/CLI/adapter tests | PASS | Exit `0` |
| Full Go test suite | PASS after retry | First exit `1` from harness PATH; focused retry and full retry exit `0` |
| Race suite | PASS | `go test -race ./...`, exit `0` |
| Vet | PASS | `make vet`, exit `0` |
| Full merge gate | PASS with deviation | `make verify`, exit `0`; lint `0 issues`; tests/race green; no called vulnerabilities; same `/dev/stderr` warning |
| Required cross-builds | PASS | Windows amd64, Darwin arm64/amd64, Linux amd64: all exit `0` |
| Focused CLI safety tests | PASS | Six tests: read-only refusal, ambiguity/missing, picker, non-TTY, child failure, cancellation |
| Focused session-index/privacy tests | PASS | Eleven tests including Codex fork identity, platform fixtures, bounded/privacy, concurrent tail, corrupt DB, oversized OpenCode output |
| Phase 1 regression | PASS | Full suite and `make verify` passed |

First `go test ./...` failure details:

- command exit `1`;
- `TestVerifyAvoidsRedundantDoctestRuns` and
  `TestQuickGateStaysFocusedAndNonRelease` could not find `make`;
- `TestProductionDeploymentRejectsInvalidWebsiteTagDate` produced an empty
  unexpected failure under the same stale PATH;
- focused retry with MSYS2 Make/GCC passed all three, then the full retry
  passed.

## 7. Configless index and refresh

- `rein sessions`, `rein sessions --json`,
  `reinstate sessions --json`, repeated sessions, `rein list --help`, and
  `rein list` all exited `0`.
- Both controlled refs were discovered with exact workspace labels and
  branches.
- Claude and Codex records each advertised resume/fork capability.
- Claude carried the required structured file reference.
- Ordering was newest-update first with agent/native-ID tie breakers.
- The three controlled records remained singular and stable.
- Corrupt derived index test: controlled invalid bytes were written to the
  derived DB; the next sessions command rebuilt a valid SQLite database,
  returned both original controlled refs, changed no vendor source, and
  created no forbidden state.
- Append/new-session refresh passed. The later Codex record became newest
  within the controlled query, and a no-change controlled refresh was
  byte-equivalent.
- Reinstate scans did not modify the controlled vendor sources.
- Full-array alias/repeat equality was not provable: exactly one unrelated
  active Codex record changed only `updated_at` and `size_bytes` between
  scans. Controlled records were byte-equivalent. Row 6 is therefore
  `PARTIAL`.
- The live corpus emitted sanitized warning codes for unrelated Codex
  oversized records, Gemini unreadable/coalesced records, and an OpenCode
  source-scan failure. No unrelated IDs or content are reported.
- `rein list` exited `0` with output distinct from `rein sessions` and did not
  contain the controlled Phase 2 composite refs. The runbook sentence saying
  it was not expected to work from the fresh home is a documentation nit, not
  a product failure.

## 8. Search and inspect

Search evidence:

- unique prompt fragment: both controlled original refs;
- AND terms: both refs;
- agent filters: exact Claude/Codex ref;
- project filters: exact alpha/unicode ref;
- branch filters: exact alpha/unicode ref;
- file filter: exact Claude ref with the structured file;
- `--limit 1`: one result;
- lower-case query: both refs;
- spaces plus Unicode `β`: exact unicode Codex ref;
- deliberate zero-match: empty result, exit `0`;
- unique metacharacter query containing `[x]_$safe`: exactly the controlled
  Claude ref;
- search JSON did not expose `prompt_preview`;
- human search contained zero occurrences of the controlled prompt fragment.

Inspect evidence for both controlled refs:

- human and JSON forms exited `0`;
- exact key/agent/workspace label/branch/message count/capabilities matched;
- file counts were bounded and Claude's expected file was present;
- previews originated from controlled user prompts;
- preview lengths were `160` and `104` characters;
- whitespace was collapsed; control-character count `0`;
- no source path was emitted;
- controlled source hashes stayed unchanged.

The first search-batch harness never started because the tool transport
rejected a NUL byte embedded in the harness regex; no subprocess/exit code
existed. The corrected batch ran completely and passed.

## 9. Last, resume, and fork

### Dry-run plans

All dry-runs exited `0`, started no vendor process, and changed no controlled
source:

- Claude resume: `claude`, `["--resume", "<controlled-id>"]`, alpha cwd.
- Codex resume: `codex`, `["resume", "<controlled-id>"]`, unicode cwd.
- Claude fork: `claude`,
  `["--resume", "<controlled-id>", "--fork-session"]`, alpha cwd.
- Codex fork: `codex`, `["fork", "<controlled-id>"]`, unicode cwd.
- global `last`, `last --agent claude`, and
  `last --project b7b45db014ed-alpha` selected the expected records.

### Real native resume

| Agent | Result | Evidence |
| ----- | ------ | -------- |
| Claude | FAIL | Three real launches; UI/input attempts made; no assistant-role challenge marker; source unchanged; two attempts ended at Reinstate exit `1`; exact process trees were closed with scoped Ctrl+C and batch confirmation |
| Codex | FAIL | Exact Codex TUI detected; Reinstate exit `0`; no assistant-role challenge marker and no source change; zero orphan |

These results prove launch attempts, not successful native challenge-response
resume. Neither row is marked PASS.

### Real vendor-native fork

| Agent | Result | Evidence |
| ----- | ------ | -------- |
| Claude | FAIL | Exact UI and argv launch, Reinstate exit `0`, source preserved, but no new session file, distinct ID, or challenge response |
| Codex | FAIL | Exact UI and argv launch, Reinstate exit `0`, source preserved, but no new rollout, distinct ID, or challenge response |

The automated `TestCodexForkKeepsItsOwnIdentity` regression passed, but that
does not replace the failed physical fork row.

### Safe failures

- Unique bare Claude and Codex IDs resolved in dry-run.
- Missing reference: documented usage exit `2`, actionable
  `session not found`, no process/source mutation.
- `resume --json`, `fork --json`, and `last --json` without `--dry-run`:
  exit `2` with the required refusal.
- Ambiguity, missing workspace/executor, read-only compatibility, and child
  failure propagation passed deterministic injected tests.

The combined safe-failure wrapper initially exited `1` because the harness
incorrectly expected missing-reference exit `4`; the product returned its
documented exit `2`. The single-gate retry passed.

## 10. Interactive switcher

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| Bare `rein` opens a real native console | PASS | Line input accepted |
| Bare `reinstate` alias opens a real native console | PASS | Exact picker fork target detected |
| `/text` filtering | PARTIAL | Unique filter input accepted; Windows Terminal UI Automation did not expose rendered rows |
| `i NUMBER` inspect | PARTIAL | Input accepted; rendered inspect state unavailable |
| invalid input recovery | PARTIAL | Invalid input followed by `q` exited cleanly with no default launch; rendered error unavailable |
| `NUMBER` resume target | PASS | Exact controlled `codex.exe` resume child detected |
| `f NUMBER` fork target | PASS | Exact controlled `cmd.exe` + `claude.exe` fork argv detected |
| `q` cancel | PASS | Exit `0`, no vendor process/source mutation |
| Ctrl+C interrupt | PASS | Scoped signal exited picker immediately at exit `1`; no `q` fallback, launch, or mutation |
| non-TTY empty input | PASS | Both aliases exit `2` in 36–37 ms with `rein sessions --json` hint |
| unrelated transcript privacy | PASS | No transcript body was captured or reported; automated picker privacy test passed |

One picker resume attempt was invalidated by assigning to reserved `$home`.
PowerShell refused the assignment and the process inherited the real profile as
`REINSTATE_HOME`, creating the accidental profile cache described in section
3. A corrected `$reinHome` retry detected the exact controlled Codex child and
closed cleanly.

The picker fork wrapper detected the exact target but first exited `1` because
the `cmd.exe` shim remained at a batch confirmation. A scoped `Y` confirmation
removed the only orphan.

## 11. Read-only adapters

### Gemini

- Installed version: `0.38.0`.
- First controlled call: Gemini process exit `0`, but no marker and no session.
- Second retry did not start because `.NET ProcessStartInfo` was given the
  script shim; wrapper exit `1`, no vendor-process exit.
- Final `cmd.exe < NUL` retry: Gemini process exit `0`, authentication error,
  no marker/session.
- Reinstate controlled Gemini search: exit `0`, zero controlled results.
- Physical discovery/search/inspect row: `NOT TESTED` because the vendor could
  not create a controlled session.

### OpenCode

- Installed version: `1.18.2`.
- `opencode session list --format json`: exit `1`,
  `no such column: name`.
- Controlled `opencode run ... --format json`: exit `1`, same schema error, no
  marker/session.
- Reinstate controlled OpenCode search: exit `0`, zero controlled results.
- Physical discovery/search/inspect row: `NOT TESTED` because the vendor
  database could not create/list a controlled session.

### Read-only mutation refusal

Focused injected CLI evidence passed the required compatibility exit `5` and
zero-launch contract. Row 30 is `PASS`; this does not convert optional physical
rows 28–29 into PASS.

## 12. Mandatory matrix

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | — | PASS — section 2 |
| 2 | Full local verification and required cross-builds | — | PASS — section 6 |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | — | PARTIAL — section 3 |
| 4 | `rein sessions` discovers exact Claude sessions | — | PASS — sections 4, 7 |
| 5 | `rein sessions` discovers exact Codex sessions | — | PASS — sections 4, 7 |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | — | PARTIAL — section 7 |
| 7 | Derived index path, rebuild, idempotency, and private permissions | — | PASS — sections 3, 7 |
| 8 | Prompt-fragment literal search | — | PASS — section 8 |
| 9 | Agent filter | — | PASS — section 8 |
| 10 | Project filter | — | PASS — section 8 |
| 11 | Branch filter | — | PASS — section 8 |
| 12 | File filter | — | PASS — section 8 |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | — | PASS — section 8 |
| 14 | `sessions` and `search` do not dump transcript passages | — | PASS — section 8 |
| 15 | `inspect` metadata/160-code-point user preview policy | — | PASS — section 8 |
| 16 | Append/new-session refresh and no-change idempotency | — | PASS — section 7 |
| 17 | `last` selects the correct resumable session and filters | — | PASS — section 9 |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | — | PASS — section 9 |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | — | PASS — section 9 |
| 20 | Claude native resume | — | FAIL — section 9 |
| 21 | Codex native resume | — | FAIL — section 9 |
| 22 | Claude vendor-native fork, source preserved | — | FAIL — section 9 |
| 23 | Codex vendor-native fork, source preserved | — | FAIL — section 9 |
| 24 | Missing/ambiguous reference and missing executor fail safely | — | PASS — sections 6, 9 |
| 25 | JSON/native-child separation and child failure propagation | — | PASS — sections 6, 9 |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | — | PARTIAL — section 10 |
| 27 | Non-TTY prompt failure is immediate and actionable | — | PASS — section 10 |
| 28 | Gemini read-only physical path, when installed | — | NOT TESTED — section 11 |
| 29 | OpenCode read-only physical path, when installed | — | NOT TESTED — section 11 |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | — | PASS — section 11 |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | — | PASS — section 6 |
| 32 | Phase 1 automated regression remains green | — | PASS — section 6 |

## 13. Findings

### Release-blocking

1. **R1 — native resume challenge failed for both full-capability agents.**
   Claude had three real attempts with no controlled response/source change;
   Codex launched and returned `0` but had no response/source change.
2. **R2 — physical vendor-native fork failed for both agents.** Both dry-run
   plans were exact and both sources were preserved, but neither real launch
   produced a distinct native identity or controlled response.
3. **R3 — isolation was contaminated by one harness attempt.** Reserved
   `$home` assignment failed and an extra derived index was created under the
   real profile. No config, state, backup, credential, or secret was created.
4. **R4 — full-array alias/idempotency parity is incomplete.** One unrelated
   active Codex record changed size/timestamp between sequential scans.
   Controlled records were equal.
5. **R5 — picker rendered filter/inspect/invalid states are incomplete.**
   Exact resume/fork targets, cancel, interrupt, and non-TTY were proved, but
   Windows Terminal UI Automation did not expose the rendered text.
6. **R6 — installed Codex is outside this commit's stated verified range.**
   Version `0.146.0` is `UNTESTED`; no vendor update/downgrade occurred.

### Non-blocking

1. Gemini `0.38.0` could not authenticate; optional physical row not tested.
2. OpenCode `1.18.2` has `no such column: name`; optional physical row not
   tested.
3. `rein list` works configlessly and remains distinct from Phase 2 sessions;
   the runbook sentence saying otherwise is documentation drift.
4. MSYS2 Make passes but cannot open `/dev/stderr` for the Makefile's `tee`;
   independent formatting verification passed.
5. The tested development commit is not reachable from `origin/main`.
6. The disposable alpha project has one untracked `.serena/` directory; it was
   not inspected or cleaned.

### Test-harness deviations and failed-command ledger

All failures are also described in context above. Additional exact harness
failures:

| Command / operation | Exit | Retry / disposition |
| ------------------- | ---- | ------------------- |
| `rg -n -i "worktree.*director|worktree" CLAUDE.md` | `1` | No match; sibling worktree used |
| First Go full suite without refreshed tool PATH | `1` | Focused failures passed, then full retry passed |
| Quoted schema `rg` helper | `1` | PowerShell quote parsing; single-quoted retry passed |
| Full-array alias/idempotency assertion wrapper | `1` | Diagnosed one unrelated Codex record changing only timestamp/size |
| Search batch containing a NUL regex | no subprocess exit | Tool transport rejected input; corrected batch passed |
| Claude resume attempt 1 | wrapper `1`; Reinstate `1` after Ctrl+C | No marker/source change; exact batch shim confirmed closed |
| Claude resume attempt 2 | wrapper `1`; Reinstate `1` | No marker/source change; exact batch shim confirmed closed |
| Claude resume attempt 3 | wrapper `1`; Reinstate initially still live | No marker/source change; exact `cmd.exe` + `claude.exe` closed |
| Codex resume attempt | wrapper `1`; Reinstate `0` | No challenge/source change; zero orphan |
| Claude real fork | wrapper `1`; Reinstate `0` | No new session/distinct ID; source preserved |
| Codex real fork | wrapper `1`; Reinstate `0` | No new rollout/distinct ID; source preserved |
| Safe-failure batch | wrapper `1`; missing-ref command `2` | Harness expected wrong exit `4`; documented exit-`2` retry passed |
| First picker resume controller using `$home` | wrapper `1` | Reserved-variable error; accidental profile cache; corrected `$reinHome` retry passed |
| Picker fork controller | wrapper `1` | Exact target detected; batch shim needed scoped `Y`; zero orphan afterward |
| Wildcard test-name `rg` helper | `1` | Windows wildcard error `123`; `-g '*_test.go'` retry passed |
| `Convert.ToHexString` list/sessions hash helper | `1` | API absent on PowerShell 5 runtime; `BitConverter` retry passed |
| Final hygiene assertion wrapper | `1` | Correctly detected untracked `.serena/` in disposable alpha project |
| First report-only validation wrapper | `1` | `git diff --name-only` omits untracked files; corrected `git status --porcelain --untracked-files=all` proof passed |
| First staged-diff validation wrapper | wrapper `1`; `git diff --cached --check` `2` | Three Markdown hard-break trailing spaces; removed and restaged |

No failed or retried command was silently relabeled PASS.

## 14. Repository hygiene

- report-only branch: `test/phase2-b7b45db014ed-windows-report`
- tested base commit: `b7b45db014edf030d820e503ee23b579c5032e69`
- changed files before report creation: `0`
- intended changed files at commit: only
  `docs/testing/results/2026-07-31-windows-phase2-b7b45db014ed.md`
- ignored build/cross-build binaries excluded
- local index/vendor/disposable-project artifacts excluded
- product code unchanged
- controlled launch process count at handoff: `0`
- secrets/transcripts committed: `false`

## 15. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=windows
test_commit=b7b45db014edf030d820e503ee23b579c5032e69
reinstate_version=v0.1.0-36-gb7b45db
report_path=docs/testing/results/2026-07-31-windows-phase2-b7b45db014ed.md
report_branch=test/phase2-b7b45db014ed-windows-report
claude_ref=claude:e9ced5ec-2e9f-4eea-85a7-1a2de2ecb62f
codex_ref=codex:019fb626-adca-7020-832d-60f71e433727
gemini_state=FAIL
opencode_state=FAIL
required_pass=23
required_partial=3
required_fail=4
required_not_tested=0
optional_physical_pass=0
optional_physical_not_tested=2
configless_local_only=FAIL
preview_privacy=PASS
claude_resume_fork=FAIL
codex_resume_fork=FAIL
picker=FAIL
phase1_regression=PASS
release_blocking_findings=6
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```

No cross-device reconciliation was performed, and this report does not claim
Phase 2 complete.
