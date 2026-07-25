/**
 * Waitlist persistence adapters.
 * Prefer Turso/libsql when TURSO_DATABASE_URL is set; otherwise GitHub Gist.
 */

import { insertWaitlistEmail, listWaitlistEmails, type WaitlistInsertResult } from './waitlist-db';

export type WaitlistStoreResult = WaitlistInsertResult;

type GistFile = { content?: string };
type GistResponse = {
  files?: Record<string, GistFile | undefined>;
  html_url?: string;
};

const GIST_FILENAME = 'waitlist.json';

function hasTurso(): boolean {
  return Boolean(process.env.TURSO_DATABASE_URL || process.env.LIBSQL_URL);
}

function hasGist(): boolean {
  return Boolean(process.env.WAITLIST_GIST_ID && process.env.GITHUB_TOKEN);
}

async function gistFetch(path: string, init?: RequestInit): Promise<Response> {
  const token = process.env.GITHUB_TOKEN;
  if (!token) throw new Error('Missing GITHUB_TOKEN for gist waitlist storage.');
  return fetch(`https://api.github.com${path}`, {
    ...init,
    headers: {
      Accept: 'application/vnd.github+json',
      Authorization: `Bearer ${token}`,
      'X-GitHub-Api-Version': '2022-11-28',
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
}

async function readGistEmails(): Promise<Array<{ email: string; created_at: string; source?: string }>> {
  const id = process.env.WAITLIST_GIST_ID!;
  const res = await gistFetch(`/gists/${id}`);
  if (!res.ok) {
    throw new Error(`Gist read failed: HTTP ${res.status}`);
  }
  const data = (await res.json()) as GistResponse;
  const raw = data.files?.[GIST_FILENAME]?.content ?? '[]';
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map((row) => {
        if (!row || typeof row !== 'object') return null;
        const email = (row as { email?: unknown }).email;
        const created_at = (row as { created_at?: unknown }).created_at;
        const source = (row as { source?: unknown }).source;
        if (typeof email !== 'string' || typeof created_at !== 'string') return null;
        return {
          email,
          created_at,
          source: typeof source === 'string' ? source : 'web',
        };
      })
      .filter((r): r is { email: string; created_at: string; source?: string } => r !== null);
  } catch {
    return [];
  }
}

async function writeGistEmails(
  rows: Array<{ email: string; created_at: string; source?: string }>,
): Promise<void> {
  const id = process.env.WAITLIST_GIST_ID!;
  const res = await gistFetch(`/gists/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({
      files: {
        [GIST_FILENAME]: {
          content: JSON.stringify(rows, null, 2),
        },
      },
    }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Gist write failed: HTTP ${res.status} ${text.slice(0, 200)}`);
  }
}

async function insertViaGist(email: string, source = 'web'): Promise<WaitlistStoreResult> {
  const rows = await readGistEmails();
  if (rows.some((r) => r.email.toLowerCase() === email.toLowerCase())) {
    return { status: 'duplicate', email };
  }
  rows.unshift({
    email,
    created_at: new Date().toISOString(),
    source,
  });
  await writeGistEmails(rows);
  return { status: 'created', email };
}

export async function storeWaitlistEmail(
  email: string,
  source = 'web',
): Promise<WaitlistStoreResult> {
  if (hasTurso()) {
    return insertWaitlistEmail(email, source);
  }
  if (hasGist()) {
    return insertViaGist(email, source);
  }
  throw new Error(
    'No waitlist storage configured. Set TURSO_DATABASE_URL or WAITLIST_GIST_ID + GITHUB_TOKEN.',
  );
}

export async function listStoredWaitlistEmails(limit = 50): Promise<Array<{ email: string; created_at: string }>> {
  if (hasTurso()) {
    return listWaitlistEmails(limit);
  }
  if (hasGist()) {
    const rows = await readGistEmails();
    return rows.slice(0, limit).map((r) => ({ email: r.email, created_at: r.created_at }));
  }
  throw new Error('No waitlist storage configured.');
}
