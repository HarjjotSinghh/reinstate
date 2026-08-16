# Review gates

What the coordinator verifies before merging any task PR into
`feat/universal-agent-coverage`. Every gate is a command or a check with a
yes-or-no answer.

---

## Gate 0 — Scope

| Check | Fail condition |
| ----- | -------------- |
| Ownership | `git diff --name-only` includes a path the task does not own in [file-ownership.md](file-ownership.md) |
| Size | The diff is large enough that a reviewer cannot hold it, and it was not split |
| Task match | The change does something the card did not ask for |

Reject on ownership before reading the code. That is the whole value of the
ownership map.

---

## Gate 1 — Build and test

```bash
make verify
CGO_ENABLED=1 go test -race ./... -count=1
go mod tidy -diff
```

| Check | Fail condition |
| ----- | -------------- |
| `make verify` | Any failure |
| Race suite | Any failure or new flake |
| Module tidiness | A diff |
| Test assertions | **An existing assertion was edited to make the change pass** |

The last row is the important one during W0. If a refactor requires changing
what a test expects, the refactor changed behavior, which T-002 and T-003
forbid. Send it back rather than accepting an edited assertion.

---

## Gate 2 — Conformance and evidence

```bash
go test ./internal/agents/... -count=1
```

| Check | Fail condition |
| ----- | -------------- |
| Conformance | The agent's `conformance.Run` is absent or failing |
| Tier agreement | Declared tier does not match the present capability constructors |
| Evidence paths | Any path in the descriptor's `Evidence` does not exist |
| Probe artifacts | A tier of T1 or above without a macOS **and** a native Windows probe committed |
| Fixtures | Fixtures absent, or copied from a real tree rather than synthetic |
| Determinism | Two scans of one fixture differ |

An agent PR without probe artifacts is not a partial success. It is a T0
descriptor at best, and the tier in the descriptor must say so.

---

## Gate 3 — Documentation contracts

```bash
go test ./internal/doctest/ -count=1
npm --prefix website test
```

| Check | Fail condition |
| ----- | -------------- |
| Doctest | Any failure |
| Website vitest | Any failure |
| Storage page | The agent's page still shows `Unverified` for a row the descriptor relies on |
| Matrix row | `docs/compatibility.md` row missing, or added with a new column |
| CHANGELOG | `[Unreleased]` bullet missing, or existing bullets reflowed |

---

## Gate 4 — Privacy and safety

| Check | How | Fail condition |
| ----- | --- | -------------- |
| Secret scanner | `make verify` includes it | Any hit |
| Real transcripts | Read the fixtures | A fixture contains plausible real content, a real repository name, or a real username |
| Probe redaction | Read the committed probe JSON | Any absolute path, username, JSON value from a session file, or unnormalized identifier |
| Read-only | Inspect the scanner | Any write, rename, truncate, or lock under an agent root |
| Bounded reads | Inspect the scanner | An unbounded read that does not use the shared ceilings |
| Exclusions | Read the descriptor | A credential or cache subtree is not in `Excluded` |

Fixtures are the most common place a real transcript leaks into a repository.
Read them, do not skim them.

---

## Gate 5 — Product truth

| Check | Fail condition |
| ----- | -------------- |
| Tier claim | Any user-visible string implies a capability above the declared tier |
| Absolute claims | "All agents", "any agent", "every agent", "seamless", "universal resume" appear in shipped copy |
| Translation claim | Anything implies a transcript is converted between vendors |
| T0 honesty | A T0 agent's reason is vague, or implies support is imminent |
| Read-only reason | A record below T3 lacks a reason resume is refused |

The website tests catch some of this mechanically. Prose in Go strings, help
text, and error messages is not covered by those tests, so it is reviewed by
eye. Error messages are product copy.

---

## Gate 6 — Regression sanity, W0 only

Before T-003 merges, verify by hand on a machine with Claude Code and Codex
installed:

```bash
rein sessions --agent all
rein search "<known string>"
rein inspect claude:<id>
rein resume claude:<id> --dry-run
rein handoff --from codex:<id> --to claude --dry-run
rein status
```

| Check | Fail condition |
| ----- | -------------- |
| Output shape | Any difference from `v0.4.0` beyond new agents appearing |
| Ordering | Session ordering changed |
| Exit codes | Any command's exit code changed |
| Index upgrade | An index built by `v0.4.0` is emptied or rebuilt silently |

The index upgrade check protects a real user's data on upgrade day. Do it
before merging the refactor, not during acceptance.

---

## Merge

When all applicable gates pass:

```bash
gh pr merge <n> --squash --delete-branch
```

Squash, into `feat/universal-agent-coverage`, never into `main`. The
integration branch reaches `main` once, at the end, as a reviewed PR.

---

## When a gate fails

Return the PR with the failing gate named and the specific line or command.
Do not fix an executor's work yourself unless it is a one-line mechanical fix,
because doing so hides a misunderstanding that will recur on their next task.

If the same gate fails three times across different tasks, the card is wrong,
not the executors. Fix the card.
