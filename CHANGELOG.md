# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.1] - 2026-08-21

### Changed

- Update the pure-Go SQLite driver from `1.55.0` to `1.56.0`. That driver backs
  the local session index and the read-only view of OpenCode's embedded store,
  so a patch bump is not assumed inert: the storage rows were re-run on both
  platforms against it. Reading an agent's embedded store still creates no
  write-ahead or shared-memory sidecar under that agent's root, an index
  written by the previous driver is read by this one and back again without
  losing a row, and a `v0.4.0`-built index still upgrades in place. No product
  code changed.

## [0.5.0] - 2026-08-21

Universal agent coverage. Reinstate now knows about a catalog of eighteen
coding agents and reads sessions from eleven of them. A structured handoff
projects from five sources and starts a **new** destination session. Same-vendor
native resume, fork, and encrypted sync remain limited to Claude Code and Codex
CLI. Authorized by dual-platform
tagged-artifact acceptance on `v0.5.0-rc.6`: Apple Silicon macOS `PASS` and
native Windows x64 `PASS`, each across the full 150-row matrix
(`docs/testing/results/2026-08-21-macos-phase5-V050RC6.md`,
`docs/testing/results/2026-08-21-windows-phase5-V050RC6.md`).

### Added

- An agent catalog under `internal/agents` with an explicit tier per agent, and
  a conformance suite that holds each descriptor's capabilities to its declared
  tier — so a tier is a statement about what the code actually does.
- `rein doctor --agents`, which emits a redacted `AGENT-PROBE-V1` artifact
  describing where each agent stores sessions, and `--acceptance-matrix`, which
  generates the required rows for a release from the catalog itself.
- Session discovery for six T1 agents — Kimi Code, Qwen Code, Pi, Cursor CLI,
  GitHub Copilot CLI, and Cline — and three T2 handoff sources: Gemini CLI,
  OpenCode, and Grok Build.

### Changed

- OpenCode sessions are read from its embedded SQLite store rather than by
  running the vendor CLI, which could only ever answer for the directory it ran
  in and left write-ahead log files under the agent root.
- An unchanged refresh no longer re-parses every session: each source
  summarises itself from directory metadata first. A warm refresh dropped from
  roughly ten seconds to under one on both acceptance hosts.
- The verified vendor ranges reach Claude Code `2.1.238` and Codex CLI
  `0.149.0`, each raised only after a session created with that version was
  resumed through Reinstate's own launch plan on both platforms.

### Fixed

- Upgrading Reinstate re-reads sessions whose files have not changed. Both
  layers of change detection asked only whether a file had moved, so an
  existing index stayed frozen across an upgrade and a reader fix reached
  nobody until the user's agent happened to write a new session. Half of this
  shipped in `v0.4.0`.
- The agent probe no longer carries a raw content hash into its artifact when
  the hash is not exactly 32, 40, or 64 characters — a Git object store under
  an agent root produced 38-character names that reached the artifact verbatim.
- An agent root environment variable is honoured even when it names a path that
  does not exist, instead of silently falling back to walking the home tree.
- Gemini sessions resolve their project on a case-insensitive filesystem, where
  the CLI records a lower-cased path but hashes the real on-disk case.

Every candidate from `v0.5.0-rc.1` to `v0.5.0-rc.6` is recorded below with the
acceptance evidence that produced it.

## [0.5.0-rc.6] - 2026-08-21

### Fixed

- `rein doctor --agents` no longer carries a raw content hash into a probe
  artifact when the hash is not exactly 32, 40, or 64 characters long. Git
  stores an object as a two-character directory plus a thirty-eight character
  file, and OpenCode keeps a Git object store under each snapshot, so a real
  agent root produced 38-character names that matched none of the fixed-length
  rules and reached the artifact verbatim. Those names are content hashes of
  the operator's own repository, which is exactly what a committed probe must
  not carry. Any hex run long enough to identify content is now collapsed to
  `<N-hex>`; the established `<32-hex>`, `<40-hex>` and `<64-hex>` tokens are
  unchanged so committed artifacts do not churn. Found by physical
  `v0.5.0-rc.5` acceptance on macOS (B4).

## [0.5.0-rc.5] - 2026-08-21


### Changed

- Widen the fail-closed Claude Code compatibility range through `2.1.238` (was
  `2.1.229`). Both acceptance hosts had auto-updated past the old ceiling and
  were refused on resume and handoff, as would every user on a current install.
  The new ceiling rests on dual-platform physical evidence rather than a version
  bump: on macOS and on Windows a session was created with Claude Code
  `2.1.238`, indexed by Reinstate, and resumed through the launch plan Reinstate
  produced; the resumed session returned a token that existed only in the
  original session's history, which a restarted session cannot answer.

- The verified Codex CLI range now reaches `0.149.0`. `0.147.0` was the
  ceiling, so both acceptance hosts — which had auto-updated past it — were
  refused with `native agent version 0.149.0 is outside the verified range`.
  The new ceiling rests on dual-platform physical evidence rather than a
  version bump: on macOS and on Windows a session was created with Codex
  `0.149.0`, indexed by Reinstate, and resumed through the launch plan
  Reinstate produced; the resumed session returned a token that existed only in
  the original session's history. The Claude Code ceiling stays at `2.1.229`
  until the same journey is completed on both platforms.

- A refresh that finds nothing changed no longer re-parses every session. Each
  home-tree source now summarises itself first — every discovered path, its
  modification time and its size — and the digest is compared with the one
  stored from the last successful scan. An identical digest skips parsing for
  that source entirely. On a real macOS tree a warm `rein sessions` refresh
  went from 10.60s to 0.82s, a 92% reduction, where before it saved 1-4%. The
  digest is written only after a scan succeeds, so a failed scan can never
  mark a source as up to date, and a source that cannot summarise itself is
  always scanned (`v0.5.0-rc.4` H4).

- OpenCode sessions are read from its embedded SQLite store instead of by
  running `opencode session list`. The vendor CLI answers only for the
  directory it runs in, so a scan could never observe a second project, and
  invoking it opened OpenCode's database and left write-ahead log and shared
  memory files under the agent root. The store is opened read-only and
  immutable, so no lock is taken and no sidecar is created. Only the `session`,
  `project` and `message` tables are read; the `credential` and `account`
  tables in the same database are never opened (`v0.5.0-rc.4` A10, C1, C6).

- Claude Code, Codex, Gemini, OpenCode, and Grok descriptors declare their
  storage page and dual-platform probe reports. The conformance suite collected
  evidence failures and logged them as escalations instead of failing, so every
  one of those agents shipped without required evidence and a descriptor naming
  a nonexistent path went unnoticed. Evidence now fails the suite like every
  other check (`v0.5.0-rc.4` macOS A5/A9).

### Fixed

- Upgrading Reinstate now re-reads sessions whose files have not changed, so a
  reader fix reaches an existing index instead of waiting for the agent to
  write something new. Both layers of change detection asked only whether a
  file had moved: `ReplaceSource` left a row alone when its path, modification
  time and size matched, and an unchanged source fingerprint skipped the scan
  outright. Neither could answer whether *this build* would read the same bytes
  the same way, and the answer differs every time a reader is fixed. The reader
  is now recorded alongside the fingerprint, and when it changes every row is
  rewritten. Observed on the Windows acceptance host: an index built before the
  Gemini workspace fix served 24 sessions with no workspace indefinitely, and
  heals to 10 resolved workspaces on the first run of the new build. An
  unchanged refresh stays fast — 0.36s against a 10.16s cold refresh
  (`v0.5.0-rc.4` Windows C1-C2, H4).

- Cursor CLI declares its root environment variable, so its sessions can be
  read from a relocated root like every other home-tree agent. `CURSOR_CONFIG_DIR`
  was recorded as unverified; it is now verified to relocate the whole root,
  `chats/` included — a session created under the override was written there
  and the real `~/.cursor` was untouched (macOS, Cursor CLI `2026.08.11`).
  Without this, Cursor was the one indexed agent that could not be pointed at a
  sanitized tree (`v0.5.0-rc.4` macOS C6).

- An agent root environment variable is honoured even when it names a path that
  does not exist. `rein doctor --agents` reported `root_env_set` and then fell
  back to walking the home tree whenever the named root was missing or lacked
  its marker, so a tester who pointed `KIMI_CODE_HOME` or `COPILOT_HOME` at a
  sanitized root that had not been created yet still had their real tree walked
  and written into a committed probe artifact — the exact outcome setting the
  variable is meant to prevent. The override now replaces the home guess
  unconditionally; when the named root is unusable the agent is reported absent
  (`v0.5.0-rc.4` macOS B7).

- Gemini sessions on Windows resolve their project again instead of showing a
  bare 64-character digest. A chat records only `projectHash`, the sha256 of
  the absolute project path, so the path has to be recovered to name the
  project. Gemini records that path lower-cased — in `projects.json` and in the
  `.project_root` marker it now writes beside each session directory — but
  hashes the path in its real on-disk case, so the digest never matched on a
  case-insensitive filesystem. The recorded spelling is now case-corrected
  against the filesystem before hashing, and `.project_root` is read as a
  second source so the join no longer depends on `projects.json` alone. On the
  Windows acceptance host this took Gemini from 0 resolved workspaces to 10
  sessions across 5 distinct projects (`v0.5.0-rc.4` Windows C1-C3, C6, D4).

- Structured handoff from an OpenCode source carries the conversation again.
  The reader looked for a filesystem `message/<id>` tree that current OpenCode
  no longer writes, so every session fell back to a metadata-only boundary and
  the capsule omitted the conversation with `source_bodies_unavailable`. It now
  snapshots and replays the embedded store, reusing the same event builder the
  filesystem layout uses. The store is opened read-only and immutable, so the
  snapshot writes nothing under the agent root (`v0.5.0-rc.4` D1-D4).

- A refusal for an unsupported native agent version names the verified range.
  It said only that the version was "outside the verified range", which leaves
  a user nothing to act on. It now reads, for example, `native agent version
  2.1.238 is outside the verified range 2.1.219 to 2.1.229 inclusive`. An open
  bound is stated rather than omitted (`v0.5.0-rc.4` macOS and Windows E4).

- OpenCode credential and cache paths are excluded from `rein doctor --agents`.
  The OpenCode data root keeps `auth.json` beside its session store and the
  descriptor declared no exclusion set at all, so the credential file's name
  appeared in probe output, which is committed as release evidence. Only the
  name was ever exposed, never the contents. macOS did not show it because
  OpenCode was off the default `PATH` there, so its root never resolved
  (`v0.5.0-rc.4` Windows B5).

- The Windows agent-storage probe wrapper emits the artifact it documents.
  `scripts/testing/agent-storage-probe.ps1` built its argument list inline as
  `@('doctor','--agents','--json') + $args`, which PowerShell passes to the
  binary as the array, a literal `+`, and `$args` separately. The wrapper
  therefore always exited with a usage error and produced nothing, so the
  documented Windows probe route never worked (`v0.5.0-rc.4` Windows B8).

