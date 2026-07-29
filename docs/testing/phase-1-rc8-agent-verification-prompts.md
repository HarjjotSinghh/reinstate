# Phase 1 RC8 autonomous two-agent acceptance prompts

Use these prompts only after `v0.1.0-rc.8` is published and both public
installers install that exact release. Do not use RC5 or older homes, profiles,
passphrases, marker sessions, or reports as RC8 evidence.

The
[Phase 1 MacBook + Windows acceptance runbook](phase-1-mac-windows-acceptance.md)
from the signed RC8 tag remains authoritative for product behavior, mandatory
gates, and expected results. This document is an intentional automation overlay:
where the tagged runbook assigns a command, hidden input, marker check, storage
inspection, or evidence capture to a human, the device-owning agent performs it
instead under the controls below. A conflict about product behavior still stops
the run; the change of operator from human to agent does not.

The only human actions in the complete run are:

1. place the same dedicated-test `R2.txt` in the home directory of each device;
2. launch Claude Code on macOS with Prompt 1;
3. give Claude's sanitized M1 report to Codex with Prompt 2; and
4. pass later sanitized report revisions between the two existing agent
   sessions in the handoff order at the end of this document.

The human does not run commands, enter hidden values, inspect storage, choose
sessions, approve individual test steps, or interpret evidence. Full
bidirectional acceptance still requires report handoffs because neither agent
may remotely operate the other physical device.

Both agents preserve `PASS`, `PARTIAL`, `FAIL`, and `NOT TESTED` honestly. A
zero exit code is never sufficient by itself.

## Private `R2.txt` contract

Create `R2.txt` as UTF-8 text with exactly one non-empty entry for each key:

```text
REINSTATE_S3_ENDPOINT=<HTTPS service endpoint without the bucket suffix>
REINSTATE_S3_BUCKET=<dedicated acceptance bucket>
REINSTATE_S3_ACCESS_KEY_ID=<dedicated acceptance access-key ID>
REINSTATE_S3_SECRET_ACCESS_KEY=<dedicated acceptance secret key>
REINSTATE_ENCRYPTION_PASSPHRASE=<dedicated RC8 test passphrase>
```

Values are the literal text after the first `=`; do not add shell or PowerShell
quotes. An `=` inside a value is valid. Newlines inside values are not supported.
The bucket credentials must be limited to the dedicated acceptance bucket and
permit the runbook's put, get, list, and delete operations.

The exact locations are:

- macOS: `$HOME/R2.txt`
- Windows: `$HOME\R2.txt`

`REINSTATE_ENCRYPTION_PASSPHRASE` is a file-field name only. It must never be
exported as an environment variable. Reinstate accepts automated passphrase
input only through `REINSTATE_PASSPHRASE_FD`.

Each agent must validate the file without printing its values: exact regular
file, not a symlink/reparse point, no duplicate or unknown keys, endpoint and
bucket separate, and owner-only access. It may tighten the file permissions
without changing its contents. It must never `cat`, `Get-Content`, paste, echo,
log, screenshot, hash for disclosure, commit, upload, or include any part of
the file in a tool result or report.

## Prompt 1 — Claude Code on the MacBook

Copy everything inside this block into a new Claude Code session launched from
the fresh RC8 disposable project:

