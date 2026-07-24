<div align="center">

<img src="assets/banner.svg" alt="Reinstate" width="100%" />

# Reinstate

### The sync layer for your entire AI development environment

**Sessions · MCP servers · Skills · Settings** — across every coding agent and every machine you own.
Encrypted so only you can read it.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/HarjjotSinghh/reinstate)](https://goreportcard.com/report/github.com/HarjjotSinghh/reinstate)
[![Release](https://img.shields.io/github/v/release/HarjjotSinghh/reinstate?include_prereleases&sort=semver)](https://github.com/HarjjotSinghh/reinstate/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/HarjjotSinghh/reinstate/ci.yml?branch=main&label=CI)](https://github.com/HarjjotSinghh/reinstate/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/HarjjotSinghh/reinstate)](go.mod)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)
[![Security Policy](https://img.shields.io/badge/Security-policy-red.svg)](SECURITY.md)

[![GitHub stars](https://img.shields.io/github/stars/HarjjotSinghh/reinstate?style=social)](https://github.com/HarjjotSinghh/reinstate/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/HarjjotSinghh/reinstate?style=social)](https://github.com/HarjjotSinghh/reinstate/network/members)
[![GitHub watchers](https://img.shields.io/github/watchers/HarjjotSinghh/reinstate?style=social)](https://github.com/HarjjotSinghh/reinstate/watchers)
[![GitHub issues](https://img.shields.io/github/issues/HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate/pulls)
[![GitHub contributors](https://img.shields.io/github/contributors/HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate/graphs/contributors)
[![GitHub last commit](https://img.shields.io/github/last-commit/HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate/commits/main)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/m/HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate/graphs/commit-activity)
[![GitHub repo size](https://img.shields.io/github/repo-size/HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate)
[![Downloads](https://img.shields.io/github/downloads/HarjjotSinghh/reinstate/total)](https://github.com/HarjjotSinghh/reinstate/releases)

<p>
  <a href="#why-reinstate"><strong>Why</strong></a> ·
  <a href="#features"><strong>Features</strong></a> ·
  <a href="#quick-start"><strong>Quick start</strong></a> ·
  <a href="#supported-agents"><strong>Agents</strong></a> ·
  <a href="#how-it-works"><strong>How it works</strong></a> ·
  <a href="#documentation"><strong>Docs</strong></a> ·
  <a href="#roadmap"><strong>Roadmap</strong></a> ·
  <a href="#contributing"><strong>Contributing</strong></a>
</p>

<sub>Created & maintained by <a href="https://github.com/HarjjotSinghh"><strong>Harjot Singh Rana</strong></a> · <a href="https://harjot.co">harjot.co</a></sub>

</div>

---

## The problem

You grind eight hours on your **Windows desktop** across Claude Code, Codex, Gemini CLI, OpenCode, Grok Build… twenty sessions, four projects, full context.

You open your **MacBook** on the couch.

Git has the code. The agent does **not** have the conversation — the rejected approaches, the files already read three times, the style constraints you established mid-thread. Vendor tools save sessions **locally**. Switching machines is context death.

```
Desktop (Windows)          Laptop (macOS)
┌─────────────────┐        ┌─────────────────┐
│ 20 sessions     │   ✗    │ empty history   │
│ MCP + skills A  │ ─────► │ MCP + skills B  │
│ full context    │        │ start from zero │
└─────────────────┘        └─────────────────┘
```

**Reinstate** makes state portable:

```
Desktop                    Encrypted cloud              Laptop
┌──────────┐   push   ┌─────────────────┐   pull   ┌──────────┐
│ sessions │ ───────► │ your R2/S3/etc  │ ───────► │ resume   │
│ + config │  age E2E │ ciphertext only │  remap   │ same IDs │
└──────────┘          └─────────────────┘  paths   └──────────┘
```

---

## Why Reinstate?

| | What you get |
| --- | --- |
| **Universal** | Multi-agent — not Claude-only, not Codex-only |
| **Offline-capable origin** | Works when the other machine is **off** (stored sync, not a live relay) |
| **Path remapping** | Windows ↔ macOS project paths rewritten so `--resume` actually finds sessions |
| **Zero-knowledge** | Client-side encryption; bring-your-own storage |
| **Config + sessions** | MCP servers, skills, agents, settings — one environment everywhere |
| **Open source** | MIT · auditable · no vendor lock-in |

Native vendor sync will always own *one* ecosystem. DIY Syncthing/Drive hacks break on absolute paths and credential sprawl. Reinstate targets the empty quadrant: **universal × cross-device × encrypted × resume-aware**.

<p align="center">
  <img src="assets/01_landscape.png" alt="Landscape: agent scope vs state portability" width="720" />
</p>

---

## Features

- **Cross-device session sync** — continue the same agent thread on another machine
- **Multi-agent adapters** — Claude Code, Codex, Gemini CLI, OpenCode, Grok Build (phased)
- **End-to-end encryption** — [age](https://github.com/FiloSottile/age), passphrase-derived keys
- **Bring-your-own storage** — Cloudflare R2, AWS S3, GCS, S3-compatible, WebDAV
- **OS-aware path remapping** — the hard problem treated as the product
- **Selective scopes** — `sessions` | `config` | `all`
- **Safe by default** — credential denylist, atomic restore, conflict forks, local backups
- **Simple CLI** — `init` · `push` · `pull` · `status` · `diff` · `conflicts`

---

## Quick start

> **Note:** v0.1 CLI is under active development. The UX below is the target
> interface. Star & watch the repo for the first stable release.

### Install

```bash
# From source
git clone https://github.com/HarjjotSinghh/reinstate.git
cd reinstate
make build
./bin/reinstate version

# Go install (when module is published)
go install github.com/HarjjotSinghh/reinstate/cmd/reinstate@latest
```

### Device A

```bash
reinstate init          # backend + passphrase + path map
reinstate push          # encrypt & upload
```

### Device B

```bash
reinstate init          # SAME backend + SAME passphrase
reinstate pull --dry-run
reinstate pull
claude --resume         # or: codex resume
```

Full walkthrough: **[docs/getting-started.md](docs/getting-started.md)**

---

## Supported agents

| Agent | Sessions | Config / MCP | Status |
| ----- | :------: | :----------: | ------ |
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code) | ✅ | ✅ | Priority (v0.1) |
| [OpenAI Codex CLI](https://github.com/openai/codex) | ✅ | ✅ | Priority (v0.1) |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | 📋 | 📋 | Phase 1 |
| [OpenCode](https://opencode.ai) | 📋 | 📋 | Phase 1 |
| [Grok Build](https://x.ai) | 📋 | 📋 | Phase 2 |

Details: **[docs/adapters.md](docs/adapters.md)**

---

## How it works

```
Adapters → Normalize (paths) → Encrypt (age) → Sync engine → Your bucket
                ↑                                         │
                └──────── pull / decrypt / remap ←────────┘
```

1. **Adapters** read each tool’s local session layout  
2. **Pathmap** rewrites absolute paths to portable tokens and back  
3. **Crypto** encrypts before any upload  
4. **Sync** uses a local manifest; restores are atomic with backups  

Deep dive: **[docs/architecture.md](docs/architecture.md)**

<p align="center">
  <img src="assets/05_architecture.png" alt="Reinstate architecture" width="720" />
</p>

---

## Security

| Guarantee | Detail |
| --------- | ------ |
| E2E encryption | Ciphertext only on remote storage |
| Credential denylist | `auth.json` and tokens never synced by default |
| No vendor API keys required | Local files only |
| Fail-safe restore | Backups + conflict forks |

Report vulnerabilities privately: **[SECURITY.md](SECURITY.md)** · model: **[docs/security-model.md](docs/security-model.md)**

---

## Documentation

| Doc | Description |
| --- | ----------- |
| [Getting started](docs/getting-started.md) | Install, init, dual-device setup |
| [Architecture](docs/architecture.md) | Pipeline, packages, design principles |
| [Adapters](docs/adapters.md) | Per-agent layouts & support matrix |
| [Security model](docs/security-model.md) | Threat model & defaults |
| [Comparison](docs/comparison.md) | vs native sync, claude-sync, DIY |
| [FAQ](docs/faq.md) | Common questions |
| [Troubleshooting](docs/troubleshooting.md) | Path remap, conflicts, large histories |
| [Roadmap](ROADMAP.md) | Phases & non-goals |
| [Contributing](CONTRIBUTING.md) | Dev setup & PR process |
| [Releasing](RELEASING.md) | Maintainer release checklist |
| [Support](SUPPORT.md) | How to get help |
| [Governance](GOVERNANCE.md) | Decision making |
| [Changelog](CHANGELOG.md) | Release history |

---

## Project activity

<!-- These graphs populate once the repo is public and has traffic -->

### Star history

[![Star History Chart](https://api.star-history.com/svg?repos=HarjjotSinghh/reinstate&type=Date)](https://star-history.com/#HarjjotSinghh/reinstate&Date)

### Contributors

[![Contributors](https://contrib.rocks/image?repo=HarjjotSinghh/reinstate)](https://github.com/HarjjotSinghh/reinstate/graphs/contributors)

### Category context

<p align="center">
  <img src="assets/03_traction.png" alt="Category traction context" width="640" />
</p>

### Insights (after the repo is public)

| Metric | Link |
| ------ | ---- |
| Pulse | [github.com/…/pulse](https://github.com/HarjjotSinghh/reinstate/pulse) |
| Traffic | [graphs/traffic](https://github.com/HarjjotSinghh/reinstate/graphs/traffic) |
| Commits | [graphs/commit-activity](https://github.com/HarjjotSinghh/reinstate/graphs/commit-activity) |
| Code frequency | [graphs/code-frequency](https://github.com/HarjjotSinghh/reinstate/graphs/code-frequency) |
| Network | [network](https://github.com/HarjjotSinghh/reinstate/network) |
| Star history | [star-history.com](https://star-history.com/#HarjjotSinghh/reinstate&Date) |
| Repobeats | Generate embed at [repobeats.axiom.co](https://repobeats.axiom.co) after traffic exists |

---

## Roadmap

| Phase | Focus | Status |
| ----- | ----- | ------ |
| **0** | Claude + Codex, push/pull, path remap, encryption | 🚧 |
| **1** | Gemini + OpenCode, config/MCP scope | 📋 |
| **2** | Shell hooks, Grok, delta sync, redaction | 📋 |
| **3** | Hosted ZK convenience, session browser, teams | 💭 |

Full detail: **[ROADMAP.md](ROADMAP.md)**

---

## Stable releases

| Channel | Tag | Notes |
| ------- | --- | ----- |
| **Latest** | [![Release](https://img.shields.io/github/v/release/HarjjotSinghh/reinstate?label=latest)](https://github.com/HarjjotSinghh/reinstate/releases/latest) | Production-minded builds |
| **Pre-release** | [![Pre-release](https://img.shields.io/github/v/release/HarjjotSinghh/reinstate?include_prereleases&label=pre)](https://github.com/HarjjotSinghh/reinstate/releases) | Early adopters |
| **SemVer** | pre-1.0 | Breaking changes possible in minors — see CHANGELOG |

Install from [GitHub Releases](https://github.com/HarjjotSinghh/reinstate/releases) (checksums attached when CI release workflow is enabled).

---

## Maintainers

| | |
| --- | --- |
| **Lead** | [Harjot Singh Rana](https://github.com/HarjjotSinghh) ([@HarjjotSinghh](https://github.com/HarjjotSinghh)) |
| **Site** | [harjot.co](https://harjot.co) |
| **X** | [@HarjjotSinghh](https://x.com/HarjjotSinghh) |

See [MAINTAINERS.md](MAINTAINERS.md) · [AUTHORS](AUTHORS) · [CODEOWNERS](.github/CODEOWNERS)

---

## Contributing

Contributions are welcome — code, adapters, docs, and bug reports.

1. Read [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md)
2. Open an issue for larger changes
3. Fork → branch → PR

```bash
make deps && make test && make build
```

Good first issues: [`good first issue`](https://github.com/HarjjotSinghh/reinstate/labels/good%20first%20issue)

---

## Community & support

- **Discussions** — [GitHub Discussions](https://github.com/HarjjotSinghh/reinstate/discussions)
- **Bugs / features** — [Issues](https://github.com/HarjjotSinghh/reinstate/issues)
- **Security** — [SECURITY.md](SECURITY.md) (private)
- **Support guide** — [SUPPORT.md](SUPPORT.md)

---

## Citation

If you use Reinstate in research or publications:

```bibtex
@software{rana_reinstate_2026,
  author = {Rana, Harjot Singh},
  title  = {Reinstate: Encrypted multi-agent session sync for AI coding tools},
  year   = {2026},
  url    = {https://github.com/HarjjotSinghh/reinstate},
  license = {MIT}
}
```

Also see [CITATION.cff](CITATION.cff).

---

## License

[MIT](LICENSE) © 2026 [Harjot Singh Rana](https://github.com/HarjjotSinghh)

See also [NOTICE](NOTICE) for third-party acknowledgements. Product names of third-party agents are trademarks of their respective owners; Reinstate is an independent project and is not affiliated with or endorsed by those vendors.

---

<div align="center">

**Work on desktop. Resume on laptop. Keep the context.**

[![Star this repo](https://img.shields.io/github/stars/HarjjotSinghh/reinstate?style=for-the-badge&logo=github)](https://github.com/HarjjotSinghh/reinstate)

<sub>Built with ☕ in New Delhi · Open source forever</sub>

</div>
