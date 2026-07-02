export const prerender = false;

import type { APIRoute } from 'astro';
import { SESSION_COOKIE, backendFetch } from '../../../../lib/server-api';

export const GET: APIRoute = async ({ url, cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (!token) {
    return new Response(JSON.stringify({ error: 'sign in required' }), { status: 401 });
  }
  const q = url.searchParams.toString();
  const path = q ? `/v1/admin/reports?${q}` : '/v1/admin/reports';
  const res = await backendFetch(path, {}, token);
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};