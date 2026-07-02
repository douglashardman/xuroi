import type { Post } from './api';
import { formatDate } from './api';
import { isNearBottom, startPolling } from './poll';
import { bindNewPost } from './thread-interactions';
import { insertThreadPost } from './thread-post';

function knownPostIds(): Set<string> {
  const ids = new Set<string>();
  document.querySelectorAll('#thread-posts article.post').forEach((el) => {
    if (el.id.startsWith('post-')) ids.add(el.id.slice(5));
  });
  return ids;
}

function updateReplyMeta(count: number, lastActivity?: string) {
  const meta = document.getElementById('thread-tmeta');
  const root = document.getElementById('thread-posts');
  if (!meta) return;
  if (lastActivity) meta.dataset.lastActivity = lastActivity;
  const started = meta.dataset.started ?? '';
  const last = meta.dataset.lastActivity ?? '';
  const locked = meta.dataset.locked === '1' ? ' · Locked' : '';
  const pinned = meta.dataset.pinned === '1' ? ' · Pinned' : '';
  const label = count === 1 ? 'reply' : 'replies';
  meta.textContent = `${count} ${label} · started ${started} · last activity ${last}${locked}${pinned}`;
  meta.dataset.replyCount = String(count);
  if (root) root.dataset.replyCount = String(count);
}

export function initThreadPoll(intervalMs = 8000) {
  const root = document.getElementById('thread-posts');
  if (!root || root.dataset.lastPage !== '1') return () => {};

  const threadId = root.dataset.threadId;
  const page = root.dataset.pollPage ?? '1';
  if (!threadId) return () => {};

  const signedIn = root.dataset.signedIn === '1';
  const isStaff = root.dataset.isStaff === '1';

  const poll = async () => {
    const res = await fetch(`/api/threads/${threadId}?page=${page}`);
    if (!res.ok) return;
    const data = await res.json();
    const posts: Post[] = data.posts ?? [];
    const known = knownPostIds();
    const stick = isNearBottom();
    let added = 0;

    for (const post of posts) {
      if (known.has(post.id)) continue;
      const article = insertThreadPost(post, { signedIn, isStaff });
      bindNewPost(article);
      known.add(post.id);
      added++;
    }

    if (added > 0 && data.thread) {
      const lastLabel = data.thread.last_activity_at
        ? formatDate(data.thread.last_activity_at)
        : undefined;
      updateReplyMeta(data.thread.reply_count, lastLabel);
      if (stick) {
        document.getElementById('thread-posts-end')?.scrollIntoView({ behavior: 'smooth', block: 'end' });
      }
    }
  };

  return startPolling(poll, intervalMs);
}