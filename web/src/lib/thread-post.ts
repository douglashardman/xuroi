import type { Post } from './api';
import { formatDate } from './api';
import { postEditFormHTML } from './post-edit-form';
import { postModGearHTML } from './mod-gear';
import { postReportButtonHTML } from './post-report';
import { ACCENT_CLASSES, PAV_CLASSES, accentIndex, avatarSrc, initials } from './theme';

function escapeText(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function excerptFromHtml(html: string): string {
  return html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 200);
}

export function renderPostArticle(post: Post, opts: { signedIn: boolean; isStaff: boolean }): HTMLElement {
  const idx = accentIndex(post.id);
  const excerpt = excerptFromHtml(post.body_html);
  const article = document.createElement('article');
  article.className = `post ${ACCENT_CLASSES[idx]} post--enter`;
  article.id = `post-${post.id}`;

  const karma =
    post.author.karma > 0
      ? `<div class="pkarma">${post.author.karma.toLocaleString()} karma</div>`
      : '';

  const quote = post.quote
    ? `<a href="${escapeText(post.quote.url)}" class="post-quote-link" title="Jump to quoted post">
        <blockquote class="post-quote">
          <cite>${escapeText(post.quote.author_name)} wrote:</cite>
          ${escapeText(post.quote.excerpt)}
        </blockquote>
      </a>`
    : '';

  const editBtn = post.can_edit
    ? `<button type="button" class="post-action" data-edit="${post.id}">Edit</button>`
    : '';
  const deleteBtn = post.can_delete
    ? `<button type="button" class="post-action post-action--danger" data-delete="${post.id}">Delete</button>`
    : '';
  const quoteBtn = opts.signedIn
    ? `<button type="button" class="post-action" data-quote="${post.id}" data-quote-author="${escapeText(post.author.name)}" data-quote-excerpt="${escapeText(excerpt)}">Quote</button>`
    : '';
  const reportBtn = opts.signedIn ? postReportButtonHTML(post.id) : '';
  const modGear = opts.isStaff ? postModGearHTML(post.id, !!post.is_warned) : '';

  const editForm = post.can_edit ? postEditFormHTML(post.id) : '';

  const avSrc = avatarSrc(post.author.avatar_url, 'sm');
  const botSvg = `<svg viewBox="0 0 24 24" width="55%" height="55%" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="8" width="14" height="11" rx="2"/><path d="M9 8V6a3 3 0 0 1 6 0v2"/><circle cx="10" cy="13" r="1" fill="currentColor" stroke="none"/><circle cx="14" cy="13" r="1" fill="currentColor" stroke="none"/><path d="M9 17h6"/></svg>`;
  const avHtml = avSrc
    ? `<div class="avatar-wrap${post.author.is_agent ? ' avatar-wrap--agent' : ''}"><img src="${escapeText(avSrc)}" alt="" class="avatar avatar--post${post.author.active_warning ? ' avatar--warned' : ''}" loading="lazy" decoding="async"${post.author.active_warning ? ' title="Active warning"' : ''} /></div>`
    : post.author.is_agent
      ? `<div class="avatar-wrap avatar-wrap--agent"><div class="avatar avatar--agent-bot avatar--post ${PAV_CLASSES[idx]}${post.author.active_warning ? ' avatar--warned' : ''}" aria-hidden="true">${botSvg}</div></div>`
      : `<div class="avatar avatar--initials avatar--post ${PAV_CLASSES[idx]}${post.author.active_warning ? ' avatar--warned' : ''}"${post.author.active_warning ? ' title="Active warning"' : ''} aria-hidden="true">${initials(post.author.name)}</div>`;

  const agentOwnerLine = post.author.is_agent && post.author.agent_label
    ? `<p class="agent-owner-line agent-owner-line--compact"><svg class="agent-owner-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="5" y="8" width="14" height="11" rx="2"/><path d="M9 8V6a3 3 0 0 1 6 0v2"/><circle cx="10" cy="13" r="1" fill="currentColor" stroke="none"/><circle cx="14" cy="13" r="1" fill="currentColor" stroke="none"/><path d="M9 17h6"/></svg><span>${escapeText(post.author.agent_label)}</span></p>`
    : '';

  article.innerHTML = `
    <div class="pside">
      ${avHtml}
      <div class="pname"><a href="${escapeText(post.author.url)}">${escapeText(post.author.name)}</a>${agentOwnerLine}</div>
      ${post.is_op ? '<div class="prole">Original post</div>' : ''}
      ${karma}
    </div>
    <div class="pbody">
      <div class="pdate">
        <div class="pdate-meta">
          <time datetime="${post.created_at}">${formatDate(post.created_at)}</time>
        </div>
        ${reportBtn}
      </div>
      ${quote}
      <div class="post-body" data-post-body="${post.id}">${post.body_html}</div>
      ${editForm}
      <div class="post-actions">
        ${editBtn}
        ${deleteBtn}
        ${quoteBtn}
        <button type="button" class="post-like" data-post-id="${post.id}" data-liked="0" title="Like this post">
          ♥ <span class="like-count">0</span>
        </button>
        ${modGear}
      </div>
    </div>
  `;

  window.setTimeout(() => article.classList.remove('post--enter'), 450);
  return article;
}

export function applyPostEdit(postId: string, post: Post, opts: { isAdmin: boolean }) {
  const article = document.getElementById(`post-${postId}`);
  if (!article) return;

  const body = article.querySelector<HTMLElement>(`[data-post-body="${postId}"]`);
  const form = article.querySelector<HTMLElement>(`[data-post-edit="${postId}"]`);
  const pdate = article.querySelector<HTMLElement>('.pdate');

  if (body) {
    body.innerHTML = post.body_html;
    body.hidden = false;
  }
  if (form) form.hidden = true;

  pdate?.querySelector('.post-edited-stamp')?.remove();
  if (post.edited_at && pdate) {
    const editedLabel = formatDate(post.edited_at);
    if (post.can_edit || opts.isAdmin) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'post-edited-stamp';
      btn.dataset.history = postId;
      btn.title = 'View edit history';
      btn.textContent = 'Edited';
      pdate.append(btn);
    } else {
      const span = document.createElement('span');
      span.className = 'post-edited-stamp';
      span.title = `Last edited ${editedLabel}`;
      span.textContent = 'Edited';
      pdate.append(span);
    }
  }

  const quoteBtn = article.querySelector<HTMLElement>('[data-quote]');
  if (quoteBtn) {
    quoteBtn.dataset.quoteExcerpt = excerptFromHtml(post.body_html);
  }

  const editBtn = article.querySelector<HTMLElement>('[data-edit]');
  if (editBtn && !post.can_edit) editBtn.remove();
}

export function removePostArticle(postId: string) {
  const article = document.getElementById(`post-${postId}`);
  if (!article) return;
  article.classList.add('post--leave');
  const done = () => article.remove();
  article.addEventListener('animationend', done, { once: true });
  window.setTimeout(done, 450);
}

export function insertThreadPost(post: Post, opts: { signedIn: boolean; isStaff: boolean }): HTMLElement {
  const anchor = document.getElementById('thread-posts-end');
  const article = renderPostArticle(post, opts);
  if (anchor) {
    anchor.insertAdjacentElement('beforebegin', article);
  } else {
    const replyBox = document.getElementById('reply-box');
    replyBox?.insertAdjacentElement('beforebegin', article);
  }
  return article;
}