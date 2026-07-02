export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../lib/server-api';

export const GET: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const res = await backendFetch('/v1/admin/site-settings', {}, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};

export const PATCH: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const body = await request.text();
  const res = await backendFetch('/v1/admin/site-settings', { method: 'PATCH', body }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), { status: res.status });
};