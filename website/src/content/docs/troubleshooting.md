---
title: "Troubleshoot Reinstate session sync"
description: "Fix installation, path remapping, passphrase, remote manifest, conflict, large Codex history, credential, and active-agent errors without exposing secrets."
order: 8
author: "Harjot Singh Rana"
status: current
updatedAt: 2026-07-27
tags: ["troubleshooting", "session-sync", "path-remapping", "passphrase", "codex"]
targetQuery: "fix Reinstate session sync"
searchIntent: "troubleshooting"
draft: false
noindex: false
---

Use the smallest possible command while diagnosing a sync problem: one agent
and one explicit session ID. Do not paste passphrases, storage credentials,
session text, raw configuration, or unredacted absolute paths into an issue.
Reinstate `v0.1.0-rc.6` is a pre-1.0 release candidate; Claude Code and Codex
resume only their own native sessions.

## Why is the `rein` binary not found after installation?

### Symptom

The shell reports that `rein` or `reinstate` is not recognized, not found, or
not a command. Calling the binary by its full installation path may still work.

### Likely cause

The installation directory is absent from the current shell's `PATH`, the
shell has not reloaded its environment, or the binary was not installed or
built successfully. `rein` and `reinstate` are names for the same CLI.

### Affected agent(s)

Claude Code and Codex. This is a Reinstate installation problem and occurs
before an agent adapter can run.

### Affected OS

macOS, native Windows, and WSL2. PATH syntax and executable discovery differ
between POSIX shells and PowerShell.

### Diagnostic commands

Run the block for your shell. These commands do not inspect session content.

```bash
command -v rein
command -v reinstate
echo "$PATH"
./bin/rein version --json
```

```powershell
Get-Command rein -ErrorAction SilentlyContinue
Get-Command reinstate -ErrorAction SilentlyContinue
$env:Path -split [IO.Path]::PathSeparator
```

The relative `./bin/rein` command in the POSIX block is only for a source
checkout where `make build` has already created the local binary. If that path
is missing, the source build did not complete.

### Corrective action

Reopen the terminal after installation. If the binary exists, add its
installation directory—not the executable itself—to the user `PATH`, then open
a new shell. If it does not exist, repeat the documented installation or run
`make build` from the repository root and resolve any build error before
changing `PATH`.

Do not download an unverified binary from a third-party mirror. Use the
[getting-started installation steps](/docs/getting-started) and verify the
release checksum.

### Expected recovery evidence

Both `rein version --json` and `reinstate version --json` exit successfully and
report the same Reinstate version. A new shell resolves `rein` without an
absolute path. Continue with `rein setup check`; do not treat command discovery
alone as proof that storage or agent compatibility is ready.

### When to file an issue

File an issue when the verified official installer reports success but a new
shell cannot resolve either binary, or when the two binary names report
different versions. Include the installer version, OS and architecture, shell
name, and a redacted installation directory. Do not include the complete
`PATH` if it contains private directory names.

## Why does `claude --resume` not see a pulled session?

### Symptom

`rein pull --agent claude --session SESSION_ID` reports a successful restore,
but `claude --resume SESSION_ID` cannot find that exact session or the session
does not appear for the destination project.

### Likely cause

The destination project path was not mapped to the exact Claude project
directory key expected on this device. A snapshot made before RC6 may also lack
the safely mappable Claude project identity required by the current adapter.

### Affected agent(s)

Claude Code only. Codex uses a different native, date-partitioned rollout
layout and must be diagnosed with its own adapter and resume command.

### Affected OS

macOS, native Windows, and WSL2, especially transfers where source and
destination use different absolute project roots. Treat native Windows and
WSL2 as separate Reinstate devices.

### Diagnostic commands

Replace `SESSION_ID` with the exact non-secret session identifier. The dry-run
must name a destination on this device, not the source device's directory key.

```bash
rein version --json
rein setup check --json
rein list --agent claude --json
rein pull --agent claude --session SESSION_ID --dry-run --json
```

Review the configured mapping locally. Use the same canonical project ID on
both devices, but give it each device's real absolute `local_root`. Do not paste
the raw roots into a public issue.

### Corrective action

Require Reinstate `0.1.0-rc.6` or newer on both devices. Correct the destination
mapping for the existing canonical project ID, close Claude Code, and repeat
the scoped dry-run. If RC6 rejects a legacy snapshot whose Claude project
identity cannot be mapped, install RC6 on the source device and push that one
session again to an intentionally fresh RC6 profile. Do not manually move the
session file into a guessed Claude directory.

After the dry-run shows the correct destination, run the same pull without
`--dry-run`, then resume through Claude Code:

