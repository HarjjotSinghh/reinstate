# Phase 2 acceptance — native-Windows Device B report

**Verdict:** `FAIL`
**Milestone:** `DEVICE_COMPLETE`
**Required counts:** `20 PASS / 1 PARTIAL / 9 FAIL / 0 NOT TESTED`
**Optional physical counts:** `0 PASS / 0 NOT TESTED / 2 FAIL`

This report covers only the exact disposable sessions and redacted paths created
for this run. No real transcript content or secret was used as evidence.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | 2026-07-30T13:01:59Z |
| Device | native-Windows Device B |
| Tested Git commit | `f23e38e13f6eaf73e2c52d4f65730d7c8f94864b` |
| Reachable from fetched `origin/main` | `false`; tested commit is the feature commit whose parent is fetched `origin/main` |
| Signed tag, if any | None; development acceptance |
| Reinstate version JSON | `{"commit":"f23e38e13f6eaf73e2c52d4f65730d7c8f94864b","date":"2026-07-30T12:08:09Z","name":"reinstate","version":"v0.1.0-rc.8-72-gf23e38e"}` |
| OS/version/build | Windows 11 Pro 10.0.26200, build 26200 |
| Architecture | OS `x64`; process `x64` |
| Native shell | Windows PowerShell Desktop 5.1.26100.8328 |
| Claude Code version/state | `2.1.220`; installed, supported, logged out |
| Codex CLI version/state | `0.145.0`; installed, Phase 1 compatibility check reports `UNTESTED` |
| Gemini CLI version/state | `0.38.0`; installed, authentication unavailable, no controlled session persisted |
| OpenCode version/state | `1.18.2`; installed, session-list/run fail with `no such column: name` |
| Git version | 2.52.0.windows.1 |
| Go version | host 1.26.1; required commands used `GOTOOLCHAIN=go1.25.12` |
| `make` / GCC | Not installed |
| Report branch | `test/phase2-f23e38e13f6e-windows-report` |
| Draft PR | `#61`, targeting `feat/phase2-local-index` |

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | PASS | Dedicated worktree `HEAD` and branch base both equal the full requested SHA. |
| Binary reports the tested commit | PASS | Both locally built aliases report the full SHA in `version --json`. |
| Alias binaries are identical | PASS | `rein.exe` and `reinstate.exe` have the same SHA-256. |
| Product tree was clean before testing | PASS with deviation | The initially supplied checkout was actually the older Phase 1 report commit. The exact object was fetched and the mandated clean dedicated worktree was created directly from the requested commit before building or testing product behavior. |
| Report branch starts at the tested commit | PASS | Branch creation and `rev-parse` both identified the requested full SHA. |
| Tested commit reachable from fetched `origin/main` | FAIL | `git merge-base --is-ancestor TEST_COMMIT origin/main` exited `1`; no tag points at the commit. |
| Report is the only committed change | PASS | Staged diff contains exactly this one report path. |
| No secret/transcript/private path was committed | PASS | Staged report scan found no absolute Windows path, username, full prompt, transcript, auth material, or secret. |

This is development acceptance. No public installer or published release was
used as Phase 2 evidence.

## 3. Isolation and local-only proof

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | FAIL overall | The prescribed isolated home was absent before first index use and remained correct for the main run, but one later invalid harness retry used the existing user profile because Windows PowerShell rejected the reserved variable name `$home`. |
| No `rein init` run | PASS | No initialization or sync/storage/conflict command was run. |
| No `config.toml` or sync state created | PASS in isolated home | Final isolated home contains only `cache/` and `cache/session-index-v1.sqlite`; config, state, and backups are absent. |
| No credential/passphrase/keyring request | PASS for Reinstate | Local Phase 2 commands requested none. Vendor authentication state was queried only as booleans. |
| No backend/network dependency | PASS | No profile/backend existed; local commands completed against vendor-local sources and the derived cache. |
| Only derived index state created | FAIL overall | Correct isolated home contains only the derived index, but the invalid `$home` retry also created `<USER_PROFILE>/cache/session-index-v1.sqlite`. It is preserved, not deleted, under the no-cleanup instruction. |
| Index and parent permissions are owner-only | PASS | Home/cache/index had zero broad or unexpected allow principals; inherited access was limited to owner, SYSTEM, and Administrators. |

