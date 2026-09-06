# `v0.6.0-rc.1` pre-tag native Windows matrix — pass 2, executor D (Matrix E)

`PHASE5-DEVICE-REPORT-V1` (partial) — **W7 second-pass executor D**, Matrix E
(per-T3-agent physical resume rows), `claude` / `codex` / `opencode` / `grok`
/ `qwen`, 30 rows (`E1`–`E6` × 5 agents). This is a targeted re-run: the
first pass (`docs/testing/results/2026-09-06-windows-v060rc1-pretag.md`,
Part B §3) recorded every Matrix E row `NOT TESTED` except `E6`, disclosing
that the physical-journey work did not fit inside that pass's time budget.
This report performs those physical journeys against the same artifact
identity and either closes each row with a real result or names exactly why
it could not be closed.

Contract:
[`docs/testing/v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md)
§B, composing
[`phase-5-universal-agent-coverage-acceptance.md`](../phase-5-universal-agent-coverage-acceptance.md)
Matrix E. Prior work read: `2026-09-06-windows-v060rc1-pretag.md` (Section 5
"not tested and why", F1/F2/RB findings, Part B/Part C evidence),
`2026-09-06-windows-range-widening-v060.md` (the isolated-config-dir method
for a real resume), `phase-5-universal-agent-coverage-acceptance.md`
Matrices C/D/E, `windows-acceptance-host.md` (ConPTY driver, WMI-adjacent
host notes), `scripts/testing/conptydriver/README.md`.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Worktree | `D:\Projects\reinstate-worktrees\v060-w7c-rerun`, branch `v060/w7c-rerun` at `86cb3421` (identical to `release/v0.6.0-rc.1` at that commit) |
| Commit range read | `git log --oneline 57c15d52..86cb3421` — carries the checklist/wizard rune-space fix (`internal/tui`), the passphrase descriptor bound, the credentials test, gitleaks config allowlist, and docs, on top of the first pass's `57c15d52` |
| Archive | `reinstate_0.0.0-86cb3421_windows_amd64.zip`, sha256 `00f20f21163e54a94898d654f75731caf88844917ff7538bab4db811e9d90bd0` — verified against `checksums.txt` beside it and by independent `sha256sum` in this session |
| Install dir | `D:\ReinstateAcceptanceProjects\v060-w7c-d\install\` (fresh, unzipped in this session) |
| `rein.exe` / `reinstate.exe` | byte-identical (`sha256sum` match: `d6a4d09a...b9d951e`) |
| `rein version --json` | `{"commit":"86cb34212a3dbd6241608595124e82e9110c78a3","date":"2026-09-06T03:52:33Z","name":"reinstate","version":"0.0.0-86cb3421"}` |
| `REINSTATE_HOME` | isolated: `D:\ReinstateAcceptanceProjects\v060-w7c-d\reinhome` (fresh index, never the developer's real home) |
| Host OS | Windows NT `10.0.26200.0` (native `windows/amd64`, not WSL) |
| Git | `git version 2.52.0.windows.1` |
| Go | `go1.26.1` (host default; not used to build anything for this report — the artifact under test is the staged snapshot, never a locally built binary) |
| UTC date | 2026-09-06 |
| Every shell | ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR XDG_DATA_HOME CLAUDE_CONFIG_DIR CODEX_HOME` before the first `rein`/vendor invocation, then set only the one isolated root variable each agent's block below needed |

Vendor versions re-checked immediately before use (dispatch's last-checked
values in parentheses): Claude Code `2.1.263` (`2.1.263`, unchanged), Codex
`0.149.0` (`0.149.0`, unchanged), OpenCode `1.18.27` (`1.18.27`, unchanged),
Grok Build `1.0.5` (`1.0.5`, unchanged), Qwen Code `0.21.12` (dispatch named
`0.21.13`; this is the same drift the first pass's harness findings §4
already recorded — immaterial to the fixture rows there, material here,
see `E2:qwen`/`E3:qwen` below).

No binary was built for any row. All five version-boundary rows (`E4`) used
a `.cmd` shim placed earlier on `PATH` that answers only `--version` with a
fabricated string and refuses every other argument — modeled on
`2026-08-21-windows-phase5-V050RC6.md`'s `E4` method and
`scripts/tuisandbox`'s fake-vendor pattern — never a rebuilt `rein` and never
a modified real vendor install.

## Verdict for this executor's 30 rows

| | Count |
| - | ----- |
| PASS | 20 |
| FAIL | 8 |
| PARTIAL | 1 |
| NOT TESTED | 1 |

**Two findings below are flagged as candidate release blockers** — see
[Release-blocking findings](#release-blocking-findings). Every other FAIL is
the same one root cause (host WMI access denial; see `E5` below), disclosed
in detail once and not repeated as five separate mysteries.

## Row table

| Row | Result | Evidence (short) |
| --- | ------ | ----------------- |
| `E1:claude` | FAIL | dry-run blocked, exit 5, `compatibility` — see below |
| `E1:codex` | PASS | session indexed; dry-run plan `resume <id>`, exit 0 |
| `E1:opencode` | PASS | session indexed; dry-run plan `--session <id>`, exit 0 |
| `E1:grok` | PASS | session indexed; dry-run plan `--resume <id>`, exit 0 |
| `E1:qwen` | PASS | session indexed; dry-run plan `--resume <id>`, exit 0 |
| `E2:claude` | FAIL | real resume never launches through `rein resume` (blocked); supplementary evidence below |
| `E2:codex` | PASS | `codex exec resume <id> "…"` returned the planted token |
| `E2:opencode` | PASS | `opencode run --session <id> "…"` returned the planted token |
| `E2:grok` | PASS | `grok --resume <id> -p "…"` returned the planted token |
| `E2:qwen` | NOT TESTED | vendor's own account credential is expired/invalid on this host (401), reproduced with the live value read from the real config — unrelated to Reinstate or isolation |
| `E3:claude` | FAIL | `rein fork` dry-run also blocked, same root cause as `E1`; supplementary evidence below |
| `E3:codex` | PASS | `codex exec fork <id> "…"` returned a new session id with the planted token |
| `E3:opencode` | PASS | `opencode run --session <id> --fork "…"` returned a new session id with the planted token |
| `E3:grok` | PASS | `grok --resume <id> --fork-session -p "…"` returned a new session id with the planted token (vendor process needed a timeout-kill after printing the correct result — see harness note) |
| `E3:qwen` | PARTIAL | `--fork-session` produced a genuinely distinct, vendor-recognized session file; content-inheritance unverified — same expired credential as `E2:qwen` |
| `E4:claude` | PASS | `2.1.218`: exit 5, range named; `2.1.262`: exit 5, range named |
| `E4:codex` | PASS | `0.132.0`: exit 5, range named; `0.150.0`: exit 5, range named |
| `E4:opencode` | PASS | `1.18.20`: exit 5, range named; `1.18.28`: exit 5, range named |
| `E4:grok` | PASS | `1.0.4`: exit 5, range named; `1.0.6`: exit 5, range named |
| `E4:qwen` | PASS | `0.21.11`: exit 5, range named; `0.21.14`: exit 5, range named |
| `E5:claude` | FAIL | active detection did not fire during a real, confirmed-running session; see WMI finding |
| `E5:codex` | FAIL | same |
| `E5:opencode` | FAIL | same |
| `E5:grok` | FAIL | same |
| `E5:qwen` | FAIL | same |
| `E6:claude` | PASS | non-interactive resume, exit 7, `environment warnings require confirmation: baseline.unavailable` |
| `E6:codex` | PASS | exit 7, same message shape |
| `E6:opencode` | PASS | exit 7, same message shape |
| `E6:grok` | PASS | exit 7, `…: baseline.unavailable, git.working_tree` |
| `E6:qwen` | PASS | exit 7, same message shape |

## Method, once, for all five agents

Per agent: a throwaway git repository under
`D:\ReinstateAcceptanceProjects\v060-w7c-d\<agent>\project\`, and an isolated
agent root (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`, `GROK_HOME`,
`QWEN_HOME` respectively — the exact variable each agent's catalog entry
declares as `RootEnv`) under
`D:\ReinstateAcceptanceProjects\v060-w7c-d\<agent>\home\`, seeded with
**only** that vendor's own credential file copied from the host's real
default location, named once and never otherwise inspected:

- `claude`: `.credentials.json`
- `codex`: `auth.json`
- `opencode`: `auth.json` (under `<XDG_DATA_HOME>\opencode\`)
- `grok`: `auth.json`
- `qwen`: `settings.json` (the field `env.BAILIAN_CODING_PLAN_API_KEY` inside
  it is qwen's credential; no separate OAuth-token file exists for this
  vendor)

No session file, transcript, or other real-tree content was copied for any
agent. Each isolated home was used to run the real, already-installed
vendor CLI with a freshly planted, single-use marker token (never a real
prompt/response), then `rein sessions --agent <key> --json` against the
same isolated `REINSTATE_HOME` indexed exactly the one session created.
Real resume/fork used the vendor's own non-interactive completion form —
`claude -p … --output-format json`, `codex exec resume/fork … "…"`,
`opencode run --session … --format json`, `grok -p … --output-format
json` — with the exact arguments `rein resume --dry-run --json`'s launch
plan named, per the fallback the range-widening report used ("the plan's
argv plus that form") for the same reason it did: this Bash-tool session
has no console of its own, and driving `conptydriver.exe` needs one
launched without its own stdio redirected (`windows-acceptance-host.md`,
"A trap in how `conptydriver` itself must be launched") — out of reach from
inside this harness. Every vendor here has a documented non-interactive
resume form, which the contract accepts as the alternative to ConPTY.

## `E1`/`E2` — `claude`

**Real environment: FAIL, both rows.** This candidate's `claude` verified
ceiling is `2.1.219` to `2.1.261` inclusive (widened from `2.1.238` the same
day by `v060/w3-ranges`, merged into this release branch). The host's real,
already-installed Claude Code has since auto-updated to `2.1.263` — two
patch versions past a ceiling that was current as of this same day's earlier
work. A session was created normally (isolated `CLAUDE_CONFIG_DIR`, real
`claude -p "Reply with exactly this text and nothing else: claude-probe-
token-E1E2" --output-format json`; session id `0e6586d4-3739-47cd-bad3-
395939e10d53`; vendor answered with the exact token) and indexed correctly
(`rein sessions --agent claude --json` showed exactly that one session).
`rein resume claude:0e6586d4-… --dry-run --json` then exited **5**
(`"code": "compatibility"`), with `agent.version` at `severity: block`:
`"native agent version 2.1.263 is outside the verified range 2.1.219 to
2.1.261 inclusive"`. `internal/preflight/policy.go`'s `Authorize` refuses
unconditionally whenever `report.Decision == DecisionBlocked` — no
`--allow-environment-warning` value, no flag, and (by code inspection) no
interactive confirmation bypasses a `block`-severity check. A real,
un-augmented `rein resume`/`rein fork` for `claude` on this exact host,
right now, **never launches the vendor at all**, for anyone, regardless of
how the invocation is made. This is disclosed as a release-blocker
candidate — see below — not because the refusal is wrong (fail-closed on an
unverified version is the documented design), but because the range this
candidate ships already trails the vendor's real shipped build by the time
of this pass.

**Supplementary evidence (not counted toward the row verdict): the
resume/fork mechanism itself is intact.** A `.cmd` shim on `PATH` ahead of
the real `claude.cmd`, forwarding every argument except `--version`
(answered `2.1.261`, an in-range value) straight to the real binary,
produces `decision: confirmation_required` (only `baseline.unavailable`) on
`rein resume --dry-run --json`, plan `claude --resume 0e6586d4-…`. Running
that exact plan against the real vendor directly — `claude --resume
0e6586d4-3739-47cd-bad3-395939e10d53 -p "What token did you reply with
earlier in this conversation? Reply with only the token, nothing else."
--output-format json` — returned `"session_id":"0e6586d4-…"` (the correct
session) and `"result":"claude-probe-token-E1E2"` (the correct token). Fork
likewise: dry-run plan (same in-range shim) `claude --resume 0e6586d4-… --
fork-session`; run directly against the real vendor, produced a **new**
session id `59cb2270-5182-492e-ab6e-f2fac62be31b` and the correct inherited
token. This predicts `E1`/`E2`/`E3` for `claude` would `PASS` once the
candidate's ceiling is bumped again (or the host's Claude Code is pinned
back in range) — the fabricated version string was the only variable
changed, and every other argument reached the unmodified, real vendor
binary unaltered.

