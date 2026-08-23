import { product, siteUrl } from '../../data/product';
import { MARKDOWN_CONTENT_TYPE } from './accept';

/** Recovery links every agent-facing 404 points at, in priority order. */
export const NOT_FOUND_LINKS = [
  { label: 'Documentation index', path: '/docs' },
  { label: 'llms.txt (curated page index)', path: '/llms.txt' },
  { label: 'llms-full.txt (all documentation as one Markdown file)', path: '/llms-full.txt' },
  { label: 'Sitemap', path: '/sitemap-index.xml' },
  { label: 'OpenAPI description of the HTTP surface', path: '/openapi.json' },
  { label: 'Developer resources', path: '/developers' },
  { label: 'Agent instructions', path: '/agent-instructions.md' },
] as const;

function safeDisplayPath(path: string | null): string {
  if (!path) return '(unknown path)';
  return path.replace(/[^\x20-\x7e]/g, '').replace(/[`<>]/g, '').slice(0, 200);
}

/** Short Markdown body for a 404 so an agent can recover without parsing HTML. */
export function notFoundMarkdown(path: string | null): string {
  const display = safeDisplayPath(path);
  return [
    `# 404: no page at ${display}`,
    '',
    `There is no ${product.name} page at \`${display}\` on ${product.siteUrl}. The address may have moved or may be incomplete.`,
    '',
    '## Where to look next',
    '',
    ...NOT_FOUND_LINKS.map((link) => `- [${link.label}](${siteUrl(link.path)})`),
    `- [Source repository](${product.repositoryUrl})`,
    '',
    `Every HTML page also has a Markdown twin at \`<page>.md\` and answers \`Accept: text/markdown\` at its canonical URL.`,
    '',
  ].join('\n');
}

export function notFoundMarkdownResponse(path: string | null): Response {
  return new Response(notFoundMarkdown(path), {
    status: 404,
    headers: {
      'Content-Type': MARKDOWN_CONTENT_TYPE,
      'Cache-Control': 'public, max-age=0, must-revalidate',
      'X-Content-Type-Options': 'nosniff',
      Vary: 'Accept',
    },
  });
}
