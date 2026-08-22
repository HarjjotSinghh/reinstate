# CLI reference

Binary names: `rein` and `reinstate` (identical behavior).

Public installers pin candidate `v0.5.1`, including the Phase 1/2 surface,
Phase 3 verified resume, the Phase 4 structured-handoff surface, and the
Phase 5 catalog/T1 index surface. Dual-platform tagged-artifact acceptance is
pending. Stable remains `v0.4.0`. Intel macOS and Linux/WSL2 remain optional
and unverified.

Stable `v0.3.0` includes the Phase 3 environment report and
`--allow-environment-warning` flag. The command synopsis below additionally
includes the `v0.4.0` structured-handoff surface.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| 0 | success |
| 1 | unexpected runtime failure |
| 2 | usage / invalid arguments |
| 3 | missing/invalid config |
| 4 | authentication or storage failure |
| 5 | agent/layout compatibility failure |
| 6 | ambiguous session reference or sync conflict |
| 7 | safety refusal |

## Commands

```text
rein
rein version [--json]
rein doctor [--json] [--self-test] [--agents] [--acceptance-matrix]
rein setup check [--json]
rein sessions [--agent claude|codex|gemini|opencode|grok|kimi|qwen|pi|cursor|copilot|cline|all] [--json]
rein search QUERY [QUERY...] [--agent ...] [--project FRAGMENT]
            [--branch FRAGMENT] [--file FRAGMENT] [--limit N] [--json]
rein inspect AGENT:SESSION_ID [--json]
rein last [--agent claude|codex|all] [--project FRAGMENT] [--dry-run] [--json]
          [--allow-environment-warning CHECK_ID ...]
rein resume AGENT:SESSION_ID [--dry-run] [--json] [--fork]
            [--with claude|codex]
            [--allow-environment-warning CHECK_ID ...]
rein fork AGENT:SESSION_ID [--dry-run] [--json]
          [--allow-environment-warning CHECK_ID ...]
rein handoff [AGENT:]SESSION_ID --to claude|codex
             [--policy checkpoint|balanced|full] [--dry-run|--no-launch]
             [--json] [--export PATH] [--allow-warning ID ...]
             [--allow-active] [--allow-untested] [--show-redactions]
             [--no-redact]
rein handoff --last [--from claude|codex|gemini|opencode|grok|qwen]
rein handoff --last [--from claude|codex|gemini|opencode|grok|kimi]
             --to claude|codex [handoff flags]
rein handoff list [--json] [--limit N]
rein handoff inspect HANDOFF_ID [--json]
             [--acknowledged|--not-acknowledged]
rein handoff export HANDOFF_ID --format json|markdown [--out PATH]
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

### `rein doctor`

`--json` emits machine-readable diagnostics. `--self-test` runs a synthetic
encryption/storage check (in-memory; it does not prove remote storage).
`--agents` lists every catalog agent and its support tier. `--agents --json`
emits the redacted `AGENT-PROBE-V1` artifact. `--agents --acceptance-matrix`
prints the generated Phase 5 acceptance row count and per-agent row list.

## Local commands

`sessions`, `search`, and `inspect` refresh and read a private derived index
without requiring `init`, config, storage credentials, a passphrase, keyring
access, or a backend. Stable `v0.2.0` used the Phase 2 v1 index. `v0.3.0`
deliberately moves to a separate path so an older binary cannot erase
new baseline metadata:

```text
$REINSTATE_HOME/cache/session-index-v2.sqlite
```

The database and its `.lock` and `.write.lock` files are owner-only and never
synced. The shared/exclusive `.lock` protects database lifetime and rebuild;
the `.write.lock` serializes ordinary writers across processes. Session rows
are safe to rebuild; successful prelaunch observations are private continuity
metadata and therefore live in the versioned v2 store rather than the Phase 2
v1 file. The database contains bounded
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

Review the plan with `--dry-run --json`. Stable `v0.3.0` also includes
the verified-resume `environment` report described below. A real launch
inherits the terminal, waits for the child, and propagates failure. JSON mode
requires `--dry-run` for `resume`, `fork`, and `last`, so native child output
cannot corrupt the JSON document. Read-only agents refuse resume/fork with
compatibility exit `5` before any environment probe or vendor launch.

On a TTY, bare `rein` refreshes and opens the numbered switcher:

```text
/text       filter
i NUMBER    inspect
f NUMBER    fork
h NUMBER    structured handoff; then choose claude or codex
NUMBER      resume
q           cancel
```

On a non-TTY, bare `rein` exits promptly with usage code `2` and a
`rein sessions --json` hint.

`rein list` remains the Phase 1 compatibility command used by sync scripts.
`rein sessions` is the canonical config-independent local listing command.

## Phase 4 structured handoff (`v0.4.0`)

A structured handoff continues the same task in a new Claude Code or Codex
session. It is not native resume: Reinstate does not reconstruct vendor history,
write a vendor-internal session file, or claim that the destination is the same
session. Source parsing and projection are local and require no source model
call. Gemini CLI, OpenCode, Grok Build, and Kimi Code CLI are source-only;
only Claude Code and Codex are destinations.

### `rein handoff`

`rein handoff [SESSION] --to AGENT` accepts:

| Flag | Contract |
| ---- | -------- |
| `--last` | Select the newest matching source instead of `SESSION`. |
| `--from AGENT` | Restrict `--last` to one source agent. |
| `--to AGENT` | Required destination: `claude` or `codex`. |
| `--policy checkpoint\|balanced\|full` | Projection policy; default `balanced`. |
| `--dry-run` | Preview using temporary files only; no durable handoff and no launch. |
| `--json` | Emit machine-readable, launch-free output; requires `--dry-run` or `--no-launch`. |
| `--no-launch` | Store the capsule and print the exact command without spawning the destination. |
| `--export PATH` | Also write the projection to `PATH`; incompatible with `--dry-run`. |
| `--allow-warning ID` | Acknowledge one exact current warning ID; repeat for each warning. |
| `--allow-active` | Freeze the last complete source record while its agent is active. |
| `--allow-untested` | Proceed with an untested source or destination layout. |
| `--show-redactions` | Show redaction categories and counts, never values. |
| `--no-redact` | Skip secret redaction. Refused with exit `2` for Grok sources. |

### `rein handoff list`

Accepts `--json` and `--limit N` (default `100`).

### `rein handoff inspect`

`rein handoff inspect HANDOFF_ID` accepts `--json`, `--acknowledged`, and
`--not-acknowledged`; the two acknowledgement flags are mutually exclusive.

### `rein handoff export`

`rein handoff export HANDOFF_ID` requires `--format json|markdown` and accepts
`--out PATH`; without `--out`, it writes to stdout.

### `rein resume --with` and `--fork`

`rein resume SESSION --with AGENT` is a convenience alias for
`rein handoff SESSION --to AGENT`. It accepts `--dry-run`, `--json`, and
repeatable `--allow-environment-warning ID`, translated to exact handoff
warning acknowledgements. With this alias, `--json` requires `--dry-run`;
`resume --with` has no `--no-launch` mode. The alias prints a one-line
structured-handoff notice. `rein resume SESSION --fork` instead invokes the
source agent's native fork path; `--with` and `--fork` are mutually exclusive.

### Handoff exit codes

Handoff exit codes use the shared table above: `2` for bad flags, unknown
agents, and invalid launch/JSON combinations; `3` for invalid local config; `5`
for an untested or unsupported source/destination layout; `6` for an ambiguous
session reference; `7` for unacknowledged warnings or safety refusal; and `1`
for runtime failure. A planned or completed handoff returns `0`. No Phase 4
handoff path uses authentication/storage code `4`.

## Phase 3 verified resume (`v0.3.0`)

Phase 3 is included in stable `v0.3.0`; tagged-artifact acceptance passed on
Apple Silicon macOS and native Windows x64. Stable `v0.2.0` does not include
it. Before any real Claude or Codex native continuation, Reinstate builds a
deterministic, local-only environment report.
The same report is exposed by `inspect` and native dry-runs and enforced by
`resume`, `fork`, `last`, and picker resume/fork.

The report covers:

- selected-source freshness and the recorded workspace;
- an offline repository fingerprint, branch, HEAD, and privacy-safe working
  tree state/digest;
- installed same-vendor executable, verified version, and recognized layout;
- bounded logical names/states for recognized instruction files, skills, and
  MCP declarations; and
- supported Node and Go runtime declarations and locally installed versions.

It does not fetch, install, repair, checkout, reset, run project scripts, or
contact a network service. It omits dirty filenames, raw remote URLs,
instruction/skill contents, MCP commands/arguments/URLs/headers/environment
values, credentials, and raw environment dumps.

`rein inspect SESSION` always emits the report. Human output prints an
`Environment decision` and deterministic check lines. JSON adds:

```json
{
  "environment": {
    "schema_version": 1,
    "session_ref": "claude:SESSION_ID",
    "decision": "confirmation_required",
    "checks": []
  }
}
```

Inspect does not prompt or launch. A successfully generated blocked report
still exits `0`; automation must read `environment.decision`. Failure to
produce an honest report exits `1`.

Native dry-runs preserve the launch-plan keys and add `environment`. They
never prompt or launch. Ready and warning-only reports exit `0`. A blocked
dry-run emits one report-bearing error and returns its applicable exit code.
A dry-run does not need warning acknowledgements because it cannot launch; if
warning flags are supplied, their IDs are validated, but a partial valid set is
not treated as launch authorization.

### Launch decisions and warning acknowledgement

| Decision | Real launch behavior |
| -------- | -------------------- |
| `ready` | Launches after a final identical refresh and preflight. |
| `confirmation_required` | On a TTY, prompts `yes`/`no` with default `no`; otherwise every warning must be acknowledged by exact ID. |
| `blocked` | Refuses without prompting; no flag can override it. |

For non-interactive use, repeat the flag for every warning in the fresh report:

```sh
rein resume claude:SESSION_ID \
  --allow-environment-warning baseline.unavailable \
  --allow-environment-warning git.branch
