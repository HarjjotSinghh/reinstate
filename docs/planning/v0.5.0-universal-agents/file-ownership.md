# File ownership

This is the anti-conflict contract. A swarm of executors working on one feature
fails on merge conflicts long before it fails on logic, so every path has
exactly one owner at a time.

**A pull request that modifies a path it does not own is rejected without
review.** That is not a judgement about the change; it is how the protocol
stays cheap.

---

## Exclusive ownership by task

| Path | Owner |
| ---- | ----- |
| `internal/agents/*.go` (core) | T-001 only |
| `internal/agents/scan/hometree/` | T-001 |
| `internal/agents/scan/cliquery/` | T-001 |
| `internal/agents/scan/embeddeddb/` | T-031 |
| `internal/agents/scan/projectfiles/` | T-033 |
| `internal/agents/conformance/` | T-004 |
| `internal/agents/probe/` | T-005 |
| `internal/agents/catalog/<key>.go` | that agent's task, exclusively |
| `internal/agents/sources/<key>/` | that agent's task, exclusively |
| `testdata/sessionindex/<key>/` | that agent's task, exclusively |
| `testdata/handoff/<key>/` | that agent's task, exclusively |
| `docs/session-storage/<key>.md` | that agent's task, exclusively |
| `docs/testing/results/agent-probes/*-<key>.json` | that agent's task |
| `internal/cli/doctor.go` agent surface | T-006 |
| `scripts/testing/agent-storage-probe.*` | T-006 |
| `docs/compatibility.md` | T-010, then append-only (see below) |
| `internal/doctest/agents_contract_test.go` | T-011 |
| `docs/session-storage-map.md` | T-012 |
| `website/**` | W6 tasks only |
| `CHANGELOG.md` | append-only (see below) |
| `ROADMAP.md`, `docs/adr/`, `docs/agent-support-tiers.md`, `docs/adapters/agent-catalog-sdk.md` | coordinator only |

The three files under the last row are the plan itself. If an executor believes
one is wrong, that is an escalation, not an edit.

---

## Shared files, and the rule that keeps them safe

Two files cannot be split, because they are single tables every agent must
appear in:

- `docs/compatibility.md` — the agent matrix
- `CHANGELOG.md` — the `[Unreleased]` list

The rule for both:

1. **Append only.** Add your row or your bullet. Never reformat, re-sort,
   re-wrap, or "tidy" the surrounding lines.
2. **One line per task**, added at the documented insertion point.
3. **Rebase, never merge.** A conflict on a single appended line resolves in
   seconds; a conflict on a reflowed table does not.
4. If your change needs a **new column** in the matrix, stop. That is a T-010
   change, and it must land before your row.

Point 4 is the common failure: an executor adds a column for their agent's
special case, and every other open PR conflicts.

---

## Files no agent task may touch

Not because they are precious, but because touching them means the task has
grown outside its shape:

| Path | Why |
| ---- | --- |
| `internal/crypto/` | No agent below T5 has any reason to |
| `internal/sync/` | No new synced agents in `v0.5.0` |
| `internal/capsule/` | Capsule format changes are a separate, reviewed decision |
| `internal/handoff/target_*.go` | No new destinations in `v0.5.0` |
| `internal/exitcode/` | No new exit codes in this phase |
| `.github/workflows/` | CI changes are coordinator work |
| `website/**` | Website is W6, after code |
| Another agent's `catalog/`, `sources/`, `testdata/`, or storage page | Obvious, and it happens anyway |

If a task genuinely requires one of these, that is an escalation with a
proposed alternative, not a quiet commit.

---

## Sequencing constraints that ownership alone does not express

| Constraint | Reason |
| ---------- | ------ |
| T-002 and T-003 run after T-001, in order, same executor | They are one refactor split for reviewability |
| T-031 before T-032 | Cline establishes the F3 scanner and host attribution; Roo reuses it |
| All agent tasks after T-004 and T-006 | They need conformance and the probe to produce evidence |
| W6 after every code task merges | The website asserts tiers that are not final until then |
| T-010 before any agent adds a matrix row | Column shape must be settled first |

---

## When two tasks want the same file

The answer is never "coordinate carefully in chat". It is one of:

1. **Split the file** so each task owns a piece. This is why per-agent storage
   pages exist instead of one map.
2. **Serialize the tasks** and record it in the sequencing table above.
3. **Give the file to the coordinator**, who applies both changes after the
   tasks merge.

Choose one explicitly and write it down. An undocumented shared file is a
merge conflict with a delay fuse.
