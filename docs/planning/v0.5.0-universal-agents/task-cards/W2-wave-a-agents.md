# W2 — Wave A agent cards

Three F1 agents, one executor each, fully parallel once W0 has landed.

**Every card follows the same nine-step recipe** in
[../../../adapters/agent-catalog-sdk.md](../../../adapters/agent-catalog-sdk.md)
under "Adding an agent". The cards below record only what is specific to that
agent. Read the recipe first.

**Universal preconditions for every card here:**

- T-004 and T-006 have merged.
- You have the agent installed and have used it in **at least two projects**,
  producing **more than one session**. An agent with one session does not
  exercise ordering, project attribution, or search.
- You have both a macOS machine and a native Windows machine, or a partner who
  runs the other platform's probe.

**Universal definition of done:**

- Storage page rows promoted from `Unverified` with the probe cited, or
  deleted with the contradiction noted.
- Probes committed for macOS and native Windows.
- Synthetic fixtures, never copied from your real tree.
- `conformance.Run` passing.
- Matrix row appended to `docs/compatibility.md`; bullet appended to
  `CHANGELOG.md`.
- Descriptor tier equals the tier the evidence supports, which may be lower
  than the target.

---

## T-020 — Kimi Code CLI

**Target:** T3 · **Owns:** `internal/agents/catalog/kimi.go`,
`internal/agents/sources/kimi/`, `testdata/sessionindex/kimi/`,
`testdata/handoff/kimi/`, `docs/session-storage/kimi.md`

**Page:** [../../../session-storage/kimi.md](../../../session-storage/kimi.md)

This is the flagship of the wave. Assign it to the strongest executor.

**Specific work.**

1. **Settle the root conflict first.** Two vendor mirrors disagree:
   `~/.kimi-code/` versus `~/.kimi/`. Probe both, determine which the running
   binary writes to, and record the finding on the page. Support both as
   ordered candidates only if the probe shows both are real.
2. **Check for `session_index.jsonl` before writing a directory walk.** The
   vendor documents a global index mapping every session ID to its directory
   and working directory. If it exists, it is the cheapest discovery path in
   the entire catalog: one file, every session, every project. Use it as the
   primary path and keep the walk as a fallback for when it is absent or stale.
3. **Decide `wire.jsonl` versus `context.jsonl`.** One mirror describes both.
   Determine which carries user-visible turns. Parsing both duplicates every
   turn.
4. **Exclude `agents/agent-*/`** subagent directories from the top-level
   session list, the same way Claude Code subagents are excluded.
5. **Classify request-trace records as omitted.** The vendor states `wire.jsonl`
   carries tool schemas, request parameters, and MCP tool listings for
   debugging. Do not attempt to interpret them.
6. **Exclude the OAuth credential directory** in the descriptor before any read.
7. For T3: capture `kimi --version` output shape and set a fail-closed range.
   Verify `kimi --continue` and `kimi --session <id>` physically on both
   platforms.

**Escalate if.** The probe shows sessions are stored somewhere neither mirror
documents. That is a significant finding and the page should record it before
any code is written.

---

## T-021 — Pi

**Target:** T3 · **Owns:** `internal/agents/catalog/pi.go`,
`internal/agents/sources/pi/`, `testdata/sessionindex/pi/`,
`testdata/handoff/pi/`, `docs/session-storage/pi.md`

**Page:** [../../../session-storage/pi.md](../../../session-storage/pi.md)

**Specific work.**

1. **Check for a machine-readable session list before parsing files.** Pi
   documents four machine-facing modes: print/JSON, RPC over stdin/stdout, and
   an SDK. If any exposes a session list, Pi is an F2 agent using a supported
   interface, which is strictly better than reading private storage. This is
   the same reasoning that made OpenCode F2. Answer this before writing a
   parser.
2. **Find the real default session directory.** Pi documents
   `PI_CODING_AGENT_DIR` for config and a separate
   `PI_CODING_AGENT_SESSION_DIR` for sessions, which implies the session
   default is not simply the config root.
3. **Determine whether sessions are project-scoped at all.** If they are
   global, set `ProjectKey` to none and derive workspace attribution from the
   record body. Say so on the page rather than inventing a bucket.
4. **Use the vendor's self-identification for process detection.** Pi sets
   `AI_AGENT=pi` and `PI_CODING_AGENT=true` for child processes. Prefer those
   over binary-name heuristics in `ProcessSpec`.
5. **Exclude HTML session exports** from discovery if they land in the session
   tree. An export is not a session.
6. For T3: capture `pi --version` output shape.

**Escalate before promoting to T3.** Pi releases very frequently. A narrow
pinned version range will be stale within weeks, and a wide one weakens the
fail-closed guarantee. The range policy is a maintainer decision.

---

## T-022 — Qwen Code

**Target:** T2 · **Owns:** `internal/agents/catalog/qwen.go`,
`internal/agents/sources/qwen/`, `testdata/sessionindex/qwen/`,
`testdata/handoff/qwen/`, `docs/session-storage/qwen.md`

**Page:** [../../../session-storage/qwen.md](../../../session-storage/qwen.md)

**This card starts as research, not code.** No vendor source has been verified.

**Specific work.**

1. Establish the official distribution, repository, and binary name. Record
   them in the page's Sources section. A row cannot leave `Unverified` without
   a source there.
2. Test the fork hypothesis. Qwen Code is commonly described as a Gemini CLI
   fork. If the session recording service was inherited, the existing Gemini
   reader is nearly reusable: `$rewindTo` replay, legacy-JSON versus JSONL
   handling, and subagent exclusion are already implemented and tested.
3. **Do not assume the hypothesis.** A fork that diverged on storage produces a
   scanner that silently finds nothing, which looks to the user like a
   Reinstate bug. Confirm with a probe.
4. If rewind records exist, replay them before emitting capsule events, exactly
   as `internal/transcript/gemini.go` does. Otherwise the capsule contains
   turns the user discarded.

**Close as T0 if.** The product cannot be identified from official sources.
Set `T0Reason: unidentified_product` and finish. That is a complete task.
