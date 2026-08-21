# OpenHands (All Hands AI)

**Confidence: Unverified** — official vendor docs recorded; no Reinstate
reader exists; no redacted device probe. Vendor documentation is not a
tier promotion.
**Current tier:** T0 (`server_backed`) · **Phase 5 target:** T0

Catalog key remains `openhands`.

**Maintainer 2026-08-16:** no network-backed Cloud / Agent Server source in
`v0.5.1`. Stay T0 `server_backed`. CLI `~/.openhands` is server persistence,
not a local session store. Revisit only with an ADR. Descriptor:
`internal/agents/catalog/openhands.go`. No index source, reader, target, or
sync adapter.

## Catalog

| Field | Value |
| ----- | ----- |
| Key | `openhands` |
| Display name | OpenHands |
| Vendor | All Hands AI |
| Tier | T0 |
| T0Reason | `server_backed` |
| Family | F5 |
| Constructors | none |
| Host candidate (not indexed) | `$OH_PERSISTENCE_DIR` or `~/.openhands` |
| Excluded before any read | `settings.json`, `agent_settings.json`, `mcp.json` |

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | All Hands AI |
| Product | OpenHands |
| Official docs | [docs.openhands.dev](https://docs.openhands.dev/overview/introduction) |
| Recommended surfaces | Agent Canvas (local browser client) and OpenHands Cloud |
| Other official surfaces | CLI (`openhands`), Docker / `openhands serve` GUI, Agent Server, Enterprise |
| Binaries (documented) | `openhands`, `agent-canvas` |
| Distribution | Official, open source plus hosted Cloud / Enterprise |
| Storage family | F5 (server-backed) |

## Research outcome

**T0, reason `server_backed`.** That is the completed result for T-042.

OpenHands is a service. Conversations are owned by an Agent Server (or Cloud /
Enterprise control plane), not by a user-facing local session tree. Agent
Canvas is a browser client: it presents backend state and is not the store.
A container filesystem is not the host's. A hosted deployment is not local.

A documented host directory exists for some local deployments. It is the
backend's persistence directory (`OH_PERSISTENCE_DIR`, default
`~/.openhands`), not a first-class local session store. Indexing it would
mean reading concurrent server state, and records would vanish when the
backend is a Cloud or remote backend, or when a Docker run omits the
volume. That is the cache / server-state trap. Do not stretch to T1.

## Why T0 is `server_backed` rather than `layout_unverified`

| Question | Official claim | Consequence |
| -------- | -------------- | ----------- |
| Who owns conversation history? | Agent Server stores conversation history, profiles, secrets, and MCP config. Canvas stores only connection info. Switching backends switches which conversations are visible. | F5. History is backend-owned. |
| What is a conversation? | "A single agent session on the active backend." Archive keeps full history on the backend. | Same. |
| Default product | Quick start recommends Agent Canvas or Cloud, not a terminal session file. | The catalog entry is this product, not a CLI-only subset. |
| Docker without a volume | Official `docker run` uses `--rm`. Without the documented bind-mount, the container is ephemeral. | Not a session store. |
| Cloud / remote / Enterprise | No host transcript. Cloud conversations are created and listed over HTTPS. | `server_backed`. |

A later CLI-only probe cannot promote this key. The same `openhands` entry
covers Canvas, Cloud, and Docker users who have nothing local.

## Documented host-side artifacts

Every row below is **Unverified**. No probe has observed the tree. Do not
treat these as a support claim.

