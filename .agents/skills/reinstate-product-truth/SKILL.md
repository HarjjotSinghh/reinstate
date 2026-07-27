---
name: reinstate-product-truth
description: Establish and enforce Reinstate's current product facts before changing website copy, metadata, docs, schema, launch materials, or comparisons. Use whenever a task could introduce or modify a product claim.
---

# Reinstate Product Truth

## Purpose

Prevent positioning drift and unsupported claims across the Reinstate website and repository.

## Current baseline

Treat these as the baseline unless newer code, tests, release notes, and documentation prove otherwise:

- Product: Reinstate
- Category: coding-agent session sync
- Primary outcome: sync coding-agent sessions across devices
- Primary audience: developers switching between work and personal computers or desktop and laptop
- Current agents: Claude Code and Codex
- Current operating systems: macOS and Windows
- Encryption: local before upload
- Storage: user-owned S3-compatible storage, including Amazon S3 and Cloudflare R2
- License: Apache-2.0
- Source: open source
- Product is not a cloud IDE, remote desktop, hosted coding agent, Git replacement, or credential-sync service

## Workflow

1. Read the source of truth if present:
   - `website/src/data/product.ts`
   - `README.md`
   - `website/src/content/docs/`
   - `CHANGELOG.md`
   - current release metadata
2. Resolve any conflict in favor of released code, tests, and current docs.
3. List every proposed claim.
4. Classify each claim:
   - verified
   - planned
   - ambiguous
   - unsupported
5. Use only verified claims in indexable pages and structured data.
6. Label planned features as roadmap items, not current capabilities.
7. Refuse to invent:
   - ratings
   - reviews
   - customer counts
   - benchmark results
   - supported agents
   - supported platforms
   - security guarantees
8. Report conflicts and the files that need synchronization.

## Output

Return:

- verified product facts
- conflicting claims
- unsupported claims removed
- files updated
- evidence used
- unresolved questions
