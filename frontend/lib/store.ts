// Global client state managed with Zustand.
// Only truly global state lives here: auth tokens and the current user profile.
// Local UI state (loading flags, form values) stays in component useState.

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from './types';

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
  setTokens: (access: string, refresh: string) => void;
  setUser: (user: User) => void;
  updateBalance: (newBalance: number) => void;
  logout: () => void;
}

/** Writes the access token to localStorage and to a short-lived cookie so
 *  Next.js middleware (edge runtime) can read it for role-based redirects. */
function persistToken(accessToken: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('access_token', accessToken);
  // 15 min lifetime matches the default JWT_ACCESS_TTL
  document.cookie = `access_token=${accessToken}; path=/; SameSite=Lax; max-age=900`;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      user: null,

      setTokens: (accessToken, refreshToken) => {
        persistToken(accessToken);
        if (typeof window !== 'undefined') {
          localStorage.setItem('refresh_token', refreshToken);
        }
        set({ accessToken, refreshToken });
      },

      setUser: (user) => set({ user }),

      updateBalance: (newBalance) => {
        const { user } = get();
        if (user) set({ user: { ...user, balance: newBalance } });
      },

      logout: () => {
        if (typeof window !== 'undefined') {
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          document.cookie = 'access_token=; path=/; max-age=0';
        }
        set({ accessToken: null, refreshToken: null, user: null });
        if (typeof window !== 'undefined') {
          window.location.href = '/login';
        }
      },
    }),
    { name: 'cu-points-auth' }
  )
);
