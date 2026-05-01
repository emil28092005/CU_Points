'use client';

import { useState, type FormEvent } from 'react';
import { Button, Input, Card } from '@/components/ui';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/store';
import type { TokenPair, Profile, UserRole } from '@/lib/types';

function parseJwtRole(token: string): UserRole | null {
  try {
    const part = token.split('.')[1];
    const padded = part.padEnd(part.length + ((4 - (part.length % 4)) % 4), '=');
    const decoded = JSON.parse(atob(padded.replace(/-/g, '+').replace(/_/g, '/'))) as {
      role?: string;
    };
    const role = decoded.role;
    if (role === 'student' || role === 'partner' || role === 'admin') return role;
    return null;
  } catch {
    return null;
  }
}

const ROLE_REDIRECT: Record<UserRole, string> = {
  student: '/dashboard',
  partner: '/scan',
  admin: '/admin/dashboard',
};

export default function LoginPage() {
  const { setTokens, setUser } = useAuthStore();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const tokens = await api.post<TokenPair>('/api/v1/auth/login', { email, password });

      setTokens(tokens.access_token, tokens.refresh_token);

      const profile = await api.get<Profile>('/api/v1/me');
      const role = parseJwtRole(tokens.access_token);
      if (!role) throw new Error('Не удалось определить роль пользователя');

      setUser({ ...profile, role });
      // Hard redirect so the browser sends a fresh request with the new cookie.
      // router.push() can use a stale Next.js router cache and miss the cookie.
      window.location.href = ROLE_REDIRECT[role];
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка входа');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <h1 className="mb-6 text-2xl font-bold text-white">CU Points</h1>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Input
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
            placeholder="student@cu.ru"
          />
          <Input
            label="Пароль"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
          />

          {error && (
            <p className="rounded-lg bg-red-900/30 px-3 py-2 text-sm text-red-400">{error}</p>
          )}

          <Button type="submit" isLoading={loading} className="mt-2 w-full">
            Войти
          </Button>
        </form>
      </Card>
    </main>
  );
}
