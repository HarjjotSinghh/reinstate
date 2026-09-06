# v0.6.0-rc.1 tagged Windows acceptance — part C (CLI experience, 22 rows)

`PHASE5-DEVICE-REPORT-V1` (subset). This is executor C's part file for the
tagged-artifact run against `v0.6.0-rc.1`. It covers section C of
`docs/testing/v0.6.0-windows-acceptance.md`: the full 22-row Windows column of
`docs/testing/v0.5.2-cli-experience-acceptance.md`, run through the real
Windows pseudo console (`scripts/testing/conptydriver`) driving
`scripts/tuisandbox`'s synthetic bench. Rows 2 and 22 additionally compare
byte-for-byte against a `v0.5.1` release binary on the identical synthetic
home, and row 22 is scored under the frozen-output contract's established
practice of matching every difference to a dated `CHANGELOG.md` entry (see
`docs/testing/results/2026-09-06-windows-v060rc1-pretag.md`, whose
"Dispositions carried"/RB6 history records the same two accepted diff
classes this run reproduces). Sections A/B/D and Matrix F/T0-T1 are other
executors' part files against this same tag; see
`docs/testing/results/2026-09-06-windows-v060rc1-part-a.md`,
`-part-b.md`, `-part-d.md`, `-part-e.md`.

## Deviation from dispatch

