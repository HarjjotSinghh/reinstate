# Phase 1 RC5 two-agent full acceptance prompts

Use these prompts only after `v0.1.0-rc.5` is published and both public
installers install that exact release. Do not use RC4 homes, profiles,
passphrases, marker sessions, or reports as RC5 evidence.

The
[Phase 1 MacBook + Windows acceptance runbook](phase-1-mac-windows-acceptance.md)
from the signed RC5 tag is the authority. These prompts split evidence
ownership between:

- Device A: Claude Code on macOS
- Device B: Codex on native 64-bit Windows PowerShell

Both agents must preserve `PASS`, `PARTIAL`, `FAIL`, and `NOT TESTED` honestly.
A zero exit code is never sufficient evidence by itself. Neither agent may
receive storage credentials, passphrases, transcript content, or ciphertext
bytes through chat.

Before starting, replace `REPORT_DATE` in each prompt with the same UTC
`YYYY-MM-DD` date. Do not replace any secret placeholder in chat.

## Prompt 1 — Claude Code on the MacBook

Copy everything inside this block into a new Claude Code session launched from
the fresh RC5 disposable project:

```text
Run the complete Device A side of Reinstate Phase 1 acceptance against the
published v0.1.0-rc.5. You are the macOS evidence owner and cross-device
coordinator. Do not delegate this work or modify Reinstate product code.

Authority and repository scope:
- Fetch origin and the signed tag v0.1.0-rc.5.
- Read docs/testing/phase-1-mac-windows-acceptance.md from that exact tag.
- Follow it literally. If this prompt and the tagged runbook conflict, stop and
  report the conflict.
- The only allowed repository change is the sanitized report:
  docs/testing/results/REPORT_DATE-macos-phase1-rc5.md
- Use a dedicated report branch: test/phase1-rc5-macos-report.
- Create that branch from the peeled v0.1.0-rc.5 tag commit before editing the
  report.
- Do not merge, tag, release, deploy, or modify a product branch.

Hard safety rules:
- Never ask for, read, paste, echo, log, screenshot, hash for disclosure, or
  commit an endpoint, bucket, access key, secret key, encryption passphrase,
  keyring value, agent auth file, transcript, downloaded .age object, or
  ciphertext bytes.
- I will enter credentials and passphrases only into Reinstate's visible hidden
  prompts in my private terminal.
- Before telling me to type a passphrase, confirm the Reinstate process is
  visibly waiting for hidden input. If it has returned to the shell, tell me to
  type nothing and rerun the command.
- Never use --all. Operate only on the two fresh RC5 marker session IDs.
- Never delete or mutate an RC3/RC4 home, profile prefix, report, real agent
  session, unrelated project, or unrelated remote object.
- Never manually move a restored vendor file to manufacture discovery.
- Keep normal approvals and sandboxing enabled.
- Record only non-secret IDs, counts, booleans, versions, exit codes, redacted
  paths, and sanitized error text.

Isolation:
- REINSTATE_HOME=$HOME/.reinstate-phase1-acceptance-rc5
- project=$HOME/Projects/reinstate-phase1-acceptance-rc5
- canonical project ID=local/reinstate-phase1-acceptance-rc5
- Create a brand-new RC5 profile and passphrase.
- Stop if either isolated path already exists or if any RC3/RC4 state would be
  reused.

Milestone M0 — release and environment:
1. Verify v0.1.0-rc.5 is an annotated signed tag reachable from origin/main.
   Record its commit without changing trust configuration to force a pass.
2. Verify https://reinstate.dev/install.sh returns HTTP 200, pins only rc.5,
   verifies both checksum layers, installs 0.1.0-rc.5, and uses no elevation.
   Exercise the replacement prompt interactively if an older version exists.
   Run the installer again and prove idempotency and one PATH entry.
3. Record macOS version/architecture, native shell, Claude Code version, Codex
   CLI version, Git version, and `rein version --json`.
4. Create the isolated project and home exactly as the tagged runbook says.
   Prove pre-init `rein setup check` exits 3 with `config missing`; device and
   both adapters must not falsely pass an unsupported state.

Milestone M1 — source sessions, init safety, push, and ciphertext:
5. Create one harmless fresh session per agent with these exact markers:
   - REINSTATE-PHASE1-RC5-MAC-CLAUDE-A1
   - REINSTATE-PHASE1-RC5-MAC-CODEX-A1
6. Identify both fresh IDs through before/after metadata. If Claude writes
   sibling candidates, use marker occurrence counts only to select the
   completed reply; do not read or print prose. If Codex does not persist a new
   rollout, stop instead of reusing an older session.
7. Execute the exact tagged docs/prompts/claude-code-setup.md (Prompt version
   5) as an end user. Have me run the private `rein init` with the canonical
   mapping. I will enter storage coordinates, credentials, and passphrase
   privately.
8. Require post-init `rein setup check` and `rein doctor --self-test` to exit 0.
9. Re-run the same `rein init --project ...` without `--force`. Before and
   after, have my private terminal calculate config.toml and state.json hashes
   and count backup sets. Record only equality/count booleans. Require safety
   exit 7, unchanged config/state, and no new backup. This is the physical F1
   default-refusal regression; do not run `--force` against the real RC5 home.
10. Dry-run and push only the selected Claude and Codex IDs. Each dry-run must
    say `would push`, not `pushed`. `rein status` with the correct passphrase
    must show exactly two sessions and both selected IDs.
11. Through the normal R2/S3 UI, have me inspect only the fresh RC5 prefix.
    Confirm no auth, token, credential, .env, or plaintext-shaped object exists.
    Have me download one .age snapshot privately, test both exact A1 marker
    strings for absence, and run `file`. Record only two marker-absence
    booleans and a ciphertext/non-text boolean. Delete the local download
    privately after the check.
12. Emit this sanitized handoff, update the report, and pause:

MAC-RC5-M1
release=v0.1.0-rc.5
tag_commit=<non-secret sha>
profile_id=<non-secret fresh rc5 uuid>
canonical_project_id=local/reinstate-phase1-acceptance-rc5
claude_session_id=<non-secret fresh uuid>
codex_session_id=<non-secret fresh uuid>
remote_session_count=2
f1_default_refusal=PASS|FAIL
ciphertext_marker_absence=PASS|FAIL
mac_report_path=docs/testing/results/REPORT_DATE-macos-phase1-rc5.md
END-MAC-RC5-M1

Do not include endpoint, bucket, credentials, passphrase, transcript text,
username, absolute local paths, or object names in the handoff.

Milestone M2 — after WINDOWS-RC5-W1-PASS:
13. Resume the exact Claude session on Mac and add:
    REINSTATE-PHASE1-RC5-MAC-CLAUDE-A2
    Exit Claude, dry-run, and push only that ID.
14. Send MAC-RC5-M2-READY with the Claude session ID plus the new non-secret
    snapshot and revision IDs. Pause until Windows completes W2.

Milestone M3 — Windows-to-Mac updates:
15. After WINDOWS-RC5-W2-READY, close both Mac agents. Dry-run and pull each
    exact ID.
16. Prove each existing target received a timestamped backup. Resume the exact
    Claude and Codex IDs and have me visually confirm:
    - REINSTATE-PHASE1-RC5-WINDOWS-CLAUDE-B1
    - REINSTATE-PHASE1-RC5-WINDOWS-CODEX-B1
    Do not copy transcript content.
17. Without modifying either restored session, push each exact ID and require
    `pushed 0 snapshot(s), skipped 1 unchanged`; prove the remote revision did
    not change.
18. Send MAC-RC5-M3-PASS with sanitized backup, resume, no-op, and revision
    evidence. Pause for conflict coordination.

Milestone M4 — divergence and final verdict:
19. Follow tagged runbook section 17 exactly. Add only the Mac conflict marker
    when Windows confirms its unpushed local divergence is ready. Push only the
    selected Claude ID. Do not resolve the Windows conflict yourself.
20. Receive the Windows keep-both result. Reconcile every mandatory section 19
    row across both reports. An unexecuted row is NOT TESTED, never PASS.
21. Commit only the report:
    test(acceptance): record macOS phase 1 rc5 results
    Push the report branch and open a draft PR. Do not merge it.

Final response contract:
- verdict and `x PASS / y PARTIAL / z FAIL / n NOT TESTED`;
- whether all 21 mandatory rows passed;
- report path, report commit, branch, and draft PR;
- release-blocking and non-blocking findings;
- exact failed command, exit code, and sanitized output for every failure;
- confirmation that no product code or secrets were committed.

Phase 1 is PASS only if all mandatory rows have real-device evidence. A
successful `rein pull` line alone proves nothing without exact-ID vendor
discovery, same-vendor resume, destination mapping, backups, no-op behavior,
conflict safety, and ciphertext-only evidence.
```

