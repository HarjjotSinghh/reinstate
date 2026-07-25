# Phase 1 acceptance — Device A (macOS) result report

Executed against
[`docs/testing/phase-1-mac-windows-acceptance.md`](../phase-1-mac-windows-acceptance.md)
by an agent operator (Claude Code) on the MacBook.

**Verdict: Device A is BLOCKED. Phase 1 is not signed off.**

Every step that can be executed without a human-entered secret and without the
Windows device passed. Sections 8–17 cannot pass on this machine at all —
not because credentials were missing, but because the installed agent CLIs are
newer than the versions RC2 recognizes, and the adapters therefore refuse
export and restore. That is a release-blocking finding, documented in F1 below.

---

## 1. Test metadata

| Field | Value |
| ----- | ----- |
| Date/time (UTC) | 2026-07-25T22:01Z – 2026-07-25T22:20Z |
| Operator | Claude Code agent, Device A |
| Mac model / macOS version | macOS 26.5.2 (build 25F84) |
| Mac architecture | `arm64` |
| Windows edition/build | not tested here — Device B report |
| Claude Code version | `2.1.220` |
| Codex CLI version | `0.145.0` |
| Git version | `2.51.0` |
| Reinstate version | `0.1.0-rc.2` (commit `57c06f2`, built 2026-07-25T17:36:52Z) |
| GitHub PR / merge commit | PR #10, merged `e07a59b` |
| Device A profile ID | **not created** — real `rein init` requires human-entered secrets (see §5) |
| Claude test session ID | `2b3b7185-8fcc-41a9-9afe-46639eb20b1c` |
| Codex test session ID | `019f9b4d-d8d2-79e1-9e23-810786676f5a` |
| Isolated home | `~/.reinstate-phase1-acceptance` |
| Disposable project | `~/Projects/reinstate-phase1-acceptance` |

No secret, credential, passphrase, or transcript content appears in this report.

---

## 2. What an agent operator may and may not do

Recorded explicitly so the sign-off is honest about coverage.

Executed by the agent:

- guide §3 prerequisites, §4 project creation, §5 macOS installer, §6 pre-init
  failure, §7 source sessions, §18 automated integrity gates;
- extra checks not in the guide (§6 of this report).

Not executable by an agent, deferred to the human operator:

- `rein init` with the real R2/S3 endpoint, bucket, access key, secret key, and
  encryption passphrase — the runbook forbids routing these through an AI
  prompt, argument, or ordinary environment variable, and `rein` reads them
  only from hidden interactive prompts;
- bucket inspection through the storage provider UI (§10);
- visual confirmation of resumed transcript markers (§13, §15, §17);
- everything on Device B.

---

## 3. Section results

| Guide section | Result | Exit code | Evidence |
| ------------- | ------ | --------- | -------- |
| §3 Prerequisites | **PASS with warning** | — | `arm64`; both agent CLIs run; versions differ from RC2 evidence (F1) |
| §4 Disposable mapped project | **PASS** | 0 | absolute, writable, git-initialized |
| §5 macOS installer — HTTP + install | **PASS** | 0 | HTTP/2 200, both checksum layers ok, `0.1.0-rc.2` |
| §5 macOS installer — idempotency/PATH | **PASS** | 0 | second run is a no-op; exactly 1 PATH entry |
| §6 Pre-init failure is honest | **PASS** | 3 | `config: config missing` |
| §7 Two source sessions on the Mac | **PASS** | 0 | both session IDs discoverable via `rein list` |
| §8 Claude setup prompt on Device A | **BLOCKED / would FAIL** | 1 | secrets are human-only; and export refuses under UNTESTED (F1) |
| §9 Push the Mac Codex session | **BLOCKED / would FAIL** | 1 | same as §8 |
| §10 Ciphertext-only remote storage | **BLOCKED** | — | requires a real bucket and provider UI |
| §11–§17 | **BLOCKED** | — | require Device B and a completed §8 |
| §18 Automated integrity gates | **PASS** | — | all required checks green on `e07a59b` |

### §3 Prerequisites

```
macOS 26.5.2 (25F84), arm64
claude  2.1.220
codex   0.145.0
git     2.51.0
```