The main isolated home remained exactly two items deep: `cache/` and
`cache/session-index-v1.sqlite`. A deliberate 19-byte corruption was rebuilt
successfully, rediscovered both controlled source sessions, and left vendor
sources unchanged.

## 4. Controlled corpus

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | `claude:e7912c9a-03c3-4a42-92e1-e63c46125a32` | `<ALPHA_PROJECT>` | User marker present; assistant challenge absent because authentication failed | full metadata; native action blocked physically by vendor login state |
| Codex source | `codex:019fb2f5-e8b8-77d2-9a17-9d135e1a5339` | `<UNICODE_PROJECT>` | Exact A1, A2, and A4 challenge responses verified | full |
| Codex fork | `codex:019fb317-9764-76e2-9824-b1f9100fd65a` | `<UNICODE_PROJECT>` | Exact F1 challenge response verified through Codex | vendor-native fork exists, but Reinstate fails to index it separately |
| Gemini | None | `<ALPHA_PROJECT>` | No controlled session persisted | installed but physical read-only path failed |
| OpenCode | None | `<UNICODE_PROJECT>` | No controlled session created | installed but physical read-only path failed |

The Codex non-interactive creation also produced one related subordinate
session in the controlled workspace. It was not used as the primary reference
or reported with transcript content.

## 5. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| Focused session-index/CLI/adapter tests | PASS | `go test ./internal/sessionindex ./internal/cli ./internal/adapter/...` exited `0`; fresh targeted Phase 2 edge, picker, ambiguity, child-failure, fixture-shape, and privacy tests also exited `0`. |
| Full Go test suite | FAIL | Uncached retry failed `TestSetupCheckJSONPreservesFailureExit`, `TestVerifyAvoidsRedundantDoctestRuns`, and `TestQuickGateStaysFocusedAndNonRelease`. |
| Race suite | FAIL | Default run exited `2` because CGO was disabled; explicit `CGO_ENABLED=1` retry exited `1` because `gcc` was absent. |
| Vet / `make verify` | FAIL | Manual `go vet ./...` exited `0`; pinned lint and vulnerability scan exited `0`; `make vet` and `make verify` could not start (`127`) because `make` was absent. |
| Required cross-builds | PASS | Windows amd64, Darwin arm64, Darwin amd64, and Linux amd64 `CGO_ENABLED=0` builds all exited `0`. |
| Phase 1 regression | FAIL | Uncached core run failed the setup-check exit assertion with installed Codex `0.145.0`; all other named core packages passed. |

Additional passing evidence:

- `gofmt -l .`: exit `0`, zero unformatted files.
- pinned `golangci-lint v2.11.4`: exit `0`, zero issues.
- pinned `govulncheck v1.6.0`: exit `0`, zero called vulnerabilities.
- Git-for-Windows `sh.exe` retry of the invalid-calendar-date doctest: exit
  `0`. This was launched from native PowerShell and was not WSL.
- synthetic tests passed for Windows/macOS/WSL source shapes, malformed and
  oversized records, incomplete concurrent tails, corrupt-store rebuild,
  private permissions, deterministic ordering, ambiguity, exact argv/cwd,
  missing executor/workspace, child exit propagation, and read-only refusal.

## 6. Configless index and refresh

- First human `rein sessions`: exit `0`; both controlled source references were
  present; neither controlled marker was printed.
- `rein sessions --json`, `reinstate sessions --json`, and an unchanged repeat:
  all exit `0`, byte-identical JSON, 100 rows at the default limit, zero
  duplicate composite keys, contract ordering verified.
- Controlled rows had the correct agent, branch, project label, workspace,
  capabilities, and stable composite identity.
- Correct isolated home contained no config, state, backups, credentials, or
  remote object.
