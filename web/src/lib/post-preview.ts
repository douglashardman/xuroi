import { hasEditorContent, prepareEditorMarkdown } from './rich-text-editor';

export async function togglePostPreview(
  editor: HTMLElement,
  previewEl: HTMLElement,
  btn: HTMLButtonElement,
) {
  const showing = !previewEl.hidden;
  if (showing) {
    previewEl.hidden = true;
    editor.hidden = false;
    btn.textContent = 'Preview';
    return;
  }
  if (!hasEditorContent(editor)) {
    previewEl.innerHTML = '<p class="preview-empty">Nothing to preview yet.</p>';
    previewEl.hidden = false;
    editor.hidden = true;
    btn.textContent = 'Edit';
    return;
  }
  btn.disabled = true;
  btn.textContent = 'Loading…';
  try {
    const markdown = await prepareEditorMarkdown(editor);
    const res = await fetch('/api/preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body_markdown: markdown }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Preview failed');
    previewEl.innerHTML = `<div class="post-body preview-body">${data.body_html}</div>`;
    previewEl.hidden = false;
    editor.hidden = true;
    btn.textContent = 'Edit';
  } catch (err) {
    btn.textContent = 'Preview';
    throw err;
  } finally {
    btn.disabled = false;
  }
}