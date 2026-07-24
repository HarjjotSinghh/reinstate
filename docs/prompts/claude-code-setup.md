# Claude Code — Reinstate end-user setup prompt

**Prompt version:** 1  
**Targets:** official Reinstate GitHub Releases only  

Copy everything below the line into Claude Code.

---

You are helping install **Reinstate** (CLI alias `rein` / `reinstate`) from an official GitHub Release.

## Hard prohibitions

- Do not use `--yolo`, danger-full-access, or disable sandbox/approvals.
- Do not read, copy, print, or sync `auth.json`, `.credentials.json`, API keys, OAuth tokens, OS keychains, or `.env` files.
- Do not ask the user to paste passphrases or storage secrets into chat.
- Do not install from `main`, an unpinned branch, or an unverified artifact.
- Do not modify unrelated repositories or publish/commit/push anything.

## Steps

1. Detect and report: OS, arch, shell, native Windows vs WSL, and whether `rein`/`reinstate` already exists.
2. Show the planned install path and download URL; wait for approval before elevating privileges.
3. Download the exact release asset for this platform from `https://github.com/HarjjotSinghh/reinstate/releases`.
4. Download `checksums.txt` and verify SHA-256 of the archive before extract.
5. Install the binary to a user-local path (e.g. `~/.local/bin`).
6. Hand interactive secrets to the human: tell them to run `rein init` themselves in a private terminal.
7. Run `rein doctor --self-test` (or `reinstate doctor --self-test`).
8. Report redacted results: versions, files changed, remaining human actions. Never print secrets or home-expanded credential paths.
