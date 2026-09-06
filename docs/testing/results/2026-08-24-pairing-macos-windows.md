# Device approval (pairing) — macOS (device A) ↔ native Windows (device B), 2026-08-24

Physical two-device journey for ticket #8 (hosted-tier pairing): the root key
generated on a Mac reaches a Windows desktop through a locally run control
plane with nothing typed on the Windows side but the sign-in, and the Windows
machine then pulls and reads what the Mac pushed. Recorded against the
branch `hop/08-pairing` at `16d2070` (public client) and `dce76e6`
(private control plane). Every command and output below is real; nothing was
paraphrased. Clock note: the control plane and both devices stamp UTC; the
lab ran at 19:50–20:10 IST on 2026-08-23, which is 14:20–14:40 UTC, so the
timestamps in the outputs read `2026-08-23T14:…Z`.

## Verdict

- **Pairing journey:** `PASS`. A 60-bit code shown on Windows, typed on
  macOS (lower case, spaces instead of dashes, through the automation
  descriptor), enrolled the Windows device; Windows pulled the Mac's session
  with its path remapped to `D:\…`.
- **Fail-closed checks on the approving side:** `PASS`. A typo was rejected
  by the checksum (exit `2`) and a wrong code with a valid checksum was
  refused (exit `7`) with the keyring untouched (1 device) and the request
  still pending on the control plane with no payload.
- **Code never reached the control plane:** `PASS`. The code is absent from
  `hopd.db` and `hopd.log`; the request row holds a 24-byte salt, a 44-byte
  binding, and a public key. The control plane recorded
  `pairing_requested`, `pairing_approved`, and `trial_started`, and stamped
  `accounts.trial_started_at` at the approval instant.
- **Not exercised physically** (covered at the `httptest` and CLI seams):
  request expiry, concurrent claims and approvals, the claim rate limit.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` (IST 2026-08-24 evening; see the clock note) |
| Device A | macOS `26.5.2`, `arm64`, Go `go1.26.5 darwin/arm64`, LAN `192.168.1.6` |
| Device B | `Microsoft Windows 11 Pro` `10.0.26200`, `windows-amd64` native (not WSL), Go `go1.26.1 windows/amd64`, LAN `192.168.1.2`, driven over SSH |
| Control plane | `hopd` from `reinstate-hosted` @ `dce76e6`, run on device A: `HOPD_ADDR=0.0.0.0:8080 HOPD_BASE_URL=http://192.168.1.6:8080 HOPD_S3_ENDPOINT=http://192.168.1.6:9000`, email sender `log`, storage `fake` |
| Locker | `scripts/testing/fakelocker -addr 0.0.0.0:9000` on device A (the journeys' in-memory fake S3 on a real address; accepts the `FAKEKEYnnnn` ids hopd's fake provider mints) |
| Client | `rein` built from `hop/08-pairing` @ `16d2070` on each device (`go build ./cmd/reinstate`); Windows also ran `go build ./...` and `go test ./internal/cli -run 'TestPairingJourney|TestLockerJourney|TestAccountJourney'` → `ok … 3.843s` |
| Sign-in | email magic links, printed by hopd and submitted by a script on the Mac (the POST a person makes in the browser) |
| Isolation | separate `REINSTATE_HOME` per device, synthetic `CLAUDE_CONFIG_DIR` under the lab directory on both; no real agent store was read |

Two bench facts that cost time, recorded for the next run:

- macOS: `security` (the Keychain) refuses when `HOME` is overridden, so the
  synthetic Claude store is pointed at with `CLAUDE_CONFIG_DIR`, not `HOME`.
  The first sign-in attempt with `HOME` overridden failed with `store device
  token in OS keyring: exit status 154` after the control plane had already
  enrolled the device, which is why the device list below shows one extra
  `darwin-arm64` row "with no root-key wrap".
- Windows: Credential Manager is unreachable from the public-key SSH logon
  (`A specified logon session does not exist`). Every hosted command on B
  ran through a scheduled task with `/IT` in the console user's session
  (`schtasks /Create … /IT`, `schtasks /Run`), output written to files. The
  first SSH-session sign-in attempt likewise enrolled a device whose token
  could not be stored, hence the extra `windows-amd64` row.

## 2. Device A: sign in, init, root key, push

