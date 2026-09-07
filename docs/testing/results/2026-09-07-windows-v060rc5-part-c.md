# `v0.6.0-rc.5` tagged-artifact native Windows acceptance — part C (CLI experience)

`PHASE5-DEVICE-REPORT-V1` (part file)

This is executor C's part of the assembled device report for the published,
signed GitHub prerelease **`v0.6.0-rc.5`**. It covers the 22-row `v0.5.2` CLI
experience contract (section C of the rc.5 dispatch), run through the real
Windows pseudo console (`scripts/testing/conptydriver`) against
`scripts/tuisandbox`'s synthetic bench, from worktree
`D:\Projects\reinstate-worktrees\v060-rc5-tagged` (branch `v060/rc5-tagged`).

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.5` |
| Full commit | `0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9` |
| Windows archive | `reinstate_0.6.0-rc.5_windows_amd64.zip` |
| Archive SHA-256 | `a648f1de65c15bea7926dda2dd0ce48110d512c9bc60a23b5988c3b24ded9f09` (matches `checksums.txt` in the coordinator's verified `rc5-draft` directory) |
| `rein.exe`/`reinstate.exe` SHA-256 | `aa74899f68356ea5127fa8129db56729c137276d255f1d8890fa9dcf276dd35f` (both files; `cmp` confirms byte-identical) |
| `rein version --json` | `{"commit":"0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9","date":"2026-09-07T12:59:29Z","name":"reinstate","version":"0.6.0-rc.5"}` |
| Install location | `D:\ReinstateAcceptanceProjects\v060-rc5-c\install\extracted\` (fresh directory, re-verified sha256 against `checksums.txt` before use) |
| Bootstrap deviation | Not applicable to this part. Per the dispatch, only executor A installs from the live bootstrap (`https://reinstate.dev/install.ps1`); this part installed from the coordinator-verified checksummed archive at `...\scratchpad\rc5-draft\`, the same archive every non-A executor uses. Executor A's part (`docs/testing/results/2026-09-07-windows-v060rc5-part-a.md`) records the live bootstrap pinning `v0.6.0-rc.5` (`website-v2026.09.07.4`) and installing a byte-identical binary. |
| Previous-release comparison binary | `reinstate_0.5.1_windows_amd64.zip`, sha256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches `checksums.txt`), extracted to `D:\ReinstateAcceptanceProjects\v060-rc5-c\install-v051\extracted\`; `rein version --json` reports `0.5.1` / `e8d1ec28edee73005a51ca8802a04ced369f4bcb` |

**Cross-check note (does not resolve, flags for reconciliation).** Executor
A's part file records a release-blocking finding, `F-RC5-STALE-RELEASE-BINARY`,
claiming the archive binary (identical SHA-256 to the one used here) is not
built from the tagged source — `doctor --agents`/`--acceptance-matrix` do not
exist, `doctor --json`'s internal version reads `0.3.0-rc.6`, and only ~4 of
~18 catalog keys are recognized. This part's own independent, repeated
testing of the **identical SHA-256** binary directly contradicts that: `rein
doctor --json` here reports `"version": "0.6.0-rc.5"` (not `0.3.0-rc.6`);
`rein doctor --agents --acceptance-matrix --json` exists and returns
`{"schema":"PHASE5-ACCEPTANCE-MATRIX-V1","row_count":178,...}` with all
catalog agents including T0 keys; and every one of this part's 22 rows
exercised genuine `v0.6.0`-and-later product behavior that could not exist in
a `0.3.0-rc.6` build — the bounded-pool readiness prober behind CLI row 13's
fix (present and working, see Row 13 below), the handoff studio (row 16/17),
the 7-/8-step `init` wizard with `--link`/`--paste` device pairing (row 20),
and the `agent.active` check and Grok capability-tier promotion (row 22).
This part flags the discrepancy for the assembler to reconcile against part
A's own evidence (a stale-PATH resolution during part A's testing — this
part hit and fixed an analogous PATH-contamination defect in its own harness,
see "Harness-hygiene finding" below — is one plausible explanation) rather
than resolving it, and does not overrule part A's own matrix rows, which this
part did not test.

## Host

| Field | Value |
| ----- | ----- |
| OS | Windows 11 Pro, `10.0.26200` (Build 26200), x64 |
| Git | `git version 2.52.0.windows.1` |
| Go | `go1.26.1 windows/amd64` (used only to build `conptydriver.exe` and run `scripts/tuisandbox`; never to build the artifact under test) |
| Date (UTC) | 2026-09-07 |
| Bench root | `D:\ReinstateAcceptanceProjects\v060-rc5-c\tuisandbox-root` (outside any Git checkout, as `scripts/tuisandbox` requires) |

## Harness-hygiene finding, fixed before any row counted

The ambient host shell's `PATH` carries real, installed vendor CLIs (Grok
Build, Kimi Code, Claude Code, Codex, OpenCode, and others under
`AppData\Roaming\npm`, `.grok\bin`, `.kimi-code\bin`, `nvm4w\nodejs`, etc.),
and its `XDG_DATA_HOME`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`, `COPILOT_HOME`,
`CURSOR_CONFIG_DIR`, `CLINE_DATA_DIR`, `PI_CODING_AGENT_DIR`, `QWEN_HOME`,
and `OH_PERSISTENCE_DIR` are unset (so each vendor falls back to its own
default, host-real location). Neither `scripts/tuisandbox`'s own generated
env fragment nor a naive `PATH` prepend isolates these. Initial runs showed
this concretely: the switcher reported **16 sessions · 4 agents** instead of
the bench's documented **9 sessions · 3 agents**, and — more seriously — the
bench's B1 fixture (`grok:...00b`, meant to be permanently `BLOCKED` because
no `grok` executable should be resolvable) instead showed `◐ Warn` /
`confirmation_required`, because the host's real, installed Grok Build
executable was found on the inherited `PATH`.

