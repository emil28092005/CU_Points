'use client';

import { useState, useEffect, useCallback } from 'react';
import { AdminNav } from '@/components/AdminNav';
import { Button, Card, Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import { formatPoints, formatDate } from '@/lib/utils';
import type { AdminStudent, AdminUsersPage } from '@/lib/types';

const PAGE_SIZE = 50;

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminStudent[]>([]);
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
      const page = await api.get<AdminUsersPage>(
        `/api/v1/admin/users?limit=${PAGE_SIZE}&offset=${currentOffset}`,
      );
      setUsers((prev) => (append ? [...prev, ...page.users] : page.users));
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
    const next = offset + PAGE_SIZE;
    setOffset(next);
    fetchPage(next, true);
  }

  const hasMore = users.length < total;

  return (
    <div className="min-h-screen">
      <AdminNav />
      <main className="mx-auto max-w-5xl p-4 pt-6 pb-10 space-y-5">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold text-white">Студенты</h1>
          {!loading && (
            <span className="text-sm text-gray-500">Всего: {total}</span>
          )}
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
            {users.length === 0 ? (
              <p className="py-10 text-center text-sm text-gray-500">Студентов нет</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-700 text-left text-xs text-gray-500">
                    <th className="px-4 py-3">Имя</th>
                    <th className="px-4 py-3">Email</th>
                    <th className="px-4 py-3">Student ID</th>
                    <th className="px-4 py-3 text-right">Баланс</th>
                    <th className="px-4 py-3 text-right">Дата регистрации</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-700">
                  {users.map((u) => (
                    <tr key={u.id} className="hover:bg-gray-700/30">
                      <td className="px-4 py-3 font-medium text-white">{u.name}</td>
                      <td className="px-4 py-3 text-gray-300">{u.email}</td>
                      <td className="px-4 py-3 text-gray-500">
                        {u.student_id || '—'}
                      </td>
                      <td className="px-4 py-3 text-right font-semibold tabular-nums text-blue-300">
                        {formatPoints(u.balance)}
                      </td>
                      <td className="px-4 py-3 text-right text-gray-500 whitespace-nowrap">
                        {formatDate(u.created_at)}
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

        {!loading && !hasMore && users.length > 0 && (
          <p className="text-center text-xs text-gray-600">
            Показано {users.length} из {total}
          </p>
        )}
      </main>
    </div>
  );
}
