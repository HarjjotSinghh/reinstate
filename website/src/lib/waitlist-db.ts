import { createClient, type Client } from '@libsql/client';
import { resolve } from 'node:path';

export type WaitlistInsertResult =
  | { status: 'created'; email: string }
  | { status: 'duplicate'; email: string };

let client: Client | null = null;

function resolveDatabaseUrl(): string {
  const url = process.env.TURSO_DATABASE_URL ?? process.env.LIBSQL_URL;
  if (!url) {
    throw new Error(
      'Missing TURSO_DATABASE_URL (or LIBSQL_URL). Provision a Turso/libsql database first.',
    );
  }
  // Normalize relative file: paths against process.cwd()
  if (url.startsWith('file:./') || url.startsWith('file:data/')) {
    const pathPart = url.replace(/^file:/, '');
    return `file:${resolve(process.cwd(), pathPart)}`;
  }
  return url;
}

export function getWaitlistClient(): Client {
  if (client) return client;

  const url = resolveDatabaseUrl();
  const authToken =
    process.env.TURSO_AUTH_TOKEN ?? process.env.LIBSQL_AUTH_TOKEN ?? undefined;

  client = createClient({
    url,
    authToken,
  });
  return client;
}

/** Reset cached client (tests only). */
export function resetWaitlistClient(): void {
  client = null;
}

export async function ensureWaitlistSchema(db: Client = getWaitlistClient()): Promise<void> {
  await db.execute(`
    CREATE TABLE IF NOT EXISTS waitlist (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      email TEXT NOT NULL UNIQUE COLLATE NOCASE,
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      source TEXT NOT NULL DEFAULT 'web'
    )
  `);
  await db.execute(`
    CREATE INDEX IF NOT EXISTS idx_waitlist_created_at ON waitlist(created_at)
  `);
}

export async function insertWaitlistEmail(
  email: string,
  source = 'web',
  db: Client = getWaitlistClient(),
): Promise<WaitlistInsertResult> {
  await ensureWaitlistSchema(db);
  try {
    await db.execute({
      sql: 'INSERT INTO waitlist (email, source) VALUES (?, ?)',
      args: [email, source],
    });
    return { status: 'created', email };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    if (/UNIQUE|unique/i.test(message)) {
      return { status: 'duplicate', email };
    }
    throw err;
  }
}

export async function listWaitlistEmails(
  limit = 50,
  db: Client = getWaitlistClient(),
): Promise<Array<{ email: string; created_at: string }>> {
  await ensureWaitlistSchema(db);
  const result = await db.execute({
    sql: 'SELECT email, created_at FROM waitlist ORDER BY created_at DESC LIMIT ?',
    args: [limit],
  });
  return result.rows.map((row) => ({
    email: String(row.email),
    created_at: String(row.created_at),
  }));
}
