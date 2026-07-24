# Roadmap

> Status legend: ✅ done · 🚧 in progress · 📋 planned · 💭 exploring · ❌ won't do (for now)

Last updated: **2026-07-25** · Maintainer: [Harjot Singh Rana](https://github.com/HarjjotSinghh)

This roadmap is a living document. Priorities follow real activation signals
(especially first successful cross-device resume) and vendor format churn.

---

## Vision

> **Reinstate is a local-first, encrypted sync layer for AI coding agent state
> across the machines you own.** Phase 1 ships same-vendor **session** sync for
> Claude Code and Codex. Broader environment sync (MCP, skills, settings) and
> additional agents come after a trustworthy `v0.1.0`.

We do **not** aim to be:

- Dropbox for raw agent trees without path intelligence
- A vendor-locked cloud for a single agent
- A real-time multiplayer agent runtime
- A cross-agent session *translator* (Claude → Codex replay)

---

## Phase 0 — Foundation 🚧

**Gate:** contracts, diagnostics, installers, fixtures, CI/release trust, and
docs are honest and verified. No claim of end-to-end session resume yet.

| Item | Status |
| ---- | ------ |
| Authority docs + ADR + compatibility matrix | 🚧 |
| CLI routing, exit codes, version JSON | 📋 |
| Versioned config/state + atomic writes | 📋 |
| Device detection (macOS / Windows / WSL2) | 📋 |
| Redacted `doctor` + synthetic self-test | 📋 |
| Synthetic fixtures + secret scanner | 📋 |
| Hard CI gates + goreleaser snapshot | 📋 |
| Checksum-verifying installers | 📋 |
| Versioned AI-agent setup prompts | 📋 |

## Phase 1 — Claude + Codex sessions (`v0.1.0`) 📋

**Gate:** a stranger can install, `init`, and resume a cross-OS Claude Code
session; Codex follows the same path. Encryption on; credentials never synced.

| Item | Status |
| ---- | ------ |
| R2/S3-compatible backend + memory test double | 📋 |
| Credentials / interactive `init` | 📋 |
| age passphrase envelopes | 📋 |
| Project identity + path mapping | 📋 |
| Manifests, push/pull, conflicts | 📋 |
| Atomic restore + backups + locks | 📋 |
| Claude Code adapter (detect/export/restore) | 📋 |
| Codex adapter (detect/export/restore) | 📋 |
| Complete CLI + human docs + release candidate | 📋 |

### SemVer progression toward `v0.1.0`

```text
v0.1.0-alpha.*  foundation + installers + prompts
v0.1.0-beta.*   Claude then Codex end-to-end
v0.1.0-rc.1     release candidate
v0.1.0          Phase 1 stable
```

## Phase 2 — Broader environment 📋

**Gate:** fresh machine bootstrap restores sessions *and* selected config
scopes after Phase 1 is stable.

| Item | Status |
| ---- | ------ |
| MCP servers, skills, instruction files | 📋 |
| Scopes: `--scope sessions \| config \| all` | 📋 |
| Gemini CLI adapter | 📋 |
| OpenCode adapter | 📋 |
| Additional backends (WebDAV, GCS) | 📋 |

## Phase 3 — Habit & trust 💭

| Item | Status |
| ---- | ------ |
| Shell hooks (pull on start / push on exit) | 💭 |
| Grok Build / Cursor adapters (when formats stabilize) | 💭 |
| Device registry / key rotation helpers | 💭 |
| Hosted zero-knowledge convenience layer | 💭 |

## Explicit non-goals (near term) ❌

| Item | Why |
| ---- | --- |
| Cross-agent session translation | Formats/tool schemas differ; resume is same-vendor |
| Multi-tenant real-time CRDT collab | Sequential dual-machine use is the dominant pattern |
| Replacing git | Git remains source truth; Reinstate is context truth |
| Shipping vendor API keys or auth proxies | Local-only file access; credentials never synced |

---

## Stable release policy

- **Pre-1.0:** minor versions may include breaking CLI/config changes (documented in CHANGELOG)
- **`v0.1.0` criteria:** Claude + Codex session sync, path remapping Windows ↔ macOS,
  security model enforced in tests, install → first resume under 5 minutes on clean machines
- Releases: GitHub Releases + checksums + SBOM; see [RELEASING.md](RELEASING.md)

## How to propose roadmap items

1. Open a feature request issue
2. Tag with `roadmap`
3. Discuss trade-offs (maintenance cost of adapters is real)
