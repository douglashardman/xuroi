export const prerender = false;

import type { APIRoute } from 'astro';
import { SESSION_COOKIE, SESSION_MAX_AGE, backendFetch } from '../../../../../lib/server-api';

export const POST: APIRoute = async ({ request, cookies }) => {
  const body = await request.json();
  const res = await backendFetch('/v1/auth/passkey/login/finish', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (res.ok && data.token) {
    cookies.set(SESSION_COOKIE, data.token, {
      path: '/',
      httpOnly: true,
      sameSite: 'lax',
      maxAge: SESSION_MAX_AGE,
    });
  }
  return new Response(JSON.stringify(data), { status: res.status });
};