/**
 * OpenAPI 3.1 description of the JSON HTTP API reinstate.dev serves. It
 * documents the website's API only; the Reinstate CLI has no hosted API and
 * no account, and nothing here invents one. Text surfaces (Markdown
 * negotiation, `llms.txt`, feeds, installers) are described at /developers
 * and in the RFC 9727 API catalog rather than as API operations.
 *
 * Invariants (enforced by `openapi.test.ts`): every operation has a unique
 * `operationId`, a `summary`, a `description`, typed parameters, and typed
 * JSON response schemas for every status, so the document works as a
 * function-calling manifest as well as human documentation.
 */
import { product, siteUrl } from '../data/product';
import { API_ERROR_CODES, API_ERRORS_URL, API_RATE_LIMITS_URL, API_VERSIONING_URL } from './api-errors';
import { API_RATE_LIMIT } from './rate-limit';
import { WAITLIST_LEGACY_DEPRECATED_AT, WAITLIST_LEGACY_PATH, WAITLIST_V1_PATH } from './waitlist-api';

export const OPENAPI_VERSION = '3.1.0';
export const OPENAPI_PATH = '/openapi.json';
/** Major version of the REST surface; the product release is recorded separately. */
export const API_VERSION = '1.0.0';

type Schema = Record<string, unknown>;

const ref = (name: string): Schema => ({ $ref: `#/components/schemas/${name}` });
const headerRef = (name: string): Schema => ({ $ref: `#/components/headers/${name}` });

const RATE_LIMIT_HEADERS: Record<string, Schema> = {
  'RateLimit-Policy': headerRef('RateLimitPolicy'),
  RateLimit: headerRef('RateLimit'),
  'RateLimit-Limit': headerRef('RateLimitLimit'),
  'RateLimit-Remaining': headerRef('RateLimitRemaining'),
  'RateLimit-Reset': headerRef('RateLimitReset'),
  Link: headerRef('Link'),
};

const DEPRECATION_HEADERS: Record<string, Schema> = {
  Deprecation: headerRef('Deprecation'),
};

function jsonResponse(description: string, schemaName: string, extraHeaders: Record<string, Schema> = {}): Schema {
  return {
    description,
    headers: { ...RATE_LIMIT_HEADERS, ...extraHeaders },
    content: { 'application/json': { schema: ref(schemaName) } },
  };
}

function problemResponse(description: string, extraHeaders: Record<string, Schema> = {}): Schema {
  return {
    description,
    headers: { ...RATE_LIMIT_HEADERS, ...extraHeaders },
    content: { 'application/problem+json': { schema: ref('ProblemDetails') } },
  };
}

const rateLimitedResponse = (extraHeaders: Record<string, Schema> = {}): Schema =>
  problemResponse(`The client exceeded ${API_RATE_LIMIT.quota} requests per ${API_RATE_LIMIT.windowSeconds} seconds (code \`rate_limited\`). Wait for Retry-After.`, {
    'Retry-After': headerRef('RetryAfter'),
    ...extraHeaders,
  });

function waitlistOperations(options: { deprecated: boolean }): Record<string, Schema> {
  const suffix = options.deprecated ? 'Legacy' : '';
  const extra = options.deprecated ? DEPRECATION_HEADERS : {};
  const note = options.deprecated
    ? ' Deprecated alias of `/api/v1/waitlist`: it keeps working, advertises `Deprecation` and `Link rel="successor-version"`, and has no scheduled Sunset.'
    : '';
  return {
    get: {
      operationId: `getWaitlistService${suffix}`,
      tags: ['waitlist'],
      summary: options.deprecated ? 'Describe the waitlist endpoint (deprecated path)' : 'Describe the waitlist endpoint',
      description: `Returns the service name, API major version, and the request shape the POST handler accepts. Useful as a liveness probe.${note}`,
      ...(options.deprecated ? { deprecated: true } : {}),
      responses: {
        '200': jsonResponse('Service descriptor.', 'WaitlistService', extra),
        '429': rateLimitedResponse(extra),
      },
    },
    post: {
      operationId: `joinWaitlist${suffix}`,
      tags: ['waitlist'],
      summary: options.deprecated ? 'Join the early-access waitlist (deprecated path)' : 'Join the early-access waitlist',
      description: `Stores one email address. Duplicate submissions are idempotent and return \`status: "duplicate"\`. Only call this when a person has asked to be added.${note}`,
      ...(options.deprecated ? { deprecated: true } : {}),
      requestBody: {
        required: true,
        content: { 'application/json': { schema: ref('WaitlistRequest') } },
      },
      responses: {
        '200': jsonResponse('The address was stored or was already present.', 'WaitlistResponse', extra),
        '400': problemResponse('The body was not JSON or the email was invalid (`code` is `invalid_json` or `invalid_email`).', extra),
        '405': problemResponse('Method not supported; the `Allow` header lists GET and POST.', { Allow: headerRef('Allow'), ...extra }),
        '429': rateLimitedResponse(extra),
        '503': problemResponse('Waitlist storage is temporarily unavailable (`code` is `storage_unavailable`); `Retry-After` is set.', {
          'Retry-After': headerRef('RetryAfter'),
          ...extra,
        }),
      },
    },
  };
}

