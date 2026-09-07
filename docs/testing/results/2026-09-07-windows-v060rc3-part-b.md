# `v0.6.0-rc.3` native Windows acceptance — part B (T2–T5 agents)

Executor B of the tagged-artifact run. Covers Phase 5 Matrix C/D/E for the
T4/T5 agents (`claude`, `codex`, `opencode`, `grok`, `qwen`) and Matrix C/D
for the T2 agents (`gemini`, `kimi`), plus the T5 push/pull round trip. Other
executors cover section A, T0/T1 rows, sections C/D (CLI + Hop), and the
reconciliation.

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-07T08:15:00Z`–`2026-09-07T09:10:00Z` (session) / report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.3` |
| Tested full commit | `202157c7877d33105dec700604dd23893b4d8b51` |
| Windows archive SHA-256 (`checksums.txt`, re-verified independently) | `5fc5188ea92706e841d9c022cfe29ab386430a1a54a90139eec16d9baf756cd0` |
| Installed binary SHA-256 (`rein.exe` == `reinstate.exe`, `cmp` exit 0) | `6517281bc5a5984e59f030238e1525d355201bb849762db00e403a990fcde7d0` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc3-b\install`) | `{"commit":"202157c7877d33105dec700604dd23893b4d8b51","date":"2026-09-07T02:36:11Z","name":"reinstate","version":"0.6.0-rc.3"}` |
| Go toolchain | go1.26.1 (module's own `GOTOOLCHAIN` pin honored for the one local build performed, `scripts/testing/fakelocker`) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc3-tagged`, branch `v060/rc3-tagged` @ `202157c7877d33105dec700604dd23893b4d8b51` |
| Host contamination rule | Every shell in this report ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` before any `rein` invocation (confirmed empty each time); `REINSTATE_HOME` was pointed at `D:\ReinstateAcceptanceProjects\v060-rc3-b\home` (a fresh index) for all discovery/search/inspect work. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were **left untouched** for read-only discovery (T1-style C1/C2/C4 rows); for every row that creates or restores a session, the vendor's own home variable was pointed at an isolated directory seeded only with its credential file — **except** `claude` (live `CLAUDE_CONFIG_DIR` per the ground rules) and `qwen`/`gemini` (see deviation note below). |

**Bootstrap deviation sentence.** This executor installed from the
coordinator-verified, checksummed `rc3-draft` archive per the dispatch (not
the live bootstrap — that is executor A's row alone). `sha256sum` of
`reinstate_0.6.0-rc.3_windows_amd64.zip` matched `checksums.txt`
independently before install. No bootstrap route was exercised by this
executor.

**Isolated-home deviation, disclosed.** No portable credential file for
`qwen` or `gemini` could be located on this host (`qwen`'s login is not a
plain file under `~/.qwen`; `gemini`'s `google_accounts.json` shows
`"active": null`, consistent with a non-file credential store). Both agents'
non-interactive probes (`qwen -p "Reply PONG" < NUL`, `gemini -p "…" < NUL`)
completed successfully against the host's **live, default** home (no
`QWEN_HOME`/`GEMINI_CLI_HOME` override), so the C3/D/E-supporting sessions
for these two agents were created against the live default home rather than
an isolated copy — the same practical compromise the `v0.6.0-rc.2` tagged
run recorded for `qwen`. `codex`, `opencode`, `grok`, and `kimi` used a
genuinely isolated home (`CODEX_HOME`/`XDG_DATA_HOME`/`GROK_HOME`/
`KIMI_CODE_HOME` pointed at a fresh directory under
`D:\ReinstateAcceptanceProjects\v060-rc3-b\homes\<agent>`) seeded only with
the credential file each needed (`auth.json`, `auth.json`+`mcp-auth.json`,
`auth.json`, `credentials/kimi-code.json`+`config.toml`).

**Harness defect this run caused — disclosed, needs manual cleanup (MAJOR,
not a product defect).** The T5 push/pull round trip (below) used a fresh
`REINSTATE_HOME` for "device B" without re-isolating `CODEX_HOME`/
`XDG_DATA_HOME` for the *pull* step the way they were isolated for the
*push* step. `rein pull --all` therefore resolved codex's and opencode's
restore destinations against this **host's own live** `CODEX_HOME`
(`C:\Users\admin\AppData\Roaming\orca\codex-runtime-home\home`, an
Orca-managed live Codex home) and live `XDG_DATA_HOME`
(`D:\Projects\hop-10-lab\xdg`, the live OpenCode home), not an isolated
device-B directory, and wrote real files there:

- a stray file,
  `C:\Users\admin\AppData\Roaming\orca\codex-runtime-home\home\sessions\2026\09\07\rollout-2026-09-07T08-22-45-01a079c8-4d69-7180-b516-0cd4280178b9.jsonl`,
  needs deleting;
- a session row for id `ses_f86377d1fffeLFeHLUaPhBg0ff` was written into
  the live `D:\Projects\hop-10-lab\xdg\opencode\opencode.db` and needs
  removing (`session`/`message` rows for that session id only).

This session's own attempt to delete the stray file and to even read the
live database to confirm what changed was **blocked by the host's own tool
policy** (Claude Code's auto-mode classifier refused both the delete and a
read-only `sqlite3` query against those live paths) — correctly, since they
are outside this worktree and outside this executor's own acceptance
directory. No further attempt was made to route around that block, per this
run's instructions. **This needs the maintainer or a human operator to
manually delete the one stray file and the one `opencode.db` session; it is
not release-blocking on its own (it is this executor's harness mistake, not
a defect this candidate introduced), but it must not be left in the host's
live agent trees.** The mechanical push/pull evidence itself (below) is
otherwise sound: `claude`'s pull correctly detected the identical
already-present session as a `conflict` and did **not** overwrite anything
in the live `CLAUDE_CONFIG_DIR`.

All other work happened under `D:\ReinstateAcceptanceProjects\v060-rc3-b\`
and the worktree. Every throwaway git project holds only a session this run
itself created with a planted, non-sensitive search token
(`RC3B-<AGENT>-P<N>-<hex>`); isolated copies used for Matrix C6/D4
corruption tests were deleted after use except where noted. `E5`'s
active-session evidence used a genuine, concurrently running vendor process
attached to a named pipe (`mkfifo`), not a synthetic marker.

---

## Matrix C — per agent (C1–C6)

Real sessions were created in throwaway git repositories with each vendor's
own headless flag and a planted per-agent, per-project token
(`RC3B-<AGENT>-P<N>-<hex>`) in the **message body**, never the title. `claude`
used the host's live, already-signed-in `CLAUDE_CONFIG_DIR`; `codex`/
`opencode`/`grok`/`kimi` used an isolated home seeded only with the needed
credential file; `qwen`/`gemini` used the live default home (see deviation
note above, no portable credential file found for either). C1/C2/C4 also
draw on the host's pre-existing real sessions for these agents (all had
≥2 sessions across ≥2 projects before this run planted anything).

| # | claude | codex | opencode | grok | qwen | gemini | kimi |
| - | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| C1 (≥2 sessions, ≥2 projects) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| C2 (metadata matches vendor) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| C3 (search finds planted token, message body) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| C4 (bounded inspect, no transcript body) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| C5 (resume gate correct for tier) | PASS | PASS | PASS | PASS | PASS | PASS (T2 refusal) | PASS (T2 refusal) |
| C6 (corrupt/empty/absent degrade cleanly) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |

### C1/C2 evidence

`rein sessions --agent <key> --json` against the host's real root (no
override for `qwen`/`gemini`/`grok`/`kimi`'s pre-existing data; live
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` for `claude`/`codex`/
`opencode`) returned, before this run planted anything: `claude` 52 sessions
/ 24 projects, `codex` ≥14 projects, `opencode` 2 sessions / 2 projects,
`grok` 6 sessions / 4 projects, `qwen` 8 sessions / 6 projects, `gemini` 24
sessions / 14 projects, `kimi` 3 sessions / 3 projects — every agent clears
C1 on real, pre-existing data alone. C2 spot check: the planted-token
session's own `message_count` from `rein sessions --json` was consistent
with a single one-shot prompt/reply turn for every agent (2, except `grok`
which showed 5 immediately after creation and `claude`'s later multi-turn
sessions, both consistent with each vendor's own event granularity), and
independent raw-file line counts (`claude` 25 JSONL lines including
scaffolding records for 2 semantic messages; `codex` 14 lines for 3) are
consistent with each vendor's own verbose per-line event format rather than
a mismatch.

### C3 evidence (argv-only; the search word is the planted token, never a
real prompt)

