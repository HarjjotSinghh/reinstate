# Cross-agent context, memory, and configuration — landscape survey

**Date:** 2026-09-06 · **For:** [Phase 7 — Project continuity](../project-continuity.md)
and [ADR 0006](../adr/0006-project-continuity-scope.md)

## Method and its limits

Two external AI analyses of the "unify context, memory, and MCP configuration
across coding agents" space named roughly twenty projects and two papers. Each
claim was checked individually with web search on 2026-09-06.

Three limits apply to everything below:

- Retrieval was search-result summaries, not direct API reads. **Star counts,
  adoption numbers, and dates are approximate** and were current on 2026-09-06.
- A project verified as *existing* is not a project this repository has used,
  audited, or benchmarked. Nothing here is a quality judgment.
- Feature descriptions come from each project's own materials.

**Four of the projects the analyses cited do not exist at the paths given.**
`B0904/Cagentmemory`, `dan-calin/shared-agent-memory`,
`nova-land/coding-agent-memory-mcp`, and `JustinBeaudry/agent-sync` returned
nothing; in three cases real projects with similar functions exist under
different names and authors. Every one of the fabrications was in the
memory category — the exact category Phase 7 enters. This is why no competitor
name reaches `ROADMAP.md`, the docs, or the website without a checked primary
source.

## 1. Rules and configuration distribution — well served

One canonical rules source, rendered into each harness's native files.

