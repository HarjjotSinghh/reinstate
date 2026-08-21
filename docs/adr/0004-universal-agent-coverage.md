# ADR 0004: Universal agent coverage, the support-tier ladder, and the agent catalog

## Status

Accepted for planning. Implementation planned for `v0.5.1` (Phase 5).
Extends [ADR 0003](0003-phase-4-rc1-scope-and-launch-route.md), which fixed the
Phase 4 scope at five source readers and two destinations. This ADR decides how
Reinstate scales past those five agents without turning "universal" into a
claim it cannot defend.

## Context

Stable `v0.4.0` supports five coding agents unevenly: Claude Code and Codex CLI
have full encrypted sync, verified resume, and bidirectional structured
handoff; Gemini CLI, OpenCode, and Grok Build have local discovery and
handoff-source reading only.

Users run far more than five agents. The requested roster is Grok, Gemini CLI,
OpenCode, ZCode (Z.ai), Kimi Code CLI, MiniMax, and Pi, plus the other majors:
Cursor CLI, GitHub Copilot CLI, Aider, Cline/Roo Code, Amp, Qwen Code, and
OpenHands.

Three problems block that expansion.

**1. Support is not a boolean.** "Supports agent X" currently means five
different things depending on which subsystem you ask. Docs work around this
with prose qualifiers per agent per phase, which does not survive fifteen
agents.

**2. Registration is scattered.** Adding one agent today means editing at
least five shared files:

| Concern | Shared registration site |
| ------- | ------------------------ |
| Encrypted sync | `defaultRegistry()` in `internal/cli/commands_impl.go` |
| Local index | `defaultLocalSources()` in `internal/cli/sessions.go` |
| Handoff source | `init()` in `internal/transcript/<agent>.go` |
| Handoff destination | `RegisterTarget()` in `internal/handoff/target_*.go` |
| Version probe | `definitions` map in `internal/agentcheck/agent.go` |

Plus agent constants in `internal/sessionindex/model.go`, the process switch in
`internal/processcheck/process.go`, the CLI validators `validateLocalAgent` and
`validateNativeAgent`, and the default enablement list in
`internal/schema/config.go`. Every one of those is a single file that every
agent must touch. Adding ten agents in parallel means ten writers per file.

**3. Storage layouts are unproven.** Vendor documentation for the new roster is
thin, mirrored inconsistently, and sometimes wrong. Kimi Code CLI is the
live example: one vendor docs mirror states the data root is `~/.kimi-code/`
while another states `~/.kimi/`, with different session subtree shapes. Neither
may be written into Reinstate as fact.

## Decision

### 1. Support is a tier, not a boolean

Every agent in the catalog carries exactly one tier per capability ladder rung
it has reached. Tiers are cumulative and each has its own evidence gate.

| Tier | Name | What the user can do |
| ---- | ---- | -------------------- |
| **T0** | Known | Reinstate names the agent and explains why it is not usable |
| **T1** | Discover | `rein sessions`, `rein search`, `rein inspect` |
| **T2** | Handoff source | `rein handoff --from <agent>` produces a capsule |
| **T3** | Verified resume | `rein resume` / `rein fork` launch the vendor's own session |
| **T4** | Handoff destination | `rein handoff --to <agent>` starts a new session there |
| **T5** | Encrypted sync | `rein push` / `rein pull` carry that agent's sessions |

`v0.4.0` in this vocabulary: Claude Code and Codex CLI are T5; Gemini CLI,
OpenCode, and Grok Build are T2.

The tier is the product claim. Public surfaces state the tier and the evidence
behind it, never a bare "supported". Full definitions and per-tier evidence
gates live in [../agent-support-tiers.md](../agent-support-tiers.md).

**Why:** a ladder makes partial support honest and shippable. Without it, every
new agent forces a binary choice between overclaiming and shipping nothing, and
the compatibility matrix becomes prose no test can lock.

### 2. One catalog, one file per agent

A new package `internal/agents` holds a catalog. Each agent is one file,
`internal/agents/catalog/<agent>.go`, that constructs a descriptor and
registers it from `init()`. There is no shared slice, map literal, or switch
statement that every agent must edit.