```text
$ rein login --email pair-lab@example.com
Warning: http://192.168.1.6:8080 is plain http to a non-loopback host; the device token will travel unencrypted.
A sign-in link was sent to pair-lab@example.com. Open it on any device to approve this one.
Waiting for approval (expires 2026-08-23T14:32:59Z; Ctrl-C to cancel)...
Signed in to Reinstate Hop as pair-lab@example.com.
This device is enrolled as "Unknown_b6:d9:b8:86:9d:a5" (darwin-arm64); its token is in the OS keyring.

$ rein init --hop --project local/pair=<lab>/userA/Projects/pair-source
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-3g6kaykhbns7vv7mtn3yj3ac8r at http://192.168.1.6:9000 (location apac, plan hop)
profile_id=cedb5b42-4f30-4b7b-86b1-a57fce405878 device_id=339d3385-9c49-48d6-951f-2d6dc8a95172
next: rein account init on this first device (or rein account join on another), then rein push

$ rein account init          # recovery code fed back through REINSTATE_RECOVERY_CODE_FD
Your recovery code (shown once, never stored anywhere):

    <recovery code redacted>

Write it down and keep it somewhere safe. It is the only copy of the
root key outside your enrolled devices.
If you lose every enrolled device and this recovery code, nobody can
recover the locker: not you, and not the operator, who only ever holds
ciphertext. Local session copies on each machine are unaffected.

account initialized: root key generated on this device, keyring written to storage
profile_id=cedb5b42-4f30-4b7b-86b1-a57fce405878 device_id=339d3385-9c49-48d6-951f-2d6dc8a95172 key_generation=1 devices=1
recovery code confirmed on this device; encryption.type=root-key

$ rein push --agent claude --all
pushed 1 snapshot(s), skipped 0 unchanged, dry_run=false
```

The pushed fixture was one synthetic Claude session whose `cwd` is the
Mac path `<lab>/userA/Projects/pair-source` and whose one user message is
`physical pairing journey macos to windows 2026-08-24`.

## 3. Device B (Windows): sign in and init for the same account

```text
PS> rein.exe login --email pair-lab@example.com      (scheduled task, /IT)
A sign-in link was sent to pair-lab@example.com. Open it on any device to approve this one.
Waiting for approval (expires 2026-08-23T14:42:33Z; Ctrl-C to cancel)...
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
exit=0

PS> rein.exe whoami
Account: pair-lab@example.com
Plan:    hop (locker location apac)
Device:  Harjots-Beast (windows-amd64, enrolled 2026-08-23T14:33:15Z)
Hop:     http://192.168.1.6:8080
exit=0

PS> rein.exe init --hop --project local/pair=D:\Projects\hop8lab\userB\Projects\pair-target
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-3g6kaykhbns7vv7mtn3yj3ac8r at http://192.168.1.6:9000 (location apac, plan hop)
profile_id=cedb5b42-4f30-4b7b-86b1-a57fce405878 device_id=375b8635-11b6-4145-93a1-8299726eeb2a
next: rein account init on this first device (or rein account join on another), then rein push
exit=0
```

Same account, same profile, same locker; a different device id.

## 4. Pairing

Device B publishes the request and shows the code (stderr, once):

```text
PS> rein.exe account join
Pairing code for this device (never sent to the control plane):

    QXEG-8K98-STMX-KHCK

On an already-enrolled device, run:  rein devices approve
and enter the code exactly as shown. The request expires at 2026-08-23T14:44:44Z (Ctrl-C to cancel).
```

Device A sees it:

```text
$ rein devices
bae85d74-aebf-4194-8afa-538db9e71ebb  Unknown_b6:d9:b8:86:9d:a5 (darwin-arm64), enrolled 2026-08-23T14:22:28Z, last seen 2026-08-23T14:22:28Z, no root-key wrap yet
339d3385-9c49-48d6-951f-2d6dc8a95172  Unknown_b6:d9:b8:86:9d:a5 (darwin-arm64), enrolled 2026-08-23T14:23:02Z, last seen 2026-08-23T14:34:50Z, holds a root-key wrap
7daa24e8-6a91-4ee6-a81b-b1034322af4f  Harjots-Beast (windows-amd64), enrolled 2026-08-23T14:30:22Z, last seen 2026-08-23T14:30:22Z, no root-key wrap yet
375b8635-11b6-4145-93a1-8299726eeb2a  Harjots-Beast (windows-amd64), enrolled 2026-08-23T14:33:15Z, last seen 2026-08-23T14:34:48Z, no root-key wrap yet
pending approval: Harjots-Beast (windows-amd64) asked to join at 2026-08-23T14:34:44Z (request 78591810-5834-4c0e-9912-814c997b0438, expires 2026-08-23T14:44:44Z)
run rein devices approve and enter the code shown on the new device
```

A typo (last character changed), through `REINSTATE_PAIRING_CODE_FD`:

```text
$ rein devices approve
Device "Harjots-Beast" (windows-amd64) asked to join this account (request opened 2026-08-23T14:34:44Z).
Approve only if that machine is yours and is showing a pairing code right now.
pairing code checksum does not match; check it for typos
exit=2
```

A wrong code with a valid checksum (`T03H-6255-1BY2-EQBB`, drawn fresh):

