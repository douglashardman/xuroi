export const prerender = false;

import type { APIRoute } from 'astro';
import { API_URL } from '../../../lib/server-api';

export const GET: APIRoute = async ({ params }) => {
  const name = params.name;
  if (!name || !/^med_[0-9a-z]{26}(_thumb)?\.webp$/.test(name)) {
    return new Response('Not found', { status: 404 });
  }

  const res = await fetch(`${API_URL}/v1/media/${name}`);
  if (!res.ok) {
    return new Response('Not found', { status: res.status });
  }

  const headers = new Headers();
  headers.set('Content-Type', 'image/webp');
  const cache = res.headers.get('Cache-Control');
  if (cache) headers.set('Cache-Control', cache);

  return new Response(res.body, { status: 200, headers });
};