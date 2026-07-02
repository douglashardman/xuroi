import type { APIRoute } from 'astro';
import { API_URL } from '../lib/server-api';

interface SitemapEntry {
  url: string;
  lastmod?: string;
  changefreq?: string;
  priority?: string;
}

interface SitemapResponse {
  site: { url: string };
  entries: SitemapEntry[];
}

export const GET: APIRoute = async () => {
  const res = await fetch(`${API_URL}/v1/seo/sitemap`);
  if (!res.ok) {
    return new Response('Sitemap unavailable', { status: 503 });
  }
  const data = (await res.json()) as SitemapResponse;
  const base = data.site.url.replace(/\/$/, '');

  const urls = data.entries
    .map((entry) => {
      const loc = `${base}${entry.url}`;
      const parts = [`<url><loc>${escapeXml(loc)}</loc>`];
      if (entry.lastmod) {
        parts.push(`<lastmod>${new Date(entry.lastmod).toISOString().slice(0, 10)}</lastmod>`);
      }
      if (entry.changefreq) parts.push(`<changefreq>${entry.changefreq}</changefreq>`);
      if (entry.priority) parts.push(`<priority>${entry.priority}</priority>`);
      parts.push('</url>');
      return parts.join('');
    })
    .join('');

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls}</urlset>`;

  return new Response(body, {
    headers: {
      'Content-Type': 'application/xml; charset=utf-8',
      'Cache-Control': 'public, max-age=3600',
    },
  });
};

function escapeXml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}