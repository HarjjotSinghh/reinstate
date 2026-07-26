# Adapters

Adapters translate between each coding agent's **on-disk layout** and
Reinstate's normalized sync model.

> Status: interfaces and docs first; implementations land per [ROADMAP.md](https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md).

## Support matrix

| Agent | Sessions | Config / MCP / skills | Path remap | Resume command | Status |
| ----- | :------: | :-------------------: | :--------: | -------------- | ------ |
| **Claude Code** | ✅ RC4 candidate | 📋 Post–Phase 1 | ✅ critical | `claude --resume SESSION_ID` | 🧪 Acceptance |
| **OpenAI Codex CLI** | ✅ RC4 candidate | 📋 Post–Phase 1 | ✅ | `codex resume SESSION_ID` | 🧪 Acceptance |
| **Gemini CLI** | 📋 Phase 1 | 📋 | ✅ | `gemini --resume` | 📋 Planned |
| **OpenCode** | 📋 Phase 1 | 📋 | ✅ | in-app session list | 📋 Planned |
| **Grok Build** | 📋 Phase 2 | 📋 | ✅ | `grok -r` / `/resume` | 📋 Planned |
| **Cursor** | 💭 best-effort | 💭 | 💭 | N/A (IDE) | 💭 Exploring |

Legend: ✅ designed · 🚧 building · 📋 planned · 💭 exploring

## Per-agent notes

### Claude Code

| | |
| --- | --- |
| **Store** | `~/.claude/projects/<munged-path>/<session-uuid>.jsonl` |
| **Format** | Append-only JSONL (`type`, `uuid`, `cwd`, `gitBranch`, …) |
| **Hard part** | Project dir is a **lossy munge** of absolute path; every line embeds `cwd` |
| **Exclude** | Credentials, plugin caches, machine-local logs |

Windows ↔ macOS path rewrite inside JSONL content is the MVP differentiator.
RC4 also maps the snapshot's canonical project ID to the destination device's
`local_root`, recomputes Claude's vendor directory key, and verifies the exact
planned restore path before reporting success.

### Codex CLI

| | |
| --- | --- |
| **Store** | `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` + state SQLite index |
| **Format** | JSONL rollout + `session_meta` |
| **Hard part** | Large histories (multi-GB); index schema versions (`state_N.sqlite`) |
| **Exclude** | Auth / API credential material |

Prefer delta/CAS sync; never full-file reupload of 300MB rollouts.

### Gemini CLI

| | |
| --- | --- |
| **Store** | `~/.gemini/tmp/<project_hash>/chats/session-*.json` |
| **Format** | Single JSON per session |
| **Hard part** | `project_hash` from absolute root path |
| **Exclude** | Treat carefully — `tmp/` naming vs durable intent |

### OpenCode

| | |
| --- | --- |
| **Store** | `~/.local/share/opencode/project/<slug>/storage/` |
| **Format** | JSON session/message/part records |
| **Hard part** | Project slug mapping |
| **Exclude** | **`auth.json` must never sync** |

### Grok Build

| | |
| --- | --- |
| **Store** | under `~/.grok/` (format evolving) |
| **Resume** | named sessions / TUI `/resume` |
| **Hard part** | Fast-moving open-source CLI — expect churn |
| **Exclude** | `auth.json`, machine-local config secrets |

## Adapter interface (target)

```go
// Package adapter — simplified sketch
type Adapter interface {
    Name() string
    Roots() []string
    Discover(ctx context.Context) ([]SessionMeta, error)
    Export(ctx context.Context, id string, w io.Writer) error
    Import(ctx context.Context, id string, r io.Reader, opts ImportOpts) error
    ProjectKey(path string) string
    Exclude() []string
}
```

## Golden fixtures

Every adapter ships fixtures under `testdata/adapters/<name>/`:

```
testdata/adapters/claude/
  session_sample.jsonl
  paths_windows.jsonl
  paths_macos.jsonl
  expected_normalized.json
```

CI runs round-trip: discover → normalize → denormalize → compare.

## Contributing an adapter

See [CONTRIBUTING.md](https://github.com/HarjjotSinghh/reinstate/blob/main/CONTRIBUTING.md#adapter-contributions).

Minimum PR requirements:

1. Implementation + fixtures
2. Defensive parsing (skip unknown fields/types)
3. Explicit credential excludes
4. Docs row in this matrix + README
5. No network in unit tests

## Adapter request template

Use [Adapter request](https://github.com/HarjjotSinghh/reinstate/issues/new?template=adapter_request.yml)
and include:

- Agent name + version
- Session file locations
- Sample **redacted** session header (no secrets)
- How resume works
