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
the bucket is configured separately. RC6 additional-device init verifies the
existing encrypted manifest before saving config and records
`remote_profile_required = true`. `status`, `diff`, `pull`, and later pushes
then fail if that manifest disappears instead of treating the profile as empty.

RC6 refuses to run `init` against a home that already contains `config.toml` or
`state.json`. Its explicit `--force` path first preserves both files in one
timestamped directory under `backups/`.

Project paths are portable only when each device defines the same canonical ID:

```bash
rein init --project github.com/acme/app=/absolute/local/path
```

## Encryption

Default: `age-scrypt` passphrase. Passphrase is not stored in config.

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
