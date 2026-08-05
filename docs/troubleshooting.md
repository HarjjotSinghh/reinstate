# Troubleshooting

## Install / binary not found

```bash
which rein reinstate
echo "$PATH"
# rebuild from source
make build && ./bin/rein version
```

## `rein sessions` says sync config is missing

`rein sessions`, `search`, and `inspect` must not require sync configuration in
Phase 2 builds. Confirm that you are not running `v0.1.0`, which predates those
commands; stable `v0.2.0` includes them:

```bash
rein version --json
rein sessions --json
```

Do not run `rein init` merely to make local search work. If a Phase 2 build
requests a bucket, credential, passphrase, or keyring entry for a local command,
record the exact version and sanitized error and report it as a bug.

## Local sessions are missing

1. Run `rein sessions --json` and inspect only compatibility states and
   warnings, not unrelated session content.
2. Confirm the vendor is installed and has at least one normally persisted
   session.
3. Use a composite reference such as `claude:<id>` or `codex:<id>`.
4. Check [compatibility.md](compatibility.md) and the sanitized local-index
   warnings. Sync-adapter compatibility and local-index capability are
   intentionally reported separately.
5. Confirm `REINSTATE_HOME` is absolute and writable. `v0.3.0-rc.1`
   uses `$REINSTATE_HOME/cache/session-index-v2.sqlite`; stable `v0.2.0` uses
   the separate `session-index-v1.sqlite` database.

Phase 2 local discovery intentionally does not reuse configured Phase 1 project
mappings, so an unmapped local project should still appear. Claude subagent
artifacts are excluded from the top-level resumable list.

## Local index is stale or corrupt

Every local command refreshes before reading. Appended/new sessions should
appear on the next `rein sessions` or `rein search`.

The database is derived state and should rebuild automatically after corruption
or schema incompatibility. If diagnosis requires a manual reset, close
Reinstate and confirm `rein version --json`. For `v0.3.0-rc.1`,
move—not immediately delete—the exact
`session-index-v2.sqlite` database, its `.lock` and `.write.lock` coordination
files, and any SQLite `-journal`/`-wal`/`-shm` companions out
of `$REINSTATE_HOME/cache/`, then rerun:

```bash
rein sessions --json
```

Moving v2 also discards private prelaunch comparison history, so the next real
launch will truthfully require the `baseline.unavailable` warning again. Keep
the moved files until the rebuilt results are verified. Never move or
edit the vendor's Claude/Codex/Gemini/OpenCode session files to repair the
index.

## Session reference is ambiguous

Bare native IDs are accepted only when one indexed agent owns the ID. Use the
composite identity from `rein sessions`:

```text
claude:<native-session-id>
codex:<native-session-id>
```

Reinstate must not guess an agent or launch anything on ambiguity.

## Bare `rein` hangs or prints help in automation

The numbered switcher is TTY-only. Scripts should use:

```bash
rein sessions --json
```

A non-TTY bare invocation should exit promptly with usage code `2` and that
hint. If it waits for input, report the terminal/shell, OS, and redacted
command invocation.

## `resume`, `fork`, or `last` refuses to launch

Review the structured plan first:

```bash
rein resume claude:SESSION_ID --dry-run --json
rein fork codex:SESSION_ID --dry-run --json
rein last --dry-run --json
```

The recorded workspace and vendor executable must exist. Claude and Codex
launch through their own native commands; Reinstate does not translate one
vendor's transcript into another. Gemini and OpenCode are read-only in Phase 2
and intentionally return compatibility exit `5` for resume/fork.

`--json` requires `--dry-run` for a launch command so native child output
cannot be mixed into the JSON document.

### v0.3.0-rc.1: read the environment decision

Stable `v0.2.0` does not include verified resume. In candidate `v0.3.0-rc.1`,
`inspect` and native dry-runs include an `environment` report:

```bash
rein inspect claude:SESSION_ID --json
rein resume claude:SESSION_ID --dry-run --json
```

- `ready` means no warning or blocker was found.
- `confirmation_required` means a real launch needs explicit consent.
- `blocked` means the report is diagnostic only; no bypass exists.

`baseline.unavailable` on the first inspection/launch is expected. Reinstate
does not pretend the currently observed repository was the historical session
environment. On a TTY, review the report and type exactly `yes` at the prompt.
The default, empty input, `no`, or EOF refuses with safety exit `7`. For
non-interactive use, repeat an exact flag for every current warning:

```bash
rein resume claude:SESSION_ID \
  --allow-environment-warning baseline.unavailable \
  --allow-environment-warning git.branch
```

