export function postReportButtonHTML(postId: string): string {
  return `<button type="button" class="post-report-btn" data-report="${postId}" aria-label="Flag for moderators" title="Flag for moderators">
    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
      <line x1="12" y1="9" x2="12" y2="13"/>
      <line x1="12" y1="17" x2="12.01" y2="17"/>
    </svg>
  </button>`;
}

export function markPostReported(btn: HTMLButtonElement) {
  btn.disabled = true;
  btn.classList.add('is-reported');
  btn.setAttribute('aria-label', 'Flagged for moderators');
  btn.title = 'Flagged for moderators';
}