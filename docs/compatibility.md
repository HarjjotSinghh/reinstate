# Compatibility

Reinstate Phase 1 (`v0.1.0`) implements **same-vendor session sync** for:

| Agent | Status in v0.1.0 |
| ----- | ---------------- |
| Claude Code | Primary — required on release matrix |
| OpenAI Codex CLI | Primary — required on release matrix |
| Gemini CLI | Not in Phase 1 |
| OpenCode | Not in Phase 1 |
| Cursor | Not in Phase 1 |
| Grok Build | Not in Phase 1 |

This table covers session compatibility only. Planned universal configuration
support will be reported separately per harness and per capability (MCP,
skills/instructions, hooks/loops, plugins, marketplaces, safe settings). A
supported session adapter will not imply configuration support. See
[universal-configuration.md](universal-configuration.md).

## Environments

| Environment | Claude Code | Codex CLI |
| ----------- | ----------- | --------- |
| macOS native (arm64) | Required | Required |
| macOS native (amd64) | Required before stable | Required before stable |
| Windows 11 native (amd64) | Required | Required |
| Windows 11 WSL2 (amd64) | Documented + smoke | Documented + smoke |

### Explicitly unsupported

- WSL1
- Treating native Windows and WSL as the same Reinstate device
- Automatic sharing of one agent-state directory between Windows and WSL
- Untested agent layouts during restore without an explicit override
- Other coding agents

## Compatibility states

Every adapter discovery result reports one of:

| State | Meaning | Behavior |
| ----- | ------- | -------- |
| `SUPPORTED` | Exact layout/version range has release evidence | Discovery, push, and pull allowed |
| `UNTESTED` | Recognizable layout, newer or unverified version | Read-only discovery; export and restore refuse |
| `UNSUPPORTED` | Known-incompatible layout/version | Fail closed; link to this page |
| `NOT_INSTALLED` | No local installation/root found | Informational |

Current source compatibility evidence covers these inclusive stable-version
ranges on macOS arm64 plus deterministic synthetic fixtures:

| Agent | Tested stable range |
| ----- | ------------------- |
| Claude Code | `2.1.219`–`2.1.220` |
| OpenAI Codex CLI | `0.133.0`–`0.145.0` |

Release candidate `v0.1.0-rc.6` contains these compatibility ranges plus
destination-device Claude project-directory remapping and exact restore-path
verification.
Versions outside them, including prereleases, are `UNTESTED` and must not be
called stable until their release matrix rows pass. Native Windows, macOS
amd64, and WSL2 remain release gates; this repository does not fabricate those
results. Phase 1 has no unsafe compatibility override.

`rein setup check` exits with compatibility code `5` when an installed adapter
is `UNTESTED` or `UNSUPPORTED`, because writes are blocked. An agent that is not
installed remains an informational `NOT_INSTALLED` result.

## Path mapping

Sessions may contain absolute paths. Reinstate rewrites known structural path
fields to the currently configured portable tokens (`${HOME}` and
`${REPO:<id>}`) so Windows ↔ macOS resume works. The lower-level mapper's
`${WORK:<alias>}` primitive is not populated by RC6 configuration or adapters.
Prose and unknown fields are left unchanged.

## Security exclusions

Never synced: auth files, API keys, OAuth tokens, `.env`, OS keyring contents,
credential stores, caches, and regenerable dependencies.