- Structured handoff from a Grok source works again. Every Grok session in a
  real repository failed `capsule validate`, so the T2 handoff was unusable.
  Three causes: the reader never applied the path backstop that Claude Code and
  Codex readers apply, so vendor paths reached the capsule verbatim;
  transcript-claimed file paths were copied into
  `task.files_touched_per_transcript` without being made portable; and
  `task.recent_user_messages` was never bounded, although the adjacent
  latest-intent field was. All 16 in-repository Grok sessions on the test
  device now project cleanly (`v0.5.0-rc.4` macOS D1-D5).

- The block path backstop rewrites paths inside a tool payload carried as JSON
  text. It applied the single-value rule, which cannot see a path sitting on a
  field inside the document, while the capsule validator walks the decoded
  structure and rejected what the backstop had left untouched. This also
  hardens the Claude Code and Codex readers.

- A transcript-claimed path outside the workspace is omitted with the reason
  `path_outside_workspace` rather than emitted as an absolute path, matching
  how live changed files were already handled.

- Gemini sessions report the project and workspace they belong to. A chat that
  records only `projectHash` surfaced that bare 64-character digest as its
  project name and carried no workspace, so Matrix C1 could not see distinct
  projects and C2 had nothing to compare. The hash is the SHA-256 of the
  absolute project path and `projects.json` lists those paths, so the two are
  now joined (`v0.5.0-rc.4` macOS C1/C2).

- `rein doctor --agents` honours an agent's documented root environment
  variable. The override was read and reported through `root_env_set`, then
  ignored whenever the real home root existed, so a tester who pointed
  `KIMI_CODE_HOME` or `COPILOT_HOME` at a sanitized root still had their home
  tree walked and written into a committed probe artifact. `rein sessions`
  already honoured it (`v0.5.0-rc.4` macOS B7).

- A resolved but empty agent root produces a valid `AGENT-PROBE-V1` document.
  The walk's nil results replaced the initialized empty collections, so the
  artifact failed its own validation (`v0.5.0-rc.4` macOS B7).

- `rein resume` and `rein fork` refuse every catalog key below T3 with exit `5`
  and a reason, whether or not the session exists. A T0 key reported exit `1`
  from an unavailable source and a T1/T2 key reported exit `2` for an unknown
  id. A resolved record still refuses with its own read-only reason
  (`v0.5.0-rc.4` macOS A7).

## [0.5.0-rc.4] - 2026-08-20

Fourth Phase 5 candidate. Physical `v0.5.0-rc.3` acceptance on macOS arm64
exercised the core matrix against the tagged artifact and found five defects:
the agent probe was not reproducible between runs, raw content hashes reached
probe output, `resume` and `fork` reported the wrong exit code for a T0 agent,
an agent-filtered query scanned every vendor source, and shell completion
offered no agent keys. Claude Code and Codex CLI remain the only T4/T5
surfaces. Current stable remains `v0.4.0`. This candidate does not authorize
stable `v0.5.0`.

### Fixed

- An agent-filtered query scans only that agent's source. `rein sessions`,
  `rein search`, and `rein last` refreshed every vendor source before applying
  `--agent`, so asking about one agent cost as much as a full refresh and one
  slow source delayed a request that never concerned it. `--agent kimi` went
  from 6.20s to 0.88s cold and 0.02s warm on a 400-session index; an
  unfiltered refresh is unchanged (`v0.5.0-rc.3` macOS H4/H5).
- Shell completion offers the agent keys each flag accepts. No completion
  function was registered, so `--agent`, `--to`, and `--from` completed to
  nothing. Each now offers exactly its own tier's keys: `sessions`/`search`
  T1+, `last` T3+, `handoff --to` T4+, `handoff --from` T2+
  (`v0.5.0-rc.3` macOS H6).

### Fixed

- `rein doctor --agents` output is reproducible again. A dir node and a file
  node can normalize to the same path, and the probe ordered the tree by path
  alone with a non-stable sort, so their order varied per run. Because the
  tree is truncated to a row ceiling after that sort, the instability changed
  which rows shipped, not just their order. Tree and name-shape ordering are
  now total (`v0.5.0-rc.3` macOS A8/B8).
- Long content hashes joined to a vendor prefix are normalized instead of being
  emitted verbatim. Git object names under a marketplace checkout surfaced as
  `pack-<40 raw hex>.idx` in probe output (`v0.5.0-rc.3` macOS B4).
- `rein resume` and `rein fork` refuse a T0 agent with exit `5` and its declared
  tier reason. A T0 agent has no index source, so the attempt previously failed
  during resolution as an unavailable source and exited `1`, while T1 and T2
  agents already refused with exit `5`. T1+ refusals keep reporting the record's
  own read-only reason (`v0.5.0-rc.3` macOS A7/F2).

## [0.5.0-rc.3] - 2026-08-20

Third Phase 5 candidate. Physical `v0.5.0-rc.2` acceptance on macOS arm64 and
native Windows x64 found three T1/T2 sources that no longer matched what their
vendor CLI writes, so Matrix C1 could not observe two distinct projects for any
of them. Claude Code and Codex CLI remain the only T4/T5 surfaces. Current
stable remains `v0.4.0`. This candidate does not authorize stable `v0.5.0`.

### Fixed

- Kimi sessions are indexed again. Kimi Code 0.36.1 writes `createdAt` and
  `updatedAt` in `state.json` as epoch milliseconds; the reader declared both
  as strings, so every session failed to decode and was dropped with a
  `session_read_failed` warning. Both encodings are now accepted
  (`v0.5.0-rc.2` macOS and Windows C1).
- Copilot sessions report their project and branch again. Copilot CLI 1.0.80
  stopped emitting `cwd` inside `events.jsonl`, which left every session at
  project `unknown` with no workspace. The sibling `workspace.yaml` is now
  read as a bounded fallback for `cwd`, `git_root`, and `branch`
  (`v0.5.0-rc.2` macOS and Windows C1/C2).
- OpenCode sessions are named after their directory instead of the opaque
  40-hex `projectId` digest the CLI reports, matching every other source and
  what the agent itself shows (`v0.5.0-rc.2` macOS C2).

## [0.5.0-rc.2] - 2026-08-19

Second Phase 5 candidate after `v0.5.0-rc.1` dual-platform tagged-artifact
acceptance FAILED (macOS 88/150, Windows 93/150). Structured-handoff
projection now pairs `tool_result` with its `tool_call` so Codex
`handoff --dry-run` capsules validate. Claude Code and Codex CLI remain
the only T4/T5 surfaces. Current stable remains `v0.4.0`. This candidate
does not authorize stable `v0.5.0`.

### Fixed

- Structured-handoff projection keeps a `tool_call` with its `tool_result`
  when the call would otherwise fall outside the balanced byte budget, and
  omits a result whose call is absent from the source (reason
  `unmatched_tool_result`). This unblocks Codex `handoff --dry-run` capsules
  that `capsule.Validate` previously rejected as unpaired `tool_result`
  events (`v0.5.0-rc.1` macOS D1).

## [0.5.0-rc.1] - 2026-08-19

First Phase 5 candidate. Ships the agent catalog and `rein doctor --agents`
(including `--json` / `--acceptance-matrix`). **No new handoff destinations
and no new synced agents** — Claude Code and Codex CLI remain the only T4/T5
surfaces. Gemini CLI, OpenCode, and Grok Build stay T2 handoff sources.
**Six new T1 agents:** Kimi Code CLI, Qwen Code, Pi, Cursor CLI, GitHub
Copilot CLI, and Cline (index and search only; resume and fork stay refused).
Every other new catalog agent is T0 with a recorded reason. Generated
acceptance matrix: **150 required rows**. Current stable remains `v0.4.0`.
This candidate does not authorize stable `v0.5.0`.

### Cline at T1

Promoted on 2026-08-19 when a native Windows probe joined the macOS one.
Both platforms write `~/.cline/data/sessions/<slug>/<slug>.json` after
`cline` 3.0.55. `rein sessions --agent cline` lists and searches them.
Resume and fork stay refused. `db/sessions.db` and `*.messages.json` are
not parsed. `cline history --json` listed the session on both platforms
and stays an F2 candidate, not a shipped read API.

Aider gained a macOS probe and a Windows install-path probe; it stays T0
(F4, no home root, no index source).

### Cline and Aider macOS probes

2026-08-19 macOS `AGENT-PROBE-V1` committed for Cline (`cline` 3.0.55) and
Aider (`aider` 0.86.2). Both stay **T0**: native Windows is still missing.

- Cline: live root `~/.cline/data`, marker `sessions`. `cline history --json`
  listed the session. Pretty-printed session JSON has no first-line keys
  in the probe. No index source.
- Aider: binary on PATH; F4 files stay inside the known project
  (`.aider.chat.history.md`). No home root.

### GitHub Copilot CLI at T1

Promoted on 2026-08-19 from committed dual-platform AGENT-PROBE-V1 artifacts.
`rein sessions --agent copilot` lists and searches
`~/.copilot/session-state/<uuid>/events.jsonl`. Resume and fork stay
refused. Windows `session-store.db` / `session.db` are not parsed. A
rename-aside probe showed an old session ID did not return, so this is a
local file tree.

### Qwen Code, Pi, and Cursor CLI at T1

Promoted on 2026-08-19 from committed dual-platform AGENT-PROBE-V1 artifacts
and synthetic macos/windows fixtures. Each agent has an F1 hometree index
source. `rein sessions --agent qwen|pi|cursor` lists and searches them.
Resume and fork stay refused: no device journey has verified native resume.

- Qwen Code: `~/.qwen/projects/<slug>/chats/<uuid-v4>.jsonl`. Runtime
  sidecars (`*-runtime.json`) are not conversations. The Claude reader is
  not reused.
- Pi: `~/.pi/agent/sessions/<slug>/*.jsonl`. Fail-closed version pin stays
  `0.73.1`.
- Cursor CLI: `~/.cursor/chats/<32-hex>/<uuid-v4>/meta.json`. Editor
  `projects/` stays excluded. `store.db` is not parsed.

### Kimi Code CLI at T1

Promoted on 2026-08-17, when a native Windows probe joined the macOS one from
the day before. The Windows device carried five sessions across three projects,
which settled what a single-session macOS run could not: `state.json` has an
identical thirteen-key shape on both platforms, and `session_index.jsonl`
enumerated exactly the five session directories on disk.

`rein sessions --agent kimi` lists and searches them. Resume and fork stay
refused: no device journey has run `kimi -r <id>` against a real session.

### Added

- F1 index sources for Qwen Code, Pi, Cursor CLI, GitHub Copilot CLI, and Cline (`internal/agents/sources/{qwen,pi,cursor,copilot,cline}`) with dual-platform synthetic fixtures.
- `rein doctor --agents` inventory, `--agents --json` (`AGENT-PROBE-V1`), and `--agents --acceptance-matrix`.
- [ADR 0004](docs/adr/0004-universal-agent-coverage.md): universal agent
  coverage, the T0–T5 support-tier ladder, and a single `internal/agents`
  catalog with one descriptor file per agent.
- [Agent support tiers](docs/agent-support-tiers.md): what each tier lets a
  user do and the evidence each tier requires.
