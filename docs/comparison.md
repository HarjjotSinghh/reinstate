# Comparison

How **Reinstate** sits next to native vendor sync, session browsers, full agentic
development environments (ADEs), single-tool sync utilities, and DIY file sync.

Reinstate is a **continuity layer** (find / verify / resume / hand off / sync) —
not another place to code.

![Landscape: agent scope vs state portability](../assets/01_landscape.png)

## Feature matrix

| Capability | Native vendor sync | claude-sync | coding-agent-sync / MCP tools | DIY (Syncthing / Drive) | **Reinstate** |
| ---------- | ------------------ | ----------- | ----------------------------- | ----------------------- | ------------- |
| Sessions across devices | Per-vendor only | Claude only | Partial | Files only | **Multi-agent** |
| Works when other machine is off | Cloud yes / Remote Control no | Yes | Yes | Yes | **Yes** |
| MCP / skills / loops / plugins / settings | Vendor-local | Partial (Claude tree) | Often one artifact class | Manual | **Universal desired-state config (planned)** |
| E2E encryption | Vendor-held plaintext | age | Often yes | Usually no | **age, BYO keys** |
| Bring-your-own storage | No | R2/S3/GCS/WebDAV | Gists / various | Your sync tool | **R2/S3-compatible (v0.1)** |
| Cross-OS path remapping | N/A | Partial | Weak | None | **First-class** |
| Delta / large history | N/A | Full-file gzip | Full-file | File-level | **Full snapshot now; append-aware planned** |
| Team session sharing | Rare | No | Rare | Manual | Roadmap |

## vs native vendor session UIs

**Claude Code** / **Codex** / **Gemini** session resume, search, and fork are
excellent *inside one ecosystem*. They will not:

- Index sessions across competing agents in one switcher
- Verify workspace + MCP/skills before resume in a vendor-neutral way
- Define an MCP/skill/plugin once and reconcile native config across competing harnesses
- Move work across devices with ciphertext on storage *you* control
- Produce explicit portable handoffs between agents

Reinstate's moat is **universality + environment verification + neutrality + BYO E2EE**.
Phase 1 proves the encrypted-sync foundation with same-vendor Claude Code and
Codex sessions; environment verification and portable handoffs follow.

## vs session browsers (e.g. SpecStory-class tools)

Unified browse/search/resume validates demand. Reinstate differentiates by
**verified environment continuity** (repo/branch/MCP/skills/runtime) and
optional **encrypted multi-device sync** — not by being “another picker.”

## vs full ADEs (Orca, Conductor, T3 Code, …)

Those products own **where coding happens** (UI, worktrees, terminals, multi-agent
layouts). Reinstate owns **whether work survives and can move**. Integrate;
do not compete as a full harness.

## vs claude-sync

claude-sync is a strong Claude-only reference (encryption, R2, path mapping).
Reinstate aims to become a **superset**. Phase 1 implements Claude Code and
Codex session adapters plus cross-OS path remapping; native acceptance is still
in progress, and config scope is post-v0.1 roadmap work.

## vs config-only tools (mcp-sync, etc.)

Config sync alone has a low ceiling (nice utility, weak product). Reinstate
treats **sessions as the acquisition feature** and **universal configuration as
continuity and retention**.

The planned configuration layer is broader than copying MCP JSON. It
normalizes desired state for MCP servers, skills/instructions, hooks/loops,
plugins, marketplaces, and safe settings; previews adapter-specific changes;
and reconciles them across harnesses and devices. It will report unsupported
fields rather than pretending all tools share one schema. Credentials remain
local.

## vs Syncthing / iCloud / OneDrive

DIY works until:

- Absolute paths break resume
- Conflict files corrupt JSONL mid-write
- Credentials and plugin trees sync by accident
- You spend an afternoon on junctions and excludes

Reinstate is resume-aware, agent-aware, and secure-by-default.

## Positioning one-liner

> Vendor sync owns *one* agent. File sync owns *bytes*. Reinstate owns
> **portable AI dev state** — sessions first, environment next — across agents and machines.

## Related

- [Architecture](architecture.md)
- [Adapters](adapters.md)
- [Universal agent configuration](universal-configuration.md)
- [Roadmap](../ROADMAP.md)
