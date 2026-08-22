# OpenCode T4 handoff-destination journey — macOS

`AGENT-TIER-JOURNEY-V1`

A single-agent tier-promotion journey, not a release candidate report. It
records the physical evidence gathered on one device for OpenCode as a
**structured handoff destination**, under
[../../agent-support-tiers.md](../../agent-support-tiers.md). Immutable once
merged; corrections are appended as a new report.

A handoff starts a **new** destination session. It is never a cross-agent
resume, and nothing below should be read as one.

Every command ran with `XDG_DATA_HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`,
`XDG_STATE_HOME`, `CLAUDE_CONFIG_DIR` and `REINSTATE_HOME` pointed at throwaway
directories. No real agent tree was written to.

## Verdict

- **Device verdict:** `PASS with a blocking limitation`
- **Platform covered:** `macos-arm64` only
- **Native Windows:** `NOT TESTED HERE`
- **Recommendation:** do not promote OpenCode to T4 until the write-ahead-log
  limitation in section 5 has a maintainer decision. The capability works; the
  lineage reconciliation the tier requires structurally cannot succeed.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `macos-arm64` |
| OS/version/build | `macOS 26.5.2 (25F84)` |
| Vendor version measured | OpenCode `1.18.21` |
| Source agent | a synthetic Claude Code session under a throwaway `CLAUDE_CONFIG_DIR` |
| Destination | real OpenCode against a throwaway `$XDG_DATA_HOME/opencode` |

## 2. New-session mechanics, measured

| Question | Answer | How it was measured |
| -------- | ------ | ------------------- |
| Can Reinstate pin the destination session id? | **No** | `opencode --session <unknown-id>` prints `Session not found: …` and creates no row |
| How is a new session started with an initial prompt? | `opencode --prompt "<text>"` | the vendor created the session, its first message and its parts by itself |
| Does the vendor block on a directory-trust prompt in a fresh workspace? | No | the TUI started straight into a brand-new directory |
| Does the vendor report the new id? | Yes, on exit | it prints `Continue  opencode -s <id>` |

`SupportsPinnedID` is therefore `false` and the destination id is reconciled
after launch, exactly as it is for Codex.

## 3. Journey rows

| Row | Result | Observation |
| --- | ------ | ----------- |
| `H1` destination offered | `PASS` | `--to opencode` is accepted and planned |
| `H2` plan argv | `PASS` | `opencode --prompt "<briefing>"`, cwd equal to the verified workspace |
| `H3` language | `PASS` | every line of CLI output reads "a new opencode session, not native resume" |
| `H4` capsule artifacts | `PASS` | capsule, fidelity, projection and bootstrap written under the private Reinstate handoff store only |
| `H5` warning gate | `PASS` | the launch refused until `baseline.unavailable` and `handoff.capability.attachment.support` were acknowledged by exact id |
| `H6` vendor creates the session | `PASS` | after launch the store held a new session whose `directory` is the verified workspace |
| `H7` briefing lands intact | `PASS` | the new session's first human turn is byte-identical to the planned bootstrap (1129 bytes, same SHA-256) |
| `H8` no vendor files written by Reinstate | `PASS` | `Materialize` writes nothing; unit tests assert the agent root and the workspace are both untouched |
| `H9` lineage recorded before launch | `PASS` | the handoff is listed even when the destination is killed |
| `H10` reconciliation | **`FAIL`** | `destination.state` stayed `unresolved` after a successful launch — see section 5 |
| `H11` uninstalled/unknown destination | `PASS` | a destination whose layout is not recognized is `UNTESTED` and refused with the compatibility exit code unless `--allow-untested` is given |

## 4. Defect found and fixed during this journey

The handoff pipeline substituted its rendered briefing into the destination argv
by switching on the agent name, and for every destination that was not Claude it
replaced the whole argv with the briefing as a single bare element. That is
correct only for a CLI whose initial prompt is a bare positional.

OpenCode's default command reads a bare positional as a **project path**, so the
pipeline silently dropped `--prompt` and planned a launch into a directory named
after the entire briefing. Observed directly: the first dry run emitted
`"args": ["structured handoff, not native resume\n\n## Goal…"]`.

The substitution is now generic: the briefing replaces the target's own
bootstrap wherever the target put it, and a plan that does not carry its own
bootstrap in its argv is refused rather than guessed at.

## 5. Blocking limitation — the vendor's write-ahead log

Reinstate opens the OpenCode store with `mode=ro&immutable=1&_pragma=query_only(1)`.
`immutable=1` is what stops SQLite creating `-wal` and `-shm` files beside a
store Reinstate does not own, and it also tells SQLite to ignore any existing
write-ahead log.

OpenCode journals in WAL mode and **does not checkpoint on exit**. Measured
twice, once quitting the TUI through its own UI and once interrupting it:

| Artifact | Size immediately after the vendor stopped |
| -------- | ----------------------------------------- |
| `opencode.db` | 4096 bytes |
| `opencode.db-wal` | 543872 bytes |
| `opencode.db-shm` | 32768 bytes |

Through the handle Reinstate is allowed to use, that store has **no `session`
table at all**. A plain `mode=ro` handle, which reads the log, lists the session
correctly.

Consequences, in order of importance:

1. **This is not a T4 problem first.** A session the user has just worked in is
   invisible to `rein sessions --agent opencode`, to search, and to resume,
   until SQLite's automatic checkpoint threshold (default 1000 pages, roughly
   4 MB of writes) is crossed by later vendor activity. That affects the
   already-shipped T1 and T2 capabilities and the T3 promotion.
2. **T4's reconciliation cannot succeed.** A handoff destination session is by
   definition brand new, so it is always the newest thing in the log. The tier
   contract allows honest `unresolved` outcomes when the destination id is not
   knowable at launch; here it is *never* knowable, which is a different claim
   from the one T4 makes.
3. The earlier T3 journey passed its reconciliation-adjacent rows because
   `opencode import` had been followed by a read-write `sqlite3` open, which
   checkpointed the log as a side effect. That was accidental, and this report
   corrects the impression.

Every candidate remedy trades against the invariant that Reinstate never writes
under an agent root — copy the database and its log to a temporary directory and
read the copy; open plain read-only and accept an `-shm` file beside the
vendor's store; or accept the limitation and surface it in `rein doctor`. That
is a product decision, not an implementation detail, and it is why this report
recommends against promoting the tier yet.

The behaviour is pinned by
`TestOpenCodeVerifyCannotSeeAnUncheckpointedSession`, which asserts the honest
outcome — unresolved, never a wrong session id and never a failed command.

## 6. Not covered by this report

- **Native Windows.** No row above was run there.
- **A second bidirectional journey.** T4 requires journeys against every
  existing destination; only Claude → OpenCode was exercised here.
- **Destination acknowledgement.** The briefing asks the destination to restate
  five bullets before mutating anything. Whether the model did so was not
  assessed; the acknowledgement is a prompt-level contract, not an enforced
  protocol.
