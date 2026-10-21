import { request } from './http';
import { getAdminToken, setAdminTokens, type TokenPayload } from '@shared/auth/token';

export type PageResult<T = Record<string, unknown>> =
  | T[]
  | {
      data?: T[];
      items?: T[];
      list?: T[];
      total?: number;
      current?: number;
      pageSize?: number;
    };

export const adminApi = {
  async login(data: { username: string; password: string }) {
    const body = new URLSearchParams();
    body.set('username', data.username);
    body.set('password', data.password);
    const result = await request<TokenPayload>('/api-admin/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      rawBody: body
    });
    setAdminTokens(result);
    return result;
  },
  me() {
    return request('/api-admin/auth/me');
  },
  changePassword(data: { current_password: string; new_password: string }) {
    return request<{ success?: boolean; must_change_password?: boolean }>('/api-admin/auth/change-password', {
      method: 'POST',
      body: data,
    });
  },
  logout() {
    return request('/api-admin/auth/logout', { method: 'POST' });
  },
  dashboardStats() {
    return request<Record<string, number>>('/api-admin/dashboard/stats');
  },
  hasToken() {
    return Boolean(getAdminToken());
  },
  list<T = Record<string, unknown>>(path: string, query?: Record<string, string | number | boolean | undefined>) {
    return request<PageResult<T>>(path, { query });
  },
  get<T = Record<string, unknown>>(path: string, query?: Record<string, string | number | boolean | undefined>) {
    return request<T>(path, { query });
  },
  create<T = Record<string, unknown>>(path: string, body: unknown) {
    return request<T>(path, {
      method: 'POST',
      body
    });
  },
  update<T = Record<string, unknown>>(path: string, body: unknown) {
    return request<T>(path, {
      method: 'PUT',
      body
    });
  },
  action<T = Record<string, unknown>>(path: string, method: 'POST' | 'PUT' | 'PATCH' = 'PUT', body: unknown = {}) {
    return request<T>(path, {
      method,
      body
    });
  },
  remove<T = Record<string, unknown>>(path: string) {
    return request<T>(path, {
      method: 'DELETE'
    });
  }
};
