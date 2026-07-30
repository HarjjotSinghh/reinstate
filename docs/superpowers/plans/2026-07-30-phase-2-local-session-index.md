# Phase 2 — Local Universal Session Index

**Status:** implemented; automated gates green; physical dual-device acceptance pending
**Date:** 2026-07-30
**Base:** `origin/main` after the stable `v0.1.0` Phase 1 release
**Branch:** `feat/phase2-local-index`

## Outcome

Phase 2 makes Reinstate useful without `init`, object storage, credentials, or
an encryption passphrase:

```text
rein
rein sessions
rein search "stripe webhook retry"
rein inspect claude:<session-id>
rein last
rein resume codex:<session-id>
rein fork claude:<session-id>
```

Claude Code and Codex receive complete local indexing, search, inspection,
native resume, and native fork support. Gemini CLI and OpenCode receive
read-only discovery, search, and inspection. Native resume remains
same-vendor; this phase does not translate transcripts or implement portable
handoffs.

## Authority and scope

`ROADMAP.md`, `docs/product-strategy.md`, and `PRODUCT.md` are authoritative.
Some older material under `references/` used a different phase numbering;
those files are research inputs, not the current delivery contract.

Phase 2 includes:

- a private local derived index across supported agents;
- a no-argument interactive CLI switcher;
- `sessions`, `search`, `inspect`, `last`, `resume`, and `fork`;
- literal search across user prompt text, structured file references, recorded
  branch, project/workspace, agent, and session identity;
- bounded metadata previews that do not dump transcripts;
- configless, offline operation; and
- native Claude/Codex launches in the recorded working directory.

Phase 2 excludes:

- workspace/capability verification or repair (Phase 3);
- cross-agent handoffs or transcript conversion (Phase 4);
- MCP, skills, hooks, plugins, settings, credentials, or configuration
  reconciliation (Phase 5);
- a web console, ACP client, editor, terminal emulator, agent scheduler, model
  router, plugin runtime, or marketplace; and
- Gemini/OpenCode mutation, resume, fork, or sync.

## Baseline rule

Implementation is based on current `origin/main`, not the pre-stable roadmap
branch that was active when planning began. Stable Phase 1 fixes—including
canonical Codex project mapping, live-session-safe restore forks,
concurrent-write protection, and vendor-safe UUID fork identities—must remain
intact.

The original dirty checkout is preserved. Phase 2 is developed in a clean
sibling worktree so unrelated website and media changes are not mixed into the
feature.

## Architecture

Phase 1's `adapter.Adapter` remains the sync contract. Phase 2 adds separate
read/index and native-execution capabilities so read-only agents never receive
dummy export or restore implementations.

```text
vendor session stores
        |
        v
local session sources
  Claude | Codex | Gemini | OpenCode
        |
        v
private derived index
  identity | metadata | user prompts | file refs | source fingerprint
        |
        +---------------------------+
        |                           |
        v                           v
query / resolve / inspect      native launch planner
        |                      Claude | Codex
        v                           |
CLI commands + picker              v
                              vendor executable
```

### Package boundaries

```text
internal/sessionindex/
  model.go          canonical local record, filters, capabilities
  index.go          refresh, search, resolve, ordering
  store.go          versioned private SQLite store and rebuild
  extract.go        bounded, terminal-safe extraction helpers
  claude.go         Claude JSONL source
  codex.go          Codex rollout JSONL source
  gemini.go         Gemini project chat JSON source
  opencode.go       official read-only session-list source
  launch.go         native resume/fork plans and structured process runner

internal/cli/
  sessions.go       Phase 2 commands and line-oriented interactive picker
```

The sync registry remains configured and mutation-capable. The local registry
is config-independent and must not inherit project mappings that hide unmapped
sessions.

### Canonical identity

Every session has a composite reference:

```text
<agent>:<native-session-id>
```

Bare native IDs are accepted only when they resolve to exactly one indexed
session. Ambiguous IDs fail with an actionable request to use the composite
reference. Ordering is deterministic: newest update first, then agent, then
native ID.

