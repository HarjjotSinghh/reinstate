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

Interactive `rein init` stores credentials in the native OS keyring. For
deliberate non-interactive automation only, use the environment provider:

```bash
export REINSTATE_S3_ACCESS_KEY_ID=...
export REINSTATE_S3_SECRET_ACCESS_KEY=...
```

Internally the backend reads keys through a `CredentialSource`. BYO storage
uses the static source above. A source may instead return keys with an expiry
(for example short-lived locker credentials); the backend refreshes them
before they lapse and, if the endpoint rejects a key early, asks the source
again and retries the request once. Conditional puts keep their semantics
across a refresh: a `412 Precondition Failed` is reported as a precondition
error, never mistaken for a credential problem.

Remote layout:

```text
<prefix>/manifest.age
<prefix>/snapshots/<opaque-id>.age
<prefix>/probes/<random>
```

Tests use an in-process memory backend (`REINSTATE_BACKEND=memory`) that exercises the same `Backend` interface.
