const API_URL = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

export interface Site {
  name: string;
  tagline?: string;
  url: string;
}

export interface CategoryLatestThread {
  title: string;
  url: string;
  author_name: string;
  last_activity_at: string;
}

export interface CategorySummary {
  id: string;
  slug: string;
  name: string;
  description: string;
  url: string;
  parent_id?: string | null;
  sort_order: number;
  is_group: boolean;
  access_level: string;
  access_levels?: string[];
  list_public: boolean;
  can_view: boolean;
  can_post?: boolean;
  locked_label?: string;
  thread_count: number;
  post_count: number;
  unread_count?: number;
  email_watching?: boolean;
  latest?: CategoryLatestThread | null;
}

export interface CategoryGroup {
  id: string;
  slug: string;
  name: string;
  description: string;
  sort_order: number;
  forums: CategorySummary[];
}

export interface ThreadSummary {
  id: string;
  title: string;
  title_prefix?: string;
  display_title?: string;
  slug: string;
  url: string;
  author_name: string;
  reply_count: number;
  view_count?: number;
  last_activity_at: string;
  is_pinned: boolean;
  is_locked: boolean;
  is_unread?: boolean;
}

export interface QuotedPost {
  id: string;
  author_name: string;
  excerpt: string;
  url: string;
}

export interface Post {
  id: string;
  author: {
    name: string;
    url: string;
    avatar_url?: string;
    is_agent: boolean;
    karma: number;
    active_warning?: boolean;
  };
  body_html: string;
  body_markdown?: string;
  quote?: QuotedPost;
  created_at: string;
  edited_at: string | null;
  is_op: boolean;
  reaction_count: number;
  reacted_by_me?: boolean;
  can_edit?: boolean;
  can_delete?: boolean;
  is_warned?: boolean;
  pending_moderation?: boolean;
  moderation_status?: string;
  is_accepted_answer?: boolean;
}

export interface Pagination {
  current: number;
  total: number;
  prev_url: string | null;
  next_url: string | null;
}

export interface HomeResponse {
  site: Site;
  groups: CategoryGroup[];
  categories: CategorySummary[];
}

export interface RecentThread {
  id: string;
  title: string;
  slug: string;
  url: string;
  category_name: string;
  category_slug: string;
  reply_count: number;
  last_activity_at: string;
  is_unread?: boolean;
}

export interface RecentThreadsResponse {
  site: Site;
  threads: RecentThread[];
}

export interface CategoryPageResponse {
  site: Site;
  category: CategorySummary;
  threads: ThreadSummary[];
  pagination: Pagination;
}

export interface ThreadPageResponse {
  site: Site;
  thread: {
    id: string;
    title: string;
    slug: string;
    url: string;
    summary?: string | null;
    reply_count: number;
    view_count?: number;
    is_locked: boolean;
    lock_reason?: string;
    is_pinned: boolean;
    email_watching?: boolean;
    created_at: string;
    last_activity_at: string;
  };
  category: { id: string; name: string; slug: string; url: string };
  posts: Post[];
  pagination: Pagination;
  ui?: {
    show_mod_bar?: boolean;
    open_report_count?: number;
    summary_label?: string;
    report_reasons?: ReportReason[];
  };
}

export interface ReportReason {
  id: string;
  label: string;
  allow_detail?: boolean;
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`);
  if (!res.ok) {
    throw new Error(`API ${path}: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export function getHome() {
  return fetchJSON<HomeResponse>('/v1/categories');
}

export interface UserProfile {
  id: string;
  display_name: string;
  url: string;
  avatar_url?: string;
  bio?: string;
  karma: number;
  post_count: number;
  joined_at: string;
  last_active_at?: string;
  hide_online?: boolean;
  friendship?: 'none' | 'friends' | 'pending_sent' | 'pending_received';
  incoming_friend_request_id?: string;
  can_message?: boolean;
  blocked_by_me?: boolean;
  blocks_me?: boolean;
}

export function getUser(slug: string) {
  return fetchJSON<UserProfile>(`/v1/users/${encodeURIComponent(slug)}`);
}

export function getRecentThreads(limit = 6) {
  return fetchJSON<RecentThreadsResponse>(`/v1/threads/recent?limit=${limit}`);
}

export function getCategory(slug: string, page = 1) {
  const q = page > 1 ? `?page=${page}` : '';
  return fetchJSON<CategoryPageResponse>(`/v1/categories/${slug}${q}`);
}

export function getThread(id: string, page = 1) {
  const q = page > 1 ? `?page=${page}` : '';
  return fetchJSON<ThreadPageResponse>(`/v1/threads/${id}${q}`);
}

export function formatLastSeen(iso: string) {
  const then = new Date(iso).getTime();
  const mins = Math.floor((Date.now() - then) / 60000);
  if (mins < 2) return 'Online now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 48) return `${hours}h ago`;
  return formatDate(iso);
}

export function userTimeZone(): string | undefined {
  if (typeof document === 'undefined') return undefined;
  const tz = document.documentElement.dataset.userTimezone;
  return tz || undefined;
}

export function formatDate(iso: string, timeZone?: string) {
  const tz = timeZone ?? userTimeZone();
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    timeZone: tz || undefined,
  });
}

export function formatDateTime(iso: string, timeZone?: string) {
  const tz = timeZone ?? userTimeZone();
  return new Date(iso).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    timeZone: tz || undefined,
  });
}

export function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return formatDate(iso);
}