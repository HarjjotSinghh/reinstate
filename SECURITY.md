# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |
| main    | :white_check_mark: |

We currently support the latest stable release and the `main` branch with
security fixes. Older pre-1.0 minors may not receive backports.

## Reporting a Vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Report security issues privately via one of:

1. **[GitHub Security Advisories](https://github.com/HarjjotSinghh/reinstate/security/advisories/new)** (preferred)
2. Email: **security@harjot.co** (or contact the maintainer listed in [MAINTAINERS.md](MAINTAINERS.md))

Include as much of the following as you can:

- Description of the issue and potential impact
- Steps to reproduce (PoC preferred)
- Affected version / commit
- Suggested fix (optional)

### What to expect

| Step | Timeline |
| ---- | -------- |
| Acknowledgement | Within **72 hours** |
| Initial assessment | Within **7 days** |
| Fix / mitigation (critical) | As fast as possible; target **14 days** |
| Public disclosure | Coordinated with reporter after a fix is available |

We will credit reporters who wish to be named (see [SECURITY-THANKS.md](docs/security-model.md) style acknowledgements once established).

## Security Model (summary)

Reinstate is designed around a **local-first, zero-knowledge** trust model:

- **Client-side encryption** before any byte leaves the machine (age with
  passphrase-derived keys by default)
- **Bring-your-own storage** (R2 / S3 / GCS / WebDAV / Gist) — ciphertext only
  at rest on remote backends
- **Hard exclusions** for credential files (`auth.json`, OAuth tokens, API key
  configs) — never synced by default
- **Atomic restore** with timestamped local backups before overwrite
- **No vendor API credentials** required — Reinstate only reads/writes
  user-owned local agent session files

Full details: [docs/security-model.md](docs/security-model.md)

## Out of scope

- Compromised host machines (malware reading local keys / passphrase)
- Weak user-chosen passphrases
- Secrets that agents already wrote into session transcripts (use redaction /
  exclude rules)
- Issues in third-party agent CLIs themselves (Claude Code, Codex, etc.)

## Supply chain

- Releases are published as signed / checksummed artifacts when available
- Dependabot and CI pin dependency updates
- Report compromised release artifacts the same way as other vulnerabilities

Thank you for helping keep Reinstate and its users safe.
