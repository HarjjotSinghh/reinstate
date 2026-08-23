# OpenCode T5 journey — native Windows x64, 2026-08-23

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

Counterpart to [2026-08-23-macos-opencode-t5-journey.md](2026-08-23-macos-opencode-t5-journey.md).
Together they are the evidence for OpenCode's T5 rung; this half is the native
Windows leg, and it is the cross-device half: the session was created and pushed
by real OpenCode on macOS and pulled here on native Windows, then resumed by the
Windows vendor binary with its workspace path remapped onto this device.

## Verdict

- **Windows journey:** `PASS`.
- **Tier claim:** T5, encrypted same-vendor sync, jointly with the macOS leg.
  Both platforms are now recorded, so the catalog wires `NewSyncAdapter` and
  advertises OpenCode at **T5**.

Nothing here touched the operator's real OpenCode store. Every store lived under
a throwaway `XDG_DATA_HOME`, and the encrypted backend was a local disk-backed
`memory` backend carried over from the macOS device, standing in for a BYO
bucket both devices share.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| CPU architecture | `AMD64`, native process |
| Go toolchain | `go1.25.13` (`go version go1.25.13 windows/amd64`; the bench's default `go` is 1.26.1, pinned with `GOTOOLCHAIN=go1.25.13`) |
| Vendor | OpenCode, single native executable at `C:\Users\admin\.bun\bin\opencode.exe` |
| Vendor version measured | `1.18.21` (bare `MAJOR.MINOR.PATCH` on stdout, identical to macOS) |
| Branch | `hop/04-opencode-t5` @ `1c21cbc` |
| Source device (leg A) | `macos-arm64`, OpenCode `1.18.21` |

## 2. What this leg establishes that macOS could not alone

- The **cross-device** round trip across operating systems: create + `rein push`
  on macOS, `rein pull` on native Windows over the same encrypted backend.
- The **path remap** direction that is the flagship multi-device case: a macOS
  session whose workspace normalised to `${HOME}/code/demo` is restored on
  Windows as `C:\Users\admin\code\demo`.
- The vendor's **own** view of the restored session on Windows: `opencode
  session list` and `opencode --pure export` read it back with both messages,
  both parts, and the assistant→user `parentID` intact — native visibility
  after a Reinstate restore, not merely a Reinstate read of its own write.

## 3. Round trip observed across the two devices

### Leg A — macOS (source), via `rein`

A session was created by the vendor itself (`opencode import` into a throwaway
`$XDG_DATA_HOME/opencode`, verified with `opencode session list`), then:

```text
$ rein list --agent opencode --json
[ { "ID": "ses_hop04mac0000000000000a1", "Agent": "opencode",
    "ProjectID": "${HOME}/code/demo", "Title": "Hop04 macOS seed session", ... } ]

$ rein push --agent opencode --session ses_hop04mac0000000000000a1 --json
{ "dry_run": false, "skipped": 0,
  "snapshots": [ "cabc51ee-ba5d-4d7a-b05a-8c36e71e07f9" ] }
```

`ProjectID` is the portable token `${HOME}/code/demo`, not the vendor's
project-table id (which on 1.18.21 is `global` for a session outside a project);
that is what lets the identity survive the cross-OS remap below.

The macOS `$REINSTATE_HOME` (config + disk-backed `memory` backend + the
snapshot) was copied to the Windows bench (`D:\Projects\hop-reinhome`) to stand
in for a shared BYO bucket.

### Leg B — native Windows (destination), via `rein`

A second, independent OpenCode store was initialised **by the vendor**
(`opencode import` of an unrelated throwaway session) under
`D:\Projects\hop-oc\xdg\opencode`, so the destination store pre-existed and was
not fabricated by Reinstate. The passphrase reached `rein` only through an
inheritable handle named by `REINSTATE_PASSPHRASE_FD` (a tiny local launcher,
never committed), never as an argument or ordinary environment value.

```text
PS> rein status --json
{ "remote_sessions": [ { "key": "opencode:ses_hop04mac0000000000000a1",
    "snapshot": "cabc51ee-ba5d-4d7a-b05a-8c36e71e07f9", ... } ],
  "revision": "cabc51ee-ba5d-4d7a-b05a-8c36e71e07f9" }

PS> rein pull --agent opencode --session ses_hop04mac0000000000000a1 --json
{ "dry_run": false,
  "plans": [ { "agent": "opencode",
      "session_id": "ses_hop04mac0000000000000a1",
      "snapshot_id": "cabc51ee-ba5d-4d7a-b05a-8c36e71e07f9",
      "destinations": [ "D:\\Projects\\hop-oc\\xdg\\opencode\\opencode.db" ],
      "backup_root": "D:\\Projects\\hop-reinhome\\backups" } ],
  "pulled": 1 }
```

### The vendor binary on Windows sees the restored session

```text
PS> opencode --pure session list
Session ID                   Title                                Updated
─────────────────────────────────────────────────────────────────────────
ses_hop04winb00000000seed1   Windows device pre-existing session  ...
ses_hop04mac0000000000000a1  Hop04 macOS seed session             ...
```

Both the pre-existing Windows session **and** the pulled macOS session are
present: the restore is non-destructive.

```text
PS> opencode --pure export ses_hop04mac0000000000000a1   # destination store
"id": "ses_hop04mac0000000000000a1",
  "id": "msg_hop04mac000000000user1",
    "text": "HOP04-USER-BODY: summarise the demo workspace",
  "id": "msg_hop04mac000000000asst1",
    "parentID": "msg_hop04mac000000000user1",
    "text": "HOP04-ASSISTANT-BODY: the demo workspace has a README.",
```

Both messages and both parts came back with their bodies intact, and the
assistant message still names the user message as its parent — the message/part
fidelity T5 requires, confirmed by the vendor's own export (the same read path
`opencode --session <id>` resumes from).

## 4. Cross-OS path remap, verified in the vendor store

The pulled session's workspace directory was read straight back from the
destination `opencode.db`:

```text
ses_hop04mac0000000000000a1 => C:\Users\admin\code\demo
ses_hop04winb00000000seed1 => D:/Projects/hop-oc/work
```

The macOS session's `${HOME}/code/demo` denormalised onto this device's home as
`C:\Users\admin\code\demo`, with Windows separators forced by the adapter's
declared platform rather than by the host's `filepath` — the platform-free
normalization the project requires. The pre-existing Windows session's path was
left untouched.

The assistant message body's `path.cwd`/`path.root` remained
`/Users/harjjotsinghh/…/reinstate-worktrees/04`, verbatim: that absolute path
belonged to no configured root or home token on the source device, so it was
correctly left as-is rather than having its separators rewritten into a path
that exists nowhere.

## 5. Faithful, WAL-aware restore backup

The pre-restore backup captured the whole store, including the write-ahead
sidecars, so rows the vendor had committed only to the log are recoverable:

```text
D:\Projects\hop-reinhome\backups\20260823T045735.635745200Z-opencode.db-store\
  opencode.db
  opencode.db-shm
  opencode.db-wal
```

## 6. Encryption, conflicts, and forks

The encrypted transport (age envelope, payload-hash verification, manifest CAS)
is the same code every synced agent uses; this journey drove it end to end with
`rein push` on one OS and `rein pull` on the other over a shared backend.
Conflict-fork safety (a fork landing beside the original with its own derived
ids, idempotent on re-restore) and the fingerprint-guarded atomic rename are
covered by the adapter unit tests, which pass natively on this device
(`go test ./internal/adapter/opencode/...` → `ok`).

## 7. Scope this report does not claim

- It did not drive two physically separate machines over a real remote BYO
  bucket; the encrypted push/pull transport is exercised here over a local
  disk-backed `memory` backend copied between the two devices, and the
  cross-OS vendor-store extract/restore ends are exercised on real vendor data
  on both platforms.
- No real store, credential, or account token was touched: every store lived
  under a throwaway `XDG_DATA_HOME`, and the `credential`/`account` tables are
  never opened by the export path.
