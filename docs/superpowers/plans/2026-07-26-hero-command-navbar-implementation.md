# Hero Command and Floating Navbar Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the existing hero action hierarchy and navbar so the install command is the primary CTA, the three-step explanation is protected from the art, and the navigation floats in a Reinstate-native bordered shell.

**Architecture:** Keep the change inside the existing Astro components. `HeroExploded.astro` owns hero copy, command interaction, supporting links, the mechanism panel, and their responsive styles. `Header.astro` owns the floating desktop/mobile navigation shell. Source-level Vitest assertions protect Phase 1 claims and the approved interaction structure, followed by production-build and browser verification.

**Tech Stack:** Astro 7, TypeScript, scoped CSS, inline SVG, Vitest, Playwright/browser inspection through the existing local preview.

---

## Chunk 1: Contract and Hero Conversion

### Task 1: Protect the approved copy and interaction hierarchy

**Files:**
- Modify: `website/src/lib/landing-copy.test.ts`
- Test: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Replace the old headline assertion**

Assert that the hero contains:

```ts
expect(hero).toContain('Switch devices without losing the thread.');
expect(hero).toContain(
  'Move encrypted Claude Code or Codex sessions between macOS and Windows.',
);
expect(hero).toContain(
  'Reinstate encrypts the session locally, then stores it in your own S3 or R2 bucket.',
);
```

- [ ] **Step 2: Add the command-first CTA contract**

Add an assertion block that requires:

```ts
expect(hero).toContain('aria-label="Copy install command"');
expect(hero).toContain('Read setup guide');
expect(hero).toContain('View on GitHub');
expect(hero).not.toContain('class="cta primary"');
expect(hero).not.toContain('class="copy-label"');
```

- [ ] **Step 3: Add the mechanism and floating-shell contract**

Require stable hooks without asserting presentation-value trivia:

```ts
expect(hero).toContain('class="step-icon"');
expect(hero).toContain('class="mechanism"');
expect(header).toContain('class="nav-frame"');
```

- [ ] **Step 4: Run the focused test and confirm it fails**

Run:

```bash
cd website && npm test -- src/lib/landing-copy.test.ts
```

Expected: FAIL because the old headline, CTA row, text copy label, and full-width header still exist.

### Task 2: Implement the command-first hero hierarchy

**Files:**
- Modify: `website/src/components/landing/HeroExploded.astro:191-252`
- Modify: `website/src/components/landing/HeroExploded.astro:763-790`
- Modify: `website/src/components/landing/HeroExploded.astro:921-1162`
- Modify: `website/src/components/landing/HeroExploded.astro:1358-1417`
- Test: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Replace the headline and supporting copy**

Use exactly:

```astro
<h1>Switch devices without losing the thread.</h1>
<p class="sub">
  Move encrypted Claude Code or Codex sessions between macOS and Windows.
  Reinstate encrypts the session locally, then stores it in your own S3 or R2 bucket.
</p>
```

- [ ] **Step 2: Replace the copy label with stable icons and a live region**

Inside the copy button, render an overlapping-pages SVG and a check SVG. Add a visually
hidden live region beside the button:

```astro
<svg class="copy-icon" aria-hidden="true">...</svg>
<svg class="copied-icon" aria-hidden="true">...</svg>
<span id="copy-status" class="sr" aria-live="polite"></span>
```

The button keeps `aria-label="Copy install command"` and gains `data-state="idle"`.

- [ ] **Step 3: Replace the button row with supporting links**

Remove `.cta-row` and both `.cta` elements. Add:

```astro
<nav class="hero-links" aria-label="Installation resources">
  <a href="/docs/getting-started">Read setup guide ...</a>
  <a href="https://github.com/HarjjotSinghh/reinstate" ...>View on GitHub ...</a>
</nav>
```

Use inline arrow/external-link icons. The command remains the visually dominant action.

- [ ] **Step 4: Add icons to the genuine three-step sequence**

Add one `span.step-icon` with a simple stroke SVG per item:

- Capture: a session/document entering a container
- Encrypt: a lock
- Reinstate: a return arrow entering a device

Keep each existing `.step-no`, title, and detail.

- [ ] **Step 5: Update copy interaction state**

Replace text mutation with state and live-region updates:

```ts
copyButton.dataset.state = 'copied';
copyButton.setAttribute('aria-label', 'Install command copied');
copyStatus.textContent = 'Install command copied';
```

On failure, select the command text and announce:

```ts
copyStatus.textContent = 'Command selected. Press Control C or Command C to copy.';
```

