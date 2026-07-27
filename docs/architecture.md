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

![Reinstate MVP architecture](../assets/05_architecture.png)

## Continuity stack

```text
┌─────────────────────────────────────────────────────────────┐
│  CLI / TUI / (later) Console                                │
│  search · inspect · resume · handoff · configure · sync     │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  Session index + workspace fingerprints + checkpoints       │
└───────────────────────────┬─────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┬─────────────────┐
          ▼                 ▼                 ▼                 ▼
   ┌────────────┐   ┌──────────────┐   ┌────────────┐   ┌────────────┐
   │  Session   │   │ Configuration│   │ Executors  │   │ Sync / E2E │
   │  adapters  │   │ adapters     │   │ native/ACP │   │ push/pull  │
   └────────────┘   └──────────────┘   └────────────┘   └────────────┘
```

The continuity stack is the target architecture for later phases. Phase 1
implements the adapter and encrypted-sync path. Local indexing, workspace
fingerprints, checkpoints, configuration adapters, executors, and ACP
integration are roadmap work.

**Target flow before execution:** find session, load history, fingerprint
workspace, check skills/MCP, optionally reconcile supported non-secret
configuration, build a portable checkpoint if needed, and choose the
destination agent.
**During execution:** Claude Code / Codex / Gemini / OpenCode (or another ADE)
own the agent loop.
**Target flow after execution:** capture updates, update the index, and
optionally sync encrypted state.

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
8. **Normalize configuration intent** — render desired state through verified
   per-harness adapters; never mirror one harness's raw config into another.
9. **Secrets stay local** — configuration profiles contain references, never
   raw API keys, OAuth tokens, cookies, or vendor credential stores.

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

Session adapters and configuration adapters have separate support states. A
harness may support session discovery before it supports any configuration
capability.

### 1A. Configuration adapters (target)

Later phases add a canonical desired-state profile for MCP servers,
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings.
Configuration adapters import and render only supported fields while preserving
unrelated native settings.

```text
native config ↔ configuration adapter ↔ Reinstate desired state
                                             ↕ encrypted sync
                                      another device
```

Every adapter must expose a capability matrix, preview native diffs, report
lossy or unsupported mappings, back up changed files, write atomically, and
fail closed on an unverified harness schema. Executable extensions also require
source/version pinning and explicit install consent.

See [universal-configuration.md](universal-configuration.md).

### 2. Path normalizer (`internal/pathmap`)

The make-or-break feature for Windows ↔ macOS dual setups:

- Store the released portable tokens `${HOME}` and `${REPO:<id>}`.
- Keep the lower-level `${WORK:<alias>}` primitive explicitly unwired until
  configuration, adapter integration, and compatibility tests ship.
- On **push**: rewrite recognized structural path fields → tokens
- On **pull**: rewrite tokens → this machine's absolute paths
- Maintain a **canonical project ID** (git remote + name, or user alias) mapped
  to per-device roots so munged slugs / hashes recompute correctly
- Do not rewrite arbitrary prose, prompts, tool output, or unknown fields

### 3. Encryption (`internal/crypto`)

- Default: [age](https://github.com/FiloSottile/age) with passphrase-derived keys
- Same passphrase on every device → same key (no keyfile copying)
- File modes: `0600` for secrets, `0700` for config dirs

### 4. Sync engine (`internal/sync`)

- Versioned local sync state stored as atomic JSON
- Immutable, UUID-addressed encrypted snapshots
- Encrypted remote JSON manifest with conditional ETag updates and bounded
  compare-and-swap retries
- Streamed full-snapshot upload/download with authenticated metadata, size, and
  SHA-256 validation
- Full snapshots in Phase 1; chunking and append-aware deltas are roadmap work
- `status` and `diff` currently read the remote manifest and therefore require
  backend access

### 5. Restore path

1. Pull ciphertext → decrypt → authenticate metadata and payload
2. Let the matching adapter validate and rewrite known structural path fields
3. If the destination exists, create a timestamped private backup
4. Refuse overwrite while the matching agent process is active
5. Restore with private permissions and atomic replacement

## What is explicitly not synced

See [security-model.md](security-model.md). Defaults exclude:

- `auth.json`, OAuth/credential stores
- Plugin caches / `node_modules` / venvs
- Machine-local logs
- User-defined globs

Future universal configuration syncs non-secret **declarations**, not vendor
auth stores or entire tool directories. Local OS-keychain entries may satisfy
portable secret references, but their values do not enter the sync payload.

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
  backend/              # R2/S3-compatible
docs/                   # human docs
testdata/               # deterministic synthetic fixtures (per adapter/OS)
```
