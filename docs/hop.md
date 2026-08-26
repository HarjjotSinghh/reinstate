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
rein sync verify [--json] [--post=false]
rein sync migrate --to byo [--endpoint URL --bucket NAME] [--switch] [--forget-hop]
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
rein sync verify           # the verification report, any time
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
guessed from bucket size. The same first push is followed, once per device,
by the verification below.

## Verifying the claim (`rein sync verify`)

The claim is that the locker holds only ciphertext sealed by your devices,
that your devices can open it, and that your account's credentials reach
your locker and nothing else. `rein sync verify` checks all of it and
prints a **verification report** written for a non-expert: each step says
what was done, what was seen, and PASS or FAIL, and every step can be
repeated by hand with an S3 client ([object format](hop/object-format.md),
"Reproducing the checks by hand").

1. **List the locker** with the credentials this device pushes with,
   following every listing page; shows `manifest.age`, `keyring.v1.json`,
   and the snapshots by their opaque ids, and records (locally) the access
   key id the listing was signed with.
2. **Fetch an object and check it is ciphertext**: the index, and the one
   snapshot the index records as updated last (snapshot ids are random, so
   the index is the only thing that knows which that is). The bytes begin
   with the age v1 header, the recipient type is named (X25519 for Hop,
   scrypt for BYO), and none of the plaintext field names occur anywhere in
   the body.
3. **Decrypt it locally** with the key held on this device and show what it
   contains. The index's revision, sessions per agent and entries, and a
   snapshot's agent, session and payload size are printed as local detail
   lines only; the step result that can be posted says just that the index
   and a snapshot envelope decrypted and the payload checksum matches.
   Nothing leaves the machine.
4. **Prove isolation**: the control plane names its **reference locker**, a
   bucket the operator owns holding one probe object; the same credentials
   are used to list it and read the probe, and both must be refused as
   **access denied** (R2 answers `AccessDenied`).

   The verdict is pinned to the answer rather than to the endpoint the
   control plane named, because both endpoint strings come from the
   control plane and neither says where the request landed. The probe
   client **refuses to follow a redirect**, so this account's credential
   is only ever sent to the host step 1 listed; the refusal must come back
   from that host, and it must be an S3 error naming its code. A **403
   with no S3 error body** is something any web server answers, so it
   shows nothing and the step is reported **not applicable** rather than
   passed. So is a run where step 1 did not pass — no locker was shown to
   accept these credentials, and a refusal of a credential nothing accepts
   is what every host gives — and a run where the locker's own storage
   endpoint is not known on this device, which leaves nothing to pin the
   reference against.

   A refusal that says the credential itself is bad (`InvalidAccessKeyId`,
   `SignatureDoesNotMatch`, `ExpiredToken`, `InvalidToken`) is what every
   bucket answers a dead credential, so it proves nothing about scope and
   **fails** the step, as does a credential that changed between step 1
   and step 4, a redirect offered by the reference endpoint, and a
   reference locker at a **different storage endpoint** than the one step
   1 listed (scheme case and a trailing slash are the same endpoint; a
   different port is a different endpoint). The step's local detail names
   the access key id and the reference endpoint so the report shows the
   credential the locker accepted is the one the reference refused, and
   where.

The report ends with `OUTCOME: PASS` or `OUTCOME: FAIL`; exit code `4` on
any failed step. The outcome sentence claims only what the steps observed:
it calls ciphertext only the objects that were actually fetched — the
index, and the snapshot the index records as updated last (a manifest-only
locker verifies only the index; a snapshot chosen without the index, because
the index would not open here, is called "one snapshot" and not the newest)
— names what was judged by name only (the other snapshots, the keyring,
anything unrecognised), says nothing about which device sealed them, and
when step 4 is not applicable it says isolation was not checked instead of
asserting it. `--json` emits the report as data (see
`testdata/verify/byo-report.golden.json` under `internal/cli` for the
shape). On a Hop profile the **step results only** — never object contents,
session ids, or project paths — are posted to the control plane for the
account console; `--post=false` keeps them local. BYO storage runs steps
1–3 and reports step 4 as not applicable.

