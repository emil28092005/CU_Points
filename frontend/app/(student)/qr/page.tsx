'use client';

import Link from 'next/link';
import { QRDisplay } from '@/components/QRDisplay';

export default function QRPage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 p-6">
      <h1 className="text-xl font-bold text-white">Оплата поинтами</h1>

      <QRDisplay />

      <p className="max-w-xs text-center text-sm text-gray-500">
        Покажи этот QR кассиру. Он действителен 5 минут и может быть использован только один раз.
      </p>

      <Link href="/dashboard" className="text-sm text-blue-400 hover:text-blue-300">
        ← Назад
      </Link>
    </main>
  );
}