```

The flag is invocation-scoped and accepts only an exact current warning ID.
Empty, duplicate, wildcard, unknown, stale, and informational IDs are usage
errors (`2`). Supplying only some current warnings is a safety refusal (`7`).
A blocked report takes its applicable blocker exit before acknowledgements are
considered; naming a blocker can never authorize it. There is no `--force`,
wildcard, persisted approval, environment-variable bypass, or
`--continue-without` alias.

A real terminal prompt accepts exactly `yes` or `no`; empty input and `no`
decline. EOF also declines. Decline/refusal returns safety exit `7` without a
vendor launch.

Resuming a session that the owning agent already has open warns with
`agent.active`. Reinstate does not refuse it: the vendor CLI owns every write to
its own store, and a second window on one session is a legitimate thing to want.
The warning exists because it is more often an accident. A host that cannot
enumerate its own processes reports that it could not tell and still resumes,
rather than claiming the session is free on evidence it never gathered. A
structured handoff does not raise this warning; it enforces its own
`--allow-active` boundary against the same signal.

The first preflight for an existing session warns with
`baseline.unavailable`; inspection never turns the current workspace into
historical truth. After an authorized native child exits successfully,
Reinstate stores the immediately preceding observation with
`reinstate_prelaunch_observed` provenance. Failed, declined, cancelled, blocked,
or child-error launches do not establish or advance it. A subsequent preflight
can compare repository identity, branch, HEAD, working-tree digest,
capabilities, and recognized runtimes with that private baseline.

Known repository replacement and stale selected-source metadata are safety
blockers (`7`). A missing workspace/executable or unverified agent layout/version
is a compatibility blocker (`5`). Probe infrastructure failure is runtime
failure (`1`). When multiple blockers exist, deterministic precedence is
runtime `1`, safety `7`, then compatibility `5`; the report still includes all
checks.

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
