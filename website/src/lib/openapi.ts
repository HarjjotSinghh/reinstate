/**
 * OpenAPI 3.1 description of every machine-readable endpoint reinstate.dev
 * serves. It documents the website's HTTP surface only; the Reinstate CLI has
 * no hosted API and no account, and nothing here invents one.
 *
 * Invariants (enforced by `openapi.test.ts`): every operation has a unique
 * `operationId`, a `summary`, a `description`, typed parameters, and at least
 * one response with a schema, so the document works as a function-calling
 * manifest as well as human documentation.
 */
import { product, siteUrl } from '../data/product';
import { API_ERROR_CODES } from './api-errors';

export const OPENAPI_VERSION = '3.1.0';
export const OPENAPI_PATH = '/openapi.json';

type Schema = Record<string, unknown>;

const ref = (name: string): Schema => ({ $ref: `#/components/schemas/${name}` });

const errorResponse = (description: string): Schema => ({
  description,
  content: { 'application/json': { schema: ref('ErrorResponse') } },
});

const textResponse = (mediaType: string, description: string, schemaName = 'TextDocument'): Schema => ({
  description,
  content: { [mediaType]: { schema: ref(schemaName) } },
});

const varyAcceptHeader: Schema = {
  description: 'Always `Accept`, so caches key the representation on the request header.',
  schema: { type: 'string', const: 'Accept' },
};

