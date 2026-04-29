'use client';

import { Badge } from '@/components/ui/badge';

const statusConfig = {
  open: { label: 'Open', className: 'bg-blue-100 text-blue-800 hover:bg-blue-100' },
  in_progress: { label: 'In Progress', className: 'bg-yellow-100 text-yellow-800 hover:bg-yellow-100' },
  resolved: { label: 'Resolved', className: 'bg-green-100 text-green-800 hover:bg-green-100' },
  closed: { label: 'Closed', className: 'bg-gray-100 text-gray-800 hover:bg-gray-100' },
  cancelled: { label: 'Cancelled', className: 'bg-red-100 text-red-800 hover:bg-red-100' },
};

export function StatusBadge({ status }) {
  const config = statusConfig[status] || statusConfig.open;
  
  return (
    <Badge variant="secondary" className={config.className}>
      {config.label}
    </Badge>
  );
}