After 1.8 seconds, restore `data-state="idle"` and `aria-label="Copy install command"`.

- [ ] **Step 6: Rebuild the conversion styles**

- Remove the blurred command shadow.
- Make the trailing copy control square, at least `3rem` wide, with centered 18–20px SVG.
- Show `.copied-icon` only for `[data-state='copied']`.
- Style `.hero-links` as understated inline links with 44px minimum hit height.
- Delete obsolete `.cta-row` and `.cta` rules.
- Make `.mechanism` a solid `--paper-2` panel with full border and `12px` radius.
- Add 36px icon tiles with a full hairline border.
- Preserve internal separators and stack them horizontally below 900px.
- Keep the mechanism above the illustration with its existing z-index and opaque surface.

- [ ] **Step 7: Run the focused test**

Run:

```bash
cd website && npm test -- src/lib/landing-copy.test.ts
```

Expected: PASS.

- [ ] **Step 8: Commit the hero conversion**

```bash
git add website/src/components/landing/HeroExploded.astro website/src/lib/landing-copy.test.ts
git commit -m "feat(website): make install command the hero CTA"
```

## Chunk 2: Floating Navigation

### Task 3: Convert the header to a floating shell

**Files:**
- Modify: `website/src/components/Header.astro:8-60`
- Modify: `website/src/components/Header.astro:102-252`
- Test: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Add an explicit floating frame**

Wrap the desktop row and mobile panel:

```astro
<header class="nav">
  <div class="nav-frame">
    <div class="shell nav-inner">...</div>
    <div id="mobile-nav" class="mobile-nav" hidden>...</div>
  </div>
</header>
```

- [ ] **Step 2: Replace the full-width bar styling**

Apply:

- transparent, borderless sticky outer `.nav`
- `top: 0`, with `padding: 12px clamp(12px, 2vw, 24px) 0`
- centered `.nav-frame` with `max-width: 88rem`
- solid `var(--paper-2)` background
- full `1px solid var(--rule)` border
- `14px` radius
- no blur or soft shadow
- slightly shorter desktop row while retaining comfortable hit targets

- [ ] **Step 3: Integrate the mobile menu**

Keep the expanded menu inside `.nav-frame`, separated with one top hairline. Ensure the
frame clips its own rounded corners without clipping focus outlines inside the menu.

- [ ] **Step 4: Verify keyboard state semantics**

Keep `aria-expanded` synchronized. Update the burger label between “Open menu” and
“Close menu” when toggled.

- [ ] **Step 5: Run the focused test**

Run:

```bash
cd website && npm test -- src/lib/landing-copy.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit the floating navigation**

```bash
git add website/src/components/Header.astro website/src/lib/landing-copy.test.ts
git commit -m "feat(website): float the primary navigation"
```

## Chunk 3: Verification and Local Preview

### Task 4: Verify behavior, themes, and responsive layout

**Files:**
- Verify: `website/src/components/landing/HeroExploded.astro`
- Verify: `website/src/components/Header.astro`
- Verify: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Run the website test suite**

Run:

```bash
cd website && npm test
```

Expected: all tests pass.

- [ ] **Step 2: Run the production build**

Run:

```bash
cd website && npm run build
```

Expected: Astro build exits successfully and writes `website/dist`.

- [ ] **Step 3: Check formatting and whitespace**

Run:

```bash
git diff --check
```

Expected: no whitespace errors.

- [ ] **Step 4: Inspect the live page on port 4321**

Use the existing local preview at:

```text
http://localhost:4321
```

If the server is not active, stop only the process bound to port 4321 and start:

```bash
cd website && npm run preview -- --host 0.0.0.0 --port 4321
```

- [ ] **Step 5: Verify responsive layouts**

Inspect at:

- 375×812
- 768×1024
- 1440×1000
- 2048×1152

Confirm:

- no page-level horizontal overflow
- floating navbar has visible breathing room on every side
- headline and copy do not collide with the illustration
- command remains the dominant action
- supporting links wrap without becoming button-like
- process panel remains opaque and fully bordered
- step icons and numbers stay aligned
- mobile menu remains inside the rounded shell

- [ ] **Step 6: Verify both themes and interactions**

In light and dark themes:

- keyboard-focus the copy control, links, theme toggle, CTA, and burger
- copy the command and confirm the check icon appears without width shift
- simulate clipboard failure and confirm command selection plus live announcement
- open and close the mobile menu
- verify body and muted text remain readable

- [ ] **Step 7: Record final evidence**

Capture the test count, build result, checked viewports, copy success/fallback behavior,
and the exact local preview URL in the handoff.
