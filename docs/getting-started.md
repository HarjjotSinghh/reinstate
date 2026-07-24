# Getting Started

**Reinstate** syncs AI coding agent sessions (and later: MCP configs, skills,
settings) across your machines — encrypted, vendor-neutral, bring-your-own
storage.

> **Status:** repository and docs are live; the v0.1 CLI is under active
> development. Commands below describe the **target UX**. Follow
> [ROADMAP.md](../ROADMAP.md) for ship status.

## Prerequisites

- Two (or more) machines you develop on (e.g. Windows desktop + MacBook)
- At least one supported agent installed (Claude Code and/or Codex first)
- An object-storage backend **or** WebDAV endpoint you control
  - **Recommended:** Cloudflare R2 (10GB free tier is plenty for sessions)

## Install (target)

### Binary (recommended)

```bash
# macOS / Linux (install script will ship with first release)
curl -fsSL https://raw.githubusercontent.com/HarjjotSinghh/reinstate/main/scripts/install.sh | sh

# Or download from GitHub Releases
# https://github.com/HarjjotSinghh/reinstate/releases
```

### From source

```bash
git clone https://github.com/HarjjotSinghh/reinstate.git
cd reinstate
make build
sudo cp bin/reinstate /usr/local/bin/   # optional
reinstate version
```

### Go install

```bash
go install github.com/HarjjotSinghh/reinstate/cmd/reinstate@latest
```

## First device (desktop)

```bash
# Interactive wizard: storage backend, passphrase, sync scope, path map
reinstate init

# Push local agent sessions (encrypted)
reinstate push

# Optional: inspect remote vs local
reinstate status
```

### What `init` asks

1. **Backend** — R2 / S3 / GCS / S3-compatible / WebDAV
2. **Credentials** — stored locally with restricted permissions (never uploaded)
3. **Encryption** — passphrase (same passphrase = same key on every device)
4. **Scope** — `sessions` | `config` | `all` (start with `sessions`)
5. **Path map** — map project roots that differ across machines
   (`C:\Users\you\work` ↔ `/Users/you/Projects`)

## Second device (laptop)

```bash
reinstate init
# Use the SAME backend + SAME passphrase
# Wizard verifies the passphrase can decrypt remote ciphertext

reinstate pull --dry-run   # always preview first
reinstate pull             # creates a timestamped local backup if needed
```

Then resume in your agent as usual:

```bash
claude --resume
# or
codex resume
```

## The magic moment

```
Windows desktop (8h of sessions)  →  reinstate push
MacBook on the couch              →  reinstate pull
                                    →  claude --resume  # same thread
```

Path remapping rewrites embedded `cwd` / project slugs so resume finds the
session even when absolute paths differ across OSes.

## Daily workflow (Phase 2+)

```bash
# Manual
reinstate push
reinstate pull

# Automated (shell hooks — roadmap)
# pull on shell start, push on exit
reinstate hooks install
```

## Scopes

| Scope | What syncs |
| ----- | ---------- |
| `sessions` | Agent conversation / rollout files (default) |
| `config` | MCP servers, skills, agents, portable settings |
| `all` | Both |

```bash
reinstate push --scope sessions
reinstate push --scope config
reinstate push --scope all
```

## Safety defaults

- Encryption is **on** — no plaintext cloud copies
- Credential files (`auth.json`, tokens) are **never** synced
- Pull never silent-overwrites: conflicts fork; backups are created first
- First pull on a new device defaults to dry-run confirmation

## Next steps

- [Architecture](architecture.md)
- [Security model](security-model.md)
- [Supported adapters](adapters.md)
- [Troubleshooting](troubleshooting.md)
- [Comparison](comparison.md)

## Need help?

See [SUPPORT.md](../SUPPORT.md).
