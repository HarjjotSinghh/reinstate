/**
 * Build-time (and dev-time) conversion of a rendered reinstate.dev page into
 * the Markdown representation served for `Accept: text/markdown` and at the
 * `/{page}.md` twin URL.
 *
 * The converter only looks at the page's `<main>` element, drops chrome and
 * decoration (navigation rails, inline SVG scenes, scripts, forms), keeps the
 * document outline, fenced code with its language, GFM tables, definition
 * lists, and absolute links so an agent can follow them without a base URL.
 */
import TurndownService from 'turndown';
import { gfm } from 'turndown-plugin-gfm';

export interface HtmlToMarkdownOptions {
  /** Absolute canonical URL of the page; used to resolve relative links and for the footer. */
  url: string;
  /** Overrides the `<title>` fallback when the page has no H1. */
  title?: string;
}

type DomNode = {
  nodeName: string;
  nodeType?: number;
  textContent?: string | null;
  getAttribute?: (name: string) => string | null;
  parentNode?: DomNode | null;
  firstChild?: DomNode | null;
  childNodes?: ArrayLike<DomNode>;
};

/** Inline elements that Astro often renders back-to-back with CSS gaps instead of whitespace. */
const INLINE_ELEMENTS = new Set([
  'a', 'abbr', 'b', 'cite', 'code', 'em', 'i', 'kbd', 'small', 'span', 'strong', 'sub', 'sup', 'time', 'button',
]);

const INLINE_TAGS = [...INLINE_ELEMENTS].join('|');
const GLUED_INLINE = new RegExp(`(</(?:${INLINE_TAGS})>)(<(?:${INLINE_TAGS})\\b)`, 'gi');

/** Inserts a space between adjacent inline elements so `<span>a</span><span>b</span>` does not read as `ab`; code blocks are left untouched. */
export function separateInlineSiblings(html: string): string {
  return html
    .split(/(<pre[\s>][\s\S]*?<\/pre>)/i)
    .map((part, index) => (index % 2 === 1 ? part : part.replace(GLUED_INLINE, '$1 $2')))
    .join('');
}

const REMOVED_ELEMENTS = [
  'script',
  'style',
  'svg',
  'noscript',
  'template',
  'iframe',
  'canvas',
  'video',
  'audio',
  'form',
  'button',
  'input',
  'select',
  'textarea',
  'aside',
] as const;

function decodeEntities(text: string): string {
  return text
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&#x27;/g, "'")
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&');
}

function attribute(node: DomNode, name: string): string {
  return node.getAttribute?.(name) ?? '';
}

function isBlankContent(content: string): boolean {
  return content.replace(/\\?[\s_*]/g, '').length === 0;
}

function resolveHref(href: string, base: string): string | null {
  const trimmed = href.trim();
  if (!trimmed || /^(javascript|data|mailto|tel):/i.test(trimmed)) return null;
  try {
    return new URL(trimmed, base).toString();
  } catch {
    return null;
  }
}

