import { insertDMBubble, type DMMessagePayload } from './dm-message';
import { bindRichTextToolbars, clearEditorAttachments, hasEditorContent, prepareEditorMarkdown } from './rich-text-editor';
import { showToast } from './toast';

export function initDMCompose(root: HTMLElement) {
  const convId = root.dataset.convId;
  const bodyId = root.dataset.bodyId ?? 'dm-body';
  const form = root.matches('form') ? root : root.querySelector('form');
  const list = document.getElementById('dm-messages');
  const body = document.getElementById(bodyId);
  const submitBtn = root.querySelector<HTMLButtonElement>('[data-dm-submit]');
  if (!convId || !form || !list || !(body instanceof HTMLElement) || !submitBtn) return;

  bindRichTextToolbars(root);

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!hasEditorContent(body)) return;

    submitBtn.disabled = true;
    let markdown: string;
    try {
      markdown = await prepareEditorMarkdown(body);
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Image upload failed', 'error');
      submitBtn.disabled = false;
      return;
    }
    if (!markdown) {
      submitBtn.disabled = false;
      return;
    }

    try {
      const res = await fetch(`/api/dm/conversations/${convId}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body_markdown: markdown }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Could not send message');

      list.querySelector('.mod-empty')?.remove();

      const msg: DMMessagePayload = {
        id: data.id,
        body_html: data.body_html,
        created_at: data.created_at,
        is_mine: true,
      };
      const article = insertDMBubble(list, msg, { justNow: true });
      article.scrollIntoView({ behavior: 'smooth', block: 'end' });

      body.innerHTML = '';
      clearEditorAttachments(body);
      body.focus();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Send failed', 'error');
    } finally {
      submitBtn.disabled = false;
    }
  });
}