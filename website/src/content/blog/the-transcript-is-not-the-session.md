---
title: "The transcript is not the session"
description: "Your session file survives the crash. That is not the same as being able to continue, because a session is a transcript plus the conditions it ran under."
answer: "A coding-agent session is a transcript plus the conditions it ran under: the repository state, the branch, and the environment on the machine. Crash recovery usually returns only the transcript, which is why a resume can succeed and still be wrong. Reinstate verifies those conditions before the agent starts."
author: "Harjot Singh Rana"
publishedAt: 2026-08-27
updatedAt: 2026-08-27
reviewedAt: 2026-08-27
tags: ["coding agents", "session continuity", "Claude Code", "verified resume", "developer workflow"]
targetQuery: "why does resuming a coding agent session go wrong"
searchIntent: "problem"
related:
  - title: "How Reinstate works"
    path: "/about/reinstate"
  - title: "Structured handoff"
    path: "/docs/handoff"
  - title: "Claude Code integration"
    path: "/integrations/claude-code"
  - title: "Why Git alone does not sync sessions"
    path: "/blog/why-git-does-not-sync-coding-agent-sessions"
  - title: "Architecture"
    path: "/docs/architecture"
draft: false
noindex: false
category: "engineering"
featured: false
---

## Three failures you have stopped noticing

It is Tuesday. You open a terminal in a repository you were deep inside on
Friday, start your agent, and type out a paragraph explaining a design the agent
itself proposed four days ago. You restate the constraint you already argued
through with it. You list the same three files. It starts cold, so you
re-explain, and neither of you finds this strange.

Second one. You are forty minutes into something hard, the context fills, and
compaction runs. What the agent is carrying forward is now a summary of the
stretch you argued through, not the stretch itself. Or the terminal dies
outright and takes the whole thing with it. Either way you have lost the
thread, and the recovery move is identical: open a fresh session and start
typing background.

Third. The session survives. You pick it up mid-task on Monday, it names four
files with total confidence, and two of them moved when you merged `main` on
Saturday. It works against the layout written down in the transcript. You find
out at the diff.

None of these read as failures now. They read as weather. You have priced them
in the way you price a slow test suite: an annoyance with a known shape,
absorbed into what running agents costs. That pricing is what I want to argue
with, because it treats all three as the same problem, and they are not.

## The words were never the thing that went missing

Go and look at the crash case. Claude Code and Codex both write the session to
a file on your disk as you work: JSON Lines, one event per line, appended live.
It keeps being written until the process dies. After the crash that file is
still sitting there. A hard kill can cut the final record short, which is why
Reinstate's readers ignore an incomplete trailing line, but the turns before it
are intact.

The third case is starker. The transcript was perfect. Nothing about it was
lost, damaged, or truncated. The repository moved underneath it.

Compaction is the one I would not overstate. What a vendor leaves in the file
after a compaction is that vendor's decision, and it differs between them, so I
am not going to make a claim about all of them from here. What Reinstate does
with the result is fixed: a vendor compaction summary is carried as a summary,
labelled as one, never as turns you actually exchanged.

So in the crash and the stale-workspace cases the words are on disk, and
retrieval is the part that already works. In the compaction case what you lost
was never on disk to begin with. It was what the model was holding. In none of
the three would getting the text back be enough to continue. What went missing
is everything around the words.

## A session is a transcript plus the conditions it ran under

Here is the definition I would like to argue for, because most of the confusion
in this category comes from not having one.

A session is a transcript **plus the conditions it ran under**: which
repository, which branch, which working tree, which agent binary, and whether
that binary is the one you think it is.

Search and preview give you back the first half. They are useful and I would
not want to work without them. But the transcript is a set of statements about
a world. This file has that shape. That test fails for this reason. Every one
of those statements has an expiry date, and the repository sets it, not the
file the agent wrote. Continuing correctly needs both halves. Getting your
words back is not the same as being able to continue.

## What a preflight actually does before the agent starts

Reinstate runs an environment preflight before every same-vendor native
continuation. It reports the environment it can actually observe, compares only
facts that have trustworthy recorded provenance, and refuses a silent bad
continuation instead of guessing. That shipped in `v0.3.0`, and it is the part
of this product I would defend hardest.