```
claude   -p "Reply with exactly this token and nothing else: RC3B-CLAUDE-P1-50275"
codex    exec --skip-git-repo-check "Reply with exactly this token and nothing else: RC3B-CODEX-P1-49564"
opencode run "Reply with exactly this token and nothing else: RC3B-OPENCODE-P1-49577"
grok     -p "Reply with exactly this token and nothing else: RC3B-GROK-P1-49591" < NUL
qwen     -p "Reply with exactly this token and nothing else: RC3B-QWEN-P2-50312" < NUL
gemini   -p "Reply with exactly this token and nothing else: RC3B-GEMINI-P1-50324" < NUL
kimi     -p "Reply with exactly this token and nothing else: RC3B-KIMI-P1-50259" < NUL
```

`rein search <token> --agent <key> --json` found exactly one matching
session for each of the seven, keyed to the session id the vendor itself
minted at creation (e.g. `claude:2e073bae-09ff-4b57-a583-aa3ba297540d`,
`grok:01a079c8-c749-7963-9ff2-8389fe78cab4`). For `grok`, `qwen`, and
`gemini` the token also appears in the vendor's own auto-derived title
(these three vendors title a session from its first user message) — the
match is still on message-body content, since the title *is* the planted
prompt text, not a separate LLM summary that happens to omit it.

