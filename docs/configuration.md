# Configuration

Default home:

- macOS/Linux/WSL: `~/.reinstate`
- Windows: `%USERPROFILE%\.reinstate`
- Override: `REINSTATE_HOME=/absolute/path`

## Files

| Path | Format | Contents |
| ---- | ------ | -------- |
| `config.toml` | TOML v1 | storage endpoint, bucket, prefix, credential_ref, agents, projects |
| `state.json` | JSON v1 | last remote etag, session revisions |
| `device.json` | JSON | optional device metadata |
| `cache/`, `backups/`, `conflicts/`, `locks/`, `logs/` | dirs | runtime |

Secrets are **never** valid config fields. Use credential refs + env/keyring.

## Encryption

Default: `age-scrypt` passphrase. Passphrase is not stored in config.
