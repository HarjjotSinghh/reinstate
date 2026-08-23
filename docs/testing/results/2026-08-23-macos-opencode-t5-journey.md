# OpenCode T5 encrypted-sync journey — macOS

`AGENT-TIER-JOURNEY-V1`

This is a single-agent tier-promotion journey, not a release candidate report.
It records the physical evidence gathered on one device for promoting OpenCode
from **T4** to **T5** under
[../../agent-support-tiers.md](../../agent-support-tiers.md). It is immutable
once merged; corrections are appended as a new report.

Nothing here was run against the operator's real OpenCode store. A session was
created **by the vendor itself** (`opencode import`) into a throwaway
`$XDG_DATA_HOME/opencode`, which is the root variable the OpenCode descriptor
declares for exactly this purpose.

## Verdict

- **Device verdict:** `PASS`
- **Platform covered:** `macos-arm64` only
- **Native Windows:** `NOT TESTED HERE` — see
  [2026-08-23-windows-opencode-t5.md](2026-08-23-windows-opencode-t5.md). This
  report does not, on its own, complete the T5 evidence gate, which requires
  macOS **and** native Windows.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` |
| Device | `macos-arm64` |
| OS/version/build | `macOS 26.5 (Darwin 25.5.0)` |
| CPU architecture | `arm64`, native process |
| Go toolchain | `go1.25.13` declared, `go1.26.5` installed |
| Vendor | OpenCode, single native Mach-O executable |
| Vendor version measured | `1.18.21` |
| Vendor data root | throwaway `$XDG_DATA_HOME/opencode` |

## 2. What T5 adds over T4

T4 already starts a new destination session and reconciles its id. T5 is the
first rung at which Reinstate **writes a session back into the vendor's own
store** from an encrypted snapshot, with cross-device path remapping, backup,
and conflict-fork safety. The adapter is
`internal/adapter/opencode` (`Detect`, `Discover`, `PlanExport`, `Export`,
`PlanRestore`, `Restore`, `Exclusions`, plus `SessionRevision`).

Because OpenCode keeps every session in one embedded SQLite database rather than
a file per session, the synced unit is a portable, deterministic JSON document
extracted from the `session`, `project`, `message` and `part` tables. The
`credential` and `account` tables are never opened — proven by
`TestExportRestoreRoundTrip`, which fails if a credential value appears in an
export.

## 3. Round-trip observed on this device

1. The vendor created a session in the source throwaway root
   (`opencode import`, verified with `opencode session list`).
2. The adapter's `PlanExport` + `Export` produced a 4608-byte snapshot archive
   whose paths were normalised to `${HOME}` / `${REPO:…}` tokens.
3. A second, independent OpenCode store (its own `$XDG_DATA_HOME/opencode`,
   initialised by the vendor) received the snapshot through the adapter's
   `PlanRestore` + `Restore`. The write went to a checkpointed working copy and
   was renamed over the destination; the vendor's directory was never written to
   out of band.
4. The adapter re-read the restored session: **2 messages, 2 parts**, workspace
   directory denormalised onto this device.
5. **The vendor binary itself** then listed the restored session from the
   destination store:

   ```text
   Session ID                    Title                               Updated
   ─────────────────────────────────────────────────────────────────────────
   ses_synthetic000000000000001  Synthetic OpenCode fixture session  …
   ```

   Native visibility after a Reinstate restore — not merely a Reinstate read of
   its own write.

### 3a. Resume fidelity, re-run after review (2026-08-23, same device)

A listing proves the session row, not its body. The leg was re-run against two
fresh vendor-initialised throwaway stores with a session whose bodies carry
unique markers, and the **vendor's own export** of the restored session in the
destination store (`opencode --pure export <id>`, the same read path
`opencode --session <id>` resumes from) was captured. Both messages and both
parts came back with their bodies intact:

```text
$ opencode --pure export ses_synthetic00000000000t5fid     # destination store
"id": "ses_synthetic00000000000t5fid",
  "role": "user",
  "id": "msg_synthetic0000000000user1",
    "text": "SYNTHETIC-USER-BODY: summarise the demo workspace layout",
    "id": "prt_synthetic0000000000user1",
  "parentID": "msg_synthetic0000000000user1",
  "role": "assistant",
  "id": "msg_synthetic0000000000asst1",
    "text": "SYNTHETIC-ASSISTANT-BODY: the demo workspace has a single README.",
    "id": "prt_synthetic0000000000asst1",
```

(`grep` of the export over `id`/`role`/`parentID`/`text` keys; the full
document also carried the assistant `path.cwd`/`path.root`, denormalised back
onto this device.) The assistant message's `parentID` still names the user
message and the vendor re-attached each part to its message, which is the
message/part fidelity T5 item 6 asks for.

Two further measurements from this re-run:

- **Revision across a pull.** The first re-run showed the restored session's
  `SessionRevision` differing from the source's, because the destination keeps
  its own `project` row timestamps (the upsert deliberately does not overwrite
  them) and those were hashed. The revision now excludes project timestamps
  (`documentRevision`); the re-run reported `revision-matches=true`, pinned by
  `TestSessionRevisionIgnoresDestinationProjectTimes`. Without this every
  pulled session would have looked locally edited on the next push.
- **Row shape on 1.18.21.** The vendor strips `id`, `sessionID` and
  `messageID` out of the `message.data` / `part.data` blobs on write (identity
  lives in the columns) and re-attaches them on export. The committed seeds
  match that shape, so the fork id-remap has no stale `data.id` to miss; it
  still rewrites those keys if a future build keeps them.
- The restored `opencode.db` keeps the vendor's file mode (`0644` here) rather
  than the working copy's `0600`.

No real store was touched: both stores lived under scratch `XDG_DATA_HOME`
roots with `XDG_CONFIG_HOME` and `HOME` pointed at scratch directories, and
`--pure` kept plugins out.

## 4. Encryption, conflicts, and forks

The encrypted transport (age envelope, payload-hash verification, manifest CAS)
is the same code every synced agent uses and is covered by `internal/sync`
tests; this journey exercised the OpenCode-specific extract/apply ends of it.
Conflict-fork safety — a fork landing beside the original with its own derived
message/part ids, its assistant `parentID` pointing at the fork's own user
message rather than the original session's, idempotent on re-restore — is
covered by `TestForkKeepBoth` and `TestForkIsIdempotent`. Live-store safety is a fingerprint-before/after
guard around the atomic rename in `Restore`.

## 5. Cross-OS remapping

The macOS↔Windows path remapping is exercised by `TestCrossOSRemapping` against
the committed `testdata/adapters/opencode/{macos,windows}` seeds: a macOS export
restores onto a Windows-shaped store with `C:\Users\…` separators, and the
device-independent session revision matches across both shapes. The **physical**
Windows leg is recorded separately and is still outstanding.

## 6. Scope this report does not claim

- It is one device. The T5 gate needs native Windows too.
- It did not drive two physical machines over a real BYO bucket; the encrypted
  push/pull transport is evidenced by unit tests, and the vendor-store
  extract/restore ends are evidenced here on real vendor data.