The five existing registration sites become **derived consumers** of the
catalog rather than independent sources of truth. `defaultRegistry()`,
`defaultLocalSources()`, `agentcheck.definitions`, the transcript reader
registry, and the handoff target registry are all populated by iterating the
catalog and selecting agents whose descriptor declares that capability.

The descriptor carries identity, tier, storage family, discovery configuration,
version range, native argv, and the evidence references that justify its tier.
The full manifest specification is
[../adapters/agent-catalog-sdk.md](../adapters/agent-catalog-sdk.md).

**Why:** this is the difference between a feature one person ships over months
and a feature a parallel team ships in weeks. One file per agent means N
executors can work simultaneously with zero merge conflicts. It also makes the
tier claim mechanically checkable: a descriptor that claims T3 without a
version range fails a conformance test rather than a review.

Consequence: the catalog refactor is a prerequisite for all per-agent work and
must land first, alone, with no behavior change to the existing five agents.

### 3. Storage layouts require device evidence, never vendor prose alone

No storage path enters shipped code from documentation alone. Promotion to T1
requires a **redacted device probe** captured on a real machine, plus synthetic
fixtures under `testdata/`.

Reinstate ships the probe itself: `rein doctor --agents --json` enumerates
every catalog agent, reports whether its declared roots exist, and emits
counts, relative shapes, and extensions. It never emits transcript content,
prompt text, file contents, absolute paths outside the agent root, or
usernames. `scripts/testing/agent-storage-probe.sh` and its PowerShell twin
wrap it for contributors who do not have a Go toolchain.

Until a probe lands, the agent's rows in
[../session-storage-map.md](../session-storage-map.md) stay `Unverified` and
the agent stays at T0. This is the existing confidence vocabulary, applied to a
larger roster.

**Why:** the alternative is shipping a scanner that reads the wrong directory,
finds nothing, and reports "no sessions" to a user who has hundreds. That
failure is silent, looks like a Reinstate bug, and is indistinguishable from an
agent the user has not used. The Kimi documentation conflict proves the risk is
real rather than theoretical.

### 4. Per-agent storage documentation moves to its own file

[../session-storage-map.md](../session-storage-map.md) becomes an index plus
the cross-OS summary. Each agent's detail moves to
`docs/session-storage/<agent>.md`.

**Why:** the map is a 380-line single file that every agent must edit. Under
parallel execution it is the single worst merge conflict in the repository.
Splitting it costs one indirection and removes the conflict entirely.

### 5. `v0.5.1` scope: breadth at T1 and T2, no new destinations, no new sync

The `v0.5.1` release gate is:

1. the catalog refactor lands with the existing five agents at unchanged tiers,
   proven by the existing test suite passing without modification of assertions
   about behavior;
2. `rein doctor --agents` ships with the redacted probe contract;
3. at least six new agents reach **T1** with dual-platform fixtures; and
4. at least three new agents reach **T2**.

**T4 and T5 do not expand in `v0.5.1`.** Claude Code and Codex CLI remain the
only handoff destinations and the only synced agents.

T3 is per-agent and evidence-driven. Kimi Code CLI and Pi are the primary
candidates because both document an explicit session-directory override and an
explicit resume argv. Neither is promised; a missing version probe blocks T3.

**Why:** ADR 0003 rejected shipping five destinations at once because ten
directional rows on an unproven dual-platform matrix makes failure near-certain
and hard to attribute. That reasoning holds with more force at fifteen agents.
Writing another vendor's private transcript store, or restoring into it from
sync, remains the highest-risk action in the product. Breadth at the read tiers
delivers the universal index and the universal exit path, which is what users
actually asked for, without touching the risky end of the ladder.

### 6. The same-vendor boundary is unchanged

[../../AGENTS.md](../../AGENTS.md) non-negotiable 3 stands exactly as written.
Native resume stays same-vendor. Cross-agent work stays explicit portable
handoff. `reconstructed_conversation` remains reserved and unemitted.

