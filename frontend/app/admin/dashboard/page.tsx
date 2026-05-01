'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { AdminNav } from '@/components/AdminNav';
import { Badge, Card, Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import { formatPoints, formatDate, formatTransactionAmount } from '@/lib/utils';
import type { Stats, AdminTransactionPage, AdminTransaction } from '@/lib/types';

interface StatCardProps {
  label: string;
  value: string | number;
}

function StatCard({ label, value }: StatCardProps) {
  return (
    <Card className="flex flex-col gap-1">
      <p className="text-xs text-gray-500 uppercase tracking-wide">{label}</p>
      <p className="text-2xl font-bold text-white">{value}</p>
    </Card>
  );
}

export default function AdminDashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [recentTxs, setRecentTxs] = useState<AdminTransaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const [s, page] = await Promise.all([
          api.get<Stats>('/api/v1/admin/stats'),
          api.get<AdminTransactionPage>('/api/v1/admin/transactions?limit=10'),
        ]);
        setStats(s);
        setRecentTxs(page.transactions);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Ошибка загрузки');
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  return (
    <div className="min-h-screen">
      <AdminNav />
      <main className="mx-auto max-w-5xl p-4 pt-6 pb-10 space-y-6">
        <h1 className="text-xl font-bold text-white">Дашборд</h1>

        {loading && (
          <div className="flex justify-center py-16">
            <Spinner size="lg" />
          </div>
        )}

        {error && (
          <p className="rounded-lg bg-red-900/30 px-4 py-3 text-sm text-red-400">{error}</p>
        )}

        {!loading && stats && (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <StatCard label="Студентов" value={stats.total_students} />
            <StatCard label="Поинтов выдано" value={formatPoints(stats.total_points_issued)} />
            <StatCard label="Поинтов потрачено" value={formatPoints(stats.total_points_spent)} />
            <StatCard label="Партнёров" value={stats.active_partners} />
          </div>
        )}

        {!loading && (
          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-base font-semibold text-gray-200">Последние транзакции</h2>
              <Link href="/admin/transactions" className="text-sm text-blue-400 hover:text-blue-300">
                Все транзакции →
              </Link>
            </div>

            <Card className="p-0 overflow-hidden">
              {recentTxs.length === 0 ? (
                <p className="py-8 text-center text-sm text-gray-500">Транзакций ещё нет</p>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-700 text-left text-xs text-gray-500">
                      <th className="px-4 py-3">Студент</th>
                      <th className="px-4 py-3">Тип</th>
                      <th className="px-4 py-3 text-right">Сумма</th>
                      <th className="px-4 py-3 text-right">Дата</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-700">
                    {recentTxs.map((tx) => (
                      <tr key={tx.id} className="hover:bg-gray-700/30">
                        <td className="px-4 py-3 text-gray-300">{tx.user_email}</td>
                        <td className="px-4 py-3">
                          <Badge type={tx.type} />
                        </td>
                        <td
                          className={`px-4 py-3 text-right font-semibold tabular-nums ${
                            tx.amount >= 0 ? 'text-green-400' : 'text-red-400'
                          }`}
                        >
                          {formatTransactionAmount(tx.amount)}
                        </td>
                        <td className="px-4 py-3 text-right text-gray-500">
                          {formatDate(tx.created_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </Card>

            <div className="mt-3 text-right">
              <Link href="/admin/grant" className="text-sm text-blue-400 hover:text-blue-300">
                Начислить поинты →
              </Link>
            </div>
          </section>
        )}
      </main>
    </div>
  );
}
