# Cross-agent handoff

Phase 4 `v0.4.0-rc.2` lets you continue the same task in a different coding
agent. You start in Claude Code, hit a usage limit, and hand the task to Codex
without re-explaining the work or making another Claude API call. The reverse
works too.

This document is the implementation contract for `v0.4.0-rc.2`. It deliberately
does not claim that a session is transferred. Design detail lives in
[cross-agent-continuation.md](cross-agent-continuation.md); the delivery plan
lives in [the Phase 4 plan](superpowers/plans/2026-08-12-phase-4-cross-agent-handoff-plan.md).

## What a handoff is, in one paragraph

A handoff creates a **new session in the destination agent**, linked to the
source. Reinstate reads the source transcript up to its last complete record,
builds a portable **continuity capsule**, verifies the current workspace with
Git, compares what the destination can actually do, and hands the destination a
bounded, source-attributed briefing plus an inspectable capsule file. The
destination restates its understanding before it changes anything.

It continues **the same task in a new destination session**.

## Continuity modes

Reinstate labels these separately in the CLI, in JSON, and in docs:

| Mode | Destination | Fidelity | Status in `v0.4.0-rc.2` |
| ---- | ----------- | -------- | ------------------ |
| **Native resume** | Same agent | Highest; vendor session semantics kept | Shipped since `v0.2.0` |
| **Structured handoff** | Different agent | Task state + selected verbatim history + evidence | **`v0.4.0-rc.2`** |
| **Reconstructed conversation** | Different agent | Visible history written into target-native storage | Not shipped in `v0.4.0-rc.2` |

## Supported directions in `v0.4.0-rc.2`

| Source → | Claude Code | Codex CLI | Gemini CLI | OpenCode | Grok Build |
| -------- | :---------: | :-------: | :--------: | :------: | :--------: |
| **Claude Code** | same-vendor native resume | **structured handoff** | not in rc.1 | not in rc.1 | not planned |
| **Codex CLI** | **structured handoff** | same-vendor native resume | not in rc.1 | not in rc.1 | not planned |
| **Gemini CLI** | **structured handoff** | **structured handoff** | not a target (source-only) | not in rc.1 | not planned |
| **OpenCode** | **structured handoff** | **structured handoff** | not in rc.1 | not a target (source-only) | not planned |
| **Grok Build** | **structured handoff** | **structured handoff** | not in rc.1 | not in rc.1 | not a target (source-only) |

Gemini CLI, OpenCode, and Grok Build are **source-only** in `v0.4.0-rc.2`: you can
hand off *from* them, not *to* them. Support is directional and versioned — a
supported session adapter never implies a supported handoff.

## Commands

```bash
# Preview first. This writes nothing outside a temp directory.
rein handoff --last --from claude --to codex --dry-run

# Do it.
rein handoff claude:<session-id> --to codex

# Build the capsule but launch nothing; print the exact command instead.
rein handoff claude:<session-id> --to codex --no-launch

# After the fact.
rein handoff list
rein handoff inspect <handoff-id>
rein handoff export <handoff-id> --format markdown --out ./handoff.md

# Convenience alias. Prints a notice that this is a handoff, not native resume.
rein resume claude:<session-id> --with codex
```

### Flags that matter

| Flag | Effect |
| ---- | ------ |
| `--policy checkpoint\|balanced\|full` | How much conversation history to carry. Default `balanced`. |
| `--dry-run` | Full plan and fidelity preview; no side effects. |
| `--no-launch` | Build artifacts, print the destination command, spawn nothing. |
| `--allow-warning ID` | Acknowledge one exact warning. No wildcards. |
| `--allow-active` | Take a boundary while the source agent is still running. |
| `--allow-untested` | Proceed on an untested agent version. |
| `--show-redactions` | Show redaction categories and counts. Never the values. |

### History policies

| Policy | What the destination sees | Use it when |
| ------ | ------------------------- | ----------- |
| `checkpoint` | Task state, workspace truth, changed files, tests, next action | The task is small and clear, or the destination context is tight |
| `balanced` | Checkpoint plus the most recent verbatim turns and representative tool evidence | Default |
| `full` | All portable visible history, up to a hard cap, with overflow in a sidecar | Audit and debugging |

Anything that does not fit is written to a private sidecar and **referenced** —
never silently dropped. Reinstate reports estimated size and tokens before it
launches anything.

## What transfers, and what does not