```text
$ rein devices approve
Device "Harjots-Beast" (windows-amd64) asked to join this account (request opened 2026-08-23T14:34:44Z).
Approve only if that machine is yours and is showing a pairing code right now.
the code does not match this pairing request (wrong code, or the request was altered in transit); nothing was approved
exit=7

$ rein account status --json | grep enrolled_devices
  "enrolled_devices": 1,

$ sqlite3 hopd.db "select status, length(coalesce(payload,'')), claims from pairing_requests;"
pending|0|24
```

The right code, typed as a person would (`qxeg 8k98 stmx khck`), through
the descriptor:

```text
$ rein devices approve
Device "Harjots-Beast" (windows-amd64) asked to join this account (request opened 2026-08-23T14:34:44Z).
Approve only if that machine is yours and is showing a pairing code right now.
approved device "Harjots-Beast" (windows-amd64); it now reads everything under key generation 1
exit=0
```

Device B's waiting `account join` completes on its next poll:

```text
device approved by "Unknown_b6:d9:b8:86:9d:a5"; this device can now read the locker
profile_id=cedb5b42-4f30-4b7b-86b1-a57fce405878 device_id=375b8635-11b6-4145-93a1-8299726eeb2a key_generation=1
exit=0
```

Control plane after the approval (SQLite on device A):

```text
$ sqlite3 hopd.db "select status, length(coalesce(payload,'')), claims, key_generation from pairing_requests;
                   select type, created_at from events where type like 'pairing%' or type='trial_started';
                   select trial_started_at from accounts;"
consumed|0|55|1
pairing_approved|2026-08-23T14:36:31.924888Z
pairing_requested|2026-08-23T14:34:44.57426Z
trial_started|2026-08-23T14:36:31.924888Z
2026-08-23T14:36:31.924888Z

$ grep -c QXEG8K98STMXKHCK hopd.db hopd.log; grep -c QXEG-8K98-STMX-KHCK hopd.db hopd.log
(no matches in either file, either spelling)

$ sqlite3 hopd.db "select substr(public_key,1,12)||'…', length(salt), length(binding), status from pairing_requests;"
age100fk9ycu…|24|44|consumed
```

The payload was handed out once and deleted (`length(payload) = 0` after
the claim, status `consumed`); 55 polls happened over the two minutes the
request waited, well under the 600 cap.

## 5. Device B reads the Mac's data

```text
PS> rein.exe account status --json
{
  "profile_id": "cedb5b42-4f30-4b7b-86b1-a57fce405878",
  "device_id": "375b8635-11b6-4145-93a1-8299726eeb2a",
  "encryption_type": "root-key",
  "enrolled_on_this_device": true,
  "enrolled_via": "join",
  "recovery_code_confirmed": false,
  "device_key_present": true,
  "keyring_present": true,
  "key_generation": 1,
  "enrolled_devices": 2,
  "device_in_keyring": true,
  "account_path": "D:/Projects/hop8lab/homeB/account.json"
}
exit=0

PS> rein.exe pull --agent claude --all --json
{
  "dry_run": false,
  "plans": [
    {
      "agent": "claude",
      "session_id": "session-pair",
      "snapshot_id": "e9ff0df8-4c74-4590-bbe6-ccb195ab7e56",
      "destinations": [
        "D:\\Projects\\hop8lab\\userB\\.claude\\projects\\D--Projects-hop8lab-userB-Projects-pair-target\\session-pair.jsonl"
      ],
      "backup_root": "D:\\Projects\\hop8lab\\homeB\\backups"
    }
  ],
  "pulled": 1
}
exit=0

PS> Get-Content D:\Projects\hop8lab\userB\.claude\projects\D--Projects-hop8lab-userB-Projects-pair-target\session-pair.jsonl
{"cwd":"D:\\Projects\\hop8lab\\userB\\Projects\\pair-target","type":"meta"}
{"message":{"content":"physical pairing journey macos to windows 2026-08-24"},"type":"user"}

PS> rein.exe devices
bae85d74-aebf-4194-8afa-538db9e71ebb  Unknown_b6:d9:b8:86:9d:a5 (darwin-arm64), enrolled 2026-08-23T14:22:28Z, last seen 2026-08-23T14:22:28Z, no root-key wrap yet
339d3385-9c49-48d6-951f-2d6dc8a95172  Unknown_b6:d9:b8:86:9d:a5 (darwin-arm64), enrolled 2026-08-23T14:23:02Z, last seen 2026-08-23T14:36:31Z, holds a root-key wrap
7daa24e8-6a91-4ee6-a81b-b1034322af4f  Harjots-Beast (windows-amd64), enrolled 2026-08-23T14:30:22Z, last seen 2026-08-23T14:30:22Z, no root-key wrap yet
375b8635-11b6-4145-93a1-8299726eeb2a  Harjots-Beast (windows-amd64), enrolled 2026-08-23T14:33:15Z, last seen 2026-08-23T14:36:47Z, holds a root-key wrap
no pending pairing requests
exit=0
```

