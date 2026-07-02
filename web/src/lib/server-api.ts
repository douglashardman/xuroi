const API_URL = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';
const SESSION_COOKIE = 'xuroi_session';
const SESSION_MAX_AGE = 30 * 24 * 60 * 60;

export { API_URL, SESSION_COOKIE, SESSION_MAX_AGE };

export interface ActiveWarning {
  message: string;
  warned_by: string;
  expires_at: string;
  warning_count: number;
  strike_number: number;
}

export interface Actor {
  id: string;
  display_name: string;
  email?: string;
  is_admin?: boolean;
  is_moderator?: boolean;
  can_perm_ban?: boolean;
  email_verified?: boolean;
  state?: string;
  has_password?: boolean;
  has_passkey?: boolean;
  active_warning?: ActiveWarning;
  entitlements?: string[];
  unread_notifications?: number;
  unread_dm?: number;
  unread_threads?: number;
  pending_friend_requests?: number;
  dm_privacy?: 'everyone' | 'friends_only' | 'off';
  timezone?: string;
  avatar_url?: string;
}

export interface FriendMember {
  id: string;
  display_name: string;
  avatar_url?: string;
  url: string;
}

export interface FriendRequest {
  id: string;
  from: FriendMember;
  to: FriendMember;
  status: string;
  created_at: string;
}

export interface DMParticipant {
  id: string;
  display_name: string;
  avatar_url?: string;
  url: string;
}

export interface DMConversationSummary {
  id: string;
  other: DMParticipant;
  last_preview: string;
  last_message_at: string;
  unread_count: number;
}

export interface DMMessage {
  id: string;
  sender_id: string;
  body_html: string;
  is_mine: boolean;
  created_at: string;
}

export interface DMConversationPage {
  id: string;
  other: DMParticipant;
  messages: DMMessage[];
}

export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  url: string;
  from_actor_id?: string;
  from_actor_name?: string;
  post_id?: string;
  thread_id?: string;
  read_at?: string;
  created_at: string;
}

export interface NotificationsResponse {
  notifications: Notification[];
  unread_count: number;
}

export interface AdminOverview {
  members: number;
  threads: number;
  posts: number;
  open_reports: number;
  banned_users: number;
}

export interface AdminUserRow {
  id: string;
  display_name: string;
  email: string;
  state: string;
  email_verified: boolean;
  post_count: number;
  joined_at: string;
  ban_reason?: string;
  banned_until?: string;
  banned_by_name?: string;
  warning_count?: number;
  permissions?: string[];
  entitlements?: string[];
  url: string;
}

export interface StaffPermissionInfo {
  id: string;
  label: string;
  description: string;
}

export async function backendFetch(
  path: string,
  init: RequestInit = {},
  sessionToken?: string | null,
) {
  const headers = new Headers(init.headers);
  if (sessionToken) {
    headers.set('X-Session-Token', sessionToken);
  }
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  return fetch(`${API_URL}${path}`, { ...init, headers });
}

import type {
  CategoryGroup,
  CategoryPageResponse,
  CategorySummary,
  HomeResponse,
  RecentThreadsResponse,
  ThreadPageResponse,
} from './api';

export async function getHome(sessionToken?: string | null): Promise<HomeResponse> {
  const res = await backendFetch('/v1/categories', {}, sessionToken);
  if (!res.ok) {
    throw new Error(`API categories: ${res.status}`);
  }
  return res.json() as Promise<HomeResponse>;
}

export async function getCategory(
  slug: string,
  page = 1,
  sessionToken?: string | null,
): Promise<CategoryPageResponse> {
  const q = page > 1 ? `?page=${page}` : '';
  const res = await backendFetch(`/v1/categories/${encodeURIComponent(slug)}${q}`, {}, sessionToken);
  if (!res.ok) {
    throw new Error(`API category ${slug}: ${res.status}`);
  }
  return res.json() as Promise<CategoryPageResponse>;
}

export async function getRecentThreads(
  limit = 6,
  sessionToken?: string | null,
  unreadOnly = false,
): Promise<RecentThreadsResponse> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (unreadOnly) params.set('unread_only', '1');
  const res = await backendFetch(`/v1/threads/recent?${params}`, {}, sessionToken);
  if (!res.ok) {
    throw new Error(`API recent threads: ${res.status}`);
  }
  return res.json() as Promise<RecentThreadsResponse>;
}

export async function getThread(
  id: string,
  page = 1,
  sessionToken?: string | null,
): Promise<ThreadPageResponse> {
  const q = page > 1 ? `?page=${page}` : '';
  const res = await backendFetch(`/v1/threads/${id}${q}`, {}, sessionToken);
  if (!res.ok) {
    throw new Error(`API thread ${id}: ${res.status}`);
  }
  return res.json() as Promise<ThreadPageResponse>;
}

export async function getMe(sessionToken?: string | null): Promise<Actor | null> {
  if (!sessionToken) return null;
  const res = await backendFetch('/v1/auth/me', {}, sessionToken);
  if (!res.ok) return null;
  return res.json() as Promise<Actor>;
}

