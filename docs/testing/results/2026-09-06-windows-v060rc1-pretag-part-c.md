# v0.6.0-rc.1 pre-tag native Windows matrix — W7 executor C (CLI experience, 22 rows), 2026-09-06

Physical native-Windows acceptance of
[`docs/testing/v0.5.2-cli-experience-acceptance.md`](../v0.5.2-cli-experience-acceptance.md)'s
22-row Windows column, run against the staged GoReleaser snapshot of the
v0.6.0-rc.1 tag tree, through a real Windows pseudo console
(`scripts/testing/conptydriver`) driving `scripts/tuisandbox`'s synthetic
bench. Rows 2 and 22 additionally compare byte-for-byte against the shipped
v0.5.1 binary. Lab-root paths are redacted as `<lab-project>`; nothing below
came from a real agent transcript, a developer's real `~/.claude`/`~/.codex`
tree, or a real credential.

## Artifact identity

| Field | Value |
| --- | --- |
| Tested commit | `57c15d5225025150ed389a0923cf633b6b227302` |
| Snapshot archive | `reinstate_0.0.0-57c15d52_windows_amd64.zip` |
| Archive SHA-256 | `d58b9a46aeb32f32de01b1472597442afa4d1e010d826edd4e1c178011dc916a` (matches `checksums.txt`; verified with `Get-FileHash -Algorithm SHA256`) |
| `rein.exe` / `reinstate.exe` SHA-256 | `27316b41f4766c694bf49c8aa73c533960e9d4d90c94f717af610e72801fecfa` (both files, byte-identical) |
| `rein version --json` | `{"commit":"57c15d5225025150ed389a0923cf633b6b227302","date":"2026-09-06T01:36:16Z","name":"reinstate","version":"0.0.0-57c15d52"}` |
| Comparison binary | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches its `checksums.txt`); installed `rein version --json` reports commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb`, version `0.5.1` |
| Worktree | `v060/w7-matrix` (branch tip moved forward under concurrent executor commits during this run; this report's evidence was gathered entirely at snapshot commit `57c15d52`, unaffected by later documentation-only commits from other executors) |
| Install location | `<lab-project>\install\` (snapshot), `<lab-project>\v051\` (v0.5.1); both unzipped fresh, never overwriting a user-installed binary |
| Driver / bench | `scripts/testing/conptydriver` and `scripts/tuisandbox`, both built from the worktree at `57c15d52` into `<lab-project>\bin\`; bench root `<lab-project>\tuisandbox\` (outside any Git checkout) |
| Host OS | Windows NT 10.0.26200.0 (Windows 11), amd64, native (never WSL) |
| Shell | PowerShell 5.1 (`$PSVersionTable.PSVersion` 5.1.26100.8328) |
| Go toolchain | go1.26.1 windows/amd64 |
| Date | 2026-09-06 |

Every ambient environment variable that could leak the orchestration shell's
own state into a row (`TERM`, `TERM_PROGRAM`, `WT_SESSION`, `NO_COLOR`,
`COLORTERM`, `LANG`, `LC_ALL`, `LC_CTYPE`, `CI` and friends,
`REINSTATE_BACKEND`, `REINSTATE_MEMORY_BACKEND_DIR`, `XDG_DATA_HOME`) was
cleared before every single launch, then only what the row required was set
— this host's shell carries `TERM=xterm-256color`, `TERM_PROGRAM=Orca`,
`NO_COLOR=1`, `LANG=en_US.UTF-8`, `COLORTERM=truecolor` ambiently, all of
which would have silently invalidated the rows that test their absence or
value if left in place. `conptydriver.exe` was always launched via
PowerShell `Start-Process` **without** stdio redirection (`-WindowStyle
Hidden` only), per the driver's own documented trap; results were read back
from its `-script`/`-raw`/`snapshot` output files, never its own stdout.

## Verdict

- **Rows run:** 22 of 22 assigned.
- **PASS:** 20 (rows 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16, 17,
  18, 19, 20, 21).
- **FAIL:** 2 (rows 14, 22).
- **Release-blocking finding:** **F1 (row 14)** — the warning checklist's
  documented spacebar acknowledgement is completely non-functional on native
  Windows; root-caused to an upstream Bubble Tea Windows-specific key
  classification with no fallback in `internal/tui/readiness/checklist.go`.
  See §3.
- **Non-blocking but real findings:** F2 (row 22 frozen-output diffs,
  attributable to dated feature work, not v0.5.2), F3/F4 (test-harness gaps
  in `scripts/testing/conptydriver` found while gathering this evidence).

## 1. Row table

| Row | Description | Result | Evidence |
| - | --- | --- | --- |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | `conptydriver -cols 100 -rows 30 -- rein.exe` with `WT_SESSION=1`, `TERM` unset. Rendered frame: header `rein 9 sessions · 3 agents · all projects`, filter prompt `❯ type to filter`, time-grouped rows with `◌` pending glyphs, split preview pane, key bar `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit`. |
| 2 | Bare `rein` on a non-TTY exits `2` with the `rein sessions --json` hint, byte-identical to `v0.5.1` | PASS | `rein.exe` (stdout/stderr redirected to files, non-TTY) on both binaries: both exit `2`; both stderr are the single line `interactive session picker requires a terminal; use \`rein sessions --json\`` — byte-identical between v0.5.1 and the snapshot. (The `cmd.Help()` usage text differs only by top-level commands legitimately added between v0.5.1 and this candidate — `account`, `daemon`, `devices`, `hop`, `login`, `sync`, `whoami` — unrelated to the v0.5.2 CLI-experience frozen-output contract, which scopes the guarantee to the refusal line and exit code.) |
| 3 | `--plain` on a capable terminal falls back to the numbered switcher | PASS | `conptydriver ... -- rein.exe --plain` with `WT_SESSION=1`. Frame is the exact frozen text: `Local sessions` / `  1  claude    auth-refactor ...` / `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:`. |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Same command with `REINSTATE_NO_TUI=1`, no `--plain`. Identical frozen numbered-picker text. |
| 5 | `TERM=dumb` does the same | PASS | Same command with `TERM=dumb`. Identical frozen numbered-picker text. |
| 6 | `TERM` unset draws the switcher on Windows | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION=1` and `TERM` confirmed absent from the environment at launch (`$env:TERM` empty). Full interactive switcher rendered (same frame shape as row 1). |
| 7 | A terminal below 40x10 falls back to plain | PASS | `conptydriver -cols 30 -rows 8 -- rein.exe` with `WT_SESSION=1`. Frame is the wrapped frozen numbered-picker text (`Choose NUMBER, /text, i NUMBER...`), not the full-screen switcher. |
| 8 | `NO_COLOR` draws the switcher with no escape sequences for colour | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION=1; NO_COLOR=1`. Full interactive switcher rendered (unicode glyphs, split pane); raw byte capture scanned for `ESC[...m` (SGR) sequences found exactly one, a bare `ESC[m` reset from conhost's own console-init boilerplate — zero colour-parameter SGR codes anywhere in the stream. |
| 9 | Typing filters; the header count follows | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION=1`; after `send "auth"` the header reads `1 session · 1 agent · all projects`, filter prompt reads `❯ auth`, and only the `auth-refactor` row remains visible. *(A cosmetic capture artifact was found and root-caused while gathering this evidence — see harness defect F3; it does not indicate a product bug.)* |
| 10 | `f` in list mode filters and does **not** fork | PASS | Same session; `key f` alone produces filter prompt `❯ f` and the key bar changes from `esc quit` to `esc clear` (a filter is active). No vendor process launched, no fork confirmation, switcher still running. |
| 11 | `tab` opens the action menu; `esc` returns without acting | PASS | `key tab` → key bar becomes `r resume f fork h hand off i inspect y copy ref esc back`; `key esc` → key bar returns to the list-mode bar `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit`, switcher still running, no action taken. |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `key ctrl+k` opens the command overlay (12 commands, `esc close` key bar); `send "hof"` narrows the list to exactly one entry: `▸ Hand off to another… new session from a briefing`. |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | PASS | After a settle period, `claude:...0001` (R1, seeded-ready) resolves to `●` with preview banner `● READY TO RESUME`; `grok:...000b` (B1, read-only) shows `○` immediately (no probe needed — `Prober.Lookup` short-circuits on `ReadOnlyReason`/`!CanResume`). *(One earlier capture, taken after only a 4s settle under heavy back-to-back process load from this same test session, showed R1 transiently as `○` instead of `●`/`◌`; ground truth (`rein resume ... --dry-run --json`) was `"decision":"ready"` throughout, a clean re-run showed the correct glyph, and a different row's capture the same load window surfaced a real, session-scoped explanation — `git.shallow — the bounded Git probe timed out` — see §3 F2b. Recorded PASS on the reproducible, settled evidence; the transient is noted, not swept away.)* |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | **FAIL** | See finding **F1** in §3. The checklist opens correctly and the equivalent-command line is correct and live-updating, but `key space` (byte `0x20`) never toggles the checkbox — confirmed twice. The documented `a` (accept-all) shortcut does toggle it and does update the equivalent command to `rein resume claude:...000007 --allow-environment-warning baseline.unavailable`, proving the screen and its other input paths work; only the spacebar path is dead. |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | `rein.exe resume claude:...000003 --allow-environment-warning baseline.unavailable --allow-environment-warning git.branch` (2 of the 4 required warnings; non-interactive, no TTY needed since refusal happens before any launch): exit `7`, stderr `environment warnings require confirmation: git.working_tree, runtime.node.declaration`. |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | `key tab` → `key h` on R1 opens the studio; `policy ◂ balanced ▸` with `rein handoff claude:...0001 --to codex --policy balanced`; `key right` → `policy ◂ full ▸` / `--policy full`; `key left` ×2 → `policy ◂ checkpoint ▸` / `--policy checkpoint`. Equivalent command tracked every change. |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | Opened the studio on `codex:...000006` (B3, foreign `repository_url`): studio shows `○ this handoff cannot be planned / handoff: environment preflight is blocked`; `key enter` does not send — studio stays open and adds the status line `this handoff cannot be planned: handoff: environment preflight is blocked`. |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | Fresh, uninitialized `REINSTATE_HOME`; `rein.exe init` under ConPTY. Step 2 (Endpoint), empty value + enter → `an endpoint is required`; `not-a-url` + enter → `the endpoint must start with https:// or http://`; valid `https://s3.amazonaws.com` + enter → step 3 (Bucket); raw Shift+Tab sequence (`ESC[Z`, since the driver has no `shift+tab` key name — see harness defect F4) → back to step 2 with the typed endpoint value still `https://s3.amazonaws.com` (preserved, not discarded). |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Source inspection (`internal/tui/wizard/wizard.go` package doc + the 7 live steps captured for row 18/20: Provider, Endpoint, Bucket, Region, Prefix, Profile choice, Review) plus `rein init --help`: no access-key/secret-key/passphrase field exists anywhere in the wizard; those are read only through the pre-existing hardened `crypto.ReadSecretFD`/hidden-prompt path after the full-screen program has already exited and restored the terminal. |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | Device A (`config.toml` seeded with `storage.type=s3`, `endpoint=https://s3.us-west-2.amazonaws.com`, non-secret only): `rein.exe init --link` → prints a wrapped `REIN1-...` code with no keys/passphrase, and the line `On the other device run: rein init --paste`. Device B, fresh home: `conptydriver ... -- rein.exe init --paste`, `send` the code, `key enter` → wizard opens pre-filled with `▸ Amazon S3` selected as the storage provider (correctly decoded from the pasted endpoint), step 1 of 8 (the extra step vs. row 18's 7 is `stepProfileID`, present only on the join path). |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | `conptydriver ... -- rein.exe` with `WT_SESSION`, `TERM_PROGRAM`, and `TERM` all absent (confirmed empty at launch). Rendered frame uses the full ASCII glyph set: `/` search prompt, `>` cursor, `*` ready, `x` blocked, `|` vertical bar, `...` ellipsis, `enter` spelled out instead of `↵` — matching `internal/ui/theme.go`'s `asciiGlyphs` table exactly, none of `● ◐ ○ ◌ ▸ │ … ↵` present. |
| 22 | Every `--json` document is byte-identical to `v0.5.1` | **FAIL** | See finding **F2** in §3. `sessions --json`, `resume --dry-run --json`, `inspect --json` each differ from v0.5.1 on the identical synthetic home; `handoff list --json` and `handoff --dry-run --json` (module the test's own directory-name substrings) were byte-identical. |

## 2. Row 2 and row 22 comparison method

Both binaries ran against the **same** `scripts/tuisandbox`-generated
synthetic home (`HOME`/`USERPROFILE`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME`
identical for both runs), each with its own, separate, empty
`REINSTATE_HOME` index/cache directory so neither binary's index schema
could corrupt or be misread by the other — both then independently rescan
the same raw Claude/Codex session files under the shared home. Output was
normalized (`json.dumps(..., indent=2, sort_keys=True)`) before diffing so
key ordering differences do not register as content differences.

## 3. Findings

### F1 — release-blocking: the checklist's spacebar acknowledgement is dead on Windows (row 14)

**Root cause, confirmed in source:** Bubble Tea's own Windows input decoder
(`key_windows.go` in `github.com/charmbracelet/bubbletea@v1.3.10`, the
version pinned in `go.mod`) deliberately classifies the space bar as
`tea.KeyRunes{Runes: []rune{' '}}` on Windows, never as `tea.KeySpace`:

```go
case coninput.VK_SPACE:
    return KeyRunes // this could be KeySpace but on unix space also produces KeyRunes
```

`internal/tui/readiness/checklist.go`'s `Update` handles `tea.KeySpace`
(`c.toggle()`) but its `tea.KeyRunes` branch only recognizes the single
letters `a`, `c`/`y`, and `q` — there is no case for a rune of `' '`. The
result: a real Windows user pressing the physical spacebar in the warning
checklist sees nothing happen, cannot tick an individual warning, and (with
more than one warning) has no way to acknowledge them one at a time — only
the `a` (accept-all) shortcut still works, because it is dispatched as a
literal letter rune on both platforms.

The same root cause also reaches `internal/tui/wizard/wizard.go:287`
(`case tea.KeySpace: if m.step == stepProfile { m.joinExisting = !m.joinExisting }`),
though that screen has a working alternative — `tab`/`down` also toggles the
same choice at that step, confirmed in the same source block — so it does
not fail any of the 22 rows on its own; it is recorded here because it
shares the exact defect class and a reader fixing one should fix both.
`internal/tui/switcher/model.go`, `internal/tui/palette/palette.go`, and
`internal/tui/wizard/field.go` each also special-case `tea.KeySpace`, but
all three also have a `tea.KeyRunes` branch that unconditionally appends
whatever rune arrived (`m.filter += string(key.Runes)` /
`f.insert(key.Runes)`), so a space delivered as `KeyRunes{' '}` still
produces the correct effect there — those are not broken.

This gap was invisible to the unit/golden suite by construction:
`internal/tui/tuitest/harness.go`'s synthetic key injector builds
`tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}` directly, bypassing
Bubble Tea's real Windows decoder entirely, so every "press space" golden
test exercises a code path a real Windows keypress never takes. This is
precisely the class of defect a physical ConPTY acceptance pass exists to
catch.

**Suggested fix (not applied — this is an acceptance report, not a patch):**
add a `case tea.KeyRunes: if len(typed.Runes) == 1 && typed.Runes[0] == ' ' { c.toggle(); return c, nil }` arm (or fold the check into the existing single-character
switch) to `Checklist.Update`, and the equivalent for `wizard.Model.updateKey`'s
`stepProfile` branch.

### F2 — row 22: real, dated diffs vs v0.5.1, not v0.5.2-introduced

Diffing normalized JSON on the same synthetic home:

- **`sessions --json`**: the `grok:...000b` record's `capabilities` differ
  — v0.5.1 reports `"fork": false, "resume": false` plus
  `"read_only_reason": "Grok Build sessions are source-only in Phase 4"`;
  the snapshot reports `"fork": true, "resume": true` and no
  `read_only_reason`. This reads as an intentional Grok tier upgrade shipped
  in a release between v0.5.1 and this candidate, not a TUI-experience
  change.
- **`resume --dry-run --json` / `inspect --json`**: the snapshot's
  `environment.checks` array has one additional entry not present in
  v0.5.1 — `{"id":"agent.active","status":"match","severity":"info",
  "provenance":"current_observation","message":"no running claude instance
  is using this session"}`. This matches the Phase 5 "active session
  detected" capability and is dated after v0.5.1.
- **`handoff list --json`**: byte-identical (both empty result sets, 60
  bytes).
- **`handoff <ref> --to codex --dry-run --json`**: identical apart from the
  two test homes' own directory names appearing inside file paths the
  command legitimately echoes back (`<lab-project>\reinstate-home-old\...`
  vs. `<lab-project>\reinstate-home-new\...`) — an artifact of this report's
  own comparison setup, not a product difference.

Per the acceptance contract's own text ("Rows 2 and 22 are the frozen-output
guard. A difference in either is a release failure regardless of how good
the interactive surfaces look"), row 22 is recorded FAIL on the two real
diffs above. Neither traces to the v0.5.2 CLI-experience work; both are
consistent with legitimate, already-shipped feature growth between v0.5.1
and 57c15d52. This is reported for the release coordinator to either
confirm as accepted, intentional drift (and move this contract's comparison
baseline forward) or to treat as an undocumented behavior change — not as a
CLI-experience regression to fix in this workstream.

### F2b — transient BLOCKED glyph under sustained rapid launches (informational, not a row failure)

While gathering row 13 and row 14 evidence, two captures taken back-to-back
with many prior ConPTY launches in quick succession on this host showed a
readiness result more pessimistic than ground truth: R1 briefly rendered
`○` (blocked) instead of `●`, and a separate capture of the same session
(W3) showed the preflight failing outright with `git.shallow — the bounded
Git probe timed out` where the underlying repository state was, and
remained, clean (`git status --short` empty, branch matching). A fresh
retry immediately after, at lower load, produced the correct result both
times. This looks like a bounded-probe timeout tuned tighter than this
host's process contention under many concurrent/rapid `git`/vendor-shim
subprocess launches from the test session itself, not a defect in any of
the 22 rows' own logic — recorded for whoever tunes `preflight`'s bounded
Git probe timeout, not as a release blocker.

### F3 — harness defect: `scripts/testing/conptydriver`'s VT model has no ECH (`CSI Ps X`) support

Root-caused while investigating an apparent stray leading digit in row 9's
captured header (`9 1 session · 1 agent · all projects`). The raw byte
capture shows the real product correctly issuing standard VT erase-then-redraw
(`ESC[H ... ESC[60X ESC[38;2;139;147;158m ESC[60C1 session · 1 agent · all
projects `) to clear the old, longer header before drawing the new, shorter
one. `scripts/testing/conptydriver/vtscreen.go`'s `applyCSI` switch has no
case for `'X'` (ECH); its documented `default` behaviour is "parsed and
discarded, not left in the byte stream" — so the erase never happens in the
harness's own screen model, leaving the stale character from the previous
frame. A real terminal (Windows Terminal, conhost itself) implements ECH and
renders this correctly; this is a capture-tool gap, not a product bug. It
also affected the row 12 palette-narrowing capture in the same way (stale
`Run diagnostics` / `4 more matches` text bleeding through under a shrunk
overlay). Suggested fix: add a `case 'X':` to `vtscreen.go`'s `applyCSI` that
blanks `get(0, 1)` cells forward from the cursor without moving it, mirroring
`eraseLine`'s cell-blanking loop.

### F4 — harness gap: `scripts/testing/conptydriver` has no `shift+tab` key name

The step-script grammar's `key` verb (`script.go`'s `KeyBytes`) recognizes
`enter`, `esc`, `tab`, `up`/`down`/`left`/`right`, `space`, `backspace`,
`ctrl+X`, and single characters, but not `shift+tab` — yet the wizard's own
key bar advertises `shift+tab back` as the way to go back a step (row 18).
A `key shift+tab` step fails to parse (`unknown key "shift+tab"`), silently
aborting the rest of the script (visible only as the driver process being
killed at the exit-timeout, with no further frames). Worked around for this
report by sending the raw sequence directly (`send "\x1b[Z"`, which Bubble
Tea's key table maps to `KeyShiftTab`); a future revision of the driver
should add `shift+tab` (and, for the same reason, probably `shift+left`/
`shift+right`/`shift+up`/`shift+down`) to `KeyBytes`.

## 4. Rows not run / deviations

None. All 22 rows were exercised against the artifact; none were skipped as
NOT TESTED.
