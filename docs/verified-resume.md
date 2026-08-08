# Verified resume

Phase 3 adds an environment preflight to every same-vendor native continuation.
Before Reinstate starts Claude Code or Codex, it reports the environment it can
actually observe, compares only facts that have trustworthy recorded
provenance, and refuses a silent bad continuation.

This document is the implementation contract for `v0.3.0`. It deliberately
does not claim that existing vendor session files contain a complete historical
environment snapshot. Portable checkpoints and cross-agent handoffs remain
Phase 4 work.

## Scope

Verified resume covers:

- the recorded workspace and its current availability;
- current repository identity, branch, HEAD, upstream relation, and working
  tree state;
- the installed same-vendor executable, version, and recognized session layout;
- privacy-safe presence metadata for instruction files, skills, and MCP
  servers;
- deterministic project runtime declarations and locally installed runtime
  versions where Reinstate has a verified probe; and
- one consistent launch decision for `inspect`, `resume`, `fork`, `last`, and
  picker selections.

Verified resume does not:

- translate or replay a session through a different vendor;
- fetch, clone, checkout, reset, install, repair, or edit the workspace;
- install or configure an agent, MCP server, skill, instruction file, or
  runtime;
- read instruction or skill contents for output;
- emit MCP commands, arguments, environment variables, or secret values;
- reconstruct a historical working tree or runtime state that the vendor did
  not record; or
- turn Reinstate into an agent runtime, terminal, editor, or scheduler.

## Truth and provenance

Every expected value is accompanied by provenance. The supported provenance
classes are:

| Provenance | Meaning |
| ---------- | ------- |
| `vendor_recorded` | A recognized structured field in the vendor session source recorded the fact. |
| `reinstate_prelaunch_observed` | Reinstate saved the live fingerprint immediately before a previously authorized, successful native launch. It is a comparison baseline, not session-start truth. |
| `reinstate_checkpoint` | A future explicit portable checkpoint recorded the fact at a named time. Phase 3 reserves this value but does not manufacture it. |
| `current_observation` | Reinstate observed the value during this preflight. It is not historical evidence. |
| `unavailable` | No trustworthy baseline exists. |

The current session layouts reliably provide a workspace and, when present, a
recorded branch. They do not reliably provide the repository identity, Git
HEAD, dirty-tree state, MCP set, skill set, instruction-file set, or runtime
versions from the beginning of the session.

Reinstate must never take the environment seen during a later index refresh or
first inspection and relabel it as the session-start environment. When no
baseline exists, the report says `unknown`; it never says `match`.

An existing session initially reports `baseline.unavailable`. After a verified
native launch completes successfully, Reinstate may save the immutable
prelaunch observation with `reinstate_prelaunch_observed` provenance. The next
launch can compare repository identity, HEAD, and a privacy-safe working-tree
digest with that observation. A failed, declined, cancelled, or blocked launch
does not establish or advance a baseline. This baseline remains private,
derived metadata and is never written into a vendor session file.

Recorded environment metadata is bounded, derived state. The private session
index may retain only recognized names, versions, digests, and provenance. It
must not retain configuration values, commands, arguments, environment
variables, instruction contents, skill contents, secrets, or transcript text.

The private state lives in
`$REINSTATE_HOME/cache/session-index-v2.sqlite`. Its owner-only `.lock` file
protects database lifetime and destructive corruption repair; its owner-only
`.write.lock` serializes ordinary writers and transactional rebuilds across
`rein` and `reinstate` processes. The database and both
locks are hard-excluded from sync. Session rows can be rediscovered from vendor
sources. After Reinstate is closed, moving the complete v2 database/lock family
aside preserves those moved bytes as recoverable evidence but removes its
comparison history from the active index. Deleting or explicitly rebuilding v2
removes the active comparison history. In every case the next launch truthfully
returns to `baseline.unavailable`.

On Unix, owner-only means mode `0700` for the cache directory and `0600` for
the database and locks. On Windows, Reinstate installs a protected DACL granting
full access only to the current user, LocalSystem, and Administrators; it does
not rely on a custom directory's inherited ACL or on Windows `chmod` behavior.

## Report schema

The environment report is additive to the Phase 2 inspect and dry-run JSON
contracts:

```json
{
  "schema_version": 1,
  "session_ref": "claude:SESSION_ID",
  "decision": "confirmation_required",
  "checks": [
    {
      "id": "git.branch",
      "status": "changed",
      "severity": "warning",
      "expected": "main",
      "actual": "feature/retry",
      "provenance": "vendor_recorded",
      "message": "the current branch differs from the branch recorded by the session",
      "repair": "switch to the recorded branch or explicitly continue without it"
    }
  ]
}
```

