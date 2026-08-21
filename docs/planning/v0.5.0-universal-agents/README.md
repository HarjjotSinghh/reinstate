# Phase 5 — universal agent coverage (`v0.5.1`)

Planning and execution documents for expanding Reinstate from five coding
agents to a catalog of many, with a capability tier for each and committed
evidence behind every tier.

**Integration branch:** `feat/universal-agent-coverage`

---

## Read in this order

| Document | Purpose |
| -------- | ------- |
| [ADR 0004](../../adr/0004-universal-agent-coverage.md) | Why, and the eight decisions that constrain everything else |
| [Agent support tiers](../../agent-support-tiers.md) | T0 through T5, and the evidence each requires |
| [Agent catalog SDK](../../adapters/agent-catalog-sdk.md) | The descriptor, the scanners, the conformance suite, the recipe |
| [agent-roster.md](agent-roster.md) | Which agents, which target tier, what is unknown |
| [work-breakdown.md](work-breakdown.md) | Eight workstreams and the dependency graph |
| [file-ownership.md](file-ownership.md) | Who may edit what. Read before your first commit |
| [review-gates.md](review-gates.md) | What the coordinator checks before merging |
| [task-cards/](task-cards/) | The tasks themselves |
| [Phase 5 acceptance](../../testing/phase-5-universal-agent-coverage-acceptance.md) | What stable `v0.5.1` must prove |
| [Storage probe](../../testing/agent-storage-probe.md) | How evidence is captured without leaking anything |

---

## The five invariants

Break any of these and the work is wrong regardless of whether it compiles.

1. **Native resume is same-vendor.** Cross-agent work is explicit portable
   handoff. There is no transcript translation at any tier, in any release.
   [AGENTS.md](../../../AGENTS.md) non-negotiable 3.
2. **Evidence before claims.** A tier requires committed probes, fixtures, and,
   at T3 and above, device reports. Vendor documentation alone is never
   sufficient; the Kimi mirrors disagree with each other today.
3. **Read-only below T5.** No task in this phase writes into a vendor's session
   store. Not to fix it, not to migrate it, not to add a session.
4. **Fail closed.** Unknown layout is exit `5`, not a best-effort parse.
5. **No real transcripts, ever.** Fixtures are synthetic. Probes are redacted.
   The secret scanner gates every commit.

---

## Definition of done for the phase

1. The catalog has landed and the five shipped agents are on it with unchanged
   behavior.
2. `rein doctor --agents` ships with the redacted probe contract enforced by a
   test.
3. At least six new agents at T1 with macOS and native Windows fixtures.
4. At least three new agents at T2.
5. Every agent in the roster has a storage page and a descriptor, including
   those that correctly land at T0.
6. Dual-platform Phase 5 acceptance passes and stable `v0.5.1` is authorized.

Criteria 3 and 4 are counts of **evidence**, not of effort. If probes come back
worse than hoped, the release scope shrinks. The tiers do not inflate.

---

## What this phase deliberately does not do

- No new handoff destinations. Claude Code and Codex CLI remain the only
  targets.
- No new synced agents. Claude Code and Codex CLI remain the only agents
  `push` and `pull` carry.
- No configuration support. That is Phase 6, and configuration support is never
  inferred from session support.
- No runtime plugin system for third-party agents. In-tree descriptors with a
  conformance suite give contributors the same extensibility without a new
  trust boundary.

---

## Outcomes that look like failure and are not

An executor who reports one of these has done the job correctly:

- "This agent keeps its history on a server. It ships at T0 with reason
  `server_backed`."
- "The local directory is a cache of account state. Indexing it would produce
  records that vanish. T0."
- "MiniMax is a model consumed through other harnesses, not a harness. There is
  nothing to add."
- "The vendor documentation is wrong. The probe found a different root, and
  here is the corrected page."
- "The evidence supports T1, not the T3 the roster targeted."

The failure mode this phase must avoid is an executor stretching to reach a
target tier. A wrong tier claim ships a promise the product cannot keep, and
users discover it at the worst possible moment.
