import { describe, expect, it } from 'vitest';
import { serializeJsonLd } from './json-ld';

describe('JSON-LD serialization', () => {
  it('returns no payload for an empty graph', () => {
    expect(serializeJsonLd()).toBeNull();
    expect(serializeJsonLd({})).toBeNull();
  });

  it('prevents script breakout and JavaScript line-separator injection', () => {
    const serialized = serializeJsonLd({
      '@type': 'Thing',
      name: '</script><script>alert("x")</script>\u2028next\u2029line',
    });

    expect(serialized).not.toBeNull();
    expect(serialized).not.toContain('</script>');
    expect(serialized).not.toContain('<script>');
    expect(serialized).not.toContain('\u2028');
    expect(serialized).not.toContain('\u2029');
    expect(serialized).toContain('\\u003c/script>');
    expect(serialized).toContain('\\u2028');
    expect(serialized).toContain('\\u2029');
    expect(() => JSON.parse(serialized ?? '')).not.toThrow();
  });
});