Checks are emitted in a fixed deterministic order, with safety-critical groups
first and stable check IDs within groups. The report contains no generation
timestamp, so identical observations produce identical normalized JSON.

### Status

| Status | Meaning |
| ------ | ------- |
| `match` | A current value equals a trustworthy recorded value. |
| `present` | A current capability or artifact is present, but no historical comparison is claimed. |
| `changed` | A current value differs from a trustworthy recorded value. |
| `missing` | A required workspace, executable, capability, or artifact is absent. |
| `unknown` | Reinstate cannot make a truthful comparison. |
| `error` | A bounded local probe failed unexpectedly. |

### Severity and decision

| Severity | Launch effect |
| -------- | ------------- |
| `info` | No launch effect. |
| `warning` | Requires a human confirmation or every exact warning ID to be acknowledged for that invocation. |
| `block` | Launch is refused and cannot be overridden. |

The aggregate decision is:

- `ready` when no warning or blocker exists;
- `confirmation_required` when at least one warning and no blocker exists; or
- `blocked` when at least one blocker exists.

## Workspace and Git contract

Git inspection is local-only and shell-free. Reinstate never fetches or
contacts a remote during preflight. Each subprocess is context-cancellable and
bounded by a short timeout.

Reinstate discovers the nearest valid physical `.git` marker without consulting
repository configuration, then pins both the Git directory and canonical
working-tree root for every probe. Repository identity reads only local config
with includes disabled. Working-tree inspection uses conversion-free plumbing;
it never invokes repository clean/process filters, external diffs, textconv,
fsmonitor commands, or recursive submodule status.

Repository-controlled includes, filters/attributes, `core.worktree`,
`core.ignoreStat`, submodules, hidden index flags, or a concurrent
branch/index/working-tree change make the working-tree observation `uncertain`.
An uncertain observation is always an explicit `git.working_tree` warning
requiring acknowledgment and never matches a previously recorded digest, even
when the visible counts and digest are equal.

The fingerprint reports:

- whether the recorded workspace exists and is a directory;
- the canonical repository root and a credential-free repository identity;
- the current symbolic branch or detached/unborn state;
- the current full Git HEAD when available;
- clean or modified working-tree state and bounded counts, never dirty
  filenames; and
- ahead/behind/diverged relation only when the comparison base is trustworthy
  and locally available.

Repository remotes are sanitized before normalization: user information,
passwords, query strings, and fragments are discarded. Human output uses a
safe host/path label. JSON may include an opaque digest for stable comparison,
but never a credential-bearing URL.

Policy:

| Condition | Result |
| --------- | ------ |
| Recorded workspace missing or not a directory | Block, compatibility exit `5` |
| Native executable missing | Block, compatibility exit `5` |
| Agent layout/version not in the verified range | Block, compatibility exit `5` |
| Different known repository identity | Block, safety exit `7` |
| Recorded branch differs from current branch | Warning |
| Trustworthy recorded HEAD differs from current HEAD | Warning with local ahead/behind/diverged detail |
| Working tree modified with no baseline, or digest/state changed from baseline | Warning; an unchanged previously observed dirty digest may match |
| Historical repository/HEAD/tree baseline unavailable | Warning `baseline.unavailable`; comparison remains `unknown` |
| Probe infrastructure failure | Block, runtime exit `1` |

When more than one blocker is observed, deterministic exit precedence is
runtime infrastructure `1`, then known safety mismatch `7`, then compatibility
`5`. The complete report still includes every check.

Detached HEAD, unborn repositories, linked worktrees, symlinked workspaces,
non-Git workspaces, missing Git, and locally unavailable comparison commits are
reported explicitly rather than collapsed into a mismatch.

## Agent and capability contract

Executable presence, recognized session layout, installed version, and native
action support are separate facts. Phase 1 export/restore compatibility does
not by itself authorize a Phase 3 native launch.

Claude Code and Codex keep their existing fail-closed verified version ranges
until physical release-candidate evidence supports a wider range. A detected
but unverified version is a compatibility blocker, not an automatic warning.
Gemini CLI and OpenCode remain read-only; their native resume/fork attempts
continue to fail with compatibility exit `5` before any launch.

Instruction, skill, and MCP probes are observation-only. They enumerate only
bounded, sanitized names in recognized Claude Code and Codex locations. A
capability is `missing` only when a trustworthy recorded requirement names it;
otherwise absent historical requirements remain `unknown` and current
capabilities may be reported as `present`.

Symlinks that escape the project or recognized agent configuration root are
not followed. Parse failures and unsupported configuration shapes produce
bounded diagnostics without raw file contents.

## Runtime contract

