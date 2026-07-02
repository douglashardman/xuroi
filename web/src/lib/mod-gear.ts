const GEAR_SVG = `<svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`;

export function modGearButtonHTML(label = 'Moderator tools'): string {
  return `<button type="button" class="mod-gear-btn" data-mod-gear-toggle aria-label="${label}" aria-expanded="false" aria-haspopup="true">${GEAR_SVG}</button>`;
}

export function postModGearHTML(postId: string, warned: boolean): string {
  const warnLabel = warned ? 'Already warned' : 'Warn author';
  const warnDisabled = warned ? ' disabled' : '';
  return `
    <div class="mod-gear-wrap mod-gear-wrap--post">
      ${modGearButtonHTML('Post moderation')}
      <div class="mod-popover mod-popover--post" data-mod-popover hidden>
        <div class="mod-popover-head">Post</div>
        <button type="button" class="mod-popover-item" data-mod="${postId}">
          <span class="mod-popover-label">Audit &amp; ban</span>
        </button>
        <button type="button" class="mod-popover-item" data-warn="${postId}"${warnDisabled}>
          <span class="mod-popover-label">${warnLabel}</span>
        </button>
        <button type="button" class="mod-popover-item mod-popover-item--danger" data-remove="${postId}">
          <span class="mod-popover-label">Remove post</span>
        </button>
      </div>
    </div>
  `;
}

export function closeAllModPopovers() {
  document.querySelectorAll('[data-mod-popover]').forEach((el) => {
    (el as HTMLElement).hidden = true;
  });
  document.querySelectorAll('[data-mod-gear-toggle]').forEach((el) => {
    el.setAttribute('aria-expanded', 'false');
  });
}

export function initModGear() {
  document.addEventListener('click', (e) => {
    const target = e.target as HTMLElement;
    const toggle = target.closest('[data-mod-gear-toggle]');
    if (toggle) {
      e.stopPropagation();
      const wrap = toggle.closest('.mod-gear-wrap');
      const popover = wrap?.querySelector('[data-mod-popover]') as HTMLElement | null;
      if (!popover) return;
      const willOpen = popover.hidden;
      closeAllModPopovers();
      if (willOpen) {
        popover.hidden = false;
        toggle.setAttribute('aria-expanded', 'true');
      }
      return;
    }
    if (!target.closest('.mod-gear-wrap')) {
      closeAllModPopovers();
    }
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeAllModPopovers();
  });
}

export function syncThreadPinLabel(pinned: boolean) {
  const btn = document.querySelector('[data-thread-pin]');
  if (!btn) return;
  btn.setAttribute('data-pinned', pinned ? '1' : '0');
  const label = btn.querySelector('.mod-popover-label');
  if (label) label.textContent = pinned ? 'Unpin thread' : 'Pin thread';
}

export function syncThreadLockLabel(locked: boolean) {
  const btn = document.querySelector('[data-thread-lock]');
  if (!btn) return;
  btn.setAttribute('data-locked', locked ? '1' : '0');
  const label = btn.querySelector('.mod-popover-label');
  if (label) label.textContent = locked ? 'Unlock thread' : 'Lock thread';
}

export function syncThreadReportCount(count: number) {
  const badge = document.querySelector('[data-mod-report-badge]') as HTMLElement | null;
  const reportsBtn = document.querySelector('[data-thread-reports]');
  const label = reportsBtn?.querySelector('.mod-popover-label');
  if (badge) {
    if (count > 0) {
      badge.textContent = count > 99 ? '99+' : String(count);
      badge.hidden = false;
    } else {
      badge.hidden = true;
    }
  }
  if (label) {
    label.textContent = count > 0 ? `Thread reports (${count})` : 'Thread reports';
  }
  reportsBtn?.setAttribute('data-report-count', String(count));
}