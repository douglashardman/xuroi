export const prerender = false;

import type { APIRoute } from 'astro';
import { API_URL, sessionFromCookieHeader } from '../../../lib/server-api';

export const GET: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const headers = new Headers();
  if (session) headers.set('X-Session-Token', session);
  const res = await fetch(`${API_URL}/v1/me/export`, { headers });
  const body = await res.text();
  return new Response(body, {
    status: res.status,
    headers: {
      'Content-Type': res.headers.get('Content-Type') ?? 'application/json',
      'Content-Disposition': res.headers.get('Content-Disposition') ?? 'attachment; filename="xuroi-export.json"',
    },
  });
};