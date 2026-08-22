# v0.5.2 — CLI experience revamp ("continuity you can see")

Turn `rein` from a flag-driven, ID-typing, copy-paste tool into an interactive
TUI that is fast, legible, and pleasant — without breaking a single script.

**Status:** M0-M5 implemented 2026-08-22. Verified on macOS (Apple Silicon) and
native Windows x64.
**Target release:** `v0.5.2`, full scope, all six milestones in one release.
**Baseline:** `v0.5.1` is shipped; this branches from it.
**Track:** runs alongside Phase 6 (universal configuration); does not block it.
**Framework:** Bubble Tea — decided, see §5.
**Tone:** restrained and polished — decided, see §4.
**Owner:** TBD.

---

## 1. Why

Reinstate's engine is strong. Its surface is not. Concretely, today:

| Friction | Evidence |
| --- | --- |
| The "picker" is a numbered re-print loop | `runSessionPicker` in [sessions.go:740](../../../internal/cli/sessions.go) — you type `3`, `/text`, `i 3`, `f 3`, `h 3`, `q`. No arrows, no scroll, no preview; the whole list reprints every round-trip. |
| Session references are UUIDs | `rein inspect claude:0a1b…`, `rein resume codex:01HX…`. Nothing to do but copy-paste. |
| Warning acknowledgement is retyping | `--allow-environment-warning CHECK_ID` and `--allow-warning ID` require reading an ID off screen and typing it back, exactly, once per warning. |
| Readiness is invisible until you commit | The preflight report already exists and is excellent. You only see it *after* choosing a session and attempting a launch. |
| `rein init` is a blocking prompt chain | [commands_impl.go:81](../../../internal/cli/commands_impl.go) — endpoint, bucket, hidden keys, then a probe that can fail and discard everything typed. No presets, no back, no retry-in-place. |
| `rein handoff` has 13 flags | `--policy checkpoint\|balanced\|full` is an invisible trade-off. You cannot see what each policy produces before committing. |
| Output has no hierarchy | `PrintHuman` is `fmt.Fprintf` + `\n`. `writeLocalSessions` emits `key\tproject\tbranch\ttitle`. `doctor` emits `Environment check: … status=… severity=… provenance=…` walls. |
| Second-device setup is manual courier work | Carry a `profile_id` UUID, a passphrase, and two S3 keys between machines by hand. |

The product strategy already names **"Reinstate CLI / TUI (primary)"** as surface
#2 ([product-strategy.md](../../product-strategy.md)). This is that surface.

### The unfair advantage

Reinstate is the only tool that knows a session's **readiness** (preflight) and
its **lineage** (forks and cross-agent handoffs). No competitor can draw those.
Both are currently buried in JSON. Surfacing them is the whole differentiator —
not decoration.

---

## 2. The five invariants

Break any of these and the work is wrong regardless of whether it compiles.

1. **`--json` and non-TTY output are frozen.** Byte-stable. The TUI is strictly
   additive on a TTY. Every existing e2e/golden test passes unchanged.
2. **The TUI is a view, never an engine.** `internal/tui/*` imports
   `sessionindex`, `preflight`, `handoff`, `sync`. Those packages never import
   `tui`. No business logic, no vendor knowledge, no parsing in the view layer.
3. **Every interactive action prints its scriptable equivalent.** The TUI
   *teaches* the flag form instead of hiding it. This is how we stay honest and
   how power users graduate.
4. **Security posture is unchanged.** Bounded 160-code-point prompt previews
   only. No assistant messages, reasoning, tool output, env dumps, or
   credentials on screen. Path redaction stays on. Read-only below T5.
5. **Native resume stays same-vendor.** The TUI must never imply that a handoff
   is a resume. Different verb, different colour, different confirmation.

---

## 3. Design

### 3.1 The switcher — bare `rein`

The flagship. Replaces the numbered loop entirely.

```
┌ rein ──────────────────────────────────────── 143 sessions · 6 agents · 4m ago ┐
│ ❯ auth▏                                       [a] all projects      [?] help   │
├──────────────────────────────────────┬─────────────────────────────────────────┤
│ TODAY                                │  claude · reinstate                     │
│ ● claude  reinstate   fix auth …     │  feat/auth-refactor · 4m ago            │
│ ◐ codex   reinstate   wire keyring   │                                         │
│ ● gemini  website     og image sizes │  READY TO RESUME                        │
│                                      │   ✔ agent installed        2.1.220      │
│ YESTERDAY                            │   ✔ workspace clean                     │
│ ○ grok    reinstate   probe kimi …   │   ✔ branch matches                      │
│ ● claude  wlibrary    checkout bug   │                                         │
│                                      │  ⤷ forked from claude:0a1b… (2d)        │
│                                      │  "let's fix the auth refactor so…"      │
├──────────────────────────────────────┴─────────────────────────────────────────┤
│ ↵ resume   f fork   h hand off   i inspect   y copy ref   ^k palette   q quit   │
└────────────────────────────────────────────────────────────────────────────────┘
```

