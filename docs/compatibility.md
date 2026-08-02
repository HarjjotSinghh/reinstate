# Compatibility

Stable Reinstate Phase 1 (`v0.1.0`) implements **same-vendor encrypted session
sync** for:

| Agent | Status in v0.1.0 |
| ----- | ---------------- |
| Claude Code | Primary — required on release matrix |
| OpenAI Codex CLI | Primary — required on release matrix |
| Gemini CLI | Not in Phase 1 |
| OpenCode | Not in Phase 1 |
| Cursor | Not in Phase 1 |
| Grok Build | Not in Phase 1 |

The `v0.2.0-rc.3` candidate adds a separate Phase 2 local capability matrix:

| Agent | Local discovery/search/inspect | Native resume/fork | Physical Phase 2 evidence |
| ----- | ------------------------------ | ------------------ | ------------------------- |
| Claude Code | Included in candidate | Same-vendor included in candidate | Development acceptance passed on macOS and Windows |
| OpenAI Codex CLI | Included in candidate | Same-vendor included in candidate | Development acceptance passed on macOS and Windows |
| Gemini CLI | Read-only included in candidate | Not supported in Phase 2 | Physical path passed on Windows; unavailable on test Mac |
| OpenCode | Read-only included in candidate | Not supported in Phase 2 | Physical path passed on Windows; unavailable on test Mac |
| Cursor | Not implemented | Not implemented | Not applicable |
| Grok Build | Not implemented | Not implemented | Not applicable |

Automated fixture/fake-runner evidence and physical evidence are reported
separately. This table does not promote candidate evidence to stable evidence;
stable `v0.1.0` remains available while `v0.2.0-rc.3` is tested.

This table covers session compatibility only. Planned universal configuration
support will be reported separately per harness and per capability (MCP,
skills/instructions, hooks/loops, plugins, marketplaces, safe settings). A
supported session adapter will not imply configuration support. See
[universal-configuration.md](universal-configuration.md).

## Environments

| Environment | Stable Phase 1 evidence | Phase 2 physical evidence |
| ----------- | ----------------------- | ------------------------- |
| macOS native (arm64) | RC8 23-row run passed for Claude/Codex | 30 required development rows passed at `b952d38` |
| macOS native (amd64) | Release cross-build/CI; no physical report claimed here | Cross-build only; no physical report claimed |
| Windows 11 native (amd64) | RC8 23-row run passed for Claude/Codex | 30 required rows plus Gemini/OpenCode physical paths passed at `b952d38` |
| Windows 11 WSL2 (amd64) | Documented fixture/smoke path; distinct from native Windows | Separate smoke pending |

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

The candidate expands Codex support through `0.146.0`, which passed the full
Phase 2 physical matrix on both tested devices. Stable `v0.1.0` contains the
earlier `0.133.0`–`0.145.0` Codex range plus
destination-device Claude project-directory remapping and exact restore-path
verification.
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
