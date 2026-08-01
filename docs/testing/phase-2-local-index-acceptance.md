# Phase 2 Local Session Index Acceptance

Use this runbook to decide whether Reinstate Phase 2 is functional on real
macOS and native Windows hardware. It covers the configless local index,
metadata search, native same-vendor resume and fork, and the interactive
switcher. It does not repeat Phase 1's encrypted two-device sync exercise.

**Current status:** development acceptance passed all 30 required rows on
macOS and native Windows against product commit `b952d38`. The sanitized
reports are
[macOS](results/2026-07-31-macos-phase2-b952d38c2dc5.md) and
[Windows](results/2026-07-31-windows-phase2-b952d38c2dc5.md). A signed release
candidate must repeat the tagged-artifact portions of this runbook on both
devices before stable promotion.

## 1. Product boundary

Phase 2 must work without:

- `rein init`;
- `config.toml`, a sync profile, or project mappings;
- S3/R2 coordinates or credentials;
- an encryption passphrase or keyring access; and
- a network backend.

Claude Code and Codex have the complete Phase 2 surface: index, search,
inspect, native resume, and vendor-native fork. Gemini CLI and OpenCode are
read-only: discovery, search, and inspect only. A read-only adapter must refuse
resume or fork with compatibility exit code `5`; it must never receive a
fabricated mutation implementation.

Native resume remains same-vendor:

```text
<agent>:<native-session-id>
```

| Session | Allowed executor |
| ------- | ---------------- |
| `claude:<id>` | Claude Code |
| `codex:<id>` | Codex CLI |
| `gemini:<id>` | Read-only in Phase 2 |
| `opencode:<id>` | Read-only in Phase 2 |

Cross-agent transcript conversion and portable handoffs are Phase 4 work.

## 2. Evidence rules and stop conditions

Run Device A and Device B independently and in parallel:

- **Device A:** native macOS, operated by Claude Code;
- **Device B:** native 64-bit Windows, operated by Codex;
- **artifact:** the same exact Git commit or signed release on both devices.

Stop and report `FAIL` when:

- the tested commit or binary differs between devices;
- a local command requires initialization, storage, credentials, a passphrase,
  or a network backend;
- default output dumps a transcript, assistant message, reasoning, tool output,
  environment dump, auth material, or an unbounded prompt;
- Reinstate modifies a vendor session merely to index, search, or inspect it;
- Reinstate launches a different vendor from the session's composite
  reference;
- a read-only adapter mutates, resumes, forks, or syncs a session; or
- a required command, automated gate, or physical row is not actually run.

Preserve `PASS`, `PARTIAL`, `FAIL`, and `NOT TESTED` honestly. A zero exit code
is not sufficient without checking the result. Optional Gemini/OpenCode
physical rows may be `NOT TESTED` when that vendor is not installed; their
fixture-backed automated rows remain required.

Never:

- read, print, summarize, screenshot, or commit unrelated transcript content;
- inspect auth files, keychains, tokens, cookies, `.env` files, or credential
  stores;
- copy a real vendor tree into the repository;
- alter a vendor file to manufacture discovery, ambiguity, or a fork;
- delete real sessions or an older Reinstate home;
- disable normal sandbox or approval controls; or
- commit anything except the sanitized report on a report branch.

Reports may contain the tested commit, versions, composite test references,
counts, booleans, exit codes, bounded controlled markers, redacted paths, and
sanitized errors. They must not contain usernames, absolute home paths, full
prompts, transcript prose, or secret values.

## 3. Test record

Record these fields on each device:

| Field | Value |
| ----- | ----- |
| UTC date/time | |
| Tested Git commit | |
| Signed tag, if any | |
| Reinstate version JSON | |
| OS edition/version/build | |
| Architecture | |
| Native shell | |
| Claude Code version/state | |
| Codex CLI version/state | |
| Gemini CLI version/state | |
| OpenCode version/state | |
| Git version | |
| Go version, for source-build acceptance | |
| Report branch/commit/PR | |

For development acceptance, build the exact clean commit on both devices and
record the commit embedded by `rein version --json`. For release-candidate
certification, also verify the annotated tag, signature, checksums, public
installer pin, and installed binary commit. Do not describe a local build as a
published release.

## 4. Isolation and controlled corpus

Choose a fresh run ID derived from the tested commit and current UTC date. Use
new paths that did not exist before this run:

```text
macOS REINSTATE_HOME:  $HOME/.reinstate-phase2-acceptance-RUN_ID
Windows REINSTATE_HOME: $HOME\.reinstate-phase2-acceptance-RUN_ID
project 1:              .../reinstate-phase2-acceptance-RUN_ID-alpha
project 2:              .../reinstate phase2 RUN_ID unicode-β
```

