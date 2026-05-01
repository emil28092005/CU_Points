'use client';

import { useState, useEffect, useRef } from 'react';
import { AdminNav } from '@/components/AdminNav';
import { Button, Card, Input, Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import { formatPoints, formatDate } from '@/lib/utils';
import type { AdminStudent, AdminUsersPage } from '@/lib/types';

interface GrantRecord {
  key: string;
  studentName: string;
  amount: number;
  description: string;
  grantedAt: string;
}

export default function GrantPage() {
  const [search, setSearch] = useState('');
  const [suggestions, setSuggestions] = useState<AdminStudent[]>([]);
  const [selected, setSelected] = useState<AdminStudent | null>(null);
  const [showDropdown, setShowDropdown] = useState(false);
  const [amount, setAmount] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState('');
  const [history, setHistory] = useState<GrantRecord[]>([]);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Debounced search: fires 300 ms after the user stops typing.
  useEffect(() => {
    if (!search.trim() || selected) {
      setSuggestions([]);
      setShowDropdown(false);
      return;
    }
    setSearching(true);
    const timer = setTimeout(async () => {
      try {
        const page = await api.get<AdminUsersPage>(
          `/api/v1/admin/users?search=${encodeURIComponent(search)}&limit=8`,
        );
        setSuggestions(page.users);
        setShowDropdown(page.users.length > 0);
      } catch {
        setSuggestions([]);
      } finally {
        setSearching(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [search, selected]);

  // Close dropdown when clicking outside.
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    }
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  function selectStudent(student: AdminStudent) {
    setSelected(student);
    setSearch(student.name + ' — ' + student.email);
    setSuggestions([]);
    setShowDropdown(false);
  }

  function clearSelection() {
    setSelected(null);
    setSearch('');
    setSuggestions([]);
  }

  async function handleGrant() {
    if (!selected || !amount) return;
    const pts = parseInt(amount, 10);
    if (isNaN(pts) || pts <= 0) return;

    setLoading(true);
    setError('');
    try {
      await api.post<{ status: string }>('/api/v1/admin/points/grant', {
        user_id: selected.id,
        amount: pts,
        description,
      });

      // Refresh selected student balance.
      const page = await api.get<AdminUsersPage>(
        `/api/v1/admin/users?search=${encodeURIComponent(selected.email)}&limit=1`,
      );
      const updated = page.users.find((s) => s.id === selected.id);
      if (updated) setSelected(updated);

      setHistory((prev) =>
        [
          {
            key: String(Date.now()),
            studentName: selected.name,
            amount: pts,
            description: description || '—',
            grantedAt: new Date().toISOString(),
          },
          ...prev,
        ].slice(0, 5),
      );
      setAmount('');
      setDescription('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка начисления');
    } finally {
      setLoading(false);
    }
  }

  const amountNum = parseInt(amount, 10);
  const canSubmit = !!selected && amountNum > 0 && !loading;

  return (
    <div className="min-h-screen">
      <AdminNav />
      <main className="mx-auto max-w-lg p-4 pt-6 pb-10 space-y-6">
        <h1 className="text-xl font-bold text-white">Начислить поинты</h1>

        <Card className="space-y-5">
          {/* Student search */}
          <div className="relative" ref={dropdownRef}>
            <div className="flex gap-2">
              <div className="flex-1">
                <Input
                  label="Поиск студента (email или имя)"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    if (selected) clearSelection();
                  }}
                  placeholder="student@cu.ru"
                  autoComplete="off"
                />
              </div>
              {selected && (
                <button
                  onClick={clearSelection}
                  className="mt-6 shrink-0 rounded-lg px-3 text-gray-400 hover:text-white transition-colors"
                  aria-label="Очистить"
                >
                  ✕
                </button>
              )}
            </div>

            {/* Suggestions dropdown */}
            {showDropdown && suggestions.length > 0 && (
              <ul className="absolute z-10 mt-1 w-full rounded-lg border border-gray-600 bg-gray-800 shadow-lg">
                {suggestions.map((s) => (
                  <li key={s.id}>
                    <button
                      className="flex w-full items-center justify-between px-4 py-2.5 text-left text-sm hover:bg-gray-700"
                      onMouseDown={(e) => {
                        e.preventDefault(); // prevent blur before click
                        selectStudent(s);
                      }}
                    >
                      <span>
                        <span className="font-medium text-white">{s.name}</span>
                        <span className="ml-2 text-gray-400">{s.email}</span>
                      </span>
                      <span className="shrink-0 text-gray-500">{formatPoints(s.balance)}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}

            {searching && !showDropdown && (
              <div className="absolute right-3 top-9">
                <Spinner size="sm" />
              </div>
            )}
          </div>

          {/* Selected student info */}
          {selected && (
            <div className="rounded-lg bg-gray-700/50 px-4 py-3">
              <p className="font-medium text-white">{selected.name}</p>
              <p className="text-sm text-gray-400">{selected.email}</p>
              <p className="mt-1 text-sm text-blue-300">
                Текущий баланс: <span className="font-semibold">{formatPoints(selected.balance)}</span>
              </p>
            </div>
          )}

          <Input
            label="Количество поинтов"
            type="number"
            min="1"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="100"
          />

          <Input
            label="Описание"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Победа в хакатоне"
          />

          {error && (
            <p className="rounded-lg bg-red-900/30 px-3 py-2 text-sm text-red-400">{error}</p>
          )}

          <Button className="w-full" onClick={handleGrant} isLoading={loading} disabled={!canSubmit}>
            Начислить
          </Button>
        </Card>

        {/* Recent grants in this session */}
        {history.length > 0 && (
          <section>
            <h2 className="mb-3 text-sm font-semibold text-gray-400 uppercase tracking-wide">
              Последние начисления (сессия)
            </h2>
            <Card className="p-0 overflow-hidden">
              <ul className="divide-y divide-gray-700">
                {history.map((h) => (
                  <li key={h.key} className="flex items-center justify-between px-4 py-3 text-sm">
                    <div>
                      <p className="font-medium text-white">{h.studentName}</p>
                      <p className="text-xs text-gray-500">{h.description}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold text-green-400">+{h.amount}</p>
                      <p className="text-xs text-gray-500">{formatDate(h.grantedAt)}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </Card>
          </section>
        )}
      </main>
    </div>
  );
}
