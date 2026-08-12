# Cross-agent continuation (roadmap)

**Status:** accepted core product direction; planned for Phase 4
**Last researched:** 2026-08-05
**Complements:** [ROADMAP.md](../ROADMAP.md),
[product-strategy.md](product-strategy.md), [architecture.md](architecture.md),
[ADR 0002](adr/0002-cross-agent-continuation.md)

## Direct answer

Cross-agent continuation is a **core Reinstate capability**, but it is **not in
the current Phase 1 CLI**.

Phase 1 restores a native session only into the same harness: Claude Code to
Claude Code, or Codex to Codex. Phase 4 will let a developer whose Claude Code
quota expires continue the same **task** in Codex, and vice versa, without
re-explaining the work from zero. Gemini CLI and OpenCode follow; Grok Build,
Orca, Cursor, Copilot CLI, and other harnesses require adapter evidence before
they receive a committed support level.

The honest product promise is:

> Continue the same work in another supported agent through an explicit,
> inspectable handoff. Reinstate preserves the source artifact and lineage,
> transfers every portable part it can, and reports everything it normalized,
> summarized, or could not carry.

It is usually a **new destination-native session linked to the source**, not the
same vendor session ID and not an indistinguishable continuation of the source
harness runtime.

## The quota-switch user story

The primary acceptance scenario is deliberately concrete:

1. A developer is midway through a task in Claude Code.
2. Claude Code reaches an account usage limit or becomes unavailable.
3. The source process may already be closed; Reinstate cannot require another
   Claude model call, Anthropic credential, or live source machine.
4. The developer previews a handoff to Codex.
5. Reinstate captures the latest complete source boundary, verifies the repo and
   worktree, builds a portable continuity capsule, and reports fidelity and
   destination capability differences.
6. Reinstate launches a **new Codex session** with the task state, portable
   conversation evidence, changed-file/test state, and next action.
7. Codex confirms its understanding before it is allowed to continue mutating
   work.

The same flow must work Codex → Claude Code and later across every supported
source/target pair.

Planned syntax, subject to a CLI RFC:

```text
rein handoff --last --from claude --to codex --dry-run
rein handoff <session-id> --to codex
rein handoff inspect <handoff-id>
rein handoff export <handoff-id> --format json|markdown
```

`rein resume <session-id> --with <agent>` may be a convenience alias, but the
explicit `handoff` verb is preferred because it does not imply native resume.

## Continuity modes

Reinstate must label these modes separately in CLI output, JSON, docs, and
telemetry:

| Mode | Destination | Fidelity | Product status |
| ---- | ----------- | -------- | -------------- |
| **Native resume** | Same harness/vendor | Highest; vendor session semantics retained | Phase 1 for Claude Code and Codex |
| **Structured handoff** | Different harness | Portable task state + selected verbatim history + evidence | Default Phase 4 path |
| **Reconstructed conversation** | Different harness | Normalized visible history projected into a new native session | Experimental, pair/version-specific |

“Same task” is a valid cross-agent claim. “Same exact session,” “all internal
context,” and “lossless native resume” are not valid cross-agent claims unless
a destination vendor eventually publishes and supports that exact import
contract.

## Research findings

### Native histories are rich but vendor-owned

