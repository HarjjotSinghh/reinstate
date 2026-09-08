# `v0.6.0-rc.8` tagged-artifact native Windows acceptance — part C

`PHASE5-DEVICE-REPORT-V1` (partial — this file covers only this executor's
assigned rows: the 22-row `v0.5.2` CLI experience contract, section C of
the `v0.6.0` Windows acceptance contract, through `scripts/testing/conptydriver`
against `scripts/tuisandbox`. Section A (automated gates), the rest of the
178-row Phase 5 matrix, and section D (Hop parity journeys) are other
executors' parts and are not duplicated here.)

Contract: [`v0.6.0-rc.8-agent-verification-prompts.md`](../v0.6.0-rc.8-agent-verification-prompts.md),
composing [`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md),
[`phase-5-universal-agent-coverage-acceptance.md`](../phase-5-universal-agent-coverage-acceptance.md)
and [`v0.5.2-cli-experience-acceptance.md`](../v0.5.2-cli-experience-acceptance.md).

## Verdict (this part)

- **Part verdict:** `PASS`
- **Rows in this part:** 22/22 required CLI experience rows.
- **Result:** **22 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED.**
- **Release-blocking findings from this part:** `0`.
- **CLI row 13 — this candidate's own targeted fix — is confirmed.** All
  three parts of the expanded method pass: 15 independent `ScopeAll`
  launches every one fully matching ground truth on every one of the 9
  fixture rows (45/45 checkpoints across three settle checkpoints per
  launch), `ScopeProject` unaffected, and the `-stale-claude` settle check
  confirms every Claude row lands on and stays on `○ Blocked` (never `◌`)
  across a 100-second window well past `ProbeTimeout × maxProbeRetries`.
  This directly reverses `v0.6.0-rc.7`'s tagged `FAIL` on this row (11 of
  15 launches wrong).
- One **harness defect** found and fixed locally in
  `scripts/testing/conptydriver/vtscreen.go` (missing `CSI Ps X` / ECH
  support), not committed per the ground rules — see §6. It produced a
  misleading capture on this executor's first attempt at row 9 only; every
  row in this report's table was captured (or re-captured) with the fix in
  place, except rows 1 and 3–8, 11–13, 21, whose captures never depend on
  an in-place erase shorter than the old content (verified individually,
  see §6).

## 0. Ground rules honored

- Worked only in `D:\Projects\reinstate-worktrees\v060-rc8-tagged` (branch
  `v060/rc8-tagged`, `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f`). Never
  touched `D:\Projects\reinstate` or other worktrees; no push, no merge, no
  `gh` write call (`gh release view` only, read-only, in artifact-identity
  verification already performed by the coordinator and re-confirmed here
  from the installed binary).
- Every shell that ran `rein`, `go build`, or `go test` first ran
  `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (Bash) or
  `Remove-Item Env:REINSTATE_BACKEND, Env:REINSTATE_MEMORY_BACKEND_DIR`
  (PowerShell, inside the reusable harness script — see §5). Neither
  variable was ever observed set on this host.
- No rows in this part touch a real vendor agent's live session data:
  every session used is `scripts/tuisandbox`'s synthetic fixture set,
  generated fresh into `D:\ReinstateAcceptanceProjects\v060-rc8-c\tuisandbox\`
  (and a second `-stale-claude` variant into `...\tuisandbox-stale\`), both
  roots outside any Git checkout. `CLAUDE_CONFIG_DIR` and `CODEX_HOME` were
  pointed at the synthetic home for every row in this part (never the
  host's live values — this part never runs a real vendor CLI session).
  `XDG_DATA_HOME` and every other agent's declared `root_env`
  (`GEMINI_CLI_HOME`, `KIMI_CODE_HOME`, `CLINE_DATA_DIR`, `COPILOT_HOME`,
  `CURSOR_CONFIG_DIR`, `PI_CODING_AGENT_DIR`, `QWEN_HOME`,
  `OH_PERSISTENCE_DIR`) were also redirected to empty, non-existent
  directories under `D:\ReinstateAcceptanceProjects\v060-rc8-c\isolated-empty\`
  — this host carries a persistent, real `XDG_DATA_HOME` (confirmed at
  `[Environment]::GetEnvironmentVariable('XDG_DATA_HOME','Process')`, a
  real path under this user's own OpenCode data), exactly the leak
  `v0.6.0-rc.7`'s own part C caught; every switcher launch in this part was
  independently confirmed to report exactly `9 sessions · 3 agents` with
  no extra rows, for every one of the 15+ launches, not sampled once.
- Every session id, project name, and title in this report is a
  `scripts/tuisandbox` fixture id, never a real host session id. No real
  transcript text, prompts, credentials, or private paths appear below.
- `scripts/testing/conptydriver` was built from this exact worktree
  (`go build ./scripts/testing/conptydriver`, `GOTOOLCHAIN=go1.25.13`) into
  `D:\ReinstateAcceptanceProjects\v060-rc8-c\conptydriver\conptydriver.exe`
  — never the tagged product binary, and never used as a substitute for it;
  the binary under test in every row is the checksummed, extracted
  `rein.exe` from the coordinator's `rc8-draft` directory.
- Every `conptydriver` invocation was launched via PowerShell
  `Start-Process -FilePath ... -WindowStyle Hidden -Wait` with **no**
  `-RedirectStandardOutput`/`-RedirectStandardError` on the driver process
  itself (`-raw <file>` and the step script's own `snapshot <path>` verb
  captured results to files instead) — the exact launch discipline
  `scripts/testing/conptydriver/README.md` and
  `docs/testing/windows-acceptance-host.md` require, confirmed necessary by
  reproducing the trap once (a redirected launch reported
  `interactive session picker requires a terminal`).
- The tagged `rein.exe` was always invoked by its own full path
  (`D:\ReinstateAcceptanceProjects\v060-rc8-c\install\rein.exe`), never a
  PATH-resolved `rein`/`reinstate`, including inside every `conptydriver`
  step script's own `--` argument.
- Rows 2 and 22 compared against `reinstate_0.5.1_windows_amd64.zip`
  (checksum-verified against the coordinator's `v051` scratch directory
  before use), extracted into this executor's own
  `D:\ReinstateAcceptanceProjects\v060-rc8-c\v051\` directory.
- All isolated homes, shims, and pairing directories under
  `D:\ReinstateAcceptanceProjects\v060-rc8-c\` are deleted after this
  report is committed; only this results file, on the shared branch, is
  committed by this part.

## 1. Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.8`, published GitHub prerelease |
| Full commit | `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f` (worktree `HEAD`) |
| Windows archive | `reinstate_0.6.0-rc.8_windows_amd64.zip` |
| Archive SHA-256 | `658dc27e607fdec14d14156dfe3edaf9d685bfa9add1e467d604ab0fa298e28f` — matches `checksums.txt` in the coordinator's pre-verified `rc8-draft` directory, independently recomputed by this executor |
| Installed `rein.exe`/`reinstate.exe` SHA-256 | `482b5de12db1312c1a767bcf40c7999de9f8cef7f47bff5bce802fab960c3b71` — byte-identical (`cmp` exit 0) |
| `rein version --json` | `{"commit":"3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f","date":"2026-09-08T22:38:21Z","name":"reinstate","version":"0.6.0-rc.8"}` |
| Install location | `D:\ReinstateAcceptanceProjects\v060-rc8-c\install\` (this executor's own fresh directory, extracted from the coordinator's checksummed `rc8-draft` archive, not built and not a user binary) |
| Previous-release comparison binary | `reinstate_0.5.1_windows_amd64.zip`, archive SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches the coordinator's `v051/checksums.txt`); extracted `rein.exe`/`reinstate.exe` SHA-256 `82b6b431db27fb2b3a1178680a66093e78c745322edb680933ce3850d078cc67`, `rein version --json` reports `"version":"0.5.1"`, commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb` |

**Bootstrap deviation (assigned methodology, not an unplanned deviation).**
Per this run's ground rules, only executor A installs from the live
`https://reinstate.dev/install.ps1` bootstrap and records that as the
artifact identity; this executor (C) installed from the coordinator's
pre-verified, checksummed draft directory
(`…/scratchpad/rc8-draft/reinstate_0.6.0-rc.8_windows_amd64.zip`) instead,
after independently re-verifying its SHA-256 against `checksums.txt` —
matched exactly, and `rein.exe`/`reinstate.exe` are byte-identical.

## 2. Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro, `10.0.26200`, native `windows/amd64`, never WSL |
| CPU architecture/native process | `amd64`, native process (no emulation) |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | host ambient `go1.26.1`; `scripts/testing/conptydriver` (test harness only, never the product binary) built pinned to `GOTOOLCHAIN=go1.25.13`, matching `go.mod`'s `toolchain go1.25.13` and every CI workflow |
| Date | `2026-09-09` |

## 3. What this part specifically re-tests

CLI row 13, this candidate's own targeted fix (`eb0a9b39`, `e7de157e`,
`v0.6.0-rc.8`'s `CHANGELOG.md` entry under `Fixed`), under the expanded
method the dispatch requires: at least 15 independent `ScopeAll` launches
all fully matching ground truth, `ScopeProject` confirmed unaffected, and a
16th launch against a `-stale-claude` bench confirming deterministic
settle on `○ Blocked` rather than `◌` looping forever. Every other row
(1–12, 14–22) is expected `PASS` again unchanged, carried from
`v0.6.0-rc.7`'s own clean part-C evidence — nothing else in this candidate
touches the CLI-experience surface.

## 4. Harness setup

`scripts/tuisandbox` was regenerated fresh from this tagged tree, twice:

```
go run ./scripts/tuisandbox -root D:\ReinstateAcceptanceProjects\v060-rc8-c\tuisandbox\home
go run ./scripts/tuisandbox -root D:\ReinstateAcceptanceProjects\v060-rc8-c\tuisandbox-stale\home -stale-claude
```

Both produced the documented 9-row fixture set (`claude:...0001` through
`codex:...0008`, plus `grok:...000b`), confirmed via
`rein sessions --json` against the tagged binary before any interactive
capture: exactly 9 sessions, 3 agents, matching the bench's own legend.
Ground truth for every row (used throughout §5–§7) was taken from
`rein resume <key> --dry-run --json` / `rein inspect <key> --json` against
this exact tagged binary and this exact bench, immediately before the
interactive captures below:

| Session | Ground truth | Expected glyph |
| ------- | ------------- | --------------- |
| `claude:...0001` (auth-refactor) | `decision: ready` | `●` |
| `codex:...0002` (keyring-store) | `decision: ready` | `●` |
| `claude:...0003` (website) | `decision: confirmation_required` (4 warnings) | `◐` |
| `claude:...0009` (checkout-flow) | `decision: confirmation_required` (1 warning) | `◐` |
| `claude:...0004` (gone-missing) | exit `5` (workspace missing) | `○` |
| `grok:...000b` (payment-adapter) | `decision: confirmation_required` | `◐` |
| `codex:...0006` (repo-drift) | exit `7` (foreign `repository_url`) | `○` |
| `claude:...0007` (deps-bump) | `decision: confirmation_required` | `◐` |
| `codex:...0008` (search-index) | `decision: confirmation_required` | `◐` |

`conptydriver` was invoked through a reusable PowerShell wrapper
(`run-step.ps1`, this executor's own scratch tooling, not committed) that,
before every launch: clears `TERM`/`NO_COLOR`/`LANG`/`LC_ALL`/`LC_CTYPE`/
`WT_SESSION`/`TERM_PROGRAM` (this host's own terminal — Orca — sets all
three of `TERM=xterm-256color`, `NO_COLOR=1`, `LANG=en_US.UTF-8` at process
scope, which a real Windows user would not have; confirmed via
`[Environment]::GetEnvironmentVariable(...,'Process')` showing them
**unset** at `User`/`Machine` scope — an environment a real user would not
carry, cleared per the contract's own evidence policy), unsets
`REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`, points `HOME`,
`REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME` at the bench, isolates
every other agent's `root_env`, then launches `conptydriver.exe` via
`Start-Process -WindowStyle Hidden -Wait` (no stdio redirection). A
"capable terminal" (rows 1, 3–5, 7–20, 22) sets `WT_SESSION` to a
synthetic session id, matching what a real Windows Terminal user carries
(`internal/ui/unicode_windows.go`'s own `windowsUnicodeDefault` trusts
`WT_SESSION`/`TERM_PROGRAM`, absent either it defaults to no — legacy
conhost, row 21's own condition); row 21 leaves it unset.

## 5. CLI experience matrix (22/22 PASS)

| # | Row | macOS | Windows | Evidence |
| - | --- | :---: | :-----: | -------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | n/a (deferred) | **PASS** | `9 sessions · 3 agents · all projects` header, filter prompt, time-grouped rows (`TODAY`/`YESTERDAY`/`THIS WEEK`/`THIS MONTH`/`OLDER`), split preview pane, key bar all draw correctly; all 9 readiness glyphs match ground truth |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical to `v0.5.1` | n/a | **PASS** | Both binaries exit `2`; stderr SHA-256 `0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b` for both, matching every prior candidate's own recorded hash. `stdout` (the cobra command list) legitimately differs — new commands/flags shipped since `v0.5.1` (`account`, `daemon`, `devices`, `hop`, `login`, `sync`, `whoami`, `--plain`) — the row's own assertion is the refusal hint + exit code, carried on stderr, which is byte-identical |
| 3 | `--plain` falls back to the numbered switcher | n/a | **PASS** | All 9 rows numbered `1`–`9`, correct `claude`/`codex`/`grok` agent column, `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` prompt |
| 4 | `REINSTATE_NO_TUI=1` does the same | n/a | **PASS** | Output SHA-256 `976857a3d68e318b7b9854e8ae0aaa942e63a6e69f35cfaaccd2b3c53f8c3eab`, identical to row 3 |
| 5 | `TERM=dumb` does the same | n/a | **PASS** | Output SHA-256 identical to rows 3/4 (same hash above) |
| 6 | `TERM` unset draws the switcher on Windows | n/a | **PASS** | `$env:TERM` confirmed empty (`ENV_CHECK TERM=''`); full interactive switcher drawn, both immediately after launch and after a 5 s settle |
| 7 | A terminal below 40×10 falls back to plain | n/a | **PASS** | 30×8 ConPTY console → wrapped numbered picker (`Choose NUMBER...` prompt, list wrapped across lines) |
| 8 | `NO_COLOR` draws the switcher with no colour escapes | n/a | **PASS** | Raw byte capture: exactly one bare `ESC[m` reset (`grep -aoP '\x1b\[[0-9;]*m'` → 1 match, `[m`), zero SGR-parameter sequences; switcher content and glyphs fully correct |
| 9 | Typing filters; the header count follows | n/a | **PASS** | `"auth"` → `1 session · 1 agent · all projects`, single matching row (`auth-refactor`), all other rows cleared from the list (see harness note, §6) |
| 10 | `f` in list mode filters and does **not** fork | n/a | **PASS** | Prompt shows `❯ f`; header stays `9 sessions` (all 9 fixture titles/branches genuinely contain the letter `f`); key bar shows `esc clear` (filter-mode semantics engaged); no fork screen anywhere |
| 11 | `tab` opens the action menu; `esc` returns without acting | n/a | **PASS** | `tab` → header label `actions`, key bar `r resume   f fork   h hand off   i inspect   y copy ref   esc back`; `esc` reverts to a frame SHA-256-identical to pre-`tab` (`fee2abef0df734943e7f8792a3a6b2688f6145274217ff28bedb5cd5862cd660`, both) |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | n/a | **PASS** | `ctrl+k` opens an overlay (`❯` prompt, horizontal rules); `"hof"` → top match `▸ Hand off to another…  new session from a briefing` |
| **13** | **Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe** | n/a | **PASS** | This candidate's targeted fix — full evidence in §7 |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | n/a | **PASS** | `website` (4 warnings) → checklist opens; 1st space toggles `baseline.unavailable` to `[x]`, equivalent command gains `--allow-environment-warning baseline.unavailable`; 2nd space reverts to `[ ]`, frame SHA-256-identical to pre-toggle (`9bb0ca5a91b0a388a2efc28908154113677d2ca7422e36ec4879f804d2f81db3`, both) |
| 15 | A partial acknowledgement is refused with exit `7` | n/a | **PASS** | 2 of 4 acknowledged (`baseline.unavailable`, `git.branch`) → CLI: `rein resume claude:...0003 --allow-environment-warning baseline.unavailable --allow-environment-warning git.branch` exits `7`, `"environment warnings require confirmation: git.working_tree, runtime.node.declaration"`; interactive checklist shows the matching inline refusal `every warning must be acknowledged before continuing`, remaining two still visibly unchecked |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | n/a | **PASS** | `tab`→`h` opens the studio on `auth-refactor→codex`; `right` cycles `balanced → full → checkpoint → balanced` (wraps), equivalent command's `--policy` follows in lock-step every step; settled projection genuinely differs per policy — `user_messages` is `carried across` under `balanced`, moves to `left behind` (`referenced · projection_budget`) under `checkpoint` |
| 17 | The studio refuses `enter` on a plan that could not be built | n/a | **PASS** | `repo-drift` (`codex`, foreign `repository_url`, exit `7`) → studio shows `○ this handoff cannot be planned / handoff: environment preflight is blocked` before any keystroke (once settled); `enter` adds an identical inline refusal line, studio stays open; second `enter`'s frame SHA-256-identical to the first (`119a4529110534e5e5e177bcd4f6801a323bb868475a5319fa39131c6808469d`, both); `rein handoff list --json` confirms no handoff record was created |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | n/a | **PASS** | Invalid endpoint (`not-a-valid-endpoint`) stays on step 2 with inline error `the endpoint must start with https:// or http://`; raw CSI-Z (`shift+tab`, sent as `\x1b[Z`) returns to step 1 with prior provider selection (`Cloudflare R2`) preserved |
| 19 | `rein init` collects no secret material inside the full-screen program | n/a | **PASS** | All 7 steps walked (provider, endpoint, bucket, region, prefix, device, review) — no credential field anywhere; `rein init --help` has no secret/access/passphrase-key flag; `↵ start setup` on the review step releases the alt-screen program (confirmed: terminal reverts to a plain, non-full-screen `S3/R2 access key:` line prompt) before any key/passphrase prompt occurs |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | n/a | **PASS** | Real two-device round trip via a local `scripts/testing/fakelocker` S3 endpoint (`127.0.0.1:9123`, in-memory, this worktree's own build): device A completes setup (`endpoint=http://127.0.0.1:9123`, `bucket=reinstate-devA`, `profile_id=d2d9487d-e757-4b36-8b21-96e46b95020d`); `rein init --link` prints a pairing code stating "This code carries no keys and no passphrase. The other device still asks for both."; device B's `rein init --paste <code>` pre-fills endpoint, bucket, resolved region (`us-east-1`, matching device A's own stored `config.toml`, not the display placeholder `auto`), key prefix (`profiles/<profile-id>`), and a new step 7 (`Profile ID`) exactly matching device A's real profile id — every field verified against device A's own `config.toml` |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | n/a | **PASS** | `WT_SESSION`/`TERM_PROGRAM`/`LANG`/`LC_ALL`/`LC_CTYPE` all unset (this row's own condition, not the capable-terminal baseline): every UI glyph renders ASCII (`*`/`!`/`x`/`/`/`>`/`...`/`|`/`-`), correctly matching each row's readiness (`*`=ready, `!`=warn, `x`=blocked, per `internal/ui/theme.go`'s own `asciiGlyphs`); session title Unicode text itself (`認証リファクタを...`) renders untouched |
| 22 | Every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences | n/a | **PASS** | 9 documents diffed byte-for-byte against `v0.5.1` on the identical bench: every non-empty diff traces to exactly three documented classes — see §8 |

## 6. Harness note — a capture defect in `conptydriver`'s `vtscreen.go`, found and fixed locally

Row 9's first capture (typing `"auth"` into the filter) rendered a corrupted
header — `9 1 session · 1 agent · all projects` (a leftover `9` glyph from
the pre-filter text) — and the row list below did not visually narrow to
the one matching session, even though the header text correctly reported
`1 session · 1 agent`. Traced to source, not assumed: the raw byte capture
showed Reinstate emitting `CSI Ps X` (`ECH`, Erase Character — e.g.
`\x1b[60X`) to blank stale trailing content before printing shorter
replacement text, exactly the technique
`scripts/testing/conptydriver/README.md`'s own "Trap 1" describes for
space-runs-as-cursor-forward, but for a *different* CSI final byte. Reading
`scripts/testing/conptydriver/vtscreen.go`'s `applyCSI` switch directly:
it implements `H`/`f`/`A`/`B`/`C`/`D`/`G`/`d`/`J`/`K`/`n`/`m`, but had **no
case for `X`** — silently falling into the same "parsed and discarded"
default as an unrelated sequence like scroll-region or insert/delete-line.
A real terminal (and every other row's capture) blanks those cells; this
renderer left the stale characters in its own grid.

This is a **test-harness defect**, not a product defect: `rein`'s own byte
stream is correct VT100/xterm (`ECH` is a standard sequence), and the
product's actual behaviour — confirmed once the harness was fixed — fully
narrows the list to the one matching row. **Fix, applied locally, not
committed** (`scripts/testing/conptydriver/vtscreen.go`): added a
`case 'X':` calling a new `eraseChars(n)` method (models `eraseLine`,
blanks `n` cells at the cursor without moving it, per the `ECH` spec).
Rebuilt (`go build ./scripts/testing/conptydriver`); the package's own unit
suite still passes unmodified (`go test ./scripts/testing/conptydriver/...`
→ `ok`). Row 9 was re-captured with the fixed binary and cleanly narrowed
(see §5's evidence line); row 10 was also re-captured with it for
consistency, though its own evidence (all 9 titles/branches genuinely
contain `f`) was unchanged either way.

**Rows 1, 3–8, 11–13, 21 were captured before this fix and were not
re-captured**, because each was individually checked and does not depend
on an in-place erase shorter than the content it replaces: rows 1, 3–8, 21
are single full-screen paints with no partial narrowing; row 13's 45
readiness-glyph checkpoints are single-character swaps (`◌`→`●`/`◐`/`○`,
same on-screen width) verified visually clean and internally
byte-identical across settle checkpoints in every one of the 15 trials;
row 11's before/tab/esc-revert frames came back byte-identical
(`fee2abef...`), which could not happen if a stale `ECH`-erased region had
leaked through; row 12's palette-open and query frames were visually
inspected and are clean. Rows 14–20, 22 were all captured after the fix.

Per the ground rules ("Commits: only your own results file"), this fix is
**not** part of this commit; `git status` shows it as an unstaged,
uncommitted modification to `scripts/testing/conptydriver/vtscreen.go` in
this shared worktree — already independently observed and correctly
attributed to this part by executor A's own part-A report (§ "Harness
notes", item 2).

## 7. Row 13 — full evidence [`PASS`, this candidate's targeted fix confirmed]

**Method** (the `v0.6.0-rc.7` tagged report's own row-13 method,
`docs/testing/results/2026-09-08-windows-v060rc7.md` §13, reused exactly,
under the dispatch's expanded, tighter bar): launch the tagged binary's own
`rein.exe`, by full path, bare (no subcommand or arguments), from a cwd
outside any tracked project and outside any Git checkout
(`D:\ReinstateAcceptanceProjects\v060-rc8-c\work`) — the switcher's
documented default `ScopeAll`. Checkpoints at ~5 s, ~12 s (cumulative), and
~30 s (cumulative) past the wait match, well past settle. Cross-check every
row's rendered glyph against the ground-truth table in §4. Repeat for 15
independent launches; **every one** must fully match on **every**
checkpoint — one wrong row on one launch is a `FAIL` for the whole row,
per the dispatch's tightened bar (`v0.6.0-rc.7` itself used a looser
15-launch sample that found 11 of 15 wrong).

### 15/15 `ScopeAll` launches — 45/45 checkpoints, 0 mismatches

Each of the 15 launches used a distinct `WT_SESSION` value (a fresh child
process each time — this readiness cache is in-memory per-process,
`internal/tui/readiness/prober.go`'s own `cache map[string]Result` field on
the prober instance, never a persisted file — so every launch is
inherently a cold, independent trial; there is no on-disk readiness cache
to separately clear between runs, unlike the session-index SQLite cache,
which this method does not touch).

| Trial | c5 | c12 | c30 | All 9 rows match ground truth |
| ----- | -- | --- | --- | ------------------------------ |
| 1–15 | match | match | match | **YES**, all 15 |

Verified programmatically, not by eye: for every trial and every
checkpoint, `grep -P "[●◐○]\s+(claude|codex|grok)\s+<project>\b"` extracted
the rendered glyph for each of the 9 fixture rows and compared it against
the §4 ground-truth table — **0 mismatches across 15 trials × 3
checkpoints × 9 rows = 405 individual row checks.** The header line read
exactly `9 sessions · 3 agents · all projects` in every one of the 15
trials' final checkpoint — no host session ever leaked in (the
`XDG_DATA_HOME`-class isolation in §0/§4 held for every trial, not
sampled). Every trial's three checkpoint files (`c5`/`c12`/`c30`) are
SHA-256-identical to each other (`sha256sum` compared per-trial) — every
launch settled to a **correct** answer and then held it, byte-for-byte,
through the full 30-second window; this is the same "wrong-but-stable"
signature `v0.6.0-rc.7`'s report described for its own failures, now with
"correct" in place of "wrong."

Representative worst-case comparison against `v0.6.0-rc.7`'s own
recorded failure (for context, not re-run): that report's worst trial had
8 of 9 rows wrong (`○` standing in for `●`/`◐`), stable across three
checkpoints. Every one of this candidate's 15 trials had 0 of 9 rows
wrong, equally stable.

### `ScopeProject` unaffected

Launched from inside the bench's own `auth-refactor` workspace (a real Git
checkout the bench itself seeds). Header reads
`1 session · 1 agent · auth-refactor`; the single row shows `● READY TO
RESUME` — correct, consistent with every prior candidate's own
`ScopeProject` evidence and not itself in question here.

### `-stale-claude` settle check (this candidate's second fix, `e7de157e`)

Bench regenerated with `-stale-claude` (out-of-range `2.1.300` Claude shim
first on `PATH`; every Claude row necessarily blocks on `agent.version`).
Ground truth, confirmed via `rein resume <key> --dry-run --json` against
this exact tagged binary before the interactive capture: all 5 Claude
fixture sessions (`auth-refactor`, `website`, `checkout-flow`,
`gone-missing`, `deps-bump`) report exit `5`, block reason
`agent.version`, message `"native agent version 2.1.300 is outside the
verified range 2.1.219 to 2.1.265 inclusive"` (one, `gone-missing`, whose
own workspace is also missing, instead reports `"the native agent version
is not determinable; the session layout is still readable"` — still a
deterministic, non-timeout block). `rein inspect claude:...0001 --json`
confirms the same repair message, not a timeout-shaped one.

Interactive: checkpoints at 15 s, 40 s, 70 s, and 100 s after launch —
`ProbeTimeout` (`internal/tui/readiness/prober.go`) is `12 s`,
`maxProbeRetries` is `3`, so 100 s is well past `12 s × 3 = 36 s`,
generously multiplied. All 5 Claude rows show `○` at the 15 s checkpoint
already (no `◌` observed at any checkpoint) and hold it, unchanged, through
100 s: all four checkpoint files (`c15`/`c40`/`c70`/`c100`) are
SHA-256-identical (`87dcfe2eaa583a8af878cc59d32dee62444a6abb672e2985411c4fb75869118b`).
The non-Claude rows (`codex:keyring-store` `●`, `grok:payment-adapter`
`◐`, `codex:repo-drift` `○`, `codex:search-index` `◐`) are unaffected by
the stale shim, exactly as expected, and match their own ground truth
throughout.

**Disposition: `PASS`.** All three parts of the expanded method — 15+
`ScopeAll` launches all fully matching, `ScopeProject` unaffected, and the
`-stale-claude` settle check landing on and staying on `○ Blocked` rather
than looping on `◌` — hold. This directly reverses `v0.6.0-rc.7`'s tagged
`FAIL` on this exact row (11 of 15 launches wrong, 73% failure rate); this
run found 0 of 15 wrong.

## 8. Row 22 — full evidence [`PASS`]

Nine documents diffed byte-for-byte between the tagged `rein.exe` and
`v0.5.1`'s `rein.exe`, both run against the identical bench:
`sessions --json`; `resume <key> --dry-run --json` for one `ready`
session, one `warn` session, both `blocked` sessions (missing-workspace
and foreign-repository), and the `grok` session; `inspect <key> --json`;
`doctor --json`; `handoff <key> --to codex --dry-run --json`; and (against
the `-stale-claude` bench) `resume` on an out-of-range Claude session.
Every non-empty diff traces to exactly three classes, all explained by
`CHANGELOG.md`:

1. **`agent.active` liveness check** (new in this line since `v0.5.1`) —
   present in every `resume`/`inspect` document's `checks` array for
   `claude`/`codex` sessions, absent from `v0.5.1`'s.
2. **Grok tier promotion** — `v0.5.1` reports `"resume": false, "fork":
   false, "read_only_reason": "Grok Build sessions are source-only in
   Phase 4"` for the grok session; this tag reports `"resume": true,
   "fork": true"` and a full environment report, per Grok's later T4
   promotion (`CHANGELOG.md`, `[0.5.2-rc.1]`/`[0.6.0-rc.1]`).
3. **Compatibility range-ceiling text** — on the `-stale-claude` fixture,
   the `agent.version` block message differs only in the numeric ceiling:
   `"...outside the verified range 2.1.219 to 2.1.238 inclusive"`
   (`v0.5.1`) vs. `"...outside the verified range 2.1.219 to 2.1.265
   inclusive"` (this tag) — the Claude Code range has widened across every
   `v0.6.0-rcN`, most recently to `2.1.265` earlier on `2026-09-09`
   (`docs/testing/results/2026-09-09-windows-range-widening-claude-v060.md`),
   documented in `CHANGELOG.md`.

`doctor --json` differs only in the `version` field (`0.6.0-rc.8` vs.
`0.5.1`) — expected, every release. `handoff --dry-run --json` is
byte-identical. No diff outside these three classes and the expected
version field was found in any of the nine documents.

## 9. Findings

No release-blocking findings from this part. One non-blocking harness
defect (missing `ECH` support in `scripts/testing/conptydriver`'s
`vtscreen.go`, §6) — found, root-caused to source, fixed locally, and
verified not to have affected any row's recorded result. No product
defects found in this part; every row this candidate expected to `PASS`
did.

## 10. Cleanup

`D:\ReinstateAcceptanceProjects\v060-rc8-c\` (bench homes, isolated agent
roots, device A/B pairing directories, snapshots, and the locally-built
`conptydriver.exe`/`fakelocker.exe` test tools) is deleted after this
report is committed. Only this results file is committed by this part.
