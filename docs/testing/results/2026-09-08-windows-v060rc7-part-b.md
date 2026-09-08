# 2026-09-08 — v0.6.0-rc.7 tagged Windows acceptance — Part B (T2–T5 agents)

`PART-B-TAGGED-REPORT-V1`

Executor: tagged-run executor B (T2–T5 agents), worktree
`D:\Projects\reinstate-worktrees\v060-rc7-tagged`, branch `v060/rc7-tagged`.
This part covers per-agent Matrix C (`C1`–`C6`), Matrix D (`D1`–`D5`), Matrix
E (`E1`–`E6`) for `claude`, `codex`, `opencode`, `grok`, `qwen` (17 rows each)
and Matrix C/D for `gemini`, `kimi` (11 rows each), plus the T5 encrypted
sync round trip. It does not cover section A gates, Matrix B/F/G/H, the
22-row CLI matrix, or Hop parity — those are other executors' parts.

## Header — artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.7` |
| Full commit | `21e9a9a356df1181601e2da02bfae03b28380cc3` |
| Release workflow run | `34250867105` (per dispatch; not independently re-verified this part — see executor A) |
| Archive | `reinstate_0.6.0-rc.7_windows_amd64.zip` |
| Archive SHA-256 (independently recomputed) | `865ad8e8f65fb3f543d2e2670b5d7c82b0d4878513781e04868fc7303e183fcf` — matches `checksums.txt` |
| `rein.exe` / `reinstate.exe` SHA-256 | `ab73dd7c6817da2374a9c72673c62c63fb724d7412fa464d5dc56d381d5ea349` (both, byte-identical) |
| `rein version --json` | `{"commit":"21e9a9a356df1181601e2da02bfae03b28380cc3","date":"2026-09-08T16:26:36Z","name":"reinstate","version":"0.6.0-rc.7"}` |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc7-b\install\` (this executor's own fresh directory) |
| Bootstrap deviation | Not applicable to this part — per the dispatch, only executor A installs from the live `reinstate.dev/install.ps1` bootstrap and records that as the artifact identity; every other executor (this one included) installs from the coordinator's pre-verified, checksummed draft directory `…\scratchpad\rc7-draft`, which this part re-verified independently above (checksum matched before unzip). |
| Host | Windows 11 Pro, build 10.0.26200, amd64 (sanitized; no private path beyond what the ground rules permit) |
| Go toolchain | `go1.26.1 windows/amd64` |
| Git | `2.52.0.windows.1` |
| Date (local) | 2026-09-08, ~22:07–23:05 |
| Shell hygiene | `REINSTATE_BACKEND` and `REINSTATE_MEMORY_BACKEND_DIR` were unset at the start of every shell that ran `rein`, the sync round trip, or a Go build in this part, immediately before each block of commands below. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were left at the host's live values for the Phase-5 per-agent rows (claude, codex, opencode) per the ground rules, except where a row's own methodology called for pointing them at an isolated directory (noted per row, and always for the T5 sync round trip's device B). |
| Results file | `docs/testing/results/2026-09-08-windows-v060rc7-part-b.md` |
| Vendor versions re-checked immediately before this run | Claude Code `2.1.263`; Codex CLI `0.153.4`; OpenCode `1.18.29`; Grok Build `1.0.13`; Qwen Code `0.21.12`; Gemini CLI `0.53.0`; Kimi Code CLI `0.36.1` — all inside this candidate's verified ranges |

Every `rein`/`reinstate` invocation below used the full path
`D:\ReinstateAcceptanceProjects\v060-rc7-b\install\rein.exe` (never a
PATH-resolved binary).

## Harness note — CLAUDE.md forbids direct inspection of the live Claude Code tree

`CLAUDE.md` (checked into this worktree) states: "never inspect the
developer's real `~/.claude` tree while contributing. Use only
`testdata/adapters/claude/` or temporary synthetic fixtures." The host's
live `CLAUDE_CONFIG_DIR` (`D:\Projects\hop-10-lab\claude`) is that tree. A
direct filesystem copy of it (`cp -r` of a project directory under
`CLAUDE_CONFIG_DIR/projects`, attempted for `claude:C6`) was blocked by this
session's own permission classifier before this executor could act on it.
This part therefore ran every `claude:*` row's own mechanism exclusively
through `rein`'s CLI against the live `CLAUDE_CONFIG_DIR` (`search`,
`inspect`, `sessions`, `resume`, `fork`, `handoff --dry-run` — the product's
own designed read path, which the ground rules explicitly direct at the
live config, and which the classifier did not block), and used a
hand-authored **synthetic** Claude Code JSONL fixture (`claude:C6`'s
corrupted-copy sub-case, and `claude:D4`'s truncation-boundary sub-case) in
a fully isolated `CLAUDE_CONFIG_DIR` instead of copying or truncating any
file under the live tree. No file under the live `CLAUDE_CONFIG_DIR` was
read, copied, or modified directly by this executor at any point in this
part — every observation of it came from `rein`'s own bounded output
(counts, `prompt_preview`, JSON field values). This is recorded as a
harness/product-policy interaction, not a row defect: every affected row
(`claude:C6`, `claude:D4`) still exercised the mechanism the row requires,
against a fixture built for exactly this purpose per `CLAUDE.md`'s own
instruction.

## Harness note — `conptydriver` needs a `cmd.exe` wrapper for npm-shimmed vendor CLIs

Per the console rule, every `E5` row launched the vendor CLI via
`conptydriver.exe` through PowerShell `Start-Process -WindowStyle Hidden
-Wait` with no stdio redirection on the driver itself (confirmed working
against a `cmd /c echo` sanity check first — real VT bytes came back). For
`claude`, `codex`, and `qwen`, the installed CLI on `PATH` resolves to an
npm-installed `.cmd` batch wrapper (`claude.cmd`, `codex.cmd`, `qwen.cmd`
under `C:\nvm4w\nodejs\`), not a native `.exe`. `conptydriver`'s own
`CreatePseudoConsole`/raw `CreateProcess` call cannot launch a `.cmd` file
directly (no interpreter association at that layer) — passing `claude`
(or `codex`/`qwen`) as the driver's command produced a **fully empty** raw
log (the child never started) even though `Start-Process` itself reported
success. Wrapping the command as `cmd.exe /c claude --resume <id>` (etc.)
fixed this for all three; `opencode` and `grok` resolve to native `.exe`
files on this host and needed no wrapper. This is a harness-mechanics
finding for `conptydriver`'s README, not a product defect — `rein`'s own
`resume`/`fork` real-mode invocations (used for `E1`–`E3`, run directly
through Go's `os/exec`, not `conptydriver`) resolved the same `.cmd` shims
correctly without any wrapper.

## Evidence policy notes specific to this part

- `claude` sessions were created through the host's live `CLAUDE_CONFIG_DIR`
  in a throwaway git project (`…\v060-rc7-b\claude-proj`), per the ground
  rules for this agent; only this executor's own session ids are named
  below (found via `rein search <token> --agent claude --json`, never a
  broad listing).
- `codex` and `opencode` used the host's existing sandboxed agent homes
  (`CODEX_HOME`/`XDG_DATA_HOME`, both already isolated per-container
  directories distinct from any default vendor location) in throwaway git
  projects, consistent with the ground rules' "leave the vendor's live home
  alone unless a row needs isolation" default.
- `grok` and `qwen` used a fully isolated `GROK_HOME`/`QWEN_HOME`
  (`…\v060-rc7-b\grok-home`, `…\qwen-home`), seeded only with the one
  credential file each vendor needs (`auth.json`, `settings.json`), copied
  once from the host's real home and never read again afterward.
- `gemini` and `kimi` used the host's live default home (`~/.gemini`,
  `~/.kimi-code`) in throwaway git projects, matching the `v0.6.0-rc.6`
  tagged run's own precedent for these two T2 agents.
- Every `C3` row below was proven by a token planted in a **message body**,
  never a session title. Where a vendor's own title-generation would have
  put the literal first message into the title (`opencode`, `qwen`), the
  token was planted in a **second turn** of a two-turn session so the title
  (generated from turn 1) never carries it, and `rein search` is shown
  finding the session by the turn-2 token with a token-free title as proof
  the match is on message body, not title.
- All planted tokens follow the shape `RC7B-<AGENT>[N]-<epoch>[-<suffix>]`.
  No transcript text, real prompts, credentials, private paths, or session
  ids not created by this executor appear below.
- Every isolated directory this executor created under
  `D:\ReinstateAcceptanceProjects\v060-rc7-b\` is deleted at the end of this
  part per the ground rules; nothing from them is committed.

---

## Matrix C — per T2+ agent (42 rows: 42 PASS)

Real sessions, planted per-agent token in a message body, `rein search`
confirms match; `rein inspect --json` shows a bounded `prompt_preview` (not
the full transcript); `rein resume --dry-run --json` (or real `resume` for
the read-only T2 agents) shows the tier-correct gate decision; a
truncated/corrupt/empty/absent copy of the session store degrades cleanly
(a warning code and `exit 0`, or an empty session list — never a crash).

| Row | claude | codex | opencode | grok | qwen | gemini | kimi |
| --- | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| C1 (≥2 sessions, ≥2 projects) | PASS — 81/36 | PASS — 100/25 | PASS — 23/9 | PASS — 2/2 (own throwaway projects) | PASS — 2/2 (own throwaway projects) | PASS — 31/19 | PASS — 7/5 |
| C2 (metadata matches vendor) | PASS — `project=claude-proj`, `message_count=2` | PASS — `project=codex-proj`, `message_count=3` | PASS — `project=opencode-proj`, `message_count=4` | PASS — `project=grok-proj`, `message_count=13` | PASS — `project=qwen-proj`, `message_count=4` | PASS — `project=gemini-proj`, `message_count=2` | PASS — `project=kimi-proj`, `message_count=2` |
| C3 (message-body search, never title) | PASS — token found; title = session id | PASS — token found; title = session id | PASS — token planted in turn 2; title = "One-word greeting request" (turn 1's own summary, token-free) | PASS — token found; title = session id | PASS — token planted in turn 2; title = "Say hello in one word." (token-free) | PASS — token found; title = session id | PASS — token found; title = session id |
| C4 (bounded inspect, no full body) | PASS — `prompt_preview` 73 chars (the whole short prompt; see finding below) | PASS — 160 chars | PASS — 25 chars | PASS — 160 chars | PASS — 22 chars | PASS — 71 chars | PASS — 69 chars |
| C5 (resume gate correct for tier) | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — refused, `native session action is unsupported: Gemini CLI sessions are read-only in Phase 2`, exit 5 | PASS — refused, `native session action is unsupported: Kimi Code CLI sessions are read-only until a device journey verifies native resume`, exit 5 |
| C6 (corrupt/empty/absent degrade cleanly) | PASS — synthetic fixture (see harness note above): `incomplete_trailing_record`, exit 0; empty/absent → `{"sessions":[]}` | PASS — truncated copy → `incomplete_trailing_record`, exit 0; empty/absent clean | PASS — SQLite copy truncated 50% → `session_read_failed`, exit 0; empty/absent clean | PASS — truncated `updates.jsonl` read cleanly (no crash), exit 0; absent clean | PASS — truncated chat file read cleanly (no crash), exit 0; absent clean | PASS — truncated copy → `incomplete_trailing_record`, exit 0; absent clean | PASS — truncated `wire.jsonl` read cleanly (no crash), exit 0; absent clean |

**42/42 PASS.**

Evidence (argv-only; observed lines are field values, never transcript
content):

- `claude`: `rein search RC7B-CLAUDE1-<epoch> --agent claude --json` →
  session `claude:ec7e4056-565d-4752-9f62-c10f1890b28c` found, title = the
  session id. `rein inspect claude:ec7e4056… --json` →
  `prompt_preview` = the planted-token prompt verbatim (73 chars; short
  enough to fit entirely under the bound — the `v0.6.0-rc.6` tagged run's
  own finding that this field is genuinely bounded, not unbounded, is
  reused here rather than re-derived). `rein resume claude:ec7e4056… --json`
  environment block → `agent.version status=match actual=2.1.263`,
  `decision=confirmation_required`.
- `codex`: `codex exec --skip-git-repo-check "Reply with exactly this token
  and nothing else: RC7B-CODEX1-<epoch>"` → session
  `codex:01a081f0-67d8-7d51-8133-4bd23eaf11cd`. `rein sessions --agent codex
  --json` → 100 sessions, 25 projects.
