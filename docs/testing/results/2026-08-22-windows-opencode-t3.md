# OpenCode T3 journey — native Windows x64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`.** It covers no release candidate, no
acceptance matrix, and no agent other than OpenCode. It is the counterpart to
`2026-08-22-macos-opencode-t3-journey.md`; neither platform is sufficient alone.

## Verdict

- **Windows journey:** `PASS` — 6 of 6 rows, 0 failed.
- **Continuation:** observed, not assumed.
- **Also established here:** the write-ahead-log visibility fix, on the platform
  where it had not previously been exercised.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `windows-amd64`, native — not WSL |
| OS | `Microsoft Windows 11 Pro`, `10.0.26200.0` |
| Vendor | OpenCode `1.18.21`, installed through `bun` |
| Declared range under test | `1.18.21`–`1.18.21` |
| Redirect | `XDG_DATA_HOME` → `C:\accept\roots\ocdata`, with its own credential |
| Workspace | `C:\accept\roots\ws-oc`, a real Git repository |

The host previously carried `1.18.2`, which is a different build from `1.18.21`
and reported `UNTESTED` against this range. It was upgraded for this journey.

## 2. Rows

| Row | Verdict | Evidence |
| --- | ------- | -------- |
| `OS1` | `PASS` | The vendor created its store under the redirected root. This is also the first confirmation on Windows that `RootEnv` plus `RootEnvSuffix` can redirect OpenCode at all; it was previously the one indexed agent that could not be. |
| `OW1` | `PASS` | One session indexed while `opencode.db` held 4096 bytes and `opencode.db-wal` held 836392. |
| `OR6` | `PASS` | Vendor directory entries unchanged across an index pass and a probe. No sidecar was added. |
| `OV1` | `PASS` | `agent.version=1.18.21`, `status=supported`. |
| `OR3` | `PASS` | Launch plan `opencode --session ses_fd672e38…` with `cwd` equal to the recorded workspace. |
| `OR1` | `PASS` | Executing that plan, the vendor recalled the first turn's token. |

## 3. The write-ahead-log row, in full

`OW1` is the row this platform most needed, because OpenCode journals in WAL
mode and does not checkpoint on exit. The store was copied the instant the
vendor process exited, before anything could checkpoint it, and read both ways:

| File | Size |
| ---- | ---- |
| `opencode.db` | 4096 bytes |
| `opencode.db-wal` | 836392 bytes |

| Read mode | Result |
| --------- | ------ |
| `mode=ro&immutable=1` — the shipped behaviour before the fix | **no `session` table at all**, 0 sessions |
| plain `mode=ro` — what the fix's private copy uses | `session` table present, 1 session |

4096 bytes is a database header. Everything the operator had just done was in
the log. An immutable in-place handle ignores that file by definition, so the
session was not merely stale in the index — it was absent, and on a new install
the whole store reads as empty.

## 4. What this journey does not establish

- **T4.** No handoff-destination row was collected. OpenCode's T4 branch is a
  draft and makes no claim this report supports.
- **The process-spawn half of resume.** `OR1` executes the argv Reinstate
  planned and `OR3` proves that argv is Reinstate's own, but the launch was not
  performed by `rein resume` itself, which starts an interactive session.
- **Any other agent.**

## 5. Harness defect corrected before any verdict was recorded

Two runs failed with `Error: Unexpected error  EUNKNOWN: unknown error, read`
before OpenCode logged anything. That reads exactly like an authentication
failure and is not one: `Start-Process` redirects stdout and stderr but leaves
stdin as a non-console handle, and OpenCode's startup read of stdin fails.
Redirecting stdin from an empty file resolves it. The credential was necessary
but not sufficient, and attributing the failure to it alone would have been
wrong.
