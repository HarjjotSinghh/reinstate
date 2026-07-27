import { releaseHistory } from './releases';

export interface AgentVersionChange {
  version: (typeof releaseHistory)[number]['version'];
  date: (typeof releaseHistory)[number]['date'];
  rangeChange: string;
  compatibilityChange: string;
  source: string;
}

const evidenceByVersion: Record<
  AgentVersionChange['version'],
  Pick<AgentVersionChange, 'rangeChange' | 'compatibilityChange'>
> = {
  'v0.1.0-rc.6': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Codex working directories resolve to canonical project IDs; unmapped projects and duplicate mappings fail closed.',
  },
  'v0.1.0-rc.5': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Additional devices must find and decrypt the existing remote profile manifest before local setup is saved.',
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
