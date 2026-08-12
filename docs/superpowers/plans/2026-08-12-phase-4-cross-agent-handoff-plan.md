# Phase 4 — Cross-agent continuation (portable handoffs)

**Status:** planned; not implemented
**Date:** 2026-08-12
**Target release:** `v0.4.0-rc.1` → `v0.4.0` stable
**Base:** `origin/main` at `75da7f1` (stable `v0.3.0`, Phase 3 verified resume)
**Branch:** `feat/phase4-cross-agent-handoff`
**Planner:** central planning model · **Executors:** Grok 4.5, medium reasoning
**Companion documents:**
[executor work packets](2026-08-12-phase-4-executor-work-packets.md) ·
[session storage map](../../session-storage-map.md) ·
[handoff product contract](../../handoff.md) ·
[acceptance contract](../../testing/phase-4-cross-agent-handoff-acceptance.md) ·
[ADR 0003](../../adr/0003-phase-4-rc1-scope-and-launch-route.md)

---

## 0. Read this first

This document is the **architecture**. The work packets document is the
**execution order**. An executor implements packets; it does not redesign from
this file. If a packet and this plan disagree, the packet is wrong — stop and
escalate rather than improvising.

Three rules override everything else in this plan:

1. **Cross-agent continuation is a new destination-native session linked to the
   source.** It is never presented as the same session, and never as native
   resume.
2. **The source model must not be required.** The motivating case is Claude
   Code hitting a usage limit. Every deterministic path must work with the
   source CLI closed, logged out, rate-limited, or offline.
3. **Imported history is untrusted, inert evidence.** It never becomes
   destination policy, and no historical tool call is ever re-executed.

---

## 1. Outcome

After `v0.4.0`, this works:

```bash
# Claude hit its usage limit mid-task. Claude is closed. No Claude API call.
rein handoff --last --from claude --to codex --dry-run
rein handoff claude:<session-id> --to codex

# And the reverse.
rein handoff codex:<session-id> --to claude --policy balanced

# Inspect, export, and trace lineage after the fact.
rein handoff list
rein handoff inspect <handoff-id>
rein handoff export <handoff-id> --format markdown
```

The destination opens as a **new native session** in the correct workspace,
restates the goal, the latest user request, the current changed files and test
state, and anything missing, then waits for acknowledgement before it mutates
anything.

### What ships in `v0.4.0-rc.1`

| Capability | Claude Code | Codex CLI | Gemini CLI | OpenCode | Grok Build |
| ---------- | :---------: | :-------: | :--------: | :------: | :--------: |
| Local index / search / inspect | shipped | shipped | shipped | shipped | **new** |
| Capsule **source** (parse → capsule) | **new** | **new** | **new** | **new** | **new** |
| Handoff **target** (launch destination) | **new** | **new** | rc.2 | rc.2 | not planned |
| Reconstructed native history | no | no | no | no | no |

The release gate for `v0.4.0` is **Claude ↔ Codex, both directions**. The other
three are source-only in rc.1 and must be labeled that way in every surface.

### Explicit non-goals for this phase

- Writing vendor-internal transcript files (Codex rollouts, Claude project
  JSONL). Deferred; see §12.
- Syncing capsules through encrypted BYO storage. Local-only in v0.4.0.
- Any MCP / skills / hooks / plugin configuration work. That is Phase 5.
- ACP client work. Phase 6.
- Summarization that requires a model call on the critical path.

---

## 2. Baseline rule (read before writing any code)

The working branch `agent/universal-agent-configuration-roadmap` is roughly
100k lines behind `origin/main`. It carries two documents Phase 4 depends on
(`docs/cross-agent-continuation.md`, `docs/adr/0002-cross-agent-continuation.md`)
that **do not exist on main**.

Sequence, exactly:

1. Land the cross-agent doc commit (`aa10607`) onto `main` via its own PR.
2. Land this plan set onto `main` via its own PR.
3. Branch `feat/phase4-cross-agent-handoff` from `origin/main` **after** both.

