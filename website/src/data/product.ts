export const product = {
  name: 'Reinstate',
  cliName: 'rein',
  siteUrl: 'https://reinstate.dev',
  repositoryUrl: 'https://github.com/HarjjotSinghh/reinstate',
  tagsUrl: 'https://github.com/HarjjotSinghh/reinstate/tags',
  licenseName: 'Apache-2.0',
  licenseUrl: 'https://www.apache.org/licenses/LICENSE-2.0',
  category: 'coding-agent session sync',
  shortDefinition:
    'Open-source encrypted coding-agent session sync across devices.',
  definition:
    'Reinstate is an open-source tool that syncs encrypted coding-agent sessions across devices, so developers can continue the same Claude Code or Codex work on another computer.',
  defaultTitle: 'Reinstate: Sync Coding-Agent Sessions Across Devices',
  defaultDescription:
    'Sync encrypted Claude Code and Codex sessions across macOS and Windows with Reinstate, using your own S3-compatible storage.',
  defaultOgImage: '/og/home.png',
  defaultOgImageAlt:
    'Reinstate syncs encrypted Claude Code and Codex sessions across devices.',
  supportedAgents: ['Claude Code', 'Codex'],
  supportedOperatingSystems: ['macOS', 'Windows'],
  supportedStorage: ['Amazon S3', 'Cloudflare R2', 'S3-compatible storage'],
  currentRelease: 'v0.1.0-rc.8',
  currentReleaseUrl:
    'https://github.com/HarjjotSinghh/reinstate/tree/v0.1.0-rc.8',
  currentReleaseDate: '2026-07-29',
  initialPublicReleaseDate: '2026-07-25',
  stableRelease: null,
  releaseStatus: 'pre-1.0 release candidate',
  lastVerified: '2026-07-27',
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
