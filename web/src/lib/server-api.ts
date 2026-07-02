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
): Promise<RecentThreadsResponse> {
  const res = await backendFetch(`/v1/threads/recent?limit=${limit}`, {}, sessionToken);
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
  post_id: string;
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