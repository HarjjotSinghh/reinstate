import { describe, expect, it } from 'vitest';
import {
  acceptsExplicitly,
  appendVaryAccept,
  notAcceptableBody,
  notAcceptableResponse,
  parseAccept,
  preferredRepresentation,
} from './accept';

describe('parseAccept', () => {
  it('parses types, q-values, and specificity in client order', () => {
    expect(parseAccept('text/markdown, text/html;q=0.8, */*;q=0.1')).toEqual([
      { type: 'text/markdown', q: 1, specificity: 2 },
      { type: 'text/html', q: 0.8, specificity: 2 },
      { type: '*/*', q: 0.1, specificity: 0 },
    ]);
  });

  it('clamps q-values, ignores unknown params, and drops garbage entries', () => {
    expect(parseAccept('text/html;q=5;level=1, nonsense, text/*;q=-1')).toEqual([
      { type: 'text/html', q: 1, specificity: 2 },
      { type: 'text/*', q: 0, specificity: 1 },
    ]);
  });

  it('returns no entries for a missing header', () => {
    expect(parseAccept(null)).toEqual([]);
    expect(parseAccept('')).toEqual([]);
  });
});

describe('preferredRepresentation', () => {
  it('defaults to HTML when the header is missing, empty, or unconstrained', () => {
    expect(preferredRepresentation(null)).toBe('text/html');
    expect(preferredRepresentation('')).toBe('text/html');
    expect(preferredRepresentation('*/*')).toBe('text/html');
    expect(preferredRepresentation('text/*')).toBe('text/html');
  });

  it('serves Markdown when it is asked for first or with a higher q', () => {
    expect(preferredRepresentation('text/markdown')).toBe('text/markdown');
    expect(preferredRepresentation('text/markdown, text/html;q=0.9')).toBe('text/markdown');
    expect(preferredRepresentation('text/html;q=0.5, text/markdown;q=0.9')).toBe('text/markdown');
    expect(preferredRepresentation('text/markdown, text/html, */*')).toBe('text/markdown');
    expect(preferredRepresentation('TEXT/MARKDOWN')).toBe('text/markdown');
  });

  it('keeps HTML when it is asked for first or with a higher q', () => {
    expect(preferredRepresentation('text/html, text/markdown')).toBe('text/html');
    expect(preferredRepresentation('text/html;q=1, text/markdown;q=0.5')).toBe('text/html');
    expect(
      preferredRepresentation('text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8'),
    ).toBe('text/html');
  });

  it('lets a specific q=0 override a wildcard (RFC 9110 §12.5.1)', () => {
    expect(preferredRepresentation('text/html;q=0, */*')).toBe('text/markdown');
    expect(preferredRepresentation('text/markdown;q=0, */*')).toBe('text/html');
  });

  it('returns null when nothing is acceptable (the 406 case)', () => {
    expect(preferredRepresentation('application/pdf')).toBeNull();
    expect(preferredRepresentation('text/markdown;q=0')).toBeNull();
    expect(preferredRepresentation('text/html;q=0, text/markdown;q=0')).toBeNull();
    expect(preferredRepresentation('image/*')).toBeNull();
  });
});

describe('acceptsExplicitly', () => {
  it('only counts the concrete media type with a non-zero q', () => {
    expect(acceptsExplicitly('text/html', 'text/html')).toBe(true);
    expect(acceptsExplicitly('*/*', 'text/html')).toBe(false);
    expect(acceptsExplicitly('text/*', 'text/html')).toBe(false);
    expect(acceptsExplicitly('text/html;q=0', 'text/html')).toBe(false);
    expect(acceptsExplicitly(null, 'text/html')).toBe(false);
  });
});

describe('appendVaryAccept', () => {
  it('adds Accept once and preserves existing values', () => {
    const headers = new Headers();
    appendVaryAccept(headers);
    expect(headers.get('Vary')).toBe('Accept');
    appendVaryAccept(headers);
    expect(headers.get('Vary')).toBe('Accept');

    const encoded = new Headers({ Vary: 'Accept-Encoding' });
    appendVaryAccept(encoded);
    expect(encoded.get('Vary')).toBe('Accept-Encoding, Accept');

    const star = new Headers({ Vary: '*' });
    appendVaryAccept(star);
    expect(star.get('Vary')).toBe('*');
  });
});

describe('406 response', () => {
  it('lists both representations and echoes the request', async () => {
    const body = notAcceptableBody('application/pdf');
    expect(body).toContain('- text/html');
    expect(body).toContain('- text/markdown');
    expect(body).toContain('You requested: application/pdf');

    const response = notAcceptableResponse(null);
    expect(response.status).toBe(406);
    expect(response.headers.get('Content-Type')).toBe('text/plain; charset=utf-8');
    expect(response.headers.get('Vary')).toBe('Accept');
    expect(response.headers.get('Cache-Control')).toBe('no-store');
    expect(await response.text()).toContain('(no Accept header)');
  });
});
