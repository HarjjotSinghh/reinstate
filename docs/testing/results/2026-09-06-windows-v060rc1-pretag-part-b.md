# `v0.6.0-rc.1` pre-tag Windows acceptance — Part B (T2–T5 agent rows)

`PHASE5-DEVICE-REPORT-V1` (per-agent excerpt) — executor **W7 executor B**,
covering Matrix C/D/E rows for `claude`, `codex`, `opencode`, `grok`, `qwen`
(T4/T5, 17 rows each) and `gemini`, `kimi` (T2, 11 rows each), per
[`v0.6.0-rc.1-agent-verification-prompts.md`](../v0.6.0-rc.1-agent-verification-prompts.md)
and the [Phase 5 contract](../phase-5-universal-agent-coverage-acceptance.md).
This is a **partial** device report: it covers only the rows assigned to this
executor. It does not stand alone as a Phase 5 verdict — see
[Scope and status](#scope-and-status).

## Artifact identity

| Field | Value |
| ----- | ----- |
| Commit under test | `57c15d5225025150ed389a0923cf633b6b227302` (`v060/w7-matrix` branch tip, identical to `release/v0.6.0-rc.1`) |
| Archive | `reinstate_0.0.0-57c15d52_windows_amd64.zip` |
| Archive SHA-256 | `d58b9a46aeb32f32de01b1472597442afa4d1e010d826edd4e1c178011dc916a` (verified against `checksums.txt` with `certutil -hashfile`, matched) |
| Installed binary check | `rein.exe` and `reinstate.exe` byte-identical (`cmp` clean; both SHA-256 `27316b41f4766c694bf49c8aa73c533960e9d4d90c94f717af610e72801fecfa`), unzipped fresh into an executor-owned directory, never a developer or shared binary |
| `rein version --json` | `{"commit":"57c15d5225025150ed389a0923cf633b6b227302","date":"2026-09-06T01:36:16Z","name":"reinstate","version":"0.0.0-57c15d52"}` |
| Install directory | `D:\ReinstateAcceptanceProjects\v060-w7-b\install\` (executor-owned, fresh) |
| Host | `windows-amd64`, native (never WSL); Microsoft Windows 11 Pro 10.0.26200 |
| Git version | `git version 2.52.0.windows.1` |
| Go version (host default) | `go1.26.1`; worktree build/toolchain pin `go1.25.13` via `GOTOOLCHAIN` |
| UTC date | 2026-09-06 |
| Report branch | `v060/w7-matrix` |
| Host contamination check | `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR XDG_DATA_HOME` run in every shell before any `rein` invocation, confirmed empty before each block below |

## Scope and status

- **Rows covered:** Matrix C (C1–C6) for `claude`, `codex`, `opencode`, `grok`,
  `qwen`, `gemini`, `kimi`; Matrix D (D1–D5) for all seven agents (`gemini`
  and `kimi` as handoff **source**, `--to claude`, per their T2 tier — they
  are never valid `--to` destinations); Matrix E (E1–E6) for the five T4/T5
  agents, **partially** (see below). Total assigned rows: **107**
  (5 × 17 + 2 × 11), matching the generated acceptance matrix's per-agent
  counts for these seven keys.
- **Required-row counts:** `70 PASS / 2 FAIL / 1 PARTIAL / 34 NOT TESTED`
  (sums to 107). Plus **3 additional rows beyond the 107** — the T5 sync
  round trip this executor's assignment also calls for — all `NOT TESTED`
  (see [Section 4](#4-t5-encrypted-sync-round-trip-claude-codex-opencode)).
  Per the contract, `PARTIAL` and `NOT TESTED` do not pass a required row, so
  **35 of 107** assigned rows do not pass.
- **This report does not authorize `v0.6.0-rc.1` acceptance by itself.** A
  large fraction of the highest-risk rows — physical vendor resume (E1/E2),
  fork (E3), active-session detection (E5), and the T5 encrypted-sync round
  trip for `claude`/`codex`/`opencode` — are `NOT TESTED` here; see
  [Section 5](#5-not-tested-and-why) for exactly why and what would close
  each gap. `PARTIAL` and `NOT TESTED` do not pass a required row per the
  contract; this executor is reporting that plainly rather than presenting an
  artificially complete matrix.
- **Part A** (section A automated gates, generated-matrix count reconciliation,
  and any other executor's per-agent rows) is a separate file this report does
  not supersede or duplicate.

## Method

Every fixture below is **synthetic**, drawn from `testdata/sessionindex/*` and
`testdata/adapters/*` already committed to this worktree, per
`CLAUDE.md`'s instruction to use only those trees or temporary synthetic
fixtures — this device's real per-agent config directories were **not**
enumerated or read as a session/session-content source anywhere in this
report. Each agent's single-project fixture was duplicated under a second
synthetic project path (with a distinct id/cwd inside the copy) so Matrix C's
"at least two distinct projects" requirement could be exercised for real
rather than assumed. Each command ran with **all seven** agent root
environment variables (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`,
`GROK_HOME`, `QWEN_HOME`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`) explicitly set
— the six not under test pointed at empty throwaway directories, and only the
target agent's variable pointed at its fixture — plus a fresh, agent-specific
`REINSTATE_HOME` per command group, so every listing reflects a rebuilt index
rather than a replayed one and no command could silently fall back to an
ambient default. For Matrix D, two of the five agents' fixtures also had
their recorded `cwd` repointed at a real, throwaway git repository
(`D:\ReinstateAcceptanceProjects\v060-w7-b\repo`) created solely for this
report, because handoff's compatibility gate correctly refuses to run from a
different repository than the one the source session recorded (see
[Reading a refusal correctly](../v0.6.0-rc.1-agent-verification-prompts.md)) —
this is the documented refusal firing correctly, not a defect.

**A harness incident during setup, disclosed rather than hidden:** the very
first `rein doctor --agents --json` call in this run was made before any
per-agent root environment variable was overridden, and this shell's ambient
`CLAUDE_CONFIG_DIR` (inherited from the orchestrating Claude Code process
itself, not a value this executor set) pointed at a real, non-synthetic
Claude Code data directory containing genuine job/workflow/plugin history.
The probe's JSON output — including real file and skill names — briefly
entered this executor's context before the mistake was recognized. No content
from that output was copied into any file this report references, no example
name from it is reproduced anywhere in this report, and the output file was
deleted immediately. Every subsequent command in this report explicitly set
all seven root environment variables before every single invocation, with the
non-target six pointed at empty throwaway directories, specifically to
prevent a repeat. This is recorded as a harness/methodology defect this
executor caused and corrected, not a product defect — see
[Section 6](#6-harness-and-methodology-findings).

---

## 1. Matrix C — Per T1+ agent (`claude`, `codex`, `opencode`, `grok`, `qwen`, `gemini`, `kimi`)

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `C1:claude` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent claude --json` → 2 sessions, projects `demo`/`demo-second`, distinct `id`s |
| `C1:codex` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent codex --json` → 2 sessions, projects `demo`/`demo-second` |
| `C1:opencode` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent opencode --json` → 2 sessions (hydrated `opencode.db` seeded from `testdata/adapters/opencode/windows/store.sql` + one added project/session row), projects `demo`/`demo-second` |
| `C1:grok` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent grok --json` → 2 sessions, projects `demo`/`demo-second` |
| `C1:qwen` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent qwen --json` → 2 sessions, projects `demo`/`demo-second` |
| `C1:gemini` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent gemini --json` → 2 sessions, projects `demo`/`demo-second` (fixture duplicated with a distinct `sessionId`/`projectHash`/`directories` to make the second project genuinely distinct, not a copy sharing one id) |
| `C1:kimi` | Lists sessions from ≥2 distinct projects | PASS | `rein sessions --agent kimi --json` → 2 sessions, projects `demo`/`demo-second` (same distinct-id fixture treatment as gemini) |
| `C2:claude` | Project/branch/title/timestamp/message_count match the source | PASS | both records: `branch=fixture/windows`, `updated_at` matches fixture timestamp, `message_count=1` matches 1 line per fixture file |
| `C2:codex` | as above | PASS | both records: `message_count=1`, `updated_at` matches rollout file mtime, `project` derived from `cwd` basename |
| `C2:opencode` | as above | PASS | both records: `message_count=2` (matches 2 seeded `message` rows per session), titles match `session.title` column |
| `C2:grok` | as above | PASS | both records: `message_count=2` matches `chat_history.jsonl` line count, `updated_at` matches `summary.json.updated_at` |
| `C2:qwen` | as above | PASS | both records: `message_count=3` matches 3 turns (user/model/tool_result excluding the trailing model reply is counted per the adapter's own rule), `branch` not exposed at this level (qwen has no branch field), timestamps match |
| `C2:gemini` | as above | PASS | both records: `message_count=2`, `updated_at` matches `lastUpdated` |
| `C2:kimi` | as above | PASS | both records: `message_count=2`, `updated_at` matches `state.json.updatedAt`, title matches `state.json.title` |
| `C3:claude` | `rein search` finds a known string by prompt | PASS | `rein search "Synthetic Claude Windows fixture request" --agent claude --json` → 2 hits, both fixture sessions |
| `C3:codex` | as above | PASS | `rein search "Synthetic Codex Windows fixture request" --agent codex --json` → 2 hits |
| `C3:opencode` | as above (by message-body text) | **FAIL (documented, non-regression)** | `rein search "Synthetic OpenCode fixture request" --agent opencode --json` → 0 hits. Consistent with the documented, pre-existing behaviour recorded in `docs/testing/results/2026-09-06-windows-range-widening-v060.md`: OpenCode's `SearchText` is built from id/title/project/workspace/branch only, never message-body text. Re-tested here by title instead: `rein search "Synthetic OpenCode fixture session" --agent opencode --json` → 2 hits, both fixture sessions (**PASS by title**). Recorded as a known, pre-existing gap, not a new regression — but it does mean C3's literal text ("finds a known string from a real session by prompt") is not met for OpenCode as written, only by title. |
| `C3:grok` | as above | PASS | `rein search "Synthetic Grok Windows user prompt" --agent grok --json` → 2 hits |
| `C3:qwen` | as above | PASS | `rein search "Map the Windows dest argv" --agent qwen --json` → 2 hits |
| `C3:gemini` | as above | PASS | `rein search "Gemini Windows rewind fixture" --agent gemini --json` → 1 hit (fixture title string; second project not touched by this term since only the first project's fixture text was reused for its title) |
| `C3:kimi` | as above | PASS | `rein search "Remap a Windows workspace" --agent kimi --json` → 1 hit |
| `C4:claude` | `rein inspect` bounded metadata, no transcript body | PASS | `rein inspect claude:claude-syn-windows --json` → 5,478 bytes; `prompt_preview` 40 chars (a preview, not the full record); no raw message array present |
| `C4:codex` | as above | PASS\* | `rein inspect codex:rollout-syn-001 --json` → 95,669 bytes, of which 68,304 bytes is an `environment.capabilities.items` array. This is **not** transcript body (no message content appears), but see the capability-scan finding in [Section 6](#6-harness-and-methodology-findings) — the large size is real ambient host data, not fixture data, and is not reproduced here. |
| `C4:opencode` | as above | PASS | `rein inspect opencode:ses_fixture001 --json` → 5,013 bytes; no `capabilities.items` growth observed for this agent |
| `C4:grok` | as above | PASS | `rein inspect grok:01987654-3210-7890-abcd-ef0123456790 --json` → 5,536 bytes |
| `C4:qwen` | as above | PASS | `rein inspect qwen:01912345-6789-7abc-def0-123456789abc --json` → 5,620 bytes |
| `C4:gemini` | as above | PASS | `rein inspect gemini:gemini-rewind-win --json` → 5,715 bytes |
| `C4:kimi` | as above | PASS | `rein inspect kimi:session_01912345-6789-7abc-def0-123456789abc --json` → 5,890 bytes |
| `C5:claude` | T3+: native resume supported by design; row does not apply below T3 | PASS | `capabilities.resume=true`; `rein resume --dry-run --json` reaches `confirmation_required`, never a read-only refusal |
| `C5:codex` | as above | PASS | same pattern; `capabilities.resume=true` |
| `C5:opencode` | as above | PASS | same pattern; `capabilities.resume=true` |
| `C5:grok` | as above | PASS | same pattern; `capabilities.resume=true` |
| `C5:qwen` | as above | PASS | same pattern; `capabilities.resume=true` |
| `C5:gemini` | T2 (below T3): read-only reason + resume refused exit 5 | PASS | `rein resume gemini:gemini-rewind-win --dry-run --json` → exit `5`; `{"code":"compatibility","message":"native session action is unsupported: Gemini CLI sessions are read-only in Phase 2"}` |
| `C5:kimi` | as above | PASS | `rein resume kimi:session_01912345-6789-7abc-def0-123456789abc --dry-run --json` → exit `5`; `"...Kimi Code CLI sessions are read-only until a device journey verifies native resume"` |
| `C6:claude` | Absent root / empty root / corrupted final record degrade cleanly, no panic, no partial record | PASS | absent root: `rc=0`, `{"sessions":[]}`; empty root: `rc=0`, `{"sessions":[]}`; corrupted copy (last 20 bytes of `session-syn-002.jsonl` truncated): `rc=0`, 1,040-byte output (surviving session still listed, corrupted one silently dropped, no crash) |
| `C6:codex` | as above | PASS | absent/empty: `rc=0`, empty list; corrupted (`rollout-syn-002.jsonl` truncated): `rc=0`, 1,029-byte output |
| `C6:opencode` | as above | PASS\* | absent/empty: `rc=0`, empty list; corrupted (raw truncation of the `opencode.db` file itself, not one JSON record — no per-record truncation primitive exists in a SQLite file): `rc=0`, 209-byte output, no crash. Weaker evidence than the other agents' line-level truncation; a corrupted-row-inside-a-valid-db case was not separately constructed. |
| `C6:grok` | as above | PASS | absent/empty: `rc=0`, empty list; corrupted (`summary.json` truncated): `rc=0`, 657-byte output |
| `C6:qwen` | as above | PASS | absent/empty: `rc=0`, empty list; corrupted (chat `.jsonl` truncated): `rc=0`, 1,134-byte output |
| `C6:gemini` | as above | PASS | absent/empty: `rc=0`, empty list; corrupted (`session-rewind.jsonl` truncated): `rc=0`, 911-byte output |
| `C6:kimi` | as above | PASS | absent/empty: `rc=0`, empty list; corrupted (`wire.jsonl` truncated): `rc=0`, 901-byte output |

\* See [Section 6](#6-harness-and-methodology-findings) for the caveat named in each starred row.

## 2. Matrix D — Per T2+ agent (all seven)

The five T4/T5 agents are handoff sources with a full destination choice;
`claude`'s own source row uses `codex` as destination (a source cannot
meaningfully hand off to itself). `gemini` and `kimi` are T2 — handoff
**source** only, never a valid `--to` destination (`rein handoff --help`
lists exactly `claude|codex|grok|opencode|qwen`) — so their D-rows below use
`--to claude` with `gemini`/`kimi` as the source, exactly as Matrix D's own
definition names T2+ (not only T4+) as in scope, and as the generated
acceptance matrix confirms (`gemini`/`kimi` each carry `D1`–`D5` in their
11-row set alongside `C1`–`C6`).

### 2a. `claude`, `codex`, `opencode`, `grok`, `qwen`

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `D1:claude` | `--dry-run` handoff produces a capsule + fidelity report | PASS | `rein handoff claude:claude-syn-windows --to codex --dry-run --json` (run from a throwaway real git repo matching the fixture's `cwd`) → `rc=0`, capsule with `fidelity`, `parse`, `destination.args`, `workspace` |
| `D1:codex` | as above | PASS | `rein handoff codex:rollout-syn-001 --to claude --dry-run --json` → `rc=0`, 31,975-byte capsule (large for the same ambient-skill-scan reason as `C4:codex`, not transcript content) |
| `D1:opencode` | as above | **FAIL** | `rein handoff opencode:ses_fixture001 --to claude --dry-run --json` → exit `5`; `{"code":"compatibility","message":"handoff: compatibility: source agent \"opencode\" is NOT_INSTALLED"}`, despite `opencode.exe 1.18.27` being on `PATH` and every Matrix C row above succeeding against the same isolated `XDG_DATA_HOME`. Root cause not isolated within this run's time budget — candidates considered but not confirmed: the fixture's `session.version` column (`1.18.21`, from the committed seed) versus the installed `1.18.27`; a handoff-specific executable probe that differs from the one `doctor`/`sessions` use. **This blocks D2–D5 for `opencode` below** (no capsule was ever produced to inspect). |
| `D1:grok` | as above | PASS | `rein handoff grok:01987654-3210-7890-abcd-ef0123456790 --to claude --dry-run --json` (repointed fixture `cwd` to the throwaway repo) → `rc=0`, 5,307-byte capsule |
| `D1:qwen` | as above | PASS | `rein handoff qwen:01912345-6789-7abc-def0-123456789abc --to claude --dry-run --json` (repointed fixture `cwd`) → `rc=0`, 5,673-byte capsule |
| `D2:claude` | Capsule contains no content the source didn't, no invented turn | PASS | `fidelity.components` shows only `exact`/`normalized`/`omitted` (never a silently-filled field); `exact` components' byte counts (40) match the fixture's single 40-character user line |
| `D2:codex` | as above | PASS | same pattern against the codex capsule's `fidelity.components` |
| `D2:opencode` | as above | NOT TESTED | blocked by `D1:opencode` |
| `D2:grok` | as above | PASS | same pattern |
| `D2:qwen` | as above | PASS | same pattern |
| `D3:claude` | Unknown records `referenced`/`omitted` with a reason, never guessed | PASS | `fidelity.components` entries `constraints`, `decisions`, `rejected_approaches` are `"portability":"omitted"` with explicit `"reason":"requires_optional_summarizer"`; `pending` is `omitted` with `"reason":"interrupted_not_replayed"` |
| `D3:codex` | as above | PASS | same shape observed in the codex capsule |
| `D3:opencode` | as above | NOT TESTED | blocked by `D1:opencode` |
| `D3:grok` | as above | PASS | same shape |
| `D3:qwen` | as above | PASS | same shape |
| `D4:claude` | Truncated source: boundary at last complete record, offset + hash recorded | PARTIAL | Attempted against a corrupted copy of the (single-record) claude fixture; truncating the only record destroyed the session entirely (`session not found`) rather than exercising a mid-stream boundary — a clean refusal, not a crash, but this single-turn fixture cannot demonstrate a "boundary at the last **complete** record" when there is only one record. Not re-attempted with a multi-record fixture in this run's time budget. |
| `D4:codex` | as above | NOT TESTED | same fixture-size limitation as claude; not attempted for codex given the claude result already showed the limitation |
| `D4:opencode` | as above | NOT TESTED | blocked by `D1:opencode` |
| `D4:grok` | as above | NOT TESTED | not attempted (time) |
| `D4:qwen` | as above | NOT TESTED | not attempted (time) |
| `D5:claude` | Two runs over an unchanged source produce byte-identical capsules | PASS | two `--dry-run` runs compared field-by-field; identical except `handoff_id`/`lineage_root` (fresh instance identifiers minted per invocation — the capsule content itself, `fidelity`, `parse`, `destination`, `workspace`, is byte-identical) |
| `D5:codex` | as above | PASS | same comparison, same result |
| `D5:opencode` | as above | NOT TESTED | blocked by `D1:opencode` |
| `D5:grok` | as above | PASS | same comparison, same result |
| `D5:qwen` | as above | PASS | same comparison, same result |

### 2b. `gemini`, `kimi` (source; destination fixed to `claude`)

An earlier pass of this run's fixture setup left both agents' recorded `cwd`
mangled by a shell-quoting error (a literal control byte in place of part of
the path), which made the workspace genuinely absent on disk and blocked
`D1` with `workspace.available: missing` / `agent.executable: missing`. That
was a fixture-authoring defect in this run, not a product defect — it is
disclosed and corrected below rather than silently fixed and hidden; the
first (broken) attempt is not counted as a row result.

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `D1:gemini` | `--dry-run` handoff produces a capsule + fidelity report | PASS | `rein handoff gemini:gemini-rewind-win --to claude --dry-run --json` (fixture `directories` repointed to the throwaway real repo) → `rc=0`, 4,076-byte capsule with `fidelity`, `parse`, `destination.args`, `workspace` |
| `D1:kimi` | as above | PASS | `rein handoff kimi:session_01912345-6789-7abc-def0-123456789abc --to claude --dry-run --json` (fixture `cwd` repointed) → `rc=0`, 4,716-byte capsule |
| `D2:gemini` | Capsule contains no content the source didn't, no invented turn | PASS | `fidelity.components` shows only `exact`/`normalized`/`omitted`, never a silently-filled field |
| `D2:kimi` | as above | PASS | same pattern |
| `D3:gemini` | Unknown records `referenced`/`omitted` with a reason, never guessed | PASS | `constraints`/`decisions` omitted with `"reason":"requires_optional_summarizer"`; `pending` omitted with `"reason":"interrupted_not_replayed"` |
| `D3:kimi` | as above | PASS | `constraints`/`decisions` omitted with `"reason":"requires_optional_summarizer"`; `metadata` omitted with `"reason":"harness_meta_record"` |
| `D4:gemini` | Truncated source: boundary at last complete record, offset + hash recorded | NOT TESTED | same single/few-record fixture-size limitation noted under `D4:claude`; not attempted given time |
| `D4:kimi` | as above | NOT TESTED | as above |
| `D5:gemini` | Two runs over an unchanged source produce byte-identical capsules | PASS | two `--dry-run` runs compared field-by-field; identical except `handoff_id`/`lineage_root` |
| `D5:kimi` | as above | PASS | same comparison, same result |

## 3. Matrix E — Per T3 agent (`claude`, `codex`, `opencode`, `grok`, `qwen`)

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `E1:claude` | `rein resume` launches the vendor CLI; real session continues | NOT TESTED | see [Section 5](#5-not-tested-and-why) |
| `E1:codex` | as above | NOT TESTED | as above |
| `E1:opencode` | as above | NOT TESTED | as above |
| `E1:grok` | as above | NOT TESTED | as above |
| `E1:qwen` | as above | NOT TESTED | as above |
| `E2:claude` | Resumed session verified as the requested one, inside the agent | NOT TESTED | as above |
| `E2:codex` | as above | NOT TESTED | as above |
| `E2:opencode` | as above | NOT TESTED | as above |
| `E2:grok` | as above | NOT TESTED | as above |
| `E2:qwen` | as above | NOT TESTED | as above |
| `E3:claude` | `rein fork` produces a distinct session where supported | NOT TESTED | as above |
| `E3:codex` | as above | NOT TESTED | as above |
| `E3:opencode` | as above | NOT TESTED | as above |
| `E3:grok` | as above | NOT TESTED | as above |
| `E3:qwen` | as above | NOT TESTED | as above |
| `E4:claude` | Below-min and above-max installed versions both yield exit 5 naming the range | NOT TESTED | not reproduced fresh against this snapshot binary in this run; see prior-record note in [Section 5](#5-not-tested-and-why) |
| `E4:codex` | as above | NOT TESTED | as above |
| `E4:opencode` | as above | NOT TESTED | as above |
| `E4:grok` | as above | NOT TESTED | as above |
| `E4:qwen` | as above | NOT TESTED | as above |
| `E5:claude` | Active session for the agent detected; resume refused/warned per policy | NOT TESTED | as above |
| `E5:codex` | as above | NOT TESTED | as above |
| `E5:opencode` | as above | NOT TESTED | as above |
| `E5:grok` | as above | NOT TESTED | as above |
| `E5:qwen` | as above | NOT TESTED | as above |
| `E6:claude` | Non-interactive invocation exits 7 rather than launching blind | PASS | `rein resume claude:claude-syn-windows < NUL` (no `--json`, no `--dry-run`) → exit `7`; stderr `environment warnings require confirmation: baseline.unavailable` |
| `E6:codex` | as above | PASS | `rein resume codex:rollout-syn-001 < NUL` → exit `7`, same refusal shape |
| `E6:opencode` | as above | PASS | `rein resume opencode:ses_fixture001 < NUL` → exit `7`, `environment warnings require confirmation: baseline.unavailable, git.working_tree` |
| `E6:grok` | as above | PASS | `rein resume grok:01987654-3210-7890-abcd-ef0123456790 < NUL` → exit `7`, same refusal shape |
| `E6:qwen` | as above | PASS | `rein resume qwen:01912345-6789-7abc-def0-123456789abc < NUL` → exit `7`, same refusal shape |

## 4. T5 encrypted-sync round trip (`claude`, `codex`, `opencode`)

Required by this executor's assignment in addition to Matrix E.

| Row | Description | Result | Evidence |
| --- | ------------ | ------ | -------- |
| `sync:claude` | Push from device A, pull on isolated device B, vendor reads the restore back | NOT TESTED | `scripts/testing/fakelocker` was built successfully from this worktree (`go build ./scripts/testing/fakelocker` → clean, `fakelocker.exe` produced) and never run; see [Section 5](#5-not-tested-and-why) |
| `sync:codex` | as above | NOT TESTED | as above |
| `sync:opencode` | as above | NOT TESTED | as above |

## 5. NOT TESTED and why

Every row above is `NOT TESTED` for one of two reasons, named per row group:

1. **E1, E2, E3, E5, and the T5 sync round trip all require a session the
   real vendor binary itself recognizes.** A synthetic fixture — however
   faithfully shaped — is not addressable by the vendor's own `--resume`/
   `--session`/equivalent flag, because the vendor CLI validates the session
   ID against its own real store. Producing a genuine session requires
   running the authenticated vendor CLI for real. `docs/testing/results/
   2026-09-06-windows-range-widening-v060.md` records a precedent method for
   this — an isolated `CLAUDE_CONFIG_DIR`/`XDG_DATA_HOME` seeded with **only**
   the vendor's own credential file, never session content — for two of the
   five agents this executor is responsible for. Reproducing that method
   safely for all five agents (`claude`, `codex`, `opencode`, `grok`, `qwen`),
   plus the two-device push/pull round trip for the three T5 agents, is a
   multi-hour physical-journey undertaking per the contract's own description
   ("Physical acceptance... has repeatedly found more harness defects than
   product defects"); it was not completed inside this run's time budget
   after the fixture-based Matrix C/D work above, and this executor chose to
   report that honestly rather than fabricate physical-resume evidence that
   was never actually exercised. **This is the largest open gap in this
   report and should be treated as blocking full `v0.6.0-rc.1` sign-off until
   closed**, either by this executor in a follow-up pass or by reassignment.
2. **E4** (version-boundary refusal) requires either an installed vendor
   binary genuinely outside the verified range, or rebuilding the binary
   under test with a temporarily narrowed range — the latter is exactly what
   the range-widening precedent did, but that produces evidence for a
   *different* binary than the exact snapshot this report is supposed to be
   evidence for, so it was not substituted in here. Not attempted fresh
   against this snapshot in this run.

## 6. Harness and methodology findings

1. **Ambient `CLAUDE_CONFIG_DIR` inheritance (process-setup risk, not a
   product defect).** The first `rein doctor --agents --json` call in this
   session ran before this executor had overridden any root environment
   variable, and inherited a real, non-synthetic `CLAUDE_CONFIG_DIR` from the
   orchestrating process. Every command in the remainder of this report
   explicitly set all seven root variables before every invocation to
   prevent recurrence. **Recommendation for future runs:** the acceptance
   scripts (or this contract's run notes) should call out that
   `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/etc. can already be set in an agent's own
   ambient shell (not only via the documented host-contamination variables
   `REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR`/`XDG_DATA_HOME`), and
   that every one of the seven agent root variables should be asserted or
   overridden, not just checked for absence, before the first `rein doctor`
   call of a session.
2. **`internal/capability/discover.go`'s Codex skill scan reads the true OS
   user profile home unconditionally.** `scanCodex` resolves the shared
   cross-agent skills root as `filepath.Join(opts.UserHome, ".agents",
   "skills")`, and `opts.UserHome` is the real `os.UserHomeDir()` — it is
   **not** redirected by `CODEX_HOME`. On this shared host that directory
   held a large real skill inventory (a low-hundreds count observed), and its
   entries appeared inside `rein inspect`/`rein resume --dry-run`/`rein
   handoff --dry-run` output for `codex` regardless of `CODEX_HOME`
   isolation — this is why `C4:codex` and `D1:codex`'s evidence sizes are an
   order of magnitude larger than every other agent's. No skill name is
   reproduced anywhere in this report. This is disclosed as a **methodology
   caveat** for this report (this executor could not produce genuinely
   isolated `codex` capability-scan evidence on this host) and as a
   **candidate product finding** worth the team's attention independent of
   this contract: `~/.agents/skills` is a real, per-user, unredacted
   inventory that appears to be unconditionally embedded into `codex`
   session-environment output, with no environment-variable override and no
   redaction pass equivalent to Matrix B's probe redaction. This may be
   intentional (the "universal agent configuration" cross-vendor skills
   convention in `AGENTS.md`), but it was surprising in context and is worth
   a deliberate decision either way.
3. **`opencode` handoff-source detection reports `NOT_INSTALLED`** while
   every other opencode command (`sessions`, `search`, `inspect`,
   `resume --dry-run`) succeeds against the identical isolated `XDG_DATA_HOME`
   in the same run. See `D1:opencode` above. Not root-caused in this run;
   flagged for follow-up given OpenCode reaching T5 is a headline feature of
   this candidate.
4. **Qwen version drift.** The host's installed `qwen --version` reports
   `0.21.12`; the dispatch's vendor-version list for this executor's
   assignment named `0.21.13`. Immaterial to the fixture-based rows above
   (which don't invoke the live `qwen` binary), but should be re-verified
   immediately before any physical `qwen` resume journey, per the contract's
   own "re-check immediately before the run" instruction.

---

_Fixtures, throwaway git repository, and all isolated `REINSTATE_HOME`/agent
root directories used to produce this report live under
`D:\ReinstateAcceptanceProjects\v060-w7-b\` on the test host and are not
committed. No transcript text, real prompt, real response, credential,
private path, or repository name appears above._
