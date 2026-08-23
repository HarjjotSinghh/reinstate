# OpenCode T5 journey — native Windows x64, 2026-08-23

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

Counterpart to [2026-08-23-macos-opencode-t5-journey.md](2026-08-23-macos-opencode-t5-journey.md).
Together they are the evidence for OpenCode's T5 rung; this half is the native
Windows leg.

## Verdict

- **Windows journey:** `PENDING` — not yet collected on a native Windows host.
- **Tier claim:** none. OpenCode's encrypted-sync adapter is code-complete and
  macOS-verified, but the catalog keeps OpenCode at **T4** and does not wire the
  adapter until this leg is recorded as `PASS`. The T5 gate requires both
  platforms; half of it is not a tier.

This report exists so the gap is stated rather than implied. It will be
replaced with a `PASS` record once the round-trip is run on a native Windows
x64 device; that is the change that flips the descriptor to T5 and wires
`NewSyncAdapter`.

## 1. What is already established for Windows path shapes

The Windows path shape is not untested — only the physical device run is
outstanding:

- `testdata/adapters/opencode/windows/store.sql` is a deterministic synthetic
  store seeded with `C:\Users\fixture-user\code\demo` paths.
- `TestCrossOSRemapping` restores a macOS-exported snapshot onto that
  Windows-shaped store and asserts the workspace directory and the embedded
  assistant `path.cwd`/`path.root` land as `C:\Users\fixture-user\code\demo`,
  with backslash separators forced by the adapter's declared `GOOS` rather than
  by the host's `filepath` — the platform-free normalization the project
  requires so a Windows path is correct regardless of which OS ran the restore.
- `TestSessionRevisionStableAndDeviceIndependent` confirms the same session has
  the same content revision on both path shapes, which is what lets the sync
  engine detect a genuine edit instead of a separator difference.

## 2. What the native Windows run must still record

1. `opencode --version` grammar on native Windows (expected identical bare
   `MAJOR.MINOR.PATCH`; the range widens only when measured on-device).
2. The embedded store under `%USERPROFILE%\.local\share\opencode` — the XDG
   layout, **not** `%LOCALAPPDATA%` — is discovered, exported, and restored.
3. A push on one OS and a pull on the other, then `opencode --session <id>`
   resuming natively on Windows — the cross-device leg the macOS report could
   not perform alone.

## 3. Why it is not in this report

No native Windows x64 host was reachable from the worktree that produced this
change. The Windows acceptance bench is coordinated separately; this file
records the gate honestly until that run exists rather than presenting a macOS
result as if it covered both platforms.
