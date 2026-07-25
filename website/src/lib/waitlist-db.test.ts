import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import {
  ensureWaitlistSchema,
  insertWaitlistEmail,
  listWaitlistEmails,
  resetWaitlistClient,
} from './waitlist-db';

describe('waitlist-db', () => {
  let dir: string;

  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'rein-waitlist-'));
    process.env.TURSO_DATABASE_URL = `file:${join(dir, 'waitlist.db')}`;
    delete process.env.TURSO_AUTH_TOKEN;
    resetWaitlistClient();
  });

  afterEach(() => {
    resetWaitlistClient();
    delete process.env.TURSO_DATABASE_URL;
    rmSync(dir, { recursive: true, force: true });
  });

  it('inserts a valid email and lists it', async () => {
    await ensureWaitlistSchema();
    const result = await insertWaitlistEmail('dev@reinstate.dev');
    expect(result).toEqual({ status: 'created', email: 'dev@reinstate.dev' });

    const rows = await listWaitlistEmails();
    expect(rows.some((r) => r.email === 'dev@reinstate.dev')).toBe(true);
  });

  it('returns duplicate for the same email', async () => {
    await insertWaitlistEmail('dup@example.com');
    const second = await insertWaitlistEmail('dup@example.com');
    expect(second.status).toBe('duplicate');
  });
});
