export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch } from '../../../lib/server-api';

export const GET: APIRoute = async ({ url }) => {
  const name = url.searchParams.get('name')?.trim() ?? '';
  if (!name) {
    return new Response(JSON.stringify({ error: 'name required' }), { status: 400 });
  }
  const res = await backendFetch(`/v1/auth/check-display-name?name=${encodeURIComponent(name)}`);
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};