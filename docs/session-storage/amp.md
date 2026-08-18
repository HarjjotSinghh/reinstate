# Amp (Sourcegraph)

**Confidence: Unverified** — official sources establish that threads are
authoritative on Amp Server; no Reinstate reader exists, and no device probe
has classified any local tree.
**Current tier:** T0 (`server_backed`) · **Recommended later outcome:**
T0 (`server_backed`). Do not promote to T1 on documentation alone.

Catalog key remains `amp`.

**Maintainer 2026-08-16:** no network-backed source in `v0.5.0`. Stay T0
`server_backed`. A reader that calls ampcode.com / `amp threads list` would
be the catalog's first online source (auth, timeouts, offline behavior, and
the Enterprise API can `DELETE` threads). Revisit only with an ADR.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Amp (ampcode.com; product originated at Sourcegraph) |
| Product | Amp |
| Binary | `amp` |
| Distribution | Official CLI, editor integrations (VS Code family, Neovim, Zed), and web at ampcode.com |
| Install | `curl https://ampcode.com/install.sh`, Homebrew `ampcode/tap/ampcode`; npm `@ampcode/cli` is documented and not recommended |
| Storage family | F5 (`FamilyRemote`) expected |
| Thread identity | `T-<uuid>` (example: `T-f9941a55-3765-421e-972f-05dc1138c3a3`) |

## Research outcome

**T0 `server_backed`.** Amp markets thread sharing, cross-device continuation,
remote control, and workspace collaboration. The vendor's own security
reference states that Amp Server stores conversations in PostgreSQL on GCP and
handles "thread syncing and storage". There is no self-hosted deployment.

That is the three-case answer from vendor documentation: if a local tree
exists, it is a cache of server state (case 2), not a local authoritative
store (case 1). Indexing it would produce records that vanish on cache clear,
re-login, or another device. T1 is not reachable from local files.

A documented thread-list and thread-message interface exists, but it is
**network-backed** (Enterprise HTTP API, plus CLI/SDK commands that talk to
ampcode.com). Every source in the catalog today is local and offline. Using
those interfaces would be a first. **Escalate; do not implement a network
reader.**

## Why threads are server-authoritative

Quoted or paraphrased only from official Amp pages:

1. Owner's Manual principle: "Threads: You can save and share your
   interactions with Amp."