- [Claude Code session documentation](https://code.claude.com/docs/en/sessions)
  says CLI sessions are project-scoped JSONL containing messages, tool use, and
  metadata. Claude can resume, branch, and export them, but the native shape and
  project lookup belong to Claude Code.
- [Codex's open-source CLI](https://github.com/openai/codex/blob/main/codex-rs/cli/src/main.rs)
  exposes native resume and fork operations. Its
  [app-server protocol](https://github.com/openai/codex/blob/main/codex-rs/app-server/README.md)
  can reconstruct turns when resuming a Codex thread, but that is a Codex
  contract, not a general transcript-import standard.
- [Gemini CLI session management](https://geminicli.com/docs/cli/session-management/)
  records prompts, responses, tool inputs/outputs, usage, and available
  reasoning summaries in a project-derived directory, then resumes by Gemini
  session ID.
- [OpenCode](https://opencode.ai/docs) supports native conversation sharing,
  but sharing or viewing a native conversation is not the same as importing it
  into another harness's agent loop.

### A common transport is not yet a common cross-vendor history

The [Agent Client Protocol](https://agentclientprotocol.com/) standardizes
client ↔ agent communication. Its `session/load` and stable
[`session/resume`](https://agentclientprotocol.com/announcements/session-resume-stabilized)
methods help a client reconnect to a session that the **same agent** knows how
to restore. ACP is useful as a future executor transport, but it does not turn a
Claude transcript into a Codex thread or define how foreign system prompts,
reasoning blocks, and tools become destination-native state.

### Message and tool contracts are materially different

- Claude's API puts its system prompt in a top-level `system` field and embeds
  `tool_use` / `tool_result` blocks inside assistant/user content. See the
  [Messages API](https://platform.claude.com/docs/en/api/messages/create) and
  [tool-call handling](https://platform.claude.com/docs/en/agents-and-tools/tool-use/handle-tool-calls).
- OpenAI Responses/Codex histories contain typed response items, call IDs, and
  model-specific reasoning state. OpenAI's
  [model guidance](https://developers.openai.com/api/docs/guides/latest-model)
  requires retained output items—and, in some modes, opaque encrypted reasoning
  items—for faithful continuation.
- Gemini uses typed parts and function call/result relationships; thinking
  models can require positional thought signatures. See
  [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling).

A tool named `Bash` in one harness and `exec_command` in another may overlap,
but its sandbox, approval policy, output shape, working directory, and side
effects are not interchangeable. Old tool calls are evidence; Reinstate must
never replay them as new executable requests.

### The market validates the workflow and the fidelity limit

Current open-source projects already target the exact quota-switch problem:

- [`continues`](https://github.com/yigitkonur/cli-continues) parses many agent
  stores and injects a structured context document into a target tool.
- [`sessionbridge`](https://github.com/tongtongtju/sessionbridge) synthesizes
  Claude Code ↔ Codex session files and maps common tool calls. Its own fidelity
  notes say encrypted reasoning, web-search results, subagent structures, and
  hook progress are lost.

These projects show real demand and useful implementation paths. They also show
why Reinstate should differentiate on verifiable fidelity, environment truth,
security, path remapping, immutable lineage, and fail-closed adapter support—not
on an unqualified “everything transfers” claim.

## What can and cannot be retained

| Source material | Portable default | Notes |
| --------------- | :--------------: | ----- |
| User-authored messages | Yes | Preserve verbatim subject to explicit redaction |
| Visible assistant replies | Yes | Preserve as source-attributed history |
| Tool names, inputs, outputs, exit state | Usually | Normalize for display/evidence; never re-execute |
| Files touched and current diff | Yes | Verify against the workspace; workspace wins over transcript claims |
| Decisions, rejected approaches, constraints, next action | Yes | Structured checkpoint with provenance |
| Attachments | When locally available | Hash, MIME-type, size, and destination support required |
| Compaction/reasoning summaries visible in the transcript | Usually | Preserve as summaries, not hidden reasoning |
| Source system/developer instructions | Audit only | Never install as destination authority automatically |
| MCP/skills/tool capability state | As a diff | Re-detect on destination; do not claim tools still exist |
| Pending approvals or unfinished tool calls | No | Close as interrupted; destination must authorize again |
| Auth tokens, cookies, API keys, account state | Never | Credentials remain local and vendor-specific |
| Hidden chain-of-thought / unavailable system prompts | No | Not available and not a legitimate portability promise |
| Opaque encrypted reasoning/signatures | Same-vendor only when officially supported | Do not translate, decrypt, or mislabel cross-vendor |
| Live processes, shell state, remote sandboxes | No | Capture observable results; verify current environment |

The complete raw source artifact may still be retained locally or in encrypted
BYO storage for provenance. “Retained in the archive” does not mean “loaded into
the destination model context.”

## Portable continuity capsule v1

The Phase 4 implementation should define a versioned, vendor-neutral capsule.
It has three separately hashable layers:

1. **Raw source** — immutable vendor artifact, encrypted when it leaves the
   machine.
2. **Canonical record** — normalized events and task/workspace state.
3. **Destination projection** — the exact Markdown/JSON/native artifact passed
   to the target, plus a fidelity report.

Conceptual schema:

```yaml
schema: reinstate.continuity-capsule/v1
identity:
  id: <uuid>
  lineage_root: <uuid>
  parent_session:
    agent: claude
    id: <source-session-id>
    artifact_sha256: <digest>
  created_at: <rfc3339>
task:
  goal: <current goal>
  latest_user_intent: <latest uncompleted request>
  constraints: []
  decisions: []
  rejected_approaches: []
  completed: []
  pending: []
  open_questions: []
  next_action: <one concrete action>
workspace:
  project_id: <canonical-project-id>
  root: ${REPO:<id>}
  branch: <branch>
  head: <commit>
  dirty: true
  changed_files: []          # live Git porcelain, portable tokens, bounded
  changed_files_omitted: 0   # entries the bound list could not carry
  tests: []
conversation:
  events: []
  full_history_ref: <private-content-addressed-reference>
capabilities:
  source: {}
  destination: {}
  missing: []
security:
  redactions: []
  source_instructions_are_untrusted_history: true
fidelity:
  overall: normalized
  components: []
```

### Canonical event requirements

Each event needs:

- a stable capsule event ID and source event pointer;
- timestamp and source order;
- actor (`user`, `assistant`, `tool`, `harness`, `unknown`) without pretending
  unlike vendor roles are identical;
- kind (`message`, `tool_call`, `tool_result`, `attachment`, `summary`,
  `checkpoint`, `metadata`);
- typed content blocks and hashes for referenced large/binary content;
- links between tool calls and results;
- original vendor type/name retained alongside any canonical category;
- a portability state: `exact`, `normalized`, `summarized`, `referenced`, or
  `omitted`, with a reason;
- redaction and truncation markers that cannot be mistaken for original text.

Unknown records must be preserved as opaque, hashed source references when
safe. An adapter must not guess their meaning.

### Truth hierarchy

When sources disagree, continuation uses this order:

1. current workspace bytes and Git state;
2. latest explicit user intent and constraints;
3. completed tool results that can still be verified;
4. structured checkpoint;
5. older conversation statements and model plans.

This prevents an obsolete early plan or hallucinated “file changed” message
from overriding the actual repository.

## Handoff pipeline

### 1. Resolve and freeze the source boundary

- Resolve the exact source session and adapter version.
- If the source process is active, request a safe stop or snapshot only through
  a supported mechanism; otherwise capture only the latest complete record.
- Never parse a partially appended JSONL record as complete.
- Hash the source artifact and record the byte boundary.

### 2. Parse without requiring the source model

- Use a version-gated source transcript reader.
- Build normalized visible events and a deterministic activity index.
- Derive files, commands, tests, errors, and the latest user request without a
  network call.
- An agent-assisted summary may improve quality while the source is available,
  but it is optional and provenance-labeled. The quota-exhaustion path cannot
  depend on it.

### 3. Verify the workspace

- Resolve canonical project/path mappings.
- Capture repo, worktree, branch, HEAD, dirty state, changed files, and relevant
  runtime fingerprints.
- Compare transcript claims with the filesystem and Git.
- Refuse automatic launch if the destination points at the wrong project or an
  unsafe/missing worktree.

### 4. Negotiate destination capabilities

Compare source and target support for:

- tool families and approval modes;
- MCP servers, skills, instructions, hooks, and plugins;
- images and other attachments;
- multiple workspace roots, sandbox/network policy, and OS/runtime;
- history import, context size, and supported launch method.

Missing capabilities become visible warnings and capsule fields, never silent
deletions.

### 5. Budget and project the context

Offer explicit history policies:

| Policy | Contents | Use |
| ------ | -------- | --- |
| `checkpoint` | Task state, workspace, changed files/tests, next action | Small/clear tasks or tight target context |
| `balanced` | Checkpoint + recent verbatim turns + representative tool evidence | Default |
| `full` | All portable visible events, usually through a private sidecar reference | Audit/debug; not necessarily all in the model prompt |

Large output belongs in a private, content-addressed sidecar that the
destination can read selectively. Reinstate must report estimated size and any
truncation before launch.

### 6. Preview security and fidelity

`--dry-run` should show:

- source and destination session identities;
- workspace and capability differences;
- counts and sizes of exact, normalized, summarized, referenced, redacted, and
  omitted records;
- every unsupported class that affects continuation;
- the launch route and files that would be written;
- whether any target-native reconstruction is experimental.

### 7. Materialize a new destination session

Preferred routes, in order:

1. a documented target import/create API that accepts the required history;
2. a supported new-session launch with an inspectable capsule file and bootstrap
   prompt;
3. ACP as a transport where the destination agent exposes suitable semantics;
4. exact-version native file/database reconstruction, **experimental only**.

Writing undocumented vendor internals must never be the universal default. It
requires exact-version fixtures, backups, a new destination ID, explicit
consent, and native resume validation. Reinstate never mutates the source
session.

### 8. Acknowledge before acting

The destination bootstrap should first restate:

- current goal and latest user request;
- critical constraints and decisions;
- current changed files and test state;
- missing capabilities or uncertain evidence;
- proposed next action.

Mutation proceeds only after this acknowledgement passes an automated or user
review gate appropriate to the destination harness.

### 9. Record lineage and hand back

Record the new native session ID, destination projection hash, and capsule ID.
Future handoffs form a lineage graph rather than overwriting history. A later
Codex → Claude hand-back starts from the newest verified task/workspace state,
while retaining links to both prior sessions.

## Adapter architecture

Session sync and cross-agent continuation need separate capability contracts.
Recognizing a native file is not enough to claim handoff support.

```text
TranscriptSource
  probe(session) -> SourceCompatibility
  snapshot(session) -> ImmutableBoundary
  parse(boundary) -> CanonicalEvents + ParseReport

HandoffTarget
  capabilities() -> TargetCapabilities
  plan(capsule, policy) -> DestinationPlan + FidelityReport
  materialize(plan) -> PreparedSession
  launch(prepared) -> NativeSessionID
  verify(nativeSessionID) -> ContinuationResult
```

Support is **directional and versioned**. Claude → Codex can be supported while
Codex → Claude is still experimental. Each direction must report independently:

- source parsing;
- task checkpoint generation;
- visible message preservation;
- tool evidence preservation;
- attachment handling;
- target launch route;
- native reconstructed-history support;
- workspace/capability verification;
- tested source and destination versions/OSes.

## Security contract

Cross-agent transfer introduces a new trust boundary. The source transcript can
contain prompt injection, malicious tool output, stale policy, and secrets.

Required rules:

1. Source system/developer messages are preserved for audit only and are never
   promoted to destination system/developer authority automatically.
2. Tool calls/results are inert evidence. No pending or historical action is
   re-executed during import.
3. The destination reloads current project/user instructions from the verified
   workspace and reports conflicts with source-era instructions.
4. Every destination permission, network action, secret lookup, and MCP login is
   authorized again under destination policy.
5. Credential paths and auth stores remain excluded. High-entropy redaction and
   a preview are available before any capsule leaves the machine.
6. Capsule and sidecar files are private (`0600`), outside the repo by default,
   and encrypted before remote sync.
7. Imported content is source-attributed and delimited so quoted instructions
   cannot masquerade as Reinstate or destination policy.
8. Native reconstruction refuses unknown versions and never bypasses a
   destination vendor's security or authentication checks.

See [security-model.md](security-model.md) for the full threat model.

## Delivery plan

### Phase 4A — Contract and fixtures

- Publish the capsule schema, fidelity vocabulary, lineage model, and CLI RFC.
- Split sync adapters from transcript-source and handoff-target capabilities.
- Add synthetic Claude and Codex fixtures for long histories, compaction,
  parallel tools, subagents, attachments, errors, partial writes, and unknown
  record types.
- Add adversarial fixtures for secret leakage and prompt injection.

### Phase 4B — Claude ↔ Codex structured handoff

- Implement both directions using a new-session context-injection route.
- Work with the source agent closed, logged out, rate-limited, or offline.
- Generate deterministic workspace/tool/test evidence without a model call.
- Add `checkpoint`, `balanced`, and `full` policies plus `--dry-run`.
- Require destination acknowledgement and record lineage.

This is the first generally available cross-agent milestone.

### Phase 4C — Reconstructed visible conversation

- Preserve all portable visible user/assistant turns and tool relationships in
  the canonical record.
- Add selective history sidecars and context-budget reports.
- Evaluate exact-version target-native reconstruction for Claude ↔ Codex behind
  an experimental flag.
- Never upgrade reconstructed mode to stable until it beats structured handoff
  on task-continuation evaluations and survives vendor version churn.

### Phase 4D — Additional harnesses

Proposed order:

1. Gemini CLI, because it publishes rich session-management behavior and has a
   local project-scoped history.
2. OpenCode, with its own store and share semantics.
3. Grok Build, Copilot CLI, Cursor/agent CLI, Orca, and others based on public
   format/API stability, user demand, fixture availability, and safe launch
   support.

“Read/index support” can land before “handoff source,” which can land before
“handoff target.” Marketing must use the narrowest implemented state.

### Phase 4E — Ecosystem integration

- Expose the capsule schema/library for third-party agents and ADEs.
- Use ACP where it reduces executor/client integration work.
- Add import/export interoperability tests without turning Reinstate into a
  model router, terminal, agent runtime, or marketplace.

## Release gates

A source/target direction cannot be marked supported until all applicable rows
pass with synthetic data and sanitized real acceptance evidence:

| Gate | Required result |
| ---- | --------------- |
| Source unavailable | Handoff succeeds with source CLI closed and no source API call |
| Quota interruption | Latest complete user intent survives an interrupted final turn |
| Long history | 200+ turns and large tool output produce a bounded, reported projection |
| Workspace truth | Branch/HEAD/dirty files/tests match the destination checkout |
| Cross-OS | Windows ↔ macOS paths remap through canonical project IDs |
| Capability mismatch | Missing tool/MCP/skill is reported before launch |
| Security | Credentials excluded; prompt-injection fixture remains inert |
| Fidelity | Every record class is exact/normalized/summarized/referenced/omitted |
| No duplicate effects | Historical writes/commands are not rerun during import |
| Destination acknowledgement | Goal, intent, constraints, files, test state, and next action are correctly restated |
| Native reconstruction | Original preserved, new ID used, exact versions gated, native resume verified |

## Success metrics

The north star remains **previously started coding tasks successfully resumed
per active user**. Cross-agent continuation adds:

- handoff launch and acknowledgement success rate by source/target pair;
- time from source stop to first useful destination action;
- percentage of handoffs requiring the user to re-explain material context;
- duplicate-work and wrong-workspace incidents;
- task completion after handoff;
- fidelity omissions and capability mismatches by adapter version.

Opt-in telemetry contains metadata and support states only—never transcript,
prompt, source-code, capsule, path, or secret content.

## Risks and mitigations

| Risk | Mitigation |
| ---- | ---------- |
| Vendor format churn | Version-gated parsers, synthetic fixtures, canaries, fail closed |
| N×N adapter explosion | One canonical event/capsule model plus directional target projections |
| Context overload and cost | History policies, selective sidecar, size/token preview |
| Stale conversation conflicts with code | Workspace truth hierarchy and destination verification |
| Prompt injection through imported history | Inert/source-attributed history; never elevate to policy |
| Secret leakage | Hard excludes, redaction preview, private files, E2EE remote storage |
| Corrupt active session | Complete-record snapshot boundaries, source immutability, backups |
| “Same session” marketing overclaim | Mode and fidelity labels in every surface |
| Unstable native DB/file synthesis | Experimental pair/version gate; prefer documented launch APIs |
| Reinstate drifts into an ADE | Agents execute; Reinstate captures, verifies, projects, launches, and records lineage |

## Decisions still requiring implementation RFCs

- Canonical serialization details and schema evolution rules.
- Whether the human-readable projection is generated from canonical JSON or
  stored as a separately signed artifact.
- Token/size estimation per destination and the default `balanced` budget.
- Optional summarizer provider interface and how to guarantee the no-source-model
  fallback remains first-class.
- Exact destination acknowledgement protocol for non-ACP CLIs.
- Retention and deletion behavior for capsule lineages and large sidecars.
- Criteria for graduating any reconstructed native pair from experimental.

Those choices can evolve. The core decision cannot: cross-agent continuation is
an explicit portable handoff with verifiable fidelity, not silent vendor-format
magic.