```bash
rein pull --agent claude --session SESSION_ID
rein list --agent claude --json
claude --resume SESSION_ID
```

### Expected recovery evidence

The dry-run reports `dry_run=true`, one planned snapshot, and a destination
under this device's Claude project directory. The mutating pull reports one
pulled snapshot. `rein list --agent claude --json` discovers the same ID at the
planned destination, and `claude --resume SESSION_ID` opens that native Claude
Code session.

### When to file an issue

File an issue when RC6 or newer plans and writes the correct destination,
`rein list` discovers the exact restored ID, and Claude Code in the tested
compatibility range still cannot resume it. Include both OS versions, Reinstate
and Claude Code versions, the transfer direction, the redacted dry-run plan,
and redacted project-path shapes.

## Why does passphrase verification fail on a second device?

### Symptom

`rein status`, `push`, or `pull` reaches storage but rejects the passphrase or
cannot decrypt the existing remote `manifest.age`.

### Likely cause

The destination did not receive the exact passphrase used to encrypt the
profile's remote manifest. Typing before Reinstate displays its hidden prompt
can send the secret to the shell instead of the Reinstate process. There is no
passphrase recovery or alternate key that can decrypt existing ciphertext.

### Affected agent(s)

Claude Code and Codex. The encrypted manifest is profile-wide, so verification
fails before a selected agent session can be read.

### Affected OS

macOS, native Windows, and WSL2. Shell and terminal behavior can differ, but
the passphrase bytes must be identical on every device.

### Diagnostic commands

First confirm local setup without entering a passphrase. Then invoke `status`
and wait for Reinstate's visible hidden prompt before typing.

```bash
rein version --json
rein setup check --json
rein status --json
```

Do not use `echo`, a command-line flag, ordinary environment variables,
clipboard logs, or chat messages to test the passphrase.

### Corrective action

Rerun the command and enter the exact original passphrase only after the hidden
prompt appears. Check keyboard layout, Caps Lock, and password-manager entry
selection. If the process already exited, start it again rather than typing
into the shell.

If the original passphrase is irretrievably lost, preserve the ciphertext in
case the passphrase is recovered. Creating a separate fresh profile or storage
prefix deliberately abandons access to the old encrypted profile; it does not
recover or re-key it.

### Expected recovery evidence

`rein status --json` exits successfully and returns the remote revision and
expected session keys. A scoped push or pull dry-run authenticates the manifest
and reports a plan without modifying local or remote state.

### When to file an issue

File an issue only when the same saved passphrase successfully decrypts the
same profile on one device but fails on another device with matching profile,
bucket, prefix, and current Reinstate version. Include redacted error output,
OS and terminal names, and version output. Never include the passphrase, its
length, hints, hashes, or password-manager screenshots.

## Why does Reinstate report a remote profile manifest is missing?

### Symptom

Reinstate can contact the configured storage service but reports that the
remote profile manifest is missing instead of showing the first device's
sessions.

### Likely cause

The additional device is looking at a different `profile_id`, bucket, or
`storage.prefix`, or the bucket name was incorrectly appended to the service
endpoint. The existing profile's encrypted `manifest.age` therefore is not at
the coordinates Reinstate was given.

### Affected agent(s)

Claude Code and Codex. The remote manifest indexes both supported agent types
for one Reinstate profile.

### Affected OS

macOS, native Windows, and WSL2. This is a profile/storage-coordinate issue,
not an agent-specific filesystem layout issue.

### Diagnostic commands

Run redacted diagnostics and a status check. `status` will request the
passphrase if it finds the encrypted manifest.

```bash
rein version --json
rein setup check --json
rein doctor --json
rein status --json
```

Compare the non-secret `profile_id`, bucket, prefix, region, and service
endpoint locally with the values used on the first device. Do not post
credentials, signed URLs, or a complete unredacted configuration.

### Corrective action

Correct the inputs so the additional device uses the exact existing
`profile_id`, bucket, and prefix. Keep the bucket name separate from the
service endpoint. Then rerun `init --profile-id` in a disposable or
intentionally reinitialized Reinstate home.

Do not create an empty `manifest.age` and do not silently substitute a new
profile ID. If you intentionally reuse an initialized home, review it first;
`rein init --force` backs up the existing config and state together before
replacing them.

### Expected recovery evidence

Initialization verifies the existing encrypted remote manifest before saving
the additional device. With the correct passphrase, `rein status --json`
reports the same remote revision and selected session keys visible from the
first device. No empty replacement manifest is created.

### When to file an issue