Behaviour:

- **Type to filter.** No `/` prefix. Filter is literal + case-insensitive,
  matching today's `search` semantics exactly (same code path).
- **↑↓ / j k** move; **PgUp/PgDn** page; **g/G** jump. Cursor and filter persist
  per project in state.
- **Readiness glyph per row**, computed lazily in the background:
  `◌` checking → `●` ready / `◐` warnings / `○` blocked or read-only.
  This is the feature. You see resumability *before* you commit.
- **Time-grouped sections.** Today / Yesterday / This week / Older. Humans
  navigate by time, not by UUID.
- **Agent identity.** Stable colour + short label per agent, honouring
  `NO_COLOR`.
- **Project-scoped by default** when run inside a known project root; `a`
  toggles to all projects. This alone removes most filtering.
- **Instant first frame.** Draw from the cached index immediately, refresh in
  the background, stream rows in. Today `openRefreshedLocalIndex` blocks before
  the first line is printed.

### 3.2 Warnings become a checklist

Kills the `--allow-environment-warning` retyping problem.

```
 codex:01HX…  ·  2 environment warnings

  [x] workspace.dirty          3 modified files
  [ ] agent.version.untested   2.1.238 vs verified 2.1.219–2.1.229
      ↳ repair: install a verified codex build, or acknowledge to proceed

  ↵ resume, acknowledging what is checked      esc cancel      c copy command

  rein resume codex:01HX… --allow-environment-warning workspace.dirty
```

Space toggles. The command line at the bottom is live and copyable (`c`, or
`y` for OSC 52 so it works over SSH and inside tmux). Same `preflight.Authorize`
call underneath — no new authorisation logic, no new bypass.

### 3.3 Handoff studio

Makes the invisible policy trade-off visible.

```
 hand off   claude:0a1b…   →   ◂  codex  ▸

 policy  ◂ balanced ▸            capsule 14.2 KB · 38 events
 ┌ included ─────────────────────────────────────────┐
 │ ✔ task boundary        ✔ open questions           │
 │ ✔ recent user turns    ✔ touched files (12)       │
 │ ✖ tool output          ✖ assistant reasoning      │
 └───────────────────────────────────────────────────┘
 redactions   3 categories · 5 values hidden          [r] categories

 ↵ send    e export    c copy command    esc cancel
```

←/→ cycles policy; the panel recomputes live from the existing dry-run
projection. `e` exports. The confirmation copy states plainly that this starts a
**new** session and is not native resume.

### 3.4 Continuity trail

In inspect, and as a one-line breadcrumb in the preview pane:

```
 gemini:f00d…  ⤷ handoff ⤷  claude:0a1b…  ⤷ fork ⤷  claude:9c2e…  ← you are here
```

Built from existing handoff records. Unique to Reinstate. `←`/`→` walks it.

### 3.5 `rein init` wizard

Steps, each reversible, none discarding typed state:

1. **Provider preset** — Cloudflare R2 · AWS S3 · Backblaze B2 · MinIO · other.
   R2 pre-fills the endpoint template with an `<account-id>` slot.
2. **Coordinates** — bucket, region, prefix, validated per keystroke.
3. **Credentials** — hidden input, unchanged `crypto.ReadHiddenSecret` path.
4. **Probe** — spinner, real steps, **retry in place** on failure. Today a
   failed probe throws the whole session away.
5. **Passphrase policy** — explain once, clearly, that it is never stored.
6. **Project map** — offer detected repos instead of demanding
   `--project ID=/abs/path` strings.
7. **Done card** — profile ID, credential ref, and exactly three next commands.

**Device pairing.** `rein init --link` prints a short base32 pairing block
carrying **only non-secret coordinates** (endpoint, bucket, prefix, profile ID).
`rein init --paste` on device two consumes it. Secrets are still typed by hand,
deliberately. QR rendering is a stretch goal (extra dependency).

### 3.6 Command palette

`^k` anywhere: fuzzy over every rein verb, with arguments pre-bound to the
selected session. One interaction model for the whole tool.

### 3.7 Live views

