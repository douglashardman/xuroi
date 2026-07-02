export const prerender = false;

import type { APIRoute } from 'astro';
import { backendFetch } from '../../../lib/server-api';

export const GET: APIRoute = async ({ url, cookies, request }) => {
  const token = cookies.get('xuroi_session')?.value;
  const q = url.searchParams.toString();
  const res = await backendFetch(q ? `/v1/search?${q}` : '/v1/search', {
    headers: request.headers,
  }, token ?? undefined);
  const data = await res.json();
  return new Response(JSON.stringify(data), { status: res.status });
};