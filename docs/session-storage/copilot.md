# GitHub Copilot CLI

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | GitHub |
| Product | GitHub Copilot CLI |
| Binary | unconfirmed |
| Distribution | Official |
| Storage family | unknown |

## The question that decides the tier

Copilot's history may be tied to the user's GitHub account rather than to the
local machine. If session state is authoritative on GitHub's side and the local
tree is only a cache, Copilot is an F5 agent and stays T0 with reason
`server_backed`. That is a legitimate and useful result.

The probe must distinguish these three cases before any parser is written:

1. **Local authoritative history** — plain files or a database on disk that
   survive a reinstall. F1 or F3, T1 reachable.
2. **Local cache of server state** — files that exist but are rebuilt from the
   account. Not indexable; records would vanish under the user. T0.
3. **No local artifact at all.** T0, reason `server_backed`.

Case 2 is the trap. A cache looks exactly like case 1 to a naive scanner, and
the failure only appears later when a user's indexed sessions disappear.
Distinguishing them requires observing the tree across a cache clear or a
re-login, which the probe task must actually do rather than infer.

## What the probe must settle

1. The binary name and whether a session or history surface is documented.
2. Whether any local artifact exists on macOS and native Windows.
3. Which of the three cases above applies.
4. Whether authentication material sits in the same tree. If so it goes in
   `Excluded` before any read.
5. Whether a documented resume argv exists.

## Sources

None verified. Establish and record vendor sources before promoting any row.
