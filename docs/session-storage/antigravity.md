# Antigravity CLI (Google)

**Confidence: Unverified** — no device probe, no index source, no reader.
Everything below is vendor documentation and must be treated as a hypothesis.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T0

Catalog key is `antigravity`. Descriptor:
`internal/agents/catalog/antigravity.go`.

## Why this agent is in the roster at all

On 2026-06-18 Google stopped serving Gemini CLI requests for Gemini Code
Assist for individuals, Google AI Pro, and Google AI Ultra accounts, and named
Antigravity CLI the migration destination. Gemini Code Assist Standard and
Enterprise licences and API-key authentication are unaffected.

So this is not a speculative addition. It is where the Gemini CLI consumer
population was told to go, and a continuity layer that indexes Gemini CLI but
not its successor has a hole in exactly the place users will notice.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Google |
| Product | Antigravity CLI |
| Binary | `agy` |
| Distribution | Official install script from `antigravity.google/cli/` |
| Install location | `~/.local/bin/agy`; on Windows `%LOCALAPPDATA%\agy\bin` |
| Storage family | F1 expected (home tree); unverified |
| Source | closed-source Go binary |

Antigravity CLI is **not** the Antigravity desktop IDE. The two share an
account and some configuration but install and run independently. This page
and the descriptor cover the CLI only.

## The collision that matters

Antigravity CLI does not create its own home root. Vendor documentation places
its configuration and data under **`~/.gemini/antigravity-cli/`** — inside the
root the shipped Gemini CLI descriptor already owns.

Two consequences, both already handled in code:

1. The Gemini CLI descriptor excludes `antigravity-cli` from its storage
   walk. Without that, a Gemini probe or scan would descend into a different
   catalog agent's tree, and `rein doctor --agents` would attribute one
   agent's files to another.
2. On Linux the OAuth token is a plain file at
   `~/.gemini/antigravity-cli/antigravity-oauth-token`. It is excluded in both
   descriptors before any read. On macOS the token goes to the Keychain under
   "Antigravity Safe Storage" instead, but a descriptor is one contract across
   platforms and excludes it everywhere.

The installer also copies an existing Gemini CLI setup — skills, MCP servers,
agent profiles — into the new tree at install time. Anyone probing Gemini CLI
for Phase 5 evidence should capture that probe **before** installing
Antigravity CLI, because the install perturbs the tree being measured.

## Claimed layout

Every row is **Unverified**. No probe has observed this tree.

| Aspect | Vendor claim |
| ------ | ------------ |
| Root | `~/.gemini/antigravity-cli/` |
| Settings | `settings.json`, written sparsely — only values differing from defaults |
| Keybindings | `keybindings.json` |
| MCP config | `mcp_config.json` |
| Conversations | a workspace-keyed cache under `cache/`, reported as `cache/last_conversations.json` |
| Credentials | `antigravity-oauth-token` (Linux); Keychain (macOS); Credential Manager (Windows) |
| Scratch workspace | `~/.gemini/<…>/scratch/` |
| Global context | `~/.gemini/GEMINI.md`; per-workspace `.antigravity.md`, `GEMINI.md`, `AGENTS.md` |

Native control surface, vendor documentation only:

| Aspect | Value |
| ------ | ----- |
| Resume | `/resume` lists and resumes previous conversation logs |
| Auto-save resume | on exit the CLI prints the exact command to resume that session |
| Fork | `/fork` branches the conversation into a separate workspace |
| Rewind | `/rewind` or `/undo` |

## What a probe must settle

1. **Whether conversations are stored at all, or only cached.** The one
   documented conversation path is named `cache/last_conversations.json`, and
   a vendor that calls something a cache usually means it. `/logout` is
   documented as clearing "local cache directories", which is the behaviour of
   a cache, not of a session store. If that file is the only record, this
   agent is closer to `server_backed` than to F1 and stays T0.
2. The real filename, format, and record shape under `cache/`. A single JSON
   file holding every workspace's last conversation is a very different
   scanner from one file per session.
3. Whether the CLI prints a parseable version, and under which flag. The
   descriptor has no `VersionSpec` because none has been observed.
4. Whether the marker `cache` is correct. It is a guess taken from the
   conversation path above, chosen so a bare `~/.gemini/antigravity-cli`
   created by the installer does not resolve as a populated agent.
5. Whether an environment variable relocates the root. None is documented, so
   the descriptor declares no `RootEnv`.

## Expected outcome

**T0 for `v0.5.0`.** The `v0.5.0` target in
[docs/agent-support-tiers.md](../agent-support-tiers.md) is T0, not T1. This
entry exists so the catalog knows the agent, so `rein doctor --agents` can
inventory it, and so the Gemini CLI exclusion has a documented reason — not
because a reader is planned in this phase.

Promotion requires the same evidence as every other agent: a committed macOS
`AGENT-PROBE-V1` artifact and a native Windows one, after real use. Vendor
documentation is not a tier promotion, and this page is entirely vendor
documentation.

## Sources

- [Installation & Auth](https://www.antigravity.google/docs/cli/install/)
- [Settings, Rendering & Keybindings](https://www.antigravity.google/docs/cli/settings/)
- [Using AGY CLI](https://www.antigravity.google/docs/cli/using/)
- [Gemini Code Assist consumer account deprecation](https://developers.google.com/gemini-code-assist/docs/deprecations/code-assist-individuals)
- [Gemini CLI transition announcement](https://github.com/google-gemini/gemini-cli/discussions/28017)