File an issue when an object listing confirms `manifest.age` exists at the
exact configured prefix, the profile coordinates match, storage credentials
can read that object, and RC6 still reports it missing. Include redacted
diagnostics, provider type, endpoint host, region, and object-key shape—not
credentials, signed requests, bucket policies, or session payloads.

## Why does Reinstate create a session conflict?

### Symptom

A push or pull exits with sync-conflict code `6`, and `rein conflicts list`
shows local and remote revisions for the same Claude Code or Codex session.

### Likely cause

Both devices changed the same session after their last common revision, or the
local session changed outside the revision Reinstate recorded. Reinstate
records this divergence instead of silently overwriting either side.

### Affected agent(s)

Claude Code and Codex, independently. Conflict records are scoped to an agent
and session ID; they do not translate or merge conversations across vendors.

### Affected OS

macOS, native Windows, and WSL2. Conflicts can occur on one device or across
any supported device pair.

### Diagnostic commands

The conflict commands show metadata, not a semantic transcript merge. Replace
`CONFLICT_ID` with the ID printed by the list command.

```bash
rein conflicts list --json
rein conflicts show CONFLICT_ID
rein status --json
rein diff --json
```

### Corrective action

Inspect the metadata and choose exactly one explicit strategy:

- `--keep-local` retains the local branch and advances the remote head with it.
- `--keep-remote` restores the remote branch over the selected local session;
  close that agent first.
- `--keep-both` preserves the local branch and restores the remote revision as
  a distinct native session.

When uncertain, prefer `--keep-both`, inspect both native sessions, and decide
later:

```bash
rein conflicts resolve CONFLICT_ID --keep-both
```

Do not edit the conflict record or session files by hand while resolving it.

### Expected recovery evidence

The resolve command prints `resolved CONFLICT_ID via keep-both` (or the chosen
strategy). The ID disappears from `rein conflicts list`. For `--keep-both`,
`rein list --agent claude` or `rein list --agent codex` shows the preserved
local session and a distinct restored session; each remains resumable only by
its own vendor.

### When to file an issue

File an issue if a conflict appears without local/remote divergence, if a
documented resolution deletes the branch it should preserve, if `--keep-both`
reuses the same session identity, or if the conflict remains after a successful
resolution. Include redacted `list`/`show` metadata, strategy, versions, OS,
agent, and reproducible steps. Do not attach session files.

## Why is a large Codex session slow to sync?

### Symptom

A selected Codex session takes substantially longer to plan, encrypt, upload,
download, or restore than a smaller rollout, especially when using `--all`.

### Likely cause

Phase 1 transfers full immutable session snapshots. Append-aware delta transfer
and retention controls remain roadmap work, so a large Codex rollout requires
processing and transferring the complete selected artifact.

### Affected agent(s)

Codex. Large Claude Code sessions can also take longer, but this entry addresses
Codex's rollout artifacts and date-partitioned native layout.

### Affected OS

macOS, native Windows, and WSL2. Runtime varies with session size, CPU, disk,
network, and the configured S3-compatible storage service.

### Diagnostic commands

Use an explicit Codex session ID. Compare the scoped plan with `--all` without
running two mutating transfers.

```bash
rein version --json
rein list --agent codex --json
rein push --agent codex --session SESSION_ID --dry-run --json
rein status --json
```

Do not print or attach the rollout to measure it. Record only non-secret
metadata such as approximate byte size, elapsed time, agent version, and
whether the delay occurred before or during network transfer.

### Corrective action

Push or pull one explicit session instead of `--all`. Let the active Codex
process finish writing, exit Codex before a mutating restore of an existing
session, and avoid repeatedly transferring an unchanged snapshot. Keep
independent backups. Do not assume roadmap delta or retention settings already
exist in the current CLI.

### Expected recovery evidence

The scoped dry-run reports exactly one planned snapshot. The mutating command
eventually reports one pushed or pulled snapshot, and `rein status --json`
shows its remote session key and snapshot revision. A repeat push without
changes reports the snapshot as unchanged/skipped rather than uploading a new
revision.

### When to file an issue

File an issue if a scoped transfer never completes, crashes, exhausts expected
resources, corrupts the restored rollout, or uploads an unchanged session
again. Include approximate size and timing, transfer direction, storage
provider type, OS, Codex and Reinstate versions, and redacted output. A large
snapshot being slower than a small one is not by itself evidence of a defect.

## Can Reinstate upload credentials from a transcript?

### Symptom

A secret, token, or credential value was pasted into or printed inside a coding
agent conversation, and that session may already have been pushed.

### Likely cause

