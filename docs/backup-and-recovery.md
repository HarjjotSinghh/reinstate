# Backup and recovery

Before replacing an existing vendor session, Reinstate copies it under:

```text
~/.reinstate/backups/<timestamp>/<vendor-relative-path>
```

Restores validate/decrypt the full artifact before mutation, write a private
temporary sibling, sync it, back up the existing destination, and atomically
rename the replacement.

List conflict metadata with:

```bash
rein conflicts list
rein conflicts show <id>
```

Resolution strategies perform their operation before removing the record:

- `--keep-local` publishes local state on top of the current remote head.
- `--keep-remote` backs up local state and restores the remote snapshot.
- `--keep-both` preserves local state and restores a vendor-safe fork with a
  distinct structural session ID.

If resolution fails, the conflict JSON remains. Never delete it merely to make
the CLI look clean.