### Local index

Use SQLite through a pure-Go driver so release builds remain
`CGO_ENABLED=0`. The index lives at:

```text
$REINSTATE_HOME/cache/session-index-v1.sqlite
```

It is derived local state, created with owner-only permissions, never included
in sync, and safe to delete. Commands refresh it incrementally before reading:

- unchanged source fingerprint: reuse the indexed record;
- append/replace/change: reparse safely and upsert;
- multiple vendor files for one native ID: coalesce deterministic segments;
- deleted source: remove only after that source completes a successful scan;
- malformed individual session: retain other sessions and emit a warning;
- corrupt or incompatible index: rebuild derived state, never modify vendor
  files.

Phase 2 indexes only:

- session identity and source fingerprint;
- timestamps, workspace/project, recorded branch, title/name;
- bounded user-authored prompt text for literal search;
- known structured file path fields; and
- message counts and capability flags.

It does not index assistant reasoning, assistant messages, tool output,
environment dumps, credentials, or auth stores. Individual records and total
derived prompt text are bounded. An incomplete final JSONL line is ignored so
an agent appending concurrently does not poison the index.

### Preview policy

Human output is metadata-first. A preview:

- comes only from a user-authored prompt;
- collapses whitespace;
- removes terminal control sequences and control characters; and
- is capped at 160 Unicode code points.

`sessions` does not print transcript bodies. `search` identifies matching
sessions without printing the matching passage. `inspect` may show the bounded
first-prompt preview but never offers a full transcript-dump mode in Phase 2.

### Search contract

Search is literal and case-insensitive. Multiple query terms are ANDed.
Dedicated flags narrow results:

```text
--agent claude|codex|gemini|opencode|all
--project <fragment>
--branch <fragment>
--file <fragment>
--limit <n>
```

The searchable corpus contains prompt text plus the canonical metadata fields.
This phase does not add embeddings, semantic search, or network calls.

### Native execution

Launch plans use an executable plus argv array—never a shell command string:

| Agent | Resume | Fork |
| --- | --- | --- |
| Claude Code | `claude --resume ID` | `claude --resume ID --fork-session` |
| Codex | `codex resume ID` | `codex fork ID` |
| Gemini CLI | read-only | read-only |
| OpenCode | read-only | read-only |

The child inherits the user's terminal streams and runs from the recorded
workspace. A missing workspace or executable fails before launch. Reinstate
waits for the native child and propagates failure. An `--dry-run` mode prints or
returns the exact structured launch plan without starting an agent.

Local read and native-execution capabilities are separate. Claude/Codex plans
use the exact Phase 2 argv contract and preflight the executor/workspace;
Gemini/OpenCode are read-only by phase contract and return compatibility exit
code 5 for native actions. Physical vendor-version evidence remains a release
gate rather than being inferred from fixture-backed discovery.

### Command semantics

- `rein` / `reinstate`: on a TTY, refresh and show a numbered, searchable CLI
  picker. Inputs support a number to resume, `/text` to filter, `i NUMBER` to
  inspect, `f NUMBER` to fork, and `q` to cancel. A non-TTY fails promptly with
  a `rein sessions --json` hint.
- `rein sessions`: canonical local listing command. `rein list` remains
  backward-compatible for Phase 1 scripts.
- `rein search QUERY`: returns matching session metadata.
- `rein inspect REF`: returns Phase 2 index metadata only. Phase 3 later adds
  environment verification to this surface.
- `rein last`: resumes the newest resumable session globally, with optional
  agent/project filters. `--dry-run` returns the plan.
- `rein resume REF`: launches the exact native session.
- `rein fork REF`: asks the same vendor to branch the session; it never invokes
  cross-agent translation.

JSON modes return deterministic schemas and never mix with native child output.
`resume`, `fork`, and `last` therefore require `--dry-run` when `--json` is
selected.

## Implementation sequence

### Increment 1 — contracts and storage

- Add canonical records, filters, composite references, capability flags, and
  deterministic ordering.
