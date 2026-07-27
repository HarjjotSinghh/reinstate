# Universal agent configuration (roadmap)

> Status: planned after Phase 1. None of the commands on this page are part of
> the current `v0.1` CLI unless explicitly documented in
> [cli-reference.md](cli-reference.md).

Reinstate will eventually make the **agent development environment** portable,
not only the session history. A user should be able to declare an MCP server,
skill, hook, loop, plugin, marketplace, instruction file, or other supported
non-secret setting once and reconcile it across:

- multiple coding-agent harnesses on one machine; and
- the same or different harnesses on multiple machines.

For example, a developer using Claude Code, Codex, Grok, OpenCode (including
provider-specific setups), and Gemini CLI should not have to copy the same MCP
JSON into every tool by hand. Reinstate will maintain one desired-state profile
and let adapters render the correct native configuration for each harness.

This is **universal agent configuration**, not blind directory mirroring and not
a Reinstate-owned coding harness.

## Intended experience

The exact CLI is subject to design, but the target workflow is:

```bash
# Define once in the Reinstate profile.
rein mcp add mobbin --url https://example.invalid/mcp --target all

# See the harness-specific changes before writing them.
rein config diff --target claude,codex,grok,opencode,gemini

# Reconcile supported harnesses on this machine.
rein config apply --all

# Encrypt and distribute the non-secret desired state to other devices.
rein config sync
```

Equivalent families are planned for other capability types:

```text
rein skill install …
rein loop install …
rein plugin install …
rein marketplace add …
rein config import|list|diff|apply|status|sync …
rein auth status …
```

Command names and schemas will be stabilized through an RFC before
implementation.

## What the master profile represents

The Reinstate profile is a canonical, versioned declaration of desired state.
It may contain:

| Capability | Portable declaration |
| ---------- | -------------------- |
| MCP servers | name, transport, command or URL, non-secret arguments, environment-variable names, secret references, target harnesses |
| Skills and instructions | source, version or digest, scope, target harnesses, installation mode |
| Hooks, commands, and loops | lifecycle event, portable workflow definition, compatibility requirements |
| Plugins and extensions | package/source identity, pinned version, target harnesses, required permissions |
| Marketplaces / registries | source declaration, trust policy, enabled targets |
| Safe settings | normalized non-secret preferences supported by an adapter |

Scopes should include user, project, device, and eventually team policy. A
device or harness can opt out of a capability without forking the whole profile.

## Normalize, then render

Harnesses do not use one common schema or filesystem layout. Reinstate therefore
must not copy one tool's configuration file over another tool's file.

The configuration pipeline is:

```text
native harness config
        ↕ import / render
configuration adapter
        ↕
versioned Reinstate desired-state profile
        ↕ encrypt / sync
other devices
```

Each configuration adapter must:

1. advertise which capabilities and fields the harness supports;
2. translate only fields it understands;
3. report unsupported or lossy mappings before apply;
4. preserve unrelated user-managed settings;
5. preview changes, back up affected files, and write atomically;
6. detect drift between desired state and native harness state; and
7. fail closed when a harness version or schema is unverified.

This follows the same honesty rule as session continuity: no silent
cross-harness magic.

## Authentication and secrets

The frustrating case is real: an OAuth-backed MCP server may ask the user to
authenticate separately in every coding tool. Reinstate should reduce that
friction without turning a portable config profile into a credential bundle.

The security contract is:

- Raw API keys, OAuth tokens, cookies, and vendor credential stores are never
  placed in the synced profile.
- Portable declarations contain **secret references**, not secret values.
- Local secrets may be resolved from the OS keychain or an explicitly
  configured secret provider.
- `rein auth status` should show which device/harness combinations still need
  authentication without printing sensitive material.
- Where an MCP protocol, provider, or harness explicitly supports safe token
  reuse, Reinstate may coordinate that supported flow.
- Otherwise, Reinstate may launch the official login flow for each target and
  track completion, but it must not copy or reinterpret private credential
  stores to bypass harness security boundaries.

The goal is “configure once, authenticate as few times as safely possible,” not
“sync every token.”

## Installation and supply-chain safety

Skills, plugins, hooks, and marketplaces may execute code. Applying them is more
sensitive than writing an inert setting. The eventual implementation must
support:

- source and version pinning;
- digest/signature verification where available;
- a dry-run showing downloads, commands, permissions, and target paths;
- explicit confirmation before executing installers or enabling new code;
- allow/deny policy by capability type and source;
- reproducible removal and rollback; and
- no automatic trust inheritance merely because a source is present on another
  device.

Reinstate coordinates native harness installation mechanisms; it does not
become a universal plugin runtime or operate its own agent marketplace.

## Relationship to session continuity

Sessions remain the first release and acquisition wedge. Universal
configuration deepens continuity:

1. Reinstate finds the task.
2. Verified resume detects missing MCP servers, skills, runtimes, or settings.
3. Universal configuration can repair supported differences.
4. Reinstate resumes natively or produces an explicit portable handoff.
5. Encrypted sync carries sessions and non-secret desired state to another
   device.

See [ROADMAP.md](../ROADMAP.md), [architecture.md](architecture.md), and
[security-model.md](security-model.md).
