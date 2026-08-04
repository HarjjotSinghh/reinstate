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
| Imported transcript injects destination policy | Source-attributed inert history; source policy stays audit-only |
| Historical tool call runs again | Never replay tools; destination authorizes new actions |
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

## Future cross-agent continuation

Imported history is untrusted evidence. Source system/developer messages are
never promoted to destination authority, historical/pending tool calls never
execute, current workspace truth overrides transcript claims, and destination
permissions/MCP authentication are authorized again.

The continuity capsule has separate hashes for the immutable source, canonical
record, and exact destination projection. It is private (`0600`) and outside the
repository by default, with redaction/fidelity preview before destination or
remote writes. Credentials, approvals, hidden reasoning, account state, and
live processes never enter the capsule. Native destination reconstruction is
experimental and exact-version gated.

See [Cross-agent continuation](cross-agent-continuation.md).

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
