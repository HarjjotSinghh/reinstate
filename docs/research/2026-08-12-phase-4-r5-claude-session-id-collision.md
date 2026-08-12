# Phase 4 research — R5 Claude `--session-id` collision policy

**Date:** 2026-08-12  
**Branch:** `wp/16-claude-target`  
**Resolves:** R5 (WP-16)  
**Rule:** if vendor docs cannot confirm collision-on-reuse behavior, refuse
after bounded regenerations and escalate — never guess silent overwrite.

---

## R5 — Claude Code `--session-id` when the UUID already exists

**Status: unknown (fail closed)**

| Claim | Confidence | Evidence |
| ----- | ---------- | -------- |
| `claude --session-id <uuid>` pins a new session ID | **Documented** | ADR 0003 launch route; Claude Code CLI accepts a UUID session id for a new interactive session |
| Behavior when `<uuid>` already names an on-disk / indexed session | **Unverified** | No vendor doc Reinstate trusts states whether Claude resumes, appends, refuses, or overwrites |

### Reinstate policy (WP-16)

1. Generate candidate IDs with `crypto/rand` UUID v4.
2. Refuse any candidate that already appears as an **indexed** Claude session.
3. Regenerate up to **8** times.
4. If every attempt collides, return `ErrClaudeSessionIDCollision` and escalate
   in the handoff report — **do not launch**, and **do not** write under
   `~/.claude/projects`.

Silent overwrite is never assumed.

### Implementation

`internal/handoff/target_claude.go` — `ClaudeTarget.Plan` + `SessionExists`.
