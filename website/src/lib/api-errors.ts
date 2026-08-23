/**
 * Structured JSON errors for every `/api/*` response, in the RFC 9457
 * `application/problem+json` shape (`type`, `title`, `status`, `detail`,
 * `instance`) plus the stable extension members agents and the waitlist form
 * rely on: `code`, `hint`, `docs`, and the original `ok`/`error` pair.
 *
 * Every API response also carries the RFC 8288 `Link` relations that point
 * at the OpenAPI description, the RFC 9727 API catalog, the developer
 * documentation, and the versioning/deprecation policy.
 */
import { siteUrl } from '../data/product';

export const API_ERROR_CODES = [
  'invalid_json',
  'invalid_email',
  'storage_unavailable',
  'not_found',
  'method_not_allowed',
  'rate_limited',
] as const;

export type ApiErrorCode = (typeof API_ERROR_CODES)[number];

export const API_ERROR_TITLES: Record<ApiErrorCode, string> = {
  invalid_json: 'Request body is not JSON',
  invalid_email: 'Email address is invalid',
  storage_unavailable: 'Waitlist storage is unavailable',
  not_found: 'API route not found',
  method_not_allowed: 'Method not allowed',
  rate_limited: 'Too many requests',
};

export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance?: string;
  code: ApiErrorCode;
  hint: string;
  docs: string;
  /** Legacy members kept for the waitlist form and early integrations. */
  ok: false;
  error: string;
}

export const PROBLEM_CONTENT_TYPE = 'application/problem+json; charset=utf-8';
export const JSON_CONTENT_TYPE = 'application/json; charset=utf-8';

export const API_DOCS_URL = siteUrl('/developers');
export const API_ERRORS_URL = siteUrl('/developers#errors');
export const API_VERSIONING_URL = siteUrl('/developers#versioning-and-deprecation');
export const API_RATE_LIMITS_URL = siteUrl('/developers#rate-limits');
export const OPENAPI_URL = siteUrl('/openapi.json');
export const API_CATALOG_URL = siteUrl('/.well-known/api-catalog');

/** RFC 8288 `Link` value advertised on every API response. */
export const API_LINK_HEADER = [
  `<${OPENAPI_URL}>; rel="service-desc"; type="application/vnd.oai.openapi+json"`,
  `<${API_CATALOG_URL}>; rel="api-catalog"`,
  `<${API_DOCS_URL}>; rel="service-doc"; type="text/html"`,
  `<${API_VERSIONING_URL}>; rel="deprecation"; type="text/html"`,
].join(', ');

export function problemType(code: ApiErrorCode): string {
  return siteUrl(`/developers#error-${code.replace(/_/g, '-')}`);
}

export function problemDetails(
  status: number,
  code: ApiErrorCode,
  detail: string,
  hint: string,
  options: { instance?: string; docs?: string } = {},
): ProblemDetails {
  return {
    type: problemType(code),
    title: API_ERROR_TITLES[code],
    status,
    detail,
    ...(options.instance ? { instance: options.instance } : {}),
    code,
    hint,
    docs: options.docs ?? API_ERRORS_URL,
    ok: false,
    error: detail,
  };
}

function baseHeaders(contentType: string, cacheControl: string): Record<string, string> {
  return {
    'Content-Type': contentType,
    'Cache-Control': cacheControl,
    'X-Content-Type-Options': 'nosniff',
    Link: API_LINK_HEADER,
  };
}

export function apiError(
  status: number,
  code: ApiErrorCode,
  detail: string,
  hint: string,
  options: { instance?: string; docs?: string; headers?: Record<string, string> } = {},
): Response {
  return new Response(JSON.stringify(problemDetails(status, code, detail, hint, options)), {
    status,
    headers: { ...baseHeaders(PROBLEM_CONTENT_TYPE, 'no-store'), ...(options.headers ?? {}) },
  });
}

export function apiJson(body: unknown, status = 200, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { ...baseHeaders(JSON_CONTENT_TYPE, 'no-store'), ...headers },
  });
}

export function methodNotAllowed(method: string, allowed: readonly string[], path: string): Response {
  const allow = allowed.join(', ');
  return apiError(
    405,
    'method_not_allowed',
    `${method} is not supported at ${path}.`,
    `Use one of: ${allow}. The Allow header lists the same methods.`,
    { instance: path, headers: { Allow: allow } },
  );
}

export function apiNotFound(path: string): Response {
  return apiError(
    404,
    'not_found',
    `No API route exists at ${path}.`,
    `The documented routes are listed in ${OPENAPI_URL}; the waitlist lives at /api/v1/waitlist.`,
    { instance: path },
  );
}
