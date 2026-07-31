# Phase 2 Parallel Autonomous Verification Prompts

These prompts automate the physical
[Phase 2 local-index acceptance runbook](phase-2-local-index-acceptance.md).
The workflow is release-neutral, but this checked-in handoff is pinned to the
OpenCode timestamp-fix development commit
`b952d38c2dc57b0a96bc696860318ea7c2975800`. Both devices must test the same exact
clean commit from source and must not substitute the later commit that updates
this prompt document. Do not call the tested build a published release.

Phase 2 has no shared remote state. The Mac and Windows prompts start
independently and run in parallel. The only cross-device handoff is the
complete final Windows report to the Mac coordinator for reconciliation.

The only human actions are:

1. place a clean checkout of
   `b952d38c2dc57b0a96bc696860318ea7c2975800` on both devices;
2. launch Claude Code on macOS with Prompt 1 and Codex on native Windows with
   Prompt 2 at approximately the same time; and
3. give the complete sanitized Windows report to the existing Mac Claude
   session after both device runs finish.

No private acceptance file is used. Do not create `R2.txt`. Do not provide
storage coordinates, credentials, a passphrase, or a profile ID. Do not run
`rein init`, `push`, `pull`, `status`, `diff`, or conflict commands during the
Phase 2 local-only run.

Both agents preserve `PASS`, `PARTIAL`, `FAIL`, and `NOT TESTED` honestly. A
zero exit code is never sufficient by itself.

## Prompt 1 — Claude Code on macOS

Copy everything inside this block into Claude Code in the clean test checkout:

