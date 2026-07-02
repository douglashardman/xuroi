export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../../lib/server-api';

export const POST: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  if (!session) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await backendFetch(`/v1/actors/${params.id}/block`, { method: 'POST' }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};

export const DELETE: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  if (!session) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await backendFetch(`/v1/actors/${params.id}/block`, { method: 'DELETE' }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};