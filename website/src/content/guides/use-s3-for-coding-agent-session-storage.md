---
title: "Use Amazon S3 for Encrypted Coding-Agent Session Storage"
description: "Configure a private Amazon S3 bucket and least-privilege credentials for Reinstate, then verify encrypted Claude Code or Codex session storage."
answer: "To use Amazon S3 with Reinstate, create a private general purpose bucket, grant a dedicated non-root credential access only to the Reinstate object prefix, initialize Reinstate with the matching regional S3 endpoint and Region, then dry-run and push one selected session."
author: "Harjot Singh Rana"
publishedAt: 2026-07-27
updatedAt: 2026-07-27
reviewedAt: 2026-07-27
tags: ["Amazon S3", "encrypted storage", "session sync", "least privilege", "coding agents"]
targetQuery: "use Amazon S3 for coding agent session storage"
searchIntent: "how-to"
related:
  - title: "Claude Code integration"
    path: "/integrations/claude-code"
  - title: "Codex CLI integration"
    path: "/integrations/codex"
  - title: "Cloudflare R2 storage guide"
    path: "/guides/use-cloudflare-r2-for-coding-agent-session-storage"
  - title: "Getting started documentation"
    path: "/docs/getting-started"
  - title: "Security model"
    path: "/docs/security-model"
  - title: "Compatibility and tested versions"
    path: "/compatibility"
draft: false
noindex: false
agent: "general"
difficulty: "intermediate"
estimatedMinutes: 15
estimatedTaskMinutes: 35
prerequisites:
  - "An AWS account and authority to create a private S3 bucket, IAM policy, and access key"
  - "Reinstate v0.1.0 on a compatible device with Claude Code or Codex CLI"
  - "A harmless session in a repository whose absolute local path you know"
  - "A long encryption passphrase that will be entered privately and is not stored"
howToSteps:
  - name: "Create a private S3 bucket"
    text: "Create a general purpose bucket in one AWS Region, keep all Block Public Access settings enabled, and record the bucket name and Region separately."
    anchor: "create-private-bucket"
  - name: "Grant only the required S3 permissions"
    text: "Attach a prefix-limited policy to a dedicated non-root IAM principal and create the access-key pair that Reinstate can store in the operating-system keyring."
    anchor: "create-scoped-credentials"
  - name: "Install and check Reinstate"
    text: "Install the pinned Reinstate release, verify its version, and run the read-only setup check before writing local configuration."
    anchor: "install-and-check"
  - name: "Initialize Reinstate with the S3 endpoint"
    text: "Pass the regional S3 service endpoint, matching AWS Region, bucket name, and project mapping to init, then enter both credential values through hidden prompts."
    anchor: "initialize-reinstate"
  - name: "Dry-run, push, and verify ciphertext storage"
    text: "Select one harmless Claude Code or Codex session, inspect a non-mutating push plan, push it, and verify only encrypted objects under the generated profile prefix."
    anchor: "push-and-verify"
---

## What this guide configures

This guide connects Reinstate `v0.1.0` to an existing Amazon S3 bucket.
Amazon Web Services owns the bucket, Region, IAM identity, access key, public
access settings, retention, and billing. Reinstate owns the local project
mapping, encrypted profile manifest, encrypted session snapshots, and
same-vendor restore workflow.

| Responsibility | Amazon S3 or IAM | Reinstate |
| --- | --- | --- |
| Create the bucket | Yes | No |
| Create and revoke credentials | Yes | No |
| Keep the bucket private | Yes | No |
| Test object write and delete access | Receives API calls | `rein init` runs the probe |
| Encrypt session content before upload | No | Yes, locally with the passphrase |
| Choose the session and project | No | Yes |
| Delete the provider bucket | Yes | No |

`rein init` does not create a bucket or IAM principal. It expects those
provider resources to exist, writes and deletes a two-byte probe under the new
profile prefix, and saves local configuration only after that probe succeeds.
The first persistent `manifest.age` is created by the first successful push,
not by first-device initialization.

## Key points

- Use a private general purpose bucket and keep Amazon S3 Block Public Access
  enabled. Reinstate does not need a website endpoint or public object access.
- The S3 service endpoint and bucket are separate inputs. For a normal Region,
  use `https://s3.AWS_REGION.amazonaws.com`, then pass the same Region through
  `--region`.
- Reinstate requires `ListBucket`, `GetObject`, `PutObject`, and `DeleteObject`
  capability for its generated `profiles/<profile_id>/` object tree.
