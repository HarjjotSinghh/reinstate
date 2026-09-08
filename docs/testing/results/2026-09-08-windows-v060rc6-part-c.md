# `v0.6.0-rc.6` tagged-artifact acceptance — part C (CLI experience, 22 rows)

`PHASE5-DEVICE-REPORT-V1` (part file)

Executor C's part of the `v0.6.0-rc.6` native Windows tagged-artifact
acceptance. Covers **section C only**: the 22-row `v0.5.2` CLI experience
contract, Windows column, through `scripts/testing/conptydriver` against
`scripts/tuisandbox`. Satisfies
[`../v0.5.2-cli-experience-acceptance.md`](../v0.5.2-cli-experience-acceptance.md)
as specialized by
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) and
[`../v0.6.0-rc.6-agent-verification-prompts.md`](../v0.6.0-rc.6-agent-verification-prompts.md).
Intended to be merged into the cumulative tagged report alongside the other
executors' parts, per the same pattern the `v0.6.0-rc.5` report used.

## Header

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.6` |
| Full commit | `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| Release workflow run | `34170072716` (per dispatch) |
| Windows archive | `reinstate_0.6.0-rc.6_windows_amd64.zip` |
| Archive SHA-256 | `ff7e232958b6141d3091e343ef0246306819dcc428fa161101416e3bead22ffd` (matches `checksums.txt`, re-verified independently before install) |
| `rein.exe` / `reinstate.exe` SHA-256 | `272efb504a62fdd2624fa6b117a2587a4869160c034adeb9fcd6281231f078c1` (both files, byte-identical) |
| `rein version --json` | `{"commit":"7c5adccaea602ff952cd31c21bafa7fdd22a2cfc","date":"2026-09-07T23:28:49Z","name":"reinstate","version":"0.6.0-rc.6"}` |
| Bootstrap deviation | This executor did **not** install from the live `https://reinstate.dev/install.ps1` bootstrap. Per the dispatch, executor A alone performs that install and records the artifact identity from it; this executor (C) installed from the coordinator's pre-verified, checksummed archive at the scratchpad `rc6-draft` directory, as every non-A executor is instructed to. |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc6-c\install\extracted\` (fresh, this executor's own) |
| Worktree / branch | `D:\Projects\reinstate-worktrees\v060-rc6-tagged`, branch `v060/rc6-tagged` at `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| Previous-release binary (rows 2/22) | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches its own `checksums.txt`), version `0.5.1` / commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb` |
| Host OS | Windows 11 Pro, NT 10.0.26200.0, x64 |
| Go toolchain | go1.26.1 windows/amd64 (build gate requires 1.25.13+; this is newer) |
| Date (UTC) | 2026-09-08 |
| Host contamination rule | Every shell that ran `rein`, `conptydriver`, or `go run`/`go build` for this part first unset `REINSTATE_BACKEND` and `REINSTATE_MEMORY_BACKEND_DIR` (the ambient shell profile sets both — `memory` / a live path — and this was confirmed and cleared before any row ran; see "Harness note" below). `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were left alone in the ambient shell and instead redirected per-row inside the isolated `scripts/tuisandbox` environment, never touching the host's live agent homes. |

### Harness note: isolating `scripts/tuisandbox` from ambient contamination

The ambient shell carries real values for `CLAUDE_CONFIG_DIR`
(`D:\Projects\hop-10-lab\claude`), `CODEX_HOME`
(`...\orca\codex-runtime-home\home`), and `XDG_DATA_HOME`
(`D:\Projects\hop-10-lab\xdg`), plus `REINSTATE_BACKEND=memory` and a live
`REINSTATE_MEMORY_BACKEND_DIR`. `scripts/tuisandbox`'s own env fragment
(`HOME`, `USERPROFILE`, `REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`)
redirects the first five; it deliberately leaves `GROK_HOME` **unset** so
the Grok source falls back to the sandboxed `$HOME/.grok` (documented in
`scripts/tuisandbox/main.go`). `XDG_DATA_HOME` is not part of the bench's
own contract and was left pointing at the host's real OpenCode data
ahead of the first attempt; every row below was run only after building
one isolated environment script (`setenv.ps1`) that: unsets
`REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`; sets `HOME`,
`USERPROFILE`, `REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME` to the
sandbox; redirects `XDG_DATA_HOME` to an empty isolated directory; clears
`GEMINI_CLI_HOME`/`GROK_HOME`/`QWEN_HOME` defensively (all three were
already unset ambiently); and rebuilds `PATH` to the bench's own `bin\`,
`C:\Program Files\Git\cmd`, and the Windows system directories only.
Verified before any row counted: `rein sessions --json` under this
environment returns exactly the bench's documented 8 indexed sessions
(the 9th, the read-only `grok` `payment-adapter` record, is visible only
via `rein inspect`/the switcher, not `sessions --json`, matching the
fixture's own read-only-record design) — no host session leaked in.

**A second harness note, encountered and resolved during this run, not a
product defect:** `conptydriver.exe` invoked directly from this
executor's own automation shell (piped stdout) produces a child that
correctly refuses to run interactively (`IsTerminal` false, matching the
driver's own documented "needs a real console of its own" trap). Every
row below was launched via PowerShell `Start-Process` without
`-RedirectStandardOutput`/`-RedirectStandardError` (`-WindowStyle Hidden`),
per the driver's own README, giving `conptydriver.exe` a genuine new
console. Separately, two early exploratory runs (never used as row
evidence) left an orphaned, still-running `rein.exe` child after
`conptydriver.exe` itself exited; a later snapshot briefly rendered a Go
runtime goroutine-dump (consistent with a `CTRL_BREAK`-class signal
hitting that stray process, which is standard Go runtime behavior on
Windows, not a panic in product logic). Every row counted below was
re-run cleanly with no other `rein`/`conptydriver` process alive
immediately before or after it (verified by process listing), and none
of the counted evidence shows this artifact. No product defect is
recorded from this.

## Matrix (22 rows: 22 PASS)

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | Header `9 sessions · 3 agents · all projects`, filter prompt, time-grouped rows, split preview, key bar. 10s and 18s checkpoints byte-identical (fully settled): 2 `READY` (`●`), 4 `WARN` (`◐`), 3 `BLOCKED` (`○`) — every glyph matches the bench's documented seed state |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical to `v0.5.1` | PASS | Both `rein.exe` (rc6) and the v0.5.1 binary exit `2` on a fully redirected, non-TTY invocation; stderr SHA-256 byte-identical between the two versions and identical to the hash the `v0.6.0-rc.5` report recorded: `0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b`. (stdout's Cobra-generated command list differs, as expected — v0.6.0 adds `account`/`daemon`/`devices`/`hop`/`login`/`whoami`; that is not part of the frozen-output guard, which is the non-TTY refusal line itself) |
| 3 | `--plain` falls back to the numbered switcher | PASS | Frozen numbered-picker text over the bench's 9 rows: `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Byte-identical rendered frame to row 3 |
| 5 | `TERM=dumb` does the same | PASS | Byte-identical rendered frame to rows 3/4 |
| 6 | `TERM` unset draws the switcher on Windows | PASS | Reuses row 1's evidence — `TERM` was never set in any row's environment |
| 7 | A terminal below 40×10 falls back to plain | PASS | `30×8`: wrapped numbered-picker text, same frozen prompt line |
| 8 | `NO_COLOR` draws the switcher with no color escape sequences | PASS | Raw byte stream over a full settled frame: exactly one bare `ESC[m` reset, zero SGR color sequences; frame content confirmed to be the full interactive switcher (not a fallback) |
| 9 | Typing filters; header count follows | PASS | `"auth"` → `1 session · 1 agent`, list narrows to the one matching row |
| 10 | `f` in list mode filters, does not fork | PASS | Filter line reads `f`, header count unchanged (`9 sessions · 3 agents`), full session list still shown — no fork screen opened |
| 11 | `tab` opens the action menu; `esc` reverts without acting | PASS | Key bar changes to `r resume   f fork   h hand off   i inspect   y copy ref   esc back`, then `esc` reverts to a frame byte-identical (whole-file `diff`, zero output) to the pre-`tab` frame |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `"hof"` → top match `▸ Hand off to another…  new session from a briefing`; key bar `↵ run   esc close` |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | **PASS** | See "Row 13 — full evidence" below |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | PASS | See "Row 14 — full evidence" below |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | 2 of 4 warnings acknowledged (`baseline.unavailable`, `git.branch`) on the `website` session (`claude:...003`, 4 warnings): exit `7`, message names exactly the two remaining: `environment warnings require confirmation: git.working_tree, runtime.node.declaration` |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | See "Row 16 — full evidence" below |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | See "Row 17 — full evidence" below |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | See "Row 18 — full evidence" below |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Walked all 7 steps of a fresh (first-device) `rein init`: provider, endpoint, bucket, region, prefix, device-role, review — no access-key/secret-key/passphrase prompt anywhere in the full-screen program; review screen states "Next you will enter your storage keys, then a passphrase" (i.e., after the program exits); `rein init --help` lists no credential flag |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | See "Row 20 — full evidence" below |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | With `WT_SESSION`, `TERM_PROGRAM`, `LANG`, `LC_ALL`, `LC_CTYPE` all explicitly cleared (the ambient shell otherwise carries `TERM_PROGRAM=Orca` and `LANG=en_US.UTF-8`, which would trigger the Unicode path): every glyph rendered ASCII — `*`/`!`/`x`/`.` for ready/warn/blocked/pending, `>` cursor, `/` search, `|` vertical bar, `enter`/`...` spelled out — never Unicode, across a full settled frame |
| 22 | Every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences | PASS | See "Row 22 — full evidence" below |

