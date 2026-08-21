---
title: "Reinstate CLI command reference"
navTitle: "CLI reference"
description: "Use every current Reinstate command with its purpose, parameters, expected evidence, platform notes, failure modes, and safe recovery path."
order: 14
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.5.0-rc.6"
updatedAt: 2026-08-16
tags: ["cli", "command-reference", "session-sync", "troubleshooting", "handoff"]
targetQuery: "Reinstate CLI commands"
searchIntent: "navigational"
draft: false
noindex: false
---

The `rein` and `reinstate` names run the same binary. This reference covers
every command shipped by Reinstate `v0.4.0`, including what it does, what
success looks like, the flags it accepts, platform-specific behavior, common
failures, and the available recovery path.

> **Current scope:** Reinstate supports configless local discovery for Claude
> Code, Codex, Gemini CLI, and OpenCode; native resume/fork remains same-vendor
> and mutation-capable only for Claude Code and Codex. Structured handoff
> continues the same task in a *new* Claude Code or Codex session. MCP servers,
> skills, plugins, marketplaces, and universal configuration remain roadmap
> work.

## Prerequisites

- Install and verify `v0.4.0` before relying on this syntax.
- Run `rein init` before commands that need configuration or remote storage.
- Close the selected Claude Code or Codex process before a mutating pull or
  `conflicts resolve --keep-remote`.
- Use synthetic sessions while learning the workflow. Never paste a real
  transcript, storage credential, or passphrase into an issue or agent chat.

## Global command rules

Use `rein` in daily work or replace it with `reinstate` in any example. The
result is identical.

```sh
rein --help
rein COMMAND --help
rein COMMAND --json
```

`--help` prints local syntax without changing state. The global `--json` flag
requests machine-readable output where the selected command supports it;
command-local `--json` flags are equivalent. JSON field names are an
integration surface, but Reinstate remains pre-1.0 and may change before v1.0.

Set `REINSTATE_HOME` to an absolute path only when you deliberately need an
isolated Reinstate home. macOS and WSL2 otherwise use `~/.reinstate`; native
Windows uses `%USERPROFILE%\.reinstate`. Native Windows and WSL2 must keep
separate homes and device identities.

Interactive encryption uses a hidden prompt. Non-interactive automation must
open a secret file or pipe and set `REINSTATE_PASSPHRASE_FD` to that descriptor
number. Reinstate rejects ordinary passphrase environment values and secret
CLI flags. The explicit non-interactive storage provider may read
`REINSTATE_S3_ACCESS_KEY_ID` and `REINSTATE_S3_SECRET_ACCESS_KEY`, but those
values must never be committed or logged.

## Exit codes

| Code | Meaning | Operator response |
| ---: | --- | --- |
| `0` | Success | Continue only after checking the command-specific evidence. |
| `1` | Unexpected runtime failure | Preserve redacted output and inspect logs; do not repeat a mutation blindly. |
| `2` | Usage or invalid arguments | Correct the documented flag, value, or positional argument. |
| `3` | Missing or invalid configuration | Run `rein setup check` and repair configuration before retrying. |
| `4` | Authentication or storage failure | Verify endpoint, bucket, credential scope, network, and passphrase privately. |
| `5` | Agent, layout, or platform compatibility failure | Stop writes and use a documented supported version or target. |
| `6` | Sync conflict | Inspect the conflict and select an explicit resolution strategy. |
| `7` | Safety refusal | Satisfy the reported safety condition; do not bypass it without understanding the state change. |

There is no general `rein undo` command. Read-only commands need no rollback;
`init --force` and mutating restores create local backups where documented;
remote snapshots are immutable and require an explicit later retention process
rather than silent deletion.

## Inspect the installed release

### `rein version`

**Purpose:** identify the installed Reinstate version and build metadata
without reading configuration, storage, or session content.

```sh
rein version
rein version --json
```

**Expected result:** human output contains the version string. JSON output
contains `version`, `commit`, and `date`; the public Reinstate installer must report
`0.4.0`.

**Parameters:** `--json` selects machine-readable output. The command accepts
no session, agent, storage, or path arguments.

**Platform differences:** syntax and fields are the same on macOS, native
Windows PowerShell, Linux, and WSL2. The reported artifact architecture must
still match the current environment.

**Failure modes:** a missing command indicates a PATH or installation problem.
An unexpected version indicates the wrong binary or an incomplete replacement.

**Undo or recovery:** this command is read-only. Reinstall the pinned release
through the checksum-verifying installer; do not rename an unrelated binary to
make the version check pass.

