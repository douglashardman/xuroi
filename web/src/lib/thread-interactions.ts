import type { Post, ReportReason } from './api';
import {
  closeAllModPopovers,
  syncThreadLockLabel,
  syncThreadPinLabel,
  syncThreadReportCount,
} from './mod-gear';
import { markPostReported, markReported } from './post-report';
import { initLightbox } from './lightbox';
import { findPostEditor } from './post-edit-form';
import {
  bindRichTextToolbars,
  clearEditorAttachments,
  hasEditorContent,
  normalizePostHtmlForEdit,
  prepareEditorMarkdown,
  setEditorHtml,
} from './rich-text-editor';
import { applyPostEdit, removePostArticle } from './thread-post';
import { confirm, promptDialog, showToast } from './toast';

function openPostEditor(postId: string) {
  const body = document.querySelector<HTMLElement>(`[data-post-body="${postId}"]`);
  const form = document.querySelector<HTMLElement>(`[data-post-edit="${postId}"]`);
  const editor = form ? findPostEditor(form) : null;
  if (!body || !form || !editor) return;

  clearEditorAttachments(editor);
  setEditorHtml(editor, normalizePostHtmlForEdit(body.innerHTML));
  bindRichTextToolbars(form);

  body.hidden = true;
  form.hidden = false;
  editor.focus();
}

function refreshThreadTmeta() {
  const el = document.getElementById('thread-tmeta');
  if (!el) return;
  const replies = Number(el.dataset.replyCount ?? '0');
  const replyWord = replies === 1 ? 'reply' : 'replies';
  const parts = [
    `${replies} ${replyWord}`,
    `started ${el.dataset.started ?? ''}`,
    `last activity ${el.dataset.lastActivity ?? ''}`,
  ];
  if (el.dataset.locked === '1') parts.push('Locked');
  if (el.dataset.pinned === '1') parts.push('Pinned');
  el.textContent = parts.join(' · ');
}

function setThreadLocked(locked: boolean) {
  const tmeta = document.getElementById('thread-tmeta');
  if (tmeta) tmeta.dataset.locked = locked ? '1' : '0';
  refreshThreadTmeta();

  const replyBox = document.getElementById('reply-box');
  if (!replyBox || replyBox.dataset.signedIn !== '1') return;
  if (locked) {
    replyBox.innerHTML = '<p class="reply-hint">This thread is locked.</p>';
  } else {
    showToast('Thread unlocked — refresh to reply', 'info');
  }
}

function setThreadPinned(pinned: boolean) {
  const tmeta = document.getElementById('thread-tmeta');
  if (tmeta) tmeta.dataset.pinned = pinned ? '1' : '0';
  refreshThreadTmeta();
}

type PanelFn = (title: string, html: string) => void;
type ClosePanelFn = () => void;

