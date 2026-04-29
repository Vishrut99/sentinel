'use client';

import { AlertTriangle } from 'lucide-react';

export function SLAWarning({ breached }) {
  if (!breached) return null;
  
  return (
    <div className="inline-flex items-center gap-1 px-2 py-1 bg-red-100 text-red-800 rounded text-sm font-medium">
      <AlertTriangle className="h-4 w-4" />
      SLA Breached
    </div>
  );
}
