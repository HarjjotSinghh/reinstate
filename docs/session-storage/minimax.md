# MiniMax Code (MiniMax)

**Confidence: Unverified** — no Reinstate reader exists, and no device
probe has been captured.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T0 until a
probe distinguishes local history from a cache of account state.

**Identification (T-044):** official coding harness found. No catalog key is
assigned in this research pass.

## Status: official harness identified

The Phase 5 roster name "MiniMax" is MiniMax's first-party coding product
**MiniMax Code**, not a MiniMax model running inside another vendor's
harness.

MiniMax's homepage calls MiniMax Code "the coding harness built for MiniMax
models." Product docs describe it as a desktop AI Agent app for software
development. The Token Plan page calls MiniMax Code "the official AI agent"
and lists MiniMax Code Desktop and MiniMax Code Web as first-party surfaces.

That identification closes the T-044 research question. It does **not**
authorize a catalog key, a descriptor, or a reader. The key, display name,
and vendor string are a public interface. This page records the product; a
later maintainer task chooses the key.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | MiniMax |
| Product | MiniMax Code |
| Former desktop name | MiniMax Agent (changelog: desktop rebrand to MiniMax Code, v3.0.33) |
| Surfaces | Official desktop app (macOS, Windows); official web app |
| Terminal client | not documented as the product. Changelog mentions a `minimax` CLI shortcut at rebrand; purpose and session behavior are unstated. Distinct from `mmx` (see below). |
| Distribution | Official installers from MiniMax (`agent.minimax.io` global, `agent.minimaxi.com` mainland China) |
| Storage family | F5 expected (desktop-first) pending probe |

## Catalog key

**None.** Do not ship `minimax`, `minimax-code`, or any other key from this
page. Reasons a guess would be wrong later:

1. The roster placeholder is `minimax`, but the identified product is MiniMax
   Code. The vendor also ships MiniMax models, MiniMax CLI (`mmx`), MiniMax
   Agent on the web, MaxClaw, and MaxHermes.
2. The changelog's `minimax` CLI shortcut is a one-line mention, not a
   documented session harness.
3. MiniMax-AI/cli already uses the binary `mmx` (renamed from `minimax`) for
   a different product.

A wrong key cannot be renamed without breaking `agent:session` references,
`rein doctor` output, and the compatibility matrix.

## What this is not

These official MiniMax artifacts are **not** the coding-session catalog
entry. Sessions written by another harness stay with that harness.

| Artifact | What it is | Why it is not the catalog agent |
| -------- | ---------- | ------------------------------- |
| MiniMax M-series models (M2, M2.1, M2.5, M3, …) | Models on the MiniMax Open Platform / Token Plan | Vendor Token Plan docs tell users to drop an `sk-cp` key into Claude Code, Codex, Cursor, OpenCode, Cline, and other OpenAI-compatible tools. Those sessions live in the host harness. Reinstate indexes harnesses, not models. |
| MiniMax CLI (`mmx`, npm `mmx-cli`, MiniMax-AI/cli) | Official platform CLI for text, image, video, speech, music, vision, and search | Built to be called *from* other agents. Config/credentials default to `~/.mmx/config.json` (`MMX_CONFIG_DIR`). Not a coding-session store. |
| Mini-Agent (`mini-agent`, MiniMax-AI/Mini-Agent) | Official "minimal yet professional demo" for building agents on MiniMax models | Demo / starter, not the product users mean by MiniMax Code. Config lives under `~/.mini-agent/` when installed as a tool. |
| MiniMax-AI/minimax-code | Official issue tracker for the desktop app | README: collect bug reports; download links point at the desktop installer. Not a CLI source tree. |
| MiniMax-AI/skills | Official skill pack for *other* coding tools | README installs into Claude Code, Codex, OpenCode, Cursor. Confirms MiniMax also sells into third-party harnesses. |
| MaxClaw / MaxHermes | Separate MiniMax agent products (cloud / OpenClaw-based, Hermes-based) | Not MiniMax Code. Do not fold them into a MiniMax Code descriptor. |

A community issue on MiniMax-AI/minimax-code asks MiniMax to ship a native
CLI-first coding harness. That is Observed only: it is consistent with the
official product being the desktop (and web) app, not a Claude Code-class
terminal agent. It is not a vendor storage claim.

## What vendor docs state

Every storage row stays **Unverified**. Vendor documentation alone is never
sufficient for a tier above T0.

| Aspect | Vendor statement | Confidence |
| ------ | ---------------- | ---------- |
| Product | Desktop AI Agent app: chat, project context, file operations, terminal, browser, skills, memory, automation | Documented |
| Platforms | macOS 11+ and Windows 10+; arm64 and Intel Mac installers; Windows 64-bit | Documented |
| Account | Sign in with a MiniMax account after install | Documented |
| Local data | FAQ: desktop "primarily uses a local runtime and local data directory." Web and mobile sync "follows their respective product surfaces." | Documented (existence only) |
| Windows data root | FAQ: Windows supports a custom data directory | Documented (override exists; path unstated) |
| Home directories named in support | Common-issues page tells users to inspect, back up, and if needed delete `~/.mavis` and `~/.minimax` (`%USERPROFILE%\.mavis`, `%USERPROFILE%\.minimax`). Those directories "may contain local settings, cache data, or session-related data." The app regenerates them after deletion. | Documented (contents unspecified) |
| History UI | Every conversation creates a task record. Sidebar: search, pin, project-grouped history, archive, jump to a turn | Documented (in-app UI, not an on-disk format) |
| Session identifier | Feedback from a task attaches task title, **session ID**, workspace path, client version | Documented (ID exists; storage location unstated) |
| Remote / IM | Remote Control pairs a phone to the running desktop conversation. Messaging integrations: Telegram, WeChat, Lark, or Feishu by locale | Documented (continuity signal, not a layout) |
| Runtime history | Windows clients after `3.0.48` "no longer depend on OpenCode." Older OpenCode / daemon troubleshooting does not apply to current versions. Changelog also mentions `/resume` framework session-id fixes in pre-2.0 builds | Documented (do not reuse the OpenCode reader) |
| Resume argv | none found in official MiniMax Code docs | Unverified / absent from docs |
| Transcript format | none published | Unverified |
| Export API | none found for conversation export | Unverified |

