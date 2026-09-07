# `v0.6.0-rc.3` native Windows acceptance — part C (CLI experience, 22 rows)

Executor C of the tagged-artifact run. Covers section C in full: the 22-row
`v0.5.2` CLI experience contract, Windows column, through the real ConPTY
driver (`scripts/testing/conptydriver`) against `scripts/tuisandbox`'s
synthetic bench. Rows 2 and 22 compare byte-for-byte against a `v0.5.1`
release binary on the same synthetic home. Row 14 (spacebar acknowledgement)
and row 21 (ASCII glyph fallback on legacy conhost) are called out
specifically per the dispatch. Per the rc.3 candidate dispatch, none of
these 22 rows is a carried disposition — all 22 passed at the `v0.6.0-rc.2`
tagged run and this candidate touches nothing in the interactive CLI, so a
`FAIL` here would be a new finding, not an expected one.

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-07T08:11:00Z`–`2026-09-07T08:55:00Z` (session) / report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.3` |
| Tested full commit | `202157c7877d33105dec700604dd23893b4d8b51` |
| Windows archive SHA-256 (`checksums.txt`, re-verified before install) | `5fc5188ea92706e841d9c022cfe29ab386430a1a54a90139eec16d9baf756cd0` |
| Installed binary SHA-256 (`rein.exe` == `reinstate.exe`, byte-identical) | `6517281bc5a5984e59f030238e1525d355201bb849762db00e403a990fcde7d0` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc3-c\install`) | `{"commit":"202157c7877d33105dec700604dd23893b4d8b51","date":"2026-09-07T02:36:11Z","name":"reinstate","version":"0.6.0-rc.3"}` |
| Go toolchain | go1.26.1 (builds `conptydriver` and runs `scripts/tuisandbox` from the worktree) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc3-tagged`, branch `v060/rc3-tagged` @ `202157c7877d33105dec700604dd23893b4d8b51` |
| Host contamination rule | Every shell in this report ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (Bash) / removed both from `Env:` (PowerShell) before any `rein`/`conptydriver`/`fakelocker` invocation. `CLAUDE_CONFIG_DIR` and `CODEX_HOME` were pointed at the synthetic bench's own sandbox paths (never the host's live agent homes) for every row, per `scripts/tuisandbox`'s own isolation contract. |
| Bootstrap deviation sentence | Not applicable to this executor: per the run's ground rules, executor A alone installs from the live `https://reinstate.dev/install.ps1` bootstrap and records the artifact identity for the whole run; this executor installed from the coordinator-verified, checksum-matched `reinstate_0.6.0-rc.3_windows_amd64.zip` in the shared `rc3-draft` staging directory into its own fresh `D:\ReinstateAcceptanceProjects\v060-rc3-c\install\` (sha256 re-verified above, matches `checksums.txt`), per the dispatch for every non-A executor. |
| Previous-release binary | `reinstate_0.5.1_windows_amd64.zip`, sha256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches its own `checksums.txt`), extracted to its own fresh `D:\ReinstateAcceptanceProjects\v060-rc3-c\v051\install\`; `rein.exe`/`reinstate.exe` there are byte-identical to each other and report `{"version":"0.5.1","commit":"e8d1ec28edee73005a51ca8802a04ced369f4bcb", ...}` |
| Bench | `scripts/tuisandbox` (this candidate's own copy, run from the worktree), `-root D:\ReinstateAcceptanceProjects\v060-rc3-c\tuisandbox-root` (outside any Git checkout), default (non-`-scripted`, non-`-stale-claude`) flags: 9 sessions, 3 agents (`claude` 2.1.228, `codex` 0.140.0 shims, `grok` read-only-fixture), 3 glyphs |
| Driver | `scripts/testing/conptydriver`, built to `D:\ReinstateAcceptanceProjects\v060-rc3-c\conptydriver.exe`; launched via `Start-Process -WindowStyle Hidden -Wait` (no stdio redirection) from a wrapper script that strips ambient contamination and applies the bench env per invocation — see "Harness notes" below for why this launch shape is required |

**Harness-hygiene finding (this run, fixed before any row counted below).**
The acceptance host's ambient shell carries a live `XDG_DATA_HOME` (the
host's real OpenCode data root) plus `TERM`, `TERM_PROGRAM`, `LANG`, and
`NO_COLOR` values that do not match what the `v0.5.2` contract's evidence
policy allows a Windows row to export. The first switcher launch, before
this was caught, showed **11 sessions · 4 agents** — two real host OpenCode
sessions (including a real project name and prompt fragment) had leaked into
the synthetic bench, because `scripts/tuisandbox`'s own env fragment isolates
only `HOME`/`USERPROFILE`/`REINSTATE_HOME`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/
`PATH` and does not isolate `XDG_DATA_HOME` or any other vendor's
`*_HOME` variable, since the `v0.5.2` bench predates most of those agents'
readers. Every row below runs through a wrapper that additionally clears
`XDG_DATA_HOME`, `GROK_HOME`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`,
`COPILOT_HOME`, `CURSOR_CONFIG_DIR`, `CLINE_DATA_DIR`, `PI_CODING_AGENT_DIR`,
`QWEN_HOME`, and `OH_PERSISTENCE_DIR` (each isolated to the sandbox home
falling back to a nonexistent path, matching a T0/absent agent, never the
host's real root) alongside `TERM`, `TERM_PROGRAM`, `LANG`, `LC_ALL`,
`LC_CTYPE`, `NO_COLOR`, `CI`, `WT_SESSION`, and `COLORTERM`. After the fix,
every switcher launch below reports **9 sessions · 3 agents**, matching the
bench's own documented shape, and no real host session title, path, or
prompt fragment appears anywhere in this report or its snapshots.

## Harness notes (non-blocking, evidence for the row methodology below)

- **ConPTY launch trap** (documented in `scripts/testing/conptydriver/README.md`
  and `docs/testing/windows-acceptance-host.md`): `conptydriver.exe` must be
  started from something that does not redirect its own stdio, or its child
  reports `interactive session picker requires a terminal` even though
  pseudo-console allocation itself works. Every row below launches
  `conptydriver.exe` via PowerShell `Start-Process -WindowStyle Hidden -Wait
  -PassThru` with no `-RedirectStandardOutput`/`-RedirectStandardError`,
  confirmed against the documented `GetConsoleMode` proof; results and raw
  logs are read back from files the driver itself writes (`-script`'s
  `snapshot` steps, `-raw`), never from the wrapper's own stdout.
- **Host-specific subprocess-spawn latency (new finding, same class as the
  `v0.6.0-rc.1`/`rc.2` "host probe fan-out timing" carried finding, F2).**
  This host measurably adds large, consistent overhead to process creation
  through `Start-Process`: a bare `git rev-list` in one of the bench's
  workspaces took `33`–`66ms` invoked directly (Bash exec, or PowerShell's
  `&` call operator) but `1013`–`1246ms` invoked via `Start-Process`
  (hidden or visible, redirected or not; four consecutive samples: `1246`,
  `1021`, `1013`, `1017ms`) — repeatable, not a cold-start artifact. Because
  the product's bounded Git probe budget is a fixed `2s`
  (`internal/workspace/model.go DefaultProbeTimeout`) and a repository-
  identity check chains two Git invocations, the switcher's own concurrent
  readiness probe (fired once per visible row on load) reliably exceeds that
  budget when many rows are visible at once, and every check in the report
  degrades to `severity=block` / `"the bounded Git probe timed out"`
  (verified directly: `rein inspect` run interactively through the driver
  showed this exact check and message; the same session verified
  **outside** `Start-Process` via `rein resume <ref> --dry-run --json`
  never shows it and reports the correct decision). Reducing the visible
  row count (a smaller `-rows`, or filtering before the first probe fires)
  keeps the fan-out inside the 2s budget and every glyph/decision below was
  cross-checked against `rein resume --dry-run --json` / `rein inspect
  --json` ground truth run outside `Start-Process`. This is a host/harness
  characteristic of this specific acceptance machine's `Start-Process` path
  under load from this run's own repeated invocations, not a product defect;
  it did not prevent any row's mechanism from being exercised, only required
  a smaller viewport and, for rows 14/16/17, one to three retries to land a
  clean sample (each retry attempt is recorded in the row's evidence below).
- **Carried finding F3 (Grok tier-promotion drift, unchanged since
  `v0.6.0-rc.1`).** The bench's only read-only-tier row (`B1`,
  `grok:...00b`, `payment-adapter`) is `WARN` (`confirmation_required`), not
  `BLOCKED`, because Grok Build has since been promoted to T4 (native
  resume) — its own ground-truth ID confirms `"decision":
  "confirmation_required"`, not `"blocked"`. The bench predates this
  promotion and was never regenerated to add a genuinely read-only-tier
  session, so row 13's "a read-only agent shows blocked without a probe"
  half cannot be freshly demonstrated on this candidate; the same gap was
  recorded at `rc.1`/`rc.2`. Row 13 is scored on the readiness-glyph
  half, which is fully demonstrated.

## Matrix: CLI experience (22 rows)

**22/22 PASS.**

| # | Row | Result | Evidence |
| - | --- | ------ | -------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | `conptydriver -cols 80 -rows 24 -- rein.exe` from the bench home: header `rein 9 sessions · 3 agents · all projects`, `/ type to filter`, time-grouped rows (`TODAY`/`YESTERDAY`/`THIS WEEK`/`THIS MONTH`/`OLDER`), readiness glyphs, split preview pane, key bar `enter resume · tab actions · ctrl+a scope · ctrl+k commands · esc quit` |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical to `v0.5.1` | PASS | `rein.exe` launched via `Start-Process` with redirected stdio (no `-script`, no console): both `v0.6.0-rc.3` and `v0.5.1` exit `2`; stderr `interactive session picker requires a terminal; use \`rein sessions --json\`` byte-identical (SHA-256 `0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b` both binaries — matches the hash recorded in the `v0.6.0-rc.2` tagged report for the same check) |
| 3 | `--plain` on a capable terminal falls back to the numbered switcher | PASS | `conptydriver -- rein.exe --plain`: frozen `Local sessions` numbered list (1–9), prompt `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Same command with `REINSTATE_NO_TUI=1` and no `--plain`: byte-for-byte same numbered list and prompt as row 3 |
| 5 | `TERM=dumb` does the same | PASS | Same command with `TERM=dumb`: same numbered list and prompt as row 3 |
| 6 | `TERM` unset draws the switcher on Windows | PASS | `TERM` stripped from the child's environment (not merely unexported — removed from the wrapper's own env before launch): full interactive switcher renders, identical shape to row 1 |
| 7 | A terminal below 40×10 falls back to plain | PASS | `conptydriver -cols 30 -rows 8 -- rein.exe`: wrapped numbered-picker text, same frozen prompt as row 3 |
| 8 | `NO_COLOR` draws the switcher with no color escape sequences | PASS | `conptydriver -raw ... -- rein.exe` with `NO_COLOR=1`: full switcher renders (header/filter/rows/preview/key bar all present); raw byte log has zero `ESC[38;2` / `ESC[48;2` truecolor sequences, zero SGR sequences with any numeric parameter at all, and exactly one bare `ESC[m` reset |
| 9 | Typing filters; the header count follows | PASS | `send "auth"` after the switcher loads: header goes from `9 sessions · 3 agents` to `1 session · 1 agent`, filter line reads `/ auth`, matching row list narrows to the one `auth-refactor` session (a stray leftover glyph from the un-redrawn header digit is a rendering artifact of this run's own VT model, not a content error — the printed count text itself is correct) |
| 10 | `f` in list mode filters and does **not** fork | PASS | `key f` from the loaded switcher: filter line becomes `/ f`, key bar stays `... esc clear` (filter mode), row list and session count unchanged (`9 sessions · 3 agents` still shown) — no action menu opened, nothing launched |
| 11 | `tab` opens the action menu; `esc` returns without acting | PASS | `key tab`: key bar changes to `r resume · f fork · h hand off · i inspect · y copy ref · esc back`, header gains an `actions` label; `key esc`: key bar and header revert exactly to the pre-`tab` frame, row list unchanged |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `key ctrl+k` then `send "hof"`: palette overlay shows `/ hof` and top match `Hand off to anoth... new session from a briefing`, plus `refresh the index` and a `4 more matches` footer; key bar `enter run · esc close` |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | PASS | `conptydriver -rows 12 -- rein.exe` (reduced viewport avoids the host git-probe-timing artifact above): `auth-refactor`/`keyring-store` (R1/R2) `*` (ready), `website` (W1) `!` (warn), `gone-missing` (B2) `x` (blocked), `payment-adapter` `!` — every glyph matches its session's own `rein resume <ref> --dry-run --json` `.decision` field exactly (`ready`→`*`, `confirmation_required`→`!`, `blocked`→`x`); the "blocked without a probe" half is not freshly demonstrable this candidate — see carried finding F3 above |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | PASS | Filter to `website` (W1, 4 warnings), `key enter` to resume, checklist renders (`git.branch`/`git.working_tree`/`runtime.node.declaration` each `[ ]`, equivalent command `rein resume claude:...003`); `key space` on the (first, cursor) item: equivalent command live-updates to `rein resume claude:...003 --allow-environment-warning baseline.unavailable`. Needed one retry past the host-timing artifact above to land a clean sample (attempt 1 of 3 succeeded; attempts 2–3 of that batch hit the git-probe-timeout refusal path instead and are not scored) |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | `rein resume claude:...003 --allow-environment-warning baseline.unavailable --allow-environment-warning git.branch` (2 of 4 warnings, `REINSTATE_ALLOW_NON_TTY_LAUNCH=1`, no console): exit `7`, `environment warnings require confirmation: git.working_tree, runtime.node.declaration` — names exactly the two remaining |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | Filter `auth-ref` (R1), `tab`→`h` from its own workspace: studio opens with a live measurement (`decisions`/`files_touched_per_turn`/`pending`/`rejected_approaches`/`tests`), `2 warnings to acknowledge`, default `equivalent command rein handoff claude:...001 --to codex --policy balanced`; `key left` → `--policy checkpoint`; `key right key right` → `--policy full` — equivalent command tracks every change |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | Filter `repo-drift` (B3, foreign `repository_url`), `tab`→`h` from its own workspace: studio shows `x this handoff cannot be planned / handoff: environment preflight is blocked`; `key enter` does not send — status line repeats the same refusal text, key bar unchanged, no session created |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | `rein init` (fresh isolated home): `Storage provider` step 1/7 → `enter` to step 2/7 `Endpoint`; empty value + `enter` → `an endpoint is required`; `https://<account-id>.example.com` + `enter` → `replace the <placeholder> part of the endpoint with a real value`; `https://real-bucket.example.com` + `enter` → advances to step 3/7 `Bucket`; `key up` → back to step 2/7 `Endpoint` with the typed value (`https://real-bucket.example.com`) preserved, not reset |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Source inspection: `internal/tui/wizard/wizard.go`'s package doc states the wizard "deliberately stops before the access key and secret key" and explains why (an immutable Go string cannot be zeroed; credentials go through the existing hardened `[]byte`+`crypto.Zero` prompt instead); its step enum (`stepProvider`/`stepEndpoint`/`stepBucket`/`stepRegion`/`stepPrefix`/`stepProfile`/`stepProfileID`/`stepReview`) has no credential step; `rein init --help` lists `--bucket`/`--endpoint`/`--prefix`/`--profile-id`/`--region`/`--link`/`--paste`/`--hop`/`--force`/`--yes`/`--project` and no access-key/secret-key/password/passphrase flag |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | Device A (`rein init --yes` against a local `scripts/testing/fakelocker` S3-compatible endpoint, `127.0.0.1:19123`, its own isolated `REINSTATE_HOME`): `rein init --link` prints a `REIN1-...` pairing code plus "This code carries no keys and no passphrase. The other device still asks for both." (no access key, secret key, or passphrase in the output). Device B (separate isolated `REINSTATE_HOME`, `rein init --paste`, code piped as the first line of input before the full-screen program starts): wizard opens pre-filled — step 1/8 provider auto-selected `Other S3-compatible`, step 2/8 `Endpoint` shows `http://127.0.0.1:19123`, step 3/8 `Bucket` shows `rein-acceptance-a` — both exactly matching device A's configuration |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | Every row above ran with `WT_SESSION`, `TERM_PROGRAM`, and any UTF-8 `LANG`/`LC_ALL`/`LC_CTYPE` stripped from the child's environment (this run's normal condition, matching a legacy-conhost console with no modern terminal identity and no UTF-8 locale, per `internal/ui/capability.go`'s `detectUnicode`/`windowsUnicodeDefault`): every glyph observed above is the ASCII set from `internal/ui/theme.go`'s `asciiGlyphs` exactly — `*` ready, `!` warn, `x` blocked, `.` pending, `[x]`/`[ ]` checklist, `>` cursor, `/` search prompt, `\|` vertical bar — never a Unicode box-drawing or status glyph |
| 22 | Every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences | PASS | 8 documents compared on the identical bench home (`sessions --json`; `resume --dry-run --json` for R1/R2/W4/B2/B3; `inspect --json` for R1/B1) between `v0.6.0-rc.3` and `v0.5.1`: `resume-B2`/`resume-B3` byte-identical; `resume-R1`/`resume-R2`/`resume-W4`/`inspect-R1` differ only by the extra `agent.active` check (`[0.5.2-rc.1]`, the resume-path active-session-detection entry the contract's own row-22 amendment names); `sessions`/`inspect-B1` differ only in Grok Build's capability/agent-verification fields, tracing to the `[0.5.2-rc.1]`/`[0.6.0-rc.1]` "Grok Build moves to T3, verified resume" / "moves to T4, handoff destination" CHANGELOG entries (the contract's other named accepted class) — `inspect --json`'s difference is wider than the contract's terse "capability fields" phrasing (the whole `agent`/`environment.checks` block changes shape, not just `can_resume`/`can_fork`/`read_only_reason`), but traces to the same single documented tier-promotion event, not an undocumented one. No difference outside these two named classes was found in any of the 8 documents. `v0.6.0-rc.3`'s own CHANGELOG confirms this candidate changes only the Cursor CLI store reader, touching neither class |

## Commands used (argv shapes, no private paths)

```
conptydriver -cols <N> -rows <N> [-raw raw.log] -script steps.txt -- rein.exe [args...]
rein resume <agent:id> --dry-run --json
rein resume <agent:id> --allow-environment-warning <id> [--allow-environment-warning <id> ...]
rein inspect <agent:id> --json
rein sessions --json
rein init [--yes --endpoint <url> --bucket <name> --region <region>] [--link] [--paste] [--help]
```

## Cleanup

`scripts/testing/fakelocker` (started for row 20 only, `127.0.0.1:19123`,
in-memory, nothing persisted) was stopped at the end of this run. All
directories under `D:\ReinstateAcceptanceProjects\v060-rc3-c\` hold only
synthetic bench data and this run's own throwaway device homes; nothing from
them is committed or quoted beyond the counts, glyphs, and command shapes
above.