export interface PostReport {
  id: string;
  kind?: 'post' | 'thread';
  post_id?: string;
  thread_id: string;
  thread_title: string;
  thread_url: string;
  post_author: string;
  post_excerpt: string;
  reporter_name: string;
  reason: string;
  created_at: string;
  post_url: string;
}

export async function getModReports(
  sessionToken: string,
  opts: { threadId?: string; limit?: number } = {},
): Promise<PostReport[]> {
  const params = new URLSearchParams();
  if (opts.threadId) params.set('thread_id', opts.threadId);
  if (opts.limit) params.set('limit', String(opts.limit));
  const q = params.toString();
  const res = await backendFetch(q ? `/v1/admin/reports?${q}` : '/v1/admin/reports', {}, sessionToken);
  if (!res.ok) {
    throw new Error(`API reports: ${res.status}`);
  }
  const data = (await res.json()) as { reports: PostReport[] };
  return data.reports ?? [];
}

export async function getAdminOverview(sessionToken: string): Promise<AdminOverview> {
  const res = await backendFetch('/v1/admin/overview', {}, sessionToken);
  if (!res.ok) throw new Error(`API admin overview: ${res.status}`);
  return res.json() as Promise<AdminOverview>;
}

export async function getAdminUsers(
  sessionToken: string,
  opts: { q?: string; limit?: number; offset?: number } = {},
): Promise<{ users: AdminUserRow[]; total: number }> {
  const params = new URLSearchParams();
  if (opts.q) params.set('q', opts.q);
  if (opts.limit) params.set('limit', String(opts.limit));
  if (opts.offset) params.set('offset', String(opts.offset));
  const q = params.toString();
  const res = await backendFetch(q ? `/v1/admin/users?${q}` : '/v1/admin/users', {}, sessionToken);
  if (!res.ok) throw new Error(`API admin users: ${res.status}`);
  const data = (await res.json()) as { users: AdminUserRow[]; total: number };
  data.users = (data.users ?? []).map((u) => ({
    ...u,
    url: `/u/${encodeURIComponent(u.display_name.toLowerCase().replace(/\s+/g, '-'))}`,
  }));
  return data;
}

export interface AdminCategoriesResponse {
  groups: CategoryGroup[];
  categories: CategorySummary[];
}

export interface AccessLevelInfo {
  id: string;
  label: string;
  description: string;
}

export async function getAccessLevels(sessionToken: string): Promise<{
  levels: AccessLevelInfo[];
  entitlements: AccessLevelInfo[];
}> {
  const res = await backendFetch('/v1/admin/access-levels', {}, sessionToken);
  if (!res.ok) throw new Error(`API access levels: ${res.status}`);
  return res.json() as Promise<{ levels: AccessLevelInfo[]; entitlements: AccessLevelInfo[] }>;
}

export type SiteSettings = {
  name?: string;
  tagline?: string;
  posts?: Record<string, unknown>;
  guests?: Record<string, unknown>;
  intelligence?: Record<string, unknown>;
  email?: Record<string, unknown>;
  admin?: Record<string, unknown>;
  moderation?: { report_reasons?: Array<{ id: string; label: string; allow_detail?: boolean }> };
  new_users?: Record<string, unknown>;
  spam?: Record<string, unknown>;
  seo?: { nofollow_user_links?: boolean };
  reserved_display_names?: string[];
};

export async function getAdminSiteSettings(sessionToken: string): Promise<SiteSettings> {
  const res = await backendFetch('/v1/admin/site-settings', {}, sessionToken);
  if (!res.ok) throw new Error(`API site settings: ${res.status}`);
  return res.json() as Promise<SiteSettings>;
}

export async function getAdminCategories(sessionToken: string): Promise<AdminCategoriesResponse> {
  const res = await backendFetch('/v1/admin/categories', {}, sessionToken);
  if (!res.ok) throw new Error(`API admin categories: ${res.status}`);
  return res.json() as Promise<AdminCategoriesResponse>;
}

export async function getPermissionCatalog(sessionToken: string): Promise<StaffPermissionInfo[]> {
  const res = await backendFetch('/v1/admin/permissions', {}, sessionToken);
  if (!res.ok) throw new Error(`API permissions: ${res.status}`);
  const data = (await res.json()) as { permissions: StaffPermissionInfo[] };
  return data.permissions ?? [];
}

export async function markAllNotificationsRead(sessionToken: string): Promise<void> {
  const res = await backendFetch('/v1/notifications/read-all', { method: 'POST' }, sessionToken);
  if (!res.ok) throw new Error(`API mark notifications read: ${res.status}`);
}

export async function markThreadRead(sessionToken: string, threadId: string): Promise<void> {
  const res = await backendFetch(`/v1/threads/${threadId}/read`, { method: 'POST' }, sessionToken);
  if (!res.ok && res.status !== 404) {
    throw new Error(`API mark thread read: ${res.status}`);
  }
}

