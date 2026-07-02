import type { Post } from './api';
import { bindRichTextToolbars, hasEditorContent, prepareEditorMarkdown } from './rich-text-editor';
import { bindNewPost } from './thread-interactions';
import { insertThreadPost } from './thread-post';
import { togglePostPreview } from './post-preview';
import { showToast } from './toast';

function afterPostUrl(threadUrl: string, postId: string): string {
  const base = threadUrl.split('#')[0];
  return `${base}?page=999#post-${postId}`;
}

function clearComposer(
  body: HTMLElement,
  quoteBox: HTMLElement | null,
  quoteIdInput: HTMLInputElement | null,
  quoteAuthorEl: HTMLElement | null,
  quoteExcerptEl: HTMLTextAreaElement | null,
) {
  body.innerHTML = '';
  if (quoteIdInput) quoteIdInput.value = '';
  if (quoteAuthorEl) quoteAuthorEl.textContent = '';
  if (quoteExcerptEl) quoteExcerptEl.value = '';
  if (quoteBox) quoteBox.hidden = true;
}

export function initReplyForm(box: HTMLElement) {
  const threadId = box.dataset.threadId;
  const threadUrl = box.dataset.threadUrl;
  const bodyId = box.dataset.bodyId;
  const isFull = box.dataset.isFull === '1';
  const onLastPage = box.dataset.lastPage === '1';
  const signedIn = box.dataset.signedIn === '1';
  const isAdmin = box.dataset.isAdmin === '1';
  if (!threadId || !threadUrl || !bodyId) return;

  bindRichTextToolbars(box);

  const draftKey = `xuroi:draft:thread:${threadId}`;
  const btn = document.getElementById('reply-submit');
  const body = document.getElementById(bodyId);
  const status = document.getElementById('reply-status');
  const quoteBox = document.getElementById('quote-compose');
  const quoteAuthorEl = document.getElementById('quote-author');
  const quoteExcerptEl = document.getElementById('quote-excerpt') as HTMLTextAreaElement | null;
  const quoteIdInput = document.getElementById('quoted-post-id') as HTMLInputElement | null;
  const quoteClear = document.getElementById('quote-clear');
  const fullReplyBtn = document.getElementById('full-reply-btn');
  const previewBtn = document.getElementById('reply-preview-btn') as HTMLButtonElement | null;
  const previewEl = document.getElementById('reply-preview') as HTMLElement | null;
  const fullReplyBase = `${threadUrl}/reply`;

  function fullReplyUrl() {
    const q = quoteIdInput?.value?.trim();
    return q ? `${fullReplyBase}?quote=${encodeURIComponent(q)}` : fullReplyBase;
  }

  fullReplyBtn?.addEventListener('click', () => {
    window.location.href = fullReplyUrl();
  });

  function setQuote(id: string, author: string, excerpt: string) {
    if (!quoteBox || !quoteIdInput) return;
    quoteIdInput.value = id || '';
    if (quoteAuthorEl) quoteAuthorEl.textContent = author || '';
    if (quoteExcerptEl) quoteExcerptEl.value = excerpt || '';
    quoteBox.hidden = !id;
    if (isFull) return;
    box.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    if (id && quoteExcerptEl) quoteExcerptEl.focus();
    else if (body instanceof HTMLElement) body.focus();
  }

  if (body instanceof HTMLElement) {
    const saved = localStorage.getItem(draftKey);
    if (saved && !hasEditorContent(body)) {
      body.innerHTML = saved;
    }
    let draftTimer: ReturnType<typeof setTimeout> | undefined;
    body.addEventListener('input', () => {
      clearTimeout(draftTimer);
      draftTimer = setTimeout(() => {
        if (hasEditorContent(body)) {
          localStorage.setItem(draftKey, body.innerHTML);
        } else {
          localStorage.removeItem(draftKey);
        }
      }, 400);
    });
  }

  if (!isFull) {
    (window as Window & { setReplyQuote?: typeof setQuote }).setReplyQuote = setQuote;
    quoteClear?.addEventListener('click', () => setQuote('', '', ''));
    document.getElementById('thread-posts')?.addEventListener('click', (e) => {
      const el = (e.target as HTMLElement).closest('[data-quote]');
      if (!el) return;
      setQuote(
        el.getAttribute('data-quote') || '',
        el.getAttribute('data-quote-author') || '',
        el.getAttribute('data-quote-excerpt') || '',
      );
    });
  } else {
    quoteClear?.addEventListener('click', () => {
      if (quoteIdInput) quoteIdInput.value = '';
      if (quoteBox) quoteBox.hidden = true;
    });
  }

  previewBtn?.addEventListener('click', async () => {
    if (!(body instanceof HTMLElement) || !previewEl) return;
    try {
      await togglePostPreview(body, previewEl, previewBtn);
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Preview failed', 'error');
    }
  });

  btn?.addEventListener('click', async () => {
    if (!(body instanceof HTMLElement) || !hasEditorContent(body)) return;
    btn.setAttribute('disabled', 'true');
    if (status) status.hidden = true;

    let text: string;
    try {
      text = await prepareEditorMarkdown(body);
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Image upload failed', 'error');
      btn.removeAttribute('disabled');
      return;
    }
    if (!text) {
      btn.removeAttribute('disabled');
      return;
    }

    const payload: Record<string, string> = { body_markdown: text };
    const quotedId = quoteIdInput?.value?.trim();
    if (quotedId) {
      payload.quoted_post_id = quotedId;
      const quoteText = quoteExcerptEl?.value?.trim();
      if (quoteText) payload.quote_markdown = quoteText;
    }

    try {
      const res = await fetch(`/api/threads/${threadId}/posts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Post failed');

      const post = data.post as Post | undefined;
      const postId = post?.id ?? data.payload?.post_id;

      if (!isFull && onLastPage && post) {
        const article = insertThreadPost(post, { signedIn, isAdmin });
        bindNewPost(article);
        clearComposer(body, quoteBox, quoteIdInput, quoteAuthorEl, quoteExcerptEl);
        localStorage.removeItem(draftKey);
        article.scrollIntoView({ behavior: 'smooth', block: 'end' });
        showToast('Posted', 'success');
      } else if (postId) {
        window.location.href = afterPostUrl(threadUrl, postId);
        return;
      } else {
        window.location.href = `${threadUrl.split('#')[0]}#reply-box`;
        return;
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Post failed', 'error');
    } finally {
      btn.removeAttribute('disabled');
    }
  });
}