The first successful push from each new device runs the same checks and
posts the report once; a push's `--json` output then carries
`verification: {outcome, posted}`. A verification that cannot run or post
never fails the push; it is noted on stderr and retried after the next
push that uploads something. The [threat model](hop/threat-model.md) states what each step proves
and what the operator can and cannot see.

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

## The daemon

`rein daemon` is a resident per-device process that keeps a device's
sessions synced without anyone running `push` and `pull` by hand, and
surfaces devices waiting to join the account. It behaves identically on
BYO storage and on Hop, and it sends nothing that `push` and `pull` do not
already send — no telemetry (ADR 0008). Its only outputs are the locker (or
your bucket), a local status file, a rotating log, and OS notifications on
that one machine.

```text
rein daemon run [--pull-every DUR] [--debounce DUR] [--poll] [--verbose]
rein daemon install       # register it to start at login, and start it now
rein daemon start|stop    # control the registered daemon
rein daemon uninstall     # stop it and remove the login registration
rein daemon status [--json]
```

`rein daemon install` registers the daemon with the platform's own
supervisor so it starts at login: a **launchd** user agent on macOS
(`~/Library/LaunchAgents`), a **systemd `--user`** unit on Linux
(`~/.config/systemd/user`, `WantedBy=default.target`), and a **Task
Scheduler** task with a logon trigger on Windows (`schtasks`,
`InteractiveToken`, per-user principal). `rein daemon run` is the
foreground loop that registration runs; run it yourself under any
supervisor you prefer, or just to watch it work.

What the loop does:

- **Watches** every detected agent's session directory (fsnotify, with a
  polling fallback — `--poll` forces polling for network homes and some
  containers) and **pushes** after a session file changes. Changes are
  debounced (`--debounce`, default 3s) and coalesced, so a burst of writes
  is one push; a session that never stops changing is still pushed at least
  every 30s. A push that hits a conflict records it and waits — the daemon
  never resolves conflicts or overwrites divergence; `rein conflicts` does.
- **Pulls** on a schedule (`--pull-every`, default 30s) so a session
  edited on another device appears here within a minute without any
  command; `pull --all` skips whatever this device already synced, so an
  idle pull is one manifest read. **Before a resume** — `rein resume`,
  `rein fork`, or the switcher — the CLI pulls once more when the daemon is
  running on this device and its last successful pull is older than 15s,
  so what launches is the latest snapshot rather than the one from the
  last tick. A pull that fails there is reported on stderr and never
  blocks the resume; while the daemon itself is mid-pull the resume simply
  uses its result. Without a running daemon nothing is pulled implicitly:
  the shell commands stay explicit.
- **Surfaces pending device approvals** (Hop only): it polls the control
  plane, and when a device asks to join it shows an OS notification, writes
  the request to the status file, and the next `rein` command on that
  machine prints one line — `device "X" wants to join your account; run
  rein devices approve`. Approval itself stays interactive: it needs the
  code typed on this device (`rein devices approve`), which the daemon
  never sees.

The daemon runs one instance per home (an advisory lock file), backs off
exponentially on errors (5s doubling to 10m; a session that keeps changing
during an outage waits for the backoff rather than retrying on every
write), rotates its log at 1 MB (three kept), and never crashes on a vendor
store caught mid-write. A vendor write caught halfway through a line can
still be pushed as a torn snapshot — it shows up as one more snapshot in
the history — and the next change pushes the whole line. A session that
diverged on this device (edited locally after another device moved its
head) is recorded once under `rein conflicts`, not once per tick: the
record is keyed by the divergence, and a scheduled `pull --all` keeps
restoring every other session while that one waits for
`rein conflicts resolve`. The plist, unit, or task definition is written
owner-only, and `rein daemon install` refuses an `--env` whose name looks
like a credential (`SECRET`, `TOKEN`, `PASSPHRASE`, …): secrets belong in
the OS keyring or the backend's own credential store, never in a plist or
unit file. `rein daemon install` needs the root-key model
(`rein account init`, which works on BYO storage too) so the daemon can run
without a passphrase prompt. A passphrase-model home can still run
`rein daemon run` under a supervisor that supplies
`REINSTATE_PASSPHRASE_FD`: the descriptor is read once, when the daemon
starts, and that passphrase serves every push and pull for the daemon's
lifetime (the pull-before-resume hook stays off on such a home, since a
shell command cannot read the same descriptor).

