# Amp (Sourcegraph)

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1 if a local
thread store exists, otherwise T0 with reason `server_backed`.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Sourcegraph |
| Product | Amp |
| Distribution | Official; CLI and editor extension |
| Storage family | unknown |

## The question that decides the tier

Amp organizes work into threads and markets sharing them, which implies
server-side thread storage. If threads are authoritative on Sourcegraph's side,
Amp is T0 with reason `server_backed`, and the local tree — if any — is a cache
that must not be indexed.

The probe must answer the same three-case question as
[GitHub Copilot CLI](copilot.md): local authoritative store, local cache of
server state, or nothing local.

## Preferred path if a server API exists

If Amp exposes a documented thread-list or thread-export interface, that is a
better integration than reading private files, and it may support tiers that
file-reading cannot. Record any such interface here before choosing a storage
family. Reinstate reads supported interfaces in preference to private storage
wherever the vendor offers one, which is why OpenCode is an F2 agent.

Note that a network-backed source would be a first for the catalog: every
current source is local and offline. That is a design decision for the
maintainer, not something an executor should introduce unilaterally. Raise it
before implementing.

## What the probe must settle

1. Whether a local thread artifact exists on macOS and native Windows.
2. Whether it is authoritative or a cache.
3. Whether a documented thread API or export command exists.
4. Whether credentials sit in the same tree.
5. Whether a documented local resume argv exists.

## Sources

None verified. Establish and record vendor sources before promoting any row.
