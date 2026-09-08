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

### Version range

`codex --version` prints one stdout line, `codex-cli <semver>`, and nothing on
stderr. The catalog's inclusive range was `0.133.0`–`0.149.0`, raised on
dual-platform physical resume evidence during `v0.5.1`. `v0.6.0` widens the
ceiling to `0.153.4` on native Windows evidence, after the acceptance host's
Codex CLI self-updated past `0.149.0` mid-cycle, under the maintainer's
standing policy that a vendor CLI self-updating past the verified ceiling
widens the range rather than blocks (2026-09-07, Q27). This widening's own
evidence is read-only — `codex --version`/`--help`/`exec --help`/
`resume --help`/`fork --help` output shape, `rein doctor --agents --json`'s
sanitized probe of the real, live `CODEX_HOME`, and a committed fixture home
exercised through `rein resume --dry-run --json` — gathered against the real
`0.153.4` binary; the host's Codex account usage limit blocks any completed
conversational turn until `10:13` local on `2026-09-08`; the completed-turn
`codex:E1`–`E3` resume/fork rows against `0.153.4` were confirmed via real,
completed conversational turns in the `v0.6.0-rc.7` tagged run (macOS
pending, ADR 0005 D3). See
[2026-09-08-windows-range-widening-codex-v060.md](../testing/results/2026-09-08-windows-range-widening-codex-v060.md).

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
