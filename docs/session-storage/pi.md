# Pi coding agent (earendil-works)

**Confidence: Unverified** — no Reinstate reader exists.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T3

Pi is an open-source minimal terminal harness. It documents two separate
directory overrides — one for configuration and one specifically for session
storage — and it exports environment variables that identify itself to child
processes, which makes process detection unusually reliable.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | earendil-works (Mario Zechner) |
| Product | Pi coding agent |
| Binary | `pi` |
| Package | `@earendil-works/pi-coding-agent` |
| Distribution | Official, npm, MIT |
| Storage family | F1 (home-dir tree) |

## Claimed layout

Every row is **Unverified** until a probe confirms it.

| Aspect | Value |
| ------ | ----- |
| Config root override | `PI_CODING_AGENT_DIR` |
| Config root default | `~/.pi/agent` |
| Session storage override | `PI_CODING_AGENT_SESSION_DIR` |
| Session storage default | not stated; presumed under the config root |
| Session format | not stated |
| Project scoping | not stated |
| Resume most recent | `pi -c` |
| Browse sessions | `pi -r` |
| One-shot | `pi -p "<prompt>"` |
| Event stream | `--mode json` |
| Process integration | RPC over stdin/stdout |
| Self-identification | sets `AI_AGENT=pi` and `PI_CODING_AGENT=true` for child processes |

The self-identification variables are worth calling out: they are an explicit
vendor mechanism for attributing child processes to Pi, so
`internal/processcheck` should prefer them over binary-name heuristics.

## What the probe must settle

1. Where sessions actually land when neither override is set. The vendor
   documents `PI_CODING_AGENT_SESSION_DIR` as a separate override, which
   implies the default is not simply the config root.
2. The on-disk session format: one file per session, a directory per session,
   or an append-only log.
3. Whether sessions are bucketed by project at all. `pi -c` continues the last
   session, and `pi -r` browses sessions, but neither documents whether the
   scope is global or per working directory. If it is global, the descriptor's
   `ProjectKey` is `none` and workspace attribution must come from the record
   body, which caps search quality until confirmed.
4. Whether the documented HTML session export writes into the session tree. If
   it does, exports are not sessions and must be excluded from discovery.
5. `pi --version` output shape, for the T3 version probe.
6. Whether the RPC or `--mode json` surfaces expose a session list. A
   documented list command would make Pi an F2 agent instead of F1, which is
   cheaper and more stable than parsing private files.

Question 6 should be answered before writing any file parser. Pi documents four
machine-facing modes; using a supported interface is always preferred to
reading private storage, the same reasoning that made OpenCode an F2 agent.

## Tier path

| Tier | Blocker |
| ---- | ------- |
| T1 | Default session directory unknown; project scoping unknown |
| T2 | Record format unknown |
| T3 | Needs `pi --version` output shape and a fail-closed supported range. Pi releases very frequently, so the range policy needs an explicit maintenance note |

Pi's release cadence is a real T3 risk. A narrow pinned range will go stale
within weeks. Decide the range policy with the maintainer before promoting.

## Sources

- [pi.dev](https://pi.dev/)
- [earendil-works/pi](https://github.com/earendil-works/pi)
- [packages/coding-agent README](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/README.md)
- [npm @earendil-works/pi-coding-agent](https://www.npmjs.com/package/@earendil-works/pi-coding-agent)
