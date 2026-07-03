export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch, sessionFromCookieHeader } from '../../../lib/server-api';

export const GET: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const res = await backendFetch('/v1/me/agent', {}, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};

export const POST: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const body = await request.text();
  const res = await backendFetch('/v1/me/agent', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};

export const PATCH: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const body = await request.text();
  const res = await backendFetch('/v1/me/agent', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body,
  }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};

export const DELETE: APIRoute = async ({ request }) => {
  const session = sessionFromCookieHeader(request.headers.get('cookie'));
  const res = await backendFetch('/v1/me/agent', { method: 'DELETE' }, session);
  const data = await res.json().catch(() => ({}));
  return new Response(JSON.stringify(data), {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};