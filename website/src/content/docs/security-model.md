---
title: "Reinstate security and encryption model"
description: "See how Reinstate encrypts session data before upload, excludes credential files, stores storage keys, handles conflicts, and defines its threat boundaries."
order: 4
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.1.0-rc.6"
updatedAt: 2026-07-27
tags: ["security", "encryption", "credentials", "threat-model", "age"]
targetQuery: "is Reinstate secure"
searchIntent: "security"
draft: false
noindex: false
---

Reinstate encrypts every remote manifest and session snapshot locally before
upload, hard-excludes known credential files, stores S3 credentials in the
native OS keyring, and never stores the encryption passphrase.

Coding-agent transcripts remain **high-sensitivity material** because they can
contain source code, architecture decisions, and secrets that a tool printed
to the terminal. Encryption reduces remote-storage risk; it does not make an
unsafe transcript harmless.

## Threat model (summary)

| Threat | Mitigation |
| ------ | ---------- |
| Cloud storage provider reads your sessions | Client-side encryption; provider only sees ciphertext |
| Network eavesdropper | TLS to backend + encrypted payloads |
| Accidental sync of API keys / OAuth | Hard denylist of credential paths (default on) |
| Overwriting good local history | Timestamped backups + conflict forks |
| Weak passphrase | age scrypt recipient + long-passphrase guidance; user responsibility |
| Compromised local machine | **Out of scope** (OS-level compromise) |
| Malicious release artifact | Checksums / supply-chain process (see SECURITY.md) |

## Trust boundaries

```
[Agent CLI] --files--> [Reinstate CLI] --ciphertext--> [Your bucket]
     ^                        |
     |                        +-- keys never leave machine
     +-- only process that "understands" sessions
```

- Reinstate does **not** call vendor agent APIs
- Reinstate does **not** require Anthropic/OpenAI/Google account credentials
- Remote backends never receive your passphrase or age identity in plaintext
  form beyond ciphertext + opaque object keys

## Encryption

| Property | Default |
| -------- | ------- |
| Algorithm | age passphrase encryption (`scrypt` recipient) |
| Key UX | Enter the same passphrase privately on each device; it is not stored |
| At rest (remote) | Ciphertext only |
| At rest (local secrets) | Passphrase is not stored; storage keys use the OS keyring |
| In transit | HTTPS/TLS to object storage |

### Passphrase guidance

- Prefer a long passphrase (diceware / password manager)
- Never commit passphrases to git or shell history
- Interactive commands read it from a hidden terminal prompt
- Automation must use `REINSTATE_PASSPHRASE_FD` pointing at a pre-opened
  descriptor; ordinary environment variables and CLI flags are rejected
- Losing the passphrase = losing ability to decrypt remote data (by design)

## What is never synced (defaults)

| Path / pattern | Reason |
| -------------- | ------ |
| `**/auth.json` | API keys / OAuth |
| `**/.credentials.json` | Tokens |
| Claude OAuth / keychain material | Credentials |
| Plugin `node_modules`, `.venv` | Huge, non-portable, regenerable |
| User-configured globs | Local policy |

Credentials and authentication files cannot be enabled for sync. Planned
universal configuration profiles may contain secret references, never secret
values.

## Future configuration reconciliation

Applying skills, plugins, hooks/loops, marketplaces, and MCP declarations adds
supply-chain and overwrite risk. Configuration adapters must manage known
fields only, preview and back up writes, report lossy mappings, pin executable
sources/versions, show permissions and commands, and require explicit consent.

Reinstate may track auth status and coordinate official login flows. Token reuse
is allowed only where the protocol, provider, or harness explicitly supports
it; copying private credential stores is never the fallback.

## Secrets inside transcripts

Agents sometimes echo `.env` values or tokens into session logs. Reinstate:

1. Encrypts everything it does sync (reduces blast radius of cloud leaks)
2. Syncs explicitly discovered Claude Code or Codex session artifacts in Phase 1
3. Plans optional transcript-redaction tooling for a later phase

**You remain responsible** for not pasting production secrets into agent chats.

## Restore safety

1. `--dry-run` available on pull
2. Existing files backed up under `~/.reinstate/backups/<timestamp>/`
3. Writes via temp file + atomic rename
4. Conflicts create `.conflict` forks — never silent last-writer-wins without notice
5. A mutating pull refuses to replace a session while the matching agent is active

## Reporting issues

Follow [SECURITY.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SECURITY.md). Private disclosure only for vulnerabilities.

## Independent review

We welcome security review PRs and responsible disclosure. A formal audit is not
claimed for pre-1.0 software — treat early versions accordingly.

## Non-affiliation

Reinstate is independent of Anthropic, OpenAI, Google, xAI, and other agent
vendors. Using those tools' local files does not imply partnership.