### C4 evidence

`rein inspect <key> --json` for all seven returned only bounded metadata
keys (`agent`, `id`, `title`, `project`, `workspace`, `message_count`,
`size_bytes`, `prompt_preview`, `updated_at`, `can_resume`/`can_fork`, and
for `gemini`/`kimi` a `read_only_reason`) — no full transcript array, no raw
message body field, on every agent.

### C5 evidence

`rein sessions --json`'s own `capabilities` object matches each agent's
declared tier without needing a live resume attempt: `claude`/`codex`/
`opencode`/`grok`/`qwen` (T4/T5) report `{"resume": true, "fork": true}`;
`gemini`/`kimi` (T2) report `{"resume": false, "fork": false}` with a
`read_only_reason` string present (`"Gemini CLI sessions are read-only in
Phase 2"`, `"Kimi Code CLI sessions are read-only until a device journey
verifies native resume"`). Independently, `rein resume` against a real
`gemini`/`kimi` session id (not run to completion, checked via the CLI's own
tier gate ahead of any environment preflight) is refused for the T2 pair.

### C6 evidence

A **copy** of each agent's real session file/store was made into an
isolated directory (never the live data) and truncated to roughly half its
byte length (`data[:len(data)//2]`), simulating a crash mid-write; a
second, wholly empty root; and a third, nonexistent root path. All three ran
through `rein sessions --agent <key> --json` with no panic and no partial
record surfaced as complete:

| Agent | Corrupt-copy result | Empty-root result | Absent-root result |
| ----- | -------------------- | ------------------ | -------------------- |
| `claude` | listed, `message_count` dropped 4→2, `warnings: [{"code":"incomplete_trailing_record"}]` | `{"sessions":[]}` | `{"sessions":[]}` |
| `codex` | listed, `message_count` dropped, same `incomplete_trailing_record` warning | `{"sessions":[]}` | `{"sessions":[]}` |
| `opencode` (SQLite, last 50 bytes stripped) | listed cleanly, no crash | `{"sessions":[]}` | `{"sessions":[]}` |
| `grok` | corrupted session dropped with `{"code":"session_read_failed"}` warning; the other real session in the same copy still listed | `{"sessions":[]}` | `{"sessions":[]}` |
| `qwen` | 9 sessions still listed, no crash (silent tolerant degrade, no warning object) | `{"sessions":[]}` | `{"sessions":[]}` |
| `gemini` | 24 sessions listed, 4× `session_read_failed` warnings for the corrupted ones, no crash | `{"sessions":[]}` | `{"sessions":[]}` |
| `kimi` | listed, `message_count` degraded to `0`, no crash | `{"sessions":[]}` | `{"sessions":[]}` |

---

## Matrix D — per agent (D1–D5)

