import { showToast } from './toast';

type OnlineMember = {
  id: string;
  display_name: string;
  url: string;
  avatar_url?: string;
};

export function initNavMenus() {
  initUserMenu();
  initOnlinePanel();
}

function initUserMenu() {
  const trigger = document.getElementById('nav-user-trigger');
  const menu = document.getElementById('nav-user-menu');
  if (!trigger || !menu) return;

  const close = () => {
    menu.hidden = true;
    trigger.setAttribute('aria-expanded', 'false');
  };

  trigger.addEventListener('click', (e) => {
    e.stopPropagation();
    const open = menu.hidden;
    document.querySelectorAll('[data-nav-popover]').forEach((el) => {
      (el as HTMLElement).hidden = true;
    });
    document.querySelectorAll('[data-nav-popover-trigger]').forEach((el) => {
      el.setAttribute('aria-expanded', 'false');
    });
    menu.hidden = !open;
    trigger.setAttribute('aria-expanded', open ? 'true' : 'false');
  });

  document.getElementById('nav-logout')?.addEventListener('click', async () => {
    await fetch('/api/auth/logout', { method: 'POST' });
    window.location.href = '/';
  });

  document.addEventListener('click', (e) => {
    if (!menu.hidden && !(e.target as HTMLElement).closest('#nav-user-wrap')) close();
  });
}

function initOnlinePanel() {
  const btn = document.getElementById('nav-online-btn');
  const panel = document.getElementById('nav-online-panel');
  const countEl = document.getElementById('nav-online-count');
  const listEl = document.getElementById('nav-online-list');
  if (!btn || !panel || !countEl || !listEl) return;

  let loaded = false;

  async function loadOnline() {
    try {
      const res = await fetch('/api/members/online');
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Could not load');
      const count = data.count ?? 0;
      const members: OnlineMember[] = data.members ?? [];
      countEl.textContent = String(count);
      btn.hidden = false;
      if (members.length === 0) {
        listEl.innerHTML = '<p class="nav-online-empty">Nobody else online right now.</p>';
        return;
      }
      listEl.innerHTML = members.map((m) => `
        <a class="nav-online-member" href="${m.url}">
          <span class="nav-online-member-name">${escapeHtml(m.display_name)}</span>
        </a>
      `).join('');
      loaded = true;
    } catch {
      btn.hidden = true;
    }
  }

  void loadOnline();
  window.setInterval(loadOnline, 60_000);

  btn.addEventListener('click', async (e) => {
    e.stopPropagation();
    if (!loaded) await loadOnline();
    const open = panel.hidden;
    document.querySelectorAll('[data-nav-popover]').forEach((el) => {
      (el as HTMLElement).hidden = true;
    });
    document.querySelectorAll('[data-nav-popover-trigger]').forEach((el) => {
      el.setAttribute('aria-expanded', 'false');
    });
    panel.hidden = !open;
    btn.setAttribute('aria-expanded', open ? 'true' : 'false');
  });

  document.addEventListener('click', (e) => {
    if (!panel.hidden && !(e.target as HTMLElement).closest('#nav-online-wrap')) {
      panel.hidden = true;
      btn.setAttribute('aria-expanded', 'false');
    }
  });
}

function escapeHtml(value: string) {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}