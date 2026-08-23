# Reinstate Hop: sign-in, devices, and the locker

Reinstate Hop is the paid hosted tier: a locker (a storage bucket provisioned
for exactly one account, holding only ciphertext) plus a console. Every client
capability stays in the free CLI; Hop gates storage and the console only. This
page covers what has landed in the client: passwordless sign-in, device
tokens, and syncing to the locker. Pairing and the daemon follow.

## Commands

```text
rein login [--email ADDRESS] [--no-browser] [--json]
rein whoami [--json]
rein init --hop [--project ID=PATH]... [--force]
rein hop status [--json]
```

## From sign-in to the first push

```bash
rein login                 # GitHub in the browser, or --email you@example.com
rein init --hop            # profile for the locker; provisions it
rein account init          # root key on this device; recovery code shown once
rein push --all            # first push: credentials minted, ciphertext lands
rein hop status            # bucket, location, usage, limits
```

A second machine runs `rein login`, `rein init --hop`, and
`rein account recover` with the recovery code, then `rein pull`.

`rein login` signs this device in. With no flag it starts a GitHub sign-in:
the CLI prints a URL, opens it in your browser, and waits. With
`--email you@example.com` the control plane sends a one-time link to that
address instead; open it on any device to approve this one. Either way, no
password exists or is ever asked for.

On approval the control plane enrols this computer as a **device** under your
**account** and issues a device token. The token is stored in the OS keyring
(macOS Keychain, Windows Credential Manager, or the supported Linux
provider), never in a file under `~/.reinstate`. `rein whoami` presents it
and prints the account, the device, and the control plane it is bound to.

Exit codes follow the usual contract: `2` for a malformed address, `4`
(`auth_storage`) when the device is not signed in or its token was rejected,
`1` for an unreachable control plane or an expired sign-in.

## The locker

`rein init --hop` writes a profile whose storage is the account's locker
(`storage.type = "hop"`; see [configuration.md](configuration.md)). The
profile id is the account id and the device id the enrolled device id, so
every device that signs in to the same account shares one profile without
copying anything. No endpoint, bucket, region, or key is stored: on every
push and pull the client asks the control plane for the locker's
coordinates and for **credentials bound to exactly that bucket**, valid for
at most an hour, and then speaks the S3 API to the locker directly through
the same backend BYO storage uses. The control plane never sees an object.
A credential that expires or is refused in the middle of a push is replaced
by a fresh one and the push carries on; a BYO profile never consults the
control plane at all.

The locker is created on `rein init --hop` (or on the first push if a
profile was written by hand) with an opaque name and in the location the
first device asked for at sign-in. `rein login` picks that hint from
`REINSTATE_HOP_LOCATION` if set, else from the machine's time zone: `apac`
(the default, and what India resolves to), `eeur`, `weur`, `enam`, `wnam`,
or `oc`. Later devices' hints do not move an existing locker.

After the first completed push the client tells the control plane once, so
the `first_push` event is counted from a push that finished rather than
guessed from bucket size.

### Limits and refusals

| Plan | Storage | Devices | Credential mints per hour |
| --- | --- | --- | --- |
| Hop | 5 GB | 5 | 60 |
| Hop Plus | 25 GB | 10 | 120 |

The control plane enforces these when it mints credentials, so a push on an
account over a limit fails before touching the locker and says which limit
and what to do (`rein hop status` shows usage against every limit). Exit
codes: `4` (`auth_storage`) when the device is not signed in, when its
token was rejected, and for every quota refusal; `1` when the control
plane could not reach the storage provider (retry).

## Choosing the control plane

The production control plane is `https://hop.reinstate.dev`. Override it for
staging or a local build with, in order of precedence:

1. `REINSTATE_HOP_URL=http://127.0.0.1:8080`
2. `[hop] url = "..."` in `config.toml` (sign-in works before `rein init`; the
   section is optional)

The URL a device signed in against travels with its token, so `rein whoami`
always asks the control plane that issued the token.

## Protocol

The client is open and the protocol is public; the control plane's source is
private. The control plane never receives a root key, a recovery code, a
passphrase, or session content. Sign-in is a device-authorization style flow:

| Step | Request | Answer |
| ---- | ------- | ------ |
| 1 | `POST /v1/login/sessions` `{method: "github"\|"email", email?, device: {name, platform, location_hint?}}` | `201 {session_id, poll_secret, verification_url?, expires_at, interval_seconds}`. For `email` the link is mailed, not returned. |
| 2 | Browser opens `verification_url` (GitHub OAuth) or the emailed link | The emailed link shows an "Approve this device?" form on `GET`; enrolment happens only on its `POST`, so mail link scanners that prefetch the link neither sign anyone in nor burn it. The control plane creates the account on first sight and enrols the device in one transaction. Each link works once and expires with the session. |
| 3 | `POST /v1/login/sessions/{id}/poll` `{poll_secret}` | `200 {status: "pending"}` until approved; then exactly once `200 {status: "approved", device_token, account, device}`; afterwards `410 {status: "consumed"}`. Past `expires_at`: `410 {status: "expired"}`. Wrong secret: `404`. |
| 4 | `GET /v1/whoami` with `Authorization: Bearer <device_token>` | `200 {account, device}`; unknown or revoked token: `401`. |
| 5 | `POST /v1/locker` (bearer) | Idempotent provisioning: `200 {endpoint, bucket, region, prefix, location_hint, plan, created_at, first_push_at?, devices, usage: {bytes, objects, observed_at}, quota: {storage_bytes, devices, mints_per_hour}}`. `GET /v1/locker` returns the same without provisioning (`404 {code: "no_locker"}` before the first `POST`). |
| 6 | `POST /v1/locker/credentials` (bearer) | `200 {access_key_id, secret_access_key, session_token, expires_at, endpoint, bucket, region}`, valid for at most an hour and scoped to the bucket. Refusals carry a `code`: `quota_storage` (403), `quota_devices` (403), `quota_push_rate` (429), `no_locker` (404), `storage_unavailable` (502). |
| 7 | `POST /v1/locker/first-push` (bearer) | `200 {first, first_push_at}`; records the first push once. |

The CLI reads the locker with `GET /v1/locker` on every hosted command and
only calls `POST /v1/locker` when the answer is `no_locker` (normally once,
from `rein init --hop`).

The device quota is enforced when credentials are minted, not when a device
enrols: a sixth device on a five-device plan can still sign in, after which
no device on the account can mint until the count is back under the plan.
Until device revocation ships there is no self-serve way out of that
state, so do not enrol more devices than the plan allows.

Device tokens are 256-bit random values prefixed `hop_`. The control plane
stores only a hash, bound to one device record (name, platform, location
hint, created, last seen). The CLI sends no telemetry; the control plane
records `sign_up`, `device_enrolled`, `locker_provisioned`, and `first_push`
events as its only product metrics.

## What this does not do yet

- Pair devices through a code (a second device enrols with the recovery
  code today).
- Revoke a device or sign out (also the only recovery from an account over
  its device quota).
- Run a daemon.
