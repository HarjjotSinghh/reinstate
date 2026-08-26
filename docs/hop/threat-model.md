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
| Operator / control plane | account identity (email or GitHub login), device records, the locker's name and location, hourly credential mints, bucket usage, product events, verification reports, billing identity | the root key, any device private key, the recovery code, a passphrase, any plaintext |
| Storage provider (Cloudflare R2) | the ciphertext objects and `keyring.v1.json`, object sizes and times, request logs | any key |

## What the operator can see

Everything here is either needed to run the service or unavoidable in
object storage; none of it is content.

- **Billing identity**: the email address or GitHub login used to sign in,
  the plan, trial start, and what the payment processor reports.
- **Devices**: how many are enrolled, their self-reported name and
  platform (`laptop`, `darwin-arm64`), when each enrolled and was last seen.
- **Locker shape**: the bucket's total size and object count (sampled by the
  provider, up to an hour stale), and from the bucket itself each object's
  size and last-modified time. That gives the number of snapshots and a
  rough push cadence and session size distribution.
- **Push cadence**: each hourly credential mint is counted (it is the
  push-rate quota), and the first completed push is reported once by the
  client. `first_push_at`, mint timestamps, and object timestamps together
  describe when the account pushes, not what.
- **Key topology**: `keyring.v1.json` is plaintext, so the number of
  enrolled device wraps, their public keys, and the number of key
  generations are visible. Public keys identify nothing outside the locker.
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
  passphrase, and pairing relays only a wrap under a code-derived key.
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
| 1. List the locker | Lists every object (every page) with the credentials this device pushes with and records the access key id locally. | The locker holds only the three object kinds in the [object format](object-format.md); names are opaque; the operator sees counts and sizes, nothing more. |
| 2. Fetch and inspect | Downloads `manifest.age` (and one snapshot) and checks the raw bytes start with `age-encryption.org/v1`, names the recipient type (`X25519` for Hop, `scrypt` for BYO), and finds none of the field names that appear in the plaintext. | What is stored is an age envelope, not content. A tampered or plaintext object fails here. |
| 3. Decrypt locally | Opens the same bytes with the key held on this device, shows the index it contains, and checks a snapshot's payload against the SHA-256 in its envelope. Nothing is sent anywhere. | The ciphertext is real (it opens to the expected structure with the expected key) and intact (authenticated; a flipped byte fails here). The key is on the device, not with the operator. |
| 4. Isolation | Asks the control plane for its **reference locker** (an operator-owned bucket, named like any locker, holding one plaintext probe), checks it lives at the **same storage endpoint** step 1 listed (ignoring scheme case and a trailing slash; a different port or host fails the step, since every host answers a foreign credential with 403), then lists it and reads the probe with the credential that just listed this locker (same access key id, recorded locally in both steps along with the reference endpoint); expects `AccessDenied` both times, **from that host**, as an S3 error naming its code. The probe client refuses to follow a redirect, so the credential is never handed to a host the control plane did not pin, and a redirect fails the step. A refusal of the credential itself (`InvalidAccessKeyId`, `SignatureDoesNotMatch`, `ExpiredToken`, `InvalidToken`) fails the step: a dead credential is refused everywhere and shows nothing about scope. | A credential minted for this account, one the locker has just accepted, is refused from any other bucket at the same endpoint. Not applicable on BYO storage (no control plane), on a control plane that advertises no reference locker, when step 1 did not pass, when the locker's own endpoint is not known on this device, or when the refusal was a 403 with no S3 error body — any web server answers 403, so that decides nothing and is never reported as a pass. |

The report is printed in full locally, with per-session detail; after the
first successful push on each new device, and on every `rein sync verify`
unless `--post=false`, the step results are posted to the control plane so
the account console can show them per device. Exit code `4` on any failed
step.

A passing report proves what the device observed at that moment against
that endpoint. It does not prove the operator's internals; it proves that
the operator's claims held when you checked, with tools you can rerun and
read.
