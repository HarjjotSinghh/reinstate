# ADR 0003: Phase 4 `v0.4.0-rc.1` scope, launch route, and capsule storage

## Status

Accepted. Implementation planned for `v0.4.0-rc.1`.
Refines [ADR 0002](0002-cross-agent-continuation.md), which established that
cross-agent continuation is a core explicit handoff. This ADR decides the four
questions ADR 0002 deliberately left open.

## Context

ADR 0002 committed Reinstate to explicit portable handoffs with measured
fidelity. It did not decide which agents ship first, how the destination session
is actually created, which Grok CLI is the target, or where capsules live.

Stable `v0.3.0` shipped the primitives Phase 4 builds on: a local session index
covering Claude Code, Codex, Gemini CLI, and OpenCode; workspace probing;
environment preflight with a launch policy; capability discovery; and hardened
native launch with executable and workspace identity guards.

Four decisions were needed before implementation could start.

## Decision

### 1. `v0.4.0-rc.1` scope: Claude ↔ Codex as the gate, three source-only readers

Claude Code ↔ Codex CLI structured handoff, **both directions**, is the release
gate. Gemini CLI, OpenCode, and Grok Build ship as capsule **sources** only —
you can hand off from them, not to them. Their destination support is scheduled
for `v0.4.0-rc.2`.

Grok Build additionally gains a local session-index source, so it appears in
`rein sessions`, `rein search`, and `rein inspect`, with native resume and fork
refused.

**Why:** one canonical capsule plus per-agent readers is the whole point of the
architecture; proving five readers against two targets validates the universal
claim without doubling the acceptance matrix. Shipping five targets in rc.1
would put ten directional rows on a physical dual-platform matrix that has
never been run before.

### 2. Launch route: new native session, capsule file, bootstrap prompt

The destination is created by launching the destination vendor's own CLI in the
verified workspace with a bounded bootstrap prompt that points at an
inspectable capsule file:

- Claude Code: `claude --session-id <new-uuid> "<bootstrap>"`
- Codex CLI: `codex "<bootstrap>"`, with the native session ID reconciled after
  launch

Reinstate writes **no vendor-internal files**. It never writes into
`~/.claude/projects`, `~/.codex/sessions`, `~/.gemini/tmp`, `~/.grok/sessions`,
or OpenCode storage.

**Why:** writing another vendor's private transcript store is the highest-risk
action available to this product. It requires exact-version fixtures, backups,
consent, and native-resume validation, and a wrong write corrupts a user's real
session history. The documented-launch route achieves the flagship quota-switch
demo with none of that risk, and it degrades honestly: the capsule file is
inert data the user can read.

Consequence: the destination must be started interactively, and Codex's session
ID is only knowable after the fact. Both are accepted.

### 3. Grok flavor: xAI Grok Build CLI

"Grok" means the official xAI Grok Build CLI: `~/.grok/sessions/`,
`grok --resume <id>`, `grok --continue`, config at `~/.grok/config.toml`.
The community `grok-cli` packages are not targeted.

Grok Build CLI has a documented mid-2026 history of transmitting repository
contents, including Git history and unredacted `.env` material, to xAI cloud
storage. Therefore:

- Grok is **not** a handoff destination in `v0.4.0`;
- redaction is unconditional on the Grok source path, and `--no-redact` is
  refused there; and
- any Grok-sourced handoff prints an explicit warning naming that behavior.

**Why:** the official harness is what users mean by "Grok". The privacy history
is a reason to constrain the integration, not to pretend the agent does not
exist — users already have sessions there and deserve a way out of them.

### 4. Capsule storage: local-only, under `$REINSTATE_HOME/handoffs/`

Capsules, projections, sidecars, and lineage live in a content-addressed store
under `$REINSTATE_HOME/handoffs/`, owner-only, append-only lineage, hard-excluded
from sync. Encrypted BYO sync of capsules is deferred.

The existing session-index SQLite database is explicitly **not** used, because
it is designed to be rebuildable and disposable, and lineage is authoritative
history that must survive an index rebuild.

**Why:** keeping rc.1 offline-testable means the entire Phase 4 acceptance
matrix runs without object storage, credentials, or a passphrase — the same
property that made Phase 2 land cleanly. Adding a sync scope would pull the
manifest, conflict, and encryption surface into a phase that already has a large
new parsing surface.

## Consequences

- The `v0.4.0` compatibility matrix gains directional handoff rows. "Supported
  session adapter" still never implies "supported handoff".
- Marketing and docs describe Gemini, OpenCode, and Grok as **source-only** in
  `v0.4.0`. Any broader claim is a product-truth violation.
- Codex-destination lineage can legitimately report `unresolved` or `ambiguous`.
  That is recorded honestly rather than guessed.
- `reconstructed_conversation` remains a reserved, unused fidelity mode. No code
  path may emit it in `v0.4.0`.
- No new exit codes are introduced; the Phase 1–3 contract is unchanged.
- A future capsule sync scope must be designed against this store's layout, not
  bolted onto the session index.

## Rejected alternatives

### Ship all five agents in both directions in rc.1

Rejected. Ten directional rows across two physical platforms, on a surface with
no prior acceptance history, makes an rc.1 failure near-certain and makes the
failure hard to attribute.

### Ship experimental native transcript reconstruction in rc.1

Rejected for this release. It writes undocumented vendor internals into a user's
real session store. It stays reserved for a post-stable experiment with
exact-version gating, backups, explicit consent, a new destination ID, and
native-resume validation.

### Export-only, with no destination launch

Rejected. It reduces the flagship quota-switch story to a manual copy-paste step
and removes the verification, capability diff, and lineage that make a Reinstate
handoff better than a hand-written summary.

### Reuse the session-index SQLite store for lineage

Rejected. That store is derived, rebuildable, and safe to delete by design.
Lineage is authoritative and must not vanish on a rebuild.

### Skip Grok entirely

Rejected. Users have real Grok sessions and the privacy history is a reason to
give them a documented, redacted exit path — not a reason to leave them stuck.
