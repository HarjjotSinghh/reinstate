# Threat model: what the operator can and cannot see

Reinstate Hop's claim is narrow and checkable: **the operator stores
ciphertext it cannot open, and an account's credentials reach that account's
locker and nothing else.** This page states the claim precisely, what it
assumes, what it does not cover, and how each line maps to a step of
`rein sync verify`. The [object format](object-format.md) is the byte-level
companion.

Vocabulary: the **operator** runs the control plane and owns the storage
account that holds every locker. The **control plane** signs accounts in,
enrols devices, provisions lockers, mints credentials, and stores
verification reports. A **verification report** is the output of
`rein sync verify`.

## Parties

| Party | Holds | Does not hold |
| --- | --- | --- |
| Device | the device's X25519 private key (OS keychain), the device token, the unwrapped root key while a command runs, the plaintext sessions | other devices' private keys |
| Person | the recovery code (shown once) | — |
| Operator / control plane | account identity (email or GitHub login), device records (including each device's coarse location hint), the locker's name and location, hourly credential mints, bucket usage, product events, verification reports, billing identity | the root key, any device private key, the recovery code, a passphrase, any plaintext |
| Storage provider (Cloudflare R2) | the ciphertext objects and `keyring.v1.json`, object sizes and times, request logs | any key |

## What the operator can see

Everything here is either needed to run the service or unavoidable in
object storage; none of it is content.

- **Billing identity**: the email address or GitHub login used to sign in,
  the plan, trial start, and what the payment processor reports.
- **Devices**: how many are enrolled, their self-reported name and
  platform (`laptop`, `darwin-arm64`), when each enrolled and was last
  seen, and a **coarse location hint** — one of `apac`, `eeur`, `weur`,
  `enam`, `wnam`, `oc`. The client derives it from the machine's time zone
  (or `REINSTATE_HOP_LOCATION`) and sends it with every sign-in; it is
  stored per device and, from the first device, per account, and it is
  what decides where the locker is provisioned. It is a continent-sized
  region, not a location, and there is no way to enrol without sending
  one — but it is sent, so it is listed here.
- **Locker shape**: the bucket's total size and object count (sampled by the
  provider, up to an hour stale), and from the bucket itself each object's
  size and last-modified time. That gives the number of snapshots and a
  rough push cadence and session size distribution.
- **Push cadence**: each hourly credential mint is counted (it is the
  push-rate quota), and the first completed push is reported once by the
  client. `first_push_at`, mint timestamps, and object timestamps together
  describe when the account pushes, not what.
- **Key topology**: `keyring.v1.json` is plaintext, and it carries more
  than counts. Visible to anyone with bucket access: the account's
  `profile_id`, every enrolled `device_id`, every device's X25519 **public**
  key, each device's enrolment time, and one entry per key generation with
  the time it started and its public root-key recipient. Each generation
  lists the devices enrolled in it, so a locker with more than one
  generation also shows which devices stopped being enrolled, and when. It
  carries no usable key. The identifiers name nothing a person is called,
  but they are stable, so an observer with bucket access can tell one
  account's locker from another's and count and follow its devices over
  time. "Bucket access" includes anyone holding a credential from
  `rein hop credentials`, which is why that command says so.
- **Verification reports**: the pass/fail per step a device posts,
  including object counts, object sizes, and the opaque object names the
  steps mention, plus the client version (`client_version`) that ran them.
  Access key ids and per-session detail stay local.
- **Product events**: `sign_up`, `device_enrolled`, `locker_provisioned`,
  `first_push`, `pairing_requested`, `pairing_approved`, `trial_started`,
  `verify_reported`, each with an account id, a device id where one applies,
  and a time. The client sends no other telemetry.

## What the operator cannot see

- **Session content**: transcripts, prompts, tool output, file names inside
  an export, project paths. Every snapshot is an age envelope sealed to the
  root key on the device before upload.
- **The index**: which agents, sessions, or projects an account syncs.
  `manifest.age` is sealed the same way; object names are random UUIDs.
- **Any key**: the root key exists in the locker only wrapped to device
  public keys and under the recovery code. The control plane never
  receives a root key, a device private key, a recovery code, or a
  passphrase.
