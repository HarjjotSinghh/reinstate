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

The continuity stack is delivered incrementally. Stable Phase 1 implements the
mutation-capable adapter and encrypted-sync path. Stable Phase 2 adds the
private local index, read capabilities, and Claude/Codex native launch plans.
Phase 3 verified resume is included in stable `v0.3.0`, whose
tagged-artifact acceptance is pending. Portable checkpoints,
configuration adapters, and ACP integration remain later roadmap work.

**Phase 3 RC1 flow before execution:** find one same-vendor session,
refresh its source, build the native launch plan, fingerprint the workspace,
observe supported skills/MCP/instructions and runtimes, authorize the immutable
report, repeat the refresh/plan/report, and launch only if nothing changed.
Future flows may optionally reconcile supported non-secret configuration or
build a portable checkpoint when changing vendors.
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
10. **Derived state is private, never authoritative** — local session rows are
    rebuildable and excluded from sync. Explicitly deleting v2 also deletes
    useful prelaunch comparison history, so the next launch returns to honest
    `baseline.unavailable` uncertainty.

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
$REINSTATE_HOME/cache/session-index-v2.sqlite
    │
    ├── sessions / search / inspect
    └── resolve ──► native launch plan (Claude/Codex)
```

The v2 registry also stores private, successful prelaunch observations used by
verified resume. A shared/exclusive `.lock` protects database lifetime and
destructive corruption repair, while a separate `.write.lock` serializes
ordinary updates and transactional rebuilds from both
binary aliases. The database and both owner-only lock files are local derived
state and are never synced. The local registry is
config-independent. It does not reuse configured Phase 1
project mappings, which would hide unmapped sessions on the same machine.
Unix permissions are enforced with `0700`/`0600`; native Windows uses a
protected DACL limited to the current user, LocalSystem, and Administrators,
including when `REINSTATE_HOME` is under a broadly inherited custom parent.

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

### 1C. Phase 3 verified-resume path (`v0.3.0`)

Phase 3 composes four read-only observers behind one deterministic preflight:

```text
selected fresh session record + prior private baseline
                    │
                    ▼
     workspace │ agent │ capability │ runtime observers
                    │
                    ▼
       immutable report + ready/warn/block decision
                    │
          exact warning authorization
                    │
       refresh + rebuild plan/report + equality check
                    │
       launch-bound refresh/report guard in real runner
                    │
                    ▼
         same-vendor native child execution
                    │ successful exit only
                    ▼
       persist reinstate_prelaunch_observed baseline
```

The workspace observer uses fixed, shell-free local Git commands. Repository
identities are credential-free opaque digests; working-tree comparisons store
state/counts/digests, not filenames or diffs. It never fetches.

The agent observer checks executable presence, a strictly parsed installed
version, a recognized same-vendor layout, and a private filesystem identity
that the native runner rechecks after its final guard. The capability observer performs
bounded known-path discovery for sanitized instruction/skill/MCP names,
scope/state, and coarse MCP transport. It does not return contents, paths,
commands, arguments, raw URLs, headers, environment values, or credentials.
The runtime observer understands only supported Node/Go declarations and runs
bounded version probes in a sanitized environment—never package-manager hooks
or project code.

First inspection cannot reconstruct history and therefore reports
`baseline.unavailable`. Only the exact observation immediately preceding an
authorized native child that exits successfully can become a private baseline.
Blocked, declined, cancelled, stale, changed-after-authorization, or failed
launches do not update it.

The report is additive to `inspect` and dry-run JSON. Human and machine output
share the same stable checks and provenance. TTY warnings default to no;
automation must acknowledge every exact current warning ID. Blockers have no
bypass. Gemini/OpenCode remain read-only and refuse before preflight.

See [verified-resume.md](verified-resume.md) for the policy and
[phase-3-verified-resume-acceptance.md](testing/phase-3-verified-resume-acceptance.md)
for the release gate.

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
  workspace/            # local-only repository/worktree fingerprinting
  agentcheck/           # same-vendor executable/version/layout checks
  capability/           # bounded name-only instruction/skill/MCP inventory
  runtimecheck/         # recognized Node/Go declarations and version probes
  environment/          # bounded recorded facts and private baselines
  preflight/            # deterministic Phase 3 report and authorization policy
  cli/                  # sync commands + local commands/picker/preflight UX
  config/               # local config + path_map
  crypto/               # age encryption
  pathmap/              # portable path rewriting
  sync/                 # manifest, push/pull, conflicts
  backend/              # R2/S3-compatible
docs/                   # human docs
testdata/               # deterministic synthetic fixtures (per adapter/OS)
```
