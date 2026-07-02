export const ACCENT_CLASSES = ['c-pink', 'c-blue', 'c-green'] as const;
export const ACCENT_VARS = ['var(--pink)', 'var(--blue)', 'var(--green)'] as const;
export const PAV_CLASSES = ['pav--pink', 'pav--blue', 'pav--green'] as const;

export function forumAccentIndex(groupIndex: number, forumIndex: number): number {
  return (groupIndex + forumIndex) % 3;
}

export function accentIndex(seed: string | number): number {
  if (typeof seed === 'number') return Math.abs(seed) % 3;
  let h = 0;
  for (let i = 0; i < seed.length; i++) h = (h + seed.charCodeAt(i)) % 3;
  return h;
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
}

export function formatCount(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1).replace(/\.0$/, '')}K`;
  return String(n);
}

export function userURL(displayName: string): string {
  const slug = displayName
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
  return `/u/${slug || 'member'}`;
}