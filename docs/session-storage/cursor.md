# Cursor CLI

**Confidence: Documented on macOS, Unverified on native Windows.** One CLI
session created `~/.cursor/chats`. There is still no index source and no
reader.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Catalog key `cursor` is **Cursor CLI**, the terminal agent. Descriptor:
`internal/agents/catalog/cursor.go`. This page is not the in-editor Cursor
agent.

## Identity

The key is decided here, before any probe.

| Aspect | Value |
| ------ | ----- |
| Vendor | Anysphere |
| Product | Cursor CLI (terminal agent) |
| Catalog key | `cursor` |
| Official current binary | `agent` ([overview](https://cursor.com/docs/cli/overview), [installation](https://cursor.com/docs/cli/installation)) |
| Specific / historical binary | `cursor-agent` (forum and older write-ups) |
| Distribution | Official install script (macOS/Linux/WSL and native Windows PowerShell) |
| Storage family | F3 (SQLite `store.db` under `~/.cursor/chats`) |

Cursor is two products:

1. **Cursor CLI** — a terminal binary with documented `ls` / `resume` /
   `--continue`. This is the catalog key.
2. **Cursor editor agent** — the in-app agent. It is not this key. If it
   later needs indexing, it gets a second key, not a blurred `cursor` row.

A forum reply states the two stores are separate (CLI under `~/.cursor/chats`,
editor under `~/.cursor/projects/<project>/agent-transcripts/`). That is not
a Reinstate probe and does not promote a row.

## Device observation (2026-08-16, macOS arm64)

`cursor-agent` version `2026.08.11-e8db854` was installed but **no CLI session
was run**. `~/.cursor/chats` was absent. Editor transcripts were present under
`~/.cursor/projects/<path-slug>/agent-transcripts/`. Roots stayed unset so the
CLI key would not inherit the editor store.

## Device evidence (2026-08-17, macOS arm64)

Artifact:
[`2026-08-17-macos-cursor.json`](../testing/results/agent-probes/2026-08-17-macos-cursor.json)

One real `cursor-agent --print --mode ask` session created the CLI store.
`projects/` stays excluded.

| Check | Result |
| ----- | ------ |
| `cursor-agent` on PATH | yes |
| `cursor-agent --version` | `2026.08.11-e8db854` |
| Resolved root | `~/.cursor` |
| Marker | `chats` present |
| `~/.cursor/chats` | **present** after one CLI session |
| Session bucket | `chats/<32-hex>/` |
| Session dir | `chats/<32-hex>/<uuid-v4>/` |
| Sidecar | `meta.json` keys `createdAtMs`, `cwd`, `hasConversation`, `schemaVersion`, `updatedAtMs`; `schemaVersion` is `1` |
| Conversation body | `store.db` (SQLite; tables `blobs(id, data)`, `meta(key, value)`). Do not open blob payloads |
| Editor store | still under `projects/*/agent-transcripts/`; not this key |

**This is why `Storage.Roots` is now `~/.cursor` with marker `chats`.** The
editor's `projects/` tree is excluded, along with plugins, skills, and other
non-session siblings so a probe is not drowned in `node_modules`.

T1 still needs a native Windows `AGENT-PROBE-V1` that shows the same chats
layout. No index source and no reader until then.

## Why T0 is `layout_unverified`

T1 is forbidden without both a macOS probe and a native Windows probe. The
descriptor therefore stays at T0 with `t0_reason=layout_unverified`. Do not
invent a reader.

`unidentified_product` is the wrong reason: the official CLI is identified.
`desktop_only` is the wrong reason: a terminal CLI exists. `server_backed`
is the wrong reason: Cloud Agent handoff is documented as a push-away path,
not as the only store.

## Family: F3 expected, not F2 yet

Vendor docs document a session-list command. That would make Cursor F2 if
the command is a supported, machine-readable list API.

| Surface | Session list? |
| ------- | ------------- |
| `agent ls` | Documented as "Resume a chat session" / "Open previous chats and resume one". No `--output-format` for `ls`. Reads as an interactive picker. |
| `agent resume` | Resume latest. Not a list. |
| `agent --resume [chatId]` / `--continue` | Resume one id (`--continue` is `--resume=-1`). Not a list. |
| `agent create-chat` | "Create a new empty chat and return its ID". Not a list. |
| `--output-format` | Documented only with `--print` (`text`, `json`, `stream-json`). Not documented on `ls`. |

There is no verified machine-readable session list. Cursor stays F3
(SQLite `store.db` under `chats/`) until a Windows probe confirms the same
layout or `agent ls --output-format json` appears.

## Claimed layout

Every row below is **Unverified**. Paths are quoted from vendor docs or
forum notes, not from a device. Do not treat them as reader input.

| Aspect | Vendor-documented value | Confidence |
| ------ | ---------------------- | ---------- |
| Config override | `$CURSOR_CONFIG_DIR` relocates the CLI config directory | Unverified |
| Config default (Unix) | `~/.cursor/cli-config.json` | Unverified |
| Config default (Windows) | `%USERPROFILE%\.cursor\cli-config.json` | Unverified |
| Linux XDG config | `$XDG_CONFIG_HOME/cursor/cli-config.json` | Unverified |
| Project CLI permissions | `<project>/.cursor/cli.json` | Unverified |
| MCP (user) | `~/.cursor/mcp.json` | Unverified |
| MCP (project) | `<project>/.cursor/mcp.json` | Unverified |
| CLI worktrees | `~/.cursor/worktrees/<reponame>/<name>` | Unverified (edits, not chats) |
| Session files | `~/.cursor/chats/<32-hex>/<uuid-v4>/{meta.json,store.db}` | Documented on macOS 2026-08-17 |
| Schema version marker | `meta.json` `schemaVersion` `1` | Documented on macOS 2026-08-17 |

Unofficial (forum; not a promotion):

| Aspect | Claim | Confidence |
| ------ | ----- | ---------- |
| CLI chats | `~/.cursor/chats` (SQLite `store.db` + `meta.json`) | Documented on macOS; Windows unverified |
| Editor chats | `~/.cursor/projects/<project>/agent-transcripts/` and/or workspace `state.vscdb` | Unverified |
| Shared store? | Staff reply: CLI and IDE do **not** share a session store | Unverified |

Native control surface (vendor-documented; still Unverified on a device):

| Aspect | Value |
| ------ | ----- |
| Version | `agent --version` / `-v` |
| List / picker | `agent ls` |
| Resume latest | `agent resume` or `agent --continue` |
| Resume specific | `agent --resume="chat-id"` |
| New chat id | `agent create-chat` |
| Cloud handoff | prepend `&` to a message; pick up at cursor.com/agents |
| Auth | `agent login` / `logout` / `status`; or `CURSOR_API_KEY` / `--api-key` |

Documented resume argv is **not** a T3 claim. T3 needs a physical dual-platform
journey. The descriptor has no `NativeSpec`.

## Concurrent writer

Unknown. Official docs do not say whether the CLI or the editor holds an
exclusive lock on any session file. If a later probe finds an exclusive
lock, discovery must degrade to a clear "close Cursor" message, not an
unexplained error. Do not open a database writable. An unrecognized schema
version fails closed (exit 5).

## Authentication material (exclude before any later read)

Do not open these if a reader is ever written.

| Location | Why it is excluded |
| -------- | ------------------ |
| `<config-dir>/cli-config.json` | CLI settings; not a session |
| `~/.cursor/mcp.json`, `<project>/.cursor/mcp.json` | MCP server config; may hold tokens |
| `CURSOR_API_KEY`, `--api-key` | Env / argv secret. Not a file. |
| Browser-login store | Vendor: "credentials are securely stored locally". Path not documented. Treat any auth file under the config dir as excluded once named. |
| `~/.cursor/worktrees/` | Working copies, not chats |

## What a later probe must settle

1. Native Windows: same `chats/<32-hex>/<uuid-v4>/{meta.json,store.db}` layout
   after one CLI session. That is the T1 gate.
2. Whether `agent ls` can emit a machine-readable list (then F2).
3. Fail-closed behavior on `schemaVersion` other than `1`.
4. Whether reading `store.db` while the CLI is running is safe (WAL present).
5. Keep `projects/`, `plugins/`, `skills/`, and `mcp.json` excluded.

## Sources

- [Cursor CLI overview](https://cursor.com/docs/cli/overview)
- [Cursor CLI installation](https://cursor.com/docs/cli/installation)
- [Using Agent in CLI](https://cursor.com/docs/cli/using)
- [Cursor CLI parameters](https://cursor.com/docs/cli/reference/parameters)
- [Cursor CLI configuration](https://cursor.com/docs/cli/reference/configuration)
- [Cursor CLI authentication](https://cursor.com/docs/cli/reference/authentication)
- [Cursor CLI past chats not showing up](https://forum.cursor.com/t/cursor-cli-past-chats-not-showing-up/152450) (forum; CLI vs IDE stores)