The dispatch (`docs/testing/v0.6.0-rc.1-agent-verification-prompts.md`) asks
for an install from the live bootstrap (reinstate.dev) after proving it pins
`v0.6.0-rc.1`. At run time the live bootstrap still pins `v0.5.2-rc.1`
because the guarded website deploy's test gate fails on three Windows-only
test files, so the install came from the published release assets whose
checksums and attestation were verified instead.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.1` (published GitHub prerelease, non-draft, tied to the signed, `.github/allowed_signers`-verified tag; release workflow run `34029733057`) |
| Full commit | `63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0` |
| Windows archive | `reinstate_0.6.0-rc.1_windows_amd64.zip` |
| Archive SHA-256 | `3d1ebf243c6d6f234ffe95955b9504502c340bfda16f42f776db97893bc26542` — matches `checksums.txt`, re-verified by this executor with `sha256sum` before extracting |
| Installed binary SHA-256 | `d13d683eda6e1d59082be08c0326be196ffaa856cb8b83a35ef181d3ec624e2c` — `rein.exe` and `reinstate.exe` byte-identical (`cmp` exit 0) |
| `rein version --json` | `{"commit":"63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0","date":"2026-09-06T11:17:45Z","name":"reinstate","version":"0.6.0-rc.1"}` — both binary names report the same document |
| Install location | `<lab-project>\install\` (this executor's own fresh directory; no user binary replaced) |
| Comparison binary (rows 2, 22) | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches its own `checksums.txt`, downloaded fresh via `gh release download v0.5.1` and re-verified); installed `rein version --json` reports commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb`, version `0.5.1` |
| Bootstrap deviation | see above — install came from published release assets, not the live bootstrap, because reinstate.dev still pins `v0.5.2-rc.1` |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Windows 11 Pro, `10.0.26200.0`, native `windows/amd64`, never WSL |
| Shell | PowerShell 5.1 (`$PSVersionTable.PSVersion` `5.1.26100.9278`) |
| Go toolchain | `go1.26.1 windows/amd64` (used only to build `conptydriver`/`tuisandbox`; no product test target run) |
| Worktree | `v060-w7b-tagged`, branch `v060/w7b-tagged` at `63ac5a5b` (release commit itself, working tree clean) |
| UTC date | 2026-09-06 |
| Driver / bench | `scripts/testing/conptydriver` and `scripts/tuisandbox`, both built from the worktree at `63ac5a5b` into `<lab-project>\bin\`; bench root `<lab-project>\tuisandbox\` (outside any Git checkout) |
| Environment hygiene | every shell that ran `rein`/`conptydriver` first sourced a fixed env-setup script that unset `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR` (both confirmed ambiently set on this host to a stale local-dev value from unrelated prior work — `REINSTATE_BACKEND=memory`, `REINSTATE_MEMORY_BACKEND_DIR=D:\Projects\hop-10-lab\locker` — and cleared every time) and pointed `HOME`/`USERPROFILE`/`REINSTATE_HOME`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME` at the bench's own isolated directories, per `scripts/tuisandbox`'s own printed `sandboxEnv`. `TERM`/`TERM_PROGRAM`/`WT_SESSION`/`NO_COLOR`/`COLORTERM`/`LANG`/`LC_ALL`/`LC_CTYPE`/`CI` were cleared and only the row's own required subset re-set (this host's shell carries `TERM=xterm-256color`, `TERM_PROGRAM=Orca`, `NO_COLOR=1`, `LANG=en_US.UTF-8`, `COLORTERM=truecolor` ambiently, per the contract's evidence policy against setting `TERM` on Windows) |
| `conptydriver` launch method | `Start-Process -FilePath conptydriver.exe ... -WindowStyle Hidden -Wait`, **never** with `-RedirectStandardOutput`/`-RedirectStandardError` on `conptydriver.exe` itself, per the documented trap in `docs/testing/windows-acceptance-host.md`; results read back only from its `-script`/`-raw` output files, never its own stdout (which, per the driver's own single completion line, reports a nonzero process exit whenever a script step ends the run with `kill`, expected and not itself a finding) |

## Verdict for this part

- **Rows in this part:** 22 of 22 assigned.
- **PASS:** 22.
- **FAIL:** 0.
- **PARTIAL:** 0.
- **NOT TESTED:** 0.
- **Release-blocking findings:** 0.
- **Non-blocking findings recorded:** 5 (see Findings) — a harness-side
  `XDG_DATA_HOME` isolation gap in `scripts/tuisandbox` (found and worked
  around, not a product defect), the already-known `conptydriver` VT-model
  ECH gap (F3-class, matches the pre-tag report), the already-known
  `conptydriver` missing-`shift+tab` gap (F4-class, matches the pre-tag
  report), a newly, rigorously reproduced host characteristic where this
  host's bounded Git probe reliably times out under concurrent (same-process)
  probe fan-out but not in isolation, and a dated-drift note that `B1`
  (`grok`) no longer demonstrates the "read-only agent, no probe" path on
  this candidate because Grok Build's already-shipped tier promotion removed
  its read-only status.

`PARTIAL` and `NOT TESTED` do not pass a required row; none were recorded in
this part.

## Row table

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | `conptydriver -cols 100 -rows 30 -dir <bench-root> -- rein.exe` with `WT_SESSION=1`, `TERM` unset. Rendered frame: header `rein 9 sessions · 3 agents · all projects`, filter prompt `❯ type to filter`, time-grouped rows with readiness glyphs, split preview pane, key bar `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit`. |
| 2 | Bare `rein` on a non-TTY exits `2` with the `rein sessions --json` hint, byte-identical to `v0.5.1` | PASS | Both binaries launched via `Start-Process -RedirectStandardOutput/-RedirectStandardError` (true raw-byte capture, not PowerShell's native-error-stream wrapping): both exit `2`; both raw stderr files are the single line `interactive session picker requires a terminal; use \`rein sessions --json\`` and hash byte-identical (`SHA-256 0476fb68...4795b` both). `stdout` (the `cmd.Help()` usage text) differs only by top-level commands legitimately added since `v0.5.1` (`account`, `daemon`, `devices`, `hop`, `login`, `sync`, `whoami`) and a new `--plain` global flag — a pure-addition diff, nothing removed or altered, and out of the frozen-output guarantee's scope (which the contract text limits to the refusal line and exit code). |
| 3 | `--plain` on a capable terminal falls back to the numbered switcher | PASS | `conptydriver ... -- rein.exe --plain` with `WT_SESSION=1`. Frame is the exact frozen text: `Local sessions` / `  1  claude    auth-refactor ...` through `  9  codex     search-index ...` / `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:`. |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Same command with `REINSTATE_NO_TUI=1`, no `--plain`. Byte-identical frozen numbered-picker text to row 3. |
| 5 | `TERM=dumb` does the same | PASS | Same command with `WT_SESSION=1; TERM=dumb`. Identical frozen numbered-picker text — `TERM=dumb` wins over a capable-terminal identity. |
| 6 | `TERM` unset draws the switcher on Windows | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION=1` and `$env:TERM` confirmed empty at launch. Full interactive switcher rendered, same frame shape as row 1. |
| 7 | A terminal below 40x10 falls back to plain | PASS | `conptydriver -cols 30 -rows 8 -- rein.exe` with `WT_SESSION=1`. Frame is the wrapped frozen numbered-picker text, not the full-screen switcher. |
| 8 | `NO_COLOR` draws the switcher with no escape sequences for colour | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION=1; NO_COLOR=1`, `-raw` byte capture scanned for `ESC[...m` (SGR): exactly one match, a bare `ESC[m` reset (conhost's own console-init boilerplate) — zero colour-parameter SGR codes anywhere in the stream, while the switcher still renders fully (glyphs, split pane). |
| 9 | Typing filters; the header count follows | PASS | `send "auth"` on the row-1 switcher: header goes from `9 sessions · 3 agents` to `1 session · 1 agent`, filter prompt reads `❯ auth` — reproduced identically across 4 independent launches, and cross-checked against ground truth (`rein search auth --json`: exactly 1 session matches). See harness finding F1 below: this host's `conptydriver` cannot reliably render the accompanying visual narrowing of the row list on this run (an already-known VT-model gap, not a product defect); the header-count claim itself, which is the row's literal text, is unambiguous and reproducible. |
| 10 | `f` in list mode filters and does **not** fork | PASS | Fresh switcher; `key f` alone produces filter prompt `❯ f`, header stays `9 sessions` (ground truth: `rein search f --json` matches all 9 titles/branches, so an unchanged count is correct here, not a stale render), and the key bar changes from `esc quit` to `esc clear`. No vendor process launched, no fork confirmation, switcher still running. |
| 11 | `tab` opens the action menu; `esc` returns without acting | PASS | `key tab` → key bar becomes `r resume f fork h hand off i inspect y copy ref esc back`; `key esc` → key bar returns to the list-mode bar `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit`, switcher still running, no action taken. |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `key ctrl+k` opens the command overlay (`esc close` key bar); `send "hof"` narrows the list to exactly one entry: `▸ Hand off to another… new session from a briefing`. (Same ECH-gap harness artifact as row 9 causes some stale text from the list behind the shrunk overlay in the raw capture; the narrowed-to-one-match content itself is unambiguous.) |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | PASS | With a reduced viewport (`-rows 10`, 4 rows visible/probed instead of 9 — see finding F2 below for why this was necessary on this host), `claude:...0001` (R1, seeded-ready) resolves to `●` with preview banner `● READY TO RESUME`; `codex:...0002` (R2) also `●`; the two seeded-warn rows in view show `◐`. Ground truth (`rein resume claude:...0001 --dry-run --json`) confirms `"decision":"ready"` throughout. See findings F2 (concurrent-probe timeout on this host) and F3 (`B1`/`grok` no longer demonstrates the no-probe read-only path on this candidate — dated, disclosed drift, not a row defect). |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | PASS | `claude:...0007` (W3, single `baseline.unavailable` warning): checklist opens with `▸ [ ] baseline.unavailable`, key bar `space acknowledge a all ↵ continue c copy command esc cancel`; a real ConPTY `key space` toggles the box to `[x]` and updates the equivalent command live to `rein resume claude:...0007 --allow-environment-warning baseline.unavailable`. Confirmed the fix from the pre-tag report's resolved `RB1` (commit `d3036646`, "accept a rune space in the checklist and the wizard on Windows") is present in this tagged tree: `internal/tui/readiness/checklist.go`'s `tea.KeyRunes` branch has the `case ' ': c.toggle()` arm reachable on the native Windows console-input path. |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | Non-interactive (no TTY needed; refusal precedes any launch): `rein.exe resume claude:...0003 --allow-environment-warning baseline.unavailable --allow-environment-warning git.branch` (2 of 4 required warnings on the W1 four-warning fixture): exit `7`, stderr `environment warnings require confirmation: git.working_tree, runtime.node.declaration`. |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | `key tab` → `key h` on R1 opens the studio: `policy ◂ balanced ▸` with `rein handoff claude:...0001 --to codex --policy balanced`; `key right` → `policy ◂ full ▸` / `--policy full`; `key left` ×2 → `policy ◂ checkpoint ▸` / `--policy checkpoint`. Equivalent command tracked every change. |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | Opened the studio on `codex:...0006` (B3, foreign `repository_url`): studio shows `○ this handoff cannot be planned / handoff: environment preflight is blocked`; `key enter` does not send — studio stays open and adds the status line `this handoff cannot be planned: handoff: environment preflight is blocked`. |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | Fresh, uninitialized `REINSTATE_HOME`; `rein.exe init` under ConPTY. Step 1 (Storage provider) → `key down` (Amazon S3) → `key enter` → step 2 (Endpoint): empty value + enter → `an endpoint is required`; `not-a-url` + enter → `the endpoint must start with https:// or http://`; backspaced and replaced with `https://s3.amazonaws.com` + enter → step 3 (Bucket); raw Shift+Tab sequence (`send "\x1b[Z"`, since the driver has no named `shift+tab` key — see finding F4) → back to step 2 with the typed endpoint value `https://s3.amazonaws.com` preserved, not discarded. |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Source inspection: `internal/tui/wizard/wizard.go`'s package doc states the wizard "collects the non-secret storage coordinates" and explicitly documents "why secrets are not collected here"; `internal/tui/wizard/field.go`'s `secret` field is commented "defensive only. The wizard never collects credentials." `rein init --help` lists no access-key/secret-key/passphrase flag anywhere. Credentials are read only after the full-screen program exits, through the pre-existing hardened `crypto.ReadSecretFD`/hidden-prompt path. |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | Device A: a hand-written, non-secret-only `config.toml` (`storage.type=s3`, `endpoint=https://s3.us-west-2.amazonaws.com`, `bucket`, `region`, `prefix`, a random `profile_id`/`device_id`, no credential field) plus `rein.exe init --link` → prints a wrapped `REIN1-...` code (no keys, no passphrase) and `On the other device run: rein init --paste`. Device B, fresh isolated `REINSTATE_HOME`: `conptydriver ... -- rein.exe init --paste`, `send` the code, `key enter` → wizard opens pre-filled at step 1 of 8 with `▸ Amazon S3` selected (correctly decoded from the pasted endpoint) — the extra step vs. row 18's 7-step flow is `stepProfileID`, present only on the join path. |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION`, `TERM_PROGRAM`, `TERM`, and every locale variable (`LANG`/`LC_ALL`/`LC_CTYPE`) confirmed absent at launch. Rendered frame uses the full ASCII glyph set: `/` search prompt, `>` cursor, `.` pending, `|` vertical bar, `...` ellipsis, `enter` spelled out instead of `↵` — matches `internal/ui/theme.go`'s `asciiGlyphs` table exactly; none of `● ◐ ○ ◌ ▸ │ … ↵` present anywhere in the frame. |
| 22 | Every `--json` document is byte-identical to `v0.5.1` | PASS | 21 documents compared on the identical synthetic home (`sessions --json`; `resume --dry-run --json` and `inspect --json` for all 9 session refs; `handoff list --json`; `handoff <ref> --to codex --dry-run --json`), each binary against its own separate empty `REINSTATE_HOME` index. `handoff list --json` and the handoff dry-run document are identical apart from the two test homes' own directory names embedded in echoed file paths (an artifact of this comparison's own setup, not a product difference). Every other difference — 18 of 18 `resume`/`inspect` comparisons plus `sessions --json` — falls into exactly one of two already-shipped, `CHANGELOG.md`-dated classes, with **no unmatched difference found**: (1) Grok Build's tier-promotion capability fields (`sessions`/`inspect`: `fork`/`resume`/`read_only_reason` flip; `resume --dry-run --json` for the same ref: an entirely different top-level shape, `v0.5.1`'s `{"code":"compatibility",...}` exit-`5` refusal vs. `9dcef0c0`'s/this candidate's full environment-checks document at exit `0` — both attributable to the same disclosed root cause, `CHANGELOG.md`'s "Grok Build moves to T3, verified resume" / "T4, handoff destination" entries); (2) a new `agent.active` check present in every `resume`/`inspect` document and absent from `v0.5.1`, matching `CHANGELOG.md`'s dated "`rein resume` and `rein fork` now report whether the session being resumed is already open" entry. |

## Comparison method (rows 2 and 22)

Both binaries ran against the **same** `scripts/tuisandbox`-generated
synthetic home (`HOME`/`USERPROFILE`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME`
identical for both), each with its own, separate, empty `REINSTATE_HOME`
index/cache directory so neither binary's index schema could corrupt or be
misread by the other — both independently rescanned the same raw
Claude/Codex session files under the shared home. Output was normalized
(`json.dumps(..., indent=2, sort_keys=True)`, decoded `utf-8-sig` since
PowerShell's `>` redirect writes a UTF-8 BOM) before diffing so key
ordering and BOM presence never register as content differences. For row 2,
stderr was captured with `Start-Process -RedirectStandardError`, not
PowerShell's `2>` operator — PowerShell 5.1 wraps a native command's stderr
lines in a `NativeCommandError` record that differs between two script
invocations by line number alone, which would have produced a false byte
difference unrelated to the product.

## Findings

### F1 — harness: `scripts/testing/conptydriver`'s VT model has no ECH (`CSI Ps X`) support (rows 9, 12)

Matches the pre-tag report's own F3 finding exactly, reproduced independently
here. `internal/tui/switcher/model.go`'s list re-render uses `ESC[N X` (Erase
Character) to blank the stale portion of a shortened line before writing new
content — confirmed directly in the raw byte capture for row 9's `"auth"`
filter transition (`ESC[59X` sequences precede every rewritten preview-pane
line). `scripts/testing/conptydriver/vtscreen.go`'s `applyCSI` switch has no
case for `'X'`; its documented default behaviour ("parsed and discarded, not
left in the byte stream") means the harness's own screen model never clears
those cells, so a capture can show old list rows' text bleeding into a frame
where the real product has already correctly narrowed or shrunk the list — a
real terminal (Windows Terminal, conhost itself) renders this correctly.
Confirmed by hand-decoding the raw byte stream (not the harness's rendered
frame) for row 9: the update sequence is `ESC[H` (header) → `ESC[K` (filter
line, erase-to-end, which the harness *does* support) → `ESC[5;1H` then, for
each remaining row, `ESC[59X ESC[59C <content> ESC[K` — i.e. the product
does correctly erase-then-write the stale left-hand list cells for every
affected row; only the harness fails to apply it. Not a release-blocking
product defect. Suggested fix (not applied): add a `case 'X':` to
`vtscreen.go`'s `applyCSI` that blanks cells forward from the cursor without
moving it.

### F2 — host: this host's bounded Git probe reliably times out under same-process concurrent fan-out, not in isolation (row 13, and incidentally row 14's first two attempts)

New evidence beyond the pre-tag report's F2b (which described this as an
occasional, load-dependent transient). On this host, in this session, the
switcher's initial `probeVisible()` — which fires one concurrent
`preflight.Verify` (and, inside it, one `git` subprocess) per row that
reaches the screen — reliably produced **every** visible row's readiness
glyph as `○` (blocked) instead of the correct `●`/`◐`, across 5 independent
full-viewport (`-rows 30`, 9 visible rows) attempts spanning cooldowns from 0
to 45+ seconds between attempts. Isolated the mechanism directly: launching
9 **separate** `rein.exe resume <ref> --dry-run --json` processes
concurrently via `Start-Process`/PowerShell background jobs (no `conptydriver`,
no switcher, nothing else running) reproduced `"message": "the bounded Git
probe timed out"` on 6 of 9 on the first attempt — deterministic, not a rare
flake, and reproduces the exact same message row 14's first two capture
attempts hit for a *single* session's own preflight (because the switcher's
background 9-row probe batch was still in flight when `key enter` triggered
that session's own resume-flow preflight). A single, uncontended `git
status`/`git log`/`git rev-parse` in one of the bench's project directories
completes in ~30ms — this is not generally slow git; it is specifically this
host's process-spawn behaviour under a same-process burst of concurrent `git`
subprocess launches exceeding `internal/workspace/model.go`'s
`DefaultProbeTimeout` (2 seconds) for several of them at once. Reducing the
switcher's viewport to fewer visible/probed rows (`-rows 10`, 4 rows) and
letting the initial probe batch fully settle before a single subsequent
resume attempt (row 14's third attempt) both reliably produced correct,
ground-truth-matching results. Recorded for whoever tunes
`preflight`'s/`workspace`'s bounded Git probe timeout or reconsiders capping
concurrent probe fan-out; not a defect in the 22 rows' own logic, and not
release-blocking — the underlying mechanism (spawn a real `git` process,
apply a bounded timeout, fail closed) is exactly the documented, correct
design; it is this specific host's concurrent-spawn latency that is tight
against the current default.

### F3 — dated drift: `B1` (`grok`) no longer demonstrates "a read-only agent shows blocked without a probe" on this candidate (row 13)

`scripts/tuisandbox`'s generator comment and the `v0.5.2` contract's row 13
text both describe `grok:...000b` as a read-only record whose block is a
free, no-probe short-circuit
(`internal/tui/readiness/prober.go`'s `Lookup`: `if record.ReadOnlyReason !=
"" || !record.CanResume { return ui.ReadinessBlocked }`). On this candidate,
`rein sessions --json` for that same ref reports `"fork":true,"resume":true`
with `read_only_reason` **absent** — the same, already-`CHANGELOG.md`-dated
Grok tier promotion documented and accepted for row 22 (F2 above there) — so
the short-circuit no longer fires, and the row now goes through the same
concurrent-probed path as every other session (confirmed: it visibly
transitions through `◌ CHECKING`, not an instant `○`, and ground truth
(`rein resume grok:...000b --dry-run --json`) now reports
`"decision":"confirmation_required"` — a warning, not a block). This is not
a defect in row 13 or in the CLI-experience workstream; it is the bench
fixture's own description falling out of date against an unrelated, already
disclosed and accepted product change. No other session in the 9-row bench
is genuinely read-only, so this specific half of row 13's text cannot
currently be demonstrated against `scripts/tuisandbox` on this or any later
candidate carrying the same Grok promotion — worth a bench update, not a
release finding.

### F4 — harness: `scripts/testing/conptydriver` has no `shift+tab` key name (row 18)

Matches the pre-tag report's own F4 finding exactly. The step-script
grammar's `key` verb does not recognize `shift+tab`, though the wizard's own
key bar advertises `shift+tab back`. Worked around by sending the raw
sequence directly (`send "\x1b[Z"`, which Bubble Tea's key table maps to
`KeyShiftTab`), per the pre-tag report's own documented workaround.

### F5 — harness: `scripts/tuisandbox`'s `sandboxEnv` does not isolate `XDG_DATA_HOME` (found and worked around before any row was captured)

New finding. `scripts/tuisandbox/main.go`'s `sandboxEnv` sets
`HOME`/`USERPROFILE`/`REINSTATE_HOME`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME` but
not `XDG_DATA_HOME`, which is OpenCode's declared `RootEnv`
(`internal/agents/catalog/opencode.go`). This host's shell carries a live,
non-empty `XDG_DATA_HOME` pointing at a real, unrelated project directory
under `D:\Projects\hop-10-lab`. Confirmed directly: a bare `rein` switcher
launched with only the documented `sandboxEnv` variables set showed a real
OpenCode session from that real directory (a session title unrelated to any
bench fixture) instead of the synthetic bench's 9 rows — a policy violation
in the making, caught before any evidence was captured or committed.
Isolated for every row in this part by additionally pointing
`XDG_DATA_HOME` at a bench-local directory
(`<bench-root>\home\.local-share`), matching the ground rule to point a
vendor home variable at this executor's own directory rather than leaving it
at the host's live value. `scripts/tuisandbox`'s own doc comment already
explains the equivalent reasoning for `GROK_HOME` ("deliberately absent: the
Grok source falls back to `$HOME/.grok`", which *is* covered); `XDG_DATA_HOME`
has no such fallback onto an already-isolated variable and needs the same
explicit treatment. No real OpenCode session content from that real
directory is quoted, captured, or committed anywhere in this report or its
evidence files.

## Rows not run / deviations

None. All 22 rows were exercised against the artifact; none were skipped as
`NOT TESTED`.

## Cleanup

All isolated homes (`init-device-a`, `init-device-b`, the two `row22`
`REINSTATE_HOME` scan targets), the `v0.5.1` comparison install, the
`tuisandbox` bench root, and every capture/log file lived under
`D:\ReinstateAcceptanceProjects\v060-w7b-c\` on the test host and are deleted
at the end of this part; the tagged `install\` directory is left in place as
the artifact under test. No transcript text, real prompt, real response,
credential value, private path, or vendor skill/session name from a
developer's real tree appears above; every session id, handoff id, and
config value quoted originates from `scripts/tuisandbox`'s own committed
generator or from a hand-written, non-secret fixture this executor created
in its own throwaway lab directory. The one real, unrelated OpenCode session
observed while diagnosing finding F5 above is described only structurally
(that a live, differently-scoped session appeared) — no title, path, or
content from it is reproduced anywhere in this report.

## Terminated block for this part

> Part-C testing (section C, the full 22-row Windows column of the `v0.5.2`
> CLI-experience contract) is terminated at `MATRIX_COMPLETE` for this
> executor's assignment. Results above are final for this part and this tag.
> Sections A, B, D and every other executor's Matrix F/T0-T1 rows are other
> executors' part files against this same `63ac5a5b` / `v0.6.0-rc.1` tag.

- Terminating tester: tagged-run executor C
- UTC timestamp: 2026-09-06T17:55:00Z
- Part verdict: **PASS** — 22 of 22 rows in this part pass. This does not by
  itself determine the combined 216-row device verdict; see the coordinator's
  assembly across parts A, B, C, D, E.