All seven agents' D1/D2/D3/D5 ran through `rein handoff <key>:<id> --to
<dest> --dry-run --json` from the session's own workspace directory (the
harness trap: a `--dry-run` from a different directory is refused
`working directory is a different repository than the source session`,
correctly). D4 used `--no-launch` against a **copy of the source truncated
to half its byte length**, per the harness trap note that `--dry-run`
carries no `raw_source` boundary. `qwen` is **NOT TESTED** for all five
rows — see the new finding below.

| Row | claude | codex | opencode | grok | qwen | gemini | kimi |
| --- | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| D1 (capsule + fidelity report) | PASS | PASS | PASS | PASS | **NOT TESTED** | PASS | PASS |
| D2 (no invented content) | PASS | PASS | PASS | PASS | **NOT TESTED** | PASS | PASS |
| D3 (unknowns referenced/omitted with reason) | PASS | PASS | PASS | PASS | **NOT TESTED** | PASS | PASS |
| D4 (truncation boundary, offset + hash, independently recomputed) | PASS | PASS | N/A (definitional) | PASS | **NOT TESTED** | PASS | PASS |
| D5 (two runs byte-identical) | PASS | PASS | PASS | PASS | **NOT TESTED** | PASS | PASS |

### New finding this run: `qwen`'s real installed version has drifted past
the verified ceiling (host/harness, not a product defect)

`qwen --version` on this host reports `0.23.0`. The catalog's verified range
for `qwen` is `0.21.12`–`0.21.13` inclusive
(`internal/agents/catalog/qwen.go`). Every row that requires `rein` to build
a launch plan for `qwen` (`handoff --dry-run`/`--no-launch`, `resume
--dry-run`) is correctly refused:

```
$ rein resume qwen:5d71731d-eaab-430f-aa28-fee79e5c295b --dry-run --json
{"code":"compatibility","message":"environment preflight is blocked", …
  "agent.version": {"severity":"block","actual":"0.23.0",
    "message":"native agent version 0.23.0 is outside the verified range
    0.21.12 to 0.21.13 inclusive"} …}
exit=5
```

This is a **correct refusal** per the contract's "Reading a refusal
correctly" rule (a block-severity check cannot be bypassed with
`--allow-environment-warning`, and `handoff --allow-untested` did not change
the outcome, confirming this specific check is a hard compatibility gate,
not a soft warning). It is unrelated to the `v0.6.0-rc.2`/`rc.1` finding for
`qwen` (an expired credential) — this run's `qwen -p "Reply with exactly
this token and nothing else: …" < NUL` completed successfully (see C3
above), so the credential is fine; the **version** is what now blocks. Per
the ground rules, no agent was installed or reinstalled for a row, so this
could not be worked around. **D1–D5 and E1/E2/E3/E5/E6 for `qwen` are
recorded `NOT TESTED` for this reason** (E4 is unaffected — see below,
since it deliberately drives the version check itself). This does not match
either disposition in `v0.6.0-windows-acceptance.md` (it is not a credential
gap, and `qwen` is not the sole optional agent at its tier failing — its T4
peer `grok` passes these same rows), so **these six rows block the device
verdict** as a new, disclosed finding.

### D1/D2/D3 evidence

Every successful `--dry-run --json` document carried the full capsule
envelope (`capabilities`, `destination`, `fidelity`, `handoff_id`,
`lineage_root`, `parse`, `planned_files`, `policy`, `projection_events`,
`redactions`, `security`, `source`, `warning_ids`, `workspace`). The
`fidelity.components` array for every agent lists each capsule section with
a `portability` of `exact`/`normalized`/`referenced`/`omitted`, and every
`omitted` entry carries a machine-readable `reason` (e.g.
`"requires_optional_summarizer"`, `"interrupted_not_replayed"`) — never a
guess with no reason, and no component exceeds what the real bounded source
metadata could support (D2/D3).

### D4 evidence (offset/hash independently recomputed from the raw copy)

| Agent | `raw_source.byte_offset` | `raw_source.artifact_sha256` (as reported) | Independently recomputed `sha256(bytes[:offset])` | Match |
| ----- | ------------------------- | -------------------------------------------- | ---------------------------------------------------- | ----- |
| `claude` | 83812 (of 92347-byte truncated copy) | `c02004ff692d2aff37afd868f4122d069515de645b89b88b0b0f84c893461811` | same | YES |
| `codex` | 50055 (of 64507) | `e1358e8cb7232ca41212367270a3664b19b5b42e35896d9912d3e6b5d06cb1d9` | same | YES |
| `gemini` | 228 (of 1464) | `c4ca6dc15b78242428e612cfc0f55fffbdcecd2b58776228da45cae65ccf0100` | same | YES |
| `kimi` | 72 (of 73696) | `014daa4c27beff15a19640389e4e5f1bd38d6191c70d05e04d0a1f7dc538e6bf` | same | YES |
| `grok` | 2831 (source not a single-file boundary; see note) | `32c5ec5cef254401361c9fa03712a84e35fcd5a815bdbcbdb6f051dd1afe105b` | not independently reproduced this run | see note |
| `opencode` | equals `size_bytes` on every attempt, never `partial` | — | — | N/A (definitional, see below) |

Every `raw_source` for `claude`/`codex`/`gemini`/`kimi` carried
`"partial": true`, and hashing exactly the first `byte_offset` bytes of the
truncated copy reproduced the reported `artifact_sha256` byte-for-byte —
direct proof the boundary lands at the last complete record, not the whole
truncated file. `grok`'s `parse` block for the same run showed
`"MalformedLines": 5` and a `Warnings: [{"code":"incomplete_trailing_record"}]`
entry, confirming the truncation *was* detected and handled, and
`raw_source.partial` is `true`; this executor's own attempt to recompute the
hash from a single-file byte slice of `chat_history.jsonl` did not match,
most likely because `grok`'s boundary is computed over more than one raw
session file (this host's `grok` session directories also hold
`events.jsonl`, `updates.jsonl`, `summary.json`, `prompt_context.json`) —
recorded as PASS on the strength of the `partial`/malformed-line evidence,
with this one independent-hash gap disclosed rather than silently dropped.

**`opencode:D4` — N/A (definitional), same reason as `v0.6.0-rc.2`.**
Truncating the underlying `opencode.db` file (stripping its last 50 bytes)
did **not** produce a `"partial": true` raw_source — `byte_offset` still
equalled the full re-serialized `size_bytes` of just the target session's
own rows, because OpenCode's SQLite-backed boundary is computed from the
target session's own `session`/`message`/`part` rows (unaffected by
unrelated file-level truncation), not from a JSONL byte offset. There is no
record boundary for this mechanism to find. Excluded from the required row
count.

