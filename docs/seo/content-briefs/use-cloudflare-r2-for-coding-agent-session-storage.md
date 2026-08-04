# Content brief: use Cloudflare R2 for coding-agent session storage

## Page

- Proposed title: Use Cloudflare R2 for Coding-Agent Session Storage
- URL: `/guides/use-cloudflare-r2-for-coding-agent-session-storage`
- Page type: provider-specific storage guide
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers using a private Cloudflare R2 bucket as
  Reinstate's S3-compatible encrypted object store.
- Primary problem: distinguish the R2 service endpoint from the bucket and
  configure scoped credentials safely.
- Primary query: `use Cloudflare R2 for coding agent session storage`
- Secondary questions: endpoint format; required token permissions; region;
  encrypted object layout; verification and cleanup.
- Search intent: how-to
- Expected next action: pass the real `init` storage probe and the separate
  local synthetic self-test, then inspect a one-session push dry-run.
- Existing-page overlap reviewed: `/docs/storage` owns common semantics; this
  guide owns R2 coordinates and provider-specific cautions.

## Product truth

- Current capabilities used: S3-compatible R2 access, local age encryption,
  immutable snapshots, encrypted manifests, probes, and scoped session sync.
- Current limitations: R2 account policy and charges are external; Reinstate accepts
  access-key ID and secret access key but not a session token; Intel macOS and
  Linux/WSL2 are preview and unverified.
- Version tested: stable Reinstate `v0.2.0` contract.
- Evidence: backend/config implementation, storage tests, CLI help, threat
  model, and provider-neutral docs.
- Claims that require verification: current Cloudflare UI, prices, limits, or
  an owner's production account result.
- Prohibited claims: free forever, formal audit, public-bucket requirement,
  credential sync, or automatic retention.

## Outline

- H1: How to use R2 for encrypted coding-agent session storage
- Direct answer: create a private R2 bucket and scoped API token, configure the
  account service endpoint separately from the bucket, pass Reinstate's real
  storage probe and local synthetic self-test, then dry-run a selected
  encrypted push.
- Structure: prerequisites; key points; bucket/token; endpoint; init; self-test;
  push; objects; parameters; failures; rollback; FAQ.
- Original value: calls out the endpoint/bucket distinction that otherwise
  creates a misleading authentication or missing-profile failure.

## Links, schema, and media

- Inbound opportunities: storage docs, security, setup, S3 guide.
- Outbound internal links: storage contract, configuration, CLI reference,
  security model, troubleshooting, compatibility.
- Primary type: `TechArticle`; additional type: visible-step `HowTo`.
- Diagram: local encryption boundary and private R2 object layout.
- Alt text: encrypted session snapshot and manifest traveling from Reinstate
  into a private Cloudflare R2 bucket.
- Raw data: no provider benchmark, pricing, or production-success claim.

## Acceptance criteria

- [x] R2 intent is distinct from the provider-neutral storage contract
- [x] Endpoint, bucket, credentials, expected evidence, and failure recovery
  are explicit
- [x] No secret or private infrastructure value appears in examples
- [x] Same-vendor and pre-1.0 limitations remain visible
- [x] Visible steps and HowTo schema remain identical
- [x] Route-specific social card and local discoverability gates pass
