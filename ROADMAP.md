# Roadmap

> Status legend: ✅ done · 🚧 in progress · 📋 planned · 💭 exploring · ❌ won't do (for now)

Last updated: **2026-07-25** · Maintainer: [Harjot Singh Rana](https://github.com/HarjjotSinghh)

This roadmap is a living document. Priorities shift based on real usage signals
(especially "first successful cross-device resume" activation) and vendor format
churn. Open an issue or discussion to influence it.

---

## Vision

> **Reinstate is the sync layer for your entire AI development environment —
> sessions, MCP servers, skills, and settings — across every coding agent and
> every machine you own, encrypted so only you can read it.**

We do **not** aim to be:

- Dropbox for raw `~/.claude` trees without path intelligence
- A vendor-locked cloud for a single agent
- A real-time multiplayer agent runtime
- A cross-agent session *translator* (Claude → Codex replay)

---

## Phase 0 — MVP (v0.1) 🚧

**Gate:** a stranger can install, `init`, and resume a cross-OS Claude Code
session in under five minutes; Codex follows the same path.

| Item | Status |
| ---- | ------ |
| CLI skeleton: `version`, `init`, `push`, `pull`, `status`, `diff`, `conflicts` | 🚧 |
| Interactive init wizard (backend, passphrase, scope, path_map) | 📋 |
| Claude Code adapter (JSONL sessions + path remapping) | 📋 |
| Codex adapter (rollouts + state index awareness) | 📋 |
| age encryption (passphrase-derived keys) | 📋 |
| S3-compatible backend (R2 first) | 📋 |
| Atomic restore + timestamped backups | 📋 |
| Conflict detection (never silent overwrite) | 📋 |
| Docs + release automation | 🚧 |

## Phase 1 — Universal environment (v0.2–0.3) 📋

**Gate:** fresh machine bootstrap restores sessions *and* MCP/skills/config.

| Item | Status |
| ---- | ------ |
| Gemini CLI adapter | 📋 |
| OpenCode adapter | 📋 |
| Config scope: MCP servers, skills, agents, instruction files | 📋 |
| Scopes: `--scope sessions \| config \| all` | 📋 |
| WebDAV + GCS backends | 📋 |
| GitHub Gist backend (size-limited) | 📋 |

## Phase 2 — Habit & trust (v0.4–0.5) 📋

**Gate:** two weeks of unattended auto-sync on dual machines with zero manual
intervention and zero unresolved conflicts.

| Item | Status |
| ---- | ------ |
| Shell hooks (pull on start / push on exit) | 📋 |
| Grok Build adapter (once format stabilizes) | 📋 |
| Cursor transcript adapter (read-only / best-effort) | 💭 |
| Append-aware delta / CAS chunking for large Codex histories | 📋 |
| Opt-in secret redaction pass | 📋 |
| Device registry / key rotation helpers | 📋 |

## Phase 3 — Convenience layer 💭

| Item | Status |
| ---- | ------ |
| Hosted zero-knowledge relay + blob store (paid convenience) | 💭 |
| Local web session browser / library UI | 💭 |
| Optional team share of selected sessions | 💭 |
| Mobile companion (view-only) | 💭 |

## Explicit non-goals (near term) ❌

| Item | Why |
| ---- | --- |
| Cross-agent session translation | Formats/tool schemas differ; resume is same-vendor |
| Multi-tenant real-time CRDT collab | Sequential dual-machine use is the dominant pattern |
| Replacing git | Git remains source truth; Reinstate is context truth |
| Shipping vendor API keys or auth proxies | Local-only file access |

---

## Stable release policy

- **Pre-1.0:** minor versions may include breaking CLI/config changes (documented in CHANGELOG)
- **1.0 criteria (target):**
  - Claude Code + Codex + Gemini adapters stable for ≥2 major vendor releases
  - Path remapping proven Windows ↔ macOS in production use
  - Security model documented and externally reviewable
  - Install → first resume under 5 minutes on clean machines
- Releases: GitHub Releases + checksums; see [RELEASING.md](RELEASING.md)

## How to propose roadmap items

1. Open a feature request issue
2. Tag with `roadmap`
3. Discuss trade-offs (maintenance cost of adapters is real)

Stars and issues both help prioritize — thank you for using Reinstate.
