# Naming — Reinstate (decided)

## Decision

| Field | Value |
|-------|--------|
| **Product name** | **Reinstate** |
| **Status** | **Chosen** (user + ChatGPT consensus over Carryover) |
| **Domain** | `reinstate.dev` — **acquire/acquired** (recommended and confirmed in thread) |
| **CLI** | `reinstate` |
| **Primary pitch** | *Reinstate restores your coding-agent workspace on any device.* |

This is **not** an open naming exercise anymore. DevSync, Rethra, Carryover, etc. are historical candidates or lineage — not the product name.

---

## Why Reinstate won

### Semantic fit

Dictionary sense: restore something to a previous effective state.

Product sense (full stack, not just “move chat”):

> Restore the session, context, tools, configuration, and workspace to a usable state on another device.

### vs Carryover (final shortlist)

| Quality | Carryover | Reinstate |
|---------|----------:|----------:|
| Memorability | 9/10 | 7.5/10 |
| Product accuracy | 8/10 | **9.5/10** |
| Platform breadth | 8/10 | **9/10** |
| Developer-tool credibility | 7.5/10 | **8.5/10** |
| Distinctiveness | 6/10 | 7/10 |
| Verbal personality | Friendly | Serious |
| Final score | 7.8/10 | **8.5/10** |

**Carryover problems:**

- Strong HR/payroll “leave balance” associations  
- Existing productivity app **Carryover** (capture → delivery)  
- Crowded “carry context forward” space (e.g. CarryForward)

**Reinstate strengths:**

- Authoritative, precise, platform-wide  
- Works as a **verb** in product language  
- Clean CLI surface  

### Example language

```bash
reinstate init
reinstate sync
reinstate list
reinstate resume
reinstate doctor
reinstate status
```

- “Reinstate this session on my MacBook.”  
- “Your workspace has been reinstated.”  
- “Three capabilities are missing before this session can be reinstated.”  
- “Reinstate supports Claude Code, Codex, and Gemini.”  

---

## Known collisions & legal notes

| Collision | Risk | Action |
|-----------|------|--------|
| **reinstate.app** — Shopify suspension risk / appeals product | Different market; same English word | Trademark clearance before heavy brand spend; coexistence often OK across classes |
| Dictionary / legal “reinstatement” (accounts, employment, USPTO, GitHub appeal) | Connotation only | Tagline teaches product meaning |
| **Devsynq / devsync repos** | Confusion if product still called DevSync | **Avoid** — product is Reinstate |
| npm/PyPI/GitHub handle availability | Variable | Reserve `reinstate`, `reinstate-dev` where possible |

Owning the domain ≠ trademark clearance. ChatGPT advice: purchase domain now; complete legal search before major public launch.

### Handles to reserve (recommended)

```text
github.com/reinstate
github.com/reinstate-dev
npm: reinstate
PyPI: reinstate
X: @reinstate / @reinstatedev
```

Unavailable socials should not block domain use.

---

## Rejected / secondary candidates (for history only)

Do **not** re-litigate unless legal forces a rename.

| Name | Notes |
|------|--------|
| **Carryover** | Final runner-up; warmer, less precise |
| **Rethread / Rethra** | Thread metaphor; Rethread product collision; Rethra coined |
| **Reboard** | Weaker semantics |
| **Remount** | Snappy; filesystem association; npm package baggage |
| **Restate** | Shorter; weaker “restore full state” |
| **Statebraid, Knotra, Synknot, CarryState, Threadcarry, Rebranch, Statepath, Layover…** | Brainstorm leftovers |
| **DevSync / Dev Synq** | Prior MCP product idea; market collision; saturated category signal |

---

## Relationship to “DevSync”

- **DevSync** = original MCP-config-sync concept  
- Research often says “expand DevSync” *product-wise*  
- **Public product name for that expansion is Reinstate**  
- In docs: “DevSync heritage” / “capability plane” is fine; shipping brand is **Reinstate**

---

## One-line brand kit

| Item | Copy |
|------|------|
| Name | Reinstate |
| Domain | reinstate.dev |
| Tagline | Restores your coding-agent workspace on any device |
| Category | Agent continuity / portable agent workspace control plane |
| Not | Cloud chat storage for terminals |
