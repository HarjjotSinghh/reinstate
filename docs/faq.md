# FAQ

## What is Reinstate?

The **continuity layer for coding-agent work**. Phase 1 implements encrypted,
bring-your-own-storage sync for same-vendor Claude Code and Codex sessions.
Stable Phase 2 adds universal local indexing, literal
search, metadata inspection, and same-vendor resume/fork without cloud
configuration. Stable `v0.3.0` added Phase 3 verified resume. Stable `v0.4.0`
adds Phase 4 explicit structured handoff, which continues the same task in a
new Claude Code or Codex session, after dual-platform tagged-artifact
acceptance PASS. A later universal configuration layer will reconcile supported MCP servers,
skills, hooks/loops, plugins, marketplaces, and safe settings across harnesses
and devices.

Spine: *Reinstate is not another place to code — it makes every place you code continuous.*

## What is `rein` vs `reinstate`?

**Same tool.**

| Name | Role |
| ---- | ---- |
| **Reinstate** | Product / brand / docs / repo |
| **`reinstate`** | Full CLI binary name |
| **`rein`** | Short alias (preferred day-to-day) |

```bash
rein version
reinstate version   # identical behavior
```

Config and data live under `~/.reinstate/` either way.

## Do local search and resume require `rein init` or a bucket?

**No.** In Phase 2 and later builds:

```bash
rein sessions
rein search "webhook retry"
rein inspect claude:SESSION_ID
rein resume claude:SESSION_ID --dry-run
```

Stable `v0.4.0` uses a private derived index at
`$REINSTATE_HOME/cache/session-index-v2.sqlite` (plus owner-only `.lock` and
`.write.lock` coordination files). Stable `v0.2.0` uses the earlier v1 index
and has no Phase 3 baselines; the paths are separate by design. Neither version
needs a sync profile, storage credentials, an encryption passphrase, keyring
access, or a network backend. Stable `v0.2.0` contains the Phase 1 sync surface
and Phase 2 local continuity. The public installers pin stable
`v0.4.0`. Intel macOS plus Linux/WSL2 remain optional and
unverified.

## Why not just use git?

Git is **source** truth. Sessions are **context** truth — the reasoning trail,
tool outputs, and decisions that are not in `git log`. Pulling commits and
asking a new agent to re-derive context is slow and incomplete.

## Will this resume a Claude session inside Codex?

**Native resume:** no — same-vendor only (Claude → Claude Code, Codex → Codex).

**Portable handoff (roadmap):** yes, as an *explicit* checkpoint (goal,
decisions, files touched, tests, next action) — not a silent perfect transcript
translation. See [product-strategy.md](product-strategy.md).

## What does verified resume verify?

In stable `v0.3.0`, `rein inspect`, native dry-runs,
direct `resume`/`fork`, `last`, and picker launches share one deterministic
environment report. It covers fresh session-source metadata, the selected
workspace and local Git state, the installed same-vendor agent/version/layout,
bounded instruction/skill/MCP logical names, and recognized Node/Go runtime
declarations.

It is verification, not repair. Reinstate does not fetch, clone, checkout,
reset, install, rewrite native configuration, run project scripts, or contact a
network service during preflight. It does not print dirty filenames, raw Git
remote URLs, instruction/skill contents, MCP commands/arguments/URLs/headers or
environment values, credentials, or raw environment dumps.

The first check reports `baseline.unavailable` because the vendor session does
not contain a complete historical environment snapshot. Only after an
authorized same-vendor child exits successfully does Reinstate retain the
immediately preceding observation as a private baseline. That proves what was
observed before the previous successful launch; it is not relabeled as
session-start truth.

## Can I bypass an environment warning or blocker?

Warnings require explicit consent for each launch. A terminal prompt defaults
to no. Non-interactive callers must repeat
`--allow-environment-warning CHECK_ID` for every exact current warning. The IDs
are invocation-scoped; unknown, stale, duplicate, wildcard, informational, and
blocker IDs are rejected.