## `E1`/`E2`/`E3` — `codex`, `opencode`, `grok`

All three: real installed version is inside the verified range (`codex
0.149.0` matches the ceiling exactly; `opencode 1.18.27` matches the
ceiling exactly; `grok 1.0.5` matches the single-version pin exactly), so
no version-gate interference. For each: created a real session with a
planted token, confirmed the vendor answered it, indexed it with the
snapshot `rein.exe`, took the dry-run plan, then ran the vendor for real
with that plan's arguments plus its non-interactive completion form.

| Agent | Session created | Token | Dry-run plan | Real resume result | Real fork result |
| ----- | ---------------- | ----- | ------------ | -------------------- | ------------------ |
| `codex` | `01a074e9-3b4d-7801-8382-b815976eaf81` | `codex-probe-token-E1E2` | `resume 01a074e9-…`, exit 0, `confirmation_required` | `codex exec resume 01a074e9-… "What token did you reply with earlier…"` → same session id, correct token | `codex exec fork 01a074e9-… "…"` → new id `01a074e9-f261-7220-90ab-45760e06f37e`, correct token |
| `opencode` | `ses_f8b14d3e1ffe57AzwULUMcc8m0` | `opencode-probe-token-E1E2` | `--session ses_f8b14d3e1ffe57AzwULUMcc8m0`, exit 0, `confirmation_required` | `opencode run --session ses_f8b14d3e1ffe57AzwULUMcc8m0 "…" --format json` → same session id, correct token | `opencode run --session … --fork "…" --format json` → new id `ses_f8b13b193ffeoAAKZBX2LtfrcM`, correct token |
| `grok` | `01a074ed-f6ad-7d23-a371-15cccfd8320c` | `grok-probe-token-E1E2` | `--resume 01a074ed-…`, exit 0, `confirmation_required` | `grok --resume 01a074ed-… -p "…" --output-format json` → same session id, correct token | `grok --resume 01a074ed-… --fork-session -p "…" --output-format json` → new id `01a074f0-bd90-7de2-a4c6-0b5cd2f012a6`, correct token (see harness note) |

