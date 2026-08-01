# Phase 2 acceptance — macOS Device A report

**Verdict (this device):** `PASS`
**Phase 2 overall after reconciliation:** `FAIL` — see section 15
**Milestone:** `FINAL_RECONCILIATION`
**Required counts (macOS device rows):** `30 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`
**Optional physical counts (macOS):** `0 PASS / 2 NOT TESTED`
**Reconciled dual-device required rows passed:** `25 of 30`

This report covers only the exact disposable sessions and paths created for
this run. No real transcript content or secret was used as evidence.

Cross-device reconciliation is complete (section 15). This device's own 30
required rows passed, but Phase 2 does **not** pass physical acceptance,
because native resume and vendor-native fork have physical evidence from macOS
only. No confirmed Reinstate defect was identified on either device.

This run did not meet its stated safety preconditions: it executed in a
permission-bypass mode. See the correction in section 2 and deviation 8 in
section 12.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | 2026-07-31T04:12:45Z start / 2026-07-31T05:05Z end |
| Device | `macOS Device A` |
| Tested Git commit | `b7b45db014edf030d820e503ee23b579c5032e69` |
| Signed tag, if any | none — development acceptance, built from source |
| Reinstate version JSON | `{"commit":"b7b45db","name":"reinstate","version":"v0.1.0-36-gb7b45db"}` |
| OS/version/build | macOS 26.5.2, build 25F84 |
| Architecture | arm64 (Apple silicon) |
| Native shell | zsh 5.9 (arm64-apple-darwin25.0) |
| Claude Code version/state | 2.1.220 — installed, full capability |
| Codex CLI version/state | 0.146.0 — installed, full capability; **above** this commit's `maximumVerifiedCodexVersion = 0.145.0` (see findings) |
| Gemini CLI version/state | `NOT_INSTALLED` |
| OpenCode version/state | `NOT_INSTALLED` |
| Git version | 2.55.0 |
| Go version | go1.26.5 darwin/arm64 (build pinned `GOTOOLCHAIN=go1.25.12`) |
| Report branch | `test/phase2-b7b45db014ed-macos-report` |
| Draft PR | see section 13 |

Development acceptance: the exact clean commit was built locally. This build is
not a published release and the public `v0.1.0` installer was not used as
evidence. `TEST_COMMIT` is not reachable from `origin/main`; it is reachable
from `origin/feat/phase2-local-index`.

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | PASS | `git rev-parse HEAD` = `b7b45db014edf030d820e503ee23b579c5032e69` |
| Binary reports the tested commit | PASS | `rein version --json` and `reinstate version --json` both report `commit=b7b45db`, `version=v0.1.0-36-gb7b45db` |
| Product tree was clean before testing | PASS | `git status --porcelain` empty before and after the entire run |
| Report branch starts at the tested commit | PASS | branch created from `b7b45db014ed…`; worktree HEAD equals the tested commit |
| Report is the only committed change | PASS | staged diff is exactly one added file (section 13) |
| No secret/transcript/private path was committed | PASS | report contains only versions, composite references, counts, booleans, exit codes, controlled markers, redacted relative paths, sanitized errors |

No commit, merge, rebase, cherry-pick, tag, release, or product-branch change
was made.

**Correction — permission mode.** An earlier revision of this report stated
that normal sandboxing and approval controls were enabled and that no
permission bypass was used. That claim was wrong and is withdrawn. No
individual tool call passed a sandbox-disabling flag, but the **session itself
ran in a permission-bypass mode**, so no tool call was gated by an approval
prompt. See deviation 8 in section 12 for the evidence and the affected-gate
analysis. This is disclosed as a deviation and is not converted into a pass.

## 3. Isolation and local-only proof

Paths are redacted; `<HOME>` denotes the isolated `REINSTATE_HOME`.

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | PASS | all three run targets proven absent before creation; `find` over the home returned 0 entries before the first command |
| No `rein init` run | PASS | `init` and every sync/storage/conflict command were never invoked |
| No `config.toml` or sync state created | PASS | after the full run the home contains exactly `<HOME>/cache` and `<HOME>/cache/session-index-v1.sqlite`; `config.toml`, `state.json`, `backups`, `devices`, `manifest.json` all absent |
| No credential/passphrase/keyring request | PASS | no command prompted for or requested a credential, passphrase, or keyring access; no keychain item exists for this run's service name |
| No backend/network dependency | PASS | every Phase 2 command succeeded with no storage coordinates, profile, or network backend |
| Only derived index state created | PASS | the only new Reinstate state is the derived `cache/session-index-v1.sqlite` |
| Index and parent permissions are owner-only | PASS | `cache` = `drwx------`, database = `-rw-------`. When Reinstate creates the home itself it is `drwx------`; the `0755` on this run's home root came from the harness's own `mkdir`, not from Reinstate |

A pre-existing, unrelated keychain item from earlier Phase 1 work is present on
this machine. It was neither read nor modified; no Phase 2 command touches it.

## 4. Controlled corpus

`RUN_ID = 20260731-b7b45db014ed`

