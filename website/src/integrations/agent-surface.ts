/**
 * Astro integration that turns the finished static build into an
 * agent-readable surface: Markdown twins for every page, `llms-full.txt`,
 * the Vercel routes that make `Accept: text/markdown` work on the canonical
 * URLs, and the dedicated Vercel function those routes target. Runs after the Vercel adapter's own `astro:build:done`
 * (Astro registers the adapter first), so the Build Output folder already
 * exists when this hook fires.
 */
import type { AstroIntegration } from 'astro';
import { fileURLToPath } from 'node:url';
import { join } from 'node:path';
import { buildAgentSurface } from '../lib/agent-surface/build';

export interface AgentSurfaceOptions {
  /** Canonical site origin; defaults to the Astro `site` setting. */
  site?: string;
  productName?: string;
}

export default function agentSurface(options: AgentSurfaceOptions = {}): AstroIntegration {
  let root = '';
  let site = options.site ?? '';

  return {
    name: 'reinstate-agent-surface',
    hooks: {
      'astro:config:done': ({ config }) => {
        root = fileURLToPath(config.root);
        site = options.site ?? config.site ?? '';
        if (!site) {
          throw new Error('reinstate-agent-surface needs `site` in astro.config.mjs to write absolute Markdown links.');
        }
      },
      'astro:build:done': async ({ dir, pages, logger }) => {
        const clientDir = fileURLToPath(dir);
        const vercelOutput = join(root, '.vercel', 'output');
        const result = await buildAgentSurface({
          clientDir,
          pathnames: pages.map((page) => page.pathname),
          site,
          productName: options.productName,
          mirrorDirs: [join(vercelOutput, 'static')],
          vercelConfigPath: join(vercelOutput, 'config.json'),
          vercelFunctionsDir: join(vercelOutput, 'functions'),
          agentFunctionEntry: join(root, 'src', 'lib', 'agent-surface', 'function.ts'),
        });
        logger.info(
          `${result.twins.length} Markdown twins, llms-full.txt (${Math.round(result.llmsFullBytes / 1024)} KiB)${
            result.vercelRoutesInjected ? ', Vercel agent routes injected' : ''
          }${result.agentFunction ? `, agent function bundled (${Math.round(result.agentFunction.bytes / 1024)} KiB, ${result.agentFunction.runtime})` : ''}${result.skipped.length ? `; skipped ${result.skipped.join(', ')}` : ''}`,
        );
      },
    },
  };
}