`E1`/`E2` are `PASS` for all three. `E3` (fork) is `PASS` for all three —
see the note below on why `grok` was tested at all here, against the
dispatch's own text.

**Harness note, `grok` fork only.** `grok --resume … --fork-session -p …
--output-format json` printed the complete, correct, well-formed JSON
result (including the new session id and the correct token) but the
process did not exit on its own afterward and had to be killed after a
60s/30s timeout; the identical invocation without `--fork-session`
exits cleanly. The result was fully flushed and captured before the kill,
so this is recorded as a vendor-process observation for anyone scripting
real `grok` forks, not a Reinstate defect — nothing in the captured output
or `rein`'s own state was incomplete.

**Correction to this executor's dispatch text.** The assignment named
`claude`, `codex`, and `opencode` as the vendors `rein fork` supports, and
asked this executor to mark `grok`/`qwen` `NOT TESTED` with "the contract's
reason" for not supporting forking. That assumption does not match this
candidate: `internal/agents/catalog/grok.go` and
`internal/agents/catalog/qwen.go` both declare a `Fork` template
(`--resume {{.SessionID}} --fork-session`), and `rein sessions --json`
reports `"fork": true` in every one of this pass's five agents' session
capabilities. Tested empirically instead of skipped: `grok` fork fully
works end to end (table above); `qwen` fork mechanically works (see next
section) but its content-continuity proof is blocked by an unrelated
credential problem. Whoever maintains the Matrix E contract text or the
per-candidate dispatch should update the "which vendors fork" list —
grok's own T4 gate description already lists real resume/fork as in scope
this candidate, so this looks like a stale assumption in the dispatch, not
a deliberate scope line.

