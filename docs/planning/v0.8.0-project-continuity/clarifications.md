# Clarifications for the maintainer — Phase 7 (project continuity)

Phase 7 is roadmap direction targeted at `v0.8.0`. Nothing is implemented, and
no work is scheduled: this directory exists so the open questions have a home
before an RFC starts.

Written 2026-09-06.

**Settled.** Twelve decisions were answered directly on 2026-09-06 and are
recorded as D1–D12 in
[ADR 0006](../../adr/0006-project-continuity-scope.md): phase placement and the
Console/Team renumbering, the `v0.8.0` target, the four in-scope blocks, the
canonical-tree-versus-`AGENTS.md` model, hybrid storage, the local MCP server,
explicit capture with human promotion, memory syncing while capsules do not,
the six gate harnesses, the gateway staying exploratory, how far positioning
moves, and the verified-sources-only rule for prior art. They are not repeated
here.

**Still open.** Each question below names the assumption an RFC would proceed
under, so none of them blocks anything.

---

## Q1 — What is the in-repo directory called?

**Assumed `.reinstate/`.** It matches the product name and does not collide
with any harness directory in use today (`.claude/`, `.codex/`, `.cursor/`,
`.opencode/`, `.gemini/`). `.rein/` is shorter and matches the preferred CLI
alias. The name is cheap to change before the RFC and expensive after, since it
lands in every user's repository.

## Q2 — `rein status` or `rein doctor`?

**Assumed a new `rein status`**, with `doctor` staying diagnostic and
repair-oriented. The risk is two commands that each answer "is my setup okay?"
in different words. The alternative is one command with a section per concern,
which keeps the surface smaller but makes `doctor` long enough that people stop
reading it.

## Q3 — How is a contradiction detected without calling a model?

**Assumed deterministic signals only:** explicit supersede links, overlapping
scope, subject tags a record declares, expiry, and evidence anchors (a file,
commit, or branch a memory was derived from that has since changed). Reinstate
makes no model calls anywhere today, including in handoffs, and this phase
should not be where that changes. Deterministic detection will miss genuine
semantic contradictions that share no scope or tag. If the miss rate makes the
feature useless in practice, the options are richer required metadata at
capture time, or an explicitly opt-in local model — and the second is a
product decision, not an implementation detail.

## Q4 — Does promotion write to the repository directly?

**Assumed it produces a reviewable change, never a commit.** `rein memory
promote` writes the file change to the working tree and prints the diff; you
commit it. Reinstate does not create commits, branches, or pull requests on
your behalf. That keeps git the source of truth for repository content and
keeps the review where reviews happen.

## Q5 — Do we import Claude Code's own per-project memory?

**Assumed import once, on `rein context adopt`, never continuous mirroring.**
Claude Code maintains its own automatic per-project memory. Reading it once
during adoption is useful; tailing and re-writing it continuously would make
Reinstate a second writer to a store another vendor owns, with the format churn
that implies. Codex, Cursor, Grok, OpenCode, and Gemini CLI get the same
treatment for whatever equivalent they have.

## Q6 — What happens to a project that is not a git repository?

**Assumed context degrades, memory does not.** Project identity and path
mapping already work for non-repo directories, so memory and its provenance
still function — the commit and branch fields are simply empty. The canonical
context tree still writes, but "reviewed in a pull request" stops being true,
and evidence anchors tied to commits stop working. Worth saying plainly in the
docs rather than discovering it.

## Q7 — Can a user opt out of the MCP server?

**Assumed yes, and assumed it is off until enabled.** A local server that any
agent can write to is not something to start by default because a user
installed a session tool. Durable context still renders into instruction files
without it; only capture and scoped retrieval require it.

## Q8 — What happens when the secret scanner rejects a memory?

**Assumed the write is refused with the reason, and nothing partial is
stored.** An agent proposing a memory containing what looks like a token gets
an error, not a redacted success. Silent redaction would teach agents that
pasting credentials into memory is fine.

## Q9 — Two developers, one repository, before Phase 9

**Assumed memory stays per-developer.** Durable context is shared because it is
committed to the repository; learned memory syncs across *your* devices only.
Shared memory between developers is Phase 9 (team continuity) and needs an
access model this phase does not define.

## Q10 — Command naming

**Assumed both `rein remember "…"` as the ergonomic form and `rein memory add`
as the complete one**, the way the CLI already offers `rein last` alongside
`rein sessions`. The RFC settles the final surface; nothing in the docs should
be read as a stabilized command until then.

## Q11 — What platform does the phase gate run on?

**Assumed the standard dual-platform contract** — Apple Silicon macOS and
native Windows x64 — because `v0.8.0` is far enough out that the macOS host
question should be resolved. If it is not, the Windows-first waiver pattern
from [ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md)
applies again, with deferred rows named in a public issue.

## Q12 — Does the landscape survey need re-running before implementation?

**Assumed yes.** The survey in
[docs/research/2026-09-06-phase-7-cross-agent-context-landscape.md](../../research/2026-09-06-phase-7-cross-agent-context-landscape.md)
is dated, and this category is moving fast enough that six months would make it
wrong. Re-run it when the RFC opens, and check specifically whether the
unclaimed rows in its final table are still unclaimed.
