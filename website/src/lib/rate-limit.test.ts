import { describe, expect, it } from 'vitest';
import { SlidingWindowLimiter, clientKey, rateLimitHeaders } from './rate-limit';

const policy = { name: 'test', quota: 3, windowSeconds: 60 };

describe('SlidingWindowLimiter', () => {
  it('allows up to the quota inside a window and reports remaining and reset', () => {
    const limiter = new SlidingWindowLimiter(policy);
    const t0 = 1_000_000;
    const first = limiter.hit('ip', t0);
    expect(first.allowed).toBe(true);
    expect(first.remaining).toBe(2);
    expect(first.resetSeconds).toBe(60);
    expect(first.headers).toEqual({
      'RateLimit-Policy': '"test";q=3;w=60',
      RateLimit: '"test";r=2;t=60',
      'RateLimit-Limit': '3',
      'RateLimit-Remaining': '2',
      'RateLimit-Reset': '60',
    });
    limiter.hit('ip', t0 + 1000);
    const third = limiter.hit('ip', t0 + 2000);
    expect(third.allowed).toBe(true);
    expect(third.remaining).toBe(0);
    const fourth = limiter.hit('ip', t0 + 3000);
    expect(fourth.allowed).toBe(false);
    expect(fourth.remaining).toBe(0);
    expect(fourth.resetSeconds).toBe(57);
    expect(fourth.headers.RateLimit).toBe('"test";r=0;t=57');
  });

  it('frees quota as old hits leave the window and keeps keys independent', () => {
    const limiter = new SlidingWindowLimiter(policy);
    const t0 = 5_000_000;
    for (let i = 0; i < 3; i += 1) limiter.hit('a', t0 + i);
    expect(limiter.hit('a', t0 + 10).allowed).toBe(false);
    expect(limiter.hit('b', t0 + 10).allowed).toBe(true);
    expect(limiter.hit('a', t0 + 60_001).allowed).toBe(true);
    expect(limiter.count('a', t0 + 60_001)).toBe(2);
  });

  it('bounds memory by evicting the oldest keys', () => {
    const limiter = new SlidingWindowLimiter(policy, { maxKeys: 2 });
    limiter.hit('one', 1);
    limiter.hit('two', 2);
    limiter.hit('three', 3);
    expect(limiter.count('one', 4)).toBe(0);
    expect(limiter.count('three', 4)).toBe(1);
  });
});

describe('clientKey', () => {
  it('prefers the first forwarded address, then x-real-ip, then the socket address', () => {
    expect(clientKey(new Request('https://x.test/', { headers: { 'x-forwarded-for': '203.0.113.5, 10.0.0.1' } }), '10.0.0.9')).toBe('203.0.113.5');
    expect(clientKey(new Request('https://x.test/', { headers: { 'x-real-ip': '198.51.100.7' } }), '10.0.0.9')).toBe('198.51.100.7');
    expect(clientKey(new Request('https://x.test/'), '10.0.0.9')).toBe('10.0.0.9');
    expect(clientKey(new Request('https://x.test/'))).toBe('anonymous');
  });
});

describe('rateLimitHeaders', () => {
  it('emits both the structured fields and the compatibility trio', () => {
    const headers = rateLimitHeaders({ name: 'api', quota: 60, windowSeconds: 60 }, 59, 12);
    expect(headers['RateLimit-Policy']).toBe('"api";q=60;w=60');
    expect(headers.RateLimit).toBe('"api";r=59;t=12');
    expect(headers['RateLimit-Limit']).toBe('60');
    expect(headers['RateLimit-Remaining']).toBe('59');
    expect(headers['RateLimit-Reset']).toBe('12');
  });
});
