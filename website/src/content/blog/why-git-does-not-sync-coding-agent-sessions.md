---
title: "Why Git Alone Does Not Sync Coding-Agent Sessions"
description: "Git preserves source history, but coding-agent continuity also depends on local conversations, tool context, vendor layouts, paths, and safe restore."
answer: "Git does not automatically sync coding-agent sessions because those conversations and vendor runtime records usually live outside the repository. Even when copied manually, they need compatibility checks, secret exclusions, path remapping, encryption, conflict handling, and native-agent restore."
author: "Harjot Singh Rana"
publishedAt: 2026-07-27
updatedAt: 2026-07-27
reviewedAt: 2026-07-27
tags: ["Git", "coding agents", "session continuity", "developer workflow", "Reinstate"]
targetQuery: "does Git sync coding agent sessions"
searchIntent: "problem"
related:
  - title: "Claude Code integration"
    path: "/integrations/claude-code"
  - title: "Codex CLI integration"
    path: "/integrations/codex"
  - title: "How Reinstate works"
    path: "/about/reinstate"
  - title: "Claude Code session sync guide"
    path: "/guides/sync-claude-code-sessions-across-devices"
  - title: "Codex session sync guide"
    path: "/guides/sync-codex-sessions-across-devices"
  - title: "Security model"
    path: "/docs/security-model"
draft: false
noindex: false
category: "engineering"
featured: true
---

## Git preserves the repository, not the whole coding session

Git is the right source of truth for code and committed project history. It
records snapshots of files that belong to a repository, plus the graph of
commits that connects them. A coding-agent session contains a different kind
of state: the conversation, tool calls, rejected approaches, files already
examined, working-directory identity, and vendor-specific resume records.

Claude Code and Codex CLI keep their resumable session artifacts in
vendor-managed locations under the user's home directory. Those files are not
part of a project repository unless someone deliberately copies or links them
into it. A normal `git push` therefore sends the repository history but leaves
the native agent session on the original machine.

This distinction is why a clean checkout on a second computer can contain every
commit and still force the agent to rebuild the task context from scratch.

## What is missing after a normal Git pull

Suppose a developer commits a refactor on a Windows workstation and opens the
repository on a MacBook. Git can restore:

- committed source files;
- branches, tags, and commit ancestry;
- repository-tracked instructions and configuration; and
- any other file intentionally added to that repository.

Git does not automatically restore:

- the Claude Code or Codex conversation that led to those edits;
- uncommitted agent reasoning or decisions preserved only in a session;
- vendor-native resume identifiers and rollout layout;
- the mapping between the Windows and macOS checkout paths;
- session-specific conflict and restore state; or
- credentials and encryption material, which should not be committed at all.

That does not make Git incomplete. It means source history and agent-session
continuity are separate responsibilities.

## Why committing raw session files is not a safe general solution

Git can technically track any file placed inside a repository. That capability
does not make a vendor session store suitable project content.

### Sessions can contain sensitive material

Coding-agent transcripts may include source excerpts, architecture decisions,
terminal output, and secrets that a person or tool printed during the session.
Committing that material can spread it to every clone, fork, cache, and backup
that receives the repository.

Reinstate's [security contract](/docs/security-model) treats transcripts as
high-sensitivity data. It encrypts selected session artifacts before upload
and hard-excludes known authentication and credential paths. Encryption does
not excuse putting secrets into transcripts, but it avoids turning session
continuity into plaintext repository history.

### Vendor layouts are not portable repository formats

Claude Code and Codex use different native session layouts. Their resume paths
are agent-specific, and compatibility can change across agent versions.
Reinstate uses separate
[Claude Code and Codex adapters](/docs/adapters) to discover, export, validate,
and restore only the supported native artifacts.

A raw directory committed on one machine has no mechanism to establish that
the destination's installed agent understands that layout. Reinstate reports
`SUPPORTED`, `UNTESTED`, `UNSUPPORTED`, or `NOT_INSTALLED` and blocks writes
when a recognized layout lacks release evidence.

### Absolute project paths change between devices

A repository might be `C:\src\app` on Windows and `/Users/dev/src/app` on
macOS. Claude Code derives project storage from the local project path, while a
Codex rollout records its working directory in structural session metadata.
Copying the bytes does not make either source path valid on the destination.

Reinstate's path-mapping layer gives the repository a canonical project ID. Its
adapters normalize allow-listed structural paths during export and expand them
through the destination's local mapping during restore. Ordinary prose is
preserved rather than rewritten merely because it looks like a path.

### Concurrent histories need session-aware conflict behavior

Two devices can both advance a session while offline. Git can represent
conflicting text changes, but a vendor-native session log is not ordinary
source code that a generic line merge can safely repair.

Reinstate uses revision metadata, dry runs, backups, active-session checks, and
conflict records to avoid silently replacing one history with another. The
user decides how to resolve divergence instead of receiving a syntactically
merged transcript with unknown resume semantics.

## The evidence in Reinstate's implementation

Reinstate's current repository separates these responsibilities explicitly:

| Concern | Repository evidence | What it establishes |
| --- | --- | --- |
| Vendor discovery and restore | [`internal/adapter`](https://github.com/HarjjotSinghh/reinstate/tree/main/internal/adapter) | Claude Code and Codex session formats are handled through distinct adapters. |
| Portable project identity | [`internal/pathmap`](https://github.com/HarjjotSinghh/reinstate/tree/main/internal/pathmap) | Known structural paths can be normalized and restored per device. |
| Client-side encryption | [`internal/crypto`](https://github.com/HarjjotSinghh/reinstate/tree/main/internal/crypto) | Remote snapshots are encrypted with a passphrase-derived age recipient. |
| Transfer and conflict state | [`internal/sync`](https://github.com/HarjjotSinghh/reinstate/tree/main/internal/sync) | Push, pull, manifests, revisions, backups, and conflicts belong to a sync protocol. |
| Product boundary | [`ROADMAP.md`](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md) | Git remains source truth; Reinstate is responsible for context continuity. |

The public `v0.1.0-rc.6` scope is deliberately narrower than the full roadmap:
same-vendor Claude Code and Codex session sync. Local universal search,
cross-agent portable handoffs, and universal agent configuration are later
phases, not hidden features of the current CLI.

## Git and Reinstate are complementary

The practical workflow uses both tools:

1. Use Git to commit and exchange source history.
2. Use Reinstate to encrypt and transfer the selected native coding-agent
   session.
3. Map the same canonical project ID to the repository's real path on each
   device.
4. Pull with a dry run, restore safely, and resume through the same vendor's
   native CLI.

Git answers, "Which code and commits belong to this project?" Reinstate answers,
"Which coding-agent context should be restored, and is this environment safe
and compatible enough to continue it?"

Reinstate does not replace Git, host a code editor, or execute the agent loop.
It is a continuity layer before and after coding-agent execution.

## When Git alone is enough

Git may be all you need when the important context is already captured in
commits, issues, design documents, and repository instructions, and you do not
need to resume a native coding-agent conversation.

Session sync becomes useful when the unfinished task depends on conversation
state that is expensive to reconstruct, when project paths differ across
devices, or when you want an encrypted transfer with explicit compatibility
and conflict checks.

For a current, command-by-command example, follow the
[Claude Code guide](/guides/sync-claude-code-sessions-across-devices) or the
[Codex guide](/guides/sync-codex-sessions-across-devices).
