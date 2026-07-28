import type { APIRoute } from 'astro';
import { createIndexNowKeyResponse } from '../lib/indexnow-key';

export const prerender = false;

export const GET: APIRoute = ({ params }) =>
  createIndexNowKeyResponse(params.indexnowKey);