Do not develop Phase 4 on the roadmap branch. Do not merge website/media
changes into a Phase 4 PR.

Everything stable `v0.3.0` guarantees stays intact: canonical project mapping,
exact restore-path verification, live-session refusal, verified-resume gating,
private index privacy rules, and the launch-boundary identity guards.

---

## 3. Architecture

### 3.1 The three layers

Every handoff produces three separately hashable layers. This is the core
structure, and it is what makes fidelity claims auditable.

```text
  ┌─────────────────────────────────────────────────────────────┐
  │ LAYER 1 — RAW SOURCE (immutable, never modified)            │
  │ vendor bytes up to a complete-record boundary               │
  │ sha256 + byte length + boundary offset + mtime              │
  └──────────────────────────┬──────────────────────────────────┘
                             │ version-gated reader (no model call)
                             v
  ┌─────────────────────────────────────────────────────────────┐
  │ LAYER 2 — CANONICAL RECORD (vendor-neutral)                 │
  │ normalized events · task checkpoint · workspace truth        │
  │ capability diff · security metadata · fidelity report        │
  └──────────────────────────┬──────────────────────────────────┘
                             │ target projection + context policy
                             v
  ┌─────────────────────────────────────────────────────────────┐
  │ LAYER 3 — DESTINATION PROJECTION (exact bytes handed over)  │
  │ bootstrap prompt · projection.md · sidecar refs             │
  │ sha256 + byte size + estimated tokens                        │
  └─────────────────────────────────────────────────────────────┘
```

Layer 1 is never edited. Layer 2 is deterministic: parsing the same Layer 1
twice must produce a byte-identical canonical record. Layer 3 is what the
destination agent actually sees, and its hash is recorded in the lineage entry.

### 3.2 Data flow

```text
 source vendor store            live workspace              destination agent
 (claude|codex|gemini|          (git + filesystem)          (claude|codex)
  opencode|grok)                        │                          ^
        │                               │                          │
        v                               v                          │
 ┌──────────────┐              ┌────────────────┐                  │
 │ transcript   │              │ workspace      │                  │
 │ Boundary     │              │ probe (P3)     │                  │
 │ + Reader     │              │ preflight (P3) │                  │
 └──────┬───────┘              └───────┬────────┘                  │
        │ canonical events             │ verified truth            │
        v                              v                           │
 ┌───────────────────────────────────────────────┐                 │
 │ handoff pipeline                              │                 │
 │  checkpoint · capability diff · redaction     │                 │
 │  context policy · fidelity aggregation        │                 │
 └───────────────────────┬───────────────────────┘                 │
                         v                                          │
              ┌─────────────────────┐        ┌──────────────────┐  │
              │ capsule store       │───────>│ HandoffTarget    │──┘
              │ $REINSTATE_HOME/    │        │ plan/materialize │
              │   handoffs/<id>/    │        │ launch/verify    │
              │ + lineage.jsonl     │<───────│ native session id│
              └─────────────────────┘        └──────────────────┘
```

### 3.3 Package boundaries (new code)