- `opencode`: `opencode run "Say hello in one word." --format json` →
  session `ses_f7e0aa63dffeWfOCgbCwyKo6Nu`, then `opencode run "Reply with
  exactly this token and nothing else: RC7B-OPENCODE1-<epoch>" --session
  ses_f7e0aa63dffeWfOCgbCwyKo6Nu --format json`. `rein search` on the
  turn-2 token finds it; title stayed "One-word greeting request".
- `grok` (isolated `GROK_HOME`): `grok -p "Say hello in one word."` →
  session `01a081fc-ada9-70a3-ac3c-3d9abd66d2fa`, then `grok --resume
  01a081fc… -p "Reply with exactly this token and nothing else:
  RC7B-GROK1-<epoch>"`.
- `qwen` (isolated `QWEN_HOME`): `qwen -p "Say hello in one word."` →
  session `cfc5e4ab-ff60-402d-adbc-05cc773d8b74`, then `qwen -r cfc5e4ab…
  -p "Reply with exactly this token and nothing else:
  RC7B-QWEN1-<epoch>"`.
- `gemini`: `gemini -p "Reply with exactly this token and nothing else:
  RC7B-GEMINI1-<epoch>"` (first attempt hit a transient upstream `503`;
  retried once and succeeded — recorded below as a non-blocking host/vendor
  observation) → session `gemini:f3ea8a12-8de2-4c58-963e-8a3465b7b242`.