| Project | What it does | Status |
| ------- | ------------ | ------ |
| [Ruler](https://github.com/intellectronica/ruler) | Markdown rules in `.ruler/` applied to many agents' native config; also distributes MCP server config and, experimentally, skills | Verified; star count and exact agent count approximate |
| [rulesync](https://github.com/dyoshikawa/rulesync) | CLI generating rules, ignore files, MCP servers, commands, and subagents for many agents, with bidirectional conversion | Verified as a project; the name is shared by unrelated projects |
| [Airulefy](https://github.com/airulefy/Airulefy) | Single rule set in `.ai/`, generated or linked into tool-specific formats | Verified |
| [vsync](https://github.com/nicepkg/vsync) | Syncs MCP servers, skills, agents, and commands across Claude Code, Cursor, OpenCode, Codex | Verified; broader than MCP, contrary to the claim made |

**Read for Reinstate:** static distribution is a solved, crowded problem.
Phase 6A must not be positioned as "we copy your rules file to other tools."
Its defensible parts are normalization with reported lossy mappings, drift
detection that preserves unrelated native settings, and reconciliation across
devices — not the copying.

## 2. MCP gateways — well served, and the reason a Reinstate gateway stays 💭

Authenticate once; every agent connects through one endpoint.

| Project | What it does | Status |
| ------- | ------------ | ------ |
| [Toolport](https://github.com/tsouth89/toolport) | Local-first MCP gateway; secrets in the OS keychain injected at call time and never written to client config; lazy tool discovery; tool integrity and quarantine; global settings applied to every client | Verified |
| [1MCP](https://github.com/1mcp-app/agent) | Aggregates MCP servers into one runtime so Cursor, Claude Code, Codex, and internal tooling share an inventory | Verified |
| [MetaMCP](https://github.com/metatool-ai/metamcp) | Aggregator, orchestrator, middleware, and gateway; namespaces with unified authenticated endpoints | Verified |
| [Docker MCP gateway](https://github.com/docker/mcp-gateway) | Official Docker reverse proxy for MCP servers with container isolation, secrets, OAuth, and a server catalog | Verified |
| [Bifrost](https://github.com/maximhq/bifrost) | AI gateway by Maxim AI, including an MCP gateway layer with access control and cost governance | Verified; company raised a $3M seed (2024) |

**Read for Reinstate:** entering this category means competing with four
credible implementations, one of them Docker's, on the largest security surface
in the product. ADR 0006 D10 keeps it exploratory and names the conditions that
would change that.

## 3. Cross-agent memory — real competitors, closest to Phase 7

| Project | What it does | Status |
| ------- | ------------ | ------ |
| [Remnic](https://github.com/joshuaswarren/remnic) | Memory as plain Markdown with YAML frontmatter on disk, no database or cloud; scoped memory, per-result provenance you can inspect, a correction workflow, evals; MCP and HTTP access across many clients | Verified |
| [Memorix](https://github.com/AVIDS2/memorix) | Local-first cross-agent memory layer over MCP; shared searchable project memory tied to the git project; git-commit and reasoning memory | Verified |
| [OpenMemory (mem0)](https://mem0.ai/blog/introducing-openmemory-mcp) | Private local-first MCP memory server; persistent context shared across Cursor, Claude Desktop, Windsurf, and other MCP clients; on-device | Verified |
| `agentmemory` | Persistent memory for coding agents with hybrid search and an MCP server; at least two unrelated projects use this name | Exists under names and authors other than those cited |

**Read for Reinstate:** the naive framing of Phase 7 — "a shared memory MCP
server for coding agents" — is already built, more than once, by people who
also wrote benchmarks for it. Remnic in particular already has provenance and
correction workflows. Phase 7 has to be justified by what these do not have:

- binding to **project identity across machines and paths**, which is
  Reinstate's existing Phase 1/2 primitive;
- provenance anchored to a **session** that can then be found and resumed;
- durable context and learned memory as **two different stores with two
  different review rules**, rather than one memory pool;
- reconciliation into the **configuration desired state** the same tool already
  renders per harness;
- encrypted multi-device sync with device revocation and key rotation, already
  shipped.

If Phase 7 ends up being "a memory server plus a rules copier", it is a worse
version of tools that already exist. The integration with session continuity is
the whole argument.

## 4. Session continuity — direct competitors to what Reinstate ships today

Surfaced by the research and worth tracking independently of Phase 7.

| Product | What it does | Status |
| ------- | ------------ | ------ |
| [BuildBetter](https://www.buildbetter.sh/) | CLI that saves each session — chat, file edits, tool calls — and shares it across a team; check out an old branch and pull up the chat that produced it, or resume a teammate's half-finished session on your machine | Partially verified from the vendor's own marketing; the exact command name, the full agent list, and PR linking are unconfirmed |
| [Junction](https://junctionpanel.dev/) | Local daemon talking to Claude Code, Codex, and OpenCode CLIs, streaming tool calls, edits, and shell output over WebSocket to a browser panel usable from phone or tablet; device pairing by six-digit code over an encrypted relay | Partially verified; the sources describe live remote control more clearly than resuming an ended session |
| [Agent Sessions](https://github.com/jazzyalex/agent-sessions) | Local-first macOS app for browsing and resuming sessions across Codex, Claude Code, OpenCode, Cursor Agent, Copilot CLI, and others | Verified; unrelated to BuildBetter despite the similar name |

**Read for Reinstate:** team continuity (Phase 9) and a browser or mobile
control surface (Phase 8) are being built by others now. Neither of the two
products above was found to do environment verification before resume, path
remapping across operating systems, or explicit cross-agent handoff with a
fidelity report — the three things Reinstate ships that a session browser does
not. That gap is the claim to defend.

## 5. Standards to interoperate with, not compete against

- **AGENTS.md** — an open format for project-specific agent instructions,
  originating in OpenAI's Codex tooling around August 2025. In December 2025
  the Linux Foundation announced the
  [Agentic AI Foundation](https://www.linuxfoundation.org/press/linux-foundation-announces-the-formation-of-the-agentic-ai-foundation),
  co-founded by OpenAI with Anthropic and Block; its inaugural projects are
  MCP, goose, and AGENTS.md. Sources report adoption by more than 60,000
  open-source projects and native support in 20+ tools including Codex, Cursor,
  Copilot, Gemini CLI, Aider, Windsurf, Zed, Factory, and Jules. Claude Code
  reads `CLAUDE.md`, not `AGENTS.md`; the documented bridge is a `@AGENTS.md`
  import line in `CLAUDE.md` — the pattern this repository already uses.
- **Agent Skills / SKILL.md** — introduced by Anthropic around October 2025 and
  since published as an open specification
  ([spec](https://github.com/anthropics/skills/blob/main/spec/agent-skills-spec.md)).
  [Cursor's documentation](https://cursor.com/docs/skills) states it discovers
  skills from `.cursor/skills/` and, for compatibility, from `.claude/skills/`
  and `.codex/skills/`.
- **MCP authorization** — the
  [2025-11-25 specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
  mandates OAuth 2.1 with PKCE, forbids token passthrough (a server must not
  accept a token not issued for it, and must use a separately issued token for
  any upstream call), and requires audience validation via RFC 8707 resource
  indicators.

**Read for Reinstate:** two consequences. Rendering into `AGENTS.md` and
`SKILL.md` is interoperating with governed multi-vendor standards, which is the
right posture — ADR 0006 D4. And "steal one harness's OAuth token and paste it
into another" is not merely inelegant, it is contrary to the MCP authorization
spec. The existing wording in `docs/universal-configuration.md` — configure
once, authenticate as few times as safely possible — is the accurate promise.

## 6. Evidence that project context changes agent behavior

- [arXiv 2601.20404](https://arxiv.org/abs/2601.20404), *On the Impact of
  AGENTS.md Files on the Efficiency of AI Coding Agents* (January 2026; ICSE
  2026 JAWs workshop). 10 repositories, 124 pull requests, agents run with and
  without an `AGENTS.md`. Reported a 28.64% reduction in median runtime and a
  16.58% reduction in output tokens, with comparable task-completion behavior.
- [arXiv 2606.15828](https://arxiv.org/abs/2606.15828), *Configuration Smells
  in AGENTS.md Files*. A catalog of six configuration smells across 100 popular
  open-source repositories: context bloat in 42% of files, lint leakage in 62%,
  skill leakage in 35%, and at least one smell in 91 of 100.

Both figures come from search-result summaries of the papers rather than a
direct read of the PDFs; treat them as directional until someone reads the
papers.

**Read for Reinstate:** together these support the phase's central design
choice. Good project context measurably helps, *and* the instruction files
people actually write rot into bloat and contradiction. That is the argument
for scoped rules, typed records with expiry, promotion through review, and
contradiction detection — rather than one large file every agent reads in full.

## What is unclaimed

Consolidating the five sections:

| Capability | Covered by existing tools |
| ---------- | ------------------------- |
| Distribute static rules to many harnesses | Yes, several |
| One MCP inventory and one authentication | Yes, several |
| Shared cross-agent memory over MCP | Yes, several, including with provenance |
| Browse and resume sessions across agents | Yes |
| Team session sharing | Emerging |
| Verified environment before resume | Not found |
| Cross-OS path remapping tied to project identity | Not found |
| Explicit cross-agent handoff with a fidelity report | Not found |
| Learned memory bound to a resumable session and a portable project identity | Not found |
| Promotion from learned memory to reviewed, committed project truth | Not found |
| Contradiction and staleness detection between memory and decisions | Not found |
| One reconciled desired state covering configuration *and* context *and* sessions, across devices | Not found |

The bottom half of that table is Phase 7's reason to exist. Each row is worth
re-checking before implementation starts: this landscape moved fast enough that
a survey six months old would already be wrong.