```text
internal/secretscan/
  scan.go            deterministic secret detection + redaction markers
  patterns.go        known credential shapes + entropy heuristic

internal/capsule/
  model.go           continuity-capsule v1 types (schema constant, layers)
  event.go           canonical event, block, portability state
  canonical.go       deterministic JSON encoding + sha256 over canonical bytes
  validate.go        contract validation, bounds, required-field enforcement
  fidelity.go        component classification + report aggregation

internal/transcript/
  reader.go          Reader interface, ParseReport, registry
  boundary.go        complete-record snapshot, hashing, byte offsets
  normalize.go       shared block/tool/actor normalization helpers
  claude.go          Claude Code JSONL reader
  codex.go           Codex rollout JSONL reader
  gemini.go          Gemini chat JSON/JSONL reader (incl. $rewindTo replay)
  opencode.go        OpenCode storage reader + metadata-only fallback
  grok.go            Grok Build session reader

internal/handoff/
  pipeline.go        Plan() and Execute(); the only orchestration entry point
  checkpoint.go      deterministic, no-model task-state derivation
  capabilitydiff.go  source vs destination capability comparison
  policy.go          checkpoint | balanced | full context budgeting
  estimate.go        deterministic size/token estimation
  projection.go      bootstrap prompt + projection.md rendering
  target.go          HandoffTarget contract + registry
  target_claude.go   Claude Code destination
  target_codex.go    Codex CLI destination
  acknowledge.go     destination acknowledgement contract + verification
  store.go           $REINSTATE_HOME/handoffs/ store, lineage, retention

internal/sessionindex/
  grok.go            NEW local source so Grok sessions are indexable

internal/cli/
  handoff.go         rein handoff (+ inspect/list/export) and --with alias
```

### 3.4 Reused, not rebuilt

Phase 4 must **not** reimplement anything below:

| Need | Existing package |
| ---- | ---------------- |
| Session discovery / resolution / freshness | `internal/sessionindex` |
| Bounded, terminal-safe text | `internal/safetext`, `sessionindex.SafeText` |
| JSONL line scanning with byte caps | `sessionindex.ScanJSONLines` |
| Workspace truth (repo, branch, HEAD, dirty, digest) | `internal/workspace` |
| Environment preflight and launch policy | `internal/preflight` |
| Capability inventory (instructions/skills/MCP) | `internal/capability` |
| Agent install + version compatibility | `internal/agentcheck`, `internal/adapter/version.go` |
| Atomic writes, private modes, Windows DACL | `internal/fsx` |
| Cross-process locking | `internal/filelock` |
| Executable/workspace identity at launch boundary | `internal/fileidentity` |
| Stable exit codes | `internal/exitcode` |
| Path portability tokens | `internal/pathmap` |

---

## 4. The continuity capsule (schema v1)

Schema identifier: `reinstate.continuity-capsule/v1`.

### 4.1 Top-level shape

```go
type Capsule struct {
    Schema       string        `json:"schema"`        // reinstate.continuity-capsule/v1
    Identity     Identity      `json:"identity"`
    RawSource    RawSource     `json:"raw_source"`
    Task         Task          `json:"task"`
    Workspace    Workspace     `json:"workspace"`
    Conversation Conversation  `json:"conversation"`
    Capabilities CapabilityDiff `json:"capabilities"`
    Security     Security      `json:"security"`
    Fidelity     Fidelity      `json:"fidelity"`
    Projection   Projection    `json:"projection"`
}
```

Rules:

- **Deterministic.** No wall-clock field inside the hashed region. `created_at`
  lives in the lineage entry, not in the hashed capsule body.
- **Bounded.** Every string and slice has an explicit maximum; validation
  rejects an over-limit capsule instead of silently truncating.
- **Private.** Absolute source paths are never serialized. Workspace paths are
  emitted as `${REPO:<canonical-id>}` tokens via `internal/pathmap`.

### 4.2 Identity and lineage

```go
type Identity struct {
    ID           string `json:"id"`            // sha256 of canonical bytes, 32 hex chars
    LineageRoot  string `json:"lineage_root"`  // first capsule ID in the chain
    Parent       Parent `json:"parent_session"`
    SchemaVer    int    `json:"schema_version"`
}

type Parent struct {
    Agent          string `json:"agent"`
    SessionID      string `json:"id"`
    ArtifactSHA256 string `json:"artifact_sha256"`
    AdapterVersion string `json:"adapter_version"`
}
```

The capsule ID is **content-derived**, not random. Two handoffs of an unchanged
source at the same boundary with the same policy produce the same ID — which is
exactly what makes the golden tests possible.

### 4.3 Canonical events

