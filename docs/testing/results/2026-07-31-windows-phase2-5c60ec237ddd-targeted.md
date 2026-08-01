# Phase 2 targeted native-Windows closeout — commit 5c60ec237ddd

**Targeted verdict:** `PASS`

**Rows rerun:** `3 PASS / 6 PASS / 20 PASS / 21 PASS / 22 PASS / 23 PASS / 26 PASS`

**Optional physical rows:** `28 PASS / 29 PARTIAL`

**New native-Windows regression gate:** `PASS`

This is a supplemental native-Windows report for the explicitly targeted
closeout only. It does not reclassify rows outside 3, 6, 20, 21, 22, 23, 26,
28, and 29, does not reconcile devices, and does not claim final Phase 2
certification.

The targeted verdict above covers the seven originally targeted rows. Optional
rows 28 and 29 were added in a later supplemental pass; row 29 is `PARTIAL`
because the OpenCode read-only path is functionally correct but emits a zero
`updated_at`, which hides the session from the default listing order. See
sections 12 and 13.

The report contains only controlled composite references, hashes, counts,
booleans, exit codes, relative labels, and sanitized errors. It contains no
transcript prose, uncontrolled session IDs, credentials, secret values,
absolute private paths, or unrelated session metadata.

## 1. Scope and environment

| Field | Value |
| ----- | ----- |
| Execution date | 2026-07-31 |
| Device | Native Windows, 64-bit |
| Shell | Windows PowerShell Desktop `5.1.26100.8328` |
| Architecture | `AMD64`; 64-bit process |
| Tested commit | `5c60ec237ddded8e314cdb8c1449080ddc923395` |
| Remote base | `origin/feat/phase2-local-index` at the tested commit |
| Reinstate version | `v0.1.0-38-g5c60ec2` |
| Claude Code | `2.1.220` |
| Codex CLI | `codex-cli 0.146.0` |
| Gemini CLI | `0.53.0` (auth type `gemini-api-key`; key never read, printed, or persisted) |
| OpenCode | `1.18.2` |
| Report branch | `test/phase2-5c60ec237ddd-windows-targeted` |
| Product files changed | `0` |

No WSL, Git Bash, vendor update, merge, tag, release, product-code edit, or
sandbox/approval bypass was used. Test artifacts and controlled vendor state
were intentionally left in place.

## 2. Provenance and build

| Command / assertion | Exit | Result |
| ------------------- | ---- | ------ |
| `git fetch origin` | `0` | Target base advanced to the requested commit |
| `git checkout 5c60ec237ddded8e314cdb8c1449080ddc923395` | `0` | Detached HEAD matched exactly |
| `git status --porcelain` | `0` | Empty before worktree creation |
| Dedicated report worktree creation | `0` | Branch created directly from the tested commit |
| `go mod download` with `GOTOOLCHAIN=go1.25.12` | `0` | Dependencies available |
| Exact full-SHA Windows build | `0` | `bin/rein.exe` built locally |
| `rein version --json` | `0` | Full commit and `v0.1.0-38-g5c60ec2` matched |
| `reinstate.exe` alias verification | `0` | Alias reported the same full commit |
| Final-closeout `git fetch origin` | `0` | Requested commit remained available from `origin` |
| Final-closeout `git checkout 5c60ec237ddded8e314cdb8c1449080ddc923395` | `0` | Product checkout detached at the exact commit |
| Final-closeout `git status --porcelain` | `0` | Empty before the row-3 preflight |
| Final-closeout exact full-SHA Windows build | `0` | `reinstate.exe` rebuilt; `rein.exe` copied byte-identically |
| Both final-closeout `version --json` checks | `0` each | Both embedded the full exact commit; binary SHA-256 values matched |

The original checkout and report worktree were both clean at the tested
commit before report creation. Ignored build outputs were not staged.

The final-closeout build command was:

```text
$env:GOTOOLCHAIN='go1.25.12'; $env:CGO_ENABLED='0'
go build -ldflags <full-SHA-version-metadata> -o bin\reinstate.exe .\cmd\reinstate
Copy-Item bin\reinstate.exe bin\rein.exe
```

The build and copy each exited `0`. Both absolute binaries then returned exit
`0` from `version --json` and identified commit
`5c60ec237ddded8e314cdb8c1449080ddc923395`.