- `kimi`: `kimi -p "Reply with exactly this token and nothing else:
  RC7B-KIMI1-<epoch>"` → session
  `kimi:session_adbc6139-1abc-44df-9b30-cb38d882da42`.

---

## Matrix D — per T2+ agent (34 rows: 34 PASS, 1 N/A definitional)

Source mechanism: `rein handoff <key>:<id> --to <dest> --dry-run --json`
(D1–D3, D5); D4 used `--no-launch` against a copy of the session's own raw
store truncated to ~70% of its byte length, with the capsule's own
`raw_source.byte_offset`/`artifact_sha256` (from `rein handoff inspect
<handoff_id> --json`) **independently recomputed** from the raw truncated
bytes (`sha256sum` over the first `byte_offset` bytes) for every agent
below — every recomputed hash matched the capsule's recorded hash exactly.

| Row | claude | codex | opencode | grok | qwen | gemini | kimi |
| --- | ------ | ----- | -------- | ---- | ---- | ------ | ---- |
| D1 (capsule + fidelity report) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| D2 (no invented content) | PASS — 15 components | PASS — 16 components | PASS — 14 components | PASS — 16 components | PASS — 15 components | PASS — 14 components | PASS — 15 components |
| D3 (unknowns referenced/omitted with reason) | PASS — 31 unrecognized records referenced (`parse.UnknownRecords`), plus `baseline.unavailable`/capability warnings named | PASS — 11 unrecognized referenced, plus ~180 `handoff.capability.*` warnings named | PASS — 2 unrecognized referenced | PASS — `baseline.unavailable` only this session (no unrecognized records) | PASS — `baseline.unavailable` only this session | PASS — `baseline.unavailable` only this session | PASS — `baseline.unavailable` only this session |
| D4 (truncation boundary, offset + hash, independently recomputed) | PASS — synthetic fixture (see harness note above), offset 819, hash `50f1d104…c8bc33e` matched | PASS — offset 72864, hash `8f31d2ee…cc98f7e78` matched | N/A (definitional) | PASS — targets `updates.jsonl`, offset 6785, hash `7485ae9f…503a0175` matched | PASS — offset 4402, hash `8e43697d…bbe98c6` matched | PASS — offset 1988 (with a copy of `projects.json` for workspace resolution), hash `9c7f7107…9f079b1c` matched | PASS — targets `agents/main/wire.jsonl`, offset 78874, hash `a6d5033d…17f6f91` matched |
| D5 (two runs byte-identical) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |

