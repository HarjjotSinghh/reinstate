# Locker object format

This is the exact layout of what Reinstate writes to remote storage, for a
Hop locker and for a BYO bucket alike. It is the reference that
`rein sync verify` checks against ([threat model](threat-model.md) says what
each check proves). Anyone with bucket access and no key can confirm every
claim on this page from the bytes alone; anyone with the key can confirm the
rest.

Vocabulary: an **account** owns one **locker** (the bucket the control
plane provisions for exactly one account); a **device** is one enrolled
machine; the **root key** is the account's single encryption key; the
**keyring** is the object that carries the root key wrapped for each device;
a **key generation** is one root key's lifetime.

## Objects

Every object lives under the profile **prefix** (`storage.prefix` for BYO,
empty for a Hop locker, whose bucket already belongs to one account):

| Object | Kind | Written by | Replaced? |
| --- | --- | --- | --- |
| `manifest.age` | age envelope around the index | every push | yes, with `If-Match` on the previous ETag (compare-and-swap) |
| `snapshots/<uuid>.age` | age envelope around one session snapshot | every push that uploads a session | never; created with `If-None-Match: *`, immutable, named by a random v4 UUID |
| `keyring.v1.json` | plaintext JSON carrying wrapped keys | `rein account init`, `rein account join`/`rein devices approve`, `rein account recover` | yes, compare-and-swap |

Nothing else is written during normal operation. `rein init` writes one
`probes/<uuid>` object holding `ok` and deletes it again, as its check
that the credentials can write; a run interrupted between the two leaves
it behind, and nothing reaps it. A BYO bucket shared by several profiles
holds one such set per prefix.

No object name encodes a device, a session, a project, or a time, and a
listing reveals only how many snapshots exist and how large they are. The
**prefix** is a different matter: on a Hop locker it is empty, but a BYO
profile initialised without `--prefix` defaults to `profiles/<profile
id>`, which is the account's own identifier, and a prefix given by hand is
whatever you chose. `rein sync verify` step 1 lists exactly what is
there.

## The age envelope

`manifest.age` and every snapshot are **age v1** files
(<https://age-encryption.org/v1>), streamed, in binary form (no ASCII
armor). The file begins with the literal line

```text
age-encryption.org/v1
```

followed by one recipient stanza per recipient, the header MAC, and the
STREAM payload (ChaCha20-Poly1305 in 64 KiB chunks). The stanza type tells a
reader which key model wrote the object without revealing the key:

| Key model | Stanza | Who holds the secret |
| --- | --- | --- |
| Hop (root key) | `-> X25519 <ephemeral share>` | the account's root key, an X25519 identity derived from 32 random bytes; exactly one stanza, for the current key generation (earlier generations only ever open older objects) |
| BYO (passphrase) | `-> scrypt <salt> <work factor>` | the passphrase typed on each device |

The header carries no account, device, or session identifier; the payload is
authenticated chunk by chunk, so a flipped byte anywhere makes decryption
fail rather than produce altered plaintext (`rein sync verify` step 3).

## Plaintext inside the envelopes

### `manifest.age` → the index

A single JSON document, at most 4 MiB:

```json
{
  "schema_version": 1,
  "revision": "<snapshot uuid of the last push>",
  "updated_at": "<RFC 3339>",
  "sessions": {
    "<agent>:<session id>": {
      "agent": "claude",
      "session_id": "<agent's session id>",
      "snapshot_id": "<uuid>",
      "project_id": "<profile project id>",
      "updated_at": "<RFC 3339>"
    }
  }
}
```

`revision` is the snapshot id of the last push, kept for display and for
local state. It is **not** the concurrency token — nothing reads it to
decide a conflict. Two mechanisms do that, and both are checked on every
push:

- The index is replaced with `If-Match` on the ETag the push read it at
  (`If-None-Match: *` when there was no index). A device whose ETag is
  stale is refused by the storage endpoint and re-reads, up to four
  times, before reporting a conflict. `keyring.v1.json` is written the
  same way.
- Each session carries its own parent: a push records the snapshot id it
  believed the session's head to be, and stops with a conflict if the
  index names a different one. Two devices pushing different sessions
  therefore never conflict with each other, however close together.

### `snapshots/<uuid>.age` → one session

Line 1 is the **snapshot envelope** (JSON, at most 1 MiB, terminated by
`\n`); everything after it is the payload, a tar export of the session as
the agent adapter produced it (at most 32 GiB):

```json
{
  "schema_version": 1,
  "kind": "reinstate-session-snapshot",
  "snapshot_id": "<uuid>",
  "parent_revision": "<uuid of the snapshot this replaces, or empty>",
  "agent": "claude",
  "adapter_schema": 1,
  "source_agent_version": "",
  "source_platform": "darwin-arm64",
  "project_id": "<profile project id>",
  "session_id": "<agent's session id>",
  "created_at": "<RFC 3339>",
  "files": [{"path": "<relative path>", "mode": 384, "size": 2048, "sha256": "<hex>"}]
}
```

