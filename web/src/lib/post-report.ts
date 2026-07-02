const FLAG_ICON = `<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
  <line x1="12" y1="9" x2="12" y2="13"/>
  <line x1="12" y1="17" x2="12.01" y2="17"/>
</svg>`;

export function postReportButtonHTML(postId: string): string {
  return `<button type="button" class="post-report-btn" data-report="${postId}" aria-label="Flag for moderators" title="Flag for moderators">${FLAG_ICON}</button>`;
}

export function threadReportButtonHTML(threadId: string): string {
  return `<button type="button" class="post-report-btn thread-report-btn" data-thread-report="${threadId}" aria-label="Flag thread for moderators" title="Flag thread for moderators">${FLAG_ICON}</button>`;
}

export function markReported(btn: HTMLButtonElement, thread = false) {
  btn.disabled = true;
  btn.classList.add('is-reported');
  const label = thread ? 'Thread flagged for moderators' : 'Flagged for moderators';
  btn.setAttribute('aria-label', label);
  btn.title = label;
}

export function markPostReported(btn: HTMLButtonElement) {
  markReported(btn, false);
}