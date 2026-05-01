// All API calls must go through this module — never call fetch directly in components.
// Automatically attaches the Bearer token and handles 401 → token refresh → retry.

import type { ApiError, ApiResponse } from './types';

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

function getAccessToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('access_token');
}

function getRefreshToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('refresh_token');
}

/** Attempts to refresh the access token using the stored refresh token.
 *  On success: updates localStorage + cookie and returns the new token.
 *  On failure: returns null so the caller can redirect to /login. */
async function tryRefresh(): Promise<string | null> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return null;

  try {
    const res = await fetch(`${BASE_URL}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) return null;

    const json = (await res.json()) as ApiResponse<{ access_token: string }>;
    const newToken = json.data.access_token;

    // Sync to localStorage + cookie so the middleware cookie stays valid.
    localStorage.setItem('access_token', newToken);
    document.cookie = `access_token=${newToken}; path=/; SameSite=Strict; max-age=900`;

    // Also patch the Zustand persist entry so the store stays consistent after page reload.
    try {
      const raw = localStorage.getItem('cu-points-auth');
      if (raw) {
        const parsed = JSON.parse(raw) as { state?: { accessToken?: string } };
        if (parsed.state) {
          parsed.state.accessToken = newToken;
          localStorage.setItem('cu-points-auth', JSON.stringify(parsed));
        }
      }
    } catch {
      // If patching the store fails it's not critical — the next setTokens call will fix it.
    }

    return newToken;
  } catch {
    return null;
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
  isRetry = false,
): Promise<T> {
  const token = getAccessToken();

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> | undefined),
  };

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  // 401: attempt refresh once, then give up and redirect to login.
  if (res.status === 401 && !isRetry) {
    const newToken = await tryRefresh();
    if (newToken) {
      return request<T>(path, options, true);
    }
    if (typeof window !== 'undefined') {
      window.location.href = '/login';
    }
    throw new Error('Session expired. Redirecting to login.');
  }

  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const err = (await res.json()) as ApiError;
      message = err.error ?? message;
    } catch {
      // response body was not JSON
    }
    throw new Error(message);
  }

  const json = (await res.json()) as ApiResponse<T>;
  return json.data;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body: unknown) => request<T>(path, { method: 'POST', body }),
};
