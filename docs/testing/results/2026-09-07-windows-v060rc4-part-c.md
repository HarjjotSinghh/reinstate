# `v0.6.0-rc.4` native Windows acceptance — part C (CLI experience, 22 rows)

Executor C of the tagged-artifact run. Covers section C in full: the
[`v0.5.2` CLI experience contract](../v0.5.2-cli-experience-acceptance.md)'s
22-row Windows column, driven through the real Windows pseudo console
(`scripts/testing/conptydriver`) against `scripts/tuisandbox`'s synthetic
bench, exactly as the rc.4 dispatch requires. Rows 2 and 22 are compared
byte-for-byte against a `v0.5.1` release binary on the same synthetic home.
Row 21 is captured on legacy conhost (no `WT_SESSION`, no `TERM_PROGRAM`, no
UTF-8 locale). Row 14 is exercised with the space bar toggling exactly one
warning.

Per the rc.4 dispatch, none of these 22 rows is a carried disposition: all
22 passed at `v0.6.0-rc.3`'s tagged run, and this candidate's own two
changes (the Pi reader fix, the Qwen Code range widening) touch neither the
interactive CLI nor its frozen-output guard. **21 of 22 rows PASS. Row 13 is
a new finding (FAIL)**, isolated below to a specific, reproducible condition
in the interactive switcher's default ("all projects") scope — not a carried
rc.3 disposition and not explained by rc.3's own already-documented
`Start-Process` timing finding (ruled out below).

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-07T12:16:00Z`–`2026-09-07T13:10:00Z` (session); report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro `10.0.26200` |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.4` |
| Tested full commit | `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Windows archive | `reinstate_0.6.0-rc.4_windows_amd64.zip` |
| Windows archive SHA-256 (re-verified against `checksums.txt` before install) | `f5ae24e2edf0364f0e6f9b0df3c5b2c21f47222ecd394b9c40fc5dbd1ae14d87` |
| Installed binary SHA-256 (`rein.exe` == `reinstate.exe`, byte-identical) | `fbea94615dabcbc30fc1ebb6e93259ddd330b62573555e551fd3112b31b9d3ef` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc4-c\install`) | `{"commit":"561ec133e7fd040d7937d555a75f0bd7dd7b878e","date":"2026-09-07T06:48:08Z","name":"reinstate","version":"0.6.0-rc.4"}` |
| Previous-release comparison binary (rows 2, 22) | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (checksum re-verified before use); `rein version --json` reports `{"commit":"e8d1ec28edee73005a51ca8802a04ced369f4bcb","version":"0.5.1", ...}` |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc4-tagged`, branch `v060/rc4-tagged` @ `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Git version | `2.52.0.windows.1` |
| Go version (host default; used only to build `conptydriver`/`tuisandbox`, not the artifact under test) | `go1.26.1` |
| Host contamination rule | Every shell that ran `rein`, `conptydriver`, or `tuisandbox` in this part first cleared `REINSTATE_BACKEND`, `REINSTATE_MEMORY_BACKEND_DIR`, and (beyond the dispatch's minimum list, after the ambient shell was found to carry the host's real OpenCode `XDG_DATA_HOME`, `TERM_PROGRAM`/`TERM_PROGRAM_VERSION`, and a `TERM` value) `XDG_DATA_HOME`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`, `COPILOT_HOME`, `CURSOR_CONFIG_DIR`, `CLINE_DATA_DIR`, `PI_CODING_AGENT_DIR`, `QWEN_HOME`, `OH_PERSISTENCE_DIR`, `GROK_HOME`, `TERM`, `WT_SESSION`, `LANG`, `LC_ALL`, `LC_CTYPE` before every row (see harness note below); `CLAUDE_CONFIG_DIR`/`CODEX_HOME` were always pointed at `scripts/tuisandbox`'s own synthetic home, never the host's live `D:\Projects\hop-10-lab\claude`. Every switcher launch reported the bench's documented `9 sessions · 3 agents`, confirming no host session leaked in. |
| Bootstrap deviation sentence | Not applicable to this executor: per this run's ground rules, executor A alone installs from the live `https://reinstate.dev/install.ps1` bootstrap and records the artifact identity for the whole run; this executor installed from the coordinator-verified, checksum-matched `reinstate_0.6.0-rc.4_windows_amd64.zip` in the shared `rc4-draft` staging directory into its own fresh `D:\ReinstateAcceptanceProjects\v060-rc4-c\install\` (sha256 re-verified above, matches `checksums.txt`), per the dispatch for every non-A executor. |
| Bench | `scripts/tuisandbox -root D:\ReinstateAcceptanceProjects\v060-rc4-c\tuisandbox-root -shell sh`, outside any Git checkout, per the generator's own requirement; 9 synthetic sessions, 3 agents (`claude`, `codex`, `grok`), fake `claude`/`codex` shims (`claude 2.1.228`, `codex 0.140.0`) on `PATH` ahead of the host's real vendor binaries |
| Driver | `scripts/testing/conptydriver` built from this worktree (`go build ./scripts/testing/conptydriver`), invoked via PowerShell `Start-Process` (without stdio redirection, `-WindowStyle Hidden`) so the child gets a genuine console per the driver's own documented trap; results read from `-script`-written snapshot files, never from captured stdout |

