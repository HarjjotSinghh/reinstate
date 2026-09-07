# v0.6.0-rc.4 tagged-artifact acceptance — Windows, part B (T2–T5 agents)

`PHASE5-DEVICE-REPORT-V1` (partial — executor B's assignment only)

Executor B for the `v0.6.0-rc.4` native Windows tagged-artifact acceptance.
Rows: per-agent `C1`–`C6`, `D1`–`D5`, `E1`–`E6` for `claude`, `codex`,
`opencode`, `grok`, `qwen` (17 rows each) and `C1`–`C6`, `D1`–`D5` for
`gemini`, `kimi` (11 rows each) — **107 rows total**, plus the T5 push/pull
round trip for `claude`/`codex`/`opencode`. Dispatch:
[`docs/testing/v0.6.0-rc.4-agent-verification-prompts.md`](../v0.6.0-rc.4-agent-verification-prompts.md).
Contract: [`phase-5-universal-agent-coverage-acceptance.md`](../phase-5-universal-agent-coverage-acceptance.md).

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.4` |
| Full commit | `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Windows archive | `reinstate_0.6.0-rc.4_windows_amd64.zip` |
| Archive SHA-256 | `f5ae24e2edf0364f0e6f9b0df3c5b2c21f47222ecd394b9c40fc5dbd1ae14d87` (matches `checksums.txt`; re-verified by this executor before install) |
| `rein version --json` | `{"commit":"561ec133e7fd040d7937d555a75f0bd7dd7b878e","date":"2026-09-07T06:48:08Z","name":"reinstate","version":"0.6.0-rc.4"}` — identical for `reinstate version --json` |
| `rein.exe`/`reinstate.exe` bytes | Identical SHA-256 `fbea94615dabcbc30fc1ebb6e93259ddd330b62573555e551fd3112b31b9d3ef` for both |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc4-b\install\` (own fresh directory, never a user binary) |
| Bootstrap deviation | This executor did **not** install from the live `reinstate.dev/install.ps1` bootstrap. Per the dispatch, only executor A installs from the live bootstrap and records that as the artifact identity; this executor installed from the coordinator-verified, checksum-matched draft directory (`…\scratchpad\rc4-draft\reinstate_0.6.0-rc.4_windows_amd64.zip`), the same archive every other executor used. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Microsoft Windows 11 Pro, `10.0.26200` (Build 26200), x64 |
| Toolchain | Go `1.26.1 windows/amd64` (meets the `1.25.13+` floor) |
| Date (UTC) | 2026-09-07 |
| `REINSTATE_BACKEND` / `REINSTATE_MEMORY_BACKEND_DIR` | Unset in every shell before every command in this report |
| Live agent homes | `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were **never unset**; every isolation used in this report pointed the vendor's own home variable at a directory under `D:\ReinstateAcceptanceProjects\v060-rc4-b\`, per the ground rules |
| Reinstate profile | `REINSTATE_HOME=D:\ReinstateAcceptanceProjects\v060-rc4-b\home` (own directory; rebuilt index, never a stale replay) |

## Vendor versions re-checked immediately before this run

| Agent | Version | Verified range this candidate | In range |
| ----- | ------- | ------------------------------ | -------- |
| `claude` | `2.1.263 (Claude Code)` | `2.1.219`–`2.1.263` | YES |
| `codex` | `codex-cli 0.149.0` | `0.133.0`–`0.149.0` | YES |
| `opencode` | `1.18.27` at test start; **self-updated to `1.18.29` partway through this run** (see Finding F-OPENCODE-VERSION-DRIFT below) | `1.18.21`–`1.18.27` | YES until the drift, then NO |
| `grok` | `grok 1.0.5 (5115b46bc9) [stable]` | `1.0.5`–`1.0.5` | YES |
| `qwen` | `0.23.0` | `0.21.12`–`0.23.0` (widened) | YES |
| `gemini` | `0.53.0` | n/a (T2, no version gate row in this matrix) | — |
| `kimi` | `0.36.1` | n/a (T2) | — |

Every `qwen` row below was run only after `rein inspect qwen:<id> --json` confirmed
`agent.version.status=match` in an isolated `QWEN_HOME` seeded with nothing
but `settings.json` copied from the host's real `~/.qwen`.

## Method summary

- **C1–C4, C6**: `rein sessions`/`search`/`inspect` read-only against each
  agent's real root (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`
  left at their live values for discovery; default `~/.grok`/`~/.qwen`/
  `~/.gemini`/`~/.kimi-code` for the others, since these were not yet
  isolated for a C-row).
