export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch } from '../../../lib/server-api';

export const GET: APIRoute = async () => {
  const res = await backendFetch('/v1/moderation/report-reasons');
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};