Blockers cannot be bypassed. Missing workspaces/executables, unverified agent
versions/layouts, known repository replacement, stale selected-source
metadata, and verifier failures must be resolved before launch. There is no
broad `--force`, wildcard, saved approval, or environment-variable bypass.

## Do I need two computers?

**No.** Multi-device sync is the flagship wedge, but one machine with multiple
agents, sessions, projects, or worktrees is a first-class user. Phase 2 local
index/search/resume is built for that workflow and does not require remote
storage.

## Does local search upload or semantically analyze my prompts?

**No.** Search is literal, case-insensitive, and local. The index stores bounded
user-authored prompt text and known metadata/file references with owner-only
permissions. It excludes assistant messages/reasoning, tool output,
environment dumps, credentials, and auth stores. `search` does not print the
matching passage, and `inspect` caps its terminal-safe user preview at 160
Unicode code points.

User prompts can themselves contain sensitive text. The local index is not a
redaction or DLP product, and a compromised local machine remains outside the
threat model.

## Is Reinstate another ADE / agent IDE?

**No.** We do not replace Claude Code, Codex, Orca, Conductor, or Cursor as
execution environments. We make work **discoverable, verifiable, portable, and
syncable** across those tools.

## Will Reinstate configure the same MCP server in every harness?

**That is planned after Phase 1.** The target is to define a server once with
`rein mcp add`, preview native changes, and apply it across selected Claude
Code, Codex, Grok, OpenCode, Gemini CLI, and future adapters. The same model
extends to skills, instructions, hooks/loops, plugins, marketplaces, and safe
settings.

Reinstate will normalize intent and render each harness's real schema; it will
not blindly copy one tool's config file. See
[universal-configuration.md](universal-configuration.md).

## Will MCP authentication also carry across tools and devices?

Reinstate should make authentication status visible and coordinate supported
login flows, reducing repetition where the MCP protocol, provider, or harness
allows safe reuse. It will not sync raw API keys, OAuth tokens, cookies, or
vendor credential stores. When safe reuse is not supported, each target may
still require its official login flow.

## Is my data sent to Reinstate servers?

**No** for the open-source CLI. You point at **your** R2/S3-compatible bucket. A future
optional hosted convenience layer would still be zero-knowledge (ciphertext
only); it is not required.

## What if I lose my passphrase?

You cannot decrypt remote data. That is intentional (zero-knowledge). Keep the
passphrase in a password manager.

## Does this work offline?

Session files remain local, but the current `status`, `diff`, `push`, and `pull`
commands read the remote manifest and need access to your storage backend.
Phase 2 `sessions`, `search`, and `inspect` work offline and without sync
configuration. Phase 3 verification is also offline and never fetches. Native
resume/fork needs only the local same-vendor executable and recorded workspace;
the vendor itself may later use its own network features after launch.

## Windows + Mac?

It is the primary design target. Stable `v0.2.0` passed the complete physical
matrix on Apple Silicon macOS and native Windows x64. Intel macOS and
Linux/WSL2 packages are preview and unverified; their missing evidence is
reported explicitly rather than fabricated.

## Is this affiliated with Anthropic / OpenAI / Google / xAI?

**No.** Independent Apache-2.0 project by Harjot Singh Rana.

## Production ready?

Pre-1.0. Stable `v0.4.0` includes Phase 1 encrypted sync, Phase 2 local
continuity, Phase 3 verified resume, and Phase 4 structured handoff. Apple
Silicon macOS and native Windows x64 are physically verified; Intel macOS and
Linux/WSL2 remain preview and unverified. See [ROADMAP.md](../ROADMAP.md) and
[CHANGELOG.md](../CHANGELOG.md). Use with backups; report bugs via GitHub Issues.

## How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md).

## How do I report a security issue?

See [SECURITY.md](../SECURITY.md) — private disclosure only.
