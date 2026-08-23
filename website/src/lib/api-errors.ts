/**
 * Structured JSON errors for every `/api/*` response so agents get a code,
 * a message, and a resolution hint instead of an HTML error page.
 *
 * The `ok`/`error` pair is the original contract the waitlist form reads;
 * `code`, `hint`, `docs`, and `status` are additive.
 */
import { siteUrl } from '../data/product';

export const API_ERROR_CODES = [
  'invalid_json',
  'invalid_email',
  'storage_unavailable',
  'not_found',
  'method_not_allowed',
] as const;

export type ApiErrorCode = (typeof API_ERROR_CODES)[number];

export interface ApiErrorBody {
  ok: false;
  status: number;
  code: ApiErrorCode;
  error: string;
  hint: string;
  docs: string;
}

export const API_DOCS_URL = siteUrl('/openapi.json');

const JSON_HEADERS = {
  'Content-Type': 'application/json; charset=utf-8',
  'Cache-Control': 'no-store',
  'X-Content-Type-Options': 'nosniff',
} as const;

export function apiErrorBody(
  status: number,
  code: ApiErrorCode,
  error: string,
  hint: string,
  docs: string = API_DOCS_URL,
): ApiErrorBody {
  return { ok: false, status, code, error, hint, docs };
}

export function apiError(
  status: number,
  code: ApiErrorCode,
  error: string,
  hint: string,
  options: { docs?: string; headers?: Record<string, string> } = {},
): Response {
  return new Response(JSON.stringify(apiErrorBody(status, code, error, hint, options.docs)), {
    status,
    headers: { ...JSON_HEADERS, ...(options.headers ?? {}) },
  });
}

export function apiJson(body: unknown, status = 200, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { ...JSON_HEADERS, ...headers },
  });
}

export function methodNotAllowed(method: string, allowed: readonly string[], path: string): Response {
  const allow = allowed.join(', ');
  return apiError(
    405,
    'method_not_allowed',
    `${method} is not supported at ${path}.`,
    `Use one of: ${allow}. The Allow header lists the same methods.`,
    { headers: { Allow: allow } },
  );
}

export function apiNotFound(path: string): Response {
  return apiError(
    404,
    'not_found',
    `No API route exists at ${path}.`,
    `The only documented API routes are listed in ${API_DOCS_URL}; the waitlist lives at /api/waitlist.`,
  );
}