- Deliberately corrupted derived index rebuilt successfully.
- Codex vendor resume appended A2; refresh found the new literal and
  metacharacter terms; unchanged refresh was byte-idempotent; Reinstate did not
  change either controlled vendor source.
- Vendor-native Codex fork created distinct ID
  `019fb317-9764-76e2-9824-b1f9100fd65a` and preserved the source. Refresh did
  not discover that fork as a distinct record.

### Release-blocking fork-index defect

The fork file contains two controlled `session_meta` IDs in order:

1. fork ID `019fb317-9764-76e2-9824-b1f9100fd65a`;
2. source ID `019fb2f5-e8b8-77d2-9a17-9d135e1a5339`.

After the fork was resumed through Codex, Reinstate returned zero rows for the
fork even with `sessions --agent codex --limit 1000`, zero F1 search matches,
and `inspect` exit `2`/not found. The Codex parser updates the record ID for
each `session_meta`; the later source ID wins, so the fork is coalesced into
the source record. This defeats distinct fork discovery and independent
Reinstate resume.

## 7. Search and inspect

| Dimension | Result | Evidence |
| --------- | ------ | -------- |
| Prompt fragment / human output | PASS | Controlled Claude and Codex references found; no controlled marker passage printed. |
| AND terms | PASS | Both controlled source references present. |
| Agent filter | PASS | Claude and Codex filters included the expected ref and excluded the other agent. |
| Project filter | PASS | Alpha and Unicode project fragments narrowed correctly. |
| Branch filter | PASS | Both controlled branch fragments narrowed correctly. |
| File filter | FAIL | Both controlled file-filter searches returned zero rows. Claude never authenticated to use its read tool; Codex replied without a structured file-reference event. |
| Case-insensitive literal | PASS | Lowercase query matched the mixed-case controlled term. |
| Unicode | PASS | Unicode-beta query and project filter matched Codex. |
| Shell metacharacters | PASS after append retry | Exactly quoted `MetaChars-[x]_$safe` matched after the Codex A2 vendor append. |
| Limit | PASS | Project-scoped `--limit 1` returned the expected controlled row. |
| Zero match | PASS | Deliberate zero-match query returned an empty result with exit `0`. |
| Inspect human/JSON | PASS | Both controlled source refs returned correct bounded metadata. |

Claude preview length was 159 code points and Codex preview length was 160.
Both were whitespace-collapsed and control-free, contained no controlled tool
file content, and exposed no private source/search fields. Human inspect
outputs were bounded. Sessions and search did not print matching transcript
passages.

## 8. Last, resume, and fork

### Dry-run and resolution

- Claude resume plan: exit `0`; exact `claude`, `["--resume", ID]`, and
  `<ALPHA_PROJECT>` cwd.
- Codex resume plan: exit `0`; exact `codex`, `["resume", ID]`, and
  `<UNICODE_PROJECT>` cwd.
- Claude fork plan: exit `0`; exact
  `claude`, `["--resume", ID, "--fork-session"]`, and cwd.
- Codex fork plan: exit `0`; exact `codex`, `["fork", ID]`, and cwd.
- Bare unique Codex ID resolved to the same exact plan.
- Global `last` matched the independently computed newest resumable row;
  `--agent claude` matched the newest Claude row; controlled project filter
  selected the controlled Codex source.
- All dry runs left controlled vendor files unchanged.

### Negative gates

| Gate | Result |
| ---- | ------ |
| Missing reference | PASS: exit `2`, actionable not-found message, no mutation |
| Ambiguous bare ID | PASS: fresh deterministic injected gate |
| Missing executable | PASS: physical path-isolated run exited `5` before launch |
| Missing workspace | PASS: physical temporary-rename run exited `5`; exact project was restored; no source mutation |
| `resume --json` without dry run | PASS: exit `2`, actionable, no launch |
| `fork --json` without dry run | PASS: exit `2`, actionable, no launch |
| `last --json` without dry run | PASS: exit `2`, actionable, no launch |
| Child failure propagation | PASS: fresh deterministic CLI/runner gates |
| Read-only native-action refusal | PASS: fresh injected-record gates prove compatibility exit `5` and zero launch/mutation |

