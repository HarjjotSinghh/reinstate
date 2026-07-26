# Two-section landing credibility pass

**Status:** approved
**Date:** 2026-07-26

## Goal

Make the current landing page feel immediately usable and technically credible
without redesigning its visual system or adding another section.

The page remains:

1. Hero and cross-device room illustration
2. Git problem scene and before/after comparison

The next section will be planned after this pass and will show a real Phase 1
terminal workflow.

## Product-truth boundary

The landing page must describe what Phase 1 does today:

- encrypted Claude Code and Codex session push/pull;
- macOS and Windows path remapping;
- user-owned S3/R2-compatible storage;
- local encryption before upload;
- native same-vendor resume after pull; and
- the short `rein` command as the CLI for Reinstate.

The page must not present later roadmap work as available:

- Gemini CLI or OpenCode adapters;
- universal session search;
- `rein sessions`, `rein search`, `rein last`, or `rein resume`;
- MCP, skills, settings, or environment restoration; or
- silent cross-agent transcript translation.

## Hero

Keep the existing isometric room, palette, and push/pull narrative.

### Copy

- Badge: `Capture. Encrypt. Reinstate.`
- Headline: `Pick up your coding session on another device.`
- Supporting copy: describe encrypted Claude Code and Codex session movement
  between machines, with data stored in the user's bucket.
- Availability: `Available now: Claude Code · Codex`
- Platform proof: `macOS ↔ Windows`
- CLI clarification: ``rein` is the CLI for Reinstate`
- Technical proof: `Apache-2.0 · one Go binary · your bucket, your keys`

### Conversion

Remove the hero waitlist form. Replace it with:

- a visible POSIX install command;
- a primary `Install Reinstate` link to getting started; and
- a secondary `View on GitHub` link.

The header CTA also becomes `Install Reinstate`.

### Mechanism

Add a compact three-step explanation inside the existing hero, not as a new
section:

1. **Capture** — select a local Claude Code or Codex session.
2. **Encrypt** — seal it locally before it reaches object storage.
3. **Reinstate** — pull it onto another machine and resume in the native agent.

The steps support the existing illustration rather than creating a competing
card grid.

### Readability

- Increase the header logo by about 12 percent.
- Increase navigation and supporting text modestly.
- Give the install command a clear copy affordance and keyboard-visible focus.
- Preserve the current display font, tracking, leading, and visual hierarchy.

## Problem and comparison

Keep the headline:

> Git has the code and its history.
> It does not have the conversation.

Replace the paragraph with an explicit boundary: Git carries code; Reinstate
carries encrypted session state and remaps project paths for the destination
machine.

### Without Reinstate

- Sessions trapped on one machine
- Paths and agent state drift apart
- Context must be rebuilt manually

### With Reinstate

- Pull sessions onto another device
- Remap paths across macOS and Windows
- Store encrypted state in your own bucket

Increase comparison headings and body text by roughly 15 percent. Improve text
contrast while preserving the recently approved translucent card background
and existing artwork.

## Accessibility and responsive behavior

- Keep light and dark themes at WCAG AA text contrast.
- Keep all actions reachable and understandable without relying on colour.
- The install command must be selectable even if clipboard access fails.
- Stack CTA and mechanism rows cleanly on narrow screens.
- Do not introduce horizontal overflow at 375 px.
- Honor existing reduced-motion behavior.

## Verification

- Run website tests.
- Run the Astro production build.
- Inspect hero and problem areas in light and dark themes.
- Inspect desktop and 375 px mobile layouts.
- Verify install and GitHub links.
- Verify clipboard success and graceful failure behavior.
- Confirm no later-phase product claims remain in the two landing sections.

## Deferred section three

Plan, but do not build, a real Phase 1 terminal demonstration using only current
commands:

`rein list` → `rein push` → `rein pull` → native `claude --resume` or
`codex resume`.
