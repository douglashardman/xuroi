import { formatDate } from './api';
import { initLightbox } from './lightbox';

export interface DMMessagePayload {
  id: string;
  body_html: string;
  created_at: string;
  is_mine?: boolean;
}

export function insertDMBubble(
  list: HTMLElement,
  msg: DMMessagePayload,
  opts: { justNow?: boolean } = {},
): HTMLElement {
  const article = document.createElement('article');
  article.className = `dm-bubble${msg.is_mine ? ' dm-bubble--mine' : ''} dm-bubble--enter`;
  article.dataset.msgId = msg.id;

  const meta = document.createElement('div');
  meta.className = 'dm-bubble-meta';
  const time = document.createElement('time');
  time.dateTime = msg.created_at;
  time.textContent = opts.justNow ? 'just now' : formatDate(msg.created_at);
  meta.append(time);

  const body = document.createElement('div');
  body.className = 'dm-bubble-body';
  body.innerHTML = msg.body_html;

  article.append(meta, body);
  list.appendChild(article);
  initLightbox(article);
  window.setTimeout(() => article.classList.remove('dm-bubble--enter'), 450);
  return article;
}