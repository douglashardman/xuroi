import TurndownService from 'turndown';
import { promptDialog, showToast } from './toast';

const turndown = new TurndownService({
  headingStyle: 'atx',
  bulletListMarker: '-',
  codeBlockStyle: 'fenced',
});

turndown.addRule('underline', {
  filter: ['u'],
  replacement: (content) => content,
});

turndown.addRule('image', {
  filter: 'img',
  replacement: (_content, node) => {
    const el = node as HTMLImageElement;
    const alt = el.getAttribute('alt') || '';
    const src = el.getAttribute('src') || '';
    return src ? `![${alt}](${src})` : '';
  },
});

export type RtAction = 'bold' | 'italic' | 'link' | 'ul' | 'ol' | 'quote' | 'code' | 'image';

const MAX_ATTACHMENTS = 10;

type LocalAttachment = {
  id: string;
  file: File;
  previewUrl: string;
  alt: string;
};

const savedRanges = new WeakMap<HTMLElement, Range>();
const attachmentLists = new WeakMap<HTMLElement, LocalAttachment[]>();

function fileAlt(file: File): string {
  return file.name.replace(/\.[^.]+$/, '').replace(/[-_]+/g, ' ');
}

function attachmentsHost(editor: HTMLElement): HTMLElement | null {
  const id = editor.id;
  if (!id) return null;
  return document.querySelector(`[data-rt-attachments="${id}"]`);
}

function updateAttachmentLabel(host: HTMLElement, count: number) {
  const label = host.querySelector('.rt-attachments-label');
  if (!label) return;
  label.textContent = count === 1 ? '1 image attached' : `${count} images attached`;
}

function renderAttachments(editor: HTMLElement) {
  const host = attachmentsHost(editor);
  if (!host) return;

  const list = attachmentLists.get(editor) ?? [];
  const listEl = host.querySelector('.rt-attachments-list');
  if (!listEl) return;

  listEl.innerHTML = '';
  for (const att of list) {
    const item = document.createElement('div');
    item.className = 'rt-attachment';
    item.dataset.attachmentId = att.id;

    const img = document.createElement('img');
    img.src = att.previewUrl;
    img.alt = att.alt;
    img.className = 'rt-attachment-thumb';

    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'rt-attachment-remove';
    remove.setAttribute('aria-label', `Remove ${att.alt}`);
    remove.textContent = '×';
    remove.addEventListener('click', () => removeAttachment(editor, att.id));

    item.append(img, remove);
    listEl.append(item);
  }

  host.hidden = list.length === 0;
  if (list.length > 0) updateAttachmentLabel(host, list.length);
}

function addAttachments(editor: HTMLElement, files: File[]) {
  const list = attachmentLists.get(editor) ?? [];
  const room = MAX_ATTACHMENTS - list.length;
  if (room <= 0) {
    showToast(`Maximum ${MAX_ATTACHMENTS} images per post`, 'warning');
    return;
  }

  const accepted = files.filter((f) => f.type.startsWith('image/')).slice(0, room);
  if (accepted.length < files.filter((f) => f.type.startsWith('image/')).length) {
    showToast(`Only ${MAX_ATTACHMENTS} images allowed — added ${accepted.length}`, 'warning');
  }

  for (const file of accepted) {
    list.push({
      id: crypto.randomUUID(),
      file,
      previewUrl: URL.createObjectURL(file),
      alt: fileAlt(file),
    });
  }

  attachmentLists.set(editor, list);
  renderAttachments(editor);
}

function removeAttachment(editor: HTMLElement, id: string) {
  const list = attachmentLists.get(editor) ?? [];
  const next = list.filter((att) => {
    if (att.id === id) URL.revokeObjectURL(att.previewUrl);
    return att.id !== id;
  });
  attachmentLists.set(editor, next);
  renderAttachments(editor);
}

function imageFilesFromClipboard(items: DataTransferItemList): File[] {
  const files: File[] = [];
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      const file = item.getAsFile();
      if (file) files.push(file);
    }
  }
  return files;
}

function bindImageDropZone(editor: HTMLElement) {
  const host = attachmentsHost(editor);
  const zones = [editor, host].filter((el): el is HTMLElement => el instanceof HTMLElement);

  for (const zone of zones) {
    zone.addEventListener('dragover', (e) => {
      if (![...e.dataTransfer?.types ?? []].includes('Files')) return;
      e.preventDefault();
      zone.classList.add('rt-drop-active');
    });
    zone.addEventListener('dragleave', () => zone.classList.remove('rt-drop-active'));
    zone.addEventListener('drop', (e) => {
      e.preventDefault();
      zone.classList.remove('rt-drop-active');
      const files = [...e.dataTransfer?.files ?? []];
      if (files.length) addAttachments(editor, files);
    });
  }
}

