export const prerender = false;

import type { APIRoute } from 'astro';
import { API_URL, SESSION_COOKIE } from '../../../lib/server-api';

export const POST: APIRoute = async ({ request, cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (!token) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const form = await request.formData();
  const res = await fetch(`${API_URL}/v1/me/avatar`, {
    method: 'POST',
    headers: { 'X-Session-Token': token },
    body: form,
  });
  const data = await res.json();
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};

export const DELETE: APIRoute = async ({ cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (!token) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const res = await fetch(`${API_URL}/v1/me/avatar`, {
    method: 'DELETE',
    headers: { 'X-Session-Token': token },
  });
  const data = await res.json();
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};