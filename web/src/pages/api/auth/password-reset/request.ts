export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch } from '../../../../lib/server-api';

export const POST: APIRoute = async ({ request }) => {
  const body = await request.json();
  const res = await backendFetch('/v1/auth/password-reset/request', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};