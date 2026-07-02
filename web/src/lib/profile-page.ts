import { openAvatarCrop } from './avatar-crop';
import { passkeyCeremony } from './passkey';
import { showToast } from './toast';

function makePendingButton(label: string) {
  const pending = document.createElement('button');
  pending.type = 'button';
  pending.className = 'btn btn--sm btn--pending';
  pending.disabled = true;
  pending.textContent = label;
  return pending;
}

function setFriendPending(actions: HTMLElement) {
  const btn = actions.querySelector('#profile-friend-btn');
  if (!btn) return;
  btn.replaceWith(makePendingButton('Friend Request Pending'));
}

function setFriends(actions: HTMLElement) {
  const btn = actions.querySelector('#profile-friend-btn, #profile-accept-friend-btn, .btn--pending');
  if (!btn) return;
  btn.replaceWith(makePendingButton('Friends'));
}

export function initVisitorProfileActions() {
  const actions = document.getElementById('profile-actions');
  if (!actions) return;

  actions.addEventListener('click', async (e) => {
    const target = e.target as HTMLElement;

    const friendBtn = target.closest('#profile-friend-btn') as HTMLButtonElement | null;
    if (friendBtn) {
      e.preventDefault();
      const recipientId = friendBtn.dataset.recipientId;
      if (!recipientId || friendBtn.disabled) return;
      friendBtn.disabled = true;
      try {
        const res = await fetch('/api/friends/requests', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ to_actor_id: recipientId }),
        });
        const data = await res.json().catch(() => ({}));
        if (res.ok || res.status === 409) {
          setFriendPending(actions);
          showToast('Friend request sent', 'success');
          return;
        }
        throw new Error(data.error || 'Could not send request');
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Request failed', 'error');
        friendBtn.disabled = false;
      }
      return;
    }

    const acceptBtn = target.closest('#profile-accept-friend-btn') as HTMLButtonElement | null;
    if (acceptBtn) {
      e.preventDefault();
      const requestId = acceptBtn.dataset.requestId;
      if (!requestId || acceptBtn.disabled) return;
      acceptBtn.disabled = true;
      try {
        const res = await fetch(`/api/friends/requests/${requestId}/accept`, { method: 'POST' });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || 'Could not accept');
        setFriends(actions);
        showToast('You are now friends', 'success');
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Accept failed', 'error');
        acceptBtn.disabled = false;
      }
      return;
    }

    const messageBtn = target.closest('#profile-message-btn') as HTMLButtonElement | null;
    if (messageBtn) {
      e.preventDefault();
      const recipientId = messageBtn.dataset.recipientId;
      if (!recipientId || messageBtn.disabled) return;
      messageBtn.disabled = true;
      try {
        const res = await fetch('/api/dm/conversations', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ recipient_id: recipientId }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Could not start conversation');
        window.location.href = data.url || `/messages/${data.conversation_id}`;
      } catch (err) {
        showToast(err instanceof Error ? err.message : 'Message failed', 'error');
        messageBtn.disabled = false;
      }
    }
  });
}

export function initOwnProfileActions() {
  const avatarInput = document.getElementById('avatar-input') as HTMLInputElement | null;
  const avatarTrigger = document.getElementById('profile-avatar-trigger');
  if (!avatarTrigger && !document.getElementById('set-password-form')) return;

  avatarTrigger?.addEventListener('click', (e) => {
    if ((e.target as HTMLElement).closest('#profile-avatar-remove')) return;
    avatarInput?.click();
  });

  document.getElementById('profile-avatar-remove')?.addEventListener('click', async (e) => {
    e.stopPropagation();
    const res = await fetch('/api/me/avatar', { method: 'DELETE' });
    if (!res.ok) {
      const data = await res.json();
      showToast(data.error || 'Could not remove photo', 'error');
      return;
    }
    showToast('Profile photo removed', 'success');
    window.location.reload();
  });

  avatarInput?.addEventListener('change', async () => {
    const file = avatarInput.files?.[0];
    avatarInput.value = '';
    if (!file) return;

    let cropped: Blob | null;
    try {
      cropped = await openAvatarCrop(file);
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Could not open image', 'error');
      return;
    }
    if (!cropped) return;

    const form = new FormData();
    form.append('file', cropped, 'avatar.jpg');
    avatarTrigger?.setAttribute('disabled', '');
    try {
      const res = await fetch('/api/me/avatar', { method: 'POST', body: form });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Upload failed');
      showToast('Profile photo updated', 'success');
      window.location.reload();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Upload failed', 'error');
      avatarTrigger?.removeAttribute('disabled');
    }
  });

  document.getElementById('set-password-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const password = new FormData(form).get('password') as string;
    const res = await fetch('/api/auth/password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password }),
    });
    const data = await res.json();
    if (!res.ok) {
      showToast(data.error || 'Could not set password', 'error');
      return;
    }
    showToast('Password saved', 'success');
    window.location.reload();
  });

  document.getElementById('dm-privacy-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const dmPrivacy = new FormData(form).get('dm_privacy') as string;
    const res = await fetch('/api/me/dm-privacy', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ dm_privacy: dmPrivacy }),
    });
    const data = await res.json();
    if (!res.ok) {
      showToast(data.error || 'Could not save privacy', 'error');
      return;
    }
    showToast('Message privacy updated', 'success');
  });

  document.getElementById('add-passkey-btn')?.addEventListener('click', async () => {
    const btn = document.getElementById('add-passkey-btn') as HTMLButtonElement;
    btn.disabled = true;
    try {
      const result = await passkeyCeremony(
        '/api/auth/passkey/register/begin',
        '/api/auth/passkey/register/finish',
      );
      if (!result.ok) throw new Error(result.error);
      showToast('Passkey added', 'success');
      window.location.reload();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Passkey failed', 'error');
    } finally {
      btn.disabled = false;
    }
  });
}