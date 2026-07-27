# Design System: Reinstate

The source of truth for every Reinstate surface: marketing, docs, README art, social cards.
It is extracted from the canonical hero (`website/src/pages/preview/exploded.astro`) and
everything else is built to match it. When this document and a component disagree, this
document wins.

---

## 1. The one-line brief

**A sharp, flat-vector isometric workroom, drawn in bold outlines on a green-neutral paper,
where the product is shown as a physical object being pulled apart, sealed, and put back.**

Not a chat product. Not a SaaS dashboard. A tool you can see the mechanism of.

---

## 2. Non-negotiable rules

These are the rules that make everything look like one system. Break them and it stops
being Reinstate.

1. **Sharp, never soft.** No `box-shadow` with blur, no `filter: blur()`, no glassmorphism,
   no soft radial glows, no gradient haze. Depth comes from **outlines, flat colour faces,
   and stacking order**. The only permitted `box-shadow` is a zero-blur ring used as a
   border substitute (`0 0 0 3px`), and even that should usually be a real border.
2. **Outlines carry the drawing.** Every illustrated solid gets a `3.2px` stroke in the ink
   colour (light theme) or the paper colour (dark theme), `stroke-linejoin: round`.
3. **Flat fills only.** Three tones per solid: top face lightest, front face mid, side face
   darkest. No gradients inside shapes.
4. **True axonometric.** All 3D is projected at build time from real `(x, y, z)` coordinates
   at a 30° isometric angle. Never hand-fudge a parallelogram.
5. **One accent.** Chartreuse is the only loud colour on the page. Amber, sky, and bone
   appear **only** as the four stream identities inside illustration and legends.
6. **Light is the default, dark is a first-class twin.** Both ship. Neither is an
   afterthought. Every token has a value in both.
7. **No em dashes** in any copy. Commas, colons, or periods.

---

## 3. Colour

OKLCH for interface tokens, hex for illustration fills (they are art, not theme surface).

### 3.1 Interface tokens

| Role | Token | Light | Dark |
|---|---|---|---|
| Page surface | `--paper` | `#e4e7dd` | `#1d2723` |
| Raised surface | `--paper-2` | `#f4f6ef` | `#26332e` |
| Primary text | `--ink` | `oklch(0.19 0.022 158)` | `oklch(0.965 0.008 145)` |
| Secondary text | `--ink-2` | `oklch(0.41 0.018 158)` | `oklch(0.76 0.014 150)` |
| Tertiary text | `--ink-3` | `oklch(0.54 0.014 158)` | `oklch(0.62 0.016 150)` |
| Hairline | `--rule` | `oklch(0.80 0.012 145)` | `oklch(0.31 0.02 158)` |
| Accent | `--prim` | `oklch(0.87 0.21 128)` | `oklch(0.88 0.21 128)` |
| Accent, text-safe | `--prim-deep` | `oklch(0.58 0.155 134)` | `oklch(0.84 0.19 128)` |
| Text on accent | `--on-prim` | `oklch(0.19 0.022 158)` | `oklch(0.17 0.04 132)` |

**`--paper` is the wall colour of the scene.** That is deliberate: the illustration's wall
continues seamlessly into the page background, so the whole hero reads as one environment.
Never set a page background that differs from the scene's wall.

`--prim` is too light to carry text on paper. Use `--prim-deep` for accent text and thin
strokes; use `--prim` only as a fill behind `--on-prim`.

### 3.2 The four stream identities

Fixed. These four colours mean sessions, MCP, skills, and settings everywhere they appear,
including README diagrams and social cards.

The four streams are visual categories, not a limit on roadmap capability
types. `skills` includes portable skills and instruction assets; `settings` is
the umbrella for hooks/loops, plugins, marketplace declarations, and other safe
harness configuration. Credentials are never represented as a portable stream.

| Stream | Top face | Front face | Side face |
|---|---|---|---|
| `sessions` | `#b8ff3c` | `#96d92a` | `#78b31d` |
| `mcp` | `#ffce4a` | `#e0ad2c` | `#bd8d1b` |
| `skills` | `#7ecdf5` | `#57aad6` | `#3d8ab5` |
| `settings` | `#fdfef9` | `#e2e6dc` | `#c3c9bc` |

### 3.3 Scene tokens

