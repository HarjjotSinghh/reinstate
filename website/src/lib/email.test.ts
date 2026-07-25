import { describe, expect, it } from 'vitest';
import { normalizeEmail, validateEmail } from './email';

describe('normalizeEmail', () => {
  it('trims and lowercases', () => {
    expect(normalizeEmail('  Dev@Example.COM ')).toBe('dev@example.com');
  });

  it('returns empty string for non-strings', () => {
    expect(normalizeEmail(null)).toBe('');
    expect(normalizeEmail(42)).toBe('');
  });
});

describe('validateEmail', () => {
  it('accepts a normal address', () => {
    const result = validateEmail('harjot@reinstate.dev');
    expect(result).toEqual({ ok: true, email: 'harjot@reinstate.dev' });
  });

  it('rejects empty', () => {
    const result = validateEmail('   ');
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/required/i);
  });

  it('rejects invalid shapes', () => {
    for (const bad of ['not-an-email', '@x.com', 'a@', 'a@b', 'a b@c.com']) {
      const result = validateEmail(bad);
      expect(result.ok).toBe(false);
    }
  });

  it('rejects overlong emails', () => {
    const local = 'a'.repeat(250);
    const result = validateEmail(`${local}@x.com`);
    expect(result.ok).toBe(false);
  });
});
