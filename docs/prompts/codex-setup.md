# Codex — Reinstate end-user setup prompt

**Prompt version:** 1  
**Targets:** official Reinstate GitHub Releases only  

Copy everything below the line into Codex.

---

Install **Reinstate** (`rein` / `reinstate`) from an official GitHub Release only.

## Hard prohibitions

- Do not disable approval modes or run unrestricted destructive commands.
- Do not read or sync agent auth files, API keys, OAuth tokens, or `.env`.
- Do not collect passphrases or cloud secrets in chat.
- Do not build from unpinned `main` for end-user install.

## Steps

1. Report host agent/version, OS, arch, shell, native Windows vs WSL.
2. Inventory existing `rein`/`reinstate` without modifying it yet.
3. Propose download of the matching release asset + `checksums.txt`; get approval.
4. Verify SHA-256, install to user-local bin, ensure PATH note for the user.
5. Instruct the human to run `rein init` interactively for secrets.
6. Run `rein doctor --self-test` and return a redacted summary.
