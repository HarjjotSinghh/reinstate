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
| Imported transcript injects destination policy | Source-attributed inert history; never promote source system/developer messages |
| Historical tool call executes again | Tool calls/results are evidence only; destination re-authorizes every action |
| Cross-agent handoff hides lost state | Component-level fidelity report and explicit omissions before launch |
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
| Key UX | Passphrase — same phrase on every device derives the same key |
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

Credential and authentication files cannot be enabled for sync in Phase 1.
The same boundary applies to the planned universal configuration profile: it
may contain secret references, but raw API keys, OAuth tokens, cookies, and
vendor credential stores are not portable configuration.

## Future configuration reconciliation

Applying MCP servers, skills, hooks/loops, plugins, marketplaces, and settings
creates additional risks:

| Risk | Required mitigation |
| ---- | ------------------- |
| A plugin or skill executes untrusted code | Pin source/version, verify digest/signature where available, show permissions and commands, require consent |
| A config adapter overwrites unrelated settings | Manage known fields only, preview diff, back up, write atomically |
| One harness cannot represent a field | Report unsupported/lossy mapping; never silently discard it |
| Repeated OAuth prompts encourage unsafe token copying | Track auth state and launch supported login flows; reuse only when provider/protocol/harness explicitly supports it |
| A second device inherits trust unexpectedly | Device-local allow/deny policy and explicit approval for executable capabilities |

See [universal-configuration.md](universal-configuration.md). Encryption protects
portable desired state in remote storage; it does not make arbitrary plugin
sources or copied credentials safe.

## Future cross-agent continuation

Cross-agent handoff crosses a second trust boundary: content produced under one
harness's prompts, tools, permissions, and sandbox enters another harness.

| Risk | Required mitigation |
| ---- | ------------------- |
| Source system prompt conflicts with destination policy | Preserve for audit only; never install it as destination authority |
| Tool output contains prompt injection | Delimit and source-attribute imported history; treat it as untrusted evidence |
| Pending or historical action is replayed | Mark interrupted; never execute during import; request destination approval again |
| Transcript says a file/test changed when it did not | Current workspace and Git state outrank conversation claims |
| Opaque reasoning/signature is incompatible | Keep same-vendor only when officially supported; otherwise report omitted |
| Long transcript floods destination context | Explicit checkpoint/balanced/full policy, sidecar references, size/token preview |
| Native destination file/DB synthesis corrupts history | Experimental exact-version gate, new ID, backup, native resume validation |
| Handoff capsule leaks secrets | Credential excludes, redaction preview, private local mode, E2EE before remote sync |

Continuity capsules live outside the repository by default with `0600`
permissions. The raw source artifact is immutable; the canonical record and
destination projection have separate hashes so the user can audit exactly what
the destination received. Source and destination credentials, account state,
approvals, cookies, tokens, and keychain entries never enter the capsule.

See [cross-agent-continuation.md](cross-agent-continuation.md).

## Secrets inside transcripts

Agents sometimes echo `.env` values or tokens into session logs. Reinstate:

1. Encrypts everything it does sync (reduces blast radius of cloud leaks)
2. Offers opt-in redaction patterns (Phase 2+) for high-entropy strings
3. Syncs only explicitly discovered Claude Code/Codex session artifacts in Phase 1

Planned cross-agent handoff also shows a redaction/fidelity preview before a
capsule is written for a destination or encrypted remote store.

**You remain responsible** for not pasting production secrets into agent chats.

## Restore safety

1. `--dry-run` available on pull
2. Existing files backed up under `~/.reinstate/backups/<timestamp>/`
3. Writes via temp file + atomic rename
4. Conflicts create `.conflict` forks — never silent last-writer-wins without notice

## Reporting issues

Follow [SECURITY.md](../SECURITY.md). Private disclosure only for vulnerabilities.

## Independent review

We welcome security review PRs and responsible disclosure. A formal audit is not
claimed for pre-1.0 software — treat early versions accordingly.

## Non-affiliation

Reinstate is independent of Anthropic, OpenAI, Google, xAI, and other agent
vendors. Using those tools' local files does not imply partnership.
