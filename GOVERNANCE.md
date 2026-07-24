# Governance

## Project

**Reinstate** is an open-source project created and led by
**Harjot Singh Rana** ([@HarjjotSinghh](https://github.com/HarjjotSinghh)).

## Decision making

| Decision type | Who decides |
| ------------- | ----------- |
| Day-to-day code merges | Core maintainers ([MAINTAINERS.md](MAINTAINERS.md)) |
| Roadmap priorities | Lead maintainer, informed by community issues/discussions |
| Security policy changes | Lead maintainer + any security-designated maintainers |
| License changes | Lead maintainer (requires clear notice; breaking license changes are avoided) |
| Adding/removing maintainers | Existing core maintainers by consensus |

While the project is small, it operates under a **benevolent dictator (BDFL)**
model with open review. As the community grows we may move toward a formal
steering committee; that change will be documented here first.

## Branch strategy

| Branch | Purpose |
| ------ | ------- |
| `main` | Stable development line; should always build and pass CI |
| `release/x.y` | Optional release stabilization branches |
| feature branches | Short-lived; opened via fork PRs |

Tags follow [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`).
Pre-1.0 releases may include breaking changes in minor versions with clear
CHANGELOG notes.

## Releases

See [RELEASING.md](RELEASING.md). Only maintainers publish GitHub Releases.

## Conflict resolution

1. Discuss in the PR or issue in good faith
2. Escalate to a core maintainer for a binding decision
3. Code of Conduct violations are handled per [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Trademark / branding

The name **Reinstate**, logos, and brand assets are used to identify this
project. Do not imply official endorsement for forks without coordination.
Forks should clearly state they are unofficial derivatives.

## Changes to governance

Propose changes via PR against this file. Material changes require approval
from the lead maintainer.