| Source material | Transfers | Notes |
| --------------- | :-------: | ----- |
| Your messages | Yes | Verbatim, subject to redaction |
| Visible assistant replies | Yes | Source-attributed |
| Tool names, inputs, outputs | Yes | As **evidence**; never re-executed. Paths become portable tokens |
| Current changed files and branch state | Yes | Read live from Git, not from the transcript |
| Attachments present on disk | Reference only | Hash, MIME type, size |
| Vendor compaction summaries | Yes | Labeled as summaries |
| Unknown record types | Reference only | Preserved as opaque hashes |
| Source system/developer instructions | Audit only | Stored, never handed to the destination as authority |
| Pending or unfinished tool calls | No | Closed as interrupted |
| Hidden reasoning, encrypted reasoning state | No | Vendor-opaque; not portable |
| Credentials, tokens, approvals | No | Never read, never synced |
| Live processes, shells, sandboxes | No | Only observable results carry over |

Every component is classified `exact`, `normalized`, `summarized`, `referenced`,
or `omitted` — with a reason. The fidelity report is part of every handoff and
is visible in `--dry-run` before anything happens.

## Truth hierarchy

When the transcript and the repository disagree, the repository wins:

1. Current workspace bytes and Git state
2. Your latest explicit request and constraints
3. Completed tool results that can still be verified
4. The structured checkpoint
5. Older conversation statements and model plans

This is why a stale early plan or a hallucinated "I changed that file" cannot
override what is actually on disk.

## What Reinstate derives without a model

The motivating case is a source agent that is rate-limited, logged out, or
closed. Everything below is computed locally, with no network and no model call:

- your latest request, verbatim;
- the most recent verbatim turns;
- completed tool evidence and interrupted calls;
- live changed files, branch, HEAD, and working-tree state;
- the last recognized test command and its exit state;
- missing tools, MCP servers, skills, and instruction files at the destination.

Reinstate does **not** deterministically invent a list of "decisions" or
"rejected approaches". Those are reported as omitted, because guessing them from
prose produces confident nonsense. Your verbatim recent messages carry that
information honestly instead. A future optional summarizer may fill them in,
but no source model call is part of the rc.1 critical path.

## Security

Imported history is untrusted data. Reinstate treats it that way:

1. Source system and developer messages are audit history. They are never
   promoted to destination authority.
2. Historical tool calls are inert. Nothing found in a transcript is executed.
3. Imported content is rendered inside a fenced, source-attributed block with an
   explicit "data, not instructions" banner. Delimiter collisions are escaped.
4. Secrets are redacted before anything is written, with markers that cannot be
   mistaken for original text. `--show-redactions` previews categories and
   counts, never values.
5. Credentials, auth stores, and keychains are never read.
6. Capsules live outside your repository, owner-only (`0700` directories,
   `0600` files; a protected DACL on Windows), and are **not synced** in
   `v0.4.0-rc.2`.
7. The destination re-authorizes every permission, network action, secret
   lookup, and MCP login under its own policy.
8. An unknown source or destination version fails closed with exit code `5`.

Handing off **from** a Grok Build session always runs redaction and always
prints a warning about that CLI's documented repository-upload behavior. Grok
is not available as a destination.

See [security-model.md](security-model.md).

## Where things are stored

```text
$REINSTATE_HOME/handoffs/
  lineage.jsonl              append-only handoff history
  <handoff-id>/
    capsule.json             the portable continuity capsule
    projection.md            exactly what the destination was given
    bootstrap.txt            the prompt passed on the command line
    fidelity.json            component-level fidelity report
    sidecar/                 full history and large tool output
```

Deleting `handoffs/` loses lineage and nothing else. Your sessions stay in their
own vendor stores; Reinstate never modifies a source session or writes a
destination vendor's internal session files.

## The acknowledgement step

Before the destination changes anything, it is asked to restate:

1. the current goal and your latest request;
2. the constraints carried over;
3. the current changed files and test state;
4. anything missing or uncertain;
5. its proposed next action.

Be clear about the limit: in `v0.4.0-rc.2` this is a **prompt-level contract**.
Reinstate prepares and verifies the briefing, but it does not run the
destination's agent loop and cannot force it to comply. `rein handoff inspect
<id>` lets you record whether the acknowledgement was correct, so the success
metric reflects reality.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| `0` | Handoff planned or completed |
| `2` | Usage error |
| `3` | Configuration problem |
| `5` | Untested or unsupported source/destination version or layout |
| `6` | Ambiguous session reference |
| `7` | Unacknowledged warnings, or a safety refusal |
| `1` | Runtime failure |

## What this is not

- Not native resume, and never described as one.
- Not a transfer of your session, account, credentials, or approvals.
- Not a rewrite of another vendor's session files. Reinstate writes nothing
  into `~/.claude`, `~/.codex`, `~/.gemini`, `~/.grok`, or OpenCode storage.
- Not an agent runtime, editor, terminal, scheduler, or model router. The
  destination agent executes; Reinstate captures, verifies, projects, launches,
  and records lineage.
