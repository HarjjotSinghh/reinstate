---
title: "Use Cloudflare R2 for Encrypted Coding-Agent Session Storage"
description: "Configure a private Cloudflare R2 bucket and bucket-scoped S3 credentials for Reinstate, then verify encrypted Claude Code or Codex storage."
answer: "To use Cloudflare R2 with Reinstate, create a private R2 bucket, issue an Object Read & Write S3 API token scoped to that bucket, initialize Reinstate with the account or jurisdiction endpoint and region auto, then dry-run and push one selected session."
author: "Harjot Singh Rana"
publishedAt: 2026-07-27
updatedAt: 2026-07-27
reviewedAt: 2026-07-27
tags: ["Cloudflare R2", "encrypted storage", "session sync", "S3 API", "coding agents"]
targetQuery: "use Cloudflare R2 for coding agent session storage"
searchIntent: "how-to"
related:
  - title: "Claude Code integration"
    path: "/integrations/claude-code"
  - title: "Codex CLI integration"
    path: "/integrations/codex"
  - title: "Amazon S3 storage guide"
    path: "/guides/use-s3-for-coding-agent-session-storage"
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
estimatedMinutes: 14
estimatedTaskMinutes: 30
prerequisites:
  - "A Cloudflare account with R2 enabled and authority to create a bucket and R2 API token"
  - "Reinstate v0.2.0-rc.1 on a compatible device with Claude Code or Codex CLI"
  - "A harmless session in a repository whose absolute local path you know"
  - "A long encryption passphrase that will be entered privately and is not stored"
howToSteps:
  - name: "Create a private R2 bucket"
    text: "Create one R2 bucket, choose its location or jurisdiction deliberately, leave public development URL and custom-domain access disabled, and record its name."
    anchor: "create-private-bucket"
  - name: "Create bucket-scoped S3 credentials"
    text: "Generate an R2 S3 API token with Object Read & Write permission for only the selected bucket, then store its access-key pair outside chat and source control."
    anchor: "create-scoped-credentials"
  - name: "Install and check Reinstate"
    text: "Install the pinned Reinstate release, verify its version, and run the read-only setup check before writing local configuration."
    anchor: "install-and-check"
  - name: "Initialize Reinstate with the R2 endpoint"
    text: "Pass the account or jurisdiction-specific R2 S3 endpoint, region auto, bucket name, and project mapping to init, then enter credentials through hidden prompts."
    anchor: "initialize-reinstate"
  - name: "Dry-run, push, and verify ciphertext storage"
    text: "Select one harmless Claude Code or Codex session, inspect a non-mutating push plan, push it, and verify only encrypted objects under the generated profile prefix."
    anchor: "push-and-verify"
---

## What this guide configures

This guide connects Reinstate `v0.2.0-rc.1` to an existing Cloudflare R2
bucket through R2's S3-compatible API. Cloudflare owns the account, bucket,
location, API token, public-access switches, retention, and billing. Reinstate
owns the local project mapping, encrypted profile manifest, encrypted session
snapshots, and same-vendor restore workflow.

| Responsibility | Cloudflare R2 | Reinstate |
| --- | --- | --- |
| Create the bucket | Yes | No |
| Generate and revoke the R2 API token | Yes | No |
| Keep public URL and custom-domain access disabled | Yes | No |
| Test object write and delete access | Receives S3 API calls | `rein init` runs the probe |
| Encrypt session content before upload | No | Yes, locally with the passphrase |
| Choose the session and project | No | Yes |
| Empty or delete the R2 bucket | Yes | No |

`rein init` does not create an R2 bucket or token. It expects them to exist,
writes and deletes a two-byte probe beneath the new profile prefix, and saves
local configuration only after that probe succeeds. The first persistent
`manifest.age` appears after the first successful push, not after first-device
initialization.

## Key points

- R2 buckets are private by default. Keep the public development URL and
  custom-domain access disabled because Reinstate uses authenticated S3 API
  requests.
- Use the R2 S3 endpoint from the token confirmation or account overview:
  `https://ACCOUNT_ID.r2.cloudflarestorage.com`. Keep the bucket name separate
  and use `--region auto`.
- EU or FedRAMP jurisdiction buckets require the matching
  `https://ACCOUNT_ID.JURISDICTION.r2.cloudflarestorage.com` endpoint.
- Choose **Object Read & Write** and apply the token to the specific bucket.
  Read-only access cannot complete Reinstate's write-and-delete initialization
  probe.
- Reinstate encrypts the manifest and snapshots before R2 receives them.
  Cloudflare's encryption at rest is an additional provider control.
- Session resume remains **same-vendor**: Claude Code to Claude Code and Codex
  CLI to Codex CLI.