RC2 compatibility evidence recognizes Claude Code `2.1.219` and Codex CLI
`0.133.0`. Both installed CLIs are newer. Per guide §1 and §3 this permits
read-only checks only, and forbids calling Phase 1 complete. See F1.

### §5 Live public installer (macOS)

`HEAD https://reinstate.dev/install.sh`

```
HTTP/2 200
content-type: text/plain; charset=utf-8
content-disposition: inline; filename="install.sh"
cache-control: public, max-age=0, must-revalidate
x-content-type-options: nosniff
content-length: 2398
```

`HEAD https://reinstate.dev/install.ps1` returned the same header shape with
`content-length: 4891`.

First run of `curl -fsSL https://reinstate.dev/install.sh | sh`:

```
installer checksum ok
checksum ok
Installed reinstate v0.1.0-rc.2 → ~/.local/bin/reinstate
Installed rein alias → ~/.local/bin/rein
```

- both checksum layers verified (pinned bootstrap hash, then release checksums);
- `rein` and `reinstate` resolve under `~/.local/bin`; `rein` is a symlink;
- `rein version --json` → `{"version": "0.1.0-rc.2", "commit": "57c06f2"}`;
- exit code `0`.

Second run reported `Reinstate v0.1.0-rc.2 is already installed` and made no
changes.

PATH safety was verified two ways:

1. In the acceptance shell, `~/.local/bin` was already on `PATH`, so the
   installer correctly left every shell profile untouched — verified by
   unchanged mtimes on `.zshrc`, `.zprofile`, `.bashrc`, `.profile` and by the
   absence of any `Reinstate` marker in them.
2. Because that leaves the PATH-append branch unexercised, it was run in an
   isolated throwaway `HOME` with `~/.local/bin` absent from `PATH`. First run
   appended one block; second run appended nothing. Final count of
   `.local/bin` lines in the generated `.zshrc`: **1**.

### §6 Pre-init failure

```
rein setup check   → exit 3
summary: 1 check(s) failed
- [fail] config: config missing
- [ok]   device: darwin-arm64
- [warn] agent.claude: layout/version untested; writes blocked
- [warn] agent.codex: layout/version untested; writes blocked
- [ok]   keyring: OS keyring provider reachable
```

Device detection is correct and does not claim an unsupported platform.
Pre-init exit codes are consistent across `rein status`, `rein push --dry-run`,
and `rein pull --dry-run` (all exit `3`, `config.toml: no such file or
directory`). One inconsistency: `rein conflicts list` exits `0` with empty
output instead of `3` (F3).

### §7 Source sessions

Two harmless marker sessions were created from the disposable project:

| Agent | Session ID | Marker delivered |
| ----- | ---------- | ---------------- |
| Claude Code | `2b3b7185-8fcc-41a9-9afe-46639eb20b1c` | yes — assistant replied with the `A1` marker |
| Codex CLI | `019f9b4d-d8d2-79e1-9e23-810786676f5a` | yes — assistant replied with the `A1` marker |

Both are discoverable through `rein list` and are mapped to the acceptance
project path. No transcript file was opened to identify them; identification
used `rein list` metadata plus directory listings only.

An earlier headless `claude -p` attempt inside a stripped sandbox environment
reported `Not logged in · Please run /login` but still created a session file
(`8ad99d38-…`). It is a stray artifact in the disposable project, not a
Reinstate defect (F5).

### §18 Automated integrity gates

Check runs on merge commit `e07a59b`:

| Check | Conclusion |
| ----- | ---------- |
| Test (ubuntu-latest) | success |
| Test (macos-latest) | success |
| Test (windows-latest) | success |
| Website | success |
| Lint | success |
| Security | success |
| Secret scan | success |
| CodeQL | success |
| Dependabot | success |
| Workflow permission and pin review | success |
| Dependency review | skipped (pull-request-only gate) |

Native Windows bootstrap execution is genuinely covered, not assumed:
`internal/doctest/bootstrap_install_contract_test.go` executes
`website/public/install.ps1` under `powershell.exe` on `windows-latest`
(twice, asserting exactly one PATH update and hash-mismatch refusal), and the
POSIX equivalent runs on macOS/Linux. The Website job additionally proves
byte-for-byte inclusion in the Astro output and the absence of a `latest` tag.

