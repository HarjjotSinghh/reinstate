# ZCode (Z.ai)

**Confidence: Documented** for identity, official distribution, and the T0
reason; **Unverified** for every session-file path.
**Current tier:** T0 (`desktop_only`) · **Phase 5 target:** T0 (research
complete). Revisit only if a later probe of the Z.ai-distributed desktop app
finds an authoritative local transcript.

Catalog key remains `zcode`. ZCode is Z.ai's official Agentic Development
Environment (ADE) for GLM models. Z.ai distributes it as a desktop application
from [zcode.z.ai](https://zcode.z.ai/en), not as a vendor terminal CLI.

## Identity

| Aspect | Value |
| ------ | ----- |
| Catalog key | `zcode` |
| Vendor | Z.ai |
| Product | ZCode |
| Official distribution | Desktop ADE: macOS `.dmg` (`ZCode.app`), Windows installer (x64 / ARM64), Linux `.deb` and `.AppImage` (beta). Latest installer observed on the vendor site: v3.7.7 (2026-08-14). |
| Official download | [zcode.z.ai/en](https://zcode.z.ai/en) · [Install](https://zcode.z.ai/en/docs/install) |
| Linux desktop binary | `zcode` from the vendor `.deb` (launches the GUI; not a session CLI) |
| URL scheme | `zcode://` (OAuth / login callback) |
| Terminal client | none from Z.ai |
| npm `zcode-app-cli` | unaffiliated third-party extractor; **not a catalog agent** |
| Storage family | F5 (desktop-only) |
| Resume argv | none — Z.ai does not ship a terminal client |

## Distribution policy

[ADR 0004](../adr/0004-universal-agent-coverage.md) decision 8 restricts the
catalog to officially distributed harnesses. The npm package `zcode-app-cli`
extracts `resources/glm` from a ZCode desktop install and launches it with a
substituted terminal UI. Its own README states it is not affiliated with or
endorsed by Z.ai and asks users to confirm redistribution rights.

Reinstate targets the Z.ai-distributed desktop application only. The npm
package is not a catalog agent and must not be used to derive a supported
layout. If that package and the desktop app happen to share a tree, that fact
is established only by probing the **desktop application**.

This mirrors how [ADR 0003](../adr/0003-phase-4-rc1-scope-and-launch-route.md)
resolved the Grok CLI flavor question: the official harness is what users mean
by the product name.

## Why the outcome is T0 `desktop_only`

The vendor markets follow-up from desktop, mobile Remote, and Feishu / WeChat
Bot. That is a signal of a coordination server (QR pairing, bot binding,
OAuth). Official docs then state that those surfaces are **control channels
for a session that already exists on the desktop**, not an independent
server-held transcript:

1. **Remote Control** "temporarily opens the current ZCode desktop window to
   your phone." The phone "doesn't sync code and never creates a runtime of
   its own." The desktop must stay running; once it disconnects, the phone
   cannot continue ([Remote Control](https://zcode.z.ai/en/docs/remote-control)).
2. **Bot Channel** forwards WeChat / Feishu messages to "ZCode Agent" which
   "continues the task in the desktop workspace"
   ([Bot Channel](https://zcode.z.ai/en/docs/bot-channel)).
3. ADE Tools: Remote Control and Bot Channel "both drive a session that
   already exists on the desktop"
   ([ADE Tools](https://zcode.z.ai/en/docs/ADE-tools)).
4. Conversations restore after an app restart, and the app recovers a
   corrupted **task index** at startup ([changelog v3.7.6](https://zcode.z.ai/en/changelog)).
   The FAQ treats "session history" as a local artifact that is "not worth
   moving" when switching machines ([FAQ Q12](https://zcode.z.ai/en/docs/qa)).

That is `desktop_only`, not `server_backed`. Authoritative history is described
as belonging to the desktop window. There is still no documented session-file
layout, no vendor session-export API, and no probe, so the agent stays at T0.
Vendor documentation alone is never enough to promote a tier.

`rein doctor --agents` should tell the user:

> ZCode detected. Session history is not readable locally
> (`desktop_only`). Reinstate cannot index ZCode sessions.

That is better than the agent being absent from the output.

## Documented local paths (not a session store)

Official docs describe a user tree at `~/.zcode` (Windows
`%USERPROFILE%\.zcode`) for **configuration, credentials, logs, and command
output**. None of these rows is a documented transcript.

| Path | What the vendor says it is |
| ---- | -------------------------- |
| `~/.zcode/cli/config.json` | MCP servers, permission defaults, plugin and skill toggles, hooks |
| `~/.zcode/v2/config.json` | Model provider setup (API keys, base URLs, model lists) |
| `~/.zcode/v2/credentials.json` | Login credentials, encrypted per device — do not copy |
| `~/.zcode/v2/telemetry-state.json` | Device identifier — do not copy |
| `~/.zcode/AGENTS.md` | Global instructions |
| `~/.zcode/agents/`, `skills/`, `commands/` | Custom subagents, skills, slash commands |
| `~/.zcode/cli/exec` | Terminal command output from past sessions; vendor says it can be deleted and is recreated |
| `~/.zcode` `logs/` | Diagnostic logs; **Export Logs** packages them |

`<workspace>/.zcode/` is project-level config that travels with the
repository.

**Do not index any of the above as sessions.** `credentials.json`,
`v2/config.json` (API keys), and `telemetry-state.json` belong in `Excluded`
before any later read. `cli/exec` is a command-output cache, not a transcript.

The FAQ mentions "session history" as something local that is not worth
migrating, but it does **not** give a path. Changelog "task index" recovery
likewise names no file.

## Unverified probe candidates

A later desktop-app probe may look here. These are candidates, not facts.
Nothing below may be treated as a supported layout.

1. `~/.zcode` (and Windows `%USERPROFILE%\.zcode`) — documented config root;
   session files, if any, are unstated.
2. Platform application-support directories typical of a Chromium / Electron
   desktop app (vendor does not name them):
   macOS `~/Library/Application Support/ZCode` or `…/zcode`;
   Windows `%APPDATA%\ZCode` or `%APPDATA%\zcode`;
   Linux `~/.config/ZCode` or `~/.config/zcode`.
3. The Electron / Chromium user-data directory for the same app id, if it
   differs from (2). Linux troubleshooting documents Chromium-style flags
   (`--disable-gpu`, `--no-sandbox`) and a `zcode.desktop` handler; that is
   not a path.

If artifacts appear, the probe must distinguish a durable transcript from a
cache of UI state. A cache is not a session. Distinguishing them requires
observing the tree across a cache clear or re-login, not inferring from a
directory name.

Do not inspect a developer's real Z.ai or ZCode tree while contributing. Use
only a dedicated probe machine and redacted reports.

## Session or export API

Z.ai does **not** publish a ZCode-desktop session-list or session-export API.

- The in-app **Export Logs** action packages diagnostic logs, not
  conversations ([Feedback](https://zcode.z.ai/en/docs/feedback)).
- First-launch **Data Migration Wizard** imports conversations from Claude
  Code and "ZCode Agent in legacy ZCode" only
  ([Install](https://zcode.z.ai/en/docs/install)). That is an inbound
  importer, not an export API.
- `POST https://api.z.ai/api/v1/agents/conversation` is the cloud
  [Agent API conversation-history](https://docs.z.ai/api-reference/agents/agent-conversation)
  endpoint (`agent_id` / `conversation_id`, slide/PDF variables). It is not
  documented as a ZCode desktop session store.

A network source for ZCode would be a first for the catalog and is a
maintainer decision. Escalate before implementing one.

## What a later probe must settle

T-043 is complete at T0. These questions are for a future revisit, not this
page:

1. Whether the vendor desktop app writes a local session artifact, and where.
2. Whether that artifact is a full transcript or a cache.
3. Whether the project path is recoverable from it.
4. Whether any of the unverified candidate directories exist on macOS and
   native Windows after a real desktop install.

Do not stretch to T1 on vendor prose. Do not derive a layout from
`zcode-app-cli`.

## Sources

- [ZCode product / downloads](https://zcode.z.ai/en) — official desktop
  installers
- [Install](https://zcode.z.ai/en/docs/install) — `ZCode.dmg` / `ZCode.app`,
  Windows installer, Linux AppImage / `.deb`; data-migration wizard
- [Welcome](https://zcode.z.ai/en/docs/welcome) — ADE; follow-up across
  desktop, Remote, and Bot
- [Remote Control](https://zcode.z.ai/en/docs/remote-control) — phone is a
  control surface; desktop must stay running
- [Bot Channel](https://zcode.z.ai/en/docs/bot-channel) — chat forwards to the
  desktop workspace
- [ADE Tools](https://zcode.z.ai/en/docs/ADE-tools) — remote/bot drive a
  desktop session
- [FAQ Q12–Q13](https://zcode.z.ai/en/docs/qa) — `~/.zcode` config map;
  session history not worth moving; `cli/exec` is recreatable
- [Feedback / Export Logs](https://zcode.z.ai/en/docs/feedback) — `~/.zcode`
  logs folder
- [Subagents](https://zcode.z.ai/en/docs/subagents) — `~/.zcode/agents/<name>.md`
- [Changelog](https://zcode.z.ai/en/changelog) — local task-index recovery;
  conversations restore after restart
- [Linux / WSL](https://zcode.z.ai/en/docs/linux-wsl) — `zcode` GUI binary;
  `zcode://`; Chromium-style flags
- [Z.AI Agent API conversation](https://docs.z.ai/api-reference/agents/agent-conversation)
  — different product; not a ZCode desktop export
- [zcode-app-cli on npm](https://www.npmjs.com/package/zcode-app-cli) —
  unaffiliated third-party package, recorded for exclusion only
