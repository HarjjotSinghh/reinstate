# Phase 1 RC4 two-agent verification prompts

Use these prompts only after `v0.1.0-rc.4` is published and both
`https://reinstate.dev/install.sh` and
`https://reinstate.dev/install.ps1` install that exact release.

The tagged
[Phase 1 acceptance runbook](phase-1-mac-windows-acceptance.md) is the
authority. These prompts split the work between:

- Device A: Claude Code on macOS
- Device B: Codex on native 64-bit Windows PowerShell

Both agents write evidence reports on separate branches. They must never
receive storage credentials, passphrases, transcript content, or downloaded
ciphertext through chat.

## Prompt 1 — Claude Code on the MacBook

Copy everything inside this block into the existing Claude Code session on the
MacBook:

```text
Run the complete Device A side of Reinstate Phase 1 acceptance against the
published v0.1.0-rc.4. You are the macOS evidence owner and cross-device
coordinator. Do not delegate this work or modify Reinstate product code.

Authority:
- Fetch origin and the signed tag v0.1.0-rc.4.
- Read docs/testing/phase-1-mac-windows-acceptance.md from that exact tag.
- Follow it literally. If this prompt and the tagged runbook conflict, stop and
  report the conflict.
- The only allowed repository change is your sanitized report:
  docs/testing/results/2026-07-26-macos-phase1-rc4.md

Hard safety rules:
- Never ask for, read, paste, echo, log, screenshot, or commit an R2/S3 secret,
  encryption passphrase, keyring value, agent auth file, transcript, or
  downloaded .age object.
- I will type credentials and passphrases only into Reinstate's visible hidden
  prompts in my private terminal.
- Before telling me to enter a passphrase, confirm the Reinstate process is
  still visibly waiting for hidden input. If it has exited, tell me to stop.
- Never use --all. Use only the two fresh RC4 marker session IDs.
- Never delete or mutate an RC3 home, RC3 profile prefix, real agent session,
  bucket, or unrelated project.
- Never manually move a restored Claude file to make discovery pass.
- Keep normal approvals and sandboxing enabled.

Isolation for this run:
- REINSTATE_HOME=$HOME/.reinstate-phase1-acceptance-rc4
- project=$HOME/Projects/reinstate-phase1-acceptance-rc4
- canonical project ID=local/reinstate-phase1-acceptance-rc4
- Create a brand-new RC4 profile and passphrase. Reusing any RC3 profile,
  state.json, passphrase, snapshot, or marker session is a test failure.

Milestone M0 — release and environment:
1. Verify the tag is an annotated signed tag on origin/main and record the tag
   commit without changing trust configuration merely to force a pass.
2. Verify the public install.sh returns HTTP 200, pins rc.4, passes both
   checksum layers, installs 0.1.0-rc.4, is idempotent, and does not duplicate a
   PATH entry. Record only sanitized output and exit codes.
3. Record macOS version/architecture, Claude Code version, Codex CLI version,
   Git version, and `rein version --json`.
4. In the fresh isolated home, prove pre-init `rein setup check` exits 3 with
   `config missing`, while the device, keyring, and both installed adapters do
   not falsely pass an unsupported state.

Milestone M1 — fresh source, push, and ciphertext:
5. Create the fresh disposable mapped project and a new harmless session in
   each agent with these exact markers:
   - REINSTATE-PHASE1-RC4-MAC-CLAUDE-A1
   - REINSTATE-PHASE1-RC4-MAC-CODEX-A1
6. Identify fresh IDs through before/after metadata. If Claude writes sibling
   candidates, count only exact marker occurrences to select the completed
   reply; do not print or inspect transcript prose. If Codex fails to persist a
   new rollout, stop and report the vendor failure instead of reusing an old
   session.
7. Execute docs/prompts/claude-code-setup.md version 4 as an end-user would.
   Have me run the private `rein init` with the canonical mapping. I will enter
   endpoint, bucket, credentials, and a new passphrase privately.
8. Require post-init `rein setup check` and `rein doctor --self-test` to pass.
9. Dry-run and push only the selected Claude and Codex IDs. The human dry-run
   output must say `would push`, not `pushed`. `rein status` must show exactly
   two remote sessions.
10. Guide me through downloading one snapshot .age object from only the fresh
    RC4 profile prefix. On my Mac, test both exact A1 marker strings for
    absence and run `file` without showing you the object or its bytes. Record
    booleans only. Confirm no credential-shaped remote object exists.
11. Emit this sanitized handoff and then pause for the Windows W1 result:

MAC-RC4-M1
release=v0.1.0-rc.4
tag_commit=<non-secret sha>
profile_id=<non-secret fresh rc4 uuid>
canonical_project_id=local/reinstate-phase1-acceptance-rc4
claude_session_id=<non-secret fresh uuid>
codex_session_id=<non-secret fresh uuid>
remote_session_count=2
ciphertext_marker_absence=PASS|FAIL
mac_report_path=docs/testing/results/2026-07-26-macos-phase1-rc4.md
END-MAC-RC4-M1

Do not include the endpoint, bucket, credentials, passphrase, transcript text,
local username, or downloaded object path in the handoff.

Milestone M2 — after Windows reports both A1 restores/resumes PASS:
12. Resume the exact Claude session on Mac and add
    REINSTATE-PHASE1-RC4-MAC-CLAUDE-A2. Exit Claude and dry-run/push only that
    ID. Send WINDOWS-RC4-M2-READY with the Claude session ID and new remote
    snapshot/revision IDs only.

Milestone M3 — after Windows completes active-agent refusal, backup, and A2:
13. Wait while Windows adds and pushes both B1 markers:
    - REINSTATE-PHASE1-RC4-WINDOWS-CLAUDE-B1
    - REINSTATE-PHASE1-RC4-WINDOWS-CODEX-B1
14. Close both Mac agents. Dry-run and pull each exact ID. Prove each existing
    target received a timestamped backup. Resume exact IDs and have me visually
    confirm B1 without copying transcript content.
15. Without changing either restored session, push each exact ID and require
    `pushed 0 snapshot(s), skipped 1 unchanged`; verify no new remote revision.
16. Send MAC-RC4-M3-PASS or a precise sanitized failure.

Milestone M4 — conflict coordination:
17. Follow runbook section 17 exactly. Add only the Mac conflict marker and
    push when the Windows-local divergence is ready. Do not resolve the Windows
    conflict yourself.
18. Receive the Windows keep-both result, reconcile every row in section 19,
    and finish the Mac report. Mark unexecuted evidence NOT TESTED, never PASS.

Report and Git workflow:
- Preserve command, exit-code, version, snapshot/revision, backup, discovery,
  and resume evidence without transcript content or secrets.
- Record every deviation and classify product defects separately from vendor,
  operator, or test-state problems.
- Create an isolated report branch from the rc.4 tag commit or current
  origin/main if it contains that exact commit:
  test/phase1-rc4-macos-report
- Commit only the report with:
  test(acceptance): record macOS phase 1 rc4 results
- Push the branch and open a draft PR. Do not merge, tag, release, deploy, or
  modify the product branch.
- Final output must include verdict, report path, commit SHA, branch, draft PR,
  release-blocking findings, non-blocking findings, and whether all 21 section
  19 rows passed.

Phase 1 is PASS only if every mandatory row passes on real devices. Zero-BS:
successful `rein pull` output alone is not proof; vendor discovery, exact-ID
resume, correct destination path, backups, no-op behavior, conflicts, and
ciphertext checks all require evidence.
```

