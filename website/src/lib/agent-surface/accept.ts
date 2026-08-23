/**
 * Accept-header negotiation between the two representations every page on
 * reinstate.dev can produce: HTML for browsers and Markdown for agents.
 *
 * Follows the acceptmarkdown.com convention and RFC 9110 §12.5.1: rank by
 * q-value, let more specific media ranges override wildcards, treat `q=0` as
 * an explicit rejection, break ties by client order, and fall back to HTML
 * when the header is missing or unconstrained.
 */

export type Representation = 'text/html' | 'text/markdown';

export const REPRESENTATIONS: readonly Representation[] = ['text/html', 'text/markdown'];

export const HTML_CONTENT_TYPE = 'text/html; charset=utf-8';
export const MARKDOWN_CONTENT_TYPE = 'text/markdown; charset=utf-8';

export interface AcceptEntry {
  type: string;
  q: number;
  /** 0 for `*\/*`, 1 for `type/*`, 2 for a concrete media type. */
  specificity: number;
}

export function parseAccept(header: string | null | undefined): AcceptEntry[] {
  if (!header) return [];
  return header
    .split(',')
    .map((raw) => raw.trim())
    .filter(Boolean)
    .map((raw) => {
      const parts = raw.split(';').map((part) => part.trim());
      const type = (parts[0] ?? '').toLowerCase();
      let q = 1;
      for (const param of parts.slice(1)) {
        const [name, value] = param.split('=').map((part) => part.trim());
        if (name?.toLowerCase() === 'q') {
          const parsed = Number(value);
          if (!Number.isNaN(parsed)) q = Math.max(0, Math.min(1, parsed));
        }
      }
      const specificity = type === '*/*' ? 0 : type.endsWith('/*') ? 1 : 2;
      return { type, q, specificity };
    })
    .filter((entry) => entry.type.includes('/'));
}

function matches(entry: AcceptEntry, candidate: string): boolean {
  if (entry.type === '*/*') return true;
  if (entry.type.endsWith('/*')) return candidate.startsWith(entry.type.slice(0, -1));
  return entry.type === candidate;
}

/**
 * Picks the representation the client prefers, or `null` when every
 * representation is rejected (the 406 case). A missing or empty header means
 * "no constraint" and yields the first candidate.
 */
export function preferredRepresentation(
  header: string | null | undefined,
  produces: readonly Representation[] = REPRESENTATIONS,
): Representation | null {
  const entries = parseAccept(header);
  if (entries.length === 0) return produces[0] ?? null;

  let best: Representation | null = null;
  let bestQ = -1;
  let bestPosition = Number.POSITIVE_INFINITY;

  for (const candidate of produces) {
    let matched: AcceptEntry | null = null;
    let matchedPosition = Number.POSITIVE_INFINITY;
    for (let index = 0; index < entries.length; index += 1) {
      const entry = entries[index]!;
      if (!matches(entry, candidate)) continue;
      if (
        matched === null ||
        entry.specificity > matched.specificity ||
        (entry.specificity === matched.specificity && index < matchedPosition)
      ) {
        matched = entry;
        matchedPosition = index;
      }
    }
    if (matched === null || matched.q <= 0) continue;
    if (matched.q > bestQ || (matched.q === bestQ && matchedPosition < bestPosition)) {
      bestQ = matched.q;
      bestPosition = matchedPosition;
      best = candidate;
    }
  }

  return best;
}

/** True when the client lists the concrete media type itself with a non-zero q. */
export function acceptsExplicitly(header: string | null | undefined, type: Representation): boolean {
  return parseAccept(header).some((entry) => entry.type === type && entry.q > 0);
}

/** Adds `Accept` to `Vary` without duplicating it or clobbering other values. */
export function appendVaryAccept(headers: Headers): void {
  const existing = headers.get('Vary');
  if (!existing) {
    headers.set('Vary', 'Accept');
    return;
  }
  const tokens = existing.split(',').map((token) => token.trim().toLowerCase());
  if (tokens.includes('*')) return;
  if (!tokens.includes('accept')) headers.set('Vary', `${existing}, Accept`);
}

export function notAcceptableBody(requested: string | null | undefined): string {
  return [
    'Not Acceptable',
    '',
    'This resource is available in:',
    ...REPRESENTATIONS.map((type) => `- ${type}`),
    '',
    `You requested: ${requested?.trim() || '(no Accept header)'}`,
    '',
    'Send Accept: text/markdown for the Markdown representation or Accept: text/html for the page.',
    '',
  ].join('\n');
}

export function notAcceptableResponse(requested: string | null | undefined): Response {
  return new Response(notAcceptableBody(requested), {
    status: 406,
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'no-store',
      Vary: 'Accept',
    },
  });
}