export async function markCategoryRead(sessionToken: string, slug: string): Promise<number> {
  const res = await backendFetch(`/v1/categories/${encodeURIComponent(slug)}/read`, { method: 'POST' }, sessionToken);
  if (!res.ok) throw new Error(`API mark category read: ${res.status}`);
  const data = (await res.json()) as { marked?: number };
  return data.marked ?? 0;
}

export async function recordThreadView(sessionToken: string, threadId: string): Promise<void> {
  await backendFetch(`/v1/threads/${threadId}/view`, { method: 'POST' }, sessionToken);
}

export async function getNotifications(
  sessionToken: string,
  opts: { limit?: number; offset?: number } = {},
): Promise<NotificationsResponse> {
  const params = new URLSearchParams();
  if (opts.limit) params.set('limit', String(opts.limit));
  if (opts.offset) params.set('offset', String(opts.offset));
  const q = params.toString();
  const res = await backendFetch(q ? `/v1/notifications?${q}` : '/v1/notifications', {}, sessionToken);
  if (!res.ok) throw new Error(`API notifications: ${res.status}`);
  const data = (await res.json()) as NotificationsResponse;
  data.notifications = data.notifications ?? [];
  data.unread_count = data.unread_count ?? 0;
  return data;
}

export async function getDMConversations(
  sessionToken: string,
  opts: { limit?: number } = {},
): Promise<DMConversationSummary[]> {
  const params = new URLSearchParams();
  if (opts.limit) params.set('limit', String(opts.limit));
  const q = params.toString();
  const res = await backendFetch(q ? `/v1/dm/conversations?${q}` : '/v1/dm/conversations', {}, sessionToken);
  if (!res.ok) throw new Error(`API dm conversations: ${res.status}`);
  const data = (await res.json()) as { conversations: DMConversationSummary[] };
  return data.conversations ?? [];
}

export async function getDMConversation(
  sessionToken: string,
  id: string,
): Promise<DMConversationPage> {
  const res = await backendFetch(`/v1/dm/conversations/${id}`, {}, sessionToken);
  if (!res.ok) throw new Error(`API dm conversation: ${res.status}`);
  const data = (await res.json()) as DMConversationPage;
  data.messages = data.messages ?? [];
  return data;
}

export async function getUserProfile(
  slug: string,
  sessionToken?: string | null,
): Promise<import('./api').UserProfile> {
  const res = await backendFetch(`/v1/users/${encodeURIComponent(slug)}`, {}, sessionToken);
  if (!res.ok) throw new Error(`API user ${slug}: ${res.status}`);
  return res.json() as Promise<import('./api').UserProfile>;
}

export async function getFriendRequests(sessionToken: string): Promise<{
  incoming: FriendRequest[];
  outgoing: FriendRequest[];
}> {
  const res = await backendFetch('/v1/friends/requests', {}, sessionToken);
  if (!res.ok) throw new Error(`API friend requests: ${res.status}`);
  const data = (await res.json()) as { incoming: FriendRequest[]; outgoing: FriendRequest[] };
  return {
    incoming: data.incoming ?? [],
    outgoing: data.outgoing ?? [],
  };
}

export async function getDMPrivacy(sessionToken: string): Promise<string> {
  const res = await backendFetch('/v1/me/dm-privacy', {}, sessionToken);
  if (!res.ok) throw new Error(`API dm privacy: ${res.status}`);
  const data = (await res.json()) as { dm_privacy: string };
  return data.dm_privacy;
}

export type EmailPreferences = {
  thread_replies_enabled: boolean;
  mentions_enabled: boolean;
};

export async function getEmailPreferences(sessionToken: string): Promise<EmailPreferences> {
  const res = await backendFetch('/v1/me/email-preferences', {}, sessionToken);
  if (!res.ok) throw new Error(`API email preferences: ${res.status}`);
  return res.json() as Promise<EmailPreferences>;
}

export type ReportReason = {
  id: string;
  label: string;
  allow_detail?: boolean;
};

const DEFAULT_REPORT_REASONS: ReportReason[] = [
  { id: 'spam', label: 'Spam or advertising' },
  { id: 'harassment', label: 'Harassment or abuse' },
  { id: 'off_topic', label: 'Off-topic' },
  { id: 'inappropriate', label: 'Inappropriate content' },
  { id: 'other', label: 'Other', allow_detail: true },
];

export async function getReportReasons(): Promise<ReportReason[]> {
  try {
    const res = await backendFetch('/v1/moderation/report-reasons');
    if (!res.ok) return DEFAULT_REPORT_REASONS;
    const data = (await res.json()) as { reasons?: ReportReason[] };
    return data.reasons?.length ? data.reasons : DEFAULT_REPORT_REASONS;
  } catch {
    return DEFAULT_REPORT_REASONS;
  }
}

export function sessionFromCookieHeader(cookieHeader: string | null): string | null {
  if (!cookieHeader) return null;
  for (const part of cookieHeader.split(';')) {
    const [name, ...rest] = part.trim().split('=');
    if (name === SESSION_COOKIE) {
      return rest.join('=') || null;
    }
  }
  return null;
}