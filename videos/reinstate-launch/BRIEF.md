---
workflow: product-launch-video
flow: automation
storyboard: no
message: "Pick up any coding task exactly where you left it"
destination: embed
aspect: 1920x1080
language: en
length: 45s
angle: continuity
---

## Intent

Launch video for **Reinstate** — the continuity layer for coding-agent work.
The hero video for an X launch post and the first slide of a Product Hunt
launch. Audience: developers who live in Claude Code and Codex all day and have
lost a session to a new laptop, a new agent, or a dead context window.

Fast-paced and confident, closer to a terminal demo reel than a SaaS explainer.
No voice-over — kinetic typography over real terminal capture, cut on the beat.
The product is a CLI, so the CLI is the star: real commands, real output.

Full treatment (authoritative beat sheet):
`../../docs/marketing/launch-video-treatment.md`

## Customizations

- **No narration.** Kinetic type carries every line; BGM + SFX carry the pacing.
- **Every command on screen is real** — drawn from the actual cobra command tree
  (`init, list, show, status, push, pull, diff, conflicts, resolve, check,
  doctor`). Nothing invented, no faked output.
- **Path remapping gets the held hero frame** at ~0:24 — `/Users/…` morphs to
  `C:\Users\…` on a split-screen macOS↔Windows seam. This is the flagship
  multi-device proof, not a footnote.
- Two deliberate held frames: the path morph (~0:24) and the end card (~0:40).
  Everything else cuts every ~1.2s.
- End card carries `go install github.com/HarjjotSinghh/reinstate@latest`,
  `reinstate.dev`, and the Apache-2.0 badge.

## Notes

- **Do not claim cross-agent transcript translation.** Phase 1 is same-vendor
  Claude Code + Codex sync. The payoff card says *hand off*, not *translate*.
- **No real transcripts, repo names, absolute paths, or tokens on screen.** Use
  synthetic fixture data only; the repo's own `testdata/` is the reference for
  what realistic-but-fake looks like.
- Brand tokens come from the live site at https://reinstate.dev.
- A 9:16 recut follows later: keep the 0:20 split screen composable as a
  vertical stack so it survives the reframe.
