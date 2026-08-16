# OpenAI Codex CLI

**Confidence: Verified** — `internal/sessionindex/codex.go`,
`internal/adapter/codex/codex.go`, fixtures in
`testdata/sessionindex/codex/forks/`.

| Aspect | Value |
| ------ | ----- |
| Root (override) | `$CODEX_HOME` |
| Root (macOS/Linux) | `~/.codex`, fallback `~/.config/codex` |
| Root (Windows) | `%USERPROFILE%\.codex` |
| Sessions | `<root>/sessions/YYYY/MM/DD/rollout-<RFC3339-ish>-<uuid>.jsonl` |
| Format | JSON Lines "rollout" records |
| Session identity | Trailing UUID of the **filename** is authoritative |
| Native resume | `codex resume <session-id>`, `codex resume --last` |
| Native fork | `codex fork <session-id>` |
| Initial prompt | Positional argument: `codex "<prompt>"` |
| Non-interactive | `codex exec …`, `codex exec --last` |

### Initial-prompt argv ceiling (R6 — Documented / Unverified)

Codex accepts a new-session prompt as `codex "<bootstrap>"` (**Documented**).
No Codex-published Windows-specific argv byte ceiling was found in vendor docs
Reinstate trusts (**Unverified**). Destination handoffs therefore enforce
`TargetCapabilities.MaxArgvBytes`, defaulting to `DefaultMaxArgvBytes`
(24 KiB) from the Phase 4 architecture plan — a Reinstate conservative budget,
not a vendor constant. Over-budget plans fall back to a short bootstrap that
references `projection.md` only. See
[research/2026-08-12-phase-4-r6-codex-argv.md](../research/2026-08-12-phase-4-r6-codex-argv.md).

### Context-window ceiling (R7 — Omitted)

No Codex CLI **harness** token ceiling is published in the vendor docs
Reinstate trusts. Capability-diff summaries omit it with the same R7 reason
as Claude Code. See
[research/2026-08-12-phase-4-r7-context-ceilings.md](../research/2026-08-12-phase-4-r7-context-ceilings.md).

### Why the filename wins

Codex names every rollout file after **that file's own** session, including
forks — but a fork also replays the source's records, so its `session_meta` can
carry the source ID. Pinning identity to the filename keeps a fork addressable
and stops it collapsing into its parent. Phase 4 readers must reuse
`codexSessionIDFromFilename` semantics rather than trusting in-file IDs.

### Record shapes Phase 4 relies on

Two coexisting representations; readers must handle both and prefer the first:

1. `{"type":"event_msg","payload":{"type":"user_message"|"agent_message", …}}`
2. `{"type":"response_item","payload":{"type":"message","role":"user"|"assistant","content":[…]}}`

`session_meta` carries `payload.git.{branch,repository_url,commit_hash}` and
`payload.cwd`. Tool activity appears as typed response items with call IDs;
reasoning items may be opaque or encrypted and are **never** translated.

### Reasoning items (R4 — Documented)

Codex rollouts may include Responses API `reasoning` items under
`response_item`. Phase 4 classifies **all** of the following as
`portability: omitted` with reason `vendor_opaque_state` and never copies
payload bodies into capsule blocks:

1. `{"type":"response_item","payload":{"type":"reasoning","encrypted_content":…}}`
2. `{"type":"response_item","payload":{"type":"reasoning",…}}` (including
   summary-only or empty-summary forms)
3. Any other `response_item` payload that carries `encrypted_content` /
   `encryptedContent`

Visible assistant text remains on `event_msg`/`agent_message` (preferred) or
`response_item`/`message`/`role=assistant` when no `event_msg` exists.
Synthetic fixtures: `testdata/handoff/codex/reasoning-items/`.