## Row-13 harness note

`conptydriver.exe` launched through PowerShell's `Start-Process` on this
host carries measurable subprocess-spawn latency, a finding first quantified
at `v0.6.0-rc.3` (report §21, "host probe fan-out timing", F2). That finding
was **ruled out as the explanation for Row 13** below by a controlled
A/B: the same 9-session, 3-agent index, probed through four different
channels under the identical `Start-Process`/ConPTY harness, gave three
correct results and one reproducibly wrong one — see Row 13.

---

## Matrix

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | Header `rein  9 sessions · 3 agents · all projects`, filter prompt (`❯ type to filter`), time-grouped rows (TODAY/YESTERDAY/THIS WEEK/THIS MONTH/OLDER), a Unicode readiness column, split preview pane, key bar `↵ resume  tab actions  ctrl+a scope  ctrl+k commands  esc quit` |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical to `v0.5.1` | PASS | Both exit `2`; **stderr SHA-256 byte-identical**, `0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b` on both `v0.6.0-rc.4` and `v0.5.1` (matches the hash recorded at `v0.6.0-rc.2`/`rc.3`). Stdout (the `cmd.Help()` fallback text) legitimately differs — new subcommands (`account`, `daemon`, `devices`, `hop`, `login`, `sync`, `whoami`) and flags (`--plain`) added since `v0.5.1` — which is auto-generated cobra help, not the frozen refusal text the row protects |
| 3 | `--plain` on a capable terminal falls back to the numbered switcher | PASS | Frozen numbered-picker text: `Local sessions` header, 9 numbered rows, `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Byte-identical to row 3 (`sha256` match) |
| 5 | `TERM=dumb` does the same | PASS | Byte-identical to row 3 (`sha256` match) |
| 6 | `TERM` unset draws the switcher on Windows | PASS | Full interactive switcher, identical shape to row 1 |
| 7 | A terminal below 40×10 falls back to plain | PASS | `-cols 30 -rows 8`: the same frozen numbered-picker text, wrapped to the narrow width |
| 8 | `NO_COLOR` draws the switcher with no escape sequences for colour | PASS | Raw byte capture (`-raw`) parsed for `ESC[...m`: **zero** SGR sequences carrying a colour parameter, **exactly one** bare `ESC[m` reset |
| 9 | Typing filters; the header count follows | PASS | Typing `auth` → header reads `1 session · 1 agent`, list narrows to the one matching row |
| 10 | `f` in list mode filters and does **not** fork | PASS | `key f` → filter line reads `f`, header narrows to `1 session · 1 agent`; no agent process launched, no action screen opened, key bar shows `esc clear` (filter active), not a fork/launch confirmation |
| 11 | `tab` opens the action menu; `esc` returns without acting | PASS | `tab` → key bar becomes `r resume  f fork  h hand off  i inspect  y copy ref  esc back`, header gains an `actions` label; `esc` → frame is **byte-identical** (`sha256` match) to the pre-`tab` frame |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `ctrl+k` then typing `hof` → top match `Hand off to another…  new session from a briefing`; `↵ run  esc close` key bar |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | **FAIL — new finding** | See "Row 13 — full evidence" below. The engine itself is correct (proven three independent ways); the interactive switcher's default "all projects" scope shows every row as `CANNOT RESUME` regardless of true state, 9/9 reproductions across 7 separate launches |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | PASS | `rein resume claude:...003` (W1, 4 warnings) from its own workspace: initial frame lists 4 unchecked warnings, equivalent command `rein resume claude:...003`; one `space` on the highlighted `baseline.unavailable` row → `[x]`, equivalent command becomes `... --allow-environment-warning baseline.unavailable`; a second `space` on the same row → reverts to `[ ]` and the original equivalent command, byte-identical to the first frame |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | Same W1 session, real (non-`--dry-run`) `resume` with only 2 of 4 warnings acknowledged (`baseline.unavailable`, `git.branch`): `environment warnings require confirmation: git.working_tree, runtime.node.declaration`, exit `7`, names exactly the two remaining |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | Switcher → `tab` → `h` on R1 (from its own workspace) opens the studio; after full measurement (~6s): `carried across`/`left behind` fidelity breakdown, `2 warnings to acknowledge`; `→` cycles policy `balanced → full → checkpoint`, and the "equivalent command" line tracks each selection exactly (`--policy balanced`/`--policy full`/`--policy checkpoint`) |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | Studio opened on B3 (`codex:...006`, foreign `repository_url`) from its own workspace: after full measurement, `○ this handoff cannot be planned / handoff: environment preflight is blocked`; `enter` does **not** send — the studio stays open and adds an inline refusal line `this handoff cannot be planned: handoff: environment preflight is blocked` under the same key bar |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | Step 1/7 provider picker; step 2/7 endpoint field pre-filled `https://`; typing `not-a-url` and pressing `enter` refuses in place with `the endpoint must start with https:// or http://` and does not advance; `up` returns one step at a time (step 3→2) with the previously typed value (`https://s3.example.com`) preserved, not reset to the template |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Walked all 7 steps (provider, endpoint, bucket, region, key prefix, device role, review); none prompts for an access key, secret key, or passphrase — step 7's review explicitly states "Next you will enter your storage keys, then a passphrase. The passphrase is never written to disk…" as work that happens **after** the full-screen program. `rein init --help` confirms: no `--access-key`/`--secret-key`/`--passphrase` flag exists; `--yes` "requires ... environment credential provider" |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | See "Row 20 — full evidence" below: a real two-device BYO round trip against a fake-but-real S3-compatible endpoint. Device A prints a `REIN1-...` code with "This code carries no keys and no passphrase."; device B's `--paste` pre-fills provider, endpoint, bucket, region, prefix, and profile ID **exactly** matching device A, down to the identical profile UUID |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | Launched with `WT_SESSION`, `TERM_PROGRAM`, `TERM_PROGRAM_VERSION`, `LANG`, `LC_ALL`, `LC_CTYPE` all unset (matching `internal/ui/unicode_windows.go`'s legacy-conhost default): every glyph is the ASCII set from `internal/ui/theme.go`'s `asciiGlyphs` (`.` pending, `/` filter, `|` vertical bar, `...` ellipsis, `enter` instead of `↵`) — never the Unicode set shown in rows 1/6 |
| 22 | Every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences | PASS | 8 documents compared on the identical bench (`sessions --json`; `resume --dry-run --json` for `claude` R1, `codex` R2, `claude` W4/detached, `grok` B1; `inspect --json` for `claude` B2, `grok` B1; `handoff --no-launch --json`): **2 byte-identical** (`handoff --no-launch --json`; `doctor --json` differs only in the version string, which is not part of the frozen contract), **6 differ, and every difference traces to exactly one of two documented, changelog-backed classes** — see "Row 22 — accepted classes" below. No third class was found or needed in this candidate's evidence |

**21/22 PASS, 1 FAIL (Row 13, new finding).**

---

## Row 13 — full evidence

**What the doc requires:** "Readiness glyphs resolve for visible rows and a
read-only agent shows blocked without a probe."

**What was found.** Launching bare `rein` from a working directory that is
**not** inside any of the bench's tracked Git repositories (e.g.
`D:\ReinstateAcceptanceProjects\v060-rc4-c\install`, or a throwaway,
unrelated Git repository created solely to test this) puts the switcher in
its documented `ScopeAll` ("all projects") mode. In that mode, **every one
of the 9 visible rows shows the `BlockedMark` glyph and the preview pane
reads `○ CANNOT RESUME`, unconditionally** — including `R1`
(`claude:...0001`, seeded READY, clean repo, branch matches) and `R2`
(`claude:...0002`/`codex`, seeded READY) — regardless of a 4–8 second
settle time after the switcher's first frame. This was reproduced **9/9**
times across 7 separately-launched processes (`cols`/`rows` varied 80×24 to
100×30; settle time varied 3s–8s; the on-disk index was both cold and
already-warmed by a prior run — no variation changed the outcome).

**Ruling out a stale/pending state.** `internal/ui/theme.go`'s `Glyph`
function maps every readiness value other than `Ready`/`Warn`/`Blocked` to
the *pending* glyph (`◌`/`.`), never to `Blocked`. The captured glyph is
unambiguously the `Blocked` glyph (`○`), meaning the probe **completed**
and returned a `Blocked` verdict — this is not an unresolved/timed-out
probe rendering as blocked.

**Ruling out the underlying engine.** The same 9-session index was probed
four different ways under the identical `Start-Process`/ConPTY harness:

1. `rein resume claude:...0001 --dry-run --json` (single-agent refresh, cwd
   outside any repo) → `"decision": "ready"`, `"source.fresh":
   {"actual": true}` — **correct**.
2. `rein --plain` then `i 1` (multi-agent/`ScopeAll`-equivalent refresh, cwd
   outside any repo, `D:\...\install`) → every check `match`/`present`
   (`git.repository`, `git.branch`, `git.head`, `git.working_tree`,
   `agent.executable`, `agent.version`, `agent.active`) — **correct**.
3. `rein` (interactive switcher), cwd **inside** `R1`'s own workspace (auto
   narrows to `ScopeProject`, 1 session, 1 agent) → `● READY TO RESUME` —
   **correct**, reproduced 3/3.