---

## 4. Supply-chain verification of the live routes

Beyond the guide's requirements:

- `https://reinstate.dev/install.sh` is **byte-identical** to
  `website/public/install.sh` at `origin/main`;
- `https://reinstate.dev/install.ps1` is **byte-identical** to
  `website/public/install.ps1` at `origin/main`;
- neither served script contains the string `latest`;
- both pin `v0.1.0-rc.2` explicitly.

---

## 5. Why sections 8–17 are blocked twice over

**Blocker A — human-only secrets.** `rein init` reads the storage endpoint,
bucket, access key, secret key, and encryption passphrase from hidden prompts.
The runbook forbids handing those to an agent. This is correct design, not a
defect.

**Blocker B — compatibility gate (the real problem).** Even with valid
credentials, RC2 refuses to export sessions from the installed agent versions.
Proven offline against a throwaway local S3-compatible stub (no real bucket, no
real credentials, synthetic passphrase, everything deleted afterwards):

```
rein init            → profile created, "all checks passed", exit 0
rein setup check     → exit 0, but: agent.claude UNTESTED, agent.codex UNTESTED
rein push --agent claude --session <A1> --dry-run
                     → exit 1: "claude compatibility UNTESTED refuses export"
rein status          → remote revision: (0 sessions)
```

So §8's mandatory results (`SUPPORTED` adapters, one pushed snapshot) cannot be
met on this Mac today, and every downstream section depends on §8.

A related positive result: when the storage probe cannot reach the endpoint,
`rein init` fails closed and writes no `config.toml`.

---

## 6. Findings

### F1 — BLOCKER: RC2's verified agent versions are already stale

`internal/adapter/claude/claude.go` pins `verifiedClaudeVersion = "2.1.219"` and
`internal/adapter/codex/codex.go` pins `verifiedCodexVersion = "0.133.0"`.
Installed: `2.1.220` and `0.145.0`. Any other version resolves to `UNTESTED`,
and `PlanExport` refuses outright while `CanRestore` refuses without an
override that Phase 1 deliberately does not expose
(`docs/compatibility.md`: "Phase 1 has no unsafe compatibility override").

Impact: on an up-to-date developer machine, RC2 can discover sessions but can
neither push nor pull them. That is the entire product.

Options, in order of preference:

1. re-run the compatibility evidence against Claude Code `2.1.220` and Codex
   CLI `0.145.0`, widen the accepted range (a floor plus a tested ceiling
   rather than one exact string), update `docs/compatibility.md`, and cut
   `v0.1.0-rc.3`;
2. add a documented, explicit, non-default override (for example
   `--allow-untested`) with a loud warning, and record it in the runbook; or
3. pin the acceptance devices to the exact recognized agent versions and accept
   that shipping RC2 breaks on the next agent release.

Option 1 is the honest one. Exact-string version matching will keep breaking
every time either vendor ships a patch.

### F2 — `setup check` reports "all checks passed" while writes are blocked

After a successful init, `rein setup check` exits `0` and prints
`summary: all checks passed`, while simultaneously reporting
`agent.claude: layout/version untested; writes blocked`. The runbook's §8
mandatory result asks for both "all checks passed" **and** `SUPPORTED`
adapters — a naive operator would mark that row PASS and only discover the
refusal at push time.

Suggested fix: treat "no adapter can write" as a failed setup gate (non-zero
exit), or print an explicit `not ready to push/pull` line in the summary.

### F3 — inconsistent pre-init exit code for `conflicts list`

`rein status`, `push --dry-run`, `pull --dry-run`, and `setup check` all exit
`3` with `config missing`. `rein conflicts list` exits `0` and prints nothing.
An operator scripting the runbook cannot distinguish "no conflicts" from "no
config".

### F4 — `rein list --agent claude` surfaces subagent sessions as top-level rows

