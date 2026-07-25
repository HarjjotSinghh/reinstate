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

Pre-release compatibility evidence currently covers Claude Code `2.1.219` and
Codex CLI `0.133.0` on macOS arm64 plus deterministic synthetic fixtures.
Other versions are `UNTESTED` and must not be called stable until their release
matrix rows pass. Native Windows, macOS amd64, and WSL2 remain release gates;
this repository does not fabricate those results. Phase 1 has no unsafe
compatibility override.

## Path mapping

Sessions may contain absolute paths. Reinstate rewrites known structural path
fields to portable tokens (`${HOME}`, `${REPO:<id>}`, `${WORK:<alias>}`) so
Windows ↔ macOS resume works. Prose and unknown fields are left unchanged.

## Security exclusions

Never synced: auth files, API keys, OAuth tokens, `.env`, OS keyring contents,
credential stores, caches, and regenerable dependencies.
