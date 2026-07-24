# Support

Welcome — here's how to get help with **Reinstate**.

## Self-serve (fastest)

| Resource | Use when |
| -------- | -------- |
| [Getting started](docs/getting-started.md) | First install / second-device setup |
| [Architecture](docs/architecture.md) | Understanding how sync works |
| [Security model](docs/security-model.md) | Encryption, keys, what is never synced |
| [Adapters](docs/adapters.md) | Agent-specific paths and resume commands |
| [Comparison](docs/comparison.md) | vs native vendor sync / claude-sync / DIY |
| [FAQ](docs/faq.md) | Common questions |
| [Troubleshooting](docs/troubleshooting.md) | Path remap, conflicts, restore failures |

## Community support

- **[GitHub Discussions](https://github.com/HarjjotSinghh/reinstate/discussions)** — questions, show-and-tell, ideas
- **[GitHub Issues](https://github.com/HarjjotSinghh/reinstate/issues)** — confirmed bugs and feature requests only
  - Bug: use the bug report template
  - Feature: use the feature request template
  - New agent: use the adapter request template

Please search existing issues and discussions before opening a new one.

## What we can and cannot help with

**In scope**

- Reinstate CLI install, config, push/pull, conflicts
- Adapter bugs (wrong paths, failed remaps, corrupt restores)
- Documentation gaps

**Out of scope**

- Support for Claude Code / Codex / Gemini / etc. themselves (use vendor channels)
- Debugging your application code or agent prompts
- Hosting / cloud account billing (R2, AWS, GCS) beyond config examples
- Real-time pair-programming or multi-writer collaboration

## Security issues

Report privately — see [SECURITY.md](SECURITY.md). **Do not** post vulnerability
details in public issues.

## Maintainer response times

This is an open-source project maintained primarily by
[Harjot Singh Rana](https://github.com/HarjjotSinghh). Best-effort SLAs:

| Type | Target first response |
| ---- | --------------------- |
| Security report | 72 hours |
| Bugs on latest stable | 5 business days |
| Features / discussions | as capacity allows |

There is **no paid support SLA** for the free open-source CLI today.

## Commercial / hosted

A hosted zero-knowledge convenience layer may be offered later (see
[ROADMAP.md](ROADMAP.md)). Until then, the CLI + bring-your-own storage is free
and fully open source under Apache-2.0.
