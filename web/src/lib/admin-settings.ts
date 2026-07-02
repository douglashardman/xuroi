import { showToast } from './toast';

export type SiteSettings = Record<string, unknown>;

function splitEmails(value: string) {
  return value.split(',').map((s) => s.trim()).filter(Boolean);
}

function splitLines(value: string) {
  return value.split(/[\n,]+/).map((s) => s.trim()).filter(Boolean);
}

function readBaseAdmin(): Record<string, unknown> {
  const el = document.getElementById('settings-admin-base');
  if (!el?.textContent?.trim()) return {};
  try {
    return JSON.parse(el.textContent) as Record<string, unknown>;
  } catch {
    return {};
  }
}

function collectReportReasons(root: HTMLElement) {
  return Array.from(root.querySelectorAll('.report-reason-row')).map((row) => ({
    id: (row.querySelector('.rr-id') as HTMLInputElement).value.trim(),
    label: (row.querySelector('.rr-label') as HTMLInputElement).value.trim(),
    allow_detail: (row.querySelector('.rr-allow-detail') as HTMLInputElement).checked,
  })).filter((r) => r.id && r.label);
}

function field<T extends HTMLElement>(id: string): T {
  const el = document.getElementById(id);
  if (!el) throw new Error(`Missing field #${id}`);
  return el as T;
}

function applySettingsFromResponse(data: SiteSettings) {
  if (typeof data.name === 'string') field<HTMLInputElement>('set-name').value = data.name;
  if (typeof data.tagline === 'string') field<HTMLInputElement>('set-tagline').value = data.tagline;

  const admin = data.admin;
  if (admin && typeof admin === 'object') {
    const base = document.getElementById('settings-admin-base');
    if (base) base.textContent = JSON.stringify(admin);
    if (Array.isArray(admin.moderator_emails)) {
      field<HTMLInputElement>('set-mod-emails').value = admin.moderator_emails.join(', ');
    }
    if (Array.isArray(admin.perm_ban_moderator_emails)) {
      field<HTMLInputElement>('set-perm-ban-emails').value = admin.perm_ban_moderator_emails.join(', ');
    }
  }

  if (Array.isArray(data.reserved_display_names)) {
    field<HTMLTextAreaElement>('set-reserved-names').value = data.reserved_display_names.join('\n');
  }
}

function normalizeReservedNames(names: string[]) {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of names) {
    const name = raw.trim().toLowerCase();
    if (!name || seen.has(name)) continue;
    seen.add(name);
    out.push(name);
  }
  return out.sort();
}

function verifySavedPayload(sent: ReturnType<typeof buildSettingsPayload>, data: SiteSettings) {
  if (!Array.isArray(data.reserved_display_names)) {
    throw new Error('API did not return reserved names — restart the API (make dev) and try again.');
  }
  const sentNames = normalizeReservedNames(sent.reserved_display_names).join('\n');
  const gotNames = normalizeReservedNames(data.reserved_display_names).join('\n');
  if (sentNames !== gotNames) {
    throw new Error('Reserved names did not persist — restart the API and try again.');
  }
}

export function buildSettingsPayload(baseAdmin: Record<string, unknown>, reasonsRoot: HTMLElement) {
  return {
    name: field<HTMLInputElement>('set-name').value.trim(),
    tagline: field<HTMLInputElement>('set-tagline').value.trim(),
    admin: {
      ...baseAdmin,
      moderator_emails: splitEmails(field<HTMLInputElement>('set-mod-emails').value),
      perm_ban_moderator_emails: splitEmails(field<HTMLInputElement>('set-perm-ban-emails').value),
    },
    posts: {
      edit_enabled: field<HTMLInputElement>('set-edit-enabled').checked,
      edit_window_minutes: Number(field<HTMLInputElement>('set-edit-window').value) || 30,
      delete_enabled: field<HTMLInputElement>('set-delete-enabled').checked,
    },
    guests: {
      read_only: field<HTMLInputElement>('set-guest-readonly').checked,
      can_attach: field<HTMLInputElement>('set-guest-attach').checked,
    },
    email: {
      enabled: field<HTMLInputElement>('set-email-enabled').checked,
      from_address: field<HTMLInputElement>('set-email-from').value.trim(),
      from_name: field<HTMLInputElement>('set-email-from-name').value.trim(),
      reply_to: field<HTMLInputElement>('set-email-reply').value.trim(),
      digest_delay_minutes: Number(field<HTMLInputElement>('set-email-digest').value) || 5,
    },
    intelligence: {
      enabled: field<HTMLInputElement>('set-intel-enabled').checked,
      summary_label: field<HTMLInputElement>('set-intel-label').value.trim() || 'TL;DR',
    },
    new_users: {
      restrict_links_hours: Number(field<HTMLInputElement>('set-link-hours').value) || 0,
      restrict_dm_hours: Number(field<HTMLInputElement>('set-dm-hours').value) || 0,
    },
    spam: {
      enabled: field<HTMLInputElement>('set-spam-enabled').checked,
      hold_for_moderation: field<HTMLInputElement>('set-spam-hold').checked,
      score_threshold: Number(field<HTMLInputElement>('set-spam-threshold').value) || 6,
      max_links_new_user: Number(field<HTMLInputElement>('set-spam-max-links').value) || 2,
      new_account_hours: Number(field<HTMLInputElement>('set-spam-new-hours').value) || 48,
    },
    seo: {
      nofollow_user_links: field<HTMLInputElement>('set-seo-nofollow').checked,
    },
    moderation: {
      report_reasons: collectReportReasons(reasonsRoot),
    },
    reserved_display_names: splitLines(field<HTMLTextAreaElement>('set-reserved-names').value),
  };
}

