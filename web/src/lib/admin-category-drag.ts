type ReorderItem = {
  category_id: string;
  sort_order: number;
  parent_id: string | null;
};

type DragKind = 'forum' | 'group';

let dragKind: DragKind | null = null;
let dragRow: HTMLElement | null = null;
let captureEl: HTMLElement | null = null;
let statusEl: HTMLElement | null = null;
let pointerId: number | null = null;

function getDragAfterElement(
  container: HTMLElement,
  y: number,
  rowSelector: string,
): HTMLElement | null {
  const rows = [...container.querySelectorAll<HTMLElement>(`:scope > ${rowSelector}:not(.dragging)`)];
  let closest: { offset: number; el: HTMLElement } | null = null;
  for (const row of rows) {
    const box = row.getBoundingClientRect();
    const offset = y - box.top - box.height / 2;
    if (offset < 0 && (!closest || offset > closest.offset)) {
      closest = { offset, el: row };
    }
  }
  return closest?.el ?? null;
}

function listAtY(selector: string, y: number): HTMLElement | null {
  for (const list of document.querySelectorAll<HTMLElement>(selector)) {
    const rect = list.getBoundingClientRect();
    if (y >= rect.top && y <= rect.bottom) {
      return list;
    }
  }
  return null;
}

function insertRow(list: HTMLElement, row: HTMLElement, y: number, rowSelector: string) {
  const after = getDragAfterElement(list, y, rowSelector);
  if (after == null) {
    list.appendChild(row);
  } else if (after !== row) {
    list.insertBefore(row, after);
  }
}

async function persistReorder(items: ReorderItem[]) {
  if (!statusEl) return;
  statusEl.textContent = 'Saving order…';
  statusEl.className = 'admin-reorder-hint admin-reorder-hint--busy';
  try {
    const res = await fetch('/api/admin/categories/reorder', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ items }),
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(body.error ?? 'Reorder failed');
    statusEl.textContent = 'Order saved';
    statusEl.className = 'admin-reorder-hint admin-reorder-hint--ok';
    window.setTimeout(() => {
      if (!statusEl) return;
      statusEl.textContent = 'Drag sections or forums to reorder';
      statusEl.className = 'admin-reorder-hint';
    }, 2000);
  } catch (err) {
    statusEl.textContent = err instanceof Error ? err.message : 'Reorder failed';
    statusEl.className = 'admin-reorder-hint admin-reorder-hint--error';
    window.setTimeout(() => location.reload(), 1200);
  }
}

function collectForumItems(): ReorderItem[] {
  const items: ReorderItem[] = [];
  document.querySelectorAll<HTMLElement>('.admin-cat-group').forEach((group) => {
    const parentId = group.dataset.groupId ?? null;
    group.querySelectorAll<HTMLElement>(':scope > .admin-cat-forums > .admin-cat-forum').forEach((row, i) => {
      const id = row.dataset.forumId;
      if (!id || !parentId) return;
      items.push({ category_id: id, sort_order: i + 1, parent_id: parentId });
    });
  });
  return items;
}

function collectGroupItems(): ReorderItem[] {
  const items: ReorderItem[] = [];
  document.querySelectorAll<HTMLElement>('#cat-tree > .admin-cat-group').forEach((group, i) => {
    const id = group.dataset.groupId;
    if (!id) return;
    items.push({ category_id: id, sort_order: i + 1, parent_id: null });
  });
  return items;
}

function clearDragState() {
  dragRow?.classList.remove('dragging');
  document.body.style.userSelect = '';
  document.querySelectorAll('.admin-cat-forums, #cat-tree').forEach((el) => {
    el.classList.remove('drag-over', 'drag-over-groups');
  });
  dragKind = null;
  dragRow = null;
  captureEl = null;
  pointerId = null;
}

function onPointerDown(e: PointerEvent) {
  if (e.button !== 0) return;

  const target = e.target as HTMLElement;
  const groupHandle = target.closest('.drag-handle--group');
  const forumHandle = target.closest('.drag-handle:not(.drag-handle--group)');

  if (groupHandle) {
    dragRow = groupHandle.closest('.admin-cat-group') as HTMLElement | null;
    dragKind = 'group';
  } else if (forumHandle) {
    dragRow = forumHandle.closest('.admin-cat-forum') as HTMLElement | null;
    dragKind = 'forum';
  } else {
    return;
  }

  if (!dragRow) return;

  e.preventDefault();
  pointerId = e.pointerId;
  dragRow.classList.add('dragging');
  document.body.style.userSelect = 'none';
  captureEl = (groupHandle ?? forumHandle) as HTMLElement;
  captureEl.setPointerCapture(e.pointerId);
}

function onPointerMove(e: PointerEvent) {
  if (!dragRow || !dragKind || pointerId !== e.pointerId) return;

  if (dragKind === 'forum') {
    const list =
      listAtY('.admin-cat-forums', e.clientY) ??
      (dragRow.parentElement as HTMLElement | null);
    if (!list) return;
    list.classList.add('drag-over');
    insertRow(list, dragRow, e.clientY, '.admin-cat-forum');
    return;
  }

  const tree = document.getElementById('cat-tree');
  if (!tree) return;
  tree.classList.add('drag-over-groups');
  insertRow(tree, dragRow, e.clientY, '.admin-cat-group');
}

async function onPointerUp(e: PointerEvent) {
  if (!dragRow || !dragKind || pointerId !== e.pointerId) return;

  const kind = dragKind;
  try {
    captureEl?.releasePointerCapture(e.pointerId);
  } catch {
    /* ignore */
  }
  clearDragState();

  if (kind === 'forum') {
    await persistReorder(collectForumItems());
  } else {
    await persistReorder([...collectGroupItems(), ...collectForumItems()]);
  }
}

export function initAdminCategoryDrag() {
  statusEl = document.getElementById('reorder-status');
  if (!statusEl) return;

  document.addEventListener('pointerdown', onPointerDown);
  document.addEventListener('pointermove', onPointerMove);
  document.addEventListener('pointerup', onPointerUp);
  document.addEventListener('pointercancel', clearDragState);
}