A pull streams the payload into a temporary file in the destination
directory while hashing it, and renames that file into place only if the
length and SHA-256 match `files[0]`; a mismatch, or any failure on the
way, removes the temporary file and leaves the destination untouched. So
no unverified byte ever reaches the destination path, though bytes do
reach the disk beside it while they are being checked. `rein sync verify`
does the same check in memory. Paths inside the export are relative, so a snapshot made on
Windows restores on macOS (path remapping is applied at pull time, never
stored).

### `keyring.v1.json` → the wrapped root key

The only plaintext JSON in the locker. It holds **no usable key**: every
copy of the root key inside it is encrypted to something the locker does
not contain.

```json
{
  "schema_version": 1,
  "profile_id": "<account id>",
  "current_generation": 1,
  "generations": [{
    "number": 1,
    "created_at": "<RFC 3339>",
    "recipient": "age1…",
    "recovery": {"kdf": "argon2id", "time": 3, "memory_kib": 65536, "threads": 4, "salt": "<base64>", "wrap": "<base64>"},
    "devices": [{"device_id": "<device id>", "public_key": "age1…", "enrolled_at": "<RFC 3339>", "wrap": "<base64>"}]
  }]
}
```

- `recipient` is the generation's root-key **public** key (age X25519
  recipient), so a device can confirm it unwrapped the right key.
- Each `devices[].wrap` is the 32-byte root key age-encrypted to that
  device's X25519 public key. The matching private key is generated on the
  device by `rein account init`, `rein account join` or `rein account
  recover` — the commands that enrol it in the account, not `rein login`,
  which only signs the device in — and kept in the OS keychain; it never
  leaves the device and is never in the locker.
- `recovery.wrap` is the root key sealed with XChaCha20-Poly1305 under a key
  derived from the recovery code by argon2id (the parameters are recorded
  beside it so they can be raised later). The recovery code is shown once
  at `rein account init` and stored nowhere.
- The `generations` array is a list because a future root key rollover
  appends to it: older generations stay so objects sealed under them
  remain readable, and every envelope written afterwards is sealed to the
  newest key only. **Nothing on this release writes a second generation.**
  Device revocation and the rollover that goes with it land with #11; see
  [security-model.md](../security-model.md) for what does and does not
  ship today.

Pairing (device approval) relays a root-key wrap under a code-derived key
through the control plane; it writes the joining device's wrap into this
object and nothing else to the locker.

## Sizes and counts an observer can derive

From a listing alone: the number of snapshots, each object's size and
last-modified time, and therefore roughly how often pushes happen and how
large sessions are.

`keyring.v1.json` is plaintext and gives up more than counts. It carries
the account's `profile_id`, every enrolled `device_id`, every device's
X25519 **public** key, each device's enrolment time, and the number of key
generations — all of it visible in the example above. It carries no usable
key. The identifiers are opaque outside the locker (they name nothing a
person is called), but anyone with bucket access can tell one account's
locker from another's and count and track its devices. See the [threat
model](threat-model.md) for what that means.

## Reproducing the checks by hand

Steps 1, 2 and 4 can be run with any S3 client. Step 3 can be, on BYO
storage; on a Hop locker it cannot, and the reason is stated below rather
than worked around.

For BYO storage use the profile's own keys and coordinates. For a Hop
locker, `rein hop credentials` mints one credential set and prints the
bucket, endpoint and region beside it:

```bash
eval "$(rein hop credentials | grep '^AWS_')"            # an hour, this bucket only
aws s3 ls s3://<bucket>/<prefix>/ --recursive            # step 1: the listing
aws s3 cp s3://<bucket>/<prefix>/manifest.age - | head -c 22  # step 2: "age-encryption.org/v1"
aws s3 cp s3://<bucket>/<prefix>/manifest.age - | strings | grep -c '"sessions"'   # step 2: 0
```

Step 4 uses the same credentials against the operator's reference locker,
whose bucket, region and probe key come from `GET /v1/verify/reference`
with the device token (`rein sync verify` prints them in step 4's detail
lines). Both requests must be refused with `AccessDenied`, from the same
storage endpoint step 1 listed, as an S3 error body naming the code:

```bash
aws s3 ls s3://<reference bucket>/                       # step 4: AccessDenied
aws s3 cp s3://<reference bucket>/<probe key> -          # step 4: AccessDenied
```

Step 3 needs the key. On BYO storage that is the passphrase you already
have: `age -d manifest.age` reproduces the check exactly. On a Hop locker
it is the account's root key, which lives in the OS keychain wrapped to
this device and is never written to a file — **no command exports it, and
none will**. A command that wrote it out would hand over every object the
account has ever written, past and future, in one step; that is a larger
exposure than the gap it would close. `rein sync verify` performs step 3
on the device and shows what the plaintext contains; on a Hop locker,
that is where the reproduction ends.