### Native action results

- Claude resume: FAIL. Reinstate launched the exact
  `claude --resume <controlled-id>` child in the controlled workspace, but the
  installed CLI was logged out. No challenge response or source append
  occurred; the console was closed with an exact-console Ctrl+C event.
- Claude fork: FAIL. Reinstate launched exact fork argv, but login state
  prevented a new identity. Source stayed byte-identical.
- Codex source resume: PARTIAL through Reinstate, PASS through vendor-native
  non-interactive resume. Picker/direct Reinstate launches reached the exact
  native child but did not accept the automated TUI challenge and required
  exact-console Ctrl+C recovery. Independent `codex exec resume <source-id>`
  produced exact A2/A4 responses and kept the same ID.
- Codex fork: FAIL for the complete Reinstate contract. `rein fork` created a
  distinct vendor ID and preserved the source. Exact vendor-native resume of
  that fork produced F1, and reciprocal hashes proved source/fork
  independence. Reinstate then collapsed the fork into the source record and
  could not discover or inspect it separately.

## 9. Interactive switcher

Real visible native Windows PowerShell consoles were driven through the
Windows console/UI input path, not stdin pipes.

| Interaction | Result |
| ----------- | ------ |
| Bare `rein`, `q` cancel | PASS: wrapper exit `0` |
| Bare `reinstate`, exact-ID filter | PASS |
| `i NUMBER` inspect | PASS: exact controlled Claude ID filter, exit `0`, no source mutation |
| Invalid input recovery | PASS: invalid text followed by `q`, exit `0` |
| EOF | PASS: Ctrl+Z/Enter exited `0` |
| Interrupt | FAIL: Ctrl+C produced no wrapper result; exact test window was recovered without force-killing unrelated processes |
| `NUMBER` resume | FAIL: exact Codex child launched, but no controlled TUI challenge response completed |
| `f NUMBER` fork | FAIL in picker attempt: target was still held by the preceding resume attempt and no new ID appeared |
| Direct native-console Codex fork | PARTIAL: distinct ID created and source preserved, but wrapper required Ctrl+C and Reinstate later collapsed the fork |
| Unrelated transcript privacy | PASS for observed/captured evidence: no screenshot or picker body was captured; only controlled result markers, booleans, and hashes were read |

The bundled Windows computer-control plugin could not connect its native pipe
(`os error 2`), so the run used visible native PowerShell consoles with unique
titles, Windows UI input, console HWND focus, and `AttachConsole` Ctrl+C
recovery scoped to exact test process trees.

## 10. Read-only adapters

| Adapter | Result | Evidence |
| ------- | ------ | -------- |
| Gemini | FAIL | Installed `0.38.0`. Two bounded non-interactive attempts returned exit `0` but produced an authentication signal, no exact controlled response, no parseable JSON, and zero new session metadata files. |
| OpenCode | FAIL | Installed `1.18.2`. `opencode session list --format json` exited `1`; sanitized error `no such column: name`. Controlled `opencode run` also exited `1`, produced no marker/ID, and hit the same error. Reinstate emitted `source_scan_failed`. |
| Read-only capability/refusal contract | PASS synthetic-only | Fresh Gemini/OpenCode fixture/fake-runner and injected-record tests passed; resume/fork refusal returns exit `5` with no launch/mutation. |

Because both vendors are installed, rows 28 and 29 are `FAIL`, not
`NOT TESTED`.

