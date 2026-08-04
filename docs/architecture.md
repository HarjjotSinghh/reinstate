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

![Reinstate MVP architecture](../assets/05_architecture.svg)

## Continuity stack

```text
┌─────────────────────────────────────────────────────────────┐
│  CLI / TUI / (later) Console                                │
│  search · inspect · resume · handoff · configure · sync     │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  Session index + workspace fingerprints + continuity capsule │
└───────────────────────────┬─────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┬─────────────────┐
          ▼                 ▼                 ▼                 ▼
   ┌────────────┐   ┌──────────────┐   ┌────────────┐   ┌────────────┐
   │  Session   │   │ Configuration│   │ Executors  │   │ Sync / E2E │
   │  adapters  │   │ adapters     │   │ native/ACP │   │ push/pull  │
   └────────────┘   └──────────────┘   └────────────┘   └────────────┘
```

The continuity stack is delivered incrementally. Stable Phase 1 implements the
mutation-capable adapter and encrypted-sync path. Development-accepted Phase 2
adds the private local index, read capabilities, and Claude/Codex native launch plans.
Workspace fingerprints, checkpoints, configuration adapters, and ACP
integration remain later roadmap work.

**Target flow before execution:** find session, load history, fingerprint
workspace, check skills/MCP, optionally reconcile supported non-secret
configuration, build a portable continuity capsule if needed, report fidelity
and capability differences, and choose the destination agent.
**During execution:** Claude Code / Codex / Gemini / OpenCode (or another ADE)
own the agent loop.
**Target flow after execution:** capture updates, update the index, and
optionally sync encrypted state.

### Native execution capability

```text
capabilities() → ExecutorCapabilities
canResume(session) → CompatibilityResult
launch(preparedSession) → ExecutionHandle
```

Phase 2 implements structured native launch plans for Claude Code and Codex.
Gemini CLI and OpenCode are deliberately read-only in this phase. Later
executor capabilities and ACP-compatible agents may extend the same boundary.
Reinstate Console may become a thin ACP **client**, not a full harness.

ACP `session/load` / `session/resume` can reduce integration work for an agent
that already owns a session. ACP does not by itself define how a foreign
vendor's transcript becomes that agent's history.

### Cross-agent continuation (target)

Cross-agent handoff has three separately hashable layers:

```text
immutable vendor artifact
          │ version-gated parse
          ▼
canonical continuity capsule
  task · normalized events · workspace · capabilities
  security · fidelity · lineage
          │ destination plan
          ▼
destination projection
  bootstrap context/sidecar or experimental native reconstruction
          │ acknowledge + launch
          ▼
new destination-native session
```

The source artifact remains immutable. The capsule records portable visible
history and task state. The projection is exactly what the target receives.
Every component is labeled `exact`, `normalized`, `summarized`, `referenced`,
or `omitted` with a reason.

Session sync and handoff support are separate contracts:

```text
TranscriptSource
  probe(session) → SourceCompatibility
  snapshot(session) → ImmutableBoundary
  parse(boundary) → CanonicalEvents + ParseReport

HandoffTarget
  capabilities() → TargetCapabilities
  plan(capsule, policy) → DestinationPlan + FidelityReport
  materialize(plan) → PreparedSession
  launch(prepared) → NativeSessionID
  verify(nativeSessionID) → ContinuationResult
```

Support is directional and versioned. The default cross-agent route launches a
new native session with an inspectable capsule. Direct synthesis of target
session files/databases is experimental, exact-version gated, backed up, and
never modifies the source.

Full schema, pipeline, security contract, and release gates:
[cross-agent-continuation.md](cross-agent-continuation.md).

## Design principles

1. **Local-first** — agents remain the sole executors of sessions; Reinstate
   owns continuity before/after, not the model loop.
2. **Zero-knowledge remote** — only ciphertext on object storage.
3. **Native resume is same-vendor** — restore puts bytes where `claude --resume`
   / `codex resume` already know how to read them.
4. **Cross-agent = portable handoffs** — preserve portable visible history and
   task evidence in a continuity capsule, create a linked destination session,
   and report fidelity; never claim silent native translation.
5. **Fail-safe conflicts** — never overwrite; fork and surface.
6. **Adapter isolation** — format churn in one agent cannot break others.
7. **Not an ADE** — no custom editor, terminal emulator, multi-agent scheduler,
   or model router as product spine.
8. **Normalize configuration intent** — render desired state through verified
   per-harness adapters; never mirror one harness's raw config into another.
