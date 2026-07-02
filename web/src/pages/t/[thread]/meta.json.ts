export const prerender = false;

import type { APIRoute } from 'astro';
import { API_URL } from '../../../lib/server-api';

export const GET: APIRoute = async ({ params }) => {
  const threadParam = params.thread ?? '';
  const sep = threadParam.lastIndexOf('--');
  if (sep < 0) {
    return new Response(JSON.stringify({ error: 'invalid thread url' }), { status: 400 });
  }
  const id = threadParam.slice(sep + 2);
  const res = await fetch(`${API_URL}/v1/threads/${id}/meta.json`);
  const body = await res.text();
  return new Response(body, {
    status: res.status,
    headers: { 'Content-Type': 'application/json' },
  });
};