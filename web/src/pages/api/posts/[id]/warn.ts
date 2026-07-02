export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../../lib/server-api';

export const POST: APIRoute = async ({ params, request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const body = await request.text();
  const res = await backendFetch(`/v1/posts/${params.id}/warn`, { method: 'POST', body }, session);
  const text = await res.text();
  let data: Record<string, unknown> = {};
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    data = { error: text || `API error (${res.status})` };
  }
  if (!res.ok && !data.error) {
    data.error = `Warning failed (${res.status})`;
  }
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};