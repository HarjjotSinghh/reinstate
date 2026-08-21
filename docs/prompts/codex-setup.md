# Codex — Reinstate end-user setup prompt

**Prompt version:** 9
**Pinned Reinstate release:** `v0.5.1`

Copy everything below the line into Codex.

---

Install and configure Reinstate end to end on this device through the official
public bootstrap pinned to `v0.5.1`. Never substitute `latest`, `main`, or
another version.

Hard safety rules:

- Keep normal approval and sandbox controls enabled.
- Do not clone the repository or build from source.
- Never inspect or sync `auth.json`, credentials, API/OAuth tokens, OS
  keychains, `.env` files, or transcript contents.
- Never ask me to put storage credentials or the encryption passphrase in chat.
  I will enter them privately into Reinstate's hidden terminal prompts.
- Before I enter a passphrase, require the Reinstate process to be visibly
  waiting at its hidden prompt. If it has exited, tell me to stop; never have
  me type a passphrase into the PowerShell prompt.
- Do not modify unrelated repositories or publish, commit, push, purchase, or
  install unrelated software.
- Before running the bootstrap or any Reinstate command, inspect whether
  `REINSTATE_HOME` is already configured. Never unset, replace, or redirect it.
  If it is set, require an absolute path, report that exact effective home, and
  ask me to confirm it. If it is unset, report the default `~/.reinstate` and
  ask me to confirm that instead. Stop on a relative or ambiguous value. Every
  later `rein` command must inherit the confirmed value unchanged.

Execution contract:

1. Report Codex version, OS, architecture, shell, native Windows versus WSL,
   any existing Reinstate binary, and the expected user-local install path.
2. Select exactly one bootstrap:
   - macOS/Linux/WSL:
     `https://reinstate.dev/install.sh`
   - native Windows PowerShell:
     `https://reinstate.dev/install.ps1`
3. Download the bootstrap to a temporary file and inspect it before execution.
   Require all of the following:
   - exact release `v0.5.1`;
   - canonical installer fetched from that exact Git tag;
   - canonical installer checksum verification before execution;
   - no `latest` resolver; and
   - binary downloads restricted to
     `https://github.com/HarjjotSinghh/reinstate/releases/download/v0.5.1/`.
   Stop if any requirement fails.
4. With normal approval, execute the inspected bootstrap. Keep its checksum,
   version, and replacement checks enabled. Install without elevation to
   `~/.local/bin` or `%LOCALAPPDATA%\Programs\Reinstate\bin`, and explain any
   PATH update.
5. Reconfirm the exact effective Reinstate home, then run
   `rein version --json` and require `0.5.0-rc.1`. Run
   `rein setup check`. Before initialization, a missing-config result is
   expected; platform, keyring, or Codex compatibility failures are blockers.
   If the home is already initialized, stop and report it; never re-run `init`
   or choose an overwrite option for me.
6. Ask whether this is Device A or an additional device. Collect only the
   non-secret S3/R2 service endpoint, bucket, canonical project ID, local
   absolute project path, and—for later devices—Device A's non-secret
   `profile_id`. Treat endpoint and bucket as separate values; the endpoint
   must not include a trailing `/<bucket>` path. Ask whether I want one
   explicitly selected Codex session or all discovered sessions. Default to
   one session. Never choose `--all` for me.
7. Prepare one command and pause for me to run it:
   - Device A: `rein init --project ID=ABSOLUTE_PATH`
   - additional device:
     `rein init --profile-id UUID --project ID=ABSOLUTE_PATH`

   Restate the exact effective home and have me use a private interactive
   terminal that inherits the confirmed `REINSTATE_HOME`. I will enter storage
   credentials in Reinstate's hidden prompts. Never read or echo them.
8. Continue with `rein setup check` and `rein doctor --self-test`. Both must
   pass.
9. On Device A, use
   `rein push --agent codex --session SESSION_ID --dry-run` for a selected
   session, or `rein push --all --dry-run` only if I explicitly selected all.
   Require human output beginning with `would push`, not `pushed`, for the
   dry-run. Summarize metadata and request approval before the matching
   mutating command.
10. On an additional device, run `rein status`, followed by
    `rein pull --agent codex --session SESSION_ID --dry-run` for a selected
    session, or `rein pull --all --dry-run` only if I explicitly selected all.
    Summarize destinations/backups and request approval before the matching
    mutating command. If Reinstate refuses an existing-target restore because
    Codex is active, do not bypass it; tell me to close Codex and retry the
    approved command. If the remote manifest is missing or status reports an
    empty revision/zero sessions when sessions are expected, stop and report
    the exact redacted failure.
11. I will type the same encryption passphrase into hidden prompts. Never route
    it through arguments, an ordinary environment variable, chat, or logs.
12. Verify `rein list --agent codex --json` discovers the restored session at a
    date-partitioned rollout path. Ask me to confirm the exact known ID with
    `codex resume SESSION_ID`; do not use picker text search, which can match
    transcript content. Never print transcript content.
13. Return a redacted completion report containing the exact release,
    bootstrap URL, checksum result, install path, non-secret profile ID,
    project mapping, completed gates, changed files, and human-only actions.

Failure is not success. Stop with the exact redacted error if artifact
validation, initialization, storage probing, decryption, restore, or
post-restore discovery fails.
