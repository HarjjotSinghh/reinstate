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
the bucket is configured separately. In current unreleased source,
additional-device init verifies the existing encrypted manifest before saving
config and records `remote_profile_required = true`. `status`, `diff`, `pull`,
and later pushes then fail if that manifest disappears instead of treating the
profile as empty.

Current unreleased source refuses to run `init` against a home that already
contains `config.toml` or `state.json`. Its explicit `--force` path first
preserves both files in one timestamped directory under `backups/`. Published
RC4 lacks this guard; do not re-run `init` against an RC4 home.

Project paths are portable only when each device defines the same canonical ID:

```bash
rein init --project github.com/acme/app=/absolute/local/path
```

## Encryption

Default: `age-scrypt` passphrase. Passphrase is not stored in config.
