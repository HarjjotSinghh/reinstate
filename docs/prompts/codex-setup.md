# Codex — Reinstate end-user setup prompt

**Prompt version:** 2
**Release placeholder:** replace `<REINSTATE_VERSION>` with an exact published
tag such as `v0.1.0-rc.1` before pasting.

Copy everything below the line into Codex.

---

Install and configure Reinstate end to end on this device using only the exact
official release `<REINSTATE_VERSION>`. If the placeholder is unchanged, stop
and request an exact version. Never substitute `latest` or `main`.

Hard safety rules:

- Keep normal approval/sandbox controls enabled.
- Never inspect or sync `auth.json`, credentials, API/OAuth tokens, OS
  keychains, `.env`, or transcript content.
- Never ask me to put storage secrets or the encryption passphrase in chat.
- Do not modify unrelated repositories or publish/commit/push anything.

Execution contract:

1. Report Codex version, OS, architecture, shell, native Windows versus WSL,
   existing Reinstate binary, and the proposed user-local install path.
2. Download only from
   `https://github.com/HarjjotSinghh/reinstate/releases/download/<REINSTATE_VERSION>/`.
   The asset contract is
   `reinstate_<VERSION_WITHOUT_LEADING_V>_<os>_<arch>.tar.gz` (`.zip` on
   Windows). Also download `checksums.txt`.
3. Verify SHA-256 before extraction. Refuse a missing checksum entry or
   mismatch. Preserve an existing installation until verification succeeds.
4. Install without elevation to `~/.local/bin` or
   `%LOCALAPPDATA%\Programs\Reinstate\bin`, explain PATH changes, then run
   `rein version` and `rein setup check`.
5. Ask whether this is Device A or an additional device. Collect only
   non-secret setup metadata: endpoint, bucket, canonical project ID, local
   absolute project path, and (for later devices) Device A's `profile_id`.
6. Prepare one command and pause for me to run it:
   - Device A: `rein init --project ID=ABSOLUTE_PATH`
   - later device: `rein init --profile-id UUID --project ID=ABSOLUTE_PATH`
   I will enter storage credentials in Reinstate's hidden terminal prompts.
   Never read or echo them.
7. Continue with `rein doctor --self-test`.
8. On Device A, run `rein push --all --dry-run`; summarize the plan and ask
   before `rein push --all`.
9. On an additional device, run `rein status`, then
   `rein pull --all --dry-run`; summarize destinations/backups and ask before
   `rein pull --all`. If Reinstate refuses an existing-target restore because
   Codex is active, do not bypass it; tell me to close Codex and run the
   approved command.
10. I will type the same encryption passphrase into hidden prompts. Never route
    it through command arguments, ordinary environment variables, chat, or
    logs.
11. Verify `rein list --agent codex --json` discovers the restored session
    metadata at its date-partitioned rollout path. Ask me to confirm it appears
    in Codex's normal resume UI; do not print transcript content.
12. Return a redacted completion report with exact release/checksum, install
    path, profile ID, project mapping, completed gates, changed files, and
    remaining human-only steps.

Failure is not success: stop with the exact redacted error if storage probing,
decryption, artifact validation, restore, or post-restore discovery fails.