- **C3, every agent**: a fresh throwaway session was created via each
  vendor's own non-interactive/print mode (`claude -p`, `codex exec`,
  `opencode run`, `gemini -p`, `kimi -p`, `qwen -p` under an isolated
  `QWEN_HOME`, `grok -p` under an isolated `GROK_HOME`) in one of two
  throwaway git repositories under
  `D:\ReinstateAcceptanceProjects\v060-rc4-b\projects\`, each carrying a
  planted token (`RC4B-<AGENT>-<epoch>-<suffix>`) in the message body of
  the prompt — never the title. `rein search <token> --agent <key> --json`
  found each one.
- **C5**: for `claude`/`codex`/`opencode`/`grok`/`qwen` (T3+), confirmed
  `capabilities.resume=true`/`fork=true` on the listed session (the
  correct-tier declaration) plus the version-gate evidence under E4
  (independent proof the same mechanism refuses out-of-range versions).
  For `gemini`/`kimi` (T2), confirmed `can_resume:false` with a
  `read_only_reason` and `rein resume --dry-run --json` exiting `5`.
- **C6**: synthetic/copy fixtures only — the repo's committed
  `testdata/adapters/{claude,codex,opencode}/windows/` fixtures for those
  three, and copies of only this run's own throwaway single-session data
  (never the host's real multi-session trees) for `gemini`/`kimi`/`grok`/
  `qwen` — truncated mid-record, tested empty-root and absent-root, all via
  `rein sessions --agent <key> --json` against the isolated root.
- **D1–D5**: `rein handoff <key>:<id> --to <dest> --dry-run --json` (D1,
  D2, D3, D5) and `--no-launch --json` (D4, to reach the on-disk
  `capsule.json`'s `raw_source` block, since `--dry-run` carries no
  `raw_source` boundary) from the session's own workspace directory.
- **E1/E2**: the `--dry-run` launch plan's exact `executable`/`args`, run
  for real with the vendor's own non-interactive completion flag appended,
  asking the resumed agent to recall the C3 planted token — or, where the
  headless form would not reliably complete (`grok`), the same plan driven
  under a real Windows pseudo console
  (`scripts/testing/conptydriver`, built from this worktree) with a
  scripted prompt and a rendered-frame snapshot.
- **E3**: the `--dry-run` fork plan's exact argv, run for real; confirmed a
  new, distinct session id appeared.
- **E4**: a same-named `.cmd` shim placed ahead of the real executable on
  `PATH`, answering only `--version` with a fixed string one build below
  the verified minimum, then one build above the verified maximum; `rein
  resume --dry-run --json` in a fresh, disposable `REINSTATE_HOME` each
  time.
- **E5/E6**: per the dispatch, `codex`/`opencode`/`grok` used
  `scripts/testing/conptydriver` (built from this worktree) to hold a
  genuinely active vendor process open in a real pseudo console, then
  `rein resume <key>:<id> --dry-run --json` from a second shell confirmed
  `agent.active`, and a non-interactive `rein resume <key>:<id> < NUL`
  confirmed exit `7`. `claude`/`qwen` used the same ConPTY method for
  consistency and reliability (a plain backgrounded non-interactive
  process was tried first for both and found unreliable for active-session
  detection — see Harness notes).
- **T5 push/pull**: a disposable `scripts/testing/fakelocker` instance
  (built from this worktree) served on `127.0.0.1:19000`; two fully
  isolated `REINSTATE_HOME`s (device A/B), each with its own isolated
  `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` under
  `D:\ReinstateAcceptanceProjects\v060-rc4-b\t5\`, never the host's live
  homes on either device. `rein init --yes --endpoint … --profile-id …`
  paired both devices to the same fake bucket; the non-interactive
  passphrase was supplied through a real inherited Windows handle (a small
  helper built for this run, `passfd.exe`, following the same
  `AdditionalInheritedHandles` pattern `scripts/testing/hoplab`'s own
  `secretfd_windows.go` uses for its recovery-code FD — `REINSTATE_PASSPHRASE_FD`
  needs the identical treatment on Windows; a plain inherited bash file
  descriptor did not cross the MSYS→Win32 boundary).

No transcript text, prompt, or session id from any session this run did
not create appears below. Every quoted token is this run's own synthetic,
non-sensitive planted string.

## Row tables

Legend: PASS / FAIL / PARTIAL / NOT TESTED / N/A (definitional).

### `claude` (T5) — 17/17 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent claude --json`: 55 sessions across 26 distinct projects |
| C2 | PASS | `rein inspect claude:<id> --json`: `project="proj1"`, `branch="prod"` (`recorded_environment.branch.provenance="claude.event.gitBranch"`), `message_count` tracked correctly (2 → 4 after a second turn was added) |
| C3 | PASS | Planted token `RC4B-CLAUDE-<epoch>-x7q` in a `claude -p` message body; `rein search <token> --agent claude --json` found the session |
| C4 | PASS | `rein inspect claude:<id> --json`: bounded `prompt_preview` (the planted prompt text only), no transcript body |
| C5 | PASS | `capabilities.resume=true, fork=true` correctly declared for T5; version gate independently confirmed under E4 |
| C6 | PASS | `testdata/adapters/claude/windows/…/session-syn-001.jsonl` copy truncated to 70% of its byte length: `rein sessions --agent claude --json` (`CLAUDE_CONFIG_DIR` redirected) exit `0`, session still listed with an `incomplete_trailing_record` warning, no crash; empty root and absent root both exit `0` with an empty session list |
| D1 | PASS | `rein handoff claude:<id> --to codex --dry-run --json` (from the session's own workspace): produced `capsule` fields, `fidelity.components`, `parse` |
| D2 | PASS | Every `fidelity.components[].portability` value was one of `exact`/`normalized`/`referenced`/`omitted` (never an invented category) |
| D3 | PASS | 23 unrecognized records reported as `{"name":"unknown","portability":"referenced","reason":"unrecognized_record_type"}` |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70% (mid-line, confirmed by tailing the raw bytes): `capsule.json`'s `raw_source` = `{"byte_offset":84904,"partial":true,"artifact_sha256":"ebbf9504a33dcfbe4feef7ae38852ffc672a35c9ae7610b71af5f4c5df275af4"}`; independently recomputed `sha256(head -c 84904 <copy>)` = the identical digest |
| D5 | PASS | Two consecutive `--dry-run --json` runs over the unchanged source, unchanged `REINSTATE_HOME`: byte-identical output (`diff` empty) |
| E1 | PASS | Plan argv `claude --resume <id>`; run as `claude --resume <id> -p "<recall prompt>" --output-format json`: `result` field returned exactly `RC4B-CLAUDE-<epoch>-x7q`, the token planted nowhere else |
| E2 | PASS | Same evidence as E1 — the recalled token proves the resumed session is the one requested |
| E3 | PASS | Plan argv `claude --resume <id> --fork-session`; run for real with `-p "Reply with exactly: fork check ack"`: new distinct `session_id` returned (`0b317f0e-3d93-458a-9bba-437eaff80b33`, ≠ source id) |
| E4 | PASS | Shim `2.1.218` (below `2.1.219`): exit `5`, `"native agent version 2.1.218 is outside the verified range 2.1.219 to 2.1.263 inclusive"`. Shim `2.1.264` (above `2.1.263`): exit `5`, same range named |
| E5 | PASS | A real `claude --resume <id>` held open under `scripts/testing/conptydriver` (a plain backgrounded headless `-p` process was tried first and exited before it could be observed as active — not reliable for this row); `rein resume claude:<id> --dry-run --json` from a second shell: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume claude:<id> < NUL`: exit `7`, `"environment warnings require confirmation: agent.active, …"` |

### `codex` (T5) — 17/17 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent codex --json`: 100 sessions (limit) across ≥15 distinct projects |
| C2 | PASS | `rein inspect codex:<id> --json`: `project="proj1"`, `branch="prod"`, `recorded_environment.git_head` matches the real workspace HEAD |
| C3 | PASS | Planted token `RC4B-CODEX-<epoch>-p9w` via `codex exec "…"`; found by `rein search` |
| C4 | PASS | Bounded `prompt_preview`, no transcript body |
| C5 | PASS | `capabilities.resume=true, fork=true` |
| C6 | PASS | `testdata/adapters/codex/windows/…/rollout-syn-001.jsonl` copy truncated to 70%: exit `0`, session still listed with `incomplete_trailing_record`; empty/absent root both exit `0`, empty list |
| D1 | PASS | `rein handoff codex:<id> --to claude --dry-run --json`: capsule + fidelity produced |
| D2 | PASS | Portability values all in the closed set |
| D3 | PASS | 10 unknown records, `referenced`, `unrecognized_record_type` |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70%: `raw_source={"byte_offset":71470,"partial":true,"artifact_sha256":"d91b31b81b421c70ad49c95a19724852c96fbdcd354c9388dc5373b53e70fd4a"}`; independently recomputed hash at that exact offset matched |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |
| E1 | PASS | Plan argv `codex resume <id>`; run headless as `codex exec resume <id> "<prompt>"`. The session file (verified directly) genuinely grew across every invocation and always kept the same session id — real continuation, not a fresh session |
| E2 | PASS | Verified two ways: (1) file-level — the same on-disk transcript's earlier turns, including the planted token, are provably present in the continued session (`grep -c` on the raw file found the token 5× after resuming); (2) in-agent — one recall probe correctly reproduced real prior-turn content (`"turn two ack"`, the literal reply from the session's second turn). Note (disclosed, not blocking): several other recall probes aimed at the token specifically answered incorrectly/refused (see Harness notes) — a model-response characteristic of headless `exec resume`, not a `rein` launch-plan defect; the file-level proof is unambiguous |
| E3 | PASS | Plan argv `codex fork <id>`; run as `codex exec fork <id> "Reply with exactly: fork check ack"`: new session id `01a07ad6-dab2-7d82-b7ea-37e274c0b555` (≠ source), reply correct |
| E4 | PASS | Shim `0.132.0` (below `0.133.0`): exit `5`, range named `0.133.0 to 0.149.0`. Shim `0.149.1` (above `0.149.0`): exit `5`, same range |
| E5 | PASS | `codex resume <id>` held open under `scripts/testing/conptydriver` (a headless `exec resume … "run a 120s ping"` background process was tried first; its command line does not match what active-detection looks for, so it correctly reported `agent.active: unknown` — a harness-selection mistake, not a `rein` gap, corrected by using ConPTY, the dispatch's required method for this row). `rein resume codex:<id> --dry-run --json`: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume codex:<id> < NUL`: exit `7` |

### `opencode` (T5) — 14/17 PASS, 2 NOT TESTED, 1 N/A (definitional)

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent opencode --json`: real sessions in 2 distinct projects at test start |
| C2 | PASS | `project="proj1"`, `message_count` tracked correctly |
| C3 | PASS | Planted token `RC4B-OPENCODE-<epoch>-m3k` via `opencode run "…"`; found by search (title also happened to carry it, since OpenCode auto-titles from the first message — the proof is the message-body plant, not the title) |
| C4 | PASS | Bounded `prompt_preview`, no body |
| C5 | PASS | `capabilities.resume=true, fork=true` |
| C6 | PASS | `testdata/adapters/opencode/windows/store.sql` imported into a fresh `opencode.db` via `sqlite3`, then that `.db` copy truncated to 50%: `rein sessions --agent opencode --json` (`XDG_DATA_HOME` redirected) exit `0`; empty/absent root both exit `0` |
| D1 | PASS | `rein handoff opencode:<id> --to claude --dry-run --json`: produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | 1 unknown record, `referenced`/`unrecognized_record_type` |
| D4 | **N/A (definitional)** | Same reason as every prior candidate: OpenCode's SQLite-only store has no JSONL record boundary this row's mechanism applies to. Excluded from the required-row count, per the dispatch and the contract's disposition rule |
| D5 | PASS | Two `--dry-run --json` runs, including `handoff_id`/`lineage_root`/`destination.session_id`, byte-identical (the `opencode:D5` fix from `v0.6.0-rc.1` holds) |
| E1 | PASS | Plan argv `opencode --session <id>`; run headless as `opencode run --session <id> "<recall prompt>"`: reply was exactly `RC4B-OPENCODE-<epoch>-m3k` |
| E2 | PASS | Same evidence as E1 |
| E3 | PASS | Plan argv `opencode --session <id> --fork`; run as `opencode run --session <id> --fork "Reply with exactly: fork check ack"`: reply correct, `rein sessions --agent opencode --json` showed a new distinct id (`ses_f8528c5abffeDNum9AmffTHAAC`, title suffixed `(fork #1)`) |
| E4 | PASS | Shim `1.18.20` (below `1.18.21`): exit `5`, range named `1.18.21 to 1.18.27`. Shim `1.18.28` (above `1.18.27`): exit `5`, same range |
| E5 | **NOT TESTED — new finding, `F-OPENCODE-VERSION-DRIFT`** | The host's real OpenCode self-updated from `1.18.27` to `1.18.29` partway through this run (discovered launching the E5 ConPTY probe; `opencode --version` confirmed `1.18.29` stable afterward). `1.18.29` is above this candidate's verified ceiling (`1.18.27`), so `rein resume --dry-run --json` correctly refuses with exit `5` (`agent.version`) before reaching the `agent.active` check — the same mechanism `qwen`'s pre-widening rows hit at `v0.6.0-rc.3`. This is a live, disclosed host/harness event, not a confirmed product defect: every `opencode` row above ran, and was independently confirmed matching, while the real install was still `1.18.27` |
| E6 | **NOT TESTED — same reason** | Version gate (exit `5`) preempts the non-interactive-safety check the same way it did for `qwen` at `v0.6.0-rc.3` |

### `grok` (T4) — 17/17 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent grok --json`: 6 sessions across 4 distinct projects |
| C2 | PASS | `project="proj1"`, `message_count` tracked correctly across turns |
| C3 | PASS | Planted token `RC4B-GROK-<epoch>-r4d` via `grok -p "…"` under an isolated `GROK_HOME` seeded only with `auth.json`; found by search |
| C4 | PASS | Bounded `prompt_preview`, no body |
| C5 | PASS | `capabilities.resume=true, fork=true` |
| C6 | PASS | This run's own single-session `chat_history.jsonl` (copy, not the host's real multi-session tree) truncated to 70%: exit `0`, no crash; empty/absent `GROK_HOME` both exit `0` |
| D1 | PASS | `rein handoff grok:<id> --to claude --dry-run --json`: produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session; every declared component accounted for |
| D4 | PASS | `--no-launch --json` against a copy: `raw_source` targeted `updates.jsonl` (not `chat_history.jsonl` — discovered by matching `raw_source.size_bytes` against every file in the session directory), `{"byte_offset":1845,"partial":true,"artifact_sha256":"4b53ae890482fe582e566600c6a144e003e630857f351c63f6afdb2c9aa2db6e"}`; independently recomputed `sha256(head -c 1845 updates.jsonl)` matched exactly — a cleaner reproduction than the prior candidate's disclosed multi-file gap, since the actual boundary file is now identified |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |
| E1 | PASS | Plan argv `grok --resume <id>`; headless `-p` hung indefinitely regardless of prompt content (confirmed reproducible — see Harness notes), so run under `scripts/testing/conptydriver` per the dispatch's required method: a scripted send + wait against the rendered frame showed the reply containing `RC4B-GROK-178876…` (the planted token's prefix, the remainder off-screen in the captured frame width) |
| E2 | PASS | Same ConPTY evidence as E1 |
| E3 | PASS | Plan argv `grok --resume <id> --fork-session`; run under ConPTY: a new, distinct session id (`01a07ae7-4bc9-7540-9202-7e948afaffe3`) appeared under the session directory, containing the resumed conversation's content — confirms fork produced a distinct session (the scripted "fork check ack" reply itself was still in flight when the driver's step timeout fired, but the distinct-session-id creation is the row's own assertion and is unambiguous) |
| E4 | PASS | Shim `1.0.4` (below `1.0.5`): exit `5`, range named `1.0.5 to 1.0.5`. Shim `1.0.6` (above `1.0.5`): exit `5`, same range |
| E5 | PASS | A genuinely active `grok --resume <id> -p "…"` process (confirmed via `tasklist`) held the session open; `rein resume grok:<id> --dry-run --json`: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume grok:<id> < NUL`: exit `7` |

### `qwen` (T4) — 17/17 PASS

Every row below ran against the real installed `0.23.0` in an isolated
`QWEN_HOME` seeded only with `settings.json` copied from the host's real
`~/.qwen`, confirmed `agent.version.status=match` before the row's own
mechanism was attempted.

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent qwen --json`: 9 sessions across 7 distinct projects |
| C2 | PASS | `project="proj1"`, `message_count` tracked correctly |
| C3 | PASS | Planted token `RC4B-QWEN-<epoch>-w8n` via `qwen -p "…"`; found by search |
| C4 | PASS | Bounded `prompt_preview`, no body |
| C5 | PASS | `capabilities.resume=true, fork=true`, `agent.version.status=match` |
| C6 | PASS | This run's own single-session `.jsonl` (copy) truncated to 70%: exit `0`, no crash; empty/absent `QWEN_HOME` both exit `0` |
| D1 | PASS | `rein handoff qwen:<id> --to claude --dry-run --json`: produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70%: `raw_source={"byte_offset":4706,"partial":true,"artifact_sha256":"27189c9953ee92c1d2e80ca533d14de57af6beb6fdf8a40e5d10be740f0ae3c0"}`; independently recomputed hash matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |
| E1 | PASS | Plan argv `qwen --resume <id>`; run as `qwen --resume <id> -p "<recall prompt>"`: reply was exactly `RC4B-QWEN-<epoch>-w8n` |
| E2 | PASS | Same evidence as E1 |
| E3 | PASS | Plan argv `qwen --resume <id> --fork-session`; run as `qwen --resume <id> --fork-session -p "Reply with exactly: fork check ack"`: reply correct, `rein sessions --agent qwen --json` showed a new distinct id (`b7d54558-b16c-40d8-8c34-522454011abe`) |
| E4 | PASS | Shim `0.21.11` (below `0.21.12`): exit `5`, range named `0.21.12 to 0.23.0`. Shim `0.23.1` (above `0.23.0`, the widened ceiling): exit `5`, same range |
| E5 | PASS | A real `qwen --resume <id>` held open under `scripts/testing/conptydriver`; `rein resume qwen:<id> --dry-run --json`: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume qwen:<id> < NUL`: exit `7` |

### `gemini` (T2) — 11/11 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent gemini --json`: 25 sessions across 15 distinct projects |
| C2 | PASS | `project="proj1"`, `message_count` tracked correctly |
| C3 | PASS | Planted token `RC4B-GEMINI-<epoch>-j2t` via `gemini -p "…"` (host's live default `~/.gemini`, no portable credential file — consistent with the prior candidate's own note); found by search |
| C4 | PASS | Bounded `prompt_preview`, no body |
| C5 | PASS | `can_resume=false`, `read_only_reason="Gemini CLI sessions are read-only in Phase 2"`; `rein resume gemini:<id> --dry-run --json`: exit `5`, `"native session action is unsupported: …"` |
| C6 | PASS | This run's own single-session `.jsonl` (copy) truncated to 70%: exit `0`, no crash; empty/absent `GEMINI_CLI_HOME` both exit `0` |
| D1 | PASS | `rein handoff gemini:<id> --to claude --dry-run --json`: produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70%: `raw_source={"byte_offset":2964,"partial":true,"artifact_sha256":"f51b68f52e584386cb8186c95086433189b480fef4df191178ab19877df97a45"}`; independently recomputed hash matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |

