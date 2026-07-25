# Verify an installation

```bash
rein version
rein setup check
```

Complete interactive setup. Device A generates a non-secret profile UUID:

```bash
rein init --project github.com/acme/app=/absolute/local/path
rein doctor --self-test
rein list --agent all --json
rein push --all --dry-run
```

On later devices, reuse the printed UUID:

```bash
rein init --profile-id <DEVICE_A_PROFILE_ID> \
  --project github.com/acme/app=/this/device/path
rein doctor --self-test
rein status
rein pull --all --dry-run
```

`status`, `push`, and `pull` request the encryption passphrase through a hidden
terminal prompt. A dry-run still fetches, authenticates, decrypts, validates,
checks compatibility, and plans destinations; it omits mutations only.

After a real pull, verify metadata with `rein list` and confirm the session in
the vendor's normal resume UI. Do not print transcripts to prove success.
