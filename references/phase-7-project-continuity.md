# Source: the project-continuity idea (2026-09-06)

Origin material for [Phase 7](../ROADMAP.md#phase-7--project-continuity-) and
[ADR 0006](../docs/adr/0006-project-continuity-scope.md). Like everything in
`references/`, this is input, not authority.

## The maintainer's problem statement

Paraphrased from the original framing, kept close to the wording used:

> I use Claude Code, Cursor, Codex, OpenCode, and Grok across one project,
> picking an agent per kind of task — Claude for frontend, Codex for backend,
> others for the rest — often several in parallel across multiple sessions.
> Every one of them has its own system for context, memory, configuration, and
> whatever else it stores about the project.
>
> There is context drift. If I tell Claude Code a rule — for example, never use
> em dashes anywhere in the project or in PRs — it holds for every Claude Code
> session I start afterwards. It does not reach Codex, Grok, or OpenCode. Those
> agents do not know the rule or the decision, so they contradict it: an em
> dash lands in a PR or on the landing page, and we get conflicts.
>
> MCP servers have the same shape. I use Slack, Notion, Figma, Vercel, GitHub
> and more. For every coding agent I have to write the MCP config again and
> authenticate again, tool by tool, agent by agent.
>
> The problem statement: **the context about a project is not unified across
> coding agents** — memory, MCP configuration, decisions, business logic,
> developer configuration, anything about the project. This happens on a single
> device as much as across devices.
>
> The solution I imagined is close to Reinstate but not the same thing.
> Reinstate makes sessions available across devices. This problem exists on one
> device with several agents. So: is this part of Reinstate, or a second
> project? And what is the right shape — user-facing markdown files, internal
> files, JSON, something else?

Two questions were asked explicitly: **should this be built**, and **should it
be part of Reinstate or a standalone project**.

## What was done with it

Two external AI analyses of the space were supplied alongside the idea. Both
concluded the work belongs inside Reinstate rather than beside it, and both
observed that the space is crowded in its obvious parts — rules distribution,
MCP gateways, cross-agent memory servers — while the write path is not solved.

The analyses are **not** stored here. Two reasons: one contained unrelated
personal material, and between them they named roughly twenty projects and two
papers, four of which turned out not to exist at the paths cited. Their claims
were checked individually instead, and only what survived was written down:

- verified survey —
  [docs/research/2026-09-06-phase-7-cross-agent-context-landscape.md](../docs/research/2026-09-06-phase-7-cross-agent-context-landscape.md)
- decisions and their rationale —
  [ADR 0006](../docs/adr/0006-project-continuity-scope.md)
- design direction —
  [docs/project-continuity.md](../docs/project-continuity.md)
- open questions —
  [docs/planning/v0.8.0-project-continuity/clarifications.md](../docs/planning/v0.8.0-project-continuity/clarifications.md)

Answers to the two questions, recorded in ADR 0006: build it, inside Reinstate,
as Phase 7, targeted at `v0.8.0`.
