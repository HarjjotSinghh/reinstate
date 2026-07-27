# Content brief: encrypted snapshot format v1

## Page

- Title: Reinstate Encrypted Session Snapshot Format v1
- URL: `/research/encrypted-snapshot-format-v1`
- Type: versioned implementation specification
- Owner: Harjot Singh Rana
- Status: agent review accepted; maintainer sign-off pending

## Audience and intent

- Audience: security reviewers, contributors, storage operators, and tool authors evaluating Reinstate.
- Problem: users need to know what current `.age` objects contain and which checks run before restore.
- Primary query: `Reinstate snapshot format`
- Secondary questions: what is in `manifest.age`; what is encrypted; what does the hash cover; is this an open standard; what can change before 1.0?
- Intent: evaluation and security.
- Next action: inspect the exact source and tests for schema v1.

## Product truth

- Capabilities: encrypted manifest JSON v1; immutable encrypted session snapshot v1; newline-delimited metadata followed by one TAR adapter artifact; size, identity, safe-path, hash, compatibility, and restore checks.
- Limitations: the v1 contract is an internal pre-1.0 Reinstate format, not a standards-body specification or cross-agent interchange standard.
- Evidence: `internal/schema/envelope.go`, `internal/schema/manifest.go`, `internal/sync/push.go`, `internal/crypto/envelope.go`, and adapter export/restore implementations.
- Version: envelope schema 1 and manifest schema 1 in `v0.1.0-rc.6`.

## Outline

- H1: Reinstate encrypted session snapshot format v1
- Direct answer: describe the two encrypted object types, the snapshot plaintext framing, and the non-standard boundary.
- H2/H3: status and scope; object layout; manifest fields; snapshot envelope fields; TAR payload; encryption; validation; evolution and migration.

## Links

- Inbound: research hub, glossary, open-source page, storage/security documentation.
- Outbound internal: `/research`, `/security`, `/docs/architecture`, `/compatibility`, `/glossary`.
- External sources: immutable source/test URLs and the age project documentation.

## Structured data

- Primary type: `TechArticle`.
- Breadcrumb: Home → Research and evidence → Encrypted snapshot format v1.
- Optional types: none.

## Media

- Text flow and field tables are preferable to a screenshot.
- Example metadata is explicitly synthetic and uses placeholder UUIDs and paths.
- No example ciphertext, passphrase, bucket, or user transcript.

## Acceptance criteria

- Every field and bound is supported by current source.
- Metadata/payload framing and hash coverage are described correctly.
- Manifest and snapshot schemas remain distinct.
- The page says it is not an open portability standard and may change before v1.0.
- No compatibility, audit, or interoperability claim is invented.
- The page has a unique social card and passes generated JSON-LD visibility checks.
