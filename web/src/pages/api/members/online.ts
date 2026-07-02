export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../lib/server-api';

export const GET: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const res = await backendFetch('/v1/members/online', {}, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};