2. Workspace thread sharing is a first-class feature. Visibility levels are
   unlisted, workspace-shared, group-shared (Enterprise), and private. Threads
   are listed at [ampcode.com/feed](https://ampcode.com/feed) and addressed as
   `https://ampcode.com/threads/T-<uuid>`.
3. Remote control continues a running CLI thread from ampcode.com. Orbs run
   threads on remote machines. Slack `@Amp` finds and starts threads through
   the account.
4. Security Reference, System Components: Amp Server "handles authentication,
   user accounts, workspaces, thread syncing and storage, and usage tracking."
   The client "provides local code and context management, local settings, and
   local thread history" and talks to Amp Server.
5. Security Reference, User Interaction: "Amp Server stores these
   conversations in its PostgreSQL database in Google Cloud Platform, which
   enables team collaboration features such as thread sharing and auditing."
6. Security Reference: "Amp doesn’t offer self-hosted deployments."
7. Thread deletion, workspace ownership, and Minimal Data Retention are
   defined as server-side policies. MDR "does not control how Amp stores
   thread data."
8. The official examples guide states: "Your threads sync to ampcode.com,
   allowing you to continue conversations across devices."
9. The Enterprise External API thread object includes `firstSyncedAt` ("When
   the thread was first synced").

Taken together, the server copy is the product. The client's "local thread
history" is described in the same paragraph as a client that syncs with
ampcode.com, and secret redaction mentions a "local cache". That is the cache
trap, not a T1 store.

## Documented thread-list and export interfaces

These exist. They are not a local, offline source.

### Amp CLI and SDK (network)

Owner's Manual documents:

| Command | Role |
| ------- | ---- |
| `amp threads continue --execute "…" --stream-json` | Continue a thread and stream JSON |
| `amp threads report <thread>` | Support diagnostic (URL or ID), not a session export |

The official examples repo (`ampcode/amp-examples-and-guides`) additionally
documents:

```
amp threads new
amp threads continue [threadId]
amp threads list
amp threads fork [threadId]
amp threads share [threadId]
amp threads compact [threadId]
```

The Amp SDK documents thread continuity: `options.continue: true` (latest) or
`options.continue: "T-…"`. Stream-JSON `session_id` values are thread IDs.

`amp threads list` is a documented list interface. It is still a call into
the user's Amp account, not a walk of a durable local tree. Treating it as
`FamilyCLIQuery` (F2) would make Amp the first network-backed catalog source.

### Amp External API (Enterprise, network)

Published OpenAPI at [ampcode.com/api/v2/openapi.json](https://ampcode.com/api/v2/openapi.json).
"Currently, this API is only available to Amp Enterprise customers."

| Method | Path | Operation |
| ------ | ---- | --------- |
| `GET` | `/api/v2/threads` | List workspace threads (includes private and group-shared); filter and paginate; ordered by first sync |
| `GET` | `/api/v2/threads/{threadID}/messages` | Thread messages; "Message shapes are currently not stable" |
| `GET` | `/api/v2/threads/{threadID}/usage` | Usage for threads younger than 90 days |
| `DELETE` | `/api/v2/threads/{threadID}` | Delete a thread |

Auth is OAuth2 client-credentials against `https://auth.ampcode.com/oauth2/token`.
Scopes include `amp.api:workspace.threads.meta:view` and
`amp.api:workspace.threads.contents:view`.

This is a real thread-list and thread-export API. It is also a network source,
Enterprise-only, credentialed, and write-capable (`DELETE`). Out of scope for
this phase. Recorded for the maintainer; not a reader design.

There is no documented local-only thread-export file format.

## Candidate local artifacts (all Unverified)

No device probe in this pass (T-006). Do not inspect a real Amp tree. Rows
below are candidates for a later redacted probe, not a layout claim.

| Aspect | Path | Role | Confidence |
| ------ | ---- | ---- | ---------- |
| Credentials | `~/.local/share/amp/secrets.json` (macOS/Linux); `%USERPROFILE%\.local\share\amp\secrets.json` (Windows) | Access token | **Documented** — **Excluded** before any read |
| MCP OAuth | `~/.amp/oauth/` | OAuth tokens | **Documented** — **Excluded** |
| User settings | `~/.config/amp/settings.json` or `settings.jsonc`; Windows `%USERPROFILE%\.config\amp\` | Config, not transcripts | **Documented** |
| Settings override | `$AMP_SETTINGS_FILE` | Alternate settings path | **Documented** |
| Personal guidance | `~/.config/amp/AGENTS.md` | Instructions, not sessions | **Documented** |
| Skills / plugins / checks | `~/.config/amp/{skills,plugins,checks}/` | Local extras | **Documented** |
| Project config | `.amp/settings.json`, `.amp/plugins/` | Workspace config | **Documented** |
| Managed policy | `/Library/Application Support/ampcode/managed-settings.json`; Linux `/etc/ampcode/`; Windows `%ProgramData%\ampcode\` | Enterprise policy | **Documented** |
| Local thread history | Path **not published**. Security Reference says the client keeps "local thread history" and a "local cache" | Unknown; treat as cache until a probe proves otherwise | **Unverified** |
| Community thread mirror | `~/.local/share/amp/threads/T-*.json` (and `%USERPROFILE%\.local\share\amp\threads\`) | Third-party report of a per-thread JSON mirror | **Observed** — probe only |
| Community logs | `~/.cache/amp/logs/` | Third-party report | **Observed** — probe only |

XDG: if `XDG_CONFIG_HOME` is set, plugins live under `$XDG_CONFIG_HOME/amp/`.
Probe should also check `$XDG_DATA_HOME/amp` and `$XDG_CACHE_HOME/amp`.

**Excluded** before any later read: `secrets.json`, `~/.amp/oauth/`,
`AMP_API_KEY`, and any file that looks like a token.

## What the T-006 probe must settle

1. Whether any of the candidate roots exist on macOS and native Windows.
2. Whether `~/.local/share/amp/threads/` (or a sibling) is present, and
   whether it survives a cache clear and a re-login. If it is rebuilt from
   the account, it is not indexable.
3. Whether credentials sit in the same tree as any thread files. If yes,
   they go in `Excluded` first.
4. Whether `amp threads list` works offline. If it fails without
   ampcode.com, it is not a local source.
5. Whether a documented resume argv (`amp threads continue [threadId]`)
   identifies a local file or only a server thread. Same-vendor resume
   does not change the T0 recommendation.

A probe that finds a local mirror does **not** promote Amp to T1. Vendor
docs already say the server copy is authoritative.

## Network-source escalation

Raised for the maintainer, not implemented:

- Amp only exposes durable history through ampcode.com: the web feed, the
  CLI/SDK thread commands, and the Enterprise External API.
- A catalog source that calls those would be the first network-backed
  source. It implies auth (`AMP_API_KEY` / OAuth), timeouts, offline
  behavior, Enterprise-only coverage for the HTTP API, and a security-model
  review (the API can `DELETE` threads).
- Recommendation: ship Amp at T0 `server_backed`. Do not add a network
  reader in this phase.

## Recording a T0 outcome

After the descriptor lands, `rein doctor --agents` should tell the user:

> Amp detected. Session history is not readable locally
> (`server_backed`). Reinstate cannot index Amp threads.

That is a complete answer. Stretching to T1 by indexing a cache is a defect.

Native resume remains same-vendor (`amp threads continue`). Cross-agent work
is an explicit portable handoff only. There is no transcript translation at
any tier.

## Sources

Official:

- [Owner's Manual](https://ampcode.com/manual) — thread sharing, remote
  control, `amp threads continue`, local config paths
- [Security Reference](https://ampcode.com/security) — server PostgreSQL
  thread store, "thread syncing and storage", "local thread history",
  `secrets.json`
- [Appendix](https://ampcode.com/manual/appendix) — `amp threads report`,
  stream-JSON `session_id`, workspace visibility, MDR vs thread data
- [Amp SDK](https://ampcode.com/manual/sdk) — `continue` by latest or thread ID
- [Amp External API OpenAPI](https://ampcode.com/api/v2/openapi.json) —
  Enterprise `GET /api/v2/threads` and `GET /api/v2/threads/{id}/messages`
- [Amp CLI Guide](https://github.com/ampcode/amp-examples-and-guides/blob/main/guides/cli/README.md)
  — "threads sync to ampcode.com"; `amp threads list` / `continue` / `share`

Community, probe hints only:

- [Where Six AI Coding CLIs Store Your Session Logs](https://allaboutcoding.ghinda.com/where-ai-coding-clis-store-session-logs/)
  — reports `~/.local/share/amp/threads/T-*.json` as a local mirror