- [Agent catalog SDK](docs/adapters/agent-catalog-sdk.md): descriptor
  specification, storage families F1–F5, shared scanners, and the conformance
  suite that enforces a tier claim against committed evidence.
- [Agent storage probe](docs/testing/agent-storage-probe.md): the
  `AGENT-PROBE-V1` redacted evidence contract for `rein doctor --agents`.
- Per-agent storage pages under [docs/session-storage/](docs/session-storage/)
  for twelve Phase 5 candidates, every row `Unverified` pending a device probe.
- [Phase 5 acceptance contract](docs/testing/phase-5-universal-agent-coverage-acceptance.md)
  and the `PHASE5-DEVICE-REPORT-V1` template.
- Execution plan under
  [docs/planning/v0.5.0-universal-agents/](docs/planning/v0.5.0-universal-agents/):
  roster, work breakdown, file ownership, review gates, task cards, and the
  coordinator handover prompt.
- Amp catalog descriptor at T0 (`server_backed`, F5). Session history is not readable locally.
- ZCode catalog descriptor at T0 (`desktop_only`): official Z.ai desktop ADE only; npm `zcode-app-cli` is not a catalog agent.
- OpenHands catalog descriptor at T0 (`server_backed`); conversations stay on the Agent Server / Cloud backend.
- GitHub Copilot CLI catalog entry at T0 (`layout_unverified`); `session-state/` stays unread until a cache-clear/re-login probe.
- Kimi Code CLI catalog descriptor at T0 (`layout_unverified`): dual-platform probes unavailable (`kimi` not installed; no native Windows host).
- Pi catalog descriptor at T0 (`layout_unverified`): F1 JSONL tree, no dual-platform probes, no T1+ claim.
- Qwen Code catalog entry at T0 (`layout_unverified`): official product identified; no dual-platform probes and no reader.
- Gemini CLI stays T2: `gemini --version` parser added; fail-closed range escalated (no maintainer, no dual-platform physical resume).
- Cline catalog descriptor at T0 (`layout_unverified`): F3 expected; no dual-platform probes, no F3 scanner.
- Aider catalog descriptor at T0 (`layout_unverified`, F4): official product identified; no dual-platform probes and no reader.
- Cursor CLI catalog descriptor at T0 (`layout_unverified`): key is the terminal agent; editor agent is out of scope; no dual-platform probes and no reader.
- Roo Code catalog descriptor at T0 (`layout_unverified`): F3 expected; no dual-platform probes, no F3 scanner.
- MiniMax Code catalog key `minimax-code` at T0 (`layout_unverified`). Token Plan API keys are not this agent.
- Gemini CLI fail-closed version pin `0.55.1` (latest stable `@google/gemini-cli`); still T2.
- Pi fail-closed version pin `0.73.1` (latest `@mariozechner/pi-coding-agent`); still T0.

### Changed

- `ROADMAP.md` renumbered: Phase 5 is now universal agent coverage, universal
  configuration moves to Phase 6, Reinstate Console to Phase 7, and team
  continuity to Phase 8.
- `docs/session-storage-map.md` gains an index of the per-agent pages. Sections
  1–5 are unchanged and still cover the five agents with shipped readers.
- `rein doctor --agents` splits its `installed` column: `installed` now reports
  only whether the executable is on `PATH`, and a new `root` column reports
  whether a candidate root resolved.
- Root discovery is marker-gated. `MustRegister` panics when a descriptor
  declares `Storage.Roots` without a `Storage.Marker`, and a declared root only
  resolves when its marker is present. An explicit `RootEnv` or fixture root
  still bypasses the gate.

- First committed `AGENT-PROBE-V1` device evidence, under
  [docs/testing/results/agent-probes/](docs/testing/results/agent-probes/):
  macOS artifacts for Kimi Code CLI, Qwen Code, and GitHub Copilot CLI. Kimi's
  root ambiguity is resolved to `~/.kimi-code` and its `session_index.jsonl`
  confirmed. **No tier moved**: one platform is not dual-platform evidence.
- Antigravity CLI catalog descriptor at T0 (`layout_unverified`). Google
  retired Gemini CLI's individual OAuth path on 2026-06-18 and named it the
  destination, so it is where those users went. It nests inside Gemini CLI's
  root at `~/.gemini/antigravity-cli`, and its documented conversation path is
  named a cache.
- `rein doctor --agents` gains `--agent-timeout`. The probe budget is now per
  agent (default 10s) rather than a single 3s budget for the whole run, and an
  agent that exceeds it is recorded with `timed_out` instead of failing the run.
- `internal/agents/sources/kimi`, an F1 index source for Kimi Code CLI, with
  synthetic fixtures under `testdata/sessionindex/kimi/{macos,windows}`. It
  fails closed on an unknown `state.json` schema version or `wire.jsonl`
  protocol major, excludes subagent trees, and takes the append-only wire log
  as the session's size and mtime authority. **Not registered on the
  descriptor**: Kimi stays T0 until a native Windows probe exists, so the
  source ships tested but unwired.
- The conformance suite enforces the dual-platform probe requirement at T1 and
  above. `docs/agent-support-tiers.md` has always required a macOS **and** a
  native Windows artifact, but the check only counted reports, so one macOS
  file satisfied it. WSL does not substitute for native Windows.

- Qwen Code's discovery marker corrected from `tmp` to `projects`. The probe
  shows conversations at `projects/<slug>/chats/`, so the Gemini-fork
  hypothesis was wrong about the store and about the project-key kind.
- The Gemini CLI descriptor now excludes `antigravity-cli`, `oauth_creds.json`,
  and `google_accounts.json` from its storage walk. Antigravity CLI installs
  into the same root and keeps an OAuth token there on Linux.
- Gemini CLI also excludes `antigravity` and `antigravity-browser-profile`.
  A 2026-08-17 Windows probe drowned in the desktop IDE's Chrome profile and
  never reached `tmp/*/chats`. Those trees are not Gemini CLI sessions.
- Gemini CLI also excludes `config` and `history` so a Windows probe can reach
  `tmp/*/chats` instead of drowning in skills and leaking project folder names.
- Grok Build excludes install and cache trees (`bundled`, `marketplace-cache`,
  `bin`, `downloads`, `docs`) plus `auth.json`. A 2026-08-17 Windows re-probe
  then reached `sessions/` (32 sessions) and is committed as
  [`2026-08-17-windows-grok.json`](docs/testing/results/agent-probes/2026-08-17-windows-grok.json).
- Grok Build also excludes `skills` and `mcp_credentials.json`.
- Qwen Code native Windows re-probe committed as
  [`2026-08-17-windows-qwen.json`](docs/testing/results/agent-probes/2026-08-17-windows-qwen.json):
  two `projects/*/chats/<uuid-v4>.jsonl` sessions after excluding `updates/`.
- Qwen Code macOS re-probe committed as
  [`2026-08-17-macos-qwen.json`](docs/testing/results/agent-probes/2026-08-17-macos-qwen.json):
  a real JSONL conversation whose first-line keys match Windows, plus
  `<uuid-v4>-runtime.json` sidecars. Still T0: no reader. Do not reuse the
  Claude reader.

- Pi macOS AGENT-PROBE-V1 committed as
  [`2026-08-17-macos-pi.json`](docs/testing/results/agent-probes/2026-08-17-macos-pi.json):
  `~/.pi/agent/sessions/<slug>/<slug>-<uuid-v4>.jsonl`, first-line keys
- Pi native Windows AGENT-PROBE-V1 committed as
  [`2026-08-17-windows-pi.json`](docs/testing/results/agent-probes/2026-08-17-windows-pi.json):
  same `sessions/<slug>/<slug>-<uuid-v4>.jsonl` shape as macOS. Still T0: no
  reader.
- Cursor CLI dual-platform chats committed
  ([`2026-08-17-macos-cursor.json`](docs/testing/results/agent-probes/2026-08-17-macos-cursor.json),
  [`2026-08-17-windows-cursor.json`](docs/testing/results/agent-probes/2026-08-17-windows-cursor.json)):
  `chats/<32-hex>/<uuid-v4>/{meta.json,store.db}`. Editor `projects/` is
  excluded. Still T0: no reader.
- GitHub Copilot CLI rename-aside probe committed as
  [`2026-08-17-windows-copilot-cache-clear.json`](docs/testing/results/agent-probes/2026-08-17-windows-copilot-cache-clear.json):
  the old session ID did not reappear in the fresh tree. Still T0: no reader.

### Fixed

- The probe kept UUID filenames that had an extra suffix (`<uuid>.runtime.json`)
  and hyphenated project files, because the UUID matcher ignored a match at
  the start of the stem and `reSafeName` accepted `[A-Za-z0-9._-]`. Those
  now collapse to `<uuid-v4>-runtime.json` and `<slug>.ext`.
- Cursor CLI's probe walked editor `projects/`, `extensions/`, `plugins/`,
  and `skills-cursor/` unless those trees are excluded. The descriptor now
  marker-gates on `chats/` and excludes the rest.
- Gemini CLI and Kimi Code now exclude top-level `skills/` so personal skill
  names do not enter a probe artifact.
- `TestIsolationFSRejectsWritesAndOutsideRoot` left a file handle open, so
  Windows CI failed during `TempDir` cleanup even though the assertions passed.
- The probe emitted repository names. Kimi Code buckets a workspace as
  `wd_<name>_<12-hex>`, where the name is the **basename of the working
  directory**, and the whole component passed through the normalizer intact.
  The earlier macOS artifact was redacted only by accident, because that
  session ran in the home directory, whose basename is the account name — which
  is also why the shape was first misread as carrying a username. A native
  Windows probe produced `wd_portfolio-25_6d65015f0cb0` and exposed it. Such
  components now collapse to `wd_<project>_<12-hex>`.
- The probe would have emitted encoded absolute paths verbatim. Cursor buckets
  projects as `Users-<user>-Documents-Projects-<repo>`, which the token
  normalizer passed through intact because every character in it is
  unremarkable, carrying the home path and repository name into the artifact.
  Such components now collapse to `<path-slug>`. Found while evaluating Cursor
  roots, before any Cursor tree was walked.
- The probe emitted the operating-system account name inside normalized path
  shapes. Kimi Code buckets a workspace as `wd_<user>_<hash>`, and nothing about
  an account name looks like a UUID, hash, or slug, so the normalizer kept it
  verbatim. It is now replaced with `<user>`, and both the probe and CLI leak
  tests assert on it.
- `rein doctor --agents` failed with `context deadline exceeded` and emitted
  nothing once several agents were installed. The 3s budget covered the entire
  run while each installed agent spawns a `--version` subprocess, so the
  evidence tool broke on exactly the machines Phase 5 needs it on.
- `rein doctor --agents` reported Qwen Code and OpenHands as installed on
  machines where neither was installed. Both descriptors declared a home root
  with no marker, so unrelated tooling that created `~/.qwen/skills` and
  `~/.openhands/skills` was enough to resolve the root, and the inventory
  treated root presence as installation. The same record reported
  `executable_on_path: false`. Since Phase 5 device reports are generated from
  this inventory, the bug manufactured evidence for agents that were absent.

