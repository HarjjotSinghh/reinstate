export const releaseHistory = [
  {
    version: 'v0.4.0-rc.7',
    date: '2026-08-15',
    summary:
      'Seventh Phase 4 candidate after v0.4.0-rc.6 physical dual-platform acceptance FAILED (Windows WMI busy-check cascade; C8/R6 2.1.230 fail-open; R4 leftover Runtime). Carries those product fixes; tagged-artifact acceptance is pending.',
  },
  {
    version: 'v0.4.0-rc.6',
    date: '2026-08-14',
    summary:
      'Sixth Phase 4 candidate after v0.4.0-rc.5 physical dual-platform acceptance FAILED (dest-ack/picker without a console; C4 remapped fixture/foreign workspaces onto any git cwd). Physical dual-platform acceptance FAILED: Windows Plan rows died on a ~305s WMI busy-check; macOS was 37 PASS / 7 FAIL.',
  },
  {
    version: 'v0.4.0-rc.5',
    date: '2026-08-13',
    summary:
      'Fifth Phase 4 candidate after v0.4.0-rc.4 physical dual-platform acceptance FAILED (Windows R4 hang 43s, F8 ~6s non-TTY delay, B6 summarized missing, C5 Windows os-roots exit 5). Physical dual-platform acceptance FAILED.',
  },
  {
    version: 'v0.4.0-rc.4',
    date: '2026-08-13',
    summary:
      'Fourth Phase 4 candidate after v0.4.0-rc.3 physical dual-platform acceptance FAILED (empty dest home UNTESTED blocking flagship A, hanging --version, Windows make verify, G3 list n=0, B6 missing summarized). Physical dual-platform acceptance FAILED.',
  },
  {
    version: 'v0.4.0-rc.3',
    date: '2026-08-13',
    summary:
      'Third Phase 4 candidate after v0.4.0-rc.2 physical dual-platform acceptance FAILED (wrong-repo cwd, non-TTY spawn, Grok-source busy-check, timed-out probe classified Runtime). Physical dual-platform acceptance FAILED.',
  },
  {
    version: 'v0.4.0-rc.2',
    date: '2026-08-13',
    summary:
      'Second Phase 4 candidate after v0.4.0-rc.1 physical acceptance FAILED: Claude source version probing, path tokenization, live changed-file reporting, probe-timeout classification, and prose-vs-path capsule validation are fixed. Physical dual-platform acceptance FAILED.',
  },
  {
    version: 'v0.4.0-rc.1',
    date: '2026-08-12',
    summary:
      'First Phase 4 candidate: explicit structured handoff of the same task into a new Claude Code or Codex session. Dual-platform tagged-artifact acceptance is pending.',
  },
  {
    version: 'v0.3.0',
    date: '2026-08-11',
    summary:
      'Phase 3 stable: verified resume for Claude Code and Codex after dual-platform RC7 acceptance PASS.',
  },
  {
    version: 'v0.3.0-rc.7',
    date: '2026-08-11',
    summary:
      'Phase 3 candidate after RC6 dual FAIL: packages non-TTY fail-closed, Windows Ctrl+C safety, capability probe demotion, isolated agent homes, and expanded local Phase 3 smoke for retest.',
  },
  {
    version: 'v0.3.0-rc.6',
    date: '2026-08-11',
    summary:
      'Phase 3 candidate that widens Claude Code through 2.1.227 and Codex CLI through 0.147.0 after RC5 host-version acceptance failures.',
  },
  {
    version: 'v0.3.0-rc.5',
    date: '2026-08-08',
    summary:
      'Corrective Phase 3 candidate after RC4 stayed unpublished: portable PowerShell artifact verification with the Windows-first product fixes intact.',
  },
  {
    version: 'v0.3.0-rc.4',
    date: '2026-08-08',
    summary:
      'Windows-first Phase 3 candidate whose signed-tag workflow failed before publication during Ubuntu PowerShell artifact verification.',
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
