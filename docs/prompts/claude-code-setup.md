# Claude Code — Reinstate end-user setup prompt

**Prompt version:** 2
**Release placeholder:** replace `<REINSTATE_VERSION>` with an exact published
tag such as `v0.1.0-rc.1` before pasting.

Copy everything below the line into Claude Code.

---

Set up Reinstate end to end on this device from the exact official release
`<REINSTATE_VERSION>`. If that placeholder was not replaced, stop and ask me
for the exact version. Do not silently choose `latest`.

Safety rules:

- Use only `https://github.com/HarjjotSinghh/reinstate/releases/download/<REINSTATE_VERSION>/`.
- Do not clone or install from `main`.
- Do not disable approvals/sandboxing or use destructive broad commands.
- Never read, copy, print, log, or sync agent auth files, keychains, API keys,
  OAuth tokens, `.env`, or credential stores.
- Never ask me to paste a storage secret or encryption passphrase into chat.
- Do not publish, commit, push, purchase, or modify unrelated repositories.

Complete this workflow:

1. Detect OS, architecture, shell, native Windows versus WSL, Claude Code
   version, and any existing `rein`/`reinstate` binary. Report the install plan.
2. Derive the exact archive name:
   `reinstate_<VERSION_WITHOUT_LEADING_V>_<os>_<arch>.tar.gz`, or `.zip` on
   native Windows.
3. Download that archive and `checksums.txt` from the exact release URL. Refuse
   missing assets, redirects outside GitHub, or checksum mismatch.
4. Install to a user-local directory without elevation (`~/.local/bin` on
   macOS/WSL; `%LOCALAPPDATA%\Programs\Reinstate\bin` on Windows). Preserve a
   valid existing install until the new archive is verified. Explain any PATH
   change.
5. Run `rein version` and `rein setup check`. Show only redacted output.
6. Ask whether this is the first or an additional device, and ask for:
   - the non-secret S3/R2 endpoint and bucket;
   - the canonical project ID and this device's absolute project path;
   - on an additional device, the non-secret `profile_id` printed by Device A.
7. Construct the appropriate command:
   - Device A: `rein init --project ID=ABSOLUTE_PATH`
   - later device: `rein init --profile-id UUID --project ID=ABSOLUTE_PATH`
   Then pause and have me run it in the private terminal. I will type storage
   credentials into its hidden prompts. Do not read or echo them. Confirm that
   Device A's printed profile UUID is saved for Device B.
8. Resume after init. Run `rein doctor --self-test`.
9. If this is Device A, run `rein push --all --dry-run`, show the destination
   plan, then ask before `rein push --all`. When Reinstate prompts, I will enter
   the encryption passphrase privately.
10. If this is an additional device, run `rein status`, then
    `rein pull --all --dry-run`. Show destination paths/backups and ask before
    `rein pull --all`. I will enter the same passphrase privately. If Reinstate
    refuses an existing-target restore because Claude Code is active, do not
    bypass it; tell me to close Claude Code and run the approved command.
11. Verify `rein list --agent claude --json` discovers the expected restored
    session metadata. Tell me to confirm with Claude Code's normal resume UI;
    do not open or print transcript contents.
12. Finish with a redacted report: exact Reinstate version, archive/checksum,
    install path, profile ID, project mapping ID, commands completed, files
    changed, and any remaining human-only action.

Do not declare success if init, the storage probe, self-test, dry-run,
push/pull, or post-restore discovery failed.
