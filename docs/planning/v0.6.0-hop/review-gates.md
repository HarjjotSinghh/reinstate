# Review gates — v0.6.0

What the coordinator verifies before merging any executor branch into
`release/v0.6.0-rc.1`. Every gate is a command or a check with a yes-or-no
answer. A verifier agent runs Gates 1–4 adversarially and reports; the
coordinator re-runs Gate 1 on the merged result.

---

## Gate 0 — Scope

| Check | Fail condition |
| ----- | -------------- |
| Ownership | `git diff --name-only release/v0.6.0-rc.1...HEAD` includes a path the workstream does not own in [file-ownership.md](file-ownership.md) |
| Task match | The change does something the card did not ask for, or leaves out something it did |
| Size | The diff cannot be held in one review and was not split |
| Commits | Not Conventional Commits, or missing the `Co-Authored-By` trailer, or a commit message that claims more than the diff does |

---

## Gate 1 — Build and test (on this Windows host)

```bash
gofmt -l .                                            # empty
GOTOOLCHAIN=go1.25.13 go vet ./...
GOTOOLCHAIN=go1.25.13 go mod tidy -diff               # empty
CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test ./... -count=1
CGO_ENABLED=1 GOTOOLCHAIN=go1.25.13 go test -race ./internal/... -count=1   # W2, W3, and any branch touching internal/
GOOS=darwin GOTOOLCHAIN=go1.25.13 go build ./... && GOOS=linux GOTOOLCHAIN=go1.25.13 go build ./...
```

| Check | Fail condition |
| ----- | -------------- |
| All of the above | Any failure other than the preflight flake W2 owns, and after W2 merges, any failure at all |
| Assertions | **An existing assertion was edited to make the change pass** |
| Hermeticity | A test reads the host's `~/.claude`, `~/.codex`, `XDG_DATA_HOME`, or `REINSTATE_MEMORY_BACKEND_DIR` without `t.Setenv` |
| Windows | A test is newly skipped on Windows without a comment naming the reason and an issue |

---

## Gate 2 — Documentation truth

```bash
CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test ./internal/doctest/... -count=1
./scripts/check-docs.ps1     # or check-docs.sh
```

| Check | Fail condition |
| ----- | -------------- |
| Doc gate | Any failure |
| Platform claims | A sentence, table cell, JSON field, or changelog line says or implies `v0.6.0` was verified on macOS |
| Service claims | A sentence implies the hosted control plane is open, has pricing, or accepts sign-ups |
| Range claims | A widened range without the "native Windows evidence; macOS pending" qualifier, or a version number not in the results doc |
| Numbers | A row count, version, date, or commit that disagrees with the artifact it describes |

---

## Gate 3 — Security and privacy

```bash
CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test ./internal/fixture -count=1      # secret scanner
git diff release/v0.6.0-rc.1...HEAD | grep -nE 'AKIA|-----BEGIN|passphrase=|Bearer [A-Za-z0-9]' # empty
```

| Check | Fail condition |
| ----- | -------------- |
| Secrets | Any credential, token, recovery code, or passphrase in a diff, fixture, or report |
| Transcripts | Any real session content, prompt, or private path in a fixture or results doc |
| Redaction | A results doc or probe artifact that names the host user, a repository, or a filename outside `testdata/` |
| Defaults | Encryption, credential exclusion, or fail-closed behaviour weakened anywhere |

---

## Gate 4 — Evidence (W3, W6, W7 only)

| Check | Fail condition |
| ----- | -------------- |
| Real binary | A resume, handoff, or sync row that ran against a fake vendor when the contract says real |
| Non-vacuous | A row that passed without exercising the mechanism it names (the B4-on-Windows lesson: check the harness before the product) |
| Counts | The report's required-row count differs from `rein doctor --agents --acceptance-matrix` on the same binary |
| Commit identity | The report names a commit that is not the branch tip it claims |
| Deferred table | A macOS row missing from the deferred list, or a deferred row marked anything but `DEFERRED` |

---

## What the coordinator does on a rejection

Small and mechanical (a typo, a missing trailer, one wrong number): fix on the
branch, note it in the merge commit, merge. Anything else: send the branch
back with the failing gate and the exact command, or open a fix card and
assign it. Never merge to see whether it works.
