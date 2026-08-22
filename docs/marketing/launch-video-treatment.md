# Reinstate — launch video treatment (45s)

Format: 1920×1080, 30fps (plus 9:16 recut). No VO — kinetic type + real
terminal capture + SFX. Target: X launch post / Product Hunt hero slide.

Build: HyperFrames (`/product-launch-video` → `/hyperframes`).
Motion rules: `slow-in-out-mastery`, `anticipation-mastery`, `rhythm-pacing`,
`attention-direction`.

Palette: repo brand tokens from `website/`. Type: one geometric sans, two weights.
Cuts land on beat every ~1.2s; two deliberate held frames (0:12, 0:38).

---

## Beat sheet

| Time | Beat | On screen | Audio |
|---|---|---|---|
| 0:00–0:03 | **Cold open** | Black. Cursor blinks. Type-on: `$ claude --resume` → error flash `no sessions found`. Hard cut to white. | Keystroke, low sub-drop |
| 0:03–0:08 | **Problem** | Kinetic type, one line per cut: "New laptop." / "New agent." / "Context gone." Behind them, a faint session list scrolls up and evaporates. | 4-on-floor enters |
| 0:08–0:11 | **Title card** | Logo lockup snaps in with overshoot. Subline: *The continuity layer for coding-agent work.* | Impact hit |
| 0:11–0:20 | **Demo 1 — index & search** | Real terminal: `rein init` → `rein list`. Rows of Claude Code + Codex sessions populate with a 40ms stagger. Callout pill: **local session index**. | Typing, row ticks |
| 0:20–0:28 | **Demo 2 — sync** | Split screen, macOS left / Windows right. Left: `rein push`. Encrypted-blob glyph arcs across the seam. Right: `rein pull` → same session appears. Path `/Users/…` morphs to `C:\Users\…` on a held frame. Pill: **E2E encrypted · BYO storage**. | Whoosh, two-note resolve |
| 0:28–0:35 | **Demo 3 — verified resume** | Right pane: `rein show <id>` → `rein status` → green ✓ checks stamp in one by one (`integrity`, `paths`, `agent`). Then the agent resumes mid-task, tokens streaming. Pill: **verified resume** | Rising ticks, snare |
| 0:35–0:40 | **Payoff** | Three cards fly into a row: *Search* · *Resume* · *Hand off*. Below, agent wordmarks (Claude Code, Codex) fade up. | Music peak |
| 0:40–0:45 | **CTA** | Held frame. Logo + `go install github.com/HarjjotSinghh/reinstate@latest` + `reinstate.dev` + Apache-2.0 badge. | Tail out |

---

## Rules

- **Every command is real.** Use `init, list, show, status, push, pull, diff,
  conflicts, resolve, check, doctor` — nothing invented. Capture actual output,
  then speed-ramp; never fake a frame.
- **Scrub the capture.** No real transcripts, repo names, paths, or tokens.
  Use `testdata/` fixtures and a synthetic project name.
- **Path remapping gets the hero frame** (0:24) — it is the flagship
  multi-device proof, not a footnote.
- **Don't claim cross-agent transcript translation.** Phase 1 is same-vendor
  sync; the third card says *hand off*, not *translate*.
- 9:16 recut: drop the split screen at 0:20 to a vertical stack; everything else
  survives.

## First render

```
/product-launch-video   # brief: this file + reinstate.dev
```
