import type { ThreadPageResponse } from './api';

export interface SeoMeta {
  title: string;
  description?: string;
  canonical?: string;
  ogImage?: string;
  ogType?: string;
  noIndex?: boolean;
}

export function absoluteUrl(siteUrl: string, path: string) {
  const base = siteUrl.replace(/\/$/, '');
  if (!path || path === '/') return base + '/';
  return base + (path.startsWith('/') ? path : `/${path}`);
}

export function pageCanonical(siteUrl: string, pathname: string, search = '') {
  const url = new URL(pathname + search, siteUrl.replace(/\/$/, '') + '/');
  return url.href;
}

export function defaultOgImage(siteUrl: string) {
  return absoluteUrl(siteUrl, '/brand/pt-bug.svg');
}

export function threadJsonLd(data: ThreadPageResponse, siteUrl: string) {
  const threadUrl = `${siteUrl}${data.thread.url}`;
  const comments = data.posts
    .filter((p) => !p.is_op)
    .map((p) => ({
      '@type': 'Comment',
      text: stripHtml(p.body_html).slice(0, 500),
      dateCreated: p.created_at,
      author: { '@type': 'Person', name: p.author.name },
    }));

  const op = data.posts.find((p) => p.is_op);

  return {
    '@context': 'https://schema.org',
    '@type': 'DiscussionForumPosting',
    headline: data.thread.title,
    url: threadUrl,
    datePublished: data.thread.created_at,
    dateModified: data.thread.last_activity_at,
    author: op
      ? { '@type': 'Person', name: op.author.name }
      : { '@type': 'Organization', name: data.site.name },
    publisher: { '@type': 'Organization', name: data.site.name, url: siteUrl },
    articleSection: data.category.name,
    commentCount: data.thread.reply_count,
    comment: comments,
  };
}

function stripHtml(html: string): string {
  return html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim();
}