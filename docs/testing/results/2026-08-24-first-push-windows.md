# First push and pull on Hop with the root key — native Windows, 2026-08-24

Physical one-device journey for ticket #8 (the hosted first push): a Windows
desktop signs in, provisions its locker, generates the root key, pushes one
Claude Code, one Codex, and one OpenCode session, is wiped (agent stores,
Reinstate home, device token, device key), signs in again, recovers from the
recovery code, pulls, and verified-resumes each session with the real
environment verifier and the real vendor binaries. The control plane and the
locker ran on the Mac on the LAN. Recorded against `hop/08-first-push-journey`
at `65420bc` (public client) and `reinstate-hosted` `main` at `0fda8f3`
(control plane). Every command and output below is real; the recovery code
is the only redaction. Clock note: the bench and the control plane stamp UTC;
the lab ran at 22:33 IST on 2026-08-23, which is 17:03 UTC.

## Verdict

- **First push:** `PASS`. Sign-in to the first successful push took
  **2.5 s** on the bench (budget 120 s; the browser round-trip was a script
  on the Mac approving the emailed link, so this is the CLI's own time plus
  the LAN). Three snapshots landed, one per agent.
- **first_push reached the control plane exactly once:** `PASS`. The
  control plane's `events` table holds one `first_push` row for the account
  after the first push, the no-op push, the wipe, the recovery, the pulls,
  and the push after recovery. `rein hop status` shows the same time
  (`first push: 2026-08-23T17:03:20Z`) before and after the wipe.
- **Wipe and recover:** `PASS`. With the Reinstate home, the agent stores,
  the device token, and the device key all gone, `rein login` → `rein init
  --hop` → `rein account recover` (code typed through the automation
  descriptor) → `rein pull --all` restored all three sessions, and
  `rein resume … --dry-run` produced a complete launch plan with
  `agent.executable`, `agent.layout`, `agent.version`, and `agent.active`
  all passing for Claude Code 2.1.238, Codex 0.149.0, and OpenCode 1.18.21.
- **Journey exposures fixed on this branch and re-verified here:** an
  unenrolled hosted profile now names the right next step (`account
  recover` / `account join`, not `account init`) when the keyring exists; a
  pull refused on one agent keeps the sessions it already restored (no
  conflict on the next pull); a configured `CLAUDE_CONFIG_DIR` or
  `CODEX_HOME` the agent has not populated yet no longer fails every push
  and pull (the first bench run failed with `GetFileAttributesEx …\.claude\projects:
  The system cannot find the path specified`; see section 5).
