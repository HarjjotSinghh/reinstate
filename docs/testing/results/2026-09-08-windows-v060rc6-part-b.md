# 2026-09-08 — v0.6.0-rc.6 tagged Windows acceptance — Part B (T2–T5 agents)

`PART-B-TAGGED-REPORT-V1`

Executor: tagged-run executor B (T2–T5 agents), worktree
`D:\Projects\reinstate-worktrees\v060-rc6-tagged`, branch `v060/rc6-tagged`.
This part covers per-agent Matrix C (`C1`–`C6`), Matrix D (`D1`–`D5`), Matrix
E (`E1`–`E6`) for `claude`, `codex`, `opencode`, `grok`, `qwen` (17 rows each)
and Matrix C/D for `gemini`, `kimi` (11 rows each), plus the T5 encrypted
sync round trip. It does not cover section A gates, Matrix B/F/G/H, the
22-row CLI matrix, or Hop parity — those are other executors' parts.

## Header — artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.6` |
| Full commit | `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| Release workflow run | `34170072716` (per dispatch; not independently re-verified this part — see executor A) |
| Archive | `reinstate_0.6.0-rc.6_windows_amd64.zip` |
| Archive SHA-256 (independently recomputed) | `ff7e232958b6141d3091e343ef0246306819dcc428fa161101416e3bead22ffd` — matches `checksums.txt` |
| `rein.exe` / `reinstate.exe` SHA-256 | `272efb504a62fdd2624fa6b117a2587a4869160c034adeb9fcd6281231f078c1` (both, byte-identical) |
| `rein version --json` | `{"commit":"7c5adccaea602ff952cd31c21bafa7fdd22a2cfc","date":"2026-09-07T23:28:49Z","name":"reinstate","version":"0.6.0-rc.6"}` |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc6-b\install\` (this executor's own fresh directory) |
| Bootstrap deviation | Not applicable to this part — per the dispatch, only executor A installs from the live `reinstate.dev/install.ps1` bootstrap and records that as the artifact identity; every other executor (this one included) installs from the coordinator's pre-verified, checksummed draft directory `…\scratchpad\rc6-draft`, which this part re-verified independently above. |
| Host | Windows 11 Pro, build 10.0.26200, amd64 (sanitized; no private path beyond what the ground rules permit) |
| Go toolchain | `go1.26.1 windows/amd64` |
| Git | `2.52.0.windows.1` |
| Date (UTC) | 2026-09-08 |
| Shell hygiene | `REINSTATE_BACKEND` and `REINSTATE_MEMORY_BACKEND_DIR` were unset at the start of every shell that ran `rein`, the Hop lab, or a Go build in this part, immediately before each block of commands below. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were left at the host's live values except where a row's own methodology called for pointing them at an isolated directory (noted per row). |
| Results file | `docs/testing/results/2026-09-08-windows-v060rc6-part-b.md` |

Every `rein`/`reinstate` invocation below used the full path
`D:\ReinstateAcceptanceProjects\v060-rc6-b\install\rein.exe` (never a
PATH-resolved binary).

## Evidence policy notes specific to this part

- `claude`, `codex`, `opencode` sessions were created through each vendor's
  own live/isolated home in throwaway git projects under
  `D:\ReinstateAcceptanceProjects\v060-rc6-b\<agent>-proj\`, per the ground
  rules for T5 agents. Only this executor's own planted session ids are
  named below.
- `grok`, `qwen` used an isolated home (`GROK_HOME`/`QWEN_HOME`) seeded only
  with the one credential file each vendor needs (`auth.json` /
  `settings.json`), copied once from the host's real home and never read
  again afterward.
- `gemini`, `kimi` used the host's live default home, per the ground rules
  for those two agents, in throwaway git projects.
- Every C3 row below was proven by a token planted in a **message body**,
  never a session title. Where a vendor's own title-generation would have
  put the literal first message into the title (`opencode`, `qwen`), the
  token was planted in a **second turn** of a two-turn session so the title
  (generated from turn 1) never carries it, and `rein search` was shown
  finding the session by the turn-2 token with a token-free title as proof
  the match is on message body, not title.
- All planted tokens follow the shape `RC6B-<AGENT>[N]-<epoch>[-<suffix>]`.
  No transcript text, real prompts, credentials, private paths, or session
  ids not created by this executor appear below.

---

## Matrix C — per T2+ agent (66 rows: 66 PASS)

Real sessions, planted per-agent token in a message body, `rein search`
confirms match; `rein inspect --json` shows a bounded `prompt_preview` (not
the full transcript); `rein resume --dry-run --json` shows the tier-correct
gate decision; a truncated/corrupt/empty/absent copy of the session store
degrades cleanly (a warning code and `exit 0`, or an empty session list —
never a crash).

| Row | claude | codex | opencode | grok | qwen | gemini | kimi |
| --- | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| C1 (≥2 sessions, ≥2 projects) | PASS — 71/33 | PASS — real session in own project, ≥2 projects on host root | PASS — 20/9 | PASS — 3/2 | PASS — 4/2 | PASS — 29/19 | PASS — 6/5 |
| C2 (metadata matches vendor) | PASS — `project=claude-proj`, `message_count=2` | PASS — `project=codex-proj`, `message_count=3` | PASS — `project=opencode-proj`, `message_count=4` | PASS — `project=grok-proj`, `message_count=10` | PASS — `project=qwen-proj`, `message_count=4` | PASS — `project=gemini-proj`, `message_count=4` | PASS — `project=kimi-proj`, `message_count=4` |
| C3 (message-body search, never title) | PASS — token found; title = session id | PASS — token found; title = session id | PASS — token planted in turn 2; title = "One-word greeting request" (turn 1's own summary, token-free) | PASS — token found; title vendor-generated from turn 1's own text, token-free | PASS — token planted in turn 2; title = "Say hello in one word." (token-free) | PASS — token found; title = session id | PASS — token found; title = session id |
| C4 (bounded inspect, no full body) | PASS — `prompt_preview` bounded (verified: a 389-byte prompt truncates to 159 chars in `inspect --json`) | PASS — 160 chars | PASS — 25 chars | PASS — 160 chars | PASS — 22 chars | PASS — 22 chars | PASS — 22 chars |
| C5 (resume gate correct for tier) | PASS — `confirmation_required` | PASS — `confirmation_required` (see finding on real host version drift, below) | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — refused, `native session action is unsupported: Gemini CLI sessions are read-only in Phase 2`, exit 5 | PASS — refused, `native session action is unsupported: Kimi Code CLI sessions are read-only until a device journey verifies native resume`, exit 5 |
| C6 (corrupt/empty/absent degrade cleanly) | PASS — truncated copy → `incomplete_trailing_record`, exit 0; empty/absent → `{"sessions":[]}` | PASS — same shape (`incomplete_trailing_record`) | PASS — SQLite copy truncated 50% → `session_read_failed`, exit 0; empty/absent clean | PASS — truncated copy read cleanly (no crash), exit 0; empty/absent clean | PASS — same shape (clean read, no crash) | PASS — same shape (clean read, no crash) | PASS — same shape (clean read, no crash) |

**42/42 PASS.**

C1 evidence (session/project counts as observed by `rein sessions --agent
<a> --json` at the time of each row, all real, no synthetic seeding beyond
what this executor itself created):

- `claude`: pre-existing live-config catalog, 71 sessions / 33 projects.
- `codex`: one throwaway session created this run (`codex exec`); the host
  root otherwise carries real, pre-existing sessions across ≥2 projects.
- `opencode`: 20 sessions / 9 projects (mixed pre-existing + this run's own).
- `grok` (isolated `GROK_HOME`): 3 sessions / 2 throwaway projects, all
  created this run.
- `qwen` (isolated `QWEN_HOME`): 4 sessions / 2 throwaway projects, all
  created this run.
- `gemini`/`kimi` (live default home): pre-existing catalogs, 29/19 and 6/5.

---

## Matrix D — per T2+ agent (34 rows: 34 PASS, 1 N/A definitional)

Source mechanism: `rein handoff <key>:<id> --to <dest> --dry-run --json`
(D1–D3, D5); D4 used `--no-launch` against a copy of the session's own raw
store truncated to ~70% of its byte length, with the capsule's own
`raw_source.byte_offset`/`artifact_sha256` **independently recomputed** from
the raw truncated bytes (`sha256sum` over the first `byte_offset` bytes) for
every agent below — every recomputed hash matched the capsule's recorded
hash exactly.

| Row | claude | codex | opencode | grok | qwen | gemini | kimi |
| --- | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| D1 (capsule + fidelity report) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| D2 (no invented content) | PASS — 15 components | PASS — 16 components | PASS — 14 components | PASS — 16 components | PASS — 15 components | PASS — 14 components | PASS — 15 components |
| D3 (unknowns referenced/omitted with reason) | PASS — 1 unrecognized (`unrecognized_record_type`) | PASS — 1 unrecognized + 1 metadata (`session_meta_referenced`) | PASS — 1 unrecognized | PASS — `metadata`/`vendor_opaque_state`, `summaries`/`grok_compaction_summary`, `user_messages`/`projection_budget` | PASS — `metadata`/`harness_metadata` (no unrecognized records this session) | PASS — none this session | PASS — `metadata`/`harness_meta_record` |
| D4 (truncation boundary, offset + hash, independently recomputed) | PASS — offset 84836, hash `0987ca1b…` matched | PASS — offset 72866, hash `d07828d6…` matched | N/A (definitional) | PASS — targets `updates.jsonl`, offset 6795, hash `67bdb1c6…` matched | PASS — offset 7390, hash `3fcfaec1…` matched | PASS — offset 4943 (with a copy of `projects.json` for workspace resolution), hash `c05d10e5…` matched | PASS — targets `agents/main/wire.jsonl`, offset 78780, hash `1a997baf…` matched |
| D5 (two runs byte-identical) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |

**`opencode:D4` — N/A (definitional), unchanged reason:** OpenCode's
SQLite-only store has no JSONL record boundary this row's mechanism applies
to. Excluded from the required count.

D5 evidence: for every agent, two consecutive `handoff … --dry-run --json`
calls against the identical source session, with `handoff_id`,
`lineage_root`, `destination.args`, and `planned_files` (which embed the
random `handoff_id`) stripped before comparison, were byte-identical
(`json.dumps(..., sort_keys=True)` equality) — confirming everything the row
actually claims deterministic (the capsule's own content) is deterministic.

---

## Matrix E — per T3+ agent (30 rows: 27 PASS, 3 NOT TESTED version drift)

Source mechanism: E1/E2 via each vendor's own non-interactive resume flag
against the dry-run plan's exact argv, asking the resumed agent to recall a
token only the original session's first turn could know; E3 the same via
each vendor's fork flag, confirming a **distinct** session id also recalls
the token; E4 via a version-reporting shim (`--version` only; PATH-prepended
ahead of the real binary) both below and above the verified range, real
`resume` (not `--dry-run`), confirming `exit 5` and the range named in the
refusal message; E5 via `scripts/testing/conptydriver` (built from this
worktree) launched through PowerShell `Start-Process -WindowStyle Hidden
-Wait` with no stdout/stderr redirection on the driver itself, holding the
vendor CLI open long enough for a second shell's `rein resume --dry-run
--json` to observe `agent.active status=present, actual=true`; E6 the same
non-interactive resume attempt with `stdin` redirected from empty, expecting
`exit 7` and the interactive-terminal message.

| Row | claude | codex | opencode | grok | qwen |
| --- | ------ | ----- | -------- | ---- | ---- |
| E1 (resume launches vendor CLI, session continues) | PASS — planted token recalled | **NOT TESTED (version drift)** — see below | PASS — planted token recalled | PASS — maintainer console transcript (below); corroborated independently by this executor's own harness (below) | PASS — planted token recalled |
| E2 (resumed session is the requested one) | PASS — same session id, `message_count` grew | **NOT TESTED (version drift)** | PASS — same session id | PASS — maintainer transcript; corroborated | PASS — same session id |
| E3 (fork produces a distinct session) | PASS — distinct id `51186022-47fa-4958-b7e0-e2eb0f47df40` | **NOT TESTED (version drift)** | PASS — distinct id `ses_f81aaae09ffe18A5iTHLMniJt8` (title suffixed `(fork #1)`) | PASS — maintainer transcript names distinct id `01a07e60-d7bf-7611-b29d-1281f790eaf9`; corroborated independently with distinct id `01a07e5f-09e0-7d92-ad64-dc68a100d085` | PASS — distinct id `149b65b3-95cd-4bdf-9982-6bfb4b31b941` |
| E4 (below-min/above-max both exit 5, naming the range) | PASS — `2.1.219–2.1.263` | PASS — `0.133.0–0.149.0` (real installed `0.153.4` itself served as the above-max case; a `0.132.0` shim served below-min) | PASS — `1.18.21–1.18.29` | PASS — `1.0.5–1.0.13` | PASS — `0.21.12–0.23.0` |
| E5 (active session detected, resume refused/warned) | PASS — via ConPTY, `agent.active status=present, actual=true` | PASS — via ConPTY, same shape | PASS — via ConPTY, same shape | PASS — via ConPTY, same shape (`grok.exe`, not `grok.cmd` — see harness note) | PASS — via ConPTY, same shape |
| E6 (non-interactive exits 7) | PASS — exit 7 | PASS — exit 7 (required an in-range version shim; the real installed `0.153.4` is itself outside range and blocks at `exit 5` before reaching this check) | PASS — exit 7 | PASS — exit 7 | PASS — exit 7 |

**27/30 PASS, 3 NOT TESTED (version drift) — `codex:E1`/`E2`/`E3`.**

### Codex version drift — `codex:E1`/`E2`/`E3` NOT TESTED (version drift)

The host's live, installed Codex CLI reports `codex-cli 0.153.4`
(`codex --version`), outside this candidate's verified range `0.133.0` to
`0.149.0` inclusive (unchanged from `v0.6.0-rc.5`; Codex is not among the
ranges this candidate widens). Every real, non-`--dry-run` resume attempt
against the true installed binary is correctly refused before reaching the
resume/recall mechanism:

```
> rein resume codex:01a07e49-eb98-7ad3-87bd-b634837daf06
Environment check: agent.version status=unknown severity=block — native agent version 0.153.4 is outside the verified range 0.133.0 to 0.149.0 inclusive
environment preflight is blocked
exit 5
```

Per the standing `NOT TESTED (version drift)` disposition
(`v0.6.0-windows-acceptance.md`), this is recorded on `codex:E1`/`E2`/`E3`
rather than scored — the affected rows still count toward the required
denominator and do not by themselves clear the device verdict; a required
Codex range widening is needed before these three can `PASS` again.
`codex:C1`–`C6`, `D1`–`D5`, and `E4`–`E6` above are unaffected: none of
those rows' mechanisms require a real, non-`--dry-run` launch against the
live vendor binary — `E4` uses a fabricated shim on both sides, `E5`/`E6`
were tested against a shim reporting `0.149.0` for the same reason (their
own mechanism is `rein`'s own gate/detection logic, not vendor task
completion), and `E1`–`E3` specifically require the real vendor CLI to
actually converse, which the version gate correctly refuses.

### `grok:E1`/`E2`/`E3` — maintainer console evidence (per dispatch)

`D:/ReinstateAcceptanceProjects/grok-manual.txt` appeared during this run's
poll window (checked immediately, then again ~40s and ~60s later; present
by the second check). Per the dispatch, the three rows are scored from it,
quoting only argv and the observed answer lines:

```
grok --version
grok 1.0.5 (5115b46bc9) [stable]

grok -p "Remember this exact token for later: RC6-GROK-4M7Q. Reply with just OK."
OK

rein search RC6-GROK-4M7Q --json
→ session grok:01a07e5d-c05b-7e62-becb-5fcd16e002de found

rein resume "grok:01a07e5d-c05b-7e62-becb-5fcd16e002de" --dry-run --json
→ agent.version status=match actual=1.0.5; decision=confirmation_required

rein resume "grok:01a07e5d-c05b-7e62-becb-5fcd16e002de"
> What was the exact token I asked you to remember?
  RC6-GROK-4M7Q

rein fork "grok:01a07e5d-c05b-7e62-becb-5fcd16e002de" --dry-run --json
→ args: --resume 01a07e5d-c05b-7e62-becb-5fcd16e002de --fork-session

rein fork "grok:01a07e5d-c05b-7e62-becb-5fcd16e002de"
> What was the exact token I asked you to remember?
  RC6-GROK-4M7Q
Resume this session with: grok --resume 01a07e60-d7bf-7611-b29d-1281f790eaf9

rein sessions --agent grok --json
→ ids 01a07e61-f1e6-72d3-8e68-a5c96a3723dc, 01a07e60-d7bf-7611-b29d-1281f790eaf9, 01a07e5d-c05b-7e62-becb-5fcd16e002de all present
```

The transcript shows the recall token correctly returned at both the resume
step and the fork step, and the catalog reflecting both the original and
forked session ids, matching the row's own pass condition exactly. Two
further `rein fork` invocations of the same already-forked session, issued
by the maintainer past the row's own requirement, showed grok-side
instability (one produced no output; the next surfaced `grok native fork
failed: exit status 0xffffffff` from the vendor's own child process exiting
non-zero after "Finishing session…") — recorded as a non-blocking
vendor/host finding below, not as this row's own result; the row's own two
required fork/resume operations both completed cleanly.

**Independent corroboration, this executor's own harness.** Contrary to the
harness assumption this dispatch was written under (`grok` cannot be driven
to a completed conversational turn from this environment), `grok 1.0.13` in
an isolated `GROK_HOME` on this host answered every headless (`-p`) and
`--resume`/`--fork-session` invocation within seconds throughout this run,
with no timeout or hang observed at any point. This executor independently
reproduced the full E1/E2/E3 mechanism before the maintainer transcript
arrived:

```
grok -p "Say hello in one word." → "Hello" (session 01a07e5e-0ffa-7c53-898a-e1502a7fa2cb)
grok --resume 01a07e5e-... -p "Reply with exactly this token and nothing else: RC6B-GROK2-1788826521" → "RC6B-GROK2-1788826521"
grok --resume 01a07e5e-... -p "What was the exact token I asked you to remember? Reply with just the token." → "RC6B-GROK2-1788826521"
grok --resume 01a07e5e-... --fork-session -p "What was the exact token I asked you to remember? Reply with just the token." → "RC6B-GROK2-1788826521" (new session id 01a07e5f-09e0-7d92-ad64-dc68a100d085)
```

This is recorded as a harness/environment observation, not a change to the
row's evidence source — the dispatch's maintainer-transcript evidence above
is what this report scores `grok:E1`/`E2`/`E3` `PASS` on — but it should
inform whether the next candidate's dispatch still needs to route these
rows around the harness.

---

## T5 encrypted sync round trip (beyond the 216; part B)

A disposable `scripts/testing/fakelocker` instance (built from this
worktree, `-accept FAKEKEY`, `-addr 127.0.0.1:9124`) served the S3-compatible
locker. Two fully isolated `REINSTATE_HOME`s, each with its own isolated
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` (never the host's live
homes on either side), were paired against it via `rein init --yes
--endpoint … --bucket … --project claude-proj=… --project codex-proj=…`.
Passphrase input used a small Windows helper built for this run (the same
inheritable-pipe mechanism `scripts/testing/hoplab/secretfd_windows.go`
already uses for `REINSTATE_RECOVERY_CODE_FD`/`REINSTATE_PAIRING_CODE_FD`,
applied here to `REINSTATE_PASSPHRASE_FD`) — no ordinary passphrase
environment value or plaintext file was used.

| Row | Result | Evidence |
| --- | ------ | -------- |
| `push:claude` | PASS | `push --agent claude --all --json` → one snapshot |
| `push:codex` | PASS | `push --agent codex --all --json` → one snapshot |
| `push:opencode` | PASS | `push --agent opencode --all --json` → one snapshot (a freshly-created isolated device-A session) |
| `pull:claude` | PASS | Device B's isolated home restored the pulled file; raw file contains the planted token 9 times |
| `pull:codex` | PASS | Device B's isolated home restored the pulled file; raw file contains the planted token 5 times |
| `pull:opencode` | PASS | Initial restore refused (`compatibility`, `NOT_INSTALLED` — no OpenCode layout yet on device B), then resolved by seeding device B's layout by running OpenCode once locally (schema only, no session content copied); the retry pulled cleanly with zero conflicts; `rein search` on device B's isolated home found the pulled session by its planted token |

**Full isolation on the scored attempt** — no write touched either device's
live agent homes.

### Harness note: `push --all` with no `--agent` filter pushes every configured agent's currently-indexed sessions, not only the T5 three

The first push attempt (before this was noticed) used `push --all` with no
`--agent` filter from a shell where only `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/
`XDG_DATA_HOME` were isolated; `grok`/`qwen`/`gemini`/`kimi`/`pi`/`cline`/
`cursor`/`copilot` were left at their live default homes for that call.
`rein sessions` (no `--agent` filter) in that same shell showed the full,
live, multi-agent catalog, meaning an unscoped `push --all` in that shell
could have bundled live-host session references from agents this row does
not concern into the one encrypted snapshot it produced. The snapshot
content is itself `age`-encrypted (this executor never decrypted it) and
`scripts/testing/fakelocker` persists nothing to disk and was torn down
immediately after, so nothing left the process's memory, but the
**methodology** was wrong: the run was discarded (fresh `fakelocker`
instance, fresh device-A `REINSTATE_HOME`) and redone with `push --agent
claude --all`, `push --agent codex --all`, `push --agent opencode --all`
issued separately, which cannot pull in another agent's live sessions
regardless of what else is on `PATH`/root env. The corrected run is what is
scored above. Recorded so the next executor scopes T5 push calls by
`--agent` from the start rather than relying on `push --all` plus home
isolation alone.

---

## Findings (this part, merged)

| ID | Severity | Row(s) | Description | Release blocking |
| -- | -------- | ------ | ------------ | ----------------- |
| — | host/harness, standing disposition | `codex:E1`, `codex:E2`, `codex:E3` | Live installed Codex CLI (`0.153.4`) drifted past this candidate's verified ceiling (`0.149.0`) mid-cycle; `rein` correctly refuses to build a launch plan (`exit 5`, range named). Recorded `NOT TESTED (version drift)` per the standing rule | Yes, per the standing rule — does not itself excuse the verdict; a widened Codex range needs a `PASS` before stable |
| — | vendor/host, non-blocking | `grok:E3` (maintainer transcript, past the row's own requirement) | A third and fourth `rein fork` against an already-forked `grok` session (issued by the maintainer beyond what the row needed) showed grok-side instability: one produced no output, the next surfaced `grok native fork failed: exit status 0xffffffff` from the vendor child exiting non-zero. `rein` itself surfaced the vendor's own failure correctly rather than hanging or reporting success | No — the row's own two required fork/resume operations both completed cleanly |
| — | harness methodology, corrected in-run, non-blocking | T5 push/pull | An unscoped `push --all` (no `--agent` filter) in a shell that isolated only `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` could bundle other agents' live-host sessions into one encrypted snapshot. No plaintext left the process (fakelocker persists nothing; snapshot content is `age`-encrypted and never decrypted by this executor), but the run was discarded and redone with per-agent `push --agent <a> --all` calls, which is what is scored | No — corrected before scoring; see the harness note above |
| — | harness observation, non-blocking | `handoff … --allow-untested --no-launch --json` (codex source, repeated calls) | Three identical, back-to-back invocations of this exact command against the same session produced two correct JSON refusals (`compatibility`, `environment preflight is blocked`, exit 5) and, on the third call, a silent exit `127`/`-1` with zero stdout and zero stderr — no JSON, no panic trace, no partial output. Not deterministically reproducible in the time available (2 of 3 clean); flagged for the coordinator/next candidate to reproduce and triage — worth checking for a resource leak or race in the `--allow-untested`+`--json`+`--no-launch` code path specifically, since `--dry-run` in place of `--no-launch` reproduced the same silent-crash shape 1 of 1 times it was tried | **Possibly** — a silent crash with no diagnostic output on a real, repeatable-enough command is worth the coordinator's attention even though it did not block scoring any row above (every row's own evidence was obtained on a clean invocation) |
| — | harness clarification, non-blocking | `claude:C4` methodology | `inspect --json`'s `prompt_preview` is a genuinely bounded field (observed cap: 159 characters on a 389-byte prompt), not an unbounded body leak — a first, shorter test looked like it might be returning the full prompt only because that prompt happened to fit entirely under the bound. Recorded so a future executor does not need to re-derive this | No — clarifies methodology only |
| — | product observation, non-blocking | `codex` `handoff` source-layout gate | `codex`'s current session layout (`sessions-rollout-jsonl`, real installed `0.153.4`) is flagged `UNTESTED` by `rein handoff` and requires `--allow-untested` to proceed, independent of and in addition to the separate `agent.version` range gate. Both gates fire correctly and independently; noted for the coordinator since it means two distinct reasons a fresh Codex install can refuse a handoff | No |

## Grok maintainer file

`D:/ReinstateAcceptanceProjects/grok-manual.txt` was checked immediately at
the start of the grok section (absent), then polled again after completing
every other row in this part; it appeared during that poll (present within
the first three 20-second checks, well inside the dispatch's 40-minute
allowance) and its content is scored above.

## Counts (this part)

- Matrix C: 42/42 PASS
- Matrix D: 34/34 PASS (`opencode:D4` excluded, N/A definitional)
- Matrix E: 27/30 PASS, 3 NOT TESTED (version drift) — `codex:E1`/`E2`/`E3`
- T5 sync round trip (beyond the 216): 6/6 PASS

**Part B total: 103 required rows scored, 100 PASS, 3 NOT TESTED (version
drift); 1 N/A (definitional) excluded from the denominator; plus 6/6 PASS on
the T5 sync round trip beyond the 216.**