## [0.4.0] - 2026-08-16

Phase 4 stable release. Explicit structured handoff continues the same task in a
*new* Claude Code or Codex session. Dual-platform tagged-artifact acceptance
passed on candidate `v0.4.0-rc.11` (Apple Silicon macOS 44/44, native Windows
x64 44/44). Native resume remains same-vendor. Gemini CLI, OpenCode, and Grok
Build remain handoff sources only. Intel macOS and Linux/WSL2 remain optional
and unverified.

### Added

- Ships the Phase 4 structured-handoff surface introduced across
  `v0.4.0-rc.1`–`v0.4.0-rc.11`: `rein handoff`, `handoff list` / `inspect` /
  `export`, `rein resume --with`, picker `h`, local `$REINSTATE_HOME/handoffs/`
  artifacts, and source readers for Claude Code, Codex, Gemini CLI, OpenCode,
  and Grok.

### Fixed

- Destination first-reply must restate the five acknowledgement bullets
  (current goal and latest request, critical constraints, changed files and
  test state, missing or uncertain evidence, proposed next action).
- Execute records lineage before dest Launch; `handoff list` recovers artifact
  directories if `lineage.jsonl` is missing.
- Dest-home workspace trust is materialized for isolated `CLAUDE_CONFIG_DIR` /
  `CODEX_HOME` so dest-ack is not blocked on the TUI trust prompt.

### Changed

- Public installers, compatibility data, and documentation pin stable `v0.4.0`.
- Fail-closed Claude Code range through `2.1.229`; Codex CLI remains
  `0.133.0`–`0.147.0`.

## [0.4.0-rc.11] - 2026-08-15

Eleventh Phase 4 candidate. `v0.4.0-rc.10` was published and physical dual-platform
acceptance FAILED (macOS 41/44 A1/A3/A7, Windows 40/44 A1/A2/A3/A5). Remaining
product defects were dest first-reply missing the five acknowledgement bullets,
lineage written after dest Launch (Mac A7 list empty), and dest TUI folder-trust
hangs. This candidate requires the five-bullet first-reply in bootstrap and
Windows one-line argv, records lineage before Launch, recovers artifact dirs in
`handoff list`, and Materializes dest-home workspace trust. Dest-ack remains
harness (logged-in throwaway dest), not product. Does not authorize stable
`v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.11-agent-verification-prompts.md`.

### Fixed

- Dest first-reply contract is explicit: the bootstrap and the Windows one-line
  `projection.md` argv both require the five acknowledgement bullets as the
  destination's first reply, without the dest-facing "cannot police" hedge
  (rc.10 A1/A3/A5).
- `handoff list` recovers artifact directories when `lineage.jsonl` is missing,
  and Execute records lineage before dest Launch so a killed dest-ack still
  lists the handoff and Claude's pinned dest session (rc.10 macOS A7).
- Dest Materialize records folder trust for the verified workspace in an
  explicit dest home (`CLAUDE_CONFIG_DIR` / `CODEX_HOME`) so throwaway dest-ack
  is not blocked on the TUI trust prompt (rc.10 Windows A2; Codex dest hang).
  Codex `config.toml` project keys with backslashes are literal-quoted, and
  Windows dest homes also get slash-style and lowercased aliases.

## [0.4.0-rc.10] - 2026-08-15

Tenth Phase 4 candidate. `v0.4.0-rc.9` was published and physical dual-platform
acceptance FAILED (macOS 38/44, Windows 38/44). Remaining product defect was
Windows CreateProcess truncating multi-line dest argv (Codex dest-ack A5/A7).
This candidate falls back to the one-line absolute `projection.md` pointer
whenever the briefing contains CR/LF. Dest-ack remains harness (logged-in
throwaway dest), not product. Does not authorize stable `v0.4.0`; stable remains
`v0.3.0`. Adds `docs/testing/v0.4.0-rc.10-agent-verification-prompts.md`.

### Fixed

- Destination argv never includes embedded CR/LF. Windows CreateProcess
  truncated a multi-line Codex bootstrap at the first line, so dest-ack never
  saw `projection.md` and Verify could not match the session. Fall back to the
  one-line absolute `projection.md` pointer whenever the briefing contains
  newlines, not only when it exceeds the argv byte budget.

## [0.4.0-rc.9] - 2026-08-15

Ninth Phase 4 candidate. `v0.4.0-rc.8` was published and physical dual-platform
acceptance FAILED (macOS 38/44, Windows 38/44). RC8 dry-run exit 0 and
`layout_recognized=true` PASSED. This candidate maps a recognized off-PATH
layout to inspect JSON `agent.status=supported` without claiming a verified
version range, and still fail-closes destination launch when the executable is
missing. Dest-ack remains harness (logged-in throwaway dest), not product. It
does not authorize stable `v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.9-agent-verification-prompts.md`.

### Fixed

- Off-PATH `Inspect` with a recognized layout reports `status=supported`
  (`executable_present=false`) instead of `not_installed`, so inspect JSON
  matches layout-only `SUPPORTED` and dry-run exit `0` (R1). Destination launch
  still blocks when the executable is missing.

## [0.4.0-rc.8] - 2026-08-15

Eighth Phase 4 candidate. `v0.4.0-rc.7` was published and physical dual-platform
acceptance FAILED (macOS 38/44, Windows 34/44). RC7 product rows (busy-check,
C8/R6 2.1.230, R4 hang, E5/E6) PASSED. This candidate fixes R1 off-PATH layout
scan and pins Go 1.25.13 for govulncheck. It does not authorize stable
`v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.8-agent-verification-prompts.md`.

### Fixed

- Off-PATH `Inspect` still scans a recognized Claude layout instead of returning
  `StatusNotInstalled` from a `LookPath` miss, so layout-only sources stay
  `SUPPORTED` and dry-run handoff exits `0` rather than `5` (R1).
- Pin the Go toolchain at `go1.25.13` so `govulncheck` is green against the
  stdlib.

## [0.4.0-rc.7] - 2026-08-15

Seventh Phase 4 candidate. `v0.4.0-rc.6` was published and physical dual-platform
acceptance FAILED. This candidate carries those product fixes. It does not
authorize stable `v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.7-agent-verification-prompts.md`.

### Fixed

- Bound Windows process listing (`Get-CimInstance` / `tasklist`) to five seconds
  with a hidden console, and treat a listing error as not-busy so Plan is not a
  5-minute runtime abort (Windows busy-check cascade).
- Fail closed when a read-only handoff can determine Claude `2.1.230` (or any
  other out-of-range version): exit 5 / `UNTESTED` without `--allow-untested`
  (C8, R6).
- Skip leftover capability/runtime observers after a spent verifier deadline so
  a hanging `--version` is Compatibility `UNTESTED`, not Runtime exit 1 (R4).
- Hide and time-bound Windows `taskkill /T` when cancelling version-probe trees.
- Do not block source-only Grok/Gemini/OpenCode read-only preflight on a missing
  native verified-resume layout (E5/E6).

## [0.4.0-rc.6] - 2026-08-14

Sixth Phase 4 candidate. `v0.4.0-rc.5` was published and physical dual-platform
acceptance FAILED. This candidate carries those product fixes. It does not
authorize stable `v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.6-agent-verification-prompts.md`.

### Fixed

- Remap a foreign/`fixture-user` workspace onto the local git checkout only
  when the recorded project leaf matches the cwd repository name, and refuse
  exit 5 / different-repository otherwise (C4).
- Refuse non-TTY destination launch at the start of `handoff`, before index
  open or version probes (F8).

## [0.4.0-rc.5] - 2026-08-13

Fifth Phase 4 candidate. `v0.4.0-rc.4` was published and physical dual-platform
acceptance FAILED. This candidate carries those product fixes. It does not
authorize stable `v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.5-agent-verification-prompts.md`.

### Fixed

- Kill hanging Windows `--version` process trees (`taskkill /T`) so Detect and
  native preflight return `UNTESTED` within the 25s budget instead of waiting
  on grandchild-held pipes (R4).
- Refuse non-TTY destination launch before `Plan` / `LookPath` / version
  probes, not after a spawn-scale delay (F8).
- Remap foreign-OS and synthetic `fixture-user` recorded workspaces onto the
  operator git checkout so Windows os-roots dry-run on macOS (and the reverse)
  emits `${REPO:…}` instead of exit 5 (C5, E5/E6).
- Classify Codex `context_compacted` / `summary` records as `summarized` and
  keep that class in `fidelity.json` (B6).

## [0.4.0-rc.4] - 2026-08-13

Fourth Phase 4 candidate. `v0.4.0-rc.3` was published and physical dual-platform
acceptance FAILED. This candidate carries those product fixes. It does not
authorize stable `v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.4-agent-verification-prompts.md`.

### Fixed

- Treat an explicit empty Claude or Codex home (`CLAUDE_CONFIG_DIR` /
  `CODEX_HOME`, or an adapter `Root`) as layout-supported so a new destination
  session can be planned (flagship A dest Plan).
- Bound hanging vendor `--version` probes (including grandchild-held pipes) so
  Detect and native preflight return `UNTESTED` within about two seconds (R4).
- Parse `REINSTATE_PASSPHRASE_FD` as a 64-bit descriptor so Windows HANDLEs are
  not truncated, and clear vendor isolation env in tests that plant under
  `$HOME/.claude` (Windows `make verify`).
- Persist `--no-launch` handoffs without requiring warning acknowledgements so
  `rein handoff list` can see the rows (G3). Launch still requires acks.
- Keep sidecared `summarized` / `KindSummary` events classified as summarized
  in `fidelity.json` (B6).

## [0.4.0-rc.3] - 2026-08-13

Third Phase 4 candidate. `v0.4.0-rc.2` was published and physical dual-platform
acceptance FAILED. This candidate carries those product fixes. It does not
authorize stable `v0.4.0`; stable remains `v0.3.0`. Adds
`docs/testing/v0.4.0-rc.3-agent-verification-prompts.md`.

### Fixed

- Refuse a structured handoff when the operator cwd is a different Git
  repository than the source session (C4).
- Fail closed on non-TTY destination launch before `LookPath` or child spawn
  (F8).
- Treat Grok, Gemini, and OpenCode as valid source agents for the busy check
  so a Grok-sourced dry-run no longer exits `1` with `unsupported agent`.
- Classify a timed-out source version probe as `UNTESTED` (exit `5`), not a
  runtime error, and do not block a read-only handoff when the source
  executable is off `PATH` but the layout is still readable.
- Register `--no-redact` on `rein handoff` (still refused for Grok).
- Honor `CLAUDE_CONFIG_DIR` and `CODEX_HOME` in `rein list` and adapter
  detection.
- Include omitted task fields in `fidelity.json`, keep omitted events omitted
  in the fidelity report, and print `projection_events` so checkpoint policy
  is inspectable.
- Discover Claude MCP servers from `settings.json` / `settings.local.json`,
  and require acknowledgement of destination MCP/skill gaps.
- Checkpoint policy stores sidecar references only — no verbatim event bodies.
- Validate website deployment tag calendar dates without Node so Windows `sh`
  doctests see `invalid website deployment date`.
