export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../../../lib/server-api';

export const PUT: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const body = await request.text();
  const res = await backendFetch(
    `/v1/admin/users/${params.id}/entitlements`,
    { method: 'PUT', body },
    session,
  );
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};