# Manual browser and accessibility acceptance

This record complements the automated static, rendered Lighthouse, link,
metadata, media, and schema gates. It is not complete until an operator tests
the deployed release in real browsers and attaches evidence. Do not convert an
unchecked row into a passing claim from source inspection alone.

## Release record

| Field | Value |
| --- | --- |
| Release/tag |  |
| Commit SHA |  |
| Deployment URL |  |
| Operator |  |
| UTC start/end |  |
| macOS browser/version |  |
| Windows browser/version |  |
| Mobile browser/device or emulator |  |
| Assistive technology/version |  |
| Evidence location and digest |  |

## Required modes

Run each representative template in:

1. light theme;
2. dark theme;
3. a narrow viewport at or below 390 CSS pixels;
4. desktop at 200% browser zoom;
5. keyboard-only navigation; and
6. JavaScript disabled, followed by a fresh navigation rather than a reload
   from browser cache.

Representative routes:

| Template | Route | Light | Dark | Narrow | 200% | Keyboard | No JS | Evidence/finding |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Homepage | `/` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Docs hub | `/docs` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Technical doc | `/docs/getting-started` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| FAQ | `/docs/faq` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Troubleshooting | `/docs/troubleshooting` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Guide | `/guides/sync-claude-code-sessions-across-devices` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Blog article | `/blog/why-git-does-not-sync-coding-agent-sessions` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Integration | `/integrations/claude-code` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Comparison | `/compare/reinstate-vs-manual-session-copying` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Data/table page | `/compatibility` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Static fact page | `/about/reinstate` | Not run | Not run | Not run | Not run | Not run | Not run |  |
| Error page | a nonexistent path | Not run | Not run | Not run | Not run | Not run | Not run |  |

If a template fails, inspect every route using that template after the fix.
Before launch, also open every canonical in the generated sitemap at narrow
width and confirm it has no horizontal page overflow, clipped primary content,
or obstructed controls.

## Keyboard and focus acceptance

- [ ] `Tab` reaches the skip link first and activating it moves focus to
      `#main-content`.
- [ ] Header, mobile menu, theme control, command-copy controls, and every
      content link are reachable in a logical order.
- [ ] Focus is visible in both themes and never hidden under sticky UI.
- [ ] The mobile menu exposes accurate expanded state and can be opened and
      closed without a pointer.
- [ ] No keyboard trap exists.
- [ ] Buttons perform actions and links navigate; no clickable non-semantic
      element substitutes for either.
- [ ] Copy feedback is announced without moving focus.
- [ ] Reduced-motion mode does not remove content or make state changes
      ambiguous.

## Zoom, reflow, and theme acceptance

- [ ] At 200% zoom, text reflows without two-dimensional scrolling except
      genuinely wide code or data tables.
- [ ] At 390 CSS pixels, no heading, command, table control, navigation item,
      or footer link is clipped or overlaps another control.
- [ ] Code blocks and tables provide local horizontal scrolling without making
      the whole page scroll sideways.
- [ ] Text, focus rings, links, controls, diagrams, and status indicators remain
      perceivable in light and dark themes.
- [ ] Meaning is not encoded by color alone.
- [ ] Theme changes do not cause a route, title, canonical, or critical content
      change.

## JavaScript-disabled acceptance

- [ ] H1, direct answer, body copy, steps, FAQs, tables, comparison facts, and
      contextual links remain in the initial HTML.
- [ ] Canonical, robots, Open Graph, Twitter, and JSON-LD metadata remain
      present.
- [ ] Primary navigation uses real links and remains usable.
- [ ] Theme and copy enhancements may be unavailable, but their absence does
      not hide instructions or block navigation.
- [ ] A nonexistent URL still returns the branded error body with an actual
      `404` response.

## Assistive-technology acceptance

- [ ] Landmarks and headings expose a coherent page outline.
- [ ] Logo, decorative art, diagrams, and responsive media have the intended
      accessible names or are correctly hidden.
- [ ] Tables announce headers and preserve row/column context.
- [ ] Form controls and error/status messages have programmatic labels.
- [ ] The Open Graph image is not used as a substitute for accessible page
      content.

## Finding record

| ID | Route/mode | Severity | Reproduction | Expected | Actual | Owner | Fix commit | Retest evidence | State |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |  |

Block launch for loss of primary content/navigation, keyboard traps, invisible
focus, unusable 200% reflow, missing labels, failed 404 status, or critical
contrast failures. Record lesser defects with an owner and deadline; do not
erase failed observations after a fix.

## Local automated preflight — 2026-07-27

Commit `59cecde121ce06a9ccbd7f5b93329a37082cdec1` passed the 16-route
Lighthouse matrix in Chromium `152.0.7977.0`: SEO, accessibility, and best
practices were 100 on every route; performance was 92–99; CLS was zero. The
homepage retained a documented 2,852 ms lab-LCP warning.

The synthetic path visualizer also passed a 390×844 Chromium interaction check
for all four direction/adapter combinations. Output matched the fixed fixtures,
and control changes produced zero network requests and zero cookie or browser
storage changes.

These automated results do not mark any table row above as manually passed. No
connected interactive browser was available for the final local review, and
the branch has not been deployed. Light/dark, keyboard-only, 200% zoom, no-JS,
assistive-technology, and real macOS/Windows browser evidence therefore remain
explicit launch gates.