9. **Secrets stay local** — configuration profiles contain references, never
   raw API keys, OAuth tokens, cookies, or vendor credential stores.
10. **Derived state is disposable** — the local index is private, rebuildable,
    excluded from sync, and never a new source of truth.
11. **Imported history is inert** — source system/developer messages are audit
    history, not destination authority, and historical tool calls never execute.

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

### 1A. Phase 2 local read/index path

Phase 1's `adapter.Adapter` remains the export/restore contract. Phase 2 uses a
separate read capability so read-only agents never receive dummy mutation
methods.

```text
vendor stores
    │
    ▼
local session sources
Claude │ Codex │ Gemini │ OpenCode
    │
    ▼
$REINSTATE_HOME/cache/session-index-v1.sqlite
    │
    ├── sessions / search / inspect
    └── resolve ──► native launch plan (Claude/Codex)
```

The local registry is config-independent. It does not reuse configured Phase 1
project mappings, which would hide unmapped sessions on the same machine.

The pure-Go SQLite index preserves `CGO_ENABLED=0` release builds. It stores:

- composite identity `<agent>:<native-session-id>`;
- source fingerprint, timestamps, workspace/project, branch, title/name;
- bounded user-authored prompt text for literal search;
- known structured file references, message counts, and capability flags.

It excludes assistant messages/reasoning, tool output, environment dumps,
credentials, auth stores, and unbounded fields. Source fingerprints enable
incremental refresh. A successfully scanned source removes stale rows; an
individual malformed session only emits a warning. An incomplete trailing
JSONL record is ignored while the vendor is appending. A corrupt or
incompatible derived database rebuilds without modifying vendor files.
Multiple vendor files carrying the same native session ID are coalesced into
one deterministic record; the composite identity remains stable across native
continuations.

Ordering is newest update first, then agent, then native ID. Search is literal,
case-insensitive, and local; multiple terms are ANDed. Human preview text comes
only from a user prompt, strips controls/collapses whitespace, and is capped at
160 Unicode code points.

### 1B. Phase 2 source capabilities

| Source | Read/index | Native resume | Native fork | Encrypted sync |
| ------ | ---------- | ------------- | ----------- | -------------- |
| Claude Code | full | `claude --resume ID` | `claude --resume ID --fork-session` | Phase 1 |
| Codex CLI | full | `codex resume ID` | `codex fork ID` | Phase 1 |
| Gemini CLI | read-only | no | no | no |
| OpenCode | read-only through documented JSON listing | no | no | no |

Launch plans are an executable plus argv and cwd, never a shell command string.
The recorded workspace must exist and the executor must be available before a
real launch. Reinstate inherits the user's terminal, waits for the child, and
propagates failure. `--dry-run` exposes the structured plan without launching.

### 1C. Cross-agent handoff adapters (target)

Transcript-source and handoff-target capabilities are separate again. An
adapter may be able to sync a native session without safely parsing its visible
history, or it may be a handoff source before it is a destination target.

### 1D. Configuration adapters (target)

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
4. If the exact destination session is active, leave it untouched and restore
   the incoming snapshot as a distinct idempotent vendor-safe fork
5. Otherwise restore with private permissions and atomic replacement, refusing
   a concurrent write detected before final rename

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
| Sync state | Versioned JSON | Small, inspectable, atomically replaced |
| Local index | Pure-Go SQLite | Incremental queryable derived state; static builds |
| Remote index | Encrypted JSON manifest | Conditional updates and conflict detection |

## Related diagrams

| Asset | Description |
| ----- | ----------- |
| [01_landscape.svg](../assets/01_landscape.svg) | Agent scope vs state portability |
| [02_demand_timeline.svg](../assets/02_demand_timeline.svg) | Demand signals on vendor trackers |
| [03_traction.svg](../assets/03_traction.svg) | GitHub traction landscape |
| [04_market.svg](../assets/04_market.svg) | Market context |
| [05_architecture.svg](../assets/05_architecture.svg) | MVP architecture |

## Package layout

```
cmd/reinstate/          # CLI entrypoint (install as reinstate + rein)
internal/
  adapter/              # per-agent adapters
  sessionindex/         # Phase 2 sources, private index, query, native plans
  cli/                  # sync commands + Phase 2 local commands/picker
  config/               # local config + path_map
  crypto/               # age encryption
  pathmap/              # portable path rewriting
  sync/                 # manifest, push/pull, conflicts
  backend/              # R2/S3-compatible
docs/                   # human docs
testdata/               # deterministic synthetic fixtures (per adapter/OS)
```
