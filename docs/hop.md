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
rein devices revoke <device-id|name>
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
keyring holds a root-key wrap for it (and in which key generation), any
revoked devices, and any pending pairing request.

## Revoking a device (key generations)

A lost laptop, a retired desktop, a machine you no longer trust: revoke it
from any other enrolled device.

```bash
rein devices                         # find its id or name
rein devices revoke desktop          # asks for the recovery code
```

Revocation starts a new **key generation**. The revoking device unwraps the
current root key, draws a fresh one, wraps it for every remaining device's
public key and under the recovery code, signs the new generation with the
account key that code derives, and appends it to the keyring with
compare-and-swap; then it tells the control plane, which
refuses the revoked device's token from then on (no more credential mints,
no pairing, no device list). Earlier generations are left exactly as they
were, so:

- every remaining device reads everything already in the locker (the
  provider opens with every generation it can unwrap and seals only to the
  current one; which generation an object needs is decided by the object's
  own age header);
- the revoked device keeps what it already pulled, cannot mint new locker
  credentials (credentials it minted before the revocation keep working
  against the bucket until they expire, up to the operator's credential
  TTL of at most an hour, so it can still push or pull within that
  window), and cannot open what any device pushes once that device has the
  new key generation. Whether a device *has* it is the qualification the
  rest of this section is about: it arrives per device, and the control
  plane's key generation floor is what carries it to a device that has not
  read the keyring yet;
- that same window is enough to **write** the keyring object, so the
  keyring authenticates itself. Every generation, the first included,
  carries a `signature`: ed25519 over its own header — the profile id, the
  account key, the generation's number, `created_at` and `recipient`, the
  number and recipient of the generation it follows, and the revocations
  that started it — under a keypair derived from the **recovery code**. The
  recovery code is the one secret no device ever holds: it is shown once,
  written down, re-entered to confirm at `rein account init`, and typed in
  again only at the two commands that need it afterwards — `rein devices
  revoke` and `rein account recover`, both of which already asked for it. A
  revoked device never held it, so it cannot sign a generation the account
  will adopt, and the root key it *did* hold until the rollover buys it
  nothing here;
- verifying needs only the public half, which the keyring publishes as
  `account_key` and each device pins locally at enrolment. So the check
  costs no key material and no prompt, and **every** command that loads the
  keyring makes it — push, pull, `devices approve`, `devices revoke`,
  `account recover`, `account join`, and the two that hold no keys at all,
  `account status` and `rein devices`. It fails closed: a generation whose
  signature is missing, malformed, or wrong makes the **whole** keyring
  untrusted, current or not. There is no partial acceptance and no path
  that accepts a keyring because its current generation happens to check
  out. Every command that acts on the keyring exits `7` (`ExitSafety`) with
  nothing written; the two diagnostics exit `0` and say the keyring is
  refused rather than reporting it as the account's key-model truth;
- the signature is a claim about one key, so it needs an anchor: it says
  every generation here was signed by the account key this object
  publishes, not that this is your account's keyring. A party that can
  write the bucket can replace the whole object with one it signed itself.
  So each device records in `account.json` the account key, the generation
  it last unwrapped, **and that generation's root-key recipient**. A
  keyring signed by a different account key was replaced (`ExitSafety`,
  "signed by a different account key"); one whose `current_generation` is
  lower is a rollback; one where the recorded generation is missing, or now
  names a different root key, was replaced rather than appended to. All
  three fail closed with nothing written, on every path, and the two
  diagnostics report them the same way. The record must carry all three
  values or the command refuses and names the way out; a record with a
  field deleted does not fall back to trusting the object;
- **what deriving a signing key from a typed code costs.** A party holding
  the keyring can guess the recovery code offline by testing candidate
  signatures. That is not a new exposure: the same object already carries
  the recovery *wrap*, which the same party can attack the same way, and
  both cost one argon2id derivation per candidate at the same parameters
  (3 passes, 64 MiB, 4 lanes). The code carries 140 bits of entropy (seven
  groups of four Crockford base32 characters at five bits each, plus a
  checksum group), so the search is 2^140 memory-hard derivations wide. The
  derivation is salted from the profile id, so work done against one
  account is worth nothing against another, and the private half is never
  stored anywhere — it exists only while a command that took the code is
  running;
- a device enrolled later (`rein account join`, or `rein account recover`
  with the recovery code) is enrolled into every generation and reads the
  whole locker too;
- the keyring only grows. Each revocation appends a generation holding one
  wrap per remaining device, and no generation is ever removed, because
  removing one would make everything written under it unreadable. A read
  accepts at most 1 MiB, so a write that would take the object past three
  quarters of that is refused (`ExitSafety`) rather than leaving the
  account with an object it could never read back. At five surviving
  devices that ceiling is around 170 revocations; reaching it means moving
  the account to a fresh locker.

The recovery code is asked for because the new generation must be signed
and must stay recoverable, and nothing but the code can do either; a wrong
code revokes nothing. A device cannot revoke itself. Revoking a device
twice is harmless: the keyring reports it already gone and the control
plane answers idempotently. A revocation that races an approval converges
in either order: an approval lands in whichever generation is current when
its compare-and-swap succeeds, and a joining device handed a payload that
names a generation the keyring has since left fails closed (no account
record) and is simply approved again. Two devices revoking the same device
at the same moment start one generation, not two.