```go
type Actor string        // user | assistant | tool | harness | unknown
type Kind  string        // message | tool_call | tool_result | attachment |
                         // summary | checkpoint | metadata | unknown
type Portability string  // exact | normalized | summarized | referenced | omitted

type Event struct {
    ID           string        `json:"id"`            // deterministic, derived from source pointer
    Order        int           `json:"order"`
    Timestamp    time.Time     `json:"timestamp,omitzero"`
    Actor        Actor         `json:"actor"`
    Kind         Kind          `json:"kind"`
    NativeType   string        `json:"native_type,omitempty"`
    NativeName   string        `json:"native_name,omitempty"`   // vendor tool name, verbatim
    Blocks       []Block       `json:"blocks,omitempty"`
    CallID       string        `json:"call_id,omitempty"`       // tool_call identity
    LinkedCallID string        `json:"linked_call_id,omitempty"`// tool_result → tool_call
    Portability  Portability   `json:"portability"`
    Reason       string        `json:"reason,omitempty"`
    Redactions   []Redaction   `json:"redactions,omitempty"`
    Truncated    bool          `json:"truncated,omitempty"`
    ContentHash  string        `json:"content_hash"`
    Source       SourcePointer `json:"source"`
}
```

Every event **must** carry a `Portability` value and, when it is not `exact`, a
machine-stable `Reason`. Validation rejects a capsule where any event lacks one.

Actor mapping is explicit per reader and never assumes vendor roles are
identical. A Codex `event_msg/user_message` and a Claude `type:"user"` record
both map to `Actor: user`, but each keeps its `NativeType`.

### 4.4 Fidelity report

```go
type Component struct {
    Name        string      `json:"name"`        // e.g. "user_messages", "tool_results"
    Portability Portability `json:"portability"`
    Count       int         `json:"count"`
    Bytes       int64       `json:"bytes"`
    Reason      string      `json:"reason,omitempty"`
}

type Fidelity struct {
    Overall    Portability  `json:"overall"`
    Mode       string       `json:"mode"`        // structured_handoff (rc.1 only value)
    Components []Component  `json:"components"`
    Unsupported []string    `json:"unsupported,omitempty"`
}
```

`Overall` is the **worst** portability across components that were included.
There is no "lossless" value. `Mode` is `structured_handoff` in v0.4.0;
`reconstructed_conversation` is reserved and unused.

### 4.5 What is portable, restated as code contract

| Source material | Default portability | Enforced by |
| --------------- | ------------------- | ----------- |
| User messages | `exact` (subject to redaction) | reader |
| Visible assistant messages | `exact` | reader |
| Tool call name + inputs | `normalized` | `normalize.go` |
| Tool results | `normalized`, or `referenced` when over budget | policy |
| Attachments present on disk | `referenced` (hash + mime + size) | reader |
| Attachments not on disk | `omitted`, reason `attachment_unavailable` | reader |
| Vendor compaction summaries | `summarized` | reader |
| Unknown record types | `referenced` (opaque hash) | reader |
| Source system/developer instructions | `referenced`, excluded from projection | `security.go` |
| Pending/unfinished tool calls | `omitted`, reason `interrupted_not_replayed` | checkpoint |
| Opaque/encrypted reasoning | `omitted`, reason `vendor_opaque_state` | reader |
| Credentials, tokens, auth stores | never read | reader + `secretscan` |
| Live processes, shells, sandboxes | never captured | n/a |

---

## 5. The pipeline

`internal/handoff.Plan()` and `Execute()` are the only orchestration entry
points. `Plan()` is pure enough to power `--dry-run`; `Execute()` calls
`Plan()` and then performs side effects.

### Step 1 — Resolve and freeze the source boundary

- Resolve the reference through `sessionindex.RefreshAndResolve`.
- Refuse if the source agent process is holding the file
  (`processcheck.SessionBusy`) unless `--allow-active` is given, in which case
  take the boundary at the last complete record and mark
  `source_may_have_advanced: true`.
- `transcript.Snapshot()` reads to the last **complete** record, records byte
  offset, SHA-256 of the prefix, size, and mtime.
