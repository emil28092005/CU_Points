'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { PartnerCard } from '@/components/PartnerCard';
import { Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import type { Partner } from '@/lib/types';

export default function PartnersPage() {
  const [partners, setPartners] = useState<Partner[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const data = await api.get<Partner[]>('/api/v1/partners');
        setPartners(data);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Ошибка загрузки партнёров');
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  return (
    <main className="mx-auto max-w-lg p-4 pb-10 pt-6">
      <div className="mb-4 flex items-center gap-3">
        <Link href="/dashboard" className="text-sm text-blue-400 hover:text-blue-300">
          ← Назад
        </Link>
        <h1 className="text-xl font-bold text-white">Партнёры</h1>
      </div>

      {loading && (
        <div className="flex justify-center py-12">
          <Spinner />
        </div>
      )}

      {error && (
        <p className="rounded-lg bg-red-900/30 px-3 py-2 text-sm text-red-400">{error}</p>
      )}

      {!loading && !error && partners.length === 0 && (
        <p className="py-8 text-center text-sm text-gray-500">Партнёры пока не добавлены</p>
      )}

      {!loading && partners.length > 0 && (
        <div className="grid gap-3 sm:grid-cols-2">
          {partners.map((p) => (
            <PartnerCard key={p.id} partner={p} />
          ))}
        </div>
      )}
    </main>
  );
}
