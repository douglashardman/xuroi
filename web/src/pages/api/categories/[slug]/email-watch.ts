export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../../lib/server-api';

export const GET: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  if (!session) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await backendFetch(`/v1/categories/${params.slug}/email-watch`, {}, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};

export const PATCH: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  if (!session) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const body = await request.text();
  const res = await backendFetch(`/v1/categories/${params.slug}/email-watch`, {
    method: 'PATCH',
    body,
    headers: { 'Content-Type': 'application/json' },
  }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};