import { openApiJson } from '../lib/openapi';

export const prerender = true;

export function GET() {
  return new Response(openApiJson(), {
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'public, max-age=300, s-maxage=3600',
      'X-Content-Type-Options': 'nosniff',
    },
  });
}