export function openApiDocument(): Record<string, unknown> {
  const deprecatedSince = WAITLIST_LEGACY_DEPRECATED_AT.slice(0, 10);
  return {
    openapi: OPENAPI_VERSION,
    info: {
      title: `${product.name} website API`,
      version: API_VERSION,
      summary: `JSON endpoints of ${product.siteUrl}: early-access waitlist, compatibility matrix, and this description. Versioned under /api/v1 with RFC 9745 deprecation signalling and IETF RateLimit headers.`,
      description: [
        `${product.name} is ${product.shortDefinition.charAt(0).toLowerCase()}${product.shortDefinition.slice(1)}`,
        'It ships as a local CLI (`rein`); there is no hosted Reinstate API, no account, no API key, no webhooks, and no credential sync.',
        'This document describes the JSON HTTP surface of the website. Stable operations live under `/api/v1`; breaking changes require a new major path, and deprecated paths keep working while advertising RFC 9745 `Deprecation` (and, once scheduled, RFC 8594 `Sunset`) headers at least 90 days before removal.',
        `Every response carries IETF \`RateLimit-Policy\` and \`RateLimit\` structured fields plus the compatibility \`RateLimit-Limit\`/\`RateLimit-Remaining\`/\`RateLimit-Reset\` form; the quota is ${API_RATE_LIMIT.quota} requests per client address per ${API_RATE_LIMIT.windowSeconds} seconds and 429 responses include \`Retry-After\`.`,
        'Errors use RFC 9457 `application/problem+json` with a stable `code`, a `hint`, and a `docs` link. No authentication is required.',
        `Every HTML page also answers \`Accept: text/markdown\` and has a \`.md\` twin; the curated index is ${siteUrl('/llms.txt')} and the full catalog of machine-readable surfaces is ${siteUrl('/.well-known/api-catalog')}.`,
      ].join(' '),
      contact: {
        name: product.maintainer.name,
        url: siteUrl('/contact'),
      },
      license: {
        name: product.licenseName,
        identifier: product.licenseName,
      },
      'x-reinstate-release': product.currentRelease,
    },
    externalDocs: {
      description: `${product.name} developer resources`,
      url: siteUrl('/developers'),
    },
    servers: [{ url: product.siteUrl, description: 'Production' }],
    tags: [
      { name: 'waitlist', description: 'Early-access waitlist.' },
      { name: 'compatibility', description: 'Current agent, platform, and version support data.' },
      { name: 'discovery', description: 'Machine-readable descriptions of this API.' },
    ],
    'x-api-lifecycle': {
      versioning: 'Major version in the URL path (`/api/v1/`). Breaking changes require a new major path; additive changes ship in place.',
      deprecationPolicy: API_VERSIONING_URL,
      deprecationSignals: ['Deprecation (RFC 9745)', 'Link rel="successor-version"', 'Sunset (RFC 8594) once a removal date exists'],
      minimumNoticeDays: 90,
      deprecated: [{ path: WAITLIST_LEGACY_PATH, since: deprecatedSince, successor: WAITLIST_V1_PATH, sunset: null }],
    },
    'x-rate-limit-policy': {
      documentation: API_RATE_LIMITS_URL,
      name: API_RATE_LIMIT.name,
      quota: API_RATE_LIMIT.quota,
      windowSeconds: API_RATE_LIMIT.windowSeconds,
      scope: 'client address, per function instance',
      headers: ['RateLimit-Policy', 'RateLimit', 'RateLimit-Limit', 'RateLimit-Remaining', 'RateLimit-Reset', 'Retry-After'],
    },
    paths: {
      [WAITLIST_V1_PATH]: waitlistOperations({ deprecated: false }),
      [WAITLIST_LEGACY_PATH]: waitlistOperations({ deprecated: true }),
      '/api/{path}': {
        get: {
          operationId: 'getUndocumentedApiRoute',
          tags: ['waitlist'],
          summary: 'Any other /api path',
          description: 'Every undocumented `/api/*` path, including unknown `/api/v1/*` paths, returns an RFC 9457 problem with `code: "not_found"` and a pointer back to this document, never an HTML page.',
          parameters: [
            {
              name: 'path',
              in: 'path',
              required: true,
              description: 'Any path segment(s) under /api that are not documented here.',
              schema: { type: 'string' },
            },
          ],
          responses: {
            '404': problemResponse('No API route at that path.'),
            '429': rateLimitedResponse(),
          },
        },
      },
      '/compatibility.json': {
        get: {
          operationId: 'getCompatibilityMatrix',
          tags: ['compatibility'],
          summary: 'Current compatibility matrix',
          description:
            'Machine-readable agent tiers, tested version ranges, operating-system environments, evidence links, and review dates. The same data renders the /compatibility page. Served as a static file, so rate-limit headers do not apply.',
          responses: {
            '200': {
              description: 'Compatibility matrix.',
              content: { 'application/json': { schema: ref('CompatibilityMatrix') } },
            },
          },
        },
      },
      [OPENAPI_PATH]: {
        get: {
          operationId: 'getOpenApiDocument',
          tags: ['discovery'],
          summary: 'This document',
          description: 'OpenAPI 3.1 description of the website JSON API. Served as a static file.',
          responses: {
            '200': {
              description: 'OpenAPI document.',
              content: { 'application/json': { schema: ref('OpenApiDocument') } },
            },
          },
        },
      },
      '/.well-known/api-catalog': {
        get: {
          operationId: 'getApiCatalog',
          tags: ['discovery'],
          summary: 'RFC 9727 API catalog',
          description: 'Linkset (RFC 9264) that points at this OpenAPI description, the developer documentation, and the other machine-readable surfaces of reinstate.dev. Served as a static file.',
          responses: {
            '200': {
              description: 'API catalog.',
              content: { 'application/linkset+json': { schema: ref('ApiCatalog') } },
            },
          },
        },
      },
    },
    components: {
      headers: {
        RateLimitPolicy: { description: 'Quota policy in the current IETF RateLimit structured-field format, e.g. `"api";q=60;w=60`.', schema: { type: 'string' } },
        RateLimit: { description: 'Remaining quota and seconds until reset in the current IETF RateLimit structured-field format, e.g. `"api";r=59;t=30`.', schema: { type: 'string' } },
        RateLimitLimit: { description: 'Compatibility form of the request quota.', schema: { type: 'integer' } },
        RateLimitRemaining: { description: 'Compatibility form of the remaining request quota.', schema: { type: 'integer' } },
        RateLimitReset: { description: 'Seconds until the current quota window resets.', schema: { type: 'integer' } },
        RetryAfter: { description: 'Seconds to wait before retrying.', schema: { type: 'integer' } },
        Allow: { description: 'Methods the path supports.', schema: { type: 'string' } },
        Link: { description: 'RFC 8288 links: `service-desc` (this document), `api-catalog`, `service-doc`, and `deprecation` (the versioning policy).', schema: { type: 'string' } },
        Deprecation: { description: `RFC 9745 deprecation date in \`@seconds\` form (${deprecatedSince}). Present only on deprecated paths.`, schema: { type: 'string' } },
      },
      schemas: {
        ProblemDetails: {
          type: 'object',
          description: 'RFC 9457 problem details with stable extension members.',
          required: ['type', 'title', 'status', 'detail', 'code', 'hint', 'docs', 'ok', 'error'],
          properties: {
            type: { type: 'string', format: 'uri', description: 'URI identifying the problem type; resolves to the error section of /developers.' },
            title: { type: 'string', description: 'Short human-readable summary of the problem type.' },
            status: { type: 'integer', description: 'HTTP status code, repeated in the body.' },
            detail: { type: 'string', description: 'Human-readable explanation specific to this occurrence.' },
            instance: { type: 'string', description: 'Request path the problem occurred on.' },
            code: { type: 'string', enum: [...API_ERROR_CODES], description: 'Stable machine-readable error code.' },
            hint: { type: 'string', description: 'What to change before retrying.' },
            docs: { type: 'string', format: 'uri', description: 'Where the error is documented.' },
            ok: { type: 'boolean', const: false, description: 'Legacy member: always false on errors.' },
            error: { type: 'string', description: 'Legacy member: same text as `detail`.' },
          },
        },
        WaitlistService: {
          type: 'object',
          required: ['ok', 'service', 'version', 'accepts'],
          properties: {
            ok: { type: 'boolean', const: true },
            service: { type: 'string', const: 'reinstate-waitlist' },
            version: { type: 'integer', const: 1, description: 'API major version.' },
            accepts: { type: 'string', description: 'Human-readable description of the POST body.' },
            deprecated: { type: 'boolean', description: 'Present and true only on the deprecated unversioned path.' },
            successor: { type: 'string', format: 'uri', description: 'Canonical versioned URL; present only on the deprecated path.' },
          },
        },
        WaitlistRequest: {
          type: 'object',
          required: ['email'],
          properties: {
            email: { type: 'string', format: 'email', maxLength: 254, description: 'Address to add; trimmed and lower-cased.' },
          },
        },
        WaitlistResponse: {
          type: 'object',
          required: ['ok', 'status', 'email', 'message'],
          properties: {
            ok: { type: 'boolean', const: true },
            status: { type: 'string', enum: ['created', 'duplicate'] },
            email: { type: 'string', format: 'email' },
            message: { type: 'string' },
          },
        },
        CompatibilityMatrix: {
          type: 'object',
          description: 'Reviewed compatibility data; additional keys may appear in later schema versions.',
          required: ['schemaVersion', 'reinstateVersion', 'lastReviewed', 'agents', 'environments'],
          properties: {
            schemaVersion: { type: 'integer', const: 1 },
            reinstateVersion: { type: 'string', description: 'Release the matrix was reviewed against.' },
            catalogLine: { type: 'string' },
            lastReviewed: { type: 'string', format: 'date' },
            owner: {
              type: 'object',
              properties: { name: { type: 'string' }, url: { type: 'string', format: 'uri' } },
            },
            fixtureCommit: { type: 'string', description: 'Git commit of the synthetic fixtures the evidence cites.' },
            agents: { type: 'array', items: ref('CompatibilityAgent') },
            environments: { type: 'array', items: ref('CompatibilityEnvironment') },
          },
          additionalProperties: true,
        },
        CompatibilityAgent: {
          type: 'object',
          required: ['id', 'name', 'vendor', 'tier', 'status', 'integrationPath', 'source'],
          properties: {
            id: { type: 'string' },
            key: { type: 'string', description: 'Value accepted by `rein --agent`.' },
            name: { type: 'string' },
            vendor: { type: 'string' },
            tier: { type: 'string', description: 'Capability tier; T5 is encrypted cross-device sync, lower tiers are read-only or handoff-only.' },
            storageFamily: { type: 'string' },
            t0Reason: { type: ['string', 'null'] },
            integrationPath: { type: 'string', description: 'Site-relative path of the integration page.' },
            status: { type: 'string' },
            minimumTestedVersion: { type: ['string', 'null'] },
            maximumTestedVersion: { type: ['string', 'null'] },
            resumeMode: { type: 'string' },
            evidenceLevel: { type: 'string' },
            evidence: { type: 'object', additionalProperties: true },
            lastTested: { type: ['string', 'null'], format: 'date' },
            source: { type: 'string', format: 'uri' },
            notes: { type: 'string' },
          },
          additionalProperties: true,
        },
        CompatibilityEnvironment: {
          type: 'object',
          required: ['id', 'name', 'status', 'source'],
          properties: {
            id: { type: 'string' },
            name: { type: 'string' },
            claudeCode: { type: 'string' },
            codex: { type: 'string' },
            status: { type: 'string' },
            lastTested: { type: ['string', 'null'], format: 'date' },
            source: { type: 'string', format: 'uri' },
            notes: { type: 'string' },
          },
          additionalProperties: true,
        },
        OpenApiDocument: {
          type: 'object',
          description: 'An OpenAPI 3.1 document.',
          required: ['openapi', 'info', 'paths'],
          properties: {
            openapi: { type: 'string', const: OPENAPI_VERSION },
            info: { type: 'object', required: ['title', 'version'], properties: { title: { type: 'string' }, version: { type: 'string' } }, additionalProperties: true },
            servers: { type: 'array', items: { type: 'object', required: ['url'], properties: { url: { type: 'string', format: 'uri' } } } },
            paths: { type: 'object', additionalProperties: true },
            components: { type: 'object', additionalProperties: true },
          },
          additionalProperties: true,
        },
        ApiCatalog: {
          type: 'object',
          description: 'RFC 9264 linkset as used by RFC 9727 API catalogs.',
          required: ['linkset'],
          properties: {
            linkset: {
              type: 'array',
              items: {
                type: 'object',
                required: ['anchor'],
                properties: {
                  anchor: { type: 'string', format: 'uri', description: 'The API or surface the links describe.' },
                  'service-desc': { type: 'array', items: ref('LinkTarget') },
                  'service-doc': { type: 'array', items: ref('LinkTarget') },
                  'service-meta': { type: 'array', items: ref('LinkTarget') },
                  status: { type: 'array', items: ref('LinkTarget') },
                },
                additionalProperties: true,
              },
            },
          },
        },
        LinkTarget: {
          type: 'object',
          required: ['href'],
          properties: {
            href: { type: 'string', format: 'uri' },
            type: { type: 'string', description: 'Media type of the target.' },
            title: { type: 'string' },
          },
        },
      },
    },
  };
}

export function openApiJson(): string {
  return `${JSON.stringify(openApiDocument(), null, 2)}\n`;
}
