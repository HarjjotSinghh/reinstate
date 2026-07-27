import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

function source(relativePath: string): string {
  return readFileSync(new URL(relativePath, import.meta.url), 'utf8');
}

describe('Open Graph brand contract', () => {
  it('keeps the generated-card mark aligned with the bare site mark', () => {
    const card = source('./og-card.ts').replace(/\s+/g, ' ');
    const mark = source('../components/LogoMark.astro').replace(/\s+/g, ' ');
    for (const geometry of [
      'x="10" y="10" width="32" height="32" rx="6"',
      'x="20" y="20" width="32" height="32" rx="6"',
      'M28.5 31 L34 36.5 L28.5 42',
      'M37.5 42 H45',
    ]) {
      expect(card).toContain(geometry);
      expect(mark).toContain(geometry);
    }
  });

  it('pins dimensions, typography, wordmark, and landing palette', () => {
    const card = source('./og-card.ts');
    expect(card).toContain("fontFamily: 'Questrial'");
    expect(card).toContain("fontFamily: 'Geist'");
    expect(card).toContain("'Reinstate'");
    expect(card).toContain("width: '1200px'");
    expect(card).toContain("height: '630px'");
    for (const color of ['#131f1a', '#e4e7dd', '#b8ff3c', '#7ecdf5', '#ffce4a']) {
      expect(card).toContain(color);
    }
  });

  it('keeps the untiled lockup in both global navigation surfaces', () => {
    expect(source('../components/Header.astro')).toMatch(
      /<BrandLockup[\s\S]{0,120}tiled=\{false\}/,
    );
    expect(source('../components/Footer.astro')).toMatch(
      /<BrandLockup[\s\S]{0,120}tiled=\{false\}/,
    );
  });
});
