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

/**
 * On Windows, closing the libsql client (resetWaitlistClient, below) can
 * return before the OS finishes releasing its file handle on the sqlite
 * file — deleting the temp directory right after can intermittently fail
 * with EBUSY even though the client is already closed. Retry the removal
 * for a short, bounded window; any other error (or one that outlasts the
 * deadline) still fails the test.
 */
async function removeTempDirWhenReleased(dir: string): Promise<void> {
  const deadline = Date.now() + 120000;
  for (;;) {
    try {
      rmSync(dir, { recursive: true, force: true });
      return;
    } catch (err) {
      const code = (err as NodeJS.ErrnoException).code;
      if (code !== 'EBUSY' || Date.now() > deadline) throw err;
      await new Promise((resolve) => setTimeout(resolve, 25));
    }
  }
}

describe('waitlist-db', () => {
  let dir: string;

  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'rein-waitlist-'));
    process.env.TURSO_DATABASE_URL = `file:${join(dir, 'waitlist.db')}`;
    delete process.env.TURSO_AUTH_TOKEN;
    resetWaitlistClient();
  });

  afterEach(async () => {
    resetWaitlistClient();
    delete process.env.TURSO_DATABASE_URL;
    await removeTempDirWhenReleased(dir);
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
