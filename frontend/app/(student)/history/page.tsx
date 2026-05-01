'use client';

import { useState, useEffect, useCallback } from 'react';
import { StudentNav } from '@/components/StudentNav';
import { TransactionList } from '@/components/TransactionList';
import { Button } from '@/components/ui';
import { api } from '@/lib/api';
import type { Transaction, PaginatedResponse } from '@/lib/types';

const PAGE_SIZE = 20;

export default function HistoryPage() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPage = useCallback(async (currentOffset: number, append: boolean) => {
    if (append) setLoadingMore(true);
    else setLoading(true);
    setError(null);

    try {
      const page = await api.get<PaginatedResponse<Transaction>>(
        `/api/v1/me/transactions?limit=${PAGE_SIZE}&offset=${currentOffset}`,
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
    fetchPage(0, false);
  }, [fetchPage]);

  function handleLoadMore() {
    const nextOffset = offset + PAGE_SIZE;
    setOffset(nextOffset);
    fetchPage(nextOffset, true);
  }

  const hasMore = transactions.length < total;

  return (
    <div className="min-h-screen">
      <StudentNav />
    <main className="mx-auto max-w-lg p-4 pb-10 pt-6">
      <h1 className="mb-4 text-xl font-bold text-white">История операций</h1>

      {error && (
        <p className="mb-4 rounded-lg bg-red-900/30 px-3 py-2 text-sm text-red-400">{error}</p>
      )}

      <div className="rounded-2xl bg-gray-800 p-4 ring-1 ring-gray-700">
        <TransactionList transactions={transactions} isLoading={loading} />
      </div>

      {!loading && hasMore && (
        <div className="mt-4 flex justify-center">
          <Button variant="secondary" onClick={handleLoadMore} isLoading={loadingMore}>
            Загрузить ещё
          </Button>
        </div>
      )}

      {!loading && !hasMore && transactions.length > 0 && (
        <p className="mt-4 text-center text-xs text-gray-600">Это все операции</p>
      )}
    </main>
    </div>
  );
}
