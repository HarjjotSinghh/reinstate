# MiniMax

**Confidence: Unverified** — the product itself is not yet identified.
**Current tier:** T0 (`unidentified_product`) · **Phase 5 target:** T0 until
identified.

## Status: blocked on identification

"MiniMax code" was named in the Phase 5 roster request, but this research pass
did not establish which MiniMax product is meant. MiniMax publishes models and
agent products; whether there is an official coding harness with local session
storage is unconfirmed.

Reinstate does not ship a descriptor for a product it cannot name. The catalog
key, display name, and vendor string are part of the public contract — they
appear in `agent:session` references, in `rein doctor` output, and in the
compatibility matrix — and they cannot be guessed and later renamed without
breaking user-facing references.

## What must be answered before any other work

1. Which MiniMax product is the coding agent: an official CLI, an editor
   extension, a web product, or a model exposed through other harnesses.
2. Whether MiniMax distributes it, or whether the coding experience is a
   MiniMax model running inside a third-party harness. **If it is the latter,
   there is no MiniMax agent to add** — the user's sessions live in whatever
   harness they used, and that harness is the catalog entry. Reinstate indexes
   harnesses, not models.
3. If an official harness exists: the binary name, the distribution channel,
   and whether local session artifacts exist.

Question 2 is the likely resolution and is a complete answer. Many models are
consumed through other agents' harnesses; those sessions are already indexable
through the harness that wrote them.

## Escalation

If questions 1 and 2 cannot be answered from official sources, do not
improvise. Report the finding to the coordinator and leave this page as the
record. An honest "not identified" costs nothing; a wrong catalog key is a
public interface mistake.

## Sources

None. This page is a research placeholder.
