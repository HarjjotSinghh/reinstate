# Content brief: use S3 for coding-agent session storage

## Page

- Proposed title: Use Amazon S3 for Coding-Agent Session Storage
- URL: `/guides/use-s3-for-coding-agent-session-storage`
- Page type: provider-specific storage guide
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending
- Target release: `v0.1.0-rc.8`
- Last reviewed: 2026-07-27

## Audience and intent

- Primary audience: developers using a private Amazon S3 bucket for encrypted
  Reinstate session snapshots.
- Primary problem: choose correct endpoint, region, bucket, and least-privilege
  object permissions without exposing credentials.
- Primary query: `use S3 for coding agent session storage`
- Secondary questions: required operations; encryption boundary; retention;
  bucket versioning; what appears remotely; how to verify.
- Search intent: how-to
- Expected next action: configure a private bucket, pass the real `init`
  storage probe and the separate redacted local self-test, then dry-run a
  synthetic session transfer.
- Existing-page overlap reviewed: `/docs/storage` owns the provider-neutral
  contract; this guide owns the AWS-specific procedure.

## Product truth

- Current capabilities used: S3-compatible backend, local age encryption,
  immutable snapshots, encrypted manifest, storage probe, scoped credentials.
- Current limitations: BYO storage charges and policy remain the user's
  responsibility; session tokens are not an RC8 credential field; stable
  platform acceptance remains open.
- Version tested: Reinstate `v0.1.0-rc.8` source contract.
- Evidence: S3 backend tests, config schema, CLI help, threat model, and current
  storage documentation.
- Claims that require verification: AWS console UI, prices, service limits, and
  any production account result.
- Prohibited claims: formally audited encryption, zero operational cost,
  automatic deletion, or credential sync.

## Outline

- H1: How to use S3 for encrypted coding-agent session storage
- Direct answer: create a private bucket, grant only required object
  operations, initialize Reinstate with its service coordinates so the real
  storage probe passes, run the local synthetic self-test, then dry-run one
  scoped push.
- Structure: prerequisites; key points; bucket/policy; init; verify; dry-run;
  expected objects; parameters; failure modes; rollback; retention; FAQ.
- Original value: maps Reinstate's exact encrypted object layout and operation
  set to an AWS S3 deployment without relying on a public bucket.

## Links, schema, and media

- Inbound opportunities: storage docs, getting started, security, S3/R2
  comparison points.
- Outbound internal links: storage contract, security model, CLI reference,
  sync guide, troubleshooting, compatibility.
- Primary type: `TechArticle`; additional type: visible-step `HowTo`.
- Diagram: local plaintext boundary → age ciphertext → private S3 bucket.
- Alt text: locally encrypted Reinstate snapshot uploaded to a private S3
  bucket while keys and plaintext remain on the developer's device.
- Raw data: no provider price or throughput benchmark is asserted.

## Acceptance criteria

- [x] Provider-specific intent does not duplicate the neutral storage page
- [x] Commands include expected evidence, parameters, platform differences,
  failure paths, and recovery
- [x] Credentials and passphrases use only documented private channels
- [x] Current limitations and user-owned retention responsibility are visible
- [x] HowTo schema mirrors visible steps
- [x] Route-specific social card and local discoverability gates pass
