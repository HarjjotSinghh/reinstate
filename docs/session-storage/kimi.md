# Kimi Code CLI (Moonshot AI)

**Confidence: Unverified** — no Reinstate reader exists.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T3

Kimi Code CLI is the strongest T3 candidate in the Phase 5 roster: the vendor
documents a data-root override, a per-project session bucket, a plain-file
transcript, a global session index, and an explicit resume argv. It is also the
reason the probe requirement exists, because two vendor documentation mirrors
disagree about the data root.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Moonshot AI |
| Product | Kimi Code CLI |
| Binary | `kimi` |
| Distribution | Official, vendor-distributed |
| Storage family | F1 (home-dir tree) |

## Claimed layout

Every row below is **Unverified** until a probe confirms it. Two mirrors state
different roots; both are recorded rather than silently reconciled.

| Aspect | Mirror A (`moonshotai.github.io`, `kimi.com`) | Mirror B (`kimi-cli.com`) |
| ------ | -------------------------------------------- | ------------------------- |
| Root override | `$KIMI_CODE_HOME` | not stated |
| Root default | `~/.kimi-code/` | `~/.kimi/` |
| Sessions | `sessions/<workDirKey>/<sessionId>/` | `sessions/<work-dir-hash>/<session-id>/` |
| Project bucket | `wd_<derived from working directory>` | MD5 of the path |
| Metadata | `state.json` (title, timestamps, workDir) | `state.json` |
| Transcript | `agents/main/wire.jsonl` | `wire.jsonl` plus `context.jsonl` |
| Subagents | `agents/agent-0/…`, each with its own `wire.jsonl` | not stated |
| Global index | `session_index.jsonl` — one record per line with `sessionId`, `sessionDir`, `workDir` | not stated |
| Input history | `user-history/<md5(workDir)>.jsonl` | `user-history/<work-dir-hash>.jsonl` |
| Plans | `agents/main/plans/<plan-id>.md` | `plans/<slug>.md` |
| Imported sessions | not stated | `imported_sessions/<session-id>/` |
| Side state | `tasks/`, `cron/`, `logs/kimi-code.log` | `logs/` |
| Credentials | OAuth credential directory, mode `0700`, files `0600` | `auth/<provider>.json` |

Native control surface, consistent across both mirrors:

| Aspect | Value |
| ------ | ----- |
| Resume most recent in cwd | `kimi --continue` (`-c`) |
| Resume specific | `kimi --session <session-id>` |
| Fork | `/fork` inside the TUI |
| Rename | `/title` (alias `/rename`) — sets the title in `state.json` |
| Export | `/export` produces a diagnostic ZIP |
| Editor integration | ACP |

## What the probe must settle

1. **Which root is real** on macOS and on native Windows. If both exist,
   which one the running binary actually writes to.
2. Whether `$KIMI_CODE_HOME` is honored, and whether it relocates sessions
   only or the whole tree including credentials.
3. Whether `session_index.jsonl` exists. If it does, it is the cheapest
   possible discovery path — one file enumerates every session across every
   project — and the scanner should prefer it over a directory walk, with the
   walk as the fallback.
4. The exact `workDirKey` shape, so `internal/pathmap` can recompute the bucket
   on a destination device instead of reusing the source key.
5. Whether `context.jsonl` exists alongside `wire.jsonl`, and which one carries
   user-visible turns. A transcript reader must not merge two representations
   of the same turn.
6. Sub-agent directories: `agents/agent-*/` must be excluded from the
   top-level session list, the same way Claude Code subagents are.
7. `state.json` key set, for title and timestamp mapping.
8. Whether the OAuth credential directory sits inside the same root. If it
   does, it goes in the descriptor's `Excluded` set before any read, and well
   before any T5 consideration.

## Tier path

| Tier | Blocker |
| ---- | ------- |
| T1 | Root ambiguity; needs macOS and Windows probes |
| T2 | `wire.jsonl` record shape unknown; needs the unknown-record and truncation policy |
| T3 | Needs a `kimi --version` output shape and a fail-closed supported range |

T4 and T5 are out of scope for `v0.5.0` per
[ADR 0004](../adr/0004-universal-agent-coverage.md).

## Notes for the reader implementation

- `wire.jsonl` is described as a request trace that includes tool schemas,
  request parameters, and MCP tool listings for debugging. That is a large
  surface of records Reinstate must classify as `omitted` or `referenced`
  rather than parse. Do not attempt to interpret request traces.
- The vendor states the layout can change between releases and should be
  treated as inspectable rather than guaranteed. That is an argument for a
  strict layout gate, not a loose parser.

## Sources

- [Sessions and context](https://www.kimi.com/code/docs/en/kimi-code-cli/guides/sessions.html)
- [Data locations (moonshotai.github.io mirror)](https://moonshotai.github.io/kimi-code/en/configuration/data-locations.html)
- [Data locations (kimi-cli.com mirror)](https://www.kimi-cli.com/en/configuration/data-locations.html) — conflicts with the above
