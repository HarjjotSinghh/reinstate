# Configuration

Default home:

- macOS/Linux/WSL: `~/.reinstate`
- Windows: `%USERPROFILE%\.reinstate`
- Override: `REINSTATE_HOME=/absolute/path`

## Files

| Path | Format | Contents |
| ---- | ------ | -------- |
| `config.toml` | TOML v1 | profile mode, storage endpoint, bucket, prefix, credential_ref, agents, projects |
| `state.json` | JSON v1 | last remote etag, session revisions |
| `device.json` | JSON | optional device metadata |
| `cache/`, `backups/`, `conflicts/`, `locks/`, `logs/` | dirs | runtime |

Secrets are **never** valid config fields. Interactive setup stores S3/R2
credentials in macOS Keychain, Windows Credential Manager, or the supported OS
keyring provider. Non-interactive setup may use the explicit
`REINSTATE_S3_ACCESS_KEY_ID` and `REINSTATE_S3_SECRET_ACCESS_KEY` provider; the
values are not written to disk.

Every device in one sync set must use the same `profile_id`, bucket, and prefix.
Copy the non-secret profile UUID printed by the first device and pass it as
`rein init --profile-id UUID` on later devices.

The endpoint is the S3/R2 service endpoint only. Do not append the bucket name;
the bucket is configured separately. Reinstate additional-device init verifies the
existing encrypted manifest before saving config and records
`remote_profile_required = true`. `status`, `diff`, `pull`, and later pushes
then fail if that manifest disappears instead of treating the profile as empty.

Reinstate refuses to run `init` against a home that already contains `config.toml` or
`state.json`. Its explicit `--force` path first preserves both files in one
timestamped directory under `backups/`.

Project paths are portable only when each device defines the same canonical ID:

```bash
rein init --project github.com/acme/app=/absolute/local/path
```

## Reinstate Hop

```toml
[hop]
url = "https://hop.reinstate.dev"
```

Optional. `REINSTATE_HOP_URL` takes precedence; the default is the production
control plane. The device token issued by `rein login` is never a config
field: it lives in the OS keyring under `reinstate` / `hop/device-token`. See
[hop.md](hop.md).

## Restore safety

A restore replaces a vendor session file, so Reinstate first checks whether an
agent is using that file. The check is scoped to the exact session being
replaced — having Claude Code or Codex open in other projects is normal and does
not block anything.

```toml
[restore]
active_agent_policy = "fork"
```

| Policy | Behavior |
| ------ | -------- |
| `fork` (default) | Never blocks. If the target session is in use, the live file is left untouched and the remote copy is restored beside it as a distinct session. |
| `scoped` | Refuse when the target session file is held open by that agent. |
| `strict` | Refuse whenever the agent runs anywhere on the host. |
| `off` | Skip the liveness check entirely. |

Under `fork`, a `pull` reports the new session it created:

```text
pulled 1 snapshot(s), dry_run=false
  claude:SESSION -> ... (backups: ...)
    SESSION is in use, so it was left unchanged; restored alongside it as 7c9e6679-7425-40de-944b-e07fc1f90ae7
```

The fork identity is a UUID derived from the snapshot. Deriving it keeps
re-pulling the same remote state idempotent: the second pull recognizes that the
fork already holds those bytes and leaves it untouched rather than rewriting and
backing it up again. Using a UUID rather than a decorated name matters because
vendors treat session identifiers as UUIDs, and a decorated form is accepted by
Claude Code's interactive resume but rejected by `claude --print --resume`. Because the live
session is never replaced, a forked restore does not record a conflict and does
not mark the original session as synchronized.

`--allow-active-agents` applies `off` to a single `rein pull` or
`rein conflicts resolve` run.

`rein conflicts resolve --keep-remote` keeps the refusal even under the `fork`
policy, because `--keep-both` is already the explicit way to preserve both
branches there.

Relaxing this policy does not remove the other protections. Restores always
write to a temporary file and rename it into place, existing targets are always
backed up first, and a restore is abandoned if the target changes on disk while
the replacement is being prepared.

Liveness is decided from three independent signals, and any one of them marks a
session in use:

1. a matching agent process holds the session file open (`lsof` on macOS and
   Linux, the Restart Manager API on Windows);
2. a matching agent process names the exact session on its command line, which
   covers `claude --resume <id>` and `codex resume <id>`; and
3. a matching agent process is working inside that session's mapped project.

An open handle alone is not enough. Claude Code appends to its session file and
closes it again, so a live Claude Code session holds no handle at all; a
handle-only check reports it as free. Detection therefore biases toward "in
use": under the default `fork` policy a false positive costs one extra session
file, while a false negative would land a restore on live work.

Signal 3 needs a process working directory. That is read with `lsof` on macOS
and Linux. Windows keeps it in the target process's PEB, which would require
cross-process memory access, so project affinity contributes nothing there and
signals 1 and 2 carry the check.

One consequence worth knowing: having an agent open anywhere inside a mapped
project makes restores of that project's sessions fork rather than replace in
place. That is the deliberate trade for never overwriting a session someone is
working in.

## Encryption

`encryption.type` selects the key model; nothing else in sync changes.

- `age-scrypt` (default): BYO storage. A passphrase typed on every device
  derives the key. The passphrase is not stored in config.
- `root-key`: the hosted key model. `rein account init` or
  `rein account recover` sets this after enrolling the device; the root key is
  unwrapped from the keyring in storage with the device key held in the OS
  keyring. No passphrase is asked for. See the "Hosted key model" section of
  [security-model.md](security-model.md).

## Universal agent configuration (roadmap)

`config.toml` above configures Reinstate itself. A later, separate desired-state
profile will describe portable agent capabilities: MCP servers,
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings.
Configuration adapters will render that profile into each selected harness's
native format and encrypted sync will distribute the non-secret profile across
devices.

The intended workflow includes `rein mcp add` and
`rein config import|diff|apply|status|sync`; these commands are not in the
current `v0.1` CLI.

Raw secret values are not valid desired-state fields. MCP/API credentials and
OAuth tokens remain in the OS keychain or an explicit local secret provider;
the profile stores only references. See
[universal-configuration.md](universal-configuration.md).
