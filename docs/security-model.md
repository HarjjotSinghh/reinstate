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
| Local index broadens plaintext exposure | Owner-only, bounded derived fields; no assistant/tool-output corpus; safe rebuild |
| Search/preview dumps sensitive history | Metadata-only results; bounded terminal-safe user-prompt preview |
| Session reference becomes shell injection | Executable + argv + cwd launch plan; no shell command string |
| Environment drift causes a wrong continuation | Local verified-resume report; exact warning consent; hard blockers fail closed |
| Environment inspection leaks config/secrets | Name/state/digest-only observations; bounded safe diagnostics; no network or project execution |
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

- Reinstate does **not** require vendor cloud APIs or account credentials for
  local indexing or encrypted sync. Phase 2 may invoke a documented local
  vendor listing command and launches supported native CLIs only after an
  explicit resume/fork action.
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

## Local continuity index

The local index is plaintext derived state on the user's own machine. Remote
E2E encryption does not apply to it. The table describes `v0.3.0-rc.6`;
stable `v0.2.0` uses a separate v1 database without baselines. Its
controls are:

| Property | Contract |
| -------- | -------- |
| Location | `$REINSTATE_HOME/cache/session-index-v2.sqlite` plus `.lock` and `.write.lock` coordination files |
| Permissions | Unix `0700`/`0600`; protected Windows DACL for current user, LocalSystem, and Administrators, independent of inherited custom-parent ACLs |
| Concurrency | Shared/exclusive `.lock` protects database lifetime/destructive repair; `.write.lock` serializes writers and transactional rebuilds |
| Sync | Hard-excluded; never uploaded or treated as a session |
| Recovery | Session rows rebuild from vendor sources; deleting v2 also loses private prelaunch baselines and must be explicit |
| Search text | Bounded user-authored prompts only |
| Metadata | Identity, timestamps, workspace/project, branch, known file refs, counts, capabilities |
| Excluded corpus | Assistant reasoning/messages, tool output, environment dumps, credentials, auth stores |

The index is not a backup and never becomes source truth. Deleting it does not
delete a vendor session. A successful source scan can remove stale derived
rows; a malformed individual session cannot erase healthy sources or cause
Reinstate to rewrite vendor files.

Default `sessions` and `search` output identifies metadata without printing
matching transcript passages. `inspect` may show one user-authored preview
after collapsing whitespace, stripping terminal/control characters, and
capping it at 160 Unicode code points. Phase 2 provides no full-transcript
dump.

Search is local, literal, and case-insensitive. It does not call an embedding,
semantic-search, analytics, or network service.

Native resume/fork uses a composite `agent:native-id` reference. Reinstate
resolves the reference, verifies the recorded workspace and executable, and
executes an argv array directly. The production runner binds and rechecks
platform-native executable/workspace identities immediately before process
creation; this rejects controlled swaps during the final guard without claiming
an atomic filesystem/process-start primitive that the host does not provide. It
never interpolates the session ID into a shell command. Gemini/OpenCode are
read-only and fail closed for launch.

## Verified-resume boundary (`v0.3.0-rc.6`)

Phase 3 is included in candidate `v0.3.0-rc.6`; tagged-artifact acceptance is
pending, and stable `v0.2.0` does not include it.
Before a Claude/Codex native continuation, Reinstate computes a deterministic
local report and applies one fail-closed policy to direct resume/fork, `last`,
and picker launch paths.

The verifier may read only bounded facts from the selected workspace and
recognized agent locations. Git commands are fixed argv calls, use no shell,
and never fetch. Runtime probes invoke only recognized version commands in a
sanitized environment and never run package-manager lifecycle scripts or
project code. Capability probes are fixed known-path reads; they do not execute
instructions, skills, MCP declarations, or native configuration.

The report and private baseline may contain:

- an opaque credential-free repository identity, branch, Git object ID, and
  working-tree state/count/digest (never dirty filenames or diffs);
- installed agent/runtime names and versions;
- sanitized instruction/skill/MCP logical names, scope/state, and coarse MCP
  transport; and
- explicit provenance for every comparison.

They exclude raw repository URLs, filesystem paths for capability entries,
instruction/skill contents, MCP commands/arguments/URLs/headers/environment
values, raw child output, raw environment variables, credentials, and
transcripts. Probe errors are converted to bounded static diagnostics.

An initial `baseline.unavailable` warning is safer than inventing session-start
truth. Current observation becomes a private
`reinstate_prelaunch_observed` baseline only after the authorized same-vendor
child exits successfully. Failed, declined, cancelled, blocked, or child-error
launches do not update it. Baselines are neither vendor-session content nor
synced state.

Warnings require explicit, invocation-scoped authorization: a TTY prompt
defaults to no; automation must provide every exact current
`--allow-environment-warning CHECK_ID`. Broad force/wildcard/persisted or
environment-variable bypasses do not exist. Known repository replacement,
stale selected-source metadata, missing workspace/executable, unverified
agent version/layout, and verifier failure are blockers and cannot be
acknowledged away.

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

## Secrets inside transcripts

Agents sometimes echo `.env` values or tokens into session logs. Reinstate:

1. Encrypts everything it does sync (reduces blast radius of cloud leaks)
2. Limits the Phase 2 derived index to bounded user-authored prompts and known
   metadata, excluding assistant and tool-output fields
3. Syncs only explicitly discovered Claude Code/Codex session artifacts in Phase 1

**You remain responsible** for not pasting production secrets into agent chats.
User-authored prompts can themselves contain secrets; the local index is not a
redaction or DLP product.

## Restore safety

1. `--dry-run` available on pull
2. Existing files backed up under `~/.reinstate/backups/<timestamp>/`
3. Writes via temp file + atomic rename
4. Conflicts create distinct vendor-safe session forks — never silent
   last-writer-wins without notice
5. A session actively in use is left untouched and the incoming snapshot is
   restored as a distinct idempotent fork

## Reporting issues

Follow [SECURITY.md](../SECURITY.md). Private disclosure only for vulnerabilities.

## Independent review

We welcome security review PRs and responsible disclosure. A formal audit is not
claimed for pre-1.0 software — treat early versions accordingly.

## Non-affiliation

Reinstate is independent of Anthropic, OpenAI, Google, xAI, and other agent
vendors. Using those tools' local files does not imply partnership.
