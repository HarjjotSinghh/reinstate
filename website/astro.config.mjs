// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import mdx from '@astrojs/mdx';
import vercel from '@astrojs/vercel';

const ignoreGeneratedDevPath = (filePath) =>
  /\/(?:\.vercel|dist)(?:\/|$)/.test(filePath.replaceAll('\\', '/'));

// https://astro.build/config
export default defineConfig({
  site: 'https://reinstate.dev',
  output: 'server',
  adapter: vercel(),
  integrations: [mdx()],
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
