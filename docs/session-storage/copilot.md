# GitHub Copilot CLI

**Confidence: Layout documented on macOS and native Windows** — T1 index
source reads `session-state/<uuid>/events.jsonl`. A rename-aside probe
showed an old session ID did not return in the fresh tree.
**Current tier:** T1 (discover) · **Phase 5 target:** T2. Resume and fork
stay refused. `session-store.db` / `session.db` are not parsed.

Catalog key remains `copilot`. Descriptor:
`internal/agents/catalog/copilot.go`. This page is GitHub Copilot CLI only —
not the retired `gh copilot` GitHub CLI extension, not VS Code / JetBrains
Copilot chat, and not the Copilot SDK session API.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | GitHub |
| Product | GitHub Copilot CLI |
| Binary | `copilot` |
| Distribution | Official: npm `@github/copilot`, WinGet `GitHub.Copilot`, Homebrew cask `copilot-cli`, install script, [github/copilot-cli releases](https://github.com/github/copilot-cli/releases/) |
| Storage family | F1 recorded from vendor docs (`FamilyHomeTree`); not a probe assignment |
| Root override | `COPILOT_HOME` (vendor; `--config-dir` is documented as deprecated) |
| Version command | `copilot version` |

The retired product is a different binary surface: `gh copilot` from the
`gh-copilot` extension. GitHub docs state that extension is retired and
replaced by GitHub Copilot CLI. Do not treat `gh copilot` as this catalog
entry.

## Device evidence (2026-08-16, macOS arm64)

Artifact:
[`2026-08-16-macos-copilot.json`](../testing/results/agent-probes/2026-08-16-macos-copilot.json)

| Check | Result |
| ----- | ------ |
| `copilot` on PATH | yes |
| `copilot --version` | `GitHub Copilot CLI 1.0.80.` followed by an update-check line |
| Resolved root | `~/.copilot` |
| macOS AGENT-PROBE-V1 | captured |
| native Windows AGENT-PROBE-V1 | absent |
| Cache-clear / re-login observation | **not run** |

```
~/.copilot/
  session-state/<uuid-v4>/
    events.jsonl                    ~70 KB   keys: id, parentId, type, timestamp, data
    checkpoints/index.md
    rewind-file-snapshots/tracking.json      keys: schema, tracking
  sidebar-sessions-state/<64-hex>.json
  command-history-state.json
  hooks/, ide/, installed-plugins/, servers/, logs/
```

The local store is real and substantial: a 70 KB event log with an explicit
`id` / `parentId` chain, checkpoints, and file snapshots for rewind. A naive
scanner would call this case 1 and promote the agent.

## Device evidence (2026-08-17, native Windows amd64)

Artifact:
[`2026-08-17-windows-copilot.json`](../testing/results/agent-probes/2026-08-17-windows-copilot.json)

Same CLI version, `1.0.80`, and the tree is **not** the same shape. SQLite
appears:

```
~\.copilot\
  session-store.db           4 KB      + -shm 32 KB, -wal 463 KB
  session-state\<uuid-v4>\
    session.db               12 KB
    events.jsonl             85 KB     keys: id, parentId, timestamp, type, data
    workspace.yaml           420 B
    checkpoints\index.md
    rewind-file-snapshots\tracking.json
    files\, research\
```

Neither `session-store.db` nor the per-session `session.db` appeared in the
macOS artifact, and a 463 KB write-ahead log means the database is being
written, not carried along as a stub.

**This unsettles the storage family.** The descriptor records `FamilyHomeTree`
from vendor documentation, but an agent with a root-level SQLite store plus a
per-session database is at least partly F3, and a reader that walks JSONL while
ignoring the database may be reading a partial or superseded view. Which of the
two is authoritative is now an open question on top of the one below.

Do not resolve it by guessing. The macOS artifact may simply predate a
migration, or the file may be created lazily on Windows only. A macOS re-probe
on the same version answers it cheaply, and until then the family assignment
stays as documented, with this contradiction recorded against it.

**The cache question is answered for one ID.** On 2026-08-17 the live
`~/.copilot` tree was renamed aside and a new CLI session was started. The
previous `session-state/<uuid-v4>` directory was **absent** from the fresh
tree and **present** in the renamed copy. GitHub did not recreate that ID.
That is local files, not a rebuild-from-account of the same session.
Promoted to T1 on 2026-08-19 by indexing `events.jsonl`. Do not parse
`session.db` or `session-store.db`. Resume and fork stay refused.

Artifact:
[`2026-08-17-windows-copilot-cache-clear.json`](../testing/results/agent-probes/2026-08-17-windows-copilot-cache-clear.json)

`copilot --version` still emits a trailing update-check line; any version
parser must take the first line.

## Claimed layout

Every row below is **Unverified**. Paths are quoted from vendor docs, not
from a device. Do not treat a documented path as a reader input.

| Aspect | Vendor-documented value | Confidence |
| ------ | ---------------------- | ---------- |
| Root default (Unix) | `~/.copilot` (`$HOME/.copilot`) | Unverified |
| Root default (Windows) | `%USERPROFILE%\.copilot` (docs also write `$HOME\.copilot` and `C:\Users\YOUR-USER\.copilot`) | Unverified |
| Root override | `$COPILOT_HOME` relocates the whole tree | Unverified |
| Sessions | `<root>/session-state/<session-id>/` | Unverified |
| Transcript | `events.jsonl` inside each session directory | Unverified |
| Session side files | plans, checkpoints, tracked files (names unspecified) | Unverified |
| Cross-session index | `<root>/session-store.db` (SQLite; rebuildable) | Unverified |
| Input / command history | `<root>/command-history-state/` | Unverified |
| Logs | `<root>/logs/process-{timestamp}-{pid}.log` | Unverified |
| Settings | `<root>/settings.json` | Unverified |
| Auth / app state | `<root>/config.json` | Unverified |
| MCP OAuth fallback | `<root>/mcp-oauth-config/` | Unverified |
| MCP secret fallback | `<root>/mcp-secrets/` | Unverified |
| Platform cache (not sessions) | macOS `~/Library/Caches/copilot`; Windows `%LOCALAPPDATA%/copilot`; Linux `$XDG_CACHE_HOME/copilot` or `~/.cache/copilot` | Unverified |
| Prior root | "Previous XDG-based configuration locations" migrated into `~/.copilot` at startup when `COPILOT_HOME` is unset. Docs do not name the old path. | Unverified |

Native control surface (vendor-documented; still Unverified on a device):

| Aspect | Value |
| ------ | ----- |
| Resume most recent | `copilot --continue` (cwd most recent, else global most recent) |
| Resume picker | `copilot --resume` / `copilot -r` |
| Resume specific | `copilot --resume SESSION-ID` (ID, ID prefix, or session name) |
| Resume exact ID | `copilot --session-id ID` |
| In-session switch | `/resume`, `/resume SESSION-ID` |
| Session id / manage | `/session` (`info`, `rename`, `delete`, `delete-all`, `prune`, …) |
| Remote session | `copilot --connect[=SESSION-ID]` (conflicts with `--resume` / `--continue`) |
| Disable account export | `--no-remote-export` or settings `"remoteExport": false` |
| Export / share | `/share`, `/share file`, `/share html`, `/share gist` |
| Reindex local store | `/chronicle reindex` |

## Authentication material (exclude before any later read)

Do not open these if a reader is ever written. They sit in the same
home-dir tree as session files.

| Location | Why it is excluded |
| -------- | ------------------ |
| `<root>/config.json` | Vendor: authentication data and `loggedInUsers`. Plaintext token fallback when the OS keychain is missing. Deleting it resets authentication. |
| `<root>/mcp-oauth-config/` | MCP OAuth token / PKCE fallback |
| `<root>/mcp-secrets/` | MCP secret placeholders |
| OS keychain service `copilot-cli` | Default OAuth store (macOS Keychain, Windows Credential Manager, Linux libsecret). Not a file; still credential material. |
| `COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, `GITHUB_TOKEN` | Env-var tokens. Not files. |
| `COPILOT_PROVIDER_API_KEY` | BYOK provider key. Not a session file. |

## What the probe must settle

1. Confirm the binary on `PATH` is `copilot` (not `gh copilot`) on macOS and
   native Windows, and capture `copilot version`.
2. Whether `<root>/session-state/<id>/events.jsonl` exists after a real
   interactive session, and whether the project path is recoverable from
   that directory or only from `session-store.db`.
3. **Cache vs store:** delete or empty the **platform cache** directory
   (not `session-state/`). Do the session directories remain? Then
   `/logout` and `/login` (or a fresh login). Do they remain, or are they
   rewritten from the account?
4. Repeat with `"remoteExport": false` so account sync cannot mask a
   missing local store.
5. Whether `session-store.db` is sufficient to list sessions if
   `session-state/` is removed (vendor says no: deleting `session-state/`
   removes the ability to resume).
6. Put `config.json`, `mcp-oauth-config/`, and `mcp-secrets/` in
   `Excluded` before any read.
7. Confirm the documented resume argv on both platforms.

Do not inspect a developer's real Copilot tree while filling this page.

## Sources

- [About GitHub Copilot CLI](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-copilot-cli)
- [Installing GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/install-copilot-cli)
- [Authenticating GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/authenticate-copilot-cli)
- [Using GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/overview)
- [About GitHub Copilot CLI session data](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/chronicle)
- [Using GitHub Copilot CLI session data](https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/chronicle)
- [GitHub Copilot CLI configuration directory](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference)
- [GitHub Copilot CLI command reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference)
- [Using the GitHub CLI Copilot extension](https://docs.github.com/en/copilot/github-copilot-in-the-cli) (retired `gh copilot`)
- [github/copilot-cli](https://github.com/github/copilot-cli)