- Reinstate encrypts the manifest and snapshots before S3 receives them. S3's
  default server-side encryption is an additional provider control, not a
  replacement for Reinstate's passphrase.
- Reinstate accepts an access-key ID and secret-access-key pair but no AWS session
  token. This is a current limitation because AWS recommends temporary
  credentials where possible.
- Session resume remains **same-vendor**: Claude Code to Claude Code and Codex
  CLI to Codex CLI.
- Reinstate v0.1.0 is a stable pre-1.0 release. Two-device Phase 1 acceptance
  passed on macOS arm64 and native Windows amd64. This guide is not itself
  acceptance evidence.

## Before you begin

Use a dedicated bucket for the first evaluation when practical. A dedicated
bucket makes IAM scope, cleanup, cost attribution, and accidental public-access
review easier than sharing an unrelated production bucket.

Amazon S3 bucket names are not secrets, but they can reveal organization or
project names. Use placeholders in tickets, screenshots, analytics, and coding
agent prompts. Never paste an access-key ID, secret access key, passphrase,
real transcript, or downloaded snapshot into a prompt.

### Platform and release qualifications

| Environment | Installer path | Current qualification |
| --- | --- | --- |
| macOS native arm64 | POSIX installer | Current source compatibility evidence covers the documented Claude Code and Codex ranges. |
| macOS native amd64 | POSIX installer | Installer-compatible; not covered by the two-device acceptance run. |
| Windows 11 native amd64 | PowerShell installer | Covered by the two-device Phase 1 acceptance run, which passed. |
| Linux native | POSIX installer | Installer-compatible; not covered by the two-device acceptance run. |
| WSL2 amd64 | POSIX installer | Installer-compatible and documented for smoke testing; not covered by the two-device acceptance run. WSL1 is unsupported. |

Storage access does not override agent compatibility. `rein setup check` must
report `SUPPORTED` for the installed agent before a push or pull.

## Command placeholders and parameters

| Value or flag | Meaning |
| --- | --- |
| `AWS_REGION` | The bucket's immutable AWS Region, such as `us-east-1`; use the real value from the bucket properties. |
| `https://s3.AWS_REGION.amazonaws.com` | The regional S3 REST service endpoint. Replace `AWS_REGION`; do not use an S3 website endpoint. |
| `YOUR_PRIVATE_BUCKET` | The bucket name only. Do not append it to the endpoint URL. |
| `local/my-project` | A stable, non-secret project ID reused on every Reinstate device. |
| `/absolute/path/to/my-project` | The current device's real absolute repository path. |
| `AGENT` | Replace with exactly `claude` or `codex`. |
| `SESSION_ID` | Copy the exact intended ID from `rein list --agent AGENT`. |
| `--region AWS_REGION` | Sets the AWS signing Region; it must match the bucket and endpoint. |
| `--session SESSION_ID` | Selects one session instead of every discovered session. |
| `--dry-run` | Authenticates, validates, and reports the plan without uploading the selected session. |

<h2 id="create-private-bucket">1. Create a private S3 bucket</h2>

In the Amazon S3 console:

1. Create a **general purpose bucket**.
2. Choose the Region where the encrypted session objects should live.
3. Keep Object Ownership at the default bucket-owner-enforced setting with
   ACLs disabled.
4. Keep all four Block Public Access settings enabled.
5. Do not enable static website hosting.
6. Record the bucket name and Region separately.