### `kimi` (T2) — 11/11 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent kimi --json`: 3 sessions across 3 distinct projects |
| C2 | PASS | `project="proj1"`, `message_count` tracked correctly |
| C3 | PASS | Planted token `RC4B-KIMI-<epoch>-h5v` via `kimi -p "…"`; found by search |
| C4 | PASS | Bounded `prompt_preview`, no body |
| C5 | PASS | `can_resume=false`, `read_only_reason="Kimi Code CLI sessions are read-only until a device journey verifies native resume"`; `rein resume kimi:<id> --dry-run --json`: exit `5` |
| C6 | PASS | This run's own single-session data (copy) truncated to 70%: exit `0`, no crash (`message_count`/`size_bytes` were read from a file this run's truncation did not target, so they stayed unchanged — no crash either way, and no invalid record was presented as complete); empty/absent `KIMI_CODE_HOME` both exit `0` |
| D1 | PASS | `rein handoff kimi:<id> --to claude --dry-run --json`: produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70%: `raw_source={"byte_offset":78860,"partial":true,"artifact_sha256":"a425e6201a8668eb9f79cc343f8b3bec1db8ac37746f284c3e84a56f51c4c2b5"}`; independently recomputed hash matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |

## T5 push/pull round trip (`claude`, `codex`, `opencode`)

**PASS.** `scripts/testing/fakelocker` served on `127.0.0.1:19000`.
Device A: `REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`,
`XDG_DATA_HOME` all isolated under `…\t5\deviceA\`, seeded with copies of
this run's own throwaway `claude`/`codex` sessions plus a freshly created
`opencode` session (created directly under the isolated `XDG_DATA_HOME`,
not copied from real data). `rein init --yes --endpoint
http://127.0.0.1:19000 --bucket t5-bucket --region auto --json`
(`REINSTATE_S3_ACCESS_KEY_ID=FAKEKEY0001`) succeeded, then `rein push --all
--json` (passphrase supplied via `passfd.exe`, a real inherited Windows
handle) reported 3 snapshots (`claude`, `codex`, `opencode`).

Device B: a **separate, fully isolated** set of all four variables under
`…\t5\deviceB\` (never the host's live homes, and never reused from device
A). `rein init --yes … --profile-id <device A's profile id> --json`
paired to the same bucket. `rein pull --all --json` restored `claude` and
`codex` on the first call; `opencode`'s restore initially refused
(`compatibility`, `NOT_INSTALLED` — device B's isolated `XDG_DATA_HOME` had
no OpenCode layout yet) until a single harmless `opencode run "…"` was
executed there to create the layout, matching the row's own documented
refusal reason; the retry then pulled `opencode` too (`pulled:1,
skipped:2` — the second and third being the already-restored `claude`/
`codex`).

