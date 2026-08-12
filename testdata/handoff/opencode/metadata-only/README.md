# OpenCode metadata-only handoff fixture

Simulates an install where `storage/message/` is absent (SQLite-only or
unrecognized layout). Session metadata comes from
`opencode session list --format json` only — no message bodies.

The reader must emit conversation `omitted` with reason
`source_bodies_unavailable` and must not invent a SQL schema.