- Reinstate v0.2.0-rc.1 is a pre-1.0 release candidate. Two-device Phase 1 acceptance
  passed on macOS arm64 and native Windows amd64. This guide is not itself
  acceptance evidence.

## Before you begin

Use a dedicated R2 bucket for the first evaluation when practical. Cloudflare
can scope an Object Read & Write token to one or more complete buckets, not to
a generated Reinstate profile prefix through the dashboard flow. A dedicated
bucket therefore gives the evaluation token a smaller and easier-to-audit
scope.

An R2 account ID and bucket name are not authentication secrets, but they can
still expose organizational identifiers. Redact them from public logs and
screenshots. Never paste the access-key ID, secret access key, encryption
passphrase, transcript, or downloaded snapshot into a coding-agent prompt.

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
| `ACCOUNT_ID` | The non-secret Cloudflare account ID shown in R2 account details. Replace it in the endpoint. |
| `JURISDICTION` | Use only for a jurisdiction bucket, currently `eu` or `fedramp`; omit the segment for a default bucket. |
| `https://ACCOUNT_ID.r2.cloudflarestorage.com` | The default R2 S3 API service endpoint. It does not include the bucket name. |
| `YOUR_PRIVATE_BUCKET` | The R2 bucket name only. Do not append it to the endpoint. |
| `local/my-project` | A stable, non-secret project ID reused on every Reinstate device. |
| `/absolute/path/to/my-project` | The current device's real absolute repository path. |
| `AGENT` | Replace with exactly `claude` or `codex`. |
| `SESSION_ID` | Copy the exact intended ID from `rein list --agent AGENT`. |
| `--region auto` | Uses the signing Region required by the R2 S3-compatible API. |
| `--session SESSION_ID` | Selects one session instead of every discovered session. |
| `--dry-run` | Authenticates, validates, and reports the plan without uploading the selected session. |

<h2 id="create-private-bucket">1. Create a private R2 bucket</h2>

In the Cloudflare dashboard:

1. Open R2 object storage and create a bucket.
2. Choose Automatic location, a location hint, or a jurisdiction according to
   your data-location requirements.
3. Choose the intended default storage class.
4. Leave the public development URL disabled.
5. Do not connect a custom domain.
6. Record the bucket name and whether it uses the default, `eu`, or `fedramp`
   jurisdiction.

