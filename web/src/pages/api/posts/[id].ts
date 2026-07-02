export const prerender = false;

import type { APIRoute } from 'astro';
import { SESSION_COOKIE, backendFetch } from '../../../lib/server-api';

export const DELETE: APIRoute = async ({ params, cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (!token) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await backendFetch(`/v1/posts/${params.id}`, { method: 'DELETE' }, token);
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};

export const PATCH: APIRoute = async ({ params, request, cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (!token) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const body = await request.json();
  const res = await backendFetch(`/v1/posts/${params.id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  }, token);
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};