**Fix.** Every row in this part ran under a fully isolated environment
(`base-env.ps1`, dot-sourced before every row): `HOME`, `REINSTATE_HOME`,
`CLAUDE_CONFIG_DIR`, `CODEX_HOME` point into the bench root (as
`scripts/tuisandbox` itself sets them); `PATH` is rebuilt from nothing but
the bench's `bin\` directory, `C:\Program Files\Git\cmd` (Git is a
prerequisite of the readiness checks themselves — its total absence produced
a false `git.available: missing` block on every row, a second harness
artifact caught and fixed the same way), and the Windows system directories
— no other directory, so no real vendor CLI is resolvable; `XDG_DATA_HOME`,
`GEMINI_CLI_HOME`, `KIMI_CODE_HOME`, `COPILOT_HOME`, `CURSOR_CONFIG_DIR`,
`CLINE_DATA_DIR`, `PI_CODING_AGENT_DIR`, and `OH_PERSISTENCE_DIR` are
redirected (never left unset) to empty directories under an isolated-homes
root; `GROK_HOME` is deliberately left unset, exactly as
`scripts/tuisandbox/fixtures.go` documents (Grok always resolves
`$HOME/.grok`, and setting an override the adapter does not read only hides
the fixture); and `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR` are
unset per the dispatch's mandatory rule, in every shell, before every row.
Ground truth for both the fix and the original defect was independently
confirmed with `rein resume grok:...00b --dry-run --json`: contaminated
`PATH` → `"decision":"confirmation_required"`, `agent.executable` present
(the real host Grok found); clean `PATH` → `"decision":"blocked"`,
`"block_exit_code":5`, `agent.executable` `missing`. Every row below reports
the bench's documented **9 sessions · 3 agents**, confirming no host session
or host executable leaked into any row's evidence. `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, and `XDG_DATA_HOME` were never left pointed at the host's real
agent homes and never literally `unset` — each was redirected to this part's
own isolated directory, per the dispatch's rule.

## The 22 rows

