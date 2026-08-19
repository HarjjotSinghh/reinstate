---
title: "Configure storage for encrypted Reinstate sessions"
navTitle: "Storage"
description: "Understand Reinstate's S3-compatible storage fields, object layout, required permissions, encrypted payloads, profile isolation, and retention limits."
order: 11
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.5.0-rc.2"
updatedAt: 2026-08-16
tags: ["storage", "s3", "cloudflare-r2", "encryption", "least-privilege"]
targetQuery: "Reinstate session storage configuration"
searchIntent: "how-to"
draft: false
noindex: false
---

Reinstate `v0.4.0` stores encrypted manifests and immutable session
snapshots in a private S3-compatible bucket you control. Configure the service
endpoint, region, bucket, and profile prefix separately; interactive
`rein init` stores the access-key pair in the operating-system keyring, not in
`config.toml`.

> **Current scope:** Cloudflare R2 is the recommended Phase 1 backend, and
> Amazon S3 is documented. Other S3-compatible services must implement the
> conditional request and object operations Reinstate uses; compatibility is
> not implied by an “S3-compatible” label alone.

## Prerequisites

- A private bucket in an S3-compatible object-storage service.
- A service endpoint and region appropriate for that bucket.
- A dedicated, non-root credential limited to the intended bucket and profile
  prefix.
- Permission to list the prefix and get, put, head, and delete its objects.
- A retention, versioning, and incident-deletion policy you understand.
- Reinstate installed on a trusted local device.

Keep public website hosting and anonymous object access disabled. Reinstate does
not require a Reinstate-hosted account or bucket.

## Storage configuration fields

`rein init` writes non-secret storage coordinates similar to:

```toml
[storage]
type = "s3"
endpoint = "https://<service-endpoint>"
region = "auto"
bucket = "reinstate"
prefix = "profiles/<opaque-profile-id>"
credential_ref = "reinstate/<profile-id>/s3"
```

Use `region = "auto"` for R2. Use the bucket's actual AWS Region and matching
regional endpoint for Amazon S3. The endpoint is the service URL only; do not
append the bucket name. `bucket` and `prefix` are independent fields.

The default prefix is `profiles/<profile_id>`. Every device in a profile must
use the same profile ID, bucket, and prefix. Credentials can differ by device
as long as each credential authorizes the same required object operations.

## Required object operations

| Reinstate behavior | S3 operation or permission |
| --- | --- |
| List the profile tree | `ListObjectsV2` / `s3:ListBucket` scoped by prefix |
| Read a manifest or snapshot | `GetObject`, `HeadObject` / `s3:GetObject` |
| Create a probe, snapshot, or manifest | `PutObject` / `s3:PutObject` |
| Remove the initialization probe | `DeleteObject` / `s3:DeleteObject` |
| Protect manifest updates | Conditional `If-Match` and `If-None-Match` writes |

Grant only the intended bucket and `profiles/<profile_id>/` object prefix.
Provider-side deny policies, object retention, or Object Lock can prevent probe
cleanup or later operations even when an allow policy looks correct.

## Remote object layout

The current backend uses:

```text
<prefix>/manifest.age
<prefix>/snapshots/<opaque-snapshot-id>.age
<prefix>/probes/<random-id>
```

`manifest.age` is an encrypted index of session heads and profile revision.
Snapshots are immutable, UUID-addressed encrypted artifacts. Probe objects
contain a small test value and are deleted during successful initialization.

The provider can observe bucket access, object timestamps, sizes, fixed layout
segments, and opaque IDs. It should not receive plaintext transcripts,
manifest JSON, or the encryption passphrase.

## Initialize and verify storage

Interactive setup is the preferred path:

```sh
rein init \
  --project github.com/acme/app=/absolute/local/path
```

Enter the endpoint, bucket, access-key ID, and secret key in the private
prompts. Initialization creates and deletes a probe before writing local
configuration. On an additional device, pass the first device's profile UUID:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project github.com/acme/app=/this/device/path
```

Reinstate requires that additional-device initialization can find the existing
encrypted `manifest.age`; it does not silently create a new empty remote
profile.

After initialization:

```sh
rein doctor --self-test
rein status --json
rein push --agent AGENT --session SESSION_ID --dry-run --json
```

The doctor self-test uses synthetic data. The scoped dry-run authenticates and
validates the real profile without publishing a snapshot.

## Retention, backup, and deletion

Phase 1 has no Reinstate retention or garbage-collection command. Full
snapshots remain until removed by an explicit provider-side policy or operator
action. Deleting a snapshot referenced by the encrypted manifest can make that
session unrestorable; deleting `manifest.age` breaks an established profile.

Treat provider lifecycle rules, versioning, replication, Object Lock, and
backup as external controls. Review how they interact with required probe
deletion and with incident-response deletion before relying on them.
Reinstate's local pre-restore backups are separate plaintext files on the
destination and are not a substitute for a remote storage recovery plan.

## Expected evidence

- `init` completes both probe creation and cleanup before saving config.
- `config.toml` contains only storage coordinates and a `credential_ref`, not
  an access key or secret key.
- A first successful push creates encrypted `manifest.age` and one opaque
  `.age` snapshot under the configured prefix.
- An additional device verifies the existing manifest and `status` reports the
  same remote revision.
- Inspecting stored bytes does not reveal plaintext transcript or manifest
  JSON.

## Failure paths

- Use [remote-manifest troubleshooting](/docs/troubleshooting#why-does-reinstate-report-a-remote-profile-manifest-is-missing)
  when an established profile appears empty or missing.
- A probe put failure usually means endpoint, region, bucket, credential, or
  `PutObject` scope is wrong; a cleanup failure can indicate missing
  `DeleteObject` or retention policy.
- A signature mismatch often indicates the wrong regional endpoint, region,
  credential pair, or system clock; preserve redacted output.
- Do not bypass a missing manifest by uploading an empty file or changing the
  profile ID.

## Security boundaries

Remote payloads are encrypted locally with age passphrase encryption before
upload and transferred over TLS. The passphrase is never sent to the storage
provider or stored by Reinstate. Interactive storage credentials are stored in
the native OS keyring; non-interactive environment credentials are not
persisted by Reinstate but remain exposed to the invoking process environment.

Encryption does not hide access timing, object size, bucket identity, endpoint,
or every key segment. It also does not protect a compromised local machine or
a secret already embedded in transcript text. Apply least privilege,
provider-side access logging, credential rotation, and an independent backup
policy.

## Related pages

- [Configure a Reinstate profile](/docs/configuration)
- [Use Cloudflare R2 step by step](/guides/use-cloudflare-r2-for-coding-agent-session-storage)
- [Use Amazon S3 step by step](/guides/use-s3-for-coding-agent-session-storage)
- [Review encryption and credential boundaries](/docs/security-model)
- [Understand the sync architecture](/docs/architecture)
