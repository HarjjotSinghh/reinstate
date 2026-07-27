import { describe, expect, it } from 'vitest';
import { createIndexNowKeyResponse } from './indexnow-key';

const KEY = 'abcDEF0123456789-dead-beef';

describe('createIndexNowKeyResponse', () => {
  it('does not disclose whether an unconfigured proof exists', async () => {
    const response = createIndexNowKeyResponse(KEY, undefined);

    expect(response.status).toBe(404);
    expect(await response.text()).toBe('Not found.\n');
  });

  it('rejects malformed configured keys', async () => {
    const response = createIndexNowKeyResponse('too-short', 'bad/key');

    expect(response.status).toBe(404);
    expect(await response.text()).toBe('Not found.\n');
  });

  it('does not disclose the configured key for a different path', async () => {
    const response = createIndexNowKeyResponse('0123456789abcdef', KEY);

    expect(response.status).toBe(404);
    expect(await response.text()).not.toContain(KEY);
  });

  it('returns the exact public ownership proof with defensive headers', async () => {
    const response = createIndexNowKeyResponse(KEY, KEY);

    expect(response.status).toBe(200);
    expect(await response.text()).toBe(`${KEY}\n`);
    expect(response.headers.get('content-type')).toBe(
      'text/plain; charset=utf-8',
    );
    expect(response.headers.get('cache-control')).toBe('no-store');
    expect(response.headers.get('x-content-type-options')).toBe('nosniff');
    expect(response.headers.get('x-robots-tag')).toBe('noindex, nofollow');
  });
});