async function uploadFile(file: File): Promise<{ url: string; alt: string }> {
  const form = new FormData();
  form.append('file', file);
  const res = await fetch('/api/media/upload', { method: 'POST', body: form });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || 'Upload failed');
  }
  return { url: data.url, alt: fileAlt(file) };
}

function saveSelection(editor: HTMLElement) {
  const sel = window.getSelection();
  if (!sel || sel.rangeCount === 0) return;
  const range = sel.getRangeAt(0);
  if (editor.contains(range.commonAncestorContainer)) {
    savedRanges.set(editor, range.cloneRange());
  }
}

function restoreSelection(editor: HTMLElement) {
  const range = savedRanges.get(editor);
  if (!range) return;
  const sel = window.getSelection();
  if (!sel) return;
  sel.removeAllRanges();
  sel.addRange(range);
}

function currentRange(): Range | null {
  const sel = window.getSelection();
  if (!sel || sel.rangeCount === 0) return null;
  return sel.getRangeAt(0);
}

const INLINE_WRAPPER_TAGS = new Set(['B', 'STRONG', 'I', 'EM', 'CODE', 'U', 'A']);

/** After formatting a selection, park the caret outside the wrapper so new typing stays plain. */
function moveCaretPastInlineFormat(editor: HTMLElement) {
  const sel = window.getSelection();
  if (!sel || sel.rangeCount === 0) return;

  const range = sel.getRangeAt(0);
  range.collapse(false);

  let node: Node | null = range.startContainer;
  if (node.nodeType === Node.TEXT_NODE) node = node.parentElement;

  while (node instanceof HTMLElement && node !== editor && INLINE_WRAPPER_TAGS.has(node.tagName)) {
    const after = document.createRange();
    after.setStartAfter(node);
    after.collapse(true);
    sel.removeAllRanges();
    sel.addRange(after);
    savedRanges.set(editor, after.cloneRange());
    return;
  }

  sel.removeAllRanges();
  sel.addRange(range);
  if (document.queryCommandState('bold')) document.execCommand('bold');
  if (document.queryCommandState('italic')) document.execCommand('italic');
  savedRanges.set(editor, range.cloneRange());
}

function closestAncestor(node: Node | null, editor: HTMLElement, tag: string): HTMLElement | null {
  let el: Element | null =
    node?.nodeType === Node.TEXT_NODE ? (node.parentElement as Element | null) : (node as Element | null);
  while (el && el !== editor) {
    if (el.tagName === tag) return el as HTMLElement;
    el = el.parentElement;
  }
  return null;
}

function selectionInsideTag(editor: HTMLElement, tag: string): boolean {
  const sel = window.getSelection();
  if (!sel?.anchorNode || !editor.contains(sel.anchorNode)) return false;
  return closestAncestor(sel.anchorNode, editor, tag) !== null;
}

function unwrapBlockquote(editor: HTMLElement) {
  const sel = window.getSelection();
  if (!sel?.anchorNode) return;
  const blockquote = closestAncestor(sel.anchorNode, editor, 'BLOCKQUOTE');
  if (!blockquote?.parentNode) return;

  const p = document.createElement('p');
  while (blockquote.firstChild) {
    p.appendChild(blockquote.firstChild);
  }
  blockquote.parentNode.replaceChild(p, blockquote);

  const range = document.createRange();
  range.selectNodeContents(p);
  range.collapse(false);
  sel.removeAllRanges();
  sel.addRange(range);
  savedRanges.set(editor, range.cloneRange());
}

function toggleQuote(editor: HTMLElement) {
  if (selectionInsideTag(editor, 'BLOCKQUOTE')) {
    unwrapBlockquote(editor);
  } else {
    document.execCommand('formatBlock', false, 'blockquote');
  }
}