| Agent | Composite reference | Disposable project label | Marker found | Capability |
| ----- | ------------------- | ------------------------ | ------------ | ---------- |
| Claude | `claude:70bda718-665a-4eb5-a624-ca66ad382c8a` | `…-RUN_ID-alpha`, branch `phase2/RUN_ID-alpha` | yes | full |
| Claude (fork) | `claude:3f4e4069-b4c2-4f6f-afa7-3a00167a7aaf` | same project, vendor-created | yes | full |
| Codex | `codex:019fb663-60eb-7591-be41-6b1b211fa37f` | `reinstate phase2 RUN_ID unicode-β`, branch `phase2/RUN_ID-beta` | yes | full |
| Codex (fork) | `codex:019fb679-a83f-7eb3-ae57-b4159b0e5fcd` | same project, vendor-created | yes | full |
| Gemini | — | — | — | not installed |
| OpenCode | — | — | — | not installed |

Controlled markers used (bounded, harmless):

```text
REINSTATE-PHASE2-<RUN_ID>-MACOS-CLAUDE-A1
REINSTATE-PHASE2-<RUN_ID>-MACOS-CODEX-A1
REINSTATE-PHASE2-<RUN_ID>-MACOS-CLAUDE-RESUME-R1
REINSTATE-PHASE2-<RUN_ID>-MACOS-CODEX-RESUME-R1
REINSTATE-PHASE2-<RUN_ID>-MACOS-CLAUDE-FORK-F1
REINSTATE-PHASE2-<RUN_ID>-MACOS-CODEX-FORK-F1
REINSTATE-PHASE2-<RUN_ID>-MACOS-CLAUDE-FORKRESUME-F2
REINSTATE-PHASE2-<RUN_ID>-MACOS-CODEX-FORKRESUME-F2
```

Every native ID was identified only from a before/after metadata diff plus the
exact controlled marker. No vendor session file was hand-edited at any point.

## 5. Automated verification

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| `make fmt-check` | PASS | exit 0 |
| Focused session-index/CLI/adapter tests | PASS | `go test -count=1 ./internal/sessionindex ./internal/cli ./internal/adapter/...` exit 0, 5/5 packages ok |
| Full Go test suite | PASS | `go test -count=1 ./...` exit 0, 21 packages ok, 0 FAIL |
| Race suite | PASS | `go test -race -count=1 ./...` exit 0, 21 packages ok, no data race |
| Vet / `make verify` | PASS | `make vet` exit 0; `make verify` exit 0, `verify ok` |
| Required cross-builds | PASS | all four `CGO_ENABLED=0` builds exit 0 and produce the correct object format: PE32+ x86-64, Mach-O arm64, Mach-O x86_64, ELF x86-64 |
| Phase 1 regression | PASS | Phase 1 packages green in the full suite: `crypto`, `sync`, `pathmap`, `config`, `credentials`, `backend/s3`, `backend/memory`, `device`, `lock`, `project`, `doctor`, `fixture`, `fsx`, `processcheck` |

Focused sessionindex + CLI verbose run: 73 tests PASS, 0 FAIL.

`make verify` reported 0 vulnerabilities in imported packages and 1 in a
required-but-uncalled module — unchanged from the commit's baseline and not a
Phase 2 regression.

No gate was skipped or interrupted. Results were re-run with `-count=1` so no
verdict rests on a cached result.

## 6. Configless index and refresh

- `rein sessions` succeeded from the fresh configless home and discovered both
  exact controlled references (82 records total).
- **Alias parity:** `rein sessions --json` and `reinstate sessions --json`
  returned identical key sets and **zero differing records**.
- **Ordering:** verified newest-update-first, then agent, then native ID across
  the whole listing.
- **No duplicates:** three consecutive scans each returned 82 records with 82
  unique composite keys.
- **Stable identity:** both controlled records were byte-identical across all
  three scans; after append and after fork each still resolved to exactly one
  record.
- **No-change idempotency:** consecutive scans differed only in two records,
  both genuinely live and being appended concurrently — this session's own
  Claude session and one unrelated live Codex session. Over the controlled
  corpus, idempotency held exactly. Recorded as a deviation in section 12.
- **Corrupt index rebuild:** the derived database was overwritten with 42 bytes
  of non-SQLite data; the next command rebuilt it (905,216 bytes), returned all
  82 records including both controlled references, kept `-rw-------`, and
  **did not modify any vendor file**.
- **Non-mutation:** controlled Claude and Codex source fingerprints
  (size + mtime + SHA-256 prefix) were identical before and after indexing,
  searching, inspecting, and after the corrupt-rebuild cycle.
- **`rein list`:** remains the Phase 1 compatibility command with its own
  `--agent` flag and its own tab-separated Phase 1 output shape. It is not
  silently redefined as the Phase 2 canonical listing command.
- Bounded-record warnings (`ignored JSONL record … larger than 4194304 bytes`)
  were emitted for unrelated oversized vendor sessions. This is the documented
  bounded-field protection operating, not a failure.

## 7. Search and inspect

All queries ran against the isolated home with no network involvement.

| Dimension | Result | Evidence |
| --------- | ------ | -------- |
| Prompt-fragment literal | PASS | each controlled marker returned exactly its own session |
| Case-insensitive | PASS | fully lowercased marker returned the same single session |
| AND terms | PASS | marker + a second present term → 1 result; two mutually exclusive markers → 0 results, no fallback |
| Agent filter | PASS | `--agent claude` and `--agent codex` each narrowed to the correct single session |
| Project filter | PASS | alpha and `unicode-β` fragments each selected the correct session |
| Branch filter | PASS | `phase2/RUN_ID-alpha` and `-beta` each selected the correct session |
| File filter | PASS | `--file phase2_<RUN_ID>_alpha.txt` selected the Claude session, whose record carries a real structured `files` entry produced by an actual Read-tool use |
| Limit | PASS | `--limit 1` returned exactly 1, consistent with the ordering contract |
| Unicode | PASS | `β Ünïcödé 数据` handled as literal query data; the `unicode-β` project name round-tripped correctly through record, filter, and launch `cwd` |
| Zero match | PASS | honest empty result, exit 0, no error and no fallback to all sessions |
| Shell metacharacters | PASS | `; touch …`, `$(id)`, backticks, `' OR 1=1 --`, `../../etc/passwd`, `%s%n` all treated as literal data: 0 results, no injection, no side-effect file created |

