# Architecture

Reinstate is intentionally boring **continuity infrastructure**. Differentiation
lives in **adapter quality**, **path normalization**, **environment verification**,
and **trust posture** — not exotic protocols or a proprietary coding harness.

Product layers and non-goals: [product-strategy.md](product-strategy.md),
[ROADMAP.md](../ROADMAP.md).

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│ Claude Code │   │    Codex    │   │ Gemini CLI  │  ...
│  ~/.claude  │   │  ~/.codex   │   │  ~/.gemini  │
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
│  path tokens (${HOME}, ${WORK}) · ignore rules    │
│  optional secret redaction                        │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Encryption (age / Argon2 passphrase KDF)         │
│  client-side only — remote never sees plaintext   │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Sync Engine                                      │
│  SQLite manifest · CAS chunks · tail-append JSONL │
│  conflict detection · atomic restore + backup     │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│  Backends: R2 · S3 · GCS · WebDAV · Gist · ...    │
└──────────────────────────────────────────────────┘
```

![Reinstate MVP architecture](../assets/05_architecture.png)

## Continuity stack

```text
┌─────────────────────────────────────────────────────────────┐
│  CLI / TUI / (later) Console                                │
│  search · inspect · resume · fork · handoff · sync          │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  Session index + workspace fingerprints + checkpoints       │
└───────────────────────────┬─────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
   ┌────────────┐   ┌──────────────┐   ┌────────────┐
   │  Adapters  │   │  Executors   │   │ Sync / E2E │
   │ discover   │   │ launch native│   │ encrypt    │
   │ transform  │   │ agent / ACP  │   │ push/pull  │
   └────────────┘   └──────────────┘   └────────────┘
```

**Before execution:** find session, load history, fingerprint workspace, check
skills/MCP, build portable checkpoint if needed, choose destination agent.  
**During execution:** Claude Code / Codex / Gemini / OpenCode (or another ADE)
own the agent loop.  
**After execution:** capture updates, update index, optional encrypted sync.

### SessionExecutor (target contract)

```text
capabilities() → ExecutorCapabilities
canResume(session) → CompatibilityResult
launch(preparedSession) → ExecutionHandle
```

Implementations: Claude Code, Codex, Gemini CLI, OpenCode; later ACP-compatible
agents. Reinstate Console may become a thin ACP **client**, not a full harness.

## Design principles

1. **Local-first** — agents remain the sole executors of sessions; Reinstate
   owns continuity before/after, not the model loop.
2. **Zero-knowledge remote** — only ciphertext on object storage.
3. **Native resume is same-vendor** — restore puts bytes where `claude --resume`
   / `codex resume` already know how to read them.
4. **Cross-agent = portable handoffs** — explicit checkpoints, never silent
   transcript translation.
5. **Fail-safe conflicts** — never overwrite; fork and surface.
6. **Adapter isolation** — format churn in one agent cannot break others.
7. **Not an ADE** — no custom editor, terminal emulator, multi-agent scheduler,
   or model router as product spine.

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

### 2. Path normalizer (`internal/pathmap`)

The make-or-break feature for Windows ↔ macOS dual setups:

- Store portable tokens: `${HOME}`, user-defined `${WORK}`, etc.
- On **push**: rewrite absolute paths → tokens (paths *and* transcript content)
- On **pull**: rewrite tokens → this machine's absolute paths
- Maintain a **canonical project ID** (git remote + name, or user alias) mapped
  to per-device roots so munged slugs / hashes recompute correctly

### 3. Encryption (`internal/crypto`)

- Default: [age](https://github.com/FiloSottile/age) with passphrase-derived keys
- Same passphrase on every device → same key (no keyfile copying)
- File modes: `0600` for secrets, `0700` for config dirs

### 4. Sync engine (`internal/sync`)

- Per-device SQLite **manifest** of remote object versions
- Content-addressed chunks for large histories (Codex multi-GB cases)
- JSONL **tail-append** awareness so routine syncs stay small
- `status` / `diff` offline-capable from the local manifest

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
| Manifest | SQLite | Offline status/diff |

## Related diagrams

| Asset | Description |
| ----- | ----------- |
| [01_landscape.png](../assets/01_landscape.png) | Agent scope vs state portability |
| [02_demand_timeline.png](../assets/02_demand_timeline.png) | Demand signals on vendor trackers |
| [03_traction.png](../assets/03_traction.png) | GitHub traction landscape |
| [04_market.png](../assets/04_market.png) | Market context |
| [05_architecture.png](../assets/05_architecture.png) | MVP architecture |

## Package layout

```
cmd/reinstate/          # CLI entrypoint (install as reinstate + rein)
internal/
  adapter/              # per-agent adapters
  config/               # local config + path_map
  crypto/               # age encryption
  pathmap/              # portable path rewriting
  sync/                 # manifest, push/pull, conflicts
  backend/              # R2/S3/WebDAV/...
docs/                   # human docs
testdata/               # golden fixtures (per adapter)
```