function createService(url: string): TurndownService {
  const service = new TurndownService({
    headingStyle: 'atx',
    codeBlockStyle: 'fenced',
    bulletListMarker: '-',
    emDelimiter: '*',
    strongDelimiter: '**',
    hr: '---',
  });
  service.use(gfm);
  service.remove([...REMOVED_ELEMENTS] as unknown as TurndownService.Filter);

  service.addRule('decorative', {
    filter: (node) =>
      attribute(node as unknown as DomNode, 'aria-hidden') === 'true' ||
      attribute(node as unknown as DomNode, 'data-markdown') === 'skip',
    replacement: (_content, node) => (INLINE_ELEMENTS.has(node.nodeName.toLowerCase()) ? ' ' : ''),
  });

  // A heading broken across lines with <br> must stay on one Markdown line.
  service.addRule('singleLineHeading', {
    filter: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'],
    replacement: (content, node) => {
      const level = Number(node.nodeName.charAt(1));
      const text = content.replace(/\s*\n\s*/g, ' ').trim();
      return text ? `\n\n${'#'.repeat(level)} ${text}\n\n` : '';
    },
  });

  service.addRule('skippedNavigation', {
    filter: (node) => {
      const classes = attribute(node as unknown as DomNode, 'class').split(/\s+/);
      if (classes.includes('mobile-nav')) return true;
      if (node.nodeName !== 'NAV') return false;
      const label = attribute(node as unknown as DomNode, 'aria-label').toLowerCase();
      return label === 'breadcrumb' || label === 'documentation';
    },
    replacement: () => '',
  });

  // Turndown indents list items with three spaces after the marker; one space reads better and is what the docs sources use.
  service.addRule('tightListItem', {
    filter: 'li',
    replacement: (content, node, options) => {
      const body = content
        .replace(/^\s+/, '')
        .replace(/\n+$/, '\n')
        .replace(/\n/gm, '\n  ');
      const parent = (node as unknown as DomNode).parentNode;
      let prefix = `${options.bulletListMarker} `;
      if (parent && parent.nodeName === 'OL') {
        const start = Number(attribute(parent, 'start') || '1');
        const siblings = Array.from(parent.childNodes ?? []).filter((child) => child.nodeName === 'LI');
        const index = siblings.indexOf(node as unknown as DomNode);
        prefix = `${start + index}. `;
      }
      const next = (node as unknown as { nextSibling?: DomNode | null }).nextSibling;
      return `${prefix}${body}${next && !/\n$/.test(body) ? '\n' : ''}`;
    },
  });

  service.addRule('codeBlock', {
    filter: (node) => node.nodeName === 'PRE',
    replacement: (_content, node) => {
      const pre = node as unknown as DomNode;
      const language =
        attribute(pre, 'data-language') ||
        (attribute(pre, 'class').match(/language-([\w-]+)/)?.[1] ?? '');
      const text = (pre.textContent ?? '').replace(/\n$/, '');
      const fence = text.includes('```') ? '````' : '```';
      return `\n\n${fence}${language}\n${text}\n${fence}\n\n`;
    },
  });

  service.addRule('absoluteLink', {
    filter: (node) => node.nodeName === 'A' && Boolean(attribute(node as unknown as DomNode, 'href')),
    replacement: (content, node) => {
      const href = resolveHref(attribute(node as unknown as DomNode, 'href'), url);
      const text = content.trim();
      if (!href) return text;
      if (!text) return '';
      return `[${text}](${href})`;
    },
  });

  service.addRule('absoluteImage', {
    filter: (node) => node.nodeName === 'IMG',
    replacement: (_content, node) => {
      const image = node as unknown as DomNode;
      const src = resolveHref(attribute(image, 'src'), url);
      if (!src) return '';
      const alt = attribute(image, 'alt').trim();
      return alt ? `![${alt}](${src})` : '';
    },
  });

  service.addRule('definitionTerm', {
    filter: (node) => node.nodeName === 'DT',
    replacement: (content) => (isBlankContent(content) ? '' : `\n- **${content.trim()}:**`),
  });

  service.addRule('definitionDescription', {
    filter: (node) => node.nodeName === 'DD',
    replacement: (content) => (isBlankContent(content) ? '' : ` ${content.trim().replace(/\s*\n\s*/g, ' ')}\n`),
  });

  service.addRule('definitionList', {
    filter: (node) => node.nodeName === 'DL',
    replacement: (content) => `\n\n${content.trim().replace(/\n{2,}/g, '\n')}\n\n`,
  });

  service.addRule('figureCaption', {
    filter: (node) => node.nodeName === 'FIGCAPTION',
    replacement: (content) => (isBlankContent(content) ? '' : `\n\n*${content.trim()}*\n\n`),
  });

  service.addRule('detailsSummary', {
    filter: (node) => node.nodeName === 'SUMMARY',
    replacement: (content) => (isBlankContent(content) ? '' : `\n\n**${content.trim()}**\n\n`),
  });

  return service;
}

function extractMain(html: string): string {
  const match = html.match(/<main[\s>][\s\S]*?<\/main>/i);
  if (match) return match[0];
  const body = html.match(/<body[\s>][\s\S]*?<\/body>/i);
  return body ? body[0] : html;
}

function metaContent(html: string, attr: 'name' | 'property', value: string): string | null {
  const pattern = new RegExp(
    `<meta[^>]*\\b${attr}=["']${value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}["'][^>]*>`,
    'i',
  );
  const tag = html.match(pattern)?.[0];
  if (!tag) return null;
  const content = tag.match(/\bcontent=["']([^"']*)["']/i)?.[1];
  return content ? decodeEntities(content).trim() : null;
}

function documentTitle(html: string): string | null {
  const title = html.match(/<title[^>]*>([\s\S]*?)<\/title>/i)?.[1];
  return title ? decodeEntities(title.replace(/\s+/g, ' ')).trim() : null;
}

function tidy(markdown: string): string {
  return markdown
    .replace(/\r\n/g, '\n')
    .replace(/[ \t]+$/gm, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

export interface MarkdownPage {
  markdown: string;
  title: string;
  description: string | null;
}

/** Converts a full HTML document into the page's Markdown representation. */
export function htmlToMarkdown(html: string, options: HtmlToMarkdownOptions): MarkdownPage {
  const service = createService(options.url);
  const description = metaContent(html, 'name', 'description');
  const title =
    options.title ??
    metaContent(html, 'property', 'og:title') ??
    documentTitle(html) ??
    options.url;

  let body = tidy(service.turndown(separateInlineSiblings(extractMain(html))));
  const headingMatch = body.match(/^# (.+)$/m);
  const heading = headingMatch?.[1]?.trim();

  if (!heading) {
    body = `# ${title}\n\n${body}`;
  }
  if (description) {
    const firstHeadingLine = body.match(/^# .+$/m);
    if (firstHeadingLine) {
      body = body.replace(firstHeadingLine[0], `${firstHeadingLine[0]}\n\n> ${description}`);
    }
  }

  const modified = metaContent(html, 'property', 'article:modified_time');
  const footer = [
    '---',
    '',
    `Source: ${options.url}`,
    `Markdown representation of the reinstate.dev page; request the same URL with \`Accept: text/html\` for the full page.${
      modified ? ` Last modified: ${modified.slice(0, 10)}.` : ''
    }`,
  ].join('\n');

  return {
    markdown: `${body}\n\n${footer}\n`,
    title: heading ?? title,
    description,
  };
}
