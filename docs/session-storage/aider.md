# Aider

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Aider is the roster's only F4 agent: its history is widely reported to live
**inside the repository** rather than under a home root. That makes it the
architectural test case for project-scoped discovery, and it should be
implemented after at least one F1 agent has landed.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Aider community project |
| Binary | `aider` |
| Distribution | Official, open source |
| Storage family | F4 (per-repository files) expected |

## Working hypothesis, not evidence

Aider is commonly reported to append a Markdown chat history and a plain input
history to the working repository, with filenames beginning `.aider.`, and to
support flags that relocate or disable them. Nothing in that sentence is
confirmed by a source Reinstate has verified.

## Why F4 changes the design

Every other agent answers "list my sessions" by walking one home root. Aider
cannot, because its history is scattered across every repository the user has
ever run it in. The consequences the descriptor must handle:

1. **Discovery scope.** The `projectfiles` scanner discovers within known
   projects — the workspaces Reinstate already tracks — not by walking the
   filesystem looking for `.aider.*`. Searching a user's whole disk is a
   privacy hazard and unbounded.
2. **Session identity.** If the history is one appended file per repository,
   there may be no session ID at all, and possibly no session boundary. The
   probe must determine whether distinct runs are separable. If they are not,
   the record is one session per repository, and that must be stated plainly
   rather than faked with synthetic IDs.
3. **Git interaction.** These files may be committed, ignored, or neither.
   Reinstate reads them and never writes, moves, or ignores them.
4. **Markdown is not a transcript format.** A rendered Markdown log is lossy
   by construction. That likely caps Aider at T1 and makes T2 a separate
   decision with an explicit fidelity statement, not an automatic next step.

## What the probe must settle

1. Exact filenames and whether both a chat log and an input log exist.
2. Whether a machine-readable history exists anywhere. A JSON or JSONL
   alternative would change the tier ceiling substantially.
3. Whether session boundaries are recoverable from the file.
4. Whether flags relocate history outside the repository, and whether a global
   default location exists.
5. Whether any documented resume mechanism exists at all. Without one, T3 is
   not reachable and the descriptor states that as a permanent property rather
   than a pending task.

## Sources

None verified. Establish and record vendor sources before promoting any row.