## 3. Row 3 — fresh configless home

**Result:** `PASS`

Run ID: `20260731-5c60ec237ddd-windows-final`.

The final closeout used a new task-specific
`$reinstateAcceptanceHome` variable. No spelling of PowerShell's reserved
`$HOME` variable was assigned. Before the first Reinstate command:

- the target path did not exist;
- forbidden config/state/backup/key/credential/profile item count was `0`;
- init/sync/storage/conflict/push/pull commands run was `0`;
- `$env:REINSTATE_HOME` exactly equaled the fresh target;
- three possible unintended profile-index locations were stamped for an
  after-run comparison without reading any session content.

The complete configless command surface used the two absolute exact-build
binaries:

| Command | Exit | Sanitized result |
| ------- | ---- | ---------------- |
| `& <absolute-rein-path> sessions` | `0` | `100` records; eight sanitized vendor warnings, with codes `coalesced_session_segments`, `oversized_record`, `session_read_failed`, and `source_scan_failed` |
| `& <absolute-rein-path> sessions --json` | `0` | Parsed JSON; `100` records |
| `& <absolute-reinstate-path> sessions --json` | `0` | Parsed JSON; `100` records |
| `& <absolute-rein-path> list --help` | `0` | Phase 1 compatibility help remained distinct |

Outputs were captured privately. No full listing, transcript text, private
path, or uncontrolled session identifier is reproduced here. Every process
received the exact isolated `REINSTATE_HOME` value. Prompt-pattern hits for
passphrase, keyring, credential, access key, secret key, or storage endpoint
were `0` across all four commands.

Afterward, the isolated home contained exactly two relative items:

```text
cache/
cache/session-index-v1.sqlite
```

Unexpected item count and forbidden config/sync/backup/key/credential/
passphrase/profile/storage/backend item count were both `0`. No unintended
profile-index stamp changed.

The first ACL assertion wrapper exited `0` but conservatively reported its
internal row predicate as false because it treated Windows `SYSTEM` and
`Administrators` as unexpected non-owner principals. The bounded diagnostic
retry exited `0` and proved, for the isolated home, cache directory, and
database independently:

- the current Windows user was the owner;
- allow principals were exactly owner, `SYSTEM`, and `Administrators`;
- broad allow principal count was `0`;
- unexpected allow principal count was `0`.

This is owner-only under the runbook's native-Windows permission model. The
initial predicate was stricter than the acceptance definition; it did not
change an ACL or product state.

## 4. Row 6 — alias parity and idempotency

**Result:** `PASS`

The same cache-only isolated home and byte-identical exact-build `rein.exe`
and `reinstate.exe` were used. One bounded PowerShell wrapper performed each
attempt without agent interaction between commands and wrote raw stdout and
stderr outside the repository:

```text
& <absolute-rein-path> sessions --json
& <absolute-rein-path> sessions --json
& <absolute-reinstate-path> sessions --json
```

The wrapper exited `0`. All nine product invocations across three attempts
exited `0`, returned `100` records, and wrote zero stderr bytes.

| Attempt | Output bytes | Raw equality | SHA-256 equality | Disposition |
| ------- | ------------ | ------------ | ---------------- | ----------- |
| `1` | `59306 / 59306 / 59306` | false | false | Quiet-window retry |
| `2` | `59306 / 59306 / 59306` | false | false | Quiet-window retry |
| `3` | `59306 / 59306 / 59306` | false | false | Proven-live exclusion analysis |

The wrapper did not convert unexplained inequality into a pass. On the final
attempt it found exactly one changed composite key, labelled here only as
`unrelated_live_vendor_record_1`. It was a non-controlled Codex record whose
size changed from `3495899` to `3499950` bytes and whose update timestamp
changed from `2026-07-31T09:37:54Z` to `2026-07-31T09:38:21Z`. Both updates
fell inside the bounded capture window
`2026-07-31T09:37:50.8648546Z` through
`2026-07-31T09:38:25.0419410Z`. Proven-live exclusion count was exactly `1`;
unexplained-difference count was `0`.

After excluding only that proven-live key, parsed evidence showed:

- identical top-level and record schemas;
- identical composite record-key sets;
- identical deterministic order across both repeated `rein` scans and the
  `reinstate` scan;
