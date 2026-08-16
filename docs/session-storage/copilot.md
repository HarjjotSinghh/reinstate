# GitHub Copilot CLI

**Confidence: Unverified** — catalog descriptor exists; no index source, no
reader, no committed probe. Vendor documentation is recorded below; it is
not a T1 gate.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1 if a
later probe shows local `session-state/` is authoritative; otherwise T0 with
reason `server_backed`

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

## The question that decides the tier

Three cases look identical to a naive scanner. A later probe (T-006) must
distinguish them by observing the tree across a **cache-directory clear** and
a **re-login**. Vendor documentation is not that observation.

1. **Local authoritative history** — files or a database on disk that survive
   a reinstall and a re-login. T1 is reachable.
2. **Local cache of server state** — files that exist but are rebuilt from
   the GitHub account and disappear when the cache clears. Not indexable.
   T0.
3. **Nothing local.** T0, reason `server_backed`.

Official docs currently **describe case 1 with an extra account copy**, not
case 3. They also describe a **separate** cache directory that is not the
session tree. That is a strong signal, not a tier promotion.

Vendor statements that frame the three cases (still Unverified):

- Every CLI session is recorded on the machine. By default it is **also**
  synced to the GitHub account.
- Resume of an interactive CLI session is documented as reading
  `~/.copilot/session-state/`.
- Sync can be turned off (`"remoteExport": false`, `--no-remote-export`).
  Copilot Enterprise / Business can leave sessions local-only when the
  "Store local sessions in the Cloud" policy is off.
- `session-store.db` is a rebuildable SQLite index (`/chronicle reindex`).
  Reindexing also syncs to the account. Treat this file as an index, not
  as the store, until a probe says otherwise.
- Platform cache (`~/Library/Caches/copilot`, `%LOCALAPPDATA%/copilot`,
  `$XDG_CACHE_HOME/copilot` or `~/.cache/copilot`; override
  `COPILOT_CACHE_HOME`) is documented as marketplace / auto-update
  ephemera. It is **not** documented as session history. Clearing it is
  the cache-clear half of the probe.
- The session picker has local and remote tabs. `--connect` opens a remote
  session. Cloud-agent work can be brought into the local CLI. Remote-only
  rows are not a local store.

**Recommended later-tier:** stay at T0 `layout_unverified` until T-006.
If `session-state/` survives re-login and a cache-directory clear, and is
not rewritten from the account, promote toward **T1**. If those files
vanish, are empty after logout, or are recreated from GitHub.com, ship
**T0 `server_backed`**. Do not index `session-store.db` or the platform
cache on the strength of directory names.

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