## `E1`–`E3` — `qwen`

**Credential state, disclosed up front.** `qwen`'s CLI has no dedicated
OAuth-token file; its account credential is the `env.BAILIAN_CODING_PLAN_API_KEY`
field inside `~/.qwen/settings.json` (`selectedType: "openai"`). Copying
`settings.json` into an isolated `QWEN_HOME`, and separately exporting that
exact key's value read live from the real file straight into the process
environment, both produced the identical result from the real vendor
endpoint: `[API Error: 401 invalid access token or token expired]`. Because
the second attempt used the value read fresh from the real, non-isolated
config file (not a copy made earlier in this session), this is conclusively
a pre-existing state of the host's own qwen login — expired or invalidated
independently of anything this report did — not a Reinstate defect and not
an isolation-portability problem. No login/OAuth flow was attempted, per
the ground rules.

**What that leaves testable.** A real session is still created on every
attempt (the vendor writes its session file before the model call fails):
`0c11f401-9218-4a61-bc77-61ea61f7240e` (title carries the attempted prompt
text, `message_count: 1` — only the user turn, no assistant reply exists).
`rein sessions --agent qwen --json` indexes it correctly; `rein resume
qwen:0c11f401-… --dry-run --json` exits 0, plan `--resume 0c11f401-…`,
`decision: confirmation_required` (`agent.version` matches — installed
`0.21.12` is inside `0.21.12`–`0.21.13`). **`E1:qwen` is `PASS`** — nothing
about it depends on the vendor completing a turn.

