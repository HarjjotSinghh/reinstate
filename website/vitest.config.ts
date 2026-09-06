import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'node',
    // Windows-only: closing the sqlite-backed waitlist test client can win
    // the JS-level close() race yet still leave the OS holding the file
    // handle for several seconds afterwards (observed 0.5-15s across
    // repeated runs on this host, most likely AV/EDR scanning a brand-new
    // temp file) before the temp directory can actually be unlinked. The
    // default 10s hook timeout is occasionally too tight for that; give it
    // more room than the retry loop in waitlist-db.test.ts / api-routes.test.ts
    // needs, so the loop's own bounded deadline is what decides a real failure.
    hookTimeout: process.platform === "win32" ? 150000 : 10000,
  },
});
