# W5 — Promotion cards

Moving a shipped agent up the ladder. One executor, after T-004.

---

## T-050 — Gemini CLI, T2 to T3

**Owns:** `internal/agents/catalog/gemini.go`, `testdata/sessionindex/gemini/`,
`docs/session-storage/gemini.md` (or the map section until T-012 moves it)

**Why this agent.** Gemini already has a verified read path
(`internal/sessionindex/gemini.go`), a tested transcript reader with `$rewindTo`
replay, committed fixtures, and a documented `gemini --resume`. The single
thing it lacks is the version probe that T3 requires. It is the cheapest
available proof that the ladder works upward, not only outward.

**Steps.**

1. Capture the `gemini --version` output shape on macOS and native Windows.
   Write the parser and add it to the descriptor's `VersionSpec`.
2. Choose a fail-closed supported range with the maintainer. Record the exact
   minimum and maximum in the descriptor and in `docs/compatibility.md`.
3. Wire the launch path: executable trust resolution, workspace identity
   guards, and process detection, reusing the Phase 3 contract that Claude and
   Codex already use. Write no new launch machinery.
4. Handle project scoping. Gemini's resume is project-scoped, so a resume from
   the wrong working directory must be refused rather than silently resolved.
5. Set `CanResume` and clear `ReadOnlyReason` for Gemini records, and confirm
   `validateNativeAgent` now accepts `gemini` through the catalog rather than
   through a literal.
6. Physical journeys on both platforms: resume a real Gemini session and
   observe the continuation. Record them for the Phase 5 device reports.

**Done when.** Matrix E in the acceptance contract passes for Gemini on both
platforms, and a version outside the range exits `5` naming the range.

**Do not attempt fork.** Confirm whether Gemini documents a fork equivalent. If
it does not, leave `Fork` nil in `NativeSpec` and state that on the page.
Inventing a fork by copying files is a vendor-internal write and is forbidden.

**Escalate if.** The version output is not stable enough to parse reliably, or
the vendor ships unversioned builds. Gemini then stays at T2, which is a
legitimate outcome and not a failure of this task.
