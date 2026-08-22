---
format: 1920x1080
duration: 35.6s
message: "Pick up any coding task exactly where you left it"
arc: Cold open → Problem → Identity → Proof ×3 → Payoff → CTA
audience: developers living in Claude Code and Codex who have lost a session to a new laptop, a new agent, or a dead context window
mode: autonomous
music: driving minimal electronic — confident, building, terminal-demo energy; no vocals
---

## Video direction

**One camera, one film, one dark family.** The video never mixes light and dark
grounds — cutting from paper to navy strobes the viewer. Everything lives in a
dark range and shifts only subtly between two steps: the **terminal ground**
`#171815` for proof beats (Frames 1, 4, 5, 6) and the **statement ground**
`#1C1F1A`, one notch lifted, for claim beats (Frames 2, 3, 7, 8). Cuts between
them read as a change of room, not a flash. Accent `{colors.coral}` (acid lime
`#ACED37`) is scarce voltage — it appears only on the thing that just changed:
the error, the encrypted payload, the passing check, the CTA. On dark grounds
the accent is always the bright lime, never the deep `#386C12`.

**Type is centred and balanced on statement frames.** Display lines carry
explicit line breaks so each phrase reads as two even lines; no word is ever
orphaned onto a line of its own, and body paragraphs use `text-wrap: pretty`
rather than hard newlines.

**No voice-over.** Reveals are paced to the **beat grid**, not a spoken line —
cuts land every ~1.2s, and within a frame each Scene window opens on a beat. The
anti-front-loading rule still governs: at t=0 show only the first cue; spread the
rest across the shot, especially the back half.

**Two deliberate held frames** carry the rhythm: the path morph (Frame 5, Scene
4) and the end card (Frame 8). Everything else keeps moving. The stillness is
what makes the path morph land.

**Type discipline** — Questrial display for claims, Geist for chrome, Geist Mono
Variable for every command, path, session id, and status line. The mono voice is
the product; never substitute inside a terminal surface.

**Never on screen:** real transcripts, real repo names, real absolute paths,
tokens, browser chrome, scrollbars, or a cross-agent "translation" claim.

---

## Frame 1 — Cold open: no sessions found

- scene: A terminal types `claude --resume` and comes back empty
- duration: 2.6s
- poster: 2s
- transition_in: cut
- status: animated
- src: compositions/frames/01-cold-open.html
- type: hook
- persuasion: Pain recognition — the exact moment the viewer has lived
- beat: stall
- blueprint: typewriter-reveal (Adapt)
- focal: composed terminal surface (no captured asset)
- roles: terminal-surface = cutout · navy ground = background
- sfx: keystroke-cluster, error-thud
- asset_candidates: (composed) — this frame is composed type on the navy terminal surface

Adapt: keep the type-on-with-caret signature; the payoff is not the completed
line but the **empty result underneath it**. The caret keeps blinking after the
error — the stall is the point.

Scene 1 (0.0–0.5s): black. A single mono caret blinks dead-center-left on the navy ground. Nothing else. Centered, ~30% of frame, 2 depth layers (ground + caret).
Scene 2 (0.5–1.8s): `$ claude --resume` types on character-by-character behind the caret → `discrete-text-sequence` + `context-sensitive-cursor`. Left-aligned on a rule-of-thirds vertical, mono at reading scale.
Scene 3 (1.8–2.6s): a beat of nothing — the command sits, caret blinking, no output. The pause IS the beat; hold still.
Scene 4 (2.6–3.5s): `No sessions found.` hard-cuts in one line below in accent lime → `discrete-text-sequence`. A single low-frequency thud lands with it. The caret keeps blinking beneath. Held read, no camera move.

---

## Frame 2 — Git has the code. Not the conversation.

- scene: The thesis, slammed in two lines on paper
- duration: 3.5s
- poster: 2.9s
- transition_in: cut
- status: animated
- src: compositions/frames/02-the-gap.html
- type: problem
- persuasion: Reframe — name the gap the viewer never had words for
- beat: recognition
- blueprint: kinetic-type-beats (Reproduce)
- focal: the two-line statement
- roles: statement = cutout · paper ground = background
- sfx: impact-soft ×2, riser-short
- asset_candidates: (composed) — pure kinetic type on paper