4. `rein` (interactive switcher), cwd outside any repo (`ScopeAll`, 9
   sessions) → every row `○ CANNOT RESUME` — **wrong**, reproduced 9/9.

Channels 1–3 all resolve correctly regardless of whether the working
directory is a Git repository at all (channel 2 was itself run from the
same non-repo `install` directory that produces the failure in channel 4),
which rules out "cwd is not a Git repository" as a sufficient explanation
on its own, and rules out the `Start-Process`/ConPTY harness overhead as
the cause (channels 1, 2, and 3 all run under the identical harness and are
all correct). The one variable that reliably separates the wrong outcome
from the three correct ones is: **the interactive switcher, specifically,
in `ScopeAll`** (`internal/tui/switcher/model.go`'s `New` sets
`scope = ScopeAll` whenever `opts.Project == ""`, i.e. whenever the caller
is not standing in a tracked project).

**Where this points in the source (not confirmed further — read-only
observation, no rebuild attempted against the tagged binary).**
`internal/tui/readiness/prober.go`'s `Lookup` returns `ReadinessBlocked`
immediately, without ever consulting the probe cache, when
`record.ReadOnlyReason != "" || !record.CanResume` — the only path that
produces `Blocked` without going through `FromReport` (which never returns
`Blocked` on a probe error). `rein sessions --json` shows
`"capabilities":{"resume":true,"fork":true}` for every one of the 9 records
including `grok` (consistent with Grok's already-documented tier
promotion), so `CanResume` is `true` in the index proper; whether the
switcher's own `ScopeAll` record-loading path populates the in-memory
`sessionindex.Record.CanResume`/`ReadOnlyReason` fields it hands to
`Lookup` differently than `ScopeProject`'s does was not established from
outside the binary — a maintainer with source access can confirm in
minutes what an 8-hour outside probe could only bound.

