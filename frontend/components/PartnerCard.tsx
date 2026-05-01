import type { Partner } from '@/lib/types';
import { Card } from '@/components/ui';

interface PartnerCardProps {
  partner: Partner;
}

export function PartnerCard({ partner }: PartnerCardProps) {
  return (
    <Card className="flex flex-col gap-1">
      <p className="font-semibold text-white">{partner.name}</p>
      <p className="text-sm text-gray-400">{partner.address}</p>
      <p className="mt-2 inline-flex items-center self-start rounded-full bg-blue-900/40 px-2.5 py-0.5 text-xs font-medium text-blue-300">
        До {partner.max_spend_pct}% поинтами
      </p>
    </Card>
  );
}
