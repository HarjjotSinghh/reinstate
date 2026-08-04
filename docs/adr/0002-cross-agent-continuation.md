# ADR 0002: Cross-agent continuation is a core explicit handoff

## Status

Accepted product and architecture direction; implementation planned for Phase 4.

## Context

Developers regularly switch between Claude Code, Codex, Gemini CLI, OpenCode,
Grok Build, and other coding agents because of account usage windows, outages,
different strengths, or preference. When a quota expires mid-task, Git preserves
the code but not the latest user intent, rejected approaches, tool evidence, or
conversation state.

Reinstate already planned portable cross-agent checkpoints, but the roadmap did
not make the quota-switch scenario or the required fidelity architecture
explicit. It also left room for readers to interpret “handoff” either as a thin
summary or as perfect native transcript translation.

Vendor session stores and tool/message contracts differ. Some state is hidden,
opaque, credential-bound, unavailable, or meaningful only inside the source
harness. A byte-for-byte copy cannot become a semantically identical foreign
session merely by changing JSON field names.

## Decision

1. Cross-agent continuation is a **core Reinstate capability** in Phase 4.
2. The primary acceptance case is Claude Code ↔ Codex continuation after the
   source reaches a usage limit. The source agent/model need not be callable.
3. Same-vendor continuation remains native resume and keeps the highest-fidelity
   path.
4. Cross-agent continuation always creates an explicit portable handoff and,
   by default, a new destination-native session with recorded lineage.
5. The handoff uses a versioned continuity capsule containing task state,
   normalized visible conversation evidence, workspace truth, capability
   differences, security/redaction metadata, and a component-level fidelity
   report.
6. User messages, visible assistant messages, and portable tool evidence should
   be retained where possible. Tool calls are never replayed as actions.
7. Source system/developer messages are audit history, not destination policy.
   Credentials, approvals, hidden reasoning, and unavailable system state do not
   transfer.
8. Reconstructed target-native histories may be built only for explicitly
   supported source/target/version pairs and remain labeled experimental until
   they pass dedicated continuation and security gates.
9. Reinstate continues to own capture, verification, projection, launch, and
   lineage—not the destination agent loop. This decision does not authorize a
   Reinstate-owned IDE, terminal, model router, or agent runtime.

Detailed design and release gates live in
[cross-agent-continuation.md](../cross-agent-continuation.md).

## Consequences

- Product language distinguishes **same task** from **same native session**.
- Phase 1 scope and its same-vendor guarantees do not change.
- Session adapters gain separate, directional transcript-source and
  handoff-target support states; “can sync” does not imply “can hand off.”
- Architecture needs an immutable raw artifact, canonical capsule, destination
  projection, fidelity report, and lineage graph.
- Security review must treat imported transcript content as untrusted and inert.
- Claude ↔ Codex structured handoff ships before broader agent coverage or
  reconstructed native histories.
- Marketing cannot claim all system messages, hidden reasoning, credentials,
  approvals, or live process state survive a cross-vendor move.

## Rejected alternatives

### Promise perfect native transcript translation

Rejected because vendor roles, tools, hidden state, signatures, policies, and
session lookup rules differ. It would create an unverifiable security and
reliability claim.

### Ship only a prose summary

Rejected as the long-term design because it discards inspectable conversation
evidence and provenance. A structured checkpoint is the default projection, but
the capsule also preserves portable visible events and a raw encrypted source
reference.

### Build a Reinstate-owned agent harness

Rejected for the current roadmap. Existing agents should execute the task.
Reinstate remains the continuity layer around them.

### Require the source model to summarize itself

Rejected as a hard dependency because the motivating case is source quota
exhaustion or outage. Agent-assisted summaries may be optional, but a
deterministic local fallback is mandatory.