**Disposition.** Recorded `FAIL`, not carried from any prior run (all 22
rows passed 22/22 at `v0.6.0-rc.2` and `v0.6.0-rc.3`) and not attributable
to the harness by the A/B above. This is the switcher's **default**
launch condition for any user not already standing inside one of their
tracked project directories, so it is not a narrow edge case.

---

## Row 20 — full evidence

`rein init --link` requires an already-initialized BYO profile, which in
turn requires a reachable S3-compatible endpoint; per the evidence policy
this run may not point at a real coordinator-managed bucket or invent a
"local backend" shortcut (`REINSTATE_BACKEND` stays unset throughout). This
executor started the **product's own test fixture**,
`internal/backend/s3/s3test` (`NewPlainOn`, the exact fake the product's own
`pairing_test.go`/`r2_init_test.go` use to drive real CLI round trips), as a
standalone `go test`-hosted HTTP server from a throwaway, never-committed
`_test.go` file (`scripts/testing/fakes3tmp/serve_test.go`, deleted before
this commit; `git status` at commit time shows nothing under that path) —
this is the sanctioned in-repo fake, not a live third-party service and not
the memory backend the ground rules exclude.

- Device A (`REINSTATE_HOME=...\reinstate-a3`): `rein init --yes --endpoint
  http://127.0.0.1:<port> --bucket reinstate-acceptance --region us-east-1`
  with `REINSTATE_S3_ACCESS_KEY_ID`/`REINSTATE_S3_SECRET_ACCESS_KEY` set to
  a value the fake accepts → `profile_id=0032c8b5-ad41-4f5d-88fe-1a109954ccab`.
- `rein init --link` on device A → a `REIN1-...` pairing code, with "This
  code carries no keys and no passphrase."
