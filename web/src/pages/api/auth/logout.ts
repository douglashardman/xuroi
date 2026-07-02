export const prerender = false;

import type { APIRoute } from 'astro';
import { SESSION_COOKIE, backendFetch } from '../../../lib/server-api';

export const POST: APIRoute = async ({ cookies }) => {
  const token = cookies.get(SESSION_COOKIE)?.value;
  if (token) {
    await backendFetch('/v1/auth/logout', { method: 'POST' }, token);
  }
  cookies.delete(SESSION_COOKIE, { path: '/' });
  return new Response(JSON.stringify({ status: 'logged_out' }));
};