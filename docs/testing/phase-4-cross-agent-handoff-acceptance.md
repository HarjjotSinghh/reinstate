# Phase 4 cross-agent handoff acceptance

This is the release-neutral acceptance contract for Reinstate Phase 4. It
extends, and does not replace, the Phase 1 encrypted-sync, Phase 2 local
continuity, and Phase 3 verified-resume matrices.

Authoritative product behavior is [Cross-agent handoff](../handoff.md). Design
authority is [cross-agent-continuation.md](../cross-agent-continuation.md) and
[ADR 0003](../adr/0003-phase-4-rc1-scope-and-launch-route.md).

A candidate passes only when installed tagged artifacts produce truthful,
privacy-safe handoffs that a destination agent can actually continue from.

The current candidate is pinned to the signed tag `v0.4.0-rc.8`. Use the
[v0.4.0-rc.8 dispatch](v0.4.0-rc.8-agent-verification-prompts.md) and the
[Phase 4 report template](results/phase-4-report-template.md). Do not start
physical testing from an integration commit, source build, draft release, or an
installer that still pins stable `v0.3.0` or `v0.4.0-rc.1` through `v0.4.0-rc.7`.

The first seven candidates, `v0.4.0-rc.1` through `v0.4.0-rc.7`,
were published and **failed** this acceptance. `v0.4.0-rc.8` carries the rc.7
product fix (R1 off-PATH layout scan) and the Go 1.25.13 toolchain pin. The rc.8
dispatch carries RC1 `R1`–`R6`, the RC2, RC3, RC5, RC6, RC7, and RC8
regression sets so the rerun confirms them rather than rediscovering
them. The superseded [v0.4.0-rc.1](v0.4.0-rc.1-agent-verification-prompts.md),
[v0.4.0-rc.2](v0.4.0-rc.2-agent-verification-prompts.md),
[v0.4.0-rc.3](v0.4.0-rc.3-agent-verification-prompts.md),
[v0.4.0-rc.4](v0.4.0-rc.4-agent-verification-prompts.md),
[v0.4.0-rc.5](v0.4.0-rc.5-agent-verification-prompts.md),
[v0.4.0-rc.6](v0.4.0-rc.6-agent-verification-prompts.md), and
[v0.4.0-rc.7](v0.4.0-rc.7-agent-verification-prompts.md) dispatches remain in
the tree as the record of what those runs were asked to do.

## Evidence policy

- Test one immutable full commit and signed tag on every device.
- Build development evidence from a clean worktree. Test release evidence from
  verified installed artifacts, not an untagged local rebuild.
- Use a fresh isolated Reinstate home and fresh disposable Git repositories.
- Isolate **every** vendor before the first command. Export all of these, even
  for agents you do not expect the run to touch:

  | Vendor | Variable |
  | ------ | -------- |
  | Reinstate | `REINSTATE_HOME` |
  | Claude Code | `CLAUDE_CONFIG_DIR` |
  | Codex CLI | `CODEX_HOME` |
  | Gemini CLI | `GEMINI_CLI_HOME` |
  | Grok Build | `GROK_HOME` |

  Each source falls back to its documented default when its variable is unset,
  so one omission silently reads the operator's real tree. A `v0.4.0-rc.1` run
  omitted `GROK_HOME` and indexed the operator's real `~/.grok` sessions; the
  product was correct and the harness was wrong, but the contract still treats
  operator-tree access as a blocker, so that run was discarded and restarted.

  **OpenCode has no equivalent override.** It is read through
  `opencode session list`, which always targets the real store. Until an
  override exists, either uninstall/skip OpenCode for the run and record its
  rows `NOT TESTED`, or record explicitly that OpenCode rows were collected
  without isolation. Never report them as isolated.
- After the first index refresh, verify the session list contains **only**
  sessions the run itself created. Any unexpected session is an isolation
  failure: destroy the Reinstate home and restart, do not continue.
- Record every installed agent version **before** the first product command and
  again after the last row. Claude Code auto-updated mid-run during
  `v0.4.0-rc.1` on both hosts; a version change invalidates every row collected
  across it, and continuing silently is a process failure.
- Create fresh controlled sessions for each agent. Do not reuse a Phase 2 or
  Phase 3 corpus, and do not reuse an earlier candidate's handoffs.
- Never commit a transcript, full prompt, response, secret, credential, MCP
  configuration value, instruction content, skill content, private absolute
  path, capsule body, or raw environment dump.