AWS documents that bucket name, owner, and Region cannot be changed after
creation, and recommends leaving Block Public Access enabled unless a workload
explicitly requires public access. Reinstate does not require it. Review the
official [S3 bucket creation guide](https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html),
[Block Public Access guidance](https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-control-block-public-access.html),
and [regional endpoint table](https://docs.aws.amazon.com/general/latest/gr/s3.html)
before choosing the Region.

Do not add an Object Lock retention rule to the evaluation prefix. Reinstate's
initialization probe must delete its temporary object, and later sync
operations update the encrypted manifest. AWS documents that Object Lock can
prevent deletion or overwrite during retention in the
[S3 Object Lock guide](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lock.html).

**Expected result:** the S3 console shows one private general purpose bucket in
the intended Region, all Block Public Access controls enabled, ACLs disabled,
and no website endpoint. You have recorded only the non-secret bucket name,
Region, and regional REST endpoint.

<h2 id="create-scoped-credentials">2. Grant only the required S3 permissions</h2>

Reinstate uses these S3 operations:

| Reinstate behavior | S3 API behavior | Required IAM action |
| --- | --- | --- |
| List the profile object tree | `ListObjectsV2` | `s3:ListBucket` |
| Read a manifest or snapshot | `GetObject` or `HeadObject` | `s3:GetObject` |
| Write the probe, manifest, or snapshot | `PutObject` | `s3:PutObject` |
| Remove the initialization probe | `DeleteObject` | `s3:DeleteObject` |

AWS's [S3 API permission mapping](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-with-s3-policy-actions.html)
confirms the bucket-level and object-level actions. The following identity
policy is a starting point for a dedicated bucket. Replace the placeholder in
both ARNs; it contains no credential:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ListReinstateProfiles",
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::REPLACE_WITH_BUCKET_NAME",
      "Condition": {
        "StringLike": {
          "s3:prefix": ["profiles/*"]
        }
      }
    },
    {
      "Sid": "ManageReinstateProfileObjects",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::REPLACE_WITH_BUCKET_NAME/profiles/*"
    }
  ]
}
```

Attach the reviewed policy to a dedicated non-root IAM principal. Reinstate does not
accept the session token that accompanies AWS STS temporary credentials, so it
cannot yet follow AWS's preferred temporary-credential path. If your
organization permits this evaluation, create a long-term access-key pair for
that dedicated least-privilege principal, rotate it according to policy, and
never create or use root access keys. Read AWS's
[IAM security best practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)
and [access-key security guidance](https://docs.aws.amazon.com/IAM/latest/UserGuide/securing_access-keys.html)
before proceeding.

**Expected result:** the dedicated credential can list only the permitted
`profiles/` keyspace and can get, put, and delete objects there. It cannot
change bucket policy, public access, lifecycle, website configuration, or
unrelated buckets. Keep the access-key ID and secret access key in a password
manager until the hidden Reinstate prompts appear.

<h2 id="install-and-check">3. Install and check Reinstate</h2>

On macOS, Linux, or WSL2:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

On native Windows PowerShell:

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

If policy requires inspection first, use the download-and-review variants in
the [getting-started documentation](/docs/getting-started).

```sh
rein version --json
rein setup check
```

**Expected result:** the pinned installer reports `v0.1.0`. Before
initialization, `rein setup check` exits with code `3` and reports
`config missing`. Resolve a platform, keyring, or installed-agent
compatibility failure separately; a working S3 bucket cannot make an
unsupported agent layout safe.

<h2 id="initialize-reinstate">4. Initialize Reinstate with the S3 endpoint</h2>

Replace every placeholder before running:

```sh
rein init \
  --endpoint https://s3.AWS_REGION.amazonaws.com \
  --region AWS_REGION \
  --bucket YOUR_PRIVATE_BUCKET \
  --project local/my-project=/absolute/path/to/my-project
```

The endpoint is the regional REST service endpoint, not the bucket URL and not
an `s3-website` endpoint. The bucket stays in `--bucket`. The first hidden
prompt accepts the dedicated access-key ID; the second accepts its secret
access key. Do not put either value in this command, an environment variable,
shell history, or a coding-agent conversation.

During initialization, Reinstate:

1. generates a non-secret `profile_id`;
2. derives the default `profiles/<profile_id>` prefix;
3. sends a conditional two-byte probe object;
4. deletes that probe;
5. stores the S3 credential reference in config and the credential values in
   the operating-system keyring; and
6. writes local `config.toml` and `state.json` only after the remote probe
   succeeds.

Then validate:

```sh
rein setup check
rein doctor --self-test
```

**Expected result:** initialization prints
`initialized reinstate home (config.toml + state.json)`, a
`profile_id=<UUID>`, and a keyring credential reference. Both checks exit `0`
when every installed-agent compatibility check also passes. The S3 bucket has
no persistent manifest yet; the temporary probe was removed.

<h2 id="push-and-verify">5. Dry-run, push, and verify ciphertext storage</h2>

Use a harmless test session and replace `AGENT` with `claude` or `codex`:

```sh
rein list --agent AGENT
rein push --agent AGENT --session SESSION_ID --dry-run
rein push --agent AGENT --session SESSION_ID
rein status
```

Enter the encryption passphrase through Reinstate's hidden prompt. It is
separate from the S3 secret access key and is not stored by Reinstate. Do not
use `--all` unless you deliberately intend to sync every discovered session.

**Expected result:** the preview reports
`would push ... snapshot(s)` with `dry_run=true`. The mutating command reports
`pushed ... snapshot(s)` with `dry_run=false`, and status reports a remote
revision. In the S3 console, the exact generated prefix contains:

```text
profiles/<profile_id>/manifest.age
profiles/<profile_id>/snapshots/<opaque-id>.age
```

There should be no plaintext transcript, auth file, `.env` file, access key,
or passphrase object. A `.age` filename alone is not cryptographic proof; the
committed [Phase 1 acceptance runbook](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/testing/phase-1-mac-windows-acceptance.md)
defines the controlled ciphertext inspection used for release qualification.
This personal check does not claim physical Reinstate acceptance.

## Security boundaries

- Reinstate encrypts the session locally before upload. Amazon S3 still sees
  the AWS account, bucket, object sizes, request timing, and encrypted object
  keys.
- Amazon S3 applies default server-side encryption to new uploads. That
  provider-side layer is useful defense in depth but does not replace
  Reinstate's client-side age encryption. See AWS's
  [default encryption documentation](https://docs.aws.amazon.com/AmazonS3/latest/userguide/default-bucket-encryption.html).
- Reinstate does not request provider-side encryption headers. A bucket default such
  as SSE-S3 applies normally; a policy that requires request-specific SSE-KMS
  headers can reject the probe or upload. A KMS-default bucket also needs the
  organization's KMS permissions in addition to the example S3 policy.
- Compromise of the bucket alone should not reveal transcript plaintext
  without the Reinstate passphrase, but an attacker with write or delete access
  can still remove, replace, or roll back ciphertext. Encryption is not
  availability protection.
- Known auth files, OAuth material, credential stores, `.env` files, caches,
  and logs are excluded. A secret typed into a transcript is still transcript
  content and must be rotated if exposed.

## Failure modes and common errors

### Why does initialization report `storage probe put failed` (exit `4`)?

**Likely cause:** Wrong endpoint, Region, bucket, credential, or missing
`s3:PutObject`.

**Safe next action:** Compare the bucket Region with both `--region` and the
official regional endpoint; review the scoped policy.

### Why does initialization report `storage probe cleanup failed` (exit `4`)?

**Likely cause:** Missing `s3:DeleteObject`, Object Lock, retention, or a deny
policy blocked probe removal.

**Safe next action:** Preserve the error, remove the stray probe through
approved provider tooling, and fix delete access or retention before retrying.

### What causes `SignatureDoesNotMatch` or an authorization failure?

**Likely cause:** Endpoint and signing Region differ, the key pair is wrong, or
policy denies the request.

**Safe next action:** Do not broaden to administrator access; correct the exact
Region, endpoint, credential, and prefix policy.

### Why is `config missing` (exit `3`) after failed init?

**Likely cause:** Reinstate intentionally did not save local config because the
remote probe failed.

**Safe next action:** Fix S3 first and rerun the same reviewed `rein init`.

### What should I do after compatibility exit `5` or `UNTESTED`?

**Likely cause:** The installed coding-agent version or layout lacks release
evidence.

**Safe next action:** Stop transfer and check the
[compatibility matrix](/compatibility).

### Why does Reinstate report `no matching local sessions found` (exit `2`)?

**Likely cause:** `AGENT` or `SESSION_ID` does not identify a discoverable
local session.

**Safe next action:** Run `rein list --agent claude` or
`rein list --agent codex` and select one exact ID.

### Why is `remote profile manifest not found` on another device?

**Likely cause:** Profile UUID, bucket, prefix, endpoint, or Region differs from
the first device.

**Safe next action:** Reuse every non-secret storage coordinate and the exact
first-device `profile_id`; do not create an empty manifest.

### What should I do after conflict exit `6`?

**Likely cause:** The local and remote copies diverged after sync.

**Safe next action:** Preserve the conflict record and both histories; resolve
explicitly instead of deleting S3 objects.

## Safe rollback and undo

Reinstate does not provide a general `rein undo` or remote-session delete
command.

1. A push `--dry-run` creates no session object, so it needs no rollback.
2. A failed storage probe does not save `config.toml` or `state.json`, although
   Reinstate may already have created its local home directories.
3. A successful first-device init has saved local configuration and the
   credential in the OS keyring, but its temporary S3 probe has been deleted.
4. If the configuration is wrong, stop before pushing. `rein init --force`
   backs up local config and state before replacement; it does not delete
   remote objects or revoke AWS credentials.
5. After a push, never delete only `manifest.age`. Stop every device, record
   the exact profile UUID, preserve needed backups, then remove the complete
   `profiles/<profile_id>/` tree through approved S3 tooling only when its loss
   is acceptable.
6. Revoke or rotate the dedicated access key after cleanup. Deleting local
   Reinstate data does not revoke the IAM key or delete S3 objects.

If S3 Versioning is enabled, an ordinary delete can leave noncurrent object
versions and a delete marker. AWS documents the separate
[version-aware deletion process](https://docs.aws.amazon.com/AmazonS3/latest/userguide/DeletingObjectVersions.html).
Permanent deletion is irreversible.

## Verification checklist

- [ ] The target is a private, general purpose S3 bucket in the intended
      Region; Block Public Access remains enabled and static website hosting is
      not in use.
- [ ] The dedicated IAM principal is limited to `s3:ListBucket`,
      `s3:GetObject`, `s3:PutObject`, and `s3:DeleteObject` for the reviewed
      bucket and `profiles/` keyspace.
- [ ] `rein setup check` and `rein doctor --self-test` complete without a
      compatibility, keyring, filesystem, or crypto failure.
- [ ] The reviewed `rein push --agent AGENT --session SESSION_ID --dry-run`
      selects exactly one intended session and reports no mutation.
- [ ] The real push succeeds, `rein status` reports the expected remote
      revision, and the bucket contains only age-encrypted `manifest.age` and
      `snapshots/*.age` objects beneath the generated profile prefix.
- [ ] No command output, shell history, commit, log, or screenshot contains the
      access-key secret or passphrase, and no physical two-device acceptance is
      claimed until the relevant release scenario has passed.

## Current limitations

- Reinstate accepts an access-key ID and secret access key, but not the session token
  required for AWS STS temporary credentials.
- Reinstate does not create or delete S3 buckets, IAM identities, policies,
  KMS keys, lifecycle rules, versioning configuration, or Object Lock rules.
- The example IAM policy assumes the default `profiles/<profile_id>` layout
  and a bucket whose default encryption does not require extra request headers.
- Glacier or other archival lifecycle transitions can make a current snapshot
  unavailable to an ordinary pull. Reinstate does not coordinate provider
  lifecycle restoration.
- Phase 1 transfers full immutable session snapshots; delta transfer,
  retention controls, and remote garbage collection remain later work.
- Current native resume is same-vendor only and the stable physical platform
  acceptance matrix remains incomplete.

## Amazon S3 storage FAQ

### Does the S3 bucket need to be public?

No. Keep Block Public Access enabled and ACLs disabled. Reinstate uses signed
S3 API requests with the dedicated credential; it does not need a public URL,
static website endpoint, CORS rule, or custom domain.

### Can Reinstate use AWS IAM Identity Center or an assumed role?

Not directly in Reinstate. Temporary AWS credentials include a session token, while
the current S3 backend accepts only an access-key ID and secret access key.
This limitation is why the guide narrowly scopes and calls for rotation of any
long-term evaluation key.

### Can I require SSE-KMS on the bucket?

A bucket default can apply SSE-KMS, but the IAM principal also needs the
relevant KMS permissions. Reinstate does not send request-specific SSE headers, so a
bucket policy that requires such a header can reject initialization. Test with
a harmless profile and follow your organization's KMS policy.

### Can several Reinstate profiles share one S3 bucket?

Yes. The default keyspace is `profiles/<profile_id>/`, so profiles have
separate object prefixes. A dedicated bucket is still simpler for
least-privilege review and cleanup because a first-device profile UUID is
generated during initialization.

### What happens if the bucket is compromised?

An attacker who has only bucket contents should receive age-encrypted manifests
and snapshots, not plaintext session data. An attacker with object permissions
can still delete or replace ciphertext, observe metadata, and deny sync. Rotate
exposed AWS credentials, preserve local copies, and follow the
[security model](/docs/security-model).

### How do I add the second computer?

Finish one successful push first so `manifest.age` exists. On the second
computer, run `rein init --profile-id DEVICE_A_PROFILE_UUID` with the same
endpoint, Region, bucket, credential scope, passphrase, and canonical project
ID but that computer's own absolute path. Then follow the agent-specific
Claude Code or Codex guide for the pull and native resume.