- Never write, lock, rename, or truncate the source.

### Step 2 — Parse without the source model

- Version-gate: unknown adapter version → exit `5` unless `--allow-untested`.
- Emit canonical events, a `ParseReport` (counts by kind, unknown records,
  malformed lines, truncation), and per-event portability.
- No network. No model. No environment mutation.

### Step 3 — Verify the workspace

- Run the existing `preflight.Verify` against the source record.
- The **workspace wins** over the transcript. If a transcript claims a file was
  changed and Git disagrees, the capsule records the Git truth and marks the
  claim `evidence_conflicts_with_workspace`.
- Blocked preflight → blocked handoff, same exit code as Phase 3.

### Step 4 — Negotiate destination capabilities

- Build a `capability.Inventory` for the destination agent.
- Diff tool families, approval modes, MCP servers, skills, instruction files,
  attachment support, multi-root support, and context ceiling.
- Every missing item becomes a visible warning **and** a capsule field. Never
  a silent deletion.
- Destination not installed, or version untested → exit `5`.

### Step 5 — Budget and project

| Policy | Verbatim conversation | Tool evidence | Sidecar |
| ------ | --------------------- | ------------- | ------- |
| `checkpoint` | none | none | full history written, referenced |
| `balanced` (default) | most recent turns within the byte budget | most recent N results, each truncated with a visible marker | full history written, referenced |
| `full` | all portable visible events up to the hard cap | all, truncated per-item | full history written, referenced |

Default projection budget: **64 KiB** of prompt-bound bytes; hard cap **2 MiB**
for `full`. Overflow always becomes a sidecar reference, never a silent drop.

Token estimation is deterministic and declared as an estimate:
`estimated_tokens = ceil(utf8_bytes / 4)`. Do not add a tokenizer dependency.

### Step 6 — Preview (`--dry-run`)

Prints, without writing anything outside a temp dir:

- source and planned destination identity;
- workspace and capability differences;
- counts and bytes per portability class;
- redaction count and redaction categories (never the matched values);
- the exact files that would be written and the exact argv that would run;
- estimated projection bytes and tokens;
- every warning ID that would need acknowledgement.

### Step 7 — Materialize the destination session

Route for v0.4.0 (see ADR 0003): **new native session + inspectable capsule
file + bootstrap prompt**. No vendor-internal writes.

| Destination | argv (working dir = verified workspace) | Session ID |
| ----------- | --------------------------------------- | ---------- |
| Claude Code | `claude --session-id <new-uuid> "<bootstrap>"` | pinned by Reinstate |
| Codex CLI | `codex "<bootstrap>"` | assigned by Codex; reconciled after launch |

The bootstrap prompt is bounded to **8 KiB** and points at
`<handoff-dir>/projection.md` for the rest. Argv length is validated against a
conservative 24 KiB ceiling so Windows never truncates silently.

Because Codex assigns its own session ID, `verify()` reconciles after the child
exits: rescan the Codex source, select rollouts whose `cwd` matches the verified
workspace and whose mtime is after launch start, and match the first user
message against the bootstrap hash. Ambiguous or no match → lineage records
`destination_session: unresolved` with a reason. It never guesses.

Reuse the Phase 3 launch guards unchanged: TTY requirement, executable identity
capture before and after the final guard, workspace identity stability, and
`BeforeExec` revalidation.

### Step 8 — Acknowledge before acting

The bootstrap requires the destination to restate, before any mutation:

1. current goal and latest user request;
2. critical constraints carried over;
3. current changed files and test state;
4. missing capabilities or uncertain evidence;
5. proposed next action.

For rc.1 this is a **prompt-level contract**, not an enforced protocol —
Reinstate cannot police another agent's loop. Say that plainly in the docs.
`rein handoff inspect` records whether the user confirmed the acknowledgement
(`--acknowledged` / `--not-acknowledged`), so the metric is honest.

### Step 9 — Record lineage

