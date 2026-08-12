# Phase 4 research — R7 context-window ceilings

**Date:** 2026-08-12  
**Branch:** `wp/13-capability-diff`  
**Resolves:** R7 (WP-13)  
**Rule:** if vendor docs cannot confirm a harness token ceiling, publish
`omitted` with a reason — never guess a model marketing number.

---

## R7 — Claude / Codex context-window ceilings for the capability diff

**Status: omitted (informational)**

Reinstate’s capability diff publishes a destination **harness** context
ceiling when a vendor documents one that is independent of whichever model
the user selected. That value would appear in
`capsule.CapabilityDiff` summaries as `context_ceiling` (token count) and,
when the destination ceiling is lower or unknown relative to a published
source ceiling, as `Missing{Kind: "context", Name: "ceiling", …}`.

### Findings

| Agent | Published harness token ceiling | Confidence | Evidence |
| ----- | ------------------------------- | ---------- | -------- |
| Claude Code | **None recorded** | Omitted | Claude Code session / headless docs used by Reinstate (`docs/session-storage-map.md` sources) describe session layout and launch argv, not a fixed CLI-owned context-token ceiling. Model context sizes vary by selected model and are not a stable harness constant. |
| Codex CLI | **None recorded** | Omitted | Codex CLI docs / session lifecycle notes used by Reinstate describe rollout storage and resume/fork, not a fixed CLI-owned context-token ceiling. |

### Capability-diff behavior (WP-13)

For both `claude` and `codex`:

- `context_ceiling`: `"omitted"`
- `context_ceiling_reason`: `no_vendor_published_harness_token_ceiling`
- No invented `Missing` context gap when **both** sides are omitted

If a future vendor doc publishes a harness-level ceiling (not a per-model
marketing figure), record it here with the source URL, then wire the number
into `publishedProfile` in `internal/handoff/capabilitydiff.go`.

### Non-goals

- Do not copy Anthropic / OpenAI **model** context sizes into the handoff
  capsule as if they were CLI harness limits.
- Do not treat projection budget (`Policy` / 64 KiB / 2 MiB) as a vendor
  context ceiling; that is Reinstate’s own projection contract (WP-14).