```text
Run the complete Device A side of Reinstate Phase 1 acceptance against the
published v0.1.0-rc.8. You are the macOS evidence owner and cross-device
coordinator. Work autonomously: do not ask me to run a command, type a secret,
inspect storage, choose a session, confirm a marker, or approve a routine test
step. Do not delegate this work or modify Reinstate product code.

Authority and automation overlay:
- Fetch origin and the signed tag v0.1.0-rc.8.
- Read docs/testing/phase-1-mac-windows-acceptance.md from that exact tag.
- Its product gates and expected results are authoritative.
- This prompt intentionally supersedes only its human-operation instructions:
  you perform those operations through the private R2.txt workflow.
- Stop and report any other conflict.

Repository scope:
- The only allowed repository change is the sanitized report:
  docs/testing/results/REPORT_DATE-macos-phase1-rc8.md
- Resolve REPORT_DATE yourself from the current UTC date in YYYY-MM-DD form;
  the human does not edit the prompt or filename.
- Use branch test/phase1-rc8-macos-report from the peeled RC8 tag commit.
- At each handoff, commit only that report, push the existing branch, and create
  or update one draft PR. Never merge, tag, release, deploy, or modify a product
  branch.
- Add R2.txt to .git/info/exclude before any report-repository mutation. Prove
  it is untracked and absent from the staged diff without displaying its
  contents.

Secret launcher:
- Use only the exact regular file $HOME/R2.txt described in the prompt preamble.
- Validate its schema and owner-only mode without returning values to the
  conversation or tool transcript.
- Build an ephemeral local launcher outside the repository and outside both
  agent session trees. It must parse R2.txt locally at runtime without printing
  values.
- For each Reinstate child process that needs storage, provide endpoint, bucket,
  access-key ID, and secret key only in that child's environment. Do not export
  them into the parent shell and do not place values in command arguments.
- For each Reinstate child process that needs encryption, open an anonymous
  pipe, write only the passphrase bytes, inherit only the read descriptor, and
  set REINSTATE_PASSPHRASE_FD to its descriptor number in that child. Never set
  an ordinary passphrase environment value.
- Do not use shell tracing, command echo, verbose HTTP/auth logging, crash
  dumps, process argument interpolation, clipboard transfer, or a temporary
  plaintext secret file. Capture stdout/stderr only after redacting any
  accidental occurrence of an R2.txt value in memory.
- The launcher must support a deliberately wrong generated passphrase without
  modifying R2.txt.
- Remove the launcher and zero/delete any transient secret buffers when the
  final report is complete. Do not delete or alter R2.txt.

Hard safety rules:
- Never disclose or commit an endpoint, bucket, access key, secret key,
  passphrase, keyring value, agent auth file, transcript, downloaded .age
  object, ciphertext bytes, username, absolute local path, or remote object
  name.
- You may inspect only exact acceptance-marker occurrence counts and exact
  challenge-response output needed to prove same-vendor resume. Never print or
  summarize surrounding transcript prose.
- Never use --all. Operate only on the two fresh RC8 marker session IDs.
- Never delete or mutate an older home/profile, real agent session, unrelated
  project, or unrelated remote object.
- Never manually move a restored vendor file to manufacture discovery.
- Keep normal sandboxing and safety controls enabled. Never use a
  permission-bypass flag. If the harness cannot perform an authorized,
  narrowly scoped step without a human approval, record it as BLOCKED/NOT
  TESTED; do not ask the human to perform it and do not bypass the control.
- Report only non-secret IDs, counts, booleans, versions, exit codes, redacted
  paths, and sanitized error text.

Isolation:
- REINSTATE_HOME=$HOME/.reinstate-phase1-acceptance-rc8
- project=$HOME/Projects/reinstate-phase1-acceptance-rc8
- canonical project ID=local/reinstate-phase1-acceptance-rc8
- Stop if either isolated path already exists or any older acceptance state
  would be reused.
- Export the exact isolated REINSTATE_HOME before invoking Prompt version 6.
  Preserve it for every command; stop if any nested workflow unsets, redirects,
  or silently falls back from it.

Milestone M0 — release and environment:
1. Verify v0.1.0-rc.8 is an annotated signed tag reachable from origin/main.
   Record its commit without changing trust configuration to force a pass.
2. Verify the live install.sh returns 200, pins only rc.8, verifies both
   checksum layers, installs 0.1.0-rc.8 without elevation, and is idempotent
   with one PATH entry. Handle the documented replacement prompt yourself.
3. Record macOS/architecture, native shell, Claude Code, Codex CLI, Git, and
   rein version --json.
4. Create the isolated project/home and prove pre-init rein setup check exits 3
   with config missing; device and adapters must not falsely pass.

Milestone M1 — source sessions, init safety, push, and ciphertext:
5. Use the installed vendors' documented non-interactive invocation/resume
   modes to create and cleanly close one harmless fresh session per agent:
   - REINSTATE-PHASE1-RC8-MAC-CLAUDE-A1
   - REINSTATE-PHASE1-RC8-MAC-CODEX-A1
   Identify both IDs from before/after metadata and exact marker counts only.
   If Codex does not persist a new rollout, fail instead of reusing an old one.
6. Execute the outcomes of the exact tagged Claude Code setup prompt, Prompt
   version 6, as an autonomous end user. The R2.txt overlay replaces its
   questions and human-run/private-input steps; all other safety and validation
   requirements remain.
7. Run first-device init non-interactively with the canonical mapping. Use
   endpoint/bucket/credentials through the child-only provider and --yes.
   Record the generated non-secret fresh RC8 profile_id.
8. Require rein setup check and rein doctor --self-test to exit 0.
9. Prove physical F1 default refusal: rerun the same init without --force.
   Calculate config.toml/state.json equality and backup counts yourself,
   recording booleans only. Require exit 7, unchanged files, and no new backup.
10. Dry-run and push only the selected Claude and Codex IDs through the
    launcher. Each dry-run must say would push, not pushed. Correct-passphrase
    status must show exactly those two sessions.
11. Inspect only the fresh profile prefix through a scoped S3-compatible API
    client using R2.txt at runtime. Do not use a human UI. Confirm no auth,
    token, credential, .env, or plaintext-shaped object exists. Download one
    .age snapshot to an owner-only temporary file, test both exact A1 marker
    byte strings for absence without printing bytes, verify it is ciphertext/
    non-text, and delete only the local download.
12. Write the complete sanitized report with row-level evidence and this block:

MAC-RC8-M1
release=v0.1.0-rc.8
tag_commit=<non-secret sha>
profile_id=<non-secret fresh rc8 uuid>
canonical_project_id=local/reinstate-phase1-acceptance-rc8
claude_session_id=<non-secret fresh uuid>
codex_session_id=<non-secret fresh uuid>
remote_session_count=2
f1_default_refusal=PASS|FAIL
ciphertext_marker_absence=PASS|FAIL
mac_report_path=docs/testing/results/REPORT_DATE-macos-phase1-rc8.md
END-MAC-RC8-M1

Commit/push only the report, open or update the draft PR, return the report
content/path/commit/PR, and pause. The human will transfer only this sanitized
report to Windows.

Milestone M2 — after a WINDOWS-RC8-W1-PASS report:
13. Validate the entire supplied Windows report and handoff without accepting
    secrets or transcript prose.
14. Autonomously resume the exact Claude session on Mac, add only
    REINSTATE-PHASE1-RC8-MAC-CLAUDE-A2, exit it, then dry-run and push only that
    ID. Update/commit/push the Mac report with MAC-RC8-M2-READY containing the
    session ID and new non-secret snapshot/revision IDs. Pause for the report
    transfer to Windows.

Milestone M3 — after a WINDOWS-RC8-W2-READY report:
15. Close both Mac agents. Dry-run and pull each exact ID; dry-runs must say
    would pull, never pulled.
16. Prove each existing target received a timestamped backup. Resume each exact
    session non-interactively and verify only these exact markers:
    - REINSTATE-PHASE1-RC8-WINDOWS-CLAUDE-B1
    - REINSTATE-PHASE1-RC8-WINDOWS-CODEX-B1
17. Without modifying either restored session, push each ID and require
    pushed 0 snapshot(s), skipped 1 unchanged; prove revisions did not change.
18. Update/commit/push the report with MAC-RC8-M3-PASS containing sanitized
    backup, resume, no-op, and revision evidence. Pause for transfer to Windows.

Milestone M4 — divergence and final verdict:
19. After Windows reports its unpushed local divergence ready, resume the exact
    Mac Claude ID, add only REINSTATE-PHASE1-RC8-CONFLICT-MAC, exit, and push
    only that ID. Update/commit/push the report with the remote revision and
    pause for Windows keep-both.
20. Consume the final Windows report. Reconcile every mandatory section 19 row
    across both reports. Independently query the RC8 publication PR/check run
    and verify every automated-integrity gate in tagged runbook section 18 is
    green. Unexecuted evidence is NOT TESTED, never PASS.
21. Finalize, commit, and push only the Mac report. Keep the PR draft/unmerged.

Every milestone response must include:
- current verdict and x PASS / y PARTIAL / z FAIL / n NOT TESTED;
- whether all 23 mandatory rows passed;
- report path, report commit, branch, and draft PR;
- release-blocking/non-blocking findings;
- exact failed command, exit code, and sanitized output for every failure;
- confirmation that no product code, R2.txt data, or other secret was committed.

Phase 1 is PASS only if all 23 mandatory rows have real-device evidence.
```