- **A pairing code, or what it relays**: approving a device relays the
  root key sealed under a key derived from the code, which the control
  plane never sees. What it holds is the joining device's public key, a
  random salt, a binding value, and the ciphertext. Guessing the code
  offline against that material costs one argon2id derivation
  (t=3, 64 MiB, 4 lanes) per candidate across 2^60 candidates; the relay
  also expires after ten minutes, hands the ciphertext out once, and
  refuses after 600 polls. The bound is the argon2id cost times the
  keyspace, not the ten minutes: the material can be kept and attacked
  afterwards.
- **Another account's locker through this account's credentials**: a
  minted credential is scoped to exactly one bucket (the provider enforces
  it; the control plane never mints for any other).
- **Anything from the verification report beyond step results**: the
  upload form carries no session identifier, project path, agent name,
  session count, index revision, or plaintext — only object counts, opaque
  object names and sizes, and each step's verdict; the local output does
  (a unit test pins the exact uploaded text).

## Assumptions

1. **The client is honest** — it is the open-source `rein` binary built from
   the public repository, or your own build of it. A modified client can
   upload whatever it likes; nothing server-side could stop it, and nothing
   here claims to. Reproducible builds and signed releases are the answer,
   not the control plane.
2. **The device is not compromised.** Malware on a device holds what the
   device holds: the unwrapped root key during a command and the plaintext
   sessions at all times, with or without Reinstate.
3. **age, X25519, ChaCha20-Poly1305, argon2id, and scrypt are sound** as
   implemented by `filippo.io/age` and `golang.org/x/crypto`.
4. **The storage provider enforces credential scope.** Step 4 of the
   verification tests this empirically every time it runs; it is the one
   assumption about the operator's infrastructure the client can probe.
5. **Randomness is good.** Root keys, device keys, snapshot ids, bucket
   names, and tokens come from the operating system's CSPRNG.
6. **The recovery code is kept by the person.** Anyone holding it and the
   keyring can unwrap the root key; that is what it is for.

## Out of scope

- **Availability and deletion.** The operator can delete, withhold, or
  corrupt ciphertext. A corrupted object fails to decrypt (step 3) rather
  than yielding wrong plaintext, but nothing prevents loss. Keep a second
  device or an export.
- **An operator that holds both the control plane and the bucket, and
  writes.** Everything above treats the operator as an observer. It also
  owns the storage account, so it can *write* to a locker, and
  `keyring.v1.json` is plaintext: an operator can replace it with a
  keyring of its own, appending a generation that wraps a root key it
  chose to every listed device's published public key. Whether a device
  would adopt such a generation turns on one property of the keyring
  format — whether each generation is authenticated by the one before it,
  by something only a holder of the previous generation's root key could
  produce. Where the format does not do that, a device that took its root
  key from the keyring would seal new objects to a key the operator holds;
  where it does, a planted generation is refused when it is read.
  `internal/keyring` is where that is decided, and
  [security-model.md](../security-model.md) describes the key model as it
  ships. Two client-side facts hold either way, and neither is a
  substitute: `rein account join` never treats a keyring that already
  lists this device as proof of enrolment (it always opens a fresh pairing
  request and uses only the root key the approval relayed, matched against
  the keyring), and everything written *before* a substitution stays
  sealed to the key the operator does not have.
  **`rein sync verify` does not examine `keyring.v1.json` at all** — it
  never fetches it, and says so by naming it among the objects it judged
  by name only — so nothing in the verification report speaks to this
  either way. A unit test holds the checks to that.
- **Traffic analysis** beyond what is listed above: request timing, IP
  addresses, and sizes are visible to the operator and the storage provider
  as with any hosted service.
- **Billing-side data** held by the payment processor.
- **Coercion or legal process against the operator**: the operator can be
  made to hand over everything it has, and everything it has is listed
  under "can see". It cannot be made to hand over what it does not hold.

## How `rein sync verify` maps to the claims