| Surface | Claimed host path | In-container path (if any) | What the vendor says it holds |
| ------- | ----------------- | -------------------------- | ----------------------------- |
| V1 persistence override | `$OH_PERSISTENCE_DIR` | — | Local state; default `~/.openhands` |
| Default Docker GUI (`openhands serve` / documented `docker run`) | `~/.openhands` | `/.openhands` | Bind-mounted so the `--rm` container can keep state |
| Agent Canvas Docker | `~/.openhands` (Windows: `%USERPROFILE%\.openhands`) | `/home/openhands/.openhands` | Settings and conversation data, only when this mount is used |
| CLI (native / uv) | `~/.openhands/conversations/<id>/conversation.json` | — | Conversation history; deleting the directory loses it |
| CLI Docker | `~/.openhands` | `/root/.openhands` | Same host tree, third in-container path |
| CLI settings (install page) | `~/.openhands/settings.json` | — | LLM configuration |
| CLI settings (command reference) | `~/.openhands/agent_settings.json` | — | LLM configuration and agent settings, including `api_key` |
| CLI other | `~/.openhands/cli_config.json`, `~/.openhands/mcp.json` | — | CLI prefs; MCP server configs |
| SDK (opt-in, not a user default) | caller-supplied `persistence_dir` (example `./.conversations`); default if unset is `workspace/conversations/` | — | `base_state.json` plus `events/event-*.json` |

The three Docker in-container paths are different (`/.openhands`,
`/home/openhands/.openhands`, `/root/.openhands`). The host path is the
same claim: `~/.openhands`. The two CLI settings filenames conflict
(`settings.json` vs `agent_settings.json`). Both are recorded; neither is
reconciled here.

npm / npx Agent Canvas does not document a bind-mount. Uninstall docs say
persisted data is not removed with the package and name `~/.openhands` as
the example directory to delete. That is not a layout.

The descriptor's discovery marker is `conversations`, taken from the CLI row
above and therefore also unverified. Its job is not to find sessions — this
agent is T0 and has no reader — but to stop a bare `~/.openhands` from
resolving at all. On a machine with no OpenHands installed, a skill installer
had already created `~/.openhands/skills`, and an unmarked root reported that
as a live persistence directory.

## Restart and upgrade

| Event | Official claim | Session store? |
| ----- | -------------- | -------------- |
| Docker `docker run --rm` **without** the `~/.openhands` mount | Container and its writable layer are discarded | No |
| Docker / Canvas update **with** the documented mount | Settings and conversation data live outside the image; uninstall does not delete that directory | Vendor-claimed persistence of **server** state. Unverified. Still F5. |
| Agent Server restart | `OH_SECRET_KEY` must stay stable or encrypted values stored with conversations cannot be restored | Persistence is keyed to the server process, not to a user home layout |
| CLI upgrade from before 1.0.0 | Settings format changed; users must redo setup | Version upgrade is not a guaranteed no-op even on the documented host tree |
| Canvas / Cloud / remote backend | History stays on that backend | Not a host session store |

The documented default Docker command includes `-v ~/.openhands:…`. That
answers "does a host-side artifact exist for a default local Docker
deployment?" with **yes, claimed**. It does **not** make that directory a
Reinstate session store. It is the server's disk.

## Project path

Not recoverable as a first-class project key from any documented host file.

| Surface | What docs give | Recoverable project path? |
| ------- | -------------- | ------------------------- |
| Agent Canvas | User chooses **Open Workspace** before starting. Conversation is backend state. | Unverified. Not documented as a field on disk. |
| Docker sandbox | `openhands serve --mount-cwd` or `SANDBOX_VOLUMES=host:container[:mode]` | Mount is an execution workspace, not a session index key. |
| Cloud API | `selected_repository` (`owner/repo`) on the conversation resource | Remote metadata, not a local path. |
| CLI `conversation.json` | Docs show only the filename under `~/.openhands/conversations/<id>/` | **Not documented.** Do not invent keys. |
| SDK persistence | "Workspace Context: Working directory and file system state" inside `base_state.json` | SDK opt-in only. Unverified shape. |

## Native resume

| Surface | Documented continue | Notes |
| ------- | ------------------- | ----- |
| CLI | `openhands --resume` lists up to 15; `openhands --resume <id>`; `openhands --resume --last` | Same-vendor CLI resume. Also `openhands acp --resume …` for IDEs. |
| Agent Canvas / Cloud | Continue inside the UI on the active backend | No terminal argv. |
| Agent Server | HTTP / WebSocket conversation APIs | Service, not a local CLI. |

There is no "resume this OpenHands conversation in your terminal" path for
the recommended Canvas / Cloud product. CLI resume does not change the
catalog tier.

