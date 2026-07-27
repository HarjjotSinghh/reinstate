# Hero Hierarchy and Navbar Scroll Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tighten the existing hero action hierarchy, narrow the process rail and navbar, and make the homepage navbar transparent only while the page is at the top.

**Architecture:** Keep the implementation inside `HeroExploded.astro` and `Header.astro`. The hero uses one conversion metadata row beneath the command, while the header owns its route-aware scroll state and mobile-menu solid-state override. Source-level Vitest contracts prevent the excluded headline from changing and protect the new structural hooks.

**Tech Stack:** Astro 7, TypeScript, scoped CSS, Vitest, Playwright browser verification against the existing port-4321 preview.

---

## Chunk 1: Contracts and Hero Hierarchy

### Task 1: Add failing source contracts

**Files:**
- Modify: `website/src/lib/landing-copy.test.ts`
- Test: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Preserve the excluded headline**

Keep:

```ts
expect(hero).toContain('Switch devices without losing the thread.');
```

- [ ] **Step 2: Add hierarchy hooks**

Require:

```ts
expect(hero).toContain('class="conversion-meta"');
expect(hero).toContain('max-width: 50rem');
```

- [ ] **Step 3: Add route-aware navbar hooks**

Require:

```ts
expect(header).toContain("const onHome = path === '/'");
expect(header).toContain("'is-home'");
expect(header).toContain("'is-scrolled'");
expect(header).toContain('window.scrollY > 12');
expect(header).toContain('max-width: 58rem');
```

- [ ] **Step 4: Run the focused test**

Run:

```bash
cd website && npm test -- src/lib/landing-copy.test.ts
```

Expected: FAIL because the new hierarchy and scroll hooks do not exist yet.

### Task 2: Consolidate the hero action cluster

**Files:**
- Modify: `website/src/components/landing/HeroExploded.astro`
- Test: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Move compatibility proof into the conversion cluster**

Wrap the existing `.hero-links` and `.availability` elements in:

```astro
<div class="conversion-meta">
  ...
</div>
```

Keep the command first and preserve all existing labels, URLs, and compatibility claims.

- [ ] **Step 2: Rebuild spacing and width**

- Set `.hero-conversion` to the same `43rem` maximum as the command.
- Make `.conversion-meta` a full-width flex row with `space-between`.
- Remove the availability element's independent top margin and entrance animation.
- Keep links and availability visually quiet relative to the command.
- Stack `.conversion-meta` below 560px.

- [ ] **Step 3: Narrow and separate the process rail**

- Change desktop `.mechanism` maximum width to `50rem`.
- Increase its top margin to approximately `1.5rem`.
- Preserve the existing `31rem` stacked mobile maximum.

- [ ] **Step 4: Run the focused test**

Run:

```bash
cd website && npm test -- src/lib/landing-copy.test.ts
```

Expected: header assertions still fail, while hero assertions pass.

## Chunk 2: Navbar State

### Task 3: Add route-aware transparent-at-top behavior

**Files:**
- Modify: `website/src/components/Header.astro`
- Test: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Mark the homepage**

Add:

```ts
const onHome = path === '/';
```

Render the outer header with an `is-home` class only on the homepage.

- [ ] **Step 2: Add scroll state**

Add a passive, requestAnimationFrame-throttled scroll listener that toggles
`is-scrolled` when:

```ts
window.scrollY > 12
```

Run the sync once on page load.

- [ ] **Step 3: Add mobile-menu state**

Toggle `is-open` on the header when opening and closing the mobile menu. An open menu
must force the frame solid even at the top.

- [ ] **Step 4: Narrow and transition the frame**

- Change `.nav-frame` maximum width from `88rem` to `58rem`.
- Preserve the existing responsive gutters.
- Set homepage top-state background and border to transparent.
- Restore normal background and border for `.is-scrolled` and `.is-open`.
- Keep non-home routes solid.
- Use a short background/border transition with no blur or shadow.

- [ ] **Step 5: Run the focused test**

Run:

```bash
cd website && npm test -- src/lib/landing-copy.test.ts
```

Expected: PASS.

## Chunk 3: Verification

### Task 4: Verify production and browser behavior

**Files:**
- Verify: `website/src/components/landing/HeroExploded.astro`
- Verify: `website/src/components/Header.astro`
- Verify: `website/src/lib/landing-copy.test.ts`

- [ ] **Step 1: Run automated gates**

```bash
cd website
npm test
npm run build
```

Expected: all tests and the Astro production build pass.

- [ ] **Step 2: Check the diff**

```bash
git diff --check
```

Expected: no whitespace errors.

- [ ] **Step 3: Verify the local preview**

Inspect `http://localhost:4321` at 375px, 768px, 1440px, and 2048px in both themes.

Confirm:

- the headline text is unchanged
- no page-level horizontal overflow
- command, links, and compatibility read as one group
- process rail is visibly narrower
- desktop navbar is approximately 58rem wide
- homepage top state is transparent
- scrolling beyond 12px restores the solid frame
- returning to the top restores transparency
- opening the mobile menu forces a solid frame
- docs navigation remains solid at the top

- [ ] **Step 4: Commit only scoped files**

```bash
git add website/src/components/landing/HeroExploded.astro website/src/components/Header.astro website/src/lib/landing-copy.test.ts
git commit -m "feat(website): tighten hero hierarchy"
```
