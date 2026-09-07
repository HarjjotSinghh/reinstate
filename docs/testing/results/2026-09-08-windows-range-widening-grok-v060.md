# Verified-range widening — Grok Build, native Windows x64, 2026-09-08

`AGENT-TIER-JOURNEY-V1` · one agent, single platform, read-only evidence only.

Reproduces the method used to widen the Claude Code, Qwen Code, and OpenCode
ranges
([`2026-09-06-windows-range-widening-v060.md`](2026-09-06-windows-range-widening-v060.md),
[`2026-09-07-windows-range-widening-qwen-v060.md`](2026-09-07-windows-range-widening-qwen-v060.md),
[`2026-09-07-windows-range-widening-opencode-v060.md`](2026-09-07-windows-range-widening-opencode-v060.md)),
under [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md)
D3, for the Grok Build ceiling — with one difference from those three records,
disclosed up front rather than papered over: this record's evidence is
**read-only**. The completed-turn resume/fork journey (`grok:E1`–`E3`) that
those three records each produced from inside this harness cannot be produced
here for Grok, for reasons independent of this change (below).

**macOS evidence: pending (ADR 0005).**
**Completed-turn evidence (`grok:E1`–`E3` against `1.0.13`): deferred to the
`v0.6.0-rc.6` tagged run, to be executed by the maintainer at their own
console and recorded by the executor. See §5.**

## 0. Why this record exists

The acceptance host's Grok Build installation self-updated past the ceiling
the catalog carried: `grok --version` now prints `1.0.13`, while the
in-tree range was `1.0.5`–`1.0.5` (`internal/sessionindex/grok.go`). This
matches the `v0.6.0-rc.5` tagged report's own §23/§24 rechecks, which already
found the same drift and recorded `rein resume grok:<id> --dry-run --json`
refusing with exit `5` (`agent.version`, `native agent version 1.0.13 is
outside the verified range 1.0.5 to 1.0.5 inclusive`) against a real,
`rein`-visible Grok session on this same host — correct, fail-closed
behavior, and the reason this record exists.

Those same rc.5 §23/§24 rechecks also found something orthogonal to the
version ceiling: five to six independent reproductions of
`F-GROK-BACKEND-CONNECTIVITY` — the installed `grok` binary hanging
indefinitely (`exit 124`) against the live xAI backend, on both the `1.0.13`
build and (in one probe) the pinned `1.0.5` build recovered from
`grok.exe.old`. Under this task's ground rules, the host state has moved on
again since that report: the maintainer's own console now confirms `1.0.13`
answers instantly, while the old `1.0.5` binary no longer completes a prompt
against xAI at all, even from the maintainer's own console. Grok cannot be
driven to a completion from this harness (headless or ConPTY) in the current
environment. The maintainer's standing policy (2026-09-07, Q27) is that when
a vendor CLI self-updates past the verified ceiling, Reinstate widens the
range on native Windows evidence rather than block — Reinstate always wants
to support the latest version — so this record widens the ceiling on the
read-only evidence that this environment *can* produce, and defers the
completed-turn rows to the maintainer's console during the `v0.6.0-rc.6`
tagged run, per the ground rules for this task.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-07 (2026-09-08 local, host timezone `+05:30`) |
| Device | `windows-amd64`, native — not WSL |
| OS | Microsoft Windows 11 Pro, 10.0.26200 |
| Worktree | `D:\Projects\reinstate-worktrees\rc6`, branch `v060/rc6` |
| Branch tip at time of this record | `36e496c5fad3c0a71d9b52e936ebc3c57f7fc924` |
| Go toolchain | `go1.25.13` (`GOTOOLCHAIN=go1.25.13`) |
| Grok Build installed | `1.0.13` (`grok 1.0.13 (5e9a58528b76) [stable]`, default `GROK_HOME`) |
| Grok Build verified range before this record | `1.0.5`–`1.0.5` |
| Grok Build verified range after this record | `1.0.5`–`1.0.13` |

No real transcript, session id, project path, or session title is quoted
anywhere below. Every observation about the real `~/.grok` tree is scoped to
file names and JSON key names only, per this task's ground rules; no content
or identifier was read out of any real session, and none is reproduced here.

## 2. Isolation