- Promote `github.com/spf13/pflag` to a direct `go.mod` requirement.

## [0.4.0-rc.2] - 2026-08-13

Second Phase 4 release candidate. `v0.4.0-rc.1` was published and its physical
dual-platform acceptance **failed**; this candidate carries the fixes for what
that run found. Claude Code was unusable as a handoff source on every real
installation, reader-emitted absolute paths were rejected by capsule
validation, `changed_files` was never populated so every destination was told
the repository was clean, a version probe that timed out was accepted as if the
agent were absent, and any message beginning with a slash aborted the handoff.

Nothing about the Phase 4 surface itself changed: this is still explicit
structured handoff of the same task into a *new* Claude Code or Codex session
on top of the stable `v0.3.0` verified-resume surface, not a cross-agent
resume, and the projection remains deliberately lossy and visible. Dual-platform
tagged-artifact acceptance on Apple Silicon macOS and native Windows x64 is
pending for this candidate; it does not authorize stable `v0.4.0`, and stable
remains `v0.3.0`.

### Changed

- Widen the fail-closed Claude Code compatibility range through `2.1.229` (was
  `2.1.228`). Claude Code auto-updates: during the `v0.4.0-rc.1` window the
  macOS host moved `2.1.225` -> `2.1.228` mid-run and the Windows host reached
  `2.1.229`, so the ceiling widened one release earlier was already stale before
  physical acceptance could start. `2.1.229` is covered by the range but has not
  completed dual-platform tagged-artifact acceptance; the Codex CLI range is
  unchanged at `0.133.0`-`0.147.0`.
- Claude handoff fixtures no longer contain a synthetic `version` file, and both
  Claude and Codex gained an `absolute-paths` fixture, so the suite exercises
  what a real installation looks like.
- Claude and Codex gained a `slash-commands` fixture whose messages open with
  `/init`, include `/compact`, and name absolute paths in prose, so the capsule
  cannot silently re-acquire the rejection of ordinary conversation.
- The Phase 4 acceptance contract now requires all five vendor isolation
  variables (`REINSTATE_HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`,
  `GEMINI_CLI_HOME`, `GROK_HOME`) and a check that the first index refresh
  contains only run-created sessions. A `v0.4.0-rc.1` run omitted `GROK_HOME`
  and indexed the operator's real `~/.grok` tree; the product was correct and
  the runbook was not. OpenCode still has no override, so its rows are
  `NOT TESTED` or explicitly recorded as un-isolated.
- `docs/testing/v0.4.0-rc.2-agent-verification-prompts.md` is the acceptance
  dispatch for this tag, and it records the `v0.4.0-rc.1` findings so the rerun
  re-verifies them instead of rediscovering them.

### Fixed

- Handoff source probing no longer depends on a `<agent-root>/version` file that
  real Claude Code installations never create. Claude and Codex transcript
  readers now share one documented contract: an unrecognized layout is
  `UNSUPPORTED`, a determinable version outside the verified range is
  `UNTESTED`, and a version that cannot be determined stays usable, so a handoff
  still works when the source agent is closed, logged out, rate limited, or
  uninstalled. Versions are resolved from the installed executable through
  `internal/agentcheck`, the same mechanism `rein inspect` reports, instead of a
  second source of truth. Previously every real Claude Code installation was
  reported `UNTESTED` and `rein handoff claude:<id>` exited 5, while
  `rein inspect` called the same agent supported in the same invocation, and the
  Codex reader applied no version check at all.
- A source agent whose version probe times out is no longer accepted as if it
  had no version at all. The bounded `--version` probe reported "unknown" when
  it merely ran out of time, and the reader contract answers "unknown" with
  SUPPORTED — the branch that exists so a handoff still works when the source
  agent is uninstalled. An installed, determinable, out-of-range agent was
  therefore accepted silently whenever the machine was briefly busy, which real
  agent CLIs can cause on their own since they are language runtimes that can
  exceed a two-second budget. A timed-out probe is now measured once more, and
  a measurement that still fails is reported as a failed measurement rather
  than an absent one: installed-but-unread is UNTESTED, refused without
  `--allow-untested`. An agent that is genuinely not installed still resolves
  to SUPPORTED, unchanged.
- A handoff now tells the destination which files actually changed. The
  workspace probe keeps the pathnames behind the counts it already computed,
  `BindWorkspace` rewrites each one into a `${REPO:<id>}/…` token, and the
  capsule, `projection.md`, and the destination bootstrap all carry the list.
  Previously every handoff reported `Changed files: (none)` and emitted no
  `changed_files` key at all, even over a dirty working tree — the destination
  was told the repository was clean when it was not. The list is capped at 64
  paths so a large dirty tree cannot exhaust the capsule or the 8 KiB bootstrap
  budget; whenever entries are dropped, at the cap or under argv pressure, the
  count of omitted entries is rendered instead of a silently short list.
- A transcript's file claims are no longer marked
  `evidence_conflicts_with_workspace` unless live Git actually produced a
  complete changed-file list. An unavailable, uncertain, or capped observation
  is missing evidence, not counter-evidence, and previously every handoff built
  without a live observation contradicted its own transcript.
- Transcript readers now rewrite the paths they lift out of vendor tool calls,
  tool results, and attachments into portable `${REPO:<id>}` / `${HOME}` tokens
  before they reach a capsule. A path outside every configured root becomes a
  stable, non-reversible `${EXTERNAL:<digest>}/<name>` token rather than an
  absolute path. Previously a Claude session that had read a file failed the
  handoff with `capsule: absolute filesystem path is not allowed`.
- Capsule canonicalization no longer treats prose as a filesystem path. It now
  checks the capsule's path-typed fields — workspace root, changed files,
  transcript file references, block paths and refs, sidecar references, and the
  path-typed keys inside tool arguments — instead of every string in the
  document, and reports the offending field by name. Message bodies, derived
  goals, and user intents are carried exactly as written. Previously any message
  beginning with a slash command (`/init`, `/compact`, `/clear`) or any sentence
  naming an absolute path aborted the handoff with
  `capsule: absolute filesystem path is not allowed`. This matches the contract
  `docs/compatibility.md` already states for path rewriting: known structural
  fields are rewritten, prose is left unchanged.

## [0.4.0-rc.1] - 2026-08-12

First Phase 4 release candidate. It adds explicit structured handoff of the
same task into a *new* Claude Code or Codex session on top of the stable
`v0.3.0` verified-resume surface. This is not a cross-agent resume: nothing
reconstructs the original session, and the projection is deliberately lossy and
visible. Dual-platform tagged-artifact acceptance on Apple Silicon macOS and
native Windows x64 is still pending; this candidate does not authorize stable
`v0.4.0`, and stable remains `v0.3.0`.

### Added

- `rein handoff` for explicit structured handoffs into a new Claude Code or
  Codex session, including deterministic `--dry-run`, launch-free
  materialization, `rein resume --with`, picker handoff actions, and local
  `handoff list`, `inspect`, and `export` history.
- A model-free handoff pipeline and canonical continuity capsule for Claude
  Code, Codex CLI, Gemini CLI, OpenCode, and Grok Build sources. Gemini,
  OpenCode, and Grok are source-only in `v0.4.0-rc.1`; destination launch is
  limited to Claude Code and Codex through their documented CLIs.
- A private, local-only `$REINSTATE_HOME/handoffs/` artifact store with
  append-only lineage, owner-only protection, and a hard sync exclusion. The
  handoff path never writes vendor-internal session files.
- Phase 4 adversarial and golden coverage for inert transcript evidence,
  source-instruction exclusion, delimiter escaping, bounded reads, secret
  redaction, capsule determinism, output parity, and 200-turn projection
  ceilings.
- Claude Code handoff destination (`internal/handoff.ClaudeTarget`): ADR 0003
  argv `claude --session-id <uuid-v4> "<bootstrap>"` in the verified workspace,
  pinned-ID verification under this device's project key, and R5 fail-closed
  collision refusal after bounded UUID regeneration (no vendor-internal writes).
  `sessionindex.OperationHandoff` lets `ExecLaunchRunner` apply the same TTY and
  identity guards to destination launches.
- Codex CLI handoff destination (`internal/handoff/target_codex.go`): launches
  `codex "<bootstrap>"` in the verified workspace, reconciles the
  vendor-assigned session ID after launch (resolved / unresolved / ambiguous),
  and falls back to a `projection.md`-only bootstrap when argv exceeds
  `DefaultMaxArgvBytes` (R6). Never writes vendor-internal Codex files.
- Handoff projection renderer (`RenderBootstrap`, `RenderProjection`,
  `RenderJSON`) with imported-history framing, delimiter escape, source
  system/developer exclusion, and an 8 KiB bootstrap ceiling
  (`internal/handoff/projection.go`; goldens under
  `testdata/handoff/golden/projection/`).
- Handoff context policies (`checkpoint` / `balanced` / `full`) with
  newest-first projection budgeting, visible truncation markers, deterministic
  token estimates (`ceil(utf8_bytes / 4)`), and sidecar references for every
  excluded event (`internal/handoff` policy + estimate; `capsule.SidecarRef`).
- Claude Code transcript reader (`internal/transcript`) that snapshots complete
  JSONL boundaries and maps user/assistant/tool/summary/attachment/unknown
  records into canonical capsule events, with synthetic fixtures under
  `testdata/handoff/claude/` and R8 attachment guidance in
  `docs/session-storage-map.md`.
- Codex CLI transcript reader (`internal/transcript/codex.go`): maps rollout
  JSONL into canonical capsule events with `event_msg`-over-`response_item`
  dedup, filename-UUID session identity for forks, and R4
  `vendor_opaque_state` omission for reasoning / encrypted reasoning items.
  Synthetic fixtures under `testdata/handoff/codex/`.
- OpenCode source-only transcript reader (`internal/transcript`): MessageV2
  storage tier under `storage/message/` + `storage/part/`, with metadata
  fallback via `opencode session list` when bodies are unavailable
  (`source_bodies_unavailable`). Windows uses the documented XDG data root
  (`%USERPROFILE%\.local\share\opencode`), not `%LOCALAPPDATA%`.
- `internal/handoff.BindWorkspace` binds Phase 3 preflight workspace truth into
  a continuity-capsule workspace with `${REPO:<id>}` portable path tokens.
  Blocked preflight reports surface as `handoff.BlockedError` with the same
  exit codes Phase 3 uses; warning reports still require acknowledgement.
- Gemini CLI source-only transcript reader (`internal/transcript/gemini.go`)
  for Phase 4 handoff capsules: legacy `messages[]` JSON and JSONL+`$set`,
  vendor-aligned `$rewindTo` replay (exclusive of the target id), and
  `kind:subagent` exclusion. Fixtures under `testdata/handoff/gemini/`.
- Phase 4 planning set for `v0.4.0` cross-agent handoff: the continuity-capsule
  design and ADR 0002, a user-facing handoff contract, a per-OS local session
  storage map for Claude Code, Codex, Gemini CLI, OpenCode, and Grok Build, the
  release-neutral acceptance matrix, ADR 0003 fixing `v0.4.0-rc.1` scope and
  launch route, and the architecture plus 27-packet execution plan.
  Documentation only; no behavior changes.

