export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../../../lib/server-api';

export const POST: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const res = await backendFetch(`/v1/mod/posts/${params.id}/restore`, { method: 'POST' }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};