function loadReportReasons(): ReportReason[] {
  const root = document.getElementById('thread-posts');
  const raw = root?.getAttribute('data-report-reasons');
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as ReportReason[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function pickReportReason(openPanel: PanelFn, closePanel: ClosePanelFn): Promise<{ reason_id: string; detail: string } | null> {
  const reasons = loadReportReasons();
  if (!reasons.length) {
    return promptDialog('Add an optional reason for moderators.', {
      title: 'Flag for moderators',
      placeholder: 'Reason (optional)',
      defaultValue: '',
    }).then((reason) => (reason === null ? null : { reason_id: '', detail: reason.trim() }));
  }

  const defaultId = reasons[0].id;
  const radios = reasons.map((r, i) => `
    <label class="report-reason-option">
      <input type="radio" name="report_reason" value="${r.id}" ${i === 0 ? 'checked' : ''} data-allow-detail="${r.allow_detail ? '1' : '0'}" />
      <span>${r.label}</span>
    </label>
  `).join('');

  return new Promise((resolve) => {
    openPanel('Flag for moderators', `
      <p class="report-reason-intro">Tell moderators what’s wrong with this post.</p>
      <form class="report-reason-form" id="report-reason-form">
        <div class="report-reason-list">${radios}</div>
        <label class="report-reason-detail" id="report-reason-detail" hidden>
          <span>Details (optional)</span>
          <textarea name="detail" rows="3" maxlength="500" placeholder="Add context for moderators"></textarea>
        </label>
        <div class="report-reason-actions">
          <button type="button" class="btn btn--sm" data-report-cancel>Cancel</button>
          <button type="submit" class="btn btn--sm btn--pink">Submit report</button>
        </div>
      </form>
    `);

    const body = document.getElementById('mod-panel-body');
    const form = body?.querySelector('#report-reason-form') as HTMLFormElement | null;
    const detailWrap = body?.querySelector('#report-reason-detail') as HTMLElement | null;
    const detailInput = detailWrap?.querySelector('textarea') as HTMLTextAreaElement | null;

    const syncDetail = () => {
      const selected = form?.querySelector('input[name="report_reason"]:checked') as HTMLInputElement | null;
      if (!detailWrap) return;
      detailWrap.hidden = selected?.dataset.allowDetail !== '1';
    };
    syncDetail();
    form?.querySelectorAll('input[name="report_reason"]').forEach((el) => {
      el.addEventListener('change', syncDetail);
    });

    const finish = (value: { reason_id: string; detail: string } | null) => {
      closePanel();
      resolve(value);
    };

    body?.querySelector('[data-report-cancel]')?.addEventListener('click', () => finish(null));
    form?.addEventListener('submit', (ev) => {
      ev.preventDefault();
      const selected = form.querySelector('input[name="report_reason"]:checked') as HTMLInputElement | null;
      if (!selected?.value) {
        showToast('Pick a reason', 'error');
        return;
      }
      finish({
        reason_id: selected.value,
        detail: detailInput?.value.trim() ?? '',
      });
    });
  });
}

function threadPostsRoot() {
  return document.getElementById('thread-posts');
}

function isStaffViewer(): boolean {
  return threadPostsRoot()?.dataset.isStaff === '1';
}

function isAdminViewer(): boolean {
  return threadPostsRoot()?.dataset.isAdmin === '1';
}

function canPermBanViewer(): boolean {
  return threadPostsRoot()?.dataset.canPermBan === '1';
}

async function staffBanAuthor(
  authorId: string,
  reason: string,
  duration: '7d' | '30d' | 'permanent',
  purgeContent: boolean,
) {
  const res = await fetch(`/api/admin/users/${authorId}/ban`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason, duration, purge_content: purgeContent }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Ban failed');
  return data;
}

function renderModAuditPanel(audit: {
  post_id: string;
  thread_id: string;
  thread_title: string;
  author_id: string;
  author_name: string;
  author_ip?: string | null;
  created_at: string;
  edited_at?: string | null;
  revision_count: number;
  reaction_count: number;
  is_op: boolean;
}) {
  const ip = audit.author_ip ?? '—';
  const durationOptions = [
    '<label><input type="radio" name="mod-ban-duration" value="7d" checked /> 7-day timeout</label>',
  ];
  if (isAdminViewer()) {
    durationOptions.push('<label><input type="radio" name="mod-ban-duration" value="30d" /> 30 days</label>');
  }
  if (isAdminViewer() || canPermBanViewer()) {
    durationOptions.push('<label><input type="radio" name="mod-ban-duration" value="permanent" /> Permanent</label>');
  }

  return `
    <dl class="mod-panel-dl">
      <dt>Post ID</dt><dd><code>${audit.post_id}</code></dd>
      <dt>Author</dt><dd>${audit.author_name}</dd>
      <dt>IP address</dt><dd><code>${ip}</code></dd>
      <dt>Posted</dt><dd>${new Date(audit.created_at).toLocaleString()}</dd>
      <dt>Edited</dt><dd>${audit.edited_at ? new Date(audit.edited_at).toLocaleString() : '—'}</dd>
      <dt>Revisions</dt><dd>${audit.revision_count}</dd>
      <dt>Likes</dt><dd>${audit.reaction_count}</dd>
      <dt>Thread</dt><dd>${audit.thread_title}</dd>
    </dl>
    <div class="mod-panel-actions">
      <button type="button" class="btn btn--sm btn--ghost" data-mod-remove="${audit.post_id}">Remove this post</button>
      <button type="button" class="btn btn--sm" data-mod-takedown="${audit.author_id}" data-duration="7d">
        Takedown (7 days)
      </button>
      ${
        isAdminViewer() || canPermBanViewer()
          ? `<button type="button" class="btn btn--sm btn--pink" data-mod-takedown="${audit.author_id}" data-duration="permanent">
        Permanent takedown
      </button>`
          : ''
      }
    </div>
    <div class="mod-panel-ban">
      <p class="form-hint">Ban the author (account + known IPs). Permanent options require the perm-ban permission.</p>
      <fieldset class="mod-ban-durations">${durationOptions.join('')}</fieldset>
      <label class="field-label">
        <input type="checkbox" id="mod-ban-purge" checked />
        Remove all their posts
      </label>
      <label class="field-label">
        Message to member
        <textarea id="mod-ban-reason" rows="3" placeholder="Shown on their banned screen"></textarea>
      </label>
      <button type="button" class="btn btn--sm btn--pink" data-mod-ban="${audit.author_id}">Ban author</button>
    </div>
  `;
}

function bindModPanelActions(panelBody: HTMLElement, openPanel: PanelFn) {
  panelBody.querySelector('[data-mod-remove]')?.addEventListener('click', async (e) => {
    const btn = e.currentTarget as HTMLButtonElement;
    const postId = btn.getAttribute('data-mod-remove');
    if (!postId) return;
    if (!(await confirm('Remove this post from the thread?'))) return;
    btn.disabled = true;
    try {
      const res = await fetch(`/api/posts/${postId}/remove`, { method: 'POST' });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Remove failed');
      removePostArticle(postId);
      if (data.thread_removed) {
        showToast('Post and spam thread removed', 'warning');
        window.setTimeout(() => {
          window.location.href = '/community';
        }, 600);
      } else {
        showToast('Post removed', 'warning');
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Remove failed', 'error');
      btn.disabled = false;
    }
  });

  panelBody.querySelectorAll('[data-mod-takedown]').forEach((el) => {
    el.addEventListener('click', async (e) => {
      const btn = e.currentTarget as HTMLButtonElement;
      const authorId = btn.getAttribute('data-mod-takedown');
      const duration = (btn.getAttribute('data-duration') ?? '7d') as '7d' | 'permanent';
      if (!authorId) return;
      const isPermanent = duration === 'permanent';
      const message = await promptDialog(
        isPermanent
          ? 'Permanent ban + remove all their posts. Use for spam, phishing, or porn hit-and-runs.'
          : '7-day ban + remove all their posts.',
        {
          title: isPermanent ? 'Permanent takedown' : 'Takedown author (7 days)',
          placeholder: 'Message to member (required)',
          defaultValue: 'Your account was removed for posting prohibited content.',
        },
      );
      if (message === null || !message.trim()) return;
      btn.disabled = true;
      panelBody.querySelectorAll('[data-mod-takedown]').forEach((b) => {
        (b as HTMLButtonElement).disabled = true;
      });
      try {
        const data = await staffBanAuthor(authorId, message.trim(), duration, true);
        const n = data.posts_removed ?? 0;
        showToast(
          isPermanent
            ? `Permanent takedown — ${n} post(s) removed`
            : `Takedown complete — banned 7 days, ${n} post(s) removed`,
          'warning',
        );
        window.setTimeout(() => window.location.reload(), 800);
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Takedown failed', 'error');
        panelBody.querySelectorAll('[data-mod-takedown]').forEach((b) => {
          (b as HTMLButtonElement).disabled = false;
        });
      }
    });
  });

  panelBody.querySelector('[data-mod-ban]')?.addEventListener('click', async (e) => {
    const btn = e.currentTarget as HTMLButtonElement;
    const authorId = btn.getAttribute('data-mod-ban');
    if (!authorId) return;
    const reason = (panelBody.querySelector('#mod-ban-reason') as HTMLTextAreaElement)?.value.trim();
    if (!reason) {
      showToast('Ban message is required', 'warning');
      return;
    }
    const duration =
      (panelBody.querySelector('input[name="mod-ban-duration"]:checked') as HTMLInputElement)?.value ?? '7d';
    const purgeContent = (panelBody.querySelector('#mod-ban-purge') as HTMLInputElement)?.checked ?? false;
    if (
      !(await confirm(
        purgeContent
          ? `Ban for ${duration} and remove all their posts?`
          : `Ban this member for ${duration}?`,
      ))
    ) {
      return;
    }
    btn.disabled = true;
    try {
      const data = await staffBanAuthor(
        authorId,
        reason,
        duration as '7d' | '30d' | 'permanent',
        purgeContent,
      );
      const purgeNote = purgeContent && data.posts_removed != null ? `, ${data.posts_removed} post(s) removed` : '';
      showToast(`Member banned (${duration})${purgeNote}`, 'warning');
      window.setTimeout(() => window.location.reload(), 800);
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Ban failed', 'error');
      btn.disabled = false;
    }
  });
}

export function initThreadInteractions(openPanel: PanelFn, closePanel: ClosePanelFn) {
  const postsRoot = document.getElementById('thread-posts');
  if (postsRoot) {
    initLightbox(postsRoot);
  }

  postsRoot?.addEventListener('click', async (e) => {
    const target = e.target as HTMLElement;

    const editOpenBtn = target.closest('[data-edit]') as HTMLButtonElement | null;
    if (editOpenBtn) {
      const id = editOpenBtn.getAttribute('data-edit');
      if (id) openPostEditor(id);
      return;
    }

    const cancelBtn = target.closest('.post-edit-cancel') as HTMLButtonElement | null;
    if (cancelBtn) {
      const form = cancelBtn.closest('.post-edit') as HTMLElement | null;
      const id = form?.getAttribute('data-post-edit');
      const body = document.querySelector<HTMLElement>(`[data-post-body="${id}"]`);
      const editor = form ? findPostEditor(form) : null;
      const status = form?.querySelector<HTMLElement>('.post-edit-status');
      if (editor) clearEditorAttachments(editor);
      if (body) body.hidden = false;
      if (form) form.hidden = true;
      if (status) status.hidden = true;
      return;
    }

    const deleteBtn = target.closest('[data-delete]') as HTMLButtonElement | null;
    if (deleteBtn) {
      const postId = deleteBtn.getAttribute('data-delete');
      if (!postId) return;
      const ok = await confirm('This post will be hidden from the thread.', {
        title: 'Delete post?',
        confirmLabel: 'Delete',
        dangerous: true,
      });
      if (!ok) return;
      deleteBtn.setAttribute('disabled', 'true');
      try {
        const res = await fetch(`/api/posts/${postId}`, { method: 'DELETE' });
        if (!res.ok) throw new Error((await res.json()).error);
        removePostArticle(postId);
        showToast('Post deleted', 'success');
      } catch {
        showToast('Could not delete post', 'error');
        deleteBtn.removeAttribute('disabled');
      }
      return;
    }

    const historyBtn = target.closest('[data-history]') as HTMLButtonElement | null;
    if (historyBtn) {
      const postId = historyBtn.getAttribute('data-history');
      if (!postId) return;
      openPanel('Edit history', '<p>Loading…</p>');
      try {
        const res = await fetch(`/api/posts/${postId}/revisions`);
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed');
        const items = (data.revisions || [])
          .map(
            (r: { revision: number; editor_name: string; edited_at: string; body_html: string }) => `
            <div class="revision-item">
              <div class="revision-meta">#${r.revision} · ${r.editor_name} · ${new Date(r.edited_at).toLocaleString()}</div>
              <div class="revision-body">${r.body_html}</div>
            </div>
          `,
          )
          .join('');
        openPanel('Edit history', items || '<p>No revisions stored.</p>');
      } catch (err) {
        openPanel('Edit history', `<p>${err instanceof Error ? err.message : 'Failed'}</p>`);
      }
      return;
    }

    const reportsBtn = target.closest('[data-thread-reports]') as HTMLButtonElement | null;
    if (reportsBtn) {
      closeAllModPopovers();
      const bar = reportsBtn.closest('[data-thread-id]');
      const threadId = bar?.getAttribute('data-thread-id');
      if (!threadId) return;
      openPanel('Thread reports', '<p>Loading…</p>');
      try {
        const res = await fetch(`/api/admin/reports?thread_id=${encodeURIComponent(threadId)}`);
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed');
        const items = (data.reports || []) as Array<{
          id: string;
          kind?: string;
          reporter_name: string;
          reason: string;
          created_at: string;
          post_author?: string;
          post_excerpt?: string;
          post_url?: string;
          thread_url: string;
          thread_title: string;
        }>;
        if (!items.length) {
          openPanel('Thread reports', '<p>No open reports on this thread.</p>');
          return;
        }
        const html = items
          .map(
            (r) => `
            <div class="mod-report-panel-item" data-report-id="${r.id}">
              <div class="mod-report-panel-meta">${r.reporter_name} · ${new Date(r.created_at).toLocaleString()}${r.kind === 'thread' ? ' · thread' : ''}</div>
              ${r.kind === 'thread'
                ? `<p class="mod-report-panel-excerpt"><a href="${r.thread_url}"><strong>Thread:</strong> ${r.thread_title}</a></p>`
                : `<p class="mod-report-panel-excerpt"><a href="${r.post_url}"><strong>${r.post_author}</strong>: ${r.post_excerpt}</a></p>`}
              ${r.reason ? `<blockquote class="mod-report-panel-reason">${r.reason}</blockquote>` : ''}
              <div class="mod-report-panel-actions">
                <a class="btn btn--sm" href="${r.kind === 'thread' ? r.thread_url : r.post_url}">${r.kind === 'thread' ? 'View thread' : 'View post'}</a>
                <button type="button" class="btn btn--sm btn--ghost" data-dismiss-report="${r.id}">Dismiss</button>
              </div>
            </div>
          `,
          )
          .join('');
        openPanel('Thread reports', html);
      } catch (err) {
        openPanel('Thread reports', `<p>${err instanceof Error ? err.message : 'Failed'}</p>`);
      }
      return;
    }

    const dismissReportBtn = target.closest('[data-dismiss-report]') as HTMLButtonElement | null;
    if (dismissReportBtn) {
      const reportId = dismissReportBtn.getAttribute('data-dismiss-report');
      if (!reportId) return;
      dismissReportBtn.setAttribute('disabled', 'true');
      try {
        const res = await fetch(`/api/admin/reports/${reportId}/dismiss`, { method: 'POST' });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Dismiss failed');
        const item = dismissReportBtn.closest('[data-report-id]');
        item?.remove();
        const count = document.querySelectorAll('[data-report-id]').length;
        syncThreadReportCount(count);
        const panelBody = document.getElementById('mod-panel-body');
        if (panelBody && !panelBody.querySelector('[data-report-id]')) {
          panelBody.innerHTML = '<p>No open reports on this thread.</p>';
        }
        showToast('Report dismissed', 'success');
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Dismiss failed', 'error');
        dismissReportBtn.removeAttribute('disabled');
      }
      return;
    }

    const warnBtn = target.closest('[data-warn]') as HTMLButtonElement | null;
    if (warnBtn && !warnBtn.disabled) {
      closeAllModPopovers();
      const postId = warnBtn.getAttribute('data-warn');
      if (!postId) return;
      const message = await promptDialog(
        'They will see a red border + banner for 8 hours. Same member within 24h counts as one strike. Each post can only be warned once.',
        {
          title: 'Warn about this post',
          placeholder: 'What they did wrong (required)',
          defaultValue: '',
        },
      );
      if (message === null || !message.trim()) return;
      warnBtn.setAttribute('disabled', 'true');
      try {
        const res = await fetch(`/api/posts/${postId}/warn`, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ message: message.trim() }),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || `Warning failed (${res.status})`);
        const warnLabel = warnBtn.querySelector('.mod-popover-label');
        if (warnLabel) warnLabel.textContent = 'Already warned';
        else warnBtn.textContent = 'Warned';
        const warnedPost = warnBtn.closest('article.post');
        const authorName = warnedPost?.querySelector('.pname a')?.textContent?.trim();
        if (authorName) {
          document.querySelectorAll('#thread-posts article.post').forEach((article) => {
            const name = article.querySelector('.pname a')?.textContent?.trim();
            if (name === authorName) {
              article.querySelector('.pav')?.classList.add('pav--warned');
            }
          });
        }
        if (data.consolidated) {
          showToast('Added to current 24h warning — no extra strike', 'warning');
        } else if (data.auto_banned) {
          showToast('Third strike — member banned 7 days', 'error');
        } else {
          showToast(`Warning ${data.warning_count} of 3 issued`, 'warning');
        }
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Warning failed', 'error');
        warnBtn.removeAttribute('disabled');
      }
      return;
    }

    const removeBtn = target.closest('[data-remove]') as HTMLButtonElement | null;
    if (removeBtn && isStaffViewer()) {
      closeAllModPopovers();
      const postId = removeBtn.getAttribute('data-remove');
      if (!postId) return;
      if (!(await confirm('Remove this post from the thread?'))) return;
      removeBtn.disabled = true;
      try {
        const res = await fetch(`/api/posts/${postId}/remove`, { method: 'POST' });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Remove failed');
        removePostArticle(postId);
        if (data.thread_removed) {
          showToast('Post and spam thread removed', 'warning');
          window.setTimeout(() => {
            window.location.href = '/community';
          }, 600);
        } else {
          showToast('Post removed', 'warning');
        }
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Remove failed', 'error');
        removeBtn.disabled = false;
      }
      return;
    }

    const modBtn = target.closest('[data-mod]') as HTMLButtonElement | null;
    if (modBtn && isStaffViewer()) {
      closeAllModPopovers();
      const postId = modBtn.getAttribute('data-mod');
      if (!postId) return;
      openPanel('Moderator tools', '<p>Loading…</p>');
      try {
        const res = await fetch(`/api/admin/posts/${postId}`);
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed');
        const html = renderModAuditPanel(data);
        openPanel('Moderator tools', html);
        const panelBody = document.getElementById('mod-panel-body');
        if (panelBody) bindModPanelActions(panelBody, openPanel);
      } catch (err) {
        openPanel('Moderator tools', `<p>${err instanceof Error ? err.message : 'Failed'}</p>`);
      }
      return;
    }

    const threadReportBtn = target.closest('[data-thread-report]') as HTMLButtonElement | null;
    if (threadReportBtn && !threadReportBtn.disabled) {
      const threadId = threadReportBtn.getAttribute('data-thread-report');
      if (!threadId) return;
      const picked = await pickReportReason(openPanel, closePanel);
      if (picked === null) return;
      try {
        const payload = picked.reason_id
          ? { reason_id: picked.reason_id, detail: picked.detail }
          : { reason: picked.detail };
        const res = await fetch(`/api/threads/${threadId}/report`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Report failed');
        markReported(threadReportBtn, true);
        showToast('Thread flagged for moderators — thanks.', 'success');
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Report failed', 'error');
      }
      return;
    }

    const reportBtn = target.closest('[data-report]') as HTMLButtonElement | null;
    if (reportBtn && !reportBtn.disabled) {
      const postId = reportBtn.getAttribute('data-report');
      if (!postId) return;
      const picked = await pickReportReason(openPanel, closePanel);
      if (picked === null) return;
      try {
        const payload = picked.reason_id
          ? { reason_id: picked.reason_id, detail: picked.detail }
          : { reason: picked.detail };
        const res = await fetch(`/api/posts/${postId}/report`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Report failed');
        markPostReported(reportBtn);
        showToast('Flagged for moderators — thanks.', 'success');
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Report failed', 'error');
        reportBtn.removeAttribute('disabled');
      }
      return;
    }

    const likeBtn = target.closest('.post-like') as HTMLButtonElement | null;
    if (likeBtn && !likeBtn.disabled) {
      const postId = likeBtn.getAttribute('data-post-id');
      if (!postId) return;
      likeBtn.setAttribute('disabled', 'true');
      try {
        const res = await fetch(`/api/posts/${postId}/reactions`, { method: 'POST' });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Like failed');
        const countEl = likeBtn.querySelector('.like-count');
        if (countEl) countEl.textContent = String(data.count);
        likeBtn.classList.toggle('is-liked', data.liked);
        likeBtn.setAttribute('data-liked', data.liked ? '1' : '0');
      } catch {
        /* unchanged */
      } finally {
        likeBtn.removeAttribute('disabled');
      }
    }
  });

  postsRoot?.addEventListener('submit', async (e) => {
    const form = (e.target as HTMLElement).closest('.post-edit') as HTMLFormElement | null;
    if (!form) return;
    e.preventDefault();

    const id = form.getAttribute('data-post-edit');
    const editor = findPostEditor(form);
    const status = form.querySelector<HTMLElement>('.post-edit-status');
    const saveBtn = form.querySelector<HTMLButtonElement>('[type="submit"]');
    if (!id || !editor) return;
    if (!hasEditorContent(editor)) {
      showToast('Write something before saving', 'warning');
      return;
    }

    saveBtn?.setAttribute('disabled', 'true');
    if (status) status.hidden = true;

    let text: string;
    try {
      text = (await prepareEditorMarkdown(editor)).trim();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Image upload failed', 'error');
      saveBtn?.removeAttribute('disabled');
      return;
    }
    if (!text) {
      showToast('Write something before saving', 'warning');
      saveBtn?.removeAttribute('disabled');
      return;
    }

    try {
      const res = await fetch(`/api/posts/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body_markdown: text }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Edit failed');

      const post = data.post as Post | undefined;
      if (post) {
        applyPostEdit(id, post, { isAdmin: isAdminViewer() });
        initLightbox(document.getElementById(`post-${id}`) ?? document);
        showToast('Post updated', 'success');
      } else {
        window.location.reload();
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Edit failed', 'error');
      if (status) {
        status.textContent = err instanceof Error ? err.message : 'Edit failed';
        status.hidden = false;
      }
    } finally {
      saveBtn?.removeAttribute('disabled');
    }
  });

  document.querySelector('[data-thread-pin]')?.addEventListener('click', async (e) => {
    closeAllModPopovers();
    const btn = e.currentTarget as HTMLButtonElement;
    const bar = btn.closest('[data-thread-id]');
    const threadId = bar?.getAttribute('data-thread-id');
    const pinned = btn.getAttribute('data-pinned') === '1';
    if (!threadId) return;
    btn.setAttribute('disabled', 'true');
    try {
      const res = await fetch(`/api/threads/${threadId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_pinned: !pinned }),
      });
      if (!res.ok) throw new Error((await res.json()).error);
      const nowPinned = !pinned;
      syncThreadPinLabel(nowPinned);
      setThreadPinned(nowPinned);
      showToast(nowPinned ? 'Thread pinned' : 'Thread unpinned', 'success');
    } catch {
      showToast('Could not update pin', 'error');
    } finally {
      btn.removeAttribute('disabled');
    }
  });

  document.querySelector('[data-thread-delete]')?.addEventListener('click', async (e) => {
    closeAllModPopovers();
    const btn = e.currentTarget as HTMLButtonElement;
    const bar = btn.closest('[data-thread-id]');
    const threadId = bar?.getAttribute('data-thread-id');
    if (!threadId) return;
    const ok = await confirm('This removes the entire thread from the forum.', {
      title: 'Delete thread?',
      confirmLabel: 'Delete thread',
      dangerous: true,
    });
    if (!ok) return;
    btn.setAttribute('disabled', 'true');
    try {
      const res = await fetch(`/api/threads/${threadId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error((await res.json()).error);
      showToast('Thread deleted', 'warning');
      window.setTimeout(() => {
        window.location.href = '/community';
      }, 600);
    } catch {
      showToast('Could not delete thread', 'error');
      btn.removeAttribute('disabled');
    }
  });

  document.querySelector('[data-thread-lock]')?.addEventListener('click', async (e) => {
    closeAllModPopovers();
    const btn = e.currentTarget as HTMLButtonElement;
    const bar = btn.closest('[data-thread-id]');
    const threadId = bar?.getAttribute('data-thread-id');
    const locked = btn.getAttribute('data-locked') === '1';
    if (!threadId) return;
    btn.setAttribute('disabled', 'true');
    try {
      const res = await fetch(`/api/threads/${threadId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_locked: !locked }),
      });
      if (!res.ok) throw new Error((await res.json()).error);
      const nowLocked = !locked;
      syncThreadLockLabel(nowLocked);
      setThreadLocked(nowLocked);
      showToast(nowLocked ? 'Thread locked' : 'Thread unlocked', 'success');
    } catch {
      showToast('Could not update lock', 'error');
    } finally {
      btn.removeAttribute('disabled');
    }
  });
}

/** Bind lightbox for a newly inserted post. */
export function bindNewPost(article: HTMLElement) {
  initLightbox(article);
}