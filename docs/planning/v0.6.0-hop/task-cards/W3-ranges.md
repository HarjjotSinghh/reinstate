# W3 — Verified-range widening on native Windows evidence

**Executor:** one Sonnet agent. **Verifier:** one Sonnet agent (re-run one
resume per widened agent independently; check the redaction).
**Branch:** `v060/w3-ranges` from `release/v0.6.0-rc.1`.
**Policy:** [ADR 0005](../../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md) D3.

Installed on the host at planning time (re-check; Claude Code auto-updates):

| Agent | Installed | Verified range in tree | Where |
| ----- | --------- | ---------------------- | ----- |
| Claude Code | `2.1.261` | `2.1.219`–`2.1.238` | `internal/adapter/claude/claude.go`, `internal/agentcheck/agent.go`, `internal/agents/catalog/claude.go` |
| OpenCode | `1.18.27` | `1.18.21`–`1.18.21` | `internal/agents/catalog/opencode.go` (and any adapter constant) |
| Codex CLI | `0.149.0` | `0.133.0`–`0.149.0` | in range |
| Qwen Code | `0.23.0` | `0.21.12`–`0.23.0` | widened on `v0.6.0-rc.4` native Windows evidence, `internal/agents/catalog/qwen.go` — see `docs/testing/results/2026-09-07-windows-range-widening-qwen-v060.md` |
| Grok Build | `1.0.5` | `1.0.5` | in range |

## T-301 / T-302 — Widen Claude Code and OpenCode

Read how `v0.5.0-rc.5` collected its widening evidence
(`RELEASING.md` "v0.5.0-rc.5 candidate gate",
`docs/testing/v0.5.0-rc.5-agent-verification-prompts.md`, and the Windows
report `docs/testing/results/2026-08-21-windows-phase5-V050RC6.md` E rows)
and reproduce that method for each agent, on this host, with the real vendor
binary:

1. Create a session with the installed version in a **throwaway project
   directory** under `D:\ReinstateAcceptanceProjects\v060-widen-<agent>\`,
   containing one token that exists nowhere else.
2. `rein sessions` / `rein search <token>` finds it; `rein inspect` reports
   the version as `UNTESTED` before the change and `SUPPORTED` after.
3. `rein resume <agent>:<id> --dry-run` produces a complete launch plan, then
   the resume runs through that plan (interactive: use W4's ConPTY driver if
   it exists, else the interactive scheduled-task path in #367, else the
   vendor's non-interactive resume form invoked with the exact argv the plan
   printed) and the resumed session returns the token from history.
4. For OpenCode also: `rein handoff --to opencode` dry-run validates (T4) and
   a push/pull round trip between two isolated Reinstate homes on this host
   restores the session with a stable revision (T5), as the OpenCode T5
   Windows record did.
5. Move the ceilings. Update every test that pins the old ceiling
   (`grep -rn '2\.1\.238\|1\.18\.21' --include=*.go --include=*.md
   --include=*.json .` outside `website/dist`, `website/.vercel`, and
   frozen changelog history). A fail-closed test must still show that
   `2.1.262` (Claude) and `1.18.28` (OpenCode) are `UNTESTED`.

Never read, list, copy, or quote the developer's own sessions. The throwaway
project is the only path that may appear, and the results doc shows it
redacted as `<lab-project>`.

## T-303 — Record what was verified

`docs/testing/results/2026-09-DD-windows-range-widening-v060.md`: host
(sanitized), branch tip, each agent's installed version, the exact commands
(argv only, no output bodies), PASS/FAIL per step, and the sentence "macOS
evidence: pending (ADR 0005)". Same redaction rules as the Phase 5 template.

## T-304 — Data and wording

`docs/compatibility.md` numbers and the note "widened on native Windows
evidence; macOS pending"; `website/src/data/compatibility.json`
`maximumTestedVersion` and `notes` (coordinate with W5: W3 edits only the two
version fields and the note strings; W5 owns everything else in that file).

## Done when

Gates 1–4; the verifier reproduced one resume per agent; the results doc is
committed; the coordinator has told W1 to do T-105.