Output includes many `agent-<hex>` rows whose project column is
`<project>/<session-uuid>/subagents`. They are real files, but they inflate the
list and make manual scope selection error-prone, which matters because the
runbook forbids `--all`. Consider hiding them behind a flag or labelling them.

### F5 — informational: stray session from headless `claude -p`

Headless invocation in a stripped sandbox environment reported
`Not logged in · Please run /login` yet still wrote a session file. Not a
Reinstate defect; recorded so the extra ID in the disposable project is not
mistaken for a sync artifact.

### F6 — informational: open high-severity Dependabot alert

`path-to-regexp`, `GHSA-9wv6-86v2-598j`, high, still open. It is a website
dependency and is not reached by the Go vulnerability scan, but it should be
closed before a public release rather than carried indefinitely.

---

## 7. Device A rows of the final sign-off checklist

| Gate | Result | Evidence |
| ---- | ------ | -------- |
| `install.sh` returns 200 and installs RC2 on Mac | **PASS** | §3 of this report |
| `install.ps1` returns 200 and installs RC2 on Windows | not tested here | Device B report; route returns 200 |
| Both installers are idempotent and PATH-safe | **PASS (macOS)** | second run no-op; 1 PATH entry |
| Pre-init missing-config failure is accurate | **PASS** | exit 3, `config missing` |
| Post-init setup check and self-test pass on both devices | **FAIL (macOS)** | self-test ok, but adapters `UNTESTED`; F1, F2 |
| Claude setup prompt completes on the Mac | **BLOCKED** | needs human secrets; blocked by F1 |
| Only two selected test sessions reach the remote manifest | **BLOCKED** | no real push performed |
| Remote manifest/snapshots are ciphertext-only | **BLOCKED** | no real bucket |
| Claude/Codex Mac-to-Windows resume | **BLOCKED** | Device B |
| Claude/Codex Windows-to-Mac resume | **BLOCKED** | Device B |
| Existing Mac targets are backed up before restore | **BLOCKED** | no restore performed |
| Unchanged pushes skip without new snapshots | **BLOCKED** | no push performed |
| Divergence records a conflict without overwrite | **BLOCKED** | Device B |
| `--keep-both` preserves both branches | **BLOCKED** | Device B |
| All required GitHub checks are green | **PASS** | `e07a59b`, 10 success, 1 skipped |

---

## 8. Safe handoff data for the Device B (Windows) operator

Non-secret, shareable:

- canonical project ID: `local/reinstate-phase1-acceptance`
- Claude test session ID: `2b3b7185-8fcc-41a9-9afe-46639eb20b1c`
- Codex test session ID: `019f9b4d-d8d2-79e1-9e23-810786676f5a`
- Reinstate version under test: `0.1.0-rc.2`
- profile ID: **none yet** — the human must run `rein init` on the Mac first,
  then hand the printed `profile_id` to Device B

Never shared: storage keys, encryption passphrase, keyring or Credential
Manager contents, agent auth files, transcript contents.

Device B can independently complete §5 (installer), §6 (pre-init failure), and
the Windows half of §18 right now. Everything after that waits on the Mac
profile ID, and — more importantly — on F1.

---

## 9. What the human operator must do next

1. Decide on F1. Nothing downstream is worth running until adapters report
   `SUPPORTED` on the machines actually being tested.
2. After F1 is fixed and a new RC is published, run `rein init` privately on
   the Mac and record `profile_id` (non-secret) for Device B.
3. Re-run §8–§10 on the Mac, then hand off to Windows for §11–§17.
4. Confirm resumed transcript markers visually — no agent can attest to that.

## 10. Cleanup state

Left in place for diagnosis, as the runbook requires:

- `~/.reinstate-phase1-acceptance` (isolated home; no `config.toml` — init was
  never completed with real credentials)
- `~/Projects/reinstate-phase1-acceptance` (disposable project, 2 marker
  sessions)
- `~/.local/bin/reinstate` and `~/.local/bin/rein` (v0.1.0-rc.2)

Removed during the run: the throwaway PATH-test `HOME`, the synthetic S3 stub
and its in-memory objects, and both synthetic Reinstate homes used for the
offline compatibility proof. No real Claude Code or Codex data was deleted, and
no transcript file was opened.