Adapters hard-exclude known credential files, authentication artifacts, tokens,
caches, and logs. They cannot determine the semantic meaning of every line in
a selected transcript. A secret embedded in session text is therefore part of
the encrypted session payload.

### Affected agent(s)

Claude Code and Codex. This boundary applies to any supported transcript that
contains sensitive text.

### Affected OS

macOS, native Windows, and WSL2. Client-side encryption limits what the storage
provider can read, but it does not make a compromised credential valid to keep
using.

### Diagnostic commands

Use metadata-only commands. Do not print, search, paste, or attach the
transcript while investigating.

```bash
rein list --agent claude --json
rein list --agent codex --json
rein status --json
rein push --agent AGENT --session SESSION_ID --dry-run --json
```

Replace `AGENT` with `claude` or `codex`. The dry-run confirms selection and
destination metadata; it does not certify that transcript prose contains no
secret.

### Corrective action

Revoke or rotate the exposed credential immediately. Treat every destination
that received the session as sensitive. Remove or quarantine affected remote
snapshot objects according to your storage provider's retention/versioning
policy, then create a clean agent session that does not contain the value.
Push only the explicitly reviewed clean session.

Reinstate has no current transcript-redaction command and cannot prove semantic
secret removal. Do not weaken the adapter exclusions or attempt to synchronize
vendor authentication files.

### Expected recovery evidence

The exposed credential is invalidated at its issuer. Storage-provider evidence
shows the affected object versions were removed, quarantined, or made
inaccessible according to your incident policy. A subsequent scoped dry-run
selects only the intended clean session. Reinstate output alone is not evidence
that a transcript is secret-free.

### When to file an issue

File a private security report—not a public issue—if Reinstate includes a
documented hard-excluded credential artifact or uploads plaintext to remote
storage. A secret included as ordinary transcript text is a documented
boundary, but report any suspected exclusion bypass. Provide synthetic
reproduction data and redacted metadata only; never send the real credential
or session.

## Why does a pull fail while the coding agent is running?

### Symptom

A mutating pull or `conflicts resolve --keep-remote` exits with safety-refusal
code `7` because Claude Code or Codex may still be writing the destination
session.

### Likely cause

An active process for the selected agent owns or may mutate an existing local
session. Reinstate refuses the overwrite to prevent a race, partial write, or
loss of newer local history. A dry-run and a restore of a genuinely new session
do not overwrite an active existing target.

### Affected agent(s)

Claude Code and Codex. The refusal is evaluated for the agent whose existing
session would be replaced.

### Affected OS

macOS, native Windows, and WSL2. Process-discovery details vary by platform.

### Diagnostic commands

The Reinstate dry-run remains safe. The process commands are observational and
may also match helper processes; review their output rather than terminating
them blindly.

```bash
rein pull --agent AGENT --session SESSION_ID --dry-run --json
pgrep -fl 'claude|codex'
```

```powershell
rein pull --agent AGENT --session SESSION_ID --dry-run --json
Get-Process claude,codex -ErrorAction SilentlyContinue
```

Replace `AGENT` with `claude` or `codex`.

### Corrective action

Save any work, exit every process for the selected agent normally, and verify
that it has stopped. Repeat the scoped dry-run, review the planned destination
and backup root, then run the same pull without `--dry-run`. Do not force-kill
an agent while it may be writing a session.

For a recorded conflict, `--keep-both` remains available because it restores a
distinct session identity. Use it only after inspecting the conflict metadata;
it is not a substitute for closing the agent before overwriting the existing
target.

### Expected recovery evidence

No selected-agent process remains. The scoped mutating pull exits successfully,
reports one pulled snapshot, creates the planned timestamped backup when an
existing target is replaced, and `rein list --agent AGENT --json` discovers the
restored session. The same vendor's native resume command opens it.

### When to file an issue

File an issue when the safety refusal persists after a normal agent exit and
process inspection shows no matching process, or when a mutating pull proceeds
while the selected agent is demonstrably active. Include OS, agent and
Reinstate versions, redacted process names, exit code, dry-run plan, and steps.
Do not attach process command lines if they contain private paths or arguments.

## Still stuck?

- Review the [FAQ](/docs/faq) and [security model](/docs/security-model).
- Follow the repository's [support policy](https://github.com/HarjjotSinghh/reinstate/blob/main/SUPPORT.md).
- Use [GitHub Issues](https://github.com/HarjjotSinghh/reinstate/issues) for
  reproducible non-security defects.
- Use the private [security policy](https://github.com/HarjjotSinghh/reinstate/security/policy)
  for suspected credential-exclusion, plaintext-upload, or other security
  failures.