### `rein doctor`

**Purpose:** produce redacted diagnostics; `--self-test` additionally exercises
synthetic encryption, in-memory sync, adapter export/restore, and atomic local
writes without reading a real transcript or contacting configured storage.

```sh
rein doctor
rein doctor --self-test --json
```

**Expected result:** the report identifies the redacted Reinstate home,
configuration state, keyring and adapter checks, and a pass/fail summary. A
self-test reports a synthetic local round trip without exposing credentials.

**Parameters:** `--self-test` enables the synthetic round trip; `--json`
returns structured diagnostics.

**Platform differences:** keyring and agent locations are platform-native.
Native Windows, WSL2, and macOS may therefore report different adapter or
keyring evidence while using the same command syntax.

**Failure modes:** missing configuration, unavailable keyring, unsupported
agent layout, local filesystem failure, crypto failure, or synthetic
export/restore mismatch produces a failed check and a nonzero exit. This
command does not prove remote storage access; use `init`, `status`, or a scoped
dry-run for that evidence. Treat redaction failure as a security defect.

**Undo or recovery:** diagnostics are read-only apart from temporary,
cleaned-up local self-test files. Correct the named prerequisite and rerun;
never publish unredacted private paths or infrastructure identifiers.

### `rein setup check`

**Purpose:** run the read-only preflight that decides whether the current
device, configuration, storage, keyring, and installed agents are ready.

```sh
rein setup check
rein setup check --json
```

**Expected result:** every installed agent has an explicit `SUPPORTED`,
`UNTESTED`, `UNSUPPORTED`, or `NOT_INSTALLED` state. The summary cannot claim
success while an installed agent is blocked.

**Parameters:** `--json` emits machine-readable checks. No mutation flags
exist.

**Platform differences:** compatibility evidence is specific to each native
agent layout and operating-system target. WSL2 and native Windows count as
different environments.

**Failure modes:** missing config uses exit `3`; an installed untested or
unsupported layout uses exit `5`; storage and keyring failures retain their
documented nonzero category.

**Undo or recovery:** the command is read-only. Install a supported agent
version, repair the reported configuration, or stop the transfer. Do not
change a compatibility result merely to clear the gate.

## Find and continue local sessions

These commands refresh a private derived index at
`$REINSTATE_HOME/cache/session-index-v1.sqlite`. They do not require
`rein init`, storage credentials, an encryption passphrase, keyring access, or
a network backend.

### `rein sessions`

**Purpose:** list local session metadata across supported read adapters.

```sh
rein sessions
rein sessions --agent claude --json
```

**Expected result:** each row has a composite `agent:native-id` reference,
timestamp, capability flags, and bounded metadata. Use `--agent
claude|codex|gemini|opencode|grok|kimi|qwen|pi|cursor|copilot|cline|all`; scripts should use `--json`.

### `rein search`

**Purpose:** find indexed sessions with literal, case-insensitive AND terms.

```sh
rein search "webhook retry" --agent codex --limit 20
rein search SESSION_ID --json
```

**Expected result:** only matching metadata rows appear. Optional `--project`,
`--branch`, and `--file` filters narrow the result. Search never prints the
matching transcript passage.

### `rein inspect`

**Purpose:** inspect one exact composite reference without dumping a transcript.

```sh
rein inspect claude:SESSION_ID
rein inspect opencode:SESSION_ID --json
```

**Expected result:** identity, workspace/project metadata, counts,
capabilities, source fingerprint, and at most a 160-code-point user preview.

### `rein last`

**Purpose:** select and launch the newest resumable Claude Code or Codex session.

```sh
rein last --agent claude --dry-run
rein last --project my-project
```

**Expected result:** `--dry-run` prints the exact executable, argument array,
and working directory without launching a vendor. Remove it only after review.

### `rein resume` and `rein fork`

**Purpose:** launch a same-vendor native continuation or native fork.

```sh
rein resume codex:SESSION_ID --dry-run
rein fork claude:SESSION_ID --dry-run
```

**Expected result:** the dry-run plan uses the recorded workspace and exact
vendor-native ID. Claude Code and Codex may launch after review. Gemini CLI and
OpenCode are read-only and refuse either action with compatibility exit `5`.
JSON without `--dry-run` is refused because native child output is not a JSON
response owned by Reinstate.

### Bare `rein` / `reinstate`

**Purpose:** open the numbered session switcher on an interactive TTY.

```sh
rein
```

Filter with `/text`, inspect with `i NUMBER`, resume with `NUMBER`, fork with
`f NUMBER`, and quit with `q`. A non-TTY bare invocation exits promptly with a
`rein sessions --json` hint.

## Configure a device

### `rein init`

**Purpose:** create one device identity, storage profile, credential reference,
project path map, `config.toml`, and `state.json` after probing storage.

```sh
rein init \
  --project github.com/acme/app=/absolute/path/to/app
```

To join an existing profile:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project github.com/acme/app=/different/absolute/path
```

**Expected result:** output reports `config.toml + state.json`, a UUID
`profile_id`, and a `credential_ref` without printing the credential.
Additional-device setup must read the existing encrypted remote manifest
before saving configuration.

**Parameters:** `--endpoint URL`, `--bucket NAME`, `--region REGION`,
`--prefix PREFIX`, repeatable `--project ID=/absolute/local/path`,
`--profile-id UUID`, `--yes`, and `--force`. `--yes` requires all storage
coordinates plus the documented environment credential provider. `--force`
backs up and replaces an initialized home; it is not a merge.

**Platform differences:** project values must use the current platform's
absolute path form, such as `/Users/me/app`, `C:\Users\me\app`, or
`/mnt/c/Users/me/app`. Store secrets in the native OS keyring during
interactive setup.

**Failure modes:** invalid or relative path maps use exit `2`; an existing home
without `--force` uses exit `7`; inaccessible storage, partial credentials,
keyring failure, or a missing established manifest uses exit `4`.

**Undo or recovery:** before first successful use, remove only the exact new
Reinstate home after preserving anything you need. For reinitialization, use
the timestamped backup printed by `--force`. Do not delete an established
remote manifest or invent a replacement profile ID.

## Discover and compare sessions

### `rein list`

**Purpose:** discover local vendor-native sessions through the selected
adapter, without uploading or restoring anything.

```sh
rein list --agent all
rein list --agent claude --json
rein list --agent codex --json
```

**Expected result:** output lists redacted session metadata, including the
agent and session identifier needed by scoped push or pull commands. An empty
list is valid only when no matching local session exists.

**Parameters:** `--agent claude|codex|all` defaults to `all`; `--json` emits
structured metadata.

**Platform differences:** adapters inspect each vendor's native
platform-specific session location. The same session may map to a different
local project path on another device.

**Failure modes:** invalid agent values use exit `2`; missing config, private
path errors, or unsupported layouts fail with their corresponding config or
compatibility exit rather than pretending the list is empty.

**Undo or recovery:** the command is read-only. Correct the agent filter or
compatibility issue and rerun. Never copy a session ID from a different vendor
into the selected adapter command.

### `rein status`

**Purpose:** decrypt and compare the remote manifest with local tracked state.

```sh
rein status
rein status --json
```

**Expected result:** output reports the remote manifest revision and
session-level local/remote relationship without exposing transcript content.

**Parameters:** `--json` selects structured output. Reinstate status has no agent or
session filter.

**Platform differences:** syntax is identical, but the hidden passphrase input
and local keyring are platform-native.

**Failure modes:** missing config, wrong passphrase, absent required remote
profile, inaccessible bucket, or tampered ciphertext fails closed. Status does
not create an empty replacement manifest.

**Undo or recovery:** status is read-only. Recheck profile coordinates and
enter the original passphrase privately. Do not reinitialize the device to
silence an unexpected remote state.

### `rein diff`

**Purpose:** show pending local-versus-remote metadata for all sessions or one
selected agent/session; it does not print transcript content.

```sh
rein diff --json
rein diff --agent claude --session SESSION_ID
```

**Expected result:** output describes pending change metadata for the selected
scope and provides the basis for a dry-run.

**Parameters:** optional `--agent AGENT`, optional `--session SESSION_ID`, and
`--json`. Pair an exact session ID with its owning adapter.

**Platform differences:** command syntax is shared; structural source paths
are compared through the configured canonical project mapping.

**Failure modes:** bad filters use exit `2`; missing configuration, wrong
passphrase, unavailable storage, or unsupported layout fails with a nonzero
exit.

**Undo or recovery:** diff is read-only. Repair the reported prerequisite, then
run the relevant push or pull with `--dry-run` before mutation.

## Transfer a selected session

### `rein push`

**Purpose:** export selected vendor-native session state, replace recognized
structural roots with canonical project tokens, encrypt locally, and upload an
immutable snapshot plus an encrypted manifest update.

```sh
rein push --agent claude --session SESSION_ID --dry-run
rein push --agent claude --session SESSION_ID
```

**Expected result:** dry-run prints `would push ... dry_run=true` and writes no
remote snapshot. A successful mutation reports `pushed`, a snapshot count, and
the unchanged-session skip count.

**Parameters:** `--agent AGENT`, either `--session SESSION_ID` or `--all`,
`--dry-run`, and `--json`. Prefer an exact session. `--all` is an explicit
bulk-selection decision, not a shortcut for uncertainty.

**Platform differences:** the adapter reads the current OS's native session
layout, while project mappings normalize only recognized structural paths.
Plain Linux is not a certified Phase 1 resume target.

**Failure modes:** missing or ambiguous selection, unsupported agent layout,
wrong passphrase, credential/storage failure, manifest race, or failed
encryption stops the operation. Credentials and auth files are hard-excluded.

**Undo or recovery:** dry-run needs no undo. Snapshots are immutable and Reinstate
has no general delete or undo command. If a push was unintended, stop further
sync, preserve the evidence, and apply the documented bucket-retention process
only after identifying the exact profile objects.

### `rein pull`

**Purpose:** download and decrypt selected remote session state, map canonical
project tokens to this device, validate the restore plan, back up an existing
target, and restore atomically.

```sh
rein pull --agent codex --session SESSION_ID --dry-run
rein pull --agent codex --session SESSION_ID
```

**Expected result:** dry-run reports `would pull ... dry_run=true` and does not
write agent state. A successful restore reports `pulled ... dry_run=false`;
the same vendor can then discover and resume the session.

**Parameters:** `--agent AGENT`, either `--session SESSION_ID` or `--all`,
`--dry-run`, and `--json`. The session must belong to the selected agent.

**Platform differences:** destination mappings expand canonical project IDs to
the current OS's configured absolute roots. Native Windows and WSL2 require
their own homes and mappings.

**Failure modes:** missing remote session, wrong passphrase, checksum or
identity mismatch, unmapped project, unsupported layout, active agent process,
or local divergence blocks mutation. Divergence records an explicit conflict
instead of silently overwriting.

**Undo or recovery:** dry-run needs no undo. For a completed replacement,
close the agent and restore the timestamped local backup only after reviewing
the current target. For a conflict, use the conflict commands below; do not
manually delete the record first.

## Inspect and resolve conflicts

### `rein conflicts list`

**Purpose:** list unresolved local conflict records for the configured home.

```sh
rein conflicts list
rein conflicts list --json
```

**Expected result:** output lists conflict IDs and redacted metadata, or an
explicit empty set when configuration is valid and no records exist.

**Parameters:** `--json` emits structured records.

**Platform differences:** none in syntax; recorded local paths and agent
locations remain device-specific private metadata.

**Failure modes:** missing or invalid configuration fails instead of appearing
as an empty conflict list.

**Undo or recovery:** listing is read-only. Use `conflicts show` before any
resolution.

### `rein conflicts show`

**Purpose:** inspect one recorded conflict and the local/remote revision
metadata needed for an informed resolution.

```sh
rein conflicts show CONFLICT_ID
```

**Expected result:** output identifies the conflict, agent, session, project,
and competing revisions without printing the full transcript.

**Parameters:** the required positional `CONFLICT_ID` comes from
`rein conflicts list`. The global `--json` request applies where supported.

**Platform differences:** none in syntax.

**Failure modes:** a missing positional ID or unknown record uses exit `2`;
invalid configuration retains the config failure rather than fabricating a
record.

**Undo or recovery:** show is read-only. Copy the exact ID and choose one
resolution strategy only after reviewing both sides.

### `rein conflicts resolve`

**Purpose:** resolve one conflict explicitly by keeping local state, replacing
with remote state, or restoring the remote state under a second session
identity.

```sh
rein conflicts resolve CONFLICT_ID --keep-local
rein conflicts resolve CONFLICT_ID --keep-remote
rein conflicts resolve CONFLICT_ID --keep-both
```

**Expected result:** the selected strategy completes, the resolution is
audited, and the conflict is no longer unresolved. `--keep-remote` verifies
the restored native session before completion.

**Parameters:** one required `CONFLICT_ID` and exactly one of `--keep-local`,
`--keep-remote`, or `--keep-both`.

**Platform differences:** remote restoration applies the destination device's
project mapping. Close the native agent process on every platform before
`--keep-remote`; `--keep-both` creates a separate local identity.

**Failure modes:** no strategy, multiple strategies, an unknown ID, changed
remote state, failed verification, or an active matching agent refuses the
operation.

**Undo or recovery:** there is no general conflict undo. Keep-remote
replacement uses the restore backup path; keep-both preserves the original;
keep-local updates remote continuity from the local side. Preserve the audit
record and backup before attempting any manual reversal.

## Generate shell completion

### `rein completion`

**Purpose:** print a completion script for one supported shell.

```sh
rein completion bash
rein completion zsh
rein completion fish
rein completion powershell
```

**Expected result:** stdout contains a shell script. Redirect or source it only
according to that shell's trusted completion-file conventions.

**Parameters:** exactly one positional shell name:
`bash|zsh|fish|powershell`.

**Platform differences:** choose the active shell, not merely the operating
system. PowerShell completion can be used on Windows; zsh is typical on
macOS; WSL2 follows its Linux shell.

**Failure modes:** a missing or unsupported shell name uses exit `2`. A
generated script can be correct while a shell startup path or execution policy
prevents it from loading.

**Undo or recovery:** generation alone is read-only. If you installed the
output, remove only the exact completion line or file you added, then start a
new shell.

## Structured handoff

### `rein handoff`

**Purpose:** continue the same task in a **new** Claude Code or Codex session.
This is not native resume.

```sh
rein handoff claude:SESSION_ID --to codex --dry-run --json
rein handoff --last --from claude --to codex
rein handoff list --json
rein resume claude:SESSION_ID --with codex --dry-run
```

**Expected result:** `--dry-run --json` reports `mode` `structured handoff`. A
live destination first-reply restates five acknowledgement bullets before
mutation.

**Parameters:** `--to claude|codex` is required. `--policy` is
`checkpoint|balanced|full` (default `balanced`). Repeat `--allow-warning ID`
for each current warning. `--json` requires `--dry-run` or `--no-launch`.

**Platform differences:** dest launch needs a real TTY. Windows dest argv uses
a one-line `projection.md` pointer when the briefing contains CR/LF.

**Failure modes:** exit `5` for wrong repo or untested dest; exit `7` for
unacknowledged warnings or non-TTY dest launch.

**Undo or recovery:** `--dry-run` writes nothing durable. `--no-launch` stores
local capsules only. Kill a dest TUI with the vendor quit command; do not
reuse a dest session id that is already indexed.

See [structured handoff](/docs/handoff) and [features](/docs/features).

## Expected evidence

A complete same-vendor transfer record contains:

- `rein version --json` showing `0.4.0`;
- a passing or truthfully blocked `rein setup check --json` on each device;
- the exact agent and `SESSION_ID` selected by `rein list`;
- successful push and pull dry-runs before each mutation;
- one encrypted remote snapshot and encrypted manifest revision without
  plaintext transcript objects;
- the restored session visible to the same vendor's native list and resume
  command; and
- redacted output that contains no passphrase, storage secret, auth token, or
  transcript text.

These checks prove the observed workflow only. They do not certify every
platform or close the project's physical two-device release gates.

## Failure paths

- Start with [troubleshooting](/docs/troubleshooting) for exact diagnostic,
  evidence, recovery, escalation, and verification contracts.
- Use [configuration troubleshooting](/docs/configuration) for profile,
  project-map, keyring, and additional-device errors.
- Use [storage troubleshooting](/docs/storage) for endpoint, bucket, policy,
  retention, or remote-manifest failures.
- Check [compatibility](/compatibility) before interpreting an untested agent
  layout as a transient sync problem.

Do not retry a mutating command repeatedly after an exit `4`, `5`, `6`, or `7`
without resolving the reported condition.

## Security boundaries

Reinstate encrypts supported session snapshots and manifests before upload,
but command output, shell history, private paths, storage coordinates, and
local backups still require careful handling. Enter credentials and
passphrases only through the documented private channels. Never weaken
checksum, compatibility, active-process, conflict, or backup checks to obtain
a green command.

The current CLI performs same-vendor native resume and explicit structured
handoff into a new destination session. It does not reconstruct vendor history,
mirror complete vendor configuration trees, sync credentials, operate agents,
or provide a Reinstate-owned plugin runtime.

## Related pages

- [Install the verified CLI](/docs/installation)
- [Configure profiles and project paths](/docs/configuration)
- [Push one selected session](/docs/sync-a-session)
- [Restore and resume one selected session](/docs/restore-a-session)
- [Continue a task with structured handoff](/docs/handoff)
- [Review current limitations](/docs/limitations)
