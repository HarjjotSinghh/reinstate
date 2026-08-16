# Kimi Code CLI (Moonshot AI)

**Confidence: Unverified** — no Reinstate reader exists.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T3

Kimi Code CLI remains a strong T3 candidate on paper: the official vendor
docs describe a data-root override, a per-project session bucket, a global
session index, a plain-file transcript, and explicit resume argv. Two vendor
mirrors still disagree about the data root. Dual-platform probes are absent,
so the catalog descriptor stays at T0.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Moonshot AI |
| Product | Kimi Code CLI |
| Binary | `kimi` |
| Distribution | Official, vendor-distributed |
| Storage family | F1 (home-dir tree) |

## Executor finding (2026-08-16)

T-020 could not produce the evidence T1 and above require.

| Check | Result |
| ----- | ------ |
| `kimi` on PATH | not installed |
| `rein doctor --agents --json` | not captured: the vendor binary is absent and has not been used |
| macOS AGENT-PROBE-V1 | **absent** |
| native Windows AGENT-PROBE-V1 | **absent** (no native Windows host in this session) |
| Physical `kimi --continue` / `kimi --session <id>` | not run |
| `kimi --version` shape | unknown |

A top-level listing of the executor macOS home showed `~/.kimi-code/` present
with `config.toml` only (no `sessions/`, no `session_index.jsonl`). `~/.kimi/`
was absent. File contents were not read. That listing is **not** an
`AGENT-PROBE-V1` artifact and does not promote any row.

T1 is forbidden without both a macOS probe and a native Windows probe. The
descriptor therefore stays at T0 with `t0_reason=layout_unverified`. That is
the complete T-020 result.

## Claimed layout

Every row below is **Unverified** until a committed probe confirms it. Two
mirrors state different roots; both are recorded rather than silently
reconciled.

| Aspect | Mirror A (`moonshotai.github.io`, `kimi.com`) | Mirror B (`kimi-cli.com`) |
| ------ | -------------------------------------------- | ------------------------- |
| Root override | `$KIMI_CODE_HOME` (relocates **all** data, including credentials) | `$KIMI_SHARE_DIR` (runtime data only; skills stay elsewhere) |
| Root default | `~/.kimi-code/` | `~/.kimi/` |
| Sessions | `sessions/<workDirKey>/<sessionId>/` | `sessions/<work-dir-hash>/<session-id>/` |
| Project bucket | `wd_<slug>_<first-12-chars-of-sha256>` | MD5 of the path |
| Metadata | `state.json` (title, lastPrompt, timestamps, forkedFrom) | `state.json` (title, approval, plan fields, subagent_instances) |
| Transcript | `agents/main/wire.jsonl` | `context.jsonl` (restore) plus `wire.jsonl` (replay / title) |
| Subagents | `agents/agent-0/…`, each with its own `wire.jsonl` | `subagents/<agent_id>/` with `context.jsonl` + `wire.jsonl` |
| Global index | `session_index.jsonl` — one record per line with `sessionId`, `sessionDir`, `workDir` | not stated |
| Input history | `user-history/<md5(workDir)>.jsonl` | `user-history/<work-dir-hash>.jsonl` |
| Plans | `agents/main/plans/<plan-id>.md` | `plans/<slug>.md` |
| Imported sessions | not stated | `imported_sessions/<session-id>/` |
| Side state | `tasks/`, `cron/`, `logs/kimi-code.log`, `bin/`, `updates/` | `logs/kimi.log` |
| Credentials | `credentials/` (dir `0700`, files `0600`; MCP under `credentials/mcp/`) | `credentials/<provider>.json` and `mcp-oauth/` |

Native control surface, consistent across both mirrors as vendor documentation
only — **not physically verified**:

| Aspect | Value |
| ------ | ----- |
| Resume most recent in cwd | `kimi --continue` (`-c`) |
| Resume specific | `kimi --session <session-id>` |
| Fork | `/fork` inside the TUI |
| Rename | `/title` (alias `/rename`) — sets the title in `state.json` |
| Export | `kimi export` / `/export` produces a diagnostic ZIP |
| Editor integration | ACP |

