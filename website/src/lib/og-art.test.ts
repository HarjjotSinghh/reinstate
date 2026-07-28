import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
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

  it('keeps every landing capture transparent and safely inside its bitmap', async () => {
    for (const variant of ogArtVariants) {
      const image = readFileSync(
        new URL(`../assets/og-art/${variant}.png`, import.meta.url),
      );
      const { data, info } = await sharp(image)
        .ensureAlpha()
        .raw()
        .toBuffer({ resolveWithObject: true });
      let minX = info.width;
      let minY = info.height;
      let maxX = -1;
      let maxY = -1;

      for (let y = 0; y < info.height; y += 1) {
        for (let x = 0; x < info.width; x += 1) {
          const alpha = data[(y * info.width + x) * info.channels + 3];
          if (alpha <= 8) continue;
          minX = Math.min(minX, x);
          minY = Math.min(minY, y);
          maxX = Math.max(maxX, x);
          maxY = Math.max(maxY, y);
        }
      }

      expect(maxX, `${variant}: visible artwork`).toBeGreaterThanOrEqual(0);
      expect(minX, `${variant}: left safe inset`).toBeGreaterThanOrEqual(5);
      expect(minY, `${variant}: top safe inset`).toBeGreaterThanOrEqual(5);
      expect(info.width - 1 - maxX, `${variant}: right safe inset`).toBeGreaterThanOrEqual(
        5,
      );
      expect(
        info.height - 1 - maxY,
        `${variant}: bottom safe inset`,
      ).toBeGreaterThanOrEqual(5);
    }
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
