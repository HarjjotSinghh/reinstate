# Two-section Landing Credibility Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the existing hero and problem/comparison sections accurately explain and convert for the currently released Phase 1 workflow.

**Architecture:** Keep `HeroExploded.astro` and `ProblemExploded.astro` as the only landing sections. Replace the hero waitlist conversion path with an install command and links, add a compact in-section mechanism row, tighten product claims, and raise supporting typography. Update the shared header only for the approved brand-size and install-CTA changes.

**Tech Stack:** Astro 7, scoped component CSS, TypeScript, Vitest, browser-use visual verification

---

## Chunk 1: Product truth and hero conversion

### Task 1: Add landing-copy contract coverage

**Files:**
- Create: `website/src/lib/landing-copy.test.ts`
- Read: `ROADMAP.md`
- Read: `website/src/content/docs/getting-started.md`

- [x] **Step 1: Write a focused source contract test**

Read the landing components as text and assert that:

- the current-agent status names Claude Code and Codex;
- the hero contains the public POSIX installer;
- the waitlist-first CTA is absent from the hero and header;
- the problem headline remains unchanged; and
- unsupported live-product claims such as `rein resume`, Gemini availability,
  and MCP/skills synchronization are absent from the two sections.

- [x] **Step 2: Run the focused test and verify it fails**

Run:

```bash
npm test -- src/lib/landing-copy.test.ts
```

Expected: failures for the old headline, waitlist CTA, and future-scope claims.

### Task 2: Update the shared header

**Files:**
- Modify: `website/src/components/Header.astro`
- Read: `website/src/components/BrandLockup.astro`

- [x] **Step 1: Increase the lockup without changing its geometry**

Change the header lockup from `size={32}` to `size={36}` and adjust the local
font-size hook proportionally.

- [x] **Step 2: Replace the waitlist CTA**

Change `Join the waitlist` and `/#join` to `Install Reinstate` and
`/docs/getting-started`.

- [x] **Step 3: Raise navigation readability**

Increase desktop navigation and CTA type modestly while preserving the current
header height and mobile collapse behavior.

### Task 3: Replace hero copy and conversion controls

**Files:**
- Modify: `website/src/components/landing/HeroExploded.astro`
- Remove import usage: `website/src/components/WaitlistForm.astro`

- [x] **Step 1: Replace the hero message**

Use:

- Badge: `Capture. Encrypt. Reinstate.`
- Headline: `Pick up your coding session on another device.`
- Body: `Reinstate moves Claude Code and Codex sessions between your machines. Everything is encrypted locally before it reaches your S3 or R2 bucket.`

- [x] **Step 2: Add the install command**

Render:

```text
curl -fsSL https://reinstate.dev/install.sh | sh
```

Include a `Copy` button whose accessible name identifies the command. Preserve
text selection and update the label to `Copied` after a successful clipboard
write. Restore the original label after a short delay and leave the command
selectable if clipboard access fails.

- [x] **Step 3: Add primary and secondary links**

- `Install Reinstate` → `/docs/getting-started`
- `View on GitHub` → `https://github.com/HarjjotSinghh/reinstate`

- [x] **Step 4: Add current support and technical proof**

Show:

- `Available now: Claude Code · Codex`
- `macOS ↔ Windows`
- ``rein` is the CLI for Reinstate`
- `Apache-2.0 · one Go binary · your bucket, your keys`

- [x] **Step 5: Add the compact three-step mechanism**

Inside the existing hero lede, render:

1. `Capture` — Select a local session
2. `Encrypt` — Seal it before upload
3. `Reinstate` — Pull and resume natively

Use one compact row on desktop and a clean stacked/grid layout on mobile. Do not
create a new `<section>`.

- [x] **Step 6: Adjust hero spacing**

Accommodate the added proof without shrinking the illustration or allowing the
lede to collide with wall art. Keep the existing illustration geometry and
command chips unchanged.

## Chunk 2: Problem clarity and comparison readability

### Task 4: Tighten the problem explanation

**Files:**
- Modify: `website/src/components/landing/ProblemExploded.astro`

- [x] **Step 1: Preserve the approved headline**

Keep:

```text
Git has the code and its history.
It does not have the conversation.
```

- [x] **Step 2: Replace the supporting paragraph**

Use:

```text
Git still handles your code. Reinstate carries the encrypted Claude Code and
Codex session state Git leaves behind, remapping project paths for the machine
you are on.
```

- [x] **Step 3: Replace the comparison claims**

Without Reinstate:

- Sessions trapped on one machine
- Paths and agent state drift apart
- Context must be rebuilt manually

With Reinstate:

- Pull sessions onto another device
- Remap paths across macOS and Windows
- Store encrypted state in your own bucket

- [x] **Step 4: Increase comparison readability**

Increase kicker, heading, list, and section-body sizes by roughly 15 percent.
Increase text contrast without changing the approved 50 percent translucent
card fill.

## Chunk 3: Responsive behavior and verification

### Task 5: Make the revised hero responsive and accessible

**Files:**
- Modify: `website/src/components/landing/HeroExploded.astro`
- Modify: `website/src/components/Header.astro`

- [x] **Step 1: Add narrow-screen layout rules**

At 560 px and below:

- stack or wrap CTA links cleanly;
- keep the install command within the viewport;
- keep the copy button at least 44 px high;
- arrange mechanism items without tiny type; and
- prevent horizontal overflow at 375 px.

- [x] **Step 2: Preserve keyboard and reduced-motion behavior**

Add visible focus states for the new buttons and links. Do not add new motion.

- [x] **Step 3: Run the focused contract test**

Run:

```bash
npm test -- src/lib/landing-copy.test.ts
```

Expected: pass.

- [x] **Step 4: Run the complete website tests**

Run:

```bash
npm test
```

Expected: all tests pass.

- [x] **Step 5: Run the production build**

Run:

```bash
npm run build
```

Expected: Astro/Vercel build completes successfully.

- [x] **Step 6: Inspect desktop light and dark modes**

Verify the hero, problem lede, and comparison cards at the desktop viewport.

- [x] **Step 7: Inspect a 375 px mobile viewport**

Verify:

- no horizontal overflow;
- the command remains readable/selectable;
- buttons remain usable;
- mechanism labels do not become decorative dust; and
- the existing illustrations retain their intended crop.

- [x] **Step 8: Confirm the scope boundary**

Run:

```bash
git diff --check
git status --short
```

Confirm only the approved header, hero, problem, test, spec, and plan files are
in scope. Do not commit, push, deploy, or build section three unless explicitly
requested.