- independent validation of `updated_at DESC, agent ASC, id ASC` ordering;
- zero duplicate composite keys in every scan;
- all nine controlled records present exactly once in every scan;
- zero controlled canonical-JSON mismatches;
- every remaining stable record byte-equivalent after canonical JSON
  encoding;
- unchanged refreshes left all stable records unchanged;
- `rein` and `reinstate` returned identical stable records.

The isolated home remained exactly cache-only after the parity wrapper.
Temporary parity outputs were retained outside the repository and were not
printed or staged.

## 5. New Windows command-shim regression gate

Command:

```text
go test -count=1 -v ./internal/cli -run '^TestNativeClaudeLaunchThroughWindowsCommandShim$'
```

Exit: `0`.

The parent test and both `resume` and `fork` subtests passed. This exercised
the production `ExecLaunchRunner` through a native `claude.cmd` command shim
and verified stdin, stdout, argv, and cwd preservation.

## 6. Fresh controlled corpus

No source or result from the earlier Device B run was carried forward.

| Purpose | Composite reference | Result |
| ------- | ------------------- | ------ |
| Claude source for rows 20/22 | `claude:df90e8b8-daf2-4030-b010-25409a785c32` | Created with exact controlled response |
| Claude fork child | `claude:067b51f5-3775-4fb6-88e1-40a86fe90f9e` | Distinct, indexed, inspected, independently resumed |
| Codex source for row 21 | `codex:019fb6a2-f2cf-7dd2-8090-9c411bb12fb9` | Created with exact controlled response |
| Codex first row-23 child attempt | `codex:019fb6b1-8572-7930-88d6-b2f224852e42` | Real response obtained, but attempt discarded from PASS proof after interrupted parent-hash capture |
| Codex clean row-23 retry source | `codex:019fb6b8-1c47-7702-98e8-6c7d3bbf5c77` | Fresh retry baseline |
| Codex clean row-23 retry child | `codex:019fb6b9-5290-7c42-b7c3-cac747b98c0d` | Distinct, indexed, inspected, independently resumed |
| Picker resume source | `codex:019fb6bd-0c3e-7190-969f-443a5fb7fe47` | Unique filtered record |
| Picker fork source | `codex:019fb6bf-6622-7c73-8eb0-60c07bfe7101` | Unique filtered record |
| Picker fork child | `codex:019fb6c0-ddcc-74f3-9333-5eac39cd82ec` | Distinct, indexed, inspected |

The isolated Reinstate home contained only:

```text
cache/
cache/session-index-v1.sqlite
```

No `config.toml`, sync state, or backups were created. The disposable Claude
workspace ended with an untracked `.serena/` directory created by vendor
tooling. It was not inspected or removed. The disposable Codex workspace
remained clean.

## 7. Row 20 — Claude native resume

**Result:** `PASS`

The controlled prompt was piped into the real absolute Reinstate binary from
the separate launcher directory, outside the recorded Claude workspace:

```text
"Reply with exactly <CONTROLLED-MARKER>" |
  & <absolute-rein-path> resume "claude:<controlled-id>"
```

Evidence:

- Reinstate exit `0`;
- launcher cwd differed from the recorded workspace;
- vendor output was exactly the controlled response;
- source-file SHA-256 changed and file size increased;
- exact assistant-role marker count changed from `0` to `1`;
- refreshed record count remained exactly `1`;
- refreshed message count was `4`;
- record workspace and source events retained the controlled Claude cwd.

This proves a real source append and controlled response, not process
detection.

## 8. Row 22 — Claude vendor-native fork

**Result:** `PASS`

The fork prompt was piped from the outside launcher directory through:

```text
"Reply with exactly <CONTROLLED-MARKER>" |
  & <absolute-rein-path> fork "claude:<controlled-id>"
```

Evidence:

- fork Reinstate exit `0`;
- output exactly matched the controlled fork response;
- parent SHA-256 and size were unchanged after fork;
- exactly one new Claude session file appeared;
- child ID was distinct from the parent;
- child contained exactly one assistant-role fork marker;
- child source events and index record retained the controlled cwd;
- refresh found exactly one child record with the original branch;
- corrected `inspect --json` check exited `0` and matched child key/cwd;
- independent piped child resume exited `0`;
- child resume output was exact, child hash changed, and child grew;
- parent SHA-256 and size remained unchanged after child resume;
- final refresh found exactly one parent and one child.

