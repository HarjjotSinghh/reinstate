# W1 — Contracts and documentation cards

One executor. Starts after T-004, runs beside Wave A.

---

## T-010 — Tier column in the compatibility matrix

**Owns:** `docs/compatibility.md`

**Goal.** Restructure the matrix around the tier ladder so agents can be
appended without a schema change, then hand the file over to append-only mode.

**Steps.**

1. Add a tier column to the capability matrix and fill it for the five shipped
   agents: claude T5, codex T5, gemini T2, opencode T2, grok T2.
2. Document the insertion point for new rows with an HTML comment, so every
   later task appends in the same place.
3. Keep the Phase 4 directional handoff table **exactly** as it is.
   `internal/doctest/phase4_cli_contract_test.go` asserts five rows verbatim,
   including `| **Grok Build** | structured handoff | structured handoff | not in v0.4.0 | not in v0.4.0 | not a target (source-only) |`.
   Changing that table breaks CI and is not part of this phase.
4. Add a short section explaining that "supported" now means a tier, linking
   [../../../agent-support-tiers.md](../../../agent-support-tiers.md).

**Done when.** Doctest passes and the file has a documented append point.

**This task must merge before any agent adds a row.** A later task that needs a
new column is a T-010 change, not a per-agent change.

---

## T-011 — Documentation contracts binding the catalog to the docs

**Owns:** `internal/doctest/agents_contract_test.go`

**Goal.** Make documentation drift a test failure. This is the load-bearing
task of W1: without it, the matrix diverges from the catalog within a release.

**Assertions to implement.**

1. Every catalog agent has a row in `docs/compatibility.md`, and the tier in
   the row equals the tier in the descriptor.
2. Every catalog agent has a storage page, and every descriptor `Evidence` path
   exists.
3. Every agent in the tier table in `docs/agent-support-tiers.md` exists in the
   catalog, and no catalog agent is missing from it.
4. Every T0 agent's reason in the docs matches its `T0Reason`.
5. `docs/cli-reference.md` lists the agent keys each command accepts, matching
   what the catalog produces.
6. No shipped documentation claims a capability above an agent's declared tier.
   Implement as a keyword scan near agent names: "resume", "sync", "push",
   "pull", "handoff to".

**Done when.** The suite fails if you deliberately raise a descriptor's tier
without touching docs, and fails if you add a doc row for an agent that is not
in the catalog.

---

## T-012 — Migrate shipped agents into per-agent storage pages

**Owns:** `docs/session-storage-map.md`, `docs/session-storage/{claude,codex,gemini,opencode,grok}.md`

**Goal.** Finish the split started in the planning branch, so no future agent
work touches a shared 380-line file.

**Steps.**

1. Move sections 1 through 5 of the map into five per-agent pages, **content
   unchanged**. This is a move, not a rewrite. Preserve the R-number research
   notes, the confidence markers, and every source link.
2. Leave in the map: the confidence vocabulary, the cross-OS summary table, the
   reader rules, and the sources list.
3. Update the index in `docs/session-storage/README.md`.
4. Fix every inbound link. `internal/doctest` and several docs reference the
   map's sections.

**Done when.** No content was lost, every link resolves, and doctest passes.

**Do this after Wave A lands**, so the move does not conflict with the new
pages being written.

---

## T-013 — User-facing documentation

**Owns:** `docs/adapters.md`, `docs/README.md`, `README.md`,
`docs/getting-started.md`

**Goal.** Explain tiers where users meet them, without overclaiming.

**Steps.**

1. `docs/adapters.md`: add the tier ladder summary and link the SDK document.
   Keep the existing fail-closed table, which is still correct.
2. `docs/README.md`: add rows for the tier document, the catalog SDK, and the
   probe contract.
3. `README.md`: update the supported-agents section to state tiers rather than
   a flat list. **Do not write "all coding agents" or any absolute claim.**
4. `docs/getting-started.md`: mention `rein doctor --agents` as the way to see
   what Reinstate found and why an agent is missing.

**Copy constraint.** `internal/doctest/phase4_cli_contract_test.go` runs a
per-paragraph classifier over these files that rejects any paragraph implying
cross-agent native identity. Do not put "native resume" and "same session" or
"cross-agent" in one paragraph, even to deny the connection. Split them.