**Preview policy — explicit statements:**

- `sessions` printed **only** metadata (composite ref, project, branch, title).
  No transcript body, assistant message, reasoning, or tool output.
- `search` identified matching sessions by the same metadata columns and
  **never printed the matching passage**.
- `inspect` returned identity, agent, title, project, workspace, branch,
  timestamp, message count, structured file references, capability flags, and
  a single bounded `prompt_preview`.
- Preview measured at exactly **160 Unicode code points** for the Claude
  session (capped) and 101 for the Codex session (natural length); both had
  **zero control characters** and fully collapsed whitespace.
- The inspect JSON schema contains no assistant, reasoning, tool-output, or
  environment field at all.
- Inspecting a missing reference failed cleanly (exit 2) without affecting
  listing or inspection of healthy sessions.

## 8. Last, resume, and fork

**Dry-run plans (captured before every real launch).** All use an argv array,
never a shell string, and the recorded workspace as `cwd`:

| Command | Executable | argv | cwd |
| ------- | ---------- | ---- | --- |
| `resume claude:<id>` | `claude` | `["--resume","<id>"]` | alpha workspace |
| `resume codex:<id>` | `codex` | `["resume","<id>"]` | `unicode-β` workspace |
| `fork claude:<id>` | `claude` | `["--resume","<id>","--fork-session"]` | alpha workspace |
| `fork codex:<id>` | `codex` | `["fork","<id>"]` | `unicode-β` workspace |

Dry-runs started no process (`pgrep` = 0 for both vendors) and left both vendor
source fingerprints byte-identical.

**`last`:** `--project` filters selected the correct controlled session for each
project (alpha → Claude reference, `unicode-β` → Codex reference) with the
correct plan. Unfiltered `last` and `--agent claude` selected this session's own
live Claude session, which is genuinely the newest record on this machine —
correct behaviour, recorded as a deviation in section 12 rather than asserted
against the controlled corpus.

**Claude native resume (row 20):** launched via `rein resume` from
`/private/tmp`, i.e. **not** the recorded workspace. Exit 0, exact challenge
marker returned. Workspace proof: the only `cwd` value recorded anywhere in the
session is the alpha workspace, no vendor project directory was created for the
launch directory, and the session file was appended in place (still one file).

**Codex native resume (row 21):** driven through a detached GNU `screen`
session attached from a real Terminal window, launched from `/private/tmp`. The
TUI reported `directory:` as the recorded `unicode-β` workspace, replayed the
controlled history, and returned the exact challenge marker. The vendor session
file grew 63,175 → 113,372 bytes and contains the resume marker. `rein resume`
waited for the native child and returned its exit code 0.

**Claude fork (row 22):** produced a **distinct** native identity
`3f4e4069-…`. The source file was byte-identical before and after
(size and SHA-256 prefix unchanged) and does **not** contain the fork marker.
Refresh discovered source and fork as separate records. The fork resumed
independently through its vendor path and returned its own marker, again
leaving the source byte-identical — branch independence proven in both
directions.

**Codex fork (row 23):** the vendor printed `Thread forked from
019fb663-…`, preserved controlled history, and created a **distinct** new
session `019fb679-…`. Source byte-identical, without the fork marker. Both
were discovered by refresh; the fork resumed independently and the source
remained byte-identical. Child exit 0 propagated.

**Refresh (row 16):** the appended resume marker became searchable after
refresh and message counts advanced (Claude 4 → 6, Codex 5 → 10); each session
kept exactly one stable composite identity. Newly created fork sessions were
discovered and became the newest records within the controlled corpus.

**Failure paths:**

| Case | Exit | Behaviour |
| ---- | ---- | --------- |
| Missing reference (`resume`, dry-run and real) | 2 | `session not found: claude:<id>`, no launch |
| Missing reference (`inspect`) | 2 | same actionable error |
| Bare unique native ID | 0 | resolved to the correct composite reference |
| Missing executable (`PATH` without vendor) | 5 | `native agent executable is unavailable: …not found in $PATH`, fails before launch |
| Missing workspace (own disposable project renamed, then restored) | 5 | `recorded session workspace is unavailable: …no such file or directory`, fails before launch |
| `--json` without `--dry-run` — `resume` | 2 | `--json requires --dry-run for native agent launches` |
| `--json` without `--dry-run` — `fork` | 2 | same |
| `--json` without `--dry-run` — `last` | 2 | same |
| Child failure propagation | non-zero | with a stub executor exiting 42, `rein` exited 1 and surfaced `native resume failed: exit status 42` |

Ambiguous bare-ID resolution was not manufactured physically, because doing so
would require altering a real vendor session file. Per runbook §10 it is proved
by the deterministic injected-record test
`TestMissingAndAmbiguousReferencesNeverLaunch` (PASS), which asserts no launch
occurs.