`E2:qwen` needs the resumed session to return a token that only exists in
history; with no assistant turn ever recorded, there is no token to return,
and the same 401 recurs on `qwen --resume 0c11f401-… -p "…"`. **`E2:qwen` is
`NOT TESTED`**, for the credential reason above, not a Reinstate mechanism
failure.

`E3:qwen`: `rein resume qwen:0c11f401-… --fork --dry-run --json` plans
`--resume 0c11f401-… --fork-session`. Running that for real
(`qwen --resume 0c11f401-… --fork-session -p "hello"`) again 401'd on the
model call, but a **new**, distinct session file appeared in the isolated
home's chat directory (`0bd0aa67-dac8-4c11-9a99-0888ffb44890.jsonl`) that
did not exist before the command ran — the vendor's fork mechanism itself
ran and produced a genuinely new, uniquely-identified session before the
model call failed. **`E3:qwen` is `PARTIAL`**: distinctness is proven,
content-inheritance is not (same root cause as `E2`).

## `E4` — all five agents (version-boundary shim)

A `.cmd` shim directory placed ahead of the real vendor on `PATH`,
answering only `--version` and refusing every other argument (so it is
never actually exec'd for the launch itself — the version check runs and
refuses before that point). One below the declared minimum and one above
the declared maximum, per agent, against each agent's session from above:

| Agent | Declared range | Below-min tried | Above-max tried | Both |
| ----- | --------------- | ----------------- | ------------------ | ---- |
| `claude` | `2.1.219`–`2.1.261` | `2.1.218` | `2.1.262` | exit 5, message names `2.1.219 to 2.1.261 inclusive` |
| `codex` | `0.133.0`–`0.149.0` | `0.132.0` | `0.150.0` | exit 5, message names `0.133.0 to 0.149.0 inclusive` |
| `opencode` | `1.18.21`–`1.18.27` | `1.18.20` | `1.18.28` | exit 5, message names `1.18.21 to 1.18.27 inclusive` |
| `grok` | `1.0.5`–`1.0.5` (single pin) | `1.0.4` | `1.0.6` | exit 5, message names `1.0.5 to 1.0.5 inclusive` |
| `qwen` | `0.21.12`–`0.21.13` | `0.21.11` | `0.21.14` | exit 5, message names `0.21.12 to 0.21.13 inclusive` |