A vendor changelog on the official mirror mentions migrating user skills from
`~/.kimi/skills/` to `~/.kimi-code/skills/`. That is a hint that `.kimi` was a
previous root; it is not session-layout evidence and does not promote a row.

## What the probe must settle

These questions remain open. Vendor documentation alone is never sufficient.

1. **Which root is real** on macOS and on native Windows. If both exist,
   which one the running binary actually writes to.
2. Whether `$KIMI_CODE_HOME` is honored, and whether it relocates sessions
   only or the whole tree including credentials. Mirror B's `$KIMI_SHARE_DIR`
   must be confirmed or discarded.
3. Whether `session_index.jsonl` exists. If it does, it is the cheapest
   possible discovery path — one file enumerates every session across every
   project — and the scanner should prefer it over a directory walk, with the
   walk as the fallback.
4. The exact `workDirKey` shape, so `internal/pathmap` can recompute the bucket
   on a destination device instead of reusing the source key.
5. Whether `context.jsonl` exists alongside `wire.jsonl`, and which one carries
   user-visible turns. A transcript reader must not merge two representations
   of the same turn.
6. Sub-agent directories: `agents/agent-*/` (Mirror A) and `subagents/`
   (Mirror B) must be excluded from the top-level session list, the same way
   Claude Code subagents are.
7. `state.json` key set, for title and timestamp mapping.
8. Whether the OAuth credential directory sits inside the same root. If it
   does, it goes in the descriptor's `Excluded` set before any read, and well
   before any T5 consideration.

Escalate if a future probe shows sessions stored somewhere neither mirror
documents.

## Policy already recorded (applies when a reader is written)

The T0 descriptor lists claimed candidate roots and exclusions so
`rein doctor --agents` can inventory them. It does **not** scan, parse, or
resume.

- **Roots.** Ordered candidates: `~/.kimi-code` first (official), `~/.kimi`
  second (conflicting mirror). First existing root with a `sessions/` marker
  wins. Both remain unverified.
- **Override.** `KIMI_CODE_HOME` is the only `RootEnv`. `KIMI_SHARE_DIR` is
  not wired until a probe shows the running binary honors it.
- **Index.** Prefer `session_index.jsonl` when present; walk `sessions/` as
  the fallback for a missing or stale index. Not implemented at T0.
- **Transcript.** Do not merge `wire.jsonl` and `context.jsonl`. Classify
  request-trace records (tool schemas, request parameters, MCP listings) as
  `omitted`. Do not interpret them.
- **Subagents.** Exclude `agents/agent-*` and `subagents/` from the top-level
  session list.
- **Credentials.** Exclude `credentials/` and `mcp-oauth/` before any read.

## Tier path

| Tier | Blocker |
| ---- | ------- |
| T1 | Root ambiguity; needs macOS **and** native Windows `AGENT-PROBE-V1` artifacts after real use in at least two projects |
| T2 | `wire.jsonl` / `context.jsonl` record shape unknown; needs the unknown-record and truncation policy |
| T3 | Needs a `kimi --version` output shape, a fail-closed supported range, and physical `--continue` / `--session` on both platforms |

T4 and T5 are out of scope for `v0.5.0` per
[ADR 0004](../adr/0004-universal-agent-coverage.md).

## Notes for the reader implementation

- Official docs describe `agents/main/wire.jsonl` as both the resume stream
  and a request trace that includes tool schemas, request parameters, and MCP
  tool listings. That is a large surface of records Reinstate must classify as
  `omitted` or `referenced` rather than parse.
- The vendor states the layout can change between releases and should be
  treated as inspectable rather than guaranteed. That is an argument for a
  strict layout gate, not a loose parser.

## Sources

- [Sessions and context](https://www.kimi.com/code/docs/en/kimi-code-cli/guides/sessions.html)
- [Data locations (moonshotai.github.io mirror)](https://moonshotai.github.io/kimi-code/en/configuration/data-locations.html)
- [Data locations (kimi-cli.com mirror)](https://www.kimi-cli.com/en/configuration/data-locations.html) — conflicts with the above