## 9. Interactive switcher

Driven through a real pty (`pty.fork`, line-oriented picker) for **both**
binary names.

| Case | Result | Evidence |
| ---- | ------ | -------- |
| Bare `rein` on a TTY | PASS | refreshes and renders `Local sessions` with a numbered list and the prompt `Choose NUMBER, /text, i NUMBER, f NUMBER, or q:` |
| Bare `reinstate` on a TTY | PASS | identical picker and behaviour |
| `/text` filter | PASS | filtering by a controlled marker narrowed 84 rows to exactly the 2 sessions carrying that marker (source + its fork); the Codex marker likewise narrowed to its 2 |
| `i NUMBER` inspect | PASS | inspected the exact displayed reference with the same bounded metadata and capped preview policy |
| `NUMBER` resume | PASS | launched `claude` with argv `[--resume] [<exact displayed id>]` in the recorded alpha workspace |
| `f NUMBER` fork | PASS | launched `codex` with argv `[fork] [<exact displayed id>]` in the recorded `unicode-β` workspace |
| Invalid input | PASS | `zzz-invalid` → `Enter a number, /text, i NUMBER, f NUMBER, or q.`; recoverable, list redisplayed, **no default session chosen** |
| Out-of-range number | PASS | `i 999` → `Invalid session number.`; recoverable, no default chosen |
| `q` cancel | PASS | exit 0, no launch, no mutation |
| EOF (Ctrl-D) | PASS | exit 0, no launch, no mutation |
| Interrupt (Ctrl-C) | PASS | terminated by SIGINT, no launch, no mutation |
| Non-TTY | PASS | piped empty stdin → exit **2** immediately for both `rein` and `reinstate`, with a `rein sessions --json` hint; no hang, no silent newest-session selection |

Picker rows display only number, agent, project label, and native ID. No
unrelated transcript body was disclosed. Full picker output is deliberately not
reproduced here because it enumerates unrelated sessions.

Row/fork targeting was captured with an argv-recording stub executor so the
exact executable, argv array, and `cwd` could be proved without launching real
agents from inside the picker; real same-vendor resume and fork are separately
proved physically in section 8.

## 10. Read-only adapters

- Gemini CLI: `NOT_INSTALLED` — optional physical row 28 is `NOT TESTED`.
- OpenCode: `NOT_INSTALLED` — optional physical row 29 is `NOT TESTED`.
- No `gemini` or `opencode` records appeared in the index (agent counts:
  codex 47, claude 37), consistent with neither vendor being present.
- Fixture/fake-runner gates are green and remain mandatory:
  `TestGeminiSourceAppliesMetadataAndRewindReadOnly`,
  `TestGeminiSourceSupportsLegacyConversationJSON`,
  `TestOpenCodeSourceUsesDocumentedJSONCommand`,
  `TestOpenCodeSourceCommandFailuresAreNonDestructive`,
  `TestOpenCodeSourceMissingExecutableIsOptional`,
  `TestOpenCodeSourceRejectsOversizedCommandOutput` — all PASS.
- **Exit-5 refusal (row 30):** proved by the deterministic injected-record gate
  `TestNativeLaunchRefusalsUseStableCodes` (PASS), which injects a read-only
  Gemini record and asserts `fork gemini:<id> --dry-run` returns
  `ExitCompatibility` with the read-only reason and no launch.
  `internal/exitcode/codes.go` defines `Compatibility = 5`.
  `TestPlanLaunchRejectsReadOnlyAndMissingWorkspace` (PASS) covers the plan
  layer. Neither vendor was installed solely for this run.

No native or mutation support was inferred from read-only indexing.

## 11. Mandatory matrix

Windows column intentionally left blank — this device does not assert peer
results.

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | PASS (§2) | |
| 2 | Full local verification and required cross-builds | PASS (§5) | |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | PASS (§3) | |
| 4 | `rein sessions` discovers exact Claude sessions | PASS (§4, §6) | |
| 5 | `rein sessions` discovers exact Codex sessions | PASS (§4, §6) | |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | PASS (§6) | |
| 7 | Derived index path, rebuild, idempotency, and private permissions | PASS (§3, §6) | |
| 8 | Prompt-fragment literal search | PASS (§7) | |
| 9 | Agent filter | PASS (§7) | |
| 10 | Project filter | PASS (§7) | |
| 11 | Branch filter | PASS (§7) | |
| 12 | File filter | PASS (§7) | |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | PASS (§7) | |
| 14 | `sessions` and `search` do not dump transcript passages | PASS (§7) | |
| 15 | `inspect` metadata/160-code-point user preview policy | PASS (§7) | |
| 16 | Append/new-session refresh and no-change idempotency | PASS (§6, §8) | |
| 17 | `last` selects the correct resumable session and filters | PASS (§8) | |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | PASS (§8) | |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | PASS (§8) | |
| 20 | Claude native resume | PASS (§8) | |
| 21 | Codex native resume | PASS (§8) | |
| 22 | Claude vendor-native fork, source preserved | PASS (§8) | |
| 23 | Codex vendor-native fork, source preserved | PASS (§8) | |
| 24 | Missing/ambiguous reference and missing executor fail safely | PASS (§8) | |
| 25 | JSON/native-child separation and child failure propagation | PASS (§8) | |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | PASS (§9) | |
| 27 | Non-TTY prompt failure is immediate and actionable | PASS (§9) | |
| 28 | Gemini read-only physical path, when installed | NOT TESTED — `NOT_INSTALLED` (§10) | |
| 29 | OpenCode read-only physical path, when installed | NOT TESTED — `NOT_INSTALLED` (§10) | |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | PASS (§10, injected-record gate) | |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | PASS (§5, §6) | |
| 32 | Phase 1 automated regression remains green | PASS (§5) | |

