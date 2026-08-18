# Task card template

Copy this shape when the coordinator adds a task. Keep cards short: a card that
restates the SDK document is a card nobody reads twice.

---

## T-0XX — Short imperative title

**Workstream:** WN · **Target tier:** TN (agent tasks only) ·
**Depends on:** T-0YY, or nothing

**Owns.** Every path this task may modify. If a path is not listed here, the
task may not touch it. See [../file-ownership.md](../file-ownership.md).

**Goal.** One or two sentences. What is true after this task that was not true
before.

**Steps.** Numbered, each one verifiable. Reference the shared recipe in
[../../../adapters/agent-catalog-sdk.md](../../../adapters/agent-catalog-sdk.md)
instead of repeating it. Record only what is specific to this task.

**Done when.** Observable conditions, not effort. Prefer commands:

```bash
make verify
go test ./internal/agents/... -count=1
```

**Escalate if.** The specific circumstances where the executor must stop and
report rather than decide. Every card should have at least one; a card with
none has not been thought through.

---

## What makes a card good

- **The unknowns are named.** An executor should not discover the hard part
  three days in.
- **The forbidden actions are explicit.** "Do not edit the test to make it
  pass" prevents more damage than any amount of encouragement.
- **A negative outcome is a valid outcome.** Several Phase 5 cards succeed by
  proving an agent cannot be supported. Say so on the card, or an executor will
  force a result.
- **It names its evidence.** Which probe, which fixture, which report. Evidence
  the card does not name is evidence nobody produces.
