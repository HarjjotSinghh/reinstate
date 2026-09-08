export const product = {
  name: 'Reinstate',
  cliName: 'rein',
  siteUrl: 'https://reinstate.dev',
  repositoryUrl: 'https://github.com/HarjjotSinghh/reinstate',
  tagsUrl: 'https://github.com/HarjjotSinghh/reinstate/tags',
  licenseName: 'Apache-2.0',
  licenseUrl: 'https://www.apache.org/licenses/LICENSE-2.0',
  category: 'coding-agent continuity',
  shortDefinition:
    'Open-source local coding-agent continuity and encrypted session sync.',
  definition:
    'Reinstate is an open-source tool that finds and continues local coding-agent sessions and syncs supported encrypted sessions across devices.',
  defaultTitle: 'Reinstate: Continue Coding-Agent Sessions Across Devices',
  defaultDescription:
    'Find local coding-agent sessions and sync encrypted Claude Code and Codex work across macOS and Windows with your own S3-compatible storage.',
  defaultOgImage: '/og/home.png',
  defaultOgImageAlt:
    'Reinstate finds local coding-agent sessions and syncs encrypted Claude Code and Codex sessions across devices.',
  supportedAgents: ['Claude Code', 'Codex'],
  supportedOperatingSystems: ['macOS', 'Windows'],
  supportedStorage: ['Amazon S3', 'Cloudflare R2', 'S3-compatible storage'],
  currentRelease: 'v0.6.0-rc.8',
  currentReleaseUrl:
    'https://github.com/HarjjotSinghh/reinstate/tree/v0.6.0-rc.8',
  currentReleaseDate: '2026-09-09',
  initialPublicReleaseDate: '2026-07-25',
  stableRelease: 'v0.5.1',
  stableReleaseUrl:
    'https://github.com/HarjjotSinghh/reinstate/releases/tag/v0.5.1',
  releaseStatus: 'v0.6.0-rc.8 candidate · stable remains v0.5.1 · v0.6.0-rc.7 tagged Windows run ended FAIL (214/0/1/0 of 215), the widened Codex CLI range fully confirmed and every rc.6-carried row cleared to PASS, CLI row 13 (all-projects switcher readiness) the one open row · this candidate fixes exactly that regression · native Windows tagged-artifact acceptance pending, macOS deferred',
  lastVerified: '2026-09-08',
  programmingLanguage: 'Go',
  requiresAccount: false,
  maintainer: {
    name: 'Harjot Singh Rana',
    url: 'https://harjot.co',
    githubUrl: 'https://github.com/HarjjotSinghh',
  },
} as const;

export function siteUrl(path = '/'): string {
  return new URL(path, product.siteUrl).toString();
}

export function ogImagePath(pathname: string): string {
  const normalized = pathname.replace(/^\/+|\/+$/g, '');
  return normalized ? `/og/${normalized}.png` : '/og/home.png';
}