The combined assertion wrapper exited `14` after all vendor mutations because
it read `inspect` fields at the JSON envelope root. No mutation was replayed.
A read-only envelope-aware inspect retry exited `0`.

## 9. Row 21 — Codex native resume

**Result:** `PASS`

The real Reinstate resume was launched in a new attached Windows Terminal
window. The controlled prompt was delivered with native Windows keyboard
events after detecting the exact new:

```text
codex resume <controlled-id>
```

Evidence:

- Windows Terminal and exact Codex process detected;
- exact resume argv matched;
- target window was activated and actual keyboard input was sent;
- exact assistant-role controlled response count became `1`;
- Codex source metadata retained the recorded cwd;
- Reinstate terminal exit was `0`;
- scoped Codex process count returned to `0`;
- pre-refresh cached source size/message count:
  `95909` bytes / `5` messages;
- post-refresh source size/message count:
  `151149` bytes / `10` messages;
- refreshed record count was exactly `1` with the correct cwd.

The controller's post-response SHA command exited `1` because the active
rollout was briefly exclusively locked. The already-delivered prompt was not
resent. A shared-read check proved the assistant response and cwd, the scoped
terminal was closed by keyboard interrupt, and SQLite pre/post refresh values
proved the source append.

## 10. Row 23 — Codex vendor-native fork

**Result:** `PASS` on a fresh-source retry.

### First attempt

The first real fork produced Reinstate exit `0` and child
`codex:019fb6b1-8572-7930-88d6-b2f224852e42` with the controlled response.
The controller result was interrupted before the parent baseline hash could
be emitted. A raw marker scan also matched the active harness's own rollout,
demonstrating that raw string presence was insufficient evidence. This
attempt was not used to mark the row PASS.

### Clean retry

Fresh source:
`codex:019fb6b8-1c47-7702-98e8-6c7d3bbf5c77`.

Separately captured parent baseline:

- SHA-256:
  `929D2398CCFE322975432D79004956E0D6CC2029B9BF2D8573F7F476BEF55C19`;
- size: `98336` bytes;
- message count: `5`.

Real native fork evidence:

- Windows Terminal, exact `codex fork <source-id>` argv, window activation,
  and actual keyboard input all proved;
- exactly one new rollout appeared;
- child:
  `codex:019fb6b9-5290-7c42-b7c3-cac747b98c0d`;
- exact assistant-role response count was `1`;
- Reinstate exit `0`;
- parent SHA-256 and size matched the separately captured baseline;
- child contained both fork and source `session_meta` IDs, and the fork ID was
  present;
- refresh preserved distinct source and child composite identities;
- child index record, branch, cwd, and `inspect --json` all matched;
- independent child resume used another attached Windows Terminal and actual
  keyboard input;
- independent resume exact response count changed `0` to `1`;
- independent resume Reinstate exit `0`;
- child grew from `128048` to `136314` bytes and message count from `9` to
  `13`;
- parent SHA-256 and size remained unchanged;
- scoped resume/fork process count returned to `0`.

This directly re-proves the changed Codex fork-identity behavior when a fork
rollout contains more than one `session_meta` identity.

## 11. Row 26 — native interactive picker

**Result:** `PASS`

### Filter, inspect, and resume selection

A fresh Codex source was uniquely searchable. Native input sequence:

```text
/<unique-controlled-token>
i 1
1
```

Evidence:

- the unique search returned exactly one record;
- native `rein` picker and Windows Terminal were detected;
- filter, inspect, and selection inputs were delivered;
- exact `codex resume <controlled-id>` process/argv was observed;
- the controlled prompt was delivered with keyboard input;
- exact assistant response count changed `0` to `1`;
- Reinstate exit `0`;
- source grew from `98229` to `107854` bytes;
- refreshed message count was `9`;
- refreshed cwd matched;
- scoped process count returned to `0`.

### Filter and fork selection

Another fresh source was uniquely searchable. Native input sequence:

```text
/<unique-controlled-token>
f 1
```

Evidence:

- native `reinstate` alias and Windows Terminal were detected;
- exact `codex fork <controlled-id>` process/argv was observed;
- the controlled prompt was delivered with keyboard input;
- exactly one distinct child appeared:
  `codex:019fb6c0-ddcc-74f3-9333-5eac39cd82ec`;
