# Gemini CLI

**Confidence: Verified (read path)** — `internal/sessionindex/gemini.go`.
**Documented** for resume semantics and `$rewindTo` on-disk behavior
(R3, 2026-08-12).

| Aspect | Value |
| ------ | ----- |
| Root (override) | `$GEMINI_CLI_HOME` |
| Root (all OSes) | `~/.gemini` / `%USERPROFILE%\.gemini` |
| Sessions | `<root>/tmp/<project-hash>/chats/session-<id>.json` or `.jsonl` |
| Checkpoints | `<root>/tmp/<project-hash>/checkpoints/checkpoint-<name>.json` |
| Format | Legacy: single JSON object with `messages[]`. Current: JSONL with `$set` metadata records and `$rewindTo` rewind records |
| Project scoping | `<project-hash>` is derived from the project root path |
| Subagents | `kind: "subagent"` sessions are excluded |
| Native resume | `gemini --resume` / `-r`; project-scoped |
| Fail-closed version pin | `0.55.1`–`0.55.1` (latest stable `@google/gemini-cli` on 2026-08-16). Nightlies excluded. T3 still needs dual-platform physical resume. |

### Individual OAuth shutdown (2026-06-18) — does not change the tier

Google stopped serving Gemini CLI requests for Gemini Code Assist for
individuals, Google AI Pro, and Google AI Ultra on 2026-06-18. Signing in with
a Google account now fails with:

```
This client is no longer supported for Gemini Code Assist for individuals.
To continue using Gemini, please migrate to the Antigravity suite of products.
```

**T2 stands.** Reinstate reads session files off disk and never authenticates,
so nothing in the read path depends on the retired flow. The binary is still
Apache-2.0 and maintained, and two auth paths still work: a Gemini API key via
`GEMINI_API_KEY`, and Gemini Code Assist Standard or Enterprise licences, which
the deprecation explicitly leaves unchanged.

Two practical consequences:

- Producing new Gemini CLI evidence on a personal machine requires
  `GEMINI_API_KEY` from AI Studio. A Google-account sign-in will not create
  sessions to probe.
- The migration destination is [Antigravity CLI](antigravity.md), which
  installs into `~/.gemini/antigravity-cli/` and copies an existing Gemini CLI
  setup across at install time. **Capture Gemini CLI probe evidence before
  installing it**, and note that the Gemini descriptor now excludes that
  subtree so the two agents do not read each other's files.

### `$rewindTo` (R3 — Documented)

On-disk JSONL is **append-only**: prior message lines stay in the file; a
`{"$rewindTo":"<messageId>"}` record is appended. The **active** conversation
truncates (vendor removes from and including the target id). Phase 4 capsule
readers must replay rewinds **before** emitting canonical events, otherwise
the capsule contains turns the user already discarded.

`internal/transcript/gemini.go` (WP-08) aligns with that vendor cut: the
target id and everything after it are excluded from the capsule. The Phase 2
index reader still uses an inclusive slice for search metadata only.

Synthetic fixtures: `testdata/sessionindex/gemini/{macos,windows}/` and
`testdata/handoff/gemini/{rewind,legacy-json,jsonl}/`.
Research note: [research/2026-08-12-phase-4-r1-r2-r3.md](../research/2026-08-12-phase-4-r1-r2-r3.md).
