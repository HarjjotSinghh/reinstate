# Terminal Proof Section Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:executing-plans to implement this plan. Track progress with the checkbox steps below.

**Goal:** Add a third landing-page section that proves the current Phase 1 handoff with real Reinstate commands and output, without implying later roadmap capabilities.

**Architecture:** Add one `TerminalProof.astro` section after the problem section and
keep it inside `ProblemExploded`'s existing floor wrapper through a component
slot. This makes the axonometric floor grid continue without a visual seam.
The section uses two accessible terminal panels, Windows source and macOS
destination, grounded by an illustrated transfer bench with a sealed
checkpoint between them. The full workflow remains static, selectable DOM
text. The illustrated connector may move, but the evidence never depends on
JavaScript.

**Tech Stack:** Astro 7, scoped component CSS, TypeScript, Vitest, existing landing-page tokens

---

## Approved content and product boundary

### Section copy

- Heading: `One session. Two machines. Four commands.`
- Supporting copy: `List and push on the first machine. Pull on the second. Resume with the agent that created the session in its native format.`
- Security proof: `Representative synthetic output. Commands and summaries match the current CLI. The passphrase stays hidden; transcript contents stay private.`
- Codex clarification: `Using Codex? The flow is identical. Finish with codex resume ses_7f3a.`

### Device A · Windows desktop

```text
PS C:\reinstate> rein list --agent claude
claude  ses_7f3a  local/reinstate

PS C:\reinstate> rein push --agent claude --session ses_7f3a
Encryption passphrase: ••••••••
pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false
```

### Encrypted bridge

```text
age encrypted
your S3 or R2 bucket
project map applied
```

### Device B · MacBook

```text
$ rein pull --agent claude --session ses_7f3a
Encryption passphrase: ••••••••
pulled 1 snapshot(s) dry_run=false

$ claude --resume ses_7f3a
```

The checked-in transcript is representative, synthetic output. Its command
names, flags, prompts, and success-message shapes must remain contract-checked
against the current CLI implementation.

## Chunk 1: Truth contract and semantic structure

### Task 1: Add a terminal-proof contract test

**Files:**
- Create: `website/src/lib/terminal-proof.test.ts`
- Read: `internal/cli/commands_impl.go`
- Read: `internal/crypto/passphrase.go`
- Read: `docs/getting-started.md`

- [x] **Step 1: Write the failing source contract**

Read the future component source plus the current CLI implementation and assert:

- the section uses `rein list`, `rein push`, and `rein pull`;
- native resume is `claude --resume ses_7f3a`, with
  `codex resume ses_7f3a` named as the Codex equivalent;
- the displayed passphrase is explicitly hidden;
- the push and pull summaries follow the current CLI output shapes;
- the section does not claim `rein resume`, cross-agent translation, MCP sync,
  skills sync, Gemini support, or OpenCode support; and
- the section names user-owned S3/R2 storage.

- [x] **Step 2: Run the test and verify it fails**

Run:

```bash
cd website
npm test -- src/lib/terminal-proof.test.ts
```

Expected: fail because `TerminalProof.astro` does not exist yet.

### Task 2: Build the semantic terminal section

**Files:**
- Create: `website/src/components/landing/TerminalProof.astro`
- Modify: `website/src/pages/index.astro`

- [x] **Step 1: Add the section shell**

Create a labelled `<section>` with the approved heading, supporting copy, and a
concise security note. Render it after the problem section, making it the only
new page section. Do not add a third repeated badge above the heading.

- [x] **Step 2: Render selectable terminal output**

Use semantic `<pre><code>` blocks or line-by-line `<code>` rows. Do not render
terminal text into canvas, SVG, or pseudo-elements. Keep prompts and output
available to selection, search, translation, and assistive technology.

- [x] **Step 3: Distinguish commands from output without colour dependence**

Use prompt characters, labels, font weight, and spacing in addition to lime
accents. Mark decorative prompt glyphs `aria-hidden="true"` where the spoken
command already contains sufficient context.

- [x] **Step 4: Add the encrypted bridge**

Between terminal panels, show one compact bridge labelled:

```text
age encrypted · your S3/R2 bucket · path map preserved
```

Do not add another large isometric scene. This section is evidence, not another
illustration flex.

## Chunk 2: Visual system and responsive flow

### Task 3: Continue the illustrated room through the proof

**Files:**
- Modify: `website/src/components/landing/TerminalProof.astro`

- [x] **Step 1: Reuse existing tokens**

Use the landing page's cream, dark green, lime, mono font, display font, rule
colour, and eight-pixel corner language. Avoid gradients and a generic feature
card grid.

- [x] **Step 2: Compose the desktop workflow**

At desktop widths:

- place the Windows terminal on the left;
- place the illustrated sealed checkpoint and encrypted bridge in the centre;
- place the macOS terminal on the right;
- keep both terminals visually equal;
- keep the heading and proof copy within the existing 88rem shell; and
- align the section with the comparison-card edges above it.

The floor grid from `ProblemExploded` must remain visible behind the section.
Use the existing 30-degree projector and three-face fill system for the
transfer bench, checkpoint, device bases, and connector. Commands stay in HTML,
not SVG.

- [x] **Step 3: Compose the mobile workflow**

At 700px and below:

- stack source terminal, illustrated encrypted checkpoint, and destination
  terminal;
- keep every command horizontally scrollable without widening the page;
- preserve a minimum 44px target for any interactive control; and
- keep `document.documentElement.scrollWidth === innerWidth` at 375px.

- [x] **Step 4: Animate only the connector**

Animate only the dashed connector to show the session moving from source to
checkpoint to destination. Never fake-type the transcript, hide completed
commands, or delay access to the full workflow. Disable the connector motion
under `prefers-reduced-motion: reduce`.

## Chunk 3: Verification

### Task 4: Verify product truth, accessibility, and rendering

**Files:**
- Modify: `website/src/lib/terminal-proof.test.ts`
- Modify: `website/src/components/landing/TerminalProof.astro`

- [x] **Step 1: Run the focused contract**

Run:

```bash
cd website
npm test -- src/lib/terminal-proof.test.ts
```

Expected: pass.

- [x] **Step 2: Run the full website suite**

Run:

```bash
npm test
```

Expected: all tests pass.

- [x] **Step 3: Run the production build**

Run:

```bash
npm run build
```

Expected: the Astro/Vercel build completes successfully.

- [x] **Step 4: Inspect desktop light and dark modes**

Verify:

- terminal text meets WCAG AA contrast;
- the encrypted bridge reads as flow, not a third terminal;
- neither panel is visually dominant; and
- the new section continues naturally from the comparison cards.

- [x] **Step 5: Inspect 375px mobile**

Verify:

- no page-level horizontal overflow;
- all commands remain selectable and readable;
- the workflow order is unambiguous; and
- reduced motion exposes the complete static state.

- [x] **Step 6: Confirm scope**

Run:

```bash
git diff --check
git status --short
```

Confirm the implementation adds only the terminal-proof component, its contract
test, and the single `index.astro` insertion. Do not add security, single-device,
open-source, or final-CTA sections in the same change.