```text
Run the complete macOS side of Reinstate Phase 2 local-index acceptance. You
are Device A's evidence owner and the final cross-device coordinator. Work
autonomously: do not ask me to run a command, select a session, confirm a
marker, inspect a file, or approve a routine test step. Do not delegate this
work and do not modify Reinstate product code.

Authority:
- Read AGENTS.md.
- Read docs/superpowers/plans/2026-07-30-phase-2-local-session-index.md.
- Read docs/testing/phase-2-local-index-acceptance.md.
- Read docs/testing/results/phase-2-report-template.md.
- The plan defines product scope. The acceptance runbook defines physical
  gates and expected results. Stop and report any conflict.

Test target:
- Set EXPECTED_TEST_COMMIT to
  b952d38c2dc57b0a96bc696860318ea7c2975800.
- Set TEST_COMMIT to the full commit at the clean checkout's HEAD and require
  TEST_COMMIT to equal EXPECTED_TEST_COMMIT. A mismatch is FAIL: stop without
  switching commits or running product behavior tests.
- Fetch origin and prove TEST_COMMIT is reachable from
  origin/fix/opencode-top-level-timestamps. Separately record whether it is
  reachable from origin/main; main reachability is not required for this
  development run.
- Do not change commits, merge, rebase, cherry-pick, or test a different build.
- If HEAD is a signed release-candidate tag, verify the annotated tag,
  signature, checksums, public installer pin, and installed binary commit.
- Otherwise build the exact commit locally, record it as development
  acceptance, and do not use the public v0.1.0 installer as Phase 2 evidence.
- Require rein version --json to identify TEST_COMMIT. Record a mismatch as
  FAIL and stop before product behavior tests.

Repository scope:
- The only allowed repository change is the sanitized report:
  docs/testing/results/REPORT_DATE-macos-phase2-TEST_ID.md
- Resolve REPORT_DATE from current UTC in YYYY-MM-DD form and TEST_ID from the
  first 12 hexadecimal characters of TEST_COMMIT.
- Use a dedicated worktree and report branch
  test/phase2-TEST_ID-macos-report from TEST_COMMIT.
- Create a new report branch and draft PR for this exact commit. Do not update,
  overwrite, merge, or close an older Phase 2 report branch or PR.
- At completion, commit only that report, push the report branch, and
  create or update one draft PR. Never merge, tag, release, deploy, or modify a
  product branch.
- Prove the staged diff contains only the report and no local index, vendor
  session, fixture copy, generated binary, transcript, or private path.

Hard safety and privacy:
- Keep normal sandboxing and approval controls enabled. Never use a
  permission-bypass flag.
- Do not read, print, summarize, screenshot, copy, commit, or upload unrelated
  transcript content, assistant reasoning/messages, tool output, environment
  dumps, auth files, tokens, cookies, .env files, keychains, or credentials.
- Inspect only before/after vendor metadata, exact controlled marker presence,
  exact challenge-response output, composite references, bounded Reinstate
  metadata, and source fingerprints needed by the runbook.
- A marker needs at least one exact occurrence; do not require a fixed
  serialization count.
- Never alter a vendor session file to manufacture indexing, ambiguity,
  malformed input, an incomplete line, or a fork. Use deterministic synthetic
  tests for destructive edge cases.
- Never delete an older Reinstate home, real agent session, or unrelated
  project. Cleanup is deferred until both reports are reviewed.
- Report only non-secret versions, composite controlled references, counts,
  booleans, exit codes, bounded controlled markers, redacted relative paths,
  and sanitized errors.

Local-only boundary:
- Create a fresh absolute REINSTATE_HOME and the two disposable projects
  prescribed by the runbook. Stop if any target already exists.
- Never run rein init or any sync/storage/conflict command.
- Do not create or use R2.txt, a sync profile, storage environment values,
  credentials, an encryption passphrase, or a keyring entry.
- Phase 2 local commands must succeed without config or a backend. The only
  permitted Reinstate state is the private derived index under
  cache/session-index-v1.sqlite.
- If any Phase 2 command requests setup, storage, a credential, a passphrase,
  keyring access, or a network backend, record FAIL and stop dependent gates.

Controlled data:
- Use harmless new disposable Git repositories, branches, relative files, and
  unique markers derived from TEST_ID.
- Create and cleanly close exactly one fresh Claude Code session and one fresh
  Codex session through their documented non-interactive invocation/resume
  modes. Identify the native IDs by before/after metadata and controlled
  markers only.
- If Gemini CLI or OpenCode is already installed, create one harmless
  disposable session for its read-only path. Do not install a missing vendor
  solely for this run.
- Do not reuse an older acceptance session or include unrelated listing rows in
  the report.
- This is a complete fresh run at a new product commit. Do not carry forward a
  PASS, command result, session reference, or source fingerprint from any
  report against `5c60ec2` or another commit.

Milestone M0 — provenance, environment, and automated gates:
1. Record TEST_COMMIT, tag state, Reinstate version JSON, macOS/architecture,
   shell, Git, Go, and all four vendor versions/compatibility states.
2. Build the exact source. First run
   `go test ./internal/sessionindex -run OpenCode -count=1`, then run every
   applicable automated and cross-build gate in runbook section 5, including
   the complete merge gate.
3. Record exact failed commands, exit codes, and sanitized output. Do not call
   a skipped or interrupted command PASS.

Milestone M1 — configless index, refresh, search, and inspect:
4. Create the isolated home/corpus and prove no config, state, backup,
   credential reference, or index exists before the first command.
5. Execute runbook sections 6 through 9 completely: sessions/list
   compatibility, rein/reinstate JSON parity, ordering, permissions,
   idempotency, all search dimensions, bounded inspect privacy, append refresh,
   and new-session refresh.
6. Prove indexing/search/inspect do not modify a vendor source and do not
   create anything except derived cache state.

Milestone M2 — last, native resume, and native fork:
7. Execute dry-run JSON plans before every real launch in runbook sections 10
   and 11. Verify executable, argv array, cwd, no launch, and no mutation.
8. Prove exact same-vendor Claude and Codex resume with controlled challenge
   responses. Distinguish vendor writes after launch from Reinstate mutation.
9. Prove vendor-native fork for both agents, source preservation, distinct
   identity, refresh discovery, independent resume, and branch independence.
10. Run missing/ambiguous reference, missing workspace/executable,
    JSON-without-dry-run, child-failure, and read-only-agent refusal gates
    without damaging real state.

Milestone M3 — interactive switcher and read-only adapters:
11. Through a real native PTY, test bare rein and reinstate: /text filtering,
    i NUMBER inspect, NUMBER resume, f NUMBER fork, invalid input, q cancel,
    EOF/interrupt, and unrelated-transcript privacy.
12. Prove non-TTY invocation exits promptly with usage code 2 and a
    rein sessions --json hint.
13. When installed, prove Gemini/OpenCode discovery, search, inspect, and
    read-only capability metadata. When absent, record NOT_INSTALLED and only
    the optional physical row as NOT TESTED. Their fixture/fake-runner tests
    still must pass. For OpenCode, additionally prove its controlled record has
    a non-zero, non-year-1 `updated_at` derived from vendor metadata and appears
    in the unfiltered default `rein sessions --json` result without requiring
    `--agent opencode`.

Milestone M4 — report:
14. Copy the report template to the resolved Mac report path and complete every
    section. Fill only the macOS column of the 32-row matrix; do not inherit
    Windows results or copy evidence from a report against another commit.
15. Include one complete PHASE2-DEVICE-REPORT-V1 block with device=macos and
    its END-PHASE2-DEVICE-REPORT-V1 terminator.
16. Commit/push only the report, create or update the draft PR, and return:
    verdict and counts; whether every required local row passed; report path,
    commit, branch, and PR; release-blocking/non-blocking findings; every failed
    command/exit/sanitized output; and confirmation that no product file,
    transcript, private path, or secret was committed.
17. Pause. The human will transfer the complete Windows report after its
    independent run.

Milestone M5 — final reconciliation after the Windows report:
18. Validate the supplied report against its branch tip and commit. Require a
    report-only diff from the same TEST_COMMIT, a complete terminated
    PHASE2-DEVICE-REPORT-V1 block with device=windows, no secret/transcript/
    absolute-private-path content, and honest gate counts.
19. Do not trust peer PASS labels alone. Reconcile all 32 rows by citing which
    device produced the physical evidence and which automated gate covers
    synthetic-only cases. Rows 28 and 29 may be NOT TESTED only when that
    vendor was not installed on that device.
20. Independently verify the tested commit's required CI/check results. Missing
    evidence is NOT TESTED, never PASS.
21. Require Windows row 29 to be PASS with explicit evidence that the fresh
    OpenCode record has a non-zero, non-year-1 `updated_at` and is present in
    the unfiltered default listing. Otherwise final Phase 2 status is not PASS.
22. Add PHASE2-FINAL-RECONCILIATION-V1 to the Mac report, finalize it, and
    commit/push only that report. Keep the PR draft/unmerged.

Phase 2 is PASS only when runbook section 15's required rows pass on both real
devices, automated gates are green, both reports test the same commit, and
there is no release-blocking finding. Until M5 completes, report the status as
pending cross-device reconciliation.
```