Append one entry to `$REINSTATE_HOME/handoffs/lineage.jsonl`:

```json
{
  "handoff_id": "…",
  "lineage_root": "…",
  "created_at": "2026-08-12T…Z",
  "source": {"agent": "claude", "session_id": "…", "artifact_sha256": "…"},
  "destination": {"agent": "codex", "session_id": "…", "state": "resolved"},
  "policy": "balanced",
  "capsule_sha256": "…",
  "projection_sha256": "…",
  "fidelity_overall": "normalized",
  "launched": true,
  "acknowledged": null
}
```

Lineage is append-only. A later hand-back links to both prior sessions rather
than overwriting.

---

## 6. Deterministic checkpoint derivation

This is the highest-risk area for over-claiming. The rules are deliberately
conservative.

| Field | Derivation | Honesty rule |
| ----- | ---------- | ------------ |
| `latest_user_intent` | Verbatim text of the last non-meta `user` message | `exact`; truncation gets a visible marker |
| `goal` | First non-meta user message + latest intent, both verbatim, bounded | `normalized`, labeled `derived_deterministic` |
| `recent_user_messages` | Last K user messages, verbatim | `exact` — this is where real constraints live |
| `completed` | Tool call/result pairs with no error, rendered as evidence lines | `normalized`; never phrased as a claim about the repo |
| `pending` | Tool calls with no matching result | `omitted` + `interrupted_not_replayed` |
| `changed_files` | **Live Git porcelain**, not the transcript | `exact` (current observation) |
| `files_touched_per_transcript` | File refs extracted from tool inputs | `referenced`, labeled as claims |
| `tests` | Last command matching a deterministic test-runner allowlist, plus exit state | `referenced` |
| `next_action` | Template: continue the latest user request, given current workspace state | `normalized`, labeled `derived_deterministic` |
| `constraints`, `decisions`, `rejected_approaches` | **Not derived deterministically** | `omitted` + `requires_optional_summarizer` |

**Do not regex-mine decisions and rejected approaches.** Inventing structure
from prose is exactly the failure mode that makes a handoff worse than useless.
The verbatim recent-user-messages block carries that information honestly.

An optional summarizer provider interface (`handoff.Summarizer`) may populate
the omitted fields when a model is available. It is off by default, its output
is provenance-labeled `agent_assisted`, and no code path may depend on it.

### Truth hierarchy (enforced, in order)

1. Current workspace bytes and Git state
2. Latest explicit user intent and constraints
3. Completed tool results that can still be verified
4. Structured checkpoint
5. Older conversation statements and model plans

---

## 7. Security contract

Every rule below needs a test that fails when the rule is removed.

1. **Source instructions are audit-only.** System/developer messages are stored
   with `portability: referenced` and are excluded from the projection body.
   Test: a fixture with a source system prompt must not appear in
   `projection.md`.
2. **Historical tool calls are inert.** No pipeline path executes a command
   found in a transcript. Test: an adversarial fixture containing
   `rm -rf /` in a tool input completes a handoff and executes nothing.
3. **Prompt injection stays quoted.** Imported content is rendered inside a
   fenced, source-attributed block with an explicit inertness banner. Fence
   collisions are escaped. Test: a fixture whose content contains a fence and
   "ignore previous instructions" cannot break out of the block.
4. **Redaction runs before write.** `secretscan` replaces matches with
   `[redacted:<category>:<sha256-prefix>]`. The marker can never be mistaken
   for original text. `--no-redact` is refused for the Grok source path.
5. **Credentials are never read.** Auth files, keychains, `.env`, token stores
   remain hard-excluded.
6. **Private on disk.** `$REINSTATE_HOME/handoffs/` is `0700`; every file is
   `0600`; Windows uses the existing protected DACL. Never inside the repo.
7. **Destination re-authorizes.** Reinstate passes no permission grant,
   approval, or credential to the destination.