### Changed

- Align user-facing docs, setup prompts, and website release notices with
  stable `v0.3.0` dual-platform tagged-artifact acceptance PASS, and point
  Homebrew/Scoop install routes at the `0.3.0` formula and bucket manifests.
- Widen the fail-closed Claude Code compatibility range through `2.1.228` (was
  `2.1.227`) so both `v0.4.0-rc.1` physical acceptance hosts run an in-range
  Claude Code install instead of exiting `5` on every Claude row. The Codex CLI
  range is unchanged at `0.133.0`–`0.147.0`. As with `v0.3.0-rc.6`, the range
  moves before the evidence: `2.1.228` is now covered by the fail-closed range
  but has not completed dual-platform tagged-artifact acceptance, which the
  `v0.4.0-rc.1` run supplies. Versions above `2.1.228` remain `UNTESTED`.

### Fixed

- Handoff artifacts written outside the store — destination planned files and
  `handoff export --out` / `handoff --export` — are now owner-only on Windows.
  They went through a plain `0600` write, which Windows ignores, leaving the
  inherited DACL in place; they now use the same protected DACL as the rest of
  `$REINSTATE_HOME/handoffs/`.

## [0.3.0] - 2026-08-11

Phase 3 stable release: verified resume for Claude Code and Codex on Apple
Silicon macOS and native Windows x64. Dual-platform tagged-artifact acceptance
passed on candidate `v0.3.0-rc.7` (`rc7_tagged_artifact_acceptance=PASS`), then
again on the published stable tag (`stable_v0.3.0_authorized=true`).

### Added

- Verified resume and fork with environment preflight, exact warning
  acknowledgements, and prelaunch baselines for Claude Code and Codex.
- Deterministic Phase 3 local smoke covering exit policies, capability
  content-free checks, fork/alias parity, and repository safety.

### Fixed

- Windows Ctrl+C at the environment-warning prompt returns safety exit `7`.
- Non-TTY native launch fails closed unless an explicit local-smoke override is set.
- Capability probe incompleteness demoted from acknowledgement-forcing warnings
  where appropriate; cancelled probes remain blocking.

### Changed

- Claude Code fail-closed range through `2.1.227`; Codex CLI through `0.147.0`.

## [0.3.0-rc.7] - 2026-08-11

Phase 3 release candidate after `v0.3.0-rc.6` dual-platform tagged-artifact
acceptance failed (macOS 16 PASS / 12 FAIL / 4 NOT TESTED) on real-launch
baseline, authenticated same-vendor resume/fork, capability mutation coverage,
and required TTY/picker evidence. RC7 packages the post-RC6 harden stack for
retest. Fresh Apple Silicon macOS and native Windows x64 tagged-artifact
acceptance is required; this candidate does not authorize stable `v0.3.0`.

### Fixed

- Handle Ctrl+C at the environment-warning prompt as a deterministic safety
  refusal on Windows, returning exit `7` without launching the vendor instead
  of allowing the console to terminate Reinstate with `0xC000013A`.
- Fail closed before spawning a native agent when stdin is not a TTY, with a
  clear safety exit pointing operators at a real terminal or `--dry-run`.
  Deterministic local smoke may set `REINSTATE_ALLOW_NON_TTY_LAUNCH=1`.

### Changed

- Treat incomplete capability probe diagnostics (for example symlink-skipped
  managed discovery) as informational checks. They still appear on environment
  reports but no longer require `--allow-environment-warning` acknowledgements
  on every resume; only cancelled/deadline probes remain blocking.
- Honor `CLAUDE_CONFIG_DIR` and `CODEX_HOME` when building the default preflight
  capability discovery roots so throwaway agent homes stay isolated from the
  operator ambient trees for those roots.

### Tests

- Expand deterministic `./bin/rein` Phase 3 local smoke with fork dry-run,
  alias parity, missing-workspace exit `5`, content-free skill/instruction/MCP
  capability rows, branch divergence, and same-path repository replacement
  exit `7`.

## [0.3.0-rc.6] - 2026-08-11

Phase 3 release candidate after `v0.3.0-rc.5` dual-platform tagged-artifact
acceptance failed with host agent versions outside the fail-closed ranges
(Claude Code `2.1.225`/`2.1.227` and Codex CLI `0.147.0`). RC6 widens those
inclusive ranges for retest on current primary-host installs. Fresh Apple
Silicon macOS and native Windows x64 tagged-artifact acceptance is required;
this candidate does not authorize stable `v0.3.0`.

### Changed

- Expand the fail-closed Claude Code compatibility range through `2.1.227`
  (was `2.1.220`) and the Codex CLI range through `0.147.0` (was `0.146.0`),
  covering current Mac and Windows primary-host installs used for Phase 3
  retest. Versions above the new maxima remain `UNTESTED`.

## [0.3.0-rc.5] - 2026-08-08

Corrective Phase 3 release candidate after the signed `v0.3.0-rc.4` release
workflow failed before publication. The RC4 draft remained unpublished and
unattested, and the live public installers stayed on `v0.3.0-rc.3`. RC5
carries the Windows-first RC4 product fixes plus a portable PowerShell verifier
that is exercised on Ubuntu before tagging. Fresh tagged-artifact acceptance
on both mandatory devices is pending; it does not authorize stable `v0.3.0`.

### Fixed

- Make the PowerShell release-artifact verifier select native `tar.exe` only
  on Windows and the host `tar` application on other PowerShell platforms;
  exercise that exact gate in pull-request release packaging before tagging.

## [0.3.0-rc.4] - 2026-08-08

Corrective Phase 3 release candidate after `v0.3.0-rc.3` failed native Windows
x64 tagged-artifact acceptance on the PowerShell 5.1 staging parser and human
output privacy gates. RC4 was developed and smoke-tested Windows-first, then
verified on Apple Silicon macOS. Its signed-tag release workflow failed during
Ubuntu PowerShell artifact verification before publication or attestation, so
RC4 has no tagged-artifact device acceptance and does not authorize stable
`v0.3.0`.

### Fixed

- Fix Windows PowerShell 5.1 parse failure in `stage-release-assets.ps1` when
  interpolating `$resolvedDist:` (RC3 artifact-gate blocker).
- Fully redact absolute workspace paths in human `inspect` / dry-run output,
  including Windows paths outside the canonical user home and sibling paths
  that share a configured-home prefix (RC3 privacy finding on installed-artifact
  human inspect).
- Tolerate GoReleaser metadata records without `extra` under PowerShell strict
  mode, and use native `tar.exe` for drive-qualified archives when MSYS2 is on
  `PATH` (RC3 Windows artifact-gate blockers).
- Duplicate configured automation passphrase descriptors before reading them,
  so Windows handle reuse cannot invalidate unrelated Go runtime handles.
- Preserve the exact parent preflight deadline across observer probes instead
  of creating an earlier nested timeout.

## [0.3.0-rc.3] - 2026-08-07

Corrective Phase 3 release candidate after `v0.3.0-rc.2` failed native Windows
x64 tagged-artifact acceptance (Codex executable trust and snapshot/PowerShell
release gates). Native Windows x64 acceptance failed again on the PowerShell
5.1 staging parser and human-output path privacy gates. It does not authorize
stable `v0.3.0`.

### Fixed

- Harden Windows trusted executable resolution for real Codex/Claude installs:
  strip quoted PATH entries, keep PATHEXT shims even when host PATHEXT is
  incomplete, fall back when EvalSymlinks fails, and stop collapsing the trust
  boundary to the volume root on unreadable ancestors (RC2 Windows blocker).
- Fix PowerShell `stage-release-assets.ps1` to resolve GoReleaser paths as
  repository-root-relative `dist/...` entries, matching the shell stager.

### Added

- Add `scripts/snapshot.ps1` for native Windows GoReleaser snapshots with
  explicit tag env vars and non-masked exit codes.

## [0.3.0-rc.2] - 2026-08-07

Corrective Phase 3 release candidate after `v0.3.0-rc.1` failed native Windows
x64 tagged-artifact acceptance. Native Windows x64 acceptance for this candidate
failed again (Codex trust and snapshot/PowerShell gates). It does not authorize
stable `v0.3.0`.

### Fixed

- Resolve Windows vendor tools through PATHEXT so extensionless names such as
  `codex` and `claude` select trusted `*.exe` / `*.cmd` shims outside the
  workspace boundary (RC1 native Windows blocker).

### Added

- Add PowerShell-native release gates with full artifact/SBOM/source parity:
  `scripts/check-release-artifacts.ps1`, `scripts/stage-release-assets.ps1`, and
  `scripts/check-release-binary-identity.ps1`, so Windows acceptance no longer
  depends on `sha256sum` / `unzip` / `jq`.
- Document the pinned native Windows acceptance host and RC2 device prompts in
  `docs/testing/windows-acceptance-host.md` and
  `docs/testing/v0.3.0-rc.2-agent-verification-prompts.md`.

## [0.3.0-rc.1] - 2026-08-05

First Phase 3 release candidate. It adds verified resume to the Phase 2 local
continuity surface. Apple Silicon macOS tagged-artifact acceptance passed;
native Windows x64 failed. This candidate does not authorize stable `v0.3.0`.

### Added

- Add deterministic verified-resume reports for workspace/repository state,
  same-vendor executable compatibility, instruction/skill/MCP presence, and
  recognized Node/Go runtimes across `inspect`, dry-runs, direct launches, and
  the interactive picker.
- Add exact invocation-scoped `--allow-environment-warning CHECK_ID`
  acknowledgments and private `reinstate_prelaunch_observed` baselines saved
  only after a successful same-vendor native child exits.
- Add local-only, privacy-safe repository fingerprints, capability transports,
  runtime declarations, stale-source checks, and deterministic report-bearing
  compatibility/safety/runtime refusals.

### Changed

- Make bounded user-prompt indexing linear instead of quadratic for long
  Claude, Codex, and Gemini sessions, with reproducible 1,000-record CLI/index
  benchmarks and tagged-artifact latency gates for issue #96.
- Document the stable `v0.2.0` package-channel rollout and advertise the
  physically verified Homebrew route on Apple Silicon macOS.
- Move the local continuity index to versioned
  `cache/session-index-v2.sqlite` storage so Phase 3 baselines cannot be
  destroyed by an older Phase 2 binary.
- Coordinate derived-index lifetime/rebuild operations through an owner-only
  `.lock` file and serialize writers through an owner-only `.write.lock` file.

### Fixed

- Emit schema-valid WinGet multi-file manifests and validate them with the
  pinned official WinGetCreate binary during Windows release packaging.
- Let a reviewed package-promotion workflow repair registry metadata for an
  immutable release without moving its signed tag or rebuilding its binaries.

### Security

- Keep environment verification local-only and shell-free; hash repository
  identities; omit dirty filenames and configuration values; bound every
  subprocess/config read; reject unverified agent versions and hard
  repository mismatches; and serialize derived-index writes across concurrent
  `rein` and `reinstate` processes.