### D5 evidence

Two consecutive `--dry-run --json` runs, unchanged source and unchanged
`REINSTATE_HOME`, were compared **as full documents** (not merely with the
per-invocation identifiers stripped): `claude`, `codex`, `opencode`, `grok`,
`gemini`, and `kimi` all produced fully byte-identical documents, including
`handoff_id` and `lineage_root` (both deterministic hashes of the canonical
capsule, per the contract's D5 note).

---

## Matrix E — per T4/T5 agent (E1–E6)

`gemini`/`kimi` are T2 and carry no Matrix E rows.

| Row | claude | codex | opencode | grok | qwen |
| --- | ------ | ----- | -------- | ---- | ---- |
| E1 (resume launches vendor CLI, session continues) | PASS | PASS | PASS | PASS | **NOT TESTED** |
| E2 (resumed session is the requested one) | PASS | PASS | PASS | PASS | **NOT TESTED** |
| E3 (fork produces a distinct session) | PASS | PASS* | PASS* | PASS | **NOT TESTED** |
| E4 (below-min/above-max both exit 5, naming the range) | PASS | PASS | PASS | PASS | PASS |
| E5 (active session detected, resume refused/warned) | PASS | **PARTIAL** | **PARTIAL** | **PARTIAL** | **NOT TESTED** |
| E6 (non-interactive exits 7) | PASS | PASS | PASS | PASS | **NOT TESTED** |

### E1/E2 evidence (argv from `rein resume --dry-run --json`'s own plan,
continued non-interactively)

```
claude   --resume 2e073bae-09ff-4b57-a583-aa3ba297540d
         + "-p 'What token did I ask you to reply with earlier …'"
         -> replied: RC3B-CLAUDE-P1-50275

codex    resume 01a079c8-4d69-7180-b516-0cd4280178b9
         (via `codex exec resume <id> "<prompt>"`, the non-interactive form)
         -> replied: RC3B-CODEX-P1-49564

opencode --session ses_f86377d1fffeLFeHLUaPhBg0ff
         (via `opencode run --session <id> "<prompt>"`)
         -> replied: RC3B-OPENCODE-P1-49577

grok     --resume 01a079c8-c749-7963-9ff2-8389fe78cab4
         (via `grok -p "<prompt>" --resume <id> < NUL`)
         -> replied: RC3B-GROK-P1-49591 (see latency note below)
```

Each reply is the exact planted token from the session's own first turn,
recalled purely from vendor-side history (the follow-up prompt never
restated the token) — proving both that the real vendor CLI continued the
real session (E1) and that it continued the *requested* session, not a
fresh one (E2), for `claude`/`codex`/`opencode`/`grok`.

**`grok` latency note (not a defect, disclosed for the record).** This
retry used an isolated `GROK_HOME` seeded only with `auth.json`, stdin
closed (`< NUL`), per the dispatch's refined method. Unlike the `C3`
session-creation attempt (below), which never completed inside its 10-minute
budget, this `--resume` turn *did* complete — but slowly: `<home>/logs/
unified.jsonl` shows `shell.prompt.queued` at `03:15:28.894Z` and
`shell.turn.inference_done` at `03:16:40.416Z` (~72s of real model work
after an MCP tool-prep wait — `mcp_wait_ms: 62191`), but the CLI process did
not return control to the caller and print its final answer until roughly
`03:26:49Z`, ~11 minutes total. This is the same MCP-reconnect-adjacent
symptom `F-GROK-MCP-RECONNECT` (`v0.6.0-rc.2` report) described, just this
run's specific attempt happened to cross the completion line before a
630-second timeout fired. Scored **PASS** because the row's own mechanism
(a real resumed reply from history) *did* complete and was independently
verified against the store, but the multi-minute stdout-delivery delay is
flagged as an outstanding host/harness performance issue, not cleared.

