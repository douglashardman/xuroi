export const prerender = false;

import type { APIRoute } from 'astro';
import { SESSION_COOKIE, backendFetch } from '../../../../../lib/server-api';

export const POST: APIRoute = async ({ params, cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (!token) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await backendFetch(`/v1/mod/posts/${params.id}/reject`, { method: 'POST' }, token);
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};