Row 31 evidence: `TestScanJSONLinesBoundsAndToleratesConcurrentTail`,
`TestSafeTextRemovesTerminalControlsAndCollapsesWhitespace`,
`TestBuildSearchTextIsBoundedAndValidUTF8`,
`TestIndexRefreshContinuesAfterSourceFailureWithoutDeletingOldRows`,
`TestStoreReplaceSearchResolveDeleteAndPermissions`,
`TestStoreRebuildsCorruptAndIncompatibleDerivedState`,
`TestStoreSerializesConcurrentRefreshAndSearch`,
`TestOpenCodeSourceRejectsOversizedCommandOutput`,
`TestBoundedCommandOutputDrainsWithoutRetainingOverflow` — all PASS, plus the
physical corrupt-index rebuild and oversized-record handling in §6.

## 12. Findings

### Release-blocking

None identified from macOS evidence. Cross-device reconciliation subsequently
raised **one** release-blocking finding that macOS alone could not detect: the
unresolved question of whether Reinstate's Windows native launch gives the
vendor child a usable console/stdin, or whether the Windows harness simply
could not deliver input. See section 15.2.

### Non-blocking

1. **Installed Codex 0.146.0 exceeds this commit's
   `maximumVerifiedCodexVersion = 0.145.0`.** Codex was already at 0.146.0
   before this run began; it was deliberately not changed, because updating or
   downgrading would alter what is under test. All Phase 2 local behaviour —
   discovery, search, inspect, native resume, native fork — worked correctly
   against 0.146.0, and Phase 2 local capability flags are set by phase
   contract rather than gated on the vendor version. The compatibility ceiling
   is a Phase 1 sync-write concern and no Phase 2 gate depends on it, but the
   verified range should be re-evaluated before a release that claims 0.146.0
   support.
2. **Child-failure exit code is generalized.** With a stub executor exiting 42,
   `rein resume` exited **1** rather than mirroring 42, while correctly
   surfacing `native resume failed: exit status 42` in the error. The runbook
   requires propagation of failure, not mirroring of the exact code, so this
   passes as specified; flagged only in case exact-code mirroring is intended.
3. **Bounded-record warnings are noisy on a large real corpus.** Indexing
   emitted repeated `ignored JSONL record … larger than 4194304 bytes` warnings
   for unrelated oversized vendor sessions. This is the documented protection
   working correctly; it is cosmetic only.

### Test-harness deviations

1. **Live sessions perturb "newest record" and no-change idempotency.** This
   session's own Claude Code session is indexed and continuously appended, and
   one unrelated live Codex session was also being written during the run.
   Between two consecutive scans exactly those two records changed
   (`updated_at`, `size_bytes`, `message_count`); the composite key set was
   identical and every controlled record was byte-identical. Idempotency and
   "newest eligible record" were therefore asserted against the controlled
   corpus, and this is recorded as a deviation — not silently converted to
   PASS. Unfiltered `rein last` correspondingly selected the live session.
2. **Ambiguous bare-ID and read-only exit-5 use injected-record gates.**
   Manufacturing physical ambiguity would require editing a real vendor session
   file, and neither read-only vendor is installed. Runbook §10 and §11
   explicitly sanction the deterministic injected tests for these two cases.
3. **Codex TUI automation.** `codex resume` / `codex fork` are interactive TUIs
   that cannot be driven from a bare pty. They were driven through a detached
   GNU `screen` session attached from a real Terminal window, with all effects
   verified against the vendor session file rather than the screen. macOS
   `screen` 4.00.03 does not support `-Logfile`, so screen logging was not used.
4. **No `timeout(1)` on macOS.** A small perl `alarm` wrapper was used instead.
5. **Picker resume/fork targeting used an argv-recording stub executor** so the
   exact executable/argv/cwd could be captured without launching interactive
   agents from within the picker. Real same-vendor resume and fork were proved
   physically and separately (§8).
6. **Vendor retries:** none required. No `API Error: 529` occurred during this
   run.
7. An initial cross-build loop and an initial pty driver were each written
   incorrectly by the harness (zsh word-splitting; input sent before the picker
   was ready). Both were corrected and re-run; only the corrected runs are
   reported as evidence.
8. **Permission-bypass mode was active for the whole run — a violation of the
   acceptance preconditions.** The acceptance prompt required normal sandboxing
   and approval controls, and the runbook lists disabling them under "Never".
   Evidence: the user-level settings carry
   `skipDangerousModePermissionPrompt: true` with no `permissions` policy, and
   **zero** approval prompts occurred across roughly fifty-five tool calls that
   included `osascript` driving Terminal.app, `mv` on a path under `$HOME`,
   `git push`, and `gh pr create` — operations that would be gated under the
   default mode. No individual call used a sandbox-disabling parameter; the
   bypass was the session-level mode, which the operator selected and the agent
   did not re-enable.

   **Affected-gate analysis.** No Reinstate verdict in this report depends on
   the agent's own approval mode: Reinstate ran as an ordinary child process
   under normal OS user permissions, and every product assertion rests on
   observable artifacts — exit codes, argv arrays, `cwd`, file fingerprints,
   permission bits, and controlled markers — none of which the harness's
   approval mode can alter. The concrete loss is a **safety control, not an
   evidence control**: the destructive-adjacent steps in this run (renaming and
   restoring the disposable `unicode-β` project for the missing-workspace gate,
   overwriting the derived index for the corrupt-rebuild gate, and stub-`PATH`
   executions) all ran without an external approval gate. Each was verified
   afterwards to have been scoped and reversed — the project was restored and
   the index rebuilt — and no vendor session file, unrelated project, or older
   Reinstate home was modified or deleted, as the final integrity sweep in
   section 3 shows. The run should nonetheless be treated as not having met its
   stated safety preconditions, and any re-run of this matrix should execute
   with normal controls enabled.