| # | Row | Result | Evidence |
| - | --- | ------ | ------------------ |
| 1 | Bare `rein` on a capable terminal draws the switcher | PASS | Header `9 sessions · 3 agents · all projects`, `❯ type to filter`, time-grouped rows (TODAY/YESTERDAY/THIS WEEK/THIS MONTH/OLDER), split preview pane, key bar `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit`. Settled frame (10s and 18s checkpoints byte-identical) shows every readiness glyph resolved: `auth-refactor`/`keyring-store` `●`, `website`/`checkout-flow`/`deps-bump`/`search-index` `◐`, `gone-missing`/`payment-adapter`/`repo-drift` `○` — see Row 13 below |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical to `v0.5.1` | PASS | Both exit `2`; stderr SHA-256 byte-identical, `0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b`. Stdout legitimately differs (new subcommands since v0.5.1: `account`, `daemon`, `devices`, `hop`, `login`, `sync`, `whoami`; new `--plain` flag) |
| 3 | `--plain` falls back to the numbered switcher | PASS | Frozen numbered-picker text: `Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` over the same 9 rows |
| 4 | `REINSTATE_NO_TUI=1` does the same | PASS | Raw byte stream SHA-256-identical to row 3 (`694a5559b1bff007611fa832c35764a7c96aee6a20c944ab7645cac0f91eb998`) |
| 5 | `TERM=dumb` does the same | PASS | Rendered frame identical to rows 3/4; raw stream differs only by a trailing console-mode-teardown sequence captured before vs. after `kill`, a harness timing artifact, not a content difference |
| 6 | `TERM` unset draws the switcher on Windows | PASS | Reuses row 1's evidence: every row in this part runs with `TERM` explicitly unset by `base-env.ps1`, and row 1 is the full interactive switcher |
| 7 | A terminal below 40×10 falls back to plain | PASS | `30×8`: wrapped numbered-picker text (`Choose NUMBER, /text, i NUMBER...` wrapped across the narrow width) |
| 8 | `NO_COLOR` draws the switcher with no colour escape sequences | PASS | Raw stream contains exactly one bare `ESC[m` reset and zero SGR sequences with colour parameters |
| 9 | Typing filters; the header count follows | PASS | `"auth"` → `1 session · 1 agent`, list narrows to the one matching row |
| 10 | `f` in list mode filters and does **not** fork | PASS | Filter line reads `f`; header stays `9 sessions · 3 agents` (title text broadly matches "f"); still the switcher list, no fork screen opened |
| 11 | `tab` opens the action menu; `esc` reverts without acting | PASS | `tab` → key bar becomes `r resume f fork h hand off i inspect y copy ref esc back` with an `actions` label; `esc` → key bar reverts exactly to `↵ resume tab actions ctrl+a scope ctrl+k commands esc quit` |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | PASS | `ctrl+k` then `"hof"` → top match `▸ Hand off to another… new session from a briefing`, with `4 more matches` below |
| 13 | Readiness glyphs resolve for visible rows and a read-only agent shows blocked without a probe | **PASS — confirmed fixed** | See "Row 13 — full evidence" below |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | PASS | One `space` toggles exactly one warning (`[ ] baseline.unavailable` → `[x] baseline.unavailable`; other 3 of 4 stay `[ ]`); equivalent command updates live from `rein resume claude:...003` to `rein resume claude:...003 --allow-environment-warning baseline.unavailable`; a second `space` reverts both the checkbox and the command line exactly |
| 15 | A partial acknowledgement is refused with exit `7` | PASS | 2 of 4 warnings acknowledged (`baseline.unavailable`, `git.branch`) on a real (non-dry-run) resume attempt: exit `7`, stderr names exactly the remaining two, `environment warnings require confirmation: git.working_tree, runtime.node.declaration` |
| 16 | The handoff studio measures each policy and the equivalent command follows the selection | PASS | Full measurement: `carried across` / `left behind` field breakdown, `2 warnings to acknowledge`; `key right` cycles `balanced → full → checkpoint`, and the `equivalent command` line updates in lock-step (`--policy balanced` → `--policy full` → `--policy checkpoint`) |
| 17 | The studio refuses `enter` on a plan that could not be built | PASS | `repo-drift` (foreign `repository_url`) session: studio shows `○ this handoff cannot be planned / handoff: environment preflight is blocked`; `enter` does not send — the same refusal line is re-shown (`this handoff cannot be planned: handoff: environment preflight is blocked`), studio stays open |
| 18 | `rein init` opens the wizard, validates per field, and allows going back | PASS | Step 1/7 provider list, `MinIO or self-hosted` selected; step 2/7 endpoint `"not a valid endpoint"` refused **in place** with `the endpoint must start with https:// or http://`, stays on step 2; `key up` returns to step 1 with `MinIO or self-hosted` still selected (preserved) |
| 19 | `rein init` collects no secret material inside the full-screen program | PASS | Walked all 7 steps (provider, endpoint, bucket, region, key prefix, this-device, review) — none asks for an access key, secret key, or passphrase; the review step states in-product: "Next you will enter your storage keys, then a passphrase." (i.e., after the full-screen program exits); `rein init --help` has no credential flag |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | PASS | Real two-"device" round trip (two isolated `REINSTATE_HOME`s on this host) against `scripts/testing/fakelocker` (the product's own fake-S3 fixture) at `http://127.0.0.1:9100`. Device A: `rein init --yes --endpoint http://127.0.0.1:9100 --bucket testbucket --region auto --prefix devA` → `profile_id=07f16384-f1c4-4789-b4f3-088733243cac`; `rein init --link` → prints a `REIN1-...` pairing code. Device B: `rein init --paste`, code sent → wizard opens pre-filled with **exactly** matching `endpoint`/`bucket`/`region`/`prefix`, a `Profile ID` step showing the identical UUID `07f16384-f1c4-4789-b4f3-088733243cac`, and `This device` pre-selected to `Join a profile from another device`; review screen confirms all five fields match device A field-for-field |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | PASS | `WT_SESSION`/`TERM_PROGRAM`/`TERM_PROGRAM_VERSION`/`LANG`/`LC_ALL`/`LC_CTYPE` all unset: every glyph is the ASCII set — `*`/`!`/`x` readiness marks, `/` filter caret, `>` cursor, `\|` pane divider, `...` ellipsis — never a Unicode character anywhere on the frame |
| 22 | Every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences | PASS | 5 documents compared on the identical bench: `handoff --no-launch --json` byte-identical; `doctor --json` differs only in the version string and `generated_at` timestamp (trivial, expected of any version bump); `sessions --json`, `resume --dry-run --json`, and `inspect --json` differ in exactly two documented classes — (1) a new `agent.active` liveness check on every environment report, and (2) Grok Build's capability/tier promotion (removed `read_only_reason`, `can_resume`/`can_fork` flipped `true`, `agent.layout`/`agent.version` upgraded from blocking to informational) — both traced to the same CHANGELOG entry ("Grok Build moves to T3, verified resume" / the `agent.active` liveness check introduced alongside it). No third class found or needed |

**22/22 PASS.**

### Row 13 — full evidence (confirmed fixed)

This is the row `v0.6.0-rc.4`'s tagged run found failing (every visible row
`CANNOT RESUME`/`Blocked` regardless of true state, in the switcher's default
all-projects scope) and this candidate's changelog claims fixed by bounding
`internal/tui/readiness.Prober`'s concurrent verifications to a small fixed
pool regardless of page size.