| Role | Token | Light | Dark |
|---|---|---|---|
| Outline | `--sv-stroke` | `#131f1a` | `#e9efe3` |
| Wall | `--sv-wall` | `#e4e7dd` | `#1d2723` |
| Floor | `--sv-floor` | `#d3d8c9` | `#16201c` |
| Floor lines | `--sv-floorline` | `#c3c9b8` | `#253029` |
| Skirting | `--sv-skirt` | `#c8cebd` | `#253029` |
| Frames, sills | `--sv-frame` | `#f4f6ef` | `#2b3831` |
| Night glass | `--sv-night` | `#16231f` | `#0c1512` |
| Furniture | `--sv-furn-1/2/3` | `#c1c9ae` `#a6b090` `#8a9673` | `#44564b` `#35453c` `#29362f` |
| Devices | `--sv-dev-1/2/3` | `#3b4c43` `#2a3831` `#1e2925` | `#46584e` `#35453c` `#27332c` |
| Sealed object | `--sv-seal-1/2/3` | `#2a3831` `#1a2621` `#121b17` | `#2b3a33` `#1f2b26` `#16201c` |
| Lit screen | `--sv-glass` | `#3f6b1f` | `#6ba32f` |
| Foliage | `--sv-leaf-1/2` | `#7fae3f` `#5f8f2c` | `#6f9c36` `#4f7a25` |
| Terracotta | `--sv-pot-1/2/3` | `#c98f5a` `#b07845` `#966235` | `#9d6c40` `#855a33` `#6d4927` |

**The dark-theme inversion trick:** the outline flips from near-black to near-white. The
drawing becomes chalk-on-blackboard line art while the stream colours stay saturated. This
is the single move that makes dark mode feel designed rather than dimmed.

### 3.4 Contrast floor

Body text ≥ 4.5:1 against its surface, large text ≥ 3:1, placeholders ≥ 4.5:1. Verify in
both themes. `--ink-3` on `--paper` is the lowest permitted body pairing.

---

## 4. Typography

| Role | Family | Weights | Notes |
|---|---|---|---|
| Display | **Questrial** (`@fontsource/questrial`) | 400 only | Marketing headings + wordmark (`--font-display`). **Not** used in docs prose. |
| Body / UI | **Geist** (`@fontsource-variable/geist`) | 400–560 | All prose, labels, buttons |
| Mono | **Geist Mono** (`@fontsource-variable/geist-mono`) | 400–600 | Commands, paths, captions, legends, in-illustration text |

Exposed as `--font-display`. Swapping the display face is a one-token change.

### Scale

| Step | Size | Tracking | Leading |
|---|---|---|---|
| Hero h1 | `clamp(2.5rem, 6vw, 5.2rem)` | `-0.038em` | `0.96` |
| Section h2 | `clamp(1.75rem, 3vw, 2.75rem)` | `-0.035em` | `1.02` |
| h3 | `1.25rem` | `-0.02em` | `1.2` |
| Body | `clamp(1rem, 0.5vw + 0.92rem, 1.15rem)` | `0` | `1.6` |
| Small | `0.875rem` | `0` | `1.55` |
| Micro / mono | `0.78rem` | `0.01em` | `1.5` |

Hard limits: display `clamp()` max ≤ `6rem`, letter-spacing floor `-0.04em`, measure ≤ 65ch,
`text-wrap: balance` on h1–h3, `text-wrap: pretty` on prose.

**Banned as display:** Inter, DM Sans, Space Grotesk, Outfit, Plus Jakarta, Instrument,
Fraunces, Playfair, Cormorant, IBM Plex.

---

## 5. Shape and space

- **Radius:** `6px` small controls, `8–9px` buttons, `12px` panels, `999px` pills. Never
  above `16px` on a rectangular panel.
- **Borders:** `1px solid var(--rule)` for interface, `3.2px var(--sv-stroke)` for
  illustration. Borders do the work shadows would.
- **Container:** `max-width: 88rem`, inline padding `clamp(1.25rem, 4vw, 3rem)`.
- **Section rhythm:** `clamp(3rem, 7vw, 6rem)` vertical, varied deliberately. Sections are
  separated by a single hairline, not by a shadowed card.
- **Cards are a last resort.** Prefer hairline-separated rows and grids. Never nest cards.

---

## 6. Illustration

The house style. Every diagram on the site, in the README, and in social cards is drawn this
way, so they compose into one world.

### Projection

30° isometric, computed at build time:

```ts
const CX = Math.cos(Math.PI / 6);
const P = (x, y, z) => [(x - y) * CX, (x + y) * 0.5 - z];
```

`+x` goes right-and-down, `+y` goes left-and-down, `+z` goes up. Every solid is a `box(x, y,
z, w, d, h)` returning three polygons: `top`, `front` (the `y+d` face), `side` (the `x+w`
face). Painter's order is back to front, which means ascending `x + y`, then ascending `z`.

### Rules

- Stroke `3.2px`, `stroke-linejoin: round`, no stroke on interior detail smaller than 8px.
- Three flat tones per solid, never a gradient.
- Text inside illustration is **Geist Mono**, set as SVG `<text>` with an explicit
  `text-anchor` and `fill` **attribute**, not a CSS class. Scoped CSS is not reliable for
  every SVG presentation property; pass colours as `fill="var(--token)"` attributes.
