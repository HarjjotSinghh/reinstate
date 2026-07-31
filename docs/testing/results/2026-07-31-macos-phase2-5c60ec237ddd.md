# Phase 2 acceptance — macOS Device A report

**macOS device verdict:** `PASS`
**Reconciled Phase 2 programme status:** `FAIL` — not a product defect; Windows rows 3 and 6 are `PARTIAL` and were not re-executed at the tested commit (see section 12)
**Milestone:** `FINAL_RECONCILIATION`
**Required counts (macOS):** `30 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED`
**Optional physical counts (macOS):** `0 PASS / 2 NOT TESTED`

This report covers only the exact disposable sessions and paths created for
this run. No real transcript content or secret was used as evidence. All
session references below are controlled sessions created by this run.

Run ID: `20260731-5c60ec237ddd`.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-07-31T08:03Z` – `2026-07-31T08:45Z` |
| Device | `macOS Device A` |
| Tested Git commit | `5c60ec237ddded8e314cdb8c1449080ddc923395` |
| Signed tag, if any | none (development acceptance, not a published release) |
| Reinstate version JSON | `{"commit":"5c60ec2","date":"2026-07-31T08:03:12Z","name":"reinstate","version":"v0.1.0-38-g5c60ec2"}` |
| OS/version/build | macOS `26.5.2`, build `25F84` |
| Architecture | `arm64` |
| Native shell | `zsh 5.9` |
| Claude Code version/state | `2.1.220`, installed, full capability |
| Codex CLI version/state | `codex-cli 0.146.0`, installed, full Phase 2 local capability |
| Gemini CLI version/state | `NOT_INSTALLED` |
| OpenCode version/state | `NOT_INSTALLED` |
| Git version | `2.55.0` |
| Go version | `go1.26.5 darwin/arm64` (build pinned `GOTOOLCHAIN=go1.25.12`) |
| Report branch | `test/phase2-5c60ec237ddd-macos-report` |
| Draft PR | #65 — https://github.com/HarjjotSinghh/reinstate/pull/65 (draft, unmerged) |

Permission-mode compliance: the session ran under `--permission-mode manual`.
`bypassPermissions` was not active at any point, no permission-bypass flag was
used, and normal sandboxing and tool-approval controls stayed enabled for the
entire run. Ordinary approval prompts were expected and observed.

## 2. Provenance and repository hygiene

| Assertion | Result | Sanitized evidence |
| --------- | ------ | ------------------ |
| Tested commit matches the requested commit | `PASS` | `HEAD=5c60ec237ddded8e314cdb8c1449080ddc923395` |
| Binary reports the tested commit | `PASS` | `rein version --json` → `commit=5c60ec2`, `version=v0.1.0-38-g5c60ec2` |
| Product tree was clean before testing | `PASS` | `git status --porcelain` produced `0` lines |
| Origin fetched; reachability recorded | `PASS` | reachable from `origin/feat/phase2-local-index`; **not** reachable from `origin/main` |
| Report branch starts at the tested commit | `PASS` | worktree `HEAD` = tested commit; branch created directly from it |
| Report is the only committed change | `PASS` | section 13 |
| No secret/transcript/private path was committed | `PASS` | section 13 |

No merge, rebase, cherry-pick, vendor update, vendor downgrade, tag, release,
or product-code edit was performed.

## 3. Isolation and local-only proof

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Fresh isolated `REINSTATE_HOME` | `PASS` | `~/.reinstate-phase2-acceptance-<RUN_ID>` did not exist before the run |
| Fresh disposable projects | `PASS` | both target project paths absent before creation |
| No `rein init` run | `PASS` | `init` was never invoked |
| No `config.toml` or sync state created | `PASS` | `config.toml`, `state.json`, `backups/` all absent before and after |
| No credential/passphrase/keyring request | `PASS` | no prompt or request from any Phase 2 command |
| No backend/network dependency | `PASS` | no sync/storage/conflict command run; all Phase 2 commands succeeded configless |
| Only derived index state created | `PASS` | full home tree after run: `./cache`, `./cache/session-index-v1.sqlite` — nothing else |
| Index and parent permissions are owner-only | `PASS` | home `drwx------`, `cache/` `drwx------`, db `-rw-------` |
| Corrupt index rebuilds cleanly | `PASS` | derived db overwritten with non-SQLite bytes → next scan exit `0`, `96`-record listing restored, permissions still `-rw-------` |

No `R2.txt`, sync profile, storage environment value, credential, encryption
passphrase, or keyring entry was created or used.

## 4. Controlled corpus

Two disposable Git repositories were created with distinct branches and
controlled relative files:

| Project label | Branch | Controlled relative file |
| ------------- | ------ | ------------------------ |
| `…-acceptance-<RUN_ID>-alpha` | `phase2/5c60ec237ddd-alpha` | `src/phase2_5c60ec237ddd_alpha.txt` |
| `reinstate phase2 <RUN_ID> unicode-β` | `phase2/5c60ec237ddd-beta` | `src/phase2_5c60ec237ddd_beta.txt` |

Controlled sessions (all created by this run):

| Role | Composite reference | Project label | Marker found | Capability |
| ---- | ------------------- | ------------- | ------------ | ---------- |
| Claude source | `claude:037ebc04-f5ea-4de3-931e-0aa0517b183e` | alpha | `…-CLAUDE-A1` | full |
| Codex source | `codex:019fb734-e14b-71c2-9141-c44dbcb35781` | unicode-β | `…-CODEX-B1` | full |
| Claude fork (CLI) | `claude:6c8583c6-f5ce-44e8-9234-2e4d63347f01` | alpha | `…-CLAUDE-FORK-F1` | full |
| Codex fork (CLI) | `codex:019fb73f-15c8-7a53-afe5-32f2f31bd249` | unicode-β | `…-CODEX-FORK-G1` | full |
| Claude fork (picker) | `claude:90ea4950-e2ed-4450-88ed-a4355898d72e` | alpha | `…-PICKER-FORK-K1` | full |
| Newest controlled session | `claude:3b817f2b-1b78-4892-8af7-2e00bf22fea5` | unicode-β | `…-NEW-C1` | full |
| Gemini | — | — | — | `NOT_INSTALLED` |
| OpenCode | — | — | — | `NOT_INSTALLED` |

Controlled markers use the form `REINSTATE-PHASE2-5c60ec237ddd-MACOS-<SLOT>`.
Marker presence was asserted as "at least one exact occurrence"; no fixed
serialization count was required. No vendor session file was ever edited to
manufacture indexing, ambiguity, malformed input, or a fork.

**Live-record disclosure.** The macOS device also hosts the operator's own live
Claude Code session, whose vendor file changes continuously during scans and
which legitimately contains the controlled marker strings because they were
typed into it. All determinism, ordering, and idempotency assertions in this
report are made against the controlled corpus with that live record excluded;
where a search returned one extra row, it was that live record and it is
identified as such rather than counted as a controlled match.

## 5. Automated verification

All gates were run from the tested commit with uncached tests (`-count=1`)
where the runbook requires them.

| Gate | Result | Sanitized evidence |
| ---- | ------ | ------------------ |
| `make fmt-check` | `PASS` | exit `0` |
| Focused `./internal/sessionindex ./internal/cli ./internal/adapter/...` | `PASS` | exit `0`; 5 packages `ok` |
| Full `go test -count=1 ./...` | `PASS` | exit `0`; `21` packages `ok`, `0` `FAIL` |
| Race `go test -race -count=1 ./...` | `PASS` | exit `0` |
| `make vet` | `PASS` | exit `0` |
| `make verify` | `PASS` | exit `0` |
| Cross-build `windows/amd64` | `PASS` | exit `0` |
| Cross-build `darwin/arm64` | `PASS` | exit `0` |
| Cross-build `darwin/amd64` | `PASS` | exit `0` |
| Cross-build `linux/amd64` | `PASS` | exit `0` |
| Phase 1 regression (`crypto`, `pathmap`, `sync`, `adapter/...`, `cli`) | `PASS` | exit `0`, uncached |

No gate was skipped or interrupted. There were no failed automated commands.

Named deterministic coverage confirmed by test name (all `PASS`):
`TestPhase2MacOSWindowsAndWSLFixtures`, `TestCommittedPlatformFixturesDiscover`,
`TestScanJSONLinesBoundsAndToleratesConcurrentTail`,
`TestFingerprintDetectsConcurrentWrite`,
`TestStoreRebuildsCorruptAndIncompatibleDerivedState`,
`TestStoreReplaceSearchResolveDeleteAndPermissions`,
`TestStoreSerializesConcurrentRefreshAndSearch`,
`TestIndexRefreshContinuesAfterSourceFailureWithoutDeletingOldRows`,
`TestRedactSecretsAndPaths`, `TestGeminiSourceAppliesMetadataAndRewindReadOnly`,
`TestGeminiSourceSupportsLegacyConversationJSON`,
`TestOpenCodeSourceUsesDocumentedJSONCommand`,
`TestOpenCodeSourceRejectsOversizedCommandOutput`,
`TestOpenCodeSourceMissingExecutableIsOptional`,
`TestOpenCodeSourceCommandFailuresAreNonDestructive`,
`TestNativeLaunchRefusalsUseStableCodes`,
`TestMissingAndAmbiguousReferencesNeverLaunch`,
`TestPlanLaunchRejectsReadOnlyAndMissingWorkspace`,
`TestPlanLaunchExactNativeArgv`, `TestExecLaunchRunnerClassifiesPreflightFailures`,
`TestNativeChildFailurePropagates`,
`TestInteractivePickerFiltersAndLaunchesExactSelection`,
`TestInteractivePickerCancelAndNonTTY`,
`TestNativeClaudeLaunchThroughWindowsCommandShim` (Windows-tagged; executed by
this commit's `windows-latest` CI job — see section 12).

### Independent CI verification for the tested commit

All six required GitHub checks for `5c60ec2` were verified directly through the
GitHub API, not inherited from a peer report:

| Check | Status | Conclusion |
| ----- | ------ | ---------- |
| `Lint` | completed | `success` |
| `Security` | completed | `success` |
| `Website` | completed | `success` |
| `Test (ubuntu-latest)` | completed | `success` |
| `Test (macos-latest)` | completed | `success` |
| `Test (windows-latest)` | completed | `success` |

Workflow run `30606254993` reports `head_sha=5c60ec237ddded8e314cdb8c1449080ddc923395`,
`status=completed`, `conclusion=success`, `event=push`,
`branch=feat/phase2-local-index`. `total_count=6`.

## 6. Configless index and refresh

`rein sessions` succeeded from the fresh configless home with exit `0` and
discovered both controlled references.

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Discovers exact controlled Claude reference | `PASS` | present in listing |
| Discovers exact controlled Codex reference | `PASS` | present in listing |
| `rein` / `reinstate` JSON parity | `PASS` | both `93` records; canonicalised JSON byte-identical |
| Deterministic ordering | `PASS` | newest `updated_at` first, then agent, then native ID — verified pairwise across the whole array |
| Stable composite identity across runs | `PASS` | identity set identical between consecutive scans |
| No duplicate records on repeat | `PASS` | unique `(agent, id)` keys == record count |
| No-change refresh idempotency | `PASS` | two consecutive scans produced byte-identical canonical JSON (`96` records each) |
| Only derived cache state created | `PASS` | section 3 |
| `rein list` not redefined as `rein sessions` | `PASS` | distinct Phase 1 output format (agent / id / encoded project dir); not byte-identical to `sessions`; exit `0` |

The `rein list` result matches this commit's corrected runbook contract: it
still succeeds from a fresh configless home with its own distinct Phase 1
output, and it has not become an alias of `rein sessions`.

### Refresh behaviour

| Assertion | Result | Evidence |
| --------- | ------ | -------- |
| Appended metadata appears after refresh | `PASS` | markers `…-CLAUDE-A2` and `…-CODEX-B2` searchable after vendor append |
| Forks discovered by refresh | `PASS` | both fork references indexed alongside their preserved sources |
| New session becomes newest eligible record | `PASS` | controlled ordering after run: `3b817f2b` (newest) → `019fb73f` → `6c8583c6` → `019fb734` → `037ebc04` |
| No-change refresh is idempotent | `PASS` | identical canonical JSON on repeat |
| Concurrent vendor append does not corrupt source or index | `PASS` | sources remained parseable and indexed throughout; `TestFingerprintDetectsConcurrentWrite`, `TestScanJSONLinesBoundsAndToleratesConcurrentTail` |
| Incomplete final JSONL line ignored until complete | `PASS` (synthetic) | `TestScanJSONLinesBoundsAndToleratesConcurrentTail` — not manufactured on a real vendor file |

Reinstate did not modify any vendor source in order to index, search, or
inspect: the controlled Claude and Codex source digests were byte-stable across
the entire index/search/inspect/dry-run phase.

Oversized-record handling was also observed physically: the scanner emitted
bounded, sanitized `oversized_record` warnings on `stderr` and in the JSON
`warnings` array while still completing the scan. Those warnings name unrelated
session IDs, so only their count (`14` warning entries, of which `1` is
`agent_not_installed` for OpenCode) is recorded here.

## 7. Search and inspect

All search results below are controlled-row counts with the live record called
out explicitly. Streams were separated so `stderr` warnings never inflated a
count.

| Case | Command shape | Exit | Result |
| ---- | ------------- | ---- | ------ |
| Prompt fragment (Claude) | `search "<A1 marker>"` | `0` | `1` row — the controlled Claude session |
| Prompt fragment (Codex) | `search "<B1 marker>"` | `0` | `1` row — the controlled Codex session |
| Case-insensitive | same marker lowercased | `0` | `1` row — same controlled session |
| AND terms (hit) | `search "PHASE2-5c60ec237ddd MACOS-CLAUDE-A1"` | `0` | `2` rows — controlled Claude + live record |
| AND terms (miss) | adds a third absent term | `0` | `No matching local sessions.` |
| `--agent claude` | `search "…-MACOS" --agent claude` | `0` | `2` rows — controlled Claude + live record; no Codex rows |
| `--agent codex` | `search "…-MACOS" --agent codex` | `0` | `1` row — controlled Codex only |
| `--project` | `--project "unicode-β"` | `0` | `1` row — controlled Codex only |
| `--branch` | `--branch "5c60ec237ddd-alpha"` | `0` | `1` row — controlled Claude only |
| `--file` | `--file "phase2_5c60ec237ddd_alpha.txt"` | `0` | `1` row — controlled Claude only |
| `--limit 1` | `--limit 1` | `0` | exactly `1` row |
| Zero match | deliberate absent token | `0` | `No matching local sessions.` — honest empty, no fallback |
| Shell metacharacters | `…$(id); rm -rf /*; \`whoami\`` | `0` | `No matching local sessions.` — treated as literal query data |
| Unicode | `"unicode-β-5c60ec237ddd-nomatch"` | `0` | `No matching local sessions.` |

The `--file` filter resolved through a structured file reference produced by a
real Read-tool use inside the controlled Claude session. Matching was literal
and case-insensitive; multiple terms were ANDed; every filter narrowed
correctly; zero match was an honest empty result rather than an error or a
fallback to all sessions. No network request and no semantic/embedding service
was involved.

**Row 14.** `sessions` and `search` emit only columnar metadata (composite
reference, project label, branch, native ID). No transcript passage, assistant
message, reasoning, tool output, or prompt body was printed.

**Row 15 — inspect preview policy.** For both controlled references, human and
JSON inspection returned accurate identity, agent, timestamps, workspace,
project, branch, message count, capability metadata, and — for Claude — known
file references.

| Preview assertion | Claude | Codex |
| ----------------- | ------ | ----- |
| Preview length in Unicode code points | `160` | `160` |
| `<= 160` | `true` | `true` |
| Preview sourced only from the user-authored prompt | `true` | `true` |
| Whitespace collapsed | `true` | `true` |
| Control characters / ESC present | `false` | `false` |
| Assistant/reasoning/tool/transcript field present | `false` | `false` |

Capability metadata reported `resume=true fork=true` for both full-capability
agents. Terminal-control stripping is additionally covered by the deterministic
suite. No real secret was inserted to test this boundary.

## 8. Last, resume, and fork

### Dry-run plans (executed before every real launch)

| Plan | Executable | argv | argv is array | cwd | Launched |
| ---- | ---------- | ---- | ------------- | --- | -------- |
| `resume claude:…` | `claude` | `["--resume","037ebc04-…"]` | `true` | alpha project | no |
| `resume codex:…` | `codex` | `["resume","019fb734-…"]` | `true` | unicode-β project | no |
| `fork claude:…` | `claude` | `["--resume","037ebc04-…","--fork-session"]` | `true` | alpha project | no |
| `fork codex:…` | `codex` | `["fork","019fb734-…"]` | `true` | unicode-β project | no |

All four matched the runbook's mandatory argv tables exactly. Plans use an argv
array, never a shell string, and use the recorded workspace as `cwd`. After all
dry-runs, both controlled source digests were unchanged and no new vendor
session file existed.

### `last`

| Case | Result |
| ---- | ------ |
| `last --dry-run --json` | selected the newest resumable record and emitted a valid `claude` plan (exit `0`) |
| `last --agent claude --dry-run --json` | agent-filtered selection, `claude` executable |
| `last --project "…-alpha" --dry-run --json` | selected `claude:037ebc04-…`, argv `["--resume","037ebc04-…"]`, cwd = alpha project |
| `last --project "unicode-β" --dry-run --json` | selected `codex:019fb734-…`, argv `["resume","019fb734-…"]` |

### Real native resume

Both real resumes were launched from **outside** the recorded workspace, so a
correct `cwd` could only come from Reinstate's recorded workspace.

| Assertion | Claude | Codex |
| --------- | ------ | ----- |
| Reinstate launched the same vendor as the composite reference | `PASS` | `PASS` |
| Exit code | `0` | `0` |
| Controlled challenge response returned | `PASS` (`…-CLAUDE-A2`) | `PASS` (`…-CODEX-B2`) |
| Challenge response persisted to the correct source session | `true` | `true` |
| Vendor appended after launch (legitimate) | `11 → 17` lines | `19 → 32` lines |
| Reinstate mutated the session before launch | `false` | `false` |

Codex requires a real terminal. Its resume was therefore driven through a
detached `screen` session attached to a real `Terminal.app` window; the vendor
answered terminal capability queries and rendered its TUI. The controlled
challenge was injected into that console and the response was confirmed inside
the vendor's own session file as an `agent_message`/`assistant` record
containing the exact controlled marker.

### Real vendor-native fork

| Assertion | Claude | Codex |
| --------- | ------ | ----- |
| Dry-run performed no mutation | `true` | `true` |
| Same vendor invoked, in the recorded workspace | `true` | `true` |
| Vendor created a distinct native identity | `true` (`6c8583c6-…`) | `true` (`019fb73f-…`) |
| Source unmodified by Reinstate before launch | `true` (digest byte-stable) | `true` (digest byte-stable) |
| Fork inherited the source's controlled history | `true` (`…-A1` present) | `true` (`…-B1` present) |
| Refresh discovered both source and fork | `true` | `true` |
| Fork resumes independently through its vendor | `true` (`…-FORK-F2`) | `true` (`…-FORK-G2`) |
| Changing the fork did not change the source | `true` (source contains `0` fork-only markers) | `true` (source contains `0` fork-only markers) |

### Negative and safety gates

| Case | Exit | Result |
| ---- | ---- | ------ |
| Missing reference (`resume claude:…dead`) | `2` | `session not found: claude:…` — no launch |
| Unique bare native ID resolves | `0` | resolved to `claude`, argv `["--resume","037ebc04-…"]` |
| `resume --json` without `--dry-run` | `2` | `--json requires --dry-run for native agent launches` |
| `fork --json` without `--dry-run` | `2` | same refusal |
| `last --json` without `--dry-run` | `2` | same refusal |
| Child failure propagation | `1` | Codex refused a non-terminal stdin; Reinstate surfaced `codex native resume failed: exit status 1` and propagated exit `1`, leaving the source unchanged (`19` lines before and after) |
| Ambiguous bare ID | — | deterministic injected gate `TestMissingAndAmbiguousReferencesNeverLaunch` (`PASS`); not manufactured by altering real state |
| Missing workspace | — | `TestPlanLaunchRejectsReadOnlyAndMissingWorkspace` (`PASS`) |
| Missing executable | — | `TestExecLaunchRunnerClassifiesPreflightFailures` (`PASS`) |

JSON responses were never contaminated by native child output, because `--json`
without `--dry-run` is refused for `resume`, `fork`, and `last`.

## 9. Interactive switcher

Console type: a real native PTY for the line-oriented picker, plus a detached
`screen` session attached to a real `Terminal.app` window for the cases that
launch a full vendor TUI. Both binary names were exercised.

| Case | Binary | Result |
| ---- | ------ | ------ |
| Bare invocation on a TTY refreshes and shows the numbered switcher | `rein`, `reinstate` | `PASS` |
| `/text` filter | `reinstate` | `PASS` — filter on the alpha project narrowed to exactly the `2` controlled alpha sessions |
| `i NUMBER` inspect | `reinstate` | `PASS` — same bounded metadata policy as `rein inspect`, preview truncated, no transcript body |
| `NUMBER` resume | `rein` | `PASS` — filtered to a single controlled reference, selected it, launched `claude` in the recorded workspace; the restored transcript showed the controlled `…-A1`/`…-A2` markers and the new turn persisted to that exact source session (`17 → 25` lines, marker `…-PICKER-RESUME-P1`) |
| `f NUMBER` fork | `rein` | `PASS` — created distinct native identity `90ea4950-…`; source digest unchanged and contains `0` occurrences of the fork-only marker |
| Invalid input | `reinstate` | `PASS` — `Enter a number, /text, i NUMBER, f NUMBER, or q.`, list redisplayed, recoverable, no default session chosen |
| Out-of-range number (`99999`) | `reinstate` | `PASS` — same recoverable rejection, no launch |
| `q` cancel | `rein`, `reinstate` | `PASS` — exit `0`, no launch, no mutation |
| EOF | `reinstate` | `PASS` — exit `0`, no launch, no mutation |
| Interrupt (`SIGINT`) | `rein` | `PASS` — picker terminated with no launch and no change to the controlled session set |
| Unrelated-transcript privacy | both | `PASS` — the picker prints only agent, project label, and native ID; no transcript body is disclosed |
| Non-TTY invocation | `rein`, `reinstate` | `PASS` — exit `2`, immediate, message `interactive session picker requires a terminal; use \`rein sessions --json\``; no hang and no silent newest-session selection |

Full picker output is deliberately not reproduced here because it enumerates
unrelated session metadata.

## 10. Read-only adapters

Gemini CLI and OpenCode are `NOT_INSTALLED` on this device. Neither was
installed solely for this run. Rows 28 and 29 are therefore honestly
`NOT TESTED` for the optional physical path.

The scanner surfaced this as a bounded, sanitized warning rather than a
failure: `agent_not_installed` — "OpenCode executable was not found; OpenCode
sessions were not indexed".

The mandatory fixture-backed gates were still executed and passed:
`TestGeminiSourceAppliesMetadataAndRewindReadOnly`,
`TestGeminiSourceSupportsLegacyConversationJSON`,
`TestOpenCodeSourceUsesDocumentedJSONCommand` (injected JSON runner),
`TestOpenCodeSourceRejectsOversizedCommandOutput`,
`TestOpenCodeSourceMissingExecutableIsOptional`,
`TestOpenCodeSourceCommandFailuresAreNonDestructive`.

**Row 30.** The read-only refusal contract is proved by the deterministic
injected-record gate `TestNativeLaunchRefusalsUseStableCodes`: an injected
Gemini record marked read-only, given `fork gemini:read-only --dry-run`, returns
`ExitCompatibility` with the read-only reason and no launch. `ExitCompatibility`
is `5` in `internal/exitcode/codes.go`. No read-only adapter was given a
fabricated mutation implementation, and no launch or mutation occurred.

## 11. Mandatory matrix

macOS column only. Windows results are reconciled separately in section 12 and
are not claimed as local evidence.

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | `PASS` — §2 | see §12 |
| 2 | Full local verification and required cross-builds | `PASS` — §5 | see §12 |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | `PASS` — §3 | see §12 |
| 4 | `rein sessions` discovers exact Claude sessions | `PASS` — §6 | see §12 |
| 5 | `rein sessions` discovers exact Codex sessions | `PASS` — §6 | see §12 |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | `PASS` — §6 | see §12 |
| 7 | Derived index path, rebuild, idempotency, and private permissions | `PASS` — §3, §6 | see §12 |
| 8 | Prompt-fragment literal search | `PASS` — §7 | see §12 |
| 9 | Agent filter | `PASS` — §7 | see §12 |
| 10 | Project filter | `PASS` — §7 | see §12 |
| 11 | Branch filter | `PASS` — §7 | see §12 |
| 12 | File filter | `PASS` — §7 | see §12 |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | `PASS` — §7 | see §12 |
| 14 | `sessions` and `search` do not dump transcript passages | `PASS` — §7 | see §12 |
| 15 | `inspect` metadata/160-code-point user preview policy | `PASS` — §7 | see §12 |
| 16 | Append/new-session refresh and no-change idempotency | `PASS` — §6 | see §12 |
| 17 | `last` selects the correct resumable session and filters | `PASS` — §8 | see §12 |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | `PASS` — §8 | see §12 |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | `PASS` — §8 | see §12 |
| 20 | Claude native resume | `PASS` — §8 | see §12 |
| 21 | Codex native resume | `PASS` — §8 | see §12 |
| 22 | Claude vendor-native fork, source preserved | `PASS` — §8 | see §12 |
| 23 | Codex vendor-native fork, source preserved | `PASS` — §8 | see §12 |
| 24 | Missing/ambiguous reference and missing executor fail safely | `PASS` — §8 | see §12 |
| 25 | JSON/native-child separation and child failure propagation | `PASS` — §8 | see §12 |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | `PASS` — §9 | see §12 |
| 27 | Non-TTY prompt failure is immediate and actionable | `PASS` — §9 | see §12 |
| 28 | Gemini read-only physical path, when installed | `NOT TESTED` — `NOT_INSTALLED`, §10 | see §12 |
| 29 | OpenCode read-only physical path, when installed | `NOT TESTED` — `NOT_INSTALLED`, §10 | see §12 |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | `PASS` — §10 | see §12 |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | `PASS` — §5, §6 | see §12 |
| 32 | Phase 1 automated regression remains green | `PASS` — §5 | see §12 |

macOS required rows 1–27 and 30–32: `30 PASS`, `0 PARTIAL`, `0 FAIL`,
`0 NOT TESTED`. Optional rows 28–29: `NOT TESTED` because neither vendor is
installed, which the runbook permits.

## 12. Cross-device reconciliation (Milestone M5)

### Validated peer artifacts

Both Windows report branches were fetched and validated independently. Their
verdict labels were not trusted on their own.

| Artifact | Assertion | Result |
| -------- | --------- | ------ |
| PR #62 | state `OPEN`, `draft=true`, unmerged, base `feat/phase2-local-index` | `PASS` |
| PR #62 | head OID equals `61c9b1ebeca0279356fc95a90ecd5cc5afe1f982` | `PASS` |
| PR #62 | branch `test/phase2-b7b45db014ed-windows-report` descends from `b7b45db014edf030d820e503ee23b579c5032e69` | `PASS` (2 commits ahead) |
| PR #62 | diff from its tested commit is report-only | `PASS` — exactly `A docs/testing/results/2026-07-31-windows-phase2-b7b45db014ed.md` |
| PR #62 | one complete terminated `PHASE2-DEVICE-REPORT-V1` block, `device=windows` | `PASS` (1 begin, 1 end) |
| PR #62 | `test_commit` field equals `b7b45db014edf030d820e503ee23b579c5032e69` | `PASS` |
| PR #62 | sanitation: no `C:\Users\` path, token, key, or passphrase | `PASS` |
| PR #64 | state `OPEN`, `draft=true`, unmerged, base `feat/phase2-local-index` | `PASS` |
| PR #64 | head OID equals `0b5ec7075ec2da5f261cea5d9e50ba892d4a0a30` | `PASS` |
| PR #64 | branch `test/phase2-5c60ec237ddd-windows-targeted` descends from the tested commit | `PASS` (1 commit ahead) |
| PR #64 | diff is report-only | `PASS` — exactly `A docs/testing/results/2026-07-31-windows-phase2-5c60ec237ddd-targeted.md` |
| PR #64 | `test_commit` field equals `5c60ec237ddded8e314cdb8c1449080ddc923395` | `PASS` |
| PR #64 | sanitation: no `C:\Users\` path, token, key, or passphrase | `PASS` |
| PR #64 | terminated transfer block present | `PASS` — `PHASE2-WINDOWS-TARGETED-CLOSEOUT-V1` … `END-…` |

**Discrepancy recorded honestly.** PR #64 does **not** carry a
`PHASE2-DEVICE-REPORT-V1` block. It carries a differently-named terminated block,
`PHASE2-WINDOWS-TARGETED-CLOSEOUT-V1`, and explicitly declares
`devices_reconciled=false` and `final_phase2_certification_claimed=false`. That
is self-consistent with its stated supplemental scope (rows 20, 21, 22, 23, 26
only), but it means the only full-matrix `PHASE2-DEVICE-REPORT-V1` block for
Windows is the one at commit `b7b45db`, not at the tested commit.

### Windows baseline result at `b7b45db`

The full Windows baseline did **not** pass. Its own block reports
`required_pass=23`, `required_partial=3`, `required_fail=4`,
`release_blocking_findings=6`, with `configless_local_only=FAIL`,
`claude_resume_fork=FAIL`, `codex_resume_fork=FAIL`, and `picker=FAIL`.

- `FAIL`: rows 20, 21, 22, 23
- `PARTIAL`: rows 3, 6, 26
- `NOT TESTED`: rows 28, 29 (vendor not installed — permitted)
- `PASS`: all remaining rows

### Independent diff analysis, `b7b45db` → `5c60ec2`

Rather than assume the baseline carries forward, the diff was inspected
directly. It contains exactly two commits and three files:

| File | Kind |
| ---- | ---- |
| `CHANGELOG.md` | documentation |
| `docs/testing/phase-2-local-index-acceptance.md` | documentation (the `rein list` configless-contract correction) |
| `internal/cli/sessions_windows_test.go` | new Windows-tagged test, `+94` lines |

**There is no product-code change between `b7b45db` and `5c60ec2`.** Filtering
the diff to non-test, non-documentation files yields an empty set. Every
Windows row's product behaviour at the tested commit is therefore identical to
the baseline, and the baseline's `PASS` rows carry forward legitimately. The
sole new test, `TestNativeClaudeLaunchThroughWindowsCommandShim`, is
Windows-tagged and was executed by this commit's `Test (windows-latest)` CI job,
which this device verified as `success` against `head_sha=5c60ec2`.

The documentation change also matches this device's independent physical
observation: `rein list` succeeded from a fresh configless home with its own
distinct Phase 1 output and was not an alias of `rein sessions`.

### Reconciled Windows coverage

| Rows | Source of evidence | Reconciled result |
| ---- | ------------------ | ----------------- |
| 20, 21, 22, 23, 26 | PR #64, physically re-executed on native Windows at the exact tested commit | `PASS` |
| 1, 2 | PR #64 §2 provenance/build at the exact tested commit, plus verified `windows-latest` CI on `5c60ec2` | `PASS` |
| 4, 5, 7–19, 24, 25, 27, 30, 31, 32 | PR #62 baseline `PASS` at `b7b45db`, carried forward on the proven zero-product-delta diff and confirmed green by `windows-latest` CI at `5c60ec2` | `PASS` |
| 3 | PR #62 `PARTIAL`; **not** re-executed at the tested commit | `PARTIAL` |
| 6 | PR #62 `PARTIAL`; **not** re-executed at the tested commit | `PARTIAL` |
| 28, 29 | Gemini could not authenticate; OpenCode returned `no such column: name` | `NOT TESTED` (optional; permitted) |

### Disposition of the baseline's six release-blocking findings

| Finding | Disposition |
| ------- | ----------- |
| R1 — native resume failed for both agents | **Closed.** Rows 20 and 21 re-executed and `PASS` at the tested commit (PR #64), and independently `PASS` on macOS. |
| R2 — vendor-native fork failed for both agents | **Closed.** Rows 22 and 23 re-executed and `PASS` at the tested commit (PR #64), and independently `PASS` on macOS. |
| R3 — isolation contaminated by one harness attempt (row 3 `PARTIAL`) | **Open as evidence, not as a defect.** The cause was a failed reserved-variable assignment in the Windows harness that created an extra derived index under the real profile. No config, state, backup, credential, or secret was created. Not a product defect; row 3 nevertheless remains `PARTIAL` on Windows because it was not re-executed cleanly at the tested commit. macOS row 3 is `PASS`. |
| R4 — full-array parity/idempotency incomplete (row 6 `PARTIAL`) | **Open as evidence, not as a defect.** One unrelated *active* Codex record changed size/timestamp between sequential scans; controlled records were equal. This is the same live-record phenomenon disclosed in section 4 of this report, which macOS handled by asserting against the controlled corpus — where parity and idempotency were exact. Row 6 remains `PARTIAL` on Windows pending a re-run that excludes live records. macOS row 6 is `PASS`. |
| R5 — picker rendered states incomplete (row 26 `PARTIAL`) | **Closed.** Row 26 re-executed and `PASS` at the tested commit (PR #64), and independently `PASS` on macOS through a real PTY and a real `Terminal.app`-attached console. |
| R6 — installed Codex `0.146.0` outside the verified range | **Out of Phase 2 scope, correctly fail-closed.** Codex `0.146.0` remains intentionally `UNTESTED` for **Phase 1 encrypted sync writes**, where setup exit `5` is the correct fail-closed compatibility refusal. This is deliberately kept distinct from Phase 2 local resume/fork evidence: Phase 2 local resume and fork with Codex `0.146.0` are physically `PASS` on both devices. No vendor update or downgrade was performed on either device. |

### Reconciled 32-row counts

| Bucket | macOS | Windows | Both devices |
| ------ | ----- | ------- | ------------ |
| Required rows (1–27, 30–32) `PASS` | `30` | `28` | `28` |
| Required rows `PARTIAL` | `0` | `2` (rows 3, 6) | `2` |
| Required rows `FAIL` | `0` | `0` | `0` |
| Required rows `NOT TESTED` | `0` | `0` | `0` |
| Optional rows (28, 29) | `2 NOT TESTED` | `2 NOT TESTED` | `2 NOT TESTED` |

### Verdict

Both reports test builds of the same product code, and the tested commit is
identical for macOS and for the Windows targeted closeout. Automated gates are
green on both devices and all six required checks are `success` at `5c60ec2`,
including `windows-latest`.

**Phase 2 does not yet meet the runbook's section 15 bar**, which requires rows
1–27 and 30–32 to be `PASS` on *both* real devices. Windows rows 3 and 6 stand
at `PARTIAL`. Those two rows require execution on native Windows at commit
`5c60ec2`:

- **row 3** — a clean fresh-configless-home isolation run with no reserved
  variable collision and no stray derived index outside the isolated home;
- **row 6** — `rein` / `reinstate` JSON parity, deterministic ordering, and
  no-change idempotency asserted against the controlled corpus with live/active
  records excluded.

No confirmed product defect remains. Both open items are gaps in Windows
*evidence* caused by harness contamination and live-record churn, and both are
already `PASS` on macOS against product code that is byte-identical across the
two commits.

## 13. Findings

### Release-blocking

`None` from the macOS device.

Outstanding at the programme level: Windows rows 3 and 6 are `PARTIAL` and have
not been re-executed at the tested commit. This is missing evidence, not a
confirmed defect.

### Non-blocking

1. The tested development commit is reachable from
   `origin/feat/phase2-local-index` but not from `origin/main`. This is a
   development acceptance run, not a published release.
2. Gemini CLI and OpenCode are not installed on this device; optional physical
   rows 28 and 29 are `NOT TESTED`. Neither was installed solely for this run.
3. `rein sessions` emits `13` bounded `oversized_record` warnings plus `1`
   `agent_not_installed` warning from the operator's pre-existing unrelated
   Codex corpus. These are correct, bounded, sanitized diagnostics; the scan
   still completes. They name unrelated session IDs, so only counts are recorded.
4. The operator's own live Claude Code session is indexed alongside the
   controlled corpus and legitimately matches controlled marker searches,
   because those markers were typed into it. Disclosed in section 4; all
   determinism assertions exclude it.
5. Codex `0.146.0` remains `UNTESTED` for Phase 1 encrypted sync writes. Setup
   exit `5` is correct fail-closed behaviour and is out of Phase 2 scope.

### Test-harness deviations

No deviation was converted into a `PASS`. Every deviation below was retried
until the gate was genuinely executed, or the gate is recorded honestly.

| # | Operation | Exit | Cause | Disposition |
| - | --------- | ---- | ----- | ----------- |
| 1 | `timeout(1)` unavailable on macOS; `gtimeout` also absent | — | platform | Wrote a bounded Perl `alarm`/`fork`/`exec` wrapper; validated it returns `0` on success and `124` on timeout before use. |
| 2 | First search matrix used `2>&1`, so `stderr` warnings inflated every row count and made the zero-match case look identical to a hit | `0` | harness error | Re-ran the entire search matrix with `stdout`/`stderr` separated. Only the corrected run is scored. |
| 3 | First dry-run plan loop used unquoted word splitting, so four commands received a malformed argument | `2` | harness error | Re-ran all four dry-run plans with correct quoting. The `2` was my own invocation, not product behaviour. Only the corrected run is scored. |
| 4 | `rein resume codex:…` with piped stdin | `1` | vendor requires a real terminal | Not a defect; recorded as positive evidence for child-failure propagation (row 25), then re-run through a real console. |
| 5 | First Codex console attempt redirected stdout to a file | `1` | harness error — `stdout is not a terminal` | Re-ran using `screen`'s own logging so the vendor kept a real PTY. |
| 6 | `screen -Logfile` rejected | `1` | macOS ships `screen 4.00.03` | Re-ran with `screen -L` from a dedicated logging directory. |
| 7 | First picker run delivered EOF before the prompt because the feeder closed immediately | `0` | harness pacing | Recorded as valid EOF-cancel evidence, then re-run with a paced feeder to exercise `/text`, `i N`, invalid input, and `q`. |
| 8 | Picker-driven Claude resume through a plain PTY | `124` | harness timeout while the vendor TUI booted | Re-run through a `Terminal.app`-attached `screen` session; gate then executed fully. |
| 9 | First `Terminal.app` picker resume produced no vendor write | `0` | the child Claude TUI inherited `CLAUDE_CODE_CHILD_SESSION` from the operating agent and disabled transcript persistence, printing its own warning | Harness artefact of running Claude inside Claude, not a Reinstate defect. Re-run with that vendor guard cleared; the gate then persisted correctly. No vendor session file was edited. |
| 10 | Picker `SIGINT` run | — | `screen` log did not capture the wrapper's echoed exit code | The substantive assertions were still verified: no launch occurred and the controlled session set was unchanged. Exit-code evidence for cancel/EOF comes from the `q` and EOF runs (both `0`) and from `TestInteractivePickerCancelAndNonTTY`. |

Rows 28 and 29 are `NOT TESTED` because Gemini CLI and OpenCode are not
installed. Ambiguous-reference, missing-workspace, missing-executable,
incomplete-JSONL-line, malformed-record, and duplicate-ID cases were satisfied
by deterministic injected/fixture gates, as the runbook permits, rather than by
damaging a real vendor file.

## 14. Repository hygiene

- report-only branch: `test/phase2-5c60ec237ddd-macos-report`
- draft PR: #65 (draft, unmerged, base `feat/phase2-local-index`)
- tested base commit: `5c60ec237ddded8e314cdb8c1449080ddc923395`
- changed files: exactly one — `docs/testing/results/2026-07-31-macos-phase2-5c60ec237ddd.md`
- private/local artifacts excluded: no local index, vendor session, fixture
  copy, generated binary, transcript, log, or private path is committed
- product code unchanged: `true`
- secrets/transcripts committed: `false`

The isolated Reinstate home, derived index, disposable projects, and controlled
vendor sessions were intentionally left in place. No older acceptance home,
real agent session, or unrelated project was inspected, modified, or deleted.

## 15. Device milestone block

```text
PHASE2-DEVICE-REPORT-V1
device=macos
test_commit=5c60ec237ddded8e314cdb8c1449080ddc923395
reinstate_version=v0.1.0-38-g5c60ec2
report_path=docs/testing/results/2026-07-31-macos-phase2-5c60ec237ddd.md
report_branch=test/phase2-5c60ec237ddd-macos-report
claude_ref=claude:037ebc04-f5ea-4de3-931e-0aa0517b183e
codex_ref=codex:019fb734-e14b-71c2-9141-c44dbcb35781
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

## 16. Final reconciliation block

```text
PHASE2-FINAL-RECONCILIATION-V1
test_commit=5c60ec237ddded8e314cdb8c1449080ddc923395
mac_report_branch=test/phase2-5c60ec237ddd-macos-report
mac_report_parent_commit=5c60ec237ddded8e314cdb8c1449080ddc923395
mac_report_commits_ahead=1
windows_report_commit=0b5ec7075ec2da5f261cea5d9e50ba892d4a0a30
windows_baseline_report_commit=61c9b1ebeca0279356fc95a90ecd5cc5afe1f982
windows_baseline_test_commit=b7b45db014edf030d820e503ee23b579c5032e69
product_code_delta_baseline_to_tested=none
mandatory_rows=32
required_dual_device_rows_passed=28
required_dual_device_rows_partial=2
required_dual_device_rows_failed=0
windows_rows_requiring_reexecution=3,6
optional_rows_28_29_macos=NOT_TESTED
optional_rows_28_29_windows=NOT_TESTED
automated_gates=PASS
ci_checks_tested_commit=6/6
ci_windows_latest=PASS
physical_macos=PASS
physical_windows=PARTIAL
confirmed_product_defects=0
release_blocking_findings=0
missing_evidence_items=2
phase2_status=FAIL
phase2_blocking_reason=windows_rows_3_and_6_partial_not_reexecuted_at_tested_commit
rc_release_justified=false
END-PHASE2-FINAL-RECONCILIATION-V1
```

The Mac report commit is identified by branch plus parent rather than by its
own SHA, because a commit cannot contain its own hash. It is the single commit
on `test/phase2-5c60ec237ddd-macos-report` whose parent is the tested commit,
which resolves it unambiguously.

`phase2_status=FAIL` is recorded because the runbook's section 15 bar requires
every required row to be `PASS` on both devices and two Windows rows are
`PARTIAL`. It does **not** indicate a product defect: `confirmed_product_defects=0`.
The gap is missing Windows evidence for rows 3 and 6.