`GROK_HOME` is Grok Build's only root override
(`internal/agents/catalog/grok.go`'s `RootEnv`). The read-only probes below
were run in two forms:

- `grok --version` and `grok --help`: read-only, stateless vendor commands
  that touch no session store; run directly against the installed binary.
- `rein doctor --agents --json`: run against a fresh, throwaway
  `REINSTATE_HOME` (this task's own `unset REINSTATE_BACKEND
  REINSTATE_MEMORY_BACKEND_DIR` rule applied, `CLAUDE_CONFIG_DIR`/
  `CODEX_HOME`/`XDG_DATA_HOME` left untouched per the ground rules), reading
  the real, live `GROK_HOME` the way `rein doctor` always does — but its own
  output is already a sanitized, aggregated shape probe (glob patterns,
  counts, median byte sizes), never raw file content or identifiers, which is
  why this is the one probe in this record that reads the live tree directly.
- File-name and JSON-key listings under the real `~/.grok/sessions/`: scoped
  to one session directory whose modification date was today
  (2026-09-08 local), per the ground rules' allowance to look at "the session
  file layout and first-line keys of a session created by the maintainer
  today under the real home (look only at file names and JSON key names ...,
  never content, never ids)". No directory name, session id, or file content
  is reproduced below — only the file-name list and the JSON key names found
  in each file's first line(s).

No Grok session was created by this record, and nothing was written under
`~/.grok` by any command run in this record.

## 3. Read-only evidence gathered

| Step | Command / probe | Result |
| ---- | ---------------- | ------ |
| Version output shape | `grok --version` | `grok 1.0.13 (5e9a58528b76) [stable]` — same shape as `1.0.5`: bare `grok <semver>`, a parenthesised hex build id, a bracketed `[stable]` channel, nothing on stderr. Parses under the existing `grokVersionPattern` regex unmodified — no pattern change needed to widen the range. |
| Launch-plan flags, `--resume` | `grok --help` | `-r, --resume [<SESSION_ID_OR_TITLE>]` — identical shape and prose to the `1.0.5` measurement in `docs/session-storage/grok.md`: non-ID values match session titles for the current directory, UUID-shaped values always mean ids, ambiguity errors on duplicate titles. Unchanged. |
| Launch-plan flags, `--fork-session` | `grok --help` | `--fork-session` — "When resuming (`--resume` / `--continue`), create a new session ID instead of reusing the original (optionally set via `--session-id`)." Unchanged in shape from the argv `rein`'s `Fork` template builds (`--resume <uuid> --fork-session`). |
| Launch-plan flags, `--continue` | `grok --help` | `-c, --continue` — "Continue the most recent session for the current working directory." Unchanged. |
| Launch-plan flags, `--session-id` | `grok --help` | `-s, --session-id <SESSION_ID>` — "Use a specific session UUID for a **new** conversation (must be a valid UUID and must not already exist under the target session directory). With `--resume`/`--continue`, only valid together with `--fork-session`... Does not resume existing sessions." Unchanged from the vendor contract `docs/session-storage/grok.md` already documents. |
| Headless single-turn form | `grok --help` | `-p, --single <PROMPT>` — "Single-turn prompt. Prints the response to stdout and exits." Present and unchanged; not exercised for a completion in this record (§0, §5). |
| Session directory layout | file names under one `sessions/<project>/<uuid>/` directory modified today | `announcement_state.json`, `chat_history.jsonl`, `chat_history.jsonl.lock`, `events.jsonl`, `prompt_context.json`, `summary.json`, `summary.json.lock`, `system_prompt.txt`, `title_refresh_idx`, `updates.jsonl`, `updates.jsonl.lock`. The index source's glob (`sessions/**/summary.json`) still matches: `summary.json` is present and in the same place. Four file names not previously catalogued in `docs/session-storage/grok.md` (`announcement_state.json`, `prompt_context.json`, `system_prompt.txt`, `title_refresh_idx`) appeared alongside the documented set; none of them is `summary.json`, `chat_history.jsonl`, or `updates.jsonl`, so none is read by the shipped reader (`internal/sessionindex/grok.go`, `internal/transcript` Grok reader), which only opens those three. A `prompt_history.jsonl` file was also observed one level up, at the per-project directory rather than the per-session directory; it sits outside the session glob and is not consulted. |
| `summary.json` key names | first-line JSON object keys, no values | `id`, `cwd`, `agent_name`, `chat_format_version`, `created_at`, `current_model_id`, `generated_title`, `git_root_dir`, `grok_home`, `info`, `last_active_at`, `next_trace_turn`, `num_chat_messages`, `num_messages`, `reasoning_effort`, `request_id`, `sandbox_profile`, `session_summary`, `updated_at`. `id` and `cwd` — the two fields the documented `Info { id, cwd }` shape names, and the two the reader actually keys off — are both still present, in the same top-level position. The reader tolerates unknown fields (the doc already describes this as "`Info { id, cwd }` + counts/timestamps/model"), so the additional keys observed here are additive, not a shape break. |
| `chat_history.jsonl` first-line keys | first JSONL record, keys only | `type`, `content` — identical to the documented `ConversationItem` shape; unchanged. |
| `updates.jsonl` first-line keys | first JSONL record, keys only | `method`, `params`, `timestamp`, plus `_meta`, `agentTimestampMs`, `elapsed_ms`, `error`, `event_name`, `eventId`, `name`, `runs`, `sessionId`, `sessionUpdate`, `status`, `update` observed across the file's 13 lines. `method`, `params`, and `timestamp` — the three fields `docs/session-storage/grok.md` documents — are present. `docs/session-storage/grok.md` already instructs readers to "treat unknown lines as opaque," so the additional envelope keys are within the documented contract, not a break of it. |
| `rein doctor --agents --json`, Grok row | isolated `REINSTATE_HOME`, real `GROK_HOME` | `"key":"grok"`, `"executable_on_path":true`, `"version_raw":"grok 1.0.13 (5e9a58528b76) [stable]"`, `"resolved_root":{"relative_to":"home","suffix":".grok"}`; the sanitized `tree` block reports glob-level shape (`sessions`, `sessions/*`, `sessions/*/*`, `sessions/*/*/*` — dir/file counts and median byte sizes only) consistent with the layout above. No id, path, or file content appears in this output — it is Reinstate's own sanitized shape probe. |

## 4. What changed between `1.0.5` and `1.0.13`, and what did not

Unchanged: the `grok --version` output shape (bare `grok <semver>`,
parenthesised hex build id, bracketed channel — the same regex the catalog
has parsed since the T3 promotion, per `internal/agents/catalog/grok.go`'s
own comment about the `[stable]` suffix fix); every launch-plan-relevant
`--help` flag `rein`'s native argv depends on (`--resume`, `--fork-session`,
`--continue`, `--session-id`); the session directory layout at the level the
index source globs (`sessions/<project>/<uuid>/summary.json`); the field
names the reader actually consumes in `summary.json` (`id`, `cwd`),
`chat_history.jsonl` (`type`, `content`), and `updates.jsonl` (`method`,
`params`, `timestamp`).

Newly observed, not newly broken: four additional per-session file names
(`announcement_state.json`, `prompt_context.json`, `system_prompt.txt`,
`title_refresh_idx`) and one additional per-project file
(`prompt_history.jsonl`), none of which the shipped reader opens; and a wider
set of top-level keys in `summary.json` and per-line keys in `updates.jsonl`
than `docs/session-storage/grok.md`'s prose enumerates, all additive to
fields the reader already tolerates as unknown. No prior device journey
recorded these file/key names at this level of detail for a session created
under `1.0.13` specifically, since no `1.0.5` build remained installed
alongside `1.0.13` on this host to diff against (the host had already
self-updated past `1.0.5` before this record began, matching the same
constraint the `v0.6.0-rc.5` §23/§24 rechecks hit).

## 5. Completed-turn evidence — deferred to the `v0.6.0-rc.6` tagged run

`grok:E1` (resume), `grok:E2` (fork), and `grok:E3` (recall) each require a
real, completed conversational turn against the live `grok` binary: a session
is created, a token is planted in the first turn, the session is resumed or
forked through `rein`'s own launch plan, and the resumed agent must complete
a real turn that recalls the token. That is exactly the step the
`v0.6.0-rc.5` tagged report's own §23 and §24 rechecks document failing
twice each, independent of `rein`, from `F-GROK-BACKEND-CONNECTIVITY`: the
vendor CLI itself never returned a completion, on either the `1.0.13` build
or the recovered `1.0.5` pinned binary, across six attempts total on that
device.

Per this task's ground rules, the host state has moved again since that
report: the old `1.0.5` binary no longer completes a prompt against xAI at
all, even from the maintainer's own console, while `1.0.13` answers
instantly from the maintainer's own console. Grok cannot be driven to a
completion from this harness (headless or ConPTY) in the current
environment — the same connectivity-shaped obstacle the rc.5 rechecks
documented, not a defect in this record's method. Read-only evidence (§3)
is everything this environment can produce today.

`grok:E1`, `grok:E2`, and `grok:E3` against `1.0.13` will therefore be
executed by the maintainer at their own console during the `v0.6.0-rc.6`
tagged run, and recorded by the executor at that time — the same division of
labor this task's ground rules specify. This record does not claim those
rows as collected.

## 6. Range declaration moved

| Location | Old | New |
| -------- | --- | --- |
| `internal/sessionindex/grok.go` (`GrokMaxVerifiedVersion`) | `1.0.5` | `1.0.13` |
| `internal/agents/catalog/grok.go` (`VersionSpec.Max`, sourced from the constant above) | `1.0.5` | `1.0.13` |

`internal/handoff/target_grok.go`'s compatibility-gate error message formats
`sessionindex.GrokMinVerifiedVersion`/`GrokMaxVerifiedVersion` directly, so it
moves with the constant and needed no edit of its own.

Tests moved with the ceiling:

- `internal/agents/catalog/grok_version_test.go`
  `TestGrokVersionParsesShippedOutput` — adds `1.0.13` cases (with build id,
  without build id, matching the same shapes already covered for `1.0.5`).
- `internal/agents/catalog/catalog_test.go`
  `TestShippedAgentsRegisterAtDeclaredTiers` — the Grok row's expected max
  moves to `1.0.13`.

`internal/agentcheck/agent.go`'s `testFallbackDefinitions()` has no Grok
entry (only `claude` and `codex`), so there is nothing to move there, the
same as the OpenCode and Qwen widening records found for their own agents.

Fail-closed boundary: no build between `1.0.6` and `1.0.12` inclusive, and no
build past `1.0.13`, was installed on this host or physically measured, so
none of them gained a literal test case — the same allowance the prior
widening records in this series used for their own untested interior and
one-past-ceiling versions.

## 7. Gates

| Gate | Command | Result |
| ---- | ------- | ------ |
| Format | `gofmt -l internal/sessionindex/grok.go internal/agents/catalog/grok.go internal/agents/catalog/grok_version_test.go internal/agents/catalog/catalog_test.go` | see the run recorded with this commit |
| Vet (owned packages) | `GOTOOLCHAIN=go1.25.13 go vet ./internal/sessionindex/... ./internal/agents/catalog/... ./internal/handoff/...` | see the run recorded with this commit |
| Vet (whole tree) | `GOTOOLCHAIN=go1.25.13 go vet ./...` | see the run recorded with this commit |
| Tidy | `GOTOOLCHAIN=go1.25.13 go mod tidy -diff` | see the run recorded with this commit |
| Unit suite | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test -p 4 ./... -count=1` | see the run recorded with this commit |
| Lint | `make lint` | see the run recorded with this commit |
| Doc gate | `go test ./internal/doctest/... -count=1` | see the run recorded with this commit |
| Doc-link script | `bash scripts/check-docs.sh` | see the run recorded with this commit |
| Website unit tests | `cd website && npm ci && npx vitest run` | see the run recorded with this commit |
| Acceptance-matrix row count | `rein doctor --agents --acceptance-matrix --json` (`"row_count"`) | still `178` — widening `Min`/`Max` on an already-registered agent changes no tier, family, or agent count, so the generated matrix's row count is unaffected |

## 8. Verdict

The verified range is widened to `1.0.5`–`1.0.13` on **read-only** native
Windows evidence only, under the maintainer's standing self-update policy
(2026-09-07, Q27). Version-output parsing, every launch-plan-relevant
`--help` flag, and the reader-consumed fields of the session file layout are
confirmed unchanged from `1.0.5`. Apple Silicon macOS evidence is deferred
(ADR 0005 D3, tracked in the same `#403` every other `v0.6.0` native-Windows-
only widening in this release cites). The completed-turn `grok:E1`–`E3` rows
against `1.0.13` are explicitly **not** collected by this record — they are
deferred to the `v0.6.0-rc.6` tagged run, to be executed by the maintainer at
their own console (where `1.0.13` is confirmed to answer instantly) and
recorded by the executor, per this task's ground rules.