## 13. Repository hygiene

- report-only branch: `test/phase2-b7b45db014ed-macos-report`
- tested base commit: `b7b45db014edf030d820e503ee23b579c5032e69`
- changed files: `docs/testing/results/2026-07-31-macos-phase2-b7b45db014ed.md` (added, 1 file)
- private/local artifacts excluded: no local index, vendor session, fixture
  copy, generated binary, transcript, log, or private path is committed
- product code unchanged: `true` — no file outside `docs/testing/results/` is touched
- secrets/transcripts committed: `false`

## 14. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=macos
test_commit=b7b45db014edf030d820e503ee23b579c5032e69
reinstate_version=v0.1.0-36-gb7b45db
report_path=docs/testing/results/2026-07-31-macos-phase2-b7b45db014ed.md
report_branch=test/phase2-b7b45db014ed-macos-report
claude_ref=claude:70bda718-665a-4eb5-a624-ca66ad382c8a
codex_ref=codex:019fb663-60eb-7591-be41-6b1b211fa37f
gemini_state=NOT_INSTALLED
opencode_state=NOT_INSTALLED
required_pass=30
required_partial=0
required_fail=0
required_not_tested=0
optional_physical_pass=0
optional_physical_not_tested=2
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

This block is the unchanged **device-level** record for macOS: its counts are
this device's own 30 required rows, and `release_blocking_findings=0` means no
release-blocking finding was detectable from macOS evidence alone. The
cross-device figure is carried by `PHASE2-FINAL-RECONCILIATION-V1` in section
15, which records `release_blocking_findings=1` and `phase2_status=FAIL`.

## 15. Final cross-device reconciliation

Performed by the macOS coordinator after independently validating the
native-Windows report. Peer PASS and FAIL labels were not taken on trust, and
neither were this device's own.

### 15.1 Windows report validation

| Check | Result | Evidence |
| ----- | ------ | -------- |
| Branch tip matches the stated commit | PASS | `origin/test/phase2-b7b45db014ed-windows-report` = `61c9b1ebeca0279356fc95a90ecd5cc5afe1f982` |
| Report-only diff from the tested commit | PASS | one added file: `docs/testing/results/2026-07-31-windows-phase2-b7b45db014ed.md` |
| Branch starts at the tested commit | PASS | merge-base = `b7b45db014edf030d820e503ee23b579c5032e69` |
| Complete terminated milestone block, `device=windows` | PASS | `PHASE2-DEVICE-REPORT-V1` … `END-PHASE2-DEVICE-REPORT-V1` |
| No secret, transcript, or absolute private path | PASS | scan of the committed file found no `C:\Users\…`, `/Users/…`, username, key, or token pattern |
| Matrix tally matches its own counts | PASS | committed matrix parses to 23 PASS / 3 PARTIAL / 4 FAIL / 2 NOT TESTED, matching its block |

**Inconsistencies found between the Windows prose, matrix, and milestone
block.** All four are label inflation in the block relative to that report's
own evidence:

1. `picker=FAIL` contradicts matrix row 26 = `PARTIAL` and its own section 10,
   which records 7 PASS and 3 PARTIAL sub-gates and no failing sub-gate.
2. `configless_local_only=FAIL` contradicts matrix row 3 = `PARTIAL` and its
   own section 3, which states the prescribed home was clean before and after,
   contained only the derived cache, and was owner-only. The contamination was
   a *separate* accidental cache created under the user profile by a PowerShell
   reserved-`$home` harness bug — a harness violation, not Reinstate requiring
   configuration.
3. `gemini_state=FAIL` and `opencode_state=FAIL` contradict matrix rows 28–29 =
   `NOT TESTED` and the prose, which attributes both to vendor-side failures
   (Gemini authentication, an OpenCode `no such column: name` schema error).
   Neither is a Reinstate failure. The template offers no `NOT_TESTED` value
   for these fields, which is a template gap worth fixing.
4. `release_blocking_findings=6` overstates. Of R1–R6, only R1 and R2 concern
   product behaviour at all; R3 is a harness bug, R4 is the live-session
   artefact both devices observed, R5 is an evidence-acquisition limit, and R6
   is a vendor-version note also raised non-blocking by this device.

### 15.2 The central divergence: rows 20–23

Windows marked native resume and vendor-native fork `FAIL` for both agents
against the identical commit this device passed. Applying the required
"the product did the wrong thing" versus "the evidence could not be obtained"
test, the Windows evidence supports the second, so these are reclassified
**NOT TESTED**, not `FAIL`:

- In all four cases the Windows report confirms Reinstate performed its whole
  responsibility correctly: *"Exact UI and argv launch"*, *"Exact Codex TUI
  detected"*, correct executable, argv array, and recorded `cwd`. Reference
  resolution, preflight, plan construction, and launch all succeeded.
- The single missing element in every case is the **challenge response** —
  i.e. the vendor agent never processed any input. That is the one link in the
  chain owned by the Windows console-input harness, and that harness is
  documented as unreliable throughout the same report: two console probes
  timed out at exit 124, and Windows Terminal UI Automation *"did not expose
  rendered rows"* and left *"rendered inspect state unavailable"* for the
  picker rows, which use the same input and observation channel.
- *"Source unchanged"* and *"no new session file / no new rollout"* follow
  necessarily from a vendor that was launched and then closed with Ctrl+C
  without ever receiving a prompt. They are not independent evidence of
  breakage. This device confirms the mechanism directly: the Codex fork rollout
  only materialised **after** a prompt was delivered to the forked thread.
- `source preserved` is the *required* behaviour for a fork, and Windows
  observed it in both fork rows — so those rows record Reinstate doing the
  right thing on the one property they could actually measure.
- Claude resume returning exit 1 twice is consistent with correct propagation
  of a child the harness killed with Ctrl+C, which is the contracted behaviour,
  not a defect.

**Scepticism in the other direction — the unresolved risk.** The Windows
evidence cannot *exclude* a genuine Windows-only defect in which Reinstate
launches the vendor without a usable console/stdin, which would look identical
to harness input failure. Two details keep this open rather than closed: the
Windows process trees show a `cmd.exe` shim between Reinstate and the vendor
(`claude`/`codex` resolve to `.cmd` shims on Windows), and that shim layer is a
plausible place for standard-input inheritance to break; and the Windows report
records the shim *"remained at a batch confirmation"*, meaning the shim itself
was interactive. Reinstate's own console I/O demonstrably worked on Windows —
the picker accepted line input, cancel, and Ctrl+C — but that proves nothing
about whether the *child* inherits a usable console. This device cannot
discriminate the two hypotheses from macOS, and neither can CI, which does not
drive interactive vendor TUIs. It is recorded as the one release-blocking
finding and requires a targeted Windows re-run.

### 15.3 Rows this device passed that the peer evidence puts in question

- **Row 26.** Both devices proved picker *targeting* by observing the launched
  child rather than by completing a real vendor conversation from inside the
  picker — this device used an argv-recording stub, Windows detected the child
  process and argv. End-to-end picker-initiated vendor conversation is
  therefore unproven on *both* devices. This does not change row 26's macOS
  verdict, because the runbook asks that fork and resume "target the exact
  displayed composite reference", which both devices demonstrated, but the
  limitation is now stated explicitly rather than left implicit.
