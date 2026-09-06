# ADR 0006: Phase 7 project continuity — scope, storage model, and boundaries

## Status

Accepted 2026-09-06 by the maintainer's direct answers to the twelve questions
below. This is a planning decision only: it changes documentation and the
roadmap. No code, CLI surface, or release contract changes, and nothing in it
affects the in-flight `v0.6.0` acceptance under
[ADR 0005](0005-v0.6.0-scope-and-windows-first-acceptance.md).

Design direction lives in
[docs/project-continuity.md](../project-continuity.md); the verified survey of
existing tools lives in
[docs/research/2026-09-06-phase-7-cross-agent-context-landscape.md](../research/2026-09-06-phase-7-cross-agent-context-landscape.md).

## Context

Phase 6 makes the *configuration* of an agent environment portable: MCP
servers, skills, instructions, hooks, plugins, and safe settings declared once
and rendered per harness (6A/6B, `v0.7.0`), with encrypted multi-device
sync already shipped (6C, `v0.6.0`).

A different class of state is still fragmented, and no phase owned it. A
project's rules, decisions, and the knowledge an agent earns while working live
inside whichever harness learned them. Claude Code keeps `CLAUDE.md` plus its
own per-project memory; Codex, Cursor, Grok, OpenCode, and Gemini CLI each keep
separate instruction files and separate stores. A rule stated to one agent
binds one agent. The next agent contradicts a decision recorded weeks earlier,
and the contradiction surfaces at review, if at all.

The maintainer raised this as a possible standalone product, together with two
external AI analyses of the space. Both analyses independently concluded that
the work belongs inside Reinstate rather than beside it, on grounds that hold
up against this repository: the same users, the same harness adapters, the same
project-identity and path-mapping layer, and the same encryption and sync
substrate would otherwise be built twice. The roadmap's own Phase 6 already
describes the configuration half of it.

Both analyses also observed that the space is crowded — rules distribution,
MCP gateways, and cross-agent memory servers all exist — and that the
unsolved part is the write path: something learned in one agent's session
becoming durable, attributable, current project state for every agent.

## Decisions

**D1 — Project continuity is a new Phase 7; Console and Team continuity
renumber to 8 and 9.** It is a peer of session sync and environment
continuity, not a sub-item of configuration. The renumbering costs edits in
`ROADMAP.md` and `website/src/pages/roadmap.astro`, which duplicates the phase
table by hand. Historical records that describe the *previous* renumbering
(CHANGELOG entries, ADR 0004, the `v0.5.0` task cards) keep their original
numbers: the record of what was decided then is not rewritten.

**D2 — Targeted at `v0.8.0`, after Phase 6A/6B in `v0.7.0`.** Rendering
project context into native instruction files reuses the configuration-adapter
contract 6A defines. Shipping the two together would put configuration, auth,
context, and memory into a single pre-1.0 release currently accepted on one
platform.

**D3 — The phase covers four blocks, all in scope.** 7A canonical project
context; 7B shared memory with provenance; 7C capture, promotion, and conflict
detection; 7D cross-agent readiness reporting. Dropping 7C would reduce the
phase to a rules-file distributor, which is the part of this space that is
already well served.

