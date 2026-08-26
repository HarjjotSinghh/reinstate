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

Nothing else is written. A BYO bucket shared by several profiles holds one
such set per prefix. No object name, prefix, or bucket name encodes an
account, a device, a session, a project, or a time; a listing reveals only
how many snapshots exist and how large they are.

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

`revision` doubles as the optimistic-concurrency token: a push that finds a
different head than it started from stops with a conflict instead of
overwriting another device's work.

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

A pull checks that the payload's length and SHA-256 match `files[0]`
before anything is written to disk; `rein sync verify` does the same check
in memory. Paths inside the export are relative, so a snapshot made on
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
  device at sign-in and kept in the OS keychain; it never leaves the
  device and is never in the locker.
- `recovery.wrap` is the root key sealed with XChaCha20-Poly1305 under a key
  derived from the recovery code by argon2id (the parameters are recorded
  beside it so they can be raised later). The recovery code is shown once
  at `rein account init` and stored nowhere.
- A device revocation starts a new generation with a fresh root key; older
  generations stay so objects sealed under them remain readable, and every
  envelope written afterwards is sealed to the new key only.

Pairing (device approval) relays a root-key wrap under a code-derived key
through the control plane; it writes the joining device's wrap into this
object and nothing else to the locker.

## Sizes and counts an observer can derive

From a listing alone: the number of snapshots, each object's size and
last-modified time, and therefore roughly how often pushes happen and how
large sessions are. From `keyring.v1.json`: the number of enrolled devices,
when each enrolled, and the number of key generations. Nothing else — see
the [threat model](threat-model.md) for what the operator does and does not
see.

## Reproducing the checks by hand

With any S3 client: for BYO storage use the profile's own keys and
coordinates; for a Hop locker, `rein hop status --json` names the endpoint
and bucket, and the credentials are the hourly ones the control plane mints
(`POST /v1/locker/credentials` with the device token):

```bash
aws s3 ls s3://<bucket>/<prefix>/ --recursive            # step 1: the listing
aws s3 cp s3://<bucket>/<prefix>/manifest.age - | head -c 22  # step 2: "age-encryption.org/v1"
aws s3 cp s3://<bucket>/<prefix>/manifest.age - | strings | grep -c '"sessions"'   # step 2: 0
```

Step 3 needs the key: `age -d -i <root key identity file> manifest.age`
for Hop (the identity is the device-unwrapped root key; `rein sync verify`
does the unwrap for you) or `age -d manifest.age` with the passphrase for
BYO. Step 4 (`rein sync verify` on Hop only) lists and reads the
operator's reference locker with the same credentials and expects
`AccessDenied` twice, from the same storage endpoint step 1 listed and with
an S3 error body naming the code; `GET /v1/verify/reference` on the control
plane names the bucket and key.
