# Open Graph artwork sources

These files are raster captures of artwork already rendered on the Reinstate
landing page. They are not newly illustrated assets.

| Variant | Landing-page source | Intended use |
| --- | --- | --- |
| `session-stack.png` | `ProblemExploded.astro` portable-state card | General product, documentation, releases |
| `stranded-workstation.png` | `ProblemExploded.astro` planning corner | Comparisons, limitations, troubleshooting |
| `device-handoff.png` | `TerminalProof.astro` handoff illustration | Integrations, sync guides, cross-device use cases |
| `local-encryption.png` | `SecurityVaultArt.astro` encryption stage | Security, privacy, encrypted backup |
| `owned-storage.png` | `SecurityVaultArt.astro` storage stage | S3, R2, and storage documentation |

To regenerate the files from the live landing page:

```bash
npm run dev
REINSTATE_OG_BASE_URL=http://127.0.0.1:4321 npm run generate:og-art
```

If Chrome is not in the default macOS location, set
`REINSTATE_CHROME_PATH` to its executable. The capture uses the light landing
palette, transparent section backgrounds, reduced motion, a fixed viewport,
complete or safely isolated artwork bounds, and PNG compression.
