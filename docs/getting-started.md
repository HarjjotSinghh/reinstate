# Getting Started

Reinstate synchronizes Claude Code and Codex CLI sessions across your machines
through client-side encrypted, user-owned object storage.

> **Release status:** the public installers currently pin
> `v0.1.0-rc.4`. It is a release candidate until the native Mac/Windows
> [Phase 1 acceptance runbook](testing/phase-1-mac-windows-acceptance.md) passes.

## Prerequisites

- macOS, native 64-bit Windows, Linux, or WSL2
- Claude Code and/or Codex CLI
- an S3-compatible bucket you control
- the endpoint, bucket name, access-key ID, and secret access key
- one long encryption passphrase that you can enter privately on every device

Cloudflare R2 is the recommended Phase 1 backend. Reinstate does not need your
Anthropic or OpenAI account credentials.

## Install

### macOS, Linux, or WSL2

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

The default installation directory is `~/.local/bin`. The bootstrap prints an
absolute `rein init` command that works immediately and adds the directory to
the appropriate shell startup file for new terminals.

### Native Windows PowerShell

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

The default installation directory is
`%LOCALAPPDATA%\Programs\Reinstate\bin`. The bootstrap adds it to the user PATH
and the current PowerShell process.

Both public bootstraps:

1. pin `v0.1.0-rc.4`;
2. download the canonical installer from that exact signed Git tag;
3. verify the canonical installer SHA-256;
4. download only the matching GitHub Release asset and `checksums.txt`;
5. verify the binary checksum and reported version; and
6. preserve an existing different version until you approve replacement.

They install the CLI only. Interactive configuration starts when you run
`rein init`.

### Inspect before executing

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

## Verify the binary

```sh
rein version --json
rein setup check
```

Before `init`, `setup check` should identify only that the Reinstate config is
missing. Platform, keyring, or installed-agent compatibility failures need to
be resolved before synchronization.

## Configure the first device

Use the same canonical project ID on every device but map it to each device's
real absolute path:

```sh
rein init \
  --project local/my-project=/absolute/path/to/my-project
```

`init` prompts for the S3/R2 endpoint, bucket, and credentials. Enter the
service endpoint only; do not append the bucket name to the endpoint URL. The
bucket has its own prompt. Credential input is hidden and stored in the native
OS keyring. Reinstate probes storage before writing local configuration.

Published RC4 can silently overwrite an existing `config.toml` and
`state.json`; do not re-run `init` against an initialized RC4 home. Current
unreleased source refuses with safety exit code `7`. Its explicit
`rein init --force` path writes the previous config and state into one
timestamped directory under `backups/` before replacing them.

Save the printed `profile_id`. It is not secret, and every later device in the
same sync set must reuse it.

Then verify:

```sh
rein setup check
rein doctor --self-test
rein list --agent all
```

The encryption passphrase is not stored. You enter it through a hidden prompt
when `status`, `push`, or `pull` needs to decrypt remote state.

## Push one session

Create or resume a harmless session in your mapped project, then find its ID:

```sh
rein list --agent claude
rein list --agent codex
```

Use an explicit agent and session ID until you intentionally decide to sync
everything:

```sh
rein push --agent claude --session SESSION_ID --dry-run
rein push --agent claude --session SESSION_ID
```

`--all` is available, but neither Reinstate nor an AI setup agent should choose
it for you.

## Configure an additional device

Run the platform installer, then reuse the first device's profile UUID and the
same canonical project ID:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project local/my-project=/different/absolute/path
```

Enter the same endpoint, bucket, credentials, and encryption passphrase. Keep
the bucket name out of the endpoint URL. Published RC4 requires a post-init
`status` check to catch wrong coordinates. Current unreleased source verifies
that `init --profile-id` can find the existing encrypted `manifest.age` before
saving local configuration; a missing profile fails without initializing the
device.

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

Close the selected coding agent before a pull that will replace an existing
local session, then restore:

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

Reinstate never needs to print transcript contents to prove success.

## Agent-assisted setup

The copy-paste prompts are version-pinned and use the same public bootstrap:

- [Claude Code setup prompt](prompts/claude-code-setup.md)
- [Codex setup prompt](prompts/codex-setup.md)

The agent may inspect, install, and run redacted checks. You must enter storage
credentials and the encryption passphrase privately, never in agent chat.

## Phase 1 sign-off

Use the
[MacBook + Windows acceptance runbook](testing/phase-1-mac-windows-acceptance.md)
to test both installers, both agent prompts, bidirectional sync, backups,
conflicts, wrong-passphrase refusal, and ciphertext-only remote storage.

## Safety defaults

- remote manifests and snapshots are encrypted;
- auth and credential files are hard-excluded;
- pull validates before mutation and backs up existing targets;
- divergent sessions create conflict records instead of silent overwrite;
- a mutating pull refuses to replace an active agent's session; and
- no plaintext passphrase is accepted through a normal argument or environment
  variable.

## Next steps

- [CLI reference](cli-reference.md)
- [Configuration](configuration.md)
- [Backup and recovery](backup-and-recovery.md)
- [Security model](security-model.md)
- [Troubleshooting](troubleshooting.md)