## Documented conversation-export APIs

These do **not** change the tier. They are recorded because they are the
only later path that could, and only as a **maintainer** decision: every
current catalog source is local and offline.

| Interface | What it is | Auth / notes |
| --------- | ---------- | ------------ |
| Agent Canvas **Export transcript** | Download `conversation-<id>.md` or `.html` from the kebab menu | Generated in the browser from events the app already has. Cloud export uses currently loaded events. Options: include tool details, include timestamps. |
| OpenHands Cloud REST | `POST /api/v1/app-conversations`, `GET /api/v1/app-conversations`, `GET /api/v1/app-conversations/search`, start-task polling | `Authorization: Bearer`. V0 `POST /api/conversations` is deprecated, removal 2026-04-01. List/status, not a full transcript dump in the overview. |
| Sandbox Server V1 | `POST/GET /api/v1/app-conversations` plus sandbox start/pause/resume | Control plane, no frontend. |
| Local Agent Server | `GET /api/conversations/count` and generated `/api/*` conversation routes; OpenAPI at `/docs` | `X-Session-API-Key`. `OH_SECRET_KEY` encrypts secrets stored with conversations. |
| SDK `persistence_dir` | Read `base_state.json` and `events/event-*.json` | Developer-configured directory. Not a default user store. Includes secrets. |

Do not implement a network source from this page.

## Secrets in the same tree

If a later task ever reads `~/.openhands`, these go in `Excluded` before
any read:

- `settings.json` / `agent_settings.json` — documented to hold `api_key`
- `OH_SECRET_KEY` material — encrypts LLM keys and secrets stored with
  conversations
- `LOCAL_BACKEND_API_KEY`, `OH_SESSION_API_KEYS_*`
- `mcp.json` may hold remote MCP headers

## What a later probe must not do

A probe is **not** required to keep this agent at T0. If one is captured
anyway:

1. Do not inspect a contributor's real `~/.openhands` or Docker volumes.
   Use a throwaway machine or a synthetic container.
2. Confirm whether a default `agent-canvas` (npm/npx, no explicit mount)
   writes `~/.openhands` on the host, and whether that tree is history or
   settings-only.
3. Observe the documented Docker mount across container `--rm` recreate
   **and** an image tag bump. If history disappears, it is not a store.
4. Do not parse `conversation.json` or `base_state.json` into a reader.
   Fail closed on unknown layout.
5. Do not promote the key off T0 because the CLI tree exists.

## Sources

- [Introduction](https://docs.openhands.dev/overview/introduction)
- [Quick start](https://docs.openhands.dev/overview/quickstart)
- [Local setup / Docker GUI](https://docs.openhands.dev/openhands/usage/run-openhands/local-setup)
- [V1 configuration (`OH_PERSISTENCE_DIR`)](https://docs.openhands.dev/openhands/usage/advanced/configuration-options)
- [Install Agent Canvas](https://docs.openhands.dev/openhands/usage/agent-canvas/setup)
- [Agent Canvas architecture (state ownership)](https://docs.openhands.dev/openhands/usage/agent-canvas/architecture)
- [Agent Canvas conversations (export, archive)](https://docs.openhands.dev/openhands/usage/agent-canvas/conversations)
- [CLI installation](https://docs.openhands.dev/openhands/usage/cli/installation)
- [CLI resume and `~/.openhands/conversations`](https://docs.openhands.dev/openhands/usage/cli/resume)
- [CLI command reference (resume argv, config files)](https://docs.openhands.dev/openhands/usage/cli/command-reference)
- [Docker sandbox mounts](https://docs.openhands.dev/openhands/usage/sandboxes/docker)
- [Cloud API](https://docs.openhands.dev/openhands/usage/cloud/cloud-api)
- [Sandbox Server V1 REST](https://docs.openhands.dev/openhands/usage/api/v1)
- [Local Agent Server](https://docs.openhands.dev/sdk/guides/agent-server/local-server)
- [SDK conversation persistence](https://docs.openhands.dev/sdk/guides/convo-persistence)