Electron / Chromium appears in official Windows troubleshooting (renderer
sandbox, `MiniMax Internal Code.exe` in one example path). That is a
packaging signal, not a session-layout claim.

## Claimed layout (all Unverified)

Do not write a scanner from this table.

| Aspect | Value | Notes |
| ------ | ----- | ----- |
| Roots named by support | `~/.minimax`, `~/.mavis` | May be settings, cache, session-related data, or all three. Deleting them is the vendor's repair step; they come back on restart. That is compatible with either a local store or a rebuildable cache. |
| Windows custom data directory | unstated path | Documented to exist; location not published. |
| App / Electron user-data | unstated | Check the platform application-support / user-data directory in a later probe. |
| Session files | not stated | History is a product UI. No file names, extensions, or schemas. |
| Credentials | MiniMax account sign-in; Token Plan / API keys for BYOK | If they sit in the same tree as any later-found transcripts, they go in `Excluded` before any read. |
| Platform CLI credentials (not this product) | `~/.mmx/config.json` | MiniMax CLI (`mmx`) only. Do not treat as MiniMax Code sessions. |
| Demo config (not this product) | `~/.mini-agent/` | Mini-Agent only. |

## The cache trap

Three cases still look identical on disk:

1. **Local authoritative history** — survives reinstall and re-login. T1
   might be reachable later.
2. **Local cache of server or account state** — rebuilt after a clear.
   Not indexable.
3. **Nothing durable locally.** T0, `server_backed`.

MiniMax Code has signals in both directions. The FAQ emphasizes a local
runtime and local data directory, and says desktop data is not simply the
web app. Remote Control, IM channels, account sign-in, and a MiniMax Code
Web surface are also consistent with account-held or hybrid state.

Distinguishing case 1 from case 2 requires observing the tree across a
cache clear or a re-login. Do not infer authority from the directory name
`.minimax` or from the phrase "session-related data."

## What a later probe must settle

No probe in this task. When one runs, it must use the MiniMax-distributed
desktop app (current 3.x, not a pre-`3.0.48` OpenCode-era build) on macOS
and native Windows:

1. Whether `~/.minimax`, `~/.mavis`, a Windows custom data directory, or
   the Electron user-data tree contains a transcript vs UI/cache/logs.
2. Which of the three cache-trap cases applies after a cache clear and
   after a re-login.
3. Whether the project or workspace path is recoverable from any local
   artifact (docs attach a workspace path to feedback, but not to a file).
4. Whether authentication material sits in the same tree.
5. Whether any documented resume argv or conversation-export API exists
   for the current desktop build. A `minimax` PATH shim, if present, is
   not a resume contract until vendor docs say so.
6. Confirm `mmx` / `~/.mmx` and `mini-agent` / `~/.mini-agent` are
   separate products and are not scanned as MiniMax Code.

If the probe finds no durable local transcript, ship T0 with
`server_backed` or `desktop_only` as the evidence warrants, and stop.
Indexing a cache is a defect.

## Recording this outcome

T-044 research is complete:

- Official harness: **MiniMax Code** (desktop; web is a sibling surface).
- Close reason: **official harness found**.
- Catalog: **no key**. Escalate key choice; do not invent one here.
- Implementation: **none**. No `internal/agents/catalog/minimax.go`.

`unidentified_product` is no longer the reason. The product is named. The
layout is not.

## Sources

Official, used for identification and the Unverified rows above:

- [MiniMax homepage — MiniMax Code](https://www.minimax.io/)
- [MiniMax Code welcome](https://agent.minimax.io/docs/code/welcome)
- [Download and install](https://agent.minimax.io/docs/code/get-started/download)
- [Tasks and history](https://agent.minimax.io/docs/code/workflows/tasks)
- [FAQ (local data directory; Windows custom directory)](https://agent.minimax.io/docs/code/help/faq)
- [Common issues (`~/.mavis`, `~/.minimax`; post-3.0.48 OpenCode note)](https://agent.minimax.io/docs/question)
- [Feedback (session ID)](https://agent.minimax.io/docs/code/help/feedback)
- [Remote Control](https://agent.minimax.io/docs/code/automation/remote-control)
- [Changelog (desktop rebrand to MiniMax Code, v3.0.33; Code 2.0 / 3.0.48)](https://agent.minimax.io/docs/changelog)
- [MiniMax-AI/minimax-code](https://github.com/MiniMax-AI/minimax-code) — desktop issue tracker
- [Token Plan quick start (models in third-party tools)](https://platform.minimax.io/docs/token-plan/quickstart)
- [MiniMax CLI (`mmx`)](https://github.com/MiniMax-AI/cli)
- [MiniMax CLI docs](https://platform.minimax.io/docs/token-plan/minimax-cli)
- [Mini-Agent demo](https://github.com/MiniMax-AI/Mini-Agent)
- [MiniMax-AI/skills](https://github.com/MiniMax-AI/skills)
