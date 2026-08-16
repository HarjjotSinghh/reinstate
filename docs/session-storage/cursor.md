# Cursor CLI

**Confidence: Unverified** — catalog descriptor exists; no index source, no
reader, no committed probe. Vendor documentation is recorded below; it is
not a T1 gate.
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
| Storage family | F3 expected; F2 blocked until `agent ls` is shown to be machine-readable |

Cursor is two products:

1. **Cursor CLI** — a terminal binary with documented `ls` / `resume` /
   `--continue`. This is the catalog key.
2. **Cursor editor agent** — the in-app agent. It is not this key. If it
   later needs indexing, it gets a second key, not a blurred `cursor` row.

A forum reply states the two stores are separate (CLI under `~/.cursor/chats`,
editor under `~/.cursor/projects/<project>/agent-transcripts/`). That is not
a Reinstate probe and does not promote a row.

## Why T0 is `layout_unverified`

T-030 cannot produce the evidence T1 requires.

| Check | Result |
| ------ | ------ |
| Official product | identified: Cursor CLI, vendor Anysphere |
| `cursor-agent` on PATH | not installed |
| Official `agent` on PATH | this host's `agent` is Grok's binary (`~/.grok/bin/agent`), not Cursor CLI |
| `rein doctor --agents --json` | not captured: the vendor CLI is absent and has not been used |
| macOS AGENT-PROBE-V1 | **absent** |
| native Windows AGENT-PROBE-V1 | **absent** (no native Windows host) |
| Real `~/.cursor` tree | **not listed** (no real transcripts) |

T1 is forbidden without both a macOS probe and a native Windows probe. The
descriptor therefore stays at T0 with `t0_reason=layout_unverified`. That is
the complete T-030 result. Do not invent a reader.

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
expected (local SQLite / editor-adjacent storage is the working hypothesis)
until a probe shows either JSON `ls` output (then F2, prefer that over
reading private files) or a confirmed on-disk layout.

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
| Session files | **not documented** | Unverified |
| Schema version marker | **not documented** | Unverified |

Unofficial (forum; not a promotion):

| Aspect | Claim | Confidence |
| ------ | ----- | ---------- |
| CLI chats | `~/.cursor/chats` (reported SQLite) | Unverified |
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

1. Confirm the binary on `PATH` is Cursor CLI (`cursor-agent` or a
   distinguishable `agent`), on macOS **and** native Windows. Capture
   `agent --version`. Do not treat an unrelated `agent` as this product.
2. Whether `agent ls` can emit a machine-readable list (then F2) or is
   interactive-only (stay off private files if a public list exists).
3. The session location on both OSes, and whether it is a database or
   plain files. If a database: schema version marker, table that carries
   turns, read-only/immutable open, fail closed on unknown schema.
4. Whether CLI and editor stores are actually separate on both OSes.
5. Whether the workspace path is recorded per session.
6. Whether reading while the editor or CLI is running is safe.
7. Put `cli-config.json`, `mcp.json`, and any discovered auth file in
   `Excluded` before any read.

Do not inspect a developer's real `~/.cursor` tree while filling this page.

## Sources

- [Cursor CLI overview](https://cursor.com/docs/cli/overview)
- [Cursor CLI installation](https://cursor.com/docs/cli/installation)
- [Using Agent in CLI](https://cursor.com/docs/cli/using)
- [Cursor CLI parameters](https://cursor.com/docs/cli/reference/parameters)
- [Cursor CLI configuration](https://cursor.com/docs/cli/reference/configuration)
- [Cursor CLI authentication](https://cursor.com/docs/cli/reference/authentication)
- [Cursor CLI past chats not showing up](https://forum.cursor.com/t/cursor-cli-past-chats-not-showing-up/152450) (forum; CLI vs IDE stores)