- exact assistant response count was `1`;
- Reinstate exit `0`;
- parent SHA-256
  `F8AA54B47B16DDD37B35E9235C371142C3888C0BC57440CBB908C3A74B2F9632`
  and size `98228` were preserved;
- child identity, cwd, index record, and inspect all matched;
- scoped process count returned to `0`.

### Invalid input, cancel, EOF, and interrupt

| Sub-gate | Result | Evidence |
| -------- | ------ | -------- |
| Invalid input recovery | PASS | Invalid command followed by `q`; exit `0`; no vendor process; controlled sources unchanged |
| `q` cancel | PASS | Exit `0`; no launch or source mutation |
| EOF | PASS | Ctrl+Z plus Enter; exit `0`; no fallback and no vendor launch |
| Ctrl+C | PASS | Kernel-captured exit `3221225786` (`0xC000013A`, signed `-1073741510`); no controlled launch |
| Privacy | PASS | No terminal transcript was captured; focused picker privacy test exited `0` |

The first Ctrl+C wrapper exited `28` because the interrupt killed the child
shell before it wrote a status file. A second wrapper exited `29` because
PowerShell exposed a blank managed `ExitCode` and observed unrelated host
Codex process churn. Read-only classification proved that process had a
PowerShell parent, no resume/fork argv, and zero controlled-ID matches.
A final retry held a native process handle and obtained the exact
`0xC000013A` exit with `GetExitCodeProcess`.

Focused privacy command:

```text
go test -count=1 -v ./internal/cli -run '^TestInteractivePickerFiltersAndLaunchesExactSelection$'
```

Exit: `0`.

## 12. Row 28 — Gemini read-only physical path

**Verdict: `PASS`.**

Gemini `0.53.0` was driven through its normal non-interactive interface with
the default models forced away from the previously observed upstream 503
high-demand path:

```text
gemini -m gemini-3.1-flash-lite -p "Reply with exactly this token and nothing else: <TEST_ID>"
```

Exit `0`. The final stdout line was exactly the controlled `TEST_ID`. The
process-scoped API key was never read, printed, persisted, or copied.

### Bounded before/after metadata snapshot

| Measure | Before | After |
| ------- | ------ | ----- |
| Gemini `chats` session files | `24` | `25` |
| Files containing the controlled `TEST_ID` | — | `1` |
| `rein sessions --agent gemini --json` records | `20` | `21` |
| New indexed record IDs (set difference) | — | `1` |

Attribution used only file names, sizes, modification times, and a literal
`TEST_ID` match, so no unrelated transcript prose was read. Exactly one new
record appeared: `gemini:b6a849e6-5aa8-4d1d-8dde-5bdd097dca3b`.

### Discovery, search, and inspection

| Assertion | Exit | Result |
| --------- | ---- | ------ |
| `rein sessions --agent gemini` | `0` | Controlled composite reference listed |
| `rein sessions --agent gemini --json` | `0` | Record present; `capabilities.resume=false`, `capabilities.fork=false` |
| `rein search <TEST_ID> --json` | `0` | Exactly `1` session returned, the controlled record |
| `rein inspect gemini:b6a849e6-… --json` | `0` | `can_resume=false`, `can_fork=false` |

`read_only_reason` was `Gemini CLI sessions are read-only in Phase 2` in every
surface. Literal search matched because the Gemini record's search text
includes bounded user-prompt text.

### Read-only enforcement

The physical controlled record carries no recorded workspace, so `resume` and
`fork` refused at the workspace guard:

- `rein resume gemini:b6a849e6-…` exited `5`;
- `rein fork gemini:b6a849e6-…` exited `5`;
- message: `recorded session workspace is unavailable: recorded session workspace is missing`;
- source file SHA-256 and size were byte-identical before and after both
  commands.

`internal/sessionindex/launch.go` orders the workspace guard before the
capability guard, so a workspace-less read-only record reports the workspace
reason rather than the read-only reason. Both map to `ExitCompatibility` (`5`)
and both refuse before any `LaunchPlan` is constructed, so zero vendor launch
is structural rather than incidental.

To exercise the read-only branch itself, a synthetic fixture carrying a
recorded workspace was indexed through `GEMINI_CLI_HOME` pointed at a
disposable tree outside the real Gemini home:

- `rein resume gemini:fixture-r28-readonly` exited `5`;
- `rein fork gemini:fixture-r28-readonly` exited `5`;
- message: `native session action is unsupported: Gemini CLI sessions are read-only in Phase 2`;
- fixture SHA-256 unchanged.

No real Gemini session file was created, modified, or removed by Reinstate.

## 13. Row 29 — OpenCode read-only physical path

**Verdict: `PARTIAL`** — the read-only contract holds, but one confirmed
metadata defect is recorded below.

The fresh active OpenCode database was proven empty before any controlled
work: `opencode session list --format json` exited `0` with empty stdout, and
`rein sessions --agent opencode --json` returned `0` sessions. The stale
incompatible database backup was not inspected, modified, restored, migrated,
deleted, or located in this report.

Exactly one controlled session was then created through the vendor's normal
interface (`opencode --pure run <controlled prompt>`), yielding
`opencode:ses_047b6cbcaffepaYgfx67TgcgVw`.

### Bounded before/after metadata snapshot

| Measure | Before | After |
| ------- | ------ | ----- |
| `opencode session list --format json` entries | `0` | `1` |
| `rein sessions --agent opencode --json` records | `0` | `1` |

### Discovery, search, and inspection

| Assertion | Exit | Result |
| --------- | ---- | ------ |
| `rein sessions --agent opencode --json` | `0` | Controlled record present |
| `rein search ses_047b6cbca… --json` | `0` | Exactly `1` session, the controlled record |
| `rein inspect opencode:ses_047b6cbca… --json` | `0` | `can_resume=false`, `can_fork=false` |

`read_only_reason` was `OpenCode sessions are read-only in Phase 2` in every
surface.

**Contract note — no prompt-text search.** Literal search on the controlled
`TEST_ID` returned `0` sessions. This is by design, not a regression: the
OpenCode record's search text is built from ID, title, project, workspace, and
branch only, because the supported `session list` command returns no prompt
text. Literal search over indexed metadata is therefore the applicable proof
and it succeeded.

### Confirmed defect — zero `updated_at`

`opencode session list --format json` reports `updated` and `created` as
top-level epoch-millisecond fields. `eventTimestamp`
(`internal/sessionindex/claude.go:367`) only matches
`timestamp`, `updatedAt`, `updated_at`, `lastUpdated`, `createdAt`, and
`created_at`, and `openCodeRecord`
(`internal/sessionindex/opencode.go:203-207`) only reads `updated` when it is
nested under a `time` object. Neither path matches this vendor shape, so the
record's `updated_at` is emitted as `0001-01-01T00:00:00Z`.

Observed impact: `SortRecords` orders by `UpdatedAt` descending, so the zero
timestamp sorts the session below every real record. With
`rein sessions --limit 200 --json`, the only OpenCode session did not appear at
all; it is reachable only through an explicit `--agent opencode` filter. The
default limit is `100`, so the physical OpenCode path is effectively invisible
in unfiltered listings on a populated host.

### Read-only enforcement

The controlled record carries a recorded workspace, so both commands reached
the capability guard:

- `rein resume opencode:ses_047b6cbca…` exited `5`;
- `rein fork opencode:ses_047b6cbca…` exited `5`;
- message: `native session action is unsupported: OpenCode sessions are read-only in Phase 2`;
- OpenCode process count after both commands: `0`;
- session count after both commands: still `1`, with identical ID and title.

**Contract note — byte-level source stability.** The active OpenCode database
file's SHA-256 changes on every read. This was attributed to the vendor, not to
Reinstate: invoking `opencode session list --format json` alone, with no
Reinstate process involved, changed the database SHA-256 on each of two
consecutive calls while producing byte-identical JSON. The change is SQLite
open-for-write bookkeeping inherent to OpenCode's command-mediated read path.
Source mutation is therefore zero at the logical/session level but cannot be
asserted at the byte level for OpenCode, unlike Gemini's file-read path, which
was proven byte-stable.

## 14. Codex Phase 1 fail-closed state

`rein setup check --json` exited `5`:

- Claude: `SUPPORTED`;
- Codex: `UNTESTED`;
- Codex check: `layout/version untested; writes blocked`;
- check code: `5`.

This is the expected fail-closed result for Codex `0.146.0` Phase 1 encrypted
sync writes. It does not fail the independently proven Phase 2 local
resume/fork rows above. No vendor CLI was updated or downgraded.

