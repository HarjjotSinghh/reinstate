# Roo Code

**Confidence: Documented** for identity and official distribution;
**Unverified** for every session-file path and host root.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Catalog key is `roo`. Descriptor: `internal/agents/catalog/roo.go`.

T-032 targeted T1. Dual-platform AGENT-PROBE-V1 artifacts are required
for T1. This executor has no native Windows host and no Roo Code
extension on the macOS host (`RooVeterinaryInc.roo-cline` is not among
VS Code or Cursor `globalStorage` names; no `roo` on PATH). No probe
JSON is committed. There is no F3 scanner and no reader.

The official extension was shut down on 15 May 2026. Shutdown does not
make the product `desktop_only` or `server_backed`. Existing installs
may still hold local task history. ZooCode is a community fork and is
not this catalog key.

## Identity

| Aspect | Value | Source |
| ------ | ----- | ------ |
| Catalog key | `roo` | this page |
| Vendor | Roo (Roo Code, Inc.) | [Marketplace](https://marketplace.visualstudio.com/items?itemName=RooVeterinaryInc.roo-cline), [License](https://github.com/RooCodeInc/Roo-Code/blob/HEAD/LICENSE) |
| Product | Roo Code | [docs.roocode.com](https://docs.roocode.com/), [roocode.com](https://roocode.com) |
| Official extension | `RooVeterinaryInc.roo-cline` | [VS Marketplace](https://marketplace.visualstudio.com/items?itemName=RooVeterinaryInc.roo-cline), [`src/package.json`](https://github.com/RooCodeInc/Roo-Code/blob/HEAD/src/package.json) (`publisher` + `name`) |
| Official CLI | none | Marketplace and docs describe an editor extension only |
| Official repository | [RooCodeInc/Roo-Code](https://github.com/RooCodeInc/Roo-Code) (archived 15 May 2026) | Marketplace and docs |
| Distribution | Official marketplace extension (discontinued) | same |
| Storage family | F3 expected (`FamilyEmbeddedDB`) | Official docs: default is "the standard VS Code extension storage location" |

One catalog entry. Roo originated as a Cline fork. That is origin, not
identity. A later probe that finds leftover Cline-shaped filenames
must still attribute records to `roo` when they live under the Roo
extension id.

## Why T0 is `layout_unverified`

The product, extension id, and official docs are settled. Official
docs also say task history is local, so this is not `server_backed`
and not `no_local_history`. It is an editor extension, not a desktop
ADE, so this is not `desktop_only`.

What is not settled is the live conversation layout on macOS and
native Windows: which host root is authoritative, which file is turns
versus UI state, whether the workspace path is recorded, and whether
`roo-cline.customStoragePath` is in use.

That is `layout_unverified`. Vendor documentation is not a tier
promotion. One-platform evidence would not be enough even if this
host had Roo Code installed.

## Cline collision

Answered independently of Cline's page and of the shared-origin
hypothesis.

**Default trees do not collide.** VS Code-compatible hosts give each
extension its own `User/globalStorage/<publisher>.<name>/` directory.

| Product | Marketplace id | Default leaf (claimed, Unverified) |
| ------- | -------------- | ---------------------------------- |
| Roo Code | `RooVeterinaryInc.roo-cline` | `…/User/globalStorage/rooveterinaryinc.roo-cline/` |
| Cline | `saoudrizwan.claude-dev` | `…/User/globalStorage/saoudrizwan.claude-dev/` |

Roo vendor source (`src/utils/storage.ts`) takes the extension's own
`globalStoragePath` and optionally replaces it with
`roo-cline.customStoragePath`. It does not name Cline's extension id,
`~/.cline`, or `CLINE_DATA_DIR`. Cline official docs do not name
Roo's extension id.

They share a parent (`User/globalStorage`) and not a leaf. A scanner
that walked the parent would mix products. A later F3 scanner must
key on the extension id (and on `roo-cline.customStoragePath` when
set). Shared origin is not a shared current store.

**Collision is possible only by user override.** If someone points
`roo-cline.customStoragePath` at Cline's tree (or the reverse), the
trees overlap. That is misconfiguration, not the default layout.
Until a probe sees that override, do not treat the two products as
one store.

A `~/.roo` name on this host is a skills directory, not a Roo Code
session store, and is not Cline's `~/.cline` hub.

## Claimed layout (all Unverified)

Official docs and archived vendor source currently describe one
surface. This is not a support claim.

### Editor-host globalStorage (vendor-documented class; path Unverified)

| Aspect | Official / source claim | Notes |
| ------ | ----------------------- | ----- |
| Default root | "standard VS Code extension storage location" | Docs do not print the OS path. VS Code resolves this to the host's `globalStorageUri` for `RooVeterinaryInc.roo-cline`. |
| Override | `roo-cline.customStoragePath` / command `roo-cline.setCustomStoragePath` | VS Code setting, not an environment variable. Absolute path required. Empty string means default. |
| Per-task directory | `<base>/tasks/<taskId>/` | From archived `src/utils/storage.ts` `getTaskDirectoryPath` |
| Settings directory | `<base>/settings/` | Same file; treat as secrets until a probe says otherwise |
| Cache directory | `<base>/cache/` | Same file; not a transcript |
| Per-task files | "Each task's history is now stored in its own file" (v3.49.0 notes) | Filenames unpublished. Do not assume Cline's `api_conversation_history.json` / `ui_messages.json`. |
| Concurrent writer | "cross-process file locking" (v3.49.0 notes) | Editor may be running. Future scanner: read-only, no lock of our own. |
| History UI | Command `roo-cline.historyButtonClicked` | In-editor surface, not a machine-readable list API |
| Settings export | `roo-code-settings.json` | Documented to contain API keys in plaintext. User-chosen path. |

### Hosts (claimed, Unverified)

`<Host>` is Code, `Code - Insiders`, VSCodium, Cursor, or Windsurf —
each a separate tree. Official docs do not enumerate hosts. A later
probe must confirm each installed host independently.

## Family stays F3, not F2

There is no official Roo Code session CLI and no documented
machine-readable history command. The in-editor History button is
not a supported machine API.

Until a probe captures a machine-readable session list, Roo stays F3
expected (per-host extension storage). Do not write an F2 `cliquery`
wrapper from the Cline fork relationship.

## F3 properties (unchanged by shutdown)

1. **Multiple hosts, multiple roots.** A record must name the host.
   Two hosts' tasks are otherwise indistinguishable.
2. **No version probe from PATH.** There is no official binary.
   An on-disk extension version marker is unconfirmed.
3. **No resume argv.** Opening the editor is not the current launch
   mechanism. T3 is not reachable through that mechanism. That is a
   property, not a later task.
4. **Concurrent writer.** The editor may be running. A future F3
   scanner must open read-only, take no lock, and fail closed on an
   unknown schema version.

## Secrets in the same tree

If a later task ever reads a host `globalStorage` root or
`roo-cline.customStoragePath`, put these in `Excluded` **before**
any read:

- `settings/` — vendor source stores settings next to tasks; export
  docs say API provider profiles include keys
- editor `SecretStorage` / keychain material (not files)
- any `roo-code-settings.json` found under the storage root

## What this task settled

1. Official product identified. Catalog key `roo` is that product.
2. Dual-platform probes unavailable. T1 is closed.
3. No F3 scanner. T-031 did not ship `scan/embeddeddb`. This task
   does not create one.
4. Default Roo and Cline trees do not collide. Attribute by
   extension id. Custom-path overlap is the only collision case.
5. Workspace / project attribution is unknown. `ProjectKey` is `none`
   until a probe shows a recorded path.

## What a later probe must settle

1. The storage root for each installed host, on macOS **and** native
   Windows, plus whether `roo-cline.customStoragePath` is set.
2. Per-task directory shape, and whether the id is stable across
   restarts.
3. Which file carries user-visible turns, and which is UI-render
   state. Do not parse both. Do not assume Cline filenames.
4. Whether the workspace or repository path is recorded.
5. Whether an on-disk schema / extension version marker exists.
6. Whether the extension exposes its own export or history surface
   that is a machine-readable list. If it is, prefer it over private
   files.

A probe is required to leave T0. Empty-install inventory is not a
probe.

## Sources

- [Roo Code docs](https://docs.roocode.com/)
- [Settings management (`customStoragePath`, export keys, reset)](https://docs.roocode.com/features/settings-management)
- [v3.49.0 notes (per-task history file, cross-process locking)](https://docs.roocode.com/update-notes/v3.49.0)
- [VS Marketplace: `RooVeterinaryInc.roo-cline`](https://marketplace.visualstudio.com/items?itemName=RooVeterinaryInc.roo-cline)
- [Official repository (archived)](https://github.com/RooCodeInc/Roo-Code)
- [Archived `src/package.json` (publisher, name, commands)](https://github.com/RooCodeInc/Roo-Code/blob/HEAD/src/package.json)
- [Archived `src/utils/storage.ts` (base path, `tasks/`, `settings/`, `cache/`)](https://github.com/RooCodeInc/Roo-Code/blob/HEAD/src/utils/storage.ts)
