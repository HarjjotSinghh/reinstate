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
| MCP / skills / settings | Vendor-local | Partial (Claude tree) | Often config-only | Manual | **Universal config scope** |
| E2E encryption | Vendor-held plaintext | age | Often yes | Usually no | **age, BYO keys** |
| Bring-your-own storage | No | R2/S3/GCS/WebDAV | Gists / various | Your sync tool | **Object storage + WebDAV** |
| Cross-OS path remapping | N/A | Partial | Weak | None | **First-class** |
| Delta / large history | N/A | Full-file gzip | Full-file | File-level | **CAS + tail-append (target)** |
| Team session sharing | Rare | No | Rare | Manual | Roadmap |

## vs native vendor session UIs

**Claude Code** / **Codex** / **Gemini** session resume, search, and fork are
excellent *inside one ecosystem*. They will not:

- Index sessions across competing agents in one switcher
- Verify workspace + MCP/skills before resume in a vendor-neutral way
- Move work across devices with ciphertext on storage *you* control
- Produce explicit portable handoffs between agents

Reinstate's moat is **universality + environment verification + neutrality + BYO E2EE**.

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
Reinstate aims to be a **superset**: multiple agents, config scope as a product
feature, and path remapping generalized for Windows ↔ macOS as the default
crossing — not an edge case.

## vs config-only tools (mcp-sync, etc.)

Config sync alone has a low ceiling (nice utility, weak product). Reinstate
treats **sessions as the acquisition feature** and **config as retention**.

## vs Syncthing / iCloud / OneDrive

DIY works until:

- Absolute paths break resume
- Conflict files corrupt JSONL mid-write
- Credentials and plugin trees sync by accident
- You spend an afternoon on junctions and excludes

Reinstate is resume-aware, agent-aware, and secure-by-default.

## Positioning one-liner

> Vendor sync owns *one* agent. File sync owns *bytes*. Reinstate owns
> **portable AI dev state** — sessions + environment — across agents and machines.

## Related

- [Architecture](architecture.md)
- [Adapters](adapters.md)
- [Roadmap](../ROADMAP.md)
