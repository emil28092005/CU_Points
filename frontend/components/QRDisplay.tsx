'use client';

import { useState, useEffect, useCallback } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { Button, Spinner } from '@/components/ui';
import { api } from '@/lib/api';
import type { QRResponse } from '@/lib/types';

const QR_TTL_SECONDS = 300; // 5 minutes, matches backend JWT TTL

export function QRDisplay() {
  const [token, setToken] = useState<string | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(QR_TTL_SECONDS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchToken = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<QRResponse>('/api/v1/me/qr');
      setToken(data.token);
      setSecondsLeft(QR_TTL_SECONDS);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось получить QR-код');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchToken();
  }, [fetchToken]);

  // Countdown tick — stops at 0.
  useEffect(() => {
    if (!token || secondsLeft <= 0) return;
    const id = setInterval(() => setSecondsLeft((s) => s - 1), 1000);
    return () => clearInterval(id);
  }, [token, secondsLeft]);

  const expired = secondsLeft <= 0;
  const minutes = Math.floor(secondsLeft / 60);
  const seconds = secondsLeft % 60;
  const countdownText = `${minutes}:${String(seconds).padStart(2, '0')}`;

  if (loading) {
    return (
      <div className="flex flex-col items-center gap-4 py-12">
        <Spinner size="lg" />
        <p className="text-sm text-gray-500">Генерируем QR-код…</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-12">
        <p className="text-sm text-red-400">{error}</p>
        <Button onClick={fetchToken}>Попробовать снова</Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center gap-5">
      {/* QR code always has a white bg — required for scanner contrast */}
      <div
        className={`rounded-2xl bg-white p-5 shadow-xl transition-opacity ${expired ? 'opacity-20' : ''}`}
      >
        {token && !expired ? (
          <QRCodeSVG value={token} size={220} level="H" includeMargin={false} />
        ) : (
          <div className="flex h-[220px] w-[220px] items-center justify-center">
            <p className="text-sm text-gray-400">QR-код истёк</p>
          </div>
        )}
      </div>

      {!expired ? (
        <p className="text-sm text-gray-400">
          Действителен ещё{' '}
          <span
            className={`font-semibold tabular-nums ${secondsLeft < 60 ? 'text-red-400' : 'text-gray-200'}`}
          >
            {countdownText}
          </span>
        </p>
      ) : (
        <Button onClick={fetchToken} isLoading={loading}>
          Обновить
        </Button>
      )}
    </div>
  );
}
