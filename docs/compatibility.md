# Compatibility

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

| Agent | Local discovery/search/inspect | Native resume/fork | Physical Phase 2 evidence |
| ----- | ------------------------------ | ------------------ | ------------------------- |
| Claude Code | Included | Same-vendor included | Tagged-artifact acceptance passed on Apple Silicon macOS and native Windows x64 |
| OpenAI Codex CLI | Included | Same-vendor included | Tagged-artifact acceptance passed on Apple Silicon macOS and native Windows x64 |
| Gemini CLI | Read-only included | Not supported in Phase 2 | Physical path passed on Windows; unavailable on test Mac |
| OpenCode | Read-only included | Not supported in Phase 2 | Physical path passed on Windows; unavailable on test Mac |
| Cursor | Not implemented | Not implemented | Not applicable |
| Grok Build | Not implemented | Not implemented | Not applicable |

Automated fixture/fake-runner evidence and physical evidence are reported
separately. Stable support is limited to the two physically verified primary
platforms; preview artifacts do not inherit their evidence.

This table covers session compatibility only. Planned universal configuration
support will be reported separately per harness and per capability (MCP,
skills/instructions, hooks/loops, plugins, marketplaces, safe settings). A
supported session adapter will not imply configuration support. See
[universal-configuration.md](universal-configuration.md).

It also does **not** imply cross-agent handoff support. Phase 4 will publish a
directional matrix with separate source-parser, structured-handoff,
destination-launch, reconstructed-history, workspace-verification, and fidelity
states.

## Cross-agent continuation (planned)

| Direction | Structured handoff | Reconstructed native history |
| --------- | ------------------ | ---------------------------- |
| Claude Code → Codex | Phase 4 first GA pair | Experimental, exact versions only |
| Codex → Claude Code | Phase 4 first GA pair | Experimental, exact versions only |
| Claude/Codex ↔ Gemini CLI | Planned after first pair | Exploring |
| Claude/Codex ↔ OpenCode | Planned after first pair | Exploring |
| Grok Build, Copilot CLI, Cursor, Orca, others | Requires adapter evidence | Uncommitted |

The GA path must work with the source CLI closed/rate-limited and no source
model call. It creates a new destination-native session with explicit lineage
and reports normalized, summarized, referenced, redacted, omitted, and
unsupported material. See
[cross-agent-continuation.md](cross-agent-continuation.md).

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

Current source compatibility evidence covers these inclusive stable-version
ranges on macOS arm64 plus deterministic synthetic fixtures:

| Agent | Tested stable range |
| ----- | ------------------- |
| Claude Code | `2.1.219`–`2.1.220` |
| OpenAI Codex CLI | `0.133.0`–`0.146.0` |

Stable `v0.2.0` expands Codex support through `0.146.0`, which passed the full
Phase 2 physical matrix on both verified platforms. It retains the
destination-device Claude project-directory remapping and exact restore-path
verification introduced in `v0.1.0`.
Versions outside them, including prereleases, are `UNTESTED` and must not be
called stable until their release matrix rows pass. The repository does not
fabricate native or physical results for a platform absent from the recorded
reports. Phase 1 has no unsafe compatibility override.

`rein setup check` exits with compatibility code `5` when an installed adapter
is `UNTESTED` or `UNSUPPORTED`, because writes are blocked. An agent that is not
installed remains an informational `NOT_INSTALLED` result.

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

## Security exclusions

Never synced: auth files, API keys, OAuth tokens, `.env`, OS keyring contents,
credential stores, caches, and regenerable dependencies.