On Windows, an account that is a member of Administrators runs under a
UAC-filtered token in an ordinary shell, and `schtasks /Create /XML` refuses
that token with "Access is denied"; run `rein daemon install` from an
elevated shell on such an account. Standard user accounts install from any
shell.

`rein daemon status` reads the status file the daemon writes after every
action: whether the daemon is registered and running, the last push and
pull, the watched roots, and — on Hop — the enrolled devices and any
pending approvals. The interactive switcher shows the same one-line
summary on its status line.

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
| 8 | `GET /v1/verify/reference` (bearer) | `200 {endpoint, bucket, region, key}`: the operator's reference locker and its probe object, for `rein sync verify` step 4. `404 {code: "no_reference"}` when the control plane has none (the step is reported as not applicable). |
| 9 | `POST /v1/verify-reports` (bearer) `{version: 1, generated_at, client_version, storage: "hop"\|"byo", outcome: "pass"\|"fail", steps: [{id, name, did, observed, status}]}` | `201 {id, received_at}`; stored per device for the console. Step results only; a body over 64 KB or with a verdict outside `pass`/`fail`/`not-applicable` is refused (400 for a bad field, 413 for an oversized body). |

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
records `sign_up`, `device_enrolled`, `locker_provisioned`, `first_push`,
`pairing_requested`, `pairing_approved`, `trial_started`, and
`verify_reported` events as its only product metrics.

## Leaving Hop

Leaving is one command to your own bucket, available at any time, including
the read-only period after a trial or subscription lapses:

```bash
export REINSTATE_S3_ACCESS_KEY_ID=... REINSTATE_S3_SECRET_ACCESS_KEY=...
rein sync migrate --to byo --endpoint https://<account>.r2.cloudflarestorage.com --bucket my-sessions
```

It asks for a new passphrase (twice; or `REINSTATE_PASSPHRASE_FD`), reads
every snapshot and the manifest from the locker, opens them with the root
key held on this device, re-seals them under the passphrase, writes them to
the bucket under a fresh profile, and reads each one back to compare
digests before the manifest is written. Nothing derived from the root key
reaches the bucket: no keyring, no device wrap, no X25519 recipient; every
object there opens with the passphrase alone. Snapshot ids are preserved, so
local state on every device keeps meaning.

The locker is only read. The command never deletes, empties, or rewrites
it, and it does not report a push; that is why it works on a lapsed,
read-only account. Deleting the locker is account deletion, a separate
step.

A run that is interrupted resumes when you run the same command again:
verified snapshots are skipped and nothing is written twice (see
`docs/cli-reference.md` for the record it keeps and the checks it makes).
Once verified, it offers to switch this device to the bucket (the Hop
config is backed up first) and to forget the device's sign-in; both are
optional, and both leave the locker and the account exactly as they were.
Both are reversible: copy `backups/<timestamp>-migrate-byo/config.toml`
and `state.json` back into the home to sync with the locker again, and
`rein login` again if the sign-in was forgotten.
Other devices follow with `rein init --profile-id <printed id>` and the
passphrase.

## What this does not do yet

- Pair devices through a code (a second device enrols with the recovery
  code today).
- Revoke a device or sign out (also the only recovery from an account over
  its device quota).
  its device quota). `rein sync migrate --to byo --forget-hop` drops this
  device's token locally but does not revoke it at the control plane.
- Run a daemon.