- Device B (fresh `REINSTATE_HOME`): `rein init --paste`, code typed then
  `enter` → wizard opens at **step 1 of 8** (one extra step versus a fresh
  init's 7, for the joining-device profile-ID field) with **every** field
  already selected/pre-filled to device A's own values:
  - provider: `Other S3-compatible` (matches A)
  - endpoint: `http://127.0.0.1:<port>` (matches A, byte-for-byte)
  - bucket: `reinstate-acceptance` (matches A)
  - region: `us-east-1` (matches A)
  - key prefix: `profiles/0032c8b5-ad41-4f5d-88fe-1a109954ccab` (matches A)
  - device role: `Join a profile from another device`, pre-selected
  - profile ID (step 7): `0032c8b5-ad41-4f5d-88fe-1a109954ccab` — **the
    identical UUID device A's own `init --yes` printed**
  - review (step 8): `device   joining profile 0032c8b5-...` — confirms the
    match end-to-end

The fake server was stopped (signal file) and the throwaway test file
removed from the worktree before this commit.

---

## Row 22 — accepted classes

Every `--json` document this row compared was produced on the identical
9-session bench, once against the `v0.6.0-rc.4` install and once against
the `v0.5.1` install, and `diff`ed. Two classes account for every non-empty
diff found; no third class was found or was needed to explain any
difference in this candidate's evidence:

1. **The `agent.active` liveness check** (added before `v0.6.0-rc.1`,
   unrelated to this candidate): every `resume --dry-run --json` and
   `inspect --json` document on `v0.6.0-rc.4` carries one extra
   `environment.checks[]` entry, `{"id":"agent.active","status":"match",
   ...}`, absent from the `v0.5.1` document. Confirmed on `claude` (R1,
   W4/detached), `codex` (R2), and `grok` (B1) — the check's `message` text
   correctly names the resumed agent (`"no running claude instance..."`,
   `"no running codex instance..."`, `"no running grok instance..."`).
2. **Grok's tier promotion** (T2→T4, already documented at
   `v0.6.0-rc.1`/`rc.2`, unrelated to this candidate): `sessions --json`
   shows `grok`'s `capabilities` as `{"resume":true,"fork":true}` on
   `v0.6.0-rc.4` versus `{"resume":false,"fork":false},
   "read_only_reason":"Grok Build sessions are source-only in Phase 4"` on
   `v0.5.1`; the full `resume --dry-run --json`/`inspect --json` shape for
   the `grok` session cascades from this one capability difference (a
   resumable-agent environment report on `v0.6.0-rc.4` versus a `blocked`,
   `exit 5` compatibility refusal on `v0.5.1`) — one root cause, not a
   second and third class.

`handoff --no-launch --json` (the JSON the handoff studio's own equivalent
command exercises, per the changelog's own description of that route) was
**byte-identical** between the two versions with no diff at all. `doctor
--json` differs only in the top-level `"version"` string, which both this
candidate's own changelog and every prior tagged report treat as expected,
not part of the frozen-output guarantee.

---

## Other non-blocking notes (harness, this run)

- The interactive switcher's `ctrl+k` palette overlay renders with some
  background list text visible through/around the overlay panel in the
  ConPTY-driven capture (row 12). The top match and prompt text are both
  unambiguous and correct; this reads as an overlay-compositing quirk of
  driving a real console this way, not a scoring concern for the row's own
  assertion (subsequence match finds the right command).
- `key shift+tab`, the wizard's own documented "back" binding, is not a
  keystroke `scripts/testing/conptydriver`'s grammar can send; `key up`
  (mentioned as an alternate in the `v0.5.2` contract's own development-log
  entry) was used instead and worked identically for every back-navigation
  case exercised (rows 18, 20).
- A `send "...\n"` step reliably fails to submit a line to a plain
  (non-full-screen) `bufio.Scanner`-based prompt (the `init --paste` code
  prompt, the plain picker's `i N` inspect command) under this ConPTY
  harness; a separate `key enter` step after `send` (no trailing `\n`)
  works every time. Recorded here so a future run does not lose time
  re-discovering it.

## Cleanup

`D:\ReinstateAcceptanceProjects\v060-rc4-c\` (install, bench, driver,
snapshots, fake-S3 helper output) is this executor's own isolated
directory; nothing under it was committed. The throwaway
`scripts/testing/fakes3tmp/` helper was deleted from the worktree before
this commit (`git status` shows it absent). No transcript text, prompt, or
session id from a real host agent tree appears anywhere above — every
session id quoted is one of `scripts/tuisandbox`'s documented synthetic
UUIDs, and every profile/endpoint value quoted (Row 20) belongs to a
throwaway fake-S3 fixture this run created and tore down.