Reproduce: the beat-slam array, two phrases resolving on a locked finale. This
is the site's own strongest line, so it gets the biggest display moment in the
video.

Scene 1 (0.0–0.4s): hard cut from navy to paper — the ground flip is the transition. Empty paper, one lime ✱ kicker `THE GAP` upper-left on the third → `discrete-text-sequence`.
Scene 2 (0.4–1.6s): `Git has the code and its history.` slams in per-line at display scale, ink on paper → `kinetic-beat-slam`. Left-aligned, occupying the upper 45% of frame. Sits alone — the second line is NOT on screen yet.
Scene 3 (1.6–3.0s): `It does not have the conversation.` slams in beneath on the next beat, same scale → `kinetic-beat-slam`. The word `conversation` carries the lime accent — the only colored word in the frame. Now both lines read as one block, ~65% of canvas.
Scene 4 (3.0–4.5s): locked finale — everything still. A hairline rule draws left-to-right beneath the block → `svg-path-draw`, then holds. No drift, no breathing.

---

## Frame 3 — Reinstate

- scene: Logo lockup with the product line
- duration: 2.3s
- poster: 1.8s
- transition_in: cut
- status: animated
- src: compositions/frames/03-identity.html
- type: identity
- persuasion: Name the answer immediately after naming the gap
- beat: arrival
- blueprint: titlecard-reveal (Adapt)
- focal: assets/favicon.svg
- roles: favicon = cutout · wordmark = supporting · paper ground = background
- sfx: impact-hard, whoosh-short
- asset_candidates: capture/assets/favicon.svg — the Reinstate mark

Adapt: keep the lockup-resolves signature, but the mark arrives with overshoot
on the downbeat rather than fading — this is the loudest cut in the video.

Scene 1 (0.0–0.6s): the mark scales in from ~0.7 with overshoot, dead-center on paper → `spring-pop-entrance`. Nothing else on screen. Centered, ~35% of frame.
Scene 2 (0.6–1.4s): the wordmark `Reinstate` slides up from behind the mark and settles beside it, forming the horizontal lockup → `waterfall-entry`. The pair recenters as one unit.
Scene 3 (1.4–3.0s): `Capture. Encrypt. Reinstate.` reveals word-by-word beneath in mono uppercase, tracked wide → `dynamic-content-sequencing`, each word on its own beat. The third word lands in lime. Held read to the cut.

---

## Frame 4 — Every session, indexed

- scene: `rein init` then `rein list` — the local session index fills with real rows
- duration: 6s
- poster: 5s
- transition_in: cut
- status: animated
- src: compositions/frames/04-index.html
- type: proof
- persuasion: Demonstration — the index exists locally, before any cloud
- beat: relief
- blueprint: compose
- focal: composed terminal surface
- roles: terminal-surface = cutout · navy ground = background · row-stack = supporting
- sfx: keystroke-cluster ×2, row-tick ×4, ui-confirm
- asset_candidates: (composed) — composed terminal, mono type only

Compose: no blueprint owns "a table fills with real rows." Built from type-on
with caret + waterfall reveal + a settling hold. Reveals stay paced across the
full 8s — the rows arrive one per beat, not as a block.

