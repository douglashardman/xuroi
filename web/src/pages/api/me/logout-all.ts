export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../lib/server-api';

export const POST: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  if (!session) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await backendFetch('/v1/me/logout-all', { method: 'POST' }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};