**From outside any tracked project (`ScopeAll`, the default, launched from
the bench's `home` root — not inside any of the bench's 8 project
directories).** Bare `rein`, header reads `9 sessions · 3 agents · all
projects`. Immediately after the first frame (`frame-initial.txt`), every
glyph is `◌` (pending) — expected, the probes have not returned yet. At a
10-second checkpoint and again at 18 seconds (byte-identical to the 10s
checkpoint — fully settled, not still resolving), every glyph matches its
seeded state:

| Session | Seeded state | Observed glyph | Correct? |
| ------- | ------------- | --------------- | -------- |
| `claude:auth-refactor` (R1) | READY | `●` | yes |
| `codex:keyring-store` (R2) | READY | `●` | yes |
| `claude:website` (W1, 4 warnings) | WARN | `◐` | yes |
| `claude:checkout-flow` (W4, 1 warning, permanent) | WARN | `◐` | yes |
| `claude:deps-bump` (W3, 1 warning, self-heals) | WARN | `◐` | yes |
| `codex:search-index` (W2, 3 warnings) | WARN | `◐` | yes |
| `claude:gone-missing` (B2, workspace missing) | BLOCKED | `○` | yes |
| `codex:repo-drift` (B3, foreign repository_url) | BLOCKED | `○` | yes |
| `grok:payment-adapter` (B1, read-only, no executable) | BLOCKED | `○` | yes |

