import type { APIRoute } from 'astro';
import { validateEmail } from '../../lib/email';
import { storeWaitlistEmail } from '../../lib/waitlist-store';

export const prerender = false;

const corsHeaders = {
  'Content-Type': 'application/json',
  'Cache-Control': 'no-store',
};

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: corsHeaders,
  });
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

export const POST: APIRoute = async ({ request }) => {
  let payload: unknown;
  try {
    payload = await request.json();
  } catch {
    return json({ ok: false, error: 'Expected JSON body with an email field.' }, 400);
  }

  const emailRaw =
    payload && typeof payload === 'object' && 'email' in payload
      ? (payload as { email: unknown }).email
      : undefined;

  const validated = validateEmail(emailRaw);
  if (!validated.ok) {
    return json({ ok: false, error: validated.error }, 400);
  }

  try {
    const result = await storeWaitlistEmail(validated.email, 'web');
    if (result.status === 'created') {
      void maybeNotifyResend(validated.email);
    }
    return json({
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
    return json(
      {
        ok: false,
        error:
          'Waitlist storage is temporarily unavailable. Try again or star the GitHub repo.',
      },
      503,
    );
  }
};

export const GET: APIRoute = async () => {
  return json({
    ok: true,
    service: 'reinstate-waitlist',
    accepts: 'POST { "email": "you@example.com" }',
  });
};