Set the absolute `REINSTATE_HOME` before every Reinstate command. Do not run
`init`. Before the first local-index command, prove that the home is absent or
empty and contains no `config.toml`, `state.json`, `backups/`, or credential
reference.

Create two harmless Git repositories with distinct branches and controlled
relative files. Use names unique to this run, for example:

```text
branch: phase2/RUN_ID-alpha
file:   src/phase2_RUN_ID_alpha.txt
marker: REINSTATE-PHASE2-RUN_ID-DEVICE-AGENT-A1
```

Create and cleanly close one fresh Claude Code session and one fresh Codex
session in the disposable projects. The controlled prompt should ask for an
exact harmless marker response and mention only the disposable relative file
and branch. Identify each native ID from a before/after metadata diff and the
controlled marker only. Do not read surrounding transcript prose.

If Gemini CLI or OpenCode is installed, create one equally harmless disposable
session through the vendor's normal interface. Do not install a missing vendor
solely for physical acceptance unless the release checklist explicitly
requires it.

Record composite references:

```text
claude:<native-id>
codex:<native-id>
gemini:<native-id>      # when installed
opencode:<native-id>    # when installed
```

Fixed marker occurrence counts are not a gate. Vendor serialization may repeat
the same controlled value. Require at least one exact occurrence and an exact
challenge response where the vendor supports non-interactive proof.

## 5. Automated gates

Run the repository's exact required verification commands from the tested
commit. At minimum:

```text
make fmt-check
go test ./internal/sessionindex ./internal/cli ./internal/adapter/...
go test ./...
go test -race ./...
make vet
make verify
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/reinstate
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/reinstate
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/reinstate
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/reinstate
```

The deterministic suite must cover:

- macOS-, Windows-, and WSL-shaped Claude/Codex fixtures;
- Gemini JSON discovery and an injected OpenCode JSON runner;
- corrupt index rebuild and private permissions;
- append, replace, deletion-after-successful-scan, malformed records, bounded
  fields, oversized records, and incomplete final JSONL lines;
- composite-reference ambiguity and deterministic ordering;
- search filters and terminal-control stripping;
- exact executable/argv/cwd launch plans, missing executable/workspace,
  cancellation, and child failure propagation;
- on native Windows, stdin/stdout, exact argv, and cwd inheritance through the
  `.cmd` shim used by the real Claude launch path;
- `sessions -> search -> inspect -> last --dry-run -> resume --dry-run ->
  fork --dry-run` without config or backend access; and
- all Phase 1 tests.

Unit tests must not use the network or real vendor session trees.

## 6. Configless local index

Run:

```text
rein sessions
rein sessions --json
reinstate sessions --json
rein list --help
```

Mandatory results:

1. `sessions` succeeds without config and discovers the exact controlled
   Claude/Codex references.
2. `rein` and `reinstate` return equivalent deterministic JSON.
3. Ordering is newest update first, then agent, then native ID.
4. Repeating the command does not duplicate records.
5. `rein list` remains the Phase 1 compatibility command and is not silently
   redefined as the Phase 2 canonical listing command. Its configured behavior
   is covered by the Phase 1 regression suite. From this fresh configless home,
   `rein list` may still succeed with its distinct Phase 1 output; the required
   assertion is that it has not been redefined as an alias of `rein sessions`.
6. The only permitted new Reinstate state is the private derived cache at
   `$REINSTATE_HOME/cache/session-index-v1.sqlite` and its SQLite support
   files while open.
7. No config, sync state, backup, credential, passphrase prompt, storage
   request, or remote object appears.
8. The database and its parent directories are owner-only under the native OS
   permission model.

Record exact controlled-reference presence and result counts only. Do not paste
the full listing when it contains unrelated sessions.

## 7. Search

Use a unique fragment from the controlled corpus. Exercise:

```text
rein search "CONTROLLED PROMPT FRAGMENT"
rein search "TERM_ONE TERM_TWO" --json
rein search "FRAGMENT" --agent claude
rein search "FRAGMENT" --agent codex
rein search "FRAGMENT" --project "PROJECT_FRAGMENT"
rein search "FRAGMENT" --branch "BRANCH_FRAGMENT"
rein search "FRAGMENT" --file "FILE_FRAGMENT"
rein search "FRAGMENT" --limit 1
rein search "DELIBERATE_ZERO_MATCH"
```

Mandatory results:

- matching is literal and case-insensitive;
- multiple query terms are ANDed;
- agent, project, branch, file, and limit filters narrow correctly;
- zero match is an honest empty result, not an error or a fallback to all
  sessions;
- search identifies matching session metadata without printing the matching
  transcript passage; and
