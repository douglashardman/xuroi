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

function looksLikeQuestion(title: string): boolean {
  const t = title.trim();
  if (t.endsWith('?')) return true;
  return /^(how|what|why|when|where|which|who|best|anyone|recommend|help)\b/i.test(t);
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
  const opText = op ? stripHtml(op.body_html).slice(0, 800) : '';

  const base = {
    '@context': 'https://schema.org',
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

  if (looksLikeQuestion(data.thread.title) && opText) {
    return {
      ...base,
      '@type': 'FAQPage',
      mainEntity: [
        {
          '@type': 'Question',
          name: data.thread.title,
          acceptedAnswer: {
            '@type': 'Answer',
            text: opText,
            dateCreated: data.thread.created_at,
            author: op ? { '@type': 'Person', name: op.author.name } : undefined,
          },
        },
      ],
    };
  }

  return {
    ...base,
    '@type': 'DiscussionForumPosting',
  };
}

export function siteSearchJsonLd(siteUrl: string, siteName: string) {
  const base = siteUrl.replace(/\/$/, '');
  return {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: siteName,
    url: base + '/',
    potentialAction: {
      '@type': 'SearchAction',
      target: {
        '@type': 'EntryPoint',
        urlTemplate: base + '/search?q={search_term_string}',
      },
      'query-input': 'required name=search_term_string',
    },
  };
}

function stripHtml(html: string): string {
  return html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim();
}