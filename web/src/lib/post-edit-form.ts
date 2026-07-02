/** Shared markup for inline post edit — keep in sync with PostEditForm.astro / FormatToolbar.astro */

export function postEditorId(postId: string): string {
  return `post-edit-${postId}`;
}

function formatToolbarHTML(editorId: string): string {
  return `
    <div class="rt-toolbar rt-toolbar--compact" data-rt-target="${editorId}" role="toolbar" aria-label="Formatting">
      <button type="button" class="rt-btn" data-rt-action="bold" title="Bold (Ctrl+B)"><strong>B</strong></button>
      <button type="button" class="rt-btn" data-rt-action="italic" title="Italic (Ctrl+I)"><em>I</em></button>
      <button type="button" class="rt-btn" data-rt-action="link" title="Link">Link</button>
      <span class="rt-toolbar-sep" aria-hidden="true"></span>
      <button type="button" class="rt-btn" data-rt-action="ul" title="Bullet list">• List</button>
      <button type="button" class="rt-btn" data-rt-action="ol" title="Numbered list">1. List</button>
      <button type="button" class="rt-btn" data-rt-action="quote" title="Quote block">“ Quote</button>
      <button type="button" class="rt-btn" data-rt-action="code" title="Code">&lt;/&gt;</button>
      <span class="rt-toolbar-sep" aria-hidden="true"></span>
      <button type="button" class="rt-btn" data-rt-action="image" title="Add images">Images</button>
      <input type="file" class="rt-image-input" accept="image/jpeg,image/png,image/gif,image/webp" multiple data-rt-image-input="${editorId}" hidden />
    </div>
  `;
}

export function postEditFormHTML(postId: string): string {
  const editorId = postEditorId(postId);
  return `
    <form class="post-edit" data-post-edit="${postId}" hidden>
      <div class="rt-editor">
        ${formatToolbarHTML(editorId)}
        <div class="rt-attachments" data-rt-attachments="${editorId}" hidden>
          <span class="rt-attachments-label">Attached images</span>
          <div class="rt-attachments-list"></div>
        </div>
        <div
          id="${editorId}"
          class="rt-editor-area"
          contenteditable="true"
          role="textbox"
          aria-multiline="true"
          data-placeholder="Edit your post…"
        ></div>
      </div>
      <div class="post-edit-actions">
        <button type="submit" class="btn btn--pink btn--sm">Save</button>
        <button type="button" class="post-edit-cancel">Cancel</button>
      </div>
      <p class="post-edit-status" hidden></p>
    </form>
  `;
}

export function findPostEditor(form: HTMLElement): HTMLElement | null {
  const editor = form.querySelector<HTMLElement>('.rt-editor-area');
  return editor;
}