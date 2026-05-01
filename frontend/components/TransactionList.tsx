import type { Transaction } from '@/lib/types';
import { Badge, Spinner } from '@/components/ui';
import { formatDate, formatTransactionAmount } from '@/lib/utils';

interface TransactionListProps {
  transactions: Transaction[];
  isLoading: boolean;
}

export function TransactionList({ transactions, isLoading }: TransactionListProps) {
  if (isLoading) {
    return (
      <div className="flex justify-center py-10">
        <Spinner />
      </div>
    );
  }

  if (transactions.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-gray-500">Пока нет операций</p>
    );
  }

  return (
    <ul className="divide-y divide-gray-700">
      {transactions.map((tx) => (
        <li key={tx.id} className="flex items-center gap-3 py-3">
          <Badge type={tx.type} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-gray-100">
              {tx.description || (tx.type === 'spend' ? 'Оплата у партнёра' : 'Начисление поинтов')}
            </p>
            <p className="text-xs text-gray-500">{formatDate(tx.created_at)}</p>
          </div>
          <span
            className={`shrink-0 text-sm font-semibold tabular-nums ${tx.amount >= 0 ? 'text-green-400' : 'text-red-400'}`}
          >
            {formatTransactionAmount(tx.amount)}
          </span>
        </li>
      ))}
    </ul>
  );
}