- Preserve original failures. A targeted recheck may supersede a row only on
  the identical product commit/artifact and with new complete evidence.
- A missing required physical result is `FAIL`, not `PASS`. An optional vendor
  path may be `NOT TESTED` only when that vendor is genuinely absent and the
  runbook forbids installing it solely for acceptance.
- No test may fetch, checkout, reset, install, repair, or mutate project or
  vendor configuration on the user's behalf.
- No test may write into a vendor's private session store.

## Required environments

Development verification runs on the implementation host and CI. Tagged
release-candidate certification requires:

1. Apple Silicon macOS (`darwin/arm64`); and
2. native Windows x64 (`windows/amd64`, never WSL for the Windows column).

Intel macOS and Linux/WSL2 remain optional, unverified evidence and do not block
a candidate. Stable promotion requires a separate decision and fresh
tagged-artifact validation on both supported platforms under `RELEASING.md`.

## Required agent installs

| Agent | Required for | Notes |
| ----- | ------------ | ----- |
| Claude Code | Every row | Source **and** destination |
| Codex CLI | Every row | Source **and** destination |
| Gemini CLI | Source rows only | `NOT TESTED` allowed if genuinely absent |
| OpenCode | Source rows only | `NOT TESTED` allowed if genuinely absent |
| Grok Build | Source rows only | `NOT TESTED` allowed if genuinely absent |

---

## Matrix A — Flagship quota-switch (required, both platforms)

The defining scenario. Every row must pass with the **source CLI fully closed**.

| ID | Row | Pass condition |
| -- | --- | -------------- |
| A1 | Claude → Codex, source closed | Handoff completes with no Claude process running and no Claude network call |
| A2 | Codex → Claude, source closed | Same, reversed |
| A3 | Source logged out | A1 repeats after clearing the Claude session/auth from the shell environment; still succeeds |
| A4 | Interrupted final turn | Source's last record is a truncated JSONL line; the latest complete user intent still survives, verbatim |
| A5 | Destination restates correctly | The destination's first reply correctly restates goal, latest request, changed files, test state, and next action |
| A6 | No duplicate effects | No command or file write from the source transcript is re-executed at the destination |
| A7 | Lineage recorded | `rein handoff list` shows the handoff; Claude destination resolves its pinned ID; Codex destination resolves or honestly reports `unresolved`/`ambiguous` |

Capture for each: exit code, mode label in output, fidelity summary counts, and
the destination's restatement (sanitized — paraphrase, never paste transcript).

## Matrix B — Fidelity and policy (required)

| ID | Row | Pass condition |
| -- | --- | -------------- |
| B1 | `--dry-run` parity | Dry-run output matches the executed run byte for byte for the same input |
| B2 | Long history | A 200+ turn source yields a bounded projection; size and truncation are reported before launch |
| B3 | `checkpoint` policy | Zero verbatim conversation events; task state still sufficient to continue |
| B4 | `balanced` policy | Projection respects the byte budget; overflow appears as sidecar references |
| B5 | `full` policy | Hard cap enforced; excluded events are referenced, never silently dropped |
| B6 | Every fidelity class present | A real report contains `exact`, `normalized`, `summarized`, `referenced`, and `omitted` with reasons |
| B7 | Attachments | Locally present attachments are referenced with hash/MIME/size; absent ones are `omitted` with a reason |
| B8 | Unknown records | An unknown vendor record type is preserved as an opaque hashed reference, not guessed |

## Matrix C — Workspace and capability truth (required)

| ID | Row | Pass condition |
| -- | --- | -------------- |
| C1 | Branch/HEAD/dirty match | Reported workspace state equals the destination checkout's real state |
| C2 | Workspace wins | A transcript claim contradicted by Git is reported as conflicting, and Git's value is used |
| C3 | Missing workspace | A deleted or moved recorded workspace blocks the handoff before launch |
| C4 | Wrong project | A destination pointing at a different repository is refused |
| C5 | Cross-OS paths | Windows ↔ macOS paths remap through canonical project IDs; no source-device absolute path appears in the capsule |
| C6 | Missing MCP | An MCP server present at the source and absent at the destination is reported before launch |
| C7 | Missing skill/instruction | Same, for skills and instruction files |
| C8 | Untested version | An out-of-range source or destination version exits `5` and leaves no partial artifact |

## Matrix D — Security (required)