- Environment (walls, floor, window) may be drawn in flat screen space. Objects are always
  axonometric. Mixing the two is intentional and fine.
- Arrows use a `<marker>`, dashed `stroke-dasharray: 13 11`, animated with `stroke-dashoffset`.

### Scene vocabulary

The hero establishes a room. Other sections reuse its furniture and props rather than
inventing new ones: desk, tower, monitor with a lit face, laptop on a low table, crate
stack, plinth, sealed cube with a chartreuse lock badge, potted plant, wall clock at 23:41,
night window with a skyline, shelf with coloured books, framed poster, rug, cardboard stack.

### The four-layer motif

Sessions, MCP, skills, settings are **always** four stacked slabs in the fixed colours,
always in that order bottom to top. Pulled apart on the source machine, flush and closed on
the destination machine. This motif is the brand mark of the product. New
configuration capabilities stay within the established `skills` or `settings`
umbrella rather than adding more slabs.

---

## 7. Motion

- Entrances only, once, on load. Duration `0.8–0.9s`, easing `cubic-bezier(0.16, 1, 0.3, 1)`.
- Layers `lift` out (translate up-left) on the source side and `drop` in (translate
  down-right) on the destination side, staggered `0.08–0.11s`.
- Continuous motion is limited to dashed `stroke-dashoffset` flow on connector wires.
- Hover: `background`/`color` transitions at `0.2s`, `transform: scale(0.97)` on `:active`.
- No parallax, no scroll hijack, no marquees, no bounce or elastic easing.
- Every animation needs a `prefers-reduced-motion: reduce` path.

---

## 8. Components

**Button, primary** — `--ink` fill, `--paper` text, radius `9px`, hover swaps to `--prim` /
`--on-prim`, `active:scale(0.97)`. No shadow.

**Button, secondary** — transparent fill, `1px var(--rule)` border, `--ink` text.

**Input group** — a single `--paper-2` container with `1px var(--rule)` border and radius
`12px`, holding a borderless input and the submit button. Focus moves the container border
to `--prim-deep`. No focus glow.

**Badge / pill** — `--paper-2` fill, `1px var(--rule)` border, radius `999px`, `0.78rem`,
with an optional `7px` `--prim-deep` dot. One badge per section maximum.

**Legend row** — a `12px` swatch with a `2px var(--sv-stroke)` border, a mono key, and a
tertiary note. This is how illustration colour is explained in text.

**Numbered steps** — CSS `counter`, number in `--prim-deep` mono, title in mono `0.9rem`,
detail in mono `0.76rem` `--ink-3`. Only for genuine sequences.

**Theme toggle** — `2.1rem` circle, `1px var(--rule)` border, sun/moon SVG stroked with
`currentColor`. State on `html.dark`, persisted to `localStorage['rein-theme']`, seeded by
an inline head script so there is no flash.

**Nav** — single row, wordmark left in the display face, links centre-right in `--ink-2`,
theme toggle, then one primary CTA. No dropdowns.

---

## 9. Layout patterns

**Hero** — centred lede (badge, h1, sub, inline waitlist, micro note) sitting on the wall of
a full-bleed scene that occupies the lower half. The scene anchors to the bottom
(`preserveAspectRatio="xMidYMax meet"`), and the page background matches the wall so there
is no visible seam.

**Under-scene rail** — a hairline-separated block carrying the legend, the numbered steps,
and the supported-agent strip. Explanatory text lives here as real HTML, never as SVG.

**Docs** — same tokens, `max-w-3xl` prose plus sidebar, mono for code, hairline separators,
no cards.

---

## 10. Responsive

- Illustration keeps its size and scrolls inside its own `overflow-x: auto` figure below
  `900px`, with a `min-width` around `1100px` and a scroll hint. The page body must never
  scroll horizontally.
- Nav links collapse below `940px`; the theme toggle and CTA remain.
- In-illustration type steps up on small screens so it stays legible at reduced scale.

---

## 11. Accessibility

WCAG 2.2 AA in both themes. Every illustration carries `<title>` and `<desc>` and is
`role="img"` with `aria-labelledby`. Decorative props are `aria-hidden`. Full keyboard paths
for nav, waitlist, and theme toggle. Visible focus on every control. Readable at 200% zoom
without horizontal scroll of the main column.

---

## 12. Refuse list

Side-stripe borders. Gradient text. Glassmorphism. Drop shadows and blur of any kind. The
big-number hero-metric template. Identical three-card icon grids. Tiny uppercase tracked
eyebrows above every section. Numbered markers on sections that are not sequences. AI purple.
Mesh gradients. Fake dashboard screenshots. Hand-drawn or sketchy SVG. Stock photos of people
coding. Mascots. A decorative terminal window used as the hero.