| Step | What it does | Claim it supports |
| --- | --- | --- |
| 1. List the locker | Lists every object (every page) with the credentials this device pushes with and records the access key id locally. | These credentials reach this account's prefix, and this is everything under it: the object names, kinds and count, as a listing with nothing hidden. It does **not** prove the locker holds only the three object kinds — an object of any other name is listed, counted as "other", and passes this step; step 2 is what looks at bytes, and only at the two objects it fetches. What the operator sees from the same listing is counts, sizes and times, and nothing more. |
| 2. Fetch and inspect | Downloads `manifest.age` (and one snapshot) and checks the raw bytes start with `age-encryption.org/v1`, names the recipient type (`X25519` for Hop, `scrypt` for BYO), and finds none of the field names that appear in the plaintext. | What is stored is an age envelope, not content. A tampered or plaintext object fails here. |
| 3. Decrypt locally | Opens the same bytes with the key held on this device, shows the index it contains, and checks a snapshot's payload against the SHA-256 in its envelope. Nothing is sent anywhere. | The ciphertext is real (it opens to the expected structure with the expected key) and intact (authenticated; a flipped byte fails here). The key is on the device, not with the operator. |
| 4. Isolation | Asks the control plane for its **reference locker** (an operator-owned bucket, named like any locker, holding one plaintext probe), checks it names a **different bucket** at the **same storage endpoint** step 1 listed — scheme, host and port, ignoring case, a trailing slash, a trailing dot and an implicit default port — then lists it and reads the probe with the credential that just listed this locker (same access key id, recorded locally in both steps along with the reference endpoint); expects `AccessDenied` both times, **from that endpoint**, as an S3 error naming its code. The probe client refuses to follow a redirect, so the credential is never handed to a host the control plane did not pin. | A credential minted for this account, one the locker has just accepted, is refused from any other bucket at the same endpoint. See below for what fails the step and what makes it not applicable. |

### What fails step 4, and what makes it not applicable

The distinction is the whole value of the step. A **failure** is something
observed that contradicts the claim, and there are four:

- the reference locker **answered** the credential — a successful listing
  or a readable probe;
- the request **landed somewhere other than the pinned endpoint**, or the
  endpoint offered a **redirect** instead of an answer (the probe refuses
  to follow one, so the credential is never sent to the redirect target);
- the control plane named a **different endpoint** than the one step 1
  listed;
- the control plane named a **plaintext `http` endpoint** that is not a
  loopback address. The step signs its request with the temporary secret
  key and session token this device pushes with, and those are never sent
  over an unencrypted connection, whatever endpoint was named — so no
  request is made, and the step fails. (A loopback address is exempt
  because nothing leaves the machine; no Hop endpoint is one.)

Everything else that stops the step is a **check that could not run**. It
is reported not applicable, with a reason beginning "Could not run", it
does not fail the verification or change the exit code, and the outcome
sentence says isolation was not checked rather than asserting it: BYO
storage (no control plane); a control plane that advertises no reference
locker, could not be reached, or answered an error; a reference row naming
this account's **own** bucket (these credentials are meant to reach it, so
it proves nothing — and treating its answer as a finding would be exactly
backwards); a step 1 that did not pass or had nothing to check; a locker
whose own bucket or endpoint is not known on this device; a reference
bucket that has been deleted, timed out, or dropped the connection; a
locker credential rejected (`InvalidAccessKeyId`, `SignatureDoesNotMatch`,
`ExpiredToken`, `InvalidToken`) or rotated between step 1 and step 4 — a
credential no bucket accepts is refused everywhere; and a 403 with no S3
error body, which any web server answers.

Most of those are faults on the operator's side of the service. Reporting
them as a failed verification would tell a customer their locker failed a
security check because a row on the control plane was wrong, and an alarm
that fires for non-events is one nobody believes when it fires for real.

The report is printed in full locally, with per-session detail; after the
first successful push on each new device, and on every `rein sync verify`
unless `--post=false`, the step results are posted to the control plane so
the account console can show them per device; a run with nothing to check
posts nothing. Exit code `7` (`safety`) on any failed step.

A passing report proves what the device observed at that moment against
that endpoint. It does not prove the operator's internals; it proves that
the operator's claims held when you checked, with tools you can rerun and
read.