To enrol a revoked machine again, four commands, in this order:

```bash
rein login                  # a new device record and token
rein init --hop --force     # the home names the new device; see below
rein account recover        # or rein account join, approved elsewhere
rein push --all             # back in sync
```

`--force` matters. The revoked machine still carries the local enrolment
record of the enrolment it lost, and both `rein account join` and `rein
account recover` refuse to run where one exists ("this device is already
enrolled", exit `7`, and the message names this step). `rein init --force`
copies `config.toml`, `state.json` and `account.json` into a timestamped
set under `backups/` and then removes the enrolment record, which is the
one file `init` does not otherwise rewrite. Nothing else removes it.

Because `--force` is reached for to remove that one file, it keeps the
**sync state** when the home is being pointed at the same profile: the
profile is the locker, so the snapshot ids in `state.json` are still this
account's and only the device id changed. Without that, the `rein push`
above would see a local session that has moved on and a remote snapshot
with no shared base between them, call it a divergence and exit `6` — the
last step of the recovery path failing on state the path itself discarded.
Re-initializing against a *different* profile is a different locker and does
start from an empty state. Either way the previous `state.json` is in the
backup set.

`rein account status` and `rein devices --json` report the current key
generation; the keyring keeps a `revoked` record on the generation each
revocation started.

### The account key generation floor

Every check described above is **local to one device**: the generation it
last unwrapped, and the root-key recipient it recorded for that generation.
A device that has not yet read the rollover holds nothing a rollback would
contradict — the pre-revocation keyring is genuine, its signatures verify,
and the generation it names is the one that device last saw. Inside its
credential window the revoked device can restore exactly that object, and a
device that has run nothing since would accept it and keep sealing to the
root key the revoked device still holds.

The control plane closes that, because it is the party that saw the rollover
the lagging device missed. It carries one number per account:

| route | who | answer |
| --- | --- | --- |
| `GET /v1/account/key-generation` (bearer) | any enrolled device | `200 {"key_generation": N, "updated_at"}`; `N` is `0` until a revocation raises it. `404 {code: "no_key_generation"}` from a control plane that does not carry one. |
| `POST /v1/account/key-generation` (bearer) `{"key_generation": N}` | any enrolled device; in practice the one doing the revoking | `200 {"key_generation": M, "updated_at"}` where `M` is the higher of what it held and `N` — the number is monotonic, so reporting a smaller one raises nothing. The CLI refuses an answer below the `N` it sent, which catches a control plane that is not keeping it monotonic; it cannot compel one, and the row below says what that leaves open. |

`rein devices revoke` raises the floor once the new generation is in the
keyring and the control plane has refused the revoked token. Every command
that reads the keyring on a Hop profile asks for the floor first and refuses
a keyring below it (`ExitSafety`, naming the control plane as the source),
before unwrapping anything: push, pull, `devices approve`, `devices revoke`,
`account recover`, `account join`, and the two diagnostics.

What that does and does not buy, precisely:

- **Against the revoked device it closes the gap.** That is the adversary
  revocation exists to stop, and the realistic one — a stolen or retired
  laptop. The control plane refuses its token, so it can neither read the
  floor nor lower it, and it cannot stop another device from being told the
  account has moved on.
- **Against an operator holding both the control plane and the bucket it
  adds nothing**, because that party serves whatever floor it likes. The
  recovery-code signature on every generation and the local anchor are what
  cover that adversary, as far as they cover it — see the list below.
- **A profile on your own bucket has no control plane to ask**, so there the
  per-device floor and the local anchor remain the whole of it. `rein
  devices revoke` requires a signed-in Hop account, so revocation is a Hop
  feature either way.
- **A control plane that does not carry the floor** (an older deployment,
  answering `404`) leaves a device that has never confirmed one with nothing
  to check a restored earlier keyring against. `rein devices revoke` prints
  that on stderr rather than reporting a protection it did not get.

**Offline.** There is no offline case on a Hop profile, and that is why this
is not a trade. Reaching the locker at all means minting credentials from
the same control plane in the same command, so a device that cannot reach it
cannot push or pull either — refusing to sync offline is not a choice being
made here. The one case with no live answer is the `404` above, and there
the floor falls back to the last one the control plane confirmed to this
device, which `account.json` records as `control_plane_key_generation` and
`control_plane_confirmed_at` and only ever raises. In fact the floor a
command uses is always the higher of the live answer and that record, so a
control plane that stops serving the route *and* one that answers below a
number it has already given this device both leave the established floor
standing; a refusal made on that basis says which floor it used and when it
was confirmed. What that does not reach is a device that has never confirmed
a floor at all: there is nothing to be higher than, and a control plane
serving `0` — or one that has genuinely never had a revocation — is
indistinguishable to it.

### What revocation establishes, and what it does not

The sentence worth being careful about is "a revoked device cannot open
anything pushed after the revocation". Unqualified it is false, and it was
false in a way that was demonstrated end to end. Here it is with the
qualification:

> A revoked device cannot open what a device pushes once that device has
> the new key generation — which it has as soon as it next reads the
> keyring, and which a control plane carrying the key generation floor
> makes it check for even before then.

Precisely: against a party that can read **and write** the locker bucket —
which a revoked device is, for the rest of its credential's TTL — no
remaining device will accept a key generation that party wrote. Writing one
means signing it under the account key, and that key is derived from the
recovery code, which no device holds and a revoked device never held. The
locally pinned account key rules out replacing the object outright, because
a keyring signed end to end under a key of the attacker's own is internally
perfect and nothing inside it could tell it apart. And restoring a
*genuine* earlier keyring — which needs no signing at all — is refused by
any device that has read the newer generation, and by any device whose
control plane reports a higher floor.

What that does *not* cover, stated plainly:

- **An operator holding both the control plane and the bucket.** The floor
  is only as good as the party serving it, and that party serves it. What
  is left against this adversary is the generation signature (it cannot
  write a generation any device will adopt without the recovery code) and
  the local anchor (it cannot talk a device that has read generation N back
  to N-1). It can still hand a device that has never read this account's
  keyring a floor of 0.
- **A device that has never contacted a control plane carrying the floor,
  on a control plane that does not carry one.** It has nothing to check a
  restored earlier keyring against until it reads the new generation for
  itself. `rein devices revoke` says so when it meets such a control plane.
- **BYO storage.** No control plane, no floor; the per-device anchor is the
  whole of it.

- **A device that has never read this account's keyring has no anchor of
  its own.** It borrows one, and there are exactly two: the recovery code
  (`rein account recover` derives the account key from the typed code and
  requires the keyring to be signed under it) and a typed pairing code
  (`rein account join` takes the root key an enrolled device relayed
  through the approval and requires the keyring's own wrap for this device
  to open to the same bytes in the same generation, so neither the relay
  nor the storage can substitute a keyring alone). A machine holding
  neither cannot enrol, which is the intended answer, but it also means a
  fresh install's anchor is only as good as those two codes.
- **Anyone who knows the recovery code can sign a generation.** That is
  what the code is: knowing it already opens the current root key from the
  recovery wrap, so this adds no secret that was not already decisive.
  Revoking a device does not revoke knowledge of the code, and there is no
  command that rotates it. If the code itself may have leaked, the account
  has to start again on a fresh locker.
- **Objects, not the key model.** Anyone with write access can still
  delete or corrupt snapshots, the manifest, or the keyring itself. That is
  denial of service, and the signature does not address it; it addresses
  reading and writing under a key someone else chose.
- **The wraps inside a generation are not covered by that generation's
  signature**, because devices are enrolled into generations after the
  fact. A wrap is only ever accepted when the key inside it derives the
  generation's recorded `recipient`, and the recipient *is* signed — so
  appending a working wrap still needs that generation's root key. What a
  writer without it can do is remove or corrupt wraps, which again denies
  service rather than substituting a key.
- **Nothing here re-encrypts what was already pushed.** A revoked device
  keeps every generation it held and everything it already pulled.

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
| 8 | `GET /v1/devices` (bearer) | `200 {devices: [{id, name, platform, location_hint?, created_at, last_seen_at, revoked_at?}]}`; revoked devices stay listed with `revoked_at`. |
| 9 | `DELETE /v1/devices/{id}` (bearer; `POST /v1/devices/{id}/revoke` is an alias) | `200 {device, revoked}`; `revoked` is `false` when it already was (idempotent). The token answers `401` on every authenticated route from then on, and the device no longer counts toward the device quota — but a locker credential minted before the revocation keeps working against the bucket until it expires, at most an hour, and the control plane cannot withdraw one. Another account's device or an unknown id: `404 {code: "device_unknown"}` (one answer, so ids cannot be probed across accounts); the calling device itself: `400 {code: "self_revoke"}`. The call carries no key material: the key generation rollover happens in the keyring before it, and `POST /v1/account/key-generation` raises the account floor after it. |

The CLI reads the locker with `GET /v1/locker` on every hosted command and
only calls `POST /v1/locker` when the answer is `no_locker` (normally once,
from `rein init --hop`).

The device quota is enforced when credentials are minted, not when a device
enrols: a sixth device on a five-device plan can still sign in, after which
no device on the account can mint until the count is back under the plan.
`rein devices revoke` is the way out: a revoked device no longer counts.

Device tokens are 256-bit random values prefixed `hop_`. The control plane
stores only a hash, bound to one device record (name, platform, location
hint, created, last seen). The CLI sends no telemetry; the control plane
records `sign_up`, `device_enrolled`, `locker_provisioned`, `first_push`,
`pairing_requested`, `pairing_approved`, `trial_started`, and
`device_revoked` events as its only product metrics.

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

- Sign out a device from itself (revoke it from another device instead;
  `rein sync migrate --to byo --forget-hop` drops this device's token
  locally but does not revoke it at the control plane).
- Run a daemon.
