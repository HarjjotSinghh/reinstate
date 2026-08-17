# Kimi Code CLI (Moonshot AI)

**Confidence: Documented on macOS, Unverified on native Windows** — no
Reinstate reader exists.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T3

A macOS probe on 2026-08-16 settled the root ambiguity and confirmed the
layout, including the global session index. Kimi Code CLI is now the strongest
T1 candidate in the roster. The tier does not move on one platform's evidence,
so the descriptor stays at T0 until a native Windows probe exists.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Moonshot AI |
| Product | Kimi Code CLI |
| Binary | `kimi` |
| Distribution | Official, vendor-distributed |
| Storage family | F1 (home-dir tree) |

## Device evidence (2026-08-16, macOS arm64)

Artifact:
[`2026-08-16-macos-kimi.json`](../testing/results/agent-probes/2026-08-16-macos-kimi.json)

| Check | Result |
| ----- | ------ |
| `kimi` on PATH | yes |
| `kimi --version` | `0.36.1` — a bare semver line, no product prefix |
| Resolved root | `~/.kimi-code` |
| `~/.kimi` | absent |
| macOS AGENT-PROBE-V1 | **captured**, one real session in one project |
| native Windows AGENT-PROBE-V1 | **absent** (no native Windows host) |
| Physical `kimi --continue` / `kimi --session <id>` | not run |

Observed tree, with variable components shape-normalized by the probe:

```
~/.kimi-code/
  session_index.jsonl                    keys: sessionId, sessionDir, workDir
  workspaces.json                        keys: workspaces, deleted_workspace_ids, version
  sessions/wd_<user>_<12-hex>-<n>/
    session-<uuid-v4>/
      state.json                         keys: id, title, titleKind, isCustomTitle,
                                               cwd, createdAt, updatedAt, lastPrompt,
                                               lastTurnReason, agents, archived,
                                               custom, version
      agents/main/wire.jsonl             keys: type, created_at, protocol_version
      logs/kimi-code.log
  user-history/<32-hex>.jsonl            keys: content
  workspace-trust/wd_<user>_<12-hex>-<n>
  cache/query-store/shard-<n>/…          local search index, not session data
  config.toml, tui.toml, device_id, logs/, updates/, telemetry/
```

What this settles:

1. **The root is `~/.kimi-code`.** Mirror A is correct and Mirror B's `~/.kimi`
   does not exist on this device. The mirror conflict is resolved for macOS.
2. **`session_index.jsonl` exists**, keyed exactly as Mirror A describes. One
   file enumerates every session, so the scanner should prefer it and keep the
   directory walk only as a fallback for a stale or missing index.
3. **The project bucket is `wd_<slug>_<12-hex>`**, not the MD5 Mirror B claims.
   The slug is the account name, which is why the probe redacts it to `<user>`.
   `internal/pathmap` must recompute this on a destination device.
4. **`state.json` carries everything the index needs** — `title`, `cwd`,
   `createdAt`, `updatedAt` — so a T1 row needs no transcript parse.
5. **No `context.jsonl` was observed**, only `agents/main/wire.jsonl`. Mirror
   B's dual-file claim is unsupported so far, though one session is thin
   evidence for a negative.
6. **No `credentials/` directory was created** by this install. The exclusion
   stays in the descriptor regardless; absence on one device is not a licence
   to drop it.

Still open: everything about native Windows, the `$KIMI_CODE_HOME` override,
multi-project and multi-session behaviour, and subagent directories — this
session used one project and never spawned a subagent.

## Claimed layout

Rows below that the macOS probe did not touch remain **Unverified**. Two
mirrors state different roots; both are recorded rather than silently
reconciled, and the root row is now settled for macOS in favour of Mirror A.

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

Items 1, 3, 4 and 7 are answered above for macOS. These remain open.

1. **Native Windows**, for every row. The macOS answer does not transfer.
2. Whether `$KIMI_CODE_HOME` is honored, and whether it relocates sessions
   only or the whole tree including credentials. Mirror B's `$KIMI_SHARE_DIR`
   must be confirmed or discarded.
3. Whether `session_index.jsonl` stays consistent across many sessions and
   projects, and what happens to it when a session directory is deleted by
   hand. A one-session probe cannot show staleness.
