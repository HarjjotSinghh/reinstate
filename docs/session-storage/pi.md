# Pi coding agent (earendil-works)

**Confidence: Unverified** — no committed AGENT-PROBE-V1, no reader.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T3

Pi is an open-source terminal harness. Vendor documentation now describes a
local JSONL session tree, but vendor documentation is not evidence. This
executor had no native Windows host and no `pi` on PATH, so dual-platform
probes were not committed. T1 and above stay closed.

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
| Fail-closed version pin | `0.73.1`–`0.73.1` (latest `@mariozechner/pi-coding-agent` on 2026-08-16). Still T0 until dual-platform probes. |

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
| Config root default | `~/.pi/agent` | Unverified |
| Session storage override | `PI_CODING_AGENT_SESSION_DIR` (overridden by `--session-dir`, then `sessionDir` in settings.json) | Unverified |
| Session storage default | `~/.pi/agent/sessions/` (under the config root, not a sibling of it) | Unverified |
| Session path | `sessions/--<cwd-with-slashes-as-hyphens>--/<timestamp>_<uuid>.jsonl` | Unverified |
| Session format | JSONL; first line `type=session` header (`version`, `id`, `cwd`); later lines tree entries with `id` / `parentId` | Unverified |
| Header versions | v1 linear (legacy), v2 tree, v3 `hookMessage` → `custom`; load migrates to v3 | Unverified |
| Project scoping | Yes: one cwd-slug directory per working directory | Unverified |
| Project key | Path slug (`/` → `-`, wrapped in `--…--`). Windows encoding unknown | Unverified |
| HTML / JSONL export | `/export [file]`, `--export <in> [out]`, RPC `export_html` write to a caller path (RPC default `/tmp/session.html`), not into the session tree | Unverified |
| Credentials | `~/.pi/agent/auth.json` (mode `0600`); OAuth after `/login` | Unverified |
| Caches / packages | `models-store.json`, `npm/`, `git/` | Unverified |
| Resume most recent | `pi -c` / `pi --continue` | Unverified |
| Resume specific | `pi --session <path\|id>` | Unverified |
| Fork | `pi --fork <path\|id>`; `/fork`, `/clone` in TUI | Unverified |
| Browse sessions | `pi -r` (TUI) | Unverified |
| Version flag | `pi -v` / `pi --version` | Unverified — output shape not captured |
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
6. **`pi --version` output shape** was not captured. This host had no `pi` on
   PATH.

## What remains blocked

| Item | Why |
| ---- | --- |
| T1 | Dual-platform AGENT-PROBE-V1 required. This session had no native Windows device and no usable local `pi` install |
| Windows path slug | Drive letters and `\` are unpublished |
| Session-dir override on disk | `PI_CODING_AGENT_SESSION_DIR` vs `--session-dir` vs `settings.json` `sessionDir` not probed |
| T3 version range | Pi publishes on roughly a daily cadence (npm `0.84.2` as of 2026-08-14; mise reports ~255 releases, ~1 day average). A narrow pin will rot in weeks. **Maintainer decision — do not guess a range** |

## Tier path

| Tier | Blocker |
| ---- | ------- |
| T1 | Needs committed macOS **and** native Windows probes against a real install with more than one session |
| T2 | Record format unprobed; unknown-layout and truncation policy untested |
| T3 | Needs a captured `pi --version` shape, dual-platform resume journeys, and a maintainer version-range policy |

T4 and T5 are out of scope for `v0.5.0`.

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