- Pin Git probes to the physically discovered repository/worktree, disable
  config includes and repository-controlled executable paths, and require an
  explicit warning acknowledgment whenever working-tree certainty cannot be
  established without trusting repository behavior or a racy observation.

## [0.2.0] - 2026-08-05

Second stable release. It ships the Phase 2 local continuity surface and the
RC3 package-distribution pipeline without changing the runtime adapter code
that passed the complete RC2 physical matrix on Apple Silicon macOS and native
Windows x64.

Stable support for this release is deliberately limited to those two verified
platforms. Intel macOS and Linux/WSL2 artifacts are built, checksummed,
SBOM-covered, and attested, but remain preview/unverified until their deferred
physical acceptance issues close.

### Added

- Add configless local session discovery, literal search, bounded metadata
  inspection, last-session selection, same-vendor native resume/fork, and the
  interactive numbered switcher.
- Add full local capabilities for Claude Code and Codex plus read-only Gemini
  CLI and OpenCode discovery.
- Add provenance-gated npm and native package artifacts together with opt-in,
  protected publication workflows for additional package managers.

### Changed

- Embed the full 40-character source commit in release binaries and keep normal
  verification CGO-free except for the explicit race gate.
- Record the v0.2.0-only limited-platform waiver: Apple Silicon macOS and
  native Windows x64 are verified; Intel macOS and Linux/WSL2 remain preview.

- Record the live package-channel rollout state and add explicit stable
  `v0.2.0` publication and post-publication documentation checklists.

## [0.2.0-rc.3] - 2026-08-02

Third Phase 2 release candidate. It keeps the RC2 product behavior and release
identity fixes unchanged while adding opt-in, provenance-gated distribution
through popular language and operating-system package managers.

### Added

- Add verified package payloads and publication workflows for npm, JSR,
  Homebrew, Chocolatey, Scoop, WinGet, and AUR, plus native Debian, RPM,
  Alpine, and Arch release packages and a maintainer onboarding guide.

## [0.2.0-rc.2] - 2026-08-02

Second Phase 2 release candidate. It supersedes `v0.2.0-rc.1`, whose
native-Windows tagged-artifact certification stopped because release binaries
embedded only a shortened commit and the required verification run inherited
CGO into ordinary Windows tests. RC2 keeps the Phase 2 product behavior
unchanged while correcting those release gates.

### Fixed

- Embed the full 40-character source commit in local and GoReleaser binaries,
  and execute a packaged artifact during release validation so shortened or
  otherwise incorrect build provenance fails before publication.
- Keep ordinary verification builds CGO-free and enable CGO explicitly only
  for the race gate, avoiding inherited-CGO Windows runtime handle failures;
  also make the formatting failure output portable on native Windows.

## [0.2.0-rc.1] - 2026-08-01

First Phase 2 release candidate. Development acceptance passed all 30 required
rows on macOS and native Windows at product commit `b952d38`. Stable promotion
remains blocked until the installed artifacts from this signed tag pass the
tagged-artifact matrix on both devices.

### Added

- Add the Phase 2 configless local continuity surface: a private derived
  session index plus `sessions`, literal `search`, metadata-only `inspect`,
  `last`, same-vendor `resume`, same-vendor `fork`, and the no-argument
  numbered switcher.
- Add full local read/execution capability contracts for Claude Code and Codex,
  with read-only Gemini CLI and OpenCode discovery paths. Composite
  `agent:native-id` references prevent cross-vendor ambiguity; bare IDs are
  accepted only when unique.
- Add a release-neutral Phase 2 physical acceptance runbook, parallel macOS
  Claude Code and native-Windows Codex operator prompts, a sanitized report
  template, prompt-contract doctests, and the completed development-acceptance
  reports. All 30 required rows passed on both devices at `b952d38`.
- Add a native-Windows regression gate proving that the real Claude
  resume/fork launch path preserves stdin, stdout, exact argv, and cwd through
  the vendor's `.cmd` shim.
- Close the landing page with a clear continuity-tool comparison and an
  accessible macOS, Linux, and Windows installation call to action.
### Changed

- Expand the fail-closed Codex CLI compatibility range through `0.146.0`, the
  stable vendor version exercised by the completed Phase 2 physical matrix on
  both macOS and native Windows.
- Make local discovery, search, and inspection independent of `rein init`,
  object storage, credentials, encryption passphrases, project mappings, and
  backend access. `rein list` remains available for Phase 1 compatibility;
  `rein sessions` is the canonical configless local listing command.
- Correct the Phase 2 acceptance contract for configless `rein list`: it may
  succeed with its existing Phase 1 output, but must not be redefined as an
  alias of the canonical `rein sessions` command.
- Coalesce multiple local rollout files that belong to one native session ID
  into one deterministic index record instead of treating vendor continuation
  segments as an ambiguity or aborting the source refresh.
- Close Phase 1 in the roadmap after stable `v0.1.0`; record Phase 2's green
  automated implementation and two-device development acceptance while keeping
  tagged-artifact release acceptance explicit.
- Replace the legacy DevSync research PNGs with canonical Reinstate SVG
  diagrams across the README, repository documentation, research references,
  landing page, and website documentation.

### Fixed

- Read OpenCode's top-level `updated` and `created` epoch timestamps so current
  sessions retain their real ordering and remain visible in default listings.
- Make all canonical Reinstate diagrams transparent and theme-aware so they
  blend into light and dark documentation backgrounds without losing contrast.
- Align the closing illustrations' monitor stands, laptop keyboards, and
  encrypted-handoff spacing, and use the Reinstate mark consistently.

### Security

- Store the rebuildable local index under
  `$REINSTATE_HOME/cache/session-index-v1.sqlite` with owner-only permissions.
  Index only bounded user-authored prompt text and known metadata/file fields;
  exclude assistant messages/reasoning, tool output, environment dumps, auth
  stores, and credentials. Default command output never offers a full
  transcript dump.
- Build native launch plans as an executable plus argv and a recorded working
  directory, never as a shell command string. Read-only adapters fail closed
  for resume/fork instead of receiving dummy mutation behavior.

## [0.1.0] - 2026-07-30

First stable release.

### Fixed

- Give `--keep-both` and in-use restore forks a real UUID identity instead of a
  decorated `<uuid>-remote-<short>` name. Vendors treat session identifiers as
  UUIDs: Claude Code accepted the decorated form on interactive resume but
  rejected it on `claude --print --resume`, leaving a fork a human could open
  and automation could not. The identity is still derived from the session and
  snapshot, so repeated restores stay idempotent.
- Skip the restore entirely when an in-use session's fork already holds the
  snapshot being pulled. A repeat pull previously rewrote that fork with
  byte-identical content and backed up the previous copy first, growing the
  backup directory by one identical file each time.

### Changed

- Correct the acceptance runbook's ordering. Sections 14d, 15, and 16 assumed a
  session could be resumed and then restored or no-op pushed unchanged, but
  resuming a Claude session appends to it, so those steps could only report
  divergence. The runbook now states the ordering requirement and documents the
  conflict route as the way to reach the same evidence after a resume.
- Promote `v0.1.0-rc.8` to the stable `v0.1.0` release. The product code is
  the candidate's code apart from the two restore fixes above, which were
  reported by that acceptance run. The two-device Phase 1 acceptance evidence
  recorded under `docs/testing/results/` covers the candidate: all 23 mandatory
  gates passed on real macOS and native Windows hardware with no
  release-blocking findings. The restore gates were re-verified on macOS against
  the patched build; the Windows-side backup gate should be re-confirmed on
  Device B.
- Replace release-candidate status language across the README, website, and
  documentation with stable-release wording, and describe behavior in
  version-agnostic terms rather than naming a candidate.

## [0.1.0-rc.8] - 2026-07-29

### Fixed

- Stop treating "no open file handle" as proof that a session is not in use.
  Claude Code appends to its session file and closes it again, so a live Claude
  Code session holds no handle and the handle-only check introduced in
  `v0.1.0-rc.7` reported it as free, letting a restore target a session someone
  was working in. Liveness now also matches an agent that names the exact
  session on its command line, or that is working inside the session's mapped
  project, and biases toward "in use" because the fork policy makes a false
  positive cheap and a false negative expensive. Unrelated agents in other
  projects still never block a restore.

### Changed

- Clarify the landing-page file-sync comparison around machine-specific project
  keys, make the failed-resume mismatch readable at normal scroll speed, and
  replace ambiguous identity ownership language with precise remapping and
  reconstruction behavior.

## [0.1.0-rc.7] - 2026-07-28

### Added

- Add a production SEO, answer-engine optimization, and AI-search foundation
  for the Astro website, including canonical metadata, crawler policy,
  structured data, sitemap, RSS, public product pages, and automated CI checks.
- Add unique 1200×630 social preview images for every indexable route, using
  Reinstate's logo, typography, palette, and axonometric illustration language.
- Add a reproducible 1280×640 GitHub repository preview plus an owner-operated,
  evidence-gated metadata, release-summary, and launch-post runbook.
- Add answer-first integration, compatibility, security, use-case, project,
  open-source, and changelog pages with explicit release-candidate boundaries.
- Add reviewed Claude Code and Codex session-sync guides, an engineering blog,
  a privacy notice, RSS distribution, and a machine-readable security contact.
- Add generated-site SEO, internal-link, anchor, social-image, and static
  performance regression gates to tests, CI, and production deployment checks.
- Add reviewed IndexNow sitemap-diff planning, server-only key proof, bounded
  batching and retries, soft-fail response logging, and non-submitting CI tests.

### Changed

- Expand documentation metadata with search intent, freshness, and topic fields
  while exposing visible review dates and breadcrumbs.
- Point the repository's website reference at the canonical `reinstate.dev`
  domain.
- Separate signed website-only deployment identity
  (`website-vYYYY.MM.DD.N`) from semantic CLI release tags while retaining
  explicit, byte-verified installer parity with the release derived from both
  public bootstraps.

### Fixed

- Scope the restore active-agent check to the exact session file being
  replaced instead of asking whether any Claude Code or Codex process is alive
  on the host. Running unrelated agents in other projects is the normal state
  of a working machine and no longer blocks `pull` or `conflicts resolve`, so
  nobody has to close background agents to restore a session. Detection uses
  open file handles (`lsof` on Unix, Restart Manager on Windows) and falls back
  to the previous host-wide answer only where handles cannot be enumerated,
  reporting that imprecision in the refusal message.
- Restore a session that genuinely is in use alongside the live one as a
  distinct vendor-safe session instead of refusing, so a restore never waits on
  a human closing an agent. The fork identity is derived from the snapshot, so
  repeating the pull is idempotent rather than accumulating copies.
- Detect a concurrent agent write to a restore target and abandon the restore
  instead of discarding those changes at the final rename.
- Allow the guarded immutable Vercel discoverability smoke to record and
  narrowly exempt the provider-injected preview `noindex` header while keeping
  the promoted production-origin check strict.
- Keep local CLI build metadata anchored to `v*` release tags so website-only
  deployment tags cannot become the reported Reinstate version.
- Parse both structured Vercel CLI 57 deployment results and legacy bare-URL
  output before immutable installer verification and production promotion.