## Prompt 2 — Codex on the Windows PC

Start this prompt only after providing Codex the sanitized `MAC-RC4-M1`
handoff:

```text
Run the complete Device B side of Reinstate Phase 1 acceptance against the
published v0.1.0-rc.4. You are the native-Windows evidence owner. Do not
delegate this work or modify Reinstate product code.

Authority:
- Fetch origin and the signed tag v0.1.0-rc.4.
- Read docs/testing/phase-1-mac-windows-acceptance.md from that exact tag.
- Follow it literally. Stop and report any conflict with this prompt.
- The only allowed repository change is your sanitized report:
  docs/testing/results/2026-07-26-windows-phase1-rc4.md

Required sanitized Mac handoff:
- profile_id=<fresh RC4 UUID>
- canonical_project_id=local/reinstate-phase1-acceptance-rc4
- claude_session_id=<fresh RC4 UUID>
- codex_session_id=<fresh RC4 UUID>
- remote_session_count=2
- ciphertext_marker_absence=PASS

Stop if the handoff is missing, refers to RC3, reuses an old profile, reports
anything except exactly two remote sessions, or lacks a real ciphertext-byte
check.

Hard safety rules:
- Use native 64-bit Windows PowerShell 5.1 or newer, not WSL.
- Never ask for, read, paste, echo, log, screenshot, or commit an R2/S3 secret,
  encryption passphrase, keyring value, agent auth file, transcript, or .age
  object.
- I will type credentials and passphrases only into Reinstate's visible hidden
  prompts in my private PowerShell.
- Before telling me to enter a passphrase, confirm Reinstate is still visibly
  waiting for hidden input. If the process returned to PowerShell, tell me to
  stop and rerun it. Never let a passphrase become a shell command.
- Never use --all. Never reuse or delete an RC3 home/profile/prefix/session.
- Never manually move a restored Claude file to make discovery pass.
- Keep normal approvals and sandboxing enabled.

Isolation:
- REINSTATE_HOME=$HOME\.reinstate-phase1-acceptance-rc4
- project=$HOME\Projects\reinstate-phase1-acceptance-rc4
- canonical project ID=local/reinstate-phase1-acceptance-rc4
- Initialize this home with the exact fresh profile UUID from MAC-RC4-M1.

Milestone W0 — release and environment:
1. Verify the tag is an annotated signed tag on origin/main and record the tag
   commit without weakening trust configuration.
2. Verify install.ps1 returns HTTP 200, pins rc.4, passes both checksum layers,
   installs 0.1.0-rc.4 without elevation, is idempotent, and leaves exactly one
   normalized user PATH entry.
3. Record Windows edition/build/architecture, native PowerShell identity,
   Claude Code version, Codex CLI version, Git version, and
   `rein version --json`.
4. In the fresh isolated home, prove pre-init `rein setup check` exits 3 with
   `config missing`, while device, keyring, and both installed adapters do not
   falsely pass unsupported states.

Milestone W1 — init, negative test, and Mac-to-Windows restore:
5. Execute docs/prompts/codex-setup.md version 4 as an end-user would. Have me
   run the private `rein init --profile-id ... --project ...` command with the
   exact canonical mapping and Windows absolute path.
6. Before any correct-passphrase command, run `rein status`, wait until the
   hidden prompt is visibly active, and have me enter one deliberately wrong
   passphrase. Require exit 4, explicit decryption refusal, zero target changes,
   and zero new backups. After the process exits, do not ask me to type
   anything secret until a newly run command visibly prompts.
7. Rerun status with the correct passphrase and require exactly two sessions.
   Require post-init `rein setup check` and `rein doctor --self-test` to pass.
8. With Claude Code and Codex closed, dry-run then pull only the exact Codex
   ID and exact Claude ID. Dry-runs must create no target or backup.
9. Prove Codex with `codex resume CODEX_SESSION_ID`. Do not use picker search;
   it can match UUID text inside another session.
10. Prove Claude in both ways:
    - from the mapped Windows project, `claude --resume` must show the selected
      ID in normal discovery;
    - `claude --resume CLAUDE_SESSION_ID` must resume the exact session.
    The restored path must be under the Windows project directory key, never
    the Mac source slug. Do not manually relocate it.
11. Have me visually confirm both A1 markers without copying transcript text.
12. Emit WINDOWS-RC4-W1-PASS with sanitized path booleans, IDs, exits, backup
    counts, and report path. On any failure, stop before section 14 and emit a
    precise finding.

Milestone W2 — after MAC-RC4-M2-READY:
13. Leave Claude open on the selected session and run the A2 pull from a
    separate PowerShell. Require safety exit 7 and no target mutation.
14. Close every Claude process, dry-run/pull again, prove a timestamped backup
    of the previous Windows target exists, and have me visually confirm A2.
15. Resume the exact selected Claude and Codex sessions, add:
    - REINSTATE-PHASE1-RC4-WINDOWS-CLAUDE-B1
    - REINSTATE-PHASE1-RC4-WINDOWS-CODEX-B1
    Exit both agents, dry-run and push only the exact two IDs, then send
    WINDOWS-RC4-W2-READY with non-secret snapshot/revision IDs.

Milestone W3 — after MAC-RC4-M3-PASS:
16. Create the Windows-local Claude divergence with
    REINSTATE-PHASE1-RC4-CONFLICT-WINDOWS and do not push it.
17. Tell the Mac executor to add/push its conflict marker. Then pull the exact
    Claude ID on Windows. Require exit 6, one conflict record, and no overwrite
    of the Windows-local session.
18. Inspect only conflict metadata. Resolve with --keep-both. Prove:
    - the original Windows-local session still exists;
    - a distinct vendor-safe remote fork exists under the Windows mapped
      project directory;
    - both exact IDs are discoverable;
    - the conflict disappears only after successful resolution.
19. Reconcile every section 19 row with the Mac report. Mark unexecuted rows
    NOT TESTED, never PASS.

Report and Git workflow:
- Preserve command, exit-code, version, snapshot/revision, backup, discovery,
  resume, no-op, and conflict evidence without secrets or transcript content.
- Separate Reinstate defects from vendor, operator, and test-state findings.
- Create an isolated report branch from the rc.4 tag commit or current
  origin/main if it contains that exact commit:
  test/phase1-rc4-windows-report
- Commit only the report with:
  test(acceptance): record Windows phase 1 rc4 results
- Push and open a draft PR. Do not merge, tag, release, deploy, delete remote
  prefixes, or modify the product branch.
- Final output must include verdict, report path, commit SHA, branch, draft PR,
  release-blocking findings, non-blocking findings, and whether all 21 section
  19 rows passed.

Phase 1 is PASS only when all 21 mandatory rows have real evidence across both
devices. A zero exit from Reinstate is necessary, not sufficient. If Claude or
Codex cannot discover and resume the exact restored session from the mapped
Windows project, the verdict is FAIL.
```

## Handoff order

1. Mac completes `MAC-RC4-M1`.
2. Windows completes `WINDOWS-RC4-W1-PASS`.
3. Mac completes `MAC-RC4-M2-READY`.
4. Windows completes `WINDOWS-RC4-W2-READY`.
5. Mac completes `MAC-RC4-M3-PASS` and prepares the remote conflict branch.
6. Windows completes keep-both, then both agents finalize their reports.

Do not clean up the fresh profile or acceptance paths until both reports have
been reviewed and Phase 1 has been signed off.