- no network request or semantic/embedding service is involved.

Exercise spaces, Unicode, and shell metacharacters as literal query data
without constructing a shell command string inside Reinstate.

## 8. Inspect and preview privacy

Run human and JSON inspection for each controlled reference:

```text
rein inspect AGENT:NATIVE_ID
rein inspect AGENT:NATIVE_ID --json
```

Mandatory results:

- identity, agent, timestamps, workspace/project, branch, message count,
  source/capability metadata, and known file references are accurate;
- any preview comes only from a user-authored prompt;
- preview whitespace is collapsed, terminal controls/control characters are
  removed, and the value is at most 160 Unicode code points;
- no assistant reasoning/message, tool output, environment dump, auth content,
  or full transcript is present; and
- inspecting one malformed session cannot prevent healthy sessions from being
  listed or inspected.

Do not insert a real secret to test this boundary. Synthetic fixtures contain
controlled sentinels for automated exclusion tests.

## 9. Refresh and ordering

After the initial index:

1. resume one controlled session through its vendor and append one new harmless
   marker;
2. create one new controlled session after all earlier sessions;
3. rerun `rein sessions --json` and the relevant searches; and
4. repeat the refresh once without changing any source.

Mandatory results:

- appended metadata/search text appears after refresh;
- the new session becomes the newest eligible record;
- unchanged sessions keep one stable composite identity;
- a no-change refresh is idempotent;
- concurrent vendor append does not corrupt the source or index; and
- an incomplete final JSONL line is ignored until complete.

The incomplete-line, deletion, malformed, and duplicate-ID cases may use
synthetic fixtures in the automated suite. Do not damage a real vendor file to
manufacture them on a physical device.

## 10. Resolve, last, and native resume

Run dry-run JSON first:

```text
rein resume claude:NATIVE_ID --dry-run --json
rein resume codex:NATIVE_ID --dry-run --json
rein last --dry-run --json
rein last --agent claude --dry-run --json
rein last --project PROJECT_FRAGMENT --dry-run --json
```

Mandatory dry-run plans:

| Agent | Executable and argv |
| ----- | ------------------- |
| Claude | `claude`, `--resume`, native ID |
| Codex | `codex`, `resume`, native ID |

The plan uses an argv array, not a shell string, and uses the recorded
workspace as `cwd`. Dry-run starts no agent and changes no vendor file.

Then run real `rein resume` for the exact Claude and Codex references. Prove
the expected vendor process starts in the recorded workspace and returns the
controlled challenge response. Reinstate must propagate a child failure.
Starting the vendor may legitimately append vendor metadata; distinguish that
from Reinstate modifying the session before launch.

Also prove:

- a unique bare native ID resolves;
- a missing reference fails with an actionable error and no launch;
- an ambiguous bare ID fails and requests `agent:id`;
- a missing workspace or executable fails before launch; and
- `--json` without `--dry-run` is refused for `resume`, `fork`, and `last`,
  so native child output never corrupts a JSON response.

Ambiguity and missing-binary cases may be satisfied by deterministic injected
tests when manufacturing them physically would require altering real state.

## 11. Vendor-native fork

For both Claude and Codex, record the source fingerprint and run:

```text
rein fork AGENT:NATIVE_ID --dry-run --json
rein fork AGENT:NATIVE_ID
```

Expected plans:

| Agent | Executable and argv |
| ----- | ------------------- |
| Claude | `claude`, `--resume`, native ID, `--fork-session` |
| Codex | `codex`, `fork`, native ID |

Mandatory results:

- dry-run performs no mutation;
- Reinstate invokes the same vendor, in the recorded workspace;
- the vendor creates a distinct native session identity;
- the original is not modified by Reinstate before launch;
- refresh discovers both source and fork;
- each resumes through its vendor's native path; and
- changing the fork does not change the source.

For an indexed Gemini or OpenCode reference, `rein fork gemini:...` and
`rein fork opencode:...` must fail with exit `5` and no process launch or
mutation. When neither vendor is installed, the same required contract is
proved by the deterministic injected-record CLI tests on each device.

## 12. Interactive switcher

Use a real native PTY on macOS and a real console/PTY path in native
PowerShell. Test both binary names.

On a TTY, bare `rein` or `reinstate` must refresh and show the numbered local
switcher. Exercise:

```text
/text       filter
i NUMBER    inspect
f NUMBER    fork
NUMBER      resume
q           cancel
```

Mandatory results:

- filtering selects only controlled matching sessions;
- inspect follows the same bounded metadata policy;
- fork and resume target the exact displayed composite reference;
- `q`, EOF, or interrupt exits without launching or mutating;
- invalid input is recoverable and does not choose a default session; and
- terminal output does not disclose unrelated transcript bodies.