function unwrapCode(editor: HTMLElement) {
  const sel = window.getSelection();
  if (!sel?.anchorNode) return;

  const codes = new Set<HTMLElement>();
  const start = closestAncestor(sel.anchorNode, editor, 'CODE');
  const end = closestAncestor(sel.focusNode, editor, 'CODE');
  if (start) codes.add(start);
  if (end) codes.add(end);

  for (const code of codes) {
    const parent = code.parentNode;
    if (!parent) continue;
    const frag = document.createDocumentFragment();
    while (code.firstChild) {
      frag.appendChild(code.firstChild);
    }
    parent.insertBefore(frag, code);
    parent.removeChild(code);
    parent.normalize();
  }
}

function updateToolbarState(toolbar: HTMLElement, editor: HTMLElement) {
  const sel = window.getSelection();
  if (!sel?.anchorNode || !editor.contains(sel.anchorNode)) return;

  const block = document.queryCommandValue('formatBlock').replace(/[<>]/g, '').toLowerCase();

  toolbar.querySelectorAll<HTMLElement>('[data-rt-action]').forEach((btn) => {
    const action = btn.getAttribute('data-rt-action');
    let active = false;
    switch (action) {
      case 'bold':
        active = document.queryCommandState('bold');
        break;
      case 'italic':
        active = document.queryCommandState('italic');
        break;
      case 'ul':
        active = document.queryCommandState('insertUnorderedList');
        break;
      case 'ol':
        active = document.queryCommandState('insertOrderedList');
        break;
      case 'quote':
        active = selectionInsideTag(editor, 'BLOCKQUOTE') || block === 'blockquote';
        break;
      case 'code':
        active = selectionInsideTag(editor, 'CODE');
        break;
      default:
        return;
    }
    btn.classList.toggle('is-active', active);
    btn.setAttribute('aria-pressed', active ? 'true' : 'false');
  });
}

function applyInlineCode(editor: HTMLElement): boolean {
  const sel = window.getSelection();
  if (!sel || sel.rangeCount === 0) return false;
  const range = sel.getRangeAt(0);
  const hadRange = !range.collapsed;
  if (range.collapsed) {
    const code = document.createElement('code');
    code.textContent = 'code';
    range.insertNode(code);
    sel.removeAllRanges();
    const r = document.createRange();
    r.selectNodeContents(code);
    sel.addRange(r);
    return false;
  }
  const code = document.createElement('code');
  try {
    range.surroundContents(code);
  } catch {
    document.execCommand('insertHTML', false, `<code>${range.toString()}</code>`);
  }
  return hadRange;
}

async function applyFormat(editor: HTMLElement, action: RtAction, toolbar?: HTMLElement) {
  editor.focus();
  restoreSelection(editor);
  const range = currentRange();
  const hadRange = !!range && !range.collapsed;

  switch (action) {
    case 'bold':
      document.execCommand('bold');
      if (hadRange) moveCaretPastInlineFormat(editor);
      break;
    case 'italic':
      document.execCommand('italic');
      if (hadRange) moveCaretPastInlineFormat(editor);
      break;
    case 'link': {
      const url = await promptDialog('Paste or type the destination URL.', {
        title: 'Insert link',
        defaultValue: 'https://',
        placeholder: 'https://example.com',
      });
      if (url) {
        document.execCommand('createLink', false, url);
        if (hadRange) moveCaretPastInlineFormat(editor);
      }
      break;
    }
    case 'ul':
      document.execCommand('insertUnorderedList');
      break;
    case 'ol':
      document.execCommand('insertOrderedList');
      break;
    case 'quote':
      toggleQuote(editor);
      break;
    case 'code':
      if (selectionInsideTag(editor, 'CODE')) {
        unwrapCode(editor);
      } else if (applyInlineCode(editor)) {
        moveCaretPastInlineFormat(editor);
      }
      break;
  }
  saveSelection(editor);
  if (toolbar) updateToolbarState(toolbar, editor);
}

