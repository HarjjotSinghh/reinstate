# Claude Code — Reinstate end-user setup prompt

**Prompt version:** 3
**Pinned Reinstate release:** `v0.1.0-rc.2`

Copy everything below the line into Claude Code.

---

Set up Reinstate end to end on this device using the official public bootstrap,
which must pin the exact release `v0.1.0-rc.2`. Never substitute `latest`,
`main`, or another version.

Safety rules:

- Keep normal approval and sandbox controls enabled.
- Do not clone the repository or build from source.
- Never read, copy, print, log, or sync agent auth files, API keys, OAuth
  tokens, `.env` files, keychains, or credential stores.
- Never ask me to paste a storage credential or encryption passphrase into
  chat. I will enter secrets privately into Reinstate's hidden terminal
  prompts.
- Do not publish, commit, push, purchase, install unrelated software, or modify
  unrelated repositories.

Complete this workflow:

1. Detect the OS, architecture, shell, native Windows versus WSL, Claude Code
   version, and any existing `rein` or `reinstate` binary. Report the install
   path you expect to use.
2. Select exactly one public bootstrap:
   - macOS/Linux/WSL:
     `https://reinstate.dev/install.sh`
   - native Windows PowerShell:
     `https://reinstate.dev/install.ps1`
3. Download the selected bootstrap to a temporary file before executing it.
   Inspect and report only its non-secret security contract:
   - it pins `v0.1.0-rc.2`;
   - it downloads the canonical installer from that exact Git tag;
   - it verifies the canonical installer checksum before execution;
   - it does not resolve `latest`; and
   - the canonical installer downloads binaries only from
     `https://github.com/HarjjotSinghh/reinstate/releases/download/v0.1.0-rc.2/`.
   Stop if any condition is false.
4. With normal approval, execute the inspected bootstrap. Do not bypass its
   checksum or replacement safeguards. It must install without elevation to
   `~/.local/bin` on macOS/WSL or
   `%LOCALAPPDATA%\Programs\Reinstate\bin` on native Windows and explain any
   PATH change.
5. Run `rein version --json`. Require version `0.1.0-rc.2`. Run
   `rein setup check`; before initialization, only a missing-config failure is
   expected. Any platform, keyring, or Claude compatibility failure must be
   reported and must not be called success.
6. Ask whether this is the first or an additional device. Collect only:
   - the non-secret S3/R2 endpoint and bucket;
   - the canonical project ID and this device's absolute project path; and
   - on an additional device, the non-secret `profile_id` printed by Device A.
   Ask whether I want one explicitly selected Claude session or all discovered
   sessions. Default to one session. Never choose `--all` for me.
7. Construct one command:
   - first device: `rein init --project ID=ABSOLUTE_PATH`
   - additional device:
     `rein init --profile-id UUID --project ID=ABSOLUTE_PATH`

   Pause and have me run it in a private interactive terminal. I will enter the
   storage credentials into Reinstate's hidden prompts. Do not read or echo
   them. Save Device A's printed profile UUID for the additional device.
8. After initialization, run `rein setup check` and
   `rein doctor --self-test`. Both must pass.
9. On the first device, use
   `rein push --agent claude --session SESSION_ID --dry-run` for a selected
   session, or `rein push --all --dry-run` only if I explicitly selected all.
   Summarize only metadata and destinations, then request approval before the
   matching mutating command. I will enter the encryption passphrase privately.
10. On an additional device, run `rein status`, then
    `rein pull --agent claude --session SESSION_ID --dry-run` for a selected
    session, or `rein pull --all --dry-run` only if I explicitly selected all.
    Summarize destinations and backup locations, then request approval before
    the matching mutating command. I will enter the same passphrase privately.
    If Reinstate refuses an existing-target restore because Claude Code is
    active, do not bypass it; tell me to close Claude Code and retry the
    approved command.
11. Verify `rein list --agent claude --json` discovers the expected restored
    session metadata. Have me confirm the session through Claude Code's normal
    resume UI. Do not open or print transcript contents.
12. Return a redacted completion report containing the exact Reinstate version,
    bootstrap URL, checksum result, install path, non-secret profile ID,
    project mapping ID, commands completed, files changed, and remaining
    human-only actions.

Do not declare success if installation, checksum verification, initialization,
storage probing, self-test, dry-run, push/pull, or post-restore discovery
failed.
