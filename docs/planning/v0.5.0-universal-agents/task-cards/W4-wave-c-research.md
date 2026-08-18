# W4 — Wave C research cards

These tasks are research first and code second. For several of them the
expected deliverable is a descriptor at **T0 with a correct reason** and a
completed storage page.

**A T0 outcome is a completed task.** Telling a user "Reinstate recognizes
OpenHands, and its history is not readable locally" is a real answer to a real
question. Stretching to reach T1 by indexing a cache is a defect, not effort.

The research half of every card has **no code dependency** and starts on day
one. Only probe capture waits for T-006.

---

## The cache trap

Three cases look identical to a naive scanner:

1. **Local authoritative history** — survives a reinstall and a re-login. T1 is
   reachable.
2. **Local cache of server state** — exists on disk, is rebuilt from the
   account, and disappears when the cache clears. **Not indexable.** Records
   would vanish under the user, and the failure surfaces weeks later as a
   Reinstate bug.
3. **Nothing local.** T0, `server_backed`.

Distinguishing 1 from 2 requires **observing the tree across a cache clear or
a re-login**. Do that. Do not infer it from the directory name.

---

## T-040 — GitHub Copilot CLI

**Target:** T1 or T0 · **Owns:** `internal/agents/catalog/copilot.go`,
`docs/session-storage/copilot.md`, plus a source package only if the outcome is
T1

**Page:** [../../../session-storage/copilot.md](../../../session-storage/copilot.md)

1. Establish the official binary name and whether a session or history surface
   is documented. Record sources on the page.
2. Probe macOS and native Windows.
3. Resolve the three-case question above, empirically.
4. If authentication material sits in the same tree, add it to `Excluded`
   before any read.
5. Check for a documented resume argv.

**Outcome T0** with `server_backed` if history is tied to the GitHub account
rather than the machine. That is likely and it is fine.

---

## T-041 — Amp

**Target:** T1 or T0 · **Owns:** `internal/agents/catalog/amp.go`,
`docs/session-storage/amp.md`, plus a source package only if the outcome is T1

**Page:** [../../../session-storage/amp.md](../../../session-storage/amp.md)

1. Establish whether threads are authoritative server-side. Amp markets thread
   sharing, which is a strong signal that they are.
2. Probe for any local artifact, then resolve the three-case question.
3. Check for a documented thread-list or thread-export interface.

**Escalate before implementing a network source.** Every source in the catalog
today is local and offline. A network-backed source is a first, with
implications for offline behavior, timeouts, credentials, and the security
model. That is a maintainer decision, not an executor decision.

---

## T-042 — OpenHands

**Target:** T0 · **Owns:** `internal/agents/catalog/openhands.go`,
`docs/session-storage/openhands.md`

**Page:** [../../../session-storage/openhands.md](../../../session-storage/openhands.md)

OpenHands typically runs as a service, often containerized. A container's
filesystem is not the host's, and a hosted deployment is not local.

1. Determine whether a host-side artifact exists for a default local
   deployment.
2. Determine whether it survives a container restart and a version upgrade. If
   it does not, it is not a session store.
3. Determine whether the project path is recoverable from it.
4. Record any documented conversation-export API. It does not change the tier
   by itself, but it is the only thing that could later.

**Expected outcome:** T0, `server_backed`. Ship the descriptor and close.

---

## T-043 — ZCode

**Target:** T0 · **Owns:** `internal/agents/catalog/zcode.go`,
`docs/session-storage/zcode.md`

**Page:** [../../../session-storage/zcode.md](../../../session-storage/zcode.md)

**The distribution policy is already decided.**
[ADR 0004](../../../adr/0004-universal-agent-coverage.md) decision 8 restricts
the catalog to officially distributed harnesses. The npm `zcode-app-cli`
package extracts the Z.ai desktop runtime and states it is unaffiliated. It is
not a catalog agent and must not be used to derive a supported layout.

1. Probe the **Z.ai-distributed desktop application**: `~/.zcode`, the platform
   application-support directory, and the Electron user-data directory.
2. If artifacts exist, determine whether they are a transcript or a cache of
   server-held state. The vendor markets task follow-up across desktop, remote,
   and chat channels, which suggests server-side state.
3. Check whether Z.ai publishes a session or export API.

**Expected outcome:** T0, `desktop_only` or `server_backed`.

---

## T-044 — MiniMax

**Target:** T0 or closed · **Owns:** `docs/session-storage/minimax.md`, and
`internal/agents/catalog/minimax.go` only if a product is identified

**Page:** [../../../session-storage/minimax.md](../../../session-storage/minimax.md)

**This task is blocked on identification and may correctly produce no code.**

1. Determine which MiniMax product is meant: an official CLI, an editor
   extension, a web product, or a model exposed through other harnesses.
2. **Determine whether MiniMax distributes a coding harness at all.** If the
   coding experience is a MiniMax model running inside a third-party harness,
   there is nothing to add: the user's sessions live in whatever harness wrote
   them, and that harness is the catalog entry. Reinstate indexes harnesses,
   not models.
3. If an official harness exists, continue with the standard recipe.

**Do not guess a catalog key.** The key, display name, and vendor string are a
public interface: they appear in `agent:session` references, in `doctor`
output, and in the compatibility matrix. A wrong key cannot be renamed later
without breaking user references.

**Close the task** by reporting "not identified" or "model, not harness" and
leaving the page as the record. That is a successful outcome.