All ten `E4` sub-checks (5 agents × 2 boundaries) behaved identically in
shape: `"code": "compatibility"`, `agent.version` at `severity: block`,
process exit `5`, the exact declared range named in the message. **`E4` is
`PASS` for all five agents.**

## `E5` — all five agents (active-session detection)

**Real result: `FAIL` for all five, one shared root cause.** For each
agent, a real vendor process was started in the background, resuming the
exact session under test with a long-form prompt (to keep it alive for
several seconds), confirmed running by PID at the moment of the check
(`ps -p <pid>`), and `rein resume <key>:<id> --dry-run --json` was run
concurrently. In every case, `agent.active` reported
`{"status":"match","severity":"info","actual":false,"message":"no running
<agent> instance is using this session"}` — a definitive, confidently-wrong
"not active" — while the process was independently confirmed alive and
holding that exact session id on its command line.

**Root cause, verified independently of Reinstate.** `internal/
processcheck/process_windows.go`'s `listProcesses` shells out to `Get-
CimInstance Win32_Process`. On this host, that call — and the legacy `Get-
WmiObject Win32_Process`, and `Get-CimInstance Win32_OperatingSystem`, and
`winmgmt /verifyrepository` — all fail: `Get-CimInstance`/`Get-WmiObject`
return `Critical error` (`HRESULT 0x8004100a`), and `winmgmt
/verifyrepository` reports `WMI repository verification failed … Access
denied (0x80041003)`. This reproduces identically from a Bash-launched
`powershell.exe` and from the native PowerShell tool (Windows PowerShell
5.1), so it is not an artifact of how this session invokes PowerShell — WMI
process enumeration is unavailable to this account/host right now,
independent of Reinstate.

**Why this is still reported as a product defect, not only a harness
footnote.** `internal/processcheck/process.go`'s `SessionBusy` catches the
`listProcesses` error internally and returns `(false, false, nil)` —
success, not-busy, no error — so the caller (`internal/preflight/
verify.go`'s `startActiveSessionProbe`) never reaches its own already-
implemented `case err != nil …: Status: StatusUnknown` branch for this
failure mode; that branch exists and is correctly written, it is just
unreachable from this specific internal error path. The result an operator
sees is indistinguishable from a verified "nothing is using this session."
The function's own doc comment states the opposite intent: "Detection
therefore deliberately biases toward busy… a false negative costs a live
session." A host where WMI is unavailable — locked down by group policy, a
disabled service, a restricted service account, security software
interference, a sandboxed/virtualized runner — is not an exotic case on
Windows, and on such a host `agent.active` will now silently and
permanently read "not active" no matter what is really running. See
[Product defects](#product-defects) and [Harness defects](#harness-defects).

## `E6` — all five agents (non-interactive refusal)

`rein resume <key>:<id>` with stdin redirected from `/dev/null`, no
`--json`, no `--dry-run`, no console:

| Agent | Exit | stderr |
| ----- | ---- | ------ |
| `claude` | 7 | `environment warnings require confirmation: baseline.unavailable` |
| `codex` | 7 | `environment warnings require confirmation: baseline.unavailable` |
| `opencode` | 7 | `environment warnings require confirmation: baseline.unavailable` |
| `grok` | 7 | `environment warnings require confirmation: baseline.unavailable, git.working_tree` |
| `qwen` | 7 | `environment warnings require confirmation: baseline.unavailable` |

**`E6` is `PASS` for all five agents.** (`claude`'s `E6` result is
independent of the version-block finding above: the non-interactive gate is
evaluated and refuses before the interactive/JSON distinction matters, and
would refuse the same way regardless of the version-range outcome — this
was re-confirmed fresh against this exact snapshot rather than assumed from
the first pass's already-passing `E6` rows.)

## Release-blocking findings

1. **`claude` native resume/fork is fully non-functional out of the box on
   this exact candidate, for any Windows host whose Claude Code has
   auto-updated past `2.1.261`.** This candidate's verified ceiling was
   widened to `2.1.261` on `v060/w3-ranges`, merged the same day as this
   pass. The vendor has already shipped `2.1.263` by the time of this
   pass — one widening cycle behind in under 24 hours. `preflight.Authorize`
   refuses unconditionally on a `block`-severity check with no override for
   any invocation shape (`--json`, interactive, `--allow-environment-
   warning`, or otherwise). This is not a logic bug — fail-closed on an
   unverified version is the documented design — but it means the range
   this candidate tags with should be re-verified against whatever Claude
   Code version is current at tag time, not merely at whenever the
   widening branch happened to land, or every user who has auto-updated
   loses native `claude` resume/fork entirely until the next release.
   Recommend: re-check the installed Claude Code version immediately before
   tagging `v0.6.0-rc.1` (not merely before this or any other test pass) and
   widen again if it has moved.
2. **`agent.active` silently reports "definitely not active" instead of
   "unknown" when process enumeration fails**, on any Windows host where
   WMI is unavailable. See `E5` above for the full mechanism
   (`processcheck.SessionBusy` swallows the `listProcesses` error). This
   defeats the documented fail-safe intent of active-session detection
   precisely in the security-hardened/locked-down environments where a
   Windows operator is statistically more likely to have two windows open
   on the same session and less likely to notice. Recommend: propagate the
   error instead of swallowing it, so the existing (already correct)
   `StatusUnknown` branch in `verify.go` is reachable.

## Product defects

- Both items above.
- (Reproduced, not new) codex's capability/skill scan is still not
  redirected by `CODEX_HOME` isolation on this candidate: `E6:codex`'s
  plain-text environment dump enumerated real, host-level
  `capability.skill.*` entries under full `CODEX_HOME` isolation. This is
  the first pass's harness/methodology finding #2, confirmed still present
  at `86cb3421` and reproduced again independently in this pass. No skill
  name is reproduced in this report.

## Harness defects

- **WMI process enumeration (`Get-CimInstance`/`Get-WmiObject
  Win32_Process`, and `Get-CimInstance Win32_OperatingSystem`) is denied on
  this specific acceptance host/account**, confirmed from both a
  Bash-launched `powershell.exe` and the native PowerShell tool. This
  prevented directly observing the "would have refused/warned" half of
  `E5` on this host for any agent (the product-defect half — the silent
  swallow — was independently confirmed by code inspection and is not
  contingent on this host quirk). Whoever runs the next pass on a host with
  working WMI should re-run `E5` for at least one agent to confirm the
  *positive* detection path (busy → warning/refusal) also behaves as
  documented.
- **`qwen`'s coding-plan API credential (`env.BAILIAN_CODING_PLAN_API_KEY`
  in `~/.qwen/settings.json`) is expired/invalid on this host as of this
  pass**, reproduced with the live value from the real config file. This
  blocked `E2:qwen` and the content-continuity half of `E3:qwen`. Whoever
  owns this host's qwen login should re-authenticate before the next
  physical-resume pass for this agent.
- **This executor's dispatch text incorrectly stated that `grok`/`qwen` do
  not support `rein fork`.** See the correction under `E1`–`E3` above; both
  do, per the catalog and per real vendor behavior, and were tested rather
  than skipped.
- One extra, unplanted `grok` session (`01a074ed-01c3-7ba3-87e1-
  b2a8857c6d64`) appeared alongside the intended one after the first probe
  attempt produced no visible stdout in this session's tool output (a tool
  buffering artifact, not a `grok` or `rein` behavior) — both sessions carry
  only the same synthetic probe prompt and token; the extra one was left
  alone (never used for any row) rather than deleted, consistent with "do
  not invent a way to remove vendor session history" precedent from
  `windows-acceptance-host.md`.

## Cleanup

All isolated homes, throwaway git repositories, and shim directories used
above live under `D:\ReinstateAcceptanceProjects\v060-w7c-d\` and
`D:\ReinstateAcceptanceProjects\v060-w7c-d\shims\` on the test host and are
not committed. No transcript text, real prompt, real response, credential
value, private path, or vendor skill/session name from the developer's real
trees appears above — every token quoted is a marker planted by this report
for the sole purpose of proving continuation.

**macOS evidence: not this executor's scope (Windows-only per the
dispatch).**
