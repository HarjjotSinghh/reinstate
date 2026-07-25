# Security Model

Reinstate handles **high-sensitivity material**: coding-agent transcripts can
contain source code, architecture decisions, and occasionally secrets that tools
printed to the terminal. This document is the contract we design against.

## Threat model (summary)

| Threat | Mitigation |
| ------ | ---------- |
| Cloud storage provider reads your sessions | Client-side encryption; provider only sees ciphertext |
| Network eavesdropper | TLS to backend + encrypted payloads |
| Accidental sync of API keys / OAuth | Hard denylist of credential paths (default on) |
| Overwriting good local history | Timestamped backups + conflict forks |
| Weak passphrase | Documented guidance; Argon2 KDF; user responsibility |
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
| Algorithm | age (X25519 / scrypt or Argon2-derived as configured) |
| Key UX | Passphrase — same phrase on every device derives the same key |
| At rest (remote) | Ciphertext only |
| At rest (local keys) | Restricted file permissions (`0600`) |
| In transit | HTTPS/TLS to object storage |

### Passphrase guidance

- Prefer a long passphrase (diceware / password manager)
- Never commit passphrases to git or shell history
- Losing the passphrase = losing ability to decrypt remote data (by design)

## What is never synced (defaults)

| Path / pattern | Reason |
| -------------- | ------ |
| `**/auth.json` | API keys / OAuth |
| `**/.credentials.json` | Tokens |
| Claude OAuth / keychain material | Credentials |
| Plugin `node_modules`, `.venv` | Huge, non-portable, regenerable |
| User-configured globs | Local policy |

Overrides that *enable* syncing credentials must be explicit and warn loudly.

## Secrets inside transcripts

Agents sometimes echo `.env` values or tokens into session logs. Reinstate:

1. Encrypts everything it does sync (reduces blast radius of cloud leaks)
2. Offers opt-in redaction patterns (Phase 2+) for high-entropy strings
3. Encourages `--scope sessions` awareness — you choose what leaves the machine

**You remain responsible** for not pasting production secrets into agent chats.

## Restore safety

1. `--dry-run` available on pull
2. Existing files backed up under `~/.reinstate/backups/<timestamp>/`
3. Writes via temp file + atomic rename
4. Conflicts create `.conflict` forks — never silent last-writer-wins without notice

## Reporting issues

Follow [SECURITY.md](https://github.com/HarjjotSinghh/reinstate/blob/main/SECURITY.md). Private disclosure only for vulnerabilities.

## Independent review

We welcome security review PRs and responsible disclosure. A formal audit is not
claimed for pre-1.0 software — treat early versions accordingly.

## Non-affiliation

Reinstate is independent of Anthropic, OpenAI, Google, xAI, and other agent
vendors. Using those tools' local files does not imply partnership.
