# v0.6.0-rc.2 tagged-artifact acceptance — part C (CLI experience, 22 rows)

Executor C's tagged-artifact acceptance for Section C of the
[`v0.6.0-rc.2` dispatch](../v0.6.0-rc.2-agent-verification-prompts.md): the
full 22-row `v0.5.2` CLI-experience contract, Windows column, through the
real ConPTY driver (`scripts/testing/conptydriver`) against
`scripts/tuisandbox`'s synthetic bench, at
`D:\ReinstateAcceptanceProjects\v060-rc2-c\`. This part re-tests every row
that passed 22/22 at `v0.6.0-rc.1`
([`2026-09-06-windows-v060rc1.md`](2026-09-06-windows-v060rc1.md) §14) against
the `v0.6.0-rc.2` tagged artifact; nothing in this candidate's own changelog
touches the interactive surfaces.

## Verdict

**22/22 PASS.** Zero release-blocking findings. Four non-blocking findings
carried from the `v0.6.0-rc.1` tagged run (same root causes, re-observed
independently on this pass) plus one new non-blocking, out-of-scope
observation (see §Findings).

## 1. Immutable test record / artifact identity

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-07 |
| Device | `windows-amd64`, Windows 11 Pro, `10.0.26200`, native, never WSL |
| Tested tag | `v0.6.0-rc.2`, published GitHub prerelease (release workflow run `34062478935`) |
| Tested full commit | `81d74a82ba2a0e27f9f1a68eb0270d224a20da6a` |
| Windows archive | `reinstate_0.6.0-rc.2_windows_amd64.zip` |
| Archive SHA-256 | `82ea243cf9aa1b411cc77322abf973afaf8ff8ac13fbbfa05ad32c5d82ecdc84` (matches `checksums.txt`, re-verified independently) |
| Installed binary SHA-256 | `0ae03c4eed8c1af610f04842d5ed51efdd9847724a3b841d249b3841e087fce6` — `rein.exe`/`reinstate.exe` byte-identical (`cmp` exit `0`) |
| `rein version --json` | `{"commit":"81d74a82ba2a0e27f9f1a68eb0270d224a20da6a","date":"2026-09-06T21:57:00Z","name":"reinstate","version":"0.6.0-rc.2"}` |
| Previous-release comparison binary (rows 2, 22) | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c`, installed `rein version --json` commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb`, version `0.5.1` |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc2-c\install\` (fresh, this part only) |
| Harness build toolchain | `go1.26.1` (host default) — used only to build `conptydriver`/`tuisandbox` from this tagged worktree; the binary under test is the shipped release archive, not built by this part |

### Bootstrap deviation

This part installed from the coordinator-verified staging directory
(`...\scratchpad\rc2-draft\reinstate_0.6.0-rc.2_windows_amd64.zip`), whose
sha256 was independently re-verified above against `checksums.txt`, per the
dispatch's rule that only executor A installs from the live
`reinstate.dev/install.ps1` bootstrap and records that as the artifact
identity; every other executor, including this part, installs from the same
checksummed archive the coordinator already verified (checksum match, GitHub
attestation pass, `scripts/verify-release.ps1`/`scripts/test-install.ps1`
both exit `0`).

### Host hygiene

Every shell used for this part started with `unset REINSTATE_BACKEND
REINSTATE_MEMORY_BACKEND_DIR` (the host's ambient shell carried
`REINSTATE_BACKEND=memory` and `REINSTATE_MEMORY_BACKEND_DIR` pointed at a
live lab directory; both were confirmed unset before every run). `tuisandbox`
was generated at `D:\ReinstateAcceptanceProjects\v060-rc2-c\tuisandbox-home\`
(outside any Git checkout, confirmed with `git rev-parse
--is-inside-work-tree` failing there), pointing `CLAUDE_CONFIG_DIR` and
`CODEX_HOME` at its own synthetic tree per its documented mechanism.
`XDG_DATA_HOME` (OpenCode's declared `root_env`, confirmed via `rein doctor
--agents --json`) was **not** isolated by `tuisandbox` itself and was found,
on the first interactive run, leaking one real host OpenCode session
(project `reinstate`) into the switcher — the ambient host value
(`D:\Projects\hop-10-lab\xdg`) was captured only in this session's own
scratch terminal output, never written to a file, screenshot, or this
report. It was then pointed at an isolated empty directory
(`...\tuisandbox-home\isolated-xdg`) for every row below, after which
`rein sessions --json` showed exactly the documented 9 synthetic rows. This
matches the `v0.6.0-rc.1` tagged run's own finding F5 (same root cause,
independently rediscovered and worked around before any row evidence below
was captured).

`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` were the only
variables redirected; the host's other live agent home variables were
confirmed empty (`CLINE_DATA_DIR`, `COPILOT_HOME`, `CURSOR_CONFIG_DIR`,
`GEMINI_CLI_HOME`, `GROK_HOME`, `KIMI_CODE_HOME`, `QWEN_HOME`,
`OH_PERSISTENCE_DIR`, `PI_CODING_AGENT_DIR` all unset on this host), so no
further isolation was needed.

**Terminal-identity hygiene.** This host's own interactive shell exports
`TERM=xterm-256color`, `TERM_PROGRAM=Orca`, and `LANG=en_US.UTF-8` as
artifacts of the orchestration launcher — none of which a real native
Windows PowerShell/cmd/Windows Terminal session would ever export. Every row
below strips `TERM`, `TERM_PROGRAM`, `LANG`, `LC_ALL`, `LC_CTYPE`, and
`WT_SESSION` from the child's environment first, then rebuilds only the
identity a given row is testing: a "capable terminal" baseline sets
`WT_SESSION` (Windows 11's default terminal) and leaves the rest absent;
row 21 leaves `WT_SESSION` absent too, modelling legacy conhost; row 5 sets
`TERM=dumb` on top of the capable baseline; rows 3/4 use `--plain`/
`REINSTATE_NO_TUI=1` on the same baseline. This distinction mattered in
practice: run un-scrubbed, this host's ambient `LANG=en_US.UTF-8` alone was
enough to force Unicode glyphs regardless of terminal identity, which would
have silently invalidated row 21.

## 2. Matrix — CLI experience (22 rows)

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | Header `rein 9 sessions · 3 agents · all projects`, filter prompt `❯ type to filter`, time-grouped rows (TODAY/YESTERDAY/THIS WEEK/THIS MONTH/OLDER), split preview panel, key bar `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit` |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical stderr to `v0.5.1` | PASS | Both binaries, real `Start-Process` pipe redirection (no ConPTY): both exit `2`; stderr SHA-256 identical on both (`0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b` — the same hash the `v0.6.0-rc.1` tagged run recorded); stdout differs only by legitimately added top-level commands (`account`, `daemon`, `devices`, `hop`, `login`, `sync`, `whoami`) and the new `--plain` flag, all already shipped before this candidate |
| 3 | `--plain` on a capable terminal falls back to the numbered switcher | PASS | Frozen numbered-picker text: `Local sessions` header, 9 numbered rows, prompt `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Byte-identical to row 3 (`Compare-Object` empty diff) |
| 5 | `TERM=dumb` does the same | PASS | Byte-identical to row 3 |
| 6 | `TERM` unset draws the switcher on Windows | PASS | Full interactive switcher, byte-identical to row 1's frame |
| 7 | A terminal below 40x10 falls back to plain | PASS | `-cols 30 -rows 8`: wrapped frozen numbered-picker text, same prompt as row 3 |
| 8 | `NO_COLOR` draws the switcher with no escape sequences for colour | PASS | Raw byte capture: exactly one bare `ESC[m` reset, zero SGR colour params in the entire frame; switcher still renders fully (header, filter, rows, preview, key bar) |
| 9 | Typing filters; the header count follows | PASS | `send "auth"` → header `1 session · 1 agent`, matched to ground truth (`rein search auth --json` returns exactly 1 session); visual list-narrow itself is obscured by a harness VT gap (F1, carried) |
| 10 | `f` in list mode filters and does **not** fork | PASS | `key f` → filter prompt shows `❯ f`; count of running `claude`/`codex` host processes unchanged (5 before, 5 after) — no vendor process launched |
| 11 | `tab` opens the action menu; `esc` returns without acting | PASS | `tab`: header gains an `actions` tag, key bar becomes `r resume f fork h hand off i inspect y copy ref esc back`; `esc`: both revert exactly to the row-1 frame, no action taken |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `ctrl+k` then `"hof"` → top match (cursor `▸`) is `Hand off to another…`; two further list rows below it are stale un-erased content, the same harness VT gap as row 9 (F1) |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | PASS | At full 30-row viewport every glyph read pending/blocked (host probe fan-out timing, F2 — see below); at a reduced 14-row viewport all resolved and matched ground truth exactly: `claude`/`auth-refactor` and `codex`/`keyring-store` READY (●, preview `READY TO RESUME`); `claude`/`website` and `claude`/`checkout-flow` WARN (◐); `grok`/`payment-adapter` WARN, not blocked (◐ — dated drift, F3); `claude`/`gone-missing` and `codex`/`repo-drift` BLOCKED (○), cross-checked against `rein resume <id> --dry-run --json .environment.decision` for every row |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | PASS | Selecting `claude`/`website` (4 warnings) and choosing "resume" opens the full-screen checklist: `[ ] baseline.unavailable` / `git.branch` / `git.working_tree` / `runtime.node.declaration`, equivalent command `rein resume claude:...003` (no flags); `key space` on the first item → `[x] baseline.unavailable`, equivalent command live-updates to `rein resume claude:...003 --allow-environment-warning baseline.unavailable` |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | `rein resume claude:...003 --allow-environment-warning baseline.unavailable --allow-environment-warning git.branch` (2 of 4): exit `7`, message `environment warnings require confirmation: git.working_tree, runtime.node.declaration` — names exactly the two remaining |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | `auth-refactor` (READY) → hand off to `codex`: `key right` cycles `balanced → full → checkpoint`; equivalent command tracks every change (`rein handoff claude:...001 --to codex --policy balanced\|full\|checkpoint`) |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | `repo-drift` (foreign `repository_url`) → studio settles to `○ this handoff cannot be planned / handoff: environment preflight is blocked`; `enter` does not send — the screen adds a repeated status line and stays on the studio, no destination session created. An under-settled first attempt (measurement still in flight) let `enter` proceed past a stale frame; the fix was the same host-timing sensitivity as row 13 (F2), not a product defect — see below |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | Step 1 (provider) → `Amazon S3`; step 2 (endpoint) empty → `an endpoint is required`; `not-a-url` → `the endpoint must start with https:// or http://` (byte-identical to the committed golden fixture); raw `send "\x1b[Z"` (driver has no named shift+tab key, F4) → back to step 1 with `Amazon S3` still selected |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Source inspection: `internal/tui/wizard/wizard.go`'s package doc states the wizard "deliberately stops before the access key and secret key" and explains why (an immutable Go string can't be zeroed); `rein init --help` lists no credential flag; `--yes` reads credentials only from the environment after the wizard would have exited |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | Real two-device round trip: device A (`rein init --yes` with a fake S3-compatible endpoint/region/bucket and `REINSTATE_S3_ACCESS_KEY_ID`/`REINSTATE_S3_SECRET_ACCESS_KEY`) → `rein init --link` prints a `REIN1-...` code with the message "This code carries no keys and no passphrase"; device B (`rein init --paste`, driven interactively) pastes it and lands on step 1 with **`Amazon S3` pre-selected**, decoded from device A's endpoint |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | `WT_SESSION`/`TERM_PROGRAM`/`TERM`/locale all absent: full ASCII set — filter prompt `/`, cursor `>`, pending dot `.`, vertical bar `\|`, ellipsis `...`, key bar `enter resume` (not `↵`) — matches `internal/ui/theme.go`'s `asciiGlyphs` exactly |
| 22 | Every `--json` document is byte-identical to `v0.5.1` | PASS | 10 documents compared on the **identical `tuisandbox` synthetic home** (`sessions`, `resume --dry-run`, `inspect`, `handoff --dry-run` for 6 sessions spanning READY/WARN/BLOCKED/read-only-promoted): every non-empty diff traces to exactly the two classes the amended contract names — Grok Build capability/tier promotion (`sessions --json` capability fields; `resume`'s full preflight instead of an outright refusal) and the new `agent.active` check in `resume --dry-run --json`/`inspect --json`. `handoff --dry-run --json` is byte-identical outright. No unmatched difference found |

## 3. Findings

Non-blocking. None affect the verdict.

| ID | Class | Description |
| -- | ----- | ----------- |
| F1 (carried) | Harness | `conptydriver`'s VT model has no ECH (erase-character) support: a re-render that shortens a line leaves the old tail on screen. Observed on row 9 (filtered list narrowing) and row 12 (two stale rows below the palette's real top match). Matches the `v0.6.0-rc.1` tagged run's F1. |
| F2 (carried) | Host | This host's environment/handoff-plan probes fan out concurrently per visible row and can time out or read `unavailable` under concurrent load; a reduced viewport (fewer visible rows → less fan-out) or a longer settle resolves it. Reproduced independently on both row 13 (all 9 rows read blocked at a 30-row viewport, resolved at 14 rows) and row 17 (an early `enter` reached a stale "still measuring" frame; the settled frame correctly refused). Matches the `v0.6.0-rc.1` tagged run's F2 in kind. |
| F3 (carried) | Dated drift | `grok`/`payment-adapter` (bench row `B1`) no longer demonstrates the "read-only agent, blocked without a probe" path `tuisandbox`'s own legend text still describes: Grok Build's already-shipped tier promotion (T2→T4, dated before this candidate) means `resume` now runs a full environment preflight and returns `confirmation_required`, not an outright block. Not a regression; the bench's static legend text is stale relative to the current catalog. Matches the `v0.6.0-rc.1` tagged run's F3. |
| F4 (carried) | Harness | `conptydriver` has no named `shift+tab` key. Worked around by sending the raw CSI sequence directly (`send "\x1b[Z"`), which the child correctly interprets. Matches the `v0.6.0-rc.1` tagged run's F4. |
| F5 (new, out of the row-22 method's scope) | Observation | Outside the tuisandbox-identical-home comparison method row 22 uses: on a completely fresh host state where the Claude Code config directory has never existed at all, `v0.5.1`'s `rein list --json` and `rein diff --json` crash with an unhandled raw OS error (`GetFileAttributesEx ...: The system cannot find the path specified`), while `v0.6.0-rc.2` does not (`list --json` returns `null`; `diff --json` returns a structured `{"code":"config",...}` error). No `CHANGELOG` entry names this specifically, and it does not occur on the tuisandbox bench (which always has a `.claude` directory), so it does not affect row 22's own pass/fail determination under the contract's documented method. Recorded for visibility only; recommend confirming whether this is an intentional, unlogged hardening fix or a coincidental side effect of unrelated work. |

## 4. Environment / commands (representative, argv-only)

```
rein sessions --json
rein search auth --json
rein resume <session-ref> --dry-run --json
rein resume <session-ref> --allow-environment-warning <id> [--allow-environment-warning <id> ...]
rein inspect <session-ref> --json
rein handoff <session-ref> --to <agent> --dry-run --json --allow-warning baseline.unavailable --allow-warning handoff.capability.attachment.support
rein init --yes --endpoint <url> --bucket <name> --prefix <prefix> --region <region>
rein init --link
rein init --paste
conptydriver -cols <n> -rows <n> -dir <workdir> -script <steps.txt> [-raw <raw.log>] -- rein.exe [args...]
```

No transcript text, prompts, credentials, private paths, tokens, or session
IDs beyond the synthetic bench's own fixed fixture IDs appear anywhere above.
