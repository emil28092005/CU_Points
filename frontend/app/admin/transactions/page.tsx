'use client';

import { useState, useEffect, useCallback } from 'react';
import { AdminNav } from '@/components/AdminNav';
import { Badge, Button, Card, Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import { formatDate, formatTransactionAmount } from '@/lib/utils';
import type { AdminTransaction, AdminTransactionPage, TransactionType } from '@/lib/types';

const PAGE_SIZE = 50;

type FilterType = 'all' | TransactionType;

const FILTERS: { value: FilterType; label: string }[] = [
  { value: 'all', label: 'Все' },
  { value: 'earn', label: 'Начисление' },
  { value: 'spend', label: 'Списание' },
  { value: 'admin_grant', label: 'Вручную' },
  { value: 'expire', label: 'Сгорание' },
];

export default function AdminTransactionsPage() {
  const [transactions, setTransactions] = useState<AdminTransaction[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [filter, setFilter] = useState<FilterType>('all');
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPage = useCallback(async (currentOffset: number, txFilter: FilterType, append: boolean) => {
    if (append) setLoadingMore(true);
    else setLoading(true);
    setError(null);

    const typeParam = txFilter !== 'all' ? `&type=${txFilter}` : '';
    try {
      const page = await api.get<AdminTransactionPage>(
        `/api/v1/admin/transactions?limit=${PAGE_SIZE}&offset=${currentOffset}${typeParam}`,
      );
      setTransactions((prev) => (append ? [...prev, ...page.transactions] : page.transactions));
      setTotal(page.total);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, []);

  useEffect(() => {
    setOffset(0);
    fetchPage(0, filter, false);
  }, [filter, fetchPage]);

  function handleLoadMore() {
    const next = offset + PAGE_SIZE;
    setOffset(next);
    fetchPage(next, filter, true);
  }

  const hasMore = transactions.length < total;

  return (
    <div className="min-h-screen">
      <AdminNav />
      <main className="mx-auto max-w-5xl p-4 pt-6 pb-10 space-y-5">
        <h1 className="text-xl font-bold text-white">Все транзакции</h1>

        {/* Type filter */}
        <div className="flex flex-wrap gap-2">
          {FILTERS.map((f) => (
            <button
              key={f.value}
              onClick={() => setFilter(f.value)}
              className={`rounded-full px-3 py-1 text-sm transition-colors ${
                filter === f.value
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>

        {loading && (
          <div className="flex justify-center py-16">
            <Spinner size="lg" />
          </div>
        )}

        {error && (
          <p className="rounded-lg bg-red-900/30 px-4 py-3 text-sm text-red-400">{error}</p>
        )}

        {!loading && (
          <Card className="p-0 overflow-x-auto">
            {transactions.length === 0 ? (
              <p className="py-10 text-center text-sm text-gray-500">Транзакций нет</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-700 text-left text-xs text-gray-500">
                    <th className="px-4 py-3">Студент</th>
                    <th className="px-4 py-3">Тип</th>
                    <th className="px-4 py-3 text-right">Сумма</th>
                    <th className="px-4 py-3">Описание</th>
                    <th className="px-4 py-3 text-right">Дата</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-700">
                  {transactions.map((tx) => (
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
                      <td className="px-4 py-3 text-gray-400 max-w-[200px] truncate">
                        {tx.description || '—'}
                      </td>
                      <td className="px-4 py-3 text-right text-gray-500 whitespace-nowrap">
                        {formatDate(tx.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </Card>
        )}

        {!loading && hasMore && (
          <div className="flex justify-center">
            <Button variant="secondary" onClick={handleLoadMore} isLoading={loadingMore}>
              Загрузить ещё
            </Button>
          </div>
        )}

        {!loading && !hasMore && transactions.length > 0 && (
          <p className="text-center text-xs text-gray-600">
            Показано {transactions.length} из {total}
          </p>
        )}
      </main>
    </div>
  );
}
