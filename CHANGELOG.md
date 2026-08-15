# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/HarjjotSinghh/reinstate/compare/v0.4.0-rc.8...HEAD
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
