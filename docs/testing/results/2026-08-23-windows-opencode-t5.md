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
| Branch | `hop/04-opencode-t5`; sections 1–7 recorded @ `1c21cbc`, the pull leg re-driven @ `7145168` (the shipped behaviour; section 8 below). This record itself lands in the docs-only commit that follows `7145168`. |
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

## 8. Pull leg re-run against the shipped commit (`7145168`)

Commits after `1c21cbc` changed what Restore writes into the vendor store
(`rewriteJSONPaths` first kept unchanged blobs byte-for-byte, which stored the
export document's indented form; `7145168` compacts them back to the vendor's
own shape). The Windows pull leg was therefore re-driven against `7145168`
exactly, on the same bench, with the same macOS-pushed snapshot.

Harness checks first, because a vacuous pass is worse than none:

- The bench repo had to be detached and force-fetched: the branch was rebased
  onto `hop/main`, so a plain `git fetch` of the bundle was rejected
  (`non-fast-forward`) and the first build in this pass was still `1c21cbc`.
  `git rev-parse --short HEAD` → `71451686`, `git status --short` clean,
  `go build -o D:\Projects\reinstate\rein.exe ./cmd/reinstate` exit 0.
- `go test ./internal/adapter/opencode/... -run 'TestExportRestoreRoundTrip|TestRewriteJSONPathsKeepsVendorBytesWhenUnchanged|TestCrossOSRemapping' -v`
  → all `PASS` natively (the first two are the new compact-row assertions).
- The passphrase launcher (anonymous inheritable pipe →
  `REINSTATE_PASSPHRASE_FD`, passphrase read from a local file that is never
  committed) was invoked with `powershell -File`, which hands `a,b` to a
  `[string[]]` parameter as the single string `"a,b"`; `rein` exited 2 with no
  output until the launcher split its arguments. Verified fixed with
  `rein status --json` listing the remote session before any pull was run.
- The destination store was reset to the vendor-initialised pre-pull copy
  (`backups\20260823T045735…-opencode.db-store\opencode.db{,-wal,-shm}`
  copied back over `D:\Projects\hop-oc\xdg\opencode\`), the
  `sessions` map in `state.json` was emptied (original kept as
  `state.json.1c21cbc`), and the reset was proven, not assumed:
  `opencode --pure session list` showed only `ses_hop04winb00000000seed1`,
  and a direct SQLite read found **no** `message`/`part` rows for
  `ses_hop04mac0000000000000a1`.

```text
PS> rein pull --agent opencode --session ses_hop04mac0000000000000a1 --dry-run --json
{ "dry_run": true, "plans": [ { "agent": "opencode",
    "session_id": "ses_hop04mac0000000000000a1",
    "snapshot_id": "cabc51ee-ba5d-4d7a-b05a-8c36e71e07f9",
    "destinations": [ "D:\\Projects\\hop-oc\\xdg\\opencode\\opencode.db" ],
    "backup_root": "D:\\Projects\\hop-reinhome\\backups" } ], "pulled": 1 }
exit 0

PS> rein pull --agent opencode --session ses_hop04mac0000000000000a1 --json
{ "dry_run": false, ... same plan ..., "pulled": 1 }
exit 0
```

Rows read straight out of `opencode.db` **before** the vendor touched the
store (a throwaway Go probe using the same `modernc.org/sqlite` driver; it
prints each blob, its byte length, newline count, and whether it equals its own
`json.Compact` form):

```text
session ses_hop04mac0000000000000a1 directory=C:\Users\admin\code\demo
session ses_hop04winb00000000seed1 directory=D:/Projects/hop-oc/work
message msg_hop04mac000000000asst1 bytes=437 newlines=0 compact=true
  {"agent":"build","cost":0,"mode":"build","modelID":"fixture-model","parentID":"msg_hop04mac000000000user1","path":{"cwd":"/Users/harjjotsinghh/Documents/Projects/reinstate-worktrees/04","root":"/Users/harjjotsinghh/Documents/Projects/reinstate-worktrees/04"},"providerID":"synthetic","role":"assistant","time":{"completed":1755950002000,"created":1755950001000},"tokens":{"cache":{"read":0,"write":0},"input":1,"output":1,"reasoning":0}}
message msg_hop04mac000000000user1 bytes=125 newlines=0 compact=true
  {"agent":"build","model":{"modelID":"fixture-model","providerID":"synthetic"},"role":"user","time":{"created":1755950000000}}
part prt_hop04mac000000000asst1 bytes=79 newlines=0 compact=true
  {"text":"HOP04-ASSISTANT-BODY: the demo workspace has a README.","type":"text"}
part prt_hop04mac000000000user1 bytes=70 newlines=0 compact=true
  {"text":"HOP04-USER-BODY: summarise the demo workspace","type":"text"}
```

Every stored blob is compact (`newlines=0`, `compact=true`), identity fields
live in the columns as the vendor writes them, and the untokenised
`path.cwd`/`path.root` came through as-is with `/` separators.

The vendor's own read path on the pulled store:

```text
PS> opencode --pure session list
Session ID                   Title                                Updated
─────────────────────────────────────────────────────────────────────────
ses_hop04winb00000000seed1   Windows device pre-existing session  8:10 PM · 8/23/2025
ses_hop04mac0000000000000a1  Hop04 macOS seed session             5:24 PM · 8/23/2025

PS> opencode --pure export ses_hop04mac0000000000000a1      # exit 0
"id": "ses_hop04mac0000000000000a1",
"directory": "C:\\Users\\admin\\code\\demo",
  "id": "msg_hop04mac000000000user1",
    "text": "HOP04-USER-BODY: summarise the demo workspace",
  "id": "msg_hop04mac000000000asst1",
    "parentID": "msg_hop04mac000000000user1",
    "text": "HOP04-ASSISTANT-BODY: the demo workspace has a README.",
```

Both sessions listed; both messages and both parts exported with bodies intact
and the assistant's `parentID` pointing at the user message. After the vendor
opened the store it re-created its own `-wal`/`-shm` sidecars (16512 / 32768
bytes), i.e. it treated the merged database as a healthy one.

Backup: `D:\Projects\hop-reinhome\backups\20260823T052717.589447600Z-opencode.db-store\opencode.db`
(no sidecars this time, because the vendor's earlier `session list` had
checkpointed and removed them before the pull). The same probe on that backup
shows only `ses_hop04winb00000000seed1`, so it is a faithful pre-pull copy.

Bench facts for this pass: `Microsoft Windows 11 Pro 10.0.26200`, OpenCode
`1.18.21`, `go version go1.26.1 windows/amd64` (the bench default; the build
was not pinned to 1.25 this time). Nothing outside `D:\Projects` and the
throwaway `XDG_DATA_HOME` was touched; the probe source was removed from the
bench checkout afterwards and the checkout is clean.

