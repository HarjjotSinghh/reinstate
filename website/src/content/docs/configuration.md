---
title: "Configure Reinstate profiles and project paths"
navTitle: "Profiles and paths"
description: "Configure a Reinstate profile, S3-compatible storage coordinates, OS-keyring credentials, and portable project mappings across development devices."
order: 10
author: "Harjot Singh Rana"
status: current
schemaType: web-page
version: "v0.4.0"
updatedAt: 2026-08-16
tags: ["configuration", "project-mapping", "profile-id", "s3", "keyring"]
targetQuery: "configure Reinstate session sync"
searchIntent: "how-to"
draft: false
noindex: false
---

Run `rein init` once per device. Devices in one sync set use the same
non-secret `profile_id`, bucket, prefix, and canonical project IDs, while each
device receives its own `device_id`, local project roots, OS-keyring credential
entry, and Reinstate home.

> **Current scope:** `config.toml` configures Phase 1 session sync only.
> Commands for portable MCP servers, skills, hooks, plugins, marketplaces, and
> cross-harness settings are roadmap work and do not exist in Reinstate.

## Prerequisites

- Reinstate `v0.4.0`, verified with `rein version --json`.
- A private S3-compatible bucket, service endpoint, region, access-key ID, and
  secret access key.
- One stable project ID and the repository's absolute path on this device.
- A supported Claude Code or Codex version if the device will push or restore
  sessions.
- The first device's exact `profile_id` when joining an existing profile.

Do not use an agent account token as an object-storage credential. Reinstate
does not need Anthropic or OpenAI account credentials.

## Reinstate home and files

The default home is `~/.reinstate` on macOS and WSL2, and
`%USERPROFILE%\.reinstate` on native Windows. An absolute
`REINSTATE_HOME` value selects a different home for deliberate isolation or
testing.

| Path | Current contents |
| --- | --- |
| `config.toml` | Schema v1 profile, storage coordinates, credential reference, enabled adapters, and project mappings |
| `state.json` | Schema v1 local and remote session revisions and last manifest revision |
| `backups/` | Timestamped pre-replacement config, state, and session backups |
| `conflicts/` | Unresolved conflict metadata and resolution audit records |
| `cache/`, `locks/`, `logs/` | Private runtime data |

Configuration and state are written atomically with owner-only permissions.
Secret values are invalid configuration fields.

## Configure the first device

Choose a canonical project ID that remains stable across devices. A repository
identity such as `github.com/acme/app` is clearer than a device-specific path:

```sh
rein init \
  --project github.com/acme/app=/absolute/path/to/app
```

Enter the S3/R2 service endpoint, bucket, and storage credentials only in the
interactive prompts. Keep the bucket name out of the endpoint URL; the bucket
has its own field. Interactive setup probes storage before it writes config,
stores storage credentials in the native OS keyring, initializes
`config.toml` and `state.json`, and prints the new `profile_id`.

Save that UUID. It is not a secret, but it is the identity that later devices
must join. The default object prefix is `profiles/<profile_id>` unless you
explicitly configure another prefix.

## Configure an additional device

Reuse the profile and project identities, but map the project to the
destination's actual absolute path:

```sh
rein init \
  --profile-id DEVICE_A_PROFILE_UUID \
  --project github.com/acme/app=/different/absolute/path
```

Use the same service endpoint, region, bucket, prefix, and storage credential
scope. Reinstate requires the existing encrypted `manifest.age` to be present and
readable before saving an additional device's config. It records
`remote_profile_required = true`, so later `status`, `diff`, `pull`, and push
operations fail if the established manifest disappears.

Native Windows and WSL2 are different devices. Give each its own home and
mapping even when both point to worktrees on the same physical computer.

## Project mapping contract

The same canonical ID must identify the same repository on every device:

```text
github.com/acme/app = /Users/me/code/app
github.com/acme/app = C:\Users\me\code\app
github.com/acme/app = /mnt/c/Users/me/code/app
```

Only one of those local roots belongs in a given device's config. During
export, adapters replace recognized structural roots with
`${REPO:<id>}`. During restore, the destination expands that token through its
own mapping. Reinstate does not rewrite arbitrary path-like prose, prompts,
tool output, or unknown fields.

## Reinitialize an existing home

Reinstate refuses ordinary `init` when `config.toml` or `state.json` already exists
and exits with safety code `7`. Review the current home before intentionally
running:

```sh
rein init --force
```

The forced path backs up the existing config and state together beneath one
timestamped `backups/` directory before replacing them. It is not a merge, and
it does not recover a lost passphrase or remote profile.

## Validate the configuration

```sh
rein setup check --json
rein doctor --self-test
rein list --agent all --json
rein status --json
```

`status` reads and decrypts the remote manifest, so wait for the hidden
passphrase prompt. The passphrase is not stored. `doctor --self-test` uses
synthetic data and does not read real session content.

## Expected evidence

- `init` reports `config.toml + state.json`, a UUID `profile_id`, and a
  `credential_ref` without printing the credential.
- The storage probe succeeds and cleans up its temporary probe object.
- `setup check` reports the device and installed adapter states truthfully.
- On an additional device, `status` returns the existing remote revision and
  session keys rather than an empty replacement profile.
- A scoped push or pull dry-run expands the canonical project ID to the correct
  local destination.

## Failure paths

- For a missing remote profile on an additional device, follow
  [remote-manifest troubleshooting](/docs/troubleshooting#why-does-reinstate-report-a-remote-profile-manifest-is-missing).
- For a passphrase failure, follow
  [passphrase troubleshooting](/docs/troubleshooting#why-does-passphrase-verification-fail-on-a-second-device).
- For a Claude session restored beneath the wrong project key, use the
  [native-resume diagnostic](/docs/troubleshooting#why-does-claude---resume-not-see-a-pulled-session).
- Do not create an empty `manifest.age`, invent a new profile ID, or use
  `--force` merely to silence a coordinate mismatch.

## Security boundaries

`config.toml` contains endpoints, bucket names, prefixes, IDs, local paths, and
a credential reference; treat paths and infrastructure names as private
metadata even though they are not secret keys. Storage keys belong in the OS
keyring. The explicit non-interactive provider may read
`REINSTATE_S3_ACCESS_KEY_ID` and `REINSTATE_S3_SECRET_ACCESS_KEY` without
persisting them, but they must never be committed, logged, or pasted into chat.

Interactive `status`, `push`, and `pull` read the encryption passphrase through
a hidden prompt. Automation must use a pre-opened descriptor through
`REINSTATE_PASSPHRASE_FD`; ordinary passphrase environment values and CLI
flags are rejected.

## Related pages

- [Choose and configure object storage](/docs/storage)
- [Push one selected session](/docs/sync-a-session)
- [Restore one selected session](/docs/restore-a-session)
- [Understand current limitations](/docs/limitations)
- [Universal agent configuration roadmap](/docs/universal-configuration)
