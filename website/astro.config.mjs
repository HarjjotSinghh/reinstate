// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import mdx from '@astrojs/mdx';
import vercel from '@astrojs/vercel';
import sitemap from '@astrojs/sitemap';

const ignoreGeneratedDevPath = (filePath) =>
  /\/(?:\.vercel|dist)(?:\/|$)/.test(filePath.replaceAll('\\', '/'));

// https://astro.build/config
export default defineConfig({
  site: 'https://reinstate.dev',
  output: 'server',
  adapter: vercel(),
  trailingSlash: 'never',
  integrations: [
    mdx(),
    sitemap({
      filter: (page) =>
        !page.includes('/api/') &&
        !page.includes('/preview') &&
        !page.endsWith('/404'),
    }),
  ],
  vite: {
    plugins: [tailwindcss()],
    server: {
      watch: {
        ignored: ignoreGeneratedDevPath,
        usePolling: true,
        interval: 100,
      },
    },
  },
  markdown: {
    shikiConfig: {
      theme: 'github-dark-default',
      wrap: true,
    },
  },
});
