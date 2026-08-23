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

## 4. Encryption, conflicts, and forks

The encrypted transport (age envelope, payload-hash verification, manifest CAS)
is the same code every synced agent uses and is covered by `internal/sync`
tests; this journey exercised the OpenCode-specific extract/apply ends of it.
Conflict-fork safety — a fork landing beside the original with its own derived
message/part ids, idempotent on re-restore — is covered by `TestForkKeepBoth`
and `TestForkIsIdempotent`. Live-store safety is a fingerprint-before/after
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