**D4 — The canonical source is the Reinstate project tree; `AGENTS.md` and
harness-native instruction files are render targets.** Scoped, typed sources
compile through the 6A adapters into delimited generated regions that never
overwrite hand-written prose. `rein context adopt` imports existing
`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, and editor rule files, so adoption is
not a rewrite. Making a single flat file canonical was rejected: it cannot
carry path scoping, provenance, supersede links, or expiry, and it is precisely
the artifact that bloats until agents skim it. Reinstate writes into the
interoperability format; it does not compete with it.

**D5 — Hybrid storage.** Durable, human-reviewed context is committed to the
repository: reviewable in pull requests, useful to a teammate who has never
installed Reinstate, and intact if Reinstate is uninstalled. Learned memory and
its provenance live in the local encrypted store. Git remains source truth;
Reinstate remains context truth.

**D6 — Reinstate ships a local MCP server as the primary read/write path for
memory.** `memory.search/add/update/supersede/promote`, exposed to any
MCP-capable harness, alongside the daemon that already keeps sync current. It
is a store with provenance, not an execution ecosystem: no agent loop, no
scheduling, no model routing, no plugin runtime. It therefore stays inside the
"client, not harness" boundary. Harnesses that cannot speak MCP still receive
durable context through rendered instruction files — they read, they do not
contribute. This adds a long-lived local surface that needs its own security
review before implementation.

**D7 — Capture is explicit; promotion is human.** A memory is created by an
MCP tool call or `rein remember`, never by mining transcripts in the
background. An agent's write enters a review queue as *proposed*; promotion to
reviewed context in the repository is a person's act. Reinstate already indexes
session history, so background mining is technically available and is
deliberately declined: deriving durable project state from conversation content
without an explicit act is a different privacy contract than this project
makes.

**D8 — Memory syncs; capsules still do not.** Learned memory is non-secret
project state and syncs like sessions, because "the same project understanding
on the other machine" is the point of the phase. Handoff capsules remain
hard-excluded from sync (Phase 4). Every memory record passes the secret
scanner before it is stored or synced, and memory is redacted and bounded in
output surfaces the way transcripts are.

**D9 — The phase gate names six harnesses:** Claude Code, Codex, Cursor, Grok,
OpenCode, and Gemini CLI. This matches the 6A gate wording plus Cursor. Support
is a configuration-axis claim only; it does not move any agent's session tier
(`docs/agent-support-tiers.md`), and each harness's evidence is recorded the
same way Phase 5 records tier evidence.

**D10 — An authenticate-once MCP gateway stays 💭, gated on three named
conditions.** Native projection and guided login flows (6B) ship first.
Reinstate builds a gateway only if declared configuration reconciles cleanly
while authentication remains the dominant reported friction, no existing
gateway can simply be declared as desired state and reconciled instead of
replaced, and routing every agent's tool calls through a Reinstate process
measurably improves continuity rather than adding a component to trust.

**D11 — Positioning extends; the homepage does not change.** The vision
paragraph, value ladder, product-layers table, validation survey, and
`docs/product-strategy.md` gain a project-understanding rung. The homepage
hero, the "pick up any coding task exactly where you left it" line, and every
shipped-capability claim stay as they are. The phase is planned, and no
surface may imply otherwise.

**D12 — Prior art is documented only from verified sources.** The two external
analyses named roughly twenty projects and two arXiv papers. Nothing enters
`ROADMAP.md`, `docs/`, or the website unless it was checked against a primary
source, with the retrieval date recorded; unverifiable claims are dropped or
marked unverified in `docs/research/`. Competitor star counts, funding claims,
and feature lists are the exact material that ages into an inaccuracy.

## Consequences

- Two non-goals are added to the roadmap: inferring durable project rules from
  transcripts without an explicit act, and parallel-agent file claims, locks,
  or arbitration. The second keeps the existing "harnesses own concurrent
  execution" boundary intact while shared memory makes coordination newly
  tempting.
- The value ladder gains a rung ("one project understanding → every agent") and
  the product-layers table gains a row, shifting cloud continuity to 5 and team
  continuity to 6.
- `rein status` becomes a named future surface. It overlaps `rein doctor`, and
  the split between them is an open question for the RFC, not a decision here.
- The MCP server, the capture contract, and the memory record schema each need
  an RFC before implementation, in line with how configuration commands were
  deferred to an RFC in `docs/universal-configuration.md`.
- Nothing here is implemented. Every table row in Phase 7 is 📋, and the phase
  is documentation until an acceptance run says otherwise.

## Alternatives rejected

- *Build it as a standalone project.* Two distributions, two adapter matrices,
  two project-identity layers, two sync and encryption stacks, and an
  integration issue between them within a month. The overlap with Reinstate is
  near total.
- *Fold it into Phase 6 as 6D/6E.* Understates a body of work larger than 6A
  and blurs two genuinely different kinds of state: declared configuration
  versus what the project knows.
- *Append it after team continuity.* Signals a priority the maintainer does not
  hold; the pain is present-tense and single-device.
- *Make `AGENTS.md` the canonical source.* Rejected in D4.
- *Mine transcripts automatically with a review queue.* Highest leverage and
  the largest privacy shift; rejected in D7. It remains available later as an
  opt-in, if explicit capture proves too sparse in practice.
- *Ship a Reinstate MCP gateway now.* Rejected in D10: a proxy that sees every
  tool call from every agent is the largest security surface in the product,
  entering a category where established tools already work.
