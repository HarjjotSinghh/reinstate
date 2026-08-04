# CLI reference

Binary names: `rein` and `reinstate` (identical behavior).

Public installers pin stable `v0.2.0`, including the Phase 1 sync surface and
the Phase 2 local commands below. Exact signed artifacts are physically
verified on Apple Silicon macOS and native Windows x64. Intel macOS and
Linux/WSL2 packages are preview and unverified for this release.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| 0 | success |
| 1 | unexpected runtime failure |
| 2 | usage / invalid arguments |
| 3 | missing/invalid config |
| 4 | authentication or storage failure |
| 5 | agent/layout compatibility failure |
| 6 | sync conflict |
| 7 | safety refusal |

## Commands

```text
rein
rein version [--json]
rein doctor [--json] [--self-test]
rein setup check [--json]
rein sessions [--agent claude|codex|gemini|opencode|all] [--json]
rein search QUERY [QUERY...] [--agent ...] [--project FRAGMENT]
            [--branch FRAGMENT] [--file FRAGMENT] [--limit N] [--json]
rein inspect AGENT:SESSION_ID [--json]
rein last [--agent claude|codex] [--project FRAGMENT] [--dry-run] [--json]
rein resume AGENT:SESSION_ID [--dry-run] [--json]
rein fork AGENT:SESSION_ID [--dry-run] [--json]
rein init [--endpoint URL] [--bucket NAME] [--region auto] [--prefix ...]
          [--profile-id UUID] [--project ID=/absolute/local/path] [--yes]
rein list [--agent claude|codex|all] [--json]
rein status [--json]
rein diff [--json]
rein push [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein pull [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein conflicts list|show|resolve ...
rein completion bash|zsh|fish|powershell
```

## Phase 2 local commands

`sessions`, `search`, and `inspect` refresh and read a private derived index
without requiring `init`, config, storage credentials, a passphrase, keyring
access, or a backend. The index lives at:

```text
$REINSTATE_HOME/cache/session-index-v1.sqlite
```

It is owner-only, never synced, safe to rebuild, and contains bounded
user-authored prompt text plus known metadata/file fields. It excludes
assistant messages/reasoning, tool output, environment dumps, credentials, and
auth stores.

Canonical references use:

```text
<agent>:<native-session-id>
```

A bare native ID is accepted only when it resolves to one indexed session.
Ambiguous IDs fail with a request to use the composite reference. Result
ordering is deterministic: newest update first, then agent, then native ID.

Search is literal and case-insensitive. Multiple query terms are ANDed.
`sessions` and `search` identify metadata without printing transcript passages.
`inspect` may show a terminal-safe, whitespace-collapsed first-user-prompt
preview capped at 160 Unicode code points; Phase 2 has no full-transcript dump.

`resume`, `fork`, and `last` build a structured executable/argv/cwd plan and
delegate execution to the source vendor:

| Agent | Resume | Fork |
| ----- | ------ | ---- |
| Claude Code | `claude --resume ID` | `claude --resume ID --fork-session` |
| Codex | `codex resume ID` | `codex fork ID` |
| Gemini CLI | read-only | read-only |
| OpenCode | read-only | read-only |

Review the plan with `--dry-run --json`. A real launch inherits the terminal,
waits for the child, and propagates failure. JSON mode requires `--dry-run` for
`resume`, `fork`, and `last`, so native child output cannot corrupt the JSON
document. A missing executable/workspace fails before launch. Read-only agents
refuse resume/fork with compatibility exit `5`.

On a TTY, bare `rein` refreshes and opens the numbered switcher:

```text
/text       filter
i NUMBER    inspect
f NUMBER    fork
NUMBER      resume
q           cancel
```

On a non-TTY, bare `rein` exits promptly with usage code `2` and a
`rein sessions --json` hint.

`rein list` remains the Phase 1 compatibility command used by sync scripts.
`rein sessions` is the canonical config-independent local listing command.

## Phase 1 encrypted sync

Interactive encryption uses a hidden terminal prompt. Non-interactive
automation must open a secret file/pipe and set `REINSTATE_PASSPHRASE_FD` to
that descriptor number; `REINSTATE_PASSPHRASE` and secret CLI flags are not
accepted.

Interactive `init` stores storage credentials in the native OS keyring.
The explicit non-interactive fallback reads
`REINSTATE_S3_ACCESS_KEY_ID` / `REINSTATE_S3_SECRET_ACCESS_KEY` without
persisting them.
Override home with `REINSTATE_HOME` (absolute path).

`rein setup check` returns compatibility exit code `5` when an installed agent
is `UNTESTED` or `UNSUPPORTED`; its summary never says all checks passed while
that agent is blocked from push/pull. `rein conflicts list` and
`rein conflicts show` require a valid config, so a missing config cannot look
like an empty conflict set.

A mutating `pull` never waits on a human closing an agent. Liveness is scoped to
the exact session file being replaced, so unrelated agents running in other
projects are ignored, and if that one session really is in use the live file is
left untouched and the remote copy is restored beside it as a distinct session.

`conflicts resolve --keep-remote` still refuses while the target session is in
use, because `--keep-both` is the explicit way to preserve both branches there.

New-session restores, `--keep-both`, and `pull --dry-run` remain available.
`--allow-active-agents` skips the liveness check for one run.

See [Configuration](configuration.md) for `restore.active_agent_policy`
(`fork` by default, or `scoped`, `strict`, `off`).

## Planned universal configuration commands

The following is roadmap direction, **not current CLI syntax**:

```text
rein mcp add|list|remove …
rein skill install|list|remove …
rein loop install|list|remove …
rein plugin install|list|remove …
rein marketplace add|list|remove …
rein config import|list|diff|apply|status|sync …
rein auth status …
```

The design goal is one non-secret desired-state profile rendered by verified
adapters into each target harness. Exact names and flags require an RFC before
implementation. See
[universal-configuration.md](universal-configuration.md).
