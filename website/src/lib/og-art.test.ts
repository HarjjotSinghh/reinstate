import { createHash } from 'node:crypto';
import sharp from 'sharp';
import { describe, expect, it } from 'vitest';
import { staticOgPages } from '../data/og-pages';
import { ogArtVariantForRoute, ogArtVariants } from './og-art';
import { renderOgCard } from './og-card';

describe('Open Graph art variants', () => {
  it.each([
    ['/', 'session-stack'],
    ['/security', 'local-encryption'],
    ['/security/?from=test#model', 'local-encryption'],
    ['/docs/storage/', 'owned-storage'],
    ['/guides/use-s3-for-coding-agent-session-storage', 'owned-storage'],
    ['/integrations/codex', 'device-handoff'],
    ['/use-cases/macos-and-windows', 'device-handoff'],
    ['/use-cases/encrypted-session-backup', 'local-encryption'],
    ['/compare/reinstate-vs-git', 'stranded-workstation'],
    ['/docs/troubleshooting', 'stranded-workstation'],
    ['/future-page', 'session-stack'],
  ] as const)('maps %s to %s', (route, expected) => {
    expect(ogArtVariantForRoute(route)).toBe(expected);
  });

  it('uses every landing-derived variant across generated static cards', () => {
    expect(
      new Set(staticOgPages.map((page) => ogArtVariantForRoute(page.route))),
    ).toEqual(new Set(ogArtVariants));
  });

  it('renders all variants as distinct deterministic 1200 × 630 PNGs', async () => {
    const fixture = {
      route: '/og-art-test',
      kind: 'Documentation',
      title: 'Reinstate Open Graph artwork',
      description: 'A deterministic rendering fixture for landing-page artwork.',
    };
    const images = await Promise.all(
      ogArtVariants.map((artVariant) => renderOgCard(fixture, { artVariant })),
    );
    const metadata = await Promise.all(images.map((image) => sharp(image).metadata()));

    for (const image of metadata) {
      expect(image.format).toBe('png');
      expect(image.width).toBe(1_200);
      expect(image.height).toBe(630);
    }

    const hashes = images.map((image) =>
      createHash('sha256').update(image).digest('hex'),
    );
    expect(new Set(hashes).size).toBe(ogArtVariants.length);

    const repeated = await renderOgCard(fixture, {
      artVariant: ogArtVariants[0],
    });
    expect(repeated.equals(images[0])).toBe(true);
  });
});

