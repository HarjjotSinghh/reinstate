# Hero Command and Floating Navbar Design

**Date:** 2026-07-26  
**Status:** Approved  
**Scope:** Existing hero and navbar only. No new landing-page sections.

## Goal

Make the first viewport read as one deliberate product story: a floating Reinstate
navigation shell, a precise cross-device promise, one obvious install action, and a
three-step mechanism panel that remains legible when the illustration rises behind it.

## Approved copy

### Headline

> Switch devices without losing the thread.

“Thread” is intentionally preferred over “context.” It is idiomatic, memorable, and
connects the hero promise to the later statement about Git not carrying the conversation.

### Supporting copy

> Move encrypted Claude Code or Codex sessions between macOS and Windows. Reinstate
> encrypts the session locally, then stores it in your own S3 or R2 bucket.

The copy stays inside Phase 1: same-vendor Claude Code and Codex session movement across
the user's own devices. It does not claim cross-agent native resume, MCP/config sync, or
future indexing features.

## Information hierarchy

The install command is the primary CTA. The separate “Install Reinstate” button is
removed because it duplicates the command and fractures the action hierarchy.

The hero action stack becomes:

1. Install command with icon-only copy control
2. Compact supporting links: “Read setup guide” and “View on GitHub”
3. Current compatibility proof
4. Bordered three-step process panel
5. Existing CLI/open-source micro-proof

Related elements stay tightly grouped; the mechanism panel receives enough separation
to read as explanation rather than another CTA.

## Install command

- Keep the dark command surface and terminal prompt.
- Replace visible “Copy” text with an overlapping-pages icon in a square trailing control.
- Preserve `aria-label="Copy install command"`.
- On success, swap the copy icon for a checkmark for 1.8 seconds.
- On clipboard failure, select the command text and announce the fallback through a
  visually hidden live region. The button width must never change.
- Maintain a minimum 44-by-44-pixel hit target and visible keyboard focus.

## Supporting links

- Remove the two large CTA buttons.
- Add two understated inline links below the command:
  - “Read setup guide” points to `/docs/getting-started`.
  - “View on GitHub” points to the repository and retains external-link semantics.
- Use standalone labels, small directional icons, and no filled button treatment.
- Stack or wrap cleanly on narrow screens.

## Three-step mechanism panel

- Wrap the existing ordered list in one solid `--paper-2` surface.
- Add a full `1px solid var(--rule)` border with a `12px` radius.
- Remove the current exposed top/bottom-only rail treatment.
- Retain internal vertical separators on wide screens.
- Add a compact outlined icon tile to each genuine sequence step:
  - Capture: a session/document entering a container
  - Encrypt: a lock
  - Reinstate: a return arrow entering a device
- Keep the sequence numbers visible as secondary mono markers.
- Stack the steps vertically below the existing responsive breakpoint, replacing vertical
  dividers with horizontal ones.
- The solid background must prevent illustration lines or art from visually colliding
  with the explanatory content.

## Floating navbar

- Keep the header sticky, but make the outer header transparent and borderless.
- Place the existing navigation inside a centered floating shell:
  - top offset: `12px` to `16px`
  - maximum width aligned with the existing `88rem` page container
  - solid `--paper-2` background
  - `1px solid var(--rule)` border
  - `12px` to `16px` radius
  - no blur, glass, gradient, or soft drop shadow
- Preserve the brand lockup, current links, theme toggle, and install CTA.
- Increase page/hero top clearance so the floating shell never overlaps headline content.
- On mobile, keep the same floating shell. The menu expands as a bordered solid panel
  directly beneath it rather than restoring a full-width bar.

## Visual-system constraints

- Preserve Reinstate's Questrial display, Geist body, and Geist Mono command typography.
- Preserve the green-neutral paper, dark green, and chartreuse palette.
- Borders provide separation. Do not add blurred shadows or glassmorphism.
- Rectangular panels stay at or below a `16px` radius.
- Both light and dark themes must remain first-class.
- Existing isometric art remains unchanged except for spacing needed to prevent overlap.

## Responsive and accessibility requirements

- No page-level horizontal overflow at 375px.
- Headline must balance without clipping from 320px through large desktop widths.
- Copy and navigation controls retain 44px minimum hit targets.
- All icon-only controls retain accessible names.
- Focus-visible treatment must work in both themes.
- Reduced-motion behavior remains intact.
- Verify at 375px, 768px, 1440px, and 2048px in light and dark themes.

## Acceptance criteria

- The headline reads “Switch devices without losing the thread.”
- The command is the single dominant install CTA.
- No visible “Copy” label remains.
- The copy icon gives success and accessible fallback feedback without layout shift.
- Setup guide and GitHub remain discoverable without competing with the command.
- The mechanism panel has full rounded borders, icons, separators, and a solid background.
- The navbar floats inside a rounded bordered shell without colliding with hero content.
- Existing landing sections and illustration narrative remain intact.
- Website tests and production build pass.