The session written on macOS under `<lab>/userA/Projects/pair-source` was
restored on Windows under the Windows project directory with `cwd`
remapped to `D:\Projects\hop8lab\userB\Projects\pair-target`; the JSON
`account_path` is slash-normalized (`D:/…`) while the pull plan reports
native Windows destinations.

## 6. After the pairing

```text
$ rein devices approve            # device A, nothing pending
no pending pairing requests; run rein account join on the new device first
exit=2

PS> rein.exe account join         # device B, already enrolled
this device is already enrolled; rein account status shows the keyring state
exit=7
```

Locker traffic from the Windows device (fake locker log, `192.168.1.2`):
four `GET keyring.v1.json` (join: load, then the post-approval
verification with a re-minted credential; status; devices), then `GET
manifest.age`, `GET snapshots/e9ff0df8-….age`, and one more keyring read
for the pull. Every object the locker holds is the keyring's wrapped blobs
or age ciphertext; no plaintext object was ever written.

## 7. Cleanup

The `hopd` and `fakelocker` processes were stopped and the lab database
left under the session scratchpad. The Windows bench keeps the
`hop/08-pairing` branch checked out under `D:\Projects\reinstate` (its
stash untouched), the lab directory `D:\Projects\hop8lab`, and the
scheduled tasks `hop8-*` were deleted. Both machines' OS keyrings still
hold the lab account's device token and device key (entries named
`reinstate` / `reinstate/cedb5b42-…/device/…`); they refer to a control
plane that no longer exists and can be removed by hand.

## 8. Addendum: expiry-at-prompt fix (review follow-up, ebe0e21)

The bench ran ebe0e21; the branch tip at the time of this section
(a301b8c) differed from it only by this document, so the recorded output
applies to that tip's code.

Review found that an approval whose request expired while the code prompt
was open left the joining device's wrap in the keyring, so the joiner's
retry restored a local enrolment with no approval behind it. The fix
(refuse before any write when the listed expiry has passed; roll the
appended wrap back when the relay then refuses) was verified at the CLI
seam, not re-run physically: the journey now expires a request while A's
prompt is open and asserts the keyring stays at two devices and that C's
retry is a fresh request. The bench re-ran the same journey on the fixed
commit:

```
PS D:\Projects\reinstate> git log --oneline -1
ebe0e210 fix(hop): refuse or roll back an approval whose pairing request expired
PS D:\Projects\reinstate> go test ./internal/cli/ -run "TestPairingJourney|TestLockerJourney|TestAccountJourney" -count=1
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	4.100s
PS D:\Projects\reinstate> go test ./internal/keyring/ ./internal/hop/ -count=1
ok  	github.com/HarjjotSinghh/reinstate/internal/keyring	0.634s
ok  	github.com/HarjjotSinghh/reinstate/internal/hop	0.830s
```

## 9. Addendum: listed keyring entry is not enrolment (review follow-up, 7fa9cb5)

A second review found that `rein account join` enrolled a device with no
code and no approval whenever the bucket's keyring already listed it with
the key this machine holds; since the joining device's public key is in
the pairing request, an operator holding both the relay and the bucket
could forge such a keyring around a root key of its own choosing, answer
the claim with `expired`, and wait for the retry. The restore shortcut is
gone: join always opens a fresh request and waits for a typed approval.
The CLI journey now plays that operator (forged keyring in the bucket,
expired request) and asserts a new pending request, no account record,
and no push. Verified at the CLI seam on the Mac and re-run on the bench
at the fixed commit; no new physical two-device run, because the change
removes a code path rather than adding one and the cross-device flow is
the one recorded in sections 1 to 7.

```
PS D:\Projects\reinstate> git log --oneline -1
7fa9cb51 fix(hop): never treat a keyring that lists this device as enrolment (#9)
PS D:\Projects\reinstate> go build ./...
PS D:\Projects\reinstate> go test ./internal/cli/ -run "TestPairingJourney|TestLockerJourney|TestAccountJourney" -count=1
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	4.520s
PS D:\Projects\reinstate> go test ./internal/keyring/ ./internal/hop/ ./internal/pairing/ -count=1
ok  	github.com/HarjjotSinghh/reinstate/internal/keyring	0.681s
ok  	github.com/HarjjotSinghh/reinstate/internal/hop	0.861s
ok  	github.com/HarjjotSinghh/reinstate/internal/pairing	0.303s
```

The bench stash (`stash@{0}: WIP on (no branch): ae2ffae`) is untouched.