## 11. Mandatory matrix

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | | PASS — §1–2 |
| 2 | Full local verification and required cross-builds | | FAIL — §5 |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | | FAIL — §3 |
| 4 | `rein sessions` discovers exact Claude sessions | | PASS — §4, §6 |
| 5 | `rein sessions` discovers exact Codex sessions | | PASS — §4, §6 |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | | PASS — §6 |
| 7 | Derived index path, rebuild, idempotency, and private permissions | | PASS — §3, §6 |
| 8 | Prompt-fragment literal search | | PASS — §7 |
| 9 | Agent filter | | PASS — §7 |
| 10 | Project filter | | PASS — §7 |
| 11 | Branch filter | | PASS — §7 |
| 12 | File filter | | FAIL — §7 |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | | PASS — §7 |
| 14 | `sessions` and `search` do not dump transcript passages | | PASS — §6–7 |
| 15 | `inspect` metadata/160-code-point user preview policy | | PASS — §7 |
| 16 | Append/new-session refresh and no-change idempotency | | FAIL — §6 |
| 17 | `last` selects the correct resumable session and filters | | PASS — §8 |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | | PASS — §8 |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | | PASS — §8 |
| 20 | Claude native resume | | FAIL — §8 |
| 21 | Codex native resume | | PARTIAL — §8–9 |
| 22 | Claude vendor-native fork, source preserved | | FAIL — §8 |
| 23 | Codex vendor-native fork, source preserved | | FAIL — §6, §8 |
| 24 | Missing/ambiguous reference and missing executor fail safely | | PASS — §8 |
| 25 | JSON/native-child separation and child failure propagation | | PASS — §5, §8 |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | | FAIL — §9 |
| 27 | Non-TTY prompt failure is immediate and actionable | | PASS — §9 |
| 28 | Gemini read-only physical path, when installed | | FAIL — §10 |
| 29 | OpenCode read-only physical path, when installed | | FAIL — §10 |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | | PASS — §5, §10 |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | | PASS — §5 |
| 32 | Phase 1 automated regression remains green | | FAIL — §5 |

## 12. Findings

### Release-blocking

1. **Codex fork identity collapse:** a real vendor-native fork exists and
   resumes independently, but its file contains fork then source
   `session_meta` IDs; Reinstate indexes the later source ID and loses the
   fork.
2. **Installed Codex version is outside the compatibility allowlist:** CLI
   `0.145.0` reports `UNTESTED`, changes setup-check exit from expected `3` to
   `5`, and breaks an uncached Phase 1 test.
3. **Required verification is red/incomplete:** full suite fails, `make` is
   absent, and race builds cannot start without GCC.
4. **Claude physical resume/fork unavailable:** installed CLI is logged out.
5. **Physical file-filter dimension returned zero controlled matches.**
6. **Picker native resume/fork/interrupt did not complete the required
   challenge/exit contract.**
7. **Gemini installed physical row failed:** authentication unavailable and no
   controlled session persisted.
8. **OpenCode installed physical row failed:** session-list/run database query
   errors with `no such column: name`.
9. **Isolation deviation:** one invalid PowerShell retry created a derived
   index under `<USER_PROFILE>/cache/` instead of the fresh isolated home.

### Non-blocking

1. The supplied checkout started on an older report commit. The requested
   object was fetched and the mandated dedicated worktree was created directly
   from it before behavior testing.
2. The tested development commit is not yet reachable from fetched
   `origin/main` and has no tag.
3. Codex non-interactive creation spawned one related subordinate controlled
   session; the primary composite ref remained unambiguous.

### Failed commands, retries, and test-harness deviations

All paths below are sanitized.