- **Row 6.** Windows' `PARTIAL` conflates two distinct properties: alias parity
  (`rein` versus `reinstate`) and cross-scan idempotency. Its evidence
  ("one unrelated active Codex record changed only `updated_at` and
  `size_bytes` between scans") concerns the second. This device measured alias
  parity within a single scan window and found **zero** differing records
  across all 82, so the alias-parity property is genuinely proven on macOS and
  is not undermined; the cross-scan drift is the live-session artefact both
  devices independently observed and recorded as a deviation.
- **Row 3.** Windows exercised a path this device did not: what Reinstate does
  when `REINSTATE_HOME` is effectively inherited from the profile. It created a
  derived cache in the default home and nothing else — correct behaviour — so
  no macOS verdict is weakened. This device's row 3 rests on a home proven
  empty beforehand and containing only the derived cache afterwards.
- No macOS row was found to rest on evidence the Windows run contradicts.

### 15.4 Independently verified CI for the tested commit

Queried from the forge directly, not from either report. All six check runs on
`b7b45db014edf030d820e503ee23b579c5032e69` are `completed` / `success`:
`Test (macos-latest)`, `Test (ubuntu-latest)`, `Test (windows-latest)`, `Lint`,
`Website`, `Security`. The legacy combined-status endpoint reports
`state=pending, total_count=0`; that endpoint is unused by this repository and
the Checks API is authoritative. Note that CI's Windows job proves the
automated Go gates on Windows — it does **not** exercise interactive vendor
TUIs, so it cannot settle rows 20–23.

### 15.5 Reconciled 32-row result

`M` = macOS physical, `W` = Windows physical, `A` = automated/injected gate,
`CI` = forge check runs.

| # | Reconciled | Physical evidence source | Note |
| - | ---------- | ------------------------ | ---- |
| 1 | PASS | M + W | both built the exact commit; both binaries report it |
| 2 | PASS | M + W + CI | both ran the full gate set and four cross-builds |
| 3 | PASS | M + W | Windows `PARTIAL` reflects a harness bug outside the prescribed home; the product requirement was met on both |
| 4 | PASS | M + W | |
| 5 | PASS | M + W | |
| 6 | PASS | M + W | alias parity proven on both; Windows `PARTIAL` conflated it with cross-scan drift (§15.3) |
| 7 | PASS | M + W | both rebuilt a corrupted derived index with owner-only permissions and no vendor mutation |
| 8 | PASS | M + W | |
| 9 | PASS | M + W | |
| 10 | PASS | M + W | |
| 11 | PASS | M + W | |
| 12 | PASS | M + W | both had a real structured file reference |
| 13 | PASS | M + W | |
| 14 | PASS | M + W | |
| 15 | PASS | M + W | previews 160/101 (M) and 160/104 (W) code points, sanitised |
| 16 | PASS | M + W | both recorded the live-session artefact as a deviation |
| 17 | PASS | M + W | |
| 18 | PASS | M + W | exact argv/cwd, no launch, no mutation |
| 19 | PASS | M + W | exact argv/cwd, no launch, no mutation |
| 20 | **macOS PASS / Windows NOT TESTED** | M only | reclassified from Windows `FAIL` (§15.2) |
| 21 | **macOS PASS / Windows NOT TESTED** | M only | reclassified from Windows `FAIL` (§15.2) |
| 22 | **macOS PASS / Windows NOT TESTED** | M only | reclassified from Windows `FAIL` (§15.2) |
| 23 | **macOS PASS / Windows NOT TESTED** | M only | reclassified from Windows `FAIL` (§15.2) |
| 24 | PASS | M + W + A | ambiguity by injected gate on both devices |
| 25 | PASS | M + W | both proved the three `--json` refusals and child-failure propagation |
| 26 | **macOS PASS / Windows PARTIAL** | M + W | Windows proved targets, cancel, interrupt, non-TTY; filter/inspect/invalid rendering NOT TESTED there |
| 27 | PASS | M + W | exit 2, immediate, with hint, on both binary names |
| 28 | NOT TESTED (both) | — | macOS: not installed. **Windows: installed but vendor-side failure** — see §15.6 |
| 29 | NOT TESTED (both) | — | macOS: not installed. **Windows: installed but vendor-side failure** — see §15.6 |
| 30 | PASS | A on both | injected-record gate asserts exit 5 and zero launch |
| 31 | PASS | A on both + M | |
| 32 | PASS | A on both + CI | Phase 1 packages green on both devices and all three CI OSes |

Required dual-device rows fully passed: **25 of 30**. Outstanding: rows 20, 21,
22, 23 (Windows evidence not obtained) and row 26 (Windows partial).

### 15.6 Rows 28–29 do not meet the runbook's literal escape clause

Section 15 of the runbook permits rows 28–29 to be `NOT TESTED` *"because the
vendor is not installed"*. That holds on macOS, where neither vendor is
present. It does **not** hold on Windows, where Gemini CLI 0.38.0 and OpenCode
1.18.2 were both installed but could not produce a controlled session — Gemini
could not authenticate and OpenCode failed with `no such column: name`. The
correct label remains `NOT TESTED`, since the evidence could not be obtained
and neither failure implicates Reinstate, but the acceptance criterion as
written is not satisfied on Windows. This needs either a Windows re-run once
those vendors work, or an amendment to the runbook wording to cover
"installed but non-functional". It is not a Reinstate defect and is not counted
as release-blocking.

### 15.7 Verdict

**Phase 2 does not pass physical acceptance.** No confirmed Reinstate defect
was found on either device, and every gate that both devices could actually
measure agrees. Acceptance fails on evidence coverage, not on demonstrated
misbehaviour: the runbook requires rows 1–27 and 30–32 to pass on **both** real
devices, and native resume and vendor-native fork have physical evidence from
macOS only.

To close Phase 2, on **native Windows** only:

1. Re-run rows 20–23 with a console-input mechanism that can actually deliver a
   prompt to an interactive vendor TUI, and confirm a controlled challenge
   response plus a distinct fork identity. This must also settle whether the
   `cmd.exe` shim breaks standard-input inheritance for the launched vendor —
   if it does, that is a genuine product defect and Phase 2 fails on merit
   rather than on coverage.
2. Re-run the row 26 sub-gates whose rendered state Windows Terminal UI
   Automation could not expose (`/text` filter, `i NUMBER` inspect, invalid
   input), by asserting against process and file effects rather than rendered
   text if UI Automation remains unavailable.
3. Optionally re-run rows 28–29 once Gemini and OpenCode are functional, or
   amend the runbook to cover the installed-but-non-functional case.

Nothing needs to be re-run on macOS for product reasons. The macOS matrix
should, however, be re-executed with normal approval controls enabled before
this run is cited as a compliant acceptance record (section 12, deviation 8).

Both report PRs remain draft and unmerged. No tag, release, or cleanup was
performed.

```text
PHASE2-FINAL-RECONCILIATION-V1
test_commit=b7b45db014edf030d820e503ee23b579c5032e69
mac_report_commit=f56b5a44c5910ae72348f5c356a2e40c9f67a02e
windows_report_commit=61c9b1ebeca0279356fc95a90ecd5cc5afe1f982
mandatory_rows=32
required_dual_device_rows_passed=25
optional_rows_28_29_macos=NOT_TESTED
optional_rows_28_29_windows=NOT_TESTED
automated_gates=PASS
physical_macos=PASS
physical_windows=FAIL
release_blocking_findings=1
phase2_status=FAIL
END-PHASE2-FINAL-RECONCILIATION-V1
```

`mac_report_commit` names the macOS **device-report** commit
`f56b5a44c5910ae72348f5c356a2e40c9f67a02e`; the finalisation commit that adds
this reconciliation section is its child on the same report-only branch and
cannot name its own hash.

`physical_windows=FAIL` records that Device B did not obtain the required
physical evidence for rows 20–23 and 26; it does not assert that Reinstate
misbehaved there. `release_blocking_findings=1` is the unresolved Windows
native-execution question in §15.2, which blocks any release claiming Windows
native resume or fork until discriminated.