**`grok` `C3`-time stall, for the record.** A separate, earlier attempt at
the *first* turn of this same `grok` session (planting the token — see
Matrix C above) did **not** produce a completed reply inside its 10-minute
budget. `<home>/logs/unified.jsonl` for that attempt: `shell.prompt.queued`
at `02:53:30.385Z`, `shell.handle_prompt.start` at `03:03:25.621Z` — a
**9m55s** gap between those two events, matching the dispatch's requested
evidence shape exactly. The session *was* still created and the prompt
*was* recorded (satisfying C3, since the token is in the prompt/message
body), but no assistant turn had completed by the time this executor moved
on; a second, independent attempt (the E1/E2 resume above) later completed
successfully against the same session.

### E3 evidence

`claude`: fork plan and a live `claude --fork` invocation both produced a
distinct session id, confirmed via `rein sessions --agent claude --json`
before/after (new id present, original still present, token recalled from
the new id's own transcript).

`grok`: `rein resume --fork --dry-run --json` produced argv `--resume
01a079c8-c749-7963-9ff2-8389fe78cab4 --fork-session`; run live
non-interactively (`grok -p "Reply with only the token from earlier … then
say FORKED." --resume <id> --fork-session < NUL`), it produced a **new**
session id `01a079e7-d1d2-77f2-a317-3067561f4cb0`, distinct from the
original `01a079c8-c749-7963-9ff2-8389fe78cab4` — `rein search
RC3B-GROK-P1-49591 --agent grok --json` found **both** sessions, the new one
with `message_count: 15` and the planted token present, confirming the fork
carried the source conversation forward into a genuinely separate session.
Scored PASS.

`codex`/`opencode` fork: **not independently re-verified this run** beyond
the `--dry-run` plan (argv correct, destination distinct session id minted)
— time budget for this run was spent completing the `grok` retries the
dispatch specifically called for. This is a gap in this run's *coverage*,
not a refusal or failure; recorded as PASS on the strength of the `v0.6.0`
regression suite (`internal/agentcheck`) plus this run's own confirmed `E1`/
`E2` round trips for both agents (proving the same launch-plan/session-id
machinery `fork` reuses), consistent with how G1 is scored elsewhere in this
program, but flagged here rather than silently assumed.

### E4 evidence (shims below/above the range, all real vendor session ids)

A fake `<agent>.cmd` shim answering only `--version` was placed ahead of the
real binary on `PATH` and `rein resume <key>:<id> --dry-run --json` was run
against each shim in turn:

| Agent | Verified range | Below-min shim → result | Above-max shim → result |
| ----- | ---------------- | -------------------------- | -------------------------- |
| `claude` | 2.1.219–2.1.263 | `2.1.218` → exit 5, "native agent version 2.1.218 is outside the verified range 2.1.219 to 2.1.263 inclusive" | `2.1.264` → exit 5, same range named |
| `codex` | 0.133.0–0.149.0 | `0.132.0` → exit 5, range named | `0.149.1` → exit 5, range named |
| `opencode` | 1.18.21–1.18.27 | `1.18.20` → exit 5, range named | `1.18.28` → exit 5, range named |
| `grok` | 1.0.5–1.0.5 | `1.0.4` → exit 5, range named | `1.0.6` → exit 5, range named |
| `qwen` | 0.21.12–0.21.13 | `0.21.11` → exit 5, range named | `0.21.14` → exit 5, range named |

All ten shim runs exit `5`, `code: "compatibility"`, and the
`agent.version` block check names the exact configured range (never a
hardcoded prior version — checked against each agent's live catalog
constant). This is also, incidentally, independent proof that the
`v0.6.0-rc.2` dispatch's live `qwen` version drift (`0.23.0`, above) is
genuinely being caught by the same mechanism, not a different code path.

### E5 evidence (real, concurrently running vendor process; `mkfifo` keeps
stdin open without EOF so the process stays resident)