8. **Fail closed on unknown versions.** Unknown source or destination layout →
   exit `5`, no partial artifact left behind.
9. **Grok destination is refused** in v0.4.0, and any Grok **source** handoff
   prints the documented upload-behavior warning.

---

## 8. CLI surface

```text
rein handoff [SESSION] --to AGENT [flags]
  --last                     resolve the newest matching source session
  --from AGENT               restrict --last to one source agent
  --to AGENT                 destination agent (required)
  --policy checkpoint|balanced|full     default: balanced
  --dry-run                  plan and preview; write nothing outside a temp dir
  --json                     machine-readable output (requires --dry-run to launch-free)
  --no-launch                build the capsule, print the exact command, do not spawn
  --export PATH              additionally write the projection to PATH
  --allow-warning ID         acknowledge one exact preflight/handoff warning
  --allow-active             take a boundary while the source agent is running
  --allow-untested           proceed on an untested source or destination version
  --show-redactions          list redaction categories and counts (never values)

rein handoff list [--json] [--limit N]
rein handoff inspect HANDOFF_ID [--json]
rein handoff export HANDOFF_ID --format json|markdown [--out PATH]

rein resume SESSION --with AGENT     convenience alias for `handoff --to AGENT`
```

Exit codes reuse the existing contract — **no new codes**:

| Code | Meaning in handoff |
| ---- | ------------------ |
| `0` | Handoff planned or completed |
| `2` | Usage: bad flags, unknown agent, `--json` without a launch-free mode |
| `3` | Config problem |
| `5` | Untested/unsupported source or destination version or layout |
| `6` | Ambiguous session reference |
| `7` | Unacknowledged warnings, or refusal on a safety rule |
| `1` | Runtime failure |

`--with` must print a one-line notice that this is a **structured handoff**,
not native resume, so the alias never becomes a fidelity overclaim.

---

## 9. On-disk layout

```text
$REINSTATE_HOME/
  handoffs/                          0700
    lineage.jsonl                    0600, append-only
    <handoff-id>/                    0700
      capsule.json                   0600  canonical record (layers 1+2)
      projection.md                  0600  exact destination projection
      bootstrap.txt                  0600  exact prompt passed as argv
      fidelity.json                  0600  fidelity report
      sidecar/
        events.jsonl                 0600  full portable history
        blobs/<sha256>               0600  large tool outputs / attachments refs
```

Retention: `rein handoff list` shows age; a future `rein handoff prune` is
scoped out of rc.1 but the store must be safely deletable by hand. Deleting
`handoffs/` loses lineage and nothing else — sessions remain in their vendor
stores.

**Sync scope:** none. `handoffs/` is hard-excluded from push/pull in v0.4.0.

---

## 10. Testing strategy

| Layer | Approach |
| ----- | -------- |
| Readers | Table-driven per record class, per agent, per OS fixture root |
| Capsule | Golden canonical JSON, byte-exact |
| Projection | Golden markdown, byte-exact |
| Determinism | Parse the same boundary twice → identical capsule ID |
| Pipeline | Fake launch runner; no vendor process ever spawned in unit tests |
| Security | Adversarial fixtures: injection, secrets, fence-breakout, huge lines |
| Performance | 200+ turn fixture parses within an absolute wall-clock ceiling |
| Docs | `internal/doctest` contract tests for the new documents |
| Cross-OS | Fixture roots for macOS, Windows, WSL per agent |

Fixture tree:

```text
testdata/handoff/
  claude/{long-history,compaction,parallel-tools,subagents,attachments,
          partial-final-record,unknown-records}/
  codex/{long-history,forks,parallel-tools,reasoning-items,
         partial-final-record,unknown-records}/
  gemini/{rewind,legacy-json,jsonl}/
  opencode/{storage,metadata-only}/
  grok/{basic,compacted}/
  adversarial/{prompt-injection,secret-leakage,fence-breakout,oversized}/
  golden/{capsule,projection}/
```

