/**
 * Waitlist API handlers shared by the canonical `/api/v1/waitlist` route and
 * the deprecated unversioned alias `/api/waitlist`.
 *
 * Versioning policy (RFC 9745 deprecation signalling): stable operations live
 * under a major-version path (`/api/v1/`). Breaking changes require a new
 * major path. A deprecated path keeps working and advertises `Deprecation`,
 * `Link rel="successor-version"`, and, once a removal date exists, `Sunset`.
 */
import type { APIRoute } from 'astro';
import { validateEmail } from './email';
import { storeWaitlistEmail } from './waitlist-store';
import { apiError, apiJson, methodNotAllowed } from './api-errors';
import { siteUrl } from '../data/product';

export const WAITLIST_METHODS = ['GET', 'POST'] as const;
export const WAITLIST_V1_PATH = '/api/v1/waitlist';
export const WAITLIST_LEGACY_PATH = '/api/waitlist';

/** The unversioned alias was deprecated when `/api/v1/` shipped (RFC 9745 `@` seconds form). */
export const WAITLIST_LEGACY_DEPRECATED_AT = '2026-08-23T00:00:00Z';

export function deprecationHeaders(): Record<string, string> {
  const seconds = Math.floor(Date.parse(WAITLIST_LEGACY_DEPRECATED_AT) / 1000);
  return {
    Deprecation: `@${seconds}`,
    Link: `<${siteUrl(WAITLIST_V1_PATH)}>; rel="successor-version"`,
  };
}

/** Optional Resend notify — never required for success. */
async function maybeNotifyResend(email: string): Promise<void> {
  const apiKey = process.env.RESEND_API_KEY;
  const to = process.env.WAITLIST_NOTIFY_TO;
  if (!apiKey || !to) return;

  try {
    await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: process.env.WAITLIST_FROM_EMAIL ?? 'Reinstate Waitlist <onboarding@resend.dev>',
        to: [to],
        subject: `New waitlist signup: ${email}`,
        text: `${email} joined the Reinstate waitlist.`,
      }),
    });
  } catch {
    // Swallow — persistence already succeeded.
  }
}

function withHeaders(response: Response, extra: Record<string, string>): Response {
  for (const [name, value] of Object.entries(extra)) {
    if (name.toLowerCase() === 'link' && response.headers.has('Link')) {
      response.headers.set('Link', `${response.headers.get('Link')}, ${value}`);
    } else {
      response.headers.set(name, value);
    }
  }
  return response;
}

export function createWaitlistRoutes(options: { deprecated?: boolean } = {}): {
  GET: APIRoute;
  POST: APIRoute;
  ALL: APIRoute;
} {
  const extra = options.deprecated ? deprecationHeaders() : {};
  const decorate = (response: Response) => withHeaders(response, extra);

  const POST: APIRoute = async ({ request, url }) => {
    let payload: unknown;
    try {
      payload = await request.json();
    } catch {
      return decorate(
        apiError(
          400,
          'invalid_json',
          'Expected JSON body with an email field.',
          'Send Content-Type: application/json with a body like {"email":"you@example.com"}.',
          { instance: url.pathname },
        ),
      );
    }

    const emailRaw =
      payload && typeof payload === 'object' && 'email' in payload
        ? (payload as { email: unknown }).email
        : undefined;

    const validated = validateEmail(emailRaw);
    if (!validated.ok) {
      return decorate(
        apiError(
          400,
          'invalid_email',
          validated.error,
          'Provide one deliverable address in the email field; it is trimmed and lower-cased before storage.',
          { instance: url.pathname },
        ),
      );
    }

    try {
      const result = await storeWaitlistEmail(validated.email, 'web');
      if (result.status === 'created') {
        void maybeNotifyResend(validated.email);
      }
      return decorate(
        apiJson({
          ok: true,
          status: result.status,
          email: result.email,
          message:
            result.status === 'created'
              ? 'You are on the list. We will write when early access opens.'
              : 'You are already on the waitlist.',
        }),
      );
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Could not save email.';
      console.error('[waitlist]', message);
      return decorate(
        apiError(
          503,
          'storage_unavailable',
          'Waitlist storage is temporarily unavailable. Try again or star the GitHub repo.',
          'Retry after the Retry-After delay; the request is idempotent, so resubmitting the same address is safe.',
          { instance: url.pathname, headers: { 'Retry-After': '60' } },
        ),
      );
    }
  };

  const GET: APIRoute = async () =>
    decorate(
      apiJson({
        ok: true,
        service: 'reinstate-waitlist',
        version: 1,
        accepts: 'POST { "email": "you@example.com" }',
        ...(options.deprecated ? { deprecated: true, successor: siteUrl(WAITLIST_V1_PATH) } : {}),
      }),
    );

  /** Any other method gets a JSON 405 with an Allow header instead of the HTML 404 shell. */
  const ALL: APIRoute = ({ request, url }) => decorate(methodNotAllowed(request.method, WAITLIST_METHODS, url.pathname));

  return { GET, POST, ALL };
}
