'use client';

import { useState, useCallback } from 'react';
import { QRScanner } from '@/components/QRScanner';
import { Button, Card, Input } from '@/components/ui';
import { api } from '@/lib/api';
import { formatPoints } from '@/lib/utils';
import { useAuthStore } from '@/lib/store';
import type { SpendResult } from '@/lib/types';

type Step = 'scan' | 'form' | 'result';

// Default cap: partner can spend at most 50% of the purchase total in points.
// The backend validates against the actual partner's max_spend_pct.
const DEFAULT_MAX_SPEND_PCT = 0.5;

export default function ScanPage() {
  const { user, logout } = useAuthStore();
  const [step, setStep] = useState<Step>('scan');
  const [qrToken, setQrToken] = useState('');
  const [purchaseTotal, setPurchaseTotal] = useState('');
  const [pointsToSpend, setPointsToSpend] = useState('');
  const [result, setResult] = useState<SpendResult | null>(null);
  const [errorMsg, setErrorMsg] = useState('');
  const [loading, setLoading] = useState(false);

  const handleScan = useCallback((token: string) => {
    setQrToken(token);
    setStep('form');
  }, []);

  function handlePurchaseTotalChange(val: string) {
    setPurchaseTotal(val);
    const total = parseFloat(val);
    if (!isNaN(total) && total > 0) {
      setPointsToSpend(String(Math.floor(total * DEFAULT_MAX_SPEND_PCT)));
    } else {
      setPointsToSpend('');
    }
  }

  function handlePointsChange(val: string) {
    const total = parseFloat(purchaseTotal);
    const max = Math.floor(total * DEFAULT_MAX_SPEND_PCT);
    const entered = parseInt(val, 10);
    // Prevent partner from setting points above the allowed cap.
    if (!isNaN(entered) && !isNaN(max) && entered > max) {
      setPointsToSpend(String(max));
    } else {
      setPointsToSpend(val);
    }
  }

  const totalNum = parseFloat(purchaseTotal) || 0;
  const pointsNum = parseInt(pointsToSpend, 10) || 0;
  const remainder = Math.max(0, totalNum - pointsNum);

  async function handleSubmit() {
    if (pointsNum <= 0) return;
    setLoading(true);
    setErrorMsg('');
    try {
      const data = await api.post<SpendResult>('/api/v1/partner/spend', {
        qr_token: qrToken,
        amount: pointsNum,
      });
      setResult(data);
      setStep('result');
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Ошибка';
      if (msg.includes('insufficient')) {
        setErrorMsg('Недостаточно поинтов на балансе студента.');
      } else if (msg.includes('already been used')) {
        setErrorMsg('Этот QR-код уже был использован. Попросите студента создать новый.');
      } else if (msg.includes('invalid or expired') || msg.includes('expired')) {
        setErrorMsg('QR-код недействителен или истёк. Попросите студента создать новый QR.');
      } else {
        setErrorMsg(msg);
      }
      setStep('result');
    } finally {
      setLoading(false);
    }
  }

  function reset() {
    setStep('scan');
    setQrToken('');
    setPurchaseTotal('');
    setPointsToSpend('');
    setResult(null);
    setErrorMsg('');
  }

  return (
    <div className="flex min-h-screen flex-col">
      {/* Partner header */}
      <header className="border-b border-gray-700 bg-gray-900 px-4 py-3">
        <div className="mx-auto flex max-w-md items-center justify-between">
          <span className="text-sm font-semibold text-white">
            CU Points — {user?.name ?? 'Партнёр'}
          </span>
          <button
            onClick={logout}
            className="text-sm text-gray-400 hover:text-white transition-colors"
          >
            Выйти
          </button>
        </div>
      </header>

      <main className="mx-auto w-full max-w-md flex-1 p-4 pt-6 pb-10">
        <h1 className="mb-6 text-xl font-bold text-white">Оплата поинтами</h1>

        {/* ── Step 1: QR scan ─────────────────────────────── */}
        {step === 'scan' && (
          <div className="space-y-3">
            <p className="text-sm text-gray-400">Наведи камеру на QR студента</p>
            <QRScanner onScan={handleScan} />
          </div>
        )}

        {/* ── Step 2: Amount entry ─────────────────────────── */}
        {step === 'form' && (
          <Card className="space-y-5">
            <div className="flex items-center gap-2">
              <span className="text-green-400 text-lg font-bold">✓</span>
              <p className="font-medium text-green-400">Студент отсканирован</p>
            </div>

            <Input
              label="Сумма покупки (₽)"
              type="number"
              min="1"
              step="1"
              value={purchaseTotal}
              onChange={(e) => handlePurchaseTotalChange(e.target.value)}
              placeholder="0"
              autoFocus
            />

            <Input
              label="Списать поинтов"
              type="number"
              min="0"
              max={Math.floor(totalNum * DEFAULT_MAX_SPEND_PCT)}
              value={pointsToSpend}
              onChange={(e) => handlePointsChange(e.target.value)}
              placeholder="0"
            />

            {totalNum > 0 && (
              <p className="text-sm text-gray-400">
                Остаток к оплате:{' '}
                <span className="font-semibold text-gray-200">{remainder.toFixed(0)} ₽</span>
              </p>
            )}

            <Button
              className="w-full"
              onClick={handleSubmit}
              isLoading={loading}
              disabled={pointsNum <= 0 || totalNum <= 0}
            >
              Списать поинты
            </Button>
            <Button variant="ghost" className="w-full" onClick={reset}>
              Отмена
            </Button>
          </Card>
        )}

        {/* ── Step 3: Result ───────────────────────────────── */}
        {step === 'result' && (
          <Card className="space-y-4 text-center">
            {result ? (
              <>
                <p className="text-5xl">✓</p>
                <p className="text-xl font-semibold text-green-400">
                  Списано {result.spent} поинтов
                </p>
                <p className="text-sm text-gray-400">
                  Новый баланс студента:{' '}
                  <span className="font-medium text-gray-200">
                    {formatPoints(result.new_balance)}
                  </span>
                </p>
              </>
            ) : (
              <>
                <p className="text-5xl">✗</p>
                <p className="text-sm text-red-400">{errorMsg || 'Неизвестная ошибка'}</p>
              </>
            )}
            <Button className="w-full mt-2" onClick={reset}>
              Новая операция
            </Button>
          </Card>
        )}
      </main>
    </div>
  );
}