4. Whether the `<12-hex>` half of the bucket is a SHA-256 prefix as Mirror A
   claims. One sample cannot distinguish hash functions.
5. Whether `context.jsonl` appears in longer sessions alongside `wire.jsonl`,
   and which one carries user-visible turns. A transcript reader must not merge
   two representations of the same turn.
6. Sub-agent directories: `agents/agent-*/` (Mirror A) and `subagents/`
   (Mirror B) must be excluded from the top-level session list, the same way
   Claude Code subagents are. This probe saw only `agents/main/`.
7. The `wire.jsonl` record vocabulary. The probe reads first-line keys only, so
   `type`, `created_at`, `protocol_version` is a header record, not the shape
   of a turn.
8. Whether the OAuth credential directory appears once the CLI is
   authenticated. It goes in `Excluded` either way.

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
| T1 | **Code complete, evidence incomplete.** `internal/agents/sources/kimi` is written and tested against both fixture platforms. The only remaining blocker is a native Windows `AGENT-PROBE-V1`; a macOS re-probe across two projects should land with it |
| T2 | `wire.jsonl` vocabulary is now known (below); still needs the unknown-record and truncation policy |
| T3 | `kimi --version` prints a bare `0.36.1`, so a range is now expressible. Still needs a fail-closed supported range and physical `--continue` / `--session` on both platforms |

T4 and T5 are out of scope for `v0.5.0` per
[ADR 0004](../adr/0004-universal-agent-coverage.md).

## Record shapes, read from the device

These come from the same macOS session as the probe artifact, read field by
field rather than through the shape normalizer. They are what
`internal/agents/sources/kimi` parses.

| Aspect | Value |
| ------ | ----- |
| Session directory | `session_<uuid-v4>` — an **underscore**. The probe artifact shows `session-<uuid-v4>` only because the normalizer trims `-_` around a token |
| Project bucket | `wd_<account-name>_<12-hex>`, confirmed |
| `state.json` `version` | `2` (integer). This is the layout gate: any other value fails closed |
| `state.json` `id` | `session_<uuid-v4>`, matching the directory |
| `state.json` `agents.main.homedir` | an **absolute path** to the session directory, so `internal/pathmap` must rewrite it before any cross-device use |
| `wire.jsonl` first record | `{"type":"metadata","protocol_version":"1.5","created_at":…}` |

Observed `wire.jsonl` record types: `metadata`, `profile.bind`,
`permission.set_mode`, `plugin.session_start`, `llm.tools_snapshot`,
`llm.request`, `turn.prompt`, `context.append_message`,
`context.append_loop_event`, `usage.record`, `turn.ended`.

Two carry indexable content:

- `turn.prompt` — `origin: {kind: "user"}` and `input: [{type: "text", text}]`.
- `context.append_message` — `message: {id, role, origin, content, toolCalls}`.

This is one short session, so the list is a floor and not a vocabulary. A T2
reader must still classify unknown records rather than assume this is all of
them.

## Why the index source ignores `session_index.jsonl`

The vendor writes a global index at the root, and it would be the cheaper
discovery path: one file enumerates every session across every project, with no
deep walk. The shipped source walks `sessions/**/state.json` anyway.

The reason is staleness. Nothing observed so far shows what happens to that
index when a session directory is deleted by hand, and an index that outlives
its sessions makes `rein list` offer threads that cannot be opened — a worse
failure than a slower scan, because the user cannot tell it is wrong. Detecting
staleness would require the walk the index was meant to avoid. It becomes an
optimisation once a probe covers several projects and a hand-deleted session.

## Implementation status

`internal/agents/sources/kimi` exists, with synthetic fixtures under
`testdata/sessionindex/kimi/{macos,windows}` and tests covering both fixture
platforms, subagent exclusion, determinism, wire-log authority, and six
corruption cases.

**It is not registered on the descriptor.** The catalog stays T0, so the
conformance suite would reject a `NewIndexSource` above the declared tier.
Promotion is a three-line descriptor change — set `Tier`, drop `T0Reason`, wire
`NewIndexSource` — plus the `Evidence.ProbeReports` and `Evidence.Fixtures`
paths, and it is gated on one thing: a native Windows `AGENT-PROBE-V1`. That
requirement is now enforced in code by `probePlatformGap`, not only documented
here.

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