Cloudflare's official [R2 bucket guide](https://developers.cloudflare.com/r2/buckets/create-buckets/)
documents creation and states that buckets are not public by default. Public
access requires a separate action described in the
[public-bucket documentation](https://developers.cloudflare.com/r2/buckets/public-buckets/);
Reinstate does not need it. Location hints are best-effort, while jurisdiction
selection controls the required endpoint and cannot later be changed. Review
the [R2 data-location guide](https://developers.cloudflare.com/r2/reference/data-location/)
before choosing.

Do not add a bucket lock to the evaluation prefix. R2 lock rules can prevent
deletion and overwrite, while Reinstate's init probe must be deleted and the
encrypted manifest changes as sessions are pushed. Review
[R2 bucket-lock behavior](https://developers.cloudflare.com/r2/buckets/bucket-locks/)
before combining retention with a live sync profile.

**Expected result:** the R2 dashboard shows one private bucket with public
development URL access disallowed and no custom domain. You have recorded only
the non-secret bucket name, account ID, and jurisdiction needed to choose the
S3 API endpoint.

<h2 id="create-scoped-credentials">2. Create bucket-scoped S3 credentials</h2>

From R2 Overview, open API Tokens and create an account or user token:

1. Choose **Object Read & Write**.
2. Choose **Apply to specific buckets only**.
3. Select only the evaluation bucket.
4. Create the token.
5. Copy the generated Access Key ID, Secret Access Key, and S3 endpoint into a
   password manager. The secret is not shown again.

Cloudflare documents this exact provider flow in
[Get started with the R2 S3 API](https://developers.cloudflare.com/r2/get-started/s3/)
and explains account-token, user-token, bucket scope, and permission behavior
in the [R2 authentication reference](https://developers.cloudflare.com/r2/api/tokens/).
Object Read & Write is the least built-in permission that covers the read,
list, write, and delete behavior needed by the current Reinstate backend.

Reinstate accepts the generated access-key ID and secret access key. It does not
accept an additional session-token field, so do not substitute R2 temporary
credentials that require one. Do not use a general Cloudflare API bearer token;
Reinstate talks to the S3-compatible API.

**Expected result:** the token is limited to Object Read & Write on the one
selected bucket. It cannot administer unrelated buckets, public access,
custom domains, lifecycle, or the rest of the Cloudflare account. You have
stored both generated credential values privately and have the matching S3 API
endpoint.

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

**Expected result:** the pinned installer reports `v0.2.0-rc.1`. Before
initialization, `rein setup check` exits with code `3` and reports
`config missing`. Resolve a platform, keyring, or installed-agent
compatibility failure separately; working R2 credentials cannot make an
unsupported agent layout safe.

<h2 id="initialize-reinstate">4. Initialize Reinstate with the R2 endpoint</h2>

For a default R2 bucket, replace every placeholder before running:

```sh
rein init \
  --endpoint https://ACCOUNT_ID.r2.cloudflarestorage.com \
  --region auto \
  --bucket YOUR_PRIVATE_BUCKET \
  --project local/my-project=/absolute/path/to/my-project
```

For a jurisdiction bucket, insert `eu` or `fedramp` between the account ID and
`r2` exactly as Cloudflare shows:

```text
https://ACCOUNT_ID.JURISDICTION.r2.cloudflarestorage.com
```

The endpoint is the R2 S3 service endpoint, not a public `r2.dev` URL, custom
domain, or bucket path. The first hidden prompt accepts the generated Access
Key ID; the second accepts its Secret Access Key. Do not put either value in
this command, an environment variable, shell history, or a coding-agent
conversation.

During initialization, Reinstate:

1. generates a non-secret `profile_id`;
2. derives the default `profiles/<profile_id>` prefix;
3. sends a conditional two-byte probe object;
4. deletes that probe;
5. stores the R2 credential reference in config and the credential values in
   the operating-system keyring; and
6. writes local `config.toml` and `state.json` only after the remote probe
   succeeds.

Cloudflare's current [S3 compatibility table](https://developers.cloudflare.com/r2/api/s3/api/)
lists the `PutObject`, `GetObject`, `HeadObject`, `DeleteObject`, and
`ListObjectsV2` operations used by Reinstate and requires Region `auto`.

Then validate:

```sh
rein setup check
rein doctor --self-test
```

**Expected result:** initialization prints
`initialized reinstate home (config.toml + state.json)`, a
`profile_id=<UUID>`, and a keyring credential reference. Both checks exit `0`
when every installed-agent compatibility check also passes. The R2 bucket has
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
separate from the R2 secret access key and is not stored by Reinstate. Do not
use `--all` unless you deliberately intend to sync every discovered session.

**Expected result:** the preview reports
`would push ... snapshot(s)` with `dry_run=true`. The mutating command reports
`pushed ... snapshot(s)` with `dry_run=false`, and status reports a remote
revision. In the R2 dashboard, the exact generated prefix contains:

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

- Reinstate encrypts the session locally before upload. Cloudflare still sees
  the account, bucket, object sizes, request timing, and encrypted object keys.
- Cloudflare documents automatic encryption at rest and TLS-protected transfer
  for R2. Those provider controls are defense in depth, not a replacement for
  Reinstate's client-side encryption. See the official
  [R2 data-security reference](https://developers.cloudflare.com/r2/reference/data-security/).
- Compromise of the bucket alone should not reveal transcript plaintext
  without the Reinstate passphrase, but an attacker with write or delete access
  can still remove, replace, or roll back ciphertext. Encryption is not
  availability protection.
- Do not enable a public development URL or custom domain for session storage.
  A private S3 endpoint with scoped credentials is the intended access path.
- Known auth files, OAuth material, credential stores, `.env` files, caches,
  and logs are excluded. A secret typed into a transcript is still transcript
  content and must be rotated if exposed.

## Failure modes and common errors

### Why does initialization report `storage probe put failed` (exit `4`)?

**Likely cause:** Wrong endpoint, bucket, credential, jurisdiction, or a
read-only token.

**Safe next action:** Compare the token confirmation endpoint and selected
bucket; require Object Read & Write on that bucket only.

### Why does initialization report `storage probe cleanup failed` (exit `4`)?

**Likely cause:** Delete is denied or a bucket lock covers the generated probe
prefix.

**Safe next action:** Preserve the error, remove the stray probe through
approved R2 tooling, and fix the token or lock before retrying.

### What causes `SignatureDoesNotMatch` or an authorization failure?

**Likely cause:** The account ID, jurisdiction endpoint, access-key pair, or
Region is wrong.

**Safe next action:** Use the exact R2 S3 endpoint, `--region auto`, and the
credential generated for the selected bucket.

### Why is `config missing` (exit `3`) after failed init?

**Likely cause:** Reinstate intentionally did not save local config because the
remote probe failed.

**Safe next action:** Fix R2 first and rerun the same reviewed `rein init`.

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

**Likely cause:** Profile UUID, bucket, prefix, endpoint, or jurisdiction
differs from the first device.

**Safe next action:** Reuse every non-secret storage coordinate and the exact
first-device `profile_id`; do not create an empty manifest.

### What should I do after conflict exit `6`?

**Likely cause:** The local and remote copies diverged after sync.

**Safe next action:** Preserve the conflict record and both histories; resolve
explicitly instead of deleting R2 objects.

## Safe rollback and undo

Reinstate does not provide a general `rein undo` or remote-session delete
command.

1. A push `--dry-run` creates no session object, so it needs no rollback.
2. A failed storage probe does not save `config.toml` or `state.json`, although
   Reinstate may already have created its local home directories.
3. A successful first-device init has saved local configuration and the
   credential in the OS keyring, but its temporary R2 probe has been deleted.
4. If the configuration is wrong, stop before pushing. `rein init --force`
   backs up local config and state before replacement; it does not delete
   remote objects or revoke the R2 token.
5. After a push, never delete only `manifest.age`. Stop every device, record
   the exact profile UUID, preserve needed backups, then remove the complete
   `profiles/<profile_id>/` tree through approved R2 tooling only when its loss
   is acceptable.
6. Revoke the dedicated R2 API token after cleanup. Deleting local Reinstate
   data does not revoke the token or delete R2 objects.

Cloudflare documents that emptying or deleting a bucket is irreversible and
that a bucket must be empty before deletion. Review the official
[R2 deletion procedure](https://developers.cloudflare.com/r2/buckets/delete-buckets/)
before any cleanup. Do not apply a short lifecycle rule as a casual undo:
new objects uploaded while the rule is active are also subject to expiration.

## Verification checklist

- [ ] The selected R2 bucket is private: its `r2.dev` development URL is
      disabled and no custom domain exposes the session objects.
- [ ] The dedicated API token has Object Read & Write access only to the
      reviewed bucket, and its S3 endpoint matches the intended standard,
      European Union, or FedRAMP jurisdiction.
- [ ] Reinstate uses that exact endpoint with `--region auto`, and
      `rein setup check` plus `rein doctor --self-test` complete without a
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

- Reinstate does not create or delete R2 buckets, API tokens, public-access
  settings, custom domains, lifecycle rules, storage classes, or bucket locks.
- Reinstate accepts the access-key pair generated for an R2 API token but no
  additional session-token field.
- Dashboard token scope is bucket-level. Reinstate generates the first profile UUID
  during initialization, so the token cannot be pre-scoped to that exact
  profile through this provider flow.
- Lifecycle expiration or an Infrequent Access policy can affect availability
  and cost. Reinstate does not coordinate R2 lifecycle or storage-class
  transitions.
- R2's S3 compatibility evolves. This guide depends only on the current
  Put/Get/Head/Delete/List and conditional-write behavior used by Reinstate.
- Phase 1 transfers full immutable session snapshots; delta transfer,
  retention controls, and remote garbage collection remain later work.
- Current native resume is same-vendor only and the stable physical platform
  acceptance matrix remains incomplete.

## Cloudflare R2 storage FAQ

### Does the R2 bucket need a public URL or custom domain?

No. Leave the public development URL disabled and do not connect a custom
domain. Reinstate authenticates directly to the private S3 API endpoint with
the bucket-scoped credential.

### Which R2 token permission does Reinstate need?

Use Object Read & Write and apply it only to the selected bucket. Read-only
cannot write snapshots or delete the initialization probe. Admin Read & Write
is broader than this workflow requires.

### Why is the R2 Region `auto`?

Cloudflare's S3 compatibility documentation specifies `auto` as the R2 Region.
The endpoint, not an AWS Region string, identifies the Cloudflare account and
any `eu` or `fedramp` jurisdiction.

### Can several Reinstate profiles share one R2 bucket?

Yes. The default keyspace is `profiles/<profile_id>/`, so profiles have
separate object prefixes. A dedicated bucket is still simpler for token scope
and cleanup because the provider token is bucket-scoped, not
profile-prefix-scoped in this setup.

### What happens if the R2 bucket is compromised?

An attacker who has only bucket contents should receive age-encrypted manifests
and snapshots, not plaintext session data. An attacker with object permissions
can still delete or replace ciphertext, observe metadata, and deny sync. Revoke
the exposed R2 token, preserve local copies, and follow the
[security model](/docs/security-model).

### How do I add the second computer?

Finish one successful push first so `manifest.age` exists. On the second
computer, run `rein init --profile-id DEVICE_A_PROFILE_UUID` with the same
endpoint, `auto` Region, bucket, token scope, passphrase, and canonical project
ID but that computer's own absolute path. Then follow the agent-specific
Claude Code or Codex guide for the pull and native resume.
