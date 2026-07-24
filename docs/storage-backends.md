# Storage backends

Phase 1 supports **S3-compatible** object storage (Cloudflare R2 first).

Config fields (no secrets):

```toml
[storage]
type = "s3"
endpoint = "https://<account>.r2.cloudflarestorage.com"
region = "auto"
bucket = "reinstate"
prefix = "profiles/<opaque-profile-id>"
credential_ref = "reinstate/<profile-id>/s3"
```

Credentials via environment:

```bash
export REINSTATE_S3_ACCESS_KEY_ID=...
export REINSTATE_S3_SECRET_ACCESS_KEY=...
```

Remote layout:

```text
<prefix>/manifest.age
<prefix>/snapshots/<opaque-id>.age
<prefix>/probes/<random>
```

Tests use an in-process memory backend (`REINSTATE_BACKEND=memory`) that exercises the same `Backend` interface.