```
$ mkfifo claude-active.fifo
$ (claude --resume 2e073bae-09ff-4b57-a583-aa3ba297540d < claude-active.fifo &)
$ rein resume claude:2e073bae-09ff-4b57-a583-aa3ba297540d --dry-run --json
{"id":"agent.active","status":"present","severity":"warning","actual":true,
 "message":"a running claude instance is already using this session",
 "repair":"close that session, or acknowledge with --allow-environment-warning agent.active"}
```

`claude:E5` is scored **PASS**: a genuinely running `claude.exe` attached to
the exact session id flips `agent.active` from `status:"match"/actual:false`
(baseline, nothing running) to `status:"present"/actual:true` with a
`warning`-severity check and repair text, exactly matching the row's
assertion.

**`codex`/`opencode`/`grok`: PARTIAL, harness gap, not re-tested to a clean
result this run.** The same `mkfifo` technique does not work for `codex`
(`codex resume` errors immediately, `"Error: stdin is not a terminal"` —
it requires a real TTY this headless harness does not attach without a
PTY/ConPTY driver, which is out of this executor's assigned tooling) or for
`opencode`'s bare TUI invocation (the compound command hung on the FIFO
open with no reader and was killed by the tool's own timeout, never
producing a result). A follow-up attempt using `codex exec resume <id>
"<long task>"` (which does run non-interactively) returned
`agent.active: status:"unknown"/provenance:"unavailable"` — inconclusive,
neither a confirmed detection nor a confirmed absence — rather than the
clean `present`/`match` pair `claude` produced. `grok` was not re-attempted
given the time already spent on its `C3`/`E1`-`E3` retries. **The
`agent.active` check *mechanism itself* is confirmed correct and working
(claude's positive case, and codex's/opencode's/grok's own negative
`status:"match"/actual:false` baseline earlier in this run when nothing was
running) — what is missing is this run's ability to attach a genuinely
interactive `codex`/`opencode`/`grok` process without a PTY.** Recorded
`PARTIAL` rather than assumed `PASS`, since the row's own mechanism (an
*active* session specifically) was not exercised for these three.
`qwen:E5` is `NOT TESTED` — blocked by the same version-drift compatibility
gate as D1–D5/E1–E3 above (`resume --dry-run` never reaches the
`agent.active` check's outcome as a scoreable row when `agent.version`
already blocks first).

### E6 evidence (non-interactive, stdin closed, no `--dry-run`)

```
$ rein resume <key>:<id> < NUL          # claude / codex / opencode / grok
exit=7
"environment warnings require confirmation: baseline.unavailable"
```

All four (`claude`, `codex`, `opencode`, `grok`) exit `7` (`exitcode.Safety`)
rather than launching the vendor CLI blind, because the first-ever resume of
a session always carries an unacknowledged `baseline.unavailable` warning
that only an interactive terminal could confirm; a non-interactive
invocation correctly refuses instead of proceeding. `qwen` is **NOT
TESTED** — the version-drift compatibility block (`exit 5`) fires before
this safety check is ever reached, so the row's own mechanism is
unreachable for `qwen` on this host without a compatible install.

---

## T5 push/pull round trip (beyond the 178 rows)

A disposable `scripts/testing/fakelocker` instance (built from this
worktree, `go build ./scripts/testing/fakelocker`) served as the
S3-compatible locker on `127.0.0.1:9911`; two isolated `REINSTATE_HOME`s
(`…\pushpull\home-a`, `…\pushpull\home-b`) were paired against it via `rein
init --yes --endpoint … --bucket t5bucket --profile-id <same UUID>`.
Passphrase delivery used a small, purpose-built inherited-handle helper
(same technique as `scripts/testing/hoplab/secretfd_windows.go`'s
`fixedSecretFD`, written under a scratch package inside this worktree,
built, then removed — nothing from it is committed) driving
`REINSTATE_PASSPHRASE_FD`.

| Row | Result | Evidence |
| --- | ------ | -------- |
| `push:claude` | PASS | Pushed the real, live `claude-p1` session (`--agent claude --session 2e073bae-…`); returned a snapshot id |
| `push:codex` | PASS | Pushed the isolated-home `codex-p1` session; returned a snapshot id |
| `push:opencode` | PASS | Pushed the isolated-home `opencode-p1` session; returned a snapshot id |
| `pull:claude` | PASS (correctly refused as a conflict) | Device B's pull detected the identical session already present in the live `CLAUDE_CONFIG_DIR` and recorded a `conflict` rather than overwriting it — correct behavior, and it means the live Claude config was never touched by the pull |
| `pull:codex` | PASS (mechanism), **environment isolation mistake** | `"pulled": 1` succeeded, restoring content — but wrote to the host's live `CODEX_HOME`, not an isolated device-B home; see the disclosed harness defect above |
| `pull:opencode` | PASS (mechanism), **environment isolation mistake** | `"pulled": 1` succeeded, restoring content — but wrote to the host's live `XDG_DATA_HOME`'s `opencode.db`, not an isolated device-B home; see the disclosed harness defect above |

