import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { product, siteUrl } from '../data/product';
import {
  breadcrumbSchema,
  homepageSchema,
  techArticleSchema,
  webPageSchema,
} from './schema';

describe('SEO product truth', () => {
  it('protects the current stable product boundary', () => {
    expect(product.supportedAgents).toEqual(['Claude Code', 'Codex']);
    expect(product.supportedOperatingSystems).toEqual(['macOS', 'Windows']);
    expect(product.licenseName).toBe('Apache-2.0');
    expect(product.requiresAccount).toBe(false);
    expect(product.releaseStatus).toBe(
      'v0.4.0-rc.4 candidate · tagged-artifact acceptance pending',
    );
    expect(product.initialPublicReleaseDate).toBe('2026-07-25');
    expect(product.stableRelease).toBe('v0.3.0');
  });

  it('keeps the advertised release synchronized with the changelog', () => {
    const changelogPath = fileURLToPath(new URL('../../../CHANGELOG.md', import.meta.url));
    const changelog = readFileSync(changelogPath, 'utf8');
    expect(changelog).toContain(`## [${product.currentRelease.slice(1)}]`);
  });

  it('builds only canonical production URLs', () => {
    expect(siteUrl('/docs/getting-started')).toBe(
      'https://reinstate.dev/docs/getting-started',
    );
  });
});

describe('structured data builders', () => {
  it('publishes truthful homepage entities without fabricated social proof', () => {
    const serialized = JSON.stringify(homepageSchema());
    expect(serialized).toContain('"SoftwareApplication"');
    expect(serialized).toContain('"SoftwareSourceCode"');
    expect(serialized).toContain(product.currentRelease);
    expect(serialized).not.toContain('aggregateRating');
    expect(serialized).not.toContain('"review"');
  });

  it('uses absolute breadcrumb URLs and stable positions', () => {
    const schema = breadcrumbSchema([
      { name: 'Home', path: '/' },
      { name: 'Docs', path: '/docs' },
    ]);
    expect(schema.itemListElement).toEqual([
      {
        '@type': 'ListItem',
        position: 1,
        name: 'Home',
        item: 'https://reinstate.dev/',
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Docs',
        item: 'https://reinstate.dev/docs',
      },
    ]);
  });

  it('emits valid webpage and article dates', () => {
    const updatedAt = new Date('2026-07-27T00:00:00Z');
    expect(
      webPageSchema({
        path: '/compatibility',
        title: 'Compatibility',
        description: 'Current compatibility.',
        updatedAt,
      }).dateModified,
    ).toBe(updatedAt.toISOString());
    expect(
      techArticleSchema({
        path: '/docs/architecture',
        title: 'Architecture',
        description: 'Reinstate architecture.',
        updatedAt,
      }).dateModified,
    ).toBe(updatedAt.toISOString());
  });
});