export function openApiDocument(): Record<string, unknown> {
  return {
    openapi: OPENAPI_VERSION,
    info: {
      title: `${product.name} website API`,
      version: product.currentRelease.replace(/^v/, ''),
      summary: `Machine-readable endpoints of ${product.siteUrl}: waitlist, compatibility data, Markdown page representations, and agent index files.`,
      description: [
        `${product.name} is ${product.shortDefinition.charAt(0).toLowerCase()}${product.shortDefinition.slice(1)}`,
        'It ships as a local CLI (`rein`); there is no hosted Reinstate API, no account, and no credential sync.',
        'This document describes the public HTTP surface of the website so agents can discover the compatibility matrix, join the waitlist, and read any page as Markdown.',
        'No authentication is required and no rate limit is published; be a polite client and cache responses as the headers allow.',
      ].join(' '),
      contact: {
        name: product.maintainer.name,
        url: product.maintainer.url,
      },
      license: {
        name: product.licenseName,
        identifier: product.licenseName,
      },
    },
    externalDocs: {
      description: `${product.name} developer resources`,
      url: siteUrl('/developers'),
    },
    servers: [{ url: product.siteUrl, description: 'Production' }],
    tags: [
      { name: 'waitlist', description: 'Early-access waitlist.' },
      { name: 'compatibility', description: 'Current agent, platform, and version support data.' },
      { name: 'content', description: 'Markdown and text representations of website pages.' },
      { name: 'discovery', description: 'Index files agents read first.' },
      { name: 'install', description: 'Installer bootstrap scripts for the CLI.' },
    ],
    paths: {
      '/api/waitlist': {
        get: {
          operationId: 'getWaitlistService',
          tags: ['waitlist'],
          summary: 'Describe the waitlist endpoint',
          description: 'Returns the service name and the request shape the POST handler accepts. Useful as a liveness probe.',
          responses: {
            '200': {
              description: 'Service descriptor.',
              content: { 'application/json': { schema: ref('WaitlistService') } },
            },
          },
        },
        post: {
          operationId: 'joinWaitlist',
          tags: ['waitlist'],
          summary: 'Join the early-access waitlist',
          description:
            'Stores one email address. Duplicate submissions are idempotent and return `status: "duplicate"`. Only call this when a person has asked to be added.',
          requestBody: {
            required: true,
            content: { 'application/json': { schema: ref('WaitlistRequest') } },
          },
          responses: {
            '200': {
              description: 'The address was stored or was already present.',
              content: { 'application/json': { schema: ref('WaitlistResponse') } },
            },
            '400': errorResponse('The body was not JSON or the email was invalid (`code` is `invalid_json` or `invalid_email`).'),
            '405': errorResponse('Method not supported; the `Allow` header lists GET and POST.'),
            '503': errorResponse('Waitlist storage is temporarily unavailable (`code` is `storage_unavailable`).'),
          },
        },
      },
      '/api/{path}': {
        get: {
          operationId: 'getUndocumentedApiRoute',
          tags: ['waitlist'],
          summary: 'Any other /api path',
          description: 'Every undocumented `/api/*` path returns a JSON 404 with `code: "not_found"` and a pointer back to this document, never an HTML page.',
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
            '404': errorResponse('No API route at that path.'),
          },
        },
      },
      '/compatibility.json': {
        get: {
          operationId: 'getCompatibilityMatrix',
          tags: ['compatibility'],
          summary: 'Current compatibility matrix',
          description:
            'Machine-readable agent tiers, tested version ranges, operating-system environments, evidence links, and review dates. The same data renders the /compatibility page.',
          responses: {
            '200': {
              description: 'Compatibility matrix.',
              content: { 'application/json': { schema: ref('CompatibilityMatrix') } },
            },
          },
        },
      },
      '/{page}': {
        get: {
          operationId: 'getPage',
          tags: ['content'],
          summary: 'Read any page as HTML or Markdown',
          description:
            'Every page negotiates on the `Accept` header: `text/markdown` returns the Markdown representation, `text/html` (or no preference) returns the page. Responses carry `Vary: Accept`. Unknown pages return 404 with a short Markdown body; an `Accept` header that rejects both representations returns 406.',
          parameters: [
            {
              name: 'page',
              in: 'path',
              required: true,
              description: 'Clean page path without extension, for example `docs/getting-started`. Use an empty value for the homepage.',
              schema: { type: 'string', pattern: '^[A-Za-z0-9_/-]*$' },
            },
            {
              name: 'Accept',
              in: 'header',
              required: false,
              description: 'Media ranges with optional q-values, for example `text/markdown, text/html;q=0.8`.',
              schema: { type: 'string', default: 'text/html' },
            },
          ],
          responses: {
            '200': {
              description: 'The negotiated representation.',
              headers: { Vary: varyAcceptHeader },
              content: {
                'text/markdown': { schema: ref('MarkdownDocument') },
                'text/html': { schema: { type: 'string' } },
              },
            },
            '404': textResponse('text/markdown', 'No page at that path; the body links to the sitemap, llms.txt, and docs index.', 'MarkdownDocument'),
            '406': textResponse('text/plain', 'Neither text/html nor text/markdown was acceptable; the body lists both.'),
          },
        },
      },
      '/{page}.md': {
        get: {
          operationId: 'getPageMarkdown',
          tags: ['content'],
          summary: 'Markdown twin of a page',
          description:
            'Static Markdown representation of the page at the same path without the extension. The homepage twin is `/index.md`. Generated at build time from the same content as the HTML.',
          parameters: [
            {
              name: 'page',
              in: 'path',
              required: true,
              description: 'Clean page path, for example `docs/faq` or `index` for the homepage.',
              schema: { type: 'string', pattern: '^[A-Za-z0-9_/-]+$' },
            },
          ],
          responses: {
            '200': textResponse('text/markdown', 'Markdown document.', 'MarkdownDocument'),
            '404': { description: 'No page at that path.' },
          },
        },
      },
      '/llms.txt': {
        get: {
          operationId: 'getLlmsIndex',
          tags: ['discovery'],
          summary: 'Curated page index for agents (llms.txt)',
          description: 'llmstxt.org-format index: what Reinstate is, when to use it, and the canonical pages grouped by purpose.',
          responses: { '200': textResponse('text/plain', 'llms.txt document.') },
        },
      },
      '/llms-full.txt': {
        get: {
          operationId: 'getLlmsFullText',
          tags: ['discovery'],
          summary: 'All documentation as one Markdown file',
          description: 'Concatenated Markdown of every indexable page, each section prefixed with its canonical URL. Built from the same content as the HTML pages.',
          responses: { '200': textResponse('text/plain', 'Concatenated Markdown.') },
        },
      },
      '/agent-instructions.md': {
        get: {
          operationId: 'getAgentInstructions',
          tags: ['discovery'],
          summary: 'When and how an agent should use Reinstate',
          description: 'Best-fit jobs, jobs Reinstate is wrong for, the CLI commands to run, and the safety rules that apply.',
          responses: { '200': textResponse('text/markdown', 'Agent instructions.', 'MarkdownDocument') },
        },
      },
      '/openapi.json': {
        get: {
          operationId: 'getOpenApiDocument',
          tags: ['discovery'],
          summary: 'This document',
          description: 'OpenAPI 3.1 description of the website HTTP surface.',
          responses: {
            '200': {
              description: 'OpenAPI document.',
              content: { 'application/json': { schema: { type: 'object', additionalProperties: true } } },
            },
          },
        },
      },
      '/sitemap-index.xml': {
        get: {
          operationId: 'getSitemapIndex',
          tags: ['discovery'],
          summary: 'Sitemap index',
          description: 'XML sitemap index listing every indexable page.',
          responses: { '200': textResponse('application/xml', 'Sitemap index.') },
        },
      },
      '/rss.xml': {
        get: {
          operationId: 'getAllUpdatesFeed',
          tags: ['discovery'],
          summary: 'All-updates RSS feed',
          description: 'RSS feed combining blog posts and changelog entries. Blog-only and changelog-only feeds live at /blog/rss.xml and /changelog/rss.xml.',
          responses: { '200': textResponse('application/rss+xml', 'RSS feed.') },
        },
      },
      '/install.sh': {
        get: {
          operationId: 'getUnixInstaller',
          tags: ['install'],
          summary: 'macOS, Linux, and WSL2 installer',
          description: `POSIX shell bootstrap that installs the pinned ${product.name} CLI release. Review it before piping it to a shell.`,
          responses: { '200': textResponse('text/plain', 'Shell script.') },
        },
      },
      '/install.ps1': {
        get: {
          operationId: 'getWindowsInstaller',
          tags: ['install'],
          summary: 'Windows PowerShell installer',
          description: `PowerShell bootstrap that installs the pinned ${product.name} CLI release on native Windows.`,
          responses: { '200': textResponse('text/plain', 'PowerShell script.') },
        },
      },
    },
    components: {
      schemas: {
        ErrorResponse: {
          type: 'object',
          description: 'Structured error returned by every /api route.',
          required: ['ok', 'status', 'code', 'error', 'hint', 'docs'],
          properties: {
            ok: { type: 'boolean', const: false },
            status: { type: 'integer', description: 'HTTP status code, repeated in the body.' },
            code: { type: 'string', enum: [...API_ERROR_CODES], description: 'Stable machine-readable error code.' },
            error: { type: 'string', description: 'Human-readable message.' },
            hint: { type: 'string', description: 'What to change before retrying.' },
            docs: { type: 'string', format: 'uri', description: 'Where the route is documented.' },
          },
        },
        WaitlistService: {
          type: 'object',
          required: ['ok', 'service', 'accepts'],
          properties: {
            ok: { type: 'boolean', const: true },
            service: { type: 'string', const: 'reinstate-waitlist' },
            accepts: { type: 'string', description: 'Human-readable description of the POST body.' },
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
        MarkdownDocument: {
          type: 'string',
          description: 'UTF-8 Markdown. Starts with a level-one heading and ends with a footer naming the canonical URL.',
        },
        TextDocument: {
          type: 'string',
          description: 'UTF-8 text.',
        },
      },
    },
  };
}

export function openApiJson(): string {
  return `${JSON.stringify(openApiDocument(), null, 2)}\n`;
}
