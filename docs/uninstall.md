# Uninstall

Remove only the binary or symlink you installed:

- macOS/WSL: `~/.local/bin/reinstate` and optional `~/.local/bin/rein`
- Windows: `%LOCALAPPDATA%\Programs\Reinstate\bin\reinstate.exe` and
  `%LOCALAPPDATA%\Programs\Reinstate\bin\rein.exe`

Reinstate data is intentionally separate:

- macOS/Linux/WSL: `~/.reinstate`
- Windows: `%USERPROFILE%\.reinstate`

Keep that directory if you may reinstall or need backups/conflict records.
Deleting it permanently removes local config, state, cache, and backups but
does not delete encrypted objects from your bucket or OS-keyring credentials.
Review exact paths before any recursive deletion.
