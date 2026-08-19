# Getting Started

> **Cross-agent status:** the current release restores same-vendor sessions. A
> Claude Code → Codex quota-switch handoff is core Phase 4 roadmap work and
> creates a new linked destination session with explicit fidelity. See
> [Cross-agent continuation](cross-agent-continuation.md).

Reinstate synchronizes Claude Code and Codex CLI sessions through
client-side-encrypted object storage that you control.

> The public installers currently pin release candidate `v0.1.0-rc.5`.

## Prerequisites

- two development machines
- Claude Code and/or Codex CLI
- an S3-compatible bucket, such as Cloudflare R2
- one encryption passphrase you can enter privately on every device

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

Windows Package Manager (WinGet):

```powershell
winget install HarjjotSinghRana.Reinstate
```

WinGet installs the published Windows archive as a portable package, registers
both `rein` and `reinstate`, and needs no elevation. Use
`winget upgrade HarjjotSinghRana.Reinstate` to update. The manifest tracks
published stable releases and can lag a new tag while the community repository
validates the submission; use the PowerShell bootstrap for an exact version.

Both bootstraps pin `v0.1.0-rc.5`, verify the exact tagged canonical installer,
verify the downloaded release binary, install without elevation, configure a
user-local PATH, and print the next command. They do not launch interactive
setup from piped input.

Default locations:

- macOS/Linux: `~/.local/bin`
- Windows: `%LOCALAPPDATA%\Programs\Reinstate\bin`

Verify:

```sh
rein version --json
rein setup check
```

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

Enter the S3/R2 endpoint, bucket, and credentials privately. Save the printed
non-secret `profile_id`, then run:

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

Use the same profile ID and canonical project ID, mapped to the local path:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path

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
before calling the release stable. It covers both agents, both directions,
backups, conflicts, wrong-passphrase refusal, and ciphertext-only storage.

## Safety

- remote manifests and snapshots are encrypted;
- auth and credential files are hard-excluded;
- pulls validate before mutation and back up existing targets;
- divergence produces conflict records instead of silent overwrite; and
- passphrases are accepted only through hidden input.
