'use client';

import { Badge } from '@/components/ui/badge';

const priorityConfig = {
  critical: { label: 'Critical', className: 'bg-red-100 text-red-800 hover:bg-red-100' },
  high: { label: 'High', className: 'bg-orange-100 text-orange-800 hover:bg-orange-100' },
  medium: { label: 'Medium', className: 'bg-yellow-100 text-yellow-800 hover:bg-yellow-100' },
  low: { label: 'Low', className: 'bg-gray-100 text-gray-800 hover:bg-gray-100' },
};

export function PriorityBadge({ priority }) {
  const config = priorityConfig[priority] || priorityConfig.low;
  
  return (
    <Badge variant="secondary" className={config.className}>
      {config.label}
    </Badge>
  );
}