## [0.1.0-rc.6] - 2026-07-27

### Changed

- Replace the legacy dark-gradient README banner with the landing page's
  paper-and-ink isometric cross-device session flow.
- Expand the post-Phase-1 roadmap from a generic MCP/skills sync bullet into
  universal agent configuration: one non-secret desired-state profile rendered
  across supported harnesses and encrypted across devices.
- Document planned MCP, skills/instructions, hooks/loops, plugins,
  marketplaces, safe settings, drift reconciliation, supply-chain controls,
  and authentication coordination while keeping credentials excluded.
- Pin the public installers, end-user setup prompts, compatibility evidence,
  and fresh-device acceptance runbook to `v0.1.0-rc.6`.
- Require setup agents to preserve and confirm an existing absolute
  `REINSTATE_HOME` instead of silently falling back to the default home.
- Add committed RC6 Mac Claude Code and native-Windows Codex acceptance prompts
  that keep evidence and report ownership isolated by device.
- Disable automatic Vercel Git deployments and require a signed-tag production
  workflow that verifies both immutable and promoted installer routes byte for
  byte.

### Fixed

- Validate an additional device's encrypted remote manifest with a readable
  object request instead of a metadata-only `HeadObject` probe, avoiding
  Cloudflare R2's generic `400 Bad Request` failure while still leaving no
  local configuration behind when the probe fails.
- Resolve Codex rollout working directories to configured canonical project
  IDs, exclude unmapped projects, normalize resolved roots during export, and
  reject duplicate mappings.
- Report `would pull` during `pull --dry-run` instead of claiming that sessions
  were restored.
- Return the stable redacted `config missing` error without exposing the
  absolute Reinstate home or `config.toml` path.
- Delimit the PowerShell replacement prompt's target-version variable so the
  requested version is visible before confirmation.

## [0.1.0-rc.5] - 2026-07-27

### Changed

- Pin the public installers, end-user setup prompts, compatibility evidence, and
  physical-device acceptance runbook to `v0.1.0-rc.5`.
- Keep local and CI verification release-equivalent while avoiding redundant
  documentation-contract, fixture-scan, and production-KDF work.
  High-level deterministic tests use real age envelopes at a reduced test-only
  scrypt cost; the ordinary full suite still covers the production default.
- Add `make quick` as an explicitly non-release fast development gate.

### Fixed

- Refuse to overwrite an initialized Reinstate home unless `rein init --force`
  is explicitly selected, and back up the previous `config.toml` and
  `state.json` together before replacement.
- Require `rein init --profile-id` to find the existing encrypted remote
  manifest before writing local configuration, catching endpoint, bucket, and
  prefix mistakes during setup.
- Return an error when a joined or established profile's `status`, `diff`,
  `pull`, or `push` cannot find `manifest.age`, instead of reporting a healthy
  empty profile. A new first-device profile may still report an empty remote
  and create its manifest on the first push.
- Bound POSIX installer replacement prompts to 30 seconds by default, reject
  invalid timeout overrides, and fail closed immediately when the active shell
  cannot perform a timed TTY read, preventing unattended `/dev/tty` hangs and
  detecting timed-read support correctly across macOS Bash 3 and Linux Bash 5.

## [0.1.0-rc.4] - 2026-07-26

### Fixed

- Map Claude Code sessions to the configured canonical project ID and derive
  restore destinations from each device's `local_root`, including Claude's
  exact Windows/macOS directory-key rules for spaces, Unicode, and long paths.
- Verify restored sessions at the exact planned vendor path instead of
  accepting a matching session ID elsewhere in the agent tree.
- Fail closed with a repush instruction when a legacy Claude snapshot lacks a
  canonical project mapping, avoiding false-success cross-device restores.
- Exclude unmapped Claude projects when canonical mappings are configured and
  require a destination mapping for canonical snapshots, including empty-map
  configurations.
- Normalize Claude transcript paths through resolved project roots while
  denormalizing them through the destination device's configured root.
- Report `would push` during `push --dry-run` instead of claiming that a
  snapshot was uploaded.

### Changed

- Pin the public installers and end-user setup prompts to `v0.1.0-rc.4`.
- Harden the two-device acceptance runbook with a fresh-profile requirement,
  exact-ID Codex resume, Claude sibling-session disambiguation, hidden-prompt
  passphrase guards, and byte-level ciphertext checks.
- Add coordinated Mac Claude Code and native-Windows Codex verification prompts
  that produce separate sanitized acceptance reports.

## [0.1.0-rc.3] - 2026-07-26

### Fixed

- Accept the tested Claude Code `2.1.219`–`2.1.220` and Codex CLI
  `0.133.0`–`0.145.0` ranges instead of requiring one exact vendor version.
- Make `setup check` fail with compatibility exit code `5` when an installed
  adapter is untested and therefore blocked from push/pull.
- Require a valid Reinstate config before `conflicts list` or `conflicts show`
  can report an empty result.
- Exclude Claude Code `subagents/` artifacts from the top-level resumable
  session list.
- Override the website's transitive `path-to-regexp` dependency to patched
  version `6.3.0`.

## [0.1.0-rc.2] - 2026-07-25

### Fixed

- Release CI restores the remote annotated tag object after checkout before
  verifying its SSH signature.

## [0.1.0-rc.1] - 2026-07-25

### Added

- Phase 0 / Phase 1 authority plan and product contracts
- ADR documenting Phase 0 foundation and Phase 1 Claude/Codex sessions scope
- Compatibility matrix and support states
- Cobra CLI with stable exit codes (`rein` / `reinstate`)
- Versioned config (TOML) and state (JSON) with atomic writes
- Device detection (macOS/Windows/Linux/WSL2; WSL1 refused)
- Redacted `doctor` / `setup check` and synthetic self-test
- Synthetic fixtures + secret scanner
- Hardened CI (fmt/vet/test/race/docs/fixtures/lint/security) and GoReleaser config
- Checksum-verifying installers (`scripts/install.sh`, `scripts/install.ps1`)
- Versioned AI-agent setup prompts under `docs/prompts/`
- S3-compatible backend client + memory test backend
- age scrypt envelopes with tamper/wrong-passphrase tests
- Path mapping, project identity, manifests, push/pull/conflicts
- Claude Code and Codex adapters (fixture-backed)
- Native OS-keyring credential storage and hidden TTY/file-descriptor passphrase input
- Streamed portable artifacts with authenticated metadata/hash validation
- Timestamped restore backups, mutation locks, profile isolation, and executable conflict resolution
- Active Claude Code/Codex process refusal before mutating session restores
- Cross-platform installer contract tests, release SBOMs, and artifact attestations
- Safe installer replacement checks, native Windows `rein.exe`, and release verification scripts
- Deterministic six-environment adapter fixtures and structured contributor/compatibility workflows
- Short CLI alias **`rein`** (same binary as `reinstate`)
- [Product strategy](docs/product-strategy.md) defining the continuity-layer
  positioning, first ICP, product layers, and non-goals

### Changed

- Relicensed core from MIT to **Apache License 2.0**
- README diagrams: ASCII boxes → Mermaid flowcharts
- Roadmap and support docs aligned to Phase 0 foundation + Phase 1 sessions
- Copy-paste setup prompts now continue through init, dry-run, sync, and restore verification
- Codex restores preserve date-partitioned rollout paths; both adapters fail closed on unverified versions
- Star history embed: dark/light `<picture>` + interactive fallback link
- Docs and CLI help prefer short alias **`rein`** (`reinstate` remains full name)
- **Product positioning:** continuity layer for coding-agent work (not multi-device-only);
  multi-device E2EE sync remains the entry wedge
- Roadmap expanded: Phase 2 local session index → verified resume → portable
  handoffs → automatic sync → thin Console/ACP client → team continuity

### Removed

- Invented `v0.0.0` changelog release history (no tag/release existed)
- Secret-bearing init flags, plaintext credential files, and ordinary environment passphrases

### Planned

See [ROADMAP.md](ROADMAP.md) for the authoritative phase list. Highlights:

- **Phase 1 public `v0.1.0`:** Claude + Codex encrypted sync (engine largely in place)
- **Phase 2:** local universal session switcher (`sessions` / `search` / `resume` / `last`)
- **Phase 3:** verified resume (workspace + capability fingerprint)
- **Phase 4:** portable cross-agent handoffs (explicit checkpoints)
- **Phase 5+:** universal cross-harness configuration + auto multi-device habit,
  thin Console/ACP client, team continuity

---

[Unreleased]: https://github.com/HarjjotSinghh/reinstate/compare/v0.5.0-rc.4...HEAD
[0.5.0-rc.4]: https://github.com/HarjjotSinghh/reinstate/compare/v0.5.0-rc.3...v0.5.0-rc.4
[0.5.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.5.0-rc.2...v0.5.0-rc.3
[0.5.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.5.0-rc.1...v0.5.0-rc.2
[0.5.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0...v0.5.0-rc.1
[0.4.0]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.11...v0.4.0
[0.4.0-rc.11]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.10...v0.4.0-rc.11
[0.4.0-rc.10]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.9...v0.4.0-rc.10
[0.4.0-rc.9]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.8...v0.4.0-rc.9
[0.4.0-rc.8]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.7...v0.4.0-rc.8
[0.4.0-rc.7]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.6...v0.4.0-rc.7
[0.4.0-rc.6]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.5...v0.4.0-rc.6
[0.4.0-rc.5]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.4...v0.4.0-rc.5
[0.4.0-rc.4]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.3...v0.4.0-rc.4
[0.4.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.2...v0.4.0-rc.3
[0.4.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.1...v0.4.0-rc.2
[0.4.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0...v0.4.0-rc.1
[0.3.0]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.7...v0.3.0
[0.3.0-rc.7]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.6...v0.3.0-rc.7
[0.3.0-rc.6]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.5...v0.3.0-rc.6
[0.3.0-rc.5]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.4...v0.3.0-rc.5
[0.3.0-rc.4]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.3...v0.3.0-rc.4
[0.3.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.2...v0.3.0-rc.3
[0.3.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.3.0-rc.1...v0.3.0-rc.2
[0.3.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/compare/v0.2.0...v0.3.0-rc.1
[0.2.0]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0...v0.2.0
[0.2.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.2.0-rc.2...v0.2.0-rc.3
[0.2.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.2.0-rc.1...v0.2.0-rc.2
[0.2.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0...v0.2.0-rc.1
[0.1.0]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.8...v0.1.0
[0.1.0-rc.8]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.7...v0.1.0-rc.8
[0.1.0-rc.7]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.6...v0.1.0-rc.7
[0.1.0-rc.6]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.5...v0.1.0-rc.6
[0.1.0-rc.5]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.4...v0.1.0-rc.5
[0.1.0-rc.4]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.3...v0.1.0-rc.4
[0.1.0-rc.3]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.2...v0.1.0-rc.3
[0.1.0-rc.2]: https://github.com/HarjjotSinghh/reinstate/compare/v0.1.0-rc.1...v0.1.0-rc.2
[0.1.0-rc.1]: https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.1.0-rc.1