- **Also on the bench:** the in-process journey `TestHopFirstPushJourney`
  passes on Windows (`ok … 2.017s`, sign-in to first push 170 ms), and the
  tagged staging journey skips cleanly without `HOP_STAGING_URL`.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` (IST 2026-08-24 evening; see the clock note) |
| Bench | `Microsoft Windows 11 Pro` `10.0.26200`, `windows-amd64` native (not WSL), Go `go1.26.1 windows/amd64`, LAN `192.168.1.2`, driven over SSH; own clone `D:\Projects\hop-08` (the shared checkout was not used) |
| Mac | macOS `26.5.2`, `arm64`, Go `go1.26.5 darwin/arm64`, LAN `192.168.1.6` |
| Control plane | `hopd` from `reinstate-hosted` @ `0fda8f3`, on the Mac: `HOPD_ADDR=0.0.0.0:8081 HOPD_BASE_URL=http://192.168.1.6:8081 HOPD_S3_ENDPOINT=http://192.168.1.6:9001 HOPD_EMAIL_SENDER=log HOPD_STORAGE=fake`, fresh `hopd.db` |
| Locker | `scripts/testing/fakelocker -addr 0.0.0.0:9001` on the Mac (the journeys' in-memory fake S3 on a real address), restarted fresh with hopd |
| Client | `rein.exe` built on the bench from `65420bc` (`go build -o bin\rein.exe ./cmd/reinstate`) |
| Sign-in | email magic links, printed by hopd and approved by a script on the Mac that does what the browser does (GET the confirm page, POST the form) |
| Agents on the bench | `claude 2.1.238`, `codex-cli 0.149.0`, `opencode 1.18.21`, all on `PATH` |
| Isolation | `REINSTATE_HOME=D:\Projects\hop-08-home\rein`; synthetic stores under `D:\Projects\hop-08-home\user` via `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME`; no real agent store was read; OpenCode's store hydrated from `testdata/adapters/opencode/windows/store.sql` |
| Secrets | the recovery code reached `rein` only through an inheritable handle named by `REINSTATE_RECOVERY_CODE_FD` (a lab launcher that feeds the code `account init` shows back into its confirmation prompt, never committed); Credential Manager holds the device token and device key, so every hosted command ran as a scheduled task with `/IT` in the console session, output to files |
| Cleanup | scheduled tasks `hop08-day1` / `hop08-day2` deleted after each run; `hopd`, `fakelocker`, and the approver stopped; `D:\Projects\hop-08-home` left for inspection |

Two lab facts, recorded for the next run:

- `fakelocker` serves every bucket name from one in-memory store. A second
  account against the same running `fakelocker` therefore finds the previous
  account's `keyring.v1.json` and `rein account init` correctly refuses
  (`a keyring already exists for this profile; enrol this device with rein
  account recover instead`). Restart `fakelocker` together with `hopd`.
- `hopd`'s `fake` storage provider reports no usage, so `Usage: 0 B in 0
  object(s)` is the provider, not the push.

## 2. Day one: sign in, init, root key, first push

The Claude Code and Codex fixtures carry `cwd` `D:\Projects\hop-08-home\user\Projects\first-push`
(one user message each); the OpenCode fixture is the committed synthetic seed.
The day-one `rein login` also printed `This device is already signed in
(device … at http://192.168.1.6:8081); continuing enrols it again as a new
device and replaces the stored token.` because Credential Manager still
held the token of the previous lab run; that notice is the one line omitted.

```text
# day one on HARJOTS-BEAST 2026-08-23T22:33:15.1718176+05:30
rein version: reinstate 0.0.0-dev (unknown unknown)
hydrated D:\Projects\hop-08-home\user\xdg\opencode\opencode.db schemaOnly false
planted: <lab>\user\.claude\version, <lab>\user\.claude\projects\D--Projects-hop-08-home-user-Projects-first-push\session-first-push.jsonl, <lab>\user\.codex\sessions\rollout-first-push.jsonl, <lab>\user\xdg\opencode\opencode.db

T0 2026-08-23T22:33:16.0200734+05:30

PS> rein.exe login --email first-push-lab-5@example.com
Warning: http://192.168.1.6:8081 is plain http to a non-loopback host; the device token will travel unencrypted.
A sign-in link was sent to first-push-lab-5@example.com. Open it on any device to approve this one.
Waiting for approval (expires 2026-08-23T17:13:17Z; Ctrl-C to cancel)...
Signed in to Reinstate Hop as first-push-lab-5@example.com.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
exit=0

PS> rein.exe whoami
Account: first-push-lab-5@example.com
Plan:    hop (locker location apac)
Device:  Harjots-Beast (windows-amd64, enrolled 2026-08-23T17:03:17Z)
Hop:     http://192.168.1.6:8081
exit=0

PS> rein.exe init --hop --project local/first-push=D:\Projects\hop-08-home\user\Projects\first-push
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-wjdvjt8tksndtsqv39dzghrxxm at http://192.168.1.6:9001 (location apac, plan hop)
profile_id=aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6 device_id=0fa37ad3-5a8b-4017-8787-bb982c721499
next: rein account init on this first device (or rein account join on another), then rein push
exit=0

PS> hoplaunch.exe -code auto -- rein.exe account init
Your recovery code (shown once, never stored anywhere):

    <recovery code redacted>

Write it down and keep it somewhere safe. It is the only copy of the
root key outside your enrolled devices.
If you lose every enrolled device and this recovery code, nobody can
recover the locker: not you, and not the operator, who only ever holds
ciphertext. Local session copies on each machine are unaffected.

account initialized: root key generated on this device, keyring written to storage
profile_id=aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6 device_id=0fa37ad3-5a8b-4017-8787-bb982c721499 key_generation=1 devices=1
recovery code confirmed on this device; encryption.type=root-key
exit=0

PS> rein.exe push --all --json
{
  "dry_run": false,
  "skipped": 0,
  "snapshots": [
    "a9307485-f46d-43fc-8b01-9918a249079b",
    "0e8974a8-9206-479b-bbd8-c8e705321aa6",
    "52e7fbb6-3f7a-4510-ac22-971e50e540c8"
  ]
}
exit=0

sign-in to first successful push: 00:00:02.5002816 (3 s; budget 120 s)

PS> rein.exe status --json
{
  "remote_sessions": [
    {
      "key": "claude:session-first-push",
      "snapshot": "a9307485-f46d-43fc-8b01-9918a249079b",
      "updated_at": "2026-08-23T17:03:18Z"
    },
    {
      "key": "codex:rollout-first-push",
      "snapshot": "0e8974a8-9206-479b-bbd8-c8e705321aa6",
      "updated_at": "2026-08-23T17:03:18Z"
    },
    {
      "key": "opencode:ses_fixture001",
      "snapshot": "52e7fbb6-3f7a-4510-ac22-971e50e540c8",
      "updated_at": "2026-08-23T17:03:18Z"
    }
  ],
  "revision": "52e7fbb6-3f7a-4510-ac22-971e50e540c8"
}
exit=0

PS> rein.exe hop status
Hop:      http://192.168.1.6:8081
Locker:   lk-wjdvjt8tksndtsqv39dzghrxxm at http://192.168.1.6:9001 (region auto, location apac)
Plan:     hop
Usage:    0 B in 0 object(s) of 5.0 GB (measured 2026-08-23T17:03:19Z)
Devices:  1 of 5
Pushes:   up to 60 credential mints per hour
Created:  2026-08-23T17:03:19Z; first push: 2026-08-23T17:03:20Z
exit=0

PS> rein.exe push --all
pushed 0 snapshot(s), skipped 3 unchanged, dry_run=false
exit=0

PS> rein.exe account status
profile_id=aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6 device_id=0fa37ad3-5a8b-4017-8787-bb982c721499
encryption: root-key
this device: enrolled via init; recovery code confirmed here: yes; device key in OS keyring: yes
keyring: present; key generation 1; 1 enrolled device(s); this device listed: yes
local record: D:/Projects/hop-08-home/rein/account.json
exit=0
```

Control plane events after day one (`sqlite3 hopd.db`):

```text
sign_up|1
device_enrolled|1
locker_provisioned|1
first_push|1
```

## 3. The device is wiped

```text
# the device is wiped, then recovered: HARJOTS-BEAST 2026-08-23T22:33:27.7455647+05:30
old profile_id=aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6 device_id=0fa37ad3-5a8b-4017-8787-bb982c721499
PS> cmdkey /delete:reinstate:hop/device-token  -> CMDKEY: Credential deleted successfully.
PS> cmdkey /delete:reinstate:reinstate/aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6/device/0fa37ad3-5a8b-4017-8787-bb982c721499  -> CMDKEY: Credential deleted successfully.
after wipe: rein home exists=False user home exists=False token entry=0 device-key entry=0
```

## 4. Sign in again, recover, pull, verified resume

```text
PS> rein.exe whoami
this device is not signed in to Reinstate Hop; run `rein login`
exit=4

PS> rein.exe login --email first-push-lab-5@example.com
Warning: http://192.168.1.6:8081 is plain http to a non-loopback host; the device token will travel 
unencrypted.
A sign-in link was sent to first-push-lab-5@example.com. Open it on any device to approve this one.
Waiting for approval (expires 2026-08-23T17:13:29Z; Ctrl-C to cancel)...
Signed in to Reinstate Hop as first-push-lab-5@example.com.
This device is enrolled as "Harjots-Beast" (windows-amd64); its token is in the OS keyring.
exit=0

PS> rein.exe whoami
Account: first-push-lab-5@example.com
Plan:    hop (locker location apac)
Device:  Harjots-Beast (windows-amd64, enrolled 2026-08-23T17:03:29Z)
Hop:     http://192.168.1.6:8081
exit=0

PS> rein.exe init --hop --project local/first-push=D:\Projects\hop-08-home\user\Projects\first-push
initialized reinstate home for Reinstate Hop (config.toml + state.json); storage.type=hop
locker lk-wjdvjt8tksndtsqv39dzghrxxm at http://192.168.1.6:9001 (location apac, plan hop)
profile_id=aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6 device_id=6478009f-74b2-4d16-b39b-9cfca8a19ffe
next: rein account init on this first device (or rein account join on another), then rein push
exit=0

PS> rein.exe pull --all
this device is not enrolled in the account's keyring yet; run rein account recover with your recovery code, or rein 
account join and approve it from an enrolled device (encryption.type is age-scrypt, the hosted tier uses root-key)
exit=3

PS> hoplaunch.exe -code <recovery code> -- rein.exe account recover
device enrolled from the recovery code; this device now reads everything written under key generation 1
profile_id=aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6 device_id=6478009f-74b2-4d16-b39b-9cfca8a19ffe key_generation=1 devices=2
If you lose every enrolled device and this recovery code, nobody can
recover the locker: not you, and not the operator, who only ever holds
ciphertext. Local session copies on each machine are unaffected.
exit=0

PS> rein.exe account status --json
{
  "profile_id": "aefe83f4-4c49-4c2d-a2d6-f62d0532c6d6",
  "device_id": "6478009f-74b2-4d16-b39b-9cfca8a19ffe",
  "encryption_type": "root-key",
  "enrolled_on_this_device": true,
  "enrolled_via": "recover",
  "recovery_code_confirmed": true,
  "device_key_present": true,
  "keyring_present": true,
  "key_generation": 1,
  "enrolled_devices": 2,
  "device_in_keyring": true,
  "account_path": "D:/Projects/hop-08-home/rein/account.json"
}
exit=0

PS> rein.exe pull --all
opencode session ses_fixture001: compatibility NOT_INSTALLED refuses restore; install and run opencode once on this 
device so its session layout exists, then pull again
exit=5

# Claude Code and Codex installed again (their layout roots exist); OpenCode not yet

PS> rein.exe pull --all
opencode session ses_fixture001: compatibility NOT_INSTALLED refuses restore; install and run opencode once on this 
device so its session layout exists, then pull again
exit=5
claude restored: True; codex restored: True

# OpenCode installed again: its store holds the vendor schema and no sessions
hydrated D:\Projects\hop-08-home\user\xdg\opencode\opencode.db schemaOnly true

PS> rein.exe pull --all --json
{
  "dry_run": false,
  "plans": [
    {
      "agent": "claude",
      "session_id": "session-first-push",
      "snapshot_id": "a9307485-f46d-43fc-8b01-9918a249079b",
      "destinations": [
        "D:\\Projects\\hop-08-home\\user\\.claude\\projects\\D--Projects-hop-08-home-user-Projects-first-push\\session-first-push.jsonl"
      ],
      "backup_root": "D:\\Projects\\hop-08-home\\rein\\backups"
    },
    {
      "agent": "codex",
      "session_id": "rollout-first-push",
      "snapshot_id": "0e8974a8-9206-479b-bbd8-c8e705321aa6",
      "destinations": [
        "D:\\Projects\\hop-08-home\\user\\.codex\\sessions\\rollout-first-push.jsonl"
      ],
      "backup_root": "D:\\Projects\\hop-08-home\\rein\\backups"
    },
    {
      "agent": "opencode",
      "session_id": "ses_fixture001",
      "snapshot_id": "52e7fbb6-3f7a-4510-ac22-971e50e540c8",
      "destinations": [
        "D:\\Projects\\hop-08-home\\user\\xdg\\opencode\\opencode.db"
      ],
      "backup_root": "D:\\Projects\\hop-08-home\\rein\\backups"
    }
  ],
  "pulled": 3
}
exit=0

PS> rein.exe conflicts list

exit=0

# restored content
claude: {"cwd":"D:\\Projects\\hop-08-home\\user\\Projects\\first-push","type":"meta"}
{"message":{"content":"synthetic first push claude windows"},"type":"user"}
codex: {"payload":{"cwd":"D:\\Projects\\hop-08-home\\user\\Projects\\first-push","id":"rollout-first-push"},"type":"session_meta"}
{"content":"synthetic first push codex windows","role":"user","type":"message"}
opencode: messages(ses_fixture001)=2

# verified resume (real environment verifier, vendor binaries on this bench)
claude: 2.1.238 (Claude Code); codex: codex-cli 0.149.0; opencode: 1.18.21

PS> rein.exe resume claude:session-first-push --dry-run
claude ["--resume" "session-first-push"] in [REDACTED_PATH]
Environment decision: confirmation_required
Environment check: agent.executable status=present severity=info provenance=current_observation — the native agent executable is available
Environment check: agent.layout status=match severity=info provenance=current_observation — the native agent session layout is recognized
Environment check: agent.version status=match severity=info provenance=current_observation — the native agent version is in the verified range
Environment check: agent.active status=match severity=info provenance=current_observation — no running claude instance is using this session
exit=0

PS> rein.exe resume codex:rollout-first-push --dry-run
codex ["resume" "rollout-first-push"] in [REDACTED_PATH]
Environment decision: confirmation_required
Environment check: agent.executable status=present severity=info provenance=current_observation — the native agent executable is available
Environment check: agent.layout status=match severity=info provenance=current_observation — the native agent session layout is recognized
Environment check: agent.version status=match severity=info provenance=current_observation — the native agent version is in the verified range
Environment check: agent.active status=match severity=info provenance=current_observation — no running codex instance is using this session
exit=0

PS> rein.exe resume opencode:ses_fixture001 --dry-run
opencode ["--session" "ses_fixture001"] in [REDACTED_PATH]
Environment decision: confirmation_required
Environment check: agent.executable status=present severity=info provenance=current_observation — the native agent executable is available
Environment check: agent.layout status=match severity=info provenance=current_observation — the native agent session layout is recognized
Environment check: agent.version status=match severity=info provenance=current_observation — the native agent version is in the verified range
Environment check: agent.active status=match severity=info provenance=current_observation — no running opencode instance is using this session
exit=0

PS> rein.exe push --all
pushed 0 snapshot(s), skipped 3 unchanged, dry_run=false
exit=0

PS> rein.exe hop status
Hop:      http://192.168.1.6:8081
Locker:   lk-wjdvjt8tksndtsqv39dzghrxxm at http://192.168.1.6:9001 (region auto, location apac)
Plan:     hop
Usage:    0 B in 0 object(s) of 5.0 GB (measured 2026-08-23T17:03:19Z)
Devices:  2 of 5
Pushes:   up to 60 credential mints per hour
Created:  2026-08-23T17:03:19Z; first push: 2026-08-23T17:03:20Z
exit=0
```

`Environment check` lines for `git.*`, `workspace.*`, `source.fresh`,
`baseline.*`, and Codex's `capability.skill.*` rows (the bench's own
installed skills) are omitted above; the decision on all three was
`confirmation_required` because no earlier prelaunch baseline exists for a
session that has just arrived, which is the documented first-launch
behaviour, not a block.

Control plane events after the whole journey (`sqlite3 hopd.db "select type,
device_id is not null, created_at from events order by created_at"`):

```text
sign_up|0|2026-08-23T17:03:17.910760000Z
device_enrolled|1|2026-08-23T17:03:17.911109000Z
locker_provisioned|0|2026-08-23T17:03:19.947624000Z
first_push|1|2026-08-23T17:03:20.224226000Z
device_enrolled|1|2026-08-23T17:03:29.961749000Z
```

One `first_push`, stamped at the first push; the second device enrolment is
the sign-in after the wipe.

## 5. What the bench found before the fix

The first wipe-and-recover run, on `8d94093`, failed at the pull that ran
before any agent had been installed again:

```text
PS> rein.exe pull --all
GetFileAttributesEx D:\Projects\hop-08-home\user\.claude\projects: The system cannot find the path specified.
exit=1
```

With `CLAUDE_CONFIG_DIR` pointing at a directory Claude Code had not
populated yet, the adapter treated the root as installed and then walked a
`projects` directory that did not exist, so every push and pull failed
before reaching any agent. Codex had the same shape with `CODEX_HOME`
(caught by the in-process journey on macOS, where Codex's `sessions`
directory was gone after the wipe). `65420bc` makes both discover nothing,
and the run above is on that build: the pull is refused only for the agent
whose store is genuinely absent (OpenCode, whose SQLite store Reinstate never
invents), names it, and the sessions already restored are kept.

## 6. In-process journeys on the bench

```text
PS> go test ./internal/cli -run TestHopFirstPushJourney -count=1 -v
=== RUN   TestHopFirstPushJourney
    hop_first_push_test.go:290: sign-in to first successful push: 170ms (budget 2m0s)
--- PASS: TestHopFirstPushJourney (0.78s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	2.017s

PS> go test -tags hopacceptance ./internal/cli -run TestHopFirstPushJourneyStaging -count=1 -v
=== RUN   TestHopFirstPushJourneyStaging
    hop_first_push_acceptance_test.go:36: HOP_STAGING_URL and HOP_DEVICE_TOKEN are not set; the staging first-push journey is skipped
--- SKIP: TestHopFirstPushJourneyStaging (0.00s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	1.244s
```

## 7. Not exercised physically

- A staging control plane and a real R2 locker (`TestHopFirstPushJourneyStaging`
  is ready for it; it needs `HOP_STAGING_URL` and a device token).
- A credential expiring mid-push (covered by `TestLockerJourneyLoginInitPushStatus`
  against the fake locker) and the quota refusals (same).
- GitHub sign-in (the lab control plane ran with email links only).