## Prompt 2 — Codex on the Windows PC

Start this prompt only after providing Codex the sanitized `MAC-RC5-M1`
handoff:

```text
Run the complete Device B side of Reinstate Phase 1 acceptance against the
published v0.1.0-rc.5. You are the native-Windows evidence owner. Do not
delegate this work or modify Reinstate product code.

Authority and repository scope:
- Fetch origin and the signed tag v0.1.0-rc.5.
- Read docs/testing/phase-1-mac-windows-acceptance.md from that exact tag.
- Follow it literally. Stop and report any conflict with this prompt.
- The only allowed repository change is the sanitized report:
  docs/testing/results/REPORT_DATE-windows-phase1-rc5.md
- Use a dedicated report branch: test/phase1-rc5-windows-report.
- Create that branch from the peeled v0.1.0-rc.5 tag commit before editing the
  report.
- Do not merge, tag, release, deploy, or modify a product branch.

Required sanitized Mac handoff:
- release=v0.1.0-rc.5
- profile_id=<fresh RC5 UUID>
- canonical_project_id=local/reinstate-phase1-acceptance-rc5
- claude_session_id=<fresh RC5 UUID>
- codex_session_id=<fresh RC5 UUID>
- remote_session_count=2
- f1_default_refusal=PASS
- ciphertext_marker_absence=PASS

Stop if the handoff is incomplete, references RC3/RC4, reuses an old profile,
reports anything except exactly two sessions, or lacks a real ciphertext-byte
check.

Hard safety rules:
- Use native 64-bit Windows PowerShell 5.1 or newer, not WSL.
- Never ask for, read, paste, echo, log, screenshot, hash for disclosure, or
  commit an endpoint, bucket, access key, secret key, passphrase, keyring value,
  agent auth file, transcript, .age object, or ciphertext bytes.
- I will enter credentials and passphrases only into Reinstate's visible hidden
  prompts in my private PowerShell.
- Before telling me to type a passphrase, confirm Reinstate is visibly waiting
  for hidden input. If it returned to PowerShell, tell me to type nothing and
  rerun the command.
- Never use --all. Never reuse or delete RC3/RC4 state or unrelated sessions.
- Never manually move a restored Claude file to manufacture discovery.
- Keep normal approvals and sandboxing enabled.
- Record only non-secret IDs, counts, booleans, versions, exits, redacted paths,
  and sanitized error text.

Isolation:
- REINSTATE_HOME=$HOME\.reinstate-phase1-acceptance-rc5
- project=$HOME\Projects\reinstate-phase1-acceptance-rc5
- canonical project ID=local/reinstate-phase1-acceptance-rc5
- Stop if either path exists before the run.

Milestone W0 — release and environment:
1. Verify v0.1.0-rc.5 is an annotated signed tag reachable from origin/main.
2. Verify https://reinstate.dev/install.ps1 returns HTTP 200, pins only rc.5,
   passes both checksum layers, installs 0.1.0-rc.5 without elevation, and is
   idempotent. Prove the normalized user PATH contains the install directory
   exactly once.
3. Record Windows edition/build/architecture, native PowerShell identity,
   Claude Code version, Codex CLI version, Git version, and
   `rein version --json`.
4. Create the fresh isolated project/home. Require pre-init
   `rein setup check` exit 3 with `config missing`; device and both adapters
   must not falsely pass unsupported states.

Milestone W1 — F3/F1/F2 regressions and Mac-to-Windows restore:
5. Physical F3 negative test: execute the exact tagged
   docs/prompts/codex-setup.md (Prompt version 5), but first have me privately
   attempt `rein init --profile-id PROFILE_ID --project ...` with the endpoint
   mistakenly containing the bucket suffix while also entering the bucket in
   its own prompt. The agent must not see either value. Require auth/storage
   exit 4, an actionable remote-profile-not-found error, and no config.toml.
   Record booleans only.
6. Repeat init with the correct endpoint-only value, same profile ID, canonical
   mapping, and Windows absolute project path. Require success. Then require
   `rein setup check` and `rein doctor --self-test` to pass.
7. Physical F1 default-refusal test: re-run the same correct init without
   `--force`. Have my private PowerShell compare config.toml/state.json hashes
   and backup counts before/after. Require safety exit 7, unchanged files, and
   no new backup. Never run `--force` against the real RC5 home.
8. Wrong-passphrase test: run `rein status`, wait for the visible hidden prompt,
   and have me enter one deliberately wrong passphrase. Require exit 4,
   decryption refusal, zero restore targets, zero backups, and no mutation.
9. Correct-passphrase status must show exactly the two selected sessions.
10. Physical F2 strict-status test in a disposable copied home:
    Have me run the following in private PowerShell after replacing only the
    non-secret UUID placeholder. Do not request or print the copied config:

    $ProfileId = "<PROFILE_ID_FROM_MAC_HANDOFF>"
    $RealHome = $env:REINSTATE_HOME
    $ProbeHome = "$RealHome-missing-manifest-probe"
    if (Test-Path -LiteralPath $ProbeHome) {
      throw "probe home already exists; stop to preserve evidence isolation"
    }
    Copy-Item -Recurse -LiteralPath $RealHome -Destination $ProbeHome
    $ProbeConfig = Join-Path $ProbeHome "config.toml"
    $OldPrefix = "profiles/$ProfileId"
    $NewPrefix = "profiles/$ProfileId-missing-manifest-probe"
    $ConfigText = [IO.File]::ReadAllText($ProbeConfig)
    if ([regex]::Matches($ConfigText, [regex]::Escape($OldPrefix)).Count -ne 1) {
      throw "expected exactly one profile prefix; stop"
    }
    $Utf8NoBom = [Text.UTF8Encoding]::new($false)
    [IO.File]::WriteAllText(
      $ProbeConfig,
      $ConfigText.Replace($OldPrefix, $NewPrefix),
      $Utf8NoBom
    )
    try {
      $env:REINSTATE_HOME = $ProbeHome
      rein status
      $ProbeExit = $LASTEXITCODE
    } finally {
      $env:REINSTATE_HOME = $RealHome
    }
    "probe_exit=$ProbeExit"

    Enter the correct passphrase only at the visible hidden prompt. Require
    exit 4 and `remote profile manifest not found`, not exit 0 with zero
    sessions. Prove the real RC5 home and remote objects are unchanged. Keep
    the probe home until both reports are reviewed; do not delete it mid-run.
11. With Claude Code and Codex closed, dry-run then pull only the exact Codex
    and Claude IDs. Dry-runs must create no target and no backup.
12. Prove Codex with `codex resume CODEX_SESSION_ID`; do not use picker search.
13. From the mapped Windows project, prove Claude normal discovery contains the
    exact selected ID, then run `claude --resume CLAUDE_SESSION_ID`. The
    restored target must be under the Windows project directory key, never the
    Mac source slug. Do not relocate it.
14. Have me visually confirm both A1 markers without copying transcript text.
15. Emit this sanitized handoff and pause:

WINDOWS-RC5-W1-PASS
release=v0.1.0-rc.5
profile_id=<non-secret fresh rc5 uuid>
claude_session_id=<non-secret uuid>
codex_session_id=<non-secret uuid>
f3_bad_coordinates_refused=PASS
f1_default_refusal=PASS
f2_missing_manifest_refused=PASS
wrong_passphrase_refused=PASS
remote_session_count=2
claude_discovery_and_resume=PASS
codex_resume=PASS
windows_report_path=docs/testing/results/REPORT_DATE-windows-phase1-rc5.md
END-WINDOWS-RC5-W1

On any failure, emit WINDOWS-RC5-W1-FAIL with the exact command, exit code, and
sanitized output, then stop before W2.

Milestone W2 — active-agent safety, backup, and Windows updates:
16. After MAC-RC5-M2-READY, leave Claude open on the selected session and run
    the A2 pull from separate PowerShell. Require safety exit 7, no mutation,
    and no backup.
17. Close every Claude process, dry-run/pull again, prove one timestamped backup
    of the previous target exists, and have me visually confirm A2.
18. Resume exact selected sessions and add:
    - REINSTATE-PHASE1-RC5-WINDOWS-CLAUDE-B1
    - REINSTATE-PHASE1-RC5-WINDOWS-CODEX-B1
    Exit both agents, dry-run, and push only the exact two IDs.
19. Send WINDOWS-RC5-W2-READY with only non-secret session, snapshot, revision,
    backup, and count evidence. Pause for MAC-RC5-M3-PASS.

Milestone W3 — conflict and final report:
20. Create Windows-local Claude divergence with:
    REINSTATE-PHASE1-RC5-CONFLICT-WINDOWS
    Do not push it. Tell the Mac executor to add/push its conflict marker.
21. Pull the exact Claude ID. Require conflict exit 6, one conflict record, no
    overwrite, and no premature backup.
22. Inspect conflict metadata only. Resolve with --keep-both and prove:
    - the original Windows-local session remains intact;
    - a distinct vendor-safe remote fork exists under the mapped Windows
      project;
    - both exact IDs are discoverable and resumable; and
    - the conflict disappears only after successful resolution.
23. Reconcile every mandatory section 19 row with the Mac report. Unexecuted
    evidence is NOT TESTED, never PASS.
24. Commit only the report:
    test(acceptance): record Windows phase 1 rc5 results
    Push the report branch and open a draft PR. Do not merge it.

Final response contract:
- verdict and `x PASS / y PARTIAL / z FAIL / n NOT TESTED`;
- whether all 21 mandatory rows passed;
- report path, commit, branch, and draft PR;
- release-blocking and non-blocking findings;
- exact failed command, exit, and sanitized output for every failure;
- confirmation that no product code or secrets were committed.

Phase 1 is PASS only when all 21 rows have real evidence across both devices.
Correct exit codes are necessary, not sufficient. Claude and Codex must
discover and resume the exact restored sessions from the mapped Windows project.
```

## Handoff order

1. Mac completes `MAC-RC5-M1`.
2. Windows completes `WINDOWS-RC5-W1-PASS`.
3. Mac completes `MAC-RC5-M2-READY`.
4. Windows completes `WINDOWS-RC5-W2-READY`.
5. Mac completes `MAC-RC5-M3-PASS` and prepares the remote conflict branch.
6. Windows completes keep-both; both agents reconcile all 21 rows and finalize
   their reports.

Do not clean up the fresh profile, remote prefix, or disposable paths until both
reports have been reviewed and Phase 1 has been signed off.
