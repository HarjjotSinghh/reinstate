# Cline

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Cline is an editor extension rather than a terminal CLI, which makes it the
first F3 agent in the catalog. Its sibling [Roo Code](roo.md) shares the same
architecture and should be implemented immediately after it, reusing whatever
the Cline task establishes.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Cline |
| Product | Cline (editor extension) |
| Host | VS Code and compatible editors |
| Distribution | Official, extension marketplace |
| Storage family | F3 (editor extension storage) expected |

## Working hypothesis, not evidence

Extension state is commonly reported to live under the host editor's
per-extension global storage directory, with one directory per task holding
JSON files for the API conversation and the rendered UI messages. None of that
is confirmed by a source Reinstate has verified, and it varies by editor host.

## Why F3 changes the design

1. **Multiple hosts, multiple roots.** VS Code, VS Code Insiders, VSCodium,
   Cursor, and Windsurf each keep separate extension storage. The descriptor's
   `Roots` must enumerate the hosts the probe confirms, and the probe must run
   against each host the tester has installed. A record must record which host
   it came from, or two hosts' sessions become indistinguishable.
2. **No `PATH` executable.** There is no binary to probe for a version, so the
   version gate is the extension's own version marker if one exists on disk.
   This caps the agent at T1 or T2 until a version source is found.
3. **No resume argv.** An editor extension is resumed by opening the editor,
   not by a command line. T3 is therefore not reachable through the current
   launch mechanism, and the descriptor should say so rather than leaving it
   pending forever.
4. **Concurrent writers.** The editor may be running while Reinstate scans.
   Reads are read-only, take no lock, and tolerate a partially written file by
   taking the last complete record.

## What the probe must settle

1. The storage root for each installed host, on macOS and native Windows.
2. The per-task directory shape and whether a task ID is stable.
3. Which file carries user-visible turns, and which is UI-render state. These
   must not both be parsed, or every turn is duplicated.
4. Whether the workspace or repository path is recorded anywhere. Without it,
   project attribution is impossible and search quality is capped.
5. Whether a version marker exists on disk.
6. Whether an export or history command exists in the extension's own surface,
   which would be preferable to reading its private files.

## Sources

None verified. Establish and record vendor sources before promoting any row.
