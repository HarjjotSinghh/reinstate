# W3 — Wave B agent cards

New storage families. These introduce scanners the catalog does not have, so
they follow Wave A rather than running beside it.

Read the nine-step recipe in
[../../../adapters/agent-catalog-sdk.md](../../../adapters/agent-catalog-sdk.md)
and the universal preconditions in
[W2-wave-a-agents.md](W2-wave-a-agents.md) first. Both apply here unchanged.

**Sequencing:** T-031 before T-032. T-030 and T-033 are independent of both.

---

## Constraints shared by every F3 agent

F3 agents are editor extensions, not terminal binaries. Four consequences the
descriptor must express honestly:

1. **Multiple hosts, multiple roots.** VS Code, Insiders, VSCodium, Cursor, and
   Windsurf keep separate extension storage. Enumerate the hosts the probe
   confirms, and record which host each session came from. Without host
   attribution, two hosts' sessions are indistinguishable.
2. **No executable on `PATH`**, so no version probe. The version gate is an
   on-disk marker if one exists.
3. **No resume argv.** An extension resumes by opening the editor. T3 is not
   reachable through the current launch mechanism. Write that as a property of
   the agent, not as a pending task, so nobody re-opens it every release.
4. **A concurrent writer.** The editor may be running during a scan. Read-only,
   no locks, tolerate a partial trailing record.

---

## T-030 — Cursor CLI

**Target:** T1 · **Owns:** `internal/agents/catalog/cursor.go`,
`internal/agents/sources/cursor/`, `testdata/sessionindex/cursor/`,
`docs/session-storage/cursor.md`

**Page:** [../../../session-storage/cursor.md](../../../session-storage/cursor.md)

**Decide the identity question before probing.** Cursor is an editor with an
in-app agent and a terminal agent. They may not share storage. Decide what the
catalog key `cursor` refers to and record the decision on the page. If both
have local history and they are separate, they are two catalog entries with two
keys, not one blurred entry.

Cursor already appears as a row in `docs/compatibility.md`'s Phase 2 matrix
without a shipped reader. This task either gives it one or records why it
cannot have one. Leaving it half-named is the worst outcome.

**Specific work.**

1. Resolve identity, then probe on macOS and native Windows.
2. If storage is a database, open read-only with an immutable pragma, gate on a
   schema version marker, and fail closed on an unrecognized version.
3. Check whether the terminal agent documents a session-list command. That
   would make Cursor F2 and remove the need to read private storage entirely.
4. Determine whether reading while the editor runs is safe. If the editor holds
   an exclusive lock, degrade to a clear message telling the user to close it,
   not to an unexplained error.

---

## T-031 — Cline

**Target:** T1 · **Owns:** `internal/agents/catalog/cline.go`,
`internal/agents/sources/cline/`, `internal/agents/scan/embeddeddb/`,
`testdata/sessionindex/cline/`, `docs/session-storage/cline.md`

**Page:** [../../../session-storage/cline.md](../../../session-storage/cline.md)

**This task owns the F3 scanner for the whole catalog.** Build it for reuse:
multi-host root resolution, host attribution on each record, read-only access,
and schema-version gating all belong in `scan/embeddeddb`, not in the Cline
source package.

**Specific work.**

1. Probe every editor host you have installed, on macOS and native Windows.
2. Determine the per-task directory shape and whether the task ID is stable
   across restarts.
3. Determine which file carries user-visible turns and which is UI-render
   state. Parsing both duplicates every turn.
4. Determine whether the workspace path is recorded. Without it, project
   attribution is impossible and search quality is capped; say so on the page.
5. Look for an on-disk version marker.
6. Check whether the extension exposes its own export or history surface,
   which would be preferable to reading private files.

---

## T-032 — Roo Code

**Target:** T1 · **Depends on:** T-031 · **Owns:**
`internal/agents/catalog/roo.go`, `internal/agents/sources/roo/`,
`testdata/sessionindex/roo/`, `docs/session-storage/roo.md`

**Page:** [../../../session-storage/roo.md](../../../session-storage/roo.md)

Reuse `scan/embeddeddb` from T-031 without modifying it. If it needs a change,
that is a T-031 follow-up, coordinated, not a parallel edit.

**Answer the six probe questions independently.** A shared origin with Cline
does not guarantee a shared current layout, and assuming it is exactly the
mistake that produces a scanner which silently finds nothing.

**Additionally:** determine whether Roo and Cline can collide in the same
storage tree. If a user has both, every record must be attributed to the
correct product.

---

## T-033 — Aider

**Target:** T1 · **Owns:** `internal/agents/catalog/aider.go`,
`internal/agents/sources/aider/`, `internal/agents/scan/projectfiles/`,
`testdata/sessionindex/aider/`, `docs/session-storage/aider.md`

**Page:** [../../../session-storage/aider.md](../../../session-storage/aider.md)

**This task owns the F4 scanner.** Aider's history lives inside repositories,
not under a home root, which inverts discovery.

**Specific work.**

1. **Discover within known projects only** — the workspaces Reinstate already
   tracks. Walking the filesystem hunting for history files is forbidden: it is
   a privacy hazard, unbounded, and produces false positives.
2. Determine the exact filenames and whether both a chat log and an input log
   exist.
3. **Determine whether session boundaries are recoverable.** If distinct runs
   are not separable inside one appended file, the record is one session per
   repository. State that plainly; do not fabricate session IDs to make the
   data model fit.
4. Look for a machine-readable history. A JSON or JSONL alternative changes the
   tier ceiling substantially.
5. Read only. Never write, move, or gitignore these files. They may be
   committed, ignored, or neither, and none of that is Reinstate's business.
6. Determine whether any documented resume mechanism exists. If none does, T3
   is permanently unreachable and the descriptor says so.

**A rendered Markdown log is lossy by construction.** Aider T2 is a separate
decision with an explicit fidelity statement, not an automatic next step. Do
not promote it in this task.