## 15. Retry and deviation ledger

| Operation | Exit / state | Disposition |
| --------- | ------------ | ----------- |
| Initial PowerShell tool-table helper | `1` | Empty-pipe parser error; collected-array retry exited `0`; no build/vendor command started |
| Computer Use initialization | no subprocess exit; OS error `2` | Native pipe unavailable; used scoped Windows Terminal plus keyboard events |
| First sessions JSON assertion | wrapper `8`; product `0` | JSON envelope mistaken for bare array; read-only retry passed |
| Claude fork combined assertion | wrapper `14`; fork/resume product exits `0` | Inspect envelope misread; no mutation replay; read-only retry `0` |
| Codex resume post-response hash | wrapper `1` | Active rollout lock; shared-read and cached-index proof completed without resending prompt |
| First shared-read diagnostic | command exit `0` with PowerShell constructor errors | Corrected FileShare reader retry exited `0` |
| Codex resume shutdown command | interrupted | Existing scoped process later proved closed; Reinstate status file contained `0` |
| First Codex fork controller | interrupted | Existing child/response and Reinstate `0` proved; attempt excluded because parent hash output was incomplete |
| First post-interrupt fork diagnostic | `1` | Missing PowerShell brace; simplified read-only retry `0` |
| Raw fork marker scan | command `0`, ambiguous | Also matched the active harness rollout; replaced with assistant-role exact matching |
| Clean Codex fork retry | `0` | Fresh source, separate parent baseline, complete PASS evidence |
| First picker search count assertion | wrapper `8`; product search `0` | One object collapsed by PowerShell; array-aware retry `0` |
| First Ctrl+C controller | wrapper `28` | Picker exited, but child shell could not write status |
| Second Ctrl+C controller | wrapper `29` | Picker exited; managed exit blank; unrelated host Codex process classified |
| First Ctrl+C diagnostic | `1` | Empty-pipe parser error; collected-array retry `0` |
| Kernel-handle Ctrl+C retry | controller `0` | Exact process exit `0xC000013A`; no controlled launch |
| First final-hygiene wrapper | `124` timeout | Reserved `$home` assignment failed and caused read-only enumeration outside the isolated home; no contents/mutation; excluded from report evidence |
| Corrected bounded hygiene wrapper | `0` | Used `$reinHome`; isolated tree was cache-only |
| Initial final-closeout build-metadata search | `1` | Exact command `rg -n "ldflags|buildDate|version\.Commit|var \(.*Commit|Commit string" Makefile* .github internal cmd -g "*.go" -g "Makefile*"`; native Windows rejected the positional wildcard; explicit-file retry exited `0`; no build had started |
| Initial final-closeout ACL-precedent glob search | `1` | Exact search used positional `docs/testing/results/*windows*.md`; native Windows rejected the wildcard; `-g "*windows*.md"` retry exited `0` |
| Two bounded no-match report searches | `1` each | Exact searches were `rg -n "ACL|permission|owner-only|owner only" <supplemental-report>` and `rg -n "Row 3|row_3|owner-only|native OS|ACL" docs/testing/results -g "*phase2*.md"`; later branch-object reads found the precedent; no files changed |
| Combined branch/report-history discovery | `1` | Exact command combined `git branch -a --list "*phase2*"` with `git log --all --oneline -- <candidate-report-path> docs/testing/results`; useful output was returned, then `git ls-tree` retries exited `0` |
| First report-branch ancestry wrapper | parser exit `1` | Inline `git merge-base` expression had a missing parenthesis; corrected bounded wrapper exited `0` and proved ancestry |
| First report patch | tool verification error; no command exit | Console-mojibake heading text did not match the UTF-8 file; no partial edit occurred; smaller UTF-8-aware patches succeeded |
| First pre-stage guard | wrapper exit `1` | Numeric `diff_check_exit=0` was incorrectly tested as a boolean false; nothing was staged; boolean-normalized retry exited `0` and staged exactly one report |
| Row-3 ACL predicate | wrapper `0`, internal predicate false | Over-strict owner-SID-only rule rejected inherited `SYSTEM`/`Administrators`; bounded native-Windows ACL classification retry exited `0` with zero broad/unexpected allows |
| Row-6 parity attempt 1 | three product exits `0`; raw/SHA unequal | Bounded quiet-window retry; no key was excluded and no verdict was drawn from this attempt |
| Row-6 parity attempt 2 | three product exits `0`; raw/SHA unequal | Second bounded quiet-window retry; no key was excluded and no verdict was drawn from this attempt |
| Row-6 parity attempt 3 | three product exits `0`; raw/SHA unequal | One non-controlled key proved live in-window; excluded count `1`; all stable and controlled canonical records equal |
| Row-28 first Gemini index lookup | product `0`; predicate false | Looked the record up by session file name; the Gemini record ID is the internal `sessionId`, not the file stem; before/after set-difference retry exited `0` and isolated exactly one new record |
| Row-28 first read-only fixture | product `0`; fixture unparsed | Windows PowerShell `Set-Content -Encoding utf8` wrote a BOM, so the JSONL metadata line failed to parse and `message_count` was `0`; BOM-free `UTF8Encoding($false)` rewrite exited `0` and indexed the workspace correctly |
| Row-29 controlled `opencode run` | no exit; moved to background, later stopped | The vendor run did not return within `300s` and held the database lock; the controlled session had already been created and was independently confirmed through `session list`; the stopped run produced no report evidence |
| Row-29 first database mutation check | product exits `5`; hash unreadable | The still-running vendor process held an exclusive lock, so both hashes were null and the equality result was vacuous; re-run after the lock cleared produced real hashes and the vendor-attribution test below |