`doctor`, `status`, `push`, `pull` render as checklists that tick off as work
completes, with inline repair actions where `check.Repair` is non-empty. Same
data, same exit codes, same `--json`.

---

## 4. Fun, calibrated

Delight that is *information*, not decoration:

- Readiness dots resolving `◌ → ●` in place as checks land.
- The continuity trail — nothing else in this category can draw it.
- Warm empty state naming which agents were scanned and what to do next.
- Honest header stat: `143 sessions · 6 agents · 4m ago`.
- Clean transition line into the vendor TUI (`→ handing off to codex`) instead
  of a hard cut.

Explicitly **not** doing: gratuitous animation, ASCII art banners on every run,
progress bars for instant work, gamification, streak counters, easter eggs that
fire without being asked. The feel-good factor comes from *fast and legible*.

---

## 5. Architecture

```
internal/ui/            capability detection, theme, glyphs, width, plain fallback
internal/tui/
  switcher/             session switcher
  readiness/            warning checklist
  handoff/              handoff studio
  wizard/               init wizard
  palette/              command palette
  components/           list, preview, statusbar, spinner, trail
```

`internal/cli/*` decides TTY vs plain and dispatches. All rendering decisions
live in `internal/ui`; all interaction in `internal/tui`.

### Framework: Bubble Tea (decided)

`charmbracelet/bubbletea` + `lipgloss` + `bubbles` (MIT). Approved 2026-08-22.

Rationale: solid Windows console support — which matters, because Windows ↔
macOS is the flagship multi-device case; a mature test harness (`teatest`) for
deterministic golden frames; and a supply-chain footprint far smaller than the
AWS SDK and `modernc.org/sqlite` already vendored.

Alternatives considered: `tview`/`tcell` (fewer deps, dated feel, weaker test
story); hand-rolled on `golang.org/x/term`, already a dependency (zero new deps,
but means writing a renderer, an input decoder, resize handling, and Windows VT
support — weeks of work that is not the product).

Pin exact versions in `go.mod` at M0 and let Dependabot track them, as with
every other dependency. Record the choice as an ADR before the first
`internal/tui` commit.

### Degradation ladder

Checked in order; first match wins:

1. `--json` → plain, always.
2. Not a TTY (either stream) → plain, always.
3. `REINSTATE_NO_TUI=1`, `--plain`, `TERM=dumb`, or `CI=true` → plain.
4. `NO_COLOR` → TUI, no colour.
5. Width < 60 → single-pane TUI, no preview.
6. Otherwise → full TUI.

Plain mode is the **current** output, byte for byte.

---

## 6. Milestones

All six land in `v0.5.2`. Each is independently reviewable and merges behind
`REINSTATE_TUI=1` until its acceptance row passes on macOS **and** native
Windows; the flag is removed and the TUI becomes the TTY default once M1, M2
and M5 are all green.

All six milestones are implemented. What each one actually shipped:

| # | Milestone | State | Notes |
| - | --- | --- | --- |
| M0 | Foundation | done | `internal/ui` (capability ladder, theme, glyphs, width-aware text), `internal/tui` runtime, `tuitest` golden-frame harness |
| M1 | Switcher | done | `internal/tui/switcher`; replaces the numbered loop; type-to-filter with actions behind `tab` so no key is ambiguous |
| M2 | Readiness + checklist | done | `internal/tui/readiness`; background probing scoped to visible rows; acknowledgement verified against the real `preflight.Authorize` |
| M3 | Handoff studio | done | `internal/tui/handoffui`; live dry-run previews per destination and policy |
| M4 | init wizard + pairing | done | `internal/tui/wizard`, `internal/pairing`; QR remains the stretch goal it always was |
| M5 | Palette + polish | done | `internal/tui/palette`; subsequence matching; OSC 52 copy |

Original estimates, kept for reference:

| # | Milestone | Ships | Est. |
| - | --- | --- | --- |
| **M0** | Foundation — `internal/ui` capability/theme layer, `internal/tui` runtime skeleton, degradation ladder, golden-frame harness | No visible change beyond aligned tables | 3–4 d |
| **M1** | **The switcher** — replaces `runSessionPicker`; filter, nav, preview, time groups, instant first frame | The flagship | 5–7 d |
| **M2** | **Readiness + warning checklist** — background preflight, glyphs, toggle-to-acknowledge, live command line | Kills the retyping | 4–5 d |
| **M3** | **Handoff studio** — destination picker, live policy preview, redaction summary, trail | Makes the trade-off visible | 4–5 d |
| **M4** | **`init` wizard + pairing** — presets, retry-in-place, `--link`/`--paste` | Fixes onboarding | 4–5 d |
| **M5** | **Live views + palette + polish** — doctor/status/push/pull, `^k`, OSC 52, themes, empty states | Coherence | 4–6 d |

