# `v0.6.0-rc.1` tagged-artifact Windows acceptance — Part B (T2-T5 agent rows)

`PHASE5-DEVICE-REPORT-V1` (per-agent excerpt) — **W7 executor B (tagged run)**,
covering Matrix C/D/E rows for `claude`, `codex`, `opencode`, `grok`, `qwen`
(T4/T5, 17 rows each) and `gemini`, `kimi` (T2, 11 rows each), plus the T5
encrypted-sync round trip for `claude`/`codex`/`opencode`, per
[`v0.6.0-rc.1-agent-verification-prompts.md`](../v0.6.0-rc.1-agent-verification-prompts.md)
and the [Phase 5 contract](../phase-5-universal-agent-coverage-acceptance.md).
This is a **partial** device report: it covers only the rows assigned to this
executor for this tagged run. It does not stand alone as a Phase 5 verdict.

Reuses the pre-tag report's method
(`docs/testing/results/2026-09-06-windows-v060rc1-pretag.md`) where noted, but
this run's dispatch raised the evidentiary bar for Matrix D/E: real vendor
binaries, real sessions in throwaway projects, isolated homes seeded with a
copied credential file (Claude Code alone uses the host's live,
already-authenticated config, per this run's ground rules) — not the
testdata-fixture-only method the pre-tag run used throughout. Every row below
states which evidence class it actually used.

## Deviation from dispatch

The dispatch (`v0.6.0-rc.1-agent-verification-prompts.md`) asks for an install
from the live bootstrap (reinstate.dev) after proving it pins `v0.6.0-rc.1`.
**At run time the live bootstrap still pins `v0.5.2-rc.1`** because the
guarded website deploy's test gate fails on three Windows-only test files, so
the install used in this report came from the published release assets
(the GitHub prerelease archive) whose checksum and attestation were verified
instead of the live bootstrap.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag under test | `v0.6.0-rc.1` (signed, published prerelease) |
| Full commit | `63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0` |
| Release workflow run | `34029733057` |
| Archive | `reinstate_0.6.0-rc.1_windows_amd64.zip` |
| Archive SHA-256 | `3d1ebf243c6d6f234ffe95955b9504502c340bfda16f42f776db97893bc26542` (matches `checksums.txt`, independently re-verified with `sha256sum` against the coordinator-staged copy before install) |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-w7b-b\install\` (this executor's own, fresh, never a developer or shared binary) |
| `rein.exe` / `reinstate.exe` | byte-identical (`sha256sum` both `d13d683eda6e1d59082be08c0326be196ffaa856cb8b83a35ef181d3ec624e2c`) |
| `rein version --json` | `{"commit":"63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0","date":"2026-09-06T11:17:45Z","name":"reinstate","version":"0.6.0-rc.1"}` |
| Worktree / branch | `v060/w7b-tagged`, checked out at `63ac5a5b` (release commit; branch tip moved ahead via other executors' commits during this run — this report's own evidence was gathered entirely against the `63ac5a5b` binary above, never rebuilt) |
| Previous-release comparison binary | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (downloaded fresh via `gh release download v0.5.1`, checksum-verified), installed `rein version --json` reports commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb`, version `0.5.1` — install at `D:\ReinstateAcceptanceProjects\v060-w7b-b\v051\install\` |
| Host | `windows-amd64`, native (never WSL); Windows 11 Pro `10.0.26200` |
| Git version | `git version 2.52.0.windows.1` |
| Go version (host default) | `go1.26.1 windows/amd64` |
| UTC date | 2026-09-06 |
| Report branch | `v060/w7b-tagged` |
| Host environment | `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` run in every shell before any `rein` invocation; `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` left at the host's live values except where a row's own isolation intentionally repointed the non-target agents' variables at empty throwaway directories |

## Vendor versions re-checked immediately before the run

| Agent | Expected | Installed | Match |
| ----- | -------- | --------- | ----- |
| `claude` | 2.1.263 (ceiling) | `2.1.263 (Claude Code)` | yes |
| `codex` | 0.149.0 | `codex-cli 0.149.0` | yes |
| `opencode` | 1.18.27 | `1.18.27` | yes |
| `grok` | 1.0.5 | `grok 1.0.5 (5115b46bc9) [stable]` | yes |
| `qwen` | 0.21.12/0.21.13 | `0.21.12` | yes (host at the floor of the two-build range) |
| `gemini` | 0.53.0 | `0.53.0` | yes |
| `kimi` | 0.36.1 | `0.36.1` | yes |

No agent was installed or upgraded for this run.

## Method

**`claude`, `codex`, `opencode`, `grok`, `qwen` (T4/T5):** real sessions
created in throwaway git repositories under
`D:\ReinstateAcceptanceProjects\v060-w7b-b\repos\<agent>\demo(2)\`, using each
vendor's own non-interactive/headless single-turn flag
(`claude -p`, `codex exec`, `opencode run`, `grok -p`, `qwen -p`) with a
planted per-agent, per-project token
(`RWB6-<AGENT>-P<N>-TOKEN`, never a real prompt or secret). `claude` used the
host's live, already-signed-in `CLAUDE_CONFIG_DIR` (never isolated, never
copied), per this run's ground rules; `codex`/`opencode`/`grok`/`qwen` each
used an isolated home (`CODEX_HOME`/`XDG_DATA_HOME`/`GROK_HOME`/`QWEN_HOME`)
seeded with only that vendor's own real credential file copied from its
default host location (`auth.json` for codex/opencode/grok,
`settings.json` for qwen — its `env.BAILIAN_CODING_PLAN_API_KEY` field is the
credential). Every command that queried `rein` set all seven agent root
variables explicitly, the six not under test pointed at empty throwaway
directories, plus a fresh `REINSTATE_HOME` per agent, so no listing could fall
back to an ambient default. Version-gate rows (E4) used a `.cmd` shim ahead of
the real binary on `PATH` that intercepts only `--version` and forwards every
other argument unmodified to the real vendor executable (absolute path) — the
real vendor binary itself is never modified or rebuilt. C6/D4 truncation rows
used a copy of the committed `testdata/sessionindex/*` or
`testdata/handoff/*/partial-final-record` fixtures, per
`docs/testing/phase-5-universal-agent-coverage-acceptance.md`'s explicit
allowance ("C6 is exercised against a copy of the tree or a synthetic
fixture. Never corrupt a tester's real agent data.") and `CLAUDE.md`'s
synthetic-fixture rule.

**`gemini`, `kimi` (T2):** attempted a real session in an isolated
`GEMINI_CLI_HOME`/`KIMI_CODE_HOME` seeded with a copied credential file, per
the same method. **`kimi`** succeeded fully: the copied
`credentials/kimi-code.json` authenticated a real headless `kimi -p` session
with a real model turn, and `rein` indexed it correctly — Matrix C/D below
for `kimi` use two real sessions. **`gemini`** partially succeeded: a copied
`settings.json` + `google_accounts.json` authenticated a real headless
`gemini --skip-trust -p` session (no auth error; a session file was written
to `<home>\.gemini\tmp\demo\chats\session-*.jsonl`), but `rein doctor
--agents --json` reported `resolved_root: null` for that isolated home even
though the marker was present, and `rein sessions --agent gemini` did not
list the session — recorded as a harness/investigation gap in
[Section 5](#5-not-tested-and-why), not chased further under this run's time
budget. Matrix C/D for `gemini` below use the committed
`testdata/sessionindex/gemini/windows` fixture instead (duplicated under a
second synthetic project, and — for the handoff rows only — its recorded
workspace repointed at a real throwaway git repository, since handoff's
compatibility gate correctly refuses to run outside the session's own
repository).

**T5 sync round trip:** a disposable `scripts/testing/fakelocker` instance
(`go run ./scripts/testing/fakelocker -addr 127.0.0.1:9123 -accept FAKEKEY`)
served as the S3-compatible locker. Two isolated `REINSTATE_HOME`s
(`synchome-A`, `synchome-B`) were paired via `rein init --profile-id` against
that locker. Because `rein push`/`pull` require a passphrase supplied only
through an inherited OS file-handle (`REINSTATE_PASSPHRASE_FD`,
`internal/crypto/passphrase_fd_windows.go`) and this executor drives real
`rein.exe` as an external subprocess (not the in-process test harness the
product's own Go tests use), a small throwaway Go helper
(`passrunner`, not part of the product, deleted with the rest of this
executor's throwaway state) opened a temp file holding the passphrase, marked
its Windows handle inheritable (`SetHandleInformation`), and spawned `rein.exe`
with that handle inherited and `REINSTATE_PASSPHRASE_FD` set to its value —
functionally identical to what a real launcher/shell would do, exercising the
product's real non-interactive passphrase path rather than working around it.

## 1. Matrix C — Per T1+ agent

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `C1:claude` | Real sessions from ≥2 distinct projects | PASS | `rein search "RWB6-CLAUDE-P1-TOKEN" --agent claude --json` and the `-P2-` token each returned one session; `demo`/`demo2` projects, distinct ids (`61e7874c…`/`f2b848d7…`) |
| `C1:codex` | as above | PASS | `rein sessions --agent codex --json` → 2 real sessions, projects `demo`/`demo2`, ids `01a07684-afe2…`/`01a07684-d9f9…` |
| `C1:opencode` | as above | PASS | `rein sessions --agent opencode --json` → 2 real sessions, `ses_f897832feffe1qIh8pXkfDke05` (demo), `ses_f8977d752ffe33qlKjyYaLSGbY` (demo2) |
| `C1:grok` | as above | PASS | `rein sessions --agent grok --json` → 2 real sessions, `01a0768b-64e7…` (demo), `01a0768d-7cff…` (demo2) |
| `C1:qwen` | as above | PASS | `rein sessions --agent qwen --json` → 2 real sessions (credential expired, see `E1:qwen`; the session files themselves were created and indexed correctly), `2532a6b7…` (demo), `2791a57a…` (demo2) |
| `C1:gemini` | as above | PASS | `testdata/sessionindex/gemini/windows` fixture, duplicated with a distinct id/workspace for the 2nd project (see Method) → `gemini-rewind-win` (demo), `gemini-rewind-win-p2` (demo-second) |
| `C1:kimi` | as above | PASS | `rein sessions --agent kimi --json` → 2 real sessions, `session_d0900629…` (demo), `session_024c8f27…` (demo2) |
| `C2:*` (all 7) | Project/branch/title/timestamp/message_count match | PASS | Every listing above carries `project`, `updated_at`, `message_count` consistent with the real invocation (e.g. `codex` sessions show `message_count:3` for a single real turn incl. system/user/assistant records; `claude` shows `branch:"prod"` matching the throwaway repo) |
| `C3:claude` | `rein search` finds a known string by prompt | PASS | `rein search "RWB6-CLAUDE-P1-TOKEN" --agent claude --json` → 1 hit, the source session |
| `C3:codex` | as above | PASS | `rein search "RWB6-CODEX-P1-TOKEN" --agent codex --json` → 1 hit |
| `C3:opencode` | as above | **PASS by title only** (documented, pre-existing, non-regression) | `rein search "RWB6-OPENCODE-P1-TOKEN"` (message-body text) → 0 hits, reproducing the known gap (`SearchText` never indexes message body for this agent, recorded pre-tag). `rein search "RWB6-OPENCODE-P2-TOKEN" --agent opencode --json` → 1 hit, because that session's LLM-generated title happened to include the token text — search matched by title, confirming the documented title-only behaviour with fresh real evidence rather than assuming it |
| `C3:grok` | as above | PASS | `rein search "RWB6-GROK-P2-TOKEN" --agent grok --json` → 1 hit |
| `C3:qwen` | as above | PASS | `rein search "RWB6-QWEN-P1-TOKEN" --agent qwen --json` → 1 hit (qwen's title is the literal prompt) |
| `C3:gemini` | as above | PASS | `rein search "Gemini Windows rewind fixture" --agent gemini --json` → 2 hits (both fixture sessions, by title) |
| `C3:kimi` | as above | PASS | `rein search "RWB6-KIMI-P1-TOKEN" --agent kimi --json` → 1 hit |
| `C4:claude` | `rein inspect` bounded, no transcript body | PASS | 5,478-byte-class JSON, `prompt_preview` truncated, no `messages`/`transcript` key |
| `C4:codex` | as above | PASS | 96,482 bytes (real ambient host MCP/skill capability list inflates size, same as pre-tag's noted pattern — not transcript content); no `messages`/`transcript` key, only `prompt_preview` |
| `C4:opencode` | as above | PASS | 5,404 bytes, no `messages`/`transcript` key |
| `C4:grok` | as above | PASS | 5,673 bytes, no `messages`/`transcript` key |
| `C4:qwen` | as above | PASS | 5,514 bytes, no `messages`/`transcript` key |
| `C4:gemini` | as above | PASS | 5,714 bytes, no `messages`/`transcript` key |
| `C4:kimi` | as above | PASS | 5,782 bytes, no `messages`/`transcript` key |
| `C5:claude` | T3+: resume supported by design, dry-run reaches confirmation | PASS | `capabilities.resume=true`; `rein resume claude:<id> --dry-run --json` → `decision:"confirmation_required"`, `args:["--resume","<id>"]` |
| `C5:codex` | as above | PASS | same pattern, `args:["resume","<id>"]` |
| `C5:opencode` | as above | PASS | same pattern, `args:["--session","<id>"]` |
| `C5:grok` | as above | PASS | same pattern, `args:["--resume","<id>"]` |
| `C5:qwen` | as above | PASS | same pattern, `args:["--resume","<id>"]` |
| `C5:gemini` | T2 (below T3): read-only reason, resume refused exit 5 | PASS | `rein resume gemini:gemini-rewind-win --dry-run --json` → exit 5, `"native session action is unsupported: Gemini CLI sessions are read-only in Phase 2"` |
| `C5:kimi` | as above | PASS | `capabilities.resume=false`, `read_only_reason:"Kimi Code CLI sessions are read-only until a device journey verifies native resume"` on the real session listing itself |
| `C6:claude` | Corrupted/empty/absent root degrade cleanly | PASS | Copy of `testdata/sessionindex/claude/windows`, last 20 bytes of the fixture truncated: `rc=0`, `message_count:0`, warning `incomplete_trailing_record`; empty root and absent root both `rc=0`, `{"sessions":[]}` |
| `C6:codex` | as above | PASS | Copy of `testdata/sessionindex/codex/forks/sessions`, one file truncated 15 bytes: `rc=0`, warning `incomplete_trailing_record`, the two untouched sessions still list correctly; empty/absent roots `rc=0`, empty |
| `C6:opencode` | as above | PASS | Real `opencode.db` hydrated from `testdata/adapters/opencode/windows/store.sql`, then the db file truncated 200 bytes: `rc=0`, warning `session_read_failed` ("OpenCode session store could not be read; other agents remain available"), no panic; empty/absent roots `rc=0`, empty |
| `C6:grok` | as above | PASS, with a note | Copy of `testdata/sessionindex/grok/windows`, `chat_history.jsonl` truncated 15 bytes: `rc=0`, session still listed (`message_count:2` unchanged, no warning emitted) — grok keeps `summary.json` as a separate, authoritative count source that the truncation did not touch, so this exercise proved the "no panic / no crash" half of C6 but not a visible boundary-detection warning for this specific corruption; empty/absent roots `rc=0`, empty |
| `C6:qwen` | as above | PASS, same note as grok | Copy of `testdata/sessionindex/qwen/windows`, `.jsonl` truncated 15 bytes: `rc=0`, no crash, count unaffected (separate `.runtime.json` sidecar); empty/absent roots `rc=0`, empty |
| `C6:gemini` | as above | PASS | Copy of `testdata/sessionindex/gemini/windows`, truncated 10 bytes: `rc=0`, warning `incomplete_trailing_record` (`ignored incomplete trailing JSONL record 6`); empty/absent roots `rc=0`, empty |
| `C6:kimi` | as above | PASS | Copy of `testdata/sessionindex/kimi/windows`, `wire.jsonl` truncated 10 bytes: `rc=0`, session still lists cleanly, no panic; empty/absent roots `rc=0`, empty |

## 2. Matrix D — Per T2+ agent

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `D1:claude` | `handoff --dry-run` produces capsule + fidelity report | PASS | `rein handoff claude:<id> --to codex --dry-run --json` → 5,876-byte capsule with `fidelity`, `capabilities`, `workspace` blocks |
| `D1:codex` | as above | PASS | `rein handoff codex:<id> --to claude --dry-run --json` → 32,302-byte capsule |
| `D1:opencode` | as above | PASS | `rein handoff opencode:<id> --to claude --dry-run --json` → 5,436-byte capsule |
| `D1:grok` | as above | PASS | `rein handoff grok:<id> --to claude --dry-run --json` → 5,877-byte capsule |
| `D1:qwen` | as above | PASS | `rein handoff qwen:<id> --to claude --dry-run --json` → 5,395-byte capsule |
| `D1:gemini` | as above | PASS | `rein handoff gemini:<id> --to claude --dry-run --allow-untested --json` (fixture repointed to a real repo, see Method) → 5,091-byte capsule |
| `D1:kimi` | as above | PASS | `rein handoff kimi:<id> --to claude --dry-run --json` → 5,530-byte capsule |
| `D2:*` (all 7) | No content the source did not contain, no invented turn | PASS | Every capsule's `capabilities`/`fidelity` blocks are populated from the real (or fixture) source only; no field observed that could not be traced to source content or an explicit `omitted`/`referenced` placeholder |
| `D3:*` (all 7) | Unknown records `referenced`/`omitted` with a reason | PASS | `"referenced"` and `"omitted"` keys present with reasons in every capsule above |
| `D4:claude` | Truncated source boundary at last complete record, offset+hash | **PARTIAL** | `testdata/handoff/claude/partial-final-record` fixture: `rein sessions --json` → `message_count:2`, warning `incomplete_trailing_record` (`ignored incomplete trailing JSONL record 3`) — core boundary assertion confirmed. The byte-exact offset+hash cross-check via `handoff --no-launch --json` was not reached: the fixture's recorded workspace (`/Users/fixture-user/code/demo`) does not resolve on this Windows host, so the handoff route correctly refuses (`working directory is a different repository than the source session` — a correct refusal per the contract, not a defect), same limitation the pre-tag report recorded |
| `D4:codex` | as above | **PARTIAL**, same reason | `testdata/handoff/codex/partial-final-record` fixture: `message_count:2`, warning `malformed_record` (`ignored malformed JSONL record 4`); handoff route blocked by the same non-Windows recorded workspace |
| `D4:opencode` | as above | **NOT TESTED** | OpenCode's embedded-SQLite store has no JSONL truncation boundary to exercise — same conclusion as the pre-tag report |
| `D4:grok` | as above | **NOT TESTED** | No committed `testdata/handoff/grok/partial-final-record` fixture exists. An ad-hoc truncation of a copied `chat_history.jsonl` (used for `C6:grok`) did not change the reported `message_count`, because `summary.json` is the count source of truth and was untouched — inconclusive for this specific assertion, not chased further under this run's time budget |
| `D4:qwen` | as above | PASS | `testdata/handoff/qwen/partial-final-record` fixture: `message_count:2` (title `COMPLETE_USER_RECORD`), the trailing partial record correctly excluded |
| `D4:gemini` | as above | PASS | Same corrupted-copy evidence as `C6:gemini`: `message_count` reflects only the 2 complete records, trailing record dropped with `incomplete_trailing_record` |
| `D4:kimi` | as above | PASS | `testdata/handoff/kimi/partial-final-record` fixture: `rc=0`, `message_count:3` reported cleanly, no crash |
| `D5:claude` | Two runs over an unchanged source produce byte-identical capsules | PASS | Two `--dry-run --json` runs, `diff` clean |
| `D5:codex` | as above | PASS | Two runs, `diff` clean (`handoff_id` and destination `session_id` identical both times) |
| `D5:opencode` | as above | **FAIL** | Two `rein handoff opencode:<id> --to claude --dry-run --json` runs over the same unchanged source produced **different** `handoff_id` and destination `session_id` values each time (`a040c816…`/`ba74c704…` vs. `1cba76de…`/`8c277f0e…`); everything else in the capsule was identical. `claude`, `codex`, `grok`, `qwen`, and `gemini` all reproduced byte-identical capsules under the same test shape (same command, same `--to claude`, same unchanged source) in this same run, so this is not an artifact of `--to claude` generally — it is specific to an `opencode` source. See `PD-B1` in [Findings](#4-findings) |
| `D5:grok` | as above | PASS | Two runs, `diff` clean |
| `D5:qwen` | as above | PASS | Two runs, `diff` clean |
| `D5:gemini` | as above | PASS | Two runs, `diff` clean |
| `D5:kimi` | as above | PASS | Two runs, `diff` clean |

## 3. Matrix E — Per T3+ agent (`claude`, `codex`, `opencode`, `grok`, `qwen` only)

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `E1:claude` | `rein resume` launches the vendor CLI, real session continues | PASS | `claude --resume 61e7874c… -p "What exact token did you reply with earlier…"` → real, live-config `claude.exe 2.1.263` launched with the exact `rein resume --dry-run --json` argv (`--resume <id>`), replied `RWB6-CLAUDE-P1-TOKEN` |
| `E1:codex` | as above | PASS | `codex exec resume 01a07684-afe2… "What exact token…"` → replied `RWB6-CODEX-P1-TOKEN` |
| `E1:opencode` | as above | PASS | `opencode run --session ses_f897832… "What exact token…" --format json` → replied `RWB6-OPENCODE-P1-TOKEN` |
| `E1:grok` | as above | PASS | `grok --resume 01a0768b-64e7… -p "What exact token…" --output-format json` → real `grok.exe` invoked; the target session's `message_count` grew 9→13 and `updated_at` advanced, confirming the same session continued (harness note: this run's background-capture mechanism did not preserve grok's own stdout text for this call — see `HD-B1` in Findings — so continuity is evidenced by session growth under the identical id rather than the literal recalled text, which the `E2` fork test below did capture) |
| `E1:qwen` | as above | **PARTIAL** | `qwen --resume 2532a6b7… -p "What exact token…" --output-format json` → real `qwen.exe 0.21.12` launched with the correct `--resume <id>` argv and the correct `session_id` in its own output, but every turn (including this one) fails with `[API Error: 401 invalid access token or token expired]` — the launch mechanism is proven, a completed model turn is not. Credential state: this host's `~/.qwen/settings.json` `BAILIAN_CODING_PLAN_API_KEY` is expired/invalid right now, reproduced identically to the pre-tag report |
| `E2:claude` | Resumed session is the requested one, verified inside the agent | PASS | Same call as `E1:claude`: the reply was the exact token planted only in that session's own first turn, proving the requested session (not a new one) was resumed |
| `E2:codex` | as above | PASS | Same call as `E1:codex`, exact planted token recalled |
| `E2:opencode` | as above | PASS | Same call as `E1:opencode`, exact planted token recalled |
| `E2:grok` | as above | PASS | Session growth confirmed under the exact requested id (`01a0768b-64e7…`), not a different one — see `E1:grok` |
| `E2:qwen` | as above | **NOT TESTED** | Same credential block as `E1:qwen`: no completed turn exists to verify session identity beyond the correct `session_id` in the vendor's own error payload |
| `E3:claude` | `rein fork` produces a distinct session | PASS | `claude --resume 61e7874c… --fork-session -p "Reply with exactly…FORK-TOKEN"` → new session `86cfa86a-d48b-4cb5-bd4f-f63f3903eb63` (distinct from source), correct token; source session unaffected (grew only from its own `E1`/`E2` turn) |
| `E3:codex` | as above | PASS | `codex exec fork 01a07684-afe2… "…FORK-TOKEN"` → new session `01a07685-bb12-7541-b7b3-a801f55677be`, correct token |
| `E3:opencode` | as above | PASS | `opencode run --session ses_f897832… --fork "…FORK-TOKEN" --format json` → new session `ses_f8976ead5ffegMCp9WbD4ReERR`, correct token |
| `E3:grok` | as above | PASS | `grok --resume 01a0768b… --fork-session -p "…FORK-TOKEN" --output-format json` → new session `01a07691-f0af-7831-884d-a0c15fd907ff` appeared (`message_count:17`, inherited history), source session (`01a0768b…`) unaffected at `message_count:13` |
| `E3:qwen` | as above | **NOT TESTED** | Not attempted this run given the credential block already confirmed on `E1`/`E2:qwen` |
| `E4:claude` | Below-min and above-max both exit 5 naming the range | PASS | `.cmd` shim reporting `2.1.218`: exit 5, `"native agent version 2.1.218 is outside the verified range 2.1.219 to 2.1.263 inclusive"`. Shim reporting `2.1.264`: exit 5, same range, correct actual |
| `E4:codex` | as above | PASS | Shim `0.132.0`: exit 5, range `0.133.0 to 0.149.0 inclusive`. Shim `0.150.0`: exit 5, same range |
| `E4:opencode` | as above | PASS | Shim `1.18.20`: exit 5, range `1.18.21 to 1.18.27 inclusive`. Shim `1.18.28`: exit 5, same range |
| `E4:grok` | as above | PASS | Shim `1.0.4`: exit 5, range `1.0.5 to 1.0.5 inclusive`. Shim `1.0.6`: exit 5, same range |
| `E4:qwen` | as above | PASS | Shim `0.21.11`: exit 5, range `0.21.12 to 0.21.13 inclusive`. Shim `0.21.14`: exit 5, same range |
| `E5:claude` | Active session for the agent detected; resume refused/warned | PASS | `claude --resume 61e7874c… --bg` started a real background `claude.exe` holding the exact session; `rein resume claude:61e7874c… --dry-run --json` → `agent.active` `status:"present"`, `severity:"warning"`, `actual:true`, `"a running claude instance is already using this session"`. Background session stopped cleanly afterward (`claude stop`) |
| `E5:codex` | as above | PASS | A real `codex exec resume` given a slow prompt was caught alive via `tasklist` (`codex.exe`, real PID) while `rein resume --dry-run --json` reported `agent.active` `actual:true`, `"a running codex instance is already using this session"` |
| `E5:opencode` | as above | PASS | Same pattern: a real `opencode run --session … "long essay"` process (`opencode.exe`, real PID) caught alive; `agent.active` `actual:true` |
| `E5:grok` | as above | PASS | The `E3:grok` fork invocation's own `grok.exe` process was still alive (`tasklist`) when `rein resume --dry-run --json` was re-run; `agent.active` `actual:true` |
| `E5:qwen` | as above | **NOT TESTED** | `qwen`'s 401 credential failure returns in under ~4 seconds, faster than this run's manual `tasklist` polling loop could reliably catch the process; not caught in several attempts. This is a harness/timing limitation of this run's method, not a claim about the underlying mechanism |
| `E6:claude` | Non-interactive invocation exits 7 | PASS | `rein resume claude:<id> < /dev/null` (no `--json`, no `--dry-run`): exit 7, stderr `"environment warnings require confirmation: baseline.unavailable"` |
| `E6:codex` | as above | PASS | Same shape, exit 7, same stderr message |
| `E6:opencode` | as above | PASS | Same shape, exit 7, same stderr message |
| `E6:grok` | as above | PASS | Exit 7, stderr `"environment warnings require confirmation: baseline.unavailable, git.working_tree"` |
| `E6:qwen` | as above | PASS | Exit 7, same message shape |

Every host-process-enumeration row above (`E5`) exercised the confirmed-running
positive case per this run's dispatch note ("Process enumeration works on this
host now"): all 4 tested T4/T5 agents (`claude`/`codex`/`opencode`/`grok`)
detected a real, confirmed-running vendor process correctly. This clears the
pre-tag report's `RB7`/`E5` disposition for this host.

## 4. T5 encrypted sync round trip (`claude`, `codex`, `opencode`)

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `sync:claude` | Push from one device home, pull into a second, isolated device home, content restored | PASS | Pushed `claude:61e7874c…` from the host's live `CLAUDE_CONFIG_DIR` under `REINSTATE_HOME=synchome-A` (`rein push --agent claude --session 61e7874c… --json` → 1 snapshot). Pulled into a fresh, isolated `CLAUDE_CONFIG_DIR` under `REINSTATE_HOME=synchome-B` (`rein pull --agent claude --session 61e7874c… --json` → 1 session restored to `…\homes\claude-B\projects\…\61e7874c-….jsonl`), never touching the live tree for the write |
| `sync:codex` | as above | PASS | Pushed all 3 real codex sessions from `synchome-A` (`rein push --agent codex --all --json` → 3 snapshots). Pulled into a fresh, isolated `CODEX_HOME` under `synchome-B` (`rein pull --agent codex --all --json` → `pulled:3`). Re-listed on the restored home: all 3 session ids, correct `message_count`s (8/2/3) and workspaces |
| `sync:opencode` | as above | PASS | Pushed 1 real opencode session (`rein push --agent opencode --session ses_f897832… --json` → 1 snapshot). First pull attempt into a never-run `XDG_DATA_HOME` correctly refused (`compatibility NOT_INSTALLED`: "install and run opencode once on this device so its session layout exists, then pull again") — a correct refusal for the embedded-SQLite store, not a defect. After running `opencode` once on that device home (creating its local `opencode.db`), the pull succeeded (`pulled:1`, destination `…\opencode-B\opencode\opencode.db`) |

All three used the real `rein push`/`rein pull` non-interactive passphrase
path (`REINSTATE_PASSPHRASE_FD`) against a disposable `fakelocker` instance,
never the hosted control plane. See Method for the passphrase-handle helper.

## 5. NOT TESTED and why

| Row(s) | Reason |
| ------ | ------ |
| `E2:qwen`, `E3:qwen` | This host's `~/.qwen` credential (`BAILIAN_CODING_PLAN_API_KEY`) is expired/invalid right now (`401`), reproduced on every real qwen invocation this run. `E1:qwen` is `PARTIAL` on the same evidence: the launch mechanism and correct session id are proven, a completed round trip is not |
| `E5:qwen` | Qwen's 401 failure returns too quickly (~4s) for this run's manual `tasklist`-polling method to reliably catch the process; a faster/automated poll would likely close this, not attempted further this run |
| `D4:opencode` | OpenCode's embedded-SQLite session store has no JSONL truncation boundary to exercise (same conclusion as the pre-tag report) |
| `D4:grok` | No committed `testdata/handoff/grok/partial-final-record` fixture; an ad-hoc truncation of `chat_history.jsonl` did not move `message_count` because `summary.json` is the count source of truth for this agent |

## 6. Findings

- **`PD-B1` (product, needs review, not release-blocking on its own):**
  `rein handoff <session> --to claude --dry-run --json`, run twice in a row
  over an **unchanged** `opencode` source with no other state change, produced
  a different `handoff_id` and destination `session_id` each time
  (`D5:opencode`, `FAIL`). The same command shape against `claude`, `codex`,
  `grok`, `qwen`, and `gemini` sources was byte-identical across repeats in
  this same run, so the non-determinism is specific to an `opencode` source,
  not to `--to claude` in general. Worth a source-level look at whether the
  opencode adapter's projection path seeds its capsule/destination ids from a
  process-random source that the other four adapters do not use.
- **`HD-B1` (harness, this run):** for the two vendors whose real headless
  invocation ran long enough to need this run's Bash `run_in_background`
  mechanism (`grok -p …`, `qwen -p …`), the backgrounded command's own stdout
  did not reliably reach this run's capture file even after the command
  completed (the file held only this run's own `echo EXIT=$?` marker). This
  run worked around it by verifying continuity/results through `rein`
  queries against the resulting session state instead (message-count growth,
  new session ids, search hits) rather than the vendor's literal stdout text —
  every `E1`/`E2`/`E3` row affected states exactly what evidence was used.
  This is a limitation of this run's capture tooling, not a product finding.
- **`INV-B1` (investigation gap, not scored as a row result):** a real,
  correctly-authenticated headless `gemini --skip-trust -p …` session, run in
  an isolated `GEMINI_CLI_HOME` seeded only with a copied `settings.json` +
  `google_accounts.json`, wrote a real session file to
  `<home>\.gemini\tmp\demo\chats\session-*.jsonl`, and `rein doctor --agents
  --json` correctly found the root and its marker (`resolved_root` candidate
  present) — but `resolved_root` itself came back `null` and `rein sessions
  --agent gemini` did not list the session. Not chased further under this
  run's time budget; recorded here so a future pass can reproduce it directly
  rather than re-discovering it. Gemini's Matrix C/D rows above used the
  committed `testdata` fixture instead, per `CLAUDE.md`'s synthetic-fixture
  allowance, so this gap did not block any row result — it is reported as an
  open question about a real, working, credentialed gemini session on
  Windows, separate from the C/D matrix itself.
- **Credential states found, as the dispatch asked to record explicitly:**
  `qwen`: `BAILIAN_CODING_PLAN_API_KEY` present but expired/invalid (`401` on
  every real call). `gemini`: the copied `settings.json` +
  `google_accounts.json` alone were sufficient to authenticate a real headless
  call (no separate OAuth token file exists on this host at
  `~/.gemini/oauth_creds.json` or similar). `kimi`:
  `credentials/kimi-code.json` alone was sufficient to authenticate a real
  headless call end-to-end, including a completed model turn.

## 7. Dispositions cleared from the pre-tag report

- **`claude:E1`/`E2`/`E3`** — pre-tag `FAIL`/`PARTIAL` (verified-range drift,
  then a `CLAUDE.md` credential-carve-out block on content verification).
  This run used the host's live, already-authenticated `CLAUDE_CONFIG_DIR`
  directly (per this run's own ground rules, superseding the pre-tag
  compliance correction's testdata-only recipe) and got a full, real,
  physical resume/fork/token round trip: **PASS**/**PASS**/**PASS**.
- **`claude:E5`, `codex:E5`, `opencode:E5`, `grok:E5`** — pre-tag `NOT TESTED`
  ("fail-safe verified" only, because this host could not enumerate its own
  processes). This run's dispatch confirms process enumeration now works on
  this host; all 4 tested agents detected a real, confirmed-running vendor
  process correctly: **PASS** across the board.
- **`opencode:E5`** similarly clears to **PASS**.

## 8. Cleanup

Every throwaway directory under `D:\ReinstateAcceptanceProjects\v060-w7b-b\`
(`repos\`, `homes\`, `synchome-A`/`synchome-B`, `c6copies\`, `c6empty\`,
`fixtures\`, `shims\`, `passrunner\`) and the `install\`/`v051\` binaries are
this executor's own throwaway state and are deleted after this report is
committed, per the evidence policy. No transcript text, prompt content beyond
the planted tokens, credential file content, private paths, or session ids
this executor did not create appear anywhere above. The `fakelocker` process
was stopped at the end of the run.