export function initAdminSettingsForm() {
  const form = document.getElementById('settings-form') as HTMLFormElement | null;
  const errorEl = document.getElementById('set-error') as HTMLElement | null;
  const reasonsRoot = document.getElementById('report-reasons');
  const saveBtn = document.getElementById('settings-save-btn') as HTMLButtonElement | null;
  const dirtyEl = document.getElementById('settings-dirty');
  if (!form || !reasonsRoot || !saveBtn) return;

  let dirty = false;
  const markDirty = () => {
    if (!dirty) {
      dirty = true;
      if (dirtyEl) dirtyEl.hidden = false;
    }
  };

  form.addEventListener('input', markDirty);
  form.addEventListener('change', markDirty);

  function addReasonRow(id = '', label = '', allowDetail = false) {
    const row = document.createElement('div');
    row.className = 'report-reason-row';
    row.innerHTML = `
      <input type="text" class="admin-settings-input rr-id" value="${id.replace(/"/g, '&quot;')}" placeholder="id" />
      <input type="text" class="admin-settings-input rr-label" value="${label.replace(/"/g, '&quot;')}" placeholder="Label" />
      <label class="admin-settings-check rr-detail">
        <input type="checkbox" class="rr-allow-detail" ${allowDetail ? 'checked' : ''} />
        <span>Detail field</span>
      </label>
      <button type="button" class="btn btn--sm btn--ghost rr-remove" aria-label="Remove reason">×</button>
    `;
    reasonsRoot.appendChild(row);
    markDirty();
  }

  document.getElementById('add-report-reason')?.addEventListener('click', () => addReasonRow());

  reasonsRoot.addEventListener('click', (e) => {
    const btn = (e.target as HTMLElement).closest('.rr-remove');
    if (!btn) return;
    btn.closest('.report-reason-row')?.remove();
    markDirty();
  });

  document.querySelectorAll<HTMLAnchorElement>('[data-settings-jump]').forEach((link) => {
    link.addEventListener('click', (e) => {
      e.preventDefault();
      const id = link.getAttribute('href')?.slice(1);
      if (!id) return;
      document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  });

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (errorEl) errorEl.hidden = true;

    const originalLabel = saveBtn.textContent;
    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving…';

    try {
      const payload = buildSettingsPayload(readBaseAdmin(), reasonsRoot);
      const res = await fetch('/api/admin/site-settings', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      let data: SiteSettings = {};
      try {
        data = await res.json();
      } catch {
        data = {};
      }
      if (!res.ok) {
        throw new Error((data.error as string) || `Save failed (${res.status})`);
      }
      if (!data.name || data.status === 'saved') {
        throw new Error('API returned an old save response — restart the API (make dev) and try again.');
      }

      verifySavedPayload(payload, data);
      applySettingsFromResponse(data);

      dirty = false;
      if (dirtyEl) dirtyEl.hidden = true;
      showToast('Settings saved', 'success');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Save failed';
      if (errorEl) {
        errorEl.textContent = msg;
        errorEl.hidden = false;
      }
      showToast(msg, 'error');
    } finally {
      saveBtn.disabled = false;
      saveBtn.textContent = originalLabel;
    }
  });
}