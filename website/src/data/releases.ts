export const releaseHistory = [
  {
    version: 'v0.3.0-rc.4',
    date: '2026-08-08',
    summary:
      'Windows-first corrective Phase 3 candidate: PowerShell 5.1 staging, absolute-path privacy, passphrase-handle safety, and deterministic preflight deadlines.',
  },
  {
    version: 'v0.3.0-rc.3',
    date: '2026-08-07',
    summary:
      'Corrective Phase 3 candidate after RC2 Windows FAIL; native Windows acceptance still failed on PowerShell 5.1 staging and human-output privacy.',
  },
  {
    version: 'v0.3.0-rc.2',
    date: '2026-08-07',
    summary:
      'Corrective Phase 3 candidate after RC1 Windows FAIL. Apple Silicon progress recorded; native Windows tagged-artifact acceptance failed (Codex trust and snapshot/PS gates).',
  },
  {
    version: 'v0.3.0-rc.1',
    date: '2026-08-05',
    summary:
      'First Phase 3 verified-resume candidate. Apple Silicon macOS tagged-artifact acceptance passed; native Windows x64 failed (not stable).',
  },
  {
    version: 'v0.2.0',
    date: '2026-08-05',
    summary:
      'Stable Phase 2 local continuity on verified Apple Silicon macOS and native Windows x64; Intel macOS and Linux/WSL2 artifacts remain preview.',
  },
  {
    version: 'v0.2.0-rc.3',
    date: '2026-08-02',
    summary:
      'Package-manager distribution candidate with verified npm, JSR, Homebrew, Scoop, Chocolatey, WinGet, AUR, and native Linux payload generation.',
  },
  {
    version: 'v0.2.0-rc.2',
    date: '2026-08-02',
    summary:
      'Full release-artifact commit provenance and deterministic native-Windows verification gates.',
  },
  {
    version: 'v0.2.0-rc.1',
    date: '2026-08-01',
    summary:
      'Phase 2 local index, literal search, metadata inspect, switcher, and same-vendor resume/fork release candidate.',
  },
  {
    version: 'v0.1.0',
    date: '2026-07-30',
    summary:
      'First stable release. Phase 1 two-device acceptance passed on v0.1.0-rc.8, whose product code this ships unchanged.',
  },
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
  return version.replace(/[^a-z0-9]+/gi, '-').toLowerCase();
}