"Universal sync layer" means: one index across every supported agent, one
search, one handoff format, one encrypted store for the agents at T5. It does
not mean translating a Claude transcript into a Codex transcript, and no
surface may imply otherwise.

**Why:** breadth increases the temptation to claim seamlessness, and breadth is
exactly when that claim becomes least true. Fifteen agents with divergent
formats make lossless translation less achievable, not more.

### 7. Roadmap renumbering

Phase 5 becomes **universal agent coverage**. The former Phase 5 (universal
configuration and automatic cross-device sync) becomes Phase 6, Console becomes
Phase 7, and team continuity becomes Phase 8.

**Why:** universal configuration renders declared state into each harness, so
it needs a catalog of harnesses to render into. Agent coverage is its
prerequisite, and the phase numbers should reflect the build order.

### 8. Only officially distributed harnesses are catalog agents

An agent qualifies when its vendor distributes the harness and documents its
behavior. Community re-packagings of a vendor runtime do not qualify, and
neither do forks that merely rename an upstream CLI.

ZCode is the live case: Z.ai distributes a desktop ADE, while the terminal
client on npm is an unaffiliated package that extracts the desktop runtime.
Reinstate targets the Z.ai-distributed artifact. If it has no local session
tree, ZCode stays at T0 with that reason recorded, exactly as ADR 0003 handled
the Grok CLI flavor question.

**Why:** an unofficial re-packaging can change layout or vanish without notice,
and pinning a fail-closed version range against it is meaningless. Naming the
agent at T0 with an honest reason serves the user better than a fragile reader.

## Consequences

- The compatibility matrix gains a tier column. "Indexed" no longer implies
  "resumable", and neither implies "synced". Existing wording that says
  "supported adapter" must be re-read against the ladder.
- `rein doctor` grows an agent-inventory surface, which becomes the primary
  support-triage tool and the contributor evidence tool at the same time.
- Per-agent docs live in `docs/session-storage/`. The map file keeps the
  confidence vocabulary, reader rules, and cross-OS summary.
- Agents at T0 are visible in `rein doctor` output with a reason. Users learn
  that Reinstate knows the agent exists and why it cannot help yet, instead of
  concluding the agent is unsupported by omission.
- The catalog refactor is a large mechanical change to five hot files. It ships
  as its own reviewed PR with no functional change, or it will be impossible to
  review against the per-agent work.
- Phase numbering changes in `ROADMAP.md`, the website roadmap page, and any
  test asserting phase content.

## Rejected alternatives

### Add each agent by editing the existing five registration sites

Rejected. Fifteen agents times five shared files is a merge-conflict wall, and
it makes parallel execution impossible. It also leaves the tier claim
unenforceable: nothing stops a scanner registration from existing while the
docs claim resume.

### Ship a plugin system so third parties add agents at runtime

Rejected for `v0.5.1`. A runtime plugin surface means executing third-party
code that reads session stores, which contradicts the security model and
non-negotiable 5. In-tree descriptors with a conformance suite deliver the same
extensibility for contributors without a new trust boundary.

### Infer session layout at runtime by scanning the home directory

Rejected. Walking a user's home directory looking for anything that resembles a
transcript is a privacy hazard, unbounded, and produces false positives that
are worse than a missing agent. Declared roots with explicit env overrides stay
the contract.

### Claim T1 from vendor documentation without a device probe

Rejected. The Kimi mirrors disagree with each other today. Documentation is
sufficient to write an `Unverified` row and to schedule a probe; it is never
sufficient to ship a reader.

### Promote several agents to handoff destinations in `v0.5.1`

Rejected, for the reason ADR 0003 gave and for one more: destination support
requires a bootstrap contract and a capability diff per agent, and the physical
acceptance matrix grows with the product of sources and destinations. Breadth
at read tiers and depth at write tiers are separate releases.

### Keep the roadmap numbering and fold coverage into the existing Phase 5

Rejected. Universal configuration has its own gate, its own adapter contract,
and its own risks. Merging two phases produces one phase that cannot be
declared done.
