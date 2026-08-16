# Compatibility

Stable `v0.4.0` is the current Phase 4 release: structured handoff plus
verified resume for Claude Code and Codex on Apple Silicon macOS and native
Windows x64, after dual-platform tagged-artifact acceptance PASS on candidate
`v0.4.0-rc.11`. Intel macOS and Linux/WSL2 remain optional,
unsupported/unverified evidence and do not block stable.

Candidate history (RC1–RC11) lived under `docs/testing/results/` and earlier
CHANGELOG sections; those records do not rewrite this stable claim.

Stable Reinstate `v0.2.0` implements **same-vendor encrypted session sync** and
the Phase 2 local continuity surface for:

| Agent | Stable v0.2.0 capability |
| ----- | ---------------- |
| Claude Code | Primary — required on release matrix |
| OpenAI Codex CLI | Primary — required on release matrix |
| Gemini CLI | Not in Phase 1 |
| OpenCode | Not in Phase 1 |
| Cursor | Not in Phase 1 |
| Grok Build | Not in Phase 1 |

The Phase 2 local capability matrix is:

| Agent | Local discovery/search/inspect | Native resume/fork | Physical Phase 2 evidence | Tier |
| ----- | ------------------------------ | ------------------ | ------------------------- | ---- |
| Claude Code | Included | Same-vendor included | Tagged-artifact acceptance passed on Apple Silicon macOS and native Windows x64 | T5 |
| OpenAI Codex CLI | Included | Same-vendor included | Tagged-artifact acceptance passed on Apple Silicon macOS and native Windows x64 | T5 |
| Gemini CLI | Read-only included | Not supported in Phase 2 | Physical path passed on Windows; unavailable on test Mac | T2 |
| OpenCode | Read-only included | Not supported in Phase 2 | Physical path passed on Windows; unavailable on test Mac | T2 |
| Cursor | Not implemented | Not implemented | Not applicable | — |
| Grok Build | Not implemented | Not implemented | Not applicable | T2 |
<!-- T-010 insertion point: append one new agent row below this line. Do not add columns, re-sort, re-wrap, or edit other rows. -->
| Amp | Not readable locally (`server_backed`) | Not implemented | Not applicable | T0 |
| ZCode | Not implemented (`desktop_only`) | Not implemented | Not applicable | T0 |

Automated fixture/fake-runner evidence and physical evidence are reported
separately. Stable support is limited to the two physically verified primary
platforms; preview artifacts do not inherit their evidence.

This table covers session compatibility only. Planned universal configuration
support will be reported separately per harness and per capability (MCP,
skills/instructions, hooks/loops, plugins, marketplaces, safe settings). A
supported session adapter will not imply configuration support. See
[universal-configuration.md](universal-configuration.md).

## Support tiers

"Supported" now means a published [support tier](agent-support-tiers.md),
not a yes-or-no flag. The matrix above records the current tier for each
shipped agent. T5 is encrypted same-vendor sync. T2 is a handoff source.
Native resume stays same-vendor.

## Phase 4 structured-handoff candidate

Handoff support is directional. A supported source reader does not imply a
supported destination target, encrypted sync, or same-vendor native execution.

| Source → destination | Claude Code | Codex CLI | Gemini CLI | OpenCode | Grok Build |
| -------------------- | :---------: | :-------: | :--------: | :------: | :--------: |
| **Claude Code** | same-vendor native resume | structured handoff | not in v0.4.0 | not in v0.4.0 | not planned |
| **Codex CLI** | structured handoff | same-vendor native resume | not in v0.4.0 | not in v0.4.0 | not planned |
| **Gemini CLI** | structured handoff | structured handoff | not a target (source-only) | not in v0.4.0 | not planned |
| **OpenCode** | structured handoff | structured handoff | not in v0.4.0 | not a target (source-only) | not planned |
| **Grok Build** | structured handoff | structured handoff | not in v0.4.0 | not in v0.4.0 | not a target (source-only) |

Every cross-agent entry above creates a new destination session for the same
task through Claude Code's or Codex's documented CLI. It does not reconstruct
vendor history or write a vendor-internal session file. Gemini, OpenCode, and
Grok are source-only in v0.4.0; attempts to use them as destinations fail closed.

The handoff path parses the source locally without a source model call, records
component-level fidelity, and stores its capsule and lineage only under the
private `$REINSTATE_HOME/handoffs/` store. The store is hard-excluded from
encrypted push/pull. Destination acknowledgement is a prompt-level contract;
it is not an enforced protocol.

## Environments

