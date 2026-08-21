# Pi coding agent (earendil-works)

**Confidence: Documented on macOS and native Windows** —
dual-platform AGENT-PROBE-V1 after real `pi -p` sessions; T1 index source shipped.
**Current tier:** T1 (discover) · **Phase 5 target:** T3

Pi is an open-source terminal harness. Dual-platform AGENT-PROBE-V1 exists
after real `pi -p` sessions. npm currently warns that
`@mariozechner/pi-coding-agent` is deprecated in favor of
`@earendil-works/pi-coding-agent`; the fail-closed pin remains `0.73.1`.
Resume and fork stay refused.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | earendil-works (Mario Zechner) |
| Product | Pi coding agent |
| Binary | `pi` |
| Package | `@earendil-works/pi-coding-agent` |
| Distribution | Official, npm, MIT |
| Storage family | F1 (home-dir tree) — see [Family](#family-f1-not-f2) |
| Catalog key | `pi` |
| Fail-closed version pin | `0.73.1`–`0.73.1`. T1: identity pin, not a resume claim. |

## Device evidence (2026-08-17, macOS arm64)

Artifact:
[`2026-08-17-macos-pi.json`](../testing/results/agent-probes/2026-08-17-macos-pi.json)

`pi -p` completed with a one-word reply. Resolved root `~/.pi/agent`. Tree:

```
~/.pi/agent/
  sessions/<slug>/<slug>-<uuid-v4>.jsonl
  settings.json
```

First-line keys on the session file: `cwd, id, timestamp, type, version`.
That matches the published `type=session` header. The probe did not capture
`version_raw`; the same terminal printed `0.73.1` for `pi --version`.

## Device evidence (2026-08-17, native Windows amd64)

Artifact:
[`2026-08-17-windows-pi.json`](../testing/results/agent-probes/2026-08-17-windows-pi.json)

Same pin `0.73.1`, same tree and first-line keys as macOS after a real
`pi -p` session. Path slugs collapsed to `<slug>`. Promoted to T1 on
2026-08-19; resume and fork stay refused.

## Family: F1, not F2

Question 6 on this page was answered from vendor docs **before** any parser.

| Surface | Session list? |
| ------- | ------------- |
| `pi -r` / `/resume` | Interactive TUI only |
| `--mode json` | Event stream for one run, not a catalog |
| `--mode rpc` | Current-session commands (`get_state`, `switch_session`, `get_entries`). No list command |
| TypeScript SDK | `SessionManager.list` / `listAll` exist in-process. Not a CLI Reinstate can call |

There is no supported machine-readable session list. Pi is F1. The SDK listing
methods are not an F2 interface.

## Claimed layout

Every on-disk row stays **Unverified** until a probe confirms it. The values
below are what the vendor currently publishes.

| Aspect | Vendor-published value | Probe status |
| ------ | ---------------------- | ------------ |
| Config root override | `PI_CODING_AGENT_DIR` | Unverified |
| Config root default | `~/.pi/agent` | **macOS:** resolved |
| Session storage override | `PI_CODING_AGENT_SESSION_DIR` (overridden by `--session-dir`, then `sessionDir` in settings.json) | Unverified |
| Session storage default | `~/.pi/agent/sessions/` (under the config root, not a sibling of it) | **macOS:** present |
| Session path | `sessions/--<cwd-with-slashes-as-hyphens>--/<timestamp>_<uuid>.jsonl` | **macOS:** `sessions/<slug>/<slug>-<uuid-v4>.jsonl` |
| Session format | JSONL; first line `type=session` header (`version`, `id`, `cwd`); later lines tree entries with `id` / `parentId` | **macOS first line:** `cwd, id, timestamp, type, version` |
| Header versions | v1 linear (legacy), v2 tree, v3 `hookMessage` → `custom`; load migrates to v3 | Unverified |
| Project scoping | Yes: one cwd-slug directory per working directory | **macOS:** one slug directory |
| Project key | Path slug (`/` → `-`, wrapped in `--…--`). Windows encoding unknown | **macOS:** collapsed to `<slug>`; Windows unknown |
| HTML / JSONL export | `/export [file]`, `--export <in> [out]`, RPC `export_html` write to a caller path (RPC default `/tmp/session.html`), not into the session tree | Unverified |
| Credentials | `~/.pi/agent/auth.json` (mode `0600`); OAuth after `/login` | Unverified |
| Caches / packages | `models-store.json`, `npm/`, `git/` | Unverified |
| Resume most recent | `pi -c` / `pi --continue` | Unverified |
| Resume specific | `pi --session <path\|id>` | Unverified |
| Fork | `pi --fork <path\|id>`; `/fork`, `/clone` in TUI | Unverified |
| Browse sessions | `pi -r` (TUI) | Unverified |
| Version flag | `pi -v` / `pi --version` | Terminal `0.73.1`; probe `version_raw` empty |
| Self-identification | CLI and RPC set `AI_AGENT=pi` and `PI_CODING_AGENT=true` for child processes | Documented; prefer over binary-name heuristics |

`PI_CODING_AGENT_SESSION_DIR` being a separate override does **not** mean the
default session root is outside `~/.pi/agent`. The published default is
`~/.pi/agent/sessions/`. A T1 scanner must still honor the session-dir
override without treating it as `<config>/sessions`.

## What this task settled

1. **No F2 list API.** Do not spawn `pi --mode rpc` or `--mode json` to
   enumerate sessions. Do not embed the Node SDK.
2. **Default session directory** is published as `~/.pi/agent/sessions/`,
   organized by working directory.
3. **Sessions are project-scoped** by a cwd slug. `ProjectKey` is `path_slug`
   if a reader is written. Do not invent a global bucket.
4. **Process detection** should use `PI_CODING_AGENT=true` and `AI_AGENT=pi`
   before the `pi` image name.
5. **HTML exports** are caller-pathed. Exclude `**/*.html` from discovery
   anyway so an export dropped into the tree is not a session.
6. **`pi --version`** printed `0.73.1` in the terminal; the probe left
   `version_raw` empty. Capture the parser input on the next run.

## What remains blocked

| Item | Why |
| ---- | --- |
| T1 | Dual-platform probes committed; no reader |
| Windows path slug | Drive letters and `\` are unpublished |
| Session-dir override on disk | `PI_CODING_AGENT_SESSION_DIR` vs `--session-dir` vs `settings.json` `sessionDir` not probed |
| T3 version range | Pi publishes on roughly a daily cadence (npm `0.84.2` as of 2026-08-14; mise reports ~255 releases, ~1 day average). A narrow pin will rot in weeks. **Maintainer decision — do not guess a range** |

## Tier path

| Tier | Blocker |
| ---- | ------- |
| T1 | Needs an index source; dual-platform probes exist |
| T2 | Record format beyond the header line unparsed; unknown-layout and truncation policy untested |
| T3 | Needs a captured `pi --version` probe shape, dual-platform resume journeys, and a maintainer version-range policy |

T4 and T5 are out of scope for `v0.5.1`.

## Notes for a future reader

- Header `type=session` is metadata, not a turn. Later lines are a tree, not a
  linear log. Compaction and abandoned branches stay in the same file.
- `custom` entries are extension state and are not LLM context.
- `auth.json` is credentials. It is excluded before any read.
- There is no transcript translation path. Cross-agent work stays an explicit
  portable handoff.

## Sources

- [pi.dev](https://pi.dev/)
- [earendil-works/pi](https://github.com/earendil-works/pi)
- [packages/coding-agent README](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/README.md)
- [Session file format](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/session-format.md)
- [RPC mode](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc.md)
- [Environment variables](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/environment-variables.md)
- [Settings (sessionDir precedence)](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md)
- [Providers (auth.json)](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md)
- [npm @earendil-works/pi-coding-agent](https://www.npmjs.com/package/@earendil-works/pi-coding-agent)
