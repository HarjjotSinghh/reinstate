# Cursor CLI

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Cursor already appears as a row in
[../compatibility.md](../compatibility.md)'s Phase 2 capability matrix without
a shipped reader. Phase 5 either gives it one or records why it cannot have
one; leaving it half-named is the worst of the three outcomes.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Anysphere |
| Product | Cursor CLI (agent) |
| Binary | `cursor-agent` (unconfirmed) |
| Distribution | Official |
| Storage family | F3 expected (unconfirmed) |

## The identity question comes first

Cursor is two things: an editor with an in-app agent, and a terminal agent.
They may or may not share session storage. The task must decide which product
the catalog key `cursor` refers to **before** probing, and record that decision
here. If both have local history and they are separate, they are two catalog
entries with two keys, not one blurred entry.

## Working hypothesis, not evidence

Editor-side chat state is commonly reported to live in the editor's own state
database rather than in plain files. If that also covers the terminal agent,
Cursor is an F3 agent, the scanner opens the database read-only with an
immutable pragma, and an unrecognized schema version fails closed.

Nothing here is confirmed.

## What the probe must settle

1. Which product the key refers to, and whether the two share storage.
2. The storage location on macOS and native Windows.
3. Whether it is a database or plain files. If a database, the schema version
   marker and the table carrying turns.
4. Whether the workspace path is recorded per session.
5. Whether a documented resume or session-list command exists on the terminal
   agent. A supported list command would move Cursor from F3 to F2 and remove
   the need to read private storage at all.
6. Whether reading the database while the editor is running is safe. If the
   editor holds an exclusive lock, discovery must degrade to a clear
   "close the editor" message rather than an unexplained error.

## Sources

None verified. Establish and record vendor sources before promoting any row.