Runtime comparison is limited to deterministic declarations Reinstate knows
how to interpret without executing project code. The first Phase 3 contract
covers recognized Node version files and supported `package.json` engine
forms, plus Go's `go` and `toolchain` directives. Unknown constraint syntax is
reported as `unknown`; it is never loosely guessed.

Runtime commands use an executable plus arguments, inherit no shell syntax,
have bounded output and duration, and do not run package-manager lifecycle
scripts. A declared/runtime mismatch is a warning unless the corresponding
agent executable cannot launch without that runtime, in which case executable
or compatibility checks block independently.

## CLI behavior

`rein inspect SESSION` always includes the environment report in human and JSON
output. A successfully generated blocked report is still a successful inspect;
automation reads `environment.decision`. Only failure to produce an honest
report makes inspect exit non-zero.

Native dry-runs preserve the existing top-level launch-plan keys and add the
environment report and decision. They never prompt and never launch. Ready and
warning reports exit `0`. A blocked dry-run emits one report-bearing error and
exits with the applicable compatibility, safety, or runtime code.

Real `resume`, `fork`, `last`, and picker launches apply the same policy:

1. resolve exactly one same-vendor session;
2. build the deterministic native launch plan;
3. run one fresh environment preflight immediately before launch;
4. refuse blockers;
5. for warnings, prompt only on a real terminal;
6. in non-interactive use, refuse warnings unless every warning is acknowledged
   by a repeatable `--allow-environment-warning CHECK_ID`; and
7. launch the unchanged vendor executable/argv/cwd only after authorization.

After authorization Reinstate refreshes and resolves the exact qualified
session again, rebuilds the native plan, repeats the complete preflight, and
requires the plan and report to be identical before execution. A source,
workspace, repository, executable, plan, or report change invalidates the
authorization and returns safety exit `7` without launching.

The real native runner then invokes the same refresh, plan, report, and exact
acknowledgement guard once more after checking the privately bound absolute
executable and workspace directory. After that guard returns, it rechecks the
platform-native filesystem identities of both targets plus executable metadata
immediately before creating the child process. This rejects replacements made
during the guard; it is a strongest-practical fail-closed check, not a claim
that process creation and the host filesystem are one atomic operation. Private
executable and workspace paths or identities are never serialized in the
environment report.

`--allow-environment-warning` accepts only a warning ID present in the freshly
computed report. It is invocation-scoped, has no wildcard, and is not persisted.
On a warning-only report, unknown, stale, duplicated, or informational IDs are
rejected. A blocked report takes its blocker exit before acknowledgements are
considered. The flag cannot bypass a missing workspace or executable, an
unverified agent layout/version, a known repository identity mismatch, or a
verifier failure. Broad `--force`,
`--continue-without`, `--no-check`, and environment-variable bypasses are
intentionally not provided.

Declining a prompt, EOF at a required prompt, or warning-level refusal in a
non-terminal returns safety exit `7`. Child exit behavior remains the existing
runtime contract.

## Performance budget

Verification is on the execution path, so it must stay local and bounded:

- no verification work during ordinary `sessions` or `search`;
- one preflight for `inspect` or dry-run; a real launch displays/authorizes one
  report, then repeats the bounded preflight after the selected-source refresh
  and requires exact equality immediately before execution;
- no recursive project-tree walk;
- fixed known-path checks for instructions, skills, and runtime declarations;
- bounded configuration parsing and name counts;
- no network; and
- a default two-second total verifier deadline, with each child command using
  the remaining context.

The release-candidate matrix records cold and warm wall-clock samples on Apple
Silicon macOS and native Windows. For installed-artifact inspect or dry-run in a
normal warm repository, the initial target is p95 below one second on Apple
Silicon and two seconds on native Windows; the release-blocking ceilings are
two and four seconds respectively. Large-corpus and cold-refresh samples use
separate documented ceilings. A timeout, an observed 20–30 second regression,
or unbounded growth is release-blocking; a timeout never silently skips a
check.

## Acceptance gate

`v0.3.0-rc.5` may be published only after:

- deterministic unit, integration, adversarial, privacy, timeout, and
  concurrency tests pass;
- Phase 1 and Phase 2 regression suites remain green;
- `make verify`, the complete race suite, vulnerability scan, and required
  cross-builds pass;
- source, snapshot, installer, archive, checksum, SBOM, and provenance checks
  pass;
- independent architecture, security/privacy, and test reviews have no open
  release blocker; and
- the reviewed implementation is merged through protected `main` before the
  signed candidate tag is created.

The release candidate is not stable certification. Installed tagged artifacts
must subsequently pass the complete Phase 3 matrix on Apple Silicon macOS and
native Windows x64 before any stable `v0.3.0` decision.
