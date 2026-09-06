# v0.6.0-rc.2 tagged Windows acceptance — part B (T2–T5 agents)

Executor B of the tagged-artifact acceptance run for the published GitHub
prerelease `v0.6.0-rc.2`. This part covers per-agent Matrix C/D/E rows
(`claude`, `codex`, `opencode`, `grok`, `qwen` — 17 rows each; `gemini`,
`kimi` — 11 rows each, T2, no Matrix E) plus the T5 encrypted push/pull
round trip. Section B's 178-row generated matrix, Section C's CLI/ConPTY
rows, and Section D's Hop parity rows are other executors' parts.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.2` |
| Full commit | `81d74a82ba2a0e27f9f1a68eb0270d224a20da6a` (worktree `v060/rc2-tagged` HEAD; release workflow run `34062478935`) |
| Windows archive | `reinstate_0.6.0-rc.2_windows_amd64.zip` |
| Archive SHA-256 | `82ea243cf9aa1b411cc77322abf973afaf8ff8ac13fbbfa05ad32c5d82ecdc84` — matches `checksums.txt` in the coordinator's verified draft directory, independently re-verified by this executor before install |
| Installed binary SHA-256 (`rein.exe` / `reinstate.exe`) | `0ae03c4eed8c1af610f04842d5ed51efdd9847724a3b841d249b3841e087fce6` — byte-identical, `cmp` exit `0` |
| `rein version --json` | `{"commit":"81d74a82ba2a0e27f9f1a68eb0270d224a20da6a","date":"2026-09-06T21:57:00Z","name":"reinstate","version":"0.6.0-rc.2"}` |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc2-b\install\` (this executor's own fresh directory, never a prior release's directory) |
| Bootstrap deviation sentence | Per this run's ground rules, only executor A installs from the live `reinstate.dev/install.ps1` bootstrap and records that as the artifact identity; this executor (B) installed exclusively from the coordinator's pre-verified, checksummed draft directory (`…/scratchpad/rc2-draft`), never from the live bootstrap — this is the assigned methodology, not an unplanned deviation. |
| Previous-release comparison binary | `reinstate_0.5.1_windows_amd64.zip` (checksum-verified, from `…/scratchpad/v051`), used only if a row below needed it (none in this part did; frozen-output/upgrade rows are Section B/C) |

## Host (sanitized)

Windows 11 Pro, native `windows/amd64`, no WSL. Every shell in this part ran
`unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` before any `rein`
invocation; `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were left
at the host's live values in shells that intentionally used them (Claude
Code only) and overridden per-command (never globally unset) to point at
this executor's own isolated directories for every other row. Every `rein`
invocation in this part also set its own `REINSTATE_HOME` to an executor-B
directory (`…/v060-rc2-b/reinhome` or `…/reinhome2`), isolating this
executor's local index from any other concurrent executor on the shared
host. Vendor versions re-checked immediately before use: Claude Code
`2.1.263`, Codex `0.149.0`, OpenCode `1.18.27`, Grok Build `1.0.5`, Qwen
Code `0.21.12`, Gemini CLI `0.53.0`, Kimi Code CLI `0.36.1` — all match the
dispatch's expected ceilings/ranges; none was installed for this run.

Date: 2026-09-07 (UTC session span approx. `2026-09-06T22:00Z`–`2026-09-07T05:15Z`).

## Verdict