Scene 1 (0.0–1.0s): hard cut to navy. `$ rein init` types on behind the caret, upper-left on the third → `discrete-text-sequence` + `context-sensitive-cursor`. Terminal surface fills ~80% of canvas, hairline lime border, 3 depth layers.
Scene 2 (1.0–2.0s): `✓ indexed 4 sessions · 2 agents` returns in lime beneath → `discrete-text-sequence`. Brief hold.
Scene 3 (2.0–3.0s): `$ rein list` types on below the previous output, same caret.
Scene 4 (3.0–5.4s): the column header sets, then four session rows waterfall in bottom-up, one per beat → `waterfall-entry`. Each row is mono: `ses_7f3a  claude-code  payments-api  2h ago  ⏸ paused` / `ses_2c81  codex  payments-api  5h ago  ✓ done` / `ses_9d14  claude-code  auth-service  1d ago  ⏸ paused` / `ses_4b67  codex  infra-terraform  2d ago  ✓ done`. Agent column in lime.
Scene 5 (5.4–6.6s): the first row (`ses_7f3a`, paused) receives a soft keyword glow and a lime left-edge marker — the eye is told which session the rest of the video is about → `asr-keyword-glow`.
Scene 6 (6.6–8.0s): still. A mono chip `LOCAL INDEX · NO CLOUD REQUIRED` fades up bottom-right and holds. No camera move.

---

## Frame 5 — macOS → Windows, paths and all

- scene: Split screen; `rein push` seals the session, `rein pull` lands it, the path rewrites itself
- duration: 7.4s
- poster: 6.6s
- transition_in: cut
- status: animated
- src: compositions/frames/05-sync.html
- type: proof
- persuasion: The flagship multi-device claim, shown not stated
- beat: transfer
- blueprint: comparison-split (Adapt)
- focal: composed dual-terminal surface
- roles: left-panel = cutout · right-panel = cutout · seam = supporting · navy ground = background
- sfx: whoosh-transfer, lock-seal, ui-confirm, impact-soft
- asset_candidates: (composed) — two composed terminal panels

Adapt: keep the mirrored split-tilt entry as the signature — the two machines
arrive from opposite wings tilting toward each other. What changes: the cards are
live terminal panels rather than feature cards, and the shot's climax is the
**path morph on the right panel**, held still.

Scene 1 (0.0–1.2s): two navy terminal panels enter from opposite wings with mirrored `rotateY` tilts, opening like a book, and settle into a 50/50 split → `split-tilt-cards`. Mono chips label them `macOS` (left) and `Windows` (right). A hairline lime seam runs the vertical center.
Scene 2 (1.2–2.6s): left panel only. `$ rein push` types on → `discrete-text-sequence`. Right panel sits dim (~45%) — attention is single-pointed.
Scene 3 (2.6–4.2s): the payload seals — four stacked mono labels `paths · metadata · messages · session` collapse into one lime capsule marked `ses_7f3a` on the left panel → `center-outward-expansion` run inward. A lock glyph draws itself closed over the capsule → `svg-path-draw`.
Scene 4 (4.2–5.6s): the capsule arcs across the seam left → right on a long curve, motion-streaking as it travels → `motion-blur-streak` + `nudge-curve`. The seam pulses lime as it crosses. Right panel brightens to full as it lands.
Scene 5 (5.6–7.0s): right panel. `$ rein pull` types on and returns `✓ ses_7f3a restored`. The left panel dims to ~45% — focus has fully handed over.
Scene 6 (7.0–9.0s): **the held hero beat.** A single mono path line sits center-right at large scale: `/Users/dev/code/payments-api`. Character-by-character it rewrites in place to `C:\Users\dev\code\payments-api` → `hacker-flip-3d`, the changed segments landing in lime. Then everything stops. Full stillness for the last ~0.8s — no drift, no push. A mono chip reads `PATHS REMAPPED`.

---

## Frame 6 — Verified, then resumed

- scene: `rein status` stamps three green checks, and the agent picks the task back up mid-sentence
- duration: 5.6s
- poster: 5s
- transition_in: cut
- status: animated
- src: compositions/frames/06-verify.html
- type: proof
- persuasion: Trust — resume is verified, not hopeful
- beat: confidence
- blueprint: agent-progress-theater (Adapt)
- focal: composed terminal surface
- roles: check-stack = cutout · agent-stream = supporting · navy ground = background
- sfx: check-tick ×3, ui-confirm, type-stream
- asset_candidates: (composed) — composed terminal, mono type only

Adapt: keep the progress-theater signature — discrete work items resolving one
by one into a completed state. What changes: the items are integrity checks, and
the theater resolves into a live agent stream rather than a summary card.