## Prompt 2 — Codex on native Windows

Copy everything inside this block into Codex in the clean test checkout:

```text
Run the complete native-Windows side of Reinstate Phase 2 local-index
acceptance. You are Device B's independent evidence owner. Work autonomously:
do not ask me to run a command, select a session, confirm a marker, inspect a
file, or approve a routine test step. Do not delegate this work and do not
modify Reinstate product code.

Authority:
- Read AGENTS.md.
- Read docs/superpowers/plans/2026-07-30-phase-2-local-session-index.md.
- Read docs/testing/phase-2-local-index-acceptance.md.
- Read docs/testing/results/phase-2-report-template.md.
- The plan defines product scope. The acceptance runbook defines physical
  gates and expected results. Stop and report any conflict.

Test target:
- Use native 64-bit Windows PowerShell, never WSL.
- Set EXPECTED_TEST_COMMIT to
  b952d38c2dc57b0a96bc696860318ea7c2975800.
- Set TEST_COMMIT to the full commit at the clean checkout's HEAD and require
  TEST_COMMIT to equal EXPECTED_TEST_COMMIT. A mismatch is FAIL: stop without
  switching commits or running product behavior tests.
- Fetch origin and prove TEST_COMMIT is reachable from
  origin/fix/opencode-top-level-timestamps. Separately record whether it is
  reachable from origin/main; main reachability is not required for this
  development run.
- Do not change commits, merge, rebase, cherry-pick, or test a different build.
- If HEAD is a signed release-candidate tag, verify the annotated tag,
  signature, checksums, public installer pin, and installed binary commit.
- Otherwise build the exact commit locally, record it as development
  acceptance, and do not use the public v0.1.0 installer as Phase 2 evidence.
- Require rein version --json to identify TEST_COMMIT. Record a mismatch as
  FAIL and stop before product behavior tests.

Repository scope:
- The only allowed repository change is the sanitized report:
  docs/testing/results/REPORT_DATE-windows-phase2-TEST_ID.md
- Resolve REPORT_DATE from current UTC in YYYY-MM-DD form and TEST_ID from the
  first 12 hexadecimal characters of TEST_COMMIT.
- Use a dedicated worktree and report branch
  test/phase2-TEST_ID-windows-report from TEST_COMMIT.
- Create a new report branch and draft PR for this exact commit. Do not update,
  overwrite, merge, or close an older Phase 2 report branch or PR.
- At completion, commit only that report, push the report branch, and
  create or update one draft PR. Never merge, tag, release, deploy, or modify a
  product branch.
- Prove the staged diff contains only the report and no local index, vendor
  session, fixture copy, generated binary, transcript, or private path.

Hard safety and privacy:
- Keep normal sandboxing and approval controls enabled. Never use a
  permission-bypass flag.
- Do not read, print, summarize, screenshot, copy, commit, or upload unrelated
  transcript content, assistant reasoning/messages, tool output, environment
  dumps, auth files, tokens, cookies, .env files, keychains, or credentials.
- Inspect only before/after vendor metadata, exact controlled marker presence,
  exact challenge-response output, composite references, bounded Reinstate
  metadata, and source fingerprints needed by the runbook.
- A marker needs at least one exact occurrence; do not require a fixed
  serialization count.
- Never alter a vendor session file to manufacture indexing, ambiguity,
  malformed input, an incomplete line, or a fork. Use deterministic synthetic
  tests for destructive edge cases.
- Never delete an older Reinstate home, real agent session, or unrelated
  project. Cleanup is deferred until both reports are reviewed.
- Report only non-secret versions, composite controlled references, counts,
  booleans, exit codes, bounded controlled markers, redacted relative paths,
  and sanitized errors.

Local-only boundary:
- Create a fresh absolute REINSTATE_HOME and the two disposable projects
  prescribed by the runbook. Stop if any target already exists.
- Never run rein init or any sync/storage/conflict command.
- Do not create or use R2.txt, a sync profile, storage environment values,
  credentials, an encryption passphrase, or a keyring entry.
- Phase 2 local commands must succeed without config or a backend. The only
  permitted Reinstate state is the private derived index under
  cache/session-index-v1.sqlite.
- If any Phase 2 command requests setup, storage, a credential, a passphrase,
  keyring access, or a network backend, record FAIL and stop dependent gates.

Controlled data:
- Use harmless new disposable Git repositories, branches, relative files, and
  unique markers derived from TEST_ID.
- Create and cleanly close exactly one fresh Claude Code session and one fresh
  Codex session through their documented non-interactive invocation/resume
  modes. Identify the native IDs by before/after metadata and controlled
  markers only.
- If Gemini CLI or OpenCode is already installed, create one harmless
  disposable session for its read-only path. Do not install a missing vendor
  solely for this run.
- Do not reuse an older acceptance session or include unrelated listing rows in
  the report.
- This is a complete fresh run at a new product commit. Do not carry forward a
  PASS, command result, session reference, or source fingerprint from any
  report against `5c60ec2` or another commit.
- OpenCode is installed on this device and row 29 is the regression target.
  Do not touch or inspect the preserved pre-rebuild OpenCode database backup,
  and do not read or print OpenCode or Gemini authentication material.

Milestone W0 — provenance, environment, and automated gates:
1. Record TEST_COMMIT, tag state, Reinstate version JSON,
   Windows edition/build/architecture, native PowerShell, Git, Go, and all four
   vendor versions/compatibility states.
2. Build the exact source. First run
   `go test ./internal/sessionindex -run OpenCode -count=1`, then run every
   applicable automated and cross-build gate in runbook section 5, including
   the complete merge gate.
3. Record exact failed commands, exit codes, and sanitized output. Do not call
   a skipped or interrupted command PASS.

Milestone W1 — configless index, refresh, search, and inspect:
4. Create the isolated home/corpus and prove no config, state, backup,
   credential reference, or index exists before the first command.
5. Execute runbook sections 6 through 9 completely: sessions/list
   compatibility, rein/reinstate JSON parity, ordering, permissions,
   idempotency, all search dimensions, bounded inspect privacy, append refresh,
   and new-session refresh.
6. Prove indexing/search/inspect do not modify a vendor source and do not
   create anything except derived cache state.

Milestone W2 — last, native resume, and native fork:
7. Execute dry-run JSON plans before every real launch in runbook sections 10
   and 11. Verify executable, argv array, cwd, no launch, and no mutation.
8. Prove exact same-vendor Claude and Codex resume with controlled challenge
   responses. Distinguish vendor writes after launch from Reinstate mutation.
9. Prove vendor-native fork for both agents, source preservation, distinct
   identity, refresh discovery, independent resume, and branch independence.
10. Run missing/ambiguous reference, missing workspace/executable,
    JSON-without-dry-run, child-failure, and read-only-agent refusal gates
    without damaging real state.

Milestone W3 — interactive switcher and read-only adapters:
11. Through a real native console/PTY, test bare rein and reinstate: /text
    filtering, i NUMBER inspect, NUMBER resume, f NUMBER fork, invalid input, q
    cancel, EOF/interrupt, and unrelated-transcript privacy.
12. Prove non-TTY invocation exits promptly with usage code 2 and a
    rein sessions --json hint.
13. When installed, prove Gemini/OpenCode discovery, search, inspect, and
    read-only capability metadata. When absent, record NOT_INSTALLED and only
    the optional physical row as NOT TESTED. Their fixture/fake-runner tests
    still must pass. OpenCode row 29 must use one newly created controlled
    session and prove all of the following: its `updated_at` is non-zero and not
    year 1; its timestamp matches the vendor's top-level `updated` or `created`
    epoch metadata; it appears in the unfiltered default
    `rein sessions --json` result without `--agent opencode`; explicit agent
    filtering, literal ID search, inspect, and read-only resume/fork refusal
    still behave as specified; no vendor process remains; and no unrelated
    vendor session content or authentication material is inspected.

Milestone W4 — report:
14. Copy the report template to the resolved Windows report path and complete
    every section. Fill only the Windows column of the 32-row matrix; do not
    inherit or wait for Mac results, and do not copy evidence from a report
    against another commit.
15. Include one complete PHASE2-DEVICE-REPORT-V1 block with device=windows and
    its END-PHASE2-DEVICE-REPORT-V1 terminator.
16. Commit/push only the report, create or update the draft PR, and return:
    verdict and counts; whether every required local row passed; report path,
    commit, branch, and PR; release-blocking/non-blocking findings; every failed
    command/exit/sanitized output; and confirmation that no product file,
    transcript, private path, or secret was committed.

Device B does not perform final cross-device reconciliation. Return the
complete latest report for transfer to the existing Mac coordinator. Phase 2
cannot be called complete until that coordinator validates both reports.
```

## Handoff order

1. Mac and Windows start in parallel from the exact commit
   `b952d38c2dc57b0a96bc696860318ea7c2975800`.
2. Mac produces its complete device report and pauses.
3. Windows produces its complete device report.
4. The human passes the complete latest Windows report to the existing Mac
   Claude session.
5. Mac validates both report branches and produces the single final
   reconciliation.

Do not clean up the isolated homes, derived indexes, disposable projects, or
controlled test sessions until both reports and the final reconciliation have
been reviewed.
