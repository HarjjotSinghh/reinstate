---
title: "How Reinstate syncs coding-agent sessions"
description: "Understand Reinstate's adapters, structural path remapping, age encryption, immutable snapshots, conflict handling, and S3-compatible storage architecture."
order: 2
updatedAt: 2026-07-27
tags: ["architecture", "session-sync", "encryption", "path-remapping", "s3"]
targetQuery: "how Reinstate works"
searchIntent: "evaluation"
draft: false
noindex: false
---

Reinstate uses per-agent adapters, structural path remapping, client-side age
encryption, immutable snapshots, and an encrypted remote manifest to move
vendor-native sessions without becoming a coding harness.

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│ Claude Code │   │    Codex    │   │ Future agents│  ...
│  ~/.claude  │   │  ~/.codex   │   │  (roadmap)   │
└──────┬──────┘   └──────┬──────┘   └──────┬──────┘
       │                 │                 │
       ▼                 ▼                 ▼
┌──────────────────────────────────────────────────┐
│              Agent Adapters (per tool)            │
│  locate · parse · exclude secrets · project IDs   │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Normalizer                                       │
│  structural path tokens · project IDs             │
│  credential hard-excludes                         │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Encryption (age / scrypt passphrase recipient)   │
│  client-side only — remote never sees plaintext   │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Sync Engine                                      │
│  immutable snapshots · encrypted JSON manifest    │
│  conflict detection · atomic restore + backup     │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Backends: R2 · S3-compatible (Phase 1)           │
└──────────────────────────────────────────────────┘
```

![Reinstate MVP architecture](/brand/05_architecture.png)

## Design principles

1. **Local-first** — agents remain the sole executors of sessions; Reinstate
   relocates files, it does not re-interpret or re-run them.
2. **Zero-knowledge remote** — only ciphertext on object storage.
3. **Native resume is same-vendor** — restore puts bytes where `claude --resume` /
   `codex resume` already know how to read them.
4. **Cross-agent handoffs are explicit roadmap work** — never silently
   translate native transcripts between vendors.
5. **Fail-safe conflicts** — never overwrite; fork and surface.
6. **Adapter isolation** — format churn in one agent cannot break others.
7. **Normalize configuration intent** — later configuration adapters render a
   canonical desired-state profile into verified native harness formats.
8. **Secrets stay local** — profiles contain references, never raw API keys,
   OAuth tokens, cookies, or vendor credential stores.

## Pipeline stages

### 1. Adapters

Each adapter knows:

| Concern | Example (Claude Code) |
| ------- | --------------------- |
| Root path | `~/.claude/projects/` |
| Session format | Append-only JSONL |
| Project key | Munged absolute path directory name |
| Resume entry | `claude --resume [id]` |
| Exclude globs | plugins, caches, credentials |

Adapters implement a small Go interface under `internal/adapter/`.

### Configuration adapters (roadmap)

Session and configuration support are separate. Later configuration adapters
will import, diff, and render supported MCP servers, skills/instructions,
hooks/loops, plugins, marketplaces, and safe settings:

```text
native config ↔ configuration adapter ↔ Reinstate desired state
                                             ↕ encrypted sync
                                      another device
```

They must preserve unrelated settings, report unsupported/lossy mappings,
preview and back up changes, write atomically, and fail closed on unverified
schemas. See
[Universal agent configuration](universal-configuration.md).

### 2. Path normalizer (`internal/pathmap`)

The make-or-break feature for Windows ↔ macOS dual setups:

- Store portable tokens: `${HOME}`, `${REPO:<id>}`, and user-defined
  `${WORK:<alias>}`.
- On **push**: rewrite recognized structural path fields → tokens
- On **pull**: rewrite tokens → this machine's absolute paths
- Maintain a **canonical project ID** (git remote + name, or user alias) mapped
  to per-device roots so munged slugs / hashes recompute correctly
- Do not rewrite arbitrary prose, prompts, tool output, or unknown fields

### 3. Encryption (`internal/crypto`)

- Default: [age](https://github.com/FiloSottile/age) passphrase encryption
- Enter the same passphrase on every device; no keyfile is copied or stored
- File modes: `0600` for secrets, `0700` for config dirs

### 4. Sync engine (`internal/sync`)

- Versioned local sync state stored as atomic JSON
- Immutable, UUID-addressed encrypted snapshots
- Encrypted remote JSON manifest with conditional ETag updates
- Streamed full-snapshot transfer with authenticated metadata, size, and
  SHA-256 validation
- Full snapshots in Phase 1; chunking and append-aware deltas are roadmap work
- `status` and `diff` currently require access to the remote manifest

### 5. Restore path

1. Pull ciphertext → decrypt
2. Path-rewrite into local layout
3. If target exists: copy to `~/.reinstate/backups/<timestamp>/`
4. Atomic write (temp + rename)
5. Refuse or warn if agent process holds the file open (liveness check)

## What is explicitly not synced

See [security-model.md](security-model.md). Defaults exclude:

- `auth.json`, OAuth/credential stores
- Plugin caches / `node_modules` / venvs
- Machine-local logs
- User-defined globs

Future configuration sync carries non-secret declarations and secret
references, never vendor auth stores or whole tool directories.

## Why not CRDTs / real-time collab?

Dominant pattern is **sequential** multi-device use (desktop by day, laptop by
night). Last-writer-wins with conflict *detection* and safe forks matches
reality without tripling complexity.

## Tech stack

| Layer | Choice | Why |
| ----- | ------ | --- |
| Language | Go | Single static binary, cross-compile, proven by peers |
| Crypto | age | Passphrase UX + auditability |
| Storage | S3-compatible first | R2 free tier; rclone-style backends later |
| Local state | Versioned JSON | Small, inspectable, atomically replaced |
| Remote index | Encrypted JSON manifest | Conditional updates and conflict detection |

## Related diagrams

| Asset | Description |
| ----- | ----------- |
| [01_landscape.png](/brand/01_landscape.png) | Agent scope vs state portability |
| [02_demand_timeline.png](/brand/02_demand_timeline.png) | Demand signals on vendor trackers |
| [03_traction.png](/brand/03_traction.png) | GitHub traction landscape |
| [04_market.png](/brand/04_market.png) | Market context |
| [05_architecture.png](/brand/05_architecture.png) | MVP architecture |

## Package layout

```
cmd/reinstate/          # CLI entrypoint (install as reinstate + rein)
internal/
  adapter/              # per-agent adapters
  config/               # local config + path_map
  crypto/               # age encryption
  pathmap/              # portable path rewriting
  sync/                 # manifest, push/pull, conflicts
  backend/              # R2/S3-compatible
docs/                   # human docs
testdata/               # golden fixtures (per adapter)
```
