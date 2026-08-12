# Phase 4 research — R6 Codex argv length ceiling on Windows

**Date:** 2026-08-12  
**Branch:** `wp/17-codex-target`  
**Resolves:** R6 (WP-17)  
**Rule:** if vendor docs cannot confirm an exact Windows argv ceiling for Codex
CLI initial prompts, publish Reinstate’s conservative budget as
Documented/Unverified — never invent a vendor-specific Windows limit.

---

## R6 — Codex initial-prompt argv behavior and Windows length ceiling

### Initial prompt (Documented)

Codex CLI accepts a new-session initial prompt as a positional argument:

```text
codex "<bootstrap>"
```

Working directory is the verified workspace. Codex assigns the native session
ID; Reinstate reconciles it after launch (ADR 0003). Reinstate does **not**
write vendor-internal files under `~/.codex`.

Evidence: [session-storage-map.md](../session-storage-map.md) §2 (Initial
prompt row); OpenAI Codex CLI launch surface used by Reinstate’s Phase 2–3
native launch plans.

### Practical Windows argv ceiling (Documented / Unverified)

| Claim | Confidence | Value |
| ----- | ---------- | ----- |
| Codex documents a Windows-specific initial-prompt byte ceiling | **Unverified** | none found in vendor docs Reinstate trusts |
| Reinstate handoff destination argv budget | **Documented** (architecture) | `TargetCapabilities.MaxArgvBytes` defaulting to `DefaultMaxArgvBytes` (**24 KiB**) |

`DefaultMaxArgvBytes` (`24 << 10`) is Reinstate’s conservative Windows-safe
ceiling from the Phase 4 architecture plan — chosen so destination argv is
validated before launch and Windows never truncates silently. It is **not** a
Codex-published constant and must not be described as one.

When a planned `codex "<bootstrap>"` argv exceeds `MaxArgvBytes`, WP-17 falls
back to a shorter bootstrap that references `projection.md` only. If even the
short form exceeds the budget, planning fails closed with
`ErrArgvExceedsBudget`.

### Non-goals

- Do not treat the classic `cmd.exe` 8191-character limit, or the
  `CreateProcess` 32767-character limit, as a Codex-documented product
  ceiling without a vendor citation.
- Do not raise `DefaultMaxArgvBytes` on speculation from community reports.