**22/22 PASS.**

### Row 13 — full evidence

**From outside any tracked project (`ScopeAll`, the default; launched with
`-dir` set to the bench's own `home` root, not any project subdirectory).**
Header reads `9 sessions · 3 agents · all projects`. At a 10-second
checkpoint and again at 18 seconds (byte-identical — fully settled), every
glyph matches its seeded state: `●` `auth-refactor` (claude), `●`
`keyring-store` (codex) — 2 `READY`; `◐` `website`, `◐` `checkout-flow`,
`◐` `deps-bump`, `◐` `search-index` — 4 `WARN`; `○` `gone-missing`, `○`
`payment-adapter` (grok), `○` `repo-drift` — 3 `BLOCKED`. Independently
cross-checked against `resume --dry-run --json`: the `grok`
`payment-adapter` session reports `"decision":"blocked"`,
`"block_exit_code":5`, `agent.executable status=missing`; the `claude`
`auth-refactor` session reports `"decision":"ready"`. Both match what the
switcher showed.

**From inside a tracked project (`ScopeProject`; launched with `-dir` set
to the `auth-refactor` project directory).** Header auto-narrows to `1
session · 1 agent · auth-refactor`; the one visible row settles to `●
READY TO RESUME` — matching its `ScopeAll` result above and its
independent `--dry-run --json` ground truth.

### Row 14 — full evidence

Selected the `website` session (`claude:...003`, seeded with 4
environment warnings: `baseline.unavailable`, `git.branch`,
`git.working_tree`, `runtime.node.declaration`) from the switcher and
pressed `enter`; the checklist screen opened (`◐ 4 environment warnings`,
all four `[ ]`, equivalent command `rein resume claude:...003`). One
`space` press toggled exactly `baseline.unavailable` to `[x]`, leaving
the other three `[ ]`, and the equivalent command live-updated to `rein
resume claude:...003 --allow-environment-warning baseline.unavailable`.
A second `space` press reverted the frame to a state byte-identical to
the pre-acknowledgement frame (all four `[ ]`, equivalent command back to
the bare `rein resume claude:...003`).

### Row 16 — full evidence

Opened from the switcher (`tab` then `h`) on the `auth-refactor` session,
launched from that session's own workspace directory. Initial state:
`policy ◂ balanced ▸`, equivalent command `rein handoff claude:...001
--to codex --policy balanced`. `key right` cycled the policy in lock-step
with the equivalent command: `balanced → full` (`--policy full`) →
`checkpoint` (`--policy checkpoint`) → wrapping back to `balanced`, at
which point the measurement had finished and the studio showed the full
projection: 8 fields `carried across` (exact/normalized/summarized) and 6
`left behind` (omitted/referenced, each with its own reason), plus `[ ] 2
warnings to acknowledge` (`baseline.unavailable`,
`handoff.capability.attachment.support`) — confirming the studio performs
a real measurement, not a static placeholder, and the equivalent command
always names the currently selected policy.

### Row 17 — full evidence

Selected the `repo-drift` session (`codex:...006`, foreign
`repository_url`) and opened the handoff studio (`tab`, `h`) targeting
`claude`. After the measurement resolved (`no measurement` /
`○ this handoff cannot be planned` / `handoff: environment preflight is
blocked`), pressing `enter` showed the refusal inline
(`this handoff cannot be planned: handoff: environment preflight is
blocked`) **without leaving the studio**; a second `enter` re-showed the
identical refusal, studio still open — matching the row's contract
exactly. Non-blocking harness observation: pressing `enter` in the first
~100ms after opening the studio, before the async measurement had
determined the plan was unbuildable, was treated as `send` once — the
alt-screen exited to the plain CLI, which then correctly refused with the
same `handoff: environment preflight is blocked` message and made no
session. No destructive action occurred in either path; this is a UX
timing nuance in a sub-second window, not a broken refusal, and is not
recorded as a defect.

### Row 18 — full evidence

Fresh `rein init` (no existing config) on an empty home. Step 1 of 7,
`Storage provider`; selected `Other S3-compatible` (4×`down`), `enter` →
step 2, `Endpoint`, pre-filled placeholder `https://`. Typed
`not-a-valid-endpoint`, `enter`: **stayed on step 2**, inline validation
error `the endpoint must start with https:// or http://` shown, nothing
lost. Sent a backtab (`ESC[Z`, the wizard's own back key — `conptydriver`
has no named `shift+tab`, so this was sent as the raw CSI sequence the
key represents): returned to step 1 with the prior selection preserved
(`▸` still on `Other S3-compatible`).

### Row 20 — full evidence

Real two-device round trip against a disposable `scripts/testing/fakelocker`
instance (`127.0.0.1:9101`, in-memory, never persisted), two fully
isolated `REINSTATE_HOME`s.

- **Device A**: non-interactive `rein init --endpoint http://127.0.0.1:9101
  --bucket row20bucket --prefix row20devA --yes` (env-var credential
  provider) → `profile_id=1cb9125f-8daf-4e05-8197-69fcae639380`. `rein init
  --link` printed a wrapped pairing code (`REIN1-...`, 4 display lines).
- **Device B** (separate isolated home): the pairing code (reassembled to
  one line, matching how the wrapped display is meant to be read back) fed
  to `rein init --paste` via the "Paste the pairing code…" stdin prompt.
  The wizard opened and **every field pre-filled exactly matching device
  A**: provider `Other S3-compatible`, endpoint `http://127.0.0.1:9101`,
  bucket `row20bucket`, region `auto`, prefix `row20devA`, device role
  `Join a profile from another device`, and step 7 (`Profile ID`) showing
  `1cb9125f-8daf-4e05-8197-69fcae639380` — **the identical profile UUID**
  device A generated. The review screen (step 8 of 8) confirmed all six
  fields together in one frame. Continuing past review correctly refused
  with `remote profile manifest not found at configured storage
  coordinates` (device A never pushed anything to the fake locker) — the
  expected, correct refusal for "join" against a profile with no manifest
  yet, not a defect.

### Row 22 — full evidence

Five documents compared between rc6 and the v0.5.1 binary, same bench:

- `sessions --json`: differs only in the `grok` session's capabilities —
  v0.5.1 reports `"resume":false,"fork":false,"read_only_reason":"Grok
  Build sessions are source-only in Phase 4"`; rc6 reports
  `"resume":true,"fork":true`, no `read_only_reason`. Matches the
  changelog's documented Grok T3 (verified-resume) promotion. No other
  difference.
- `resume claude:...001 --dry-run --json`: differs only by one added
  check block, `agent.active` (`status=match, actual=false`). Matches the
  changelog's documented `agent.active` liveness check, added since
  `v0.5.1`.
- `inspect claude:...001 --json`: same single `agent.active` addition,
  nothing else.
- `handoff claude:...001 --to codex --dry-run --allow-warning
  baseline.unavailable --allow-warning
  handoff.capability.attachment.support --json`: **byte-identical**, zero
  diff.
- `doctor --json`: differs only in the `version` field (`0.5.1` vs.
  `0.6.0-rc.6`).

No third, undocumented difference class found in any of the five
documents.

## Commands used (argv only, no session content)

```
rein.exe version --json
rein.exe sessions --json
rein.exe inspect grok:<id> --json
rein.exe resume claude:<id> --dry-run --json
rein.exe resume grok:<id> --dry-run --json
rein.exe resume claude:<id> --allow-environment-warning <id> --allow-environment-warning <id>
rein.exe init --link
rein.exe init --paste
rein.exe init --endpoint <url> --bucket <name> --prefix <name> --yes
rein.exe doctor --json
rein.exe handoff <session> --to <agent> --dry-run --allow-warning <id> --allow-warning <id> --json
conptydriver.exe -cols N -rows N -dir <path> -script <path> [-raw <path>] -- rein.exe [args]
```

## Disposition

All 22 required CLI-experience rows `PASS`. No regression from
`v0.6.0-rc.5`'s own 22/22 `PASS` result; nothing in this candidate's own
changes (the `grok` backend/version fix and the `MatrixH:H7` rule
refinement) touches the interactive surfaces, and none was found here.