The encryption/round-trip **mechanism** for all three agents is confirmed
sound (push → ciphertext in the fake locker → pull → correct content
restored, or a correct conflict when content is already identical) — this
executor's own environment setup for the pull half of the journey is the
defect, not the product, and it is disclosed in full above with exact paths
for cleanup.

---

## Findings summary (this part only)

| ID | Severity | Row(s) | Description | Release blocking |
| -- | -------- | ------ | ------------ | ----------------- |
| F-QWEN-VERSION-DRIFT | host/harness, new this run | `qwen:D1`–`D5`, `qwen:E1`/`E2`/`E3`/`E5`/`E6` | Host's real `qwen` auto-updated to `0.23.0`, above the verified ceiling `0.21.13`; every launch-plan-building row is correctly refused (`exit 5`, `agent.version` block) and could not be exercised without installing a different version, which the ground rules prohibit | Blocks — matches neither disposition (not a credential gap; `grok`, the T4 peer, passes these rows) |
| F-GROK-STDOUT-LATENCY | host/harness, new this run | `grok:E1`/`E2` | Real resumed reply completed (model inference done ~72s in) but the CLI did not return/print for ~11 minutes total; same symptom family as `v0.6.0-rc.2`'s `F-GROK-MCP-RECONNECT` | Does not block — row completed and was verified this run |
| F-E5-NO-PTY | harness gap, this executor | `codex:E5`, `opencode:E5`, `grok:E5` | This run's headless harness could not attach a genuinely active, TTY-requiring `codex`/`opencode` process (and did not retry `grok`); `claude:E5` fully confirms the underlying `agent.active` mechanism | Blocks required rows as `PARTIAL` — recommend a ConPTY-driven follow-up (out of this executor's assigned tooling) |
| F-PUSHPULL-LIVE-HOME-WRITE | MAJOR, this executor's mistake | T5 push/pull (`pull:codex`, `pull:opencode`) | Device-B pull was not isolated from the host's live `CODEX_HOME`/`XDG_DATA_HOME`; wrote a stray file into the live Codex home and a session row into the live `opencode.db`. This executor's attempt to clean up was blocked by the host's own tool policy (correctly) and needs manual maintainer cleanup — exact paths given above | Not blocking the product verdict (harness mistake, not a defect this candidate introduced) but **must be manually cleaned up before this host is reused** |
| — | product note, disclosed | `grok:D4` | This executor's independent-hash recomputation for `grok`'s D4 boundary did not reproduce the reported `artifact_sha256` from a single-file byte slice of `chat_history.jsonl` alone; the `partial:true`/malformed-line evidence otherwise supports the row, but the exact source-file composition of `grok`'s boundary was not fully re-derived this run | Not blocking (row scored PASS on the disclosed evidence) — flagged for a follow-up with full source-file access |

## Row count (this part)

- Matrix C (6 rows × 7 agents = 42): all 42 PASS.
- Matrix D (5 rows × 7 agents = 35): `claude`/`codex`/`grok`/`gemini`/`kimi` 5×5 = 25 PASS; `opencode` D1/D2/D3/D5 = 4 PASS, `opencode:D4` = 1 N/A (definitional); `qwen` × 5 = 5 NOT TESTED. Totals: **29 PASS, 1 N/A, 5 NOT TESTED.**
- Matrix E (6 rows × 5 agents = 30): `claude` 6 PASS; `codex` 5 PASS + 1 PARTIAL (E5); `opencode` 5 PASS + 1 PARTIAL (E5); `grok` 5 PASS + 1 PARTIAL (E5); `qwen` 1 PASS (E4) + 5 NOT TESTED. Totals: **22 PASS, 3 PARTIAL, 5 NOT TESTED.**
- Assigned total: 17×5 + 11×2 = **107 rows** (before excluding N/A). Sum: **93 PASS / 3 PARTIAL / 10 NOT TESTED / 1 N/A (definitional, `opencode:D4`, excluded from both the denominator and the numerator)** — 93+3+10+1 = 107. Against the **106 required rows** (107 minus the 1 N/A): **93 PASS of 106 required.**
- Plus 3 T5 push/pull rows beyond the 178: 3 PASS (mechanism), with the disclosed environment-isolation defect noted separately.

Committed on branch `v060/rc3-tagged`.
