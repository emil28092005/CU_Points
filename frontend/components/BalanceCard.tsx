'use client';

import { useRouter } from 'next/navigation';
import { Card, Button } from '@/components/ui';
import { formatPoints } from '@/lib/utils';

interface BalanceCardProps {
  balance: number;
  name: string;
}

export function BalanceCard({ balance, name }: BalanceCardProps) {
  const router = useRouter();

  return (
    <Card className="bg-gradient-to-br from-blue-700 to-blue-900 text-white ring-0 shadow-lg">
      <p className="text-sm text-blue-200">{name}</p>
      <p className="mt-2 text-5xl font-bold tracking-tight">{formatPoints(balance)}</p>
      <p className="mt-1 text-sm text-blue-300">Доступно поинтов</p>
      <Button
        variant="secondary"
        className="mt-5 bg-white/10 text-white hover:bg-white/20 border border-white/20"
        onClick={() => router.push('/qr')}
      >
        Показать QR
      </Button>
    </Card>
  );
}
