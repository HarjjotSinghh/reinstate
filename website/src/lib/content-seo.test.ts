import { describe, expect, it } from 'vitest';
import {
  hasCompleteSocialOverride,
  isSafeContentCanonical,
} from './content-seo';

describe('content SEO overrides', () => {
  it.each([
    'https://reinstate.dev/',
    'https://reinstate.dev/docs/getting-started',
    'https://reinstate.dev/guides/use-cloudflare-r2-for-coding-agent-session-storage',
  ])('accepts a clean same-origin canonical: %s', (value) => {
    expect(isSafeContentCanonical(value)).toBe(true);
  });

  it.each([
    'not a URL',
    'http://reinstate.dev/docs',
    'https://example.com/docs',
    'https://reinstate.dev/Docs',
    'https://reinstate.dev/docs/',
    'https://reinstate.dev/docs?source=duplicate',
    'https://reinstate.dev/docs#section',
  ])('rejects an unsafe or non-canonical URL: %s', (value) => {
    expect(isSafeContentCanonical(value)).toBe(false);
  });

  it('requires custom social images and alternative text as a pair', () => {
    expect(hasCompleteSocialOverride({})).toBe(true);
    expect(
      hasCompleteSocialOverride({
        ogImage: '/og/custom.png',
        ogImageAlt: 'A custom branded Reinstate social card for one guide.',
      }),
    ).toBe(true);
    expect(hasCompleteSocialOverride({ ogImage: '/og/custom.png' })).toBe(false);
    expect(
      hasCompleteSocialOverride({
        ogImageAlt: 'A custom branded Reinstate social card for one guide.',
      }),
    ).toBe(false);
  });
});