The host PowerShell profile repeatedly emitted PSReadLine prediction warnings
because command output was redirected. These warnings did not alter captured
product exit codes.

## 16. Final hygiene

- original checkout HEAD:
  `5c60ec237ddded8e314cdb8c1449080ddc923395`;
- original checkout tracked status: clean;
- report branch starts at the tested commit;
- report worktree was clean before this report;
- only the requested supplemental report is intended for staging;
- product files changed: `0`;
- isolated Reinstate home: cache-only;
- final row-3/row-6 isolated home: cache-only;
- final parity evidence: retained outside the repository;
- target launch process count: `0`;
- target Windows Terminal window count: `0`;
- no transcripts, credentials, secrets, absolute private paths, build
  artifacts, index files, launcher helpers, or disposable workspaces are in
  the report diff;
- Gemini API key: never read, printed, persisted, copied, or reported;
- OpenCode stale-database backup: not inspected, modified, restored, migrated,
  deleted, or located in this report; left in place for review;
- row-28 synthetic fixture: written outside the repository and outside the real
  Gemini home, reached only through `GEMINI_CLI_HOME`;
- controlled Gemini and OpenCode sessions: left in place;
- no cleanup or device reconciliation was performed.

## 17. Targeted transfer block

```text
PHASE2-WINDOWS-TARGETED-CLOSEOUT-V1
device=windows
test_commit=5c60ec237ddded8e314cdb8c1449080ddc923395
reinstate_version=v0.1.0-38-g5c60ec2
report_path=docs/testing/results/2026-07-31-windows-phase2-5c60ec237ddd-targeted.md
report_branch=test/phase2-5c60ec237ddd-windows-targeted
native_windows_command_shim=PASS
row_3_configless_isolation=PASS
row_6_alias_parity_idempotency=PASS
row_20_claude_resume=PASS
row_21_codex_resume=PASS
row_22_claude_fork=PASS
row_23_codex_fork=PASS
row_26_picker=PASS
row_28_gemini_read_only=PASS
row_29_opencode_read_only=PARTIAL
targeted_pass=7
targeted_partial=0
targeted_fail=0
targeted_not_tested=0
optional_physical_pass=1
optional_physical_partial=1
optional_physical_fail=0
gemini_cli_version=0.53.0
opencode_version=1.18.2
opencode_updated_at_defect=true
opencode_prompt_text_search=unsupported_by_design
opencode_byte_level_source_stability=vendor_inherent_change
codex_phase1_encrypted_writes=UNTESTED
codex_phase2_local_resume_fork=PASS
product_files_changed=0
secrets_or_transcripts_committed=false
devices_reconciled=false
final_phase2_certification_claimed=false
END-PHASE2-WINDOWS-TARGETED-CLOSEOUT-V1
```

No device reconciliation was performed, and this report does not claim final
Phase 2 certification.