Roughly **five to six weeks** of focused work to `v0.5.2`.

M1 and M2 carry the majority of the felt improvement, so land them first and
dogfood them behind the flag while M3-M5 are built. If the release must be cut
short, M3 and M4 are the deferrable pair — never M5, which is what makes the
whole surface feel coherent rather than half-converted.

---

## 7. Testing and acceptance

- **Golden frames** via `teatest` at 80×24, 120×40, 200×60, with a fixed clock
  and the existing deterministic `testdata/` fixtures.
- **Frozen-output guard:** the full existing `internal/cli` e2e suite runs
  unmodified in CI. A diff in plain or JSON output fails the build.
- **Degradation matrix:** `NO_COLOR`, `TERM=dumb`, `CI=true`, `--plain`,
  `REINSTATE_NO_TUI=1`, piped stdin, piped stdout, width 40.
- **`REINSTATE_TUI_SCRIPT`** — a deterministic keystroke-sequence driver, so
  physical acceptance on Windows and macOS produces committed evidence rather
  than screenshots.
- **New acceptance rows** replacing today's "interactive picker via PTY; `q`
  exits" row, plus a retained legacy-plain row.

---

## 8. Success criteria

| Metric | Today | Target |
| --- | --- | --- |
| Keystrokes to resume the most recent session | read list, type number, Enter | `rein`, Enter |
| Keystrokes to resume an arbitrary session | read list, type ref or `i N` then number | `rein`, type 3–4 chars, Enter |
| Acknowledging 2 warnings | read and retype 2 exact IDs | 2 spacebars |
| `rein` → first frame, warm index | blocks on refresh | < 120 ms |
| Readiness visible before commit | no | yes |
| `--json` / non-TTY regressions | — | zero |

---

## 9. Risks

| Risk | Mitigation |
| --- | --- |
| Terminal compatibility (conhost, tmux, SSH, CI) | Degradation ladder + `--plain` + golden frames at fixed widths + physical Windows acceptance |
| Scope creep toward an ADE | Invariant 2. The TUI never runs the agent loop; it always delegates. Non-goals in [product-strategy.md](../../product-strategy.md) stand |
| Silent divergence between TUI and JSON | Invariant 3 — every screen prints its command; both read one engine call |
| New dependency surface | One decision, one review, pinned + Dependabot as with every other dep |
| Derailing Phase 6 | Separate branch, separate milestones, independently shippable |

---

## 9a. What the build changed about the plan

Three decisions were made during implementation that the plan did not anticipate.

**Actions moved behind `tab`.** The plan showed bare letters (`f` fork, `h` hand
off) alongside type-to-filter. Those cannot coexist: a key cannot mean "filter"
and "fork" at the same moment. Letters always filter; `tab` opens an action menu
where letters are accelerators, and `ctrl+k` opens the palette. A test asserts
that `f` in list mode filters and does not fork.

**The wizard collects no secrets.** A Bubble Tea text input holds its value in an
immutable Go string, which cannot be zeroed and may be copied by the runtime.
The existing prompt reads secrets into a `[]byte` wiped with `crypto.Zero`.
Rather than trade that away for a nicer prompt, the wizard collects coordinates,
exits, and the caller reads credentials through the unchanged hardened path.

**No native clipboard, and no `bubbles/textinput`.** Copying goes through OSC 52
so the terminal the human is looking at performs it, which is the only thing
that works over SSH. `bubbles/textinput` depends on a host-side clipboard
binding, so the wizard uses a hand-written field instead.

## 10. Decisions

Settled 2026-08-22:

| Decision | Outcome |
| --- | --- |
| TUI framework | **Bubble Tea** (`bubbletea` + `lipgloss` + `bubbles`, MIT) |
| Release shape | **Full scope in `v0.5.2`**, branching from shipped `v0.5.1` |
| Tone | **Restrained and polished** — delight from speed and legibility, per §4 |

Still open, resolvable during M0 and not blocking:

1. **Owner and reviewer** for the track.
2. **QR pairing** in `rein init --link` (§3.5) — stretch goal; needs a
   pure-Go QR dependency. Base32 pairing block ships regardless.
3. **Theme surface** — whether `--theme` is user-configurable in `v0.5.2` or
   ships with one well-tuned default plus `NO_COLOR`.
