'use client';

import { useEffect, useRef, useState } from 'react';
import { Spinner } from '@/components/ui';

interface QRScannerProps {
  onScan: (token: string) => void;
}

export function QRScanner({ onScan }: QRScannerProps) {
  const [cameraError, setCameraError] = useState<string | null>(null);
  const [starting, setStarting] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);
  // Keep a stable ref to onScan so restarting the scanner when the parent
  // re-renders (e.g. state changes in the parent) is not needed.
  const onScanRef = useRef(onScan);
  onScanRef.current = onScan;
  const stopRef = useRef<(() => Promise<void>) | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // Use a unique id per mount — prevents html5-qrcode conflicts on StrictMode
    // double-invoke (the second run would find the first run's leftover DOM).
    const id = `qr-${Date.now()}`;
    container.id = id;

    let cancelled = false;

    import('html5-qrcode').then(({ Html5Qrcode }) => {
      if (cancelled) return;

      const scanner = new Html5Qrcode(id);
      stopRef.current = () => scanner.stop();

      scanner
        .start(
          { facingMode: 'environment' },
          { fps: 10, qrbox: { width: 220, height: 220 } },
          (text) => {
            if (!cancelled) onScanRef.current(text);
            scanner.stop().catch(() => {});
          },
          undefined,
        )
        .then(() => {
          if (!cancelled) setStarting(false);
        })
        .catch((err: unknown) => {
          if (cancelled) return;
          setStarting(false);
          const msg = String(err).toLowerCase();
          if (msg.includes('permission') || msg.includes('notallowed')) {
            setCameraError('Нет доступа к камере. Разрешите доступ в настройках браузера.');
          } else if (msg.includes('notfound') || msg.includes('no camera') || msg.includes('no cameras')) {
            setCameraError('Камера не найдена на этом устройстве.');
          } else {
            setCameraError('Не удалось запустить камеру.');
          }
        });
    });

    return () => {
      cancelled = true;
      stopRef.current?.().catch(() => {});
    };
  }, []); // intentionally empty — scanner starts once on mount

  if (cameraError) {
    return (
      <div className="flex min-h-[240px] items-center justify-center rounded-2xl bg-gray-800 p-6 text-center ring-1 ring-gray-700">
        <p className="text-sm text-red-400">{cameraError}</p>
      </div>
    );
  }

  return (
    <div className="relative min-h-[240px] overflow-hidden rounded-2xl bg-black ring-1 ring-gray-700">
      {starting && (
        <div className="absolute inset-0 flex items-center justify-center">
          <Spinner />
        </div>
      )}
      {/* html5-qrcode injects <video> and overlay elements here */}
      <div ref={containerRef} className="w-full" />
    </div>
  );
}