| ID | Row | Pass condition |
| -- | --- | -------------- |
| D1 | Prompt injection inert | An injected "ignore previous instructions" fixture stays inside the quoted, attributed block and changes nothing |
| D2 | Delimiter breakout | Content containing the import delimiter cannot escape the block |
| D3 | Source system prompt excluded | A source system/developer instruction never appears in `projection.md` |
| D4 | Secret redaction | Planted synthetic credentials are redacted before write; `--show-redactions` shows categories and counts only |
| D5 | No credential read | No auth file, keychain entry, `.env`, or token store is read during a handoff |
| D6 | Private on disk | `handoffs/` is `0700`, files `0600` (Unix) or protected DACL (Windows), and outside the repository |
| D7 | No vendor writes | No file under `~/.claude`, `~/.codex`, `~/.gemini`, `~/.grok`, or OpenCode storage is created or modified |
| D8 | No sync leakage | `rein push`/`pull` never includes `handoffs/` |
| D9 | Grok warning | A Grok-sourced handoff prints the upload-behavior warning; `--no-redact` is refused with exit `2` |
| D10 | Source immutability | The source session file's SHA-256 is identical before and after the handoff |

## Matrix E — Source-only agents (optional, per install)

| ID | Row | Pass condition |
| -- | --- | -------------- |
| E1 | Gemini → Claude | Handoff completes; rewound turns are absent from the capsule |
| E2 | Gemini → Codex | Same |
| E3 | OpenCode → Claude | Storage tier used when available; metadata fallback reports `omitted` honestly |
| E4 | OpenCode → Codex | Same |
| E5 | Grok → Claude | Completes with redaction forced and the warning printed |
| E6 | Grok → Codex | Same |
| E7 | Gemini/OpenCode/Grok as destination | **Refused** with a clear source-only message |
| E8 | Grok native resume/fork | **Refused** with the read-only reason |

## Matrix F — CLI contract (required)

| ID | Row | Pass condition |
| -- | --- | -------------- |
| F1 | Mode labeling | Every human and JSON surface says "structured handoff", never "resume" or "same session" |
| F2 | `--with` alias | `rein resume … --with codex` produces the same plan and prints the handoff notice |
| F3 | `--json` gating | `--json` without `--dry-run`/`--no-launch` exits `2` |
| F4 | Warning acknowledgement | Unknown, duplicate, wildcard, and stale warning IDs all exit `2`; unacknowledged warnings exit `7` |
| F5 | `--no-launch` | Prints the exact command and spawns nothing |
| F6 | `inspect`/`export` | Both reproduce the stored artifacts exactly |
| F7 | Ambiguous reference | A bare native ID matching two agents exits `6` with both qualified options |
| F8 | Non-TTY | A launch attempt without a terminal fails closed before spawning |

## Matrix G — Performance (required)

| ID | Row | Ceiling |
| -- | --- | ------- |
| G1 | 200-turn parse → capsule → projection | Under the absolute wall-clock ceiling recorded in the candidate dispatch |
| G2 | `handoff --dry-run` on the largest local session | Same ceiling |
| G3 | `handoff list` with 100 stored handoffs | Under the list ceiling |

Record absolute milliseconds on both platforms. A regression against the prior
candidate is a finding even when it stays under the ceiling.

---

## Reporting

Each device produces one sanitized report from
`results/phase-4-report-template.md`, containing:

- device, OS build, architecture, and installed agent versions;
- the exact tag, full commit, and artifact checksums tested;
- one row per matrix ID with `PASS` / `FAIL` / `NOT TESTED` and a reason;
- exit codes and sanitized output excerpts;
- absolute timings for Matrix G;
- any deviation from the dispatch, explicitly called out.

Reconciliation rule: a candidate passes only when every required row on **both**
supported platforms is `PASS`. One `FAIL` on one platform fails the candidate.

## Candidate dispatches

The current per-tag instructions are the
[`v0.4.0-rc.8` dispatch](v0.4.0-rc.8-agent-verification-prompts.md). It fixes
report branches, artifact and full-commit checks, the report-template
substitutions for this tag, the pre-run agent-version record, RC1 `R1`–`R6`
re-verification, the RC2, RC3, RC5, RC6, RC7, and RC8 regression sets, corpus sizes, absolute performance
ceilings, and final reconciliation rules before physical testing starts. Do not
begin a run without the dispatch for the exact tested tag. The superseded
[`v0.4.0-rc.1`](v0.4.0-rc.1-agent-verification-prompts.md) through
[`v0.4.0-rc.7`](v0.4.0-rc.7-agent-verification-prompts.md) dispatches are
retained only as the record of those failed runs.
