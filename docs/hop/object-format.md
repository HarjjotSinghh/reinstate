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

In the JSON examples below, `<…>` marks a value that varies per account or
per object, and `<schema version>` marks the format version this build
reads and writes — the constant in the Go package that owns the document,
not a number this page repeats. A test
(`internal/doctest/object_format_test.go`) parses each example on this
page, substitutes those constants, and holds every field name in it to the
struct the code decodes into, so an example here cannot drift from the
format without the build saying so.

## Objects

Every object lives under the profile **prefix** (`storage.prefix` for BYO,
empty for a Hop locker, whose bucket already belongs to one account):

| Object | Kind | Written by | Replaced? |
| --- | --- | --- | --- |
| `manifest.age` | age envelope around the index | every push | yes, with `If-Match` on the previous ETag (compare-and-swap) |
| `snapshots/<uuid>.age` | age envelope around one session snapshot | every push that uploads a session | never; created with `If-None-Match: *`, immutable, named by a random v4 UUID |
| `keyring.v1.json` | plaintext JSON carrying wrapped keys | the account commands that change the device set or the root key (`rein account init`, `rein account join`/`rein devices approve`, `rein account recover`) | yes, compare-and-swap |

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
  "schema_version": <schema version>,
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
  "schema_version": <schema version>,
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
  "schema_version": <schema version>,
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
- The `generations` array is a list because a root-key rollover appends to
  it: older generations stay, so objects sealed under them remain readable
  by every device that held that key, and every envelope written afterwards
  is sealed to the newest key only. `current_generation` names the newest;
  a device opens the generation it holds a wrap in. Which commands roll the
  key over is a property of the release rather than of the format:
  [security-model.md](../security-model.md) describes the key model as it
  ships. What a generation is worth against a party that can *write* to
  the bucket — and what `rein sync verify` does and does not check about
  this object, which is nothing — is in the [threat
  model](threat-model.md).

Pairing (device approval) relays a root-key wrap under a code-derived key
through the control plane; it writes the joining device's wrap into this
object and nothing else to the locker.

## Sizes and counts an observer can derive

From a listing alone: the number of snapshots, each object's size and
last-modified time, and therefore roughly how often pushes happen and how
large sessions are.

`keyring.v1.json` is plaintext and gives up more than counts. It carries
the account's `profile_id`, every enrolled `device_id`, every device's
X25519 **public** key, each device's enrolment time, and one entry per key
generation with the time it started and its public root-key recipient —
all of it visible in the example above. Because each generation lists the
devices enrolled *in that generation*, a locker with more than one shows
which devices stopped being enrolled and when. It carries no usable key.
The identifiers are opaque outside the locker (they name nothing a person
is called), but anyone with bucket access — which includes anyone holding
an hourly credential from `rein hop credentials` — can tell one account's
locker from another's and count and track its devices. See the [threat
model](threat-model.md) for what that means.

## Reproducing the checks by hand

Steps 1, 2 and 4 can be run with any S3 client. Step 3 can be, on BYO
storage; on a Hop locker it cannot, and the reason is stated below rather
than worked around.

For BYO storage use the profile's own keys and coordinates. For a Hop
locker, `rein hop credentials --export` mints one credential set and
prints it as shell `export` statements — the coordinates included, so
nothing below has to be filled in by hand:

```bash
eval "$(rein hop credentials --export)"

aws s3api list-objects-v2 --endpoint-url "$AWS_ENDPOINT_URL" --bucket "$REIN_LOCKER_BUCKET" --prefix "$REIN_LOCKER_PREFIX" --query 'Contents[].Key' --output text

aws s3 cp --endpoint-url "$AWS_ENDPOINT_URL" "s3://$REIN_LOCKER_BUCKET/${REIN_LOCKER_PREFIX}manifest.age" - | head -c 22

aws s3 cp --endpoint-url "$AWS_ENDPOINT_URL" "s3://$REIN_LOCKER_BUCKET/${REIN_LOCKER_PREFIX}manifest.age" - | grep -a -c '"sessions"'
```

The first command is step 1: every object under this account's prefix.
`REIN_LOCKER_PREFIX` is that prefix as the locker record gives it —
empty on a locker that has none, and ending in `/` on one that has a
prefix — so both lines above work either way and nothing has to be edited.
Hop lockers are provisioned without a prefix today, but the locker record
carries the field (`internal/hop.Locker.Prefix`) and the client honours it
everywhere, so the recipe passes it rather than assuming it away. On BYO
storage the prefix is the one in your own `config.toml`. The second and
third commands are step 2: the first 22
bytes are the age v1 header line `age-encryption.org/v1`, and `"sessions"`
— a field name that is in the index before encryption — appears nowhere in
the body, so `grep -c` prints `0` (and exits 1, as `grep` does when it
finds nothing).

`--export` writes `export` statements rather than bare assignments on
purpose: `aws` reads its credentials from the environment, and variables
that are assigned but not exported are invisible to it. It sets
`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`,
`AWS_ENDPOINT_URL`, `AWS_REGION`, `AWS_DEFAULT_REGION`,
`REIN_LOCKER_BUCKET` and `REIN_LOCKER_PREFIX`. `--endpoint-url` is passed
explicitly as well because `AWS_ENDPOINT_URL` is only honoured by newer
AWS CLI versions.

Step 4 uses the same credentials against the operator's reference locker.
Its bucket and probe key are printed by `rein sync verify` in step 4's
detail line ("reference locker `<bucket>` at `<endpoint>`, probe
`<key>`"); paste them in. Both requests must be refused with
`AccessDenied`, from the same storage endpoint step 1 listed, as an S3
error body naming the code:

```bash
REF_BUCKET=paste-the-reference-bucket-here
REF_KEY=paste-the-probe-key-here

aws s3api list-objects-v2 --endpoint-url "$AWS_ENDPOINT_URL" --bucket "$REF_BUCKET"

aws s3api get-object --endpoint-url "$AWS_ENDPOINT_URL" --bucket "$REF_BUCKET" --key "$REF_KEY" /dev/stdout
```

The endpoint is the same `$AWS_ENDPOINT_URL` throughout, and that is the
point of step 4: a refusal only shows bucket scope when it comes from the
endpoint that just accepted the credential. `rein sync verify` checks the
scheme, host and port, and refuses to send the credential to a plaintext
`http` endpoint — with one exemption, a loopback address (`localhost`,
`127.0.0.0/8`, `::1`), where the request does not leave the machine. That
is what the test fakes and a locally run control plane use; no Hop
endpoint is one. Running these commands by hand has no such guard: `aws`
sends the credential wherever `--endpoint-url` points, so check the scheme
yourself.

These commands are exercised by
`internal/cli/hop_recipe_test.go`, which runs this section's shell through
`sh` against the in-process fake locker and then makes the same four
requests with the credentials the first line printed. What that test does
not run is the `aws` binary itself: it stands in a shim that records the
arguments and serves the object bodies, so the recipe's shell and its
requests are covered and the AWS CLI's own argument handling is not.

Step 3 needs the key. On BYO storage that is the passphrase you already
have: `age -d manifest.age` reproduces the check exactly. On a Hop locker
it is the account's root key, which lives in the OS keychain wrapped to
this device and is never written to a file — **no command exports it, and
none will**. A command that wrote it out would hand over every object the
account has ever written, past and future, in one step; that is a larger
exposure than the gap it would close. `rein sync verify` performs step 3
on the device and shows what the plaintext contains; on a Hop locker,
that is where the reproduction ends.
