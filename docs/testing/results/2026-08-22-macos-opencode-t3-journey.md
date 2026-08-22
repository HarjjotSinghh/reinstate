# OpenCode T3 verified-resume journey — macOS

`AGENT-TIER-JOURNEY-V1`

This is a single-agent tier-promotion journey, not a release candidate report.
It records the physical evidence gathered on one device for promoting OpenCode
from **T2** to **T3** under
[../../agent-support-tiers.md](../../agent-support-tiers.md). It is immutable
once merged; corrections are appended as a new report.

Nothing here was run against the operator's real OpenCode store. Every command
ran with `XDG_DATA_HOME` pointed at a throwaway parent directory, which is the
root variable the OpenCode descriptor declares for exactly this purpose.

## Verdict

- **Device verdict:** `PASS`
- **Platform covered:** `macos-arm64` only
- **Native Windows:** `NOT TESTED HERE` — coordinated separately. This report
  does not, on its own, complete the T3 evidence gate, which requires macOS
  **and** native Windows.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `macos-arm64` |
| OS/version/build | `macOS 26.5.2 (25F84)` |
| CPU architecture | `arm64`, native process |
| Base commit | `f4a3963fee66b426afc93239259ce33b1a0d5dcd` |
| Go toolchain | `go1.25.13` declared, `go1.26.5` installed |
| Vendor | OpenCode, single native Mach-O executable |
| Vendor version measured | `1.18.21` |
| Vendor data root | throwaway `$XDG_DATA_HOME/opencode` |

## 2. Vendor version grammar

`opencode --version` was run through the same sanitized environment the version
probe uses — no `HOME`, no `XDG_DATA_HOME`, working directory set to the volume
root — and answered on stdout with a bare stable version and an empty stderr.

| Observation | Result |
| ----------- | ------ |
| stdout | one line, bare `MAJOR.MINOR.PATCH` |
| stderr | empty |
| exit code | `0` |
| behaviour without `HOME` | unchanged; no data root is created |

`VersionSpec.Min` and `VersionSpec.Max` are both `1.18.21` because that is the
only build measured. The range is deliberately not widened by assumption; it
widens when another build is physically measured.

## 3. Session fixture

A session was created **by the vendor itself**, using `opencode import`, into
the throwaway root. No rows were written by Reinstate or by hand. The vendor
recorded the session with an id, a title, a slug, an opaque 40-hex project id,
and the working directory the import ran in — which is what the index reads and
what resume launches into.

## 4. Journey rows

| Row | Result | Observation |
| --- | ------ | ----------- |
| `J1` list | `PASS` | `rein sessions --agent opencode --json` returned the vendor-written session with `capabilities.resume = true` and `capabilities.fork = true` |
| `J2` layout | `PASS` | `agent.layout` = `match`, `embedded-sqlite-session-store`; the marker is a regular file, not a directory |
| `J3` version | `PASS` | `agent.version` = `match`, actual `1.18.21`, inside the declared range |
| `J4` executable | `PASS` | `agent.executable` = `present` through trusted resolution |
| `J5` resume argv | `PASS` | plan argv `opencode --session <id>`, cwd equal to the vendor-recorded directory |
| `J6` fork argv | `PASS` | plan argv `opencode --session <id> --fork` |
| `J7` vendor accepts resume argv | `PASS` | the real binary started its TUI on that argv under a pty |
| `J8` vendor accepts fork argv | `PASS` | the real binary started its TUI on that argv under a pty |
| `J9` continuation observed | `PASS` | the vendor TUI set its own window title to the imported session's title and rendered the recorded working directory in its footer — the vendor opened the session Reinstate named, not a new one |
| `J10` negative control | `PASS` | a deliberately misspelled option was rejected by the vendor with exit `1` and usage, so J7–J9 are acceptance and not silent tolerance |
| `J11` liveness detects | `PASS` | with that session open in the vendor TUI, `agent.active` = `present`/`warning`, "a running opencode instance is already using this session" |
| `J12` liveness clears | `PASS` | after closing it, `agent.active` = `match`, actual `false` |
| `J13` no sidecar | `PASS` | `-wal` and `-shm` were deleted, then `rein sessions`, `rein resume --dry-run` and `rein search` were run; neither file reappeared beside the vendor store |
| `J14` root variable | `PASS` | every row above ran with `XDG_DATA_HOME` pointed at a throwaway parent; the real store was never opened |

## 5. Defects found and fixed during this journey

Three defects were found by running the journey, and each is fixed in the same
change as the promotion. J2, J11 and J14 all failed before the fixes.

1. **The layout probe required a directory marker.** An agent whose sessions
   live in one embedded database reported `layout_recognized = false` even with
   the store present, so resume was refused. `internal/agentcheck` now accepts a
   real regular file as a marker, and still rejects symlinks and every other
   file kind.

2. **The root variable's suffix was dropped on the resume path.**
   `StorageSpec.RootEnvSuffix` was honoured by the storage probe but not by the
   version and layout probe, so with `XDG_DATA_HOME` set the marker was looked
   for one directory above the store. That is the single case the variable
   exists to serve.

3. **The liveness check could not see an agent Reinstate launched itself.**
   This one is not OpenCode-specific. `ps -axo pid=,comm=,args=` fixes the
   `comm` column at sixteen characters, so a process started by absolute path is
   reported with a truncated path as its image — measured here as
   `/Users/harjjotsi` for a 43-character path. Reinstate launches vendors by
   their verified absolute path, so `agent.active` was answering "nothing is
   running" for a session that was open at that moment, for every agent on
   macOS, not only OpenCode. Matching now also considers `argv[0]`.

## 6. Not covered by this report

- **Native Windows.** No row above was run there. `ps` column truncation is a
  macOS behaviour; the Windows process listing takes a different path and needs
  its own measurement, as does the file-marker layout probe.
- **An out-of-range vendor version producing exit `5`.** Only one OpenCode build
  exists on this device, and installing a second would have modified the
  operator's own installation. That path is covered by unit tests, not by this
  device.
- **A session carrying real model turns.** The fixture session was created by
  the vendor's own importer and has no assistant turns, because generating them
  would require provider credentials. Resume was observed opening the correct
  session; it was not observed continuing a conversation in progress.
