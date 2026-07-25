/**
 * Pure email normalization + validation for waitlist signup.
 * Kept dependency-free so unit tests can import without Astro.
 */

const EMAIL_RE =
  /^[a-z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/i;

export type EmailValidation =
  | { ok: true; email: string }
  | { ok: false; error: string };

export function normalizeEmail(raw: unknown): string {
  if (typeof raw !== 'string') return '';
  return raw.trim().toLowerCase();
}

export function validateEmail(raw: unknown): EmailValidation {
  const email = normalizeEmail(raw);
  if (!email) {
    return { ok: false, error: 'Email is required.' };
  }
  if (email.length > 254) {
    return { ok: false, error: 'Email is too long.' };
  }
  if (!EMAIL_RE.test(email)) {
    return { ok: false, error: 'Enter a valid email address.' };
  }
  return { ok: true, email };
}
