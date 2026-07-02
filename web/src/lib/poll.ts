export type PollStop = () => void;

export function startPolling(
  fn: () => void | Promise<void>,
  intervalMs: number,
): PollStop {
  let busy = false;

  const tick = async () => {
    if (document.hidden || busy) return;
    busy = true;
    try {
      await fn();
    } finally {
      busy = false;
    }
  };

  const timer = window.setInterval(() => void tick(), intervalMs);
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) void tick();
  });

  return () => window.clearInterval(timer);
}

export function isNearBottom(threshold = 140): boolean {
  const doc = document.documentElement;
  return doc.scrollHeight - window.scrollY - window.innerHeight < threshold;
}

export function isNearScrollBottom(el: HTMLElement, threshold = 80): boolean {
  return el.scrollHeight - el.scrollTop - el.clientHeight < threshold;
}