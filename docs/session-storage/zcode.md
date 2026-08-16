# ZCode (Z.ai)

**Confidence: Unverified** — no Reinstate reader exists.
**Current tier:** T0 (`desktop_only`) · **Phase 5 target:** T0, revisited only
if a probe finds a local session tree in the vendor-distributed application.

ZCode is Z.ai's agentic development environment for GLM models. It is
distributed as a desktop application, not a terminal CLI, which puts it in the
same category as other GUI-first harnesses rather than in the same category as
Claude Code or Codex.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Z.ai |
| Product | ZCode |
| Distribution | Desktop application (macOS `.dmg` and other platform installers) |
| Terminal client | `zcode-app-cli` on npm — **unaffiliated with Z.ai** |
| Storage family | F5 (desktop-only) pending probe |

## Distribution policy

[ADR 0004](../adr/0004-universal-agent-coverage.md) decision 8 restricts the
catalog to officially distributed harnesses. The npm package `zcode-app-cli`
works by extracting the `resources/glm` runtime out of the ZCode desktop
installation and launching it with a substituted terminal UI. Its own README
states it is not affiliated with or endorsed by Z.ai and asks users to confirm
redistribution rights.

Reinstate therefore targets the Z.ai-distributed application only. The npm
client is not a catalog agent and must not be used to derive a supported
layout. If the extracted runtime and the desktop application share a session
tree, that fact is established by probing the **desktop application**.

This mirrors how [ADR 0003](../adr/0003-phase-4-rc1-scope-and-launch-route.md)
resolved the Grok CLI flavor question: the official harness is what users mean
by the product name.

## What is known

| Aspect | Value |
| ------ | ----- |
| Config (unofficial CLI) | `~/.zcode/cli/config.json`, Windows `%USERPROFILE%\.zcode\cli\config.json` |
| Session storage | not stated anywhere Reinstate has verified |
| Cross-device behavior | vendor markets follow-up across desktop, remote, and bot channels, which suggests server-side session state |
| Resume argv | none — no vendor terminal client exists |

The vendor's own description of continuing the same task from desktop, a
remote surface, and chat channels is a strong signal that authoritative session
state lives on Z.ai infrastructure rather than on disk. If that is true, ZCode
stays T0 with reason `server_backed`, and that is a complete and useful answer
for a user asking whether Reinstate can index their ZCode work.

## What the probe must settle

1. Whether the desktop application writes any local session artifact, and
   where. Check `~/.zcode`, the platform application-support directory, and
   the Electron user-data directory.
2. If artifacts exist, whether they are a full transcript or a cache of
   server-held state. A cache is not a session; indexing one produces records
   that vanish.
3. Whether Z.ai publishes a session or export API. A supported interface would
   make ZCode an F2 agent and would be preferable to reading Electron internals.

## Recording a T0 outcome

If the probe finds no usable local artifact, that is a completed task, not a
failed one. The descriptor ships with `Tier: TierKnown` and the accurate
`T0Reason`, and `rein doctor --agents` tells the user:

> ZCode detected. Session history is not readable locally
> (`server_backed`). Reinstate cannot index ZCode sessions.

That is materially better than the agent being absent from the output, which a
user reads as "Reinstate has never heard of this".

## Sources

- [ZCode](https://zcode.z.ai/en)
- [ZCode documentation](https://zcode.z.ai/en/docs/welcome)
- [zcode-app-cli on npm](https://www.npmjs.com/package/zcode-app-cli) — unaffiliated third-party package, recorded for context only
