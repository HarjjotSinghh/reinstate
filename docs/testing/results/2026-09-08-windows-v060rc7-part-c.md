# v0.6.0-rc.7 tagged Windows acceptance — Part C (CLI experience, 22 rows)

Executor C. Source: the 22 rows of the `v0.5.2` CLI experience contract
(Windows column), run through
[`scripts/testing/conptydriver`](../../scripts/testing/conptydriver) against
[`scripts/tuisandbox`](../../scripts/tuisandbox), per the
[`v0.6.0-rc.7` tagged-artifact dispatch](v0.6.0-rc.7-agent-verification-prompts.md)
and the [`v0.6.0` native Windows acceptance contract](v0.6.0-windows-acceptance.md#c--cli-experience-required).

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.7` |
| Full commit | `21e9a9a356df1181601e2da02bfae03b28380cc3` |
| Windows archive | `reinstate_0.6.0-rc.7_windows_amd64.zip` |
| Archive SHA-256 | `865ad8e8f65fb3f543d2e2670b5d7c82b0d4878513781e04868fc7303e183fcf` (verified against the coordinator's `checksums.txt` before install, and re-verified again here) |
| Install source | `checksums.txt`-matched, `gh attestation`-verified copy the coordinator staged; **not** the live bootstrap (executor A alone installs from `https://reinstate.dev/install.ps1` and records that as the run's artifact-identity source; every other executor, including this one, installs from the staged, checksummed archive per the dispatch's own instruction) |
| `rein.exe` / `reinstate.exe` | byte-identical, SHA-256 `ab73dd7c6817da2374a9c72673c62c63fb724d7412fa464d5dc56d381d5ea349` |
| `rein version --json` | `{"commit":"21e9a9a356df1181601e2da02bfae03b28380cc3","date":"2026-09-08T16:26:36Z","name":"reinstate","version":"0.6.0-rc.7"}` |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-rc7-c\install\` (own, fresh; never a user-installed binary) |
| `v0.5.1` comparison binary | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches the coordinator's `checksums.txt`), extracted to `D:\ReinstateAcceptanceProjects\v060-rc7-c\v051\` |

**Bootstrap deviation sentence.** This report's artifact identity comes from
the coordinator-staged, checksum- and attestation-verified copy at
`…\scratchpad\rc7-draft`, not from the live `reinstate.dev` bootstrap — per
the dispatch, only executor A installs from the live bootstrap and records
that as the run's bootstrap-identity evidence; this executor's install is a
second, independent installation of the identical checksummed artifact.

## Host

| Field | Value |
| ----- | ----- |
| OS | Windows 11 Pro, build `10.0.26200`, native Windows x64 (`windows/amd64`), never WSL |
| Shell | Git Bash (MSYS) for non-interactive/env-comparison rows; Windows PowerShell 5.1 for every ConPTY-driven row |
| Go toolchain | `go1.26.1 windows/amd64` (satisfies the `go1.25.13+` floor) |
| Git | `2.52.0.windows.1` |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc7-tagged` (branch `v060/rc7-tagged`, at `21e9a9a356df1181601e2da02bfae03b28380cc3`) |
| Test root | `D:\ReinstateAcceptanceProjects\v060-rc7-c\` (own directory; deleted of session-bearing content is not required here since `tuisandbox` writes only synthetic fixtures, no real agent data) |
| `REINSTATE_BACKEND` / `REINSTATE_MEMORY_BACKEND_DIR` | Unset at the top of every shell that ran `rein`, confirmed by `envsetup.ps1`/`envsetup.sh` (checked in as part of this run's scratch tooling, not committed) doing `Remove-Item Env:… -ErrorAction SilentlyContinue` / `unset` first, every invocation |
| `scripts/testing/conptydriver` | Built from this tree (`go build ./scripts/testing/conptydriver`) into the test root, not committed |

## Harness note: `XDG_DATA_HOME` isolation gap in `scripts/tuisandbox`

`scripts/tuisandbox`'s legend documents redirecting `HOME`, `REINSTATE_HOME`,
`CLAUDE_CONFIG_DIR`, and `CODEX_HOME`, but **not** `XDG_DATA_HOME` — OpenCode's
own `root_env` (confirmed via `rein doctor --agents --json`:
`"root_env":"XDG_DATA_HOME","root_env_set":true`). On this host that variable
was already set to the operator's real OpenCode data root, and a first,
naive switcher launch showed **31 real host OpenCode sessions** (real
project names, one identifiably the operator's own employer name) mixed
into the tuisandbox bench's 9 synthetic rows — a direct hit on the evidence
policy's "never point an interactive surface at the developer's real agent
tree" rule. Caught before any capture was taken from that state (the
probe frame with the leaked data was never saved to this report or
committed). Fixed for the remainder of this run by additionally pointing
`XDG_DATA_HOME` at an empty, isolated directory under the test root (per
the ground rules: redirect the vendor home variable, never unset it) —
confirmed afterward that `rein sessions --json` reports exactly the 9
synthetic sessions and no others. Recorded here as a harness defect in
`scripts/tuisandbox` (its env-isolation legend is incomplete for hosts where
`XDG_DATA_HOME` is already populated), not a product defect, and not scored
against any row below since every row's own evidence was captured only
after the fix.

## Verdict

**21 PASS / 1 FAIL (new finding, row 13) / 0 PARTIAL / 0 NOT TESTED** of 22
required rows.

| # | Row | macOS | Windows |
| - | --- | ----- | ------- |
| 1 | Bare `rein` on a capable terminal draws the switcher | n/a (deferred) | PASS |
| 2 | Bare `rein` on a non-TTY exits `2`, byte-identical to `v0.5.1` | n/a (deferred) | PASS |
| 3 | `--plain` falls back to the numbered switcher | n/a (deferred) | PASS |
| 4 | `REINSTATE_NO_TUI=1` does the same | n/a (deferred) | PASS |
| 5 | `TERM=dumb` does the same | n/a (deferred) | PASS |
| 6 | `TERM` unset draws the switcher on Windows | n/a (deferred) | PASS |
| 7 | A terminal below 40×10 falls back to plain | n/a (deferred) | PASS |
| 8 | `NO_COLOR` draws the switcher with no color escapes | n/a (deferred) | PASS |
| 9 | Typing filters; header count follows | n/a (deferred) | PASS |
| 10 | `f` in list mode filters, does not fork | n/a (deferred) | PASS |
| 11 | `tab` opens the action menu; `esc` reverts without acting | n/a (deferred) | PASS |
| 12 | `ctrl+k` opens the palette; a subsequence query finds its command | n/a (deferred) | PASS |
| 13 | Readiness glyphs resolve for visible rows; a read-only agent shows blocked without a probe | n/a (deferred) | **FAIL — new finding** |
| 14 | The warning checklist acknowledges with the spacebar and shows the equivalent command | n/a (deferred) | PASS |
| 15 | A partial acknowledgement is refused with exit `7` | n/a (deferred) | PASS |
| 16 | The handoff studio measures each policy and the equivalent command follows | n/a (deferred) | PASS |
| 17 | The studio refuses `enter` on a plan that could not be built | n/a (deferred) | PASS |
| 18 | `rein init` opens the wizard, validates per field, allows going back | n/a (deferred) | PASS |
| 19 | `rein init` collects no secret material inside the full-screen program | n/a (deferred) | PASS |
| 20 | `rein init --link` prints a code that `--paste` consumes on the other device | n/a (deferred) | PASS |
| 21 | Glyphs degrade to ASCII where Unicode is not safe (legacy conhost) | n/a | PASS |
| 22 | Every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences | n/a (deferred) | PASS |

macOS is deferred under ADR 0005 for this candidate; nothing above claims
macOS evidence.

## Evidence

All commands below ran with `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`
unset, `HOME`/`REINSTATE_HOME`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME`
redirected under `D:\ReinstateAcceptanceProjects\v060-rc7-c\tuisandbox-root\`,
and the tagged `rein.exe` invoked by full path. `TERM` was never set.

### Row 1 — bare `rein` draws the switcher

```
rein
```

`9 sessions · 3 agents · all projects` header, filter prompt, time-grouped
rows (`TODAY`/`YESTERDAY`/`THIS WEEK`/`THIS MONTH`/`OLDER`), a split preview
pane, and the key bar all draw correctly and identically across every one of
the 15+ separate launches gathered for this report (see row 13, whose own
defect is in the readiness *glyph value* each row resolves to, not in
whether the switcher structure itself draws). Two checkpoints of one launch
10s and 18s apart are byte-identical (settled, non-flickering frame).

### Row 2 — non-TTY exits `2`, byte-identical to `v0.5.1`

```
rein < /dev/null > out.txt 2> err.txt
```

Both `v0.6.0-rc.7` and `v0.5.1` exit `2`. stderr is byte-identical between
the two binaries and matches every prior candidate's own recorded hash:

```
interactive session picker requires a terminal; use `rein sessions --json`
```

SHA-256 of stderr, both binaries: `0476fb687fa1c1209d243016ab020460b8ad55696e4052f491b1fcb8cdb4795b`.
stdout differs only in the command/flag list (new `account`, `daemon`,
`devices`, `hop`, `login`, `whoami` subcommands and the `--plain` flag,
all shipped and changelog-documented since `v0.5.1`), which the row does
not constrain.

### Row 3 — `--plain` falls back to the numbered switcher

```
rein --plain
```

`Local sessions` header, all 9 rows numbered `1`–`9`, `Choose NUMBER, /text,
i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:` prompt.

### Row 4 — `REINSTATE_NO_TUI=1` does the same

```
REINSTATE_NO_TUI=1 rein
```

Output SHA-256-identical to row 3's frame.

### Row 5 — `TERM=dumb` does the same

```
TERM=dumb rein
```

Output SHA-256-identical to rows 3 and 4.

### Row 6 — `TERM` unset draws the switcher on Windows

Confirmed `$env:TERM` unset (`$null -eq $env:TERM` → `true`), then:

```
rein
```

Full interactive switcher drawn (not the plain fallback) — same structure
as row 1.

### Row 7 — a terminal below 40×10 falls back to plain

ConPTY console sized 30×8, then:

```
rein
```

Numbered picker, wrapped to the small viewport, ending on `9  codex
search-index …` and the `Choose NUMBER, …` prompt.

### Row 8 — `NO_COLOR` draws the switcher with no color escapes

```
NO_COLOR=1 rein
```

captured with `-raw` alongside the rendered frame. The switcher draws in
full; scanning the raw byte stream for `ESC[...m` (SGR) sequences finds
exactly one, the bare reset `ESC[m`, and zero sequences carrying any SGR
parameter (no color).

### Row 9 — typing filters; header count follows

From the settled 9-row frame, `send "auth"` → header becomes `1 session · 1
agent · all projects`, list narrows to the single `auth-refactor` row.

### Row 10 — `f` in list mode filters, does not fork

From the settled 9-row frame, `key f` → filter prompt shows `f`; header
stays `9 sessions · 3 agents · all projects` (all rows, `f` alone below
the live-filter's effective minimum, matches broadly); no fork screen
appears anywhere in the frame.

### Row 11 — `tab` opens the action menu; `esc` reverts without acting

`key tab` → header row grows an `actions` label and the key bar becomes `r
resume   f fork   h hand off   i inspect   y copy ref   esc back`. `key esc`
→ frame is byte-identical to the pre-`tab` snapshot (whole-file diff: 0
lines).

### Row 12 — `ctrl+k` opens the palette; a subsequence query finds its command

`key ctrl+k` then `send "hof"` → palette overlay opens with prompt `❯ hof`
and top match `▸ Hand off to another…  new session from a briefing`.

### Row 13 — readiness glyphs resolve for visible rows [FAIL — new finding, regression from `v0.6.0-rc.6`]

**Method** (per the rc.4 report's own row-13 method, reused because it is
the one that originally caught this defect class): launch bare `rein` from
a cwd **outside any tracked project** (`D:\ReinstateAcceptanceProjects\v060-rc7-c\out`,
not inside any Git checkout) — the switcher's documented default
`ScopeAll` — wait well past settle (10s–47s, confirmed byte-identical
across checkpoints within a run), and cross-check every row's rendered
glyph against ground truth from `rein resume <id> --dry-run --json` /
`rein inspect <id> --json` on the same 9 fixture sessions. Then repeat from
inside a tracked project workspace (`ScopeProject`).

**Ground truth** (`● Ready`, `◐ Warn`, `○ Blocked`), by session:

| Session | Ground truth (`resume --dry-run --json` / exit code) | Expected glyph |
| ------- | ------------------------------------------------------ | --------------- |
| `claude:…0001` (auth-refactor) | `decision: ready` | `●` |
| `codex:…0002` (keyring-store) | `decision: ready` | `●` |
| `claude:…0003` (website) | `decision: confirmation_required` | `◐` |
| `claude:…0009` (checkout-flow) | `decision: confirmation_required` | `◐` |
| `claude:…0004` (gone-missing) | exit `5` (workspace missing) | `○` |
| `grok:…000b` (payment-adapter) | `decision: confirmation_required` | `◐` |
| `codex:…0006` (repo-drift) | exit `7` (foreign `repository_url`) | `○` |
| `claude:…0007` (deps-bump) | `decision: confirmation_required` | `◐` |
| `codex:…0008` (search-index) | `decision: confirmation_required` | `◐` |

(`grok:…000b`'s own `tuisandbox` legend text calls it a permanently
`BLOCKED`, read-only row — stale relative to Grok Build's own tier
promotion to T3/T4 verified resume, documented in `CHANGELOG.md`'s
`[0.5.2-rc.1]`/`[0.6.0-rc.1]` entries: `rein inspect` on this exact session
now reports `can_resume:true, can_fork:true`, no `read_only_reason`. This
is a `tuisandbox` comment-staleness note, not a defect on its own, and
ground truth — not the stale comment — is what row 13 is scored against.)

**Result across 15 independent `ScopeAll` launches** (`D:\…\out`, never
inside a Git checkout; a fresh, cleared probe cache was also exercised for
3 of the 15 to rule out cross-run cache staleness as the cause):

| Trial group | Launches | Fully matched ground truth | Had ≥1 wrong row |
| ------------ | -------- | --------------------------- | ----------------- |
| Initial 10s/18s checkpoints | 2 | 0 | 2 |
| 20s/35s checkpoint | 1 | 1 | 0 |
| 12s single-checkpoint trials | 5 | 1 | 4 |
| 12s/27s/47s checkpoints | 4 | 2 | 2 |
| Cold (cleared) probe cache | 3 | 0 | 3 |
| **Total** | **15** | **4 (27%)** | **11 (73%)** |

The wrong glyph is **always** `○` (Blocked) standing in for a row whose
ground truth is `●` (Ready) or `◐` (Warn) — never the reverse — and which
specific row(s) go wrong varies between launches (not the same row every
time). One representative wrong trial (worst observed, 8 of 9 rows wrong):

```
 rein                                      9 sessions · 3 agents · all projects
 ...
▸ ○ claude   auth-refac… Fix the aut…    (expected ●)
  ○ codex    keyring-st… Wire the ke…    (expected ●)
  ○ claude   website     Open graph …    (expected ◐)
  ○ claude   checkout-f… Checkout th…    (expected ◐)
  ○ claude   gone-missi… Probe the v…    (○, correct)
  ○ grok     payment-ad… Trace the f…    (expected ◐)
  ○ codex    repo-drift  認証リファ…     (○, correct)
  ◐ claude   deps-bump   Bump the pi…    (◐, correct)
  ○ codex    search-ind… Try a small…    (expected ◐)
```

**Non-self-correcting.** Four trials held a wrong frame open through three
checkpoints spanning 12s, 27s, and 47s (`vl-3-t12/t27/t47`,
`vl-4-t12/t27/t47`, each internally byte-identical across all three
checkpoints) — a wrong verdict is not a slow-to-settle one; once cached
wrong for a given process's lifetime, it stays wrong.

**`ScopeProject` is unaffected.** Launched from inside the `auth-refactor`
workspace (`ScopeProject`, "1 session · 1 agent · auth-refactor"),
`auth-refactor` correctly shows `● READY TO RESUME` — checked once,
consistent with `v0.6.0-rc.6`'s own row 13 evidence, and not itself in
question here.

**Where this points in the source** (read-only observation, no rebuild
attempted): `internal/tui/readiness/prober.go`'s own doc comment on
`maxConcurrentProbes` already names this exact failure shape: "the
resulting process-creation stampede blows through each report's own
timeout budget before its checks can finish, and a timed-out check reports
itself blocked — indistinguishable, on screen, from a session that
genuinely cannot resume." `maxConcurrentProbes = 4` bounds concurrency but,
on this host, does not eliminate the effect — plausibly because this
acceptance run shares the host with three other executors' own concurrent
`rein`/`go test`/vendor-CLI processes (a real, if acceptance-run-specific,
form of host contention). `FromReport`'s own doc comment says a probe
*error* should map to `Unknown` (pending glyph `◌`), not `Blocked` — the
wrong glyph observed here is unambiguously `○` (`Blocked`, confirmed by
exact Unicode codepoint, not the visually similar `◌` Pending), so the
wrong verdict is most consistent with `preflight.Verify`'s own internal
per-check timeout resolving to a `blocked`-severity check result (a
successful, non-error probe that itself concluded the session is
blocked), not with the outer probe's error path — this last step was not
traced into `internal/preflight` itself; a rebuild against the tagged
binary's own source was not attempted, consistent with this run's
read-only evidence boundary.

**Disposition.** `FAIL`, **new finding, not a carried disposition** — the
`v0.6.0-rc.7` dispatch explicitly expects CLI row 13 to `PASS` again
unchanged, carried from `v0.6.0-rc.6`'s own clean `PASS`, since nothing in
this candidate touches the switcher or the readiness prober. This
candidate's Section B (Phase 5) and Section D (Hop parity) results, if
found clean elsewhere in this run's assembled report, are not affected by
this finding — it is isolated to the interactive switcher's `ScopeAll`
readiness-glyph pipeline. Given the mechanism reproduces wrong ~73% of the
time under this exact, contract-specified method, this is scored `FAIL`
rather than `PARTIAL`: the row's own text ("readiness glyphs resolve for
visible rows") is not met on most launches, and "resolve" cannot fairly be
read to include "resolve to a value that happens to be wrong."

### Row 14 — the warning checklist acknowledges with the spacebar [PASS]

Selected the `website` session (`W1`, 4 warnings: `baseline.unavailable`,
`git.branch`, `git.working_tree`, `runtime.node.declaration`), pressed
enter to reach the checklist. `key space` on the first item toggles
`[ ]`→`[x]` for exactly `baseline.unavailable` and no other row; the
"equivalent command" line live-updates from

```
rein resume claude:5f0a1c00-0000-4000-8000-000000000003
```

to

```
rein resume claude:5f0a1c00-0000-4000-8000-000000000003 --allow-environment-warning baseline.unavailable
```

A second `key space` reverts the checkbox to `[ ]` and the command line
back to its base form — the pre-toggle and reverted frames are SHA-256
byte-identical.

### Row 15 — a partial acknowledgement is refused with exit `7` [PASS]

```
rein resume claude:5f0a1c00-0000-4000-8000-000000000003 --allow-environment-warning baseline.unavailable --allow-environment-warning git.branch
```

Exit `7`. Message:

```
environment warnings require confirmation: git.working_tree, runtime.node.declaration
```

Names exactly the two of four warnings not acknowledged. The interactive
checklist shows the equivalent inline refusal (`every warning must be
acknowledged before continuing`) and stays open on the same `enter`
keystroke, matching the CLI's own refusal.

### Row 16 — the handoff studio measures each policy [PASS]

From the `auth-refactor` workspace, `tab` → `h` opens the studio
(`claude:…0001` → `codex`, default policy `balanced`). Cycling policy with
`key right` walks `balanced → full → checkpoint → balanced` (wraps
correctly), with the equivalent command tracking in lock-step
(`--policy balanced|full|checkpoint`) and a **real** projection recomputed
per policy — e.g. `checkpoint` drops `user_messages` from "carried across"
and adds `files_touched_per_transcript` to "left behind", where `balanced`
and `full` both carry `user_messages`.

### Row 17 — the studio refuses `enter` on a plan that could not be built [PASS]

From the `repo-drift` workspace (`codex:…0006`, foreign `repository_url`,
`BLOCKED`), the studio shows `○ this handoff cannot be planned / handoff:
environment preflight is blocked` before any keystroke. `key enter` adds
an identical inline refusal line and the studio stays open (does not
crash or exit); a second `enter` produces a frame SHA-256-identical to
the first refusal.

### Row 18 — `rein init` opens the wizard, validates, allows going back [PASS]

```
rein init
```

7-step wizard. Selected "Other S3-compatible" at step 1, `enter` → step 2
(Endpoint). Typed `not-a-valid-url`, `enter` → inline error `the endpoint
must start with https:// or http://`, stays on step 2. Sent the raw CSI
sequence for shift+tab (`send "\x1b[Z"` — `conptydriver`'s grammar has no
named `shift+tab` key; the wizard's own key bar advertises
`shift+tab back`, and Bubble Tea's own decoder recognizes the raw `CSI Z`
sequence a real terminal's Shift+Tab sends) → returns to step 1 with
"Other S3-compatible" still selected.

### Row 19 — `rein init` collects no secret material inside the full-screen program [PASS]

Walked all 7 steps to Review (provider/endpoint/bucket/region/prefix/device
role), confirming no credential field anywhere in the full-screen program.
`rein init --help` lists `--bucket`, `--endpoint`, `--force`, `--hop`,
`--link`, `--paste`, `--prefix`, `--profile-id`, `--project`, `--region`,
`--yes` — no access-key/secret-key/passphrase flag. Pressing `enter` on
the Review step's "start setup" **exits the full-screen program entirely**
(confirmed: the next captured frame is plain scrollback text, not a
Bubble Tea frame) before attempting the storage probe — so any subsequent
key/passphrase prompt (the review step's own text: "Next you will enter
your storage keys, then a passphrase…") necessarily happens outside the
full-screen renderer, which had already returned control to the plain
terminal by that point.

### Row 20 — `rein init --link` prints a code `--paste` consumes on the other device [PASS]

Real two-device round trip against a local fake S3-compatible endpoint
(`scripts/testing/fakelocker`, `127.0.0.1:9123`, lab fixture, nothing
persisted, never a real bucket). Device A: `rein init --yes --endpoint
http://127.0.0.1:9123 --bucket labbucket --region auto` (env credential
provider `REINSTATE_S3_ACCESS_KEY_ID`/`REINSTATE_S3_SECRET_ACCESS_KEY`)
completed with `profile_id=7e34918f-c0ee-4921-81aa-5704dfa53dda`. `rein
init --link` printed a wrapped pairing code. Device B, a separate isolated
`REINSTATE_HOME`: `rein init --paste`, pasted the reconstructed code
(concatenated verbatim from the wrapped display, no re-typing) — the
wizard opened pre-filled and, walked through all 8 steps to Review, showed
every field matching device A exactly:

| Field | Device A | Device B (pasted) |
| ----- | -------- | ------------------ |
| Provider | Other S3-compatible | Other S3-compatible |
| Endpoint | `http://127.0.0.1:9123` | `http://127.0.0.1:9123` |
| Bucket | `labbucket` | `labbucket` |
| Region | `auto` | `auto` |
| Prefix | `profiles/7e34918f-c0ee-4921-81aa-5704dfa53dda` | `profiles/7e34918f-c0ee-4921-81aa-5704dfa53dda` |
| Profile ID | `7e34918f-c0ee-4921-81aa-5704dfa53dda` | `7e34918f-c0ee-4921-81aa-5704dfa53dda` (device role auto-set to "Join a profile from another device") |

### Row 21 — glyphs degrade to ASCII on legacy conhost [PASS]

Cleared `WT_SESSION`, `TERM_PROGRAM`, `LANG`, `LC_ALL`, `LC_CTYPE` (and
`TERM`, already unset) — the exact variable set `internal/ui/capability.go`'s
`detectUnicode` and `windowsUnicodeDefault` consult — then:

```
rein
```

Every UI glyph rendered ASCII: `*` (ready), `!` (warn), `x` (blocked), `>`
(cursor), `/` (search prompt), `|`/`-` (box drawing), `enter` spelled out
in the key bar rather than `↵`. (Session *content* — the Japanese fixture
title `認証リファクタを修正する` — is untouched, as expected: the row is
about the UI's own glyph set, not arbitrary Unicode text in session data.)

### Row 22 — every `--json` document is byte-identical to `v0.5.1`, except changelog-explained differences [PASS]

Compared on the identical `tuisandbox` home, both binaries invoked by full
path:

| Document | Diff found | Explained by |
| -------- | ---------- | ------------- |
| `sessions --json` | `grok` session: `capabilities.resume`/`.fork` `false→true`, `read_only_reason` removed | `CHANGELOG.md` `[0.5.2-rc.1]`/`[0.6.0-rc.1]`: "Grok Build moves to T3, verified resume" (capability/tier-promotion class) |
| `resume <claude ready session> --dry-run --json` | new `agent.active` check present | `CHANGELOG.md` `[0.5.2-rc.1]`: the `agent.active` liveness check, shipped for every T3+ agent alongside OpenCode/Qwen reaching T3 | 
| `inspect <same session> --json` | same `agent.active` addition | same |
| `inspect grok:…000b --json` | `can_resume`/`can_fork` `false→true`, `read_only_reason` removed, `agent.executable`/`agent.layout`/`agent.version`/`agent.active` change from `missing`/`unrecognized`/absent to `present`/`recognized`/`1.0.13`/`match` | same Grok tier-promotion class as above — `v0.5.1`'s catalog does not recognize Grok as a supported agent at all, so it never probes the real binary; `v0.6.0-rc.7`'s does |
| `handoff <claude ready session> --to codex --dry-run --json` | none | byte-identical |
| `doctor --json` | `version` field only | expected version-string drift, not a behavior change |

No unexplained (unmatched-to-a-changelog-entry) difference was found in
any of the six documents compared. This report found exactly the **two**
documented accepted classes named in
[`v0.6.0-windows-acceptance.md`](v0.6.0-windows-acceptance.md#c--cli-experience-required)
(tier-promotion capability fields; the `agent.active` check) — no
independent `v0.6.0-rc.7`-specific pre-tag report naming a third class
could be located in this worktree to reconcile against the "three
accepted classes" language in this run's own dispatch text; see
Questions below.

## Release-blocking findings

- **CLI row 13 — new finding, not carried.** The interactive switcher's
  default `ScopeAll` scope resolves at least one visible row's readiness
  glyph to the wrong value (`○ Blocked` standing in for `●`/`◐`) in 11 of
  15 independent launches (73%) using the exact contract-specified method,
  non-self-correcting within a launch's lifetime once wrong. `ScopeProject`
  is unaffected. See row 13's full evidence above for the source-level
  pointer (`internal/tui/readiness/prober.go`'s own documented
  process-creation-stampede failure mode). This blocks CLI row 13 and, by
  the contract's own "a row is PASS only if its mechanism was exercised"
  rule, the Section C device verdict.

## Harness defects (non-blocking)

- `scripts/tuisandbox`'s env-isolation legend omits `XDG_DATA_HOME`
  (OpenCode's own `root_env`); on a host where that variable is already
  set, the switcher mixes real host OpenCode sessions into the bench.
  Caught and fixed before any evidence was captured; see "Harness note"
  above.
- `grok:…000b`'s own generator comment (`scripts/tuisandbox/main.go`)
  describes it as permanently, hardcoded `BLOCKED` — stale relative to
  Grok Build's real tier promotion; ground truth (not the stale comment)
  is what this report scores against.
- `scripts/testing/conptydriver`'s grammar has no named `shift+tab` key
  (documented gap); the raw `CSI Z` byte sequence via `send "\x1b[Z"`
  substitutes successfully (row 18).
- Launching `conptydriver.exe` in a way that redirects its own
  stdout/stderr (the default result of running it from an automation
  tool) reproduces the documented "needs a real console of its own" trap
  (`interactive session picker requires a terminal`) even though the
  target binary is genuinely being driven correctly; worked around per
  the tool's own README using PowerShell `Start-Process -WindowStyle
  Hidden` (no `-RedirectStandardOutput`/`-RedirectStandardError`) with all
  results written to files.

## Questions

- This run's own dispatch text says "the pre-tag report lists the three
  accepted classes" for row 22's amended definition. Only two classes are
  named in `v0.6.0-windows-acceptance.md` itself, and this report's own
  empirical diff across six `--json` documents found and fully explained
  exactly those two, with zero unmatched differences. No `v0.6.0-rc.7`
  pre-tag report (`docs/testing/results/*rc7-pretag*`) exists in this
  worktree to check for a third named class. If a third class exists in a
  report not present here, it was not exercised against by this run
  because no document diffed by this executor showed a third distinct
  difference shape.