Independently cross-checked against `rein resume grok:...00b --dry-run
--json` and `rein resume claude:...001 --dry-run --json` under the same
isolated harness: `"decision":"blocked","block_exit_code":5` (grok,
`agent.executable` `missing`) and `"decision":"ready"` (claude R1, every git
and agent check `match`), both matching what the switcher showed. This is
the corrected behaviour: at `v0.6.0-rc.4`, every one of these 9 rows showed
`○ CANNOT RESUME` regardless of this ground truth; here, all 9 resolve
correctly.

**From inside a tracked project (`ScopeProject`, cwd =
`.../home/Projects/auth-refactor`).** Header auto-narrows to `1 session · 1
agent · auth-refactor`; the one visible row (`claude:auth-refactor`, R1)
settles to `● READY TO RESUME` — matching its `ScopeAll` result above and its
independent `--dry-run --json` ground truth.

**Disposition: fixed, confirmed.** Matches the rc.5 dispatch's expectation
exactly (`docs/testing/v0.6.0-rc.5-agent-verification-prompts.md`'s carried
dispositions table: "Fixed — expect `PASS`; the switcher's all-projects
scope resolves readiness correctly regardless of page size").

## Harness observations (non-blocking, this run)

- A handoff-studio frame captured mid-repaint (row 16's `balanced.txt`, and
  `third-policy.txt`) shows some overlapping/un-cleared text from the
  previous frame at a few cell positions (e.g. a stray `▸` or fragment of
  the prior policy's description bleeding into a field label). The
  underlying data on both frames is legible and correct; this is a
  snapshot-timing artifact of capturing mid-transition, not a rendering
  defect — a `sleep` before each snapshot was already used, and a longer one
  would likely clear it, but the assertion under test (policy cycling, the
  equivalent-command line) is unambiguous either way.
- `send "text\n"` (a single step, escaped newline) reliably fails to submit
  to a plain, non-alt-screen prompt under this harness (the wizard's
  post-review access-key/secret-key/passphrase prompts, row 20); a separate
  `send "text"` followed by its own `key enter` step works every time. This
  matches the same trap the `v0.6.0-rc.4` tagged run recorded for a
  `bufio.Scanner`-based prompt.
- `key h` typed directly from the switcher's **list** mode (not the `tab`
  action menu) is consumed as a filter character, not the handoff shortcut —
  correct per the product's own contract ("Letters always filter. Actions
  live behind `tab` and `ctrl+k`"), but a script that presses `h` without
  first pressing `tab` reaches the wrong screen. Every row in this part that
  needed the handoff studio (16, 17) goes through `tab` first.
- Readiness settle time was slower on this host under real-TTY ConPTY
  sessions than a first guess suggests: several rows needed a 6–10 second
  `sleep` after the first frame before glyphs stopped changing; row 1's own
  two checkpoints (10s, 18s byte-identical) established the actual settle
  bound empirically rather than assuming one.

## Commands used

Every command below ran with `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`
unset and the isolated `PATH`/home-variable set described above.

```
conptydriver.exe -cols 100 -rows 30 -script <row>.txt -raw raw.log -exit-timeout 5-8s -- rein.exe [args]
rein.exe resume <ref> --dry-run --json
rein.exe resume <ref> --allow-environment-warning <ID> [--allow-environment-warning <ID> ...]
rein.exe sessions --json
rein.exe inspect <ref> --json
rein.exe doctor --json
rein.exe handoff <ref> --to codex --no-launch --json --allow-warning <ID> --allow-warning <ID>
rein.exe init [--yes --endpoint URL --bucket NAME --region auto --prefix PREFIX] [--link] [--paste]
```

Step-script grammar per `scripts/testing/conptydriver/README.md`; every
`conptydriver.exe` invocation ran under `Start-Process -WindowStyle Hidden`
(never with its own stdout/stderr redirected) so it received a genuine
console of its own, per that README's documented trap.

- Terminating tester: executor C
- UTC timestamp: 2026-09-07
- Part verdict: **22/22 PASS**, zero release-blocking findings in this part's
  own scope (section C of the rc.5 dispatch). See "Cross-check note" above
  for a discrepancy this part flags but does not resolve.
