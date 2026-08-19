---
title: "Install and sync Reinstate across devices"
navTitle: "Getting started"
description: "Install Reinstate, configure encrypted S3-compatible storage, and restore a Claude Code or Codex session safely on another development machine."
order: 1
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.5.0-rc.2"
updatedAt: 2026-08-16
tags: ["installation", "session-sync", "claude-code", "codex", "s3"]
targetQuery: "how to sync coding-agent sessions across devices"
searchIntent: "how-to"
draft: false
noindex: false
---

Reinstate finds and continues local coding-agent sessions without setup, and
optionally synchronizes same-vendor Claude Code and Codex CLI sessions across
machines through client-side-encrypted, user-owned S3-compatible storage.

> **Release status:** the public installers pin candidate `v0.5.0-rc.2`.
> Dual-platform tagged-artifact acceptance is pending. Stable remains `v0.4.0`.
> Intel macOS and Linux/WSL2 are optional and unverified.

## Prerequisites

- Apple Silicon macOS or native 64-bit Windows for mandatory Phase 3 acceptance
- Intel macOS, Linux, or WSL2 only as preview, unverified environments; WSL1 is refused
- Claude Code and/or Codex CLI
- an S3-compatible bucket you control, such as Cloudflare R2
- its endpoint, bucket name, access-key ID, and secret access key
- one long encryption passphrase you can enter privately on every device

Reinstate never needs your Anthropic or OpenAI account credentials.

## Install

macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

Native Windows PowerShell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

Both bootstraps pin `v0.4.0`, verify the exact tagged canonical installer
and release binary, install without elevation, configure a user-local PATH, and
print the next command. They install the CLI only; interactive configuration
begins when you run `rein init`.

The POSIX bootstrap can install the binary on Intel macOS, Linux, or WSL2, but
those environments are optional and unverified for v0.4.0.

Default locations:

- macOS/Linux: `~/.local/bin`
- Windows: `%LOCALAPPDATA%\Programs\Reinstate\bin`

Verify:

```sh
rein version --json
rein setup check
```

Local discovery needs no initialization or cloud account:

```sh
rein sessions
rein search "webhook retry"
rein inspect claude:SESSION_ID
rein resume claude:SESSION_ID --dry-run
rein handoff claude:SESSION_ID --to codex --dry-run --json
```

These commands use a private derived local index. Native resume and fork stay
same-vendor; Gemini CLI and OpenCode records are read-only for native launch.
Structured handoff can start a new Claude Code or Codex session from those
sources.

Before initialization, `setup check` should report only the missing Reinstate
config. Platform, keyring, or agent-compatibility failures are blockers.

## Inspect first

macOS/Linux:

```sh
curl -fsSL https://reinstate.dev/install.sh -o reinstate-install.sh
less reinstate-install.sh
sh reinstate-install.sh
```

Windows PowerShell:

```powershell
$Installer = Join-Path $env:TEMP "reinstate-install.ps1"
Invoke-WebRequest https://reinstate.dev/install.ps1 -OutFile $Installer
Get-Content $Installer
& ([ScriptBlock]::Create([IO.File]::ReadAllText($Installer)))
```

## First device

```sh
rein init \
  --project local/my-project=/absolute/path/to/my-project
```

Enter the S3/R2 service endpoint, bucket, and credentials privately. Do not add
the bucket name to the endpoint URL. Credential input is hidden and stored in
the native OS keyring. Reinstate probes storage before writing local
configuration.

Save the printed non-secret `profile_id`, then run:

```sh
rein setup check
rein doctor --self-test
rein list --agent all
```

Create a harmless session and select its ID:

```sh
rein list --agent claude
rein push --agent claude --session SESSION_ID --dry-run
rein push --agent claude --session SESSION_ID
```

Use `--all` only after you explicitly decide to sync every discovered session.

## Additional device

Use the same profile ID and canonical project ID, mapped to this device's local
path:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, credentials, and encryption passphrase. Reinstate
requires the existing encrypted remote manifest to be readable before it saves
the additional device's configuration.

Validate without mutation:

```sh
rein setup check
rein doctor --self-test
rein status

# Claude Code
rein pull --agent claude --session SESSION_ID --dry-run

# Codex
rein pull --agent codex --session SESSION_ID --dry-run
```

Close the selected agent before replacing an existing local session, then run:

```sh
# Claude Code
rein pull --agent claude --session SESSION_ID
rein list --agent claude
claude --resume SESSION_ID

# Codex
rein pull --agent codex --session SESSION_ID
rein list --agent codex
codex resume SESSION_ID
```

## Agent-assisted setup

Copy the pinned setup prompt for your agent:

- [Claude Code prompt](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/prompts/claude-code-setup.md)
- [Codex prompt](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/prompts/codex-setup.md)

The agent can inspect, install, and run redacted checks. You enter storage
credentials and the encryption passphrase privately, never in chat.

## Prove Phase 1

Run the complete
[MacBook + Windows acceptance checklist](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/testing/phase-1-mac-windows-acceptance.md)
to evaluate both agents, both directions, backups, conflicts, wrong-passphrase
refusal, and ciphertext-only storage. Passing the committed checklist is a
release gate; this page does not claim that the gate has passed.

## Safety

- remote manifests and snapshots are encrypted;
- auth and credential files are hard-excluded;
- pulls validate before mutation and back up existing targets;
- divergence produces conflict records instead of silent overwrite;
- mutating pulls refuse to replace a session while the matching agent is
  active; and
- passphrases are accepted through hidden input or an explicit pre-opened file
  descriptor, never a normal CLI argument or environment value.