- Add the pure-Go SQLite store, schema versioning, private permissions,
  upsert/delete/rebuild, and corruption handling.
- Test query escaping, ambiguity, deletion, replacement, ordering, and
  permissions.

### Increment 2 — Claude and Codex full indexing

- Reuse existing discovery roots without configured-project filtering.
- Stream JSONL with bounded records and tolerate an incomplete trailing line.
- Extract only user prompt text and known structured metadata/file fields.
- Exclude Claude subagent artifacts.
- Add synthetic macOS, native-Windows, and WSL-shaped fixtures.
- Test malformed, oversized, multipart, appended, and concurrently changing
  records.

### Increment 3 — Gemini and OpenCode read paths

- Discover Gemini `tmp/<project>/chats/session-*.json` metadata defensively.
- Use OpenCode's documented local `session list --format json` read surface
  behind an injectable runner and timeout.
- Advertise discovery/search/inspect only.
- Add deterministic fakes and synthetic fixtures; unit tests never use the
  network.

### Increment 4 — execution and CLI

- Implement exact reference resolution and native launch plans.
- Add injectable process runner tests for argv, cwd, missing executable,
  cancellation, and child failure.
- Add all Phase 2 commands, JSON output, filters, dry-runs, and alias parity.
- Replace the no-argument help failure with the TTY picker while preserving
  explicit non-TTY behavior.
- Keep every Phase 1 command and exit-code contract green.

### Increment 5 — documentation and autonomous acceptance

- Update `README.md`, `ROADMAP.md`, `CHANGELOG.md`, getting started, CLI,
  architecture, security, compatibility, adapter, FAQ, and troubleshooting
  docs.
- Create a release-neutral Phase 2 physical acceptance runbook.
- Create parallel macOS-Claude and native-Windows-Codex autonomous operator
  prompts plus a sanitized evidence schema/template.
- Add doctests that prevent prompt drift and unsafe secret/transcript
  instructions.

## Verification gates

### Local automated gates

```text
gofmt
go test ./internal/sessionindex ./internal/cli ./internal/adapter/...
go test ./...
go test -race ./...
make vet
make verify
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/reinstate
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/reinstate
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/reinstate
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/reinstate
```

The synthetic end-to-end test copies deterministic vendor trees into isolated
temporary roots and uses an injected structured launch runner, then exercises:

```text
sessions -> search -> inspect -> last --dry-run
         -> resume --dry-run -> fork --dry-run
```

It verifies no `config.toml`, remote state, credential request, passphrase
prompt, or network backend is involved.

### Physical acceptance

macOS and native Windows run independently and in parallel against the exact
same commit or signed release candidate. Each device proves:

- clean artifact provenance and full local verification;
- a fresh isolated `REINSTATE_HOME` with no config;
- Claude and Codex discovery, every search dimension, inspect, and last;
- exact-ID native resume and vendor-native fork for both agents;
- picker selection, filtering, inspect, fork, cancel, and non-TTY behavior;
- append refresh, deterministic ordering, ambiguity, missing-agent, malformed
  fixture, and privacy failure paths;
- Gemini/OpenCode read-only behavior when installed; otherwise `NOT TESTED`;
- no storage credentials, passphrase, keyring, or remote objects; and
- Phase 1 regression tests remain green.

The two device reports use `PASS`, `PARTIAL`, `FAIL`, and `NOT TESTED` and are
reconciled once. Unlike Phase 1, there is no serialized cross-device state
machine because Phase 2 has no shared remote state.

## Completion gate

Phase 2 is complete only when:

1. all roadmap commands work without config or cloud access;
2. Claude and Codex full indexing/search/resume/fork pass automated tests;
3. Gemini and OpenCode read paths pass fixture tests and honest physical
   evidence where installed;
4. default output never dumps full transcripts or tool output;
5. all local verification and cross-build gates pass;
6. the macOS and native-Windows autonomous reports reconcile without a
   release-blocking finding; and
7. roadmap/docs claims match the recorded evidence.