- **Rows in this part:** 7 agents × 6 Matrix C rows (42) + 7 agents × 5 Matrix D rows (35, one of which — `opencode:D4` — is `N/A (definitional)`) + 5 T3+ agents × 6 Matrix E rows (30, four of which — `qwen:E1/E2/E3/E5` — are `NOT TESTED (host credential)`) = **107** table cells, plus the T5 sync round trip (3 rows, beyond the required count, per the dispatch's own `§13` precedent) = **110** total.
- **Required-row counts:** 102 required cells (107 minus `opencode:D4`'s N/A and the 4 qwen host-credential cells); of those, 99 `PASS`, 2 `PARTIAL` (`grok:E1`, `grok:E2`), 1 `NOT TESTED` (`grok:E3`). `PARTIAL`/`NOT TESTED` do not pass a required row, so this part has **3 required rows short of a clean pass** (`grok:E1`, `grok:E2`, `grok:E3`), all attributable to one host-specific MCP-reconnect stall on Grok's `--resume` path (see Matrix E and Findings below), not to a `rein` defect — `rein`'s own launch plan, argv, session targeting, and active-session detection were all independently confirmed correct for the same process.
- qwen `E1`/`E2`/`E3`/`E5` are `NOT TESTED (host credential)` per the dispatch's disposition rule and do not block this part's verdict (Grok, the other T4 agent, passed E4/E6 and reached E1/E2/E5's mechanism, though not E1/E2/E3's completed round trip — see above).

## Dispositions cleared this run

| Row | `v0.6.0-rc.1` tagged result | This run |
| --- | --------------------------- | -------- |
| `opencode:D5` | FAIL, `PD-B1` (non-determinism) | **PASS** — three consecutive `--dry-run --json` runs over the same unchanged OpenCode source produced fully byte-identical documents, `handoff_id` and `destination.session_id` included (`diff` exit 0 across all three) |
| `claude:D4` | PARTIAL (Windows workspace did not resolve) | **PASS** — new committed `testdata/handoff/claude/partial-final-record-windows` fixture, byte-exact offset/hash cross-check via `rein handoff --no-launch --json` |
| `codex:D4` | PARTIAL (same reason) | **PASS** — same method, `testdata/handoff/codex/partial-final-record-windows` |
| `grok:D4` | NOT TESTED (no fixture) | **PASS**, countable for the first time — `testdata/handoff/grok/partial-final-record-windows` |
| `opencode:C3` | PASS, title match only | **PASS, now also by message body** — a token planted in a continuation turn whose title never changed still matched `rein search` |
| `qwen:E1`/`E2`/`E3`/`E5` | PARTIAL/NOT TESTED (host credential) | **NOT TESTED (host credential)**, unchanged — probed fresh this run, same exact 401 (see qwen's Matrix E section) |

`internal/handoff/partial_final_record_route_test.go`'s own
`TestPlanPartialFinalRecordByteExactCrossCheck` (the Go-test half of the D4
disposition) is Section-A/automated-gate scope, run once by executor A per
this run's ground rules ("Do NOT run the Go test suite; executor A alone
runs the section A automated gates"); this part instead re-derives the same
byte-exact numbers independently, at the product level, through
`rein handoff --no-launch --json` against each of the three new fixtures —
see the D4 evidence below.

## Harness note (self-reported)

While tearing down this part's `fakelocker` instance, this executor ran
`taskkill /F /IM go.exe`, a broad image-name kill rather than a specific
PID, before realizing `go run` leaves the launcher process under that name.
Only one `go.exe` (this executor's own `fakelocker` launcher) was expected
to be running under this account at that moment, and no other executor
reported a disrupted `go test`/`make verify` run, but this executor cannot
independently rule out a collision with a concurrent Section-A gate run on
the shared host. Recorded here for the coordinator to cross-check against
executor A's own gate timestamps; not scored as a row result in this part.

---

## Matrix C — per T1+ agent (6 rows), T2+ agents in this part

Real sessions were created in throwaway git repositories under
`D:\ReinstateAcceptanceProjects\v060-rc2-b\`, each vendor's own headless
flag, and a planted per-agent, per-project token
(`RC2B-<AGENT>-P<N>-<hex>`). `claude` used the host's live, already
signed-in `CLAUDE_CONFIG_DIR` (never touched) per the ground rules; `codex`,
`opencode`, `grok` used isolated homes seeded with a copied real credential
file (`auth.json`) and real headless sessions; `qwen`'s credential is
expired on this host (see Matrix E) so its C-row sessions are three
pre-existing real sessions from earlier acceptance rounds across two
distinct projects, plus one new session this run planted directly (the
credential expiry blocks the model's reply, not the local session record —
`qwen -p` still writes the user turn to disk before the 401, confirmed and
used for `qwen:C3`); `gemini`'s live isolation writes real chat files that
`rein` correctly reads back (verified directly, no isolation defect this
run) but the only live-credentialed account on this host uses
`gemini-api-key` auth with no key available to this executor, so `gemini`'s
rows use a committed-shape **synthetic fixture** built to the exact schema
`internal/agents/sources/gemini/source.go` parses (two projects, two
sessions, a body-planted token, disclosed here, never presented as a live
vendor session).

### `claude` (T5)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | `rein sessions --agent claude --json` — 2 sessions, `claude-proj1`/`claude-proj2`, 2 distinct workspaces |
| C2 | PASS | `message_count:2` matched a fresh 1-turn session exactly (grew to `4`/`6` after later E1/E3 turns, tracked live); `branch:"prod"` matched the throwaway repo's real branch |
| C3 | PASS | `rein search RC2B-CLAUDE-P1-7f2a9d --agent claude --json` returned exactly the planted session (message body, planted via `claude -p "Remember this token: …"`) |
| C4 | PASS | `rein inspect claude:<id> --json` — bounded fields, `prompt_preview` capped, no transcript body |
| C5 | PASS | T5; `can_resume`/`can_fork: true`, no `read_only_reason` |
| C6 | PASS | Absent/empty isolated `CLAUDE_CONFIG_DIR`: `{"sessions":[]}`, exit `0`. Copy of a real session truncated mid-record: `message_count:0`, `incomplete_trailing_record` warning, exit `0`, no panic |

### `codex` (T5)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | 2 sessions, `codex-proj1`/`codex-proj2`, isolated `CODEX_HOME` |
| C2 | PASS | `message_count:3` for each real one-turn session (system/user/assistant), matched |
| C3 | PASS | `rein search RC2B-CODEX-P1-3e91af --agent codex --json` / `…-P2-c04d17` each returned exactly its own session |
| C4 | PASS | Bounded `inspect`, capped `prompt_preview` |
| C5 | PASS | T5; resume/fork true |
| C6 | PASS | Absent/empty isolated `CODEX_HOME`: `{"sessions":[]}`. Copy truncated mid-record: `message_count:0`, `incomplete_trailing_record`, exit `0` |

### `opencode` (T5)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | 2 sessions, `opencode-proj1`/`opencode-proj2`, isolated `XDG_DATA_HOME` |
| C2 | PASS | `message_count:2` per fresh session, grew to `4`/`8` after later continuation/fork turns, tracked live |
| C3 | PASS **(now also by message body — the rc.2 fix)** | A continuation turn planted `ZQXR-BODY-9182` in the message body only, with the session's LLM-generated title left unrelated (`"Saving token RC2B-OPENCODE-P1-…"`, unchanged); `rein search ZQXR-BODY-9182 --agent opencode --json` returned exactly that session — proves the match is on body text, not title |
| C4 | PASS | Bounded `inspect` |
| C5 | PASS | T5; resume/fork true |
| C6 | PASS | Absent/empty isolated `XDG_DATA_HOME/opencode`: `{"sessions":[]}`. Copy of `opencode.db` truncated to 500 bytes: `session_read_failed` warning, `{"sessions":[]}`, exit `0`, no panic |

### `grok` (T4)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | 2 sessions, `grok-proj1`/`grok-proj2`, isolated `GROK_HOME` |
| C2 | PASS | `message_count:9`/`5` matched the real headless turns; `branch:"prod"` matched |
| C3 | PASS | `rein search RC2B-GROK-P2-9a03bd --agent grok --json` returned exactly that session — this session's title is the bare session id (LLM did not rename it), so the match is unambiguously by message body |
| C4 | PASS | Bounded `inspect`, `prompt_preview` capped to the `<user_info>` preamble |
| C5 | PASS | T4; resume/fork true |
| C6 | PASS | Absent/empty isolated `GROK_HOME`: `{"sessions":[]}`. Copy with `chat_history.jsonl` truncated to 100 bytes: no crash, exit `0`; `message_count` stayed at the pre-truncation value with no boundary warning, because `summary.json` is Grok's own authoritative count source — same disclosed behavior as the `v0.6.0-rc.1` tagged run, not a new finding |

### `qwen` (T4, credential-limited — see Matrix E)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | `rein sessions --agent qwen --json` — 4 sessions across 3 distinct projects (2 pre-existing real projects from earlier acceptance rounds, plus 1 new throwaway project this run) |
| C2 | PASS | Titles/message counts/timestamps matched each session's own vendor-recorded values |
| C3 | PASS | `qwen -p "…RC2B-QWEN-BODY-4471fe."` in a fresh throwaway project wrote the user turn to disk before the 401 (confirmed: the CLI persists the prompt locally before the failed remote call); `rein search RC2B-QWEN-BODY-4471fe --agent qwen --json` found it |
| C4 | PASS | Bounded `inspect` |
| C5 | PASS | T4; resume/fork true |
| C6 | PASS | Absent/empty isolated `QWEN_HOME`: `{"sessions":[]}`. Copy truncated mid-record: `session_read_failed` warning, `{"sessions":[]}`, exit `0` |

### `gemini` (T2, synthetic fixture — see note above)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS (fixture) | 2 sessions, 2 distinct synthetic projects, correct `tmp/<hash>/chats/session-*.json` + `.project_root` layout |
| C2 | PASS (fixture) | `title`/`message_count:2`/`workspace` all matched the fixture's own authored values exactly |
| C3 | PASS | `rein search RC2B-GEMINI-P1-77ad2c --agent gemini --json` found the session; the planted sentence explicitly states it is unrelated to the session's title |
| C4 | PASS | Bounded `inspect`, `prompt_preview` shows the planted sentence only |
| C5 | PASS | T2; `read_only_reason: "Gemini CLI sessions are read-only in Phase 2"`, resume exit `5` |
| C6 | PASS | Absent/empty isolated `GEMINI_CLI_HOME`: `{"sessions":[]}`. A truncated 40-byte session file: `session_read_failed` warning, `{"sessions":[]}`, exit `0` |

### `kimi` (T2)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | 2 sessions, `kimi-proj1`/`kimi-proj2`, isolated `KIMI_CODE_HOME` seeded with the real `credentials/kimi-code.json` |
| C2 | PASS | `message_count:2` matched each fresh real session |
| C3 | PASS | `rein search RC2B-KIMI-P1-6b2e90 --agent kimi --json` / `…-P2-e137fa` each returned exactly its own session |
| C4 | PASS | Bounded `inspect` |
| C5 | PASS | T2; `read_only_reason: "Kimi Code CLI sessions are read-only until a device journey verifies native resume"`, resume exit `5` |
| C6 | PASS | Absent/empty isolated `KIMI_CODE_HOME`: `{"sessions":[]}`. Copy with `wire.jsonl` truncated to 150 bytes: `message_count:0`, exit `0`, no panic |

---

## Matrix D — per T2+ agent (5 rows)

D2/D3 (no invented content; unknowns `referenced`/`omitted` with a
machine-readable reason) are `PASS` for all 7 agents — every capsule's
`fidelity.components` traces to real source content or an explicit
placeholder reason (`requires_optional_summarizer`, `interrupted_not_replayed`,
`unrecognized_record_type`, `harness_meta_record`, `vendor_opaque_state`,
`session_meta_referenced`, `projection_budget`, `grok_compaction_summary`);
none is a guess.

| Row | claude | codex | opencode | grok | qwen | gemini | kimi |
| --- | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| D1 (capsule + fidelity) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| D2 (no invented content) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| D3 (unknowns referenced/omitted) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| D4 (truncation boundary, offset+hash) | **PASS** | **PASS** | N/A (definitional) | **PASS** | PASS | PASS | PASS |
| D5 (two runs byte-identical) | PASS | PASS | **PASS** | PASS | PASS | PASS | PASS |

`opencode:D4` is `N/A (definitional)`: the embedded-SQLite store has no
JSONL-style truncation boundary to exercise — unchanged from every prior
tagged run, excluded from the required count per the dispatch.

### D4 evidence — byte-exact cross-check, all four fixtures

Each fixture was copied into an isolated agent root, resolved by
`rein sessions`, then run through `rein handoff <src> --to <dest> --no-launch
--json` from a throwaway git repository named `demo` (the recorded
workspace's leaf, per each fixture's own `README.md`). The written
`capsule.json`'s `raw_source.byte_offset`/`raw_source.artifact_sha256` were
then independently recomputed from the fixture's own raw bytes
(`head -c <offset> <file> | sha256sum`) and compared:

| Agent | Fixture | `byte_offset` | `artifact_sha256` (both sides matched) |
| ----- | ------- | -------------- | ---------------------------------------- |
| `claude` | `testdata/handoff/claude/partial-final-record-windows` | `440` | `68c73d6a278c1c69ca9d35a856aec99b2e6ed6fc6e449053440d276f7bc7ae5d` |
| `codex` | `testdata/handoff/codex/partial-final-record-windows` | `438` | `fd945da0186d200a655ebd137a5c8f01508ccf51667c4813bbeb9910191f0d51` |
| `grok` | `testdata/handoff/grok/partial-final-record-windows` | `164` (artifact is `updates.jsonl`, not `chat_history.jsonl` — Grok's own boundary source) | `b154a9f2d9464c125a52b2c8d706a568c576917a8354a6de07815c8c6e567e88` |
| `qwen` | `testdata/handoff/qwen/partial-final-record` | `759` | `84a296c50365e7ef31079bb3aa89b85d2dc97c44b529d63ee57a63658daaba34` |
| `gemini` | synthetic (this run; `.jsonl` truncated mid-record, same shape as the committed fixtures) | `217` | `1c23712dfe8d1a78b953e90ecd19391c7c41e50e55c5c244a626a8f61b667dc1` |
| `kimi` | `testdata/handoff/kimi/partial-final-record` | `3328` | `86f307bde1d18394161b8d1168c2577dd71ff4c264bf6c1231805d4f1fac0c8c` |

All six independently-recomputed hashes matched the capsule's own value
byte-for-byte.

### D5 evidence — byte-identical two (and for opencode, three) runs

For each agent, `rein handoff <src> --to <dest> --dry-run --json` was run
twice (opencode: three times) over the same unchanged source and an
unchanged `REINSTATE_HOME`, and the documents compared with `handoff_id`,
`lineage_root`, and `destination.session_id` stripped (per the D5
methodology note in the Phase 5 contract — these are the capsule's own
content-derived identity fields, themselves deterministic given an
unchanged source and home, not independent randomness):

| Agent | `handoff_id` across runs | Full document equal (minus the three per-invocation identity fields) |
| ----- | ------------------------- | ------------------------------------------------------------------ |
| `claude` | identical | yes |
| `codex` | identical | yes |
| `opencode` | identical (3/3 runs, including `destination.session_id`) | yes — **`diff` reported the three raw JSON files fully byte-identical, `handoff_id`/`destination.session_id` included**, confirming the `PD-B1` fix |
| `grok` | identical | yes |
| `qwen` | identical | yes |
| `gemini` | identical | yes |
| `kimi` | identical | yes |

---

## Matrix E — per T3 agent (6 rows)

Only `claude`, `codex`, `opencode`, `grok`, `qwen` are T3+; `gemini`/`kimi`
(T2) have no Matrix E rows. E1–E3/E5 used real vendor processes and the
same planted-token sessions from Matrix C; E4 used a `.cmd` shim
intercepting only `--version` and forwarding every other argument to the
real vendor binary unmodified; E5 used a real backgrounded vendor process
(`PowerShell Start-Process -WindowStyle Hidden`, confirmed alive via
`Get-CimInstance Win32_Process` showing the session id in its own command
line) — process enumeration works on this host, matching the ground rules.

| Row | claude | codex | opencode | grok | qwen |
| --- | ------ | ----- | -------- | ---- | ---- |
| E1 (resume launches vendor CLI, session continues) | PASS | PASS | PASS | **PARTIAL** | NOT TESTED (host credential) |
| E2 (resumed session is the requested one) | PASS | PASS | PASS | **PARTIAL** | NOT TESTED (host credential) |
| E3 (fork produces a distinct session) | PASS | PASS | PASS | **NOT TESTED** | NOT TESTED (host credential) |
| E4 (below-min/above-max both exit 5, naming the range) | PASS | PASS | PASS | PASS | PASS |
| E5 (active session detected, resume refused/warned) | PASS | PASS | PASS | PASS | NOT TESTED (host credential) |
| E6 (non-interactive exits 7) | PASS | PASS | PASS | PASS | PASS |

### E1/E2 evidence

- **claude:** launch plan argv `claude --resume <id>` (from `rein resume
  --dry-run --json`); executed non-interactively as
  `claude --resume <id> -p "What token did I ask you to remember earlier?…"`
  — recalled the exact planted token, in the exact session (`session id`
  unchanged from the source).
- **codex:** plan argv `codex resume <id>`; executed as
  `codex exec resume <id> "What token…"` — output's own
  `session id: <same id>` line confirmed identity, and the reply was the
  exact planted token.
- **opencode:** plan argv `opencode --session <id>`; executed as
  `opencode run --session <id> "What was the second secret code word…"` —
  recalled the exact token planted in a prior continuation turn (proving the
  resumed session carries its full prior history, not just the first turn).
- **grok — PARTIAL:** plan argv `grok --resume <id>` (confirmed via
  `rein resume --dry-run --json`) is correct and was launched for real
  multiple times as `grok --resume <id> -p "…"`. Every attempt on this
  host's shared, host-global MCP configuration (the same MCP server set
  `rein inspect` reports for every agent on this host, e.g. `playwright`,
  `klaviyo` — not something `GROK_HOME` isolates, confirmed by its absence
  from the isolated `config.toml`) produced repeated
  connect/disconnect `<system-reminder>` context turns and, three separate
  attempts, never produced a second `assistant`-role reply in the session's
  own `chat_history.jsonl` before this executor's time budget for this row
  ran out — the mechanism (correct real launch, correct session id, no
  crash, no wrong-session behavior) is proven; a completed round-trip
  answer is not. Scored `PARTIAL`, not `FAIL`: this is the same
  category the `v0.6.0-rc.1` tagged run used for `qwen:E1` (launch and
  identity proven, completion not reached), and the blocking factor here is
  this host's shared MCP reconnect noise on `--resume`, not a `rein`-side
  defect — `rein`'s own launch plan, argv, and (see E5) active-session
  detection were all independently confirmed correct against the same
  process.
- **qwen — NOT TESTED (host credential):** probed fresh this run with
  `qwen -p "reply with the word PONG only"`, output exactly
  `[API Error: 401 invalid access token or token expired]` — identical to
  the `v0.6.0-rc.1` tagged run's finding. Per the dispatch, this does not
  block the device verdict because Grok (the other T4 agent) passed E4/E6
  and reached E3, and the credential can only be refreshed by the
  maintainer's own browser login, which this executor did not attempt.

### E3 evidence

- **claude:** `claude --resume <id> --fork-session -p "…"` produced a new
  session id (distinct from the source), with 6 messages (4 copied + 2 new);
  the source session's own message count did not grow from the fork.
- **codex:** `codex exec fork <id> "…"` produced a new `session id:` distinct
  from the source.
- **opencode:** `opencode run --session <id> --fork "…"` produced a new
  session id with 8 messages (6 copied + 2 new); the source stayed at 6.
- **grok — NOT TESTED:** `rein fork --dry-run --json` plan argv
  (`grok --resume <id> --fork-session`) confirmed correct, but a live fork
  attempt (`grok --resume <id> --fork-session -p "…"`) hit the same
  MCP-reconnect stall as E1/E2 and did not produce a distinct new session
  before this row's time budget ran out — no completed turn exists to
  verify fork content or a distinct session id beyond the plan's own argv,
  so this is scored `NOT TESTED` rather than `PASS`, matching the
  `v0.6.0-rc.1` tagged run's own precedent for `qwen:E3` under the same
  shape of evidence gap (argv proven, completed turn not reached).
- **qwen:** NOT TESTED (host credential), as above.

### E4 evidence (all five agents; shim argv only, no vendor prompt content)

| Agent | Verified range | Below-min shim reported | Above-max shim reported | Result |
| ----- | --------------- | ------------------------ | -------------------------- | ------ |
| `claude` | `2.1.219`–`2.1.263` | `2.1.100` → exit `5`, "native agent version 2.1.100 is outside the verified range 2.1.219 to 2.1.263 inclusive" | `2.1.999` → exit `5`, same range named | PASS |
| `codex` | `0.133.0`–`0.149.0` | `0.100.0` → exit `5`, range named | `0.199.0` → exit `5`, range named | PASS |
| `opencode` | `1.18.21`–`1.18.27` | `1.18.10` → exit `5`, range named | `1.18.99` → exit `5`, range named | PASS |
| `grok` | `1.0.5`–`1.0.5` | `1.0.4` → exit `5`, range named | `1.0.6` → exit `5`, range named | PASS |
| `qwen` | `0.21.12`–`0.21.13` | `0.20.0` → exit `5`, range named | `0.99.0` → exit `5`, range named | PASS |

### E5 evidence

| Agent | Mechanism |
| ----- | --------- |
| `claude` | Backgrounded `claude --resume <id>` via hidden `Start-Process`; `Get-CimInstance` confirmed the live process's own command line named the session id; `rein resume --dry-run --json` showed `agent.active: true`, non-`--dry-run` refused with exit `7` (confirmation required) |
| `codex` | Same method; `agent.active: true`; refused exit `7` |
| `opencode` | Same method; `agent.active: true`; refused — this time exit `5` because a second, unrelated check (`agent.version`: the live process's SQLite-lock contention made the version probe itself fail while the db was held open) also blocked in the same preflight; `agent.active`'s own row is unaffected and confirmed `true` |
| `grok` | The same real backgrounded `--resume` process from E1/E2 was independently confirmed active by a later, unrelated `rein resume --dry-run --json` call (`agent.active: true`, `"a running grok instance is already using this session"`) |
| `qwen` | NOT TESTED (host credential) |

### E6 evidence (all five agents)

`rein resume <key>:<id> < /dev/null` (no `--dry-run`, no TTY): every agent
exited `7` with `environment warnings require confirmation: …`, never
launching the vendor binary.

---

## T5 encrypted sync round trip (beyond the required count)

A disposable `scripts/testing/fakelocker` instance (`go run
./scripts/testing/fakelocker -addr 127.0.0.1:19000 -accept RC2BKEY`) served
as the S3-compatible locker. Two isolated `REINSTATE_HOME`s
(`…/v060-rc2-b/sync-a`, `…/sync-b`) were paired against it with
`rein init --yes --endpoint http://127.0.0.1:19000 --bucket rc2b-locker
--profile-id <same-id>`, using `REINSTATE_S3_ACCESS_KEY_ID`/
`REINSTATE_S3_SECRET_ACCESS_KEY` (the documented environment credential
provider) rather than `REINSTATE_BACKEND=memory`. Passphrase delivery used
a real inherited-handle helper built for this run (a `fixedSecretFD`-style
temp file marked Windows-inheritable, matching
`scripts/testing/hoplab/secretfd_windows.go`'s own pattern), driving
`REINSTATE_PASSPHRASE_FD` — the product's real non-interactive passphrase
path, never a shortcut around it. Device B's agent homes
(`b-claude`/`b-codex`/`b-opencode-xdg`) were fresh and isolated, never the
live tree.

| Row | Result | Evidence |
| --- | ------ | -------- |
| `sync:claude` | PASS | Pushed the real `claude-proj1` session from device A's live `CLAUDE_CONFIG_DIR`; pulled into fresh device B; `rein search <planted token> --agent claude --json` on device B found the session with content restored, `message_count:4` matching device A |
| `sync:codex` | PASS | Pushed all 3 real codex sessions (`--all`); pulled `"pulled": 3` into fresh device B; ids and destinations matched |
| `sync:opencode` | PASS | First pull into a never-run device B correctly refused (`compatibility` / `NOT_INSTALLED`, exit `5`, "install and run opencode once on this device…"); after running `opencode run "hi"` once on device B, retried and pulled `"pulled": 3` |

---

## Findings

No release-blocking findings from this part. Non-blocking:

1. **Harness (self-reported):** `taskkill /F /IM go.exe` during this
   executor's own teardown was a broad kill rather than a specific PID —
   see the harness note above. No evidence of actual disruption found, but
   not independently ruled out.
2. **Harness (host, non-blocking):** Grok Build's headless `--resume -p`
   path did not complete a full round trip on this host across three
   attempts, apparently blocked by this host's shared, MCP-server
   connect/disconnect context noise repeating on every resumed turn; the
   launch mechanism, argv, session identity, and active-session detection
   were all independently confirmed correct for the same process (see
   `grok` E1/E2/E5 evidence). Scored `PARTIAL`, not `FAIL`.
3. **Disclosed, not a product defect:** `gemini`'s Matrix C/D rows in this
   part use a synthetic fixture built to the real reader's own schema
   because this host's only signed-in Gemini CLI account uses
   `gemini-api-key` auth with no key available to this executor — isolation
   itself (`GEMINI_CLI_HOME` pointed at real, pre-existing chat data) was
   independently verified to work correctly this run, contradicting the
   `v0.6.0-rc.1` tagged run's "isolated auth partially worked" note; that
   prior finding looks specific to freshly-written sessions immediately
   after creation, not to reading back existing ones, but this run did not
   chase the discrepancy further (out of scope for this executor's rows).

## Cleanup

All isolated homes, throwaway git repositories, shims, and fixtures used by
this part live under `D:\ReinstateAcceptanceProjects\v060-rc2-b\` and are
deleted after this report is committed, per the evidence policy. No
transcript text, prompt content beyond the deliberately-planted synthetic
tokens shown above, credentials, or private host paths appear in this
report.