export function bindRichTextToolbars(root: ParentNode = document) {
  root.querySelectorAll('[data-rt-target]:not([data-rt-bound])').forEach((toolbar) => {
    toolbar.setAttribute('data-rt-bound', '1');
    const targetId = toolbar.getAttribute('data-rt-target');
    const editor = targetId ? document.getElementById(targetId) : null;
    if (!editor || !editor.isContentEditable) return;

    const imageInput = toolbar.querySelector<HTMLInputElement>(`[data-rt-image-input="${targetId}"]`);

    const syncToolbar = () => {
      saveSelection(editor);
      updateToolbarState(toolbar as HTMLElement, editor);
    };

    editor.addEventListener('keyup', syncToolbar);
    editor.addEventListener('mouseup', syncToolbar);
    editor.addEventListener('focus', syncToolbar);

    document.addEventListener('selectionchange', () => {
      const sel = window.getSelection();
      if (!sel?.anchorNode || !editor.contains(sel.anchorNode)) return;
      updateToolbarState(toolbar as HTMLElement, editor);
    });

    bindImageDropZone(editor);

    toolbar.querySelectorAll('[data-rt-action]').forEach((btn) => {
      btn.addEventListener('mousedown', (e) => {
        e.preventDefault();
        saveSelection(editor);
      });
      btn.addEventListener('click', async (e) => {
        e.preventDefault();
        const action = btn.getAttribute('data-rt-action') as RtAction | null;
        if (action === 'image') {
          imageInput?.click();
          return;
        }
        if (action) await applyFormat(editor, action, toolbar as HTMLElement);
      });
    });

    imageInput?.addEventListener('change', () => {
      const files = [...(imageInput.files ?? [])];
      imageInput.value = '';
      if (!files.length) return;
      addAttachments(editor, files);
    });

    editor.addEventListener('keydown', (e) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key === 'b' || e.key === 'B') {
        e.preventDefault();
        void applyFormat(editor, 'bold', toolbar as HTMLElement);
      }
      if (e.key === 'i' || e.key === 'I') {
        e.preventDefault();
        void applyFormat(editor, 'italic', toolbar as HTMLElement);
      }
    });

    editor.addEventListener('paste', (e) => {
      const items = e.clipboardData?.items;
      if (items) {
        const images = imageFilesFromClipboard(items);
        if (images.length) {
          e.preventDefault();
          addAttachments(editor, images);
          return;
        }
      }
      e.preventDefault();
      const text = e.clipboardData?.getData('text/plain') ?? '';
      document.execCommand('insertText', false, text);
    });
  });
}

export function getEditorMarkdown(editor: HTMLElement): string {
  const html = editor.innerHTML.trim();
  if (!html || html === '<br>') return '';
  return turndown.turndown(html).trim();
}

export function getEditorPlainText(editor: HTMLElement): string {
  return (editor.innerText || '').replace(/\u00a0/g, ' ').trim();
}

export function hasEditorContent(editor: HTMLElement): boolean {
  const attachments = attachmentLists.get(editor) ?? [];
  return getEditorPlainText(editor) !== '' || attachments.length > 0;
}

/** @deprecated use hasEditorContent */
export function isEditorEmpty(editor: HTMLElement): boolean {
  return !hasEditorContent(editor);
}

/** Prepare stored post HTML for editing (unwrap galleries, restore full-res image URLs). */
export function normalizePostHtmlForEdit(html: string): string {
  const div = document.createElement('div');
  div.innerHTML = html;

  div.querySelectorAll<HTMLImageElement>('img[data-full-src]').forEach((img) => {
    const full = img.getAttribute('data-full-src');
    if (full) img.setAttribute('src', full);
    img.removeAttribute('data-full-src');
  });

  div.querySelectorAll('.post-gallery').forEach((gallery) => {
    const parent = gallery.parentNode;
    if (!parent) return;
    const imgs = [...gallery.querySelectorAll('img')];
    for (const img of imgs) {
      const p = document.createElement('p');
      p.appendChild(img.cloneNode(true));
      parent.insertBefore(p, gallery);
    }
    gallery.remove();
  });

  return div.innerHTML;
}

export function clearEditorAttachments(editor: HTMLElement) {
  const list = attachmentLists.get(editor) ?? [];
  for (const att of list) URL.revokeObjectURL(att.previewUrl);
  attachmentLists.set(editor, []);
  renderAttachments(editor);
}

export function setEditorHtml(editor: HTMLElement, html: string) {
  editor.innerHTML = html;
}

export async function prepareEditorMarkdown(editor: HTMLElement): Promise<string> {
  const text = getEditorMarkdown(editor);
  const attachments = [...(attachmentLists.get(editor) ?? [])];
  if (attachments.length === 0) return text;

  const uploads = await Promise.all(
    attachments.map(async (att) => {
      const { url } = await uploadFile(att.file);
      URL.revokeObjectURL(att.previewUrl);
      return `![${att.alt}](${url})`;
    }),
  );

  attachmentLists.set(editor, []);
  renderAttachments(editor);

  const images = uploads.join('\n\n');
  if (!text) return images;
  return `${text}\n\n${images}`;
}