The provenance rule is the whole design. A vendor session file records some
things reliably: the workspace, and often a branch. It does not reliably record
repository identity, HEAD, working-tree state, or which agent version you were
running when the session started. Where there is no trustworthy baseline, the
report says `unknown`. It never says `match`. The first time you continue a
session that predates any recorded observation, a lot of lines come back
`unknown`, and that is the honest answer rather than a poor one.

What that means when you are actually at the terminal: a warning holds the
launch until you confirm it, or acknowledge each warning by its exact ID, with
no wildcards. A different known repository identity is not a warning at all. It
is a block, and there is no flag that overrides it. Reinstate is allowed to stop
you. Stopping you is the feature.

The alternative is not neutral. An agent that continues against a workspace it
cannot recognise does not sit there confused. It works, confidently, against
statements that stopped being true, and you pay for that in review time and in
edits you have to undo. I am not going to put a number on how often that happens
or what it costs, because I do not have one I could defend.

## Two words we are strict about: resume and handoff

This is the second thing I want to plant, and it matters more than the first
because it is where the category is loosest.

A **resume** restores a session. It is same-vendor only: Claude Code to Claude
Code, Codex to Codex. Those are the two agents Reinstate resumes natively. The
vendor's own native continuation is doing the work; the preflight decides
whether starting it is safe.

A **handoff** is a different operation with a different name because it is a
different thing. It builds an explicit structured capsule and starts a **new**
Claude Code or Codex session from it. It does not translate a native session and
it does not transfer one. There is no version of this where a Claude Code
session becomes a Codex session. Handoff shipped in `v0.4.0`.

Since a handoff is lossy, the loss should be legible before you accept it. What
the capsule carries:

- your messages, verbatim, subject to redaction;
- the visible assistant replies, attributed to their source;
- tool names, inputs, and outputs, carried as **evidence** and never re-executed;
- the current changed files and branch state, read live from Git rather than
  from the transcript;
- attachments by reference: hash, type, size;
- vendor compaction summaries, labelled as summaries rather than as turns.

What it drops:

- pending or unfinished tool calls, closed as interrupted;
- hidden or encrypted reasoning state, which is vendor-opaque and not portable;
- credentials, tokens, and approvals, which are never read;
- live processes, shells, and sandboxes; only observable results carry over.

And one thing it deliberately refuses to manufacture. Reinstate does not derive
a list of "decisions" or "rejected approaches" from prose, because guessing them
produces confident nonsense. They are reported as omitted. Your own recent
messages, verbatim, carry that information honestly instead.

Every component comes back classified `exact`, `normalized`, `summarized`,
`referenced`, or `omitted`, with a reason, and you can see the whole
classification in a dry run before anything launches. There is a related rule
inside the handoff: where the transcript and the repository disagree, the
repository wins. A plan from ninety turns ago does not get to overrule what is on
disk now.

The drop list is published on purpose. A continuation you cannot audit is worse
than one that refuses, because the refusal costs you a minute and the unaudited
one costs you a review cycle you did not budget for.
The [handoff documentation](/docs/handoff) carries the full table, and
[limitations](/docs/limitations) is the page I would read first if you are
trying to find the edges rather than the pitch.

## What you can check today without installing anything

Your agent's session files are already on your disk. That is not a Reinstate
feature; it is how Claude Code and Codex work. Go and open the directory. The
text you thought you lost on Friday is very likely sitting in it.

Then ask the question that the text cannot answer for you. Before you continue
anything: which branch was this written against, does that repository still have
the same identity, is the working tree the one those file paths described, and
is the agent binary the same one that produced them? You can almost always get
the words back. That was never the question. The question is whether anything
those words assumed has changed since.

If the answer is yes and you continue anyway, you are asking an agent to act on
statements you already know are stale. If you want to see what the checking
looks like when it is written down, [restoring a session](/docs/restore-a-session)
walks the path, and the source is Apache-2.0, so the check list is readable
rather than something you take on faith. Stable is `v0.5.1`, dated 2026-08-21.
When a session does move between machines, snapshots are encrypted locally with
an age envelope before upload and land in storage you own: S3, Cloudflare R2, or
S3-compatible. Auth and credential files are excluded by policy, and there is no
Reinstate account anywhere in that path. The mandatory release targets for agent
resume today are Apple Silicon macOS and native Windows x64. Intel macOS and
Linux/WSL2 are optional and unverified, and are not certified Phase 1
agent-resume targets.

The transcript was never the hard part. The conditions it assumed are, and
nothing in the file will tell you whether they still hold.

---
