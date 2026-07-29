export const releaseHistory = [
  {
    version: 'v0.1.0-rc.8',
    date: '2026-07-29',
    summary:
      'Session liveness detection that no longer depends on an open file handle.',
  },
  {
    version: 'v0.1.0-rc.7',
    date: '2026-07-28',
    summary:
      'Session-scoped restore safety that no longer blocks on unrelated running agents.',
  },
  {
    version: 'v0.1.0-rc.6',
    date: '2026-07-27',
    summary:
      'Canonical project mapping, exact restore checks, and remote-manifest validation.',
  },
  {
    version: 'v0.1.0-rc.5',
    date: '2026-07-27',
    summary:
      'Safer re-initialization, joined-profile checks, and bounded installer confirmation.',
  },
  {
    version: 'v0.1.0-rc.4',
    date: '2026-07-26',
    summary:
      'Destination-aware Claude paths and portable Codex structural paths.',
  },
  {
    version: 'v0.1.0-rc.3',
    date: '2026-07-26',
    summary:
      'Tested version ranges, fail-closed setup checks, and a dependency security patch.',
  },
  {
    version: 'v0.1.0-rc.2',
    date: '2026-07-25',
    summary: 'Signed-tag verification correction.',
  },
  {
    version: 'v0.1.0-rc.1',
    date: '2026-07-25',
    summary: 'Initial Phase 0 and Phase 1 release-candidate foundation.',
  },
] as const;

export function releaseAnchor(version: string): string {
  const candidate = version.match(/-rc\.(\d+)$/)?.[1];
  return candidate ? `rc${candidate}` : version.replace(/[^a-z0-9]+/gi, '-');
}
