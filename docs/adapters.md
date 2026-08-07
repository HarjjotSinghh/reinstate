# Adapters

Reinstate separates per-agent capabilities:

- **local read adapters** discover bounded metadata and user-prompt search text;
- **native executors** resume/fork through the same vendor;
- **sync adapters** export and restore vendor-native session files; and
- **environment observers** report safe current facts before same-vendor
  execution without mutating native configuration; and
- later **configuration adapters** normalize portable intent and render each
  harness's native MCP/skills/plugins/settings format.

One capability never implies another. Read-only agents do not receive dummy
resume, fork, export, or restore implementations.

## Capability matrix

| Adapter | Phase 2 local index | Native resume/fork | `v0.1.0` encrypted sync | Universal config |
| ------- | ------------------- | ------------------ | ------------------------- | ---------------- |
| Claude Code | Included in `v0.2.0` | Included in `v0.2.0` | Supported | Later |
| OpenAI Codex CLI | Included in `v0.2.0` | Included in `v0.2.0` | Supported | Later |
| Gemini CLI | Read-only in `v0.2.0` | No | No | Later |
| OpenCode | Read-only in `v0.2.0` | No | No | Later |
| Cursor | Exploring | No | No | Exploring |
| Grok Build | Exploring | No | No | Planned |

Phase 2 automated gates and the complete tagged-artifact matrix passed on Apple
Silicon macOS and native Windows x64. Stable `v0.2.0` support is limited to
those verified platforms; Intel macOS and Linux/WSL2 remain preview/unverified.

## Phase 2 local read contract

All local records use:

```text
<agent>:<native-session-id>
```

Local discovery is config-independent and does not inherit Phase 1 project
mappings that would hide unmapped sessions. Sources expose only identity,
timestamps, workspace/project, recorded branch, title/name, bounded
user-authored prompt text, known file fields, counts, source fingerprint, and
capabilities. Assistant messages/reasoning, tool output, environment dumps,
credentials, and auth stores are excluded from the index.

| Adapter | Read source |
| ------- | ----------- |
| Claude Code | Stream project JSONL; exclude subagent artifacts; ignore incomplete trailing record |
| Codex CLI | Stream date-partitioned rollout JSONL and structural session metadata |
| Gemini CLI | Defensively read recognizable project chat JSON under the vendor data root |
| OpenCode | Use the documented local JSON session-list surface through a bounded runner |

Local-index capabilities and Phase 1 sync compatibility are separate
contracts. The local record advertises whether native resume/fork is available;
the existing sync adapter continues to enforce its verified version range
before export or restore.

## Native execution

| Agent | Resume | Fork |
| ----- | ------ | ---- |
| Claude Code | `claude --resume ID` | `claude --resume ID --fork-session` |
| Codex | `codex resume ID` | `codex fork ID` |

Plans store executable, argv, and recorded cwd separately. They never construct
a shell command string. Gemini/OpenCode resume or fork fails with compatibility
exit `5`.

## Phase 3 environment-observer contract (`v0.3.0-rc.2`)

Phase 3 verification is included in release candidate `v0.3.0-rc.2`; it is not
part of stable `v0.2.0`, its tagged-artifact acceptance is pending, and it does
not authorize stable `v0.3.0`. It adds observation to Claude/Codex native
execution, not a new execution adapter and not cross-vendor translation.

| Adapter | Phase 3 verified-resume observation |
| ------- | ----------------------------------- |
| Claude Code | Workspace, executable/version/layout, instruction/skill/MCP names, and recognized runtimes |
| OpenAI Codex CLI | Workspace, executable/version/layout, instruction/skill/MCP names, and recognized runtimes |
| Gemini CLI | No native launch; read-only refusal occurs before preflight |
| OpenCode | No native launch; read-only refusal occurs before preflight |

The verifier treats these capabilities independently:

- executable discovery and a strictly parsed installed version;
- a recognized, same-vendor session layout rooted at the selected record;
- instruction, skill, and MCP declaration names/state in recognized user and
  project locations; and
- supported Node/Go project declarations and installed runtime versions.

Capability enumeration is bounded and passive. It never executes a skill,
instruction, configuration file, MCP command, package manager, or project
script. It reports sanitized logical names, scope/state, and an MCP transport
classification (`stdio`, `http`, `sse`, or `unknown`) only. It does not report
paths, file contents, commands, arguments, raw URLs, headers, environment
values, authentication state, or credentials. Escaping symlinks and
unsupported/malformed shapes yield fixed diagnostics rather than raw parser
output.

An observed capability is not automatically a historical requirement. On the
first inspection, presence is current truth only. A prior successful prelaunch
baseline or a recognized vendor-recorded requirement is required before
Reinstate claims a match/missing/change comparison. The baseline is saved only
after an authorized native child exits successfully.

See [Verified resume](verified-resume.md) for launch decisions, exact warning
acknowledgements, exit codes, and provenance.

## Future configuration adapters

Planned configuration targets include Claude Code, Codex, Gemini CLI, OpenCode,
and Grok. Each adapter will advertise support independently for MCP servers,
skills/instructions, hooks/loops, plugins, marketplaces, and safe settings. An
adapter that can resume sessions does not automatically support configuration,
and unsupported or lossy mappings must be reported before apply.

See [universal-configuration.md](universal-configuration.md).

## Compatibility states

See [compatibility.md](compatibility.md). Phase 1 sync adapters report
`SUPPORTED`, `UNTESTED`, `UNSUPPORTED`, or `NOT_INSTALLED`; Phase 2 local
records report read/native capabilities independently.

## Claude project identity

Claude Code stores a project beneath a directory key derived from that
device's absolute project path. Reinstate records the configured canonical
project ID in snapshots and recomputes Claude's directory key from the
destination device's `local_root`. Snapshot archive paths remain source
metadata; they are never reused as cross-device restore destinations.

Reinstate verifies the exact planned destination after restore. A matching session
ID elsewhere in `~/.claude/projects` is not accepted as success.

The Phase 2 local reader intentionally sees all local top-level Claude sessions,
including projects that are not mapped for encrypted sync.

## Codex project identity

Codex stores the source working directory in each rollout's structural
`session_meta.cwd`. When project mappings are configured, Reinstate resolves that
directory to the configured canonical project ID during discovery and excludes
rollouts outside those mapped roots. Export normalizes the resolved source root
to a `${REPO:<id>}` token, and restore expands it through the destination
device's `local_root`. This keeps Windows and macOS paths out of portable
session identity while preserving Codex's native date-partitioned rollout
layout.

The Phase 2 local reader similarly indexes local Codex rollouts without
requiring a canonical sync mapping.

## Exclusions

Sync adapters hard-exclude auth, credentials, tokens, caches, logs, and
regenerable dependencies. The Phase 2 index additionally excludes assistant
messages/reasoning, tool output, environment dumps, and auth stores. Future
configuration profiles may carry secret **references** but never secret
values. Fixtures are synthetic and scanned for secrets.
