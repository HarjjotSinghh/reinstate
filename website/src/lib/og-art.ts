export const ogArtVariants = [
  'session-stack',
  'stranded-workstation',
  'device-handoff',
  'local-encryption',
  'owned-storage',
] as const;

export type OgArtVariant = (typeof ogArtVariants)[number];

function normalizedRoute(route: string): string {
  const pathname = route.split(/[?#]/, 1)[0] || '/';
  const withLeadingSlash = pathname.startsWith('/') ? pathname : `/${pathname}`;
  return withLeadingSlash === '/' ? '/' : withLeadingSlash.replace(/\/+$/, '');
}

function previewVariant(route: string): OgArtVariant {
  let hash = 0;
  for (const character of route) {
    hash = (hash * 31 + character.charCodeAt(0)) >>> 0;
  }
  return ogArtVariants[hash % ogArtVariants.length];
}

const localEncryptionRoutes = new Set([
  '/security',
  '/privacy',
  '/docs/security-model',
  '/research/encrypted-snapshot-format-v1',
  '/use-cases/encrypted-session-backup',
]);

const ownedStorageRoutes = new Set([
  '/docs/storage',
  '/guides/use-s3-for-coding-agent-session-storage',
  '/guides/use-cloudflare-r2-for-coding-agent-session-storage',
]);

const deviceHandoffRoutes = new Set([
  '/compatibility',
  '/compatibility/agent-version-history',
  '/docs/adapters',
  '/docs/architecture',
  '/docs/configuration',
  '/docs/getting-started',
  '/docs/installation',
  '/docs/restore-a-session',
  '/docs/sync-a-session',
  '/guides/move-a-coding-agent-session-from-mac-to-windows',
  '/guides/sync-claude-code-sessions-across-devices',
  '/guides/sync-codex-sessions-across-devices',
  '/tools/path-mapping-visualizer',
]);

const strandedWorkstationRoutes = new Set([
  '/404',
  '/blog/why-git-does-not-sync-coding-agent-sessions',
  '/docs/comparison',
  '/docs/limitations',
  '/docs/troubleshooting',
]);

export function ogArtVariantForRoute(route: string): OgArtVariant {
  const normalized = normalizedRoute(route);

  if (normalized === '/preview' || normalized.startsWith('/preview/')) {
    return previewVariant(normalized);
  }
  if (localEncryptionRoutes.has(normalized)) return 'local-encryption';
  if (ownedStorageRoutes.has(normalized)) return 'owned-storage';
  if (
    deviceHandoffRoutes.has(normalized) ||
    normalized === '/integrations' ||
    normalized.startsWith('/integrations/') ||
    normalized === '/use-cases' ||
    normalized.startsWith('/use-cases/')
  ) {
    return 'device-handoff';
  }
  if (
    strandedWorkstationRoutes.has(normalized) ||
    normalized === '/compare' ||
    normalized.startsWith('/compare/')
  ) {
    return 'stranded-workstation';
  }

  return 'session-stack';
}

