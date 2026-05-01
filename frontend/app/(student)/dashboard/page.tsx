'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { BalanceCard } from '@/components/BalanceCard';
import { TransactionList } from '@/components/TransactionList';
import { Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/store';
import type { Profile, Transaction, PaginatedResponse } from '@/lib/types';

export default function StudentDashboardPage() {
  const { user, updateBalance } = useAuthStore();

  const [profile, setProfile] = useState<Profile | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loadingProfile, setLoadingProfile] = useState(true);
  const [loadingTxs, setLoadingTxs] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const [p, page] = await Promise.all([
          api.get<Profile>('/api/v1/me'),
          api.get<PaginatedResponse<Transaction>>('/api/v1/me/transactions?limit=5'),
        ]);
        setProfile(p);
        updateBalance(p.balance);
        setTransactions(page.transactions);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Ошибка загрузки');
      } finally {
        setLoadingProfile(false);
        setLoadingTxs(false);
      }
    }
    load();
  }, [updateBalance]);

  if (loadingProfile) {
    return (
      <main className="flex min-h-screen items-center justify-center">
        <Spinner size="lg" />
      </main>
    );
  }

  if (error) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6">
        <p className="text-red-400">{error}</p>
      </main>
    );
  }

  const displayName = profile?.name ?? user?.name ?? '';
  const balance = profile?.balance ?? user?.balance ?? 0;

  return (
    <main className="mx-auto max-w-lg space-y-6 p-4 pb-10 pt-6">
      <h1 className="text-xl font-bold text-white">
        Привет, {displayName.split(' ')[0]} 👋
      </h1>

      <BalanceCard balance={balance} name={displayName} />

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-base font-semibold text-gray-200">Последние операции</h2>
          <Link href="/history" className="text-sm text-blue-400 hover:text-blue-300">
            Вся история →
          </Link>
        </div>
        <div className="rounded-2xl bg-gray-800 p-4 ring-1 ring-gray-700">
          <TransactionList transactions={transactions} isLoading={loadingTxs} />
        </div>
      </section>
    </main>
  );
}