Scene 1 (0.0–1.0s): hard cut. Single-panel navy terminal, full bleed. `$ rein status ses_7f3a` types on → `discrete-text-sequence`.
Scene 2 (1.0–3.4s): three check rows stamp in one per beat, each an SVG tick drawing itself then locking lime → `svg-path-draw` + `waterfall-entry`: `✓ integrity` / `✓ paths` / `✓ agent`. Left-aligned stack, ~45% of canvas.
Scene 3 (3.4–4.4s): the three checks condense upward into one lime chip `VERIFIED` → `scale-swap-transition`. The stack's space clears for what comes next.
Scene 4 (4.4–7.0s): beneath it, the resumed agent stream types on — a mono assistant line continuing mid-task, cursor streaming → `discrete-text-sequence` + `context-sensitive-cursor`. It reads as a session picked up mid-thought, not restarted. Camera holds; the streaming text is the only motion.

---

## Frame 7 — Search. Resume. Hand off.

- scene: Three capability cards assemble, agent names beneath
- duration: 4s
- poster: 3.2s
- transition_in: cut
- status: animated
- src: compositions/frames/07-payoff.html
- type: payoff
- persuasion: Compress the proof into three retainable words
- beat: resolution
- blueprint: grid-card-assemble (Reproduce)
- focal: the three-card row
- roles: cards = cutout · agent-wordmarks = supporting · paper ground = background
- sfx: whoosh-short ×3, impact-soft
- asset_candidates: (composed) — composed cards on paper

Reproduce: the staggered assemble into a locked grid. Three cards, one per beat.

Scene 1 (0.0–0.5s): hard cut back to paper — the ground flip signals the argument is over and the summary has begun. Empty, with a lime ✱ kicker `WHAT YOU GET` on the upper third.
Scene 2 (0.5–2.6s): three cards fly in from below and lock into an even row, one per beat → `waterfall-entry` + `grid-card-assemble`. Each is hairline-bordered paper with a Questrial title and a mono sub-line: **Search** `every session, locally` · **Resume** `verified, same-vendor` · **Hand off** `explicit and portable`. Row occupies ~55% of canvas.
Scene 3 (2.6–3.8s): beneath the row, `Claude Code` and `Codex` fade up as mono wordmarks flanking a lime `·`, with a small mono note `same-vendor native resume` → `dynamic-content-sequencing`.
Scene 4 (3.8–5.0s): still. Hairline rule draws under the whole block and holds.

---

## Frame 8 — Install it

- scene: End card — mark, install command, domain, license
- duration: 4.2s
- poster: 3.4s
- transition_in: cut
- status: animated
- src: compositions/frames/08-cta.html
- type: cta
- persuasion: One command, zero friction, verifiable license
- beat: invitation
- blueprint: cta-morph-press (Adapt)
- focal: assets/favicon.svg
- roles: favicon = cutout · install-command = supporting · paper ground = background
- sfx: impact-hard, ui-confirm
- asset_candidates: capture/assets/favicon.svg — the Reinstate mark

Adapt: keep the shared-center morph signature — the mark condenses into the
install command at one transform origin, so it reads as the product becoming the
command. This is the video's second held frame.

Scene 1 (0.0–0.8s): the lockup from Frame 3 returns dead-center on paper, arriving with a single hard impact → `spring-pop-entrance`. Centered, ~35% of frame.
Scene 2 (0.8–1.8s): the lockup condenses upward at the same screen center as the install command scales up into its place beneath → `card-morph-anchor`. Mono, on a hairline-bordered lime-tinted chip: `go install github.com/HarjjotSinghh/reinstate@latest`.
Scene 3 (1.8–2.8s): `reinstate.dev` reveals beneath in Geist, with `Apache-2.0 · one Go binary · your bucket, your keys` as a small mono line under it → `dynamic-content-sequencing`.
Scene 4 (2.8–5.0s): **fully held.** Nothing moves except a slow lime underline drawing across `reinstate.dev` → `svg-path-draw`, resolving at ~4.2s. Then complete stillness to the end of the video.
