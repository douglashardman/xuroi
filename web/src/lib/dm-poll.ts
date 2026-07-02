import { insertDMBubble, type DMMessagePayload } from './dm-message';
import { isNearScrollBottom, startPolling } from './poll';

interface DMMessageResponse {
  id: string;
  sender_id: string;
  body_html: string;
  is_mine: boolean;
  created_at: string;
}

function knownMessageIds(list: HTMLElement): Set<string> {
  const ids = new Set<string>();
  list.querySelectorAll<HTMLElement>('[data-msg-id]').forEach((el) => {
    const id = el.dataset.msgId;
    if (id) ids.add(id);
  });
  return ids;
}

export function initDMPoll(convId: string, intervalMs = 7000) {
  const list = document.getElementById('dm-messages');
  if (!list) return () => {};

  const poll = async () => {
    const res = await fetch(`/api/dm/conversations/${convId}`);
    if (!res.ok) return;
    const data = await res.json();
    const messages: DMMessageResponse[] = data.messages ?? [];
    const known = knownMessageIds(list);
    const stick = isNearScrollBottom(list);
    let added = 0;

    for (const m of messages) {
      if (known.has(m.id)) continue;
      const payload: DMMessagePayload = {
        id: m.id,
        body_html: m.body_html,
        created_at: m.created_at,
        is_mine: m.is_mine,
      };
      insertDMBubble(list, payload);
      known.add(m.id);
      added++;
    }

    if (added > 0) {
      list.querySelector('.mod-empty')?.remove();
      if (stick) {
        list.lastElementChild?.scrollIntoView({ behavior: 'smooth', block: 'end' });
      }
    }
  };

  return startPolling(poll, intervalMs);
}