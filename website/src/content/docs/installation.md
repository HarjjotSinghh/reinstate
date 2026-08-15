---
title: "Install the Reinstate CLI"
navTitle: "Install the CLI"
description: "Install the pinned Reinstate release candidate on macOS, native Windows, or WSL2, verify its checksum and version, and diagnose PATH failures."
order: 9
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.4.0-rc.8"
updatedAt: 2026-08-04
tags: ["installation", "cli", "macos", "windows", "wsl2"]
targetQuery: "install Reinstate CLI"
searchIntent: "how-to"
draft: false
noindex: false
---

Install Reinstate with the official bootstrap for your operating system, then
require `rein version --json` to report `v0.4.0-rc.8`. The bootstrap verifies
the canonical tagged installer, release checksum, downloaded binary, and
reported version before replacing an existing installation.

> **Release status:** `v0.4.0-rc.8` is a pre-1.0 candidate whose tagged-artifact
> acceptance is pending on Apple Silicon macOS and native Windows x64. Stable
> remains `v0.3.0`; Intel macOS and Linux/WSL2 remain optional and unverified.

## Prerequisites

- Apple Silicon macOS or native Windows x64 for mandatory Phase 4 acceptance, or
  an optional unverified environment (Intel macOS, Linux, or WSL2). WSL1 is refused.
- A user account that can write to a user-local installation directory.
- HTTPS access to `reinstate.dev` and the project's GitHub Release assets.
- A new terminal after PATH changes.

Installation does not require an Anthropic account credential, OpenAI account
credential, S3 credential, or Reinstate passphrase. Those secrets belong to
later setup prompts, never the installer command or an agent chat.

## Install with Homebrew on Apple Silicon macOS

```sh
brew install HarjjotSinghh/tap/reinstate
```

The tap formula tracks stable `v0.3.0`. Intel macOS and Linuxbrew
remain optional and unverified for this release; use the bootstrap below if
you prefer the canonical installer path.

## Install with the official bootstrap

For macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

The default binary directory is `~/.local/bin`. The script installs
`reinstate` and its shorter `rein` alias without elevation, updates the
appropriate user shell startup file when needed, and prints an absolute next
command that works in the current shell.

For native Windows PowerShell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

The default directory is
`%LOCALAPPDATA%\Programs\Reinstate\bin`. The script installs
`reinstate.exe` and `rein.exe`, adds the directory to the user PATH and current
PowerShell process, and does not configure WSL.

Native Windows and WSL2 must use separate Reinstate homes and device
identities. Do not point WSL2 at `%USERPROFILE%\.reinstate` or share one native
agent-state directory between the two environments.

## Inspect the installer before execution

Download and read the script when your environment requires review before
execution.

macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh -o reinstate-install.sh
less reinstate-install.sh
sh reinstate-install.sh
```

Native Windows:

```powershell
$Installer = Join-Path $env:TEMP "reinstate-install.ps1"
Invoke-WebRequest https://reinstate.dev/install.ps1 -OutFile $Installer
Get-Content $Installer
& ([ScriptBlock]::Create([IO.File]::ReadAllText($Installer)))
```

Both bootstraps pin the exact Reinstate tag, retrieve the canonical installer from
that tag, verify its SHA-256, download the matching release asset and
`checksums.txt`, verify the asset, and execute the downloaded binary's version
check before installation.

## Replacing an existing version

An already installed different version is preserved until you explicitly
approve replacement. On POSIX, the replacement prompt has a 30-second default
timeout and refuses when the active shell cannot perform a timed TTY read. A
timeout, unsupported shell, or invalid timeout preserves the existing binary.

After reviewing the requested version change, deliberate POSIX automation may
set `REINSTATE_CONFIRM_REPLACE=1`. The timeout can be set from 1 through 300
seconds with `REINSTATE_CONFIRM_TIMEOUT_SECONDS`. These switches approve only
binary replacement; they do not approve configuration, credentials, or session
transfer.

## Verify the installed CLI

Open a new terminal and run:

```sh
rein version --json
reinstate version --json
rein setup check
```

Before `rein init`, `setup check` should identify missing Reinstate
configuration. An installed agent can also report `SUPPORTED`, `UNTESTED`,
`UNSUPPORTED`, or `NOT_INSTALLED`; resolve platform, keyring, or compatibility
failures before synchronization.

## Expected evidence

- Both binary names resolve without an absolute path and report the same
  `0.2.0` version.
- The binary architecture matches the current environment.
- The installer reports successful checksum and release-version checks.
- `rein setup check` runs as a read-only preflight and does not claim the
  uninitialized device is ready to sync.

Save command output only after redacting private paths. Binary verification is
not proof of stable support outside Apple Silicon macOS and native Windows x64.

## Failure paths

- If the shell cannot resolve `rein`, follow
  [binary and PATH troubleshooting](/docs/troubleshooting#why-is-the-rein-binary-not-found-after-installation).
- Stop after any missing checksum entry, SHA-256 mismatch, release version
  mismatch, unsupported architecture, or installer download failure.
- Treat `setup check` exit code `5` as a compatibility blocker, not a warning
  to bypass.
- Report a suspected malicious or substituted release through the private
  [security policy](https://github.com/HarjjotSinghh/reinstate/security/policy).

Do not solve an installer failure by downloading an unofficial mirror or
disabling checksum verification.

## Security boundaries

The installers establish release identity and binary integrity; they do not
make the pre-1.0 product formally audited. They do not request agent
credentials, storage credentials, or the session passphrase. Run installers
with ordinary user permissions, inspect them when policy requires, and keep
release output free of secrets.

An agent may help inspect commands and redacted results. Enter later storage
credentials and the encryption passphrase privately into Reinstate's hidden
prompts, never into Claude Code, Codex, shell history, or a public issue.

## Related pages

- [Configure Reinstate after installation](/docs/configuration)
- [Complete first-device setup](/docs/getting-started)
- [Check current platform and agent compatibility](/compatibility)
- [Review the security model](/docs/security-model)
