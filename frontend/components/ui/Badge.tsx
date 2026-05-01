import type { TransactionType } from '@/lib/types';

const TYPE_LABELS: Record<TransactionType, string> = {
  earn: 'Начисление',
  spend: 'Списание',
  admin_grant: 'Начисление',
  expire: 'Сгорание',
};

const TYPE_CLASSES: Record<TransactionType, string> = {
  earn: 'bg-green-900/50 text-green-400',
  spend: 'bg-red-900/50 text-red-400',
  admin_grant: 'bg-blue-900/50 text-blue-400',
  expire: 'bg-gray-700 text-gray-400',
};

interface BadgeProps {
  type: TransactionType;
}

export function Badge({ type }: BadgeProps) {
  return (
    <span
      className={`inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-xs font-medium ${TYPE_CLASSES[type]}`}
    >
      {TYPE_LABELS[type]}
    </span>
  );
}