Do not copy IDs from an older run: the report is recomputed before launch.
Unknown, stale, duplicate, wildcard, and informational IDs are usage errors
(`2`); acknowledging only some current warnings is a safety refusal (`7`). A
blocked report returns its blocker exit regardless of supplied IDs—blockers
are never acknowledgements. There is no `--force` or persisted approval.

Common non-overridable blockers:

| Symptom | Exit | Action |
| ------- | ---- | ------ |
| Workspace missing/not a directory, executable missing, or agent version/layout unverified | `5` | Restore a supported same-vendor workspace/agent installation; do not substitute another vendor. |
| Known repository identity differs or selected vendor source could not refresh | `7` | Check that the recorded path still names the intended repository/session and retry. |
| Bounded Git, agent, capability, or runtime probe failed | `1` | Resolve the local probe failure or timeout; do not bypass it. |

When several blockers exist, exit precedence is runtime `1`, safety `7`, then
compatibility `5`; inspect the complete report because all checks remain
present. A blocked `inspect` itself exits `0` when the report was generated
successfully—automation must read `environment.decision`.

If a successful child later appears not to have established a baseline, verify
that it exited normally. Declined, cancelled, blocked, failed, or child-error
launches intentionally do not store one. Moving/resetting the v2 index also
removes baseline history, so the next launch warns again.

## `pull` does not make `claude --resume` see sessions

Usually a **path remap** issue:

1. Run `rein version --json` and require `0.1.0` or newer.
2. Confirm the same canonical project ID maps to this device's absolute
   `local_root` in `config.toml`.
3. Run a scoped `rein pull --agent claude --session SESSION_ID --dry-run` and
   verify the planned destination is under this device's Claude project
   directory, not the source device's directory key.
4. Close Claude Code, run the scoped pull, then require both
   `rein list --agent claude --json` and
   `claude --resume SESSION_ID` to find the exact restored session.

Do not manually move the session file. Reinstate rejects legacy snapshots whose
Claude project identity cannot be mapped safely; reinstall Reinstate on the source
device and push that selected session again to a fresh Reinstate profile.

Open an issue with OS pair (e.g. Windows 11 → macOS 15), agent version, and
**redacted** paths.

## Passphrase verification failed on second device

You must use the **exact same passphrase** as device 1. There is no recovery
from a wrong phrase against existing ciphertext.

Wait until Reinstate is visibly showing its hidden prompt before typing the
passphrase. If the process has already exited, rerun the command; otherwise the
secret can become a shell-history entry instead of input to Reinstate.

## Remote profile manifest not found

Reinstate could reach the configured storage location but did not find the
profile's encrypted `manifest.age`. Check all three non-secret coordinates:

1. `profile_id` and `storage.prefix` match the first device.
2. `storage.bucket` is the same bucket used by the first device.
3. `storage.endpoint` is the service endpoint only and does not end in the
   bucket name.

Do not work around this by creating an empty manifest or using a new profile
ID. Correct the setup inputs and rerun `init --profile-id` in a disposable or
intentionally reinitialized home. If reusing an initialized home, review it
first. Reinstate provides `init --force`, which backs up config and state together
before replacing them.

## Conflicts after using both machines the same day

Expected if both sides modified the same session. Reinstate records a conflict
rather than overwriting. Inspect its metadata with `rein conflicts show`, then
choose `--keep-local`, `--keep-remote`, or `--keep-both`. Keep-both creates a
distinct vendor-safe UUID session. Do not delete either branch until both
resume successfully and the active conflict list is empty.

## Huge Codex history / slow sync

Large `~/.codex/sessions` trees need delta/CAS sync (roadmap). Until then:

- Select a specific session instead of `--all`; retention controls are planned
- Exclude very old rollouts via config globs

## Accidentally almost synced credentials

Defaults block common credential paths. If you overrode excludes:

1. Rotate any exposed keys immediately
2. Remove objects from the remote bucket if uploaded
3. Restore excludes to secure defaults

## Agent was running during pull

You do not need to close unrelated agents. The default `fork` policy scopes
liveness to the exact selected session. If that session is in use, Reinstate
leaves it untouched and restores the incoming snapshot beside it under a
distinct vendor-safe UUID. Repeating the pull reuses the same fork without
rewriting or backing up identical content.

If configuration explicitly selects the `scoped` refusal policy, close the
named session or change the reviewed policy; do not use a permission bypass.
`--keep-remote` still refuses to replace a target that is in use, while
`--keep-both` preserves both.

## Still stuck?

- [FAQ](faq.md)
- [SUPPORT.md](../SUPPORT.md)
- [GitHub Issues](https://github.com/HarjjotSinghh/reinstate/issues)