Pipe empty input to the command to prove non-TTY invocation fails promptly with
usage exit `2` and a `rein sessions --json` hint. It must not hang or silently
select the newest session.

## 13. Gemini and OpenCode read paths

When installed, require each controlled session to appear in `sessions`,
literal search, and `inspect`. Require capability metadata to say read-only.
Resume, fork, push, and pull must not be offered as Phase 2 capabilities.

When not installed, record `NOT_INSTALLED` and mark only the optional physical
row `NOT TESTED`. The automated fixture/fake-runner gate remains mandatory.
Record the exact installed vendor version and any sanitized discovery warning;
do not infer native or mutation support from successful read-only indexing.

## 14. Phase 1 regression

All existing Phase 1 unit, fixture, CLI, path-map, encryption, conflict, and
restore-safety tests must remain green. In particular, local indexing must not:

- reuse the configured sync registry and hide unmapped local sessions;
- change `rein list`, `push`, `pull`, `status`, `diff`, or conflict semantics;
- include `cache/session-index-v1.sqlite` in a snapshot;
- weaken compatibility refusal for sync writes; or
- change stable exit codes.

A full physical cloud re-run is required for a stable release when shared
sync/adapter behavior changes. It is not part of every local Phase 2 edit loop.

## 15. Mandatory sign-off matrix

Mark each row separately. A dual-device row needs evidence from both reports.

| # | Gate | macOS | Windows |
| - | ---- | ----- | ------- |
| 1 | Exact tested commit/binary provenance | | |
| 2 | Full local verification and required cross-builds | | |
| 3 | Fresh configless home; no `init`, credentials, passphrase, or backend | | |
| 4 | `rein sessions` discovers exact Claude sessions | | |
| 5 | `rein sessions` discovers exact Codex sessions | | |
| 6 | `rein` / `reinstate` JSON parity and deterministic ordering | | |
| 7 | Derived index path, rebuild, idempotency, and private permissions | | |
| 8 | Prompt-fragment literal search | | |
| 9 | Agent filter | | |
| 10 | Project filter | | |
| 11 | Branch filter | | |
| 12 | File filter | | |
| 13 | AND terms, limit, case, Unicode, and zero-match behavior | | |
| 14 | `sessions` and `search` do not dump transcript passages | | |
| 15 | `inspect` metadata/160-code-point user preview policy | | |
| 16 | Append/new-session refresh and no-change idempotency | | |
| 17 | `last` selects the correct resumable session and filters | | |
| 18 | Claude dry-run plan has exact argv/cwd and no mutation | | |
| 19 | Codex dry-run plan has exact argv/cwd and no mutation | | |
| 20 | Claude native resume | | |
| 21 | Codex native resume | | |
| 22 | Claude vendor-native fork, source preserved | | |
| 23 | Codex vendor-native fork, source preserved | | |
| 24 | Missing/ambiguous reference and missing executor fail safely | | |
| 25 | JSON/native-child separation and child failure propagation | | |
| 26 | TTY picker filter, inspect, resume, fork, and cancel | | |
| 27 | Non-TTY prompt failure is immediate and actionable | | |
| 28 | Gemini read-only physical path, when installed | | |
| 29 | OpenCode read-only physical path, when installed | | |
| 30 | Read-only resume/fork refusal with exit `5` (physical or injected-record gate) | | |
| 31 | Malformed/concurrent/oversized fixture and privacy gates | | |
| 32 | Phase 1 automated regression remains green | | |

Phase 2 passes physical acceptance only when rows 1–27 and 30–32 are `PASS` on
both devices, rows 28–29 are `PASS` or honestly `NOT TESTED` because the vendor
is not installed, there is no release-blocking finding, and both reports name
the same tested commit.

## 16. Parallel reports and reconciliation

Use [the Phase 2 report template](results/phase-2-report-template.md).

1. Mac and Windows start independently against the same exact commit.
2. Each device completes its entire local matrix and publishes one cumulative
   sanitized report on a report-only branch.
3. The Windows report is passed to the Mac coordinator.
4. The Mac coordinator validates both report commits/diffs and reconciles the
   32 rows once.

There is no Phase 1-style eight-step ping-pong because no Phase 2 gate depends
on shared remote state.

## 17. Cleanup

Cleanup is optional and happens only after both reports are reviewed.

- Keep reports and automated logs after sanitization.
- Do not delete real or unrelated vendor sessions.
- Remove only the exact disposable projects, isolated Reinstate home, and
  controlled vendor sessions created for this run after reviewing their IDs.
- The index is derived and safe to remove, but deleting it does not delete any
  vendor session.
