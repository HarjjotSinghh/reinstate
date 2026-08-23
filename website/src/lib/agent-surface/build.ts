/**
 * Post-build generation of the static agent surface:
 *
 * - one Markdown twin per prerendered HTML page (`/docs/faq` → `docs/faq.md`),
 * - `llms-full.txt`, every indexable page as one Markdown file,
 *
 * written into each static output directory (Astro's `dist/client` and the
 * Vercel Build Output `static` folder), plus the agent routes injected into
 * the Vercel Build Output `config.json`.
 */
import { mkdir, readFile, stat, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { htmlToMarkdown } from './html-to-markdown';
import { markdownFileFor, pagePathFromBuildPathname } from './paths';
import { injectAgentRoutes, type VercelOutputConfig } from './vercel-routes';
import { bundleAgentFunction, type BundledAgentFunction } from './bundle';

export interface BuildAgentSurfaceOptions {
  /** Directory that holds the prerendered HTML (`dist/client`). */
  clientDir: string;
  /** Page pathnames as Astro reports them from `astro:build:done`. */
  pathnames: string[];
  /** Canonical site origin, e.g. `https://reinstate.dev`. */
  site: string;
  /** Extra static directories that must receive the same files (Vercel `static`). */
  mirrorDirs?: string[];
  /** Vercel Build Output `config.json` to inject the agent routes into, if it exists. */
  vercelConfigPath?: string;
  /** Vercel Build Output `functions` directory; the dedicated agent function is bundled there when set. */
  vercelFunctionsDir?: string;
  /** Absolute path of the agent function entry (`src/lib/agent-surface/function.ts`). */
  agentFunctionEntry?: string;
  /** Product name used in the llms-full.txt header. */
  productName?: string;
}

export interface BuiltTwin {
  pagePath: string;
  file: string;
  title: string;
  bytes: number;
}

export interface BuildAgentSurfaceResult {
  twins: BuiltTwin[];
  llmsFullBytes: number;
  vercelRoutesInjected: boolean;
  /** Present when the dedicated agent function was bundled into the Build Output. */
  agentFunction: BundledAgentFunction | null;
  skipped: string[];
}

/** Pages that never get a twin or a place in llms-full.txt. */
export function isExcludedFromTwins(pagePath: string): boolean {
  return pagePath === '/404' || pagePath.startsWith('/404/');
}

/** Pages that get a twin (so negotiation works) but stay out of llms-full.txt. */
export function isExcludedFromFullText(pagePath: string): boolean {
  return isExcludedFromTwins(pagePath) || pagePath === '/preview' || pagePath.startsWith('/preview/');
}

/** Reading order for llms-full.txt: product facts first, then docs, then everything else alphabetically. */
export const FULL_TEXT_PRIORITY = [
  '/',
  '/about/reinstate',
  '/developers',
  '/docs',
  '/docs/getting-started',
  '/docs/installation',
  '/docs/configuration',
  '/docs/features',
  '/docs/cli-reference',
  '/docs/sync-a-session',
  '/docs/restore-a-session',
  '/docs/handoff',
  '/docs/storage',
  '/docs/adapters',
  '/docs/architecture',
  '/docs/security-model',
  '/docs/troubleshooting',
  '/docs/faq',
  '/docs/limitations',
  '/docs/comparison',
  '/docs/universal-configuration',
  '/compatibility',
  '/compatibility/agent-version-history',
  '/security',
  '/integrations',
  '/guides',
] as const;

export function fullTextOrder(pagePaths: string[]): string[] {
  const priority = new Map<string, number>(FULL_TEXT_PRIORITY.map((path, index) => [path, index]));
  return [...pagePaths].sort((a, b) => {
    const pa = priority.get(a) ?? Number.POSITIVE_INFINITY;
    const pb = priority.get(b) ?? Number.POSITIVE_INFINITY;
    if (pa !== pb) return pa - pb;
    return a.localeCompare(b);
  });
}

async function exists(path: string): Promise<boolean> {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}

/** Locates the HTML file Astro wrote for a page, whichever `build.format` produced it. */
export async function htmlFileForPage(clientDir: string, pagePath: string): Promise<string | null> {
  const relative = pagePath === '/' ? '' : pagePath.slice(1);
  const candidates = relative
    ? [join(clientDir, relative, 'index.html'), join(clientDir, `${relative}.html`)]
    : [join(clientDir, 'index.html')];
  for (const candidate of candidates) {
    if (await exists(candidate)) return candidate;
  }
  return null;
}

async function writeEverywhere(dirs: string[], relativeFile: string, content: string): Promise<void> {
  await Promise.all(
    dirs.map(async (dir) => {
      const target = join(dir, relativeFile);
      await mkdir(dirname(target), { recursive: true });
      await writeFile(target, content, 'utf8');
    }),
  );
}

export function llmsFullHeader(productName: string, site: string, count: number): string {
  return [
    `# ${productName}: full documentation`,
    '',
    `> Every indexable ${site} page as Markdown, in reading order. Each section starts with its canonical URL. The curated index is ${site}/llms.txt; the HTTP surface is described by ${site}/openapi.json.`,
    '',
    `Pages: ${count}`,
    '',
  ].join('\n');
}

export async function buildAgentSurface(options: BuildAgentSurfaceOptions): Promise<BuildAgentSurfaceResult> {
  const site = options.site.replace(/\/+$/, '');
  const productName = options.productName ?? 'Reinstate';
  const outputDirs = [options.clientDir, ...(options.mirrorDirs ?? [])];
  const presentDirs: string[] = [];
  for (const dir of outputDirs) {
    if (await exists(dir)) presentDirs.push(dir);
  }

  const twins: BuiltTwin[] = [];
  const skipped: string[] = [];
  const markdownByPage = new Map<string, string>();

  const pagePaths = [...new Set(options.pathnames.map(pagePathFromBuildPathname))].sort();
  for (const pagePath of pagePaths) {
    if (isExcludedFromTwins(pagePath)) {
      skipped.push(pagePath);
      continue;
    }
    const htmlFile = await htmlFileForPage(options.clientDir, pagePath);
    if (!htmlFile) {
      skipped.push(pagePath);
      continue;
    }
    const html = await readFile(htmlFile, 'utf8');
    const page = htmlToMarkdown(html, { url: `${site}${pagePath}` });
    const file = markdownFileFor(pagePath);
    await writeEverywhere(presentDirs, file, page.markdown);
    markdownByPage.set(pagePath, page.markdown);
    twins.push({ pagePath, file, title: page.title, bytes: Buffer.byteLength(page.markdown, 'utf8') });
  }

  const fullTextPages = fullTextOrder([...markdownByPage.keys()].filter((path) => !isExcludedFromFullText(path)));
  const sections = fullTextPages.map((pagePath) => {
    const markdown = markdownByPage.get(pagePath)!;
    return `<!-- ${site}${pagePath} -->\n\n${markdown.trimEnd()}\n`;
  });
  const llmsFull = `${llmsFullHeader(productName, site, fullTextPages.length)}\n${sections.join('\n---\n\n')}`;
  await writeEverywhere(presentDirs, 'llms-full.txt', llmsFull);

  let vercelRoutesInjected = false;
  if (options.vercelConfigPath && (await exists(options.vercelConfigPath))) {
    const config = JSON.parse(await readFile(options.vercelConfigPath, 'utf8')) as VercelOutputConfig;
    const updated = injectAgentRoutes(config);
    await writeFile(options.vercelConfigPath, `${JSON.stringify(updated, null, '\t')}\n`, 'utf8');
    vercelRoutesInjected = true;
  }

  let agentFunction: BundledAgentFunction | null = null;
  if (options.vercelFunctionsDir && options.agentFunctionEntry && (await exists(options.vercelFunctionsDir))) {
    agentFunction = await bundleAgentFunction({ entry: options.agentFunctionEntry, functionsDir: options.vercelFunctionsDir });
  }

  return {
    twins,
    llmsFullBytes: Buffer.byteLength(llmsFull, 'utf8'),
    vercelRoutesInjected,
    agentFunction,
    skipped,
  };
}
