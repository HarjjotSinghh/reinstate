# Hero Hierarchy and Navbar Scroll Design

**Date:** 2026-07-26
**Status:** Approved
**Scope:** Existing hero hierarchy and navbar behavior only. The hero headline is explicitly excluded.

## Goal

Reduce the number of equally weighted horizontal bands in the hero and make the floating
navbar feel integrated with the opening scene rather than permanently boxed.

## Explicit exclusion

The current headline remains:

> Switch devices without losing the thread.

No headline copy, line break, width, tracking, leading, or font-size change is part of
this implementation. A replacement headline will be chosen separately.

## Hero hierarchy

The hero becomes three clear groups:

1. Narrative: badge, headline, supporting paragraph
2. Action: install command plus one compact metadata row
3. Explanation: the narrower Capture, Encrypt, Reinstate process rail

The command remains the dominant action.

The existing resource links and compatibility proof move into one shared metadata row
directly beneath the command:

- Left: Read setup guide, View on GitHub
- Right: available agents and operating systems

This removes one visual tier and makes the command, resources, and proof read as one
conversion cluster. On narrow screens the row stacks without changing the information
order.

The process rail receives more space above it so it reads as explanation rather than
another command control.

## Process rail

- Reduce the desktop maximum width from `62rem` to `50rem`.
- Keep three equal columns and the existing icon, number, heading, and detail structure.
- Preserve the solid raised background, full border, rounded corners, and internal
  separators.
- Preserve the existing stacked mobile layout and `31rem` mobile maximum.

## Navbar width

- Reduce the desktop maximum width from `88rem` to `58rem`, approximately 65 percent of
  the previous maximum.
- Preserve the responsive outer gutter, so tablet and mobile continue using the available
  width rather than becoming an unusably narrow capsule.
- Keep the existing brand lockup, links, theme control, install CTA, and mobile menu.

## Navbar scroll behavior

The transparent-at-top behavior applies only to the homepage.

- At `scrollY <= 12`, the navbar frame background and border are transparent.
- At `scrollY > 12`, restore the normal `--paper-2` background and `--rule` border.
- Returning to the top restores transparency.
- Transition background and border colors over approximately 180ms.
- Docs and other routes remain solid at the top.
- Opening the mobile menu forces the frame solid even at the top so menu links never sit
  directly over illustration content.
- Keep the outer sticky header transparent at all times.
- Use a requestAnimationFrame-throttled passive scroll listener.

## Acceptance criteria

- The headline source and rendered text are unchanged.
- The command, resource links, and compatibility proof read as one compact cluster.
- The process rail is `50rem` wide on desktop and keeps its existing mobile behavior.
- The homepage navbar is `58rem` wide on desktop.
- Homepage navbar border/background are invisible at the top and return after scrolling
  beyond 12px.
- Returning to the top restores transparency.
- Non-homepage navbars are solid immediately.
- Opening the mobile menu forces a solid frame.
- No page-level horizontal overflow at 375px, 768px, 1440px, or 2048px.
- Light and dark themes, keyboard focus, copy feedback, and mobile menu behavior remain
  intact.
