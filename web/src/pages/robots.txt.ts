import type { APIRoute } from 'astro';

export const GET: APIRoute = ({ site }) => {
  const base = (site?.toString() ?? 'http://localhost:4321').replace(/\/$/, '');
  const body = `User-agent: *
Allow: /

Disallow: /mod/
Disallow: /admin/
Disallow: /api/

Sitemap: ${base}/sitemap.xml
`;
  return new Response(body, {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'public, max-age=86400',
    },
  });
};