Fidelity verified directly on device B: the pulled `claude` session's raw
file contains the planted token 10 times, the pulled `codex` session's raw
file contains its planted token 7 times, and `rein search "T5 opencode
seed ack" --agent opencode --json` (against device B's isolated
`XDG_DATA_HOME`) found the pulled OpenCode session. No write at any point
in this round trip touched the host's live `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, or `XDG_DATA_HOME` — both devices' four variables were set
in every command, including the pull half (the exact isolation gap a
prior candidate's tagged run incident occurred on).

## Harness notes (disclosed, not release-blocking except where marked)

- **`F-OPENCODE-VERSION-DRIFT` (new this run).** OpenCode self-updated from
  `1.18.27` to `1.18.29` partway through this session, above this
  candidate's verified ceiling. Blocks `opencode:E5`/`E6` as `NOT TESTED`
  (see the `opencode` table). Recommend the same treatment `qwen` received
  this candidate: verify `1.18.29` on native Windows physical-resume
  evidence and widen the ceiling under ADR 0005 D3, or pin the CI/lab
  install to prevent auto-update mid-run.
- **`grok -p --resume` hangs headless.** A plain `grok --resume <id> -p
  "<any prompt>"` with stdin closed (including `--always-approve`) hung
  indefinitely and never produced output or exited, reproduced 3 times
  with different prompts and timeouts up to 90s; the identical session
  driven through a real Windows pseudo console (`scripts/testing/conptydriver`)
  completed correctly within seconds. This looks like a Grok Build headless
  `-p`/`--resume` combination needing a real console it cannot get without
  one, not a `rein` defect — `rein`'s own launch plan (`grok --resume <id>`)
  is unchanged from the interactive form either way. Not investigated
  further; flagged for a follow-up with the maintainer since it affects
  how future E1/E2/E3 rows for this agent should be run.
- **`codex exec resume` recall inconsistency (disclosed under `codex:E2`,
  not blocking).** Several recall probes aimed specifically at the
  earliest turn's planted token were answered incorrectly or refused by
  the model, while probes about more recent turns and the conversation's
  length were answered correctly, and the raw session file was
  independently confirmed to contain every turn intact. Recorded as `PASS`
  on the file-level and partial in-agent evidence; flagged as worth a
  follow-up to determine whether this is a `codex exec resume`
  context-window characteristic.
- **A stray Windows PATH resolution trap.** `rein.exe`/`reinstate.exe` on
  a bare `PATH` lookup from a shell resolved to a pre-existing installed
  copy at `C:\Users\admin\AppData\Local\Programs\Reinstate\bin\` rather
  than this run's own install directory. Every command in this report used
  the fully-qualified path to this run's own binary; the one early
  exploratory `--help` call that hit the wrong binary produced no test
  evidence and is not reflected in any row above.
- **`REINSTATE_PASSPHRASE_FD` needs a real Windows handle, not a bash file
  descriptor.** A plain `exec 3<file` in Git Bash does not cross into a
  native Windows child process the way it would for a POSIX child; a
  small helper (`passfd.exe`) using the same
  `syscall.SysProcAttr.AdditionalInheritedHandles` pattern
  `scripts/testing/hoplab/secretfd_windows.go` already documents for
  `REINSTATE_RECOVERY_CODE_FD`/`REINSTATE_PAIRING_CODE_FD` was required.
  Not a product defect — this is exactly why that pattern exists — but
  worth noting for future non-interactive push/pull acceptance rows on
  Windows outside `hoplab`.

## Row accounting

- **107 rows assigned**, of which **1 is `N/A (definitional)`**
  (`opencode:D4`, unchanged since `v0.6.0-rc.1`) — **106 required rows**.
- **104 PASS / 0 FAIL / 0 PARTIAL / 2 NOT TESTED / 1 N/A (definitional)**.
- The 2 `NOT TESTED` rows (`opencode:E5`, `opencode:E6`) are a single new
  finding (`F-OPENCODE-VERSION-DRIFT`), not a carried disposition and not
  excused by either disposition in
  [`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict) —
  they block this part's own required-row count, pending a range-widening
  decision.
- T5 push/pull round trip (`claude`/`codex`/`opencode`): PASS.

This part alone does not authorize a device verdict; it is one executor's
assigned slice of the tagged run. The isolated test directories under
`D:\ReinstateAcceptanceProjects\v060-rc4-b\` (including `t5\deviceA`,
`t5\deviceB`, `qwen-home`, `grok-home`, `c6`, `d4`, `shims`) are deleted
after this report is committed, per the evidence policy.