| Environment | Stable Phase 1 evidence | Phase 2 physical evidence |
| ----------- | ----------------------- | ------------------------- |
| macOS native (arm64) | RC8 23-row run passed for Claude/Codex | Stable verified: all 30 required tagged-artifact rows passed |
| macOS native (amd64) | Release cross-build/CI | Preview/unverified; no physical report claimed ([#97](https://github.com/HarjjotSinghh/reinstate/issues/97)) |
| Windows 11 native (amd64) | RC8 23-row run passed for Claude/Codex | Stable verified: all required rows plus Gemini/OpenCode physical paths passed |
| Windows 11 WSL2 (amd64) | Documented fixture path; distinct from native Windows | Preview/unverified; no physical report claimed ([#98](https://github.com/HarjjotSinghh/reinstate/issues/98)) |

### Explicitly unsupported

- WSL1
- Treating native Windows and WSL as the same Reinstate device
- Automatic sharing of one agent-state directory between Windows and WSL
- Untested agent layouts during restore without an explicit override
- Mutation/sync for Gemini CLI and OpenCode in Phase 2
- Other coding agents without an explicit local source contract

## Phase 1 sync compatibility states

Every Phase 1 sync-adapter discovery result reports one of:

| State | Meaning | Behavior |
| ----- | ------- | -------- |
| `SUPPORTED` | Exact layout/version range has required evidence | Sync export/restore capabilities allowed |
| `UNTESTED` | Recognizable layout, newer or unverified version | Sync discovery may proceed; export/restore refuse |
| `UNSUPPORTED` | Known-incompatible layout/version | Fail closed; link to this page |
| `NOT_INSTALLED` | No local installation/root found | Informational |

The Phase 3 source compatibility policy keeps these inclusive agent version
ranges until a later release widens them with fresh device evidence.
Stable `v0.3.0` widened the fail-closed ranges so its primary-host installs
(Claude Code through `2.1.227`, Codex CLI through `0.147.0`) were `SUPPORTED`
after dual-platform tagged-artifact acceptance. The Phase 4 candidates widen the Claude Code
ceiling further, through `2.1.229`, so the acceptance hosts are not blocked by
their installed agent. Claude Code auto-updates faster than a fail-closed
ceiling can be revised: `v0.4.0-rc.1` widened to `2.1.228` and both hosts had
already moved past it within a day, so `v0.4.0` keeps `2.1.229`.
Neither `2.1.228` nor `2.1.229` has completed dual-platform tagged-artifact
acceptance. Versions above the maxima remain `UNTESTED` until a
later matrix expands them again:

| Agent | Inclusive source-tested range (v0.4.0) |
| ----- | ------------------- |
| Claude Code | `2.1.219`–`2.1.229` |
| OpenAI Codex CLI | `0.133.0`–`0.147.0` |

Stable `v0.2.0` still documents the older Phase 2 physical ceiling (Claude
`2.1.219`–`2.1.220`, Codex through `0.146.0`). Destination-device Claude
project-directory remapping and exact restore-path verification from `v0.1.0`
remain.
Versions outside the current product maxima, including prereleases, are
`UNTESTED` and must not be called stable until their release matrix rows pass. The repository does not
fabricate native or physical results for a platform absent from the recorded
reports. Phase 1 has no unsafe compatibility override.

`rein setup check` exits with compatibility code `5` when an installed adapter
is `UNTESTED` or `UNSUPPORTED`, because writes are blocked. An agent that is not
installed remains an informational `NOT_INSTALLED` result.

### Reading a source session vs. writing a vendor tree

Handoff source probing and sync adapters answer the same question differently,
on purpose:

| | Sync adapters (`internal/adapter`) | Transcript readers (`internal/transcript`) |
| --- | --- | --- |
| Unrecognized layout | `UNSUPPORTED` | `UNSUPPORTED` |
| Version outside the verified range | `UNTESTED` | `UNTESTED` |
| Version cannot be determined | `UNTESTED` (fail closed) | `SUPPORTED`, layout only (fail open) |

A sync restore writes into a vendor tree, so acting on an unknown layout can
destroy session state and must fail closed. A structured handoff only reads a
file that already exists, and it exists precisely when the agent is closed,
logged out, rate limited, or uninstalled — the situations users reach for a
handoff in. Absence of version information is not evidence of incompatibility
there, so it does not block.

Every transcript reader resolves the version from the installed executable
through the same probe `rein inspect` reports; no reader reads a vendor version
file. Gemini CLI, OpenCode, and Grok Build have no version probe and are always
judged on layout alone, which is the same rule rather than an exception.

Phase 2 local records carry per-session `can_resume` and `can_fork`
capabilities. Claude/Codex native actions use the exact documented argv in the
Phase 2 acceptance matrix and preflight the executable and recorded workspace;
physical version evidence remains a separate release gate. Gemini/OpenCode are
read-only by phase contract.

## Path mapping

Sessions may contain absolute paths. Reinstate rewrites known structural path
fields to the currently configured portable tokens (`${HOME}` and
`${REPO:<id>}`) so Windows ↔ macOS resume works. The lower-level mapper's
`${WORK:<alias>}` primitive is not populated by Reinstate configuration or adapters.
Prose and unknown fields are left unchanged.

Continuity capsules hold no absolute path in any field that carries a path.
Transcript readers tokenize the structural paths they lift out of a transcript —
tool-call inputs such as `file_path`, `workdir`, and argv entries, tool-result
output, and attachment references — before any capsule is built. A path that
belongs to no configured root cannot be rewritten for the destination device and
usually embeds the operator's account name, so it becomes
`${EXTERNAL:<digest>}/<name>`: a stable, non-reversible identity plus the file's
base name. It is deliberately not resolvable on the destination.

Capsule canonicalization enforces that rule on the path-typed fields —
`workspace.root`, `workspace.changed_files`, `task.changed_files`,
`task.files_touched_per_transcript`, block `path`, `ref` and `meta` values, the
sidecar references, and the path-typed keys inside tool arguments — and names
the offending field when it refuses one. A reader that forgets to tokenize
therefore fails loudly instead of leaking the operator's home directory to the
destination.

Message bodies stay untouched as prose, and prose is never judged as a path. A
message that opens with a slash command (`/init`, `/compact`) or names
`/etc/hosts` mid-sentence is carried exactly as the user wrote it: rewriting or
refusing it would corrupt the record of what was actually said. Capsules are
local-only in v0.4.0 and the destination is a local process, so a path a user
typed into a sentence stays where the user put it.

## Security exclusions

Never synced: auth files, API keys, OAuth tokens, `.env`, OS keyring contents,
credential stores, caches, and regenerable dependencies.