## Prompt 2 — Codex on the Windows PC

Start this prompt only after giving Codex the complete sanitized Mac M1 report:

```text
Run the complete Device B side of Reinstate Phase 1 acceptance against the
published v0.1.0-rc.8. You are the native-Windows evidence owner. Work
autonomously: do not ask me to run a command, type a secret, inspect storage,
choose a session, confirm a marker, or approve a routine test step. Do not
delegate this work or modify Reinstate product code.

Authority and automation overlay:
- Fetch origin and the signed tag v0.1.0-rc.8.
- Read docs/testing/phase-1-mac-windows-acceptance.md from that exact tag.
- Its product gates and expected results are authoritative.
- This prompt intentionally supersedes only its human-operation instructions:
  you perform those operations through the private R2.txt workflow.
- Stop and report any other conflict.

Repository scope:
- The only allowed repository change is the sanitized report:
  docs/testing/results/REPORT_DATE-windows-phase1-rc8.md
- Resolve REPORT_DATE yourself from the current UTC date in YYYY-MM-DD form;
  the human does not edit the prompt or filename.
- Use branch test/phase1-rc8-windows-report from the peeled RC8 tag commit.
- At each handoff, commit only that report, push the existing branch, and create
  or update one draft PR. Never merge, tag, release, deploy, or modify a product
  branch.
- Add R2.txt to .git/info/exclude before any report-repository mutation. Prove
  it is untracked and absent from the staged diff without displaying its
  contents.

Required Mac report:
- It contains MAC-RC8-M1 for v0.1.0-rc.8, the fresh profile/session IDs,
  canonical project ID local/reinstate-phase1-acceptance-rc8, exactly two
  remote sessions, f1_default_refusal=PASS, and
  ciphertext_marker_absence=PASS.
- Stop if it references older state, lacks physical ciphertext evidence, or
  contains a secret. Do not repeat any suspected secret.

Secret launcher:
- Use only the exact regular file $HOME\R2.txt described in the prompt preamble.
- Validate its schema, reparse-point status, and owner-only ACL without
  returning values to the conversation or tool transcript.
- Build an ephemeral local launcher outside the repository and both agent
  session trees. Parse R2.txt locally at runtime without printing values.
- Supply endpoint, bucket, access-key ID, and secret key only in each Reinstate
  child's environment. Do not persist them in the parent PowerShell or place
  them in command arguments.
- For encryption input, create an inheritable anonymous pipe/handle, write only
  the passphrase bytes, pass only its read handle to the Reinstate child, and
  set REINSTATE_PASSPHRASE_FD to that numeric inherited handle. Never set an
  ordinary passphrase environment value.
- Do not use command echo, transcription, verbose auth logging, process
  argument interpolation, clipboard transfer, or a temporary plaintext secret
  file. Redact captured output in memory before reporting.
- Support a deliberately wrong generated passphrase without changing R2.txt.
- Remove the launcher and transient secret buffers after the final report. Do
  not delete or alter R2.txt.

Hard safety rules:
- Use native 64-bit Windows PowerShell 5.1 or newer, never WSL.
- Never disclose or commit an endpoint, bucket, access key, secret key,
  passphrase, keyring value, auth file, transcript, .age object, ciphertext
  bytes, username, absolute local path, or remote object name.
- Inspect only exact acceptance-marker counts and exact challenge-response
  output. Never print or summarize surrounding transcript prose.
- Never use --all, reuse/delete older state, mutate unrelated sessions, or move
  a restored Claude file to manufacture discovery.
- Keep normal sandboxing and safety controls enabled. Never use a
  permission-bypass flag. If a step requires human approval, record it
  BLOCKED/NOT TESTED instead of asking the human or bypassing the control.
- Report only non-secret IDs, counts, booleans, versions, exits, redacted paths,
  and sanitized errors.

Isolation:
- REINSTATE_HOME=$HOME\.reinstate-phase1-acceptance-rc8
- project=$HOME\Projects\reinstate-phase1-acceptance-rc8
- canonical project ID=local/reinstate-phase1-acceptance-rc8
- Stop if either isolated path already exists or older state would be reused.
- Set and preserve the exact isolated REINSTATE_HOME before invoking Prompt
  version 6 or any Reinstate command.

Milestone W0 — release and environment:
1. Verify v0.1.0-rc.8 is an annotated signed tag reachable from origin/main.
2. Verify live install.ps1 returns 200, pins only rc.8, verifies both checksum
   layers, installs 0.1.0-rc.8 without elevation, is idempotent, and produces
   exactly one normalized user PATH entry.
3. Record Windows/build/architecture, native PowerShell, Claude Code, Codex CLI,
   Git, and rein version --json.
4. Create the fresh isolated project/home and prove pre-init setup check exits 3
   with config missing without false adapter passes.

Milestone W1 — F3/F1/F2 and Mac-to-Windows restore:
5. Execute the outcomes of the exact tagged Codex setup prompt, Prompt version
   6, as an autonomous end user. The R2.txt overlay replaces its questions and
   human-run/private-input steps; all other requirements remain.
6. Physical F3: derive a bad endpoint in memory by appending the bucket suffix
   while also retaining the separate bucket. Run additional-device init
   non-interactively with --yes, the Mac profile ID, and canonical mapping.
   Require exit 4, actionable remote-profile-not-found/storage failure, and no
   config.toml. Never print either coordinate.
7. Repeat with the correct endpoint-only value and same profile/mapping.
   Require success, setup check exit 0, and doctor --self-test exit 0.
8. Physical F1: rerun correct init without --force; record config/state
   equality and backup-count booleans. Require exit 7, no mutation, no backup.
9. Wrong-passphrase test: run status through the launcher with a generated
   wrong phrase. Require exit 4, decryption refusal, zero restore targets, zero
   backups, and no mutation. Correct-passphrase status must show exactly two
   selected sessions.
10. Physical F2 in a disposable copied home: replace exactly the profile prefix
    with a missing-manifest probe suffix, run status with the correct phrase,
    and require exit 4 plus remote profile manifest not found, never exit 0
    with zero sessions. Prove the real home and remote objects unchanged.
11. With both agents closed, dry-run then pull only the exact Codex and Claude
    IDs. Dry-runs must say would pull, never pulled, and create no target/backup.
12. Prove exact-ID Codex discovery/resume with codex resume CODEX_SESSION_ID.
13. From the mapped Windows project, prove normal Claude discovery contains the
    exact ID, then use claude --resume CLAUDE_SESSION_ID. The restored target
    must use the Windows project key, never the Mac source slug.
14. Verify only the two exact A1 markers/challenge responses, without printing
    transcript prose.
15. Write the complete sanitized report with row-level evidence and:

WINDOWS-RC8-W1-PASS
release=v0.1.0-rc.8
profile_id=<non-secret fresh rc8 uuid>
claude_session_id=<non-secret uuid>
codex_session_id=<non-secret uuid>
f3_bad_coordinates_refused=PASS
f1_default_refusal=PASS
f2_missing_manifest_refused=PASS
wrong_passphrase_refused=PASS
remote_session_count=2
claude_discovery_and_resume=PASS
codex_resume=PASS
windows_report_path=docs/testing/results/REPORT_DATE-windows-phase1-rc8.md
END-WINDOWS-RC8-W1

On failure, use WINDOWS-RC8-W1-FAIL with exact command, exit, and sanitized
output, and do not execute dependent gates. Commit/push only the report, open or
update the draft PR, return the report content/path/commit/PR, and pause for the
human to transfer that report to Mac.

Milestone W2 — after a MAC-RC8-M2-READY report:
16. Validate the supplied Mac report, then leave Claude open on the exact
    restored session and run pull for that ID separately. Require active-agent
    safety exit 7 and no mutation.
17. Close Claude autonomously, dry-run and pull the same ID, prove a timestamped
    backup, then resume and verify only
    REINSTATE-PHASE1-RC8-MAC-CLAUDE-A2.
18. Resume the exact Windows Claude and Codex sessions, add only:
    - REINSTATE-PHASE1-RC8-WINDOWS-CLAUDE-B1
    - REINSTATE-PHASE1-RC8-WINDOWS-CODEX-B1
    Exit both, dry-run, and push only those IDs.
19. Update/commit/push the report with WINDOWS-RC8-W2-READY containing
    sanitized active-refusal, backup, A2-resume, and remote revision evidence.
    Pause for transfer to Mac.

Milestone W3 — conflict and final verdict:
20. After MAC-RC8-M3-PASS, resume the exact Windows Claude session, add only
    REINSTATE-PHASE1-RC8-CONFLICT-WINDOWS, exit, and do not push. Update the
    report with WINDOWS-RC8-CONFLICT-LOCAL-READY and pause for transfer to Mac.
21. After the Mac report confirms its conflict-marker push, close Claude and
    pull the exact ID. Require exit 6, one recorded conflict, and no overwrite.
22. Inspect sanitized conflict metadata, resolve it with --keep-both, prove the
    local branch and distinct vendor-safe remote fork both resume with their
    expected exact marker, and prove the active conflict list is empty.
23. Reconcile all mandatory section 19 rows with the latest Mac report.
    Independently query the RC8 publication PR/check run and verify every
    automated-integrity gate in tagged runbook section 18 is green. Unexecuted
    evidence is NOT TESTED, never PASS.
24. Finalize, commit, and push only the Windows report. Keep the PR
    draft/unmerged and return it for Mac's final reconciliation.

Every milestone response must include:
- current verdict and x PASS / y PARTIAL / z FAIL / n NOT TESTED;
- whether all 23 mandatory rows passed;
- report path, commit, branch, and draft PR;
- release-blocking/non-blocking findings;
- exact failed command, exit code, and sanitized output for every failure;
- confirmation that no product code, R2.txt data, or other secret was committed.

Phase 1 is PASS only when all 23 rows have real evidence across both devices.
```

## Handoff order

Every handoff is the complete latest sanitized report, not an ad-hoc chat note.
The receiving agent validates the embedded milestone block before continuing.

1. Mac produces `MAC-RC8-M1`; human passes the Mac report to Windows.
2. Windows produces `WINDOWS-RC8-W1-PASS`; human passes the Windows report to Mac.
3. Mac produces `MAC-RC8-M2-READY`; human passes the Mac report to Windows.
4. Windows produces `WINDOWS-RC8-W2-READY`; human passes the Windows report to Mac.
5. Mac produces `MAC-RC8-M3-PASS`; human passes the Mac report to Windows.
6. Windows produces `WINDOWS-RC8-CONFLICT-LOCAL-READY`; human passes the Windows
   report to Mac.
7. Mac records the conflict-marker push; human passes the Mac report to Windows.
8. Windows completes keep-both and its final report; human passes that report
   to Mac for final reconciliation.

Do not clean up the fresh profile, remote prefix, R2.txt, or disposable paths
until both reports have been reviewed and Phase 1 has been signed off.
