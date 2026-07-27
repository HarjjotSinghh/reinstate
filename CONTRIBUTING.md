# Contributing to Reinstate

Thanks for your interest in contributing! Reinstate is built in public for
developers who live on more than one machine and more than one AI coding agent.

This guide covers how to report bugs, propose features, and submit code.

## Table of contents

- [Code of Conduct](#code-of-conduct)
- [Ways to contribute](#ways-to-contribute)
- [Development setup](#development-setup)
- [Contributor guides](#contributor-guides)
- [Pull request process](#pull-request-process)
- [Adapter contributions](#adapter-contributions)
- [Coding standards](#coding-standards)
- [Commit messages](#commit-messages)
- [Security](#security)

## Code of Conduct

Participation is governed by our [Code of Conduct](CODE_OF_CONDUCT.md).
Be kind, be constructive, and assume good intent.

## Ways to contribute

| Area | How |
| ---- | --- |
| **Bugs** | [Open a bug report](https://github.com/HarjjotSinghh/reinstate/issues/new?template=bug_report.yml) |
| **Features** | [Feature request](https://github.com/HarjjotSinghh/reinstate/issues/new?template=feature_request.yml) |
| **New agents** | [Adapter request](https://github.com/HarjjotSinghh/reinstate/issues/new?template=adapter_request.yml) |
| **Docs** | PRs to `docs/` and README are always welcome |
| **Code** | Fork → branch → PR (see below) |
| **Questions** | Public Q&A through the [question issue form](https://github.com/HarjjotSinghh/reinstate/issues/new?template=question.yml) |

Good first issues are labeled [`good first issue`](https://github.com/HarjjotSinghh/reinstate/labels/good%20first%20issue)
and [`help wanted`](https://github.com/HarjjotSinghh/reinstate/labels/help%20wanted).

## Development setup

### Prerequisites

- Go **1.25.12+** (the pinned toolchain is declared in `go.mod`)
- `make`

### Clone and build

```bash
git clone https://github.com/HarjjotSinghh/reinstate.git
cd reinstate
make deps
make build
./bin/rein version          # short alias (preferred)
./bin/reinstate version     # full name
```

### Run tests

```bash
make quick
make test
make test-race
make lint
make verify
```

Use `make quick` while iterating. It runs formatting, vet, and the product
packages while omitting the slow production-KDF and subprocess/document
contract packages, and it reuses Go's test cache for unchanged packages. It is
deliberately not a release gate.

`make test` runs every package, including documentation, installer, and fixture
secret-scan contracts.
`make test-race` race-instruments product packages while excluding
`internal/doctest`, whose subprocess/document contracts cannot expose product
memory races, and the stateless `internal/crypto` wrapper. High-level CLI,
doctor, and sync tests use the real age envelope format with a reduced
test-only scrypt cost. `make test` still exercises the production crypto
default. This preserves the useful functional and race signals without
repeating expensive KDF and documentation work.

### Local smoke test

```bash
# Scan synthetic fixtures for accidental secrets
make fixture-scan
```

## Contributor guides

- [Development workflow](docs/contributing/development.md)
- [Documentation workflow](docs/contributing/documentation.md)
- [Release process](docs/contributing/release-process.md)

## Pull request process

1. **Open an issue first** for larger changes so we can align on design.
2. Fork the repo and create a branch from `main`:
   ```bash
   git checkout -b feat/my-change
   ```
3. Make focused commits (one logical change per commit when possible).
4. Add or update tests for behavioral changes.
5. Update docs if you change CLI flags, config, or adapters.
6. Ensure `make verify` passes.
7. Open a PR against `main` using the PR template.
8. Address review feedback. Maintainers aim for a first response within
   **5 business days**.

### PR checklist (summary)

- [ ] Tests added/updated
- [ ] `make verify` passes
- [ ] Docs updated if user-facing
- [ ] CHANGELOG updated for user-visible behavior
- [ ] Config/schema migration impact documented
- [ ] No secrets or real session files committed
- [ ] Conventional commit title preferred
- [ ] Linked issue (`Closes #123`) when applicable

## Adapter contributions

New agent adapters are a first-class contribution path. See
[docs/adapters.md](docs/adapters.md) for the adapter interface and synthetic-fixture
requirements, then read
[Contributing an adapter](docs/adapters/contributing-an-adapter.md) and the
[fixture policy](docs/contributing/testing.md).

Minimum for a new adapter PR:

1. Implementation under `internal/adapter/<name>/`
2. Synthetic fixtures under `testdata/adapters/<name>/`
3. Docs entry in `docs/adapters.md`, `docs/compatibility.md`, and the README
   support matrix
4. Defensive parsing (unknown line types must not crash)
5. Explicit **exclude list** for credential / cache paths

## Coding standards

- Prefer clear, boring code over clever abstractions
- Keep the CLI surface small: `init`, `push`, `pull`, `status`, `diff`, `conflicts`
- Security defaults must be safe (encryption on, credentials excluded)
- No network calls in unit tests without explicit test doubles
- Never log passphrases, keys, or full session contents

## Commit messages

We prefer [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(adapter): add Gemini CLI session adapter
fix(pathmap): rewrite Windows cwd into macOS home tokens
docs: clarify R2 setup in getting-started
chore(ci): pin golangci-lint version
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `ci`, `build`, `revert`.

## Security

Do **not** open public issues for vulnerabilities. Follow [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the
[Apache License 2.0](LICENSE).

No CLA is required. Do not add `Signed-off-by` lines unless the repository
explicitly adopts DCO in a future governance change.

---

Questions? Open a redacted [question issue](https://github.com/HarjjotSinghh/reinstate/issues/new?template=question.yml) or ping [@HarjjotSinghh](https://github.com/HarjjotSinghh).
