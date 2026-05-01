'use client';

import { StudentNav } from '@/components/StudentNav';
import { QRDisplay } from '@/components/QRDisplay';

export default function QRPage() {
  return (
    <div className="min-h-screen">
      <StudentNav />
      <main className="flex flex-col items-center justify-center gap-6 p-6 pt-12">
        <h1 className="text-xl font-bold text-white">Оплата поинтами</h1>

        <QRDisplay />

        <p className="max-w-xs text-center text-sm text-gray-500">
          Покажи этот QR кассиру. Он действителен 5 минут и может быть использован только один раз.
        </p>
      </main>
    </div>
  );
}
