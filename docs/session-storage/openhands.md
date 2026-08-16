# OpenHands (All Hands AI)

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T0, expected
reason `server_backed`.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | All Hands AI |
| Product | OpenHands |
| Distribution | Official, open source; commonly run as a server or in a container |
| Storage family | F5 expected |

## Why the expected outcome is T0

OpenHands is architecturally unlike the terminal agents in this roster. It
typically runs as a service with a web interface, often inside a container, and
its runtime state belongs to that service rather than to a user's home
directory. Three consequences:

1. **The session may not be on the user's machine at all.** A container's
   filesystem is not the host's, and a hosted deployment is not local.
2. **There is no "resume this session in your terminal" argv.** Continuation
   happens inside the OpenHands interface.
3. **Even if a host-side directory exists, it belongs to a server**, which may
   be writing to it concurrently, and which may treat it as internal state
   rather than as a durable user artifact.

## What the probe must settle

1. Whether a host-side artifact exists for a default local deployment, and
   where.
2. Whether it survives a container restart and a version upgrade.
3. Whether the project or repository path is recoverable from it.
4. Whether OpenHands exposes a documented conversation-export API. If it does,
   record it: it does not change the tier by itself, but it is the only path
   that could later.

## Recording a T0 outcome

If the probe confirms there is nothing durable and local, ship the descriptor
at T0 with reason `server_backed` and close the task as complete.
`rein doctor --agents` then tells the user OpenHands is recognized and why its
sessions are not indexable, which is a correct and useful answer.

Do not stretch to reach T1 here. An index full of records that disappear when a
container is recreated is worse than no records.

## Sources

None verified. Establish and record vendor sources before promoting any row.
