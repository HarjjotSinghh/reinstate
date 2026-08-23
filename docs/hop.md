# Reinstate Hop: sign-in, devices, and the locker

Reinstate Hop is the paid hosted tier: a locker (a storage bucket provisioned
for exactly one account, holding only ciphertext) plus a console. Every client
capability stays in the free CLI; Hop gates storage and the console only. This
page covers what has landed in the client: passwordless sign-in, device
tokens, syncing to the locker, and device approval (pairing). The daemon
follows.

## Commands

```text
rein login [--email ADDRESS] [--no-browser] [--json]
rein whoami [--json]
rein init --hop [--project ID=PATH]... [--force]
rein hop status [--json]
rein account join
rein devices [--json]
rein devices approve [--request ID]
```

## Your first push

Four commands take a machine from nothing to ciphertext in the locker; the
whole run is meant to fit in under two minutes, most of it the browser tab.

```bash
rein login                 # GitHub in the browser, or --email you@example.com
rein init --hop            # profile for the locker; provisions it
rein account init          # root key on this device; recovery code shown once
rein push --all            # first push: credentials minted, ciphertext lands
rein hop status            # bucket, location, usage, limits, first push time
```

What each step leaves behind:

- `rein login` stores a **device token** in the OS keyring and nothing else.
  The control plane now knows this device; the locker does not exist yet.
- `rein init --hop` writes the profile (the account is the profile, this
  device is the device) and provisions the locker. No endpoint, bucket, or
  key lands in `config.toml`.
- `rein account init` generates the **root key** on this device, writes the
  **keyring** to the locker with the first minted credential, and shows the
  **recovery code** once. Write it down: the operator cannot recover the
  locker for you.
- `rein push --all` encrypts every session Reinstate can find (Claude Code,
  Codex, OpenCode, and the other synced agents) under the root key and
  uploads it. The first completed push is reported to the control plane
  exactly once, so `rein hop status` shows a first-push time from then on; a
  later push that has nothing to send reports nothing.

### The same machine, wiped

A reinstalled machine (or a new laptop standing in for one) gets its
sessions back with the recovery code and nothing else:

```bash
rein login                 # a new device token
rein init --hop            # the profile again; the locker already exists
rein account recover       # enter the recovery code; this device joins the keyring
rein pull --all            # sessions are decrypted into each agent's own layout
rein resume claude:<id>    # verified resume, as before
```

Install and run each agent once before `rein pull`: Reinstate restores into
the vendor's own layout and never invents it. A pull on a machine where an
agent is still missing names that agent and stops; the sessions restored
before it are kept and remembered, so the next pull carries on rather than
reporting a conflict. A `rein push` or `rein pull` on a device that has not
enrolled yet says so, and tells you whether this is the first device
(`rein account init`) or an additional one (`rein account recover`, or
`rein account join` approved from an enrolled device).

### Adding a second machine instead

A second machine runs `rein login`, `rein init --hop`, and
`rein account join`; it shows a short code, you enter that code on the first
machine with `rein devices approve`, and the second machine pulls. When no
enrolled device is at hand, `rein account recover` with the recovery code is
the fallback.

The whole journey is exercised end to end by `TestHopFirstPushJourney`
(`internal/cli`, against the in-process fake control plane and locker) and,
with `-tags hopacceptance`, by `TestHopFirstPushJourneyStaging` against a
real control plane named by `HOP_STAGING_URL`. That suite signs in twice
(day one, and again after the wipe, as a new device), either with a real
`rein login --email` for `HOP_LOGIN_EMAIL` whose links you approve within
`HOP_LOGIN_TIMEOUT` (default 5m), or with two pre-issued tokens of one
account in `HOP_DEVICE_TOKEN` and `HOP_DEVICE_TOKEN_2`; without those it
skips. A run of both modes against `hopd` and the lab locker is recorded in
`docs/testing/results/2026-08-24-first-push-acceptance-lab.md`.

## Adding a device (pairing)

Nothing is typed on the new device except the sign-in. On it:

```bash
rein login
rein init --hop
rein account join          # shows a code such as P5EZ-6PN0-WDB5-T0J0 and waits
```

On any machine that is already enrolled:

```bash
rein devices               # lists devices and the pending request
rein devices approve       # asks for the code shown on the new device
```

The enrolled machine checks the code against the request, appends a wrap of
the root key for the new device to the keyring (compare-and-swap, so two
approvals at once both land), and relays the root key sealed so that only
the holder of the code can open it. The new machine opens it, confirms that
the keyring's wrap for itself opens to the same root key of the same
generation, and from then on reads and writes the locker.

What the control plane sees: the new device's public key, a random salt, a
binding value, and later a ciphertext; never the code. The code is 60
random bits plus a checksum (`XXXX-XXXX-XXXX-XXXX`, Crockford base32; O/I/L
are accepted for 0/1); the wrapping key is argon2id over the code and salt,
so guessing the code offline against the relayed material costs one
memory-hard derivation per candidate across 2^60 candidates. The relay
expires after ten minutes, hands the ciphertext out once, and refuses
after 600 polls. A wrong code on the approving side fails closed with
nothing written anywhere; a typo is caught by the checksum first. A request
that has already expired when the code is entered (the prompt can sit open
while you walk to the other machine) is refused before anything is
written; if the control plane refuses the relay after the wrap was
appended (expired or decided meanwhile), the approving device removes that
wrap again, so the new machine's next `rein account join` is a fresh
request rather than a silent enrolment without an approval behind it. The
joining machine never treats a keyring that already lists it as proof of
enrolment: its public key is published in the request, so a control plane
that also holds the bucket could write a keyring wrapping a root key of its
own choosing for that key. `rein account join` therefore always opens a
fresh request and waits for a code to be typed on an enrolled device, and
only the root key received through that approval (and matched against the
keyring) is ever used. If a device's local account record is lost, run
`rein account join` again and approve it again, or `rein account recover`
with the recovery code. The code
can be supplied to automation through `REINSTATE_PAIRING_CODE_FD` (a
pre-opened descriptor, like `REINSTATE_PASSPHRASE_FD`); it is never a flag
or a plain environment value. The full protocol and threat argument are in
the control plane's `docs/hop.md`, "Pairing".

`rein devices` shows every device enrolled under the account, whether the
keyring holds a root-key wrap for it, and any pending pairing request.

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

`rein devices approve` compares a pending request's expiry with this
machine's clock before it writes anything; a clock that is far wrong can
therefore refuse a request that is in fact still open (`expired at ...`,
exit `2`). Fix the clock, then run `rein account join` again on the new
device for a fresh request.

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
