# Compatibility

Release candidate `v0.3.0-rc.1` added Phase 3 verified resume to the stable
`v0.2.0` surface. Apple Silicon macOS tagged-artifact acceptance passed;
native Windows x64 failed (including extensionless Codex executable trust).
RC2 still failed Windows acceptance (Codex trust and snapshot/PowerShell
gates). RC3 still failed Windows acceptance on the PowerShell 5.1 staging parser
and absolute-path privacy gates. RC4 fixed those findings after Windows-first
product smoke, but its release workflow failed before publication while the
PowerShell artifact verifier ran on Ubuntu. RC5 published that portable
verifier but failed dual-platform acceptance on out-of-range agent versions.
Current candidate `v0.3.0-rc.6` widens the fail-closed Claude/Codex ranges for
retest; its tagged-artifact acceptance is pending.
Intel macOS and Linux/WSL2 remain optional, unsupported/unverified evidence and
do not block RC6. Passing RC6
will not authorize stable `v0.3.0`; that requires a separate reviewed promotion
and fresh tagged-artifact validation.

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
ranges until a later candidate widens them with fresh device evidence.
Candidate `v0.3.0-rc.6` widens the fail-closed ranges so current primary-host
installs (Claude Code through `2.1.227`, Codex CLI through `0.147.0`) are
`SUPPORTED` for dual-platform retest; versions above the maxima remain
`UNTESTED` until a later matrix expands them again:

| Agent | Inclusive source-tested range (RC6) |
| ----- | ------------------- |
| Claude Code | `2.1.219`–`2.1.227` |
| OpenAI Codex CLI | `0.133.0`–`0.147.0` |

Stable `v0.2.0` still documents the older Phase 2 physical ceiling (Claude
`2.1.219`–`2.1.220`, Codex through `0.146.0`). RC6 does not rewrite that stable
claim; it only expands the RC product gates pending fresh tagged-artifact
acceptance. Destination-device Claude project-directory remapping and exact
restore-path verification from `v0.1.0` remain.
Versions outside the current product maxima, including prereleases, are
`UNTESTED` and must not be called stable until their release matrix rows pass. The repository does not
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
