# Releasing

How maintainers cut a **Reinstate** release.

## Versioning

- **Semantic Versioning**: `MAJOR.MINOR.PATCH`
- Pre-1.0: breaking changes allowed in `MINOR` with clear CHANGELOG entries
- Git tags are prefixed: `v0.1.0`

## Preconditions

- [ ] `main` is green on CI
- [ ] CHANGELOG `[Unreleased]` section is accurate
- [ ] No open P0 security issues
- [ ] For a stable release, fresh Apple Silicon macOS and native Windows x64
      acceptance rows pass and the separate stable promotion decision is
      recorded
- [ ] For a release candidate, prior candidate failures are recorded and every
      known release blocker has a regression test or an explicit unresolved
      disposition
- [ ] Claude Code and Codex exact versions/layouts are recorded in compatibility docs
- [ ] Wrong-passphrase, tamper, backup, rollback, conflict, and installer tests pass
- [ ] Snapshot archives, source archive, checksums, and SBOMs were inspected
- [ ] Builds and vulnerability scans use the pinned Go 1.25.13 toolchain
- [ ] For `v0.6.0`, the stable acceptance row is native Windows x64 under the
      [Windows-first waiver](#v060-windows-first-waiver) instead of the
      dual-platform row above, with the macOS rows recorded as deferred
      rather than passed

### Supported platform boundary

The maintainer-approved `v0.2.0` reconciliation in
`docs/testing/results/2026-08-02-macos-phase2-V020RC2.md` permits stable
publication with verified support limited to Apple Silicon macOS
(`darwin/arm64`) and native Windows x64 (`windows/amd64`). Those remain the
supported mandatory platforms for Phase 3 and Phase 4 candidate and stable
acceptance.

Intel macOS and Linux/WSL2 artifacts are unsupported/unverified optional
evidence: they may be built, checksummed, SBOM-covered, and attested, but their
absence or failure does not block a candidate or stable promotion. Never
describe them as physically certified or supported. Issues #97 and #98 track
that optional physical evidence.

### v0.6.0 Windows-first waiver

[ADR 0005](docs/adr/0005-v0.6.0-scope-and-windows-first-acceptance.md) permits
`v0.6.0-rc.N` candidate and stable acceptance on native Windows x64
(`windows/amd64`, never WSL) alone while the Apple Silicon macOS host is in
repair. Every row the prior contracts require on macOS is carried as
**deferred**, listed in
[#403](https://github.com/HarjjotSinghh/reinstate/issues/403), and re-run
against the same stable tag once that hardware returns.
No document, data file, or release note may state that `v0.6.0` was verified
on macOS; if that later macOS run fails, the fix ships as `v0.6.1`, not by
editing the record. This mirrors the
`v0.2.0` reconciliation above: the boundary is narrowed and named, not hidden.
It governs `v0.6.0` acceptance only — the dual-platform requirement above is
unchanged for every other release.

### v0.3.0-rc.1 candidate evidence

The committed
[`v0.3.0-rc.1` dispatch](docs/testing/v0.3.0-rc.1-agent-verification-prompts.md)
required two tagged-artifact reports: Apple Silicon macOS and native Windows
x64. macOS passed (32/32); native Windows failed (9 PASS / 23 FAIL) with six
root blockers, including Windows executable trust for extensionless `codex`
and host/tooling gaps. RC1 does **not** authorize stable `v0.3.0`.

### v0.3.0-rc.3 candidate evidence

The Phase 3 candidate used
[`v0.3.0-rc.3` dispatch](docs/testing/v0.3.0-rc.3-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Native
Windows x64 failed on PowerShell 5.1 staging and human-output privacy; do not
promote stable from RC3.

### v0.3.0-rc.4 candidate evidence

The Windows-first corrective candidate used
[`v0.3.0-rc.4` dispatch](docs/testing/v0.3.0-rc.4-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Its signed
tag workflow failed during Ubuntu PowerShell artifact verification before the
draft was published or attested. No RC4 device report applies; do not promote
stable from RC4.

### v0.3.0-rc.5 candidate evidence

The corrective release-workflow candidate used
[`v0.3.0-rc.5` dispatch](docs/testing/v0.3.0-rc.5-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). Dual-platform
tagged-artifact acceptance failed when primary-host Claude/Codex installs were
outside the fail-closed ranges; do not promote stable from RC5.

### v0.3.0-rc.6 candidate evidence

The compatibility-widen candidate used
[`v0.3.0-rc.6` dispatch](docs/testing/v0.3.0-rc.6-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). RC6 expanded
Claude Code through `2.1.227` and Codex CLI through `0.147.0`. Dual-platform
tagged-artifact acceptance failed (macOS 16 PASS / 12 FAIL / 4 NOT TESTED) on
real-launch baseline, authenticated resume/fork, capability mutation matrix,
and required TTY/picker evidence; do not promote stable from RC6.

### v0.3.0 stable evidence

The post-RC6 harden candidate uses
[`v0.3.0` dispatch](docs/testing/v0.3.0-rc.7-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md). RC7 packages
non-TTY fail-closed native launch, Windows Ctrl+C safety at the warning prompt,
capability probe demotion to informational, isolated throwaway agent homes for
capability discovery, and expanded deterministic local Phase 3 smoke. Those two
device reports decide RC7 tagged-artifact acceptance only. Stable promotion still
requires a separate reviewed stable decision and fresh tagged-artifact
validation on the same two supported platforms.

### v0.3.0-rc.2 candidate evidence

[`v0.3.0-rc.2` dispatch](docs/testing/v0.3.0-rc.2-agent-verification-prompts.md)
plus the pinned
[Windows acceptance host](docs/testing/windows-acceptance-host.md) were used for
tagged-artifact acceptance. Native Windows x64 failed again (Codex trust and
snapshot/PowerShell staging gates among the blockers); do not promote stable
from RC2. Corrective product fixes land before `v0.3.0-rc.3`. Stable promotion
still requires a separate reviewed stable decision and fresh tagged-artifact
validation on the same two supported platforms. Intel macOS and WSL2 remain
unsupported/unverified optional evidence and are never stable blockers.

### v0.4.0-rc.1 candidate evidence

The `v0.4.0-rc.1` dual-platform run **failed**. Claude Code was unusable as a
handoff source on every real installation (a source probe required an
`<agent-root>/version` file real installs never create, so Claude-sourced
handoffs exited `5`), reader-emitted absolute paths were rejected by capsule
validation, `changed_files` was never populated, a timed-out version probe was
accepted as an absent agent, and any message beginning with a slash aborted the
handoff. One run also omitted `GROK_HOME` and indexed the operator's real
`~/.grok` tree, so that run was discarded and restarted. RC1 does **not**
authorize stable `v0.4.0`, and the corrective fixes land in `v0.4.0-rc.2`.

### v0.4.0-rc.2 candidate gate

The Phase 4 candidate uses the committed
[`v0.4.0-rc.2` dispatch](docs/testing/v0.4.0-rc.2-agent-verification-prompts.md)
and [Phase 4 acceptance contract](docs/testing/phase-4-cross-agent-handoff-acceptance.md).
Start its two independent device runs only after the signed tag is published,
all release artifacts verify, and both live installer routes pin that exact
candidate. Required Claude ↔ Codex structured handoff, fidelity, workspace,
security, CLI, and performance rows must all pass on Apple Silicon macOS and
native Windows x64. Gemini CLI, OpenCode, and Grok Build remain optional
source-only rows and may be `NOT TESTED` only when genuinely absent; do not
install them solely for acceptance.

The candidates widen the fail-closed Claude Code range through `2.1.229` (was
`2.1.227` at stable `v0.3.0`, then `2.1.228` at `v0.4.0-rc.1`) so both physical
acceptance hosts run an in-range install; the Codex CLI range stays
`0.133.0`–`0.147.0`. Neither patch has physical evidence yet — the dual-platform
run is what supplies it, exactly as `v0.3.0-rc.6` widened first and tested
afterwards.

Expect to repeat this. Claude Code auto-updates, and during the `v0.4.0-rc.1`
window both hosts moved past the ceiling within a day (macOS `2.1.225` ->
`2.1.228`, Windows `2.1.228` -> `2.1.229`). Re-check both hosts' installed
versions immediately before tagging, not only when planning the candidate.

`v0.4.0-rc.2` publication meant ready for tagged-artifact acceptance. The
physical dual-platform run **FAILED**: wrong-repo cwd was not refused, non-TTY
destination launch still spawned, Grok-source busy-check exited with
`unsupported agent "grok"`, and a timed-out version probe classified Runtime.
RC2 does **not** authorize stable `v0.4.0`. Corrective product fixes landed in
`v0.4.0-rc.3`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.3 failures landed in
`v0.4.0-rc.4`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.4 failures landed in
`v0.4.0-rc.5`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.5 failures land in
`v0.4.0-rc.6`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for those rc.6 failures land in
`v0.4.0-rc.7`. That candidate was published and physical dual-platform
acceptance **FAILED**. Corrective product fixes for the remaining rc.7 product
defect (R1 off-PATH layout scan) and the Go 1.25.13 toolchain pin land in
`v0.4.0-rc.8`. That candidate was published and physical dual-platform
acceptance **FAILED** (macOS 38/44, Windows 38/44). The remaining product
defect is Windows off-PATH `inspect` JSON `agent.status=not_installed` despite
`layout_recognized=true`; dest-ack A1–A7 stayed harness/uncollected. That
inspect mapping lands in `v0.4.0-rc.9`. That candidate was published and physical
dual-platform acceptance **FAILED** (macOS 38/44, Windows 38/44). The remaining
product defect is Windows CreateProcess truncating multi-line dest argv (Codex
dest-ack A5/A7). That dest-argv fix lands in `v0.4.0-rc.10`.

### v0.4.0-rc.9 candidate gate

`v0.4.0-rc.9` was published and physical dual-platform tagged-artifact
acceptance **FAILED** (macOS 38/44, Windows 38/44). Dest-ack A4 PASS;
A1/A2/A3/A5/A6/A7 FAIL. Remaining product defect was Windows dest argv CR/LF
truncation. Historical dispatch:
[`v0.4.0-rc.9`](docs/testing/v0.4.0-rc.9-agent-verification-prompts.md).
Publication was not evidence that the matrix passed.

### v0.4.0-rc.10 candidate gate

`v0.4.0-rc.10` was published and physical dual-platform tagged-artifact
acceptance **FAILED** (macOS 41/44 A1/A3/A7, Windows 40/44 A1/A2/A3/A5). Dest-ack
A4/A6 PASS both OS; Windows dest argv one-line `projection.md` held. Remaining
product defects were dest first-reply missing the five bullets, lineage after
Launch, and dest TUI folder trust. Historical dispatch:
[`v0.4.0-rc.10`](docs/testing/v0.4.0-rc.10-agent-verification-prompts.md).
Publication was not evidence that the matrix passed.

### v0.4.0-rc.11 candidate gate

The Phase 4 candidate uses the committed
[`v0.4.0-rc.11` dispatch](docs/testing/v0.4.0-rc.11-agent-verification-prompts.md)
and [Phase 4 acceptance contract](docs/testing/phase-4-cross-agent-handoff-acceptance.md).
Start its two independent device runs only after the signed tag is published,
all release artifacts verify, and both live installer routes pin that exact
candidate. Required Claude ↔ Codex structured handoff, fidelity, workspace,
security, CLI, and performance rows must all pass on Apple Silicon macOS and
native Windows x64, including RC1 R1–R6, the RC2, RC3, RC5, RC6, RC7, RC8, RC9,
RC10, and RC11 regression sets.
Gemini CLI,
OpenCode, and Grok Build remain optional source-only rows and may be
`NOT TESTED` only when genuinely absent; do not install them solely for
acceptance.

The fail-closed Claude Code range stays `2.1.219`–`2.1.229`; the Codex CLI range
stays `0.133.0`–`0.147.0`. Re-check both hosts' installed versions immediately
before tagging. Claude Code auto-updates. A PATH Claude of `2.1.230` must
fail-closed (exit 5) without `--allow-untested`.

`v0.4.0-rc.11` publication meant ready for tagged-artifact acceptance. That
acceptance later passed 44/44 on both mandatory platforms and authorized
stable `v0.4.0`.

### v0.4.0 stable evidence

Dual-platform tagged-artifact acceptance PASS on candidate `v0.4.0-rc.11`:

- Apple Silicon macOS 44/44 (`docs/testing/results/2026-08-15-macos-phase4-V040RC11.md`)
- native Windows x64 44/44 (`docs/testing/results/2026-08-15-windows-phase4-V040RC11.md`)

This authorizes the signed stable `v0.4.0` tag. Intel macOS and Linux/WSL2
remain optional and unverified. Native resume remains same-vendor. Gemini CLI,
OpenCode, and Grok Build remain handoff sources only.

### v0.5.0-rc.1 candidate gate

The first Phase 5 candidate. Catalog refactor landed. Probe shipped. Six new
T1 agents (Kimi, Qwen, Pi, Cursor CLI, Copilot, Cline) and three T2 handoff
sources (Gemini, OpenCode, Grok). Claude Code and Codex stay the only T4/T5
surfaces. Gemini is **not** promoted to T3. Generated matrix:
`rein doctor --agents --acceptance-matrix` reports **150** required rows.

Dispatch:
[`v0.5.0-rc.1`](docs/testing/v0.5.0-rc.1-agent-verification-prompts.md).
Contract:
[Phase 5 acceptance](docs/testing/phase-5-universal-agent-coverage-acceptance.md).

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.5.0`. Current stable remains `v0.4.0`.

### v0.5.0-rc.5 candidate gate

The fifth Phase 5 candidate. Physical `v0.5.0-rc.4` acceptance found one defect
that made every other reader fix unreachable: an index never re-read a source
after Reinstate itself changed. Both layers of change detection asked only
whether a file had moved, so an upgrade left an existing index frozen and a
reader fix reached nobody until the user's agent happened to write a new
session. On the Windows host an index built before the Gemini workspace fix
served 24 sessions with no workspace indefinitely; it heals on the first run of
this candidate. Note the row-level half of that predates `v0.5.0`: it shipped
in `v0.4.0`.

This candidate also carries Gemini project-path recovery on case-insensitive
filesystems, an agent root override that is honoured when it names a missing
path rather than silently walking the home tree, Cursor CLI's verified root
variable, incremental refresh, and Windows portability of the documentation
tests on a non-system drive.

Both verified vendor ranges move on dual-platform physical resume evidence —
Claude Code through `2.1.238` and Codex CLI through `0.149.0`. On each platform
a session was created with the new version, indexed, resumed through the launch
plan Reinstate produced, and the resumed session returned a token that existed
only in the original session's history.

Pre-tag dual-platform matrix state on the candidate commit, run with the
harnesses rather than the tagged artifact: 150/150 on macOS and 150/150 on
Windows. That is not tagged-artifact acceptance and does not substitute for it.

Dispatch:
[`v0.5.0-rc.5`](docs/testing/v0.5.0-rc.5-agent-verification-prompts.md).
Contract:
[Phase 5 acceptance](docs/testing/phase-5-universal-agent-coverage-acceptance.md).

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.5.0`. Current stable remains `v0.4.0`.

### v0.5.0-rc.6 candidate gate

The sixth Phase 5 candidate. Physical `v0.5.0-rc.5` tagged-artifact acceptance
ran the full matrix on both platforms and **failed**: macOS 149/150,
Windows 150/150. One required row failed, **B4** on macOS, so `v0.5.0-rc.5`
did not authorize stable `v0.5.0`.

B4: the agent probe carried a raw 38-character Git object hash into its
artifact. OpenCode keeps a Git object store under each snapshot, and Git stores
an object as a two-character directory plus a thirty-eight character file. The
shape normalizer recognised only exactly 32, 40 and 64 characters, so a
38-character stem matched nothing and reached the artifact verbatim. Those names
are content hashes of the operator's own repository. This candidate collapses
any hex run long enough to identify content; the established tokens are
unchanged so committed artifacts do not churn.

Everything else passed on both platforms, including the upgrade path the
previous candidate existed to fix: an index built by the previous release and
then opened by the tagged binary re-read what the old reader could not resolve
(0 to 3 resolved workspaces on macOS, 0 to 10 on Windows) without losing a row,
and the previous release could still reopen it. All four E1/E2 journeys ran
against the installed tagged binary on both platforms.

Physical acceptance also found more harness defects than product defects, two of
them false passes. The `v0.5.0-rc.6` dispatch records them so a later run does
not repeat them.

Dispatch:
[`v0.5.0-rc.6`](docs/testing/v0.5.0-rc.6-agent-verification-prompts.md).
Contract:
[Phase 5 acceptance](docs/testing/phase-5-universal-agent-coverage-acceptance.md).

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.5.0`. Current stable remains `v0.4.0`.

### v0.5.0 stable evidence

Dual-platform tagged-artifact acceptance PASS on candidate `v0.5.0-rc.6`:

- Apple Silicon macOS 150/150
  (`docs/testing/results/2026-08-21-macos-phase5-V050RC6.md`)
- native Windows x64 150/150
  (`docs/testing/results/2026-08-21-windows-phase5-V050RC6.md`)

This authorizes the signed stable `v0.5.0` tag. Intel macOS and Linux/WSL2
remain optional and unverified. Native resume and encrypted sync remain
same-vendor and remain limited to Claude Code and Codex CLI. Gemini CLI,
OpenCode, and Grok Build remain handoff sources only; Gemini was **not**
promoted to T3.

Two dispositions are carried into this release rather than resolved by it, and
both are recorded in the device reports. Both are fixed after `v0.5.1`; the
records below describe `v0.5.0` and are left as they were written.

- **E5** — active-session detection is implemented and exercised on the handoff
  and restore paths; `rein resume` applies no such guard.
- **B4 on Windows** — the row passes there without exercising the fix, because
  that host's OpenCode object stores hold only packed objects and OpenCode
  declares no root environment variable to redirect the probe with. The
  end-to-end evidence is on macOS; the Windows evidence is the unit tests that
  cover the same code path in CI.

Five candidates were published and failed physical acceptance before this one:
`v0.5.0-rc.1` through `v0.5.0-rc.5`. The defect that `v0.5.0-rc.5` acceptance
found — an agent probe carrying a raw Git object hash — is why `v0.5.0-rc.6`
exists.

### v0.5.2-rc.1 candidate evidence

Published 2026-08-23: the interactive CLI, OpenCode and Grok Build at T3/T4,
and Qwen at T4. Never certified on either platform; no device report exists
(#366). Its content ships inside `v0.6.0-rc.1` rather than standing alone,
and there is no stable `v0.5.2` (ADR 0005, D4). #366 is re-pointed at the
`v0.6.0` Windows run below.

### v0.6.0-rc.1 candidate gate

Carries the Hop client — sign-in, the locker, device pairing and revocation,
key rotation, machine migration, and `rein daemon` — and everything
`v0.5.2-rc.1` introduced: the interactive switcher, the handoff studio, the
setup wizard, and the `ctrl+k` palette. OpenCode reaches T5 and Kimi Code CLI
reaches T2.

Governed by
[`docs/testing/v0.6.0-windows-acceptance.md`](docs/testing/v0.6.0-windows-acceptance.md),
which composes the Phase 5 generated matrix (**178** rows, per
`rein doctor --agents --acceptance-matrix` on a binary built from the release
commit), the 22-row CLI matrix, and the 16 Hop parity journey rows of section
D (hosted #16) into one Windows column with an explicit deferred-macOS table,
under the [Windows-first waiver](#v060-windows-first-waiver).

The pre-tag snapshot run (a snapshot build of the release commit, before the
tag is signed and pushed) is evidence that the candidate is ready for
tagged-artifact acceptance; it does not itself authorize anything past that.
It is recorded at
`docs/testing/results/2026-09-06-windows-v060rc1-pretag.md`: four passes on
2026-09-06, the last on commit `9dcef0c0`, ending at **185 of 200** required
rows `PASS` (Phase 5 matrix plus CLI rows; the 16 Hop parity rows are
recorded separately at 14 `PASS` / 2 `PARTIAL`). The run found two Windows
product defects, both fixed on the branch and re-verified against a fresh
snapshot: the warning checklist ignored the space bar on native Windows
(CLI row 14), and a failed process enumeration was reported as "not busy"
(Matrix E, row E5). Claude Code auto-updated from `2.1.261` to `2.1.263`
between passes and the ceiling moved with it.

**Dispositions carried into this candidate**, to be cleared or re-recorded
by the tagged-artifact run:

- **`opencode` C3** — search by message text finds nothing because the
  OpenCode reader indexes id, title, project, workspace, and branch only,
  as it has since `v0.5.0`; search by title passes. Documented reader
  behaviour, tracked as #405. Not a regression.
- **`opencode` D4** — a SQLite-only OpenCode store has no JSONL record
  boundary to truncate; the row is definitional for that layout.
- **E5 for every T3+ agent** — this host's WMI repository is damaged (both
  `Get-CimInstance Win32_Process` and `tasklist` fail), so an active session
  cannot be detected here; the fail-safe the fix introduced (the check
  reports that it could not run, and does not refuse) was verified instead.
  Host condition, clarifications Q14.
- **`claude` E2, E3, D4** — every spawned `claude` process on this host
  fails OAuth refresh while interactive sessions hold the token; `rein`
  produces the correct launch plan at `2.1.263` (E1 passes) but the resumed
  session cannot answer. Host condition, clarifications Q16.
- **`qwen` E2, E3, D4** — the host's Qwen Code credential is expired.
- **`gemini` D4, `kimi` D4** — no non-interactive credential on the host.

The tagged run must record each of these as `PASS` or as the same
disposition with the same reason; a new reason is a new finding.

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.6.0`. Current stable remains `v0.5.1`.

### v0.6.0-rc.1 candidate evidence

`v0.6.0-rc.1` was published 2026-09-06 as a signed GitHub prerelease with
both live installer routes pinning it. Its tagged-artifact native Windows
acceptance is recorded at
[`docs/testing/results/2026-09-06-windows-v060rc1.md`](docs/testing/results/2026-09-06-windows-v060rc1.md):
**201 of 216** required rows `PASS` (3 `PARTIAL`, 7 `FAIL`, 5 `NOT TESTED`),
device verdict `FAIL`. Every Hop parity row (section D, 16/16) and every CLI
experience row (section C, 22/22) passed; the 7 required-row failures were
all in the Phase 5 generated matrix:

- `cursor:C6` (MAJOR, `F-CURSOR-ROOTENV`) — `CURSOR_CONFIG_DIR` isolated only
  the `doctor --agents` probe, never `sessions`/`search`/`inspect`/`resume`/
  `fork`, because Cursor's source never set `hometree.Config.RootEnv`.
- `cline:C2`, `cursor:C2` (MINOR) — `message_count` unconditionally `0` for
  both agents instead of derived from the vendor's own message-bearing file.
- `cline:C3`, `cursor:C3` (MINOR) — search indexes id/title/project/workspace
  only, never message body, for both agents.
- `MatrixH:H6` (MINOR, `F-COMPLETION-PUSHPULL`) — `push --agent`/
  `pull --agent` offered no shell-completion candidates.
- `opencode:D5` (`PD-B1`) — two `--dry-run` handoffs over an unchanged
  OpenCode source produced different `handoff_id`/destination `session_id`,
  because the source boundary hashed OpenCode's entire shared `opencode.db`
  instead of just that session's own rows.

`claude:D4`/`codex:D4` re-recorded `PARTIAL` (fixture workspace does not
resolve on Windows; correct refusal) and `opencode:D4` re-recorded
`NOT TESTED` (definitional: a SQLite-only store has no JSONL boundary),
carried unchanged from the pre-tag report. `grok:D4` was `NOT TESTED` for a
new reason: no committed `partial-final-record` fixture existed for Grok
Build. `qwen:E1/E2/E3/E5` stayed `PARTIAL`/`NOT TESTED` on an expired host
credential, a host condition rather than a product defect. This report does
**not** authorize stable `v0.6.0`. Corrective product fixes for the 7
required-row failures, plus the `grok:D4` fixture gap, land in
`v0.6.0-rc.2`.

### v0.6.0-rc.2 candidate gate

The corrective candidate. It changes no agent's tier and widens no
compatibility range — `v0.6.0-rc.1`'s Claude Code, OpenCode, and Codex CLI
ranges are unchanged — and fixes exactly the defects `v0.6.0-rc.1`'s tagged
Windows run found:

- Cursor CLI session discovery, search, inspect, resume, and fork now honour
  `CURSOR_CONFIG_DIR` (`hometree.Config.RootEnv` set in `config()`, matching
  every sibling T1 source), closing `F-CURSOR-ROOTENV`. A new conformance
  check fails a hometree agent whose source ignores its declared root
  environment variable, so this class of gap cannot regress silently.
- Cline and Cursor `message_count` is derived from each vendor's own
  message-bearing file (Cline's per-task `*.messages.json` sidecar; Cursor's
  sibling `store.db`) instead of hard-coded `0`; Cursor `size_bytes` now
  covers `store.db` as well as `meta.json`.
- `rein push --agent` and `rein pull --agent` offer shell-completion
  candidates, closing `F-COMPLETION-PUSHPULL`.
- An OpenCode-sourced handoff's boundary hash now covers only that session's
  own rows instead of the whole shared `opencode.db`, so repeated
  `--dry-run` invocations over an unchanged session are deterministic,
  closing `PD-B1`.
- Committed Windows-shaped `partial-final-record` fixtures for Claude Code,
  Codex, and Grok Build, plus a pipeline-level test
  (`internal/handoff/partial_final_record_route_test.go`) that drives the
  real `handoff.Plan()` route against every one and cross-checks the
  resulting capsule's byte-exact truncation offset and SHA-256, closing the
  `grok:D4` fixture gap.

`cline:C3`, `cursor:C3`, and the pre-existing `opencode:C3` gap (search
excluded message body, indexing id/title/project/workspace only) are now
fixed as well (closes #405): all three sources index user-authored message
text through the same bounded, sanitized builder every reader uses, capped
at `MaxSearchTextBytes`; `opencode:C3`, which previously passed by title
match alone, now also matches on body.

Governed by the same
[`docs/testing/v0.6.0-windows-acceptance.md`](docs/testing/v0.6.0-windows-acceptance.md)
contract, specialised by
[`docs/testing/v0.6.0-rc.2-agent-verification-prompts.md`](docs/testing/v0.6.0-rc.2-agent-verification-prompts.md).
`rein doctor --agents --acceptance-matrix` on a binary built from this tree
reports **178** Phase 5 rows (core `A:10, B:9, G:8, H:6` = 33, unchanged from
`v0.6.0-rc.1`), plus the 22-row CLI matrix and the 16 Hop parity rows —
**216** rows in total, the same count as `v0.6.0-rc.1`, of which
`opencode:D4` is `N/A (definitional)` under the disposition rules in
[`docs/testing/v0.6.0-windows-acceptance.md`](docs/testing/v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict):
**215 required**. A `qwen:E1`/`E2`/`E3`/`E5` row unable to complete because
the host's Qwen Code credential is expired is recorded
`NOT TESTED (host credential)` under the same rules and does not block the
verdict, provided every required agent and at least one other T4 agent
(`grok`) pass the same rows.

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.6.0`. Current stable remains `v0.5.1`.

### v0.6.0-rc.2 candidate evidence

`v0.6.0-rc.2` was published 2026-09-07 as a signed GitHub prerelease with
both live installer routes pinning it. Its tagged-artifact native Windows
acceptance is recorded at
[`docs/testing/results/2026-09-07-windows-v060rc2.md`](docs/testing/results/2026-09-07-windows-v060rc2.md):
device verdict `FAIL`, **203 PASS / 5 PARTIAL / 2 FAIL / 5 NOT TESTED** of
**215** required rows (as first assembled: `204/6/0/5`; `cursor:C2` and
`cursor:C3` were re-scored `PASS`/`PARTIAL` → `FAIL` post-commit — see
below). All 22 CLI-experience rows passed and 15 of 16 Hop parity rows
passed; the seven required-row failures `v0.6.0-rc.1` found were cleared.
Eleven required rows blocked the verdict:

- `cline:C3`, `cursor:C3`, `pi:C3` (`PARTIAL`, host/harness) — the fixed
  search-by-message-body mechanism was confirmed via a substitute method,
  but the specific "session created via the vendor's own CLI" evidence the
  dispatch calls for could not be produced this run (credential-policy
  block, broken `cursor-agent` install, no configured Pi model/API key).
- `grok:E1`/`E2` (`PARTIAL`) and `grok:E3` (`NOT TESTED`), new this run
  (`F-GROK-MCP-RECONNECT`) — this host's shared, global MCP server set
  repeats connect/disconnect context noise on every `--resume` turn,
  blocking a completed reply; `rein`'s own launch plan, argv, and
  active-session detection were independently confirmed correct.
- `qwen:E1`/`E2`/`E3` (`NOT TESTED (host credential)`, blocking) — the
  host-credential disposition that would excuse these requires `grok` (the
  other optional T4 agent) to `PASS` the same row, which it did not this
  run because of the finding above; `qwen:E5` did qualify (`grok:E5` was
  `PASS`) and did not block.
- `MatrixH:H7` (`PARTIAL`, new this run) — UAC elevation was declined twice
  with no interactive operator available to accept the prompt; the
  identical mechanism passed in full at `v0.6.0-rc.1` on this same host,
  so this is availability of an acceptor, not a regression, but it still
  does not match either contract disposition.

A post-commit coordinator finding (report §0.12) is what corrective work
after this report addresses: a schema-only, read-only inspection of the
host's two real Cursor CLI `2026.08.11` `store.db` files found
`blobs(id TEXT, data BLOB)` and `meta(key, value)` — not the
`messages`/`message`/`bubbles` tables the `v0.6.0-rc.2` reader recognized
and no real store ever had. `cursor:C2` (real `message_count` still `0`)
and `cursor:C3` (search by body cannot succeed against tables the real store
lacks) were re-scored `FAIL` on that finding; the fix targets a schema that
does not exist. This report does **not** authorize stable `v0.6.0`.
Corrective work for the Cursor CLI store schema, which closes `cursor:C2`
and `cursor:C3`, lands in `v0.6.0-rc.3`; the `grok` stall, the `cline` and
`pi` C3 real-session evidence gap, the expired Qwen credential, and
`MatrixH:H7`'s operator-availability gap are host and operator conditions
that remain open and are not product changes in that candidate.

### v0.6.0-rc.3 candidate gate

The corrective candidate. It changes exactly the Cursor CLI store reader
and nothing else: no agent's tier moves, and no compatibility range widens
— `v0.6.0-rc.2`'s Claude Code, OpenCode, and Codex CLI ranges are
unchanged.

- Cursor CLI's `message_count` now counts `blobs` rows whose `data` is a
  JSON object with `role` `user` or `assistant` (`system` rows excluded,
  matching the search policy) instead of the `messages`/`message`/
  `bubbles` tables `v0.6.0-rc.2` guessed at and no real store ever had.
  `search_text` and `PromptPreview` come from the `content` of `user`-role
  blobs only. Every row is read bounded (4 MiB per row in the SQL `SELECT`
  itself, `role` read by a streaming decoder that stops as soon as it has
  that field), and the scan is bounded in total to 20,000 rows. A row whose
  first byte is not `{` — the majority of rows in both real inspected
  stores — is skipped by that one byte, never decoded. On this host's own
  real Cursor CLI data both real sessions now report a non-zero
  `message_count` and are found by `rein search <word> --agent cursor`.

This candidate does not attempt the `grok` MCP-reconnect finding, `cline:C3`'s
or `pi:C3`'s real-vendor-session evidence gap (host-credential and harness
issues, unrelated to the Cursor store schema), or `MatrixH:H7`'s
operator-availability gap — those remain open findings from the `v0.6.0-rc.2`
tagged run and are re-tested as carried dispositions rather than closed here.
`cursor:C2` and `cursor:C3` are the T1 C-tier gap this candidate does close:
the store-schema fix above is expected to flip both to `PASS` against the
real host store, which under the contract's disposition rule (a
`NOT TESTED (host credential)` row is excused once any *one* other optional
agent at the same tier `PASS`es the same row) is also expected to excuse
`cline:C3` if Cline is still unauthenticated when this run happens.

Governed by the same
[`docs/testing/v0.6.0-windows-acceptance.md`](docs/testing/v0.6.0-windows-acceptance.md)
contract, specialised by
[`docs/testing/v0.6.0-rc.3-agent-verification-prompts.md`](docs/testing/v0.6.0-rc.3-agent-verification-prompts.md).
`rein doctor --agents --acceptance-matrix` on a binary built from this tree
reports **178** Phase 5 rows (core `A:10, B:9, G:8, H:6` = 33, unchanged from
`v0.6.0-rc.2`), plus the 22-row CLI matrix and the 16 Hop parity rows —
**216** rows in total, the same count as `v0.6.0-rc.2`, of which
`opencode:D4` is `N/A (definitional)` under the disposition rules in
[`docs/testing/v0.6.0-windows-acceptance.md`](docs/testing/v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict):
**215 required**.

Publication means ready for tagged-artifact acceptance. It does **not**
authorize stable `v0.6.0`. Current stable remains `v0.5.1`.

## Steps

### 1. Prepare the release commit

The release commit itself must contain both public bootstrap files pinned to
the exact new CLI tag, with each canonical installer digest recomputed from the
final `scripts/install.*` bytes. It must also contain synchronized changelog,
compatibility, citation, website release truth, and candidate-dispatch updates.
Do not tag a commit whose bootstraps still name the previous release: the
signed website deployment later requires those files to be byte-identical to
the CLI tag, so a post-tag pin-only edit cannot repair it.

```bash
# Update release truth, public bootstrap pins, and compatibility evidence.
git add --all
git commit -m "chore(release): vX.Y.Z"

# Verify the exact clean release commit before pushing its PR branch.
GOTOOLCHAIN=go1.25.13 go mod tidy -diff
make verify
make snapshot
./scripts/stage-release-assets.sh dist
./scripts/check-release-artifacts.sh dist
sh scripts/test-install.sh dist
git diff --exit-code -- go.mod go.sum
test -z "$(git status --porcelain)"

git push -u origin release/vX.Y.Z
# Open a draft PR, pass protected-main CI, review, and merge.
```

GoReleaser's before hook runs `go mod tidy`; the post-snapshot diff and clean
check prove that it did not silently change the committed module graph or any
other tracked release input. `stage-release-assets.sh` is required before the
artifact and installer checks because raw GoReleaser binaries originate inside
target-specific directories and must be staged under their checksummed release
asset names.

### 2. Tag and push

```bash
export REINSTATE_SIGNING_KEY="$HOME/.ssh/reinstate_release_signing"
git -c gpg.format=ssh \
  -c user.signingkey="$REINSTATE_SIGNING_KEY" \
  tag -s vX.Y.Z -m "Reinstate vX.Y.Z"
git push origin vX.Y.Z
```

The tag must point at the reviewed commit on protected `main`. Do not move or
reuse a published tag. The matching public key and maintainer principal must
be present in `.github/allowed_signers` so CI can verify the signature without
depending on machine-local keyring state. `$REINSTATE_SIGNING_KEY` lives only
on the maintainer's own machines; an agent never holds it and never runs this
step.

### 3. GitHub Release workflow

The release workflow builds binary and source archives, generates checksums and
per-binary-archive SBOMs, tests installer contracts, publishes a draft release,
and creates GitHub artifact attestations.

Before publishing the draft:

1. Confirm asset names match `reinstate_<version-without-v>_<os>_<arch>`.
2. Run both official installers against the exact draft assets.
3. Verify checksums and `gh attestation verify` for each archive.
4. Confirm archive contents include binary, license, notice, README, and changelog.
5. Mark prerelease tags as pre-release; publish stable only after every release gate.

Publishing the verified GitHub draft triggers **Publish package managers**.
That workflow re-verifies the signed tag, main ancestry, checksums, and GitHub
attestations before any enabled downstream job can run. Registry jobs are
opt-in and protected by the `package-publish` environment. Complete the account,
secret, repository-variable, and first-publication steps in
[Package-manager publishing](docs/package-manager-publishing.md) before enabling
them.

Stable releases may promote to npm, JSR, Homebrew, Chocolatey, Scoop, WinGet,
and AUR. Prereleases promote only to npm (`next`) and JSR. Do not publish a
registry package from a draft release or from locally rebuilt binaries.

For stable `v0.2.0`, complete both the **Stable v0.2.0 publication reminder**
and the **Post-publication documentation reminder** in
[Package-manager publishing](docs/package-manager-publishing.md). Enabled CI is
not evidence that a package is publicly listed, accepted by an external
registry, or verified on its native platform.

The GitHub draft now also contains raw binaries and `.deb`, `.rpm`, `.apk`, and
Arch package files. Confirm those files are covered by `checksums.txt` and
artifact attestations before publishing the draft.

### 4. Publish the public installer routes

Automatic Vercel Git deployments are disabled. The public bootstrap pins were
already committed before the CLI tag in Step 1. After the GitHub release is
published, confirm both tagged bootstrap files pin that same release, update a
clean local `main`, and create a signed, annotated
`website-vYYYY.MM.DD.N` tag at the exact `origin/main` commit. Link the existing
Vercel project if necessary. Push the tag and wait for the
**Validate signed website deployment tag** workflow to pass, then run:

```bash
./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N
```

The script derives the CLI release tag independently from `install.sh` and
`install.ps1`, refuses a mismatch, and verifies that release before it deploys.
It deploys without moving the production alias, verifies both installers
against the derived CLI release at the immutable deployment URL, promotes only
that verified deployment, and verifies both live routes again. Never run
`vercel --prod` directly for a release.

The script runs the website's own check chain locally before it deploys. A
host that cannot run that chain (native Windows: the SQLite-backed tests and
the PNG-render reproducibility check are bound to the Linux runner) may
instead present the CI run that already certified the exact commit:

```bash
REINSTATE_DEPLOY_CI_RUN=<ci run id> ./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N
```

The script then requires that run's head to be `HEAD` and its **Website**
job to have succeeded, and still builds locally and plans IndexNow. It is a
substitution of evidence, not a skip: the same checks ran, on the runner
that is the arbiter for them.

For a release candidate, start its committed candidate-specific acceptance
dispatch only after both live routes install the new exact version. Apple
Silicon macOS and native Windows x64 own the mandatory two-device matrix;
provision Windows per
[windows-acceptance-host.md](docs/testing/windows-acceptance-host.md) before
product rows. A passing candidate matrix certifies only that exact tagged
candidate. Stable promotion remains a separate reviewed decision with fresh
tagged-artifact evidence on both supported platforms. Native macOS amd64 and
WSL2 amd64 are unsupported/unverified optional evidence and do not block a
candidate or stable promotion.

### 5. Publish website-only changes

A website deployment is not a CLI release. Use a signed, annotated
`website-vYYYY.MM.DD.N` deployment tag when reviewed website changes need to
ship without advancing the current Reinstate version. The website tag must
point at the exact current `origin/main` commit; it must not be published as a
GitHub Release or described as a new CLI version.

The public installer identity remains explicit inside both committed bootstrap
files. The script derives their CLI release tag, requires them to agree, and
verifies the corresponding published release. Push the website tag and wait
for the **Validate signed website deployment tag** workflow to pass before
running:

```bash
./scripts/deploy-website-production.sh website-vYYYY.MM.DD.N
```

For example, while Reinstate remains current:

```bash
./scripts/deploy-website-production.sh website-v2026.07.28.1
```

The guarded script verifies the website tag signature, requires it to match
clean local `main` and `origin/main`, verifies the derived signed CLI release,
and requires both committed public installers to match that release before
deploying. It then applies the same immutable-deployment checks, installer byte
comparisons, production discovery smoke tests, promotion, and live-origin
verification. A `website-v...` tag does not satisfy or replace any CLI release,
compatibility, or acceptance gate.

### 6. Announce (optional)

- GitHub Discussions "Show and tell" / announcements
- X/Twitter [@HarjjotSinghh](https://x.com/HarjjotSinghh)
- Relevant community threads (only when the release is useful, not spam)

## Hotfix releases

1. Branch from the release tag if needed: `release/x.y`
2. Cherry-pick the fix
3. Bump **PATCH**, release as above
4. Merge back to `main`

## Rollback

Published releases are immutable. If a defect escapes, issue a new patch or
prerelease. Do not move the tag or replace assets.
