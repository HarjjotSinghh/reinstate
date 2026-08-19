# Cline

**Confidence: Documented on macOS; Unverified on native Windows.**
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Catalog key is `cline`. Descriptor: `internal/agents/catalog/cline.go`.

A 2026-08-19 macOS probe after `cline` 3.0.55 wrote a real session under
`~/.cline/data/sessions`. Native Windows is still missing, so T1 stays
closed. There is no index source and no reader.

## Identity

| Aspect | Value | Source |
| ------ | ----- | ------ |
| Catalog key | `cline` | this page |
| Vendor | Cline (Cline Bot Inc.) | [Marketplace](https://marketplace.visualstudio.com/items?itemName=saoudrizwan.claude-dev), [License](https://github.com/cline/cline/blob/HEAD/LICENSE) |
| Product | Cline | [cline.bot](https://cline.bot), [docs.cline.bot](https://docs.cline.bot/) |
| Official extension | `saoudrizwan.claude-dev` | [VS Marketplace](https://marketplace.visualstudio.com/items?itemName=saoudrizwan.claude-dev) |
| Official CLI | `cline` | [CLI reference](https://docs.cline.bot/cli/cli-reference) |
| Official repository | [cline/cline](https://github.com/cline/cline) | Marketplace and docs |
| Distribution | Official marketplace extension and official CLI | same |
| Storage family | F3 expected (`FamilyEmbeddedDB`) | Hub docs name SQLite + JSON; editor globalStorage is the historical F3 candidate. Neither is probed. |

One catalog entry. Official config docs describe a shared `~/.cline/`
root across IDE, CLI, and SDK. That is not two products. A later probe
that finds a *separate* authoritative tree under an editor host must
attribute the host, not invent a second key.

## Device evidence (2026-08-19, macOS arm64)

Artifact:
[`2026-08-19-macos-cline.json`](../testing/results/agent-probes/2026-08-19-macos-cline.json)

`cline` 3.0.55 on PATH. A one-shot prompt (unauthorized; no account)
still created a session directory. Resolved root `~/.cline/data`.

```
~/.cline/data/
  sessions/<slug>/<slug>.json
  sessions/<slug>/<slug>.messages.json
  db/sessions.db   (+ shm/wal)
  settings/providers.json   excluded
```

Pretty-printed JSON: the probe's first-line sampler recorded no keys
(`{` is not a JSON object). A key-only inspection of the session file
saw `cwd`, `session_id`, `workspace_root`, `prompt`, `started_at`,
`status`, `messages_path`. `cline history --json` listed that session.
That is an F2 candidate, not a shipped read API. Native Windows is
absent, so the row stays T0.

## Why T0 is `layout_unverified`

macOS now has a live tree. Native Windows does not. T1 requires both.

## Claimed layout (all Unverified)

Official docs currently describe two surfaces. Neither is a support
claim.

### Shared home tree / hub (vendor-documented)

| Aspect | Official claim | Notes |
| ------ | -------------- | ----- |
| Config / data root | `~/.cline/` (Windows: `%USERPROFILE%\.cline`) | Shared across IDE, CLI, SDK |
| Data override | `$CLINE_DATA_DIR` replaces `~/.cline/data/` | Also `--data-dir` (default `~/.cline`) |
| Session files | `~/.cline/data/sessions/` | "Session data" |
| Hub index | `sessions.db` (SQLite) under that directory | "SQLite index for efficient listing" |
| Authoritative record | `[session-id].json` beside the db | "source of truth for each session’s state" |
| Other SQLite | `~/.cline/data/db/` (example: `cron.db`) | Not a transcript |
| Secrets | `~/.cline/data/settings/providers.json` | Documented as API keys and provider config |
| Settings | `global-settings.json`, `cline_mcp_settings.json` | Same settings directory |
| CLI history surface | `cline history` / `cline h` | "List session history or manage saved sessions". Output shape unpublished |
| CLI resume | `cline --id <session-id>` | Not a T3 `NativeSpec` |

### Editor-host globalStorage (community-reported, not official docs)

VS Code-compatible hosts keep per-extension `User/globalStorage`.
Community reports name:

- macOS: `~/Library/Application Support/<Host>/User/globalStorage/saoudrizwan.claude-dev/tasks`
- Windows: `%APPDATA%\<Host>\User\globalStorage\saoudrizwan.claude-dev\tasks`

`<Host>` is Code, `Code - Insiders`, VSCodium, Cursor, or Windsurf —
each a separate tree. Official docs do **not** name this path. A later
probe must confirm it independently of `~/.cline/`.

## Family stays F3, not F2

`cline history` is the F2 question. Official docs do not say it prints
JSON, and this host has no `cline` binary to ask. A TUI or human-only
list is not a supported machine API.

Until a probe captures a machine-readable session list, Cline stays F3
expected (SQLite index plus per-host extension storage). Do not write
an F2 `cliquery` wrapper from the command name.

## F3 properties (unchanged by the CLI docs)

1. **Multiple hosts, multiple roots.** A record must name the host.
   Two hosts' tasks are otherwise indistinguishable.
2. **No version probe from PATH for the extension.** The CLI has
   `cline -V` / `cline version`; output shape is uncaptured. T3 still
   needs a fail-closed range and dual-platform resume journeys.
3. **No resume argv for the extension.** Opening the editor is not the
   current launch mechanism. T3 is not reachable for the extension
   through that mechanism. That is a property, not a later task.
4. **Concurrent writer.** The editor or hub may be running. A future
   F3 scanner must open read-only, take no lock, and fail closed on an
   unknown schema version.

## Secrets in the same tree

If a later task ever reads `~/.cline` or a host `globalStorage` root,
put these in `Excluded` **before** any read:

- `data/settings/providers.json` — documented API keys
- any OAuth / secret cache found under `data/settings/`
- editor `SecretStorage` / keychain material (not files)

## What this task settled

1. Official product identified. Catalog key `cline` is that product.
2. Dual-platform probes unavailable. T1 is closed.
3. No F3 scanner. T-032 (Roo Code) cannot reuse `scan/embeddeddb`
   until a later T-031 follow-up builds it.
4. `cline history` is a documented list surface and a future F2
   candidate. It is not a shipped read API.
5. Workspace / project attribution is unknown. `ProjectKey` is `none`
   until a probe shows a recorded path.

## What a later probe must settle

1. The storage root for each installed host, on macOS **and** native
   Windows, plus whether `~/.cline/data/sessions/` is the same store
   or a second tree.
2. Per-task / per-session directory shape, and whether the id is
   stable across restarts.
3. Which file carries user-visible turns, and which is UI-render
   state. Do not parse both.
4. Whether the workspace or repository path is recorded.
5. Whether an on-disk schema / extension version marker exists.
6. Whether `cline history` (or another vendor export) is a
   machine-readable list. If it is, prefer it over private files.

A probe is required to leave T0. Empty-install inventory is not a
probe.

## Sources

- [Cline product](https://cline.bot)
- [Cline docs](https://docs.cline.bot/)
- [Config (`~/.cline/`, `CLINE_DATA_DIR`)](https://docs.cline.bot/getting-started/config)
- [CLI reference (`cline history`, `--id`, `--data-dir`)](https://docs.cline.bot/cli/cli-reference)
- [Hub-spoke session persistence (`sessions.db`, `[session-id].json`)](https://docs.cline.bot/sdk/architecture/hub-spoke)
- [Tasks / History UI](https://docs.cline.bot/core-workflows/task-management)
- [VS Marketplace: `saoudrizwan.claude-dev`](https://marketplace.visualstudio.com/items?itemName=saoudrizwan.claude-dev)
- [Official repository](https://github.com/cline/cline)
