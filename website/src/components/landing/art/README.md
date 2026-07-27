# Landing art library

Reusable axonometric scenes for the marketing site.

## Contents

| Path | What it is |
|------|------------|
| **`SecurityVaultArt.astro`** | **Live** security art. Four-stage product flow with shared laptop geometry and package continuity. |
| `SecurityPipelineArt.astro` | Archived Windows→age→MacBook pipeline. Keep for other pages. |
| `snapshots/…` | Frozen full-section snapshot. Archival only. |

## Live security story

```
Capture locally  →  Encrypt locally  →  Store in your bucket  →  Restore anywhere
Keys never leave     Encrypted with age    S3-compatible          Same session,
this device            (same package,        storage                new device
                        sealed)            No vendor-hosted
HTTPS on both bucket hops (upload and restore)
```

Rules:

- Build-time 30° iso only (`lib/iso`); one platform component reused.
- Same laptop geometry left and right (state differs only).
- Same package silhouette open → sealed (lime strip continuity).
- Screen-space labels; action-based titles; arrowheads on every hop.
- Sliding drawer storage (no hinged door).
- Lime = path + data + success; titles use softer lime border pills.

## Design rules

`website/AGENTS.md`: true axonometric solids, near-black outlines, three flat
tones, no blur/glass. Light default + dark twin tokens on the art host.
