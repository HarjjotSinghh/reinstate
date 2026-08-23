import type { APIRoute } from 'astro';
import { validateEmail } from '../../lib/email';
import { storeWaitlistEmail } from '../../lib/waitlist-store';
import { apiError, apiJson, methodNotAllowed } from '../../lib/api-errors';

export const prerender = false;

export const WAITLIST_METHODS = ['GET', 'POST'] as const;

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

export const POST: APIRoute = async ({ request }) => {
  let payload: unknown;
  try {
    payload = await request.json();
  } catch {
    return apiError(
      400,
      'invalid_json',
      'Expected JSON body with an email field.',
      'Send Content-Type: application/json with a body like {"email":"you@example.com"}.',
    );
  }

  const emailRaw =
    payload && typeof payload === 'object' && 'email' in payload
      ? (payload as { email: unknown }).email
      : undefined;

  const validated = validateEmail(emailRaw);
  if (!validated.ok) {
    return apiError(
      400,
      'invalid_email',
      validated.error,
      'Provide one deliverable address in the email field; it is trimmed and lower-cased before storage.',
    );
  }

  try {
    const result = await storeWaitlistEmail(validated.email, 'web');
    if (result.status === 'created') {
      void maybeNotifyResend(validated.email);
    }
    return apiJson({
      ok: true,
      status: result.status,
      email: result.email,
      message:
        result.status === 'created'
          ? 'You are on the list. We will write when early access opens.'
          : 'You are already on the waitlist.',
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Could not save email.';
    console.error('[waitlist]', message);
    return apiError(
      503,
      'storage_unavailable',
      'Waitlist storage is temporarily unavailable. Try again or star the GitHub repo.',
      'Retry after the Retry-After delay; the request is idempotent, so resubmitting the same address is safe.',
      { headers: { 'Retry-After': '60' } },
    );
  }
};

export const GET: APIRoute = async () =>
  apiJson({
    ok: true,
    service: 'reinstate-waitlist',
    accepts: 'POST { "email": "you@example.com" }',
  });

/** Any other method gets a JSON 405 with an Allow header instead of the HTML 404 shell. */
export const ALL: APIRoute = ({ request, url }) =>
  methodNotAllowed(request.method, WAITLIST_METHODS, url.pathname);
