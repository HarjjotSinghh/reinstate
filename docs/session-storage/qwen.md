# Qwen Code (Alibaba)

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T2

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Alibaba |
| Product | Qwen Code |
| Binary | `qwen` (unconfirmed) |
| Distribution | unconfirmed |
| Storage family | F1 expected (unconfirmed) |

## Working hypothesis, not evidence

Qwen Code is commonly described as a fork of Gemini CLI. If that is accurate,
its storage would mirror the Gemini layout already documented in
[../session-storage-map.md](../session-storage-map.md) section 3: a home root,
a `tmp/<project-hash>/chats/` subtree, JSONL records with `$set` metadata and
`$rewindTo` rewind records, and a project hash derived from the project root
path.

**This is a hypothesis to test, not a row to ship.** A fork that diverged on
storage produces a scanner that silently finds nothing. Treat the Gemini reader
as a starting point for the parser only after the probe confirms the shape.

The upside if the hypothesis holds is large: the `$rewindTo` replay logic, the
legacy-JSON versus JSONL handling, and the subagent exclusion are already
implemented and tested in `internal/transcript/gemini.go`, so Qwen becomes a
descriptor plus a thin mapping rather than a new reader.

## Required research before implementation

This agent's first task is source establishment, not coding:

1. Identify the official distribution and repository, and record them here.
2. Confirm the binary name and whether a root environment override exists.
3. Confirm whether the fork relationship extends to the session recording
   service, or only to the CLI surface.
4. Capture macOS and native Windows probes.
5. Determine whether rewind records exist. If they do, the reader must replay
   them before emitting capsule events, exactly as the Gemini reader does,
   or the capsule will contain turns the user discarded.

If steps 1 through 3 cannot be answered from vendor sources, Qwen stays T0 with
reason `unidentified_product` and the task closes as complete.

## Sources

None verified. Populate this section as part of the research task above; a row
in this file without a source here is not permitted to leave `Unverified`.