| Command / attempt | Exit | Result / retry |
| ----------------- | ---- | -------------- |
| Combined environment/vendor version probe | 124 timeout | Retried each vendor independently; all four version probes exited `0`. |
| `make fmt-check` | 127 | `make` absent. Manual `gofmt -l .` passed. |
| `go test ./...` initial | 1 | Invalid-date doctest lacked `sh`; two repository-policy doctests lacked `make`. |
| `go test -race ./...` | 2 | CGO disabled. |
| `CGO_ENABLED=1 go test -race ./...` | 1 | `gcc` not found. |
| Filtered race package enumeration, first two wrappers | 1, 2 | PowerShell quoting error; corrected enumeration found 23 packages, then race build exited `1` for missing GCC. |
| `make vet` | 127 | Manual `go vet ./...` passed. |
| `make verify` | 127 | Manual component gates run separately; full result remains FAIL. |
| `claude -p <CONTROLLED_PROMPT> --output-format json` | 1 | Created one controlled metadata file with an API-error event and no assistant response. `claude auth status --json` exited `1`, `loggedIn=false`. |
| First JSON-search matrix wrapper | 1; child searches `2` | PowerShell array binding omitted valid argv. Minimal direct command and corrected full matrix passed except file filter. |
| First two composite dry-run wrapper batches | 1; child plans `2` | PowerShell positional-array flattening. Direct commands subsequently produced all exact plans with exit `0`. |
| `opencode session list --format json` | 1 | `no such column: name`; no stdout. |
| `opencode run ... --format json` | 1 | Same database error; no controlled session. |
| Windows computer-control initialization | unavailable | Native pipe missing (`os error 2`); used scoped visible-console automation instead. |
| Picker interrupt attempt | 1 | Ctrl+C yielded no wrapper result; exact titled console recovered without broad process termination. |
| Picker Codex resume attempt | 1 | Exact source became locked, but no A3 response or wrapper result; exact process tree closed with `AttachConsole` Ctrl+C. |
| Picker Codex fork while resume remained active | 1 at evidence wrapper; picker wrapper wrote `0` | No new fork. Retried after closing the exact resume tree; direct `rein fork` created the distinct fork. |
| First HWND resume helper | 1 | PowerShell comparison expression was malformed; harmless HWND probe then passed. |
| Final HWND resume retry | 1 | Reserved `$home` assignment failed; no challenge/source change; wrong-home derived index created and preserved. |
| Initial report patch location | no process exit | Patch tool wrote the one untracked report into the stale checkout. It was moved into the dedicated report worktree before staging; the stale checkout was re-proven clean. |
| `rein inspect codex:<FORK_ID> --json` | 2 | Actionable not found; confirms fork-index defect. |
| `go test ./...` with Git-for-Windows `sh` retry | 1 | Invalid-date test recovered; failed setup-check compatibility test plus two missing-`make` doctests remained. |
| `go test ./internal/cli -count=1` | 1 | Reproduced Codex `0.145.0` compatibility/exit regression. |
| Uncached Phase 1 core package set | 1 | All named packages passed except `internal/cli` for the same assertion. |

Expected negative gates also executed: missing reference exit `2`; missing
executor and workspace exit `5`; JSON-without-dry-run exits `2`; non-TTY exits
`2`. Those are PASS results, not failed acceptance commands.

No force-kill was used. Exact controlled console trees that required recovery
received normal Windows Ctrl+C console events. No cleanup was performed.

## 13. Repository hygiene

- report-only branch: `test/phase2-f23e38e13f6e-windows-report`
- tested base commit: `f23e38e13f6eaf73e2c52d4f65730d7c8f94864b`
- changed files: `docs/testing/results/2026-07-30-windows-phase2-f23e38e13f6e.md` only
- private/local artifacts excluded: binaries, indexes, console result markers,
  cross-builds, disposable repositories, and all vendor session files
- product code unchanged: `true`
- secrets/transcripts committed: `false`
- cleanup performed: `false`

## 14. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=windows
test_commit=f23e38e13f6eaf73e2c52d4f65730d7c8f94864b
reinstate_version=v0.1.0-rc.8-72-gf23e38e
report_path=docs/testing/results/2026-07-30-windows-phase2-f23e38e13f6e.md
report_branch=test/phase2-f23e38e13f6e-windows-report
claude_ref=claude:e7912c9a-03c3-4a42-92e1-e63c46125a32
codex_ref=codex:019fb2f5-e8b8-77d2-9a17-9d135e1a5339
gemini_state=FAIL
opencode_state=FAIL
required_pass=20
required_partial=1
required_fail=9
required_not_tested=0
optional_physical_pass=0
optional_physical_not_tested=0
configless_local_only=FAIL
preview_privacy=PASS
claude_resume_fork=FAIL
codex_resume_fork=FAIL
picker=FAIL
phase1_regression=FAIL
release_blocking_findings=9
product_files_changed=0
secrets_or_transcripts_committed=false
END-PHASE2-DEVICE-REPORT-V1
```