Every fixture is synthetic and passes `make fixture-scan`. No contributor's
real agent tree is ever read into the repository.

---

## 11. Release ladder

| Release | Contents | Gate |
| ------- | -------- | ---- |
| `v0.4.0-rc.1` | Everything in §1: capsule, readers ×5, pipeline, Claude↔Codex targets, CLI, store, fixtures, docs | Dual-platform acceptance run on macOS arm64 + Windows amd64 |
| `v0.4.0-rc.2` | rc.1 acceptance fixes; Gemini/OpenCode targets behind `--experimental` | Re-run the full matrix on tagged artifacts |
| `v0.4.0-rc.3+` | Hardening only; no new surface | Same |
| `v0.4.0` stable | Promotion decision + fresh tagged-artifact validation on both supported platforms | `RELEASING.md` |

RC1 is not stable and must not be described as GA cross-agent support anywhere.

---

## 12. Deferred, with the reason

| Item | Deferred to | Why |
| ---- | ----------- | --- |
| Reconstructed native history (writing vendor internals) | v0.4.x experimental, after stable | Needs exact-version fixtures, backups, consent, and native-resume validation; a wrong write corrupts a user's real session store |
| Capsule sync through BYO storage | Phase 5 | Pulls the whole manifest/conflict surface into the Phase 4 matrix |
| Gemini / OpenCode as destinations | rc.2 | Launch routes and capability inventories are not yet built for them |
| Grok as a destination | not planned for v0.4.0 | Documented repository-upload behavior; needs a separate privacy design |
| ACP transport | Phase 6 | Does not define cross-vendor history import |
| Agent-assisted summarization on by default | after stable | Must never become a dependency of the quota-exhaustion path |
| `rein handoff prune` | v0.4.x | Manual deletion is safe and sufficient for rc.1 |

---

## 13. Open research the executor must resolve

These are **blocking** for the packets that need them. Each must end as a
committed synthetic fixture plus a note in
[session-storage-map.md](../../session-storage-map.md).

| # | Question | Blocks |
| - | -------- | ------ |
| R1 | Exact OpenCode `storage/message/<session-id>/` record schema and its Windows root | WP-09 |
| R2 | Exact Grok Build session filename and JSON schema; how the workspace key is encoded | WP-10 |
| R3 | Whether Grok `/compact` and Gemini `$rewindTo` destroy or preserve prior turns | WP-08, WP-10 |
| R4 | Codex reasoning-item shapes that must be classified `vendor_opaque_state` | WP-07 |
| R5 | Claude Code `--session-id` behavior when the UUID already exists (collision policy) | WP-16 |
| R6 | Codex initial-prompt argv behavior and its practical length ceiling on Windows | WP-17 |
| R7 | Claude/Codex context-window ceilings to publish in the capability diff | WP-13 |
| R8 | Attachment storage: are Claude image blocks inline base64 or path references? | WP-06 |

Rule for all of them: if a question cannot be answered from vendor
documentation or a synthetic fixture, the corresponding capability ships as
`omitted` with a reason — never as a guess.

---

## 14. Definition of done for `v0.4.0-rc.1`

- [ ] All 27 work packets merged, each with tests
- [ ] `make verify` green on macOS arm64 and Windows amd64
- [ ] Claude → Codex and Codex → Claude handoffs succeed with the source CLI
      closed and no source API call
- [ ] Every fidelity class appears in a real report and is byte-golden-tested
- [ ] Adversarial injection and secret fixtures pass with zero leakage
- [ ] A 200+ turn source produces a bounded, reported projection under ceiling
- [ ] Windows ↔ macOS paths remap through canonical project IDs
- [ ] `--dry-run` output matches the executed run exactly, byte for byte
- [ ] Docs updated: README, ROADMAP, adapters, compatibility, cli-reference,
      handoff.md, CHANGELOG, website product-truth register
- [ ] Acceptance runbook and per-tag verification prompts committed
- [ ] No new exit codes; no vendor-internal writes; no capsule sync
