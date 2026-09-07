# v0.6.0-rc.5 tagged-artifact acceptance — Windows, part B (T2–T5 agents)

`PHASE5-DEVICE-REPORT-V1` (partial — executor B's assignment only)

Executor B for the `v0.6.0-rc.5` native Windows tagged-artifact acceptance.
Rows: per-agent `C1`–`C6`, `D1`–`D5`, `E1`–`E6` for `claude`, `codex`,
`opencode`, `grok`, `qwen` (17 rows each) and `C1`–`C6`, `D1`–`D5` for
`gemini`, `kimi` (11 rows each) — **107 rows total** (`opencode:D4` is
`N/A (definitional)`, **106 required**), plus the T5 push/pull round trip
for `claude`/`codex`/`opencode`. Dispatch:
[`docs/testing/v0.6.0-rc.5-agent-verification-prompts.md`](../v0.6.0-rc.5-agent-verification-prompts.md).
Contract: [`phase-5-universal-agent-coverage-acceptance.md`](../phase-5-universal-agent-coverage-acceptance.md).

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.5` |
| Full commit | `0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9` |
| Windows archive | `reinstate_0.6.0-rc.5_windows_amd64.zip` |
| Archive SHA-256 | `a648f1de65c15bea7926dda2dd0ce48110d512c9bc60a23b5988c3b24ded9f09` (matches `checksums.txt`; re-verified by this executor before install) |
| `rein version --json` | `{"commit":"0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9","date":"2026-09-07T12:59:29Z","name":"reinstate","version":"0.6.0-rc.5"}` — identical for `reinstate version --json` |
| `rein.exe`/`reinstate.exe` bytes | Identical SHA-256 `aa74899f68356ea5127fa8129db56729c137276d255f1d8890fa9dcf276dd35f` for both |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc5-b\install\` (own fresh directory, never a user binary) |
| Bootstrap deviation | This executor did **not** install from the live `reinstate.dev/install.ps1` bootstrap. Per the dispatch, only executor A installs from the live bootstrap and records that as the artifact identity; this executor installed from the coordinator-verified, checksum-matched draft directory (`…\scratchpad\rc5-draft\reinstate_0.6.0-rc.5_windows_amd64.zip`), the same archive every other executor used. |
| Signed tag / release | Not re-verified by this executor; per the coordinator's assignment, the tag's signature against `.github/allowed_signers`, `gh attestation verify`, and `scripts/verify-release.ps1`/`scripts/test-install.ps1` were already confirmed against this exact draft directory before this executor started. `gh release view v0.6.0-rc.5 --json tagName,isPrerelease,isDraft,publishedAt` (read-only) independently confirms `isDraft:false`, `isPrerelease:true`, tag `v0.6.0-rc.5`. |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Microsoft Windows 11 Pro, `10.0.26200` (Build 26200), x64 |
| Toolchain | Go `1.26.1 windows/amd64` (meets the `1.25.13+` floor) |
| Date (UTC) | 2026-09-07 |
| `REINSTATE_BACKEND` / `REINSTATE_MEMORY_BACKEND_DIR` | Unset explicitly at the start of this run, per the ground rule. This host's shell profile re-sets both on every fresh shell regardless (a fixture unrelated to this candidate), and this executor did not re-unset them before every individual command — only `init`/`push`/`pull` read `REINSTATE_BACKEND` at all (it selects the sync backend; `sessions`/`search`/`inspect`/`resume`/`fork`/`handoff`/`doctor` never consult it, so the C/D/E-row evidence above is unaffected either way). The one place this mattered was the first T5 push/pull attempt, which silently used the contaminated value — see `F-T5-BACKEND-CONTAMINATION` below. Both variables were explicitly unset immediately before every `init`/`push`/`pull` command from that point on, including the corrected T5 round trip. |
| Live agent homes | `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were **never unset**; every isolation used in this report pointed the vendor's own home variable (or, for T5, all three plus `REINSTATE_HOME`) at a directory under `D:\ReinstateAcceptanceProjects\v060-rc5-b\`, per the ground rules |
| Reinstate profile | `REINSTATE_HOME=D:\ReinstateAcceptanceProjects\v060-rc5-b\home` (own directory; rebuilt index, never a stale replay) |

## Vendor versions re-checked immediately before this run

| Agent | Version | Verified range this candidate | In range |
| ----- | ------- | ------------------------------ | -------- |
| `claude` | `2.1.263 (Claude Code)` | `2.1.219`–`2.1.263` | YES |
| `codex` | `codex-cli 0.149.0` | `0.133.0`–`0.149.0` | YES |
| `opencode` | `1.18.29` | `1.18.21`–`1.18.29` (widened this candidate) | YES |
| `grok` | `grok 1.0.5 (5115b46bc9) [stable]` | `1.0.5`–`1.0.5` | YES |
| `qwen` | `0.21.12` | `0.21.12`–`0.23.0` | YES |
| `gemini` | `0.53.0` | n/a (T2, no version gate row in this matrix) | — |
| `kimi` | `0.36.1` | n/a (T2) | — |

Every `qwen` row below ran in an isolated `QWEN_HOME` seeded only with
`settings.json` copied from the host's real `~/.qwen`, confirmed
`agent.version.status=match` before the row's own mechanism was attempted.
Every `grok` row ran in an isolated `GROK_HOME` seeded only with `auth.json`
copied from the host's real `~/.grok`, stdin closed.

## Method summary

- **C1–C4, C6**: `rein sessions`/`search`/`inspect` read-only. `claude`/
  `codex`/`opencode`/`gemini`/`kimi` used the host's live default roots
  (`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` left at their live
  values; default `~/.gemini`/`~/.kimi-code` for the T2 pair). `grok`/`qwen`
  used the isolated `GROK_HOME`/`QWEN_HOME` above throughout.
- **C3, every agent**: a fresh throwaway session was created via each
  vendor's own non-interactive/print mode (`claude -p`, `codex exec`,
  `opencode run`, `gemini -p`, `kimi -p`, `qwen -p`, `grok -p`) in one of two
  throwaway git repositories under
  `D:\ReinstateAcceptanceProjects\v060-rc5-b\projects\`, each carrying a
  planted token (`RC5B-<AGENT>-<epoch>-<suffix>`) in the message body of the
  prompt — never the title. `rein search <token> --agent <key> --json` found
  each one.
- **C5**: for `claude`/`codex`/`opencode`/`grok`/`qwen` (T3+), confirmed
  `capabilities.resume=true`/`fork=true` on the listed session plus the
  version-gate evidence under E4. For `gemini`/`kimi` (T2), confirmed
  `can_resume:false` with a `read_only_reason` and
  `rein resume --dry-run --json` exiting `5`.
- **C6**: for `claude`/`codex`/`gemini`/`kimi`/`qwen`/`grok`, a copy of this
  run's own throwaway single-session data, truncated to 70% of its byte
  length (mid-record, confirmed by inspecting the raw tail), plus an empty
  root and an absent root, all tested via `rein sessions --agent <key>
  --json` against the isolated root. For `opencode`, the repo's committed
  `testdata/adapters/opencode/windows/store.sql` imported into a fresh
  `opencode.db` via `sqlite3`, then truncated to 50%.
- **D1–D5**: `rein handoff <key>:<id> --to <dest> --dry-run --json` (D1, D2,
  D3, D5) and `--no-launch --json` (D4, to reach the on-disk `capsule.json`'s
  `raw_source` block, since `--dry-run` carries no `raw_source` boundary)
  from the session's own workspace directory. D4 for every agent used an
  isolated copy of the agent's real root directory structure containing only
  the one truncated session (plus, for `gemini`, the non-secret
  `projects.json` project-name registry copied alongside it — its absence
  degraded the workspace/project resolution, not the truncation-boundary
  mechanism itself being verified).
- **E1/E2**: the `--dry-run` launch plan's exact `executable`/`args`, run for
  real with the vendor's own non-interactive completion flag appended,
  asking the resumed agent to recall the C3 planted token.
- **E3**: the `--dry-run` fork plan's exact argv, run for real; confirmed a
  new, distinct session id appeared.
- **E4**: a same-named `.exe` shim (a small Go binary built with
  `-ldflags -X main.Version=… -X main.Format=…`, one build below the
  verified minimum, then one build above the verified maximum) placed ahead
  of the real executable on `PATH`; `rein resume --dry-run --json` in a
  fresh, disposable `REINSTATE_HOME` each time. See the harness notes below
  for why an env-var-driven shim (the first attempt) does not work against
  this preflight path.
- **E5/E6**: per the dispatch, `codex`/`opencode`/`grok` used
  `scripts/testing/conptydriver` (built from this worktree, launched only
  via PowerShell `Start-Process -FilePath conptydriver.exe -ArgumentList …
  -WindowStyle Hidden -Wait`, per the console rule) to hold a genuinely
  active vendor process open in a real pseudo console, then
  `rein resume <key>:<id> --dry-run --json` from a second shell confirmed
  `agent.active`, and a non-interactive `rein resume <key>:<id> < /dev/null`
  confirmed exit `7`. `claude`/`qwen` used the same ConPTY method for
  consistency and reliability.
- **T5 push/pull**: a disposable `scripts/testing/fakelocker` instance
  (built from this worktree) served on `127.0.0.1:19100`; two fully isolated
  `REINSTATE_HOME`s (device A/B), each with its own isolated
  `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` under
  `D:\ReinstateAcceptanceProjects\v060-rc5-b\t5\`, never the host's live
  homes on either device. `rein init --yes --endpoint … --profile-id …`
  paired both devices to the same fake bucket; the non-interactive
  passphrase was supplied through a real inherited Windows handle
  (`passfd.exe`, a small helper built for this run following the same
  `AdditionalInheritedHandles` pattern `scripts/testing/hoplab`'s own
  `secretfd_windows.go` uses for its recovery-code FD).

No transcript text, prompt, or session id from any session this run did not
create appears below. Every quoted token is this run's own synthetic,
non-sensitive planted string.

## Row tables

Legend: PASS / FAIL / PARTIAL / NOT TESTED / N/A (definitional).

### `claude` (T5) — 17/17 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent claude --json`: 66 sessions across 63 distinct projects |
| C2 | PASS | `rein inspect claude:<id> --json`: `project="proj1"`, `branch="prod"` (`recorded_environment.branch.provenance="claude.event.gitBranch"`), `message_count` tracked correctly (2 → 4 after a second real turn was added, same session id) |
| C3 | PASS | Planted token `RC5B-CLAUDE-<epoch>-x9k` in a `claude -p` message body; `rein search <token> --agent claude --json` found the session |
| C4 | PASS | `rein inspect claude:<id> --json`: bounded `prompt_preview` (the planted prompt text only), no transcript body |
| C5 | PASS | `capabilities.resume=true, fork=true` correctly declared for T5; version gate independently confirmed under E4 |
| C6 | PASS | This run's own throwaway session copy truncated to 70% (mid-record, confirmed by tailing raw bytes): `rein sessions --agent claude --json` (`CLAUDE_CONFIG_DIR` redirected) exit `0`, session still listed with `incomplete_trailing_record`, no crash; empty root and absent root both exit `0` with an empty session list |
| D1 | PASS | `rein handoff claude:<id> --to codex --dry-run --json` (from the session's own workspace): produced `fidelity.components`, `parse`, `capabilities` |
| D2 | PASS | 15 fidelity components; every `portability` value one of `exact`/`normalized`/`referenced`/`omitted` (never invented) |
| D3 | PASS | 1 unrecognized record reported `{"reason":"unrecognized_record_type"}`, `portability:"referenced"` |
| D4 | PASS | `--no-launch --json` against an isolated copy truncated to 70%: `capsule.json`'s `raw_source={"byte_offset":87467,"partial":true,"artifact_sha256":"ecf85131828ee58760251acb77e5c6039470dc84c27b1141add4bc873b9a35c2"}`; independently recomputed `sha256(head -c 87467 <copy>)` matched exactly |
| D5 | PASS | Two consecutive `--dry-run --json` runs over the unchanged source, unchanged `REINSTATE_HOME`: byte-identical output (`diff` empty) |
| E1 | PASS | Plan argv `claude --resume <id>`; run as `claude --resume <id> -p "<recall prompt>" --output-format json`: `result` field returned exactly `RC5B-CLAUDE-<epoch>-x9k`, the planted token, nowhere else in the prompt |
| E2 | PASS | Same evidence as E1 — the recalled token proves the resumed session is the requested one |
| E3 | PASS | Plan argv `claude --resume <id> --fork-session`; run for real with `-p "Reply with exactly: fork check ack"`: new distinct `session_id` returned (`e6dea144-13c6-4936-9181-9a45ae25ecdc`, ≠ source id), reply correct |
| E4 | PASS | Shim `2.1.218` (below `2.1.219`): exit `5`, `"native agent version 2.1.218 is outside the verified range 2.1.219 to 2.1.263 inclusive"`. Shim `2.1.264` (above `2.1.263`): exit `5`, same range named |
| E5 | PASS | A real `claude --resume <id>` held open under `scripts/testing/conptydriver` (via `cmd.exe /c claude --resume <id>`); `rein resume claude:<id> --dry-run --json` from a second shell: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume claude:<id> < /dev/null`: exit `7`, `"environment warnings require confirmation: agent.active, …"` |

### `codex` (T5) — 17/17 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent codex --json`: 100 sessions (limit) across 20 distinct projects |
| C2 | PASS | `rein inspect codex:<id> --json`: `project="proj2"`, `branch="prod"`, `recorded_environment.git_head` matches the real workspace HEAD |
| C3 | PASS | Planted token `RC5B-CODEX-<epoch>-p3z` via `codex exec "…"`; found by `rein search` |
| C4 | PASS | Bounded `prompt_preview` (system-injected preamble plus the prompt, 160 chars), no full transcript body |
| C5 | PASS | `capabilities.resume=true, fork=true` |
| C6 | PASS | Isolated copy of this run's own rollout file truncated to 70%: exit `0`, session still listed with `incomplete_trailing_record`; empty/absent root both exit `0`, empty list |
| D1 | PASS | `rein handoff codex:<id> --to claude --dry-run --json`: 16 fidelity components produced |
| D2 | PASS | Portability values all in the closed set |
| D3 | PASS | 1 unrecognized record, `referenced`/`unrecognized_record_type` |
| D4 | PASS | `--no-launch --json` against an isolated copy truncated to 70%: `raw_source={"byte_offset":71473,"partial":true,"artifact_sha256":"3ffdbd28915b75b6157734f8310d4b65879da0c51c54504aa4ae785d2ce03ebd"}`; independently recomputed hash at that exact offset matched |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |
| E1 | PASS | Plan argv `codex resume <id>`; run headless as `codex exec resume <id> "<prompt>"`: the on-disk session file genuinely grew (104,266 → 109,241 bytes), kept the same session id, and the reply contained the exact planted token (`grep -c` found it 2× in the reply output, 8× total in the raw file after resuming) |
| E2 | PASS | Same file-level and in-reply proof as E1 — real continuation of the requested session, not a fresh one |
| E3 | PASS | Plan argv `codex fork <id>`; run as `codex exec fork <id> "Reply with exactly: fork check ack"`: new session id `01a07c54-f355-7dc1-aff3-ee038a80fa40` (≠ source `01a07c52-06dd-7f82-be88-eb3b241dc8b9`), reply correct |
| E4 | PASS | Shim `0.132.0` (below `0.133.0`): exit `5`, range named `0.133.0 to 0.149.0`. Shim `0.149.1` (above `0.149.0`): exit `5`, same range |
| E5 | PASS | `codex resume <id>` held open under `scripts/testing/conptydriver` (`cmd.exe /c codex resume <id>`, the dispatch's required method for this row); `rein resume codex:<id> --dry-run --json`: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume codex:<id> < /dev/null`: exit `7` |

### `opencode` (T5) — 16/16 PASS (required), 1 N/A (definitional)

This candidate's specific re-test target: `opencode` self-updated to `1.18.29`,
inside the widened `1.18.21`–`1.18.29` range. Every row ran for real against
that installed version.

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent opencode --json`: 11 sessions across 7 distinct projects at test start |
| C2 | PASS | `project="proj1"`, `message_count` tracked correctly |
| C3 | PASS | Planted token `RC5B-OPENCODE-<epoch>-q7m` via `opencode run "…"` in the message body; `rein search <token> --agent opencode --json` found the session (the auto-generated title also happened to include the token, since OpenCode auto-titles from the first message — the proof is the message-body plant, not the title) |
| C4 | PASS | Bounded `prompt_preview` (the planted prompt text only), no body |
| C5 | PASS | `capabilities.resume=true, fork=true` |
| C6 | PASS | `testdata/adapters/opencode/windows/store.sql` imported into a fresh `opencode.db`, then that `.db` copy truncated to 50%: `rein sessions --agent opencode --json` (`XDG_DATA_HOME` redirected) exit `0` with a `session_read_failed` warning and zero listed sessions (the truncated SQLite file is corrupt at the container level, unlike a JSONL truncation, so no session degrades to partial — this is the same disposition every prior candidate recorded for this row); empty/absent root both exit `0` |
| D1 | PASS | `rein handoff opencode:<id> --to claude --dry-run --json`: 14 fidelity components produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | 1 unrecognized record, `referenced`/`unrecognized_record_type` |
| D4 | **N/A (definitional)** | OpenCode's SQLite-only store has no JSONL record boundary this row's mechanism applies to — unchanged since `v0.6.0-rc.1`. Excluded from the required-row count, per the dispatch and contract's disposition rule |
| D5 | PASS | Two `--dry-run --json` runs, including `handoff_id`/`lineage_root`/`destination.session_id`, byte-identical |
| E1 | PASS | Plan argv `opencode --session <id>`; run headless as `opencode run --session <id> "<recall prompt>"`: reply was exactly `RC5B-OPENCODE-<epoch>-q7m` |
| E2 | PASS | Same evidence as E1 |
| E3 | PASS | Plan argv `opencode --session <id> --fork`; run as `opencode run --session <id> --fork "Reply with exactly: fork check ack"`: reply correct, `rein sessions --agent opencode --json` showed a new distinct id (`ses_f83a818b4ffeKB85HabJlHA134`, title suffixed `(fork #1)`) |
| E4 | PASS | Shim `1.18.20` (below `1.18.21`): exit `5`, range named `1.18.21 to 1.18.29`. Shim `1.18.30` (above the widened `1.18.29`): exit `5`, same range |
| **E5** | **PASS — this candidate's re-test target** | A real `opencode.exe --session <id>` (the executable launched directly, not via a shell wrapper) held open under `scripts/testing/conptydriver`; `rein resume opencode:<id> --dry-run --json` from a second shell: `agent.version` `status=match, actual="1.18.29"` (confirms the widened ceiling now clears the version gate) **and** `agent.active` `status=present, actual=true` — the row's own mechanism (active-session detection), not just the version-gate refusal that blocked this row at `v0.6.0-rc.4` |
| **E6** | **PASS — this candidate's re-test target** | `rein resume opencode:<id> < /dev/null`: exit `7`, `"environment warnings require confirmation: agent.active, …"` — the non-interactive safety check, reached because the version gate no longer preempts it |

### `grok` (T4) — 14/17 PASS, 3/17 PARTIAL (see harness note)

Every row ran against the real installed `1.0.5` in an isolated `GROK_HOME`
seeded only with `auth.json` copied from the host's real `~/.grok`, stdin
closed throughout.

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent grok --json`: 3 sessions in the isolated home (this run's own), 1 distinct project |
| C2 | PASS | `project="proj1"`, `message_count=9` tracked correctly |
| C3 | PASS | Planted token `RC5B-GROK-<epoch>-t8h` via `grok -p "…"`; found by `rein search` (see harness note — this row completed after an unexpectedly long wait) |
| C4 | PASS | Bounded `prompt_preview` (system preamble plus prompt), no full transcript body |
| C5 | PASS | `capabilities.resume=true, fork=true` |
| C6 | PASS | This run's own single-session directory (copy, not the host's real multi-session tree) with `updates.jsonl` truncated to 70%: exit `0`, no crash; empty/absent `GROK_HOME` both exit `0` |
| D1 | PASS | `rein handoff grok:<id> --to claude --dry-run --json`: 16 fidelity components produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against a copy: `raw_source` targeted `updates.jsonl` (matching the prior candidate's own finding that this, not `chat_history.jsonl`, is the boundary file), `{"byte_offset":1273,"partial":true,"artifact_sha256":"a95b8433664cb2528d07eb32c556bfdb6964da9a78db376c0f66ccce8bcecd39"}`; independently recomputed `sha256(head -c 1273 updates.jsonl)` matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |
| E1 | **PARTIAL** | Plan argv `grok --resume <id>`; the launch itself is confirmed correct (the vendor process starts, attaches to the requested session, and replays its real prior history — the original planted-token prompt reappeared verbatim on load, proving the recorded session, not a fresh one, was opened). The recall half of the row's mechanism — asking the resumed agent a new question and reading back its answer — was attempted three times (headless `-p`, twice under a real ConPTY console) and never completed: two attempts were abandoned after 25+ minutes and ~60s with zero output, and a third, run to its full budget, still showed `Waiting for response… 7m59s` with an on-screen `Connection` error indicator when the driver's window closed. See `F-GROK-BACKEND-CONNECTIVITY` below. Recorded `PARTIAL`, not `PASS`: the launch mechanism is proven, the recall mechanism is not, and nothing was fabricated or extrapolated from the C3 evidence (which used a *different* invocation — a fresh session, not a resume) |
| E2 | **PARTIAL** | Same basis as E1 — the resumed session is confirmably the requested one (history replay proves it), but the specific in-agent recall confirmation could not be completed within this run's time budget |
| E3 | **PARTIAL** | Plan argv `grok --resume <id> --fork-session`; not attempted live given E1/E2's outcome on the identical connectivity path (a fork-and-recall exchange would need the same completion request that failed to complete three times above). Fork's distinct-session-creation half is independently supported by an unrelated `grok --resume <other-id>` ConPTY session in this same isolated home producing a new, persisted session id, but that is not this row's own token-carrying session, so this row is not scored on it |
| E4 | PASS | Shim `1.0.4` (below `1.0.5`): exit `5`, range named `1.0.5 to 1.0.5`. Shim `1.0.6` (above `1.0.5`): exit `5`, same range |
| E5 | PASS | A genuinely active `grok --resume <id>` held open under `scripts/testing/conptydriver` (`grok.exe` launched directly); `rein resume grok:<id> --dry-run --json` from a second shell: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume grok:<id> < /dev/null`: exit `7` |

### `qwen` (T4) — 17/17 PASS

Every row below ran against the real installed `0.21.12` in an isolated
`QWEN_HOME` seeded only with `settings.json` copied from the host's real
`~/.qwen`, confirmed `agent.version.status=match` before the row's own
mechanism was attempted.

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent qwen --json`: 1 session in the isolated home (this run's own) |
| C2 | PASS | `project="proj1"`, `message_count=2` tracked correctly |
| C3 | PASS | Planted token `RC5B-QWEN-<epoch>-y2k` via `qwen -p "…"`; found by search |
| C4 | PASS | Bounded `prompt_preview` (the planted prompt text only), no body |
| C5 | PASS | `capabilities.resume=true, fork=true`, `agent.version.status=match` (`0.21.12` in `0.21.12`–`0.23.0`) |
| C6 | PASS | This run's own single-session `.jsonl` (copy) truncated to 70%: exit `0`, no crash; empty/absent `QWEN_HOME` both exit `0` |
| D1 | PASS | `rein handoff qwen:<id> --to claude --dry-run --json`: 15 fidelity components produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70%: `raw_source={"byte_offset":2428,"partial":true,"artifact_sha256":"db8c909413e6d14cb61c2e157807923054cb353aa0a2674ec4c7deb0ee19840d"}`; independently recomputed hash matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |
| E1 | PASS | Plan argv `qwen --resume <id>`; run as `qwen --resume <id> -p "<recall prompt>"`: reply was exactly `RC5B-QWEN-<epoch>-y2k` |
| E2 | PASS | Same evidence as E1 |
| E3 | PASS | Plan argv `qwen --resume <id> --fork-session`; run as `qwen --resume <id> --fork-session -p "Reply with exactly: fork check ack"`: reply correct, `rein sessions --agent qwen --json` showed a new distinct id (`7a7b3d1b-c402-433d-94d6-c25ba08f718a`) |
| E4 | PASS | Shim `0.21.11` (below `0.21.12`): exit `5`, range named `0.21.12 to 0.23.0`. Shim `0.23.1` (above `0.23.0`): exit `5`, same range |
| E5 | PASS | A real `qwen --resume <id>` held open under `scripts/testing/conptydriver` (`cmd.exe /c qwen --resume <id>`); `rein resume qwen:<id> --dry-run --json`: `agent.active` `status=present, actual=true` |
| E6 | PASS | `rein resume qwen:<id> < /dev/null`: exit `7` |

### `gemini` (T2) — 11/11 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent gemini --json`: 28 sessions across 17 distinct projects |
| C2 | PASS | `project="proj1"`, `message_count=2` tracked correctly |
| C3 | PASS | Planted token `RC5B-GEMINI-<epoch>-f6d` via `gemini -p "…"` (host's live default `~/.gemini`); found by search |
| C4 | PASS | Bounded `prompt_preview` (the planted prompt text only), no body |
| C5 | PASS | `can_resume=false`, `read_only_reason="Gemini CLI sessions are read-only in Phase 2"`; `rein resume gemini:<id> --dry-run --json`: exit `5`, `"native session action is unsupported: …"` |
| C6 | PASS | This run's own single-session `.jsonl` (copy) truncated to 70%: exit `0`, session still listed with `incomplete_trailing_record`; empty/absent `GEMINI_CLI_HOME` both exit `0` |
| D1 | PASS | `rein handoff gemini:<id> --to claude --dry-run --json`: 14 fidelity components produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against an isolated copy (the truncated session file plus a copy of the host's non-secret `projects.json` project-name registry, required for gemini's adapter to resolve the workspace/project id — without it the row's own preflight blocked on an unrelated workspace check before reaching the truncation boundary, a harness discovery not a row defect) truncated to 70%: `raw_source={"byte_offset":228,"partial":true,"artifact_sha256":"6117bd635d549898b3baed80c427a6b64d9ebcafe64a8b34a16f93b37a06491f"}`; independently recomputed hash matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |

### `kimi` (T2) — 11/11 PASS

| Row | Result | Evidence |
| --- | ------ | -------- |
| C1 | PASS | `rein sessions --agent kimi --json`: 5 sessions across 4 distinct projects |
| C2 | PASS | `project="proj1"`, `message_count=2` tracked correctly |
| C3 | PASS | Planted token `RC5B-KIMI-<epoch>-z3v` via `kimi -p "…"`; found by search |
| C4 | PASS | Bounded `prompt_preview` (the planted prompt text only), no body |
| C5 | PASS | `can_resume=false`, `read_only_reason="Kimi Code CLI sessions are read-only until a device journey verifies native resume"`; `rein resume kimi:<id> --dry-run --json`: exit `5` |
| C6 | PASS | This run's own single-session directory (copy) with `agents/main/wire.jsonl` truncated to 70%: exit `0`, no crash; empty/absent `KIMI_CODE_HOME` both exit `0` |
| D1 | PASS | `rein handoff kimi:<id> --to claude --dry-run --json`: 15 fidelity components produced |
| D2 | PASS | Portability values in the closed set |
| D3 | PASS | No unknown records this session |
| D4 | PASS | `--no-launch --json` against a copy truncated to 70%: `raw_source={"byte_offset":78891,"partial":true,"artifact_sha256":"646a23450e21e06a822259ecd9bf3e69fa7c101c599e12c057809e36f5cb7965"}`; independently recomputed hash matched exactly |
| D5 | PASS | Two `--dry-run --json` runs byte-identical |

## T5 push/pull round trip (`claude`, `codex`, `opencode`)

**PASS**, on the second, corrected attempt — see `F-T5-BACKEND-CONTAMINATION`
below for the first attempt's invalidation. `scripts/testing/fakelocker`
served on `127.0.0.1:19100`. Device A: `REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, `XDG_DATA_HOME` all isolated under `…\t5\deviceA\`, seeded with
copies of this run's own throwaway `claude`/`codex` sessions plus a freshly
created `opencode` session (created directly under the isolated
`XDG_DATA_HOME`, not copied from real data). `rein init --yes --endpoint
http://127.0.0.1:19100 --bucket t5-bucket-v2 --region auto --json`
(`REINSTATE_S3_ACCESS_KEY_ID=FAKEKEY0001`, `REINSTATE_BACKEND`/
`REINSTATE_MEMORY_BACKEND_DIR` explicitly unset) succeeded, then
`rein push --all --json` (passphrase supplied via `passfd.exe`, a real
inherited Windows handle) reported 4 snapshots (`claude` ×2, `codex`,
`opencode`) — confirmed by the fakelocker access log showing real
`PUT`/`GET` traffic against `t5-bucket-v2/profiles/40796752-…/`.

Device B: a **separate, fully isolated** set of all four variables under
`…\t5\deviceB\` (never the host's live homes, and never reused from device
A). `rein init --yes … --profile-id <device A's profile id> --json` paired
to the same bucket. `rein pull --all --json` restored `claude` (×2) and
`codex` on the first call; `opencode`'s restore initially refused
(`compatibility`, `NOT_INSTALLED` — device B's isolated `XDG_DATA_HOME` had
no OpenCode layout yet), and after establishing the layout it refused again
with a `conflict` (`local session diverged`) because the layout-bootstrap
step itself had written unrelated session rows into the same single SQLite
file OpenCode uses for every session — an expected consequence of OpenCode's
single-file store (the same root cause `opencode:D4` is `N/A (definitional)`
for), not a `rein` defect. Resolved by seeding device B's OpenCode layout
from device A's **schema only** (`sqlite3 opencode.db ".schema"` replayed
into a fresh, zero-row database) instead of running a real session there;
the retry then pulled `opencode` cleanly with zero conflicts.

Fidelity verified directly on device B: the pulled `claude` session's raw
file contains the planted token 9 times (the second, credential-less
`claude` session correctly carries none), the pulled `codex` session's raw
file contains its planted token 8 times, and
`rein search "T5 opencode seed ack" --agent opencode --json` (against
device B's isolated `XDG_DATA_HOME`) found the pulled OpenCode session by
its title. No write at any point in this corrected round trip touched the
host's live `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, or `XDG_DATA_HOME`.

## Harness notes (disclosed, not release-blocking except where marked)

- **`F-T5-BACKEND-CONTAMINATION` (new this run, corrected in-run).** This
  executor's shell profile persistently sets `REINSTATE_BACKEND=memory` and
  `REINSTATE_MEMORY_BACKEND_DIR` on every fresh shell (unrelated to this
  candidate; a fixture used by other work on this host). The dispatch's
  "unset in every shell" rule was followed at the start of this run but not
  re-applied in the specific commands that ran the first T5 `init`/`push`/
  `pull` attempt, so that attempt silently routed through the host's shared,
  persistent Hop-lab locker (`D:\Projects\hop-10-lab\locker`) instead of
  this run's own `fakelocker` — confirmed after the fact by the fakelocker's
  own access log showing zero push/pull traffic despite `push`/`pull`
  reporting success. Caught before this report was written. The full T5
  round trip was re-run from a clean profile id with `REINSTATE_BACKEND`/
  `REINSTATE_MEMORY_BACKEND_DIR` explicitly unset in every command, and
  independently confirmed correct this time via real `PUT`/`GET`/`DELETE`
  traffic in the fakelocker's own log (see the T5 section above). The
  mistaken profile's key material was left in the shared locker (a
  destructive cleanup command targeting a directory outside this run's own
  tree was correctly refused by the host's own command-safety layer); it is
  orphaned and encrypted, and does not affect any other profile there.
  Recommend: this dispatch's rule should say "unset in every shell that
  reads `REINSTATE_HOME` **or performs a push/pull**" explicitly, since a
  read-only `rein sessions`/`search`/`inspect` call is unaffected by this
  variable and does not itself reveal the contamination.
- **`F-GROK-BACKEND-CONNECTIVITY` (new this run, disclosed, blocks 3
  required rows as `PARTIAL`, not a `rein` defect).** `grok`'s own
  completion requests were unreliable on this host throughout this run,
  contradicting the dispatch's note that "the maintainer re-logged in on
  2026-09-07 and `grok -p` answers within seconds now":
  - `C3`'s planting call (`grok -p "Reply with exactly this token…"`, a
    fresh session, not a resume) took roughly 10 minutes to produce output,
    eventually succeeding with the exact correct text — confirmed
    completed, not extrapolated (this run observed the literal output).
  - `E1`/`E2`'s resume-and-recall call (`grok --resume <id> -p "<recall
    prompt>"`) ran headless for 25+ minutes with zero output and was
    abandoned (killed) before completing.
  - Two further attempts under a real ConPTY console (`scripts/testing/
    conptydriver`, `grok --resume <id>` with the recall question sent as
    real keystrokes) reached `Waiting for response… 59s` and, on a third,
    generously budgeted attempt, `Waiting for response… 7m59s` with a
    `Connection` error indicator visible in the rendered frame when the
    driver's own time budget expired — genuinely still trying, not frozen
    (the raw byte stream kept growing with spinner-frame updates the whole
    time), but never producing an answer within any window this run could
    practically budget.
  - Confirmed this is not `rein`-specific: a 25-second `grok -p` diagnostic
    against the host's live, non-isolated default `~/.grok` also timed out
    with zero output.

  This looks like a live account/model-endpoint connectivity problem on
  this host today (the visible `Connection` error text, `reqwest error
  stream: error sending request`, points at the vendor CLI's own outbound
  HTTP layer), not a `rein` launch-plan defect — `rein`'s own plan
  (`grok --resume <id>`) is unchanged, correctly reopens the requested
  session (its history replays intact), and correctly detects the process
  as active (`E5`) the entire time. It blocks `E1`, `E2`, and `E3` — the
  three rows whose mechanism specifically requires a completed reply — at
  `PARTIAL`, per the contract's rule that `PARTIAL` does not pass a
  required row. Flagged for the maintainer: this dispatch's claim about
  `grok -p` latency does not hold on this host today, and future `grok`
  rows needing a completion should budget for the possibility of no
  completion at all inside a normal run, not only a slow one.
- **A version shim must not depend on env vars for `resume`/`fork`
  preflight (new finding this run, harness-only).** `internal/agentcheck`'s
  version probe (used by `resume --dry-run`/`fork --dry-run`, unlike
  `doctor --agents` which uses a plain unrestricted `PATH` lookup) resolves
  the vendor executable through `internal/executabletrust.Resolve`, which
  passes a **plain, unmodified `os.Environ()`** to the child process. A
  first attempt at a version shim read `SHIM_VERSION`/`SHIM_FORMAT` from the
  environment and worked correctly under `doctor --agents` but silently
  failed under `resume --dry-run` (`agent.version` reported `status:unknown,
  actual:""` — the shim ran, but with unset env vars, since... on further
  inspection this was not an environment-stripping issue but a Windows
  `PATH` translation trap, below). Fixed by baking the version and vendor
  output format into the shim binary at build time
  (`-ldflags -X main.Version=… -X main.Format=…`) instead. Not a `rein`
  defect — the shim construction was the problem — but worth documenting
  for the next executor who reaches for an env-var-driven shim.
- **Git Bash `PATH` prepend must use `/d/...`-style paths, not
  `D:/...`-style, or Windows children never see the prepended directory
  at all.** `export PATH="D:/some/dir:$PATH"` in Git Bash mistranslates to
  a broken Windows `PATH` for a native child process (confirmed via
  `cmd.exe /c echo %PATH%`, which showed the intended directory silently
  split into `D` and a bogus path under the Git install prefix). Using
  `export PATH="/d/some/dir:$PATH"` (the MSYS-native form) translates
  correctly. This is what actually caused the shim failures above, not the
  environment-passing behavior first suspected — both findings are recorded
  since both cost real time to isolate.
- **`conptydriver` under PowerShell `Start-Process -ArgumentList` needs
  manual quoting for any argument containing spaces.** `Start-Process`
  joins a string-array `-ArgumentList` with plain spaces, not per-element
  quoting, so an unquoted multi-word prompt argument (e.g. `grok -p "Reply
  with exactly …"`) arrives at `conptydriver.exe` already split into
  separate `os.Args` elements, and `windows.ComposeCommandLine` then
  re-quotes each fragment independently — the vendor CLI sees `Reply`,
  `with`, `exactly`, … as separate positional arguments and refuses with a
  usage error. Fix: wrap the argument in its own literal double quotes
  inside the PowerShell array element (e.g. `'"Reply with exactly …"'`) so
  the naive space-join still produces one correctly quoted token.
- **A stray Windows `PATH` resolution trap (same as the prior candidate).**
  `rein.exe`/`reinstate.exe` on a bare `PATH` lookup can resolve to a
  pre-existing installed copy rather than this run's own install directory.
  Every command in this report used the fully-qualified path to this run's
  own binary.
- **`REINSTATE_PASSPHRASE_FD` needs a real Windows handle, not a bash file
  descriptor (same as the prior candidate).** `passfd.exe`, built for this
  run using the same `syscall.SysProcAttr.AdditionalInheritedHandles`
  pattern `scripts/testing/hoplab/secretfd_windows.go` already documents,
  was required for the non-interactive T5 push/pull passphrase.

## Row accounting

- **107 rows assigned**, of which **1 is `N/A (definitional)`**
  (`opencode:D4`, unchanged since `v0.6.0-rc.1`) — **106 required rows**.
- **103 PASS / 0 FAIL / 3 PARTIAL / 0 NOT TESTED / 1 N/A (definitional)**.
- The 3 `PARTIAL` rows are `grok:E1`, `grok:E2`, `grok:E3`
  (`F-GROK-BACKEND-CONNECTIVITY`, disclosed above) — a live vendor-backend
  connectivity problem on this host, confirmed independent of `rein`, not a
  product defect. Per the contract's rule, `PARTIAL` does not pass a
  required row, so this part's own required-row count is **103/106**, not
  106/106, pending either a healthier connectivity window for a re-attempt
  or the coordinator's own judgment on this disclosed, non-product finding.
- T5 push/pull round trip (`claude`/`codex`/`opencode`): PASS, on the
  corrected second attempt (`F-T5-BACKEND-CONTAMINATION`, disclosed above).

This part alone does not authorize a device verdict; it is one executor's
assigned slice of the tagged run. The isolated test directories under
`D:\ReinstateAcceptanceProjects\v060-rc5-b\` (including `t5\deviceA`,
`t5\deviceB`, `isolated\*`, `shims`, `out`) are deleted after this report is
committed, per the evidence policy.