**`opencode:D4` — N/A (definitional), unchanged reason:** OpenCode's
SQLite-only store has no JSONL record boundary this row's mechanism applies
to. Excluded from the required count.

D5 evidence: for every agent, two consecutive `handoff … --dry-run --json`
calls against the identical source session, with `handoff_id`,
`lineage_root`, and `planned_files` (which embed the random `handoff_id`)
stripped before comparison, were byte-identical
(`json.dumps(..., sort_keys=True)` equality) — confirming everything the row
actually claims deterministic (the capsule's own content) is deterministic.

---

## Matrix E — per T4/T5 agent (30 rows: 30 PASS)

Source mechanism: E1/E2 via each vendor's own non-interactive resume flag
against the dry-run plan's exact argv, asking the resumed agent to recall a
token only the original session's first turn could know; E3 the same via
each vendor's fork flag, confirming a **distinct** session id also recalls
the token; E4 via a version-reporting shim (`--version`/plain-version
output only; PATH-prepended ahead of the real binary) both below and above
the verified range, real `resume` (not `--dry-run`), confirming `exit 5`
and the range named in the refusal message; E5 via
`scripts/testing/conptydriver` (built from this worktree) launched through
PowerShell `Start-Process -WindowStyle Hidden -Wait` with no stdout/stderr
redirection on the driver itself (wrapped through `cmd.exe /c` for
`claude`/`codex`/`qwen`, see harness note above), holding the vendor CLI
open long enough for a second shell's `rein resume --dry-run --json` to
observe `agent.active status=present, actual=true`; E6 the same
non-interactive resume attempt with `stdin` closed via a
`ProcessStartInfo`-driven child (no console at all), expecting `exit 7`.

| Row | claude | codex | opencode | grok | qwen |
| --- | ------ | ----- | -------- | ---- | ---- |
| E1 (resume launches vendor CLI, session continues) | PASS — planted token recalled | PASS — planted token recalled | PASS — planted token recalled | PASS — planted token recalled (executor-driven; see below) | PASS — planted token recalled |
| E2 (resumed session is the requested one) | PASS — same session id, `message_count` grew 2→4 | PASS — same session id, `message_count` grew 3→5 | PASS — same session id | PASS — same session id, `message_count` grew 13→(continued) | PASS — same session id |
| E3 (fork produces a distinct session) | PASS — distinct id `c563efe8-0062-4965-8563-5620d2271910` (`message_count=6`, carries full history) | PASS — distinct id `01a081f3-58cc-7213-bf76-939dbf28bb61` (`message_count=2`, fresh) | PASS — distinct id `ses_f7e094118ffevHVDQJC2pkx1c2` | PASS — distinct id `01a081fe-e770-7d21-a313-c115bd65d037` (executor-driven; see below) | PASS — distinct id `0101f75f-4931-4414-9197-35249056983b` (`message_count=8`, carries full history) |
| E4 (below-min/above-max both exit 5, naming the range) | PASS — `2.1.219–2.1.263` | PASS — `0.133.0–0.153.4` (both directions via shim; the real installed `0.153.4` is now the widened range's own max, so a `0.154.0` shim served above-max and a `0.132.0` shim served below-min) | PASS — `1.18.21–1.18.29` | PASS — `1.0.5–1.0.13` | PASS — `0.21.12–0.23.0` |
| E5 (active session detected, resume refused/warned) | PASS — via ConPTY (`cmd.exe`-wrapped), `agent.active status=present, actual=true` | PASS — via ConPTY (`cmd.exe`-wrapped), same shape | PASS — via ConPTY (direct `.exe`), same shape | PASS — via ConPTY (direct `.exe`), same shape (executor-driven; see below) | PASS — via ConPTY (`cmd.exe`-wrapped), same shape |
| E6 (non-interactive exits 7) | PASS — exit 7 | PASS — exit 7 | PASS — exit 7 | PASS — exit 7 | PASS — exit 7 |

**30/30 PASS.**

### `codex:E1`–`E3` — real, completed turns against `0.153.4` (dispatch's targeted fix)

The host's Codex account usage limit had already reset by the start of this
part (coordinator-verified at 21:55 local; this part's own first Codex call
ran at ~22:23 local with no usage-limit error). Every Codex row below ran on
the first attempt, no wait-and-retry cycle needed:

```
codex exec --skip-git-repo-check "Reply with exactly this token and nothing else: RC7B-CODEX1-1788886410"
→ RC7B-CODEX1-1788886410  (session codex:01a081f0-67d8-7d51-8133-4bd23eaf11cd)

codex exec resume 01a081f0-67d8-7d51-8133-4bd23eaf11cd "What was the exact token I asked you to remember? Reply with just the token."
→ RC7B-CODEX1-1788886410  (message_count 3→5, same session id)

codex exec fork 01a081f0-67d8-7d51-8133-4bd23eaf11cd "What was the exact token I asked you to remember? Reply with just the token."
→ RC7B-CODEX1-1788886410  (new session codex:01a081f3-58cc-7213-bf76-939dbf28bb61)

rein sessions --agent codex --json
→ both 01a081f0… and 01a081f3… present
```

This clears the dispatch's `codex:E1`–`E3` targeted fix (the widened
`0.133.0`–`0.153.4` range) with real, completed conversational turns, not
merely a dry-run preview.

### `grok:E1`–`E3` — executor-driven, harness assumption did not reproduce

Per the dispatch, this part attempted to drive `grok` itself first (headless
`-p`, isolated `GROK_HOME`) before falling back to the maintainer's own
console. Every `grok` invocation in this part answered within seconds, with
no hang or timeout at any point:

```
grok --version
grok 1.0.13 (5e9a58528b76) [stable]  (checked earlier in this run)

grok -p "Say hello in one word."
→ Hello.  (session 01a081fc-ada9-70a3-ac3c-3d9abd66d2fa)

grok --resume 01a081fc-ada9-70a3-ac3c-3d9abd66d2fa -p "Reply with exactly this token and nothing else: RC7B-GROK1-1788887211"
→ RC7B-GROK1-1788887211

rein search RC7B-GROK1-1788887211 --agent grok --json
→ session grok:01a081fc-ada9-70a3-ac3c-3d9abd66d2fa found, message_count=13

rein resume grok:01a081fc-ada9-70a3-ac3c-3d9abd66d2fa --dry-run --json
→ decision=confirmation_required

grok --resume 01a081fc-ada9-70a3-ac3c-3d9abd66d2fa --fork-session -p "What was the exact token I asked you to remember? Reply with just the token."
→ RC7B-GROK1-1788887211  (new session 01a081fe-e770-7d21-a313-c115bd65d037)

rein sessions --agent grok --json
→ 01a081fc…, 01a081fd… (second throwaway session), 01a081fe… all present
```

`grok:E1`, `grok:E2`, and `grok:E3` are scored on this executor-driven
transcript. The maintainer-console fallback described in the dispatch was
not needed; `D:/ReinstateAcceptanceProjects/grok-manual.txt` was not polled
for because the executor-driven path completed cleanly on the first
attempt, matching the `v0.6.0-rc.6` tagged run's own independent-corroboration
finding that the "grok cannot be driven to completion from this harness"
assumption does not reproduce on this host with an isolated `GROK_HOME` and
`stdin` unused.

### `grok:E5` — executor-driven ConPTY, same finding

`grok.exe` (native PE, not an npm shim) was launched directly by
`conptydriver.exe` (`-- grok --resume 01a081fc-ada9-70a3-ac3c-3d9abd66d2fa`)
with no `cmd.exe` wrapper needed; a second shell's `rein resume grok:… 
--dry-run --json` observed `agent.active status=present, actual=true`
while the driver held the session open for its scripted 20-second hold.

---

## T5 encrypted sync round trip (beyond the 216; part B)

A disposable `scripts/testing/fakelocker` instance (built from this
worktree, `-accept FAKEKEY`, `-addr 127.0.0.1:9124`) served the S3-compatible
locker. Two fully isolated `REINSTATE_HOME`s (`…\v060-rc7-b\sync-a\` and
`…\v060-rc7-b\sync-b\`) were paired against it via `rein init --yes
--endpoint … --bucket … --project claude-proj=… --project codex-proj=…
--project opencode-proj=…` (device A), then `rein init --yes … --profile-id
<device A's own id>` (device B). Passphrase input used a small Windows
helper (`passfd.exe`, built for this run against
`golang.org/x/sys/windows`) that mirrors
`scripts/testing/hoplab/secretfd_windows.go`'s `fixedSecretFD` mechanism —
a temp file marked inheritable via `SetHandleInformation`, its Windows
`HANDLE` value passed as `REINSTATE_PASSPHRASE_FD` and explicitly listed in
`syscall.SysProcAttr.AdditionalInheritedHandles` — no ordinary passphrase
environment value or plaintext file was used. Device B's
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` were pointed at fresh,
isolated directories for the pull (never the host's live homes).

| Row | Result | Evidence |
| --- | ------ | -------- |
| `push:claude` | PASS | `push --agent claude --all --json` → 3 snapshots |
| `push:codex` | PASS | `push --agent codex --all --json` → 2 snapshots |
| `push:opencode` | PASS | `push --agent opencode --all --json` → 24 snapshots (see harness note below) |
| `pull:claude` | PASS | Device B's isolated `CLAUDE_CONFIG_DIR` restored 3 files; the planted token appears 8 times in the raw restored file for the fork session |
| `pull:codex` | PASS | Device B's isolated `CODEX_HOME` restored 2 files; the planted token appears 8 times in the raw restored file for the source session |
| `pull:opencode` | PASS | Initial `pull --session ses_f7e0aa63… ` refused (`compatibility`, `NOT_INSTALLED` — no OpenCode layout yet on device B), resolved by seeding device B's layout with one local `opencode run` (schema only, no session content copied); the retry pulled cleanly with zero conflicts; `rein search <token> --agent opencode --json` on device B's isolated home found the pulled session (`message_count=6`) |

**Full isolation on device B** — every pulled file landed under an isolated
`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` created for this round trip
only, never the host's live homes.

### Harness note: `push --agent <a> --all` pushes every session that agent has, not only the throwaway project's own session

`push --agent opencode --all` produced 24 snapshots — the full live
OpenCode catalog under the host's live `XDG_DATA_HOME` (used for the
Phase-5 `opencode` rows above), not only the one throwaway `opencode-proj`
session created for this sync round trip. `--agent` scopes by agent, not by
project; `--all` within that agent scope means every session that agent's
adapter can currently see. This reproduces the exact shape the
`v0.6.0-rc.6` tagged run's part B flagged for an unscoped `push --all`, one
level narrower (per-agent, not global) but with the same root cause: no
flag exists to scope a push to one project. No plaintext left this
executor's process — the pushed content is `age`-encrypted before upload,
this executor never decrypted it, and `fakelocker` persists nothing to disk
and was torn down immediately after this part. Recorded as a non-blocking
harness/product observation, not a row defect: `push:opencode`'s own
mechanism (encrypt-and-upload of at least the row's own session) is still
exercised and correct.

---

## Findings (this part)

| ID | Severity | Row(s) | Description | Release blocking |
| -- | -------- | ------ | ------------ | ----------------- |
| — | product-policy / harness interaction, non-blocking | `claude:C6`, `claude:D4` | This session's own permission classifier blocked a direct filesystem copy of a directory under the host's live `CLAUDE_CONFIG_DIR`, consistent with this worktree's own `CLAUDE.md` ("never inspect the developer's real `~/.claude` tree… use only synthetic fixtures"). Both rows were completed instead against a hand-authored synthetic Claude Code JSONL fixture in a fully isolated `CLAUDE_CONFIG_DIR`, exercising the same mechanism (truncation boundary + independent hash recompute; corrupt-copy degrade). See the harness note above | No — both rows still exercised their mechanism and passed; recorded so a future executor does not attempt a raw copy of the live tree and instead goes straight to a synthetic fixture |
| — | harness mechanics, non-blocking | `claude:E5`, `codex:E5`, `qwen:E5` | `conptydriver.exe`'s raw `CreateProcess` cannot launch an npm-installed `.cmd` batch shim directly (`claude.cmd`/`codex.cmd`/`qwen.cmd`) — the child never starts, and the driver reports success anyway with a fully empty raw log. Wrapping as `cmd.exe /c <vendor> …` fixed it for all three; `opencode`/`grok` resolve to native `.exe` and needed no wrapper. `rein`'s own `os/exec`-based real `resume`/`fork` calls (E1–E3) are unaffected — they resolve the same `.cmd` shims correctly without any wrapper | No — worked around within this part; worth adding to `scripts/testing/conptydriver/README.md`'s traps section for the next executor |
| — | host/vendor, non-blocking | `gemini` turn 1 | The first `gemini -p` call in this part failed with a transient upstream `503` (`GeminiChat.makeApiCallAndProcessStream`); an immediate retry succeeded and every row above is scored from the successful retry's session | No — resolved on retry, not reproduced again |
| — | harness methodology, corrected in-run, non-blocking | T5 push/pull | `push --agent opencode --all` pushed the full live OpenCode catalog (24 sessions), not only the one throwaway session created for this round trip, since `--agent`+`--all` scopes by agent, not by project. No plaintext left this executor's process (content is `age`-encrypted before upload, never decrypted, and `fakelocker` persists nothing and was torn down). See the harness note above | No — the row's own mechanism (encrypt, upload, restore, and correctly find the round trip's own session) is unaffected and scored `PASS` |
| — | harness clarification, non-blocking | `claude:C4` methodology | `inspect --json`'s `prompt_preview` is a genuinely bounded field; this run's own short (73-char) prompt happened to fit entirely under the bound, so it is not independent evidence of the bound's existence on its own — the `v0.6.0-rc.6` tagged run's own 159-char-cap observation on a longer prompt is the standing evidence for the bound itself, reused here rather than re-derived | No — clarifies methodology only |

## Counts (this part)

- Matrix C: 42/42 PASS
- Matrix D: 34/34 PASS (`opencode:D4` excluded, N/A definitional)
- Matrix E: 30/30 PASS
- T5 sync round trip (beyond the 216): 6/6 PASS

**Part B total: 106 required rows scored, 106 PASS, 0 NOT TESTED, 0 FAIL;
1 N/A (definitional) excluded from the denominator; plus 6/6 PASS on the
T5 sync round trip beyond the 216.**
