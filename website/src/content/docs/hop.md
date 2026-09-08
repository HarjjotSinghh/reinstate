---
title: "Reinstate Hop: Sign-In, Devices, and the Locker"
navTitle: "Reinstate Hop"
description: "See how Reinstate Hop's passwordless sign-in, device pairing, encrypted locker, verification, and daemon work in the v0.6.0-rc.8 candidate."
order: 17
author: "Harjot Singh Rana"
status: current
schemaType: tech-article
version: "v0.6.0-rc.8"
updatedAt: 2026-09-05
tags: ["hop", "hosted-tier", "keyring", "device-pairing", "daemon"]
targetQuery: "what is reinstate hop"
searchIntent: "how-to"
draft: false
noindex: false
---

Reinstate Hop is the optional hosted tier: a **locker** (a storage bucket
provisioned for exactly one account) plus a console. Every capability lives
in the CLI; Hop only gates storage and the console.

> **Status.** The hosted control plane this client talks to by default,
> `https://hop.reinstate.dev`, is not open yet. Nothing here is behind a
> build tag or a flag — the client ships anyway, and the protocol and
> journeys below are public and testable today against a control plane you
> run yourself or point the client at (see
> [Choosing the control plane](#choosing-the-control-plane)). No price,
> trial, or sign-up is attached to any of this, and Hop ships in the
> `v0.6.0-rc.8` release candidate, not yet a stable claim — stable remains
> `v0.5.1`, where Hop does not exist.

Every session object Reinstate writes to the locker is ciphertext, with one
documented exception: `keyring.v1.json` holds no usable key, but it does give
up the account's profile id, every enrolled device's id and public key, and
one entry per key generation with the time it started. The full object
format and what it is worth to an observer are in the
[object format](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/hop/object-format.md)
and [threat model](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/hop/threat-model.md)
references.

## Commands

```text
rein login [--email ADDRESS] [--no-browser] [--json]
rein whoami [--json]
rein init --hop [--project ID=PATH]... [--force]
rein hop status [--json]
rein hop credentials [--json] [--export]
rein account init [--json]
rein account join
rein account recover [--json]
rein account status [--json]
rein devices [--json]
rein devices approve [--request ID]
rein devices revoke <device-id|name>
rein sync verify [--json] [--post=false]
rein sync migrate --to byo [--endpoint URL --bucket NAME] [--switch] [--forget-hop]
rein daemon run [--pull-every DURATION] [--debounce DURATION]
rein daemon install|start|stop|uninstall
rein daemon status [--json]
```

## Your first push

Four commands take a machine from nothing to ciphertext in the locker:

```bash
rein login                 # GitHub in the browser, or --email you@example.com
rein init --hop             # profile for the locker; provisions it
rein account init           # root key on this device; recovery code shown once
rein push --agent claude --session SESSION_ID   # or --all: credentials minted, ciphertext lands
rein hop status              # bucket, location, usage, limits, first push time
rein sync verify              # the verification report, any time
```

`rein login` stores a device token in the OS keyring only; the control plane
now knows this device, but the locker does not exist yet. `rein init --hop`
provisions the locker without writing an endpoint, bucket, or key to
`config.toml`. `rein account init` generates a **root key** on this device,
writes the **keyring** with the first minted credential, and shows the
**recovery code** exactly once — write it down, because the operator cannot
recover the locker for you. `rein push` (optionally with `--all`, so neither
Reinstate nor a coding agent selects every discovered session on your
behalf by default) then encrypts the selected sessions under the root key
and uploads them.

### Recovering a wiped machine

```bash
rein login                 # a new device token
rein init --hop              # the profile again; the locker already exists
rein account recover        # enter the recovery code; this device joins the keyring
rein pull --agent claude --session SESSION_ID   # or --all: sessions decrypt into each agent's own layout
rein resume claude:<id>       # verified resume, as before
```

Install and run each agent once before `rein pull`; Reinstate restores into
the vendor's own layout and never invents it. A second machine can instead
run `rein login`, `rein init --hop`, and `rein account join`, which shows a
short code approved from an enrolled device with `rein devices approve`.

## Adding and revoking devices

Pairing needs nothing typed on the new device except the sign-in:

```bash
rein login && rein init --hop && rein account join   # shows a short code, waits
```

On an already-enrolled device, `rein devices approve` takes that code,
appends the new device's wrap of the root key to the keyring, and relays the
root key sealed so only the code's holder can open it. A wrong code fails
closed with nothing written; an expired request is refused before anything
is written. The control plane never sees the code itself, only protocol
metadata; the full derivation (Argon2id plus HKDF-bound AEAD, versioned for
upgrades) is in
[docs/hop.md, "Adding a device"](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/hop.md#adding-a-device-pairing).

Revoking a lost or retired device is one command from any other enrolled
device:

```bash
rein devices                  # find its id or name
rein devices revoke desktop    # asks for the recovery code
```

Revocation starts a new **key generation**: the revoking device wraps a
fresh root key for every remaining device and under the recovery code, then
tells the control plane to refuse the revoked device's token from then on.
Earlier objects stay readable by every generation that can unwrap them; the
revoked device keeps what it already pulled but cannot open anything sealed
under the new generation. The Console can *request* a revocation, but only
an enrolled device holding the recovery code can *perform* one — the control
plane never holds the root key or the recovery code.

## The locker

`rein init --hop` writes a profile whose storage is the account's locker.
The profile id is the account id and the device id the enrolled device id,
so every device signed in to the same account shares one profile. No
endpoint, bucket, region, or key is stored locally: on every push and pull
the client asks the control plane for credentials bound to exactly that
bucket, valid for at most an hour, then speaks the S3 API to the locker
directly — the control plane never sees an object. A BYO profile never
consults the control plane at all.

The locker is created in the location the first device asked for at
sign-in (`REINSTATE_HOP_LOCATION`, else the machine's time zone; `apac` by
default and for India). Later devices' hints do not move an existing
locker.

### Limits

What the control plane is built to enforce; which of these the hosted
service will actually offer, and on what terms, is not published — this
documents the protocol, not an offer.

| Plan | Storage | Devices | Credential mints per hour |
| ---- | ------- | ------- | -------------------------- |
| Hop | 5 GB | 5 | 60 |
| Hop Plus | 25 GB | 10 | 120 |

## Verifying the claim (`rein sync verify`)

The claim: every session object in the locker is ciphertext sealed by your
devices, your devices can open it, and your account's credentials reach
your locker and nothing else. `rein sync verify` checks this in four steps
and prints a report written for a non-expert, each step scored PASS, FAIL,
or NOT APPLICABLE:

1. **List the locker** with this device's push credentials.
2. **Fetch an object and check it is ciphertext** — the index and the most
   recently updated snapshot.
3. **Decrypt it locally** with the key held on this device and check the
   payload checksum. Nothing leaves the machine; on a Hop locker the root
   key is never exported, because a command that wrote it out would expose
   every object the account has ever written.
4. **Prove isolation**: the same credentials must be refused, as access
   denied, against a reference locker the control plane names. The step
   fails only on something that contradicts the claim (the reference
   answered, a redirect was followed, a plaintext endpoint was used); every
   other outcome — no reference advertised, the control plane unreachable,
   a credential rotated mid-check — is reported "not applicable," not a
   pass.

The report ends `OUTCOME: PASS`, `FAIL`, `NOT VERIFIED` (no answer from the
control plane or storage endpoint), or `NOT YET VERIFIABLE` (nothing pushed
yet). Exit code `7` on any failed step, `0` when every step passed or did
not apply, `1` when a required endpoint could not be reached at all. On a
Hop profile, only the step **results** — never object contents, session
ids, or project paths — are posted to the control plane for the account
console; `--post=false` keeps them local. BYO storage runs steps 1–3 and
reports step 4 as not applicable. The first successful push from a new
device runs the same checks automatically.

## The daemon

`rein daemon` is a resident per-device process that keeps a device's
sessions synced without anyone running `push` and `pull` by hand, and it
behaves identically on BYO storage and on Hop — no telemetry, nothing sent
that `push` and `pull` do not already send.

```text
rein daemon run [--pull-every DUR] [--debounce DUR] [--poll] [--verbose]
rein daemon install       # register at login, and start it now
rein daemon start|stop    # control the registered daemon
rein daemon uninstall     # stop it and remove the login registration
rein daemon status [--json]
```

`rein daemon install` registers with the platform's own supervisor: a
launchd user agent on macOS, a systemd `--user` unit on Linux, and a Task
Scheduler task with a logon trigger on Windows. The loop watches every
detected agent's session directory and pushes after a change (debounced,
coalesced, and pushed at least every 30s regardless), pulls on a schedule
(default every 30s), and pulls once more before a resume when its last pull
is older than 15s. On Hop only, it also polls for pending device-pairing
requests and surfaces each as an OS notification and a status-file entry;
approval itself stays interactive and the daemon never sees the pairing
code. `rein daemon install` needs the root-key model (`rein account init`,
which also works on BYO storage); a passphrase-model home can run
`rein daemon run` under a supervisor that supplies
`REINSTATE_PASSPHRASE_FD` instead.

## Choosing the control plane

The production control plane is `https://hop.reinstate.dev`. Point the
client at a staging or self-hosted instance instead with, in order of
precedence:

1. `REINSTATE_HOP_URL=http://127.0.0.1:8080`
2. `[hop] url = "..."` in `config.toml`

When the configured control plane cannot be reached at all — no DNS answer,
connection refused, a failed TLS handshake — `rein login` and `rein whoami`
print one sentence naming the URL and the cause and pointing at this page,
instead of a raw network error. A control plane that *answers* (a rejected
token, a quota refusal) is reported the way it always was.

## Protocol

The client is open and the protocol is public; the control plane's source is
private. The control plane never receives a root key, a recovery code, a
passphrase, or session content — sign-in is a device-authorization style
flow (`POST /v1/login/sessions`, poll, claim), and every other Hop journey
above follows the same rule: what crosses the wire is metadata, tokens, and
ciphertext, never a key. The full request and response shapes are in
[docs/hop.md, "Protocol"](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/hop.md#protocol).

## Leaving Hop

Leaving is one command to a bucket you own, available at any time,
including a read-only period the control plane may apply to a lapsed
account:

```bash
export REINSTATE_S3_ACCESS_KEY_ID=... REINSTATE_S3_SECRET_ACCESS_KEY=...
rein sync migrate --to byo --endpoint https://<account>.r2.cloudflarestorage.com --bucket my-sessions
```

It reads every snapshot and the manifest from the locker, opens them with
the root key held on this device, re-seals them under a new passphrase, and
writes them to the bucket under a fresh profile — nothing derived from the
root key reaches the destination. The locker is only read, never deleted or
emptied, so the command works on a lapsed, read-only account. An
interrupted run resumes from where it left off without writing anything
twice. Afterwards it offers to switch this device to the bucket and to
forget the device's Hop sign-in; both are optional and reversible. Other
devices join with `rein init --profile-id <printed id>` and the passphrase.

## What this does not do yet

- Sign out a device from itself (revoke it from another device instead).
- Billing, pricing, a trial, or a self-service sign-up: this page documents
  the protocol, not an offer.
- Deploying the hosted control plane to production — the client and
  protocol ship ahead of that, per
  [ADR 0005](https://github.com/HarjjotSinghh/reinstate/blob/main/docs/adr/0005-v0.6.0-scope-and-windows-first-acceptance.md).

## Related pages

- [Reinstate security and encryption model](/docs/security-model)
- [Configure storage for encrypted Reinstate sessions](/docs/storage)
- [Reinstate CLI reference](/docs/cli-reference)
- [Reinstate roadmap](/roadmap)
- [Changelog](/changelog)
