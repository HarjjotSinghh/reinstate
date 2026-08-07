import { releaseHistory } from './releases';

export interface AgentVersionChange {
  version: (typeof releaseHistory)[number]['version'];
  date: (typeof releaseHistory)[number]['date'];
  rangeChange: string;
  compatibilityChange: string;
  source: string;
  implementationSource?: string;
}

const evidenceByVersion: Record<
  AgentVersionChange['version'],
  Pick<
    AgentVersionChange,
    'rangeChange' | 'compatibilityChange' | 'implementationSource'
  >
> = {
  'v0.3.0-rc.3': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Hardens Windows trusted executable resolution and PowerShell snapshot/staging gates after RC2 native Windows FAIL without widening Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0. Tagged-artifact acceptance is pending.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.3/internal/executabletrust/resolve.go',
  },
  'v0.3.0-rc.2': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Corrects Windows trusted executable resolution for extensionless vendor names via PATHEXT without widening Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0. Tagged-artifact acceptance is pending.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.2/internal/executabletrust/resolve.go',
  },
  'v0.3.0-rc.1': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Adds Phase 3 verified-resume observation and authorization without widening the Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0 ranges. macOS acceptance passed; native Windows failed.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.1/internal/preflight/verify.go',
  },
  'v0.2.0': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Ships the RC2-tested runtime adapter tree unchanged. Apple Silicon macOS and native Windows x64 are verified stable platforms; Intel macOS and Linux/WSL2 artifacts remain preview until their deferred physical acceptance completes.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.2.0/internal/adapter/codex/codex.go',
  },
  'v0.2.0-rc.3': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'No adapter behavior change from v0.2.0-rc.2. This candidate adds verified package-manager distribution metadata and native package artifacts without widening the tested Claude Code or Codex CLI ranges.',
  },
  'v0.2.0-rc.2': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'No adapter behavior change from v0.2.0-rc.1. This candidate corrects release-artifact provenance and native-Windows verification without widening the tested Claude Code or Codex CLI ranges.',
  },
  'v0.2.0-rc.1': {
    rangeChange:
      'Expanded the inclusive Codex CLI range from 0.133.0–0.145.0 to 0.133.0–0.146.0; the Claude Code range remains 2.1.219–2.1.220.',
    compatibilityChange:
      'Codex CLI 0.146.0 passed the complete Phase 2 physical matrix on both macOS and native Windows. The adapter still fails closed for stable versions above 0.146.0 and for every prerelease.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.2.0-rc.1/internal/adapter/codex/codex.go',
  },
  'v0.1.0': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'No behavior change from v0.1.0-rc.8. The stable release ships that candidate\u2019s product code unchanged, so the two-device Phase 1 acceptance evidence applies directly to it.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0/internal/processcheck/process.go',
  },
  'v0.1.0-rc.8': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Restore liveness no longer depends on an open file handle. Claude Code closes its session file between appends, so a live session held no handle and RC7 treated it as free. Detection now also matches an agent naming the exact session on its command line, or working inside the session\u2019s mapped project.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.8/internal/processcheck/process.go',
  },
  'v0.1.0-rc.7': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Restore liveness detection became session-scoped. A restore now checks whether the exact target session file is held open, using operating-system file handles, instead of asking whether any Claude Code or Codex process is running. A session that is genuinely in use is restored alongside the live one rather than refused.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.7/internal/processcheck/process.go',
  },
  'v0.1.0-rc.6': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Codex working directories resolve to canonical project IDs. Additional-device init changed from RC5’s metadata-only manifest probe to reading the complete remote ciphertext; init still does not decrypt it or validate the passphrase.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.6/internal/cli/commands_impl.go',
  },
  'v0.1.0-rc.5': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Additional-device init required a metadata-only existence probe for the remote manifest before saving local setup; it did not read or decrypt the ciphertext.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.5/internal/cli/commands_impl.go',
  },
  'v0.1.0-rc.4': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Claude restore paths became destination-aware, legacy unmapped snapshots fail closed, and exact native restore locations are verified.',
  },
  'v0.1.0-rc.3': {
    rangeChange:
      'Introduced the current inclusive ranges: Claude Code 2.1.219–2.1.220 and Codex CLI 0.133.0–0.145.0.',
    compatibilityChange:
      'Installed versions outside a tested stable range become UNTESTED and setup check exits with compatibility code 5.',
  },
  'v0.1.0-rc.2': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Release CI corrected signed-tag verification; no adapter compatibility change is recorded.',
  },
  'v0.1.0-rc.1': {
    rangeChange:
      'Claude Code and Codex adapters launched with fixture-backed checks, but this changelog does not publish an exact historical version range.',
    compatibilityChange:
      'Established fail-closed adapter compatibility states and same-vendor session restore.',
  },
};

export const agentVersionHistory: AgentVersionChange[] = releaseHistory.map(
  ({ version, date }) => ({
    version,
    date,
    ...evidenceByVersion[version],
    source: `https://github.com/HarjjotSinghh/reinstate/blob/${version}/CHANGELOG.